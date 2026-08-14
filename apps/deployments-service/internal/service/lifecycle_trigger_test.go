package deployments

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/obiente/cloud/apps/shared/pkg/auth"
	"github.com/obiente/cloud/apps/shared/pkg/database"

	deploymentsv1 "github.com/obiente/cloud/apps/shared/proto/obiente/cloud/deployments/v1"

	"connectrpc.com/connect"
)

func TestComposeDeploymentNeedsContainerVerification(t *testing.T) {
	t.Setenv("ENABLE_SWARM", "false")
	if !composeDeploymentNeedsContainerVerification() {
		t.Fatal("non-Swarm Compose deployment must verify a running container")
	}

	t.Setenv("ENABLE_SWARM", "true")
	if composeDeploymentNeedsContainerVerification() {
		t.Fatal("Swarm Compose deployment already performs job-aware convergence")
	}
}

func TestAbortedDeletionClearsIdleCancellationMarker(t *testing.T) {
	db := newDeploymentServiceTestDB(t)
	service := NewService(context.Background(), database.NewDeploymentRepository(db, nil), nil, nil)
	now := time.Now()
	control := database.DeploymentBuildControl{DeploymentID: "deployment-aborted-delete", CancelRequestedAt: &now, CreatedAt: now, UpdatedAt: now}
	if err := db.Create(&control).Error; err != nil {
		t.Fatalf("seed build cancellation marker: %v", err)
	}
	service.clearDeploymentBuildCancellationAfterAbort(control.DeploymentID)
	var count int64
	if err := db.Model(&database.DeploymentBuildControl{}).Where("deployment_id = ?", control.DeploymentID).Count(&count).Error; err != nil {
		t.Fatalf("count build cancellation markers: %v", err)
	}
	if count != 0 {
		t.Fatal("aborted deletion left an idle cancellation marker")
	}
}

func TestInterruptedBuildHistoryIsFinalizedWithDetachedContext(t *testing.T) {
	db := newDeploymentServiceTestDB(t)
	service := NewService(context.Background(), database.NewDeploymentRepository(db, nil), nil, nil)
	startedAt := time.Now().Add(-time.Minute)
	build := database.BuildHistory{
		ID: "build-superseded", DeploymentID: "deployment-superseded", OrganizationID: "org",
		BuildNumber: 1, Status: 2, StartedAt: startedAt,
	}
	if err := db.Create(&build).Error; err != nil {
		t.Fatalf("seed interrupted build: %v", err)
	}
	service.finalizeInterruptedBuildHistory(build.ID, startedAt)
	var got database.BuildHistory
	if err := db.First(&got, "id = ?", build.ID).Error; err != nil {
		t.Fatalf("reload interrupted build: %v", err)
	}
	if got.Status != 4 || got.CompletedAt == nil || got.Error == nil || !strings.Contains(*got.Error, "superseded") {
		t.Fatalf("interrupted build was not finalized: %#v", got)
	}
}

func TestUnregisterDeploymentBuildConfirmsDurableRelease(t *testing.T) {
	db := newDeploymentServiceTestDB(t)
	service := NewService(context.Background(), database.NewDeploymentRepository(db, nil), nil, nil)
	control := database.DeploymentBuildControl{
		DeploymentID: "deployment-complete",
		BuildToken:   "completed-token",
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}
	if err := db.Create(&control).Error; err != nil {
		t.Fatalf("seed build lease: %v", err)
	}
	service.activeBuilds[control.DeploymentID] = activeDeploymentBuild{token: control.BuildToken}

	if !service.unregisterDeploymentBuild(control.DeploymentID, control.BuildToken) {
		t.Fatal("completed build lease was not confirmed released")
	}
	var count int64
	if err := db.Model(&database.DeploymentBuildControl{}).Where("deployment_id = ?", control.DeploymentID).Count(&count).Error; err != nil {
		t.Fatalf("count build leases: %v", err)
	}
	if count != 0 {
		t.Fatal("completed build lease remained durable after confirmed release")
	}
	if _, active := service.activeBuilds[control.DeploymentID]; active {
		t.Fatal("completed build lease remained active in memory")
	}
}

func TestRequestedDeploymentCommitSHARequiresSystemPrincipal(t *testing.T) {
	commitSHA := strings.Repeat("a", 40)

	_, err := requestedDeploymentCommitSHA(context.Background(), commitSHA)
	if connect.CodeOf(err) != connect.CodePermissionDenied {
		t.Fatalf("error code = %s, want %s", connect.CodeOf(err), connect.CodePermissionDenied)
	}

	got, err := requestedDeploymentCommitSHA(auth.WithSystemUser(context.Background()), commitSHA)
	if err != nil {
		t.Fatalf("trusted commit override: %v", err)
	}
	if got != commitSHA {
		t.Fatalf("commit override = %q, want %q", got, commitSHA)
	}
}

func TestRequestedDeploymentCommitSHARejectsInvalidSHA(t *testing.T) {
	_, err := requestedDeploymentCommitSHA(auth.WithSystemUser(context.Background()), "not-a-commit")
	if connect.CodeOf(err) != connect.CodeInvalidArgument {
		t.Fatalf("error code = %s, want %s", connect.CodeOf(err), connect.CodeInvalidArgument)
	}
}

func TestShouldDeployStoredComposeOnlyForPlainCompose(t *testing.T) {
	tests := []struct {
		name       string
		strategy   deploymentsv1.BuildStrategy
		composeYML string
		want       bool
	}{
		{name: "plain compose", strategy: deploymentsv1.BuildStrategy_PLAIN_COMPOSE, composeYML: "services: {}", want: true},
		{name: "repository compose", strategy: deploymentsv1.BuildStrategy_COMPOSE_REPO, composeYML: "services: {}", want: false},
		{name: "empty plain compose", strategy: deploymentsv1.BuildStrategy_PLAIN_COMPOSE, composeYML: "", want: false},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			deployment := &database.Deployment{
				BuildStrategy: int32(test.strategy),
				ComposeYaml:   test.composeYML,
			}
			if got := shouldDeployStoredCompose(deployment); got != test.want {
				t.Fatalf("shouldDeployStoredCompose() = %t, want %t", got, test.want)
			}
		})
	}
}
