package databases

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"testing"

	"connectrpc.com/connect"
	"github.com/obiente/cloud/apps/shared/pkg/auth"
	"github.com/obiente/cloud/apps/shared/pkg/database"
	authv1 "github.com/obiente/cloud/apps/shared/proto/obiente/cloud/auth/v1"
	databasesv1 "github.com/obiente/cloud/apps/shared/proto/obiente/cloud/databases/v1"
	"google.golang.org/protobuf/proto"
)

func newManagedBackupTestService(t *testing.T) (*Service, context.Context) {
	t.Helper()
	db := newDatabaseServiceTestDB(t)
	seedDatabaseServiceIsolationData(t, db)
	if err := db.AutoMigrate(&database.DatabaseBackup{}, &database.DatabaseConnection{}); err != nil {
		t.Fatal(err)
	}
	if err := db.Model(&database.DatabaseInstance{}).Where("id = ?", "db-org-a-owner").Updates(map[string]any{"instance_id": "test-container", "desired_state": "running"}).Error; err != nil {
		t.Fatal(err)
	}
	conn := &database.DatabaseConnection{ID: "connection-example", DatabaseID: "db-org-a-owner", Username: "root", Password: "synthetic-password", DatabaseName: "primary_db"}
	if err := db.Create(conn).Error; err != nil {
		t.Fatal(err)
	}
	directory := t.TempDir()
	if err := os.WriteFile(filepath.Join(directory, ".obiente-backup-volume"), []byte("obiente-cloud-backups-v1\n"), 0600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("DATABASE_BACKUP_DIRECTORY", directory)
	t.Setenv("DATABASE_BACKUP_KEEP", "2")
	svc := &Service{permissionChecker: auth.NewPermissionChecker(), repo: database.NewDatabaseRepository(db, nil), connRepo: database.NewDatabaseConnectionRepository(db), backupRepo: database.NewDatabaseBackupRepository(db)}
	svc.backupExec = func(ctx context.Context, id string, cmd []string, input io.Reader, output io.Writer, env []string) error {
		if input == nil && cmd[0] == "pg_dump" {
			_, err := io.WriteString(output, "CREATE TABLE example(id integer);")
			return err
		}
		if input != nil {
			_, err := io.Copy(io.Discard, input)
			return err
		}
		return nil
	}
	return svc, auth.WithUser(context.Background(), &authv1.User{Id: "user-org-a", Email: "owner@example.test"})
}
func createTestManagedBackup(t *testing.T, s *Service, ctx context.Context) string {
	t.Helper()
	response, err := s.CreateBackup(ctx, connect.NewRequest(&databasesv1.CreateBackupRequest{OrganizationId: "org-a", DatabaseId: "db-org-a-owner"}))
	if err != nil {
		t.Fatal(err)
	}
	if response.Msg.Backup.Status != databasesv1.DatabaseBackupStatus_BACKUP_CREATING {
		t.Fatal("backup claimed completion before export")
	}
	return response.Msg.Backup.Id
}
func TestManagedBackupLifecycleRetentionAndRestart(t *testing.T) {
	s, ctx := newManagedBackupTestService(t)
	id := createTestManagedBackup(t, s, ctx)
	if _, err := s.CreateBackup(ctx, connect.NewRequest(&databasesv1.CreateBackupRequest{OrganizationId: "org-a", DatabaseId: "db-org-a-owner"})); connect.CodeOf(err) != connect.CodeFailedPrecondition {
		t.Fatalf("duplicate pending operation: %v", err)
	}
	// A replacement worker discovers and completes persisted work.
	restarted := *s
	restarted.processBackupQueue(ctx)
	item, err := s.backupRepo.GetByID(ctx, id)
	if err != nil || item.Status != 2 || item.Checksum == "" {
		t.Fatalf("backup completion: %+v %v", item, err)
	}
	response, err := s.RestoreBackup(ctx, connect.NewRequest(&databasesv1.RestoreBackupRequest{OrganizationId: "org-a", DatabaseId: "db-org-a-owner", BackupId: id, TargetDatabaseName: proto.String("restored_copy")}))
	if err != nil {
		t.Fatal(err)
	}
	if response.Msg.Backup.RestoreStatus != "queued" {
		t.Fatal("restore claimed completion before import")
	}
	if err = s.processBackup(ctx, id); err != nil {
		t.Fatal(err)
	}
	item, _ = s.backupRepo.GetByID(ctx, id)
	if item.RestoreStatus != "completed" || item.RestoredAt == nil {
		t.Fatalf("restore state: %+v", item)
	}
	for i := 0; i < 2; i++ {
		next := createTestManagedBackup(t, s, ctx)
		if err = s.processBackup(ctx, next); err != nil {
			t.Fatal(err)
		}
	}
	rows, err := s.backupRepo.GetAll(ctx, "db-org-a-owner", nil)
	if err != nil || len(rows) != 2 {
		t.Fatalf("retention: %d %v", len(rows), err)
	}
	if _, err = os.Stat(filepath.Join(os.Getenv("DATABASE_BACKUP_DIRECTORY"), id+".sql.gz")); !os.IsNotExist(err) {
		t.Fatal("retention did not remove the expired file", err)
	}
	remove := rows[0].ID
	if _, err = s.DeleteBackup(ctx, connect.NewRequest(&databasesv1.DeleteBackupRequest{OrganizationId: "org-a", DatabaseId: "db-org-a-owner", BackupId: remove})); err != nil {
		t.Fatal(err)
	}
	s.processBackupQueue(ctx)
	if _, err = s.backupRepo.GetByID(ctx, remove); err == nil {
		t.Fatal("deletion did not finish")
	}
}
func TestManagedBackupRecoveryDoesNotRepeatAnInterruptedRestore(t *testing.T) {
	s, ctx := newManagedBackupTestService(t)
	id := createTestManagedBackup(t, s, ctx)
	if err := s.processBackup(ctx, id); err != nil {
		t.Fatal(err)
	}
	item, _ := s.backupRepo.GetByID(ctx, id)
	item.RestoreStatus = "restoring"
	item.RestoreTarget = "partial_copy"
	if err := s.backupRepo.Update(ctx, item); err != nil {
		t.Fatal(err)
	}
	s.backupExec = func(context.Context, string, []string, io.Reader, io.Writer, []string) error {
		t.Fatal("interrupted import repeated")
		return nil
	}
	s.processBackupQueue(ctx)
	item, _ = s.backupRepo.GetByID(ctx, id)
	if item.RestoreStatus != "failed" || item.RestoreError == nil {
		t.Fatalf("interruption hidden: %+v", item)
	}
}
func TestManagedBackupRechecksPermissionAndRejectsCorruptRestore(t *testing.T) {
	s, ctx := newManagedBackupTestService(t)
	id := createTestManagedBackup(t, s, ctx)
	if err := database.DB.Model(&database.Organization{}).Where("id = ?", "org-a").Update("status", "suspended").Error; err != nil {
		t.Fatal(err)
	}
	if err := s.processBackup(ctx, id); err == nil {
		t.Fatal("suspended organization exported")
	}
	item, _ := s.backupRepo.GetByID(ctx, id)
	if item.Status != 3 {
		t.Fatal("failed job not recorded")
	}
	if err := database.DB.Model(&database.Organization{}).Where("id = ?", "org-a").Update("status", "active").Error; err != nil {
		t.Fatal(err)
	}
	id = createTestManagedBackup(t, s, ctx)
	if err := s.processBackup(ctx, id); err != nil {
		t.Fatal(err)
	}
	for _, target := range []string{"primary_db", "../outside", "name;DROP DATABASE primary_db"} {
		_, err := s.RestoreBackup(ctx, connect.NewRequest(&databasesv1.RestoreBackupRequest{OrganizationId: "org-a", DatabaseId: "db-org-a-owner", BackupId: id, TargetDatabaseName: &target}))
		if err == nil {
			t.Fatalf("unsafe restore target accepted: %s", target)
		}
	}
	if err := os.WriteFile(filepath.Join(os.Getenv("DATABASE_BACKUP_DIRECTORY"), id+".sql.gz"), []byte("corrupt"), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := s.RestoreBackup(ctx, connect.NewRequest(&databasesv1.RestoreBackupRequest{OrganizationId: "org-a", DatabaseId: "db-org-a-owner", BackupId: id, TargetDatabaseName: proto.String("safe_new_name")})); err != nil {
		t.Fatal(err)
	}
	s.backupExec = func(context.Context, string, []string, io.Reader, io.Writer, []string) error {
		t.Fatal("corrupt data reached database")
		return nil
	}
	if err := s.processBackup(ctx, id); err == nil {
		t.Fatal("corrupt backup restored")
	}
	item, _ = s.backupRepo.GetByID(ctx, id)
	if item.RestoreError == nil || !strings.Contains(*item.RestoreError, "No target database was created") {
		t.Fatal("verification failure did not explain that data was preserved")
	}
}
func TestManagedBackupTenantIsolationAndMissingVolume(t *testing.T) {
	s, ctx := newManagedBackupTestService(t)
	id := createTestManagedBackup(t, s, ctx)
	if err := s.processBackup(ctx, id); err != nil {
		t.Fatal(err)
	}
	other := auth.WithUser(context.Background(), &authv1.User{Id: "user-org-b"})
	if _, err := s.GetBackup(other, connect.NewRequest(&databasesv1.GetBackupRequest{OrganizationId: "org-b", DatabaseId: "db-org-a-owner", BackupId: id})); connect.CodeOf(err) != connect.CodePermissionDenied {
		t.Fatalf("cross-tenant read: %v", err)
	}
	if _, err := s.DeleteBackup(other, connect.NewRequest(&databasesv1.DeleteBackupRequest{OrganizationId: "org-b", DatabaseId: "db-org-b-owner", BackupId: id})); connect.CodeOf(err) != connect.CodeNotFound {
		t.Fatalf("cross-resource delete: %v", err)
	}
	t.Setenv("DATABASE_BACKUP_DIRECTORY", t.TempDir())
	response, err := s.ListBackups(ctx, connect.NewRequest(&databasesv1.ListBackupsRequest{OrganizationId: "org-a", DatabaseId: "db-org-a-owner"}))
	if err != nil {
		t.Fatal(err)
	}
	if response.Msg.ManagedAvailable || response.Msg.UnavailableReason == "" {
		t.Fatal("missing volume advertised as ready")
	}
	if _, err = s.CreateBackup(ctx, connect.NewRequest(&databasesv1.CreateBackupRequest{OrganizationId: "org-a", DatabaseId: "db-org-a-owner"})); connect.CodeOf(err) != connect.CodeFailedPrecondition {
		t.Fatalf("missing volume accepted: %v", err)
	}
}
func TestManagedBackupConcurrentWorkersExportOnce(t *testing.T) {
	s, ctx := newManagedBackupTestService(t)
	id := createTestManagedBackup(t, s, ctx)
	var calls atomic.Int32
	s.backupExec = func(_ context.Context, _ string, _ []string, _ io.Reader, out io.Writer, _ []string) error {
		calls.Add(1)
		_, err := io.WriteString(out, "CREATE TABLE example(id int);")
		return err
	}
	var wait sync.WaitGroup
	for i := 0; i < 2; i++ {
		wait.Add(1)
		go func() {
			defer wait.Done()
			if err := s.processBackup(ctx, id); err != nil {
				t.Error(err)
			}
		}()
	}
	wait.Wait()
	if calls.Load() != 1 {
		t.Fatalf("export ran %d times", calls.Load())
	}
}
func TestManagedBackupExportFailureIsRecorded(t *testing.T) {
	s, ctx := newManagedBackupTestService(t)
	id := createTestManagedBackup(t, s, ctx)
	s.backupExec = func(_ context.Context, _ string, _ []string, _ io.Reader, out io.Writer, _ []string) error {
		_, _ = io.WriteString(out, "partial")
		return fmt.Errorf("Docker unavailable")
	}
	if err := s.processBackup(ctx, id); err == nil {
		t.Fatal("failed export succeeded")
	}
	item, _ := s.backupRepo.GetByID(ctx, id)
	if item.Status != 3 || item.CompletedAt != nil {
		t.Fatalf("incorrect failure state: %+v", item)
	}
	files, _ := os.ReadDir(os.Getenv("DATABASE_BACKUP_DIRECTORY"))
	for _, file := range files {
		if strings.HasSuffix(file.Name(), ".gz") || strings.HasSuffix(file.Name(), ".partial") {
			t.Fatal("partial export retained", file.Name())
		}
	}
}

func TestManagedBackupDeletionRetriesAfterVolumeReturns(t *testing.T) {
	s, ctx := newManagedBackupTestService(t)
	id := createTestManagedBackup(t, s, ctx)
	if err := s.processBackup(ctx, id); err != nil {
		t.Fatal(err)
	}
	_, err := s.DeleteBackup(ctx, connect.NewRequest(&databasesv1.DeleteBackupRequest{OrganizationId: "org-a", DatabaseId: "db-org-a-owner", BackupId: id}))
	if err != nil {
		t.Fatal(err)
	}
	volume := os.Getenv("DATABASE_BACKUP_DIRECTORY")
	t.Setenv("DATABASE_BACKUP_DIRECTORY", t.TempDir())
	if err = s.processBackup(ctx, id); err == nil {
		t.Fatal("missing volume reported successful deletion")
	}
	item, err := s.backupRepo.GetByID(ctx, id)
	if err != nil || item.Status != 4 {
		t.Fatal("deletion state lost", err)
	}
	t.Setenv("DATABASE_BACKUP_DIRECTORY", volume)
	s.processBackupQueue(ctx)
	if _, err = s.backupRepo.GetByID(ctx, id); err == nil {
		t.Fatal("deletion did not recover")
	}
}
func TestManagedBackupFailurePreservesPreviousRecoveryPoint(t *testing.T) {
	s, ctx := newManagedBackupTestService(t)
	t.Setenv("DATABASE_BACKUP_KEEP", "1")
	previous := createTestManagedBackup(t, s, ctx)
	if err := s.processBackup(ctx, previous); err != nil {
		t.Fatal(err)
	}
	next := createTestManagedBackup(t, s, ctx)
	s.backupExec = func(context.Context, string, []string, io.Reader, io.Writer, []string) error {
		return fmt.Errorf("Docker unavailable")
	}
	if err := s.processBackup(ctx, next); err == nil {
		t.Fatal("failed backup succeeded")
	}
	item, err := s.backupRepo.GetByID(ctx, previous)
	if err != nil {
		t.Fatal("recovery point was deleted", err)
	}
	reader, err := configuredBackupStore().Read(ctx, previous, item.Checksum, item.SizeBytes)
	if err != nil {
		t.Fatal("previous backup no longer readable", err)
	}
	_ = reader.Close()
}

func TestManagedBackupSizeFailureExplainsTheLimit(t *testing.T) {
	s, ctx := newManagedBackupTestService(t)
	t.Setenv("DATABASE_BACKUP_MAX_BYTES", "4")
	id := createTestManagedBackup(t, s, ctx)
	if err := s.processBackup(ctx, id); err == nil {
		t.Fatal("oversized backup accepted")
	}
	item, err := s.backupRepo.GetByID(ctx, id)
	if err != nil {
		t.Fatal(err)
	}
	if item.ErrorMessage == nil || !strings.Contains(*item.ErrorMessage, "size limit") {
		t.Fatalf("unhelpful size error: %+v", item.ErrorMessage)
	}
}
