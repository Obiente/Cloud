package databases

import (
	"context"
	"fmt"
	"os"

	"github.com/obiente/cloud/apps/shared/pkg/auth"

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
		proxyHost = conn.Host
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

	// Generate new password
	newPassword := generateRandomPassword(32)

	// TODO: Actually reset the password in the database
	// For now, just update the connection record with the encrypted password
	encryptedPassword, err := s.secretManager.EncryptPassword(newPassword)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to encrypt password: %w", err))
	}
	conn.Password = encryptedPassword
	if err := s.connRepo.Update(ctx, conn); err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to update password: %w", err))
	}

	username := conn.Username
	if req.Msg.Username != nil && *req.Msg.Username != "" {
		username = *req.Msg.Username
	}

	res := connect.NewResponse(&databasesv1.ResetDatabasePasswordResponse{
		DatabaseId:  req.Msg.GetDatabaseId(),
		Username:    username,
		NewPassword: newPassword,
		Message:     "Password has been reset. Please save this password now - it will not be shown again.",
	})
	return res, nil
}
