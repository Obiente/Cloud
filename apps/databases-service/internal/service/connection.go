package databases

import (
	"context"
	"fmt"

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

	// Create connection info without password (for security)
	connInfo := &databasesv1.DatabaseConnectionInfo{
		DatabaseId:     req.Msg.GetDatabaseId(),
		Host:           conn.Host,
		Port:           conn.Port,
		DatabaseName:   conn.DatabaseName,
		Username:       conn.Username,
		SslRequired:    conn.SSLRequired,
		SslCertificate: conn.SSLCertificate,
	}

	// Generate connection strings (without password)
	connInfo.PostgresqlUrl = fmt.Sprintf("postgresql://%s:***@%s:%d/%s?sslmode=require",
		conn.Username, conn.Host, conn.Port, conn.DatabaseName)
	connInfo.MysqlUrl = fmt.Sprintf("mysql://%s:***@%s:%d/%s?ssl-mode=REQUIRED",
		conn.Username, conn.Host, conn.Port, conn.DatabaseName)
	connInfo.MongodbUrl = fmt.Sprintf("mongodb://%s:***@%s:%d/%s?ssl=true",
		conn.Username, conn.Host, conn.Port, conn.DatabaseName)
	connInfo.RedisUrl = fmt.Sprintf("redis://:***@%s:%d",
		conn.Host, conn.Port)

	connInfo.ConnectionInstructions = fmt.Sprintf(
		"Connect to your database using:\nHost: %s\nPort: %d\nDatabase: %s\nUsername: %s\n\nNote: Password is only shown once during database creation or password reset.",
		conn.Host, conn.Port, conn.DatabaseName, conn.Username,
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
	// For now, just update the connection record
	conn.Password = newPassword
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

