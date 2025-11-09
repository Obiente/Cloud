package database

import (
	"fmt"
	"time"

	"gorm.io/gorm"
)

// VPSSizeCatalog represents a VPS size definition in the catalog
type VPSSizeCatalog struct {
	ID                  string    `gorm:"primaryKey;column:id" json:"id"`
	Name                string    `gorm:"column:name" json:"name"`
	Description         string    `gorm:"column:description;type:text" json:"description"`
	CPUCores            int32     `gorm:"column:cpu_cores" json:"cpu_cores"`
	MemoryBytes         int64     `gorm:"column:memory_bytes" json:"memory_bytes"`
	DiskBytes           int64     `gorm:"column:disk_bytes" json:"disk_bytes"`
	BandwidthBytesMonth int64     `gorm:"column:bandwidth_bytes_month;default:0" json:"bandwidth_bytes_month"` // 0 = unlimited
	PriceCentsPerMonth  int64     `gorm:"column:price_cents_per_month" json:"price_cents_per_month"`
	Available           bool      `gorm:"column:available;default:true" json:"available"`
	Region              string    `gorm:"column:region;index" json:"region"` // Empty = all regions
	CreatedAt           time.Time `gorm:"column:created_at" json:"created_at"`
	UpdatedAt           time.Time `gorm:"column:updated_at" json:"updated_at"`
}

func (VPSSizeCatalog) TableName() string {
	return "vps_size_catalog"
}

// VPSRegionCatalog represents a VPS region definition
type VPSRegionCatalog struct {
	ID        string    `gorm:"primaryKey;column:id" json:"id"`
	Name      string    `gorm:"column:name" json:"name"`
	Location  string    `gorm:"column:location" json:"location"`
	Country   string    `gorm:"column:country" json:"country"`
	Features  string    `gorm:"column:features;type:jsonb" json:"features"` // JSON array of features
	Available bool      `gorm:"column:available;default:true" json:"available"`
	CreatedAt time.Time `gorm:"column:created_at" json:"created_at"`
	UpdatedAt time.Time `gorm:"column:updated_at" json:"updated_at"`
}

func (VPSRegionCatalog) TableName() string {
	return "vps_region_catalog"
}

// GetVPSSizeCatalog retrieves a size from the catalog
func GetVPSSizeCatalog(sizeID string, region string) (*VPSSizeCatalog, error) {
	var size VPSSizeCatalog
	query := DB.Where("id = ? AND available = ?", sizeID, true)

	// Filter by region if specified (empty region = available in all regions)
	if region != "" {
		query = query.Where("(region = ? OR region = '')", region)
	}

	if err := query.First(&size).Error; err != nil {
		return nil, err
	}
	return &size, nil
}

// ListVPSSizeCatalog lists all available sizes, optionally filtered by region
func ListVPSSizeCatalog(region string) ([]VPSSizeCatalog, error) {
	var sizes []VPSSizeCatalog
	query := DB.Where("available = ?", true)

	if region != "" {
		query = query.Where("(region = ? OR region = '')", region)
	}

	if err := query.Order("price_cents_per_month ASC").Find(&sizes).Error; err != nil {
		return nil, err
	}
	return sizes, nil
}

// GetVPSRegionCatalog retrieves a region from the catalog
func GetVPSRegionCatalog(regionID string) (*VPSRegionCatalog, error) {
	var region VPSRegionCatalog
	if err := DB.Where("id = ? AND available = ?", regionID, true).First(&region).Error; err != nil {
		return nil, err
	}
	return &region, nil
}

// ListVPSRegionCatalog lists all available regions
func ListVPSRegionCatalog() ([]VPSRegionCatalog, error) {
	var regions []VPSRegionCatalog
	if err := DB.Where("available = ?", true).Order("name ASC").Find(&regions).Error; err != nil {
		return nil, err
	}
	return regions, nil
}

// ListAllVPSSizeCatalog lists all sizes, optionally including unavailable ones
func ListAllVPSSizeCatalog(region string, includeUnavailable bool) ([]VPSSizeCatalog, error) {
	var sizes []VPSSizeCatalog
	query := DB

	if !includeUnavailable {
		query = query.Where("available = ?", true)
	}

	if region != "" {
		query = query.Where("(region = ? OR region = '')", region)
	}

	if err := query.Order("price_cents_per_month ASC").Find(&sizes).Error; err != nil {
		return nil, err
	}
	return sizes, nil
}

// CreateVPSSizeCatalog creates a new VPS size in the catalog
func CreateVPSSizeCatalog(size *VPSSizeCatalog) error {
	if err := DB.Create(size).Error; err != nil {
		return fmt.Errorf("failed to create VPS size: %w", err)
	}
	return nil
}

// UpdateVPSSizeCatalog updates an existing VPS size in the catalog
func UpdateVPSSizeCatalog(sizeID string, updates map[string]interface{}) error {
	if err := DB.Model(&VPSSizeCatalog{}).Where("id = ?", sizeID).Updates(updates).Error; err != nil {
		return fmt.Errorf("failed to update VPS size: %w", err)
	}
	return nil
}

// DeleteVPSSizeCatalog deletes a VPS size from the catalog
func DeleteVPSSizeCatalog(sizeID string) error {
	if err := DB.Where("id = ?", sizeID).Delete(&VPSSizeCatalog{}).Error; err != nil {
		return fmt.Errorf("failed to delete VPS size: %w", err)
	}
	return nil
}

// InitVPSCatalog initializes the VPS catalog with default sizes and regions
func InitVPSCatalog() error {
	// Initialize default sizes
	defaultSizes := []VPSSizeCatalog{
		{
			ID:                  "small",
			Name:                "Small VPS",
			Description:         "Perfect for small applications, development, and testing. 1 CPU core, 1GB RAM, 10GB SSD storage.",
			CPUCores:            1,
			MemoryBytes:         1 * 1024 * 1024 * 1024,  // 1 GB
			DiskBytes:           10 * 1024 * 1024 * 1024, // 10 GB
			BandwidthBytesMonth: 0,                       // Unlimited
			PriceCentsPerMonth:  500,                     // $5/month
			Available:           true,
			Region:              "", // Available in all regions
		},
		{
			ID:                  "medium",
			Name:                "Medium VPS",
			Description:         "Ideal for medium-sized applications and production workloads. 2 CPU cores, 2GB RAM, 20GB SSD storage.",
			CPUCores:            2,
			MemoryBytes:         2 * 1024 * 1024 * 1024,  // 2 GB
			DiskBytes:           20 * 1024 * 1024 * 1024, // 20 GB
			BandwidthBytesMonth: 0,                       // Unlimited
			PriceCentsPerMonth:  1000,                    // $10/month
			Available:           true,
			Region:              "",
		},
		{
			ID:                  "large",
			Name:                "Large VPS",
			Description:         "Great for larger applications and high-traffic websites. 4 CPU cores, 4GB RAM, 40GB SSD storage.",
			CPUCores:            4,
			MemoryBytes:         4 * 1024 * 1024 * 1024,  // 4 GB
			DiskBytes:           40 * 1024 * 1024 * 1024, // 40 GB
			BandwidthBytesMonth: 0,                       // Unlimited
			PriceCentsPerMonth:  2000,                    // $20/month
			Available:           true,
			Region:              "",
		},
		{
			ID:                  "xlarge",
			Name:                "Extra Large VPS",
			Description:         "For resource-intensive applications and databases. 8 CPU cores, 8GB RAM, 80GB SSD storage.",
			CPUCores:            8,
			MemoryBytes:         8 * 1024 * 1024 * 1024,  // 8 GB
			DiskBytes:           80 * 1024 * 1024 * 1024, // 80 GB
			BandwidthBytesMonth: 0,                       // Unlimited
			PriceCentsPerMonth:  4000,                    // $40/month
			Available:           true,
			Region:              "",
		},
	}

	// Upsert sizes
	for _, size := range defaultSizes {
		var existing VPSSizeCatalog
		if err := DB.Where("id = ?", size.ID).First(&existing).Error; err != nil {
			if err == gorm.ErrRecordNotFound {
				// Create new
				if err := DB.Create(&size).Error; err != nil {
					return fmt.Errorf("failed to create size %s: %w", size.ID, err)
				}
			} else {
				return fmt.Errorf("failed to check size %s: %w", size.ID, err)
			}
		} else {
			// Update existing
			size.CreatedAt = existing.CreatedAt
			if err := DB.Save(&size).Error; err != nil {
				return fmt.Errorf("failed to update size %s: %w", size.ID, err)
			}
		}
	}

	// Initialize default regions
	defaultRegions := []VPSRegionCatalog{
		{
			ID:        "us-east-1",
			Name:      "US East (N. Virginia)",
			Location:  "Ashburn, Virginia, USA",
			Country:   "US",
			Features:  `["nvme_storage", "low_latency"]`,
			Available: true,
		},
		{
			ID:        "us-west-1",
			Name:      "US West (California)",
			Location:  "San Francisco, California, USA",
			Country:   "US",
			Features:  `["nvme_storage"]`,
			Available: true,
		},
		{
			ID:        "eu-west-1",
			Name:      "EU West (Ireland)",
			Location:  "Dublin, Ireland",
			Country:   "IE",
			Features:  `["nvme_storage", "gdpr_compliant"]`,
			Available: true,
		},
		{
			ID:        "eu-central-1",
			Name:      "EU Central (Germany)",
			Location:  "Frankfurt, Germany",
			Country:   "DE",
			Features:  `["nvme_storage", "gdpr_compliant"]`,
			Available: true,
		},
	}

	// Upsert regions
	for _, region := range defaultRegions {
		var existing VPSRegionCatalog
		if err := DB.Where("id = ?", region.ID).First(&existing).Error; err != nil {
			if err == gorm.ErrRecordNotFound {
				// Create new
				if err := DB.Create(&region).Error; err != nil {
					return fmt.Errorf("failed to create region %s: %w", region.ID, err)
				}
			} else {
				return fmt.Errorf("failed to check region %s: %w", region.ID, err)
			}
		} else {
			// Update existing
			region.CreatedAt = existing.CreatedAt
			if err := DB.Save(&region).Error; err != nil {
				return fmt.Errorf("failed to update region %s: %w", region.ID, err)
			}
		}
	}

	return nil
}
