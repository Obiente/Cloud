package deployments

import (
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

	// Parse custom domains from JSON
	if db.CustomDomains != "" {
		// For now, we'll skip parsing JSON and use empty array
		// In a real implementation, parse the JSON string
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
func protoToDBDeployment(proto *deploymentsv1.Deployment, orgID string, createdBy string) *database.Deployment {
	if proto == nil {
		return nil
	}

	db := &database.Deployment{
		ID:             proto.GetId(),
		Name:           proto.GetName(),
		Domain:         proto.GetDomain(),
		Type:           int32(proto.GetType()),
		Branch:         proto.GetBranch(),
		Status:         int32(proto.GetStatus()),
		HealthStatus:   proto.GetHealthStatus(),
		Environment:    int32(proto.GetEnvironment()),
		BandwidthUsage: proto.GetBandwidthUsage(),
		StorageUsage:   proto.GetStorageUsage(),
		BuildTime:      proto.GetBuildTime(),
		Size:           proto.GetSize(),
		OrganizationID: orgID,
		CreatedBy:      createdBy,
	}

	// Handle optional fields
	if proto.RepositoryUrl != nil {
		repoURL := proto.GetRepositoryUrl()
		db.RepositoryURL = &repoURL
	}
	if proto.BuildCommand != nil {
		buildCmd := proto.GetBuildCommand()
		db.BuildCommand = &buildCmd
	}
	if proto.InstallCommand != nil {
		installCmd := proto.GetInstallCommand()
		db.InstallCommand = &installCmd
	}

	// Handle timestamps
	if proto.LastDeployedAt != nil {
		db.LastDeployedAt = proto.LastDeployedAt.AsTime()
	}
	if proto.CreatedAt != nil {
		db.CreatedAt = proto.CreatedAt.AsTime()
	}

	// Custom domains - convert to JSON
	// For now, just store as empty
	db.CustomDomains = "[]"

	return db
}
