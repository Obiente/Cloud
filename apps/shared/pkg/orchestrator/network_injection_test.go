package orchestrator

import (
	"testing"

	"gopkg.in/yaml.v3"
)

func TestSanitizeCompose_NetworkInjection(t *testing.T) {
	deploymentID := "test-deploy-123"
	composeYaml := `version: '3.8'
services:
  backend:
    image: myapp/backend:latest
    healthcheck:
      test: ["CMD-SHELL", "curl -f http://localhost:3000/health || exit 1"]
  frontend:
    image: myapp/frontend:latest
    depends_on:
      backend:
        condition: service_healthy
  cache:
    image: redis:6
`

	sanitizer := NewComposeSanitizer(deploymentID)
	sanitizedYaml, err := sanitizer.SanitizeComposeYAML(composeYaml)
	if err != nil {
		t.Fatalf("SanitizeComposeYAML failed: %v", err)
	}

	// Parse sanitized YAML
	var compose map[string]interface{}
	if err := yaml.Unmarshal([]byte(sanitizedYaml), &compose); err != nil {
		t.Fatalf("Failed to parse sanitized YAML: %v", err)
	}

	// Check that networks section exists
	networks, ok := compose["networks"].(map[string]interface{})
	if !ok {
		t.Errorf("compose should have networks section")
		return
	}

	// Check that deployment network is defined
	expectedNetworkName := "deployment-" + deploymentID
	networkDef, ok := networks[expectedNetworkName].(map[string]interface{})
	if !ok {
		t.Errorf("network %s should be defined in networks section", expectedNetworkName)
		return
	}

	// Check that network is marked as external
	if external, ok := networkDef["external"].(bool); !ok || !external {
		t.Errorf("network %s should be marked as external=true, got %v", expectedNetworkName, networkDef["external"])
	}

	// Check that obiente-network is also defined as external
	obienteNetDef, ok := networks["obiente-network"].(map[string]interface{})
	if !ok {
		t.Errorf("obiente-network should be defined in networks section")
		return
	}

	if external, ok := obienteNetDef["external"].(bool); !ok || !external {
		t.Errorf("obiente-network should be marked as external=true, got %v", obienteNetDef["external"])
	}

	// Check that all services are connected to both networks
	services, ok := compose["services"].(map[string]interface{})
	if !ok {
		t.Errorf("compose should have services section")
		return
	}

	expectedServices := []string{"backend", "frontend", "cache"}
	for _, expectedService := range expectedServices {
		service, ok := services[expectedService].(map[string]interface{})
		if !ok {
			t.Errorf("service %s should exist", expectedService)
			continue
		}

		serviceNetworks, ok := service["networks"].([]interface{})
		if !ok {
			t.Errorf("service %s should have networks defined", expectedService)
			continue
		}

		// Check if deployment network is in the list
		hasDeploymentNet := false
		hasObienteNet := false
		for _, net := range serviceNetworks {
			if netStr, ok := net.(string); ok {
				if netStr == expectedNetworkName {
					hasDeploymentNet = true
				}
				if netStr == "obiente-network" {
					hasObienteNet = true
				}
			}
		}

		if !hasDeploymentNet {
			t.Errorf("service %s should be connected to deployment network %s", expectedService, expectedNetworkName)
		}
		if !hasObienteNet {
			t.Errorf("service %s should be connected to obiente-network", expectedService)
		}
	}
}

