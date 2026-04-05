package databases

import (
	"context"
	"database/sql"
	"errors"
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

var errUnsupportedQueryDatabaseType = errors.New("query execution not supported for this database type")

// ExecuteQuery executes a SQL query on a database
func (s *Service) ExecuteQuery(ctx context.Context, req *connect.Request[databasesv1.ExecuteQueryRequest]) (*connect.Response[databasesv1.ExecuteQueryResponse], error) {
	orgID := req.Msg.GetOrganizationId()
	if orgID == "" {
		if eff, ok := resolveUserDefaultOrgID(ctx); ok {
			orgID = eff
		}
	}

	queryCtx, cancel, db, queryType, readOnly, err := s.prepareQueryExecution(ctx, orgID, req.Msg.GetDatabaseId(), req.Msg.GetDatabaseName(), req.Msg.GetQuery(), req.Msg.TimeoutSeconds)
	if err != nil {
		return nil, err
	}
	defer cancel()
	defer db.Close()

	startTime := time.Now()

	if !readOnly {
		result, err := db.ExecContext(queryCtx, req.Msg.GetQuery())
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
	rows, err := db.QueryContext(queryCtx, req.Msg.GetQuery())
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

		resultRows = append(resultRows, buildQueryResultRow(columns, values))
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

func (s *Service) StreamQuery(ctx context.Context, req *connect.Request[databasesv1.StreamQueryRequest], stream *connect.ServerStream[databasesv1.QueryResultRow]) error {
	orgID := req.Msg.GetOrganizationId()
	if orgID == "" {
		if eff, ok := resolveUserDefaultOrgID(ctx); ok {
			orgID = eff
		}
	}

	queryCtx, cancel, db, _, readOnly, err := s.prepareQueryExecution(ctx, orgID, req.Msg.GetDatabaseId(), req.Msg.GetDatabaseName(), req.Msg.GetQuery(), req.Msg.TimeoutSeconds)
	if err != nil {
		return err
	}
	defer cancel()
	defer db.Close()

	if !readOnly {
		return connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("streaming queries only supports read-only statements"))
	}

	rows, err := db.QueryContext(queryCtx, req.Msg.GetQuery())
	if err != nil {
		return connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("query execution failed: %w", err))
	}
	defer rows.Close()

	columns, err := rows.Columns()
	if err != nil {
		return connect.NewError(connect.CodeInternal, fmt.Errorf("failed to get columns: %w", err))
	}

	for rows.Next() {
		values := make([]interface{}, len(columns))
		valuePtrs := make([]interface{}, len(columns))
		for i := range values {
			valuePtrs[i] = &values[i]
		}

		if err := rows.Scan(valuePtrs...); err != nil {
			return connect.NewError(connect.CodeInternal, fmt.Errorf("failed to scan row: %w", err))
		}

		if err := stream.Send(buildQueryResultRow(columns, values)); err != nil {
			return err
		}
	}

	if err := rows.Err(); err != nil {
		return connect.NewError(connect.CodeInternal, fmt.Errorf("row iteration error: %w", err))
	}

	return nil
}

func (s *Service) prepareQueryExecution(ctx context.Context, organizationID, databaseID, databaseName, query string, timeoutSeconds *int32) (context.Context, context.CancelFunc, *sql.DB, string, bool, error) {
	queryType, readOnly, err := classifySQLQuery(query)
	if err != nil {
		return nil, nil, nil, "", false, connect.NewError(connect.CodeInvalidArgument, err)
	}

	requiredPermission := auth.PermissionDatabaseManage
	if readOnly {
		requiredPermission = auth.PermissionDatabaseRead
	}
	if err := s.checkDatabasePermission(ctx, databaseID, requiredPermission); err != nil {
		return nil, nil, nil, "", false, connect.NewError(connect.CodePermissionDenied, err)
	}

	dbInstance, conn, directHost, directPort, password, err := s.resolveDirectConnectionDetails(ctx, databaseID)
	if err != nil {
		return nil, nil, nil, "", false, mapQueryPreparationError(err)
	}
	if dbInstance.OrganizationID != organizationID {
		return nil, nil, nil, "", false, connect.NewError(connect.CodeNotFound, fmt.Errorf("database not found"))
	}

	if databaseName == "" {
		databaseName = conn.DatabaseName
	}

	timeout := 10 * time.Second
	if timeoutSeconds != nil && *timeoutSeconds > 0 {
		timeout = time.Duration(*timeoutSeconds) * time.Second
	}
	queryCtx, cancel := context.WithTimeout(ctx, timeout)

	db, err := openSQLQueryConnection(databasesv1.DatabaseType(dbInstance.Type), conn.Username, password, directHost, directPort, databaseName)
	if err != nil {
		cancel()
		if errors.Is(err, errUnsupportedQueryDatabaseType) {
			return nil, nil, nil, "", false, connect.NewError(connect.CodeUnimplemented, fmt.Errorf("query execution is only supported for PostgreSQL, MySQL, and MariaDB databases"))
		}
		return nil, nil, nil, "", false, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to connect: %w", err))
	}
	if err := db.PingContext(queryCtx); err != nil {
		cancel()
		db.Close()
		return nil, nil, nil, "", false, connect.NewError(connect.CodeUnavailable, fmt.Errorf("database unreachable at %s:%d: %w", directHost, directPort, err))
	}

	return queryCtx, cancel, db, queryType, readOnly, nil
}

func openSQLQueryConnection(dbType databasesv1.DatabaseType, username, password, host string, port int32, databaseName string) (*sql.DB, error) {
	switch dbType {
	case databasesv1.DatabaseType_POSTGRESQL:
		connStr := fmt.Sprintf("postgres://%s:%s@%s:%d/%s?sslmode=disable&connect_timeout=5",
			url.QueryEscape(username), url.QueryEscape(password), host, port, url.QueryEscape(databaseName))
		return sql.Open("postgres", connStr)
	case databasesv1.DatabaseType_MYSQL, databasesv1.DatabaseType_MARIADB:
		connStr := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?timeout=5s",
			url.QueryEscape(username), url.QueryEscape(password), host, port, url.QueryEscape(databaseName))
		return sql.Open("mysql", connStr)
	default:
		return nil, errUnsupportedQueryDatabaseType
	}
}

func buildQueryResultRow(columns []string, values []interface{}) *databasesv1.QueryResultRow {
	cells := make([]*databasesv1.QueryResultCell, len(columns))
	for i, val := range values {
		cell := &databasesv1.QueryResultCell{
			ColumnName: columns[i],
			IsNull:     val == nil,
		}
		if val != nil {
			cellValue := stringifyQueryValue(val)
			cell.Value = &cellValue
		}
		cells[i] = cell
	}

	return &databasesv1.QueryResultRow{
		Cells: cells,
	}
}

func stringifyQueryValue(val interface{}) string {
	switch v := val.(type) {
	case []byte:
		return string(v)
	default:
		return fmt.Sprintf("%v", val)
	}
}

func mapQueryPreparationError(err error) error {
	errText := err.Error()
	switch {
	case strings.Contains(errText, "database not found"):
		return connect.NewError(connect.CodeNotFound, err)
	case strings.Contains(errText, "database is stopped"):
		return connect.NewError(connect.CodeFailedPrecondition, err)
	case strings.Contains(errText, "failed to wake database"):
		return connect.NewError(connect.CodeUnavailable, err)
	case strings.Contains(errText, "failed to get connection info"):
		return connect.NewError(connect.CodeInternal, err)
	default:
		return connect.NewError(connect.CodeInternal, err)
	}
}
