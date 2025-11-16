package deployments

import (
	"fmt"
	"strings"

	"gopkg.in/yaml.v3"
)

// ComposeService represents a service in Docker Compose
type ComposeService struct {
	Ports    []interface{}          `yaml:"ports"`
	Image    string                 `yaml:"image"`
	Build    interface{}            `yaml:"build"`
	Env      map[string]interface{} `yaml:"environment"`
	Volumes  []interface{}           `yaml:"volumes"`
	Networks map[string]interface{} `yaml:"networks"`
	Labels   map[string]interface{} `yaml:"labels"`
}

// ComposeFile represents the structure of a Docker Compose file
type ComposeFile struct {
	Version  string                          `yaml:"version"`
	Services map[string]ComposeService       `yaml:"services"`
	Networks map[string]interface{}         `yaml:"networks"`
	Volumes  map[string]interface{}          `yaml:"volumes"`
}

// ExtractServiceNames extracts service names from Docker Compose YAML
func ExtractServiceNames(composeYaml string) ([]string, error) {
	if composeYaml == "" {
		return []string{"default"}, nil
	}

	var compose ComposeFile
	if err := yaml.Unmarshal([]byte(composeYaml), &compose); err != nil {
		// If parsing fails, try a more lenient approach
		// Extract service names using regex/simple parsing
		return extractServiceNamesSimple(composeYaml), nil
	}

	serviceNames := make([]string, 0, len(compose.Services))
	for serviceName := range compose.Services {
		if serviceName != "" {
			serviceNames = append(serviceNames, serviceName)
		}
	}

	// If no services found, return default
	if len(serviceNames) == 0 {
		return []string{"default"}, nil
	}

	return serviceNames, nil
}

// extractServiceNamesSimple extracts service names using simple string parsing
// This is a fallback when YAML parsing fails
func extractServiceNamesSimple(composeYaml string) []string {
	serviceNames := []string{"default"}
	
	// Look for "services:" section
	lines := strings.Split(composeYaml, "\n")
	inServices := false
	indentLevel := 0
	
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		
		// Skip comments and empty lines
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}
		
		// Check if we're in the services section
		if strings.HasPrefix(trimmed, "services:") {
			inServices = true
			indentLevel = len(line) - len(strings.TrimLeft(line, " "))
			continue
		}
		
		// If we hit a top-level key after services, we're done
		if inServices {
			currentIndent := len(line) - len(strings.TrimLeft(line, " "))
			if currentIndent <= indentLevel && trimmed != "" {
				// Check if this is a top-level key (not a service)
				if !strings.HasSuffix(trimmed, ":") || 
				   strings.Contains(trimmed, "version:") ||
				   strings.Contains(trimmed, "networks:") ||
				   strings.Contains(trimmed, "volumes:") {
					break
				}
			}
			
			// Extract service name (first word before colon at services indent level + 1)
			if currentIndent == indentLevel+2 || currentIndent == indentLevel+1 {
				if strings.Contains(trimmed, ":") {
					serviceName := strings.TrimSpace(strings.Split(trimmed, ":")[0])
					if serviceName != "" && !contains(serviceNames, serviceName) {
						serviceNames = append(serviceNames, serviceName)
					}
				}
			}
		}
	}
	
	return serviceNames
}

func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

// ExtractServicePort extracts the port for a specific service from Docker Compose
func ExtractServicePort(composeYaml string, serviceName string) (int, error) {
	if composeYaml == "" {
		return 80, nil
	}

	var compose ComposeFile
	if err := yaml.Unmarshal([]byte(composeYaml), &compose); err != nil {
		return 80, fmt.Errorf("failed to parse compose YAML: %w", err)
	}

	service, exists := compose.Services[serviceName]
	if !exists {
		return 80, fmt.Errorf("service %s not found", serviceName)
	}

	// Try to extract port from ports mapping
	if len(service.Ports) > 0 {
		// Ports can be in format "8080:80" or just 8080
		for _, portEntry := range service.Ports {
			portStr := fmt.Sprintf("%v", portEntry)
			if strings.Contains(portStr, ":") {
				// Format: "8080:80" - extract the container port (right side)
				parts := strings.Split(portStr, ":")
				if len(parts) >= 2 {
					var port int
					if _, err := fmt.Sscanf(parts[1], "%d", &port); err == nil {
						return port, nil
					}
				}
			} else {
				// Format: just "8080"
				var port int
				if _, err := fmt.Sscanf(portStr, "%d", &port); err == nil {
					return port, nil
				}
			}
		}
	}

	// Default port if not found
	return 80, nil
}

