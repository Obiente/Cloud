package deployments

import (
	"context"
	"fmt"
	"log"

	"github.com/obiente/cloud/apps/shared/pkg/auth"

	deploymentsv1 "github.com/obiente/cloud/apps/shared/proto/obiente/cloud/deployments/v1"

	"connectrpc.com/connect"
)

// GetDeploymentCompose retrieves the Docker Compose configuration for a deployment
func (s *Service) GetDeploymentCompose(ctx context.Context, req *connect.Request[deploymentsv1.GetDeploymentComposeRequest]) (*connect.Response[deploymentsv1.GetDeploymentComposeResponse], error) {
	deploymentID := req.Msg.GetDeploymentId()
	orgID := req.Msg.GetOrganizationId()
	if err := s.permissionChecker.CheckScopedPermission(ctx, orgID, auth.ScopedPermission{Permission: "deployments.read", ResourceType: "deployment", ResourceID: deploymentID}); err != nil {
		return nil, connect.NewError(connect.CodePermissionDenied, err)
	}
	dbDep, err := s.repo.GetByID(ctx, deploymentID)
	if err != nil {
		return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("deployment %s not found", deploymentID))
	}
	return connect.NewResponse(&deploymentsv1.GetDeploymentComposeResponse{ComposeYaml: dbDep.ComposeYaml}), nil
}

// ValidateDeploymentCompose validates a Docker Compose configuration
func (s *Service) ValidateDeploymentCompose(ctx context.Context, req *connect.Request[deploymentsv1.ValidateDeploymentComposeRequest]) (*connect.Response[deploymentsv1.ValidateDeploymentComposeResponse], error) {
	deploymentID := req.Msg.GetDeploymentId()
	orgID := req.Msg.GetOrganizationId()
	if err := s.permissionChecker.CheckScopedPermission(ctx, orgID, auth.ScopedPermission{Permission: "deployments.update", ResourceType: "deployment", ResourceID: deploymentID}); err != nil {
		return nil, connect.NewError(connect.CodePermissionDenied, err)
	}

	composeYaml := req.Msg.GetComposeYaml()

	// Perform comprehensive validation using Docker Compose CLI (validation only, no save)
	validationErrors := ValidateCompose(ctx, composeYaml)

	// Convert validation errors to proto
	protoErrors := make([]*deploymentsv1.ComposeValidationError, 0, len(validationErrors))
	var firstErrorMsg string
	for _, ve := range validationErrors {
		protoErrors = append(protoErrors, &deploymentsv1.ComposeValidationError{
			Line:        ve.Line,
			Column:      ve.Column,
			Message:     ve.Message,
			Severity:    ve.Severity,
			StartLine:   ve.StartLine,
			EndLine:     ve.EndLine,
			StartColumn: ve.StartColumn,
			EndColumn:   ve.EndColumn,
		})
		if ve.Severity == "error" && firstErrorMsg == "" {
			firstErrorMsg = ve.Message
		}
	}

	res := connect.NewResponse(&deploymentsv1.ValidateDeploymentComposeResponse{
		ValidationErrors: protoErrors,
	})

	// Set legacy validation_error for backward compatibility
	if firstErrorMsg != "" {
		res.Msg.ValidationError = &firstErrorMsg
	}

	return res, nil
}

// UpdateDeploymentCompose updates the Docker Compose configuration for a deployment
func (s *Service) UpdateDeploymentCompose(ctx context.Context, req *connect.Request[deploymentsv1.UpdateDeploymentComposeRequest]) (*connect.Response[deploymentsv1.UpdateDeploymentComposeResponse], error) {
	deploymentID := req.Msg.GetDeploymentId()
	orgID := req.Msg.GetOrganizationId()
	if err := s.permissionChecker.CheckScopedPermission(ctx, orgID, auth.ScopedPermission{Permission: "deployments.update", ResourceType: "deployment", ResourceID: deploymentID}); err != nil {
		return nil, connect.NewError(connect.CodePermissionDenied, err)
	}
	dbDep, err := s.repo.GetByID(ctx, deploymentID)
	if err != nil {
		return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("deployment %s not found", deploymentID))
	}

	composeYaml := req.Msg.GetComposeYaml()

	// Perform comprehensive validation using Docker Compose CLI
	validationErrors := ValidateCompose(ctx, composeYaml)

	// Check if there are any errors (severity == "error")
	hasErrors := false
	var firstErrorMsg string
	for _, ve := range validationErrors {
		if ve.Severity == "error" {
			hasErrors = true
			if firstErrorMsg == "" {
				firstErrorMsg = ve.Message
			}
			break
		}
	}

	// Only save if there are no errors (warnings are OK)
	if !hasErrors {
		dbDep.ComposeYaml = composeYaml
		if err := s.repo.Update(ctx, dbDep); err != nil {
			return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to update compose: %w", err))
		}

		// If deployment is currently running, redeploy with new compose file
		if dbDep.Status == int32(deploymentsv1.DeploymentStatus_RUNNING) && s.manager != nil {
			log.Printf("[UpdateDeploymentCompose] Redeploying running deployment %s with updated compose file", deploymentID)
			// Stop existing deployment first
			_ = s.manager.StopComposeDeployment(ctx, deploymentID)
			_ = s.manager.RemoveComposeDeployment(ctx, deploymentID)
			// Deploy new compose file
			if err := s.manager.DeployComposeFile(ctx, deploymentID, composeYaml); err != nil {
				log.Printf("[UpdateDeploymentCompose] Failed to redeploy compose file for deployment %s: %v", deploymentID, err)
				// Continue anyway - compose file is saved, user can manually redeploy
			} else {
				log.Printf("[UpdateDeploymentCompose] Successfully redeployed compose file for deployment %s", deploymentID)
			}
		}
	}

	// Reload to get updated state
	dbDep, _ = s.repo.GetByID(ctx, deploymentID)

	// Convert validation errors to proto
	protoErrors := make([]*deploymentsv1.ComposeValidationError, 0, len(validationErrors))
	for _, ve := range validationErrors {
		protoErrors = append(protoErrors, &deploymentsv1.ComposeValidationError{
			Line:        ve.Line,
			Column:      ve.Column,
			Message:     ve.Message,
			Severity:    ve.Severity,
			StartLine:   ve.StartLine,
			EndLine:     ve.EndLine,
			StartColumn: ve.StartColumn,
			EndColumn:   ve.EndColumn,
		})
	}

	res := connect.NewResponse(&deploymentsv1.UpdateDeploymentComposeResponse{
		Deployment:       dbDeploymentToProto(dbDep),
		ValidationErrors: protoErrors,
	})

	// Set legacy validation_error for backward compatibility
	if firstErrorMsg != "" {
		res.Msg.ValidationError = &firstErrorMsg
	}

	return res, nil
}

