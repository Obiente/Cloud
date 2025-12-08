package databases

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/obiente/cloud/apps/shared/pkg/auth"

	databasesv1 "github.com/obiente/cloud/apps/shared/proto/obiente/cloud/databases/v1"

	"connectrpc.com/connect"
	_ "github.com/go-sql-driver/mysql"
	_ "github.com/lib/pq"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// ExecuteQuery executes a SQL query on a database
func (s *Service) ExecuteQuery(ctx context.Context, req *connect.Request[databasesv1.ExecuteQueryRequest]) (*connect.Response[databasesv1.ExecuteQueryResponse], error) {
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

	// Connect to database
	dbName := req.Msg.GetDatabaseName()
	if dbName == "" {
		dbName = conn.DatabaseName
	}

	var db *sql.DB
	startTime := time.Now()
	switch databasesv1.DatabaseType(dbInstance.Type) {
	case databasesv1.DatabaseType_POSTGRESQL:
		connStr := fmt.Sprintf("postgres://%s:%s@%s:%d/%s?sslmode=require",
			conn.Username, conn.Password, conn.Host, conn.Port, dbName)
		var err error
		db, err = sql.Open("postgres", connStr)
		if err != nil {
			return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to connect: %w", err))
		}
		defer db.Close()
	case databasesv1.DatabaseType_MYSQL, databasesv1.DatabaseType_MARIADB:
		connStr := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s",
			conn.Username, conn.Password, conn.Host, conn.Port, dbName)
		var err error
		db, err = sql.Open("mysql", connStr)
		if err != nil {
			return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to connect: %w", err))
		}
		defer db.Close()
	default:
		return nil, connect.NewError(connect.CodeUnimplemented, fmt.Errorf("query execution not yet supported for database type %d", dbInstance.Type))
	}

	// Set timeout
	timeout := 30 * time.Second
	if req.Msg.TimeoutSeconds != nil && *req.Msg.TimeoutSeconds > 0 {
		timeout = time.Duration(*req.Msg.TimeoutSeconds) * time.Second
	}
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	// Execute query
	rows, err := db.QueryContext(ctx, req.Msg.GetQuery())
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("query execution failed: %w", err))
	}
	defer rows.Close()

	// Get column information
	columns, err := rows.Columns()
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to get columns: %w", err))
	}

	columnTypes, err := rows.ColumnTypes()
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to get column types: %w", err))
	}

	// Build result columns
	resultColumns := make([]*databasesv1.QueryResultColumn, len(columns))
	for i, col := range columns {
		dataType := "unknown"
		if i < len(columnTypes) {
			dataType = columnTypes[i].DatabaseTypeName()
		}
		resultColumns[i] = &databasesv1.QueryResultColumn{
			Name:     col,
			DataType: dataType,
		}
	}

	// Read rows
	maxRows := 1000
	if req.Msg.MaxRows != nil && *req.Msg.MaxRows > 0 {
		maxRows = int(*req.Msg.MaxRows)
	}

	var resultRows []*databasesv1.QueryResultRow
	rowCount := 0
	truncated := false

	for rows.Next() {
		if rowCount >= maxRows {
			truncated = true
			break
		}

		// Create scan destination
		values := make([]interface{}, len(columns))
		valuePtrs := make([]interface{}, len(columns))
		for i := range values {
			valuePtrs[i] = &values[i]
		}

		if err := rows.Scan(valuePtrs...); err != nil {
			return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to scan row: %w", err))
		}

		// Convert to result cells
		cells := make([]*databasesv1.QueryResultCell, len(columns))
		for i, val := range values {
			cell := &databasesv1.QueryResultCell{
				ColumnName: columns[i],
				IsNull:     val == nil,
			}
			if val != nil {
				cellValue := fmt.Sprintf("%v", val)
				cell.Value = &cellValue
			}
			cells[i] = cell
		}

		resultRows = append(resultRows, &databasesv1.QueryResultRow{
			Cells: cells,
		})
		rowCount++
	}

	if err := rows.Err(); err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("row iteration error: %w", err))
	}

	executionTime := time.Since(startTime)

	res := connect.NewResponse(&databasesv1.ExecuteQueryResponse{
		Columns:         resultColumns,
		Rows:            resultRows,
		RowCount:        int32(rowCount),
		QueryType:       stringPtr("SELECT"), // TODO: Detect query type
		Truncated:       truncated,
		ExecutedAt:      timestamppb.Now(),
		ExecutionTimeMs: int32(executionTime.Milliseconds()),
	})
	return res, nil
}

// StreamQuery streams query results (placeholder)
func (s *Service) StreamQuery(ctx context.Context, req *connect.Request[databasesv1.StreamQueryRequest], stream *connect.ServerStream[databasesv1.QueryResultRow]) error {
	// Placeholder implementation
	return connect.NewError(connect.CodeUnimplemented, fmt.Errorf("streaming queries not yet implemented"))
}

