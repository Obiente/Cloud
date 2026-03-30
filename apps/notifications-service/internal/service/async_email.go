package notifications

import (
	"context"
	"sync"
	"time"

	"github.com/obiente/cloud/apps/shared/pkg/logger"

	notificationsv1 "github.com/obiente/cloud/apps/shared/proto/obiente/cloud/notifications/v1"
)

const (
	asyncEmailQueueSize   = 256
	asyncEmailWorkerCount = 4
	asyncEmailTimeout     = 30 * time.Second
)

type asyncEmailTask struct {
	userID      string
	notifType   notificationsv1.NotificationType
	severity    notificationsv1.NotificationSeverity
	title       string
	message     string
	actionURL   *string
	actionLabel *string
}

var (
	asyncEmailInitOnce sync.Once
	asyncEmailQueue    chan asyncEmailTask
	asyncEmailCtx      context.Context
	asyncEmailCancel   context.CancelFunc
	asyncEmailWorkers  sync.WaitGroup
)

func InitAsyncEmailDispatcher(parent context.Context) {
	asyncEmailInitOnce.Do(func() {
		if parent == nil {
			parent = context.Background()
		}

		asyncEmailCtx, asyncEmailCancel = context.WithCancel(parent)
		asyncEmailQueue = make(chan asyncEmailTask, asyncEmailQueueSize)

		for i := 0; i < asyncEmailWorkerCount; i++ {
			asyncEmailWorkers.Add(1)
			go runAsyncEmailWorker()
		}
	})
}

func ShutdownAsyncEmailDispatcher(ctx context.Context) {
	if asyncEmailCancel == nil {
		return
	}

	asyncEmailCancel()

	done := make(chan struct{})
	go func() {
		defer close(done)
		asyncEmailWorkers.Wait()
	}()

	select {
	case <-done:
	case <-ctx.Done():
		logger.Warn("[Notifications] Timed out waiting for async email workers to stop")
	}
}

func enqueueNotificationEmail(userID string, notifType notificationsv1.NotificationType, severity notificationsv1.NotificationSeverity, title, message string, actionURL, actionLabel *string) {
	InitAsyncEmailDispatcher(context.Background())

	if asyncEmailCtx != nil {
		select {
		case <-asyncEmailCtx.Done():
			logger.Warn("[Notifications] Skipping async email for user %s because the dispatcher is shutting down", userID)
			return
		default:
		}
	}

	task := asyncEmailTask{
		userID:      userID,
		notifType:   notifType,
		severity:    severity,
		title:       title,
		message:     message,
		actionURL:   actionURL,
		actionLabel: actionLabel,
	}

	select {
	case asyncEmailQueue <- task:
	default:
		logger.Warn("[Notifications] Dropping async email for user %s because the queue is full", userID)
	}
}

func runAsyncEmailWorker() {
	defer asyncEmailWorkers.Done()

	for {
		select {
		case <-asyncEmailCtx.Done():
			return
		case task := <-asyncEmailQueue:
			taskCtx, cancel := context.WithTimeout(asyncEmailCtx, asyncEmailTimeout)
			logger.Info("[Notifications] Starting email check for user %s, type %s", task.userID, notificationTypeToString(task.notifType))
			if err := sendNotificationEmailIfEnabled(taskCtx, task.userID, task.notifType, task.severity, task.title, task.message, task.actionURL, task.actionLabel); err != nil {
				logger.Warn("[Notifications] Failed to send email notification for user %s: %v", task.userID, err)
			}
			cancel()
		}
	}
}
