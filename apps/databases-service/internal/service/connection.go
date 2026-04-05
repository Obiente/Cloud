package databases

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/obiente/cloud/apps/shared/pkg/auth"
	"github.com/obiente/cloud/apps/shared/pkg/database"
	"github.com/obiente/cloud/apps/shared/pkg/logger"

	databasesv1 "github.com/obiente/cloud/apps/shared/proto/obiente/cloud/databases/v1"

	"connectrpc.com/connect"
)

// GetDatabaseConnectionInfo gets connection information for a database
func (s *Service) GetDatabaseConnectionInfo(ctx context.Context, req *connect.Request[databasesv1.GetDatabaseConnectionInfoRequest]) (*connect.Response[databasesv1.GetDatabaseConnectionInfoResponse], error) {
	orgID := req.Msg.GetOrganizationId()
	if orgID == "" {
		if eff, ok := resolveUserDefaultOrgID(ctx); ok {
			orgID = eff
		}
	}

	// Check resource-level permission
	if err := s.checkDatabasePermission(ctx, req.Msg.GetDatabaseId(), auth.PermissionDatabaseRead); err != nil {
		return nil, connect.NewError(connect.CodePermissionDenied, err)
	}

	// Get database
	dbInstance, err := s.repo.GetByID(ctx, req.Msg.GetDatabaseId())
	if err != nil {
		return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("database not found: %w", err))
	}

	// Verify organization ownership
	if dbInstance.OrganizationID != orgID {
		return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("database not found"))
	}

	// Get connection info (without password for security)
	conn, err := s.connRepo.GetByDatabaseID(ctx, req.Msg.GetDatabaseId())
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to get connection info: %w", err))
	}

	// Determine proxy host and port for connection info
	proxyHost := os.Getenv("DATABASE_PROXY_HOST")
	if proxyHost == "" {
		proxyHost = database.DefaultMyObienteCloudDomain(req.Msg.GetDatabaseId())
		if proxyHost == "" {
			proxyHost = conn.Host
		}
	}

	// Determine the external port based on database type
	var externalPort int32
	switch databasesv1.DatabaseType(dbInstance.Type) {
	case databasesv1.DatabaseType_POSTGRESQL:
		externalPort = 5432
	case databasesv1.DatabaseType_MYSQL, databasesv1.DatabaseType_MARIADB:
		externalPort = 3306
	case databasesv1.DatabaseType_MONGODB:
		externalPort = 27017
	case databasesv1.DatabaseType_REDIS:
		// Use allocated port from registry
		if s.routeRegistry != nil {
			if route, ok := s.routeRegistry.LookupByID(req.Msg.GetDatabaseId()); ok && route.RedisPort > 0 {
				externalPort = int32(route.RedisPort)
			} else {
				if err := s.routeRegistry.LoadFromDatabase(ctx); err == nil {
					if route, ok := s.routeRegistry.LookupByID(req.Msg.GetDatabaseId()); ok && route.RedisPort > 0 {
						externalPort = int32(route.RedisPort)
						break
					}
				}
				externalPort = conn.Port
			}
		} else {
			externalPort = conn.Port
		}
	default:
		externalPort = conn.Port
	}

	// The routing key is the database ID (db-{id})
	routingDBName := req.Msg.GetDatabaseId()

	// Create connection info without password (for security)
	connInfo := &databasesv1.DatabaseConnectionInfo{
		DatabaseId:     req.Msg.GetDatabaseId(),
		Host:           proxyHost,
		Port:           externalPort,
		DatabaseName:   routingDBName,
		Username:       conn.Username,
		SslRequired:    false,
		SslCertificate: conn.SSLCertificate,
	}

	// Generate connection strings (without password) using proxy host/port
	switch databasesv1.DatabaseType(dbInstance.Type) {
	case databasesv1.DatabaseType_POSTGRESQL:
		connInfo.PostgresqlUrl = fmt.Sprintf("postgresql://%s:***@%s:%d/%s?sslmode=prefer",
			conn.Username, proxyHost, externalPort, routingDBName)
	case databasesv1.DatabaseType_MYSQL, databasesv1.DatabaseType_MARIADB:
		connInfo.MysqlUrl = fmt.Sprintf("mysql://%s:***@%s:%d/%s",
			conn.Username, proxyHost, externalPort, routingDBName)
	case databasesv1.DatabaseType_MONGODB:
		connInfo.MongodbUrl = fmt.Sprintf("mongodb://%s:***@%s:%d/%s",
			conn.Username, proxyHost, externalPort, routingDBName)
	case databasesv1.DatabaseType_REDIS:
		connInfo.RedisUrl = fmt.Sprintf("redis://:***@%s:%d",
			proxyHost, externalPort)
	}

	connInfo.ConnectionInstructions = fmt.Sprintf(
		"Connect to your database using:\nHost: %s\nPort: %d\nDatabase: %s\nUsername: %s\n\nNote: Password is only shown once during database creation or password reset.",
		proxyHost, externalPort, routingDBName, conn.Username,
	)

	res := connect.NewResponse(&databasesv1.GetDatabaseConnectionInfoResponse{
		ConnectionInfo: connInfo,
	})
	return res, nil
}

// ResetDatabasePassword resets the password for a database user
func (s *Service) ResetDatabasePassword(ctx context.Context, req *connect.Request[databasesv1.ResetDatabasePasswordRequest]) (*connect.Response[databasesv1.ResetDatabasePasswordResponse], error) {
	orgID := req.Msg.GetOrganizationId()
	if orgID == "" {
		if eff, ok := resolveUserDefaultOrgID(ctx); ok {
			orgID = eff
		}
	}

	// Check resource-level permission
	if err := s.checkDatabasePermission(ctx, req.Msg.GetDatabaseId(), auth.PermissionDatabaseUpdate); err != nil {
		return nil, connect.NewError(connect.CodePermissionDenied, err)
	}

	// Get database
	dbInstance, err := s.repo.GetByID(ctx, req.Msg.GetDatabaseId())
	if err != nil {
		return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("database not found: %w", err))
	}

	// Verify organization ownership
	if dbInstance.OrganizationID != orgID {
		return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("database not found"))
	}

	// Get connection info
	conn, err := s.connRepo.GetByDatabaseID(ctx, req.Msg.GetDatabaseId())
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to get connection info: %w", err))
	}

	username := conn.Username
	if requestedUsername := strings.TrimSpace(req.Msg.GetUsername()); requestedUsername != "" && requestedUsername != conn.Username {
		return nil, connect.NewError(
			connect.CodeUnimplemented,
			fmt.Errorf("password reset currently supports only the primary database user %q", conn.Username),
		)
	}

	// Generate new password
	newPassword := generateRandomPassword(32)

	resetCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	if err := s.applyPrimaryDatabasePassword(resetCtx, req.Msg.GetDatabaseId(), databasesv1.DatabaseType(dbInstance.Type), username, newPassword); err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to reset password in database engine: %w", err))
	}

	// Persist the new password only after the database engine accepted it.
	encryptedPassword := newPassword
	if s.secretManager != nil {
		if encrypted, encErr := s.secretManager.EncryptPassword(newPassword); encErr == nil {
			encryptedPassword = encrypted
		} else {
			logger.Warn("Failed to encrypt reset password for %s: %v. Storing plaintext fallback.", req.Msg.GetDatabaseId(), encErr)
		}
	} else {
		logger.Warn("Secret manager not configured for reset password on %s. Storing plaintext fallback.", req.Msg.GetDatabaseId())
	}
	conn.Password = encryptedPassword
	if err := s.connRepo.Update(ctx, conn); err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to update password: %w", err))
	}

	res := connect.NewResponse(&databasesv1.ResetDatabasePasswordResponse{
		DatabaseId:  req.Msg.GetDatabaseId(),
		Username:    username,
		NewPassword: newPassword,
		Message:     "Password has been reset. Please save this password now - it will not be shown again.",
	})
	return res, nil
}

func (s *Service) applyPrimaryDatabasePassword(ctx context.Context, databaseID string, dbType databasesv1.DatabaseType, username, newPassword string) error {
	switch dbType {
	case databasesv1.DatabaseType_POSTGRESQL:
		db, _, err := s.openDirectConnection(ctx, databaseID, databaseID)
		if err != nil {
			return err
		}
		defer db.Close()

		statement := fmt.Sprintf(
			"ALTER USER %s WITH PASSWORD %s",
			quoteIdentifier(username, databasesv1.DatabaseType_POSTGRESQL),
			quoteSQLLiteral(newPassword),
		)
		if _, err := db.ExecContext(ctx, statement); err != nil {
			return fmt.Errorf("postgres password update failed: %w", err)
		}
		return nil
	case databasesv1.DatabaseType_MYSQL, databasesv1.DatabaseType_MARIADB:
		return fmt.Errorf("password reset is not implemented yet for %s databases", dbType.String())
	case databasesv1.DatabaseType_MONGODB:
		return fmt.Errorf("password reset is not implemented yet for mongodb databases")
	case databasesv1.DatabaseType_REDIS:
		return fmt.Errorf("password reset is not implemented yet for redis databases")
	default:
		return fmt.Errorf("unsupported database type %d", dbType)
	}
}

func quoteSQLLiteral(value string) string {
	return "'" + strings.ReplaceAll(value, "'", "''") + "'"
}
