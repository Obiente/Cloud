package database

import (
	"time"

	"gorm.io/gorm"
)

// DeploymentLocation tracks where deployments are running across the cluster
type DeploymentLocation struct {
	ID           string    `gorm:"primaryKey" json:"id"`
	DeploymentID string    `gorm:"index;not null" json:"deployment_id"`
	NodeID       string    `gorm:"index;not null" json:"node_id"`       // Swarm node ID
	NodeHostname string    `json:"node_hostname"`                       // Swarm node hostname
	NodeIP       string    `json:"node_ip"`                             // Node IP address
	ContainerID  string    `gorm:"uniqueIndex" json:"container_id"`     // Docker container ID
	ServiceID    string    `gorm:"index" json:"service_id"`             // Docker service ID (if using services)
	TaskID       string    `json:"task_id"`                             // Swarm task ID
	Status       string    `gorm:"index;not null" json:"status"`        // running, stopped, failed, etc.
	Port         int       `json:"port"`                                // Assigned port for this deployment
	Domain       string    `gorm:"index" json:"domain"`                 // Custom domain for this deployment
	HealthStatus string    `gorm:"default:'unknown'" json:"health_status"` // healthy, unhealthy, unknown
	LastHealthCheck time.Time `json:"last_health_check"`
	CPUUsage     float64   `json:"cpu_usage"`                           // CPU usage percentage
	MemoryUsage  int64     `json:"memory_usage"`                        // Memory usage in bytes
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// NodeMetadata stores information about cluster nodes
type NodeMetadata struct {
	ID               string    `gorm:"primaryKey" json:"id"`              // Swarm node ID
	Hostname         string    `gorm:"uniqueIndex;not null" json:"hostname"`
	IP               string    `json:"ip"`
	Role             string    `gorm:"index" json:"role"`                 // manager, worker
	Availability     string    `gorm:"index" json:"availability"`         // active, pause, drain
	Status           string    `json:"status"`                            // ready, down
	TotalCPU         int       `json:"total_cpu"`                         // Total CPU cores
	TotalMemory      int64     `json:"total_memory"`                      // Total memory in bytes
	UsedCPU          float64   `json:"used_cpu"`                          // Used CPU percentage
	UsedMemory       int64     `json:"used_memory"`                       // Used memory in bytes
	DeploymentCount  int       `gorm:"default:0" json:"deployment_count"` // Number of deployments on this node
	MaxDeployments   int       `gorm:"default:50" json:"max_deployments"` // Max deployments allowed
	Labels           string    `gorm:"type:jsonb" json:"labels"`          // Node labels (JSON)
	LastHeartbeat    time.Time `json:"last_heartbeat"`
	CreatedAt        time.Time `json:"created_at"`
	UpdatedAt        time.Time `json:"updated_at"`
}

// DeploymentRouting stores routing configuration for deployments
// Supports multiple routing rules per deployment (e.g., different services/ports on different domains)
type DeploymentRouting struct {
	ID               string    `gorm:"primaryKey" json:"id"`
	DeploymentID     string    `gorm:"index:idx_deployment_domain_service;not null" json:"deployment_id"`
	Domain           string    `gorm:"index:idx_deployment_domain_service;not null" json:"domain"`
	ServiceName      string    `gorm:"index:idx_deployment_domain_service;default:default" json:"service_name"` // Service name (e.g., "api", "web", "admin")
	PathPrefix       string    `json:"path_prefix"`
	TargetPort       int       `gorm:"not null" json:"target_port"`
	Protocol         string    `gorm:"default:'http'" json:"protocol"` // http, https, grpc
	LoadBalancerAlgo string    `gorm:"default:'round-robin'" json:"load_balancer_algo"` // round-robin, least-conn, ip-hash
	SSLEnabled       bool      `gorm:"default:true" json:"ssl_enabled"`
	SSLCertResolver  string    `json:"ssl_cert_resolver"`
	Middleware       string    `gorm:"type:jsonb" json:"middleware"` // Middleware configuration (JSON)
	CreatedAt        time.Time `json:"created_at"`
	UpdatedAt        time.Time `json:"updated_at"`
}

// DeploymentMetrics stores historical metrics for deployments
type DeploymentMetrics struct {
	ID            uint      `gorm:"primaryKey" json:"id"`
	DeploymentID  string    `gorm:"index;not null" json:"deployment_id"`
	NodeID        string    `gorm:"index" json:"node_id"`
	CPUUsage      float64   `json:"cpu_usage"`
	MemoryUsage   int64     `json:"memory_usage"`
	NetworkRxBytes int64    `json:"network_rx_bytes"`
	NetworkTxBytes int64    `json:"network_tx_bytes"`
	DiskReadBytes int64     `json:"disk_read_bytes"`
	DiskWriteBytes int64    `json:"disk_write_bytes"`
	RequestCount   int64    `json:"request_count"`
	ErrorCount     int64    `json:"error_count"`
	Timestamp      time.Time `gorm:"index" json:"timestamp"`
}

// InitDeploymentTracking creates the tables for deployment tracking
func InitDeploymentTracking() error {
	if err := DB.AutoMigrate(
		&DeploymentLocation{},
		&NodeMetadata{},
		&DeploymentRouting{},
		&DeploymentMetrics{},
	); err != nil {
		return err
	}
	return nil
}

// GetDeploymentLocations returns all locations where a deployment is running
func GetDeploymentLocations(deploymentID string) ([]DeploymentLocation, error) {
	var locations []DeploymentLocation
	result := DB.Where("deployment_id = ? AND status = ?", deploymentID, "running").Find(&locations)
	return locations, result.Error
}

// GetNodeByID returns node metadata by ID
func GetNodeByID(nodeID string) (*NodeMetadata, error) {
	var node NodeMetadata
	result := DB.First(&node, "id = ?", nodeID)
	return &node, result.Error
}

// GetAvailableNodes returns nodes available for deployment
func GetAvailableNodes() ([]NodeMetadata, error) {
	var nodes []NodeMetadata
	result := DB.Where("availability = ? AND status = ?", "active", "ready").
		Where("deployment_count < max_deployments").
		Order("deployment_count ASC, used_cpu ASC, used_memory ASC").
		Find(&nodes)
	return nodes, result.Error
}

// UpdateNodeMetrics updates resource usage for a node
func UpdateNodeMetrics(nodeID string, usedCPU float64, usedMemory int64) error {
	return DB.Model(&NodeMetadata{}).
		Where("id = ?", nodeID).
		Updates(map[string]interface{}{
			"used_cpu":       usedCPU,
			"used_memory":    usedMemory,
			"last_heartbeat": time.Now(),
		}).Error
}

// RecordDeploymentLocation records a new deployment location
func RecordDeploymentLocation(location *DeploymentLocation) error {
	return DB.Transaction(func(tx *gorm.DB) error {
		// Create deployment location
		if err := tx.Create(location).Error; err != nil {
			return err
		}
		
		// Increment deployment count on the node
		if err := tx.Model(&NodeMetadata{}).
			Where("id = ?", location.NodeID).
			UpdateColumn("deployment_count", gorm.Expr("deployment_count + ?", 1)).
			Error; err != nil {
			return err
		}
		
		return nil
	})
}

// RemoveDeploymentLocation removes a deployment location
func RemoveDeploymentLocation(containerID string) error {
	return DB.Transaction(func(tx *gorm.DB) error {
		var location DeploymentLocation
		if err := tx.Where("container_id = ?", containerID).First(&location).Error; err != nil {
			return err
		}
		
		// Delete deployment location
		if err := tx.Delete(&location).Error; err != nil {
			return err
		}
		
		// Decrement deployment count on the node
		if err := tx.Model(&NodeMetadata{}).
			Where("id = ?", location.NodeID).
			UpdateColumn("deployment_count", gorm.Expr("deployment_count - ?", 1)).
			Error; err != nil {
			return err
		}
		
		return nil
	})
}

// GetDeploymentRouting returns routing configuration for a deployment
func GetDeploymentRouting(deploymentID string) (*DeploymentRouting, error) {
	var routing DeploymentRouting
	result := DB.First(&routing, "deployment_id = ?", deploymentID)
	return &routing, result.Error
}

// GetDeploymentRoutings returns all routing configurations for a deployment
func GetDeploymentRoutings(deploymentID string) ([]DeploymentRouting, error) {
	var routings []DeploymentRouting
	result := DB.Where("deployment_id = ?", deploymentID).Find(&routings)
	return routings, result.Error
}

// GetDeploymentRoutingByDomain returns routing configuration for a specific domain
func GetDeploymentRoutingByDomain(domain string) (*DeploymentRouting, error) {
	var routing DeploymentRouting
	result := DB.Where("domain = ?", domain).First(&routing)
	return &routing, result.Error
}

// UpsertDeploymentRouting creates or updates deployment routing
func UpsertDeploymentRouting(routing *DeploymentRouting) error {
	return DB.Save(routing).Error
}

// RecordMetrics records deployment metrics
func RecordMetrics(metrics *DeploymentMetrics) error {
	return DB.Create(metrics).Error
}

// GetRecentMetrics gets recent metrics for a deployment
func GetRecentMetrics(deploymentID string, since time.Time) ([]DeploymentMetrics, error) {
	var metrics []DeploymentMetrics
	result := DB.Where("deployment_id = ? AND timestamp >= ?", deploymentID, since).
		Order("timestamp DESC").
		Limit(1000).
		Find(&metrics)
	return metrics, result.Error
}

// CleanOldMetrics removes metrics older than retention period
func CleanOldMetrics(retentionDays int) error {
	cutoff := time.Now().AddDate(0, 0, -retentionDays)
	return DB.Where("timestamp < ?", cutoff).Delete(&DeploymentMetrics{}).Error
}

