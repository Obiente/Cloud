package service

import (
	"context"
	"io"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/obiente/cloud/apps/shared/pkg/database"
	pkgsftp "github.com/pkg/sftp"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestModerationFileTransferLoginAndOpenHandles(t *testing.T) {
	for _, restriction := range []string{"banned_user", "suspended_user", "banned_org", "suspended_org", "revoked_credential", "expired_credential"} {
		t.Run(restriction, func(t *testing.T) {
			db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
			if err != nil {
				t.Fatal(err)
			}
			raw, _ := db.DB()
			raw.SetMaxOpenConns(1)
			previous := database.DB
			database.DB = db
			t.Cleanup(func() { database.DB = previous; raw.Close() })
			if err := db.AutoMigrate(&database.Organization{}, &database.UserBan{}, &database.GameServer{}); err != nil {
				t.Fatal(err)
			}
			// SQLite needs datetime columns and CURRENT_TIMESTAMP in place of the
			// production model's PostgreSQL timestamptz columns and now() defaults.
			if err := db.Exec(`CREATE TABLE file_transfer_credentials (
				id TEXT PRIMARY KEY, name TEXT NOT NULL, key_hash TEXT NOT NULL UNIQUE,
				user_id TEXT NOT NULL, organization_id TEXT NOT NULL,
				resource_type TEXT NOT NULL, resource_id TEXT NOT NULL, scopes TEXT NOT NULL,
				last_used_at DATETIME, expires_at DATETIME, revoked_at DATETIME,
				created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
				updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP, deleted_at DATETIME
			)`).Error; err != nil {
				t.Fatal(err)
			}
			db.Create(&database.Organization{ID: "org", Slug: "org", Status: "active"})
			db.Create(&database.GameServer{ID: "gs", OrganizationID: "org"})
			secret := "synthetic-transfer-secret"
			credential := &database.FileTransferCredential{ID: "credential", Name: "Test", UserID: "user", OrganizationID: "org", ResourceType: "gameserver", ResourceID: "gs", Scopes: "read,write", KeyHash: database.HashFileTransferSecret(secret)}
			if err := db.Create(credential).Error; err != nil {
				t.Fatal(err)
			}
			root := t.TempDir()
			volume := filepath.Join(root, "gameserver-gs-data")
			if err := os.Mkdir(volume, 0700); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(filepath.Join(volume, "existing.txt"), []byte("retained data"), 0600); err != nil {
				t.Fatal(err)
			}
			authenticator := NewAuthenticator(root)
			session, err := authenticator.Authenticate(context.Background(), secret)
			if err != nil {
				t.Fatal(err)
			}
			handler := newSFTPHandler(session)
			reader, err := handler.Fileread(&pkgsftp.Request{Filepath: "existing.txt"})
			if err != nil {
				t.Fatal(err)
			}
			defer reader.(io.Closer).Close()
			writer, err := handler.Filewrite(&pkgsftp.Request{Filepath: "new.txt"})
			if err != nil {
				t.Fatal(err)
			}
			defer writer.(io.Closer).Close()
			list, err := handler.Filelist(&pkgsftp.Request{Method: "List", Filepath: "/"})
			if err != nil {
				t.Fatal(err)
			}
			switch restriction {
			case "banned_user", "suspended_user":
				typ := "banned"
				if restriction == "suspended_user" {
					typ = "suspended"
				}
				db.Create(&database.UserBan{ID: "ban", UserID: "user", Type: typ, BannedAt: time.Now()})
			case "banned_org", "suspended_org":
				status := "banned"
				if restriction == "suspended_org" {
					status = "suspended"
				}
				db.Model(&database.Organization{}).Where("id = ?", "org").Update("status", status)
			case "revoked_credential":
				db.Model(credential).Update("revoked_at", time.Now())
			case "expired_credential":
				db.Model(credential).Update("expires_at", time.Now().Add(-time.Hour))
			}
			if _, err := authenticator.Authenticate(context.Background(), secret); err == nil {
				t.Fatal("restricted login accepted")
			}
			if _, err := reader.ReadAt(make([]byte, 4), 0); err == nil {
				t.Fatal("open read handle bypassed restriction")
			}
			if _, err := writer.WriteAt([]byte("blocked"), 0); err == nil {
				t.Fatal("open write handle bypassed restriction")
			}
			if _, err := list.ListAt(make([]os.FileInfo, 10), 0); err == nil {
				t.Fatal("open directory listing bypassed restriction")
			}
			if err := handler.Filecmd(&pkgsftp.Request{Method: "Remove", Filepath: "existing.txt"}); err == nil {
				t.Fatal("delete bypassed restriction")
			}
			if data, err := os.ReadFile(filepath.Join(volume, "existing.txt")); err != nil || string(data) != "retained data" {
				t.Fatal("retained file changed")
			}
		})
	}
}
