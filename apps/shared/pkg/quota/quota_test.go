package quota

import (
	"fmt"
	"testing"

	"github.com/obiente/cloud/apps/shared/pkg/database"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestCurrentAllocationsCombinesReservationsAndRunningLocations(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(fmt.Sprintf("file:%s?mode=memory&cache=shared", t.Name())), &gorm.Config{})
	if err != nil {
		t.Fatalf("open test database: %v", err)
	}
	if err := db.AutoMigrate(&database.Deployment{}, &database.DeploymentLocation{}); err != nil {
		t.Fatalf("migrate test database: %v", err)
	}
	previousDB := database.DB
	database.DB = db
	t.Cleanup(func() { database.DB = previousDB })

	replicasTwo, replicasOne := int32(2), int32(1)
	memory := int64(100)
	cpu := int64(512)
	deployments := []database.Deployment{
		{ID: "ordinary", OrganizationID: "org", Status: 3, Replicas: &replicasTwo, MemoryBytes: &memory, CPUShares: &cpu},
		{ID: "compose", OrganizationID: "org", Status: 3, Replicas: &replicasOne, MemoryBytes: &memory, CPUShares: &cpu},
		{ID: "failed", OrganizationID: "org", Status: 4, Replicas: &replicasOne, MemoryBytes: &memory, CPUShares: &cpu},
	}
	if err := db.Create(&deployments).Error; err != nil {
		t.Fatalf("seed deployments: %v", err)
	}
	locations := make([]database.DeploymentLocation, 0, 7)
	for deploymentID, count := range map[string]int{"ordinary": 2, "compose": 3, "failed": 2} {
		for index := 0; index < count; index++ {
			locations = append(locations, database.DeploymentLocation{
				ID: fmt.Sprintf("%s-%d", deploymentID, index), DeploymentID: deploymentID,
				NodeID: "node", ContainerID: fmt.Sprintf("container-%s-%d", deploymentID, index), Status: "running",
			})
		}
	}
	if err := db.Create(&locations).Error; err != nil {
		t.Fatalf("seed locations: %v", err)
	}

	gotReplicas, gotMemory, gotCPU, err := NewChecker().currentAllocations("org", "")
	if err != nil {
		t.Fatalf("calculate allocations: %v", err)
	}
	if gotReplicas != 7 || gotMemory != 300 || gotCPU != 2 {
		t.Fatalf("allocations = (%d, %d, %d), want (7, 300, 2)", gotReplicas, gotMemory, gotCPU)
	}
}

func TestCurrentAllocationsReservesInFlightReplicaWithoutLocation(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(fmt.Sprintf("file:%s?mode=memory&cache=shared", t.Name())), &gorm.Config{})
	if err != nil {
		t.Fatalf("open test database: %v", err)
	}
	if err := db.AutoMigrate(&database.Deployment{}, &database.DeploymentLocation{}); err != nil {
		t.Fatalf("migrate test database: %v", err)
	}
	previousDB := database.DB
	database.DB = db
	t.Cleanup(func() { database.DB = previousDB })
	replicas := int32(3)
	memory, cpu := int64(256), int64(256)
	if err := db.Create(&database.Deployment{ID: "building", OrganizationID: "org", Status: 2, Replicas: &replicas, MemoryBytes: &memory, CPUShares: &cpu}).Error; err != nil {
		t.Fatalf("seed building deployment: %v", err)
	}

	gotReplicas, gotMemory, gotCPU, err := NewChecker().currentAllocations("org", "")
	if err != nil {
		t.Fatalf("calculate allocations: %v", err)
	}
	if gotReplicas != 3 || gotMemory != 768 || gotCPU != 1 {
		t.Fatalf("allocations = (%d, %d, %d), want (3, 768, 1)", gotReplicas, gotMemory, gotCPU)
	}
}

func TestQuotaReadFailuresDoNotGrantUnlimitedResources(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	raw, _ := db.DB()
	prior := database.DB
	database.DB = db
	t.Cleanup(func() { database.DB = prior; raw.Close() })
	if err := db.AutoMigrate(&database.OrgQuota{}, &database.OrganizationPlan{}); err != nil {
		t.Fatal(err)
	}
	if _, _, err := GetEffectiveLimits("missing"); err == nil {
		t.Fatal("missing quota granted unlimited resources")
	}
	if err := db.Create(&database.OrgQuota{OrganizationID: "example", PlanID: "missing-plan"}).Error; err != nil {
		t.Fatal(err)
	}
	if _, _, err := GetEffectiveLimits("example"); err == nil {
		t.Fatal("missing plan granted unlimited resources")
	}
	if err := db.Create(&database.OrganizationPlan{ID: "missing-plan", Name: "Example", MemoryBytes: 4096, CPUCores: 2}).Error; err != nil {
		t.Fatal(err)
	}
	if mem, cpu, err := GetEffectiveLimits("example"); err != nil || mem != 4096 || cpu != 2 {
		t.Fatalf("valid plan: %d %d %v", mem, cpu, err)
	}
	if err := db.Migrator().DropTable(&database.OrgQuota{}); err != nil {
		t.Fatal(err)
	}
	if _, err := NewChecker().getPlanVPSMax("example"); err == nil {
		t.Fatal("failed VPS quota query ignored")
	}
	if _, err := NewChecker().getQuota("example"); err == nil {
		t.Fatal("failed quota query ignored")
	}
}
