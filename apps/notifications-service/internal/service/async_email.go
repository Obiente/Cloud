package notifications

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/obiente/cloud/apps/shared/pkg/database"
	"github.com/obiente/cloud/apps/shared/pkg/logger"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const (
	asyncEmailWorkerCount = 4
	asyncEmailTimeout     = 30 * time.Second
	emailDeliveryLease    = 2 * time.Minute
)

// EmailDelivery is a durable outbox entry owned by the notifications service.
// Completed includes notifications intentionally skipped by user preferences.
type EmailDelivery struct {
	NotificationID string     `gorm:"primaryKey"`
	Payload        string     `gorm:"type:jsonb;not null"`
	Status         string     `gorm:"not null;index:idx_email_delivery_due"`
	AvailableAt    time.Time  `gorm:"not null;index:idx_email_delivery_due"`
	Attempts       int        `gorm:"not null;default:0"`
	LeaseUntil     *time.Time `gorm:"index"`
	LeaseToken     string
	LastError      string
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

func (EmailDelivery) TableName() string { return "notification_email_deliveries" }

type emailDispatcher struct {
	db      *gorm.DB
	ctx     context.Context
	cancel  context.CancelFunc
	workers sync.WaitGroup
	wake    chan struct{}
	now     func() time.Time
	send    func(context.Context, *database.Notification) error
}

var dispatcherMu sync.Mutex
var activeEmailDispatcher *emailDispatcher

func InitAsyncEmailDispatcher(parent context.Context) {
	dispatcherMu.Lock()
	defer dispatcherMu.Unlock()
	if activeEmailDispatcher != nil && activeEmailDispatcher.ctx.Err() == nil {
		return
	}
	if parent == nil {
		parent = context.Background()
	}
	ctx, cancel := context.WithCancel(parent)
	d := &emailDispatcher{db: database.DB, ctx: ctx, cancel: cancel, wake: make(chan struct{}, 1), now: time.Now, send: func(ctx context.Context, n *database.Notification) error {
		return sendNotificationEmailIfEnabled(ctx, n.UserID, stringToNotificationType(n.Type), stringToNotificationSeverity(n.Severity), n.Title, n.Message, n.ActionURL, n.ActionLabel)
	}}
	activeEmailDispatcher = d
	for range asyncEmailWorkerCount {
		d.workers.Add(1)
		go d.run()
	}
}

func ShutdownAsyncEmailDispatcher(ctx context.Context) {
	dispatcherMu.Lock()
	d := activeEmailDispatcher
	dispatcherMu.Unlock()
	if d == nil {
		return
	}
	d.cancel()
	done := make(chan struct{})
	go func() { defer close(done); d.workers.Wait() }()
	select {
	case <-done:
	case <-ctx.Done():
		logger.Warn("[Notifications] Timed out waiting for email workers; unfinished deliveries remain queued")
	}
}

// The notification and delivery become visible together. Wakeups are only an
// optimization; polling recovers work after missed wakeups or process restarts.
func persistNotification(ctx context.Context, n *database.Notification) error {
	if n.Metadata == "" {
		n.Metadata = "{}"
	}
	payload, err := json.Marshal(n)
	if err != nil {
		return err
	}
	if err := database.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(n).Error; err != nil {
			return err
		}
		return tx.Create(&EmailDelivery{NotificationID: n.ID, Payload: string(payload), Status: "pending", AvailableAt: time.Now()}).Error
	}); err != nil {
		return err
	}
	dispatcherMu.Lock()
	d := activeEmailDispatcher
	dispatcherMu.Unlock()
	if d != nil {
		select {
		case d.wake <- struct{}{}:
		default:
		}
	}
	return nil
}

func (d *emailDispatcher) run() {
	defer d.workers.Done()
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()
	for {
		if d.ctx.Err() != nil {
			return
		}
		worked, err := d.deliverNext(d.ctx)
		if err != nil {
			logger.Warn("[Notifications] Email outbox unavailable: %v", err)
		}
		if worked && err == nil {
			continue
		}
		select {
		case <-d.ctx.Done():
			return
		case <-d.wake:
		case <-ticker.C:
		}
	}
}

func (d *emailDispatcher) deliverNext(ctx context.Context) (bool, error) {
	if err := ctx.Err(); err != nil {
		return false, err
	}
	var delivery EmailDelivery
	now := d.now()
	err := d.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		result := tx.Clauses(clause.Locking{Strength: "UPDATE", Options: "SKIP LOCKED"}).
			Where("(status = ? AND available_at <= ?) OR (status = ? AND lease_until <= ?)", "pending", now, "processing", now).
			Order("available_at, notification_id").Limit(1).Find(&delivery)
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected == 0 {
			return gorm.ErrRecordNotFound
		}
		until := now.Add(emailDeliveryLease)
		delivery.Status = "processing"
		delivery.LeaseToken = uuid.NewString()
		delivery.LeaseUntil = &until
		delivery.Attempts++
		return tx.Save(&delivery).Error
	})
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return false, nil
	}
	if err != nil {
		return false, err
	}

	var notification database.Notification
	decodeErr := json.Unmarshal([]byte(delivery.Payload), &notification)
	var sendErr error
	if decodeErr == nil && notification.UserID != "" {
		sendCtx, cancel := context.WithTimeout(ctx, asyncEmailTimeout)
		sendErr = d.send(sendCtx, &notification)
		cancel()
	} else {
		sendErr = fmt.Errorf("invalid persisted email notification")
	}

	updates := map[string]any{"status": "completed", "lease_until": nil, "lease_token": "", "last_error": ""}
	if sendErr != nil {
		message := sendErr.Error()
		if len(message) > 1024 {
			message = message[:1024]
		}
		next := d.now().Add(emailRetryDelay(delivery.Attempts))
		if ctx.Err() != nil {
			next = d.now()
		}
		updates["status"] = "pending"
		updates["available_at"] = next
		updates["last_error"] = message
		if decodeErr != nil || notification.UserID == "" {
			updates["status"] = "failed"
		}
		logger.Warn("[Notifications] Email delivery %s attempt %d failed; persisted for recovery", delivery.NotificationID, delivery.Attempts)
	}
	// A canceled worker can still release its lease. If this write fails, another
	// process will recover the entry after lease expiry. SMTP delivery is at least
	// once: losing acknowledgement after the provider accepts may cause a retry.
	cleanup, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	result := d.db.WithContext(cleanup).Model(&EmailDelivery{}).Where("notification_id = ? AND status = ? AND lease_token = ?", delivery.NotificationID, "processing", delivery.LeaseToken).Updates(updates)
	if result.Error != nil {
		return true, result.Error
	}
	if result.RowsAffected != 1 {
		return true, fmt.Errorf("email delivery lease changed before acknowledgement")
	}
	return true, nil
}

func emailRetryDelay(attempt int) time.Duration {
	// Retry through long outages without discarding accepted messages.
	return min(30*time.Second*time.Duration(1<<min(max(attempt-1, 0), 5)), 15*time.Minute)
}
