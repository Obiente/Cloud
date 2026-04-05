package provisioner

import (
	"context"
	"fmt"
	"time"

	"github.com/obiente/cloud/apps/shared/pkg/database"
	"github.com/obiente/cloud/apps/shared/pkg/docker"
	"github.com/obiente/cloud/apps/shared/pkg/logger"
)

// DockerProvisioner handles database provisioning using Docker
type DockerProvisioner struct {
	client *docker.Client
}

// DatabaseConfig contains configuration for a new database
type DatabaseConfig struct {
	DatabaseID  string            // Unique database ID
	Type        DatabaseType      // PostgreSQL, MySQL, MongoDB, etc.
	Version     string            // Database engine version
	Username    string            // Admin username
	Password    string            // Admin password
	Port        int               // Internal port
	CPUCores    float64           // CPU cores
	MemoryBytes int64             // Memory in bytes
	DiskBytes   int64             // Disk space in bytes
	Metadata    map[string]string // Custom metadata/labels
}

// ProvisioningResult contains result information
type ProvisioningResult struct {
	ContainerID string // Docker container ID
	Host        string // Internal hostname
	Port        int    // Exposed port
	Status      string // Current status
}

// DatabaseType enum
type DatabaseType int32

const (
	DatabaseType_UNSPECIFIED DatabaseType = 0
	DatabaseType_POSTGRESQL  DatabaseType = 1
	DatabaseType_MYSQL       DatabaseType = 2
	DatabaseType_MONGODB     DatabaseType = 3
	DatabaseType_REDIS       DatabaseType = 4
	DatabaseType_MARIADB     DatabaseType = 5
)

func (dt DatabaseType) String() string {
	switch dt {
	case DatabaseType_POSTGRESQL:
		return "postgresql"
	case DatabaseType_MYSQL:
		return "mysql"
	case DatabaseType_MONGODB:
		return "mongodb"
	case DatabaseType_REDIS:
		return "redis"
	case DatabaseType_MARIADB:
		return "mariadb"
	default:
		return "unknown"
	}
}

// NewDockerProvisioner creates a new Docker provisioner
func NewDockerProvisioner() (*DockerProvisioner, error) {
	dcli, err := docker.New()
	if err != nil {
		return nil, fmt.Errorf("failed to create docker client: %w", err)
	}

	return &DockerProvisioner{
		client: dcli,
	}, nil
}

// ProvisionDatabase creates and starts a new database container
func (p *DockerProvisioner) ProvisionDatabase(ctx context.Context, cfg *DatabaseConfig) (*ProvisioningResult, error) {
	logger.Info("Provisioning database: %s (type: %s)", cfg.DatabaseID, cfg.Type.String())

	// Get image and config for database type
	image := p.getImageForType(cfg.Type, cfg.Version)
	containerName := fmt.Sprintf("obiente-%s", cfg.DatabaseID)

	hostname := database.DefaultMyObienteCloudDomain(cfg.DatabaseID)

	// Get standard port for database type
	var standardPort int
	var entrypoint string
	switch cfg.Type {
	case DatabaseType_POSTGRESQL:
		standardPort = 5432
		entrypoint = "postgres"
	case DatabaseType_MYSQL, DatabaseType_MARIADB:
		standardPort = 3306
		entrypoint = "mysql"
	case DatabaseType_MONGODB:
		standardPort = 27017
		entrypoint = "mongodb"
	case DatabaseType_REDIS:
		standardPort = 6379
		entrypoint = "redis"
	default:
		standardPort = cfg.Port
		entrypoint = "database"
	}

	// Container labels for identification (proxy handles routing, not Traefik)
	labels := map[string]string{
		"cloud.obiente.service":     "database",
		"cloud.obiente.database_id": cfg.DatabaseID,
	}

	_ = entrypoint // No longer used for Traefik routing

	// Build environment variables for database initialization
	env := []string{}

	switch cfg.Type {
	case DatabaseType_POSTGRESQL:
		env = append(env,
			fmt.Sprintf("POSTGRES_USER=%s", cfg.Username),
			fmt.Sprintf("POSTGRES_PASSWORD=%s", cfg.Password),
			fmt.Sprintf("POSTGRES_DB=%s", cfg.DatabaseID),
		)
	case DatabaseType_MYSQL, DatabaseType_MARIADB:
		env = append(env,
			fmt.Sprintf("MYSQL_USER=%s", cfg.Username),
			fmt.Sprintf("MYSQL_PASSWORD=%s", cfg.Password),
			fmt.Sprintf("MYSQL_DATABASE=%s", cfg.DatabaseID),
			fmt.Sprintf("MYSQL_ROOT_PASSWORD=%s", cfg.Password),
		)
	case DatabaseType_MONGODB:
		env = append(env,
			fmt.Sprintf("MONGO_INITDB_ROOT_USERNAME=%s", cfg.Username),
			fmt.Sprintf("MONGO_INITDB_ROOT_PASSWORD=%s", cfg.Password),
			fmt.Sprintf("MONGO_INITDB_DATABASE=%s", cfg.DatabaseID),
		)
	}

	// Create container using docker client
	containerID, err := p.client.CreateContainer(ctx, &docker.ContainerConfig{
		Name:   containerName,
		Image:  image,
		Env:    env,
		Labels: labels,
		// Don't expose ports - Traefik will handle routing
		// PortBindings: nil,
		Networks:      []string{"obiente-databases"},
		RestartPolicy: "unless-stopped",
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create database container: %w", err)
	}

	// Start the container
	if err := p.client.StartContainer(ctx, containerID); err != nil {
		return nil, fmt.Errorf("failed to start database container: %w", err)
	}

	logger.Info("Database container provisioned: %s (ID: %s)", containerName, containerID)

	return &ProvisioningResult{
		ContainerID: containerID,
		Host:        hostname,
		Port:        standardPort,
		Status:      "running",
	}, nil
}

// DeprovisionDatabase stops and removes a database container
func (p *DockerProvisioner) DeprovisionDatabase(ctx context.Context, containerID string) error {
	logger.Info("Deprovisioning database container: %s", containerID)

	// Stop container with timeout
	if err := p.client.StopContainer(ctx, containerID, 30*time.Second); err != nil {
		logger.Warn("Failed to stop container: %v", err)
		// Continue anyway
	}

	// Remove container
	if err := p.client.RemoveContainer(ctx, containerID, true); err != nil {
		return fmt.Errorf("failed to remove container: %w", err)
	}

	logger.Info("Database container removed: %s", containerID)
	return nil
}

// StartDatabase starts a stopped database container
func (p *DockerProvisioner) StartDatabase(ctx context.Context, containerID string) error {
	logger.Info("Starting database container: %s", containerID)
	return p.client.StartContainer(ctx, containerID)
}

// StopDatabase stops a running database container
func (p *DockerProvisioner) StopDatabase(ctx context.Context, containerID string) error {
	logger.Info("Stopping database container: %s", containerID)
	return p.client.StopContainer(ctx, containerID, 30*time.Second)
}

// RestartDatabase restarts a database container
func (p *DockerProvisioner) RestartDatabase(ctx context.Context, containerID string) error {
	logger.Info("Restarting database container: %s", containerID)
	return p.client.RestartContainer(ctx, containerID, 30*time.Second)
}

// getImageForType returns the Docker image for a database type
func (p *DockerProvisioner) getImageForType(dbType DatabaseType, version string) string {
	if version == "" {
		version = "latest"
	}

	switch dbType {
	case DatabaseType_POSTGRESQL:
		return fmt.Sprintf("postgres:%s", version)
	case DatabaseType_MYSQL:
		return fmt.Sprintf("mysql:%s", version)
	case DatabaseType_MARIADB:
		return fmt.Sprintf("mariadb:%s", version)
	case DatabaseType_MONGODB:
		return fmt.Sprintf("mongo:%s", version)
	case DatabaseType_REDIS:
		return fmt.Sprintf("redis:%s", version)
	default:
		return "postgres:latest"
	}
}

// ReconcileContainer ensures an existing database container matches current config.
// This handles containers created before config changes (e.g. network migration).
func (p *DockerProvisioner) ReconcileContainer(ctx context.Context, containerID string, cfg *DatabaseConfig) error {
	inspect, err := p.client.ContainerInspect(ctx, containerID)
	if err != nil {
		return fmt.Errorf("failed to inspect container %s: %w", containerID, err)
	}

	desiredNetwork := "obiente-databases"
	desiredLabels := map[string]string{
		"cloud.obiente.service":     "database",
		"cloud.obiente.database_id": cfg.DatabaseID,
	}

	// Check network: ensure container is on obiente-databases
	if inspect.NetworkSettings != nil {
		if _, onNetwork := inspect.NetworkSettings.Networks[desiredNetwork]; !onNetwork {
			logger.Info("Reconcile: connecting container %s to network %s", containerID, desiredNetwork)
			if err := p.client.NetworkConnect(ctx, desiredNetwork, containerID); err != nil {
				logger.Warn("Reconcile: failed to connect %s to %s: %v", containerID, desiredNetwork, err)
			}
		}
	}

	// Check labels: log mismatches (labels can't be updated without recreating)
	if inspect.Config != nil {
		for k, desired := range desiredLabels {
			actual, exists := inspect.Config.Labels[k]
			if !exists || actual != desired {
				logger.Debug("Reconcile: container %s has stale label %s=%q (want %q) — will apply on next recreate", containerID, k, actual, desired)
			}
		}

		// Warn about old Traefik labels that should be removed
		for k := range inspect.Config.Labels {
			if len(k) > 8 && k[:8] == "traefik." {
				logger.Debug("Reconcile: container %s has obsolete Traefik label %s — will be removed on next recreate", containerID, k)
			}
		}
	}

	return nil
}

// Close closes the Docker client
func (p *DockerProvisioner) Close() error {
	if p.client != nil {
		return p.client.Close()
	}
	return nil
}
