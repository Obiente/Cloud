package deployments

import (
	"context"
	"fmt"
	"log"
	"time"

	deploymentsv1 "api/gen/proto/obiente/cloud/deployments/v1"
	"api/internal/auth"
	"api/internal/database"

	"connectrpc.com/connect"
)

// GetDeploymentRoutings retrieves routing rules for a deployment
func (s *Service) GetDeploymentRoutings(ctx context.Context, req *connect.Request[deploymentsv1.GetDeploymentRoutingsRequest]) (*connect.Response[deploymentsv1.GetDeploymentRoutingsResponse], error) {
	deploymentID := req.Msg.GetDeploymentId()
	orgID := req.Msg.GetOrganizationId()

	// Check permissions
	if err := s.permissionChecker.CheckScopedPermission(ctx, orgID, auth.ScopedPermission{Permission: "deployments.view", ResourceType: "deployment", ResourceID: deploymentID}); err != nil {
		return nil, connect.NewError(connect.CodePermissionDenied, err)
	}

	// Verify deployment exists and belongs to organization
	dbDeployment, err := s.repo.GetByID(ctx, deploymentID)
	if err != nil {
		return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("deployment %s not found", deploymentID))
	}
	if dbDeployment.OrganizationID != orgID {
		return nil, connect.NewError(connect.CodePermissionDenied, fmt.Errorf("deployment does not belong to organization"))
	}

	// Get all routing rules
	dbRoutings, err := database.GetDeploymentRoutings(deploymentID)
	if err != nil {
		// If no routing rules exist, return empty list (not an error)
		return connect.NewResponse(&deploymentsv1.GetDeploymentRoutingsResponse{Rules: []*deploymentsv1.RoutingRule{}}), nil
	}

	// Convert to proto
	rules := make([]*deploymentsv1.RoutingRule, 0, len(dbRoutings))
	for _, dbRouting := range dbRoutings {
		rules = append(rules, &deploymentsv1.RoutingRule{
			Id:              dbRouting.ID,
			DeploymentId:    dbRouting.DeploymentID,
			Domain:          dbRouting.Domain,
			ServiceName:     dbRouting.ServiceName,
			PathPrefix:      dbRouting.PathPrefix,
			TargetPort:      int32(dbRouting.TargetPort),
			Protocol:        dbRouting.Protocol,
			SslEnabled:      dbRouting.SSLEnabled,
			SslCertResolver: dbRouting.SSLCertResolver,
		})
	}

	return connect.NewResponse(&deploymentsv1.GetDeploymentRoutingsResponse{Rules: rules}), nil
}

// UpdateDeploymentRoutings updates routing rules for a deployment
func (s *Service) UpdateDeploymentRoutings(ctx context.Context, req *connect.Request[deploymentsv1.UpdateDeploymentRoutingsRequest]) (*connect.Response[deploymentsv1.UpdateDeploymentRoutingsResponse], error) {
	deploymentID := req.Msg.GetDeploymentId()
	orgID := req.Msg.GetOrganizationId()

	// Check permissions
	if err := s.permissionChecker.CheckScopedPermission(ctx, orgID, auth.ScopedPermission{Permission: "deployments.update", ResourceType: "deployment", ResourceID: deploymentID}); err != nil {
		return nil, connect.NewError(connect.CodePermissionDenied, err)
	}

	// Verify deployment exists and belongs to organization
	dbDeployment, err := s.repo.GetByID(ctx, deploymentID)
	if err != nil {
		return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("deployment %s not found", deploymentID))
	}
	if dbDeployment.OrganizationID != orgID {
		return nil, connect.NewError(connect.CodePermissionDenied, fmt.Errorf("deployment does not belong to organization"))
	}

	// Delete all existing routing rules for this deployment
	existingRoutings, _ := database.GetDeploymentRoutings(deploymentID)
	for _, routing := range existingRoutings {
		if err := database.DB.Delete(&routing).Error; err != nil {
			log.Printf("[UpdateDeploymentRoutings] Warning: Failed to delete existing routing %s: %v", routing.ID, err)
		}
	}

	// Create new routing rules
	newRules := make([]*deploymentsv1.RoutingRule, 0, len(req.Msg.GetRules()))
	for _, rule := range req.Msg.GetRules() {
		// Generate ID if not provided
		ruleID := rule.GetId()
		if ruleID == "" {
			ruleID = fmt.Sprintf("route-%s-%s-%s-%d", deploymentID, rule.GetDomain(), rule.GetServiceName(), rule.GetTargetPort())
		}

		// Set defaults
		serviceName := rule.GetServiceName()
		if serviceName == "" {
			serviceName = "default"
		}
		protocol := rule.GetProtocol()
		if protocol == "" {
			protocol = "http"
		}

		dbRouting := &database.DeploymentRouting{
			ID:              ruleID,
			DeploymentID:    deploymentID,
			Domain:          rule.GetDomain(),
			ServiceName:     serviceName,
			PathPrefix:      rule.GetPathPrefix(),
			TargetPort:      int(rule.GetTargetPort()),
			Protocol:        protocol,
			SSLEnabled:      rule.GetSslEnabled(),
			SSLCertResolver: rule.GetSslCertResolver(),
			Middleware:      "{}",
			CreatedAt:       time.Now(),
			UpdatedAt:       time.Now(),
		}

		if err := database.UpsertDeploymentRouting(dbRouting); err != nil {
			log.Printf("[UpdateDeploymentRoutings] Warning: Failed to create routing rule for %s: %v", rule.GetDomain(), err)
			continue
		}

		// Convert back to proto for response
		newRules = append(newRules, &deploymentsv1.RoutingRule{
			Id:              dbRouting.ID,
			DeploymentId:    dbRouting.DeploymentID,
			Domain:          dbRouting.Domain,
			ServiceName:     dbRouting.ServiceName,
			PathPrefix:      dbRouting.PathPrefix,
			TargetPort:      int32(dbRouting.TargetPort),
			Protocol:        dbRouting.Protocol,
			SslEnabled:      dbRouting.SSLEnabled,
			SslCertResolver: dbRouting.SSLCertResolver,
		})
	}

	return connect.NewResponse(&deploymentsv1.UpdateDeploymentRoutingsResponse{Rules: newRules}), nil
}

// GetDeploymentServiceNames extracts service names from a deployment's Docker Compose file
func (s *Service) GetDeploymentServiceNames(ctx context.Context, req *connect.Request[deploymentsv1.GetDeploymentServiceNamesRequest]) (*connect.Response[deploymentsv1.GetDeploymentServiceNamesResponse], error) {
	deploymentID := req.Msg.GetDeploymentId()
	orgID := req.Msg.GetOrganizationId()

	// Check permissions
	if err := s.permissionChecker.CheckScopedPermission(ctx, orgID, auth.ScopedPermission{Permission: "deployments.view", ResourceType: "deployment", ResourceID: deploymentID}); err != nil {
		return nil, connect.NewError(connect.CodePermissionDenied, err)
	}

	// Verify deployment exists and belongs to organization
	dbDeployment, err := s.repo.GetByID(ctx, deploymentID)
	if err != nil {
		return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("deployment %s not found", deploymentID))
	}
	if dbDeployment.OrganizationID != orgID {
		return nil, connect.NewError(connect.CodePermissionDenied, fmt.Errorf("deployment does not belong to organization"))
	}

	// Extract service names from Docker Compose
	serviceNames, err := ExtractServiceNames(dbDeployment.ComposeYaml)
	if err != nil {
		log.Printf("[GetDeploymentServiceNames] Warning: Failed to parse compose for deployment %s: %v", deploymentID, err)
		// Return default service name on error
		serviceNames = []string{"default"}
	}

	return connect.NewResponse(&deploymentsv1.GetDeploymentServiceNamesResponse{
		ServiceNames: serviceNames,
	}), nil
}

