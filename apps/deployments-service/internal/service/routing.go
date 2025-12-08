package deployments

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/obiente/cloud/apps/shared/pkg/auth"
	"github.com/obiente/cloud/apps/shared/pkg/database"

	deploymentsv1 "github.com/obiente/cloud/apps/shared/proto/obiente/cloud/deployments/v1"

	"connectrpc.com/connect"
)

// GetDeploymentRoutings retrieves routing rules for a deployment
func (s *Service) GetDeploymentRoutings(ctx context.Context, req *connect.Request[deploymentsv1.GetDeploymentRoutingsRequest]) (*connect.Response[deploymentsv1.GetDeploymentRoutingsResponse], error) {
	deploymentID := req.Msg.GetDeploymentId()
	orgID := req.Msg.GetOrganizationId()

	// Check permissions
	if err := s.permissionChecker.CheckScopedPermission(ctx, orgID, auth.ScopedPermission{Permission: "deployments.read", ResourceType: "deployment", ResourceID: deploymentID}); err != nil {
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

	// Get available domains for this deployment (default + verified custom domains)
	availableDomains := s.getAvailableDomainsForDeployment(dbDeployment)
	
	// Create new routing rules
	newRules := make([]*deploymentsv1.RoutingRule, 0, len(req.Msg.GetRules()))
	for _, rule := range req.Msg.GetRules() {
		// Validate domain ownership
		ruleDomain := rule.GetDomain()
		if ruleDomain != "" {
			domainAllowed := false
			for _, allowedDomain := range availableDomains {
				if allowedDomain == ruleDomain {
					domainAllowed = true
					break
				}
			}
			
			if !domainAllowed {
				return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("domain %s is not available for this deployment. You can only use the default domain or verified custom domains", ruleDomain))
			}
		}

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

		// Ensure SSL is disabled for HTTP protocol, regardless of what the client sends
		sslEnabled := rule.GetSslEnabled()
		if protocol == "http" {
			sslEnabled = false
		} else if protocol == "https" {
			sslEnabled = true
		}
		
		var dbRouting *database.DeploymentRouting
		
		// Check if routing already exists
		var existingRouting database.DeploymentRouting
		err := database.DB.Where("id = ?", ruleID).First(&existingRouting).Error
		
		if err == nil {
			// Update existing routing - explicitly update all fields including SSLEnabled
			// Using Updates with map to ensure boolean false values are properly saved
			updateData := map[string]interface{}{
				"domain":           rule.GetDomain(),
				"service_name":     serviceName,
				"path_prefix":      rule.GetPathPrefix(),
				"target_port":      int(rule.GetTargetPort()),
				"protocol":         protocol,
				"ssl_enabled":      sslEnabled, // Explicitly set to ensure false values are saved
				"ssl_cert_resolver": rule.GetSslCertResolver(),
				"updated_at":       time.Now(),
			}
			
			if updateErr := database.DB.Model(&existingRouting).Updates(updateData).Error; updateErr != nil {
				log.Printf("[UpdateDeploymentRoutings] Warning: Failed to update routing rule for %s: %v", rule.GetDomain(), updateErr)
				continue
			}
			// Read updated record back
			if readErr := database.DB.Where("id = ?", ruleID).First(&existingRouting).Error; readErr != nil {
				log.Printf("[UpdateDeploymentRoutings] Warning: Failed to read updated routing: %v", readErr)
				continue
			}
			dbRouting = &existingRouting
		} else {
			// Create new routing
			newRouting := &database.DeploymentRouting{
				ID:              ruleID,
				DeploymentID:    deploymentID,
				Domain:          rule.GetDomain(),
				ServiceName:     serviceName,
				PathPrefix:      rule.GetPathPrefix(),
				TargetPort:      int(rule.GetTargetPort()),
				Protocol:        protocol,
				SSLEnabled:      sslEnabled,
				SSLCertResolver: rule.GetSslCertResolver(),
				Middleware:      "{}",
				CreatedAt:       time.Now(),
				UpdatedAt:       time.Now(),
			}

			if createErr := database.DB.Create(newRouting).Error; createErr != nil {
				log.Printf("[UpdateDeploymentRoutings] Warning: Failed to create routing rule for %s: %v", rule.GetDomain(), createErr)
				continue
			}
			dbRouting = newRouting
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
	if err := s.permissionChecker.CheckScopedPermission(ctx, orgID, auth.ScopedPermission{Permission: "deployments.read", ResourceType: "deployment", ResourceID: deploymentID}); err != nil {
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

// getAvailableDomainsForDeployment returns a list of domains that can be used for routing
// Includes: default domain + verified custom domains
func (s *Service) getAvailableDomainsForDeployment(dbDeployment *database.Deployment) []string {
	domains := []string{}
	
	// Add default domain
	if dbDeployment.Domain != "" {
		domains = append(domains, dbDeployment.Domain)
	}
	
	// Add verified custom domains
	if dbDeployment.CustomDomains != "" {
		var customDomains []string
		if err := json.Unmarshal([]byte(dbDeployment.CustomDomains), &customDomains); err == nil {
			for _, entry := range customDomains {
				parts := strings.Split(entry, ":")
				if len(parts) == 0 {
					continue
				}
				domain := parts[0]
				
				// Check if domain is verified
				isVerified := false
				if len(parts) >= 2 && parts[1] == "verified" {
					isVerified = true
				} else if len(parts) >= 4 && parts[1] == "token" && parts[3] == "verified" {
					isVerified = true
				}
				
				if isVerified {
					// Check for duplicates
					alreadyAdded := false
					for _, existingDomain := range domains {
						if existingDomain == domain {
							alreadyAdded = true
							break
						}
					}
					if !alreadyAdded {
						domains = append(domains, domain)
					}
				}
			}
		}
	}
	
	return domains
}

