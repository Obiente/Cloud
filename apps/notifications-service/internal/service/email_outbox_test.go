package notifications

import (
	"context"
	"errors"
	"fmt"
	"os"
	"sync/atomic"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/obiente/cloud/apps/shared/pkg/auth"
	"github.com/obiente/cloud/apps/shared/pkg/database"
	authv1 "github.com/obiente/cloud/apps/shared/proto/obiente/cloud/auth/v1"
	notificationsv1 "github.com/obiente/cloud/apps/shared/proto/obiente/cloud/notifications/v1"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func emailOutboxDB(t *testing.T) *gorm.DB {
	t.Helper()
	dsn := os.Getenv("OBIENTE_TEST_POSTGRES_DSN")
	if dsn == "" {
		t.Skip("requires an isolated PostgreSQL database")
	}
	base, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	schema := fmt.Sprintf("notification_outbox_%d", time.Now().UnixNano())
	if err := base.Exec("CREATE SCHEMA " + schema).Error; err != nil {
		t.Fatal(err)
	}
	db, err := gorm.Open(postgres.Open(dsn+" search_path="+schema), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	previous := database.DB
	database.DB = db
	t.Cleanup(func() {
		database.DB = previous
		raw, _ := db.DB()
		raw.Close()
		base.Exec("DROP SCHEMA " + schema + " CASCADE")
		raw, _ = base.DB()
		raw.Close()
	})
	if err := db.AutoMigrate(&database.Notification{}, &EmailDelivery{}); err != nil {
		t.Fatal(err)
	}
	return db
}

func queueTestNotification(t *testing.T, id string) {
	t.Helper()
	if err := persistNotification(context.Background(), &database.Notification{ID: id, UserID: "user-synthetic", Type: "SYSTEM", Severity: "HIGH", Title: "Synthetic notification", Message: "Test delivery", CreatedAt: time.Now()}); err != nil {
		t.Fatal(err)
	}
}
func outboxDelivery(t *testing.T, db *gorm.DB, id string) EmailDelivery {
	t.Helper()
	var delivery EmailDelivery
	if err := db.First(&delivery, "notification_id = ?", id).Error; err != nil {
		t.Fatal(err)
	}
	return delivery
}

func TestEmailOutboxAndNotificationCommitTogether(t *testing.T) {
	db := emailOutboxDB(t)
	db.Callback().Create().Before("gorm:create").Register("test:outbox_failure", func(tx *gorm.DB) {
		if tx.Statement.Table == "notification_email_deliveries" {
			tx.AddError(errors.New("synthetic outbox failure"))
		}
	})
	ctx := auth.WithUser(context.Background(), &authv1.User{Id: "admin-synthetic", Roles: []string{auth.RoleSuperAdmin}})
	request := connect.NewRequest(&notificationsv1.CreateNotificationRequest{UserId: "user-synthetic", Title: "Synthetic", Message: "Test"})
	if _, err := (&Service{}).CreateNotification(ctx, request); connect.CodeOf(err) != connect.CodeInternal {
		t.Fatalf("error=%v", err)
	}
	for _, model := range []any{&database.Notification{}, &EmailDelivery{}} {
		var count int64
		db.Model(model).Count(&count)
		if count != 0 {
			t.Fatalf("partial %T count=%d", model, count)
		}
	}
	db.Callback().Create().Remove("test:outbox_failure")
	response, err := (&Service{}).CreateNotification(ctx, request)
	if err != nil {
		t.Fatal(err)
	}
	if got := outboxDelivery(t, db, response.Msg.Notification.Id); got.Status != "pending" {
		t.Fatalf("status=%s", got.Status)
	}
}

func TestEmailOutboxRetainsBurstBeyondFormerQueueLimit(t *testing.T) {
	db := emailOutboxDB(t)
	for i := range 300 {
		queueTestNotification(t, fmt.Sprintf("notification-%d", i))
	}
	var count int64
	if err := db.Model(&EmailDelivery{}).Where("status = ?", "pending").Count(&count).Error; err != nil {
		t.Fatal(err)
	}
	if count != 300 {
		t.Fatalf("persisted=%d", count)
	}
}

func TestEmailOutboxRetriesAfterWorkerRestart(t *testing.T) {
	db := emailOutboxDB(t)
	queueTestNotification(t, "retry")
	now := time.Now().Add(time.Second)
	first := &emailDispatcher{db: db, now: func() time.Time { return now }, send: func(context.Context, *database.Notification) error { return errors.New("synthetic SMTP outage") }}
	if worked, err := first.deliverNext(context.Background()); err != nil || !worked {
		t.Fatalf("worked=%v err=%v", worked, err)
	}
	entry := outboxDelivery(t, db, "retry")
	if entry.Status != "pending" || entry.Attempts != 1 || !entry.AvailableAt.After(now) || entry.LastError == "" {
		t.Fatalf("entry=%+v", entry)
	}
	now = entry.AvailableAt.Add(time.Second)
	second := &emailDispatcher{db: db, now: func() time.Time { return now }, send: func(context.Context, *database.Notification) error { return nil }}
	if worked, err := second.deliverNext(context.Background()); err != nil || !worked {
		t.Fatalf("worked=%v err=%v", worked, err)
	}
	entry = outboxDelivery(t, db, "retry")
	if entry.Status != "completed" || entry.Attempts != 2 {
		t.Fatalf("entry=%+v", entry)
	}
}

func TestEmailOutboxReclaimsExpiredLease(t *testing.T) {
	db := emailOutboxDB(t)
	queueTestNotification(t, "lease")
	now := time.Now().Add(time.Second)
	until := now.Add(time.Minute)
	if err := db.Model(&EmailDelivery{}).Where("notification_id = ?", "lease").Updates(map[string]any{"status": "processing", "lease_token": "old-process", "lease_until": until}).Error; err != nil {
		t.Fatal(err)
	}
	d := &emailDispatcher{db: db, now: func() time.Time { return now }, send: func(context.Context, *database.Notification) error { return nil }}
	if worked, err := d.deliverNext(context.Background()); err != nil || worked {
		t.Fatalf("active lease stolen: %v %v", worked, err)
	}
	now = until.Add(time.Second)
	if worked, err := d.deliverNext(context.Background()); err != nil || !worked {
		t.Fatalf("expired lease not recovered: %v %v", worked, err)
	}
	if entry := outboxDelivery(t, db, "lease"); entry.Status != "completed" {
		t.Fatalf("status=%s", entry.Status)
	}
}

func TestEmailOutboxConcurrentWorkersClaimOnce(t *testing.T) {
	db := emailOutboxDB(t)
	queueTestNotification(t, "concurrent")
	arrived := make(chan struct{})
	release := make(chan struct{})
	defer close(release)
	var sends atomic.Int32
	d := &emailDispatcher{db: db, now: time.Now, send: func(context.Context, *database.Notification) error {
		sends.Add(1)
		close(arrived)
		<-release
		return nil
	}}
	result := make(chan error, 1)
	go func() { _, err := d.deliverNext(context.Background()); result <- err }()
	// The defer releases the sender before cleanup waits for its acknowledgement.
	t.Cleanup(func() {
		select {
		case err := <-result:
			if err != nil {
				t.Error(err)
			}
		case <-time.After(5 * time.Second):
			t.Error("worker did not finish")
		}
	})
	select {
	case <-arrived:
	case <-time.After(5 * time.Second):
		t.Fatal("worker did not claim notification")
	}
	if worked, err := d.deliverNext(context.Background()); err != nil || worked {
		t.Fatalf("concurrent claim: %v %v", worked, err)
	}
	if sends.Load() != 1 {
		t.Fatalf("sends=%d", sends.Load())
	}
}

func TestEmailOutboxShutdownKeepsWorkForRestart(t *testing.T) {
	db := emailOutboxDB(t)
	queueTestNotification(t, "shutdown")
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	d := &emailDispatcher{db: db, now: time.Now, send: func(ctx context.Context, _ *database.Notification) error { cancel(); return ctx.Err() }}
	if worked, err := d.deliverNext(ctx); err != nil || !worked {
		t.Fatalf("worked=%v err=%v", worked, err)
	}
	entry := outboxDelivery(t, db, "shutdown")
	if entry.Status != "pending" || entry.LeaseUntil != nil {
		t.Fatalf("entry=%+v", entry)
	}
	d.send = func(context.Context, *database.Notification) error { return nil }
	if worked, err := d.deliverNext(context.Background()); err != nil || !worked {
		t.Fatalf("restart worked=%v err=%v", worked, err)
	}
}

func TestEmailOutboxRecoversLostAcknowledgement(t *testing.T) {
	db := emailOutboxDB(t)
	queueTestNotification(t, "ack")
	now := time.Now().Add(time.Second)
	d := &emailDispatcher{db: db, now: func() time.Time { return now }, send: func(context.Context, *database.Notification) error { return nil }}
	db.Callback().Update().Before("gorm:update").Register("test:ack_failure", func(tx *gorm.DB) {
		if updates, ok := tx.Statement.Dest.(map[string]any); ok && updates["status"] == "completed" {
			tx.AddError(errors.New("synthetic acknowledgement outage"))
		}
	})
	if _, err := d.deliverNext(context.Background()); err == nil {
		t.Fatal("acknowledgement failure hidden")
	}
	if entry := outboxDelivery(t, db, "ack"); entry.Status != "processing" {
		t.Fatal("failed acknowledgement lost delivery")
	}
	db.Callback().Update().Remove("test:ack_failure")
	now = now.Add(emailDeliveryLease + time.Second)
	if worked, err := d.deliverNext(context.Background()); err != nil || !worked {
		t.Fatalf("recovery=%v %v", worked, err)
	}
	if entry := outboxDelivery(t, db, "ack"); entry.Status != "completed" {
		t.Fatalf("status=%s", entry.Status)
	}
}

func TestEmailRetryDelayIsBounded(t *testing.T) {
	if emailRetryDelay(1) != 30*time.Second || emailRetryDelay(1000) > 15*time.Minute {
		t.Fatal("unexpected retry schedule")
	}
}

func TestEmailOutboxRejectsStaleAcknowledgement(t *testing.T) {
	db := emailOutboxDB(t)
	queueTestNotification(t, "stale")
	d := &emailDispatcher{db: db, now: time.Now, send: func(context.Context, *database.Notification) error {
		// Simulate recovery by another process after this sender's lease expired.
		return db.Model(&EmailDelivery{}).Where("notification_id = ?", "stale").Update("lease_token", "replacement-worker").Error
	}}
	if worked, err := d.deliverNext(context.Background()); !worked || err == nil {
		t.Fatalf("stale acknowledgement accepted: worked=%v err=%v", worked, err)
	}
	if entry := outboxDelivery(t, db, "stale"); entry.Status != "processing" || entry.LeaseToken != "replacement-worker" {
		t.Fatalf("replacement worker's lease overwritten: %+v", entry)
	}
}

func TestEmailOutboxRetainsInvalidPayloadForInspection(t *testing.T) {
	db := emailOutboxDB(t)
	queueTestNotification(t, "invalid")
	if err := db.Model(&EmailDelivery{}).Where("notification_id = ?", "invalid").Update("payload", "{}").Error; err != nil {
		t.Fatal(err)
	}
	d := &emailDispatcher{db: db, now: time.Now, send: func(context.Context, *database.Notification) error {
		t.Fatal("invalid payload sent")
		return nil
	}}
	if worked, err := d.deliverNext(context.Background()); !worked || err != nil {
		t.Fatalf("worked=%v err=%v", worked, err)
	}
	if entry := outboxDelivery(t, db, "invalid"); entry.Status != "failed" || entry.LastError == "" {
		t.Fatalf("invalid payload discarded: %+v", entry)
	}
}
