package databases

import (
	"context"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"

	"connectrpc.com/connect"
	"databases-service/internal/provisioner"
	"github.com/google/uuid"
	"github.com/obiente/cloud/apps/shared/pkg/database"
	databasesv1 "github.com/obiente/cloud/apps/shared/proto/obiente/cloud/databases/v1"
	"google.golang.org/protobuf/proto"
)

func TestPostgresDumpRestoresSchemaAndData(t *testing.T) {
	if os.Getenv("OBIENTE_DOCKER_TESTS") != "1" {
		t.Skip("requires an isolated Docker daemon")
	}
	if os.Getenv("DOCKER_HOST") == "" {
		t.Fatal("explicit DOCKER_HOST required")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	name := "obiente-test-dump-" + uuid.NewString()
	t.Cleanup(func() {
		cleanup, stop := context.WithTimeout(context.Background(), 30*time.Second)
		defer stop()
		_ = exec.CommandContext(cleanup, "docker", "rm", "-fv", name).Run()
	})
	out, err := exec.CommandContext(ctx, "docker", "run", "-d", "--name", name, "--network", "none", "-e", "POSTGRES_HOST_AUTH_METHOD=trust", "postgres:17-alpine").CombinedOutput()
	if err != nil {
		t.Fatalf("create fixture: %v: %s", err, out)
	}
	p, err := provisioner.NewDockerProvisioner()
	if err != nil {
		t.Fatal(err)
	}
	defer p.Close()
	for {
		if _, err = p.ExecInDatabaseWithInputEnv(ctx, name, []string{"pg_isready", "-h", "127.0.0.1", "-U", "postgres"}, nil, nil); err == nil {
			break
		}
		select {
		case <-ctx.Done():
			t.Fatal(ctx.Err())
		case <-time.After(250 * time.Millisecond):
		}
	}
	run := func(sql string) string {
		t.Helper()
		output, err := p.ExecInDatabaseWithInputEnv(ctx, name, []string{"psql", "-U", "postgres", "-v", "ON_ERROR_STOP=1", "-At", "-c", sql}, nil, nil)
		if err != nil {
			t.Fatalf("fixture SQL: %v", err)
		}
		return strings.TrimSpace(output)
	}
	run("CREATE TABLE dump_fixture(id integer PRIMARY KEY, value text NOT NULL); INSERT INTO dump_fixture VALUES (1, 'preserved'), (2, 'Unicode: café');")
	command, env, err := exportDumpCommand(databasesv1.DatabaseType_POSTGRESQL, "postgres", "", "postgres", true, true)
	if err != nil {
		t.Fatal(err)
	}
	dump, err := p.ExecInDatabaseWithInputEnv(ctx, name, command, nil, env)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(dump, "CREATE TABLE") || !strings.Contains(dump, "café") {
		t.Fatal("dump omitted schema or data")
	}
	run("DROP TABLE dump_fixture")
	command, env, err = importDumpCommand(databasesv1.DatabaseType_POSTGRESQL, "postgres", "", "postgres")
	if err != nil {
		t.Fatal(err)
	}
	if _, err = p.ExecInDatabaseWithInputEnv(ctx, name, command, []byte(dump), env); err != nil {
		t.Fatal(err)
	}
	if got := run("SELECT id, value FROM dump_fixture ORDER BY id"); got != "1|preserved\n2|Unicode: café" {
		t.Fatalf("restored data differs: %q", got)
	}
	if _, err = p.ExecInDatabaseWithInputEnv(ctx, name, command, []byte("THIS IS NOT SQL;"), env); err == nil {
		t.Fatal("invalid restore reported success")
	}
	verifyManagedBackupRoundTrip(t, p, name, databasesv1.DatabaseType_POSTGRESQL, "postgres", "postgres", "")
	t.Log("native export/import preserved schema, rows, and Unicode; invalid SQL returned an error")
}

func TestMySQLFamilyDumpRestoresSchemaAndData(t *testing.T) {
	if os.Getenv("OBIENTE_DOCKER_TESTS") != "1" {
		t.Skip("requires an isolated Docker daemon")
	}
	if os.Getenv("DOCKER_HOST") == "" {
		t.Fatal("explicit DOCKER_HOST required")
	}
	for _, tc := range []struct {
		name, image, client string
		kind                databasesv1.DatabaseType
	}{
		{"MariaDB", "mariadb:11.4", "mariadb", databasesv1.DatabaseType_MARIADB},
		{"MySQL", "mysql:8.4", "mysql", databasesv1.DatabaseType_MYSQL},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
			defer cancel()
			name := "obiente-test-dump-" + uuid.NewString()
			t.Cleanup(func() {
				cleanup, stop := context.WithTimeout(context.Background(), 30*time.Second)
				defer stop()
				_ = exec.CommandContext(cleanup, "docker", "rm", "-fv", name).Run()
			})
			const password = "synthetic-dump-fixture-password"
			out, err := exec.CommandContext(ctx, "docker", "run", "-d", "--name", name, "--network", "none", "-e", "MYSQL_ROOT_PASSWORD="+password, "-e", "MYSQL_DATABASE=audit_dump", tc.image).CombinedOutput()
			if err != nil {
				t.Fatalf("create fixture: %v: %s", err, out)
			}
			p, err := provisioner.NewDockerProvisioner()
			if err != nil {
				t.Fatal(err)
			}
			defer p.Close()
			env := []string{"MYSQL_PWD=" + password}
			sqlCommand := []string{tc.client, "--default-character-set=utf8mb4", "-h", "127.0.0.1", "-u", "root", "-N", "-B", "audit_dump", "-e"}
			for {
				if _, err = p.ExecInDatabaseWithInputEnv(ctx, name, append(sqlCommand, "SELECT 1"), nil, env); err == nil {
					break
				}
				select {
				case <-ctx.Done():
					t.Fatal(ctx.Err())
				case <-time.After(250 * time.Millisecond):
				}
			}
			run := func(sql string) string {
				t.Helper()
				output, err := p.ExecInDatabaseWithInputEnv(ctx, name, append(sqlCommand, sql), nil, env)
				if err != nil {
					t.Fatalf("fixture SQL: %v", err)
				}
				return strings.TrimSpace(output)
			}
			run("CREATE TABLE dump_fixture(id integer PRIMARY KEY, value text NOT NULL) DEFAULT CHARSET=utf8mb4; INSERT INTO dump_fixture VALUES (1,'preserved'),(2,'Unicode: café'); CREATE VIEW dump_view AS SELECT id,value FROM dump_fixture; CREATE PROCEDURE dump_count() SELECT COUNT(*) FROM dump_fixture; CREATE EVENT dump_event ON SCHEDULE EVERY 1 DAY DISABLE DO INSERT INTO dump_fixture VALUES(3,'event');")
			command, dumpEnv, err := exportDumpCommand(tc.kind, "root", password, "audit_dump", true, true)
			if err != nil {
				t.Fatal(err)
			}
			dump, err := p.ExecInDatabaseWithInputEnv(ctx, name, command, nil, dumpEnv)
			if err != nil {
				t.Fatal(err)
			}
			run("DROP DATABASE audit_dump; CREATE DATABASE audit_dump CHARACTER SET utf8mb4")
			command, dumpEnv, err = importDumpCommand(tc.kind, "root", password, "audit_dump")
			if err != nil {
				t.Fatal(err)
			}
			if _, err = p.ExecInDatabaseWithInputEnv(ctx, name, command, []byte(dump), dumpEnv); err != nil {
				t.Fatal(err)
			}
			if got := run("SELECT id,value FROM dump_view ORDER BY id"); got != "1\tpreserved\n2\tUnicode: café" {
				t.Fatalf("restored data differs: %q", got)
			}
			if got := run("CALL dump_count()"); got != "2" {
				t.Fatalf("restored procedure differs: %q", got)
			}
			if got := run("SELECT STATUS FROM information_schema.EVENTS WHERE EVENT_SCHEMA='audit_dump' AND EVENT_NAME='dump_event'"); got != "DISABLED" {
				t.Fatalf("restored event differs: %q", got)
			}
			if _, err = p.ExecInDatabaseWithInputEnv(ctx, name, command, []byte("THIS IS NOT SQL;"), dumpEnv); err == nil {
				t.Fatal("invalid restore reported success")
			}
			verifyManagedBackupRoundTrip(t, p, name, tc.kind, "audit_dump", "root", password)
			t.Log("native export/import preserved rows, Unicode, view, procedure and event; invalid SQL failed")
		})
	}
}

func verifyManagedBackupRoundTrip(t *testing.T, p *provisioner.DockerProvisioner, container string, kind databasesv1.DatabaseType, source, user, password string) {
	t.Helper()
	s, ctx := newManagedBackupTestService(t)
	s.backupExec = p.ExecStream
	if err := database.DB.Model(&database.DatabaseInstance{}).Where("id = ?", "db-org-a-owner").Updates(map[string]any{"instance_id": container, "type": int32(kind)}).Error; err != nil {
		t.Fatal(err)
	}
	if err := database.DB.Model(&database.DatabaseConnection{}).Where("database_id = ?", "db-org-a-owner").Updates(map[string]any{"database_name": source, "username": user, "password": password}).Error; err != nil {
		t.Fatal(err)
	}
	id := createTestManagedBackup(t, s, ctx)
	if err := s.processBackup(ctx, id); err != nil {
		t.Fatal("managed export", err)
	}
	target := "managed_restore"
	restore := func() {
		t.Helper()
		_, err := s.RestoreBackup(ctx, connect.NewRequest(&databasesv1.RestoreBackupRequest{OrganizationId: "org-a", DatabaseId: "db-org-a-owner", BackupId: id, TargetDatabaseName: proto.String(target)}))
		if err != nil {
			t.Fatal(err)
		}
	}
	restore()
	if err := s.processBackup(ctx, id); err != nil {
		t.Fatal("managed restore", err)
	}
	verify := func(databaseName string) {
		t.Helper()
		var command, env []string
		query := "SELECT value FROM dump_fixture WHERE id=2"
		if kind == databasesv1.DatabaseType_POSTGRESQL {
			command = []string{"psql", "-U", user, "-d", databaseName, "-At", "-c", query}
			env = []string{"PGPASSWORD=" + password}
		} else {
			binary := "mysql"
			if kind == databasesv1.DatabaseType_MARIADB {
				binary = "mariadb"
			}
			command = []string{binary, "--default-character-set=utf8mb4", "-u", user, "--database=" + databaseName, "-N", "-B", "--execute=" + query}
			env = []string{"MYSQL_PWD=" + password}
		}
		output, err := p.ExecInDatabaseWithInputEnv(ctx, container, command, nil, env)
		if err != nil || strings.TrimSpace(output) != "Unicode: café" {
			t.Fatalf("managed restore data: %q %v", output, err)
		}
	}
	verify(target)
	verify(source)
	restore()
	if err := s.processBackup(ctx, id); err == nil {
		t.Fatal("existing restore target was overwritten")
	}
	verify(target)
	verify(source)
	t.Log("managed shared-volume backup restored into a new database; original and existing targets preserved")
}
