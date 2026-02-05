package databases

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/obiente/cloud/apps/shared/pkg/auth"
	commonv1 "github.com/obiente/cloud/apps/shared/proto/obiente/cloud/common/v1"
	databasesv1 "github.com/obiente/cloud/apps/shared/proto/obiente/cloud/databases/v1"

	"connectrpc.com/connect"
)

// GetTableData retrieves paginated data from a table
func (s *Service) GetTableData(ctx context.Context, req *connect.Request[databasesv1.GetTableDataRequest]) (*connect.Response[databasesv1.GetTableDataResponse], error) {
	orgID := req.Msg.GetOrganizationId()
	if orgID == "" {
		if eff, ok := resolveUserDefaultOrgID(ctx); ok {
			orgID = eff
		}
	}

	if err := s.checkDatabasePermission(ctx, req.Msg.GetDatabaseId(), auth.PermissionDatabaseRead); err != nil {
		return nil, connect.NewError(connect.CodePermissionDenied, err)
	}

	dbInstance, err := s.repo.GetByID(ctx, req.Msg.GetDatabaseId())
	if err != nil {
		return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("database not found: %w", err))
	}
	if dbInstance.OrganizationID != orgID {
		return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("database not found"))
	}

	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	db, dbType, err := s.openDirectConnection(ctx, req.Msg.GetDatabaseId(), req.Msg.GetDatabaseName())
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	defer db.Close()

	tableName := quoteIdentifier(req.Msg.GetTableName(), databasesv1.DatabaseType(dbType))

	// Build WHERE clause from filters
	whereClause, whereArgs := buildWhereClause(req.Msg.GetFilters(), databasesv1.DatabaseType(dbType))

	// Get total count
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM %s", tableName)
	if whereClause != "" {
		countQuery += " WHERE " + whereClause
	}

	var totalRows int32
	if err := db.QueryRowContext(ctx, countQuery, whereArgs...).Scan(&totalRows); err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to count rows: %w", err))
	}

	// Build SELECT query
	page := req.Msg.GetPage()
	perPage := req.Msg.GetPerPage()
	if page < 1 {
		page = 1
	}
	if perPage < 1 || perPage > 500 {
		perPage = 50
	}

	offset := (page - 1) * perPage

	selectQuery := fmt.Sprintf("SELECT * FROM %s", tableName)
	if whereClause != "" {
		selectQuery += " WHERE " + whereClause
	}

	// Add ORDER BY
	if req.Msg.SortColumn != nil && *req.Msg.SortColumn != "" {
		sortCol := quoteIdentifier(*req.Msg.SortColumn, databasesv1.DatabaseType(dbType))
		sortDir := "ASC"
		if req.Msg.SortDirection != nil && strings.ToUpper(*req.Msg.SortDirection) == "DESC" {
			sortDir = "DESC"
		}
		selectQuery += fmt.Sprintf(" ORDER BY %s %s", sortCol, sortDir)
	}

	// Add LIMIT and OFFSET
	switch databasesv1.DatabaseType(dbType) {
	case databasesv1.DatabaseType_POSTGRESQL:
		selectQuery += fmt.Sprintf(" LIMIT %d OFFSET %d", perPage, offset)
	case databasesv1.DatabaseType_MYSQL, databasesv1.DatabaseType_MARIADB:
		selectQuery += fmt.Sprintf(" LIMIT %d, %d", offset, perPage)
	}

	rows, err := db.QueryContext(ctx, selectQuery, whereArgs...)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to query data: %w", err))
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
	var resultRows []*databasesv1.QueryResultRow
	for rows.Next() {
		values := make([]interface{}, len(columns))
		valuePtrs := make([]interface{}, len(columns))
		for i := range values {
			valuePtrs[i] = &values[i]
		}

		if err := rows.Scan(valuePtrs...); err != nil {
			return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to scan row: %w", err))
		}

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
	}

	if err := rows.Err(); err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("row iteration error: %w", err))
	}

	return connect.NewResponse(&databasesv1.GetTableDataResponse{
		Columns:   resultColumns,
		Rows:      resultRows,
		TotalRows: totalRows,
		Pagination: &commonv1.Pagination{
			Page:    page,
			PerPage: perPage,
			Total:   totalRows,
		},
	}), nil
}

// UpdateTableRow updates a single row in a table
func (s *Service) UpdateTableRow(ctx context.Context, req *connect.Request[databasesv1.UpdateTableRowRequest]) (*connect.Response[databasesv1.UpdateTableRowResponse], error) {
	orgID := req.Msg.GetOrganizationId()
	if orgID == "" {
		if eff, ok := resolveUserDefaultOrgID(ctx); ok {
			orgID = eff
		}
	}

	if err := s.checkDatabasePermission(ctx, req.Msg.GetDatabaseId(), auth.PermissionDatabaseManage); err != nil {
		return nil, connect.NewError(connect.CodePermissionDenied, err)
	}

	if len(req.Msg.GetWhereCells()) == 0 {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("where_cells is required to identify the row"))
	}

	if len(req.Msg.GetSetCells()) == 0 {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("set_cells is required"))
	}

	dbInstance, err := s.repo.GetByID(ctx, req.Msg.GetDatabaseId())
	if err != nil {
		return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("database not found: %w", err))
	}
	if dbInstance.OrganizationID != orgID {
		return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("database not found"))
	}

	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	db, dbType, err := s.openDirectConnection(ctx, req.Msg.GetDatabaseId(), req.Msg.GetDatabaseName())
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	defer db.Close()

	tableName := quoteIdentifier(req.Msg.GetTableName(), databasesv1.DatabaseType(dbType))

	// Build SET clause
	var setClauses []string
	var setArgs []interface{}
	argIndex := 1

	for _, cell := range req.Msg.GetSetCells() {
		colName := quoteIdentifier(cell.GetColumnName(), databasesv1.DatabaseType(dbType))
		if cell.GetIsNull() {
			setClauses = append(setClauses, fmt.Sprintf("%s = NULL", colName))
		} else {
			setClauses = append(setClauses, fmt.Sprintf("%s = %s", colName, placeholder(argIndex, databasesv1.DatabaseType(dbType))))
			setArgs = append(setArgs, cell.GetValue())
			argIndex++
		}
	}

	// Build WHERE clause
	var whereClauses []string
	var whereArgs []interface{}

	for _, cell := range req.Msg.GetWhereCells() {
		colName := quoteIdentifier(cell.GetColumnName(), databasesv1.DatabaseType(dbType))
		if cell.GetIsNull() {
			whereClauses = append(whereClauses, fmt.Sprintf("%s IS NULL", colName))
		} else {
			whereClauses = append(whereClauses, fmt.Sprintf("%s = %s", colName, placeholder(argIndex, databasesv1.DatabaseType(dbType))))
			whereArgs = append(whereArgs, cell.GetValue())
			argIndex++
		}
	}

	query := fmt.Sprintf("UPDATE %s SET %s WHERE %s",
		tableName,
		strings.Join(setClauses, ", "),
		strings.Join(whereClauses, " AND "))

	allArgs := append(setArgs, whereArgs...)
	result, err := db.ExecContext(ctx, query, allArgs...)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("failed to update row: %w", err))
	}

	affected, _ := result.RowsAffected()

	return connect.NewResponse(&databasesv1.UpdateTableRowResponse{
		AffectedRows: int32(affected),
	}), nil
}

// InsertTableRow inserts a new row into a table
func (s *Service) InsertTableRow(ctx context.Context, req *connect.Request[databasesv1.InsertTableRowRequest]) (*connect.Response[databasesv1.InsertTableRowResponse], error) {
	orgID := req.Msg.GetOrganizationId()
	if orgID == "" {
		if eff, ok := resolveUserDefaultOrgID(ctx); ok {
			orgID = eff
		}
	}

	if err := s.checkDatabasePermission(ctx, req.Msg.GetDatabaseId(), auth.PermissionDatabaseManage); err != nil {
		return nil, connect.NewError(connect.CodePermissionDenied, err)
	}

	if len(req.Msg.GetCells()) == 0 {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("cells is required"))
	}

	dbInstance, err := s.repo.GetByID(ctx, req.Msg.GetDatabaseId())
	if err != nil {
		return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("database not found: %w", err))
	}
	if dbInstance.OrganizationID != orgID {
		return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("database not found"))
	}

	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	db, dbType, err := s.openDirectConnection(ctx, req.Msg.GetDatabaseId(), req.Msg.GetDatabaseName())
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	defer db.Close()

	tableName := quoteIdentifier(req.Msg.GetTableName(), databasesv1.DatabaseType(dbType))

	var columns []string
	var placeholders []string
	var args []interface{}
	argIndex := 1

	for _, cell := range req.Msg.GetCells() {
		columns = append(columns, quoteIdentifier(cell.GetColumnName(), databasesv1.DatabaseType(dbType)))
		if cell.GetIsNull() {
			placeholders = append(placeholders, "NULL")
		} else {
			placeholders = append(placeholders, placeholder(argIndex, databasesv1.DatabaseType(dbType)))
			args = append(args, cell.GetValue())
			argIndex++
		}
	}

	query := fmt.Sprintf("INSERT INTO %s (%s) VALUES (%s)",
		tableName,
		strings.Join(columns, ", "),
		strings.Join(placeholders, ", "))

	result, err := db.ExecContext(ctx, query, args...)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("failed to insert row: %w", err))
	}

	affected, _ := result.RowsAffected()

	return connect.NewResponse(&databasesv1.InsertTableRowResponse{
		AffectedRows: int32(affected),
	}), nil
}

// DeleteTableRows deletes rows from a table
func (s *Service) DeleteTableRows(ctx context.Context, req *connect.Request[databasesv1.DeleteTableRowsRequest]) (*connect.Response[databasesv1.DeleteTableRowsResponse], error) {
	orgID := req.Msg.GetOrganizationId()
	if orgID == "" {
		if eff, ok := resolveUserDefaultOrgID(ctx); ok {
			orgID = eff
		}
	}

	if err := s.checkDatabasePermission(ctx, req.Msg.GetDatabaseId(), auth.PermissionDatabaseManage); err != nil {
		return nil, connect.NewError(connect.CodePermissionDenied, err)
	}

	if len(req.Msg.GetWhereCells()) == 0 {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("where_cells is required to identify the row(s) to delete"))
	}

	dbInstance, err := s.repo.GetByID(ctx, req.Msg.GetDatabaseId())
	if err != nil {
		return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("database not found: %w", err))
	}
	if dbInstance.OrganizationID != orgID {
		return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("database not found"))
	}

	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	db, dbType, err := s.openDirectConnection(ctx, req.Msg.GetDatabaseId(), req.Msg.GetDatabaseName())
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	defer db.Close()

	tableName := quoteIdentifier(req.Msg.GetTableName(), databasesv1.DatabaseType(dbType))

	// Build WHERE clause
	var whereClauses []string
	var args []interface{}
	argIndex := 1

	for _, cell := range req.Msg.GetWhereCells() {
		colName := quoteIdentifier(cell.GetColumnName(), databasesv1.DatabaseType(dbType))
		if cell.GetIsNull() {
			whereClauses = append(whereClauses, fmt.Sprintf("%s IS NULL", colName))
		} else {
			whereClauses = append(whereClauses, fmt.Sprintf("%s = %s", colName, placeholder(argIndex, databasesv1.DatabaseType(dbType))))
			args = append(args, cell.GetValue())
			argIndex++
		}
	}

	query := fmt.Sprintf("DELETE FROM %s WHERE %s",
		tableName,
		strings.Join(whereClauses, " AND "))

	result, err := db.ExecContext(ctx, query, args...)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("failed to delete row(s): %w", err))
	}

	affected, _ := result.RowsAffected()

	return connect.NewResponse(&databasesv1.DeleteTableRowsResponse{
		AffectedRows: int32(affected),
	}), nil
}

// Helper functions

func placeholder(index int, dbType databasesv1.DatabaseType) string {
	switch dbType {
	case databasesv1.DatabaseType_POSTGRESQL:
		return fmt.Sprintf("$%d", index)
	default:
		return "?"
	}
}

func buildWhereClause(filters []*databasesv1.ColumnFilter, dbType databasesv1.DatabaseType) (string, []interface{}) {
	if len(filters) == 0 {
		return "", nil
	}

	var clauses []string
	var args []interface{}
	argIndex := 1

	for _, f := range filters {
		colName := quoteIdentifier(f.GetColumnName(), dbType)
		op := strings.ToUpper(f.GetOperator())

		switch op {
		case "IS NULL":
			clauses = append(clauses, fmt.Sprintf("%s IS NULL", colName))
		case "IS NOT NULL":
			clauses = append(clauses, fmt.Sprintf("%s IS NOT NULL", colName))
		case "=", "!=", "<>", ">", ">=", "<", "<=":
			clauses = append(clauses, fmt.Sprintf("%s %s %s", colName, op, placeholder(argIndex, dbType)))
			args = append(args, f.GetValue())
			argIndex++
		case "LIKE", "ILIKE":
			if dbType == databasesv1.DatabaseType_POSTGRESQL && op == "ILIKE" {
				clauses = append(clauses, fmt.Sprintf("%s ILIKE %s", colName, placeholder(argIndex, dbType)))
			} else {
				clauses = append(clauses, fmt.Sprintf("%s LIKE %s", colName, placeholder(argIndex, dbType)))
			}
			args = append(args, f.GetValue())
			argIndex++
		case "IN":
			// For IN, expect comma-separated values
			values := strings.Split(f.GetValue(), ",")
			var placeholders []string
			for _, v := range values {
				placeholders = append(placeholders, placeholder(argIndex, dbType))
				args = append(args, strings.TrimSpace(v))
				argIndex++
			}
			clauses = append(clauses, fmt.Sprintf("%s IN (%s)", colName, strings.Join(placeholders, ", ")))
		default:
			// Default to equality
			clauses = append(clauses, fmt.Sprintf("%s = %s", colName, placeholder(argIndex, dbType)))
			args = append(args, f.GetValue())
			argIndex++
		}
	}

	return strings.Join(clauses, " AND "), args
}
