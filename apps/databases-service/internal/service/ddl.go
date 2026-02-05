package databases

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/obiente/cloud/apps/shared/pkg/auth"
	databasesv1 "github.com/obiente/cloud/apps/shared/proto/obiente/cloud/databases/v1"

	"connectrpc.com/connect"
)

// CreateTable creates a new table in the database
func (s *Service) CreateTable(ctx context.Context, req *connect.Request[databasesv1.CreateTableRequest]) (*connect.Response[databasesv1.CreateTableResponse], error) {
	orgID := req.Msg.GetOrganizationId()
	if orgID == "" {
		if eff, ok := resolveUserDefaultOrgID(ctx); ok {
			orgID = eff
		}
	}

	if err := s.checkDatabasePermission(ctx, req.Msg.GetDatabaseId(), auth.PermissionDatabaseManage); err != nil {
		return nil, connect.NewError(connect.CodePermissionDenied, err)
	}

	dbInstance, err := s.repo.GetByID(ctx, req.Msg.GetDatabaseId())
	if err != nil {
		return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("database not found: %w", err))
	}
	if dbInstance.OrganizationID != orgID {
		return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("database not found"))
	}

	ctx, cancel := context.WithTimeout(ctx, 60*time.Second)
	defer cancel()

	db, dbType, err := s.openDirectConnection(ctx, req.Msg.GetDatabaseId(), req.Msg.GetDatabaseName())
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	defer db.Close()

	// Build CREATE TABLE statement
	ddl := buildCreateTableSQL(req.Msg, databasesv1.DatabaseType(dbType))

	// Execute DDL
	if _, err := db.ExecContext(ctx, ddl); err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("failed to create table: %w", err))
	}

	// Get the newly created table info
	conn, _ := s.connRepo.GetByDatabaseID(ctx, req.Msg.GetDatabaseId())
	dbName := req.Msg.GetDatabaseName()
	if dbName == "" && conn != nil {
		dbName = conn.DatabaseName
	}

	table, err := s.introspectSingleTable(ctx, db, databasesv1.DatabaseType(dbType), dbName, req.Msg.GetTableName())
	if err != nil {
		// Table was created but we couldn't get its info
		return connect.NewResponse(&databasesv1.CreateTableResponse{
			Success: true,
		}), nil
	}

	return connect.NewResponse(&databasesv1.CreateTableResponse{
		Success: true,
		Table:   table,
	}), nil
}

// AlterTable modifies an existing table
func (s *Service) AlterTable(ctx context.Context, req *connect.Request[databasesv1.AlterTableRequest]) (*connect.Response[databasesv1.AlterTableResponse], error) {
	orgID := req.Msg.GetOrganizationId()
	if orgID == "" {
		if eff, ok := resolveUserDefaultOrgID(ctx); ok {
			orgID = eff
		}
	}

	if err := s.checkDatabasePermission(ctx, req.Msg.GetDatabaseId(), auth.PermissionDatabaseManage); err != nil {
		return nil, connect.NewError(connect.CodePermissionDenied, err)
	}

	dbInstance, err := s.repo.GetByID(ctx, req.Msg.GetDatabaseId())
	if err != nil {
		return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("database not found: %w", err))
	}
	if dbInstance.OrganizationID != orgID {
		return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("database not found"))
	}

	ctx, cancel := context.WithTimeout(ctx, 60*time.Second)
	defer cancel()

	db, dbType, err := s.openDirectConnection(ctx, req.Msg.GetDatabaseId(), req.Msg.GetDatabaseName())
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	defer db.Close()

	// Execute each operation
	for _, op := range req.Msg.GetOperations() {
		ddl := buildAlterTableSQL(req.Msg.GetTableName(), op, databasesv1.DatabaseType(dbType))
		if ddl == "" {
			continue
		}
		if _, err := db.ExecContext(ctx, ddl); err != nil {
			return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("failed to alter table: %w", err))
		}
	}

	// Get updated table info
	conn, _ := s.connRepo.GetByDatabaseID(ctx, req.Msg.GetDatabaseId())
	dbName := req.Msg.GetDatabaseName()
	if dbName == "" && conn != nil {
		dbName = conn.DatabaseName
	}

	table, err := s.introspectSingleTable(ctx, db, databasesv1.DatabaseType(dbType), dbName, req.Msg.GetTableName())
	if err != nil {
		return connect.NewResponse(&databasesv1.AlterTableResponse{
			Success: true,
		}), nil
	}

	return connect.NewResponse(&databasesv1.AlterTableResponse{
		Success: true,
		Table:   table,
	}), nil
}

// DropTable drops a table from the database
func (s *Service) DropTable(ctx context.Context, req *connect.Request[databasesv1.DropTableRequest]) (*connect.Response[databasesv1.DropTableResponse], error) {
	orgID := req.Msg.GetOrganizationId()
	if orgID == "" {
		if eff, ok := resolveUserDefaultOrgID(ctx); ok {
			orgID = eff
		}
	}

	if err := s.checkDatabasePermission(ctx, req.Msg.GetDatabaseId(), auth.PermissionDatabaseManage); err != nil {
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

	var ddl string
	tableName := quoteIdentifier(req.Msg.GetTableName(), databasesv1.DatabaseType(dbType))

	if req.Msg.GetIfExists() {
		ddl = fmt.Sprintf("DROP TABLE IF EXISTS %s", tableName)
	} else {
		ddl = fmt.Sprintf("DROP TABLE %s", tableName)
	}

	if req.Msg.GetCascade() && databasesv1.DatabaseType(dbType) == databasesv1.DatabaseType_POSTGRESQL {
		ddl += " CASCADE"
	}

	if _, err := db.ExecContext(ctx, ddl); err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("failed to drop table: %w", err))
	}

	return connect.NewResponse(&databasesv1.DropTableResponse{
		Success: true,
	}), nil
}

// RenameTable renames a table
func (s *Service) RenameTable(ctx context.Context, req *connect.Request[databasesv1.RenameTableRequest]) (*connect.Response[databasesv1.RenameTableResponse], error) {
	orgID := req.Msg.GetOrganizationId()
	if orgID == "" {
		if eff, ok := resolveUserDefaultOrgID(ctx); ok {
			orgID = eff
		}
	}

	if err := s.checkDatabasePermission(ctx, req.Msg.GetDatabaseId(), auth.PermissionDatabaseManage); err != nil {
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

	oldName := quoteIdentifier(req.Msg.GetOldName(), databasesv1.DatabaseType(dbType))
	newName := quoteIdentifier(req.Msg.GetNewName(), databasesv1.DatabaseType(dbType))

	var ddl string
	switch databasesv1.DatabaseType(dbType) {
	case databasesv1.DatabaseType_POSTGRESQL:
		ddl = fmt.Sprintf("ALTER TABLE %s RENAME TO %s", oldName, newName)
	case databasesv1.DatabaseType_MYSQL, databasesv1.DatabaseType_MARIADB:
		ddl = fmt.Sprintf("RENAME TABLE %s TO %s", oldName, newName)
	default:
		return nil, connect.NewError(connect.CodeUnimplemented, fmt.Errorf("rename not supported for this database type"))
	}

	if _, err := db.ExecContext(ctx, ddl); err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("failed to rename table: %w", err))
	}

	return connect.NewResponse(&databasesv1.RenameTableResponse{
		Success: true,
	}), nil
}

// TruncateTable truncates a table
func (s *Service) TruncateTable(ctx context.Context, req *connect.Request[databasesv1.TruncateTableRequest]) (*connect.Response[databasesv1.TruncateTableResponse], error) {
	orgID := req.Msg.GetOrganizationId()
	if orgID == "" {
		if eff, ok := resolveUserDefaultOrgID(ctx); ok {
			orgID = eff
		}
	}

	if err := s.checkDatabasePermission(ctx, req.Msg.GetDatabaseId(), auth.PermissionDatabaseManage); err != nil {
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

	// Get approximate row count before truncate
	var rowCount int64
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM %s", tableName)
	_ = db.QueryRowContext(ctx, countQuery).Scan(&rowCount)

	ddl := fmt.Sprintf("TRUNCATE TABLE %s", tableName)
	if req.Msg.GetCascade() && databasesv1.DatabaseType(dbType) == databasesv1.DatabaseType_POSTGRESQL {
		ddl += " CASCADE"
	}

	if _, err := db.ExecContext(ctx, ddl); err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("failed to truncate table: %w", err))
	}

	return connect.NewResponse(&databasesv1.TruncateTableResponse{
		Success:     true,
		RowsDeleted: rowCount,
	}), nil
}

// CreateIndex creates an index on a table
func (s *Service) CreateIndex(ctx context.Context, req *connect.Request[databasesv1.CreateIndexRequest]) (*connect.Response[databasesv1.CreateIndexResponse], error) {
	orgID := req.Msg.GetOrganizationId()
	if orgID == "" {
		if eff, ok := resolveUserDefaultOrgID(ctx); ok {
			orgID = eff
		}
	}

	if err := s.checkDatabasePermission(ctx, req.Msg.GetDatabaseId(), auth.PermissionDatabaseManage); err != nil {
		return nil, connect.NewError(connect.CodePermissionDenied, err)
	}

	dbInstance, err := s.repo.GetByID(ctx, req.Msg.GetDatabaseId())
	if err != nil {
		return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("database not found: %w", err))
	}
	if dbInstance.OrganizationID != orgID {
		return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("database not found"))
	}

	ctx, cancel := context.WithTimeout(ctx, 120*time.Second) // Indexes can take time
	defer cancel()

	db, dbType, err := s.openDirectConnection(ctx, req.Msg.GetDatabaseId(), req.Msg.GetDatabaseName())
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	defer db.Close()

	idx := req.Msg.GetIndex()
	tableName := quoteIdentifier(req.Msg.GetTableName(), databasesv1.DatabaseType(dbType))
	indexName := quoteIdentifier(idx.GetName(), databasesv1.DatabaseType(dbType))

	var columns []string
	for _, col := range idx.GetColumnNames() {
		columns = append(columns, quoteIdentifier(col, databasesv1.DatabaseType(dbType)))
	}

	var ddl strings.Builder
	ddl.WriteString("CREATE ")
	if idx.GetIsUnique() {
		ddl.WriteString("UNIQUE ")
	}
	ddl.WriteString("INDEX ")
	if req.Msg.GetConcurrently() && databasesv1.DatabaseType(dbType) == databasesv1.DatabaseType_POSTGRESQL {
		ddl.WriteString("CONCURRENTLY ")
	}
	if req.Msg.GetIfNotExists() {
		ddl.WriteString("IF NOT EXISTS ")
	}
	ddl.WriteString(indexName)
	ddl.WriteString(" ON ")
	ddl.WriteString(tableName)

	// Index type for PostgreSQL
	if idx.Type != nil && databasesv1.DatabaseType(dbType) == databasesv1.DatabaseType_POSTGRESQL {
		ddl.WriteString(fmt.Sprintf(" USING %s", *idx.Type))
	}

	ddl.WriteString(" (")
	ddl.WriteString(strings.Join(columns, ", "))
	ddl.WriteString(")")

	if _, err := db.ExecContext(ctx, ddl.String()); err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("failed to create index: %w", err))
	}

	return connect.NewResponse(&databasesv1.CreateIndexResponse{
		Success: true,
	}), nil
}

// DropIndex drops an index
func (s *Service) DropIndex(ctx context.Context, req *connect.Request[databasesv1.DropIndexRequest]) (*connect.Response[databasesv1.DropIndexResponse], error) {
	orgID := req.Msg.GetOrganizationId()
	if orgID == "" {
		if eff, ok := resolveUserDefaultOrgID(ctx); ok {
			orgID = eff
		}
	}

	if err := s.checkDatabasePermission(ctx, req.Msg.GetDatabaseId(), auth.PermissionDatabaseManage); err != nil {
		return nil, connect.NewError(connect.CodePermissionDenied, err)
	}

	dbInstance, err := s.repo.GetByID(ctx, req.Msg.GetDatabaseId())
	if err != nil {
		return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("database not found: %w", err))
	}
	if dbInstance.OrganizationID != orgID {
		return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("database not found"))
	}

	ctx, cancel := context.WithTimeout(ctx, 60*time.Second)
	defer cancel()

	db, dbType, err := s.openDirectConnection(ctx, req.Msg.GetDatabaseId(), req.Msg.GetDatabaseName())
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	defer db.Close()

	indexName := quoteIdentifier(req.Msg.GetIndexName(), databasesv1.DatabaseType(dbType))

	var ddl strings.Builder
	ddl.WriteString("DROP INDEX ")
	if req.Msg.GetConcurrently() && databasesv1.DatabaseType(dbType) == databasesv1.DatabaseType_POSTGRESQL {
		ddl.WriteString("CONCURRENTLY ")
	}
	if req.Msg.GetIfExists() {
		ddl.WriteString("IF EXISTS ")
	}
	ddl.WriteString(indexName)
	if req.Msg.GetCascade() && databasesv1.DatabaseType(dbType) == databasesv1.DatabaseType_POSTGRESQL {
		ddl.WriteString(" CASCADE")
	}

	if _, err := db.ExecContext(ctx, ddl.String()); err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("failed to drop index: %w", err))
	}

	return connect.NewResponse(&databasesv1.DropIndexResponse{
		Success: true,
	}), nil
}

// GetTableDDL gets the DDL statement for a table
func (s *Service) GetTableDDL(ctx context.Context, req *connect.Request[databasesv1.GetTableDDLRequest]) (*connect.Response[databasesv1.GetTableDDLResponse], error) {
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

	ddl, err := getTableDDL(ctx, db, databasesv1.DatabaseType(dbType), req.Msg.GetTableName())
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to get table DDL: %w", err))
	}

	return connect.NewResponse(&databasesv1.GetTableDDLResponse{
		Ddl: ddl,
	}), nil
}

// Helper functions

func quoteIdentifier(name string, dbType databasesv1.DatabaseType) string {
	// Escape any existing quotes
	switch dbType {
	case databasesv1.DatabaseType_POSTGRESQL:
		escaped := strings.ReplaceAll(name, `"`, `""`)
		return fmt.Sprintf(`"%s"`, escaped)
	case databasesv1.DatabaseType_MYSQL, databasesv1.DatabaseType_MARIADB:
		escaped := strings.ReplaceAll(name, "`", "``")
		return fmt.Sprintf("`%s`", escaped)
	default:
		return name
	}
}

func buildCreateTableSQL(req *databasesv1.CreateTableRequest, dbType databasesv1.DatabaseType) string {
	var sb strings.Builder
	tableName := quoteIdentifier(req.GetTableName(), dbType)

	sb.WriteString("CREATE TABLE ")
	sb.WriteString(tableName)
	sb.WriteString(" (\n")

	var columnDefs []string
	var pkColumns []string

	for _, col := range req.GetColumns() {
		colDef := buildColumnDefinition(col, dbType)
		columnDefs = append(columnDefs, "  "+colDef)
		if col.GetAutoIncrement() && dbType == databasesv1.DatabaseType_POSTGRESQL {
			// For PostgreSQL, auto_increment columns become part of primary key
		}
	}

	// Primary key
	if pk := req.GetPrimaryKey(); pk != nil && len(pk.GetColumnNames()) > 0 {
		pkColumns = pk.GetColumnNames()
	} else {
		// Auto-detect PK from column definitions with auto_increment
		for _, col := range req.GetColumns() {
			if col.GetAutoIncrement() {
				pkColumns = append(pkColumns, col.GetName())
			}
		}
	}

	if len(pkColumns) > 0 {
		var quotedPKCols []string
		for _, col := range pkColumns {
			quotedPKCols = append(quotedPKCols, quoteIdentifier(col, dbType))
		}
		pkDef := fmt.Sprintf("  PRIMARY KEY (%s)", strings.Join(quotedPKCols, ", "))
		columnDefs = append(columnDefs, pkDef)
	}

	// Foreign keys
	for _, fk := range req.GetForeignKeys() {
		fkDef := buildForeignKeyConstraint(fk, dbType)
		columnDefs = append(columnDefs, "  "+fkDef)
	}

	sb.WriteString(strings.Join(columnDefs, ",\n"))
	sb.WriteString("\n)")

	// Table comment for PostgreSQL (MySQL handles it differently)
	if req.Comment != nil && dbType == databasesv1.DatabaseType_MYSQL {
		sb.WriteString(fmt.Sprintf(" COMMENT '%s'", escapeString(*req.Comment)))
	}

	return sb.String()
}

func buildColumnDefinition(col *databasesv1.ColumnDefinition, dbType databasesv1.DatabaseType) string {
	var parts []string
	parts = append(parts, quoteIdentifier(col.GetName(), dbType))

	dataType := col.GetDataType()
	if col.GetAutoIncrement() {
		switch dbType {
		case databasesv1.DatabaseType_POSTGRESQL:
			// Use SERIAL types for PostgreSQL
			switch strings.ToLower(dataType) {
			case "integer", "int", "int4":
				dataType = "SERIAL"
			case "bigint", "int8":
				dataType = "BIGSERIAL"
			case "smallint", "int2":
				dataType = "SMALLSERIAL"
			default:
				dataType = "SERIAL"
			}
		case databasesv1.DatabaseType_MYSQL, databasesv1.DatabaseType_MARIADB:
			parts = append(parts, dataType)
			parts = append(parts, "AUTO_INCREMENT")
			goto afterType
		}
	}
	parts = append(parts, dataType)
afterType:

	if !col.GetIsNullable() && !col.GetAutoIncrement() {
		parts = append(parts, "NOT NULL")
	}

	if col.GetIsUnique() {
		parts = append(parts, "UNIQUE")
	}

	if col.DefaultValue != nil {
		parts = append(parts, fmt.Sprintf("DEFAULT %s", *col.DefaultValue))
	}

	return strings.Join(parts, " ")
}

func buildForeignKeyConstraint(fk *databasesv1.ForeignKeyDefinition, dbType databasesv1.DatabaseType) string {
	var fromCols, toCols []string
	for _, c := range fk.GetFromColumns() {
		fromCols = append(fromCols, quoteIdentifier(c, dbType))
	}
	for _, c := range fk.GetToColumns() {
		toCols = append(toCols, quoteIdentifier(c, dbType))
	}

	var sb strings.Builder
	if fk.GetName() != "" {
		sb.WriteString(fmt.Sprintf("CONSTRAINT %s ", quoteIdentifier(fk.GetName(), dbType)))
	}
	sb.WriteString(fmt.Sprintf("FOREIGN KEY (%s) REFERENCES %s (%s)",
		strings.Join(fromCols, ", "),
		quoteIdentifier(fk.GetToTable(), dbType),
		strings.Join(toCols, ", ")))

	if fk.GetOnDelete() != "" {
		sb.WriteString(fmt.Sprintf(" ON DELETE %s", fk.GetOnDelete()))
	}
	if fk.GetOnUpdate() != "" {
		sb.WriteString(fmt.Sprintf(" ON UPDATE %s", fk.GetOnUpdate()))
	}

	return sb.String()
}

func buildAlterTableSQL(tableName string, op *databasesv1.AlterTableOperation, dbType databasesv1.DatabaseType) string {
	quotedTable := quoteIdentifier(tableName, dbType)

	switch o := op.GetOperation().(type) {
	case *databasesv1.AlterTableOperation_AddColumn:
		colDef := buildColumnDefinition(o.AddColumn.GetColumn(), dbType)
		ddl := fmt.Sprintf("ALTER TABLE %s ADD COLUMN %s", quotedTable, colDef)
		if o.AddColumn.AfterColumn != nil && (dbType == databasesv1.DatabaseType_MYSQL || dbType == databasesv1.DatabaseType_MARIADB) {
			ddl += fmt.Sprintf(" AFTER %s", quoteIdentifier(*o.AddColumn.AfterColumn, dbType))
		}
		return ddl

	case *databasesv1.AlterTableOperation_DropColumn:
		ddl := fmt.Sprintf("ALTER TABLE %s DROP COLUMN %s", quotedTable, quoteIdentifier(o.DropColumn.GetColumnName(), dbType))
		if o.DropColumn.GetCascade() && dbType == databasesv1.DatabaseType_POSTGRESQL {
			ddl += " CASCADE"
		}
		return ddl

	case *databasesv1.AlterTableOperation_ModifyColumn:
		return buildModifyColumnSQL(tableName, o.ModifyColumn, dbType)

	case *databasesv1.AlterTableOperation_RenameColumn:
		oldCol := quoteIdentifier(o.RenameColumn.GetOldName(), dbType)
		newCol := quoteIdentifier(o.RenameColumn.GetNewName(), dbType)
		switch dbType {
		case databasesv1.DatabaseType_POSTGRESQL:
			return fmt.Sprintf("ALTER TABLE %s RENAME COLUMN %s TO %s", quotedTable, oldCol, newCol)
		case databasesv1.DatabaseType_MYSQL, databasesv1.DatabaseType_MARIADB:
			// MySQL requires full column definition for CHANGE, but RENAME COLUMN works in newer versions
			return fmt.Sprintf("ALTER TABLE %s RENAME COLUMN %s TO %s", quotedTable, oldCol, newCol)
		}

	case *databasesv1.AlterTableOperation_AddForeignKey:
		fkDef := buildForeignKeyConstraint(o.AddForeignKey.GetForeignKey(), dbType)
		return fmt.Sprintf("ALTER TABLE %s ADD %s", quotedTable, fkDef)

	case *databasesv1.AlterTableOperation_DropForeignKey:
		constraintName := quoteIdentifier(o.DropForeignKey.GetConstraintName(), dbType)
		switch dbType {
		case databasesv1.DatabaseType_POSTGRESQL:
			return fmt.Sprintf("ALTER TABLE %s DROP CONSTRAINT %s", quotedTable, constraintName)
		case databasesv1.DatabaseType_MYSQL, databasesv1.DatabaseType_MARIADB:
			return fmt.Sprintf("ALTER TABLE %s DROP FOREIGN KEY %s", quotedTable, constraintName)
		}

	case *databasesv1.AlterTableOperation_AddUnique:
		var cols []string
		for _, c := range o.AddUnique.GetColumnNames() {
			cols = append(cols, quoteIdentifier(c, dbType))
		}
		constraintName := ""
		if o.AddUnique.GetName() != "" {
			constraintName = fmt.Sprintf("CONSTRAINT %s ", quoteIdentifier(o.AddUnique.GetName(), dbType))
		}
		return fmt.Sprintf("ALTER TABLE %s ADD %sUNIQUE (%s)", quotedTable, constraintName, strings.Join(cols, ", "))

	case *databasesv1.AlterTableOperation_DropConstraint:
		constraintName := quoteIdentifier(o.DropConstraint.GetConstraintName(), dbType)
		switch dbType {
		case databasesv1.DatabaseType_POSTGRESQL:
			return fmt.Sprintf("ALTER TABLE %s DROP CONSTRAINT %s", quotedTable, constraintName)
		case databasesv1.DatabaseType_MYSQL, databasesv1.DatabaseType_MARIADB:
			// MySQL doesn't have a generic DROP CONSTRAINT, need to know the type
			return fmt.Sprintf("ALTER TABLE %s DROP INDEX %s", quotedTable, constraintName)
		}
	}

	return ""
}

func buildModifyColumnSQL(tableName string, mod *databasesv1.ModifyColumnOperation, dbType databasesv1.DatabaseType) string {
	quotedTable := quoteIdentifier(tableName, dbType)
	quotedCol := quoteIdentifier(mod.GetColumnName(), dbType)

	var statements []string

	switch dbType {
	case databasesv1.DatabaseType_POSTGRESQL:
		if mod.NewDataType != nil {
			statements = append(statements, fmt.Sprintf("ALTER TABLE %s ALTER COLUMN %s TYPE %s", quotedTable, quotedCol, *mod.NewDataType))
		}
		if mod.IsNullable != nil {
			if *mod.IsNullable {
				statements = append(statements, fmt.Sprintf("ALTER TABLE %s ALTER COLUMN %s DROP NOT NULL", quotedTable, quotedCol))
			} else {
				statements = append(statements, fmt.Sprintf("ALTER TABLE %s ALTER COLUMN %s SET NOT NULL", quotedTable, quotedCol))
			}
		}
		if mod.GetDropDefault() {
			statements = append(statements, fmt.Sprintf("ALTER TABLE %s ALTER COLUMN %s DROP DEFAULT", quotedTable, quotedCol))
		} else if mod.DefaultValue != nil {
			statements = append(statements, fmt.Sprintf("ALTER TABLE %s ALTER COLUMN %s SET DEFAULT %s", quotedTable, quotedCol, *mod.DefaultValue))
		}

	case databasesv1.DatabaseType_MYSQL, databasesv1.DatabaseType_MARIADB:
		// MySQL MODIFY COLUMN requires full column definition
		// For simplicity, we'll do what we can with ALTER
		if mod.NewDataType != nil || mod.IsNullable != nil {
			// Build MODIFY statement
			var parts []string
			parts = append(parts, quotedCol)
			if mod.NewDataType != nil {
				parts = append(parts, *mod.NewDataType)
			} else {
				parts = append(parts, "/* existing type */") // This is a limitation
			}
			if mod.IsNullable != nil && !*mod.IsNullable {
				parts = append(parts, "NOT NULL")
			}
			if mod.DefaultValue != nil {
				parts = append(parts, fmt.Sprintf("DEFAULT %s", *mod.DefaultValue))
			}
			statements = append(statements, fmt.Sprintf("ALTER TABLE %s MODIFY COLUMN %s", quotedTable, strings.Join(parts, " ")))
		}
	}

	// Return first statement; multiple modifications need multiple calls
	if len(statements) > 0 {
		return statements[0]
	}
	return ""
}

func getTableDDL(ctx context.Context, db *sql.DB, dbType databasesv1.DatabaseType, tableName string) (string, error) {
	switch dbType {
	case databasesv1.DatabaseType_POSTGRESQL:
		return getTableDDLPostgres(ctx, db, tableName)
	case databasesv1.DatabaseType_MYSQL, databasesv1.DatabaseType_MARIADB:
		return getTableDDLMySQL(ctx, db, tableName)
	default:
		return "", fmt.Errorf("unsupported database type")
	}
}

func getTableDDLPostgres(ctx context.Context, db *sql.DB, tableName string) (string, error) {
	// PostgreSQL doesn't have SHOW CREATE TABLE, so we reconstruct it
	// This is a simplified version - a full implementation would use pg_dump or a more complex query

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("CREATE TABLE %s (\n", quoteIdentifier(tableName, databasesv1.DatabaseType_POSTGRESQL)))

	// Get columns
	rows, err := db.QueryContext(ctx, `
		SELECT column_name, data_type, character_maximum_length, is_nullable, column_default
		FROM information_schema.columns
		WHERE table_name = $1 AND table_schema NOT IN ('pg_catalog', 'information_schema')
		ORDER BY ordinal_position
	`, tableName)
	if err != nil {
		return "", err
	}
	defer rows.Close()

	var columns []string
	for rows.Next() {
		var colName, dataType, isNullable string
		var maxLen *int
		var defaultVal *string
		if err := rows.Scan(&colName, &dataType, &maxLen, &isNullable, &defaultVal); err != nil {
			return "", err
		}

		colDef := fmt.Sprintf("  %s %s", quoteIdentifier(colName, databasesv1.DatabaseType_POSTGRESQL), dataType)
		if maxLen != nil && *maxLen > 0 {
			colDef = fmt.Sprintf("  %s %s(%d)", quoteIdentifier(colName, databasesv1.DatabaseType_POSTGRESQL), dataType, *maxLen)
		}
		if isNullable == "NO" {
			colDef += " NOT NULL"
		}
		if defaultVal != nil {
			colDef += fmt.Sprintf(" DEFAULT %s", *defaultVal)
		}
		columns = append(columns, colDef)
	}

	// Get primary key
	pkRows, err := db.QueryContext(ctx, `
		SELECT kcu.column_name
		FROM information_schema.table_constraints tc
		JOIN information_schema.key_column_usage kcu ON tc.constraint_name = kcu.constraint_name
		WHERE tc.table_name = $1 AND tc.constraint_type = 'PRIMARY KEY'
		ORDER BY kcu.ordinal_position
	`, tableName)
	if err == nil {
		defer pkRows.Close()
		var pkCols []string
		for pkRows.Next() {
			var col string
			if err := pkRows.Scan(&col); err == nil {
				pkCols = append(pkCols, quoteIdentifier(col, databasesv1.DatabaseType_POSTGRESQL))
			}
		}
		if len(pkCols) > 0 {
			columns = append(columns, fmt.Sprintf("  PRIMARY KEY (%s)", strings.Join(pkCols, ", ")))
		}
	}

	sb.WriteString(strings.Join(columns, ",\n"))
	sb.WriteString("\n);")

	return sb.String(), nil
}

func getTableDDLMySQL(ctx context.Context, db *sql.DB, tableName string) (string, error) {
	var name, ddl string
	err := db.QueryRowContext(ctx, "SHOW CREATE TABLE "+quoteIdentifier(tableName, databasesv1.DatabaseType_MYSQL)).Scan(&name, &ddl)
	if err != nil {
		return "", err
	}
	return ddl, nil
}

func escapeString(s string) string {
	return strings.ReplaceAll(s, "'", "''")
}
