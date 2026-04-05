package databases

import (
	"context"
	"database/sql"
	"fmt"
	"net/url"
	"time"

	"github.com/obiente/cloud/apps/shared/pkg/auth"
	"github.com/obiente/cloud/apps/shared/pkg/database"
	commonv1 "github.com/obiente/cloud/apps/shared/proto/obiente/cloud/common/v1"
	databasesv1 "github.com/obiente/cloud/apps/shared/proto/obiente/cloud/databases/v1"

	"connectrpc.com/connect"
)

func (s *Service) resolveDirectConnectionDetails(ctx context.Context, databaseID string) (*database.DatabaseInstance, *database.DatabaseConnection, string, int32, string, error) {
	dbInstance, err := s.repo.GetByID(ctx, databaseID)
	if err != nil {
		return nil, nil, "", 0, "", fmt.Errorf("database not found: %w", err)
	}

	conn, err := s.connRepo.GetByDatabaseID(ctx, databaseID)
	if err != nil {
		return nil, nil, "", 0, "", fmt.Errorf("failed to get connection info: %w", err)
	}

	directHost := fmt.Sprintf("obiente-%s", databaseID)
	directPort := conn.Port
	if s.routeRegistry != nil {
		if route, ok := s.routeRegistry.LookupByID(databaseID); ok {
			directPort = int32(route.InternalPort)
			if route.Stopped {
				if route.DBStatus == 5 { // STOPPED
					return nil, nil, "", 0, "", fmt.Errorf("database is stopped")
				}
				if s.routeRegistry.OnWake != nil {
					wakeCtx, wakeCancel := s.detachedContext(30 * time.Second)
					ip, err := s.routeRegistry.OnWake(wakeCtx, route)
					wakeCancel()
					if err != nil {
						return nil, nil, "", 0, "", fmt.Errorf("failed to wake database: %w", err)
					}
					directHost = ip
				}
			}
		}
	}

	password := conn.Password
	if s.secretManager != nil {
		if decrypted, err := s.secretManager.DecryptPassword(conn.Password); err == nil {
			password = decrypted
		}
	}

	return dbInstance, conn, directHost, directPort, password, nil
}

// openDirectConnection opens a direct SQL connection to a database using the overlay network,
// matching the pattern from query.go. It handles wake/sleep, password decryption, etc.
func (s *Service) openDirectConnection(ctx context.Context, databaseID string, dbName string) (*sql.DB, int32, error) {
	dbInstance, conn, directHost, directPort, password, err := s.resolveDirectConnectionDetails(ctx, databaseID)
	if err != nil {
		return nil, 0, err
	}

	if dbName == "" {
		dbName = conn.DatabaseName
	}

	dbType := databasesv1.DatabaseType(dbInstance.Type)
	var db *sql.DB
	switch dbType {
	case databasesv1.DatabaseType_POSTGRESQL:
		connStr := fmt.Sprintf("postgres://%s:%s@%s:%d/%s?sslmode=disable&connect_timeout=5",
			url.QueryEscape(conn.Username), url.QueryEscape(password), url.QueryEscape(directHost), directPort, url.QueryEscape(dbName))
		db, err = sql.Open("postgres", connStr)
	case databasesv1.DatabaseType_MYSQL, databasesv1.DatabaseType_MARIADB:
		connStr := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?timeout=5s",
			url.QueryEscape(conn.Username), url.QueryEscape(password), url.QueryEscape(directHost), directPort, url.QueryEscape(dbName))
		db, err = sql.Open("mysql", connStr)
	default:
		return nil, dbInstance.Type, fmt.Errorf("unsupported database type %d", dbInstance.Type)
	}
	if err != nil {
		return nil, dbInstance.Type, fmt.Errorf("failed to open connection: %w", err)
	}

	if err := db.PingContext(ctx); err != nil {
		db.Close()
		return nil, dbInstance.Type, fmt.Errorf("database unreachable: %w", err)
	}

	return db, dbInstance.Type, nil
}

// GetDatabaseSchema gets the schema information for a database
func (s *Service) GetDatabaseSchema(ctx context.Context, req *connect.Request[databasesv1.GetDatabaseSchemaRequest]) (*connect.Response[databasesv1.GetDatabaseSchemaResponse], error) {
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

	dbName := req.Msg.GetDatabaseName()
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	db, dbType, err := s.openDirectConnection(ctx, req.Msg.GetDatabaseId(), dbName)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	defer db.Close()

	conn, _ := s.connRepo.GetByDatabaseID(ctx, req.Msg.GetDatabaseId())
	if dbName == "" && conn != nil {
		dbName = conn.DatabaseName
	}

	tables, err := s.introspectTables(ctx, db, databasesv1.DatabaseType(dbType), dbName)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to introspect tables: %w", err))
	}

	views, err := s.introspectViews(ctx, db, databasesv1.DatabaseType(dbType), dbName)
	if err != nil {
		views = []*databasesv1.ViewInfo{} // non-fatal
	}

	functions, err := s.introspectFunctions(ctx, db, databasesv1.DatabaseType(dbType), dbName)
	if err != nil {
		functions = []*databasesv1.FunctionInfo{} // non-fatal
	}

	dbNamePtr := &dbName
	res := connect.NewResponse(&databasesv1.GetDatabaseSchemaResponse{
		DatabaseId:   req.Msg.GetDatabaseId(),
		DatabaseName: dbNamePtr,
		Tables:       tables,
		Views:        views,
		Functions:    functions,
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

	conn, _ := s.connRepo.GetByDatabaseID(ctx, req.Msg.GetDatabaseId())
	dbName := req.Msg.GetDatabaseName()
	if dbName == "" && conn != nil {
		dbName = conn.DatabaseName
	}

	tables, err := s.introspectTables(ctx, db, databasesv1.DatabaseType(dbType), dbName)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to list tables: %w", err))
	}

	// Apply pagination
	page := int(req.Msg.GetPage())
	perPage := int(req.Msg.GetPerPage())
	if page < 1 {
		page = 1
	}
	if perPage < 1 || perPage > 100 {
		perPage = 50
	}

	total := int32(len(tables))
	start := (page - 1) * perPage
	end := start + perPage
	if start > len(tables) {
		start = len(tables)
	}
	if end > len(tables) {
		end = len(tables)
	}

	res := connect.NewResponse(&databasesv1.ListTablesResponse{
		Tables: tables[start:end],
		Pagination: &commonv1.Pagination{
			Page:    int32(page),
			PerPage: int32(perPage),
			Total:   total,
		},
	})
	return res, nil
}

// GetTableStructure gets the structure of a specific table
func (s *Service) GetTableStructure(ctx context.Context, req *connect.Request[databasesv1.GetTableStructureRequest]) (*connect.Response[databasesv1.GetTableStructureResponse], error) {
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

	conn, _ := s.connRepo.GetByDatabaseID(ctx, req.Msg.GetDatabaseId())
	dbName := req.Msg.GetDatabaseName()
	if dbName == "" && conn != nil {
		dbName = conn.DatabaseName
	}

	table, err := s.introspectSingleTable(ctx, db, databasesv1.DatabaseType(dbType), dbName, req.Msg.GetTableName())
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to get table structure: %w", err))
	}

	res := connect.NewResponse(&databasesv1.GetTableStructureResponse{
		Table: table,
	})
	return res, nil
}

// introspectTables queries INFORMATION_SCHEMA to get all tables with their columns, indexes, and foreign keys
func (s *Service) introspectTables(ctx context.Context, db *sql.DB, dbType databasesv1.DatabaseType, dbName string) ([]*databasesv1.TableInfo, error) {
	switch dbType {
	case databasesv1.DatabaseType_POSTGRESQL:
		return s.introspectTablesPostgres(ctx, db)
	case databasesv1.DatabaseType_MYSQL, databasesv1.DatabaseType_MARIADB:
		return s.introspectTablesMySQL(ctx, db, dbName)
	default:
		return nil, fmt.Errorf("unsupported database type for introspection")
	}
}

func (s *Service) introspectSingleTable(ctx context.Context, db *sql.DB, dbType databasesv1.DatabaseType, dbName string, tableName string) (*databasesv1.TableInfo, error) {
	tables, err := s.introspectTables(ctx, db, dbType, dbName)
	if err != nil {
		return nil, err
	}
	for _, t := range tables {
		if t.Name == tableName {
			return t, nil
		}
	}
	return nil, fmt.Errorf("table %q not found", tableName)
}

// --- PostgreSQL introspection ---

func (s *Service) introspectTablesPostgres(ctx context.Context, db *sql.DB) ([]*databasesv1.TableInfo, error) {
	// Get tables with row counts and sizes
	tableRows, err := db.QueryContext(ctx, `
		SELECT
			t.table_name,
			t.table_schema,
			COALESCE(s.n_live_tup, 0) AS row_count,
			COALESCE(pg_total_relation_size(quote_ident(t.table_schema) || '.' || quote_ident(t.table_name)), 0) AS size_bytes
		FROM information_schema.tables t
		LEFT JOIN pg_stat_user_tables s ON s.relname = t.table_name AND s.schemaname = t.table_schema
		WHERE t.table_schema NOT IN ('pg_catalog', 'information_schema')
		  AND t.table_type = 'BASE TABLE'
		ORDER BY t.table_schema, t.table_name
	`)
	if err != nil {
		return nil, fmt.Errorf("failed to query tables: %w", err)
	}
	defer tableRows.Close()

	type tableEntry struct {
		info *databasesv1.TableInfo
	}
	tableMap := make(map[string]*tableEntry)
	var tableOrder []string

	for tableRows.Next() {
		var name, schema string
		var rowCount, sizeBytes int64
		if err := tableRows.Scan(&name, &schema, &rowCount, &sizeBytes); err != nil {
			return nil, err
		}
		key := schema + "." + name
		te := &tableEntry{
			info: &databasesv1.TableInfo{
				Name:      name,
				Schema:    schema,
				Type:      "table",
				RowCount:  rowCount,
				SizeBytes: sizeBytes,
			},
		}
		tableMap[key] = te
		tableOrder = append(tableOrder, key)
	}

	// Get columns
	colRows, err := db.QueryContext(ctx, `
		SELECT
			table_schema,
			table_name,
			column_name,
			data_type || CASE
				WHEN character_maximum_length IS NOT NULL THEN '(' || character_maximum_length || ')'
				WHEN numeric_precision IS NOT NULL AND data_type NOT IN ('integer', 'bigint', 'smallint') THEN '(' || numeric_precision || ',' || COALESCE(numeric_scale, 0) || ')'
				ELSE ''
			END AS full_type,
			CASE WHEN is_nullable = 'YES' THEN true ELSE false END AS is_nullable,
			column_default,
			ordinal_position
		FROM information_schema.columns
		WHERE table_schema NOT IN ('pg_catalog', 'information_schema')
		ORDER BY table_schema, table_name, ordinal_position
	`)
	if err != nil {
		return nil, fmt.Errorf("failed to query columns: %w", err)
	}
	defer colRows.Close()

	for colRows.Next() {
		var schema, tableName, colName, dataType string
		var isNullable bool
		var defaultValue *string
		var ordinalPos int32
		if err := colRows.Scan(&schema, &tableName, &colName, &dataType, &isNullable, &defaultValue, &ordinalPos); err != nil {
			return nil, err
		}
		key := schema + "." + tableName
		if te, ok := tableMap[key]; ok {
			col := &databasesv1.ColumnInfo{
				Name:            colName,
				DataType:        dataType,
				IsNullable:      isNullable,
				DefaultValue:    defaultValue,
				OrdinalPosition: ordinalPos,
			}
			te.info.Columns = append(te.info.Columns, col)
		}
	}

	// Get primary keys and unique constraints
	constraintRows, err := db.QueryContext(ctx, `
		SELECT
			tc.table_schema,
			tc.table_name,
			tc.constraint_type,
			kcu.column_name
		FROM information_schema.table_constraints tc
		JOIN information_schema.key_column_usage kcu
			ON tc.constraint_name = kcu.constraint_name
			AND tc.table_schema = kcu.table_schema
		WHERE tc.table_schema NOT IN ('pg_catalog', 'information_schema')
		  AND tc.constraint_type IN ('PRIMARY KEY', 'UNIQUE')
		ORDER BY tc.table_schema, tc.table_name, kcu.ordinal_position
	`)
	if err == nil {
		defer constraintRows.Close()
		for constraintRows.Next() {
			var schema, tableName, constraintType, colName string
			if err := constraintRows.Scan(&schema, &tableName, &constraintType, &colName); err != nil {
				continue
			}
			key := schema + "." + tableName
			if te, ok := tableMap[key]; ok {
				for _, col := range te.info.Columns {
					if col.Name == colName {
						if constraintType == "PRIMARY KEY" {
							col.IsPrimaryKey = true
						}
						if constraintType == "UNIQUE" {
							col.IsUnique = true
						}
					}
				}
			}
		}
	}

	// Get indexes
	indexRows, err := db.QueryContext(ctx, `
		SELECT
			schemaname,
			tablename,
			indexname,
			indexdef
		FROM pg_indexes
		WHERE schemaname NOT IN ('pg_catalog', 'information_schema')
		ORDER BY schemaname, tablename, indexname
	`)
	if err == nil {
		defer indexRows.Close()
		for indexRows.Next() {
			var schema, tableName, indexName, indexDef string
			if err := indexRows.Scan(&schema, &tableName, &indexName, &indexDef); err != nil {
				continue
			}
			key := schema + "." + tableName
			if te, ok := tableMap[key]; ok {
				isUnique := false
				isPrimary := false
				if len(indexDef) > 0 {
					if contains(indexDef, "UNIQUE") {
						isUnique = true
					}
				}
				if contains(indexName, "_pkey") {
					isPrimary = true
					isUnique = true
				}
				idx := &databasesv1.IndexInfo{
					Name:      indexName,
					IsUnique:  isUnique,
					IsPrimary: isPrimary,
				}
				// Extract column names from index definition (simplified)
				idx.ColumnNames = extractIndexColumns(indexDef)
				te.info.Indexes = append(te.info.Indexes, idx)
			}
		}
	}

	// Get foreign keys
	fkRows, err := db.QueryContext(ctx, `
		SELECT
			tc.table_schema,
			tc.table_name,
			tc.constraint_name,
			kcu.column_name,
			ccu.table_name AS foreign_table_name,
			ccu.column_name AS foreign_column_name,
			rc.delete_rule,
			rc.update_rule
		FROM information_schema.table_constraints tc
		JOIN information_schema.key_column_usage kcu
			ON tc.constraint_name = kcu.constraint_name AND tc.table_schema = kcu.table_schema
		JOIN information_schema.constraint_column_usage ccu
			ON tc.constraint_name = ccu.constraint_name AND tc.table_schema = ccu.table_schema
		JOIN information_schema.referential_constraints rc
			ON tc.constraint_name = rc.constraint_name AND tc.table_schema = rc.constraint_schema
		WHERE tc.constraint_type = 'FOREIGN KEY'
		  AND tc.table_schema NOT IN ('pg_catalog', 'information_schema')
		ORDER BY tc.table_schema, tc.table_name, tc.constraint_name
	`)
	if err == nil {
		defer fkRows.Close()
		// Group by constraint name
		fkMap := make(map[string]*databasesv1.ForeignKeyInfo)
		for fkRows.Next() {
			var schema, tableName, constraintName, colName, foreignTable, foreignCol, deleteRule, updateRule string
			if err := fkRows.Scan(&schema, &tableName, &constraintName, &colName, &foreignTable, &foreignCol, &deleteRule, &updateRule); err != nil {
				continue
			}
			fullKey := schema + "." + tableName + "." + constraintName
			if fk, ok := fkMap[fullKey]; ok {
				fk.FromColumns = append(fk.FromColumns, colName)
				fk.ToColumns = append(fk.ToColumns, foreignCol)
			} else {
				fk := &databasesv1.ForeignKeyInfo{
					Name:        constraintName,
					FromTable:   tableName,
					FromColumns: []string{colName},
					ToTable:     foreignTable,
					ToColumns:   []string{foreignCol},
					OnDelete:    &deleteRule,
					OnUpdate:    &updateRule,
				}
				fkMap[fullKey] = fk
				key := schema + "." + tableName
				if te, ok := tableMap[key]; ok {
					te.info.ForeignKeys = append(te.info.ForeignKeys, fk)
				}
			}
		}
	}

	// Build result in order
	result := make([]*databasesv1.TableInfo, 0, len(tableOrder))
	for _, key := range tableOrder {
		if te, ok := tableMap[key]; ok {
			result = append(result, te.info)
		}
	}
	return result, nil
}

// --- MySQL introspection ---

func (s *Service) introspectTablesMySQL(ctx context.Context, db *sql.DB, dbName string) ([]*databasesv1.TableInfo, error) {
	tableRows, err := db.QueryContext(ctx, `
		SELECT
			TABLE_NAME,
			TABLE_SCHEMA,
			COALESCE(TABLE_ROWS, 0),
			COALESCE(DATA_LENGTH + INDEX_LENGTH, 0) AS size_bytes
		FROM information_schema.TABLES
		WHERE TABLE_SCHEMA = ? AND TABLE_TYPE = 'BASE TABLE'
		ORDER BY TABLE_NAME
	`, dbName)
	if err != nil {
		return nil, fmt.Errorf("failed to query tables: %w", err)
	}
	defer tableRows.Close()

	type tableEntry struct {
		info *databasesv1.TableInfo
	}
	tableMap := make(map[string]*tableEntry)
	var tableOrder []string

	for tableRows.Next() {
		var name, schema string
		var rowCount, sizeBytes int64
		if err := tableRows.Scan(&name, &schema, &rowCount, &sizeBytes); err != nil {
			return nil, err
		}
		te := &tableEntry{
			info: &databasesv1.TableInfo{
				Name:      name,
				Schema:    schema,
				Type:      "table",
				RowCount:  rowCount,
				SizeBytes: sizeBytes,
			},
		}
		tableMap[name] = te
		tableOrder = append(tableOrder, name)
	}

	// Get columns
	colRows, err := db.QueryContext(ctx, `
		SELECT
			TABLE_NAME,
			COLUMN_NAME,
			COLUMN_TYPE,
			CASE WHEN IS_NULLABLE = 'YES' THEN 1 ELSE 0 END,
			COLUMN_DEFAULT,
			ORDINAL_POSITION,
			COLUMN_KEY
		FROM information_schema.COLUMNS
		WHERE TABLE_SCHEMA = ?
		ORDER BY TABLE_NAME, ORDINAL_POSITION
	`, dbName)
	if err != nil {
		return nil, fmt.Errorf("failed to query columns: %w", err)
	}
	defer colRows.Close()

	for colRows.Next() {
		var tableName, colName, dataType, colKey string
		var isNullable bool
		var defaultValue *string
		var ordinalPos int32
		if err := colRows.Scan(&tableName, &colName, &dataType, &isNullable, &defaultValue, &ordinalPos, &colKey); err != nil {
			return nil, err
		}
		if te, ok := tableMap[tableName]; ok {
			col := &databasesv1.ColumnInfo{
				Name:            colName,
				DataType:        dataType,
				IsNullable:      isNullable,
				DefaultValue:    defaultValue,
				OrdinalPosition: ordinalPos,
				IsPrimaryKey:    colKey == "PRI",
				IsUnique:        colKey == "UNI" || colKey == "PRI",
			}
			te.info.Columns = append(te.info.Columns, col)
		}
	}

	// Get indexes
	idxRows, err := db.QueryContext(ctx, `
		SELECT
			TABLE_NAME,
			INDEX_NAME,
			NON_UNIQUE,
			COLUMN_NAME,
			INDEX_TYPE
		FROM information_schema.STATISTICS
		WHERE TABLE_SCHEMA = ?
		ORDER BY TABLE_NAME, INDEX_NAME, SEQ_IN_INDEX
	`, dbName)
	if err == nil {
		defer idxRows.Close()
		idxMap := make(map[string]*databasesv1.IndexInfo)
		for idxRows.Next() {
			var tableName, indexName, colName, indexType string
			var nonUnique int
			if err := idxRows.Scan(&tableName, &indexName, &nonUnique, &colName, &indexType); err != nil {
				continue
			}
			mapKey := tableName + "." + indexName
			if idx, ok := idxMap[mapKey]; ok {
				idx.ColumnNames = append(idx.ColumnNames, colName)
			} else {
				idxType := indexType
				idx := &databasesv1.IndexInfo{
					Name:        indexName,
					IsUnique:    nonUnique == 0,
					IsPrimary:   indexName == "PRIMARY",
					ColumnNames: []string{colName},
					Type:        &idxType,
				}
				idxMap[mapKey] = idx
				if te, ok := tableMap[tableName]; ok {
					te.info.Indexes = append(te.info.Indexes, idx)
				}
			}
		}
	}

	// Get foreign keys
	fkRows, err := db.QueryContext(ctx, `
		SELECT
			kcu.TABLE_NAME,
			kcu.CONSTRAINT_NAME,
			kcu.COLUMN_NAME,
			kcu.REFERENCED_TABLE_NAME,
			kcu.REFERENCED_COLUMN_NAME,
			rc.DELETE_RULE,
			rc.UPDATE_RULE
		FROM information_schema.KEY_COLUMN_USAGE kcu
		JOIN information_schema.REFERENTIAL_CONSTRAINTS rc
			ON kcu.CONSTRAINT_NAME = rc.CONSTRAINT_NAME AND kcu.TABLE_SCHEMA = rc.CONSTRAINT_SCHEMA
		WHERE kcu.TABLE_SCHEMA = ? AND kcu.REFERENCED_TABLE_NAME IS NOT NULL
		ORDER BY kcu.TABLE_NAME, kcu.CONSTRAINT_NAME, kcu.ORDINAL_POSITION
	`, dbName)
	if err == nil {
		defer fkRows.Close()
		fkMap := make(map[string]*databasesv1.ForeignKeyInfo)
		for fkRows.Next() {
			var tableName, constraintName, colName, refTable, refCol, deleteRule, updateRule string
			if err := fkRows.Scan(&tableName, &constraintName, &colName, &refTable, &refCol, &deleteRule, &updateRule); err != nil {
				continue
			}
			mapKey := tableName + "." + constraintName
			if fk, ok := fkMap[mapKey]; ok {
				fk.FromColumns = append(fk.FromColumns, colName)
				fk.ToColumns = append(fk.ToColumns, refCol)
			} else {
				fk := &databasesv1.ForeignKeyInfo{
					Name:        constraintName,
					FromTable:   tableName,
					FromColumns: []string{colName},
					ToTable:     refTable,
					ToColumns:   []string{refCol},
					OnDelete:    &deleteRule,
					OnUpdate:    &updateRule,
				}
				fkMap[mapKey] = fk
				if te, ok := tableMap[tableName]; ok {
					te.info.ForeignKeys = append(te.info.ForeignKeys, fk)
				}
			}
		}
	}

	result := make([]*databasesv1.TableInfo, 0, len(tableOrder))
	for _, key := range tableOrder {
		if te, ok := tableMap[key]; ok {
			result = append(result, te.info)
		}
	}
	return result, nil
}

// --- Views and functions introspection ---

func (s *Service) introspectViews(ctx context.Context, db *sql.DB, dbType databasesv1.DatabaseType, dbName string) ([]*databasesv1.ViewInfo, error) {
	var query string
	var args []interface{}

	switch dbType {
	case databasesv1.DatabaseType_POSTGRESQL:
		query = `SELECT table_name, table_schema, COALESCE(view_definition, '')
			FROM information_schema.views
			WHERE table_schema NOT IN ('pg_catalog', 'information_schema')
			ORDER BY table_schema, table_name`
	case databasesv1.DatabaseType_MYSQL, databasesv1.DatabaseType_MARIADB:
		query = `SELECT TABLE_NAME, TABLE_SCHEMA, COALESCE(VIEW_DEFINITION, '')
			FROM information_schema.VIEWS
			WHERE TABLE_SCHEMA = ?
			ORDER BY TABLE_NAME`
		args = append(args, dbName)
	default:
		return nil, nil
	}

	rows, err := db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var views []*databasesv1.ViewInfo
	for rows.Next() {
		var name, schema, definition string
		if err := rows.Scan(&name, &schema, &definition); err != nil {
			continue
		}
		views = append(views, &databasesv1.ViewInfo{
			Name:       name,
			Schema:     schema,
			Definition: definition,
		})
	}
	return views, nil
}

func (s *Service) introspectFunctions(ctx context.Context, db *sql.DB, dbType databasesv1.DatabaseType, dbName string) ([]*databasesv1.FunctionInfo, error) {
	var query string
	var args []interface{}

	switch dbType {
	case databasesv1.DatabaseType_POSTGRESQL:
		query = `SELECT routine_name, routine_schema, COALESCE(data_type, 'void')
			FROM information_schema.routines
			WHERE routine_schema NOT IN ('pg_catalog', 'information_schema')
			  AND routine_type = 'FUNCTION'
			ORDER BY routine_schema, routine_name
			LIMIT 200`
	case databasesv1.DatabaseType_MYSQL, databasesv1.DatabaseType_MARIADB:
		query = `SELECT ROUTINE_NAME, ROUTINE_SCHEMA, COALESCE(DATA_TYPE, 'void')
			FROM information_schema.ROUTINES
			WHERE ROUTINE_SCHEMA = ? AND ROUTINE_TYPE = 'FUNCTION'
			ORDER BY ROUTINE_NAME
			LIMIT 200`
		args = append(args, dbName)
	default:
		return nil, nil
	}

	rows, err := db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var functions []*databasesv1.FunctionInfo
	for rows.Next() {
		var name, schema, returnType string
		if err := rows.Scan(&name, &schema, &returnType); err != nil {
			continue
		}
		functions = append(functions, &databasesv1.FunctionInfo{
			Name:       name,
			Schema:     schema,
			ReturnType: returnType,
		})
	}
	return functions, nil
}

// --- Helper functions ---

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsStr(s, substr))
}

func containsStr(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func extractIndexColumns(indexDef string) []string {
	// Extract columns from PostgreSQL index definition like:
	// CREATE UNIQUE INDEX users_pkey ON public.users USING btree (id)
	// Find content between last ( and )
	start := -1
	end := -1
	for i := len(indexDef) - 1; i >= 0; i-- {
		if indexDef[i] == ')' && end == -1 {
			end = i
		}
		if indexDef[i] == '(' && start == -1 {
			start = i + 1
			break
		}
	}
	if start < 0 || end < 0 || start >= end {
		return nil
	}
	colStr := indexDef[start:end]
	// Split by comma and trim
	var cols []string
	current := ""
	for _, ch := range colStr {
		if ch == ',' {
			trimmed := trimSpace(current)
			if trimmed != "" {
				cols = append(cols, trimmed)
			}
			current = ""
		} else {
			current += string(ch)
		}
	}
	if trimmed := trimSpace(current); trimmed != "" {
		cols = append(cols, trimmed)
	}
	return cols
}

func trimSpace(s string) string {
	start := 0
	end := len(s)
	for start < end && (s[start] == ' ' || s[start] == '\t') {
		start++
	}
	for end > start && (s[end-1] == ' ' || s[end-1] == '\t') {
		end--
	}
	return s[start:end]
}
