package databases

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	"connectrpc.com/connect"
	"databases-service/internal/backups"
	"github.com/google/uuid"
	"github.com/obiente/cloud/apps/shared/pkg/auth"
	"github.com/obiente/cloud/apps/shared/pkg/database"
	"github.com/obiente/cloud/apps/shared/pkg/logger"
	authv1 "github.com/obiente/cloud/apps/shared/proto/obiente/cloud/auth/v1"
	commonv1 "github.com/obiente/cloud/apps/shared/proto/obiente/cloud/common/v1"
	databasesv1 "github.com/obiente/cloud/apps/shared/proto/obiente/cloud/databases/v1"
)

var errBackupValidation = errors.New("backup validation failed")

func configuredBackupStore() backups.Store {
	size := int64(1 << 30)
	if value, err := strconv.ParseInt(os.Getenv("DATABASE_BACKUP_MAX_BYTES"), 10, 64); err == nil && value > 0 {
		size = value
	}
	return backups.Store{Directory: os.Getenv("DATABASE_BACKUP_DIRECTORY"), MaxBytes: size}
}
func backupRetention() int {
	value, err := strconv.Atoi(os.Getenv("DATABASE_BACKUP_KEEP"))
	if err != nil || value < 1 || value > 1000 {
		return 10
	}
	return value
}
func supportsManagedBackup(kind int32) bool {
	return kind == int32(databasesv1.DatabaseType_POSTGRESQL) || kind == int32(databasesv1.DatabaseType_MYSQL) || kind == int32(databasesv1.DatabaseType_MARIADB)
}
func (s *Service) backupAvailability(instance *database.DatabaseInstance) string {
	if !supportsManagedBackup(instance.Type) {
		return "Managed backups are supported for PostgreSQL, MySQL and MariaDB."
	}
	if s.backupExec == nil {
		return "Managed backups are unavailable on this server."
	}
	root, err := configuredBackupStore().Open()
	if err != nil {
		return "The shared backup volume is unavailable. Contact your administrator."
	}
	_ = root.Close()
	return ""
}
func (s *Service) authorizeBackup(ctx context.Context, orgID, id, permission string) (*database.DatabaseInstance, error) {
	if orgID == "" {
		orgID, _ = resolveUserDefaultOrgID(ctx)
	}
	if err := s.checkDatabasePermission(ctx, id, permission); err != nil {
		return nil, connect.NewError(connect.CodePermissionDenied, err)
	}
	instance, err := s.repo.GetByID(ctx, id)
	if err != nil || instance.OrganizationID != orgID {
		return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("database not found"))
	}
	return instance, nil
}
func (s *Service) ownedBackup(ctx context.Context, databaseID, id string) (*database.DatabaseBackup, error) {
	backup, err := s.backupRepo.GetByID(ctx, id)
	if err != nil || backup.DatabaseID != databaseID {
		return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("backup not found"))
	}
	return backup, nil
}
func (s *Service) hasBackupWork(ctx context.Context, id string) (bool, error) {
	var count int64
	err := database.DB.WithContext(ctx).Model(&database.DatabaseBackup{}).Where("database_id = ? AND (status IN ? OR restore_status IN ?)", id, []int32{1, 4}, []string{"queued", "restoring"}).Count(&count).Error
	return count > 0, err
}
func (s *Service) queueBackupOperation(ctx context.Context, id string, operation func(context.Context) error) error {
	busy, err := s.hasBackupWork(ctx, id)
	if err != nil {
		return connect.NewError(connect.CodeUnavailable, err)
	}
	if busy {
		return connect.NewError(connect.CodeFailedPrecondition, fmt.Errorf("a backup operation is already in progress"))
	}
	bounded, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	return database.WithResourceRuntimeLock(bounded, "database:"+id, func(locked context.Context) error {
		busy, err := s.hasBackupWork(locked, id)
		if err != nil {
			return err
		}
		if busy {
			return connect.NewError(connect.CodeFailedPrecondition, fmt.Errorf("a backup operation is already in progress"))
		}
		return operation(locked)
	})
}
func (s *Service) wakeBackupWorker() {
	if s.backupWake != nil {
		select {
		case s.backupWake <- struct{}{}:
		default:
		}
	}
}

func (s *Service) ListBackups(ctx context.Context, req *connect.Request[databasesv1.ListBackupsRequest]) (*connect.Response[databasesv1.ListBackupsResponse], error) {
	instance, err := s.authorizeBackup(ctx, req.Msg.OrganizationId, req.Msg.DatabaseId, auth.PermissionDatabaseRead)
	if err != nil {
		return nil, err
	}
	page, perPage := req.Msg.Page, req.Msg.PerPage
	if page < 1 {
		page = 1
	}
	if perPage < 1 {
		perPage = 50
	}
	if perPage > 100 {
		perPage = 100
	}
	rows, err := s.backupRepo.GetAll(ctx, instance.ID, &database.BackupFilters{Limit: int64(perPage), Offset: (int64(page) - 1) * int64(perPage)})
	if err != nil {
		return nil, connect.NewError(connect.CodeUnavailable, err)
	}
	var total int64
	if err = database.DB.WithContext(ctx).Model(&database.DatabaseBackup{}).Where("database_id = ?", instance.ID).Count(&total).Error; err != nil {
		return nil, connect.NewError(connect.CodeUnavailable, err)
	}
	result := &databasesv1.ListBackupsResponse{Pagination: &commonv1.Pagination{Page: page, PerPage: perPage, Total: int32(total)}, RetainedCount: int32(backupRetention())}
	result.UnavailableReason = s.backupAvailability(instance)
	result.ManagedAvailable = result.UnavailableReason == ""
	for _, row := range rows {
		result.Backups = append(result.Backups, dbBackupToProto(row))
	}
	return connect.NewResponse(result), nil
}
func (s *Service) GetBackup(ctx context.Context, req *connect.Request[databasesv1.GetBackupRequest]) (*connect.Response[databasesv1.GetBackupResponse], error) {
	if _, err := s.authorizeBackup(ctx, req.Msg.OrganizationId, req.Msg.DatabaseId, auth.PermissionDatabaseRead); err != nil {
		return nil, err
	}
	backup, err := s.ownedBackup(ctx, req.Msg.DatabaseId, req.Msg.BackupId)
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(&databasesv1.GetBackupResponse{Backup: dbBackupToProto(backup)}), nil
}
func (s *Service) CreateBackup(ctx context.Context, req *connect.Request[databasesv1.CreateBackupRequest]) (*connect.Response[databasesv1.CreateBackupResponse], error) {
	instance, err := s.authorizeBackup(ctx, req.Msg.OrganizationId, req.Msg.DatabaseId, auth.PermissionDatabaseUpdate)
	if err != nil {
		return nil, err
	}
	if reason := s.backupAvailability(instance); reason != "" {
		return nil, connect.NewError(connect.CodeFailedPrecondition, fmt.Errorf("%s", reason))
	}
	user, err := auth.GetUserFromContext(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeUnauthenticated, err)
	}
	name := strings.TrimSpace(req.Msg.GetName())
	if name == "" {
		name = "Backup " + time.Now().UTC().Format("2006-01-02 15:04 UTC")
	}
	if len(name) > 120 || len(req.Msg.GetDescription()) > 2000 {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("backup name or description is too long"))
	}
	backup := &database.DatabaseBackup{ID: uuid.NewString(), DatabaseID: instance.ID, OrganizationID: instance.OrganizationID, CreatedBy: user.Id, Name: name, Description: req.Msg.Description, Status: 1}
	err = s.queueBackupOperation(ctx, instance.ID, func(locked context.Context) error { return s.backupRepo.Create(locked, backup) })
	if err != nil {
		return nil, err
	}
	s.wakeBackupWorker()
	return connect.NewResponse(&databasesv1.CreateBackupResponse{Backup: dbBackupToProto(backup)}), nil
}
func (s *Service) DeleteBackup(ctx context.Context, req *connect.Request[databasesv1.DeleteBackupRequest]) (*connect.Response[databasesv1.DeleteBackupResponse], error) {
	if _, err := s.authorizeBackup(ctx, req.Msg.OrganizationId, req.Msg.DatabaseId, auth.PermissionDatabaseDelete); err != nil {
		return nil, err
	}
	if _, err := s.ownedBackup(ctx, req.Msg.DatabaseId, req.Msg.BackupId); err != nil {
		return nil, err
	}
	err := s.queueBackupOperation(ctx, req.Msg.DatabaseId, func(locked context.Context) error {
		backup, err := s.ownedBackup(locked, req.Msg.DatabaseId, req.Msg.BackupId)
		if err != nil {
			return err
		}
		backup.Status = 4
		return s.backupRepo.Update(locked, backup)
	})
	if err != nil {
		return nil, err
	}
	s.wakeBackupWorker()
	return connect.NewResponse(&databasesv1.DeleteBackupResponse{Success: true}), nil
}

var restoreNamePattern = regexp.MustCompile(`^[A-Za-z][A-Za-z0-9_]{0,62}$`)

func (s *Service) RestoreBackup(ctx context.Context, req *connect.Request[databasesv1.RestoreBackupRequest]) (*connect.Response[databasesv1.RestoreBackupResponse], error) {
	instance, err := s.authorizeBackup(ctx, req.Msg.OrganizationId, req.Msg.DatabaseId, auth.PermissionDatabaseManage)
	if err != nil {
		return nil, err
	}
	if reason := s.backupAvailability(instance); reason != "" {
		return nil, connect.NewError(connect.CodeFailedPrecondition, fmt.Errorf("%s", reason))
	}
	user, err := auth.GetUserFromContext(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeUnauthenticated, err)
	}
	target := req.Msg.GetTargetDatabaseName()
	if !restoreNamePattern.MatchString(target) {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("choose a new database name: a letter followed by up to 62 letters, digits or underscores"))
	}
	var backup *database.DatabaseBackup
	err = s.queueBackupOperation(ctx, instance.ID, func(locked context.Context) error {
		backup, err = s.ownedBackup(locked, instance.ID, req.Msg.BackupId)
		if err != nil {
			return err
		}
		if backup.Status != 2 || backup.Checksum == "" || backup.Engine != instance.Type {
			return connect.NewError(connect.CodeFailedPrecondition, fmt.Errorf("backup is incomplete or incompatible"))
		}
		if target == backup.SourceDatabase {
			return connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("restore must use a new database name"))
		}
		backup.RestoreStatus = "queued"
		backup.RestoreTarget = target
		backup.RestoreUser = user.Id
		backup.RestoreError = nil
		backup.RestoredAt = nil
		return s.backupRepo.Update(locked, backup)
	})
	if err != nil {
		return nil, err
	}
	s.wakeBackupWorker()
	return connect.NewResponse(&databasesv1.RestoreBackupResponse{Database: dbDatabaseToProto(instance), Backup: dbBackupToProto(backup), Message: "Restore queued into new database " + target}), nil
}

// Work is durable in the metadata database. A restarted worker safely retries
// incomplete exports/deletions; interrupted imports are reported, never repeated.
func (s *Service) StartBackupWorker(ctx context.Context) {
	go func() {
		ticker := time.NewTicker(15 * time.Second)
		defer ticker.Stop()
		for {
			s.processBackupQueue(ctx)
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
			case <-s.backupWake:
			}
		}
	}()
}
func (s *Service) processBackupQueue(ctx context.Context) {
	var pending []database.DatabaseBackup
	listing, cancelListing := context.WithTimeout(ctx, 5*time.Second)
	defer cancelListing()
	if err := database.DB.WithContext(listing).Where("status IN ? OR restore_status IN ?", []int32{1, 4}, []string{"queued", "restoring"}).Order("created_at").Limit(100).Find(&pending).Error; err != nil {
		if ctx.Err() == nil {
			logger.Warn("List pending database backups: %v", err)
		}
		return
	}
	for _, item := range pending {
		if ctx.Err() != nil {
			return
		}
		operation, cancel := context.WithTimeout(ctx, 10*time.Minute)
		err := s.processBackup(operation, item.ID)
		cancel()
		if err != nil && ctx.Err() == nil {
			logger.Warn("Process database backup %s: %v", item.ID, err)
		}
	}
}
func (s *Service) processBackup(ctx context.Context, id string) error {
	item, err := s.backupRepo.GetByID(ctx, id)
	if err != nil {
		return err
	}
	return database.WithResourceRuntimeLock(ctx, "database:"+item.DatabaseID, func(locked context.Context) error {
		item, err := s.backupRepo.GetByID(locked, id)
		if err != nil {
			return err
		}
		store := configuredBackupStore()
		if item.Status == 4 {
			if err = store.Delete(item.ID); err != nil {
				return err
			}
			return s.backupRepo.Delete(locked, item.ID)
		}
		if item.RestoreStatus == "restoring" {
			item.RestoreStatus = "failed"
			message := "Restore was interrupted. The target database may contain partial data; inspect it before retrying with a new name."
			item.RestoreError = &message
			return s.backupRepo.Update(locked, item)
		}
		if item.Status != 1 && item.RestoreStatus != "queued" {
			return nil
		}
		restoring := item.RestoreStatus == "queued"
		permission, user := auth.PermissionDatabaseUpdate, item.CreatedBy
		if restoring {
			permission, user = auth.PermissionDatabaseManage, item.RestoreUser
		}
		authorized := auth.WithUser(locked, &authv1.User{Id: user})
		instance, err := s.authorizeBackup(authorized, item.OrganizationID, item.DatabaseID, permission)
		if err == nil && (instance.DesiredState == "stopped" || instance.InstanceID == nil || *instance.InstanceID == "") {
			err = fmt.Errorf("database is stopped or unavailable")
		}
		if err == nil && s.backupExec == nil {
			err = fmt.Errorf("database backup execution is unavailable")
		}
		var conn *database.DatabaseConnection
		var password string
		if err == nil {
			conn, err = s.connRepo.GetByDatabaseID(locked, item.DatabaseID)
			if err == nil {
				password = conn.Password
				if s.secretManager != nil {
					password, err = s.secretManager.DecryptPassword(password)
				}
			}
		}
		if err == nil {
			if restoring {
				err = s.restoreManagedBackup(locked, item, instance, conn, password, store)
			} else {
				var command, env []string
				kind := databasesv1.DatabaseType(instance.Type)
				user := conn.Username
				if mysqlBackupEngine(kind) {
					user = "root"
				}
				command, env, err = exportDumpCommand(kind, user, password, conn.DatabaseName, true, true)
				if mysqlBackupEngine(kind) {
					command, env = mysqlAdministrationCommand(command), nil
				}
				if err == nil {
					item.SizeBytes, item.Checksum, err = store.Write(locked, item.ID, func(output io.Writer) error {
						return s.backupExec(locked, *instance.InstanceID, command, nil, output, env)
					})
					if err == nil {
						now := time.Now().UTC()
						item.CompletedAt = &now
						item.Status = 2
						item.ErrorMessage = nil
						item.SourceDatabase = conn.DatabaseName
						item.Engine = instance.Type
						path := item.ID + ".sql.gz"
						item.BackupPath = &path
					}
				}
			}
		}
		if err != nil {
			logger.Warn("Database backup operation %s failed: %v", id, err)
			if restoring {
				item.RestoreStatus = "failed"
				message := "Restore failed. An existing database is never overwritten; a newly created target may contain partial data."
				if errors.Is(err, errBackupValidation) {
					message = "Backup file verification failed. No target database was created. Check the shared backup volume and restore an intact backup."
				}
				item.RestoreError = &message
			} else {
				item.Status = 3
				message := "Backup failed. Check database availability and the shared backup volume, then retry."
				if errors.Is(err, backups.ErrSizeLimit) {
					message = "Backup exceeds the configured size limit. Ask your administrator to increase the backup size limit."
				}
				if connect.CodeOf(err) == connect.CodePermissionDenied {
					message = "Your access changed before this backup could run. Contact your administrator."
				}
				item.ErrorMessage = &message
				_ = store.Delete(item.ID)
			}
		}
		persist, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if saveErr := s.backupRepo.Update(persist, item); saveErr != nil {
			return saveErr
		}
		if err == nil && !restoring {
			return s.applyBackupRetention(persist, item.DatabaseID)
		}
		return err
	})
}
func (s *Service) restoreManagedBackup(ctx context.Context, item *database.DatabaseBackup, instance *database.DatabaseInstance, conn *database.DatabaseConnection, password string, store backups.Store) error {
	if item.Engine != instance.Type || item.RestoreTarget == conn.DatabaseName || !restoreNamePattern.MatchString(item.RestoreTarget) {
		return fmt.Errorf("restore target or database engine is invalid")
	}
	input, err := store.Read(ctx, item.ID, item.Checksum, item.SizeBytes)
	if err != nil {
		return fmt.Errorf("%w: %v", errBackupValidation, err)
	}
	defer input.Close()
	item.RestoreStatus = "restoring"
	if err = s.backupRepo.Update(ctx, item); err != nil {
		return err
	}
	command, env, err := createRestoreDatabaseCommand(databasesv1.DatabaseType(instance.Type), conn.Username, password, conn.DatabaseName, item.RestoreTarget)
	if err != nil {
		return err
	}
	if err = s.backupExec(ctx, *instance.InstanceID, command, nil, io.Discard, env); err != nil {
		return err
	}
	kind := databasesv1.DatabaseType(instance.Type)
	user := conn.Username
	if mysqlBackupEngine(kind) {
		user = "root"
	}
	command, env, err = importDumpCommand(kind, user, password, item.RestoreTarget)
	if mysqlBackupEngine(kind) {
		command, env = mysqlAdministrationCommand(command), nil
	}
	if err != nil {
		return err
	}
	if err = s.backupExec(ctx, *instance.InstanceID, command, input, io.Discard, env); err != nil {
		return err
	}
	now := time.Now().UTC()
	item.RestoreStatus = "completed"
	item.RestoredAt = &now
	item.RestoreError = nil
	return nil
}
func createRestoreDatabaseCommand(kind databasesv1.DatabaseType, user, password, source, target string) ([]string, []string, error) {
	if !restoreNamePattern.MatchString(target) {
		return nil, nil, fmt.Errorf("invalid restore database name")
	}
	if kind == databasesv1.DatabaseType_POSTGRESQL {
		return []string{"psql", "-v", "ON_ERROR_STOP=1", "-U", user, "-d", source, "-c", `CREATE DATABASE "` + target + `"`}, []string{"PGPASSWORD=" + password}, nil
	}
	if kind == databasesv1.DatabaseType_MYSQL || kind == databasesv1.DatabaseType_MARIADB {
		binary := "mysql"
		if kind == databasesv1.DatabaseType_MARIADB {
			binary = "mariadb"
		}
		// The provisioned application account only owns its initial schema.
		// Use the container's separate root credential, including after an app
		// password rotation, and grant access only to the new target.
		statement := "SET SESSION sql_mode='NO_BACKSLASH_ESCAPES'; CREATE DATABASE `" + target + "` CHARACTER SET utf8mb4;"
		if user != "root" {
			quote := func(value string) string { return "'" + strings.ReplaceAll(value, "'", "''") + "'" }
			grantTarget := strings.ReplaceAll(target, "_", `\_`)
			grantPrefix := "GRANT ALL PRIVILEGES ON `"
			grantSuffix := "`.* TO `" + strings.ReplaceAll(user, "`", "``") + "`@`%`"
			if kind == databasesv1.DatabaseType_MYSQL {
				statement += " SET @restore_grant=CONCAT(" + quote(grantPrefix) + ",IF(@@GLOBAL.partial_revokes," + quote(target) + "," + quote(grantTarget) + ")," + quote(grantSuffix) + "); PREPARE restore_grant FROM @restore_grant; EXECUTE restore_grant; DEALLOCATE PREPARE restore_grant;"
			} else {
				statement += " " + grantPrefix + grantTarget + grantSuffix + ";"
			}
		}
		return mysqlAdministrationCommand([]string{binary, "--protocol=socket", "-u", "root", "--execute=" + statement}), nil, nil
	}
	return nil, nil, fmt.Errorf("unsupported backup engine")
}
func (s *Service) applyBackupRetention(ctx context.Context, id string) error {
	status := int32(2)
	rows, err := s.backupRepo.GetAll(ctx, id, &database.BackupFilters{Status: &status})
	if err != nil {
		return err
	}
	for index, item := range rows {
		if index < backupRetention() || item.RestoreStatus == "queued" || item.RestoreStatus == "restoring" {
			continue
		}
		item.Status = 4
		if err = s.backupRepo.Update(ctx, item); err != nil {
			return err
		}
		if err = configuredBackupStore().Delete(item.ID); err != nil {
			return err
		}
		if err = s.backupRepo.Delete(ctx, item.ID); err != nil {
			return err
		}
	}
	return nil
}

func mysqlBackupEngine(kind databasesv1.DatabaseType) bool {
	return kind == databasesv1.DatabaseType_MYSQL || kind == databasesv1.DatabaseType_MARIADB
}

// Root credentials stay inside the container and out of command arguments and
// service memory. Provisioning installs this credential independently of the
// application's current password. Positional arguments are never shell code.
func mysqlAdministrationCommand(command []string) []string {
	return append([]string{"sh", "-eu", "-c", `MYSQL_PWD="${MARIADB_ROOT_PASSWORD:-${MYSQL_ROOT_PASSWORD:-}}"; : "${MYSQL_PWD:?missing database administration credential}"; export MYSQL_PWD; exec "$@"`, "managed-backup"}, command...)
}
