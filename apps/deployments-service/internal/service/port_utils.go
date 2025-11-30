package deployments

import (
	"github.com/obiente/cloud/apps/shared/pkg/database"
)

// determineDeploymentPort resolves the desired container port for a deployment.
// Priority order:
//  1. Routing rules that match the default service (service name "" or "default")
//  2. First routing rule with a target port
//  3. Deployment-level port configured on the deployment record
//
// If no port can be resolved, returns 0 which signals that the deployment does
// not expose a web port (background/worker workloads).
func determineDeploymentPort(deploymentID string, dep *database.Deployment) int {
	port := 0

	if routings, err := database.GetDeploymentRoutings(deploymentID); err == nil && len(routings) > 0 {
		firstRoutingPort := 0
		for _, routing := range routings {
			if routing.TargetPort <= 0 {
				continue
			}
			if routing.ServiceName == "" || routing.ServiceName == "default" {
				return routing.TargetPort
			}
			if firstRoutingPort == 0 {
				firstRoutingPort = routing.TargetPort
			}
		}
		if port == 0 && firstRoutingPort > 0 {
			port = firstRoutingPort
		}
	}

	if port == 0 && dep != nil && dep.Port != nil && *dep.Port > 0 {
		port = int(*dep.Port)
	}

	return port
}
