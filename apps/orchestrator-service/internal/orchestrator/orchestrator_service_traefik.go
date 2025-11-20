package orchestrator

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
	"time"

	"github.com/obiente/cloud/apps/shared/pkg/database"
	"github.com/obiente/cloud/apps/shared/pkg/logger"
)

// Traefik operations for orchestrator service

func (os *OrchestratorService) syncMicroserviceTraefikLabels() {
	// Run every 30 seconds
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	// Run immediately on startup
	os.updateMicroserviceTraefikLabels()

	for {
		select {
		case <-ticker.C:
			os.updateMicroserviceTraefikLabels()
		case <-os.ctx.Done():
			return
		}
	}
}

func (os *OrchestratorService) updateMicroserviceTraefikLabels() {
	// Get node configuration
	nodeID := os.deploymentManager.GetNodeID()
	if nodeID == "" {
		logger.Debug("[Orchestrator] No node ID available, skipping Traefik label sync")
		return
	}

	// Get node metadata to check for subdomain configuration
	// Try to find by node ID first, then fall back to hostname if not found
	var node database.NodeMetadata
	err := database.DB.Where("id = ?", nodeID).First(&node).Error
	if err != nil {
		// Node ID not found - try to find by hostname as fallback
		// This handles cases where the node ID in the database doesn't match Docker Swarm node ID
		nodeHostname := os.deploymentManager.GetNodeHostname()
		if nodeHostname != "" {
			logger.Debug("[Orchestrator] Node %s not found by ID, trying to find by hostname %s", nodeID, nodeHostname)
			err = database.DB.Where("hostname = ?", nodeHostname).First(&node).Error
			if err != nil {
				logger.Debug("[Orchestrator] Node not found by ID %s or hostname %s, skipping Traefik label sync: %v", nodeID, nodeHostname, err)
				return
			}
			logger.Info("[Orchestrator] Found node by hostname %s (ID: %s, database ID: %s)", nodeHostname, nodeID, node.ID)
		} else {
			logger.Debug("[Orchestrator] Node %s not found in database, skipping Traefik label sync: %v", nodeID, err)
			return
		}
	}

	// Get environment configuration (for global settings)
	domain := os.getEnvOrDefault("DOMAIN", "localhost")
	useTraefikRouting := os.getEnvOrDefault("USE_TRAEFIK_ROUTING", "true")

	// Only sync if domain routing is enabled
	if useTraefikRouting != "true" && useTraefikRouting != "1" {
		logger.Debug("[Orchestrator] Traefik routing disabled, skipping label sync")
		return
	}

	// Get node-specific configuration from database labels
	var useNodeSpecificDomains string
	var domainPattern string

	// Parse node labels for node-specific configuration
	if node.Labels != "" {
		var labels map[string]interface{}
		if err := json.Unmarshal([]byte(node.Labels), &labels); err == nil {
			// Check for node-specific domain settings in database
			// Handle both boolean and string values for robustness
			if val, exists := labels["obiente.use_node_specific_domains"]; exists {
				switch v := val.(type) {
				case bool:
					if v {
						useNodeSpecificDomains = "true"
					} else {
						useNodeSpecificDomains = "false"
					}
				case string:
					// Handle string values like "true", "false", "1", "0"
					useNodeSpecificDomains = strings.ToLower(v)
				case float64:
					// Handle JSON numbers (1 = true, 0 = false)
					if v == 1 {
						useNodeSpecificDomains = "true"
					} else {
						useNodeSpecificDomains = "false"
					}
				}
			}
			if pattern, ok := labels["obiente.service_domain_pattern"].(string); ok && pattern != "" {
				domainPattern = pattern
			}
		} else {
			logger.Warn("[Orchestrator] Failed to parse node labels for node %s: %v", nodeID, err)
		}
	}

	// Default values if not configured in labels
	if useNodeSpecificDomains == "" {
		useNodeSpecificDomains = "false"
	}
	if domainPattern == "" {
		domainPattern = "node-service"
	}

	// Get node subdomain from database labels
	nodeSubdomain := os.getNodeSubdomain(&node)
	enableNodeSpecific := (useNodeSpecificDomains == "true" || useNodeSpecificDomains == "1") && nodeSubdomain != ""
	useNodeSubdomainPattern := domainPattern == "service-node"

	// Log node subdomain configuration
	logger.Info("[Orchestrator] Node %s subdomain config: subdomain=%s, useNodeSpecificDomains=%s, domainPattern=%s, enableNodeSpecific=%v",
		nodeID, nodeSubdomain, useNodeSpecificDomains, domainPattern, enableNodeSpecific)

	// Define all microservices that need Traefik labels
	// IMPORTANT: api-gateway and dashboard ALWAYS use shared domains (no node subdomains)
	// This ensures proper load balancing across all nodes/clusters
	microservices := []MicroserviceConfig{
		{Name: "api-gateway", Port: 3001, BaseHost: "api"},
		{Name: "auth-service", Port: 3002, BaseHost: "auth-service"},
		{Name: "organizations-service", Port: 3003, BaseHost: "organizations-service"},
		{Name: "billing-service", Port: 3004, BaseHost: "billing-service"},
		{Name: "deployments-service", Port: 3005, BaseHost: "deployments-service"},
		{Name: "gameservers-service", Port: 3006, BaseHost: "gameservers-service"},
		{Name: "orchestrator-service", Port: 3007, BaseHost: "orchestrator-service"},
		{Name: "vps-service", Port: 3008, BaseHost: "vps-service"},
		{Name: "support-service", Port: 3009, BaseHost: "support-service"},
		{Name: "audit-service", Port: 3010, BaseHost: "audit-service"},
		{Name: "superadmin-service", Port: 3011, BaseHost: "superadmin-service"},
		{Name: "dns-service", Port: 8053, BaseHost: "dns-service"},
	}

	// Update labels for each microservice
	// Note: Only Swarm services are updated dynamically. For compose deployments,
	// node subdomain configuration should be set via environment variables.
	for _, svc := range microservices {
		// API Gateway ALWAYS uses shared domain (api.DOMAIN) for load balancing
		// Dashboard is handled separately in docker-compose with static labels
		forceSharedDomain := svc.Name == "api-gateway"

		// Use shared domain if forced or if node-specific is disabled
		useSharedDomain := forceSharedDomain || !enableNodeSpecific

		if err := os.updateServiceTraefikLabels(svc, domain, nodeSubdomain, !useSharedDomain, useNodeSubdomainPattern); err != nil {
			logger.Warn("[Orchestrator] Failed to update Traefik labels for %s: %v", svc.Name, err)
		}
	}
}

func (os *OrchestratorService) updateServiceTraefikLabels(svc MicroserviceConfig, domain string, nodeSubdomain string, enableNodeSpecific bool, useNodeSubdomainPattern bool) error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Build service domain
	var serviceDomain string
	if enableNodeSpecific && nodeSubdomain != "" {
		// Sanitize node subdomain
		sanitizedNode := strings.ToLower(nodeSubdomain)
		sanitizedNode = strings.ReplaceAll(sanitizedNode, "_", "-")
		sanitizedNode = strings.ReplaceAll(sanitizedNode, " ", "-")

		if useNodeSubdomainPattern {
			// Pattern: service-name.node.domain (e.g., "auth-service.node1.obiente.cloud")
			serviceDomain = fmt.Sprintf("%s.%s.%s", svc.BaseHost, sanitizedNode, domain)
		} else {
			// Pattern: node-service-name.domain (e.g., "node1-auth-service.obiente.cloud")
			serviceDomain = fmt.Sprintf("%s-%s.%s", sanitizedNode, svc.BaseHost, domain)
		}
		logger.Debug("[Orchestrator] Using node-specific domain for %s: %s (subdomain: %s, pattern: %v)",
			svc.Name, serviceDomain, nodeSubdomain, useNodeSubdomainPattern)
	} else {
		// Standard domain (e.g., "auth-service.obiente.cloud")
		serviceDomain = fmt.Sprintf("%s.%s", svc.BaseHost, domain)
		logger.Debug("[Orchestrator] Using shared domain for %s: %s", svc.Name, serviceDomain)
	}

	// Generate Traefik labels
	labels := os.generateMicroserviceTraefikLabels(svc.Name, serviceDomain, svc.Port)

	// Try to update as Swarm service first
	updateArgs := []string{"service", "update"}
	for k, v := range labels {
		updateArgs = append(updateArgs, "--label-add", fmt.Sprintf("%s=%s", k, v))
	}
	updateArgs = append(updateArgs, svc.Name)

	cmd := exec.CommandContext(ctx, "docker", updateArgs...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		outputStr := string(output)
		// Check if service doesn't exist - might be a regular container instead
		if strings.Contains(outputStr, "service not found") ||
			strings.Contains(outputStr, "No such service") ||
			strings.Contains(outputStr, "not found") {
			// Service not found - might be a compose deployment
			// For compose deployments, node subdomain should be configured via environment variables
			logger.Debug("[Orchestrator] Service %s not found as Swarm service (may be compose deployment)", svc.Name)
			logger.Debug("[Orchestrator] For compose deployments, configure node subdomain via NODE_SUBDOMAIN environment variable")
			return nil
		}
		return fmt.Errorf("failed to update service labels: %v, output: %s", err, outputStr)
	}

	logger.Info("[Orchestrator] Updated Traefik labels for Swarm service %s: %s", svc.Name, serviceDomain)
	return nil
}

func (os *OrchestratorService) generateMicroserviceTraefikLabels(serviceName string, serviceDomain string, port int) map[string]string {
	labels := make(map[string]string)

	// Enable Traefik
	labels["traefik.enable"] = "true"
	labels["cloud.obiente.traefik"] = "true"

	// HTTP router
	routerName := serviceName
	labels["traefik.http.routers."+routerName+".rule"] = fmt.Sprintf("Host(`%s`)", serviceDomain)
	labels["traefik.http.routers."+routerName+".entrypoints"] = "web"
	labels["traefik.http.routers."+routerName+".service"] = routerName

	// HTTPS router
	secureRouterName := routerName + "-secure"
	labels["traefik.http.routers."+secureRouterName+".rule"] = fmt.Sprintf("Host(`%s`)", serviceDomain)
	labels["traefik.http.routers."+secureRouterName+".entrypoints"] = "websecure"
	labels["traefik.http.routers."+secureRouterName+".tls.certresolver"] = "letsencrypt"
	labels["traefik.http.routers."+secureRouterName+".service"] = routerName

	// Service definition
	labels["traefik.http.services."+routerName+".loadbalancer.server.port"] = fmt.Sprintf("%d", port)
	labels["traefik.http.services."+routerName+".loadbalancer.passHostHeader"] = "true"

	// Health check configuration
	// Traefik automatically respects Docker Swarm health checks for service discovery
	// Additionally, we configure HTTP health checks via labels for better reliability
	labels["traefik.http.services."+routerName+".loadbalancer.healthcheck.path"] = "/health"
	labels["traefik.http.services."+routerName+".loadbalancer.healthcheck.interval"] = "30s"
	labels["traefik.http.services."+routerName+".loadbalancer.healthcheck.timeout"] = "5s"
	labels["traefik.http.services."+routerName+".loadbalancer.healthcheck.scheme"] = "http"

	// Load balancing configuration
	// Traefik automatically load balances when multiple services have the same router rule
	// In Swarm mode, when multiple nodes run the same service with the same router rule,
	// Traefik will distribute requests across all healthy replicas
	//
	// IMPORTANT: API Gateway and Dashboard ALWAYS use shared domains for load balancing:
	//   - API Gateway: Always uses "api.DOMAIN" (e.g., "api.obiente.cloud")
	//   - Dashboard: Always uses "DOMAIN" (e.g., "obiente.cloud")
	//   - All nodes/clusters register with the same domain
	//   - Traefik automatically load balances across all nodes/clusters
	//
	// For other microservices:
	//   - Shared load balancing (default): Configure node labels with use_node_specific_domains=false
	//     All nodes register with same domain (e.g., "auth-service.obiente.cloud")
	//     Traefik automatically load balances across all nodes
	//
	//   - Node-specific routing: Configure node labels with use_node_specific_domains=true
	//     Each node registers with node-specific domain (e.g., "node1-auth-service.obiente.cloud")
	//     API Gateway routes to specific nodes based on node subdomain
	//
	// Load balancing strategy: Traefik uses round-robin by default
	// Health checks: Traefik automatically excludes unhealthy services from load balancing
	// Multi-cluster: Works across multiple Swarm clusters when they share the same domain

	return labels
}

func (os *OrchestratorService) getNodeSubdomain(node *database.NodeMetadata) string {
	// Check node labels first (database configuration)
	if node.Labels != "" {
		var labels map[string]interface{}
		if err := json.Unmarshal([]byte(node.Labels), &labels); err == nil {
			// Check obiente.subdomain first (new format)
			if subdomain, ok := labels["obiente.subdomain"].(string); ok && subdomain != "" {
				return subdomain
			}
			// Fallback to subdomain (backwards compatibility)
			if subdomain, ok := labels["subdomain"].(string); ok && subdomain != "" {
				return subdomain
			}
		}
	}

	// Extract from hostname (fallback)
	if node.Hostname != "" {
		parts := strings.Split(node.Hostname, ".")
		if len(parts) > 0 {
			return parts[0]
		}
		return node.Hostname
	}

	return ""
}
