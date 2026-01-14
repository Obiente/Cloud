package orchestrator

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/obiente/cloud/apps/shared/pkg/database"
	"github.com/obiente/cloud/apps/shared/pkg/logger"
	"github.com/obiente/cloud/apps/shared/pkg/quota"
	"github.com/obiente/cloud/apps/shared/pkg/utils"

	"gopkg.in/yaml.v3"
)

// Config operations for deployments

func envFlagEnabled(name string) bool {
	v := strings.TrimSpace(os.Getenv(name))
	if v == "" {
		return false
	}
	switch strings.ToLower(v) {
	case "1", "true", "yes", "y", "on", "enabled":
		return true
	default:
		return false
	}
}

func (dm *DeploymentManager) getPlanLimitsForDeployment(deploymentID string) (maxMemoryBytes int64, maxCPUCores int, err error) {
	// Get deployment to find organization ID
	var deployment database.Deployment
	if err := database.DB.Where("id = ?", deploymentID).First(&deployment).Error; err != nil {
		return 0, 0, fmt.Errorf("failed to get deployment: %w", err)
	}

	// Use shared helper function to get effective limits
	maxMemoryBytes, maxCPUCores, err = quota.GetEffectiveLimits(deployment.OrganizationID)
	if err != nil {
		return 0, 0, fmt.Errorf("failed to get effective limits: %w", err)
	}

	// Log warning if limits were capped (this would be logged in GetEffectiveLimits if we add logging there)
	// For now, we'll log here if needed
	return maxMemoryBytes, maxCPUCores, nil
}

func (dm *DeploymentManager) applyPlanLimits(config *DeploymentConfig) error {
	maxMemoryBytes, maxCPUCores, err := dm.getPlanLimitsForDeployment(config.DeploymentID)
	if err != nil {
		logger.Warn("[DeploymentManager] Failed to get plan limits for deployment %s: %v", config.DeploymentID, err)
		// Continue without limits if we can't get them
		return nil
	}

	// Cap memory if limit is set (non-zero)
	if maxMemoryBytes > 0 && config.Memory > maxMemoryBytes {
		logger.Info("[DeploymentManager] Capping memory for deployment %s from %d bytes to plan limit %d bytes", config.DeploymentID, config.Memory, maxMemoryBytes)
		config.Memory = maxMemoryBytes
	}

	// Cap CPU if limit is set (non-zero)
	// Convert CPU cores to CPU shares (1024 shares = 1 core)
	if maxCPUCores > 0 {
		maxCPUShares := int64(maxCPUCores) * 1024
		if config.CPUShares > maxCPUShares {
			logger.Info("[DeploymentManager] Capping CPU for deployment %s from %d shares (%d cores) to plan limit %d shares (%d cores)",
				config.DeploymentID, config.CPUShares, config.CPUShares/1024, maxCPUShares, maxCPUCores)
			config.CPUShares = maxCPUShares
		}
	}

	return nil
}

func (dm *DeploymentManager) injectPlanLimitsIntoCompose(composeYaml string, deploymentID string, maxMemoryBytes int64, maxCPUCores int) (string, error) {
	// Parse YAML
	var compose map[string]interface{}
	if err := yaml.Unmarshal([]byte(composeYaml), &compose); err != nil {
		return "", fmt.Errorf("failed to parse compose YAML: %w", err)
	}

	// Detect if we're in Swarm mode
	isSwarmMode := utils.IsSwarmModeEnabled()

	// Convert CPU cores to CPU shares (1024 shares = 1 core)
	maxCPUShares := int64(maxCPUCores) * 1024

	// Inject limits into each service
	if services, ok := compose["services"].(map[string]interface{}); ok {
		for serviceName, serviceData := range services {
			service, ok := serviceData.(map[string]interface{})
			if !ok {
				continue
			}

			if isSwarmMode {
				// For Swarm mode, limits go in deploy.resources.limits
				var deploy map[string]interface{}
				if existingDeploy, ok := service["deploy"].(map[string]interface{}); ok {
					deploy = existingDeploy
				} else {
					deploy = make(map[string]interface{})
					service["deploy"] = deploy
				}

				// Get or create resources section
				var resources map[string]interface{}
				if existingResources, ok := deploy["resources"].(map[string]interface{}); ok {
					resources = existingResources
				} else {
					resources = make(map[string]interface{})
					deploy["resources"] = resources
				}

				// Get or create limits section
				var limits map[string]interface{}
				if existingLimits, ok := resources["limits"].(map[string]interface{}); ok {
					limits = existingLimits
				} else {
					limits = make(map[string]interface{})
					resources["limits"] = limits
				}

				// Apply memory limit (cap existing limit if present, or set if not)
				if maxMemoryBytes > 0 {
					// Convert bytes to a human-readable format for compose (e.g., "2G" for 2GB)
					maxMemoryMB := maxMemoryBytes / (1024 * 1024)
					var maxMemoryStr string
					if maxMemoryMB >= 1024 {
						maxMemoryStr = fmt.Sprintf("%dG", maxMemoryMB/1024)
					} else {
						maxMemoryStr = fmt.Sprintf("%dM", maxMemoryMB)
					}

					// Cap existing memory limit if present, otherwise set it
					if existingMemory, ok := limits["memory"].(string); ok && existingMemory != "" {
						// Parse existing memory limit and cap it
						existingBytes := parseMemoryString(existingMemory)
						if existingBytes > maxMemoryBytes {
							limits["memory"] = maxMemoryStr
							logger.Info("[DeploymentManager] Capped memory limit for service %s from %s to plan limit %s", serviceName, existingMemory, maxMemoryStr)
						}
					} else {
						limits["memory"] = maxMemoryStr
					}
				}

				// Apply CPU limit (cap existing limit if present, or set if not)
				if maxCPUCores > 0 {
					maxCPUsStr := fmt.Sprintf("%.2f", float64(maxCPUCores))
					// Cap existing CPU limit if present, otherwise set it
					if existingCPUs, ok := limits["cpus"].(string); ok && existingCPUs != "" {
						// Parse existing CPU limit and cap it
						existingCores := parseCPUString(existingCPUs)
						if existingCores > float64(maxCPUCores) {
							limits["cpus"] = maxCPUsStr
							logger.Info("[DeploymentManager] Capped CPU limit for service %s from %s to plan limit %s", serviceName, existingCPUs, maxCPUsStr)
						}
					} else {
						limits["cpus"] = maxCPUsStr
					}
				}

				logger.Debug("[DeploymentManager] Applied plan limits to service %s in deploy.resources.limits (Swarm mode): memory=%s, cpus=%s",
					serviceName, limits["memory"], limits["cpus"])
			} else {
				// For non-Swarm mode, limits go in resources.limits
				var resources map[string]interface{}
				if existingResources, ok := service["resources"].(map[string]interface{}); ok {
					resources = existingResources
				} else {
					resources = make(map[string]interface{})
					service["resources"] = resources
				}

				// Get or create limits section
				var limits map[string]interface{}
				if existingLimits, ok := resources["limits"].(map[string]interface{}); ok {
					limits = existingLimits
				} else {
					limits = make(map[string]interface{})
					resources["limits"] = limits
				}

				// Apply memory limit (cap existing limit if present, or set if not)
				if maxMemoryBytes > 0 {
					maxMemoryMB := maxMemoryBytes / (1024 * 1024)
					var maxMemoryStr string
					if maxMemoryMB >= 1024 {
						maxMemoryStr = fmt.Sprintf("%dG", maxMemoryMB/1024)
					} else {
						maxMemoryStr = fmt.Sprintf("%dM", maxMemoryMB)
					}

					// Cap existing memory limit if present, otherwise set it
					if existingMemory, ok := limits["memory"].(string); ok && existingMemory != "" {
						existingBytes := parseMemoryString(existingMemory)
						if existingBytes > maxMemoryBytes {
							limits["memory"] = maxMemoryStr
							logger.Info("[DeploymentManager] Capped memory limit for service %s from %s to plan limit %s", serviceName, existingMemory, maxMemoryStr)
						}
					} else {
						limits["memory"] = maxMemoryStr
					}
				}

				// Apply CPU limit (cap existing limit if present, or set if not)
				if maxCPUShares > 0 {
					maxCPUsStr := fmt.Sprintf("%.2f", float64(maxCPUShares)/1024.0)
					// Cap existing CPU limit if present, otherwise set it
					if existingCPUs, ok := limits["cpus"].(string); ok && existingCPUs != "" {
						existingCores := parseCPUString(existingCPUs)
						maxCores := float64(maxCPUShares) / 1024.0
						if existingCores > maxCores {
							limits["cpus"] = maxCPUsStr
							logger.Info("[DeploymentManager] Capped CPU limit for service %s from %s to plan limit %s", serviceName, existingCPUs, maxCPUsStr)
						}
					} else {
						limits["cpus"] = maxCPUsStr
					}
				}

				logger.Debug("[DeploymentManager] Applied plan limits to service %s in resources.limits (non-Swarm mode): memory=%s, cpus=%s",
					serviceName, limits["memory"], limits["cpus"])
			}
		}
	}

	// Marshal back to YAML
	modifiedYaml, err := yaml.Marshal(compose)
	if err != nil {
		return "", fmt.Errorf("failed to marshal modified compose YAML: %w", err)
	}

	return string(modifiedYaml), nil
}

func (dm *DeploymentManager) injectTraefikLabelsIntoCompose(composeYaml string, deploymentID string, routings []database.DeploymentRouting) (string, error) {
	// Parse YAML
	var compose map[string]interface{}
	if err := yaml.Unmarshal([]byte(composeYaml), &compose); err != nil {
		return "", fmt.Errorf("failed to parse compose YAML: %w", err)
	}

	// Detect if we're in Swarm mode using ENABLE_SWARM environment variable
	// In Swarm mode, labels must be in deploy.labels
	// In non-Swarm mode, labels must be at the top-level labels
	isSwarmMode := utils.IsSwarmModeEnabled()
	if isSwarmMode {
		logger.Debug("[DeploymentManager] ENABLE_SWARM=true - labels will be placed in deploy.labels")
	} else {
		logger.Debug("[DeploymentManager] ENABLE_SWARM=false or not set - labels will be placed at top-level")
	}

	// Get deployment domain from routings if available
	var deploymentDomain string
	if len(routings) > 0 {
		deploymentDomain = routings[0].Domain
	}

	// Inject labels into each service
	if services, ok := compose["services"].(map[string]interface{}); ok {
		for serviceName, serviceData := range services {
			if service, ok := serviceData.(map[string]interface{}); ok {
				// Extract port from compose service if available
				var servicePort *int
				if ports, ok := service["ports"].([]interface{}); ok && len(ports) > 0 {
					// Try to extract port from first port mapping
					if portStr, ok := ports[0].(string); ok {
						// Format: "host:container" or just "container"
						parts := strings.Split(portStr, ":")
						if len(parts) >= 2 {
							if p, err := strconv.Atoi(parts[len(parts)-1]); err == nil {
								servicePort = &p
							}
						} else if len(parts) == 1 {
							if p, err := strconv.Atoi(parts[0]); err == nil {
								servicePort = &p
							}
						}
					}
				} else if expose, ok := service["expose"].([]interface{}); ok && len(expose) > 0 {
					// Check exposed ports
					if portStr, ok := expose[0].(string); ok {
						if p, err := strconv.Atoi(portStr); err == nil {
							servicePort = &p
						}
					}
				}

				// Get the actual Swarm network name dynamically (supports any stack name)
				// This will find networks matching the pattern *_obiente-network
				swarmNetworkName, err := dm.getSwarmNetworkName(context.Background())
				if err != nil {
					logger.Warn("[DeploymentManager] Failed to get Swarm network name, using fallback: %v", err)
					swarmNetworkName = "obiente_obiente-network"
				}

				// Generate Traefik labels for this service
				traefikLabels := generateTraefikLabels(deploymentID, serviceName, routings, servicePort, swarmNetworkName)

				// When Traefik handles routing, we should not expose ports to the host
				// Convert ports to expose (internal network only) if Traefik labels are present
				if len(traefikLabels) > 0 {
					// Check if service has ports that should be converted to expose
					if ports, ok := service["ports"].([]interface{}); ok && len(ports) > 0 {
						// Extract container ports from ports mapping
						exposedPorts := []interface{}{}
						for _, port := range ports {
							var containerPort string
							switch v := port.(type) {
							case string:
								// Format: "host:container" or "container" or "container/protocol"
								if strings.Contains(v, ":") {
									parts := strings.SplitN(v, ":", 2)
									if len(parts) == 2 {
										containerPort = strings.TrimSpace(parts[1])
									}
								} else {
									containerPort = v
								}
							case map[string]interface{}:
								// Port mapping object format
								if target, ok := v["target"].(int); ok {
									containerPort = strconv.Itoa(target)
								} else if published, ok := v["published"].(int); ok {
									containerPort = strconv.Itoa(published)
								}
							}
							if containerPort != "" {
								exposedPorts = append(exposedPorts, containerPort)
							}
						}

						// Remove ports and add expose instead (internal network only)
						delete(service, "ports")
						if len(exposedPorts) > 0 {
							// Merge with existing expose if present
							var existingExpose []interface{}
							if existing, ok := service["expose"].([]interface{}); ok {
								existingExpose = existing
							}
							// Combine and deduplicate
							exposeMap := make(map[string]bool)
							for _, p := range existingExpose {
								if portStr, ok := p.(string); ok {
									exposeMap[portStr] = true
								}
							}
							for _, p := range exposedPorts {
								if portStr, ok := p.(string); ok {
									exposeMap[portStr] = true
								}
							}
							// Convert back to list
							finalExpose := []interface{}{}
							for port := range exposeMap {
								finalExpose = append(finalExpose, port)
							}
							if len(finalExpose) > 0 {
								service["expose"] = finalExpose
								logger.Info("[DeploymentManager] Converted ports to expose for service %s (Traefik routing - no host port exposure)", serviceName)
							}
						}
					}
				}

				// Determine the health check port - ALWAYS use routing target port if available
				// Never use compose file port as it may be a default value
				var healthCheckPort *int
				routingPortUsed := false

				if len(routings) > 0 {
					// Find routing for this service
					// Service name matching: exact match, or "default" service matches routing with ServiceName="default" or ""
					for _, routing := range routings {
						serviceMatches := routing.ServiceName == serviceName ||
							(serviceName == "default" && (routing.ServiceName == "" || routing.ServiceName == "default")) ||
							(routing.ServiceName == "default" && serviceName == "")

						if serviceMatches {
							if routing.TargetPort > 0 {
								healthCheckPort = &routing.TargetPort
								routingPortUsed = true
								logger.Info("[DeploymentManager] Using routing target port %d for health check (service: %s, routing service: %s, domain: %s)", routing.TargetPort, serviceName, routing.ServiceName, routing.Domain)
								break
							}
						}
					}

					// If no matching service found but we have routings, use first routing's target port as fallback
					if !routingPortUsed && len(routings) > 0 && routings[0].TargetPort > 0 {
						healthCheckPort = &routings[0].TargetPort
						routingPortUsed = true
						logger.Info("[DeploymentManager] No exact service match found, using first routing's target port %d for health check (service: %s, routing service: %s)", routings[0].TargetPort, serviceName, routings[0].ServiceName)
					}
				}

				// Optional default healthcheck injection.
				// IMPORTANT: In Swarm, failing healthchecks can trigger task restarts (SIGTERM/143).
				// We therefore default this behavior OFF and only allow it when explicitly enabled.
				// Healthchecks are ONLY injected when:
				// 1. OBIENTE_ENABLE_AUTO_HEALTHCHECK env var is set to true
				// 2. Actual routing rules exist in the database (routingPortUsed = true)
				// 3. No user-defined healthcheck already exists in the compose file
				autoHealthchecksEnabled := envFlagEnabled("OBIENTE_ENABLE_AUTO_HEALTHCHECK")
				if autoHealthchecksEnabled && routingPortUsed && healthCheckPort != nil {
					_, hasHealthcheck := service["healthcheck"]
					if !hasHealthcheck {
						// Best-effort check: only runs if `nc` already exists in the image.
						// If it's missing, we exit 0 to avoid restart loops.
						healthcheckCmd := fmt.Sprintf(`sh -c 'if command -v nc >/dev/null 2>&1; then nc -z localhost %d; else exit 0; fi'`, *healthCheckPort)
						healthcheck := map[string]interface{}{
							"test":         []interface{}{"CMD-SHELL", healthcheckCmd},
							"interval":     "30s",
							"timeout":      "10s",
							"retries":      3,
							"start_period": "40s",
						}
						service["healthcheck"] = healthcheck
						logger.Info("[DeploymentManager] Added optional health check for service %s on routing port %d (routing rules exist)", serviceName, *healthCheckPort)
					}
				}
				// Helper function to get or create labels map from various formats
				getLabelsMap := func(labelsValue interface{}) map[string]interface{} {
					labels := make(map[string]interface{})
					if labelsValue == nil {
						return labels
					}
					if existingLabels, ok := labelsValue.(map[string]interface{}); ok {
						labels = existingLabels
					} else if existingLabelsList, ok := labelsValue.([]interface{}); ok {
						// Convert list format to map format
						for _, labelItem := range existingLabelsList {
							if labelStr, ok := labelItem.(string); ok {
								// Parse "key=value" or "key: value" format
								if strings.Contains(labelStr, "=") {
									parts := strings.SplitN(labelStr, "=", 2)
									if len(parts) == 2 {
										labels[strings.TrimSpace(parts[0])] = strings.TrimSpace(parts[1])
									}
								}
							}
						}
					}
					return labels
				}

				// Remove old Traefik labels that might conflict
				removeTraefikLabels := func(labels map[string]interface{}) {
					traefikKeysToRemove := []string{}
					for key := range labels {
						if strings.HasPrefix(key, "traefik.http.routers.") ||
							strings.HasPrefix(key, "traefik.http.services.") ||
							key == "traefik.enable" ||
							key == "cloud.obiente.traefik" {
							traefikKeysToRemove = append(traefikKeysToRemove, key)
						}
					}
					for _, key := range traefikKeysToRemove {
						delete(labels, key)
					}
				}

				// Add Traefik and management labels
				addLabels := func(labels map[string]interface{}) {
					// Merge new Traefik labels (Traefik labels take precedence)
					for k, v := range traefikLabels {
						labels[k] = v
					}

					// Add management labels
					labels["cloud.obiente.managed"] = "true"
					labels["cloud.obiente.deployment_id"] = deploymentID
					labels["cloud.obiente.service_name"] = serviceName
					// Only set cloud.obiente.traefik if Traefik labels were generated (i.e., routing rules exist)
					if len(traefikLabels) > 0 {
						labels["cloud.obiente.traefik"] = "true" // Required for Traefik discovery
					}
					if deploymentDomain != "" {
						labels["cloud.obiente.domain"] = deploymentDomain
					}
				}

				if isSwarmMode {
					// For Docker Swarm mode, Traefik requires labels to be in deploy.labels
					// Get or create deploy section
					var deploy map[string]interface{}
					if existingDeploy, ok := service["deploy"].(map[string]interface{}); ok {
						deploy = existingDeploy
					} else {
						deploy = make(map[string]interface{})
						service["deploy"] = deploy
					}

					// Get or create labels map under deploy.labels
					labels := getLabelsMap(deploy["labels"])
					removeTraefikLabels(labels)
					addLabels(labels)

					// Update deploy.labels (required for Swarm mode Traefik discovery)
					deploy["labels"] = labels
					if len(traefikLabels) > 0 {
						logger.Debug("[DeploymentManager] Added %d Traefik labels to deploy.labels for service %s (Swarm mode)", len(traefikLabels), serviceName)
					}
				} else {
					// For non-Swarm mode (Docker Compose), Traefik reads labels from top-level labels
					// Get or create top-level labels
					labels := getLabelsMap(service["labels"])
					removeTraefikLabels(labels)
					addLabels(labels)

					// Update top-level labels (required for non-Swarm mode Traefik discovery)
					service["labels"] = labels
					if len(traefikLabels) > 0 {
						logger.Debug("[DeploymentManager] Added %d Traefik labels to top-level labels for service %s (non-Swarm mode)", len(traefikLabels), serviceName)
					}
				}
			}
		}
	}

	// Ensure network configuration is set correctly for Swarm mode
	// In Swarm mode, services must be on the network that Traefik monitors
	if isSwarmMode {
		// Ensure networks section exists
		var networks map[string]interface{}
		if existingNetworks, ok := compose["networks"].(map[string]interface{}); ok {
			networks = existingNetworks
		} else {
			networks = make(map[string]interface{})
			compose["networks"] = networks
		}

		// Get the actual Swarm network name dynamically (supports any stack name)
		// This will find networks matching the pattern *_obiente-network
		swarmNetworkName, err := dm.getSwarmNetworkName(context.Background())
		if err != nil {
			logger.Warn("[DeploymentManager] Failed to get Swarm network name, using fallback: %v", err)
			// Fallback: try to find any network ending with _obiente-network
			swarmNetworkName = "obiente_obiente-network"
		}

		// Add or update obiente-network to be external (references the Swarm network)
		// In Swarm mode, the network name is prefixed with stack name: {stack-name}_obiente-network
		networkConfig := map[string]interface{}{
			"external": true,
			"name":     swarmNetworkName, // Use the dynamically discovered Swarm network name
		}
		networks["obiente-network"] = networkConfig

		// Ensure all services are connected to the network
		if services, ok := compose["services"].(map[string]interface{}); ok {
			for serviceName, serviceData := range services {
				if service, ok := serviceData.(map[string]interface{}); ok {
					// Get or create networks section for this service
					var serviceNetworks map[string]interface{}
					if existingServiceNetworks, ok := service["networks"].(map[string]interface{}); ok {
						serviceNetworks = existingServiceNetworks
					} else if existingServiceNetworksList, ok := service["networks"].([]interface{}); ok {
						// Convert list format to map format
						serviceNetworks = make(map[string]interface{})
						for _, netItem := range existingServiceNetworksList {
							if netStr, ok := netItem.(string); ok {
								serviceNetworks[netStr] = nil
							}
						}
					} else {
						serviceNetworks = make(map[string]interface{})
					}

					// Ensure obiente-network is in the service's networks
					serviceNetworks["obiente-network"] = nil
					service["networks"] = serviceNetworks
					logger.Debug("[DeploymentManager] Ensured service %s is connected to obiente-network (Swarm mode)", serviceName)
				}
			}
		}
	}

	// Marshal back to YAML
	labeledYaml, err := yaml.Marshal(compose)
	if err != nil {
		return "", fmt.Errorf("failed to marshal labeled compose YAML: %w", err)
	}

	return string(labeledYaml), nil
}

func generateTraefikLabels(deploymentID string, serviceName string, routings []database.DeploymentRouting, servicePort *int, networkName string) map[string]string {
	labels := make(map[string]string)

	// Filter routings for this service name
	serviceRoutings := []database.DeploymentRouting{}
	for _, routing := range routings {
		if routing.ServiceName == serviceName || (serviceName == "default" && routing.ServiceName == "") {
			serviceRoutings = append(serviceRoutings, routing)
		}
	}

	// If no specific routing found, don't enable Traefik unless we have a port
	// User must configure routing before the service will be accessible via Traefik
	if len(serviceRoutings) == 0 {
		// Only enable Traefik if we have port information (but no routing rules means no router will be created)
		// We still need to NOT enable it to avoid Traefik errors
		return labels // Return empty - don't enable Traefik for services without routing rules
	}

	// Enable Traefik only when we have routing rules
	labels["traefik.enable"] = "true"
	labels["cloud.obiente.traefik"] = "true" // Required for Traefik to discover this container

	// Specify which network Traefik should use (prevents using ingress network IPs)
	if networkName != "" {
		labels["traefik.docker.network"] = networkName
	}

	// Generate labels for each routing rule
	for idx, routing := range serviceRoutings {
		routerName := deploymentID
		if serviceName != "default" {
			routerName = deploymentID + "-" + serviceName
		}
		if idx > 0 {
			routerName = fmt.Sprintf("%s-%d", routerName, idx)
		}

		// Build rule: Host or Host + PathPrefix
		rule := "Host(`" + routing.Domain + "`)"
		if routing.PathPrefix != "" {
			rule = rule + " && PathPrefix(`" + routing.PathPrefix + "`)"
		}
		labels["traefik.http.routers."+routerName+".rule"] = rule

		// Entrypoints - respect protocol field
		// HTTP protocol should use web (no SSL), HTTPS protocol or SSLEnabled=true should use websecure
		shouldUseSSL := false
		if routing.Protocol == "https" {
			// HTTPS protocol always uses SSL
			shouldUseSSL = true
		} else if routing.Protocol == "http" {
			// HTTP protocol never uses SSL, regardless of SSLEnabled flag
			shouldUseSSL = false
		} else {
			// For other protocols (grpc, etc.) or if protocol is not set, use SSLEnabled flag
			shouldUseSSL = routing.SSLEnabled
		}

		if shouldUseSSL {
			labels["traefik.http.routers."+routerName+".entrypoints"] = "websecure"
			if routing.SSLCertResolver != "" && routing.SSLCertResolver != "internal" {
				labels["traefik.http.routers."+routerName+".tls.certresolver"] = routing.SSLCertResolver
			} else if routing.SSLCertResolver == "internal" {
				// For internal SSL, don't set certresolver (let app handle it)
				labels["traefik.http.routers."+routerName+".entrypoints"] = "web"
			}
		} else {
			// HTTP-only: explicitly set web entrypoint and ensure no TLS labels
			labels["traefik.http.routers."+routerName+".entrypoints"] = "web"
			// Explicitly remove any TLS configuration for HTTP-only routers
			// Note: Docker label deletion in compose requires the label to not exist at all
			// We rely on only setting web entrypoint which won't trigger TLS
		}

		// Service name label (used for both service definition and router reference)
		serviceNameLabel := routerName

		// Explicitly set the service for the router (required for Swarm mode)
		labels["traefik.http.routers."+routerName+".service"] = serviceNameLabel

		// Service port
		labels["traefik.http.services."+serviceNameLabel+".loadbalancer.server.port"] = strconv.Itoa(routing.TargetPort)
	}

	return labels
}

func parseMemoryString(memoryStr string) int64 {
	memoryStr = strings.TrimSpace(memoryStr)
	if memoryStr == "" {
		return 0
	}

	// Remove any whitespace and convert to uppercase for parsing
	memoryStr = strings.ToUpper(strings.TrimSpace(memoryStr))

	// Extract number and unit
	var numStr string
	var unit string
	for i, r := range memoryStr {
		if r >= '0' && r <= '9' || r == '.' {
			numStr += string(r)
		} else {
			unit = memoryStr[i:]
			break
		}
	}

	if numStr == "" {
		return 0
	}

	num, err := strconv.ParseFloat(numStr, 64)
	if err != nil {
		return 0
	}

	// Convert to bytes based on unit
	switch unit {
	case "B", "":
		return int64(num)
	case "K", "KB":
		return int64(num * 1024)
	case "M", "MB":
		return int64(num * 1024 * 1024)
	case "G", "GB":
		return int64(num * 1024 * 1024 * 1024)
	case "T", "TB":
		return int64(num * 1024 * 1024 * 1024 * 1024)
	default:
		return 0
	}
}

func parseCPUString(cpuStr string) float64 {
	cpuStr = strings.TrimSpace(cpuStr)
	if cpuStr == "" {
		return 0
	}

	cores, err := strconv.ParseFloat(cpuStr, 64)
	if err != nil {
		return 0
	}

	return cores
}
// addTraefikNetworkToRoutedServices adds obiente-network to services that have routing configured
// This allows Traefik to discover and route to these services on the shared obiente-network
// while maintaining deployment isolation through the deployment-specific network
func (dm *DeploymentManager) addTraefikNetworkToRoutedServices(composeYaml string, routings []database.DeploymentRouting) (string, error) {
	// Parse the compose file
	var compose map[string]interface{}
	if err := yaml.Unmarshal([]byte(composeYaml), &compose); err != nil {
		return "", fmt.Errorf("failed to parse compose YAML: %w", err)
	}

	// Build a set of service names that have routing configured
	servicesToRoute := make(map[string]bool)
	for _, routing := range routings {
		// ServiceName identifies which service this routing rule applies to
		// If empty, it means the "default" service or all services
		serviceName := strings.TrimSpace(routing.ServiceName)
		if serviceName == "" || serviceName == "default" {
			// For default, apply to first service (common pattern)
			if services, ok := compose["services"].(map[string]interface{}); ok {
				for svc := range services {
					servicesToRoute[svc] = true
					break // Only first service
				}
			}
		} else {
			servicesToRoute[serviceName] = true
		}
	}

	// Add obiente-network to routed services
	if services, ok := compose["services"].(map[string]interface{}); ok {
		for serviceName, serviceData := range services {
			if servicesToRoute[serviceName] {
				if service, ok := serviceData.(map[string]interface{}); ok {
					// Get existing networks if any
					var networks []interface{}
					if existingNets, ok := service["networks"].([]interface{}); ok {
						networks = existingNets
					}

					// Add obiente-network if not already present
					hasObienteNetwork := false
					for _, net := range networks {
						if netStr, ok := net.(string); ok && netStr == "obiente-network" {
							hasObienteNetwork = true
							break
						}
					}

					if !hasObienteNetwork {
						networks = append(networks, "obiente-network")
						service["networks"] = networks
						logger.Debug("[DeploymentManager] Added obiente-network to routed service %s for Traefik discovery", serviceName)
					}
				}
			}
		}
	}

	// Ensure obiente-network is defined in networks section
	if _, ok := compose["networks"]; !ok {
		compose["networks"] = make(map[string]interface{})
	}

	networks, ok := compose["networks"].(map[string]interface{})
	if !ok {
		networks = make(map[string]interface{})
		compose["networks"] = networks
	}

	// Mark obiente-network as external (it already exists in the system)
	if _, exists := networks["obiente-network"]; !exists {
		networks["obiente-network"] = map[string]interface{}{
			"external": true,
		}
	}

	// Marshal back to YAML
	result, err := yaml.Marshal(compose)
	if err != nil {
		return "", fmt.Errorf("failed to marshal compose YAML: %w", err)
	}

	return string(result), nil
}