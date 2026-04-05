package database

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/moby/moby/client"
	"gorm.io/gorm"

	"github.com/obiente/cloud/apps/shared/pkg/logger"
)

// DeploymentLocation tracks where deployments are running across the cluster
type DeploymentLocation struct {
	ID              string    `gorm:"primaryKey" json:"id"`
	DeploymentID    string    `gorm:"index;not null" json:"deployment_id"`
	NodeID          string    `gorm:"index;not null" json:"node_id"`          // Swarm node ID
	NodeHostname    string    `json:"node_hostname"`                          // Swarm node hostname
	NodeIP          string    `json:"node_ip"`                                // Node IP address
	ContainerID     string    `gorm:"uniqueIndex" json:"container_id"`        // Docker container ID
	ServiceID       string    `gorm:"index" json:"service_id"`                // Docker service ID (if using services)
	TaskID          string    `json:"task_id"`                                // Swarm task ID
	Status          string    `gorm:"index;not null" json:"status"`           // running, stopped, failed, etc.
	Port            int       `json:"port"`                                   // Assigned port for this deployment
	Domain          string    `gorm:"index" json:"domain"`                    // Custom domain for this deployment
	HealthStatus    string    `gorm:"default:'unknown'" json:"health_status"` // healthy, unhealthy, unknown
	LastHealthCheck time.Time `json:"last_health_check"`
	CPUUsage        float64   `json:"cpu_usage"`    // CPU usage percentage
	MemoryUsage     int64     `json:"memory_usage"` // Memory usage in bytes
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

// NodeMetadata stores information about cluster nodes
type NodeMetadata struct {
	ID              string    `gorm:"primaryKey" json:"id"` // Swarm node ID
	Hostname        string    `gorm:"uniqueIndex;not null" json:"hostname"`
	IP              string    `json:"ip"`
	Role            string    `gorm:"index" json:"role"`                 // manager, worker
	Availability    string    `gorm:"index" json:"availability"`         // active, pause, drain
	Status          string    `json:"status"`                            // ready, down
	TotalCPU        int       `json:"total_cpu"`                         // Total CPU cores
	TotalMemory     int64     `json:"total_memory"`                      // Total memory in bytes
	UsedCPU         float64   `json:"used_cpu"`                          // Used CPU percentage
	UsedMemory      int64     `json:"used_memory"`                       // Used memory in bytes
	DeploymentCount int       `gorm:"default:0" json:"deployment_count"` // Number of deployments on this node
	MaxDeployments  int       `gorm:"default:50" json:"max_deployments"` // Max deployments allowed
	Labels          string    `gorm:"type:jsonb" json:"labels"`          // Node labels (JSON)
	Region          string    `gorm:"index" json:"region"`               // Region identifier (e.g., "us-east-1", "eu-west-1")
	LastHeartbeat   time.Time `json:"last_heartbeat"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

// DeploymentRouting stores routing configuration for deployments
// Supports multiple routing rules per deployment (e.g., different services/ports on different domains)
type DeploymentRouting struct {
	ID              string    `gorm:"primaryKey" json:"id"`
	DeploymentID    string    `gorm:"index:idx_deployment_domain_service;not null" json:"deployment_id"`
	Domain          string    `gorm:"index:idx_deployment_domain_service;not null" json:"domain"`
	ServiceName     string    `gorm:"index:idx_deployment_domain_service;default:default" json:"service_name"` // Service name (e.g., "api", "web", "admin")
	PathPrefix      string    `json:"path_prefix"`
	TargetPort      int       `gorm:"not null" json:"target_port"`
	Protocol        string    `gorm:"default:'http'" json:"protocol"`   // http, https, grpc
	SSLEnabled      bool      `gorm:"default:false" json:"ssl_enabled"` // Default to false (HTTP protocol doesn't use SSL)
	SSLCertResolver string    `json:"ssl_cert_resolver"`
	Middleware      string    `gorm:"type:jsonb" json:"middleware"` // Middleware configuration (JSON)
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

// GameServerHTTPRoute stores HTTP routing configuration for game servers.
// These routes are translated into Traefik labels on the game server container.
type GameServerHTTPRoute struct {
	ID              string    `gorm:"primaryKey" json:"id"`
	GameServerID    string    `gorm:"index:idx_gs_route_domain;not null" json:"game_server_id"`
	Domain          string    `gorm:"index:idx_gs_route_domain;not null" json:"domain"`
	PathPrefix      string    `json:"path_prefix"`
	TargetPort      int       `gorm:"not null" json:"target_port"`
	Protocol        string    `gorm:"default:'http'" json:"protocol"`
	SSLEnabled      bool      `gorm:"default:false" json:"ssl_enabled"`
	SSLCertResolver string    `json:"ssl_cert_resolver"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

// GameServerDomainVerification tracks custom domain verification state for game servers.
type GameServerDomainVerification struct {
	ID           string     `gorm:"primaryKey" json:"id"`
	GameServerID string     `gorm:"index:idx_gs_domain_verification;not null" json:"game_server_id"`
	Domain       string     `gorm:"index:idx_gs_domain_verification;not null" json:"domain"`
	Token        string     `gorm:"not null" json:"token"`
	Status       string     `gorm:"index;default:'pending'" json:"status"`
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`
	VerifiedAt   *time.Time `json:"verified_at,omitempty"`
}

// DeploymentMetrics stores historical metrics for deployments
type DeploymentMetrics struct {
	ID             uint      `gorm:"primaryKey" json:"id"`
	DeploymentID   string    `gorm:"index;not null" json:"deployment_id"`
	ContainerID    string    `gorm:"index" json:"container_id"` // Container ID for multi-container deployments
	NodeID         string    `gorm:"index" json:"node_id"`
	CPUUsage       float64   `json:"cpu_usage"`
	MemoryUsage    int64     `json:"memory_usage"`
	NetworkRxBytes int64     `json:"network_rx_bytes"`
	NetworkTxBytes int64     `json:"network_tx_bytes"`
	DiskReadBytes  int64     `json:"disk_read_bytes"`
	DiskWriteBytes int64     `json:"disk_write_bytes"`
	RequestCount   int64     `json:"request_count"`
	ErrorCount     int64     `json:"error_count"`
	Timestamp      time.Time `gorm:"index" json:"timestamp"`
}

// InitDeploymentTracking creates the tables for deployment tracking
func InitDeploymentTracking() error {
	if err := DB.AutoMigrate(
		&DeploymentLocation{},
		&NodeMetadata{},
		&DeploymentRouting{},
		&GameServerHTTPRoute{},
		&GameServerDomainVerification{},
		&GameServerLocation{},
		&DatabaseLocation{},
	); err != nil {
		return err
	}

	// Note: DeploymentMetrics and DeploymentUsageHourly are now handled by InitMetricsTables
	// to use the separate metrics database

	return nil
}

// createMetricsIndexes creates composite indexes for metrics queries
// Requires MetricsDB (TimescaleDB) - will fail if not available
func createMetricsIndexes() error {
	if MetricsDB == nil {
		return fmt.Errorf("metrics database (TimescaleDB) not initialized - cannot create metrics indexes")
	}
	db := MetricsDB

	// Composite index for deployment_id + timestamp (most common query pattern)
	if err := db.Exec(`
		CREATE INDEX IF NOT EXISTS idx_deployment_metrics_deployment_timestamp 
		ON deployment_metrics(deployment_id, timestamp DESC)
	`).Error; err != nil {
		return fmt.Errorf("failed to create deployment_timestamp index: %w", err)
	}

	// Composite index for timestamp + deployment_id (for time-range queries)
	if err := db.Exec(`
		CREATE INDEX IF NOT EXISTS idx_deployment_metrics_timestamp_deployment 
		ON deployment_metrics(timestamp DESC, deployment_id)
	`).Error; err != nil {
		return fmt.Errorf("failed to create timestamp_deployment index: %w", err)
	}

	// Composite index for container_id + timestamp (container-specific queries)
	if err := db.Exec(`
		CREATE INDEX IF NOT EXISTS idx_deployment_metrics_container_timestamp 
		ON deployment_metrics(container_id, timestamp DESC)
	`).Error; err != nil {
		return fmt.Errorf("failed to create container_timestamp index: %w", err)
	}

	// Index for hourly aggregates (deployment_id + hour)
	if err := db.Exec(`
		CREATE INDEX IF NOT EXISTS idx_deployment_usage_hourly_deployment_hour 
		ON deployment_usage_hourly(deployment_id, hour DESC)
	`).Error; err != nil {
		return fmt.Errorf("failed to create usage_hourly index: %w", err)
	}

	if err := db.Exec(`
		CREATE INDEX IF NOT EXISTS idx_deployment_runtime_logs_deployment_timestamp
		ON deployment_runtime_logs(deployment_id, timestamp DESC)
	`).Error; err != nil {
		return fmt.Errorf("failed to create deployment_runtime_logs index: %w", err)
	}

	if err := db.Exec(`
		CREATE INDEX IF NOT EXISTS idx_deployment_runtime_logs_service_timestamp
		ON deployment_runtime_logs(service_name, timestamp DESC)
	`).Error; err != nil {
		return fmt.Errorf("failed to create deployment_runtime_logs service index: %w", err)
	}

	// Game server metrics indexes
	if err := db.Exec(`
		CREATE INDEX IF NOT EXISTS idx_game_server_metrics_gameserver_timestamp 
		ON game_server_metrics(game_server_id, timestamp DESC)
	`).Error; err != nil {
		return fmt.Errorf("failed to create game_server_timestamp index: %w", err)
	}

	if err := db.Exec(`
		CREATE INDEX IF NOT EXISTS idx_game_server_metrics_timestamp_gameserver 
		ON game_server_metrics(timestamp DESC, game_server_id)
	`).Error; err != nil {
		return fmt.Errorf("failed to create game_server_timestamp_deployment index: %w", err)
	}

	if err := db.Exec(`
		CREATE INDEX IF NOT EXISTS idx_game_server_metrics_container_timestamp 
		ON game_server_metrics(container_id, timestamp DESC)
	`).Error; err != nil {
		return fmt.Errorf("failed to create game_server_container_timestamp index: %w", err)
	}

	if err := db.Exec(`
		CREATE INDEX IF NOT EXISTS idx_game_server_usage_hourly_gameserver_hour 
		ON game_server_usage_hourly(game_server_id, hour DESC)
	`).Error; err != nil {
		return fmt.Errorf("failed to create game_server_usage_hourly index: %w", err)
	}

	return nil
}

// GetDeploymentLocations returns all locations where a deployment is running
func GetDeploymentLocations(deploymentID string) ([]DeploymentLocation, error) {
	var locations []DeploymentLocation
	result := DB.Where("deployment_id = ? AND status = ?", deploymentID, "running").Find(&locations)
	return locations, result.Error
}

// GetAllDeploymentLocations returns all locations for a deployment regardless of status
func GetAllDeploymentLocations(deploymentID string) ([]DeploymentLocation, error) {
	var locations []DeploymentLocation
	result := DB.Where("deployment_id = ?", deploymentID).Find(&locations)
	return locations, result.Error
}

// GetAllDeploymentLocationsByDeploymentIDs returns all tracked locations grouped by deployment ID.
func GetAllDeploymentLocationsByDeploymentIDs(deploymentIDs []string) (map[string][]DeploymentLocation, error) {
	if len(deploymentIDs) == 0 {
		return map[string][]DeploymentLocation{}, nil
	}

	var locations []DeploymentLocation
	if err := DB.Where("deployment_id IN ?", deploymentIDs).Find(&locations).Error; err != nil {
		return nil, err
	}

	grouped := make(map[string][]DeploymentLocation, len(deploymentIDs))
	for _, location := range locations {
		grouped[location.DeploymentID] = append(grouped[location.DeploymentID], location)
	}

	return grouped, nil
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

// GetProxmoxNodes returns nodes with Proxmox capability
func GetProxmoxNodes() ([]NodeMetadata, error) {
	var allNodes []NodeMetadata
	result := DB.Where("availability = ? AND status = ?", "active", "ready").
		Find(&allNodes)
	if result.Error != nil {
		return nil, result.Error
	}

	// Filter nodes with Proxmox capability
	var proxmoxNodes []NodeMetadata
	for _, node := range allNodes {
		if HasProxmoxCapability(node) {
			proxmoxNodes = append(proxmoxNodes, node)
		}
	}

	return proxmoxNodes, nil
}

// HasProxmoxCapability checks if a node has Proxmox capability based on labels
// Exported so it can be used by other packages
func HasProxmoxCapability(node NodeMetadata) bool {
	if node.Labels == "" {
		return false
	}

	var labels map[string]interface{}
	if err := json.Unmarshal([]byte(node.Labels), &labels); err != nil {
		return false
	}

	// Check if node has proxmox label set to true
	if proxmox, ok := labels["proxmox"].(string); ok && proxmox == "true" {
		return true
	}
	if proxmox, ok := labels["proxmox"].(bool); ok && proxmox {
		return true
	}

	return false
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
// Uses upsert logic:
//   - for Swarm-backed services, prefer a stable logical row keyed by deployment_id + service_id
//   - otherwise, fall back to container_id
func RecordDeploymentLocation(location *DeploymentLocation) error {
	return DB.Transaction(func(tx *gorm.DB) error {
		if location.ServiceID != "" {
			var existingByService DeploymentLocation
			serviceResult := tx.Where("deployment_id = ? AND service_id = ?", location.DeploymentID, location.ServiceID).First(&existingByService)
			if serviceResult.Error == nil {
				location.ID = existingByService.ID
				if err := tx.Model(&existingByService).Updates(location).Error; err != nil {
					return err
				}
				return nil
			}
			if serviceResult.Error != nil && serviceResult.Error != gorm.ErrRecordNotFound {
				return serviceResult.Error
			}
		}

		// Check if location with this container ID already exists
		var existing DeploymentLocation
		result := tx.Where("container_id = ?", location.ContainerID).First(&existing)

		if result.Error == nil {
			// Location exists - update it (don't change ID)
			location.ID = existing.ID
			if err := tx.Model(&existing).Updates(location).Error; err != nil {
				return err
			}
		} else if result.Error == gorm.ErrRecordNotFound {
			// Location doesn't exist - create new
			// Ensure ID is set
			if location.ID == "" {
				location.ID = fmt.Sprintf("loc-%s-%s", location.DeploymentID, location.ContainerID[:12])
			}

			if err := tx.Create(location).Error; err != nil {
				return err
			}

			// Increment deployment count on the node only for new locations
			if err := tx.Model(&NodeMetadata{}).
				Where("id = ?", location.NodeID).
				UpdateColumn("deployment_count", gorm.Expr("deployment_count + ?", 1)).
				Error; err != nil {
				return err
			}
		} else {
			// Other database error
			return result.Error
		}

		return nil
	})
}

// ValidateAndRefreshLocations validates container IDs and removes stale entries
// Returns fresh locations after cleanup
func ValidateAndRefreshLocations(deploymentID string) ([]DeploymentLocation, error) {
	// Use moby client directly
	mobyClient, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return nil, fmt.Errorf("failed to create docker client: %w", err)
	}
	defer mobyClient.Close()

	// Get current locations from DB
	locations, err := GetDeploymentLocations(deploymentID)
	if err != nil {
		return nil, err
	}

	var validLocations []DeploymentLocation
	for _, loc := range locations {
		// Try to inspect the container to verify it exists
		_, err := mobyClient.ContainerInspect(context.Background(), loc.ContainerID, client.ContainerInspectOptions{})
		if err != nil {
			// Container doesn't exist on this node
			// In Swarm mode, container might be on a different node, so be conservative
			// Don't remove the location record - it might be running on another node
			// The periodic sync on the correct node will handle cleanup
			// Only include in validLocations if we can verify it exists locally
			logger.Debug("[ValidateAndRefreshLocations] Container %s not found on local node (location node: %s), preserving record for DNS", loc.ContainerID[:12], loc.NodeID)
			// Don't include in validLocations since we can't verify it exists
			// But don't remove it either - let the sync on the correct node handle it
			continue
		}
		// Container exists on this node - include it
		validLocations = append(validLocations, loc)
	}

	// If we couldn't verify any running containers locally, try to discover actual containers
	// using the deployment label. This also covers the case where there are NO DB records yet.
	if len(validLocations) == 0 {
		logger.Info("[ValidateAndRefreshLocations] No locally verifiable containers found, attempting to discover actual containers for deployment %s", deploymentID)

		// Look for containers with deployment label using moby filters
		filterArgs := make(client.Filters)
		filterArgs.Add("label", fmt.Sprintf("cloud.obiente.deployment_id=%s", deploymentID))

		containersResult, listErr := mobyClient.ContainerList(context.Background(), client.ContainerListOptions{
			All:     true,
			Filters: filterArgs,
		})

		if listErr == nil && len(containersResult.Items) > 0 {
			logger.Info("[ValidateAndRefreshLocations] Found %d actual containers for deployment %s", len(containersResult.Items), deploymentID)
			// Register the actual containers
			for _, c := range containersResult.Items {
				// Get container details to extract full info
				infoResult, infoErr := mobyClient.ContainerInspect(context.Background(), c.ID, client.ContainerInspectOptions{})
				if infoErr != nil {
					continue
				}
				info := infoResult.Container

				// Extract deployment ID from labels
				var containerDeploymentID string
				if info.Config != nil && info.Config.Labels != nil {
					containerDeploymentID = info.Config.Labels["cloud.obiente.deployment_id"]
				}

				if containerDeploymentID != deploymentID {
					continue
				}

				// Get node info for registration
				// Note: InspectResponse doesn't have Node field in standard Docker API
				// We'll use default values and let the registry sync handle proper node info
				nodeID := "local-unknown"
				nodeHostname := "unknown"

				// Get domain from labels
				domain := ""
				if info.Config != nil && info.Config.Labels != nil {
					domain = info.Config.Labels["cloud.obiente.domain"]
				}

				location := &DeploymentLocation{
					ID:           fmt.Sprintf("loc-%s-%s", deploymentID, c.ID[:12]),
					DeploymentID: deploymentID,
					NodeID:       nodeID,
					NodeHostname: nodeHostname,
					ContainerID:  c.ID,
					Status:       string(c.State),
					Domain:       domain,
				}

				// Extract port if available
				if len(c.Ports) > 0 {
					location.Port = int(c.Ports[0].PublicPort)
				}

				if regErr := RecordDeploymentLocation(location); regErr == nil {
					validLocations = append(validLocations, *location)
					logger.Info("[ValidateAndRefreshLocations] Registered actual container %s for deployment %s", c.ID[:12], deploymentID)
				}
			}
		}
	}

	return validLocations, nil
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
	// Check if routing already exists
	var existing DeploymentRouting
	err := DB.Where("id = ?", routing.ID).First(&existing).Error

	if err == nil {
		// Update existing routing - preserve CreatedAt timestamp
		// Only update UpdatedAt and other fields, not CreatedAt
		updateData := map[string]interface{}{
			"deployment_id":     routing.DeploymentID,
			"domain":            routing.Domain,
			"service_name":      routing.ServiceName,
			"path_prefix":       routing.PathPrefix,
			"target_port":       routing.TargetPort,
			"protocol":          routing.Protocol,
			"ssl_enabled":       routing.SSLEnabled,
			"ssl_cert_resolver": routing.SSLCertResolver,
			"middleware":        routing.Middleware,
			"updated_at":        time.Now(),
		}
		return DB.Model(&existing).Updates(updateData).Error
	} else if err == gorm.ErrRecordNotFound {
		// Create new routing
		if routing.CreatedAt.IsZero() {
			routing.CreatedAt = time.Now()
		}
		if routing.UpdatedAt.IsZero() {
			routing.UpdatedAt = time.Now()
		}
		return DB.Create(routing).Error
	} else {
		// Database error
		return err
	}
}

// GetGameServerHTTPRoutes returns all HTTP routes for a game server.
func GetGameServerHTTPRoutes(gameServerID string) ([]GameServerHTTPRoute, error) {
	var routes []GameServerHTTPRoute
	result := DB.Where("game_server_id = ?", gameServerID).Find(&routes)
	return routes, result.Error
}

// GetGameServerHTTPRouteByDomain returns a game server HTTP route for a specific domain.
func GetGameServerHTTPRouteByDomain(domain string) (*GameServerHTTPRoute, error) {
	var route GameServerHTTPRoute
	result := DB.Where("domain = ?", domain).First(&route)
	return &route, result.Error
}

// GetGameServerHTTPRouteByID returns a specific game server HTTP route.
func GetGameServerHTTPRouteByID(routeID string) (*GameServerHTTPRoute, error) {
	var route GameServerHTTPRoute
	result := DB.Where("id = ?", routeID).First(&route)
	return &route, result.Error
}

// UpsertGameServerHTTPRoute creates or updates a game server HTTP route.
func UpsertGameServerHTTPRoute(route *GameServerHTTPRoute) error {
	var existing GameServerHTTPRoute
	err := DB.Where("id = ?", route.ID).First(&existing).Error

	if err == nil {
		updateData := map[string]interface{}{
			"game_server_id":    route.GameServerID,
			"domain":            route.Domain,
			"path_prefix":       route.PathPrefix,
			"target_port":       route.TargetPort,
			"protocol":          route.Protocol,
			"ssl_enabled":       route.SSLEnabled,
			"ssl_cert_resolver": route.SSLCertResolver,
			"updated_at":        time.Now(),
		}
		return DB.Model(&existing).Updates(updateData).Error
	} else if err == gorm.ErrRecordNotFound {
		if route.CreatedAt.IsZero() {
			route.CreatedAt = time.Now()
		}
		if route.UpdatedAt.IsZero() {
			route.UpdatedAt = time.Now()
		}
		return DB.Create(route).Error
	}

	return err
}

// DeleteGameServerHTTPRoute removes a game server HTTP route.
func DeleteGameServerHTTPRoute(routeID string) error {
	return DB.Delete(&GameServerHTTPRoute{}, "id = ?", routeID).Error
}

// GetGameServerDomainVerification returns a domain verification record for a game server/domain pair.
func GetGameServerDomainVerification(gameServerID string, domain string) (*GameServerDomainVerification, error) {
	var verification GameServerDomainVerification
	result := DB.Where("game_server_id = ? AND domain = ?", gameServerID, domain).First(&verification)
	return &verification, result.Error
}

// UpsertGameServerDomainVerification creates or updates a game server domain verification record.
func UpsertGameServerDomainVerification(verification *GameServerDomainVerification) error {
	var existing GameServerDomainVerification
	err := DB.Where("game_server_id = ? AND domain = ?", verification.GameServerID, verification.Domain).First(&existing).Error

	if err == nil {
		updateData := map[string]interface{}{
			"token":       verification.Token,
			"status":      verification.Status,
			"updated_at":  time.Now(),
			"verified_at": verification.VerifiedAt,
		}
		return DB.Model(&existing).Updates(updateData).Error
	} else if err == gorm.ErrRecordNotFound {
		if verification.CreatedAt.IsZero() {
			verification.CreatedAt = time.Now()
		}
		if verification.UpdatedAt.IsZero() {
			verification.UpdatedAt = time.Now()
		}
		return DB.Create(verification).Error
	}

	return err
}

// GetVerifiedGameServerDomains returns verified custom domains for a game server.
func GetVerifiedGameServerDomains(gameServerID string) ([]GameServerDomainVerification, error) {
	var records []GameServerDomainVerification
	result := DB.Where("game_server_id = ? AND status = ?", gameServerID, "verified").Find(&records)
	return records, result.Error
}

// RecordMetrics records deployment metrics
// Uses MetricsDB if available, otherwise falls back to main DB
func RecordMetrics(metrics *DeploymentMetrics) error {
	targetDB := MetricsDB
	if targetDB == nil {
		targetDB = DB
	}
	return targetDB.Create(metrics).Error
}

// GetRecentMetrics gets recent metrics for a deployment
// Uses MetricsDB if available, otherwise falls back to main DB
func GetRecentMetrics(deploymentID string, since time.Time) ([]DeploymentMetrics, error) {
	var metrics []DeploymentMetrics
	targetDB := MetricsDB
	if targetDB == nil {
		targetDB = DB
	}
	result := targetDB.Where("deployment_id = ? AND timestamp >= ?", deploymentID, since).
		Order("timestamp DESC").
		Limit(1000).
		Find(&metrics)
	return metrics, result.Error
}

// CleanOldMetrics removes metrics older than retention period
// Uses MetricsDB if available, otherwise falls back to main DB
func CleanOldMetrics(retentionDays int) error {
	cutoff := time.Now().AddDate(0, 0, -retentionDays)
	targetDB := MetricsDB
	if targetDB == nil {
		targetDB = DB
	}
	return targetDB.Where("timestamp < ?", cutoff).Delete(&DeploymentMetrics{}).Error
}
