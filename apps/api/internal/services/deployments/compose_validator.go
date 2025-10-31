package deployments

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

// ValidationError represents a validation error with location information
type ValidationError struct {
	Line        int32  `json:"line"`         // 1-based line number
	Column      int32  `json:"column"`       // 1-based column number
	Message     string `json:"message"`      // Error message
	Severity    string `json:"severity"`     // "error" or "warning"
	StartLine   int32  `json:"start_line"`   // Start line for multi-line errors
	EndLine     int32  `json:"end_line"`     // End line for multi-line errors
	StartColumn int32  `json:"start_column"` // Start column
	EndColumn   int32  `json:"end_column"`   // End column
}

// ValidateCompose validates Docker Compose YAML using Docker Compose CLI for accurate validation
func ValidateCompose(ctx context.Context, composeYaml string) []ValidationError {
	var errors []ValidationError

	if composeYaml == "" {
		return errors
	}

	// First, do basic YAML syntax validation
	var rootNode yaml.Node
	if err := yaml.Unmarshal([]byte(composeYaml), &rootNode); err != nil {
		// Try to extract line number from YAML error
		line, col := extractErrorLocation(err.Error(), composeYaml)
		errors = append(errors, ValidationError{
			Line:        line,
			Column:      col,
			Message:     fmt.Sprintf("YAML syntax error: %s", cleanYAMLError(err.Error())),
			Severity:    "error",
			StartLine:   line,
			EndLine:     line,
			StartColumn: col,
			EndColumn:   col + 10,
		})
		return errors // Can't validate further if YAML is invalid
	}

	// Use Docker Compose config command for accurate validation
	dockerErrors := validateWithDockerCompose(ctx, composeYaml)
	if len(dockerErrors) > 0 {
		return dockerErrors
	}

	// Additional structure validation if Docker validation passes
	lines := strings.Split(composeYaml, "\n")
	errors = append(errors, validateComposeStructure(&rootNode, lines)...)

	return errors
}

// validateWithDockerCompose uses `docker compose config` to validate the compose file
func validateWithDockerCompose(ctx context.Context, composeYaml string) []ValidationError {
	var errors []ValidationError

	// Create a temporary file with the compose content
	tmpDir, err := os.MkdirTemp("", "compose-validate-*")
	if err != nil {
		// Fallback to basic validation if we can't create temp file
		return errors
	}
	defer os.RemoveAll(tmpDir)

	composeFile := filepath.Join(tmpDir, "docker-compose.yml")
	if err := os.WriteFile(composeFile, []byte(composeYaml), 0644); err != nil {
		return errors
	}

	// Run docker compose config --quiet to validate
	// This command validates the compose file without outputting the normalized config
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "docker", "compose", "-f", composeFile, "config", "--quiet")
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	err = cmd.Run()
	if err == nil {
		// Validation passed
		return errors
	}

	// Parse error output to extract line numbers and messages
	errorOutput := stderr.String()
	errors = parseDockerComposeErrors(errorOutput, composeYaml)

	return errors
}

// parseDockerComposeErrors parses Docker Compose error output and extracts validation errors
func parseDockerComposeErrors(errorOutput string, composeYaml string) []ValidationError {
	var errors []ValidationError
	lines := strings.Split(composeYaml, "\n")

	// Docker Compose errors typically have formats like:
	// - "services.<name>.<field>: <message>"
	// - "line X: <message>"
	// - "yaml: line X: column Y: <message>"

	scanner := bufio.NewScanner(strings.NewReader(errorOutput))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		// Try to extract line number from various formats
		lineNum, colNum, message := extractDockerComposeError(line, lines)

		if lineNum > 0 {
			errors = append(errors, ValidationError{
				Line:        int32(lineNum),
				Column:      int32(colNum),
				Message:     message,
				Severity:    "error",
				StartLine:   int32(lineNum),
				EndLine:     int32(lineNum),
				StartColumn: int32(colNum),
				EndColumn:   int32(colNum + 30),
			})
		} else {
			// If we can't extract line number, add as general error
			errors = append(errors, ValidationError{
				Line:        1,
				Column:      1,
				Message:     line,
				Severity:    "error",
				StartLine:   1,
				EndLine:     1,
				StartColumn: 1,
				EndColumn:   50,
			})
		}
	}

	return errors
}

// extractDockerComposeError extracts line number, column, and message from Docker Compose error
func extractDockerComposeError(errorLine string, composeLines []string) (int, int, string) {
	// Pattern 1: "yaml: line X: column Y: message"
	re1 := regexp.MustCompile(`yaml:\s*line\s+(\d+):\s*column\s+(\d+):\s*(.+)`)
	if matches := re1.FindStringSubmatch(errorLine); len(matches) >= 4 {
		if line, err := strconv.Atoi(matches[1]); err == nil {
			if col, err := strconv.Atoi(matches[2]); err == nil {
				return line, col, matches[3]
			}
		}
	}

	// Pattern 2: "line X: message"
	re2 := regexp.MustCompile(`line\s+(\d+):\s*(.+)`)
	if matches := re2.FindStringSubmatch(errorLine); len(matches) >= 3 {
		if line, err := strconv.Atoi(matches[1]); err == nil {
			return line, 1, matches[2]
		}
	}

	// Pattern 3: "services.<name>.<field>: message" - try to find the field in the file
	if strings.Contains(errorLine, "services.") {
		parts := strings.SplitN(errorLine, ":", 2)
		if len(parts) == 2 {
			servicePath := strings.TrimSpace(parts[0])
			message := strings.TrimSpace(parts[1])

			// Try to find the service and field in the compose file
			lineNum := findServiceFieldLine(servicePath, composeLines)
			if lineNum > 0 {
				return lineNum, 1, message
			}
		}
	}

	// Pattern 4: Try to find referenced keys in the error message
	keyRe := regexp.MustCompile(`['"]([^'"]+)['"]`)
	if matches := keyRe.FindAllStringSubmatch(errorLine, -1); len(matches) > 0 {
		for _, match := range matches {
			if len(match) >= 2 {
				key := match[1]
				lineNum := findKeyLine(composeLines, key)
				if lineNum > 0 {
					return lineNum, 1, errorLine
				}
			}
		}
	}

	return 0, 0, errorLine
}

// findServiceFieldLine finds the line number of a service field in Docker Compose
// e.g., "services.web.ports" -> finds the "ports:" line under "web:" service
func findServiceFieldLine(servicePath string, lines []string) int {
	// servicePath format: "services.web.ports"
	parts := strings.Split(servicePath, ".")
	if len(parts) < 3 || parts[0] != "services" {
		return 0
	}

	serviceName := parts[1]
	fieldName := parts[2]

	// Find the service name
	serviceLine := 0
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, serviceName+":") {
			serviceLine = i + 1
			break
		}
	}

	if serviceLine == 0 {
		return 0
	}

	// Find the field name within the service (with proper indentation)
	for i := serviceLine; i < len(lines); i++ {
		line := lines[i]
		trimmed := strings.TrimSpace(line)

		// Check if we've moved to a different top-level key (no indentation)
		if !strings.HasPrefix(line, " ") && !strings.HasPrefix(line, "\t") && strings.HasSuffix(trimmed, ":") {
			break
		}

		if strings.HasPrefix(trimmed, fieldName+":") {
			return i + 1
		}
	}

	return serviceLine // Fallback to service line if field not found
}

// cleanYAMLError removes unnecessary prefixes from YAML error messages
func cleanYAMLError(errMsg string) string {
	// Remove "yaml:" prefix if present
	errMsg = strings.TrimPrefix(errMsg, "yaml: ")
	// Remove common prefixes
	errMsg = strings.TrimPrefix(errMsg, "error ")
	return errMsg
}

// validateComposeStructure validates the structure of a Docker Compose file
func validateComposeStructure(rootNode *yaml.Node, lines []string) []ValidationError {
	var errors []ValidationError

	// Check for services section
	servicesFound := false
	var servicesNode *yaml.Node

	if rootNode.Kind == yaml.DocumentNode && len(rootNode.Content) > 0 {
		rootMap := rootNode.Content[0]
		if rootMap.Kind == yaml.MappingNode {
			for i := 0; i < len(rootMap.Content); i += 2 {
				if i+1 >= len(rootMap.Content) {
					break
				}
				keyNode := rootMap.Content[i]
				valueNode := rootMap.Content[i+1]

				if keyNode.Value == "services" {
					servicesFound = true
					servicesNode = valueNode
					break
				}
			}
		}
	}

	if !servicesFound {
		// Find line number where "services:" should be
		lineNum := findKeyLine(lines, "services")
		if lineNum == 0 {
			lineNum = 1 // Default to first line
		}
		errors = append(errors, ValidationError{
			Line:        int32(lineNum),
			Column:      1,
			Message:     "Docker Compose file must contain a 'services:' section",
			Severity:    "error",
			StartLine:   int32(lineNum),
			EndLine:     int32(lineNum),
			StartColumn: 1,
			EndColumn:   20,
		})
		return errors
	}

	// Validate services if found
	if servicesNode != nil && servicesNode.Kind == yaml.MappingNode {
		for i := 0; i < len(servicesNode.Content); i += 2 {
			if i+1 >= len(servicesNode.Content) {
				break
			}
			serviceNameNode := servicesNode.Content[i]
			serviceDefNode := servicesNode.Content[i+1]

			if serviceNameNode != nil && serviceDefNode != nil {
				serviceName := serviceNameNode.Value
				serviceLine := int32(serviceNameNode.Line)

				// Validate service name
				if serviceName == "" {
					errors = append(errors, ValidationError{
						Line:        serviceLine,
						Column:      int32(serviceNameNode.Column),
						Message:     "Service name cannot be empty",
						Severity:    "error",
						StartLine:   serviceLine,
						EndLine:     serviceLine,
						StartColumn: int32(serviceNameNode.Column),
						EndColumn:   int32(serviceNameNode.Column) + 10,
					})
				}

				// Validate service has image or build
				if serviceDefNode.Kind == yaml.MappingNode {
					hasImage := false
					hasBuild := false

					for j := 0; j < len(serviceDefNode.Content); j += 2 {
						if j+1 >= len(serviceDefNode.Content) {
							break
						}
						keyNode := serviceDefNode.Content[j]
						if keyNode != nil {
							if keyNode.Value == "image" {
								hasImage = true
							}
							if keyNode.Value == "build" {
								hasBuild = true
							}
						}
					}

					if !hasImage && !hasBuild {
						errors = append(errors, ValidationError{
							Line:        serviceLine,
							Column:      int32(serviceNameNode.Column),
							Message:     fmt.Sprintf("Service '%s' must specify either 'image' or 'build'", serviceName),
							Severity:    "error",
							StartLine:   serviceLine,
							EndLine:     serviceLine + 5,
							StartColumn: int32(serviceNameNode.Column),
							EndColumn:   int32(serviceNameNode.Column) + int32(len(serviceName)),
						})
					}

					// Validate ports format if present
					for j := 0; j < len(serviceDefNode.Content); j += 2 {
						if j+1 >= len(serviceDefNode.Content) {
							break
						}
						keyNode := serviceDefNode.Content[j]
						valueNode := serviceDefNode.Content[j+1]

						if keyNode != nil && keyNode.Value == "ports" {
							errors = append(errors, validatePorts(valueNode)...)
						}
					}
				}
			}
		}
	}

	return errors
}

// validatePorts validates port mappings
func validatePorts(portsNode *yaml.Node) []ValidationError {
	var errors []ValidationError

	if portsNode.Kind == yaml.SequenceNode {
		for _, portEntry := range portsNode.Content {
			if portEntry != nil {
				portStr := portEntry.Value
				portLine := int32(portEntry.Line)
				portCol := int32(portEntry.Column)

				// Validate port format: "host:container" or just "container"
				if portStr != "" {
					if strings.Contains(portStr, ":") {
						parts := strings.Split(portStr, ":")
						if len(parts) != 2 {
							errors = append(errors, ValidationError{
								Line:        portLine,
								Column:      portCol,
								Message:     fmt.Sprintf("Invalid port mapping format: '%s'. Expected format: 'host:container' or 'container'", portStr),
								Severity:    "error",
								StartLine:   portLine,
								EndLine:     portLine,
								StartColumn: portCol,
								EndColumn:   portCol + int32(len(portStr)),
							})
						} else {
							// Validate host port
							if _, err := strconv.Atoi(parts[0]); err != nil {
								errors = append(errors, ValidationError{
									Line:        portLine,
									Column:      portCol,
									Message:     fmt.Sprintf("Invalid host port: '%s'. Must be a number", parts[0]),
									Severity:    "error",
									StartLine:   portLine,
									EndLine:     portLine,
									StartColumn: portCol,
									EndColumn:   portCol + int32(len(parts[0])),
								})
							}
							// Validate container port
							if _, err := strconv.Atoi(parts[1]); err != nil {
								errors = append(errors, ValidationError{
									Line:        portLine,
									Column:      portCol + int32(len(parts[0])+1),
									Message:     fmt.Sprintf("Invalid container port: '%s'. Must be a number", parts[1]),
									Severity:    "error",
									StartLine:   portLine,
									EndLine:     portLine,
									StartColumn: portCol + int32(len(parts[0])+1),
									EndColumn:   portCol + int32(len(portStr)),
								})
							}
						}
					} else {
						// Single port number
						if _, err := strconv.Atoi(portStr); err != nil {
							errors = append(errors, ValidationError{
								Line:        portLine,
								Column:      portCol,
								Message:     fmt.Sprintf("Invalid port: '%s'. Must be a number", portStr),
								Severity:    "error",
								StartLine:   portLine,
								EndLine:     portLine,
								StartColumn: portCol,
								EndColumn:   portCol + int32(len(portStr)),
							})
						}
					}
				}
			}
		}
	}

	return errors
}

// extractErrorLocation tries to extract line and column from YAML error message
func extractErrorLocation(errorMsg string, yamlContent string) (int32, int32) {
	// Try to match patterns like "line 5: column 10:" or "line 5, column 10"
	re := regexp.MustCompile(`line (\d+).*column (\d+)`)
	matches := re.FindStringSubmatch(errorMsg)
	if len(matches) >= 3 {
		if line, err := strconv.Atoi(matches[1]); err == nil {
			if col, err := strconv.Atoi(matches[2]); err == nil {
				return int32(line), int32(col)
			}
		}
	}

	// Try to match just line number
	re2 := regexp.MustCompile(`line (\d+)`)
	matches2 := re2.FindStringSubmatch(errorMsg)
	if len(matches2) >= 2 {
		if line, err := strconv.Atoi(matches2[1]); err == nil {
			return int32(line), 1
		}
	}

	return 1, 1 // Default to first line, first column
}

// findKeyLine finds the line number of a key in YAML content
func findKeyLine(lines []string, key string) int {
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, key+":") {
			return i + 1 // 1-based line number
		}
	}
	return 0
}
