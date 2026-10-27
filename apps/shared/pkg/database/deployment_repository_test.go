package database

import (
	"context"
	"sort"
	"testing"
	"time"

	deploymentsv1 "github.com/obiente/cloud/apps/shared/proto/obiente/cloud/deployments/v1"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestDeploymentRepositoryTenantIsolation(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	db := newDeploymentRepositoryTestDB(t)
	repo := NewDeploymentRepository(db, nil)
	deletedAt := time.Now()

	seedDeployments(t, db,
		&Deployment{ID: "dep-org-a-user-1", Name: "org-a-user-1", OrganizationID: "org-a", CreatedBy: "user-1"},
		&Deployment{ID: "dep-org-a-user-2", Name: "org-a-user-2", OrganizationID: "org-a", CreatedBy: "user-2"},
		&Deployment{ID: "dep-org-a-deleted", Name: "org-a-deleted", OrganizationID: "org-a", CreatedBy: "user-1", DeletedAt: &deletedAt},
		&Deployment{ID: "dep-org-b-user-1", Name: "org-b-user-1", OrganizationID: "org-b", CreatedBy: "user-1"},
	)

	t.Run("org scoped list never returns another organization", func(t *testing.T) {
		got, err := repo.GetAll(ctx, "org-a", &DeploymentFilters{IncludeAll: true})
		if err != nil {
			t.Fatalf("GetAll returned error: %v", err)
		}

		assertDeploymentIDs(t, got, []string{"dep-org-a-user-1", "dep-org-a-user-2"})
	})

	t.Run("creator filter stays inside organization boundary", func(t *testing.T) {
		got, err := repo.GetAll(ctx, "org-a", &DeploymentFilters{UserID: "user-1"})
		if err != nil {
			t.Fatalf("GetAll returned error: %v", err)
		}

		assertDeploymentIDs(t, got, []string{"dep-org-a-user-1"})
	})

	t.Run("include all includes org peers but not other organizations", func(t *testing.T) {
		got, err := repo.GetAll(ctx, "org-a", &DeploymentFilters{UserID: "user-1", IncludeAll: true})
		if err != nil {
			t.Fatalf("GetAll returned error: %v", err)
		}

		assertDeploymentIDs(t, got, []string{"dep-org-a-user-1", "dep-org-a-user-2"})
	})

	t.Run("count matches list isolation and ignores soft deleted deployments", func(t *testing.T) {
		got, err := repo.Count(ctx, "org-a", &DeploymentFilters{IncludeAll: true})
		if err != nil {
			t.Fatalf("Count returned error: %v", err)
		}
		if got != 2 {
			t.Fatalf("Count(org-a) = %d, want 2", got)
		}
	})
}

func TestNormalizeDeploymentJSONFieldsDefaultsEmptyValues(t *testing.T) {
	t.Parallel()

	deployment := &Deployment{}
	normalizeDeploymentJSONFields(deployment)

	if deployment.CustomDomains != "[]" {
		t.Fatalf("CustomDomains = %q, want []", deployment.CustomDomains)
	}
	if deployment.Groups != "[]" {
		t.Fatalf("Groups = %q, want []", deployment.Groups)
	}
	if deployment.EnvVars != "{}" {
		t.Fatalf("EnvVars = %q, want {}", deployment.EnvVars)
	}
	if deployment.BuildArgs != "{}" {
		t.Fatalf("BuildArgs = %q, want {}", deployment.BuildArgs)
	}
	if deployment.DockerfileVolumes != "[]" {
		t.Fatalf("DockerfileVolumes = %q, want []", deployment.DockerfileVolumes)
	}
	if deployment.DockerfileBuildOptions != "{}" {
		t.Fatalf("DockerfileBuildOptions = %q, want {}", deployment.DockerfileBuildOptions)
	}
}

func TestNormalizeDeploymentJSONFieldsPreservesExistingValues(t *testing.T) {
	t.Parallel()

	deployment := &Deployment{
		CustomDomains:          `["example.com"]`,
		Groups:                 `["prod"]`,
		EnvVars:                `{"APP_ENV":"production"}`,
		BuildArgs:              `{"NODE_VERSION":"22"}`,
		DockerfileVolumes:      `[{"name":"data","mount_path":"/data"}]`,
		DockerfileBuildOptions: `{"target":"runtime"}`,
	}
	normalizeDeploymentJSONFields(deployment)

	if deployment.CustomDomains != `["example.com"]` ||
		deployment.Groups != `["prod"]` ||
		deployment.EnvVars != `{"APP_ENV":"production"}` ||
		deployment.BuildArgs != `{"NODE_VERSION":"22"}` ||
		deployment.DockerfileVolumes != `[{"name":"data","mount_path":"/data"}]` ||
		deployment.DockerfileBuildOptions != `{"target":"runtime"}` {
		t.Fatalf("normalizeDeploymentJSONFields changed existing values: %#v", deployment)
	}
}

func newDeploymentRepositoryTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to open in-memory database: %v", err)
	}

	pool, err := db.DB()
	if err != nil {
		t.Fatal(err)
	}
	pool.SetMaxOpenConns(1)
	t.Cleanup(func() { _ = pool.Close() })
	if err := db.AutoMigrate(&Deployment{}); err != nil {
		t.Fatalf("failed to migrate deployment schema: %v", err)
	}

	return db
}

func seedDeployments(t *testing.T, db *gorm.DB, deployments ...*Deployment) {
	t.Helper()

	for _, deployment := range deployments {
		if err := db.Create(deployment).Error; err != nil {
			t.Fatalf("failed to seed deployment %s: %v", deployment.ID, err)
		}
	}
}

func assertDeploymentIDs(t *testing.T, deployments []*Deployment, want []string) {
	t.Helper()

	got := make([]string, 0, len(deployments))
	for _, deployment := range deployments {
		got = append(got, deployment.ID)
	}
	sort.Strings(got)
	sort.Strings(want)

	if len(got) != len(want) {
		t.Fatalf("deployment IDs = %v, want %v", got, want)
	}
	for i := range got {
		if got[i] != want[i] {
			t.Fatalf("deployment IDs = %v, want %v", got, want)
		}
	}
}

func TestDeploymentIntentSurvivesObservationAndLateBuild(t *testing.T) {
	db := newDeploymentRepositoryTestDB(t)
	repo := NewDeploymentRepository(db, nil)
	dep := &Deployment{ID: "recovery-test", OrganizationID: "test-org"}
	if err := repo.Create(t.Context(), dep); err != nil {
		t.Fatal(err)
	}
	if err := repo.UpdateStatus(t.Context(), dep.ID, int32(deploymentsv1.DeploymentStatus_RUNNING)); err != nil {
		t.Fatal(err)
	}
	var current Deployment
	db.First(&current, "id = ?", dep.ID)
	if current.DesiredState != "running" {
		t.Fatal("successful deployment did not acquire intent")
	}
	if err := repo.SetDesiredState(t.Context(), dep.ID, "stopped"); err != nil {
		t.Fatal(err)
	}
	if err := repo.UpdateStatus(t.Context(), dep.ID, int32(deploymentsv1.DeploymentStatus_RUNNING)); err != nil {
		t.Fatal(err)
	}
	db.First(&current, "id = ?", dep.ID)
	if current.DesiredState != "stopped" {
		t.Fatal("late completion overrode stop")
	}
	if err := repo.Update(t.Context(), dep); err != nil {
		t.Fatal(err)
	}
	db.First(&current, "id = ?", dep.ID)
	if current.DesiredState != "stopped" {
		t.Fatal("stale configuration overwrote stop")
	}
	if err := repo.SetDesiredState(t.Context(), dep.ID, "running"); err != nil {
		t.Fatal(err)
	}
	if err := repo.UpdateStatus(t.Context(), dep.ID, int32(deploymentsv1.DeploymentStatus_FAILED)); err != nil {
		t.Fatal(err)
	}
	db.First(&current, "id = ?", dep.ID)
	if current.DesiredState != "running" {
		t.Fatal("runtime failure erased retry intent")
	}
}
