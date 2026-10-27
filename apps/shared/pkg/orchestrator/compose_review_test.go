package orchestrator

import (
	"context"
	"github.com/moby/moby/api/types/container"
	"github.com/moby/moby/client"
	"github.com/obiente/cloud/apps/shared/pkg/database"
	deploymentsv1 "github.com/obiente/cloud/apps/shared/proto/obiente/cloud/deployments/v1"
	"gopkg.in/yaml.v3"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestComposeRecoveryIngressUsesPersistedPreviewEnvironment(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	raw, _ := db.DB()
	defer raw.Close()
	if err := db.AutoMigrate(&database.Deployment{}); err != nil {
		t.Fatal(err)
	}
	dep := database.Deployment{ID: "preview-test", Environment: int32(deploymentsv1.Environment_PULL_REQUEST)}
	if err := db.Create(&dep).Error; err != nil {
		t.Fatal(err)
	}
	for _, requested := range []string{"", "shared-platform"} {
		ingress, err := resolveComposeIngress(t.Context(), db, dep.ID, requested)
		if err != nil {
			t.Fatal(err)
		}
		output, err := (&DeploymentManager{}).addTraefikNetworkToRoutedServices("services:\n  web:\n    image: nginx:alpine\n", []database.DeploymentRouting{{ServiceName: "web"}}, ingress)
		if err != nil {
			t.Fatal(err)
		}
		var spec struct {
			Networks map[string]struct{ Name string }
		}
		if err := yaml.Unmarshal([]byte(output), &spec); err != nil {
			t.Fatal(err)
		}
		if len(spec.Networks) == 0 {
			t.Fatal("no ingress network")
		}
		for _, network := range spec.Networks {
			if network.Name != PreviewIngressNetworkNameForDeployment(dep.ID) {
				t.Fatalf("preview joined %q", network.Name)
			}
		}
	}
	if _, err := resolveComposeIngress(t.Context(), db, "missing", ""); err == nil {
		t.Fatal("missing trust classification allowed")
	}
}

type composeReviewClient struct {
	client.APIClient
	exitCode int
}

func (c *composeReviewClient) ContainerList(context.Context, client.ContainerListOptions) (client.ContainerListResult, error) {
	return client.ContainerListResult{Items: []container.Summary{
		{ID: "web", State: container.StateRunning, Labels: map[string]string{"com.docker.compose.service": "web"}},
		{ID: "worker", State: container.StateExited, Labels: map[string]string{"com.docker.compose.service": "worker"}},
	}}, nil
}
func (c *composeReviewClient) ContainerInspect(context.Context, string, client.ContainerInspectOptions) (client.ContainerInspectResult, error) {
	return client.ContainerInspectResult{Container: container.InspectResponse{State: &container.State{ExitCode: c.exitCode}}}, nil
}
func TestComposeRecoveryDistinguishesJobsFromCleanServiceExits(t *testing.T) {
	t.Setenv("ENABLE_SWARM", "false")
	previous := database.DB
	database.DB = nil
	t.Cleanup(func() { database.DB = previous })
	for _, tc := range []struct {
		name, dependency, restart string
		exitCode                  int
		complete                  bool
	}{
		{name: "omitted restart policy"},
		{name: "explicit no restart", restart: "    restart: no\n"},
		{name: "short dependency", dependency: "    depends_on: [worker]\n"},
		{name: "healthy dependency", dependency: "    depends_on:\n      worker:\n        condition: service_healthy\n"},
		{name: "completed job", dependency: "    depends_on:\n      worker:\n        condition: service_completed_successfully\n", complete: true},
		{name: "failed job", dependency: "    depends_on:\n      worker:\n        condition: service_completed_successfully\n", exitCode: 1},
	} {
		t.Run(tc.name, func(t *testing.T) {
			manager := &DeploymentManager{dockerClient: &composeReviewClient{exitCode: tc.exitCode}}
			err := manager.RecoverComposeDeployment(t.Context(), "test", "services:\n  web:\n    image: busybox\n"+tc.dependency+"  worker:\n    image: busybox\n"+tc.restart)
			if tc.complete {
				if err != nil {
					t.Fatal(err)
				}
			} else if err == nil || !strings.Contains(err.Error(), "cannot verify organization") {
				t.Fatalf("missing service did not enter recovery: %v", err)
			}
		})
	}
}

func TestPreviewComposeRecoveryFailsClosedWithoutIsolatedIngress(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Docker command stub requires POSIX shell")
	}
	t.Setenv("ENABLE_SWARM", "false")
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	raw, _ := db.DB()
	defer raw.Close()
	previous := database.DB
	database.DB = db
	t.Cleanup(func() { database.DB = previous })
	if err := db.AutoMigrate(&database.Deployment{}, &database.Organization{}); err != nil {
		t.Fatal(err)
	}
	for _, row := range []interface{}{&database.Organization{ID: "org-preview", Slug: "org-preview", Status: "active"}, &database.Deployment{ID: "preview-test", OrganizationID: "org-preview", Environment: int32(deploymentsv1.Environment_PULL_REQUEST), DesiredState: "running"}} {
		if err := db.Create(row).Error; err != nil {
			t.Fatal(err)
		}
	}
	dir := t.TempDir()
	script := "#!/bin/sh\nif [ \"$1\" = network ] && [ \"$2\" = ls ]; then echo deployment-preview-test; exit 0; fi\nif [ \"$1\" = ps ]; then exit 0; fi\nexit 55\n"
	if err := os.WriteFile(filepath.Join(dir, "docker"), []byte(script), 0700); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", dir)
	dm := &DeploymentManager{dockerClient: &composeReviewClient{}}
	err = dm.RecoverComposeDeployment(t.Context(), "preview-test", "services:\n  missing:\n    image: busybox\n")
	if err == nil || !strings.Contains(err.Error(), "no ingress proxy container") {
		t.Fatalf("preview recovery bypassed isolated ingress requirement: %v", err)
	}
}
