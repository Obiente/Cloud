package databases

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/obiente/cloud/apps/shared/pkg/auth"
	"github.com/obiente/cloud/apps/shared/pkg/database"
	databasesv1 "github.com/obiente/cloud/apps/shared/proto/obiente/cloud/databases/v1"

	"connectrpc.com/connect"
)

const maxInlineDumpBytes = 32 * 1024 * 1024

// ExportDatabaseDump exports a SQL dump using the database engine's native dump tool.
func (s *Service) ExportDatabaseDump(ctx context.Context, req *connect.Request[databasesv1.ExportDatabaseDumpRequest]) (*connect.Response[databasesv1.ExportDatabaseDumpResponse], error) {
	orgID := req.Msg.GetOrganizationId()
	if orgID == "" {
		if eff, ok := resolveUserDefaultOrgID(ctx); ok {
			orgID = eff
		}
	}

	dbInstance, conn, password, err := s.authorizeDumpOperation(ctx, orgID, req.Msg.GetDatabaseId(), auth.PermissionDatabaseRead)
	if err != nil {
		return nil, err
	}

	if s.provisioner == nil {
		return nil, connect.NewError(connect.CodeFailedPrecondition, fmt.Errorf("database dump export requires docker provisioner"))
	}
	if dbInstance.InstanceID == nil || *dbInstance.InstanceID == "" {
		return nil, connect.NewError(connect.CodeFailedPrecondition, fmt.Errorf("database container is not available"))
	}
	if err := validateDumpFormat(req.Msg.GetFormat()); err != nil {
		return nil, err
	}

	dbName := req.Msg.GetDatabaseName()
	if dbName == "" {
		dbName = conn.DatabaseName
	}

	includeSchema, includeData := dumpSections(req.Msg.GetIncludeSchema(), req.Msg.GetIncludeData())
	cmd, env, err := exportDumpCommand(databasesv1.DatabaseType(dbInstance.Type), conn.Username, password, dbName, includeSchema, includeData)
	if err != nil {
		return nil, err
	}

	dumpCtx, cancel := context.WithTimeout(ctx, 5*time.Minute)
	defer cancel()
	output, err := s.provisioner.ExecInDatabaseWithInputEnv(dumpCtx, *dbInstance.InstanceID, cmd, nil, env)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to export dump: %w", err))
	}

	dumpData := []byte(output)
	if len(dumpData) > maxInlineDumpBytes {
		return nil, connect.NewError(connect.CodeResourceExhausted, fmt.Errorf("dump is %d bytes; inline exports are limited to %d bytes", len(dumpData), maxInlineDumpBytes))
	}

	fileName := fmt.Sprintf("%s.sql", sanitizeDumpFileName(dbName))
	return connect.NewResponse(&databasesv1.ExportDatabaseDumpResponse{
		FileName:    fileName,
		ContentType: "application/sql",
		DumpData:    dumpData,
		SizeBytes:   int64(len(dumpData)),
	}), nil
}

// ImportDatabaseDump imports a SQL dump using the database engine's native client.
func (s *Service) ImportDatabaseDump(ctx context.Context, req *connect.Request[databasesv1.ImportDatabaseDumpRequest]) (*connect.Response[databasesv1.ImportDatabaseDumpResponse], error) {
	orgID := req.Msg.GetOrganizationId()
	if orgID == "" {
		if eff, ok := resolveUserDefaultOrgID(ctx); ok {
			orgID = eff
		}
	}

	if len(req.Msg.GetDumpData()) == 0 {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("dump_data is required"))
	}
	if len(req.Msg.GetDumpData()) > maxInlineDumpBytes {
		return nil, connect.NewError(connect.CodeResourceExhausted, fmt.Errorf("dump is %d bytes; inline imports are limited to %d bytes", len(req.Msg.GetDumpData()), maxInlineDumpBytes))
	}
	if err := validateDumpFormat(req.Msg.GetFormat()); err != nil {
		return nil, err
	}
	if req.Msg.GetDropExisting() {
		return nil, connect.NewError(connect.CodeUnimplemented, fmt.Errorf("drop_existing imports are not implemented yet"))
	}

	dbInstance, conn, password, err := s.authorizeDumpOperation(ctx, orgID, req.Msg.GetDatabaseId(), auth.PermissionDatabaseManage)
	if err != nil {
		return nil, err
	}

	if s.provisioner == nil {
		return nil, connect.NewError(connect.CodeFailedPrecondition, fmt.Errorf("database dump import requires docker provisioner"))
	}
	if dbInstance.InstanceID == nil || *dbInstance.InstanceID == "" {
		return nil, connect.NewError(connect.CodeFailedPrecondition, fmt.Errorf("database container is not available"))
	}

	dbName := req.Msg.GetDatabaseName()
	if dbName == "" {
		dbName = conn.DatabaseName
	}

	cmd, env, err := importDumpCommand(databasesv1.DatabaseType(dbInstance.Type), conn.Username, password, dbName)
	if err != nil {
		return nil, err
	}

	importCtx, cancel := context.WithTimeout(ctx, 5*time.Minute)
	defer cancel()
	if _, err := s.provisioner.ExecInDatabaseWithInputEnv(importCtx, *dbInstance.InstanceID, cmd, req.Msg.GetDumpData(), env); err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("failed to import dump: %w", err))
	}

	return connect.NewResponse(&databasesv1.ImportDatabaseDumpResponse{
		Success:   true,
		Message:   "SQL dump imported successfully",
		SizeBytes: int64(len(req.Msg.GetDumpData())),
	}), nil
}

func (s *Service) authorizeDumpOperation(ctx context.Context, orgID, databaseID, permission string) (*database.DatabaseInstance, *database.DatabaseConnection, string, error) {
	if err := s.checkDatabasePermission(ctx, databaseID, permission); err != nil {
		return nil, nil, "", connect.NewError(connect.CodePermissionDenied, err)
	}

	dbInstance, conn, _, _, password, err := s.resolveDirectConnectionDetails(ctx, databaseID)
	if err != nil {
		return nil, nil, "", mapQueryPreparationError(err)
	}
	if dbInstance.OrganizationID != orgID {
		return nil, nil, "", connect.NewError(connect.CodeNotFound, fmt.Errorf("database not found"))
	}

	return dbInstance, conn, password, nil
}

func dumpSections(includeSchema, includeData bool) (bool, bool) {
	if !includeSchema && !includeData {
		return true, true
	}
	return includeSchema, includeData
}

func validateDumpFormat(format databasesv1.DatabaseDumpFormat) error {
	if format == databasesv1.DatabaseDumpFormat_DATABASE_DUMP_FORMAT_UNSPECIFIED ||
		format == databasesv1.DatabaseDumpFormat_SQL {
		return nil
	}
	return connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("unsupported dump format %s", format.String()))
}

func exportDumpCommand(dbType databasesv1.DatabaseType, username, password, dbName string, includeSchema, includeData bool) ([]string, []string, error) {
	switch dbType {
	case databasesv1.DatabaseType_POSTGRESQL:
		args := []string{"pg_dump", "--no-owner", "--no-privileges", "-U", username, "-d", dbName}
		if includeSchema && !includeData {
			args = append(args, "--schema-only")
		}
		if includeData && !includeSchema {
			args = append(args, "--data-only")
		}
		return args, []string{"PGPASSWORD=" + password}, nil
	case databasesv1.DatabaseType_MYSQL, databasesv1.DatabaseType_MARIADB:
		args := []string{"mysqldump", "--single-transaction", "--skip-lock-tables", "-u", username}
		if includeSchema && !includeData {
			args = append(args, "--no-data")
		}
		if includeData && !includeSchema {
			args = append(args, "--no-create-info")
		}
		args = append(args, dbName)
		return args, []string{"MYSQL_PWD=" + password}, nil
	default:
		return nil, nil, connect.NewError(connect.CodeUnimplemented, fmt.Errorf("SQL dump export is not supported for %s", dbType.String()))
	}
}

func importDumpCommand(dbType databasesv1.DatabaseType, username, password, dbName string) ([]string, []string, error) {
	switch dbType {
	case databasesv1.DatabaseType_POSTGRESQL:
		return []string{"psql", "-v", "ON_ERROR_STOP=1", "-U", username, "-d", dbName}, []string{"PGPASSWORD=" + password}, nil
	case databasesv1.DatabaseType_MYSQL, databasesv1.DatabaseType_MARIADB:
		return []string{"mysql", "-u", username, dbName}, []string{"MYSQL_PWD=" + password}, nil
	default:
		return nil, nil, connect.NewError(connect.CodeUnimplemented, fmt.Errorf("SQL dump import is not supported for %s", dbType.String()))
	}
}

func sanitizeDumpFileName(name string) string {
	name = strings.TrimSpace(name)
	if name == "" {
		return "database"
	}
	replacer := strings.NewReplacer("/", "_", "\\", "_", "\x00", "_")
	return replacer.Replace(name)
}
