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
		BuildStrategy:  deploymentsv1.BuildStrategy(db.BuildStrategy),
		Branch:         db.Branch,
		Status:         deploymentsv1.DeploymentStatus(db.Status),
		HealthStatus:   db.HealthStatus,
		Environment:    deploymentsv1.Environment(db.Environment),
		BandwidthUsage: db.BandwidthUsage,
		StorageUsage:   db.StorageBytes,
		BuildTime:      db.BuildTime,
		Size:           db.Size,
	}

	// Parse custom domains from JSON
	if db.CustomDomains != "" {
		var customDomains []string
		if err := json.Unmarshal([]byte(db.CustomDomains), &customDomains); err == nil {
			deployment.CustomDomains = customDomains
		}
	}

	// Parse groups from JSON
	if db.Groups != "" {
		var groups []string
		if err := json.Unmarshal([]byte(db.Groups), &groups); err == nil {
			deployment.Groups = groups
		}
	}

	if db.RepositoryURL != nil {
		deployment.RepositoryUrl = proto.String(*db.RepositoryURL)
	}
	if db.GitHubIntegrationID != nil {
		deployment.GithubIntegrationId = proto.String(*db.GitHubIntegrationID)
	}
	if db.BuildCommand != nil {
		deployment.BuildCommand = proto.String(*db.BuildCommand)
	}
	if db.InstallCommand != nil {
		deployment.InstallCommand = proto.String(*db.InstallCommand)
	}
	if db.StartCommand != nil {
		deployment.StartCommand = proto.String(*db.StartCommand)
	}
	if db.DockerfilePath != nil {
		deployment.DockerfilePath = proto.String(*db.DockerfilePath)
	}
	if db.ComposeFilePath != nil {
		deployment.ComposeFilePath = proto.String(*db.ComposeFilePath)
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
		BuildStrategy:  int32(protoDep.GetBuildStrategy()),
		Branch:         protoDep.GetBranch(),
		Status:         int32(protoDep.GetStatus()),
		HealthStatus:   protoDep.GetHealthStatus(),
		Environment:    int32(protoDep.GetEnvironment()),
		BandwidthUsage: protoDep.GetBandwidthUsage(),
		StorageBytes:   protoDep.GetStorageUsage(),
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
	if protoDep.GithubIntegrationId != nil {
		integrationID := protoDep.GetGithubIntegrationId()
		db.GitHubIntegrationID = &integrationID
	}
	if protoDep.BuildCommand != nil {
		buildCmd := protoDep.GetBuildCommand()
		db.BuildCommand = &buildCmd
	}
	if protoDep.InstallCommand != nil {
		installCmd := protoDep.GetInstallCommand()
		db.InstallCommand = &installCmd
	}
	if protoDep.StartCommand != nil {
		startCmd := protoDep.GetStartCommand()
		db.StartCommand = &startCmd
	}
	if protoDep.DockerfilePath != nil {
		dockerfilePath := protoDep.GetDockerfilePath()
		db.DockerfilePath = &dockerfilePath
	}
	if protoDep.ComposeFilePath != nil {
		composeFilePath := protoDep.GetComposeFilePath()
		db.ComposeFilePath = &composeFilePath
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

	// Custom domains stored as JSON string
	if len(protoDep.GetCustomDomains()) > 0 {
		customDomainsJSON, _ := json.Marshal(protoDep.GetCustomDomains())
		db.CustomDomains = string(customDomainsJSON)
	} else {
		db.CustomDomains = "[]"
	}

	// Groups stored as JSON array
	if len(protoDep.GetGroups()) > 0 {
		groupsJSON, _ := json.Marshal(protoDep.GetGroups())
		db.Groups = string(groupsJSON)
	} else {
		db.Groups = "[]"
	}

	// Env vars stored as JSON object
	if len(protoDep.GetEnvVars()) > 0 {
		envJSON, _ := json.Marshal(protoDep.GetEnvVars())
		db.EnvVars = string(envJSON)
	} else {
		db.EnvVars = "{}"
	}

	return db
}
