package deployments

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"api/internal/auth"

	deploymentsv1 "github.com/obiente/cloud/apps/shared/proto/obiente/cloud/deployments/v1"

	"connectrpc.com/connect"
)

// GetDeploymentEnvVars retrieves environment variables for a deployment
func (s *Service) GetDeploymentEnvVars(ctx context.Context, req *connect.Request[deploymentsv1.GetDeploymentEnvVarsRequest]) (*connect.Response[deploymentsv1.GetDeploymentEnvVarsResponse], error) {
	deploymentID := req.Msg.GetDeploymentId()
	orgID := req.Msg.GetOrganizationId()
	if err := s.permissionChecker.CheckScopedPermission(ctx, orgID, auth.ScopedPermission{Permission: "deployments.view", ResourceType: "deployment", ResourceID: deploymentID}); err != nil {
		return nil, connect.NewError(connect.CodePermissionDenied, err)
	}
	dbDep, err := s.repo.GetByID(ctx, deploymentID)
	if err != nil {
		return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("deployment %s not found", deploymentID))
	}

	envFileContent := dbDep.EnvFileContent

	return connect.NewResponse(&deploymentsv1.GetDeploymentEnvVarsResponse{
		EnvFileContent: envFileContent,
	}), nil
}

// UpdateDeploymentEnvVars updates environment variables for a deployment
func (s *Service) UpdateDeploymentEnvVars(ctx context.Context, req *connect.Request[deploymentsv1.UpdateDeploymentEnvVarsRequest]) (*connect.Response[deploymentsv1.UpdateDeploymentEnvVarsResponse], error) {
	deploymentID := req.Msg.GetDeploymentId()
	orgID := req.Msg.GetOrganizationId()
	if err := s.permissionChecker.CheckScopedPermission(ctx, orgID, auth.ScopedPermission{Permission: "deployments.update", ResourceType: "deployment", ResourceID: deploymentID}); err != nil {
		return nil, connect.NewError(connect.CodePermissionDenied, err)
	}
	_, err := s.repo.GetByID(ctx, deploymentID)
	if err != nil {
		return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("deployment %s not found", deploymentID))
	}

	envFileContent := req.Msg.GetEnvFileContent()

	// Parse to generate env_vars map for backward compatibility with existing code
	envVarsMap := parseEnvFileToMap(envFileContent)

	// Marshal env vars to JSON
	envJSON, err := json.Marshal(envVarsMap)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to marshal env vars: %w", err))
	}

	// Update only the env vars fields using repository method
	if err := s.repo.UpdateEnvVars(ctx, deploymentID, envFileContent, string(envJSON)); err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to update env vars: %w", err))
	}

	// Reload to get updated state
	dbDep, _ := s.repo.GetByID(ctx, deploymentID)
	res := connect.NewResponse(&deploymentsv1.UpdateDeploymentEnvVarsResponse{Deployment: dbDeploymentToProto(dbDep)})
	return res, nil
}

// parseEnvVars parses environment variables from JSON string stored in database
func parseEnvVars(envVarsJSON string) map[string]string {
	if envVarsJSON == "" {
		return make(map[string]string)
	}
	var envMap map[string]string
	if err := json.Unmarshal([]byte(envVarsJSON), &envMap); err != nil {
		return make(map[string]string)
	}
	return envMap
}

// parseEnvFileToMap parses a .env file content and extracts key-value pairs (ignores comments)
// Used internally to maintain backward compatibility with EnvVars JSON field
func parseEnvFileToMap(envFileContent string) map[string]string {
	envMap := make(map[string]string)
	if envFileContent == "" {
		return envMap
	}

	lines := strings.Split(envFileContent, "\n")
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		// Skip empty lines and comments
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}

		equalIndex := strings.Index(trimmed, "=")
		if equalIndex == -1 {
			continue
		}

		key := strings.TrimSpace(trimmed[:equalIndex])
		value := strings.TrimSpace(trimmed[equalIndex+1:])

		// Remove quotes if present
		if len(value) >= 2 {
			if (value[0] == '"' && value[len(value)-1] == '"') || (value[0] == '\'' && value[len(value)-1] == '\'') {
				value = value[1 : len(value)-1]
			}
		}

		envMap[key] = value
	}

	return envMap
}

