package databases

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/obiente/cloud/apps/shared/pkg/auth"
	commonv1 "github.com/obiente/cloud/apps/shared/proto/obiente/cloud/common/v1"
	databasesv1 "github.com/obiente/cloud/apps/shared/proto/obiente/cloud/databases/v1"

	"connectrpc.com/connect"
)

// GetDatabaseSchema gets the schema information for a database
func (s *Service) GetDatabaseSchema(ctx context.Context, req *connect.Request[databasesv1.GetDatabaseSchemaRequest]) (*connect.Response[databasesv1.GetDatabaseSchemaResponse], error) {
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

	// Get connection info
	conn, err := s.connRepo.GetByDatabaseID(ctx, req.Msg.GetDatabaseId())
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to get connection info: %w", err))
	}

	// Connect to the database and introspect
	dbName := req.Msg.GetDatabaseName()
	if dbName == "" {
		dbName = conn.DatabaseName
	}

	// Build connection string based on database type
	var db *sql.DB
	switch databasesv1.DatabaseType(dbInstance.Type) {
	case databasesv1.DatabaseType_POSTGRESQL:
		connStr := fmt.Sprintf("postgres://%s:%s@%s:%d/%s?sslmode=require",
			conn.Username, conn.Password, conn.Host, conn.Port, dbName)
		db, err = sql.Open("postgres", connStr)
	case databasesv1.DatabaseType_MYSQL, databasesv1.DatabaseType_MARIADB:
		connStr := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s",
			conn.Username, conn.Password, conn.Host, conn.Port, dbName)
		db, err = sql.Open("mysql", connStr)
	default:
		return nil, connect.NewError(connect.CodeUnimplemented, fmt.Errorf("introspection not yet supported for database type %d", dbInstance.Type))
	}

	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to connect to database: %w", err))
	}
	defer db.Close()

	// Get tables
	tables, err := s.introspectTables(ctx, db, databasesv1.DatabaseType(dbInstance.Type), dbName)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to introspect tables: %w", err))
	}

	dbNamePtr := &dbName
	res := connect.NewResponse(&databasesv1.GetDatabaseSchemaResponse{
		DatabaseId:   req.Msg.GetDatabaseId(),
		DatabaseName: dbNamePtr,
		Tables:       tables,
	})
	return res, nil
}

// ListTables lists tables in a database
func (s *Service) ListTables(ctx context.Context, req *connect.Request[databasesv1.ListTablesRequest]) (*connect.Response[databasesv1.ListTablesResponse], error) {
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

	// Get connection info and introspect
	// Similar to GetDatabaseSchema but just return table list
	// Implementation would be similar...

	res := connect.NewResponse(&databasesv1.ListTablesResponse{
		Tables: []*databasesv1.TableInfo{},
		Pagination: &commonv1.Pagination{
			Page:    req.Msg.GetPage(),
			PerPage: req.Msg.GetPerPage(),
			Total:   0,
		},
	})
	return res, nil
}

// Helper function to introspect tables (placeholder)
func (s *Service) introspectTables(ctx context.Context, db *sql.DB, dbType databasesv1.DatabaseType, schema string) ([]*databasesv1.TableInfo, error) {
	// This is a placeholder - actual implementation would query INFORMATION_SCHEMA or equivalent
	// based on the database type
	return []*databasesv1.TableInfo{}, nil
}

// GetTableStructure gets the structure of a specific table
func (s *Service) GetTableStructure(ctx context.Context, req *connect.Request[databasesv1.GetTableStructureRequest]) (*connect.Response[databasesv1.GetTableStructureResponse], error) {
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

	// Get database and introspect table
	// Implementation would query the database for table structure...

	res := connect.NewResponse(&databasesv1.GetTableStructureResponse{
		Table: &databasesv1.TableInfo{
			Name: req.Msg.GetTableName(),
		},
	})
	return res, nil
}

