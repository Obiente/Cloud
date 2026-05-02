package database

import (
	"strings"
	"testing"
	"time"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestInitVPSCatalogPreservesExistingSizeSettings(t *testing.T) {
	db := newVPSCatalogTestDB(t)
	now := time.Now().UTC().Add(-time.Hour)

	customMedium := VPSSizeCatalog{
		ID:                  "medium",
		Name:                "Custom Medium",
		Description:         "admin configured",
		CPUCores:            3,
		MemoryBytes:         3 * 1024 * 1024 * 1024,
		DiskBytes:           33 * 1024 * 1024 * 1024,
		BandwidthBytesMonth: 12345,
		MinimumPaymentCents: 10,
		Available:           false,
		Region:              "eu-custom",
		CreatedAt:           now,
		UpdatedAt:           now,
	}
	if err := db.Create(&customMedium).Error; err != nil {
		t.Fatalf("seed custom medium size: %v", err)
	}
	if err := db.Model(&VPSSizeCatalog{}).Where("id = ?", customMedium.ID).Update("available", customMedium.Available).Error; err != nil {
		t.Fatalf("set custom medium availability: %v", err)
	}

	if err := InitVPSCatalog(); err != nil {
		t.Fatalf("init VPS catalog: %v", err)
	}

	var medium VPSSizeCatalog
	if err := db.First(&medium, "id = ?", "medium").Error; err != nil {
		t.Fatalf("load medium size: %v", err)
	}

	if medium.MinimumPaymentCents != customMedium.MinimumPaymentCents {
		t.Fatalf("medium minimum payment = %d, want preserved %d", medium.MinimumPaymentCents, customMedium.MinimumPaymentCents)
	}
	if medium.Name != customMedium.Name ||
		medium.CPUCores != customMedium.CPUCores ||
		medium.MemoryBytes != customMedium.MemoryBytes ||
		medium.DiskBytes != customMedium.DiskBytes ||
		medium.BandwidthBytesMonth != customMedium.BandwidthBytesMonth ||
		medium.Available != customMedium.Available ||
		medium.Region != customMedium.Region {
		t.Fatalf("medium size was overwritten: got %+v, want preserved %+v", medium, customMedium)
	}

	var seededSizes []VPSSizeCatalog
	if err := db.Order("id ASC").Find(&seededSizes).Error; err != nil {
		t.Fatalf("list seeded sizes: %v", err)
	}
	if len(seededSizes) != 4 {
		t.Fatalf("seeded size count = %d, want 4", len(seededSizes))
	}

	expectedMinimums := map[string]int64{
		"large":  50,
		"medium": 10,
		"small":  0,
		"xlarge": 100,
	}
	for _, size := range seededSizes {
		if got, want := size.MinimumPaymentCents, expectedMinimums[size.ID]; got != want {
			t.Fatalf("%s minimum payment = %d, want %d", size.ID, got, want)
		}
	}
}

func TestInitVPSCatalogPreservesExistingRegionSettings(t *testing.T) {
	db := newVPSCatalogTestDB(t)

	customRegion := VPSRegionCatalog{
		ID:        "eu-west-1",
		Name:      "Private Dublin",
		Location:  "Custom location",
		Country:   "NL",
		Features:  `["admin_configured"]`,
		Available: false,
	}
	if err := db.Create(&customRegion).Error; err != nil {
		t.Fatalf("seed custom region: %v", err)
	}
	if err := db.Model(&VPSRegionCatalog{}).Where("id = ?", customRegion.ID).Update("available", customRegion.Available).Error; err != nil {
		t.Fatalf("set custom region availability: %v", err)
	}

	if err := InitVPSCatalog(); err != nil {
		t.Fatalf("init VPS catalog: %v", err)
	}

	var region VPSRegionCatalog
	if err := db.First(&region, "id = ?", "eu-west-1").Error; err != nil {
		t.Fatalf("load region: %v", err)
	}
	if region.Name != customRegion.Name ||
		region.Location != customRegion.Location ||
		region.Country != customRegion.Country ||
		region.Features != customRegion.Features ||
		region.Available != customRegion.Available {
		t.Fatalf("region was overwritten: got %+v, want preserved %+v", region, customRegion)
	}
}

func newVPSCatalogTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	dbName := "file:vps_catalog_" + strings.NewReplacer("/", "_", " ", "_").Replace(t.Name()) + "?mode=memory&cache=shared"
	db, err := gorm.Open(sqlite.Open(dbName), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite db: %v", err)
	}
	if err := db.AutoMigrate(&VPSSizeCatalog{}, &VPSRegionCatalog{}); err != nil {
		t.Fatalf("migrate sqlite db: %v", err)
	}

	previousDB := DB
	DB = db
	t.Cleanup(func() {
		DB = previousDB
	})

	return db
}
