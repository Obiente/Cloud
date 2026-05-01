package database

import (
	"context"
	"sort"
	"testing"
	"time"

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

func newDeploymentRepositoryTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to open in-memory database: %v", err)
	}

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
