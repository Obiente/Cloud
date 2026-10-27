package deployments

import (
	"context"
	"io"
	"net/http"
	"strings"
	"sync"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/obiente/cloud/apps/shared/pkg/auth"
	"github.com/obiente/cloud/apps/shared/pkg/database"
	"github.com/obiente/cloud/apps/shared/pkg/orchestrator"
	deploymentsv1 "github.com/obiente/cloud/apps/shared/proto/obiente/cloud/deployments/v1"
)

type pausedRedeployStrategy struct {
	started chan string
	resume  chan struct{}
	guard   chan error
}

type redeployNotificationTransport struct{ t *testing.T }

func (transport redeployNotificationTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	if !strings.Contains(req.URL.Path, "/obiente.cloud.notifications.v1.NotificationService/") {
		transport.t.Errorf("unexpected HTTP request: %s", req.URL.Path)
	}
	return &http.Response{StatusCode: http.StatusOK, Header: http.Header{"Content-Type": []string{"application/json"}}, Body: io.NopCloser(strings.NewReader("{}")), Request: req}, nil
}

func (*pausedRedeployStrategy) Name() string                                 { return "test-redeploy" }
func (*pausedRedeployStrategy) Detect(context.Context, string) (bool, error) { return false, nil }
func (s *pausedRedeployStrategy) Build(ctx context.Context, deployment *database.Deployment, config *BuildConfig) (*BuildResult, error) {
	var current database.Deployment
	if err := database.DB.WithContext(ctx).First(&current, "id = ?", deployment.ID).Error; err != nil {
		return nil, err
	}
	s.started <- current.DesiredState
	select {
	case <-s.resume:
	case <-ctx.Done():
		return nil, ctx.Err()
	}
	// Use the actual materialization guard. With a later stop it must return
	// before touching Docker, even though the queued build has now completed.
	err := (&orchestrator.DeploymentManager{}).CreateDeploymentLocked(ctx, &orchestrator.DeploymentConfig{DeploymentID: config.DeploymentID})
	s.guard <- err
	return nil, err
}

func TestTriggerStoppedDeploymentRecordsIntentBeforeBuildAndPreservesLaterStop(t *testing.T) {
	// The real handler emits a build notification. Keep all of its HTTP calls
	// inside this test, including when a developer has service URLs configured.
	previousTransport := http.DefaultTransport
	http.DefaultTransport = redeployNotificationTransport{t}
	t.Cleanup(func() { http.DefaultTransport = previousTransport })
	db := newDeploymentServiceTestDB(t)
	if err := db.AutoMigrate(&database.BuildLog{}, &database.DeploymentRouting{}); err != nil {
		t.Fatal(err)
	}
	dep := database.Deployment{ID: "stopped-redeploy", OrganizationID: "org", DesiredState: "stopped", Status: int32(deploymentsv1.DeploymentStatus_STOPPED), BuildStrategy: int32(deploymentsv1.BuildStrategy_DOCKERFILE)}
	if err := db.Create(&dep).Error; err != nil {
		t.Fatal(err)
	}
	ctx := auth.WithSystemUser(context.Background())
	background, cancel := context.WithCancel(context.Background())
	service := NewService(background, database.NewDeploymentRepository(db, nil), nil, nil)
	strategy := &pausedRedeployStrategy{started: make(chan string, 1), resume: make(chan struct{}), guard: make(chan error, 1)}
	service.buildRegistry.Register(deploymentsv1.BuildStrategy_DOCKERFILE, strategy)
	var resumeOnce sync.Once
	t.Cleanup(func() {
		resumeOnce.Do(func() { close(strategy.resume) })
		defer cancel()
		deadline := time.Now().Add(3 * time.Second)
		for {
			var count int64
			if err := db.Model(&database.DeploymentBuildControl{}).Where("deployment_id = ?", dep.ID).Count(&count).Error; err != nil {
				t.Error(err)
				break
			}
			if count == 0 {
				break
			}
			if time.Now().After(deadline) {
				t.Error("build did not release its lease")
				break
			}
			time.Sleep(10 * time.Millisecond)
		}
		globalBuildLogRegistry.mu.Lock()
		streamer := globalBuildLogRegistry.streamers[dep.ID]
		delete(globalBuildLogRegistry.streamers, dep.ID)
		globalBuildLogRegistry.mu.Unlock()
		if streamer != nil {
			streamer.Close()
		}
	})
	request := connect.NewRequest(&deploymentsv1.TriggerDeploymentRequest{DeploymentId: dep.ID})
	if _, err := service.TriggerDeployment(ctx, request); err != nil {
		t.Fatal(err)
	}
	select {
	case desired := <-strategy.started:
		if desired != "running" {
			t.Fatalf("build started with desired state %q, want running", desired)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("build did not start")
	}
	// A runtime outage still records stop intent durably for recovery.
	_, err := service.StopDeployment(ctx, connect.NewRequest(&deploymentsv1.StopDeploymentRequest{DeploymentId: dep.ID, OrganizationId: dep.OrganizationID}))
	if connect.CodeOf(err) != connect.CodeUnavailable {
		t.Fatalf("stop without runtime = %v", err)
	}
	// A rejected duplicate must not reinstate running intent for this build.
	if _, err := service.TriggerDeployment(ctx, request); connect.CodeOf(err) != connect.CodeAborted {
		t.Fatalf("duplicate trigger = %v", err)
	}
	resumeOnce.Do(func() { close(strategy.resume) })
	select {
	case err := <-strategy.guard:
		if err == nil || !strings.Contains(err.Error(), "superseded by stopped intent") {
			t.Fatalf("late build ignored stop: %v", err)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("build did not reach the materialization guard")
	}
}
