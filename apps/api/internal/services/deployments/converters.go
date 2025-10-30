package deployments

import (
	"encoding/json"

	deploymentsv1 "api/gen/proto/obiente/cloud/deployments/v1"
	"api/internal/database"

	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// dbDeploymentToProto converts a database Deployment to a proto Deployment
func dbDeploymentToProto(db *database.Deployment) *deploymentsv1.Deployment {
	if db == nil {
		return nil
	}

	deployment := &deploymentsv1.Deployment{
		Id:             db.ID,
		Name:           db.Name,
		Domain:         db.Domain,
		Type:           deploymentsv1.DeploymentType(db.Type),
		Branch:         db.Branch,
		Status:         deploymentsv1.DeploymentStatus(db.Status),
		HealthStatus:   db.HealthStatus,
		Environment:    deploymentsv1.Environment(db.Environment),
		BandwidthUsage: db.BandwidthUsage,
		StorageUsage:   db.StorageUsage,
		BuildTime:      db.BuildTime,
		Size:           db.Size,
	}

	// Parse custom domains from JSON (omitted for brevity)
	if db.CustomDomains != "" {
		deployment.CustomDomains = []string{}
	}

	if db.RepositoryURL != nil {
		deployment.RepositoryUrl = proto.String(*db.RepositoryURL)
	}
	if db.BuildCommand != nil {
		deployment.BuildCommand = proto.String(*db.BuildCommand)
	}
	if db.InstallCommand != nil {
		deployment.InstallCommand = proto.String(*db.InstallCommand)
	}

	// Runtime fields
	if db.Image != nil {
		deployment.Image = proto.String(*db.Image)
	}
	if db.Port != nil {
		deployment.Port = proto.Int32(*db.Port)
	}
	if db.Replicas != nil {
		deployment.Replicas = proto.Int32(*db.Replicas)
	}

	// Parse env vars from JSON
	if db.EnvVars != "" {
		var envMap map[string]string
		if err := json.Unmarshal([]byte(db.EnvVars), &envMap); err == nil {
			deployment.EnvVars = envMap
		} else {
			deployment.EnvVars = make(map[string]string)
		}
	} else {
		deployment.EnvVars = make(map[string]string)
	}

	// Convert timestamps
	if !db.LastDeployedAt.IsZero() {
		deployment.LastDeployedAt = timestamppb.New(db.LastDeployedAt)
	}
	if !db.CreatedAt.IsZero() {
		deployment.CreatedAt = timestamppb.New(db.CreatedAt)
	}

	return deployment
}

// protoToDBDeployment converts a proto Deployment to a database Deployment
func protoToDBDeployment(protoDep *deploymentsv1.Deployment, orgID string, createdBy string) *database.Deployment {
	if protoDep == nil {
		return nil
	}

	db := &database.Deployment{
		ID:             protoDep.GetId(),
		Name:           protoDep.GetName(),
		Domain:         protoDep.GetDomain(),
		Type:           int32(protoDep.GetType()),
		Branch:         protoDep.GetBranch(),
		Status:         int32(protoDep.GetStatus()),
		HealthStatus:   protoDep.GetHealthStatus(),
		Environment:    int32(protoDep.GetEnvironment()),
		BandwidthUsage: protoDep.GetBandwidthUsage(),
		StorageUsage:   protoDep.GetStorageUsage(),
		BuildTime:      protoDep.GetBuildTime(),
		Size:           protoDep.GetSize(),
		OrganizationID: orgID,
		CreatedBy:      createdBy,
	}

	// Handle optional fields
	if protoDep.RepositoryUrl != nil {
		repoURL := protoDep.GetRepositoryUrl()
		db.RepositoryURL = &repoURL
	}
	if protoDep.BuildCommand != nil {
		buildCmd := protoDep.GetBuildCommand()
		db.BuildCommand = &buildCmd
	}
	if protoDep.InstallCommand != nil {
		installCmd := protoDep.GetInstallCommand()
		db.InstallCommand = &installCmd
	}
	if protoDep.Image != nil {
		img := protoDep.GetImage()
		db.Image = &img
	}
	if protoDep.Port != nil {
		p := protoDep.GetPort()
		db.Port = &p
	}
	if protoDep.Replicas != nil {
		r := protoDep.GetReplicas()
		db.Replicas = &r
	}

	// Handle timestamps
	if protoDep.LastDeployedAt != nil {
		db.LastDeployedAt = protoDep.LastDeployedAt.AsTime()
	}
	if protoDep.CreatedAt != nil {
		db.CreatedAt = protoDep.CreatedAt.AsTime()
	}

	// Custom domains stored as JSON string (keep empty for now)
	db.CustomDomains = "[]"

	// Env vars stored as JSON object
	if len(protoDep.GetEnvVars()) > 0 {
		envJSON, _ := json.Marshal(protoDep.GetEnvVars())
		db.EnvVars = string(envJSON)
	} else {
		db.EnvVars = "{}"
	}

	return db
}
