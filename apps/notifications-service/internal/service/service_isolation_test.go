package notifications

import (
	"context"
	"slices"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/obiente/cloud/apps/shared/pkg/auth"
	"github.com/obiente/cloud/apps/shared/pkg/database"
	authv1 "github.com/obiente/cloud/apps/shared/proto/obiente/cloud/auth/v1"
	notificationsv1 "github.com/obiente/cloud/apps/shared/proto/obiente/cloud/notifications/v1"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestNotificationServiceUserIsolation(t *testing.T) {
	db := newNotificationServiceTestDB(t)
	service := &Service{}
	seedNotificationServiceIsolationData(t, db)

	ctx := auth.WithUser(context.Background(), &authv1.User{
		Id:    "user-org-a",
		Email: "user-org-a@example.com",
	})

	list, err := service.ListNotifications(ctx, connect.NewRequest(&notificationsv1.ListNotificationsRequest{}))
	if err != nil {
		t.Fatalf("list notifications: %v", err)
	}
	if got := notificationIDs(list.Msg.Notifications); !slices.Equal(got, []string{"notif-user-a"}) {
		t.Fatalf("list returned %v, want only visible notifications for user", got)
	}
	if got := list.Msg.Pagination.GetTotal(); got != 1 {
		t.Fatalf("notification total = %d, want 1", got)
	}

	_, err = service.GetNotification(ctx, connect.NewRequest(&notificationsv1.GetNotificationRequest{
		NotificationId: "notif-user-b",
	}))
	if err == nil {
		t.Fatal("cross-user get succeeded, want not found")
	}
	if connect.CodeOf(err) != connect.CodeNotFound {
		t.Fatalf("cross-user get code = %v, want %v: %v", connect.CodeOf(err), connect.CodeNotFound, err)
	}

	_, err = service.MarkAsRead(ctx, connect.NewRequest(&notificationsv1.MarkAsReadRequest{
		NotificationId: "notif-user-b",
	}))
	if err == nil {
		t.Fatal("cross-user mark-as-read succeeded, want not found")
	}
	if connect.CodeOf(err) != connect.CodeNotFound {
		t.Fatalf("cross-user mark-as-read code = %v, want %v: %v", connect.CodeOf(err), connect.CodeNotFound, err)
	}

	var userBNotification database.Notification
	if err := db.First(&userBNotification, "id = ?", "notif-user-b").Error; err != nil {
		t.Fatalf("fetch user-b notification: %v", err)
	}
	if userBNotification.Read {
		t.Fatal("cross-user mark-as-read changed user-b notification")
	}
}

func newNotificationServiceTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite db: %v", err)
	}
	if err := db.AutoMigrate(&database.Notification{}); err != nil {
		t.Fatalf("migrate sqlite db: %v", err)
	}

	previousDB := database.DB
	database.DB = db
	t.Cleanup(func() {
		database.DB = previousDB
	})

	return db
}

func seedNotificationServiceIsolationData(t *testing.T, db *gorm.DB) {
	t.Helper()

	now := time.Now().UTC()
	orgA := "org-a"
	notifications := []*database.Notification{
		testNotification("notif-user-a", "user-org-a", &orgA, "visible", false, now),
		testNotification("notif-user-a-client-only", "user-org-a", &orgA, "client only", true, now),
		testNotification("notif-user-b", "user-org-b", &orgA, "other user", false, now),
	}
	for _, notification := range notifications {
		if err := db.Create(notification).Error; err != nil {
			t.Fatalf("seed notification %s: %v", notification.ID, err)
		}
	}
}

func testNotification(id, userID string, orgID *string, title string, clientOnly bool, now time.Time) *database.Notification {
	return &database.Notification{
		ID:             id,
		UserID:         userID,
		OrganizationID: orgID,
		Type:           "INFO",
		Severity:       "LOW",
		Title:          title,
		Message:        title + " message",
		Read:           false,
		Metadata:       "{}",
		ClientOnly:     clientOnly,
		CreatedAt:      now,
		UpdatedAt:      now,
	}
}

func notificationIDs(notifications []*notificationsv1.Notification) []string {
	ids := make([]string, 0, len(notifications))
	for _, notification := range notifications {
		ids = append(ids, notification.GetId())
	}
	slices.Sort(ids)
	return ids
}
