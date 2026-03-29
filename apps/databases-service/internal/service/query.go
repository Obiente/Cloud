package databases

import (
	"context"
	"database/sql"
	"fmt"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/obiente/cloud/apps/shared/pkg/auth"

	databasesv1 "github.com/obiente/cloud/apps/shared/proto/obiente/cloud/databases/v1"

	"connectrpc.com/connect"
	_ "github.com/go-sql-driver/mysql"
	_ "github.com/lib/pq"
	"google.golang.org/protobuf/types/known/timestamppb"
)

var sqlCommentPattern = regexp.MustCompile(`(?s)/\*.*?\*/|--[^\r\n]*|#[^\r\n]*`)

// ExecuteQuery executes a SQL query on a database
func (s *Service) ExecuteQuery(ctx context.Context, req *connect.Request[databasesv1.ExecuteQueryRequest]) (*connect.Response[databasesv1.ExecuteQueryResponse], error) {
	orgID := req.Msg.GetOrganizationId()
	if orgID == "" {
		if eff, ok := resolveUserDefaultOrgID(ctx); ok {
			orgID = eff
		}
	}

	queryType, readOnly, err := classifySQLQuery(req.Msg.GetQuery())
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}

	requiredPermission := auth.PermissionDatabaseManage
	if readOnly {
		requiredPermission = auth.PermissionDatabaseRead
	}

	// Check resource-level permission
	if err := s.checkDatabasePermission(ctx, req.Msg.GetDatabaseId(), requiredPermission); err != nil {
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

	// Connect to database directly on overlay network
	dbName := req.Msg.GetDatabaseName()
	if dbName == "" {
		dbName = conn.DatabaseName
	}

	// Check if database is sleeping/stopped and handle accordingly
	directHost := fmt.Sprintf("obiente-%s", req.Msg.GetDatabaseId())
	directPort := conn.Port
	if s.routeRegistry != nil {
		if route, ok := s.routeRegistry.LookupByID(req.Msg.GetDatabaseId()); ok {
			directPort = int32(route.InternalPort)
			if route.Stopped {
				if route.DBStatus == 5 { // STOPPED
					return nil, connect.NewError(connect.CodeFailedPrecondition, fmt.Errorf("database is stopped"))
				}
				// SLEEPING - wake it
				if s.routeRegistry.OnWake != nil {
					wakeCtx, wakeCancel := context.WithTimeout(context.Background(), 30*time.Second)
					ip, err := s.routeRegistry.OnWake(wakeCtx, route)
					wakeCancel()
					if err != nil {
						return nil, connect.NewError(connect.CodeUnavailable, fmt.Errorf("failed to wake database: %w", err))
					}
					directHost = ip
				}
			}
		}
	}

	// Decrypt password for direct connection
	password := conn.Password
	if s.secretManager != nil {
		if decrypted, err := s.secretManager.DecryptPassword(conn.Password); err == nil {
			password = decrypted
		}
	}

	// Set timeout early so it applies to the connection attempt too
	timeout := 10 * time.Second
	if req.Msg.TimeoutSeconds != nil && *req.Msg.TimeoutSeconds > 0 {
		timeout = time.Duration(*req.Msg.TimeoutSeconds) * time.Second
	}
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	var db *sql.DB
	startTime := time.Now()
	switch databasesv1.DatabaseType(dbInstance.Type) {
	case databasesv1.DatabaseType_POSTGRESQL:
		connStr := fmt.Sprintf("postgres://%s:%s@%s:%d/%s?sslmode=disable&connect_timeout=5",
			url.QueryEscape(conn.Username), url.QueryEscape(password), url.QueryEscape(directHost), directPort, url.QueryEscape(dbName))
		var err error
		db, err = sql.Open("postgres", connStr)
		if err != nil {
			return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to connect: %w", err))
		}
		defer db.Close()
	case databasesv1.DatabaseType_MYSQL, databasesv1.DatabaseType_MARIADB:
		connStr := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?timeout=5s",
			url.QueryEscape(conn.Username), url.QueryEscape(password), url.QueryEscape(directHost), directPort, url.QueryEscape(dbName))
		var err error
		db, err = sql.Open("mysql", connStr)
		if err != nil {
			return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to connect: %w", err))
		}
		defer db.Close()
	default:
		return nil, connect.NewError(connect.CodeUnimplemented, fmt.Errorf("query execution not yet supported for database type %d", dbInstance.Type))
	}

	// Verify connectivity before running query
	if err := db.PingContext(ctx); err != nil {
		return nil, connect.NewError(connect.CodeUnavailable, fmt.Errorf("database unreachable at %s:%d: %w", directHost, directPort, err))
	}

	if !readOnly {
		result, err := db.ExecContext(ctx, req.Msg.GetQuery())
		if err != nil {
			return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("query execution failed: %w", err))
		}

		affectedRows, err := result.RowsAffected()
		if err != nil {
			affectedRows = 0
		}

		executionTime := time.Since(startTime)

		res := connect.NewResponse(&databasesv1.ExecuteQueryResponse{
			Columns:         nil,
			Rows:            nil,
			RowCount:        0,
			AffectedRows:    int32Ptr(int32(affectedRows)),
			QueryType:       stringPtr(queryType),
			Truncated:       false,
			ExecutedAt:      timestamppb.Now(),
			ExecutionTimeMs: int32(executionTime.Milliseconds()),
		})
		return res, nil
	}

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
		QueryType:       stringPtr(queryType),
		Truncated:       truncated,
		ExecutedAt:      timestamppb.Now(),
		ExecutionTimeMs: int32(executionTime.Milliseconds()),
	})
	return res, nil
}

func classifySQLQuery(query string) (queryType string, readOnly bool, err error) {
	cleaned := sqlCommentPattern.ReplaceAllString(query, " ")
	cleaned = strings.TrimSpace(cleaned)

	for strings.HasSuffix(cleaned, ";") {
		cleaned = strings.TrimSpace(strings.TrimSuffix(cleaned, ";"))
	}

	if cleaned == "" {
		return "", false, fmt.Errorf("query is required")
	}

	if strings.Contains(cleaned, ";") {
		return "", false, fmt.Errorf("multiple SQL statements are not supported")
	}

	fields := strings.Fields(cleaned)
	if len(fields) == 0 {
		return "", false, fmt.Errorf("query is required")
	}

	queryType = strings.ToUpper(fields[0])

	switch queryType {
	case "SELECT", "SHOW", "DESCRIBE", "DESC", "EXPLAIN", "VALUES":
		return queryType, true, nil
	case "INSERT", "UPDATE", "DELETE", "CREATE", "ALTER", "DROP", "TRUNCATE", "GRANT", "REVOKE":
		return queryType, false, nil
	default:
		return queryType, false, nil
	}
}

func int32Ptr(v int32) *int32 {
	return &v
}

// StreamQuery streams query results (placeholder)
func (s *Service) StreamQuery(ctx context.Context, req *connect.Request[databasesv1.StreamQueryRequest], stream *connect.ServerStream[databasesv1.QueryResultRow]) error {
	// Placeholder implementation
	return connect.NewError(connect.CodeUnimplemented, fmt.Errorf("streaming queries not yet implemented"))
}
