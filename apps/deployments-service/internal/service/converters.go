package deployments

import (
	"encoding/json"
	"strings"

	"github.com/obiente/cloud/apps/shared/pkg/database"

	deploymentsv1 "github.com/obiente/cloud/apps/shared/proto/obiente/cloud/deployments/v1"

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
	if db.AutoDeploy != nil {
		deployment.AutoDeploy = proto.Bool(*db.AutoDeploy)
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
	if db.BuildPath != nil {
		deployment.BuildPath = proto.String(*db.BuildPath)
	}
	if db.BuildOutputPath != nil {
		deployment.BuildOutputPath = proto.String(*db.BuildOutputPath)
	}
	if db.UseNginx != nil {
		deployment.UseNginx = proto.Bool(*db.UseNginx)
	}
	if db.NginxConfig != nil {
		deployment.NginxConfig = proto.String(*db.NginxConfig)
	}

	// Health check configuration
	if db.HealthcheckType != nil {
		hcType := deploymentsv1.HealthCheckType(*db.HealthcheckType)
		deployment.HealthcheckType = &hcType
	}
	if db.HealthcheckPort != nil {
		deployment.HealthcheckPort = proto.Int32(*db.HealthcheckPort)
	}
	if db.HealthcheckPath != nil {
		deployment.HealthcheckPath = proto.String(*db.HealthcheckPath)
	}
	if db.HealthcheckExpectedStatus != nil {
		deployment.HealthcheckExpectedStatus = proto.Int32(*db.HealthcheckExpectedStatus)
	}
	if db.HealthcheckCustomCommand != nil {
		deployment.HealthcheckCustomCommand = proto.String(*db.HealthcheckCustomCommand)
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

	// Per-deployment resource limits (stored in DB as memory_bytes + cpu_shares)
	if db.CPUShares != nil && *db.CPUShares > 0 {
		deployment.CpuLimit = proto.Float64(float64(*db.CPUShares) / 1024.0)
	}
	if db.MemoryBytes != nil && *db.MemoryBytes > 0 {
		deployment.MemoryLimit = proto.Int64(*db.MemoryBytes / (1024 * 1024))
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
	deployment.BuildArgs = parseJSONStringMap(db.BuildArgs)
	deployment.DockerfileVolumes = parseDockerfileVolumesToProto(db.DockerfileVolumes)
	deployment.DockerfileBuildOptions = parseDockerfileBuildOptionsToProto(db.DockerfileBuildOptions)

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
	if protoDep.AutoDeploy != nil {
		autoDeploy := protoDep.GetAutoDeploy()
		db.AutoDeploy = &autoDeploy
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
	if protoDep.BuildPath != nil {
		buildPath := protoDep.GetBuildPath()
		db.BuildPath = &buildPath
	}
	if protoDep.BuildOutputPath != nil {
		buildOutputPath := protoDep.GetBuildOutputPath()
		db.BuildOutputPath = &buildOutputPath
	}
	if protoDep.UseNginx != nil {
		useNginx := protoDep.GetUseNginx()
		db.UseNginx = &useNginx
	}
	if protoDep.NginxConfig != nil {
		nginxConfig := protoDep.GetNginxConfig()
		db.NginxConfig = &nginxConfig
	}

	// Health check configuration
	if protoDep.HealthcheckType != nil {
		hcType := int32(protoDep.GetHealthcheckType())
		db.HealthcheckType = &hcType
	}
	if protoDep.HealthcheckPort != nil {
		hcPort := protoDep.GetHealthcheckPort()
		db.HealthcheckPort = &hcPort
	}
	if protoDep.HealthcheckPath != nil {
		hcPath := strings.TrimSpace(protoDep.GetHealthcheckPath())
		if hcPath != "" {
			db.HealthcheckPath = &hcPath
		}
	}
	if protoDep.HealthcheckExpectedStatus != nil {
		hcStatus := protoDep.GetHealthcheckExpectedStatus()
		db.HealthcheckExpectedStatus = &hcStatus
	}
	if protoDep.HealthcheckCustomCommand != nil {
		hcCmd := strings.TrimSpace(protoDep.GetHealthcheckCustomCommand())
		if hcCmd != "" {
			db.HealthcheckCustomCommand = &hcCmd
		}
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
	if len(protoDep.GetBuildArgs()) > 0 {
		buildArgsJSON, _ := json.Marshal(protoDep.GetBuildArgs())
		db.BuildArgs = string(buildArgsJSON)
	} else {
		db.BuildArgs = "{}"
	}
	db.DockerfileVolumes = marshalDockerfileVolumes(protoDep.GetDockerfileVolumes())
	db.DockerfileBuildOptions = marshalDockerfileBuildOptions(protoDep.GetDockerfileBuildOptions())

	return db
}

func parseJSONStringMap(raw string) map[string]string {
	values := make(map[string]string)
	if strings.TrimSpace(raw) == "" {
		return values
	}
	_ = json.Unmarshal([]byte(raw), &values)
	return values
}

func parseDockerfileVolumesToProto(raw string) []*deploymentsv1.DockerfileVolume {
	if strings.TrimSpace(raw) == "" {
		return nil
	}
	var volumes []*deploymentsv1.DockerfileVolume
	if err := json.Unmarshal([]byte(raw), &volumes); err == nil {
		return volumes
	}
	var stored []struct {
		Name      string `json:"name"`
		MountPath string `json:"mount_path"`
		ReadOnly  bool   `json:"read_only"`
	}
	if err := json.Unmarshal([]byte(raw), &stored); err != nil {
		return nil
	}
	volumes = make([]*deploymentsv1.DockerfileVolume, 0, len(stored))
	for _, volume := range stored {
		volumes = append(volumes, &deploymentsv1.DockerfileVolume{
			Name:      volume.Name,
			MountPath: volume.MountPath,
			ReadOnly:  volume.ReadOnly,
		})
	}
	return volumes
}

func marshalDockerfileVolumes(volumes []*deploymentsv1.DockerfileVolume) string {
	if len(volumes) == 0 {
		return "[]"
	}
	data, err := json.Marshal(volumes)
	if err != nil {
		return "[]"
	}
	return string(data)
}

func parseDockerfileBuildOptionsToProto(raw string) *deploymentsv1.DockerfileBuildOptions {
	if strings.TrimSpace(raw) == "" {
		return nil
	}
	var options deploymentsv1.DockerfileBuildOptions
	if err := json.Unmarshal([]byte(raw), &options); err == nil {
		return &options
	}
	var stored DockerfileBuildOptions
	if err := json.Unmarshal([]byte(raw), &stored); err != nil {
		return nil
	}
	protoOptions := &deploymentsv1.DockerfileBuildOptions{
		NoCache: proto.Bool(stored.NoCache),
		Pull:    proto.Bool(stored.Pull),
		Labels:  stored.Labels,
	}
	if strings.TrimSpace(stored.Target) != "" {
		protoOptions.Target = proto.String(stored.Target)
	}
	if strings.TrimSpace(stored.Platform) != "" {
		protoOptions.Platform = proto.String(stored.Platform)
	}
	return protoOptions
}

func marshalDockerfileBuildOptions(options *deploymentsv1.DockerfileBuildOptions) string {
	if options == nil {
		return "{}"
	}
	data, err := json.Marshal(options)
	if err != nil {
		return "{}"
	}
	return string(data)
}
