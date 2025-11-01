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

	lines := strings.Split(composeYaml, "\n")

	// First, do basic YAML syntax validation
	var rootNode yaml.Node
	if err := yaml.Unmarshal([]byte(composeYaml), &rootNode); err != nil {
		// Try to extract line number from YAML error
		line, col := extractErrorLocation(err.Error(), composeYaml)
		
		// Get the actual line content for better error context
		actualLine := ""
		if line > 0 && int(line) <= len(lines) {
			actualLine = lines[line-1]
		}
		
		// Improve error message with context
		errMsg := cleanYAMLError(err.Error())
		enhancedMsg := enhanceYAMLSyntaxError(errMsg, line, actualLine)
		
		// Calculate proper end column
		endCol := col + 10
		if actualLine != "" && int(col) <= len(actualLine) {
			// Try to find the end of the problematic token
			lineFromCol := actualLine[col-1:]
			if endIdx := strings.IndexAny(lineFromCol, " \t,:])}\n"); endIdx > 0 {
				endCol = col + int32(endIdx)
			} else {
				endCol = int32(len(actualLine) + 1)
			}
		}
		
		errors = append(errors, ValidationError{
			Line:        line,
			Column:      col,
			Message:     enhancedMsg,
			Severity:    "error",
			StartLine:   line,
			EndLine:     line,
			StartColumn: col,
			EndColumn:   endCol,
		})
		return errors // Can't validate further if YAML is invalid
	}

	// Validate version format if present (as warnings)
	versionWarnings := validateVersion(composeYaml)
	if len(versionWarnings) > 0 {
		errors = append(errors, versionWarnings...)
		// Continue with Docker Compose validation even if version has warnings
	}

	// Validate that no host port mappings are used (users must use routing system instead)
	portMappingErrors := validateNoHostPortMappings(composeYaml)
	if len(portMappingErrors) > 0 {
		errors = append(errors, portMappingErrors...)
		// Continue with Docker Compose validation even if port mappings are found
	}

	// Use Docker Compose config command for accurate validation
	// Docker Compose is the authoritative validator - we trust it completely
	dockerErrors := validateWithDockerCompose(ctx, composeYaml)
	return append(errors, dockerErrors...)
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
	var stdout bytes.Buffer
	cmd.Stderr = &stderr
	cmd.Stdout = &stdout // Some errors/warnings might go to stdout

	err = cmd.Run()

	// Parse both errors and warnings from output
	// Docker Compose outputs warnings even when validation passes
	errorOutput := stderr.String()
	stdOutput := stdout.String()
	if errorOutput == "" && stdOutput != "" {
		// Sometimes errors/warnings go to stdout instead
		errorOutput = stdOutput
	}
	
	// Always parse warnings (can appear even if validation passes)
	warnings := parseDockerComposeWarnings(errorOutput, composeYaml)
	errors = append(errors, warnings...)
	
	// Parse errors if Docker Compose returned an error exit code
	if err != nil {
		// Parse error output to extract line numbers and messages
		parsedErrors := parseDockerComposeErrors(errorOutput, composeYaml)
		if len(parsedErrors) == 0 && errorOutput != "" {
			// If we couldn't parse structured errors but there's output, add it as a general error
			errors = append(errors, ValidationError{
				Line:        1,
				Column:      1,
				Message:     errorOutput,
				Severity:    "error",
				StartLine:   1,
				EndLine:     1,
				StartColumn: 1,
				EndColumn:   50,
			})
		} else {
			errors = append(errors, parsedErrors...)
		}
	}

	return errors
}

// parseDockerComposeWarnings parses Docker Compose warning output
func parseDockerComposeWarnings(output string, composeYaml string) []ValidationError {
	var warnings []ValidationError
	lines := strings.Split(composeYaml, "\n")

	// Docker Compose warnings have format:
	// time="..." level=warning msg="<file>: <message>"
	warningRegex := regexp.MustCompile(`level=warning\s+msg="([^"]+)"`)
	matches := warningRegex.FindAllStringSubmatch(output, -1)

	for _, match := range matches {
		if len(match) < 2 {
			continue
		}
		msg := match[1]
		
		// Remove file path prefix if present (e.g., "/tmp/file.yml: message" -> "message")
		if idx := strings.LastIndex(msg, ": "); idx > 0 {
			msg = msg[idx+2:]
		}
		
		// Try to locate the warning in the compose file
		lineNum := int32(1)
		colNum := int32(1)
		
		// Check for common warnings that we can locate
		if strings.Contains(msg, "version") && strings.Contains(msg, "obsolete") {
			// Version warning - typically on line 1 or 2
			for i, line := range lines {
				if strings.Contains(strings.ToLower(line), "version") {
					lineNum = int32(i + 1)
					colNum = int32(strings.Index(line, "version") + 1)
					if colNum == 0 {
						colNum = 1
					}
					break
				}
			}
		}
		
		// Calculate end column
		endCol := colNum + 20
		if lineNum <= int32(len(lines)) {
			actualLine := lines[lineNum-1]
			if colNum <= int32(len(actualLine)) {
				lineFromCol := actualLine[colNum-1:]
				if endIdx := strings.IndexAny(lineFromCol, " \t,:])}\n"); endIdx > 0 {
					endCol = colNum + int32(endIdx)
				} else {
					endCol = int32(len(actualLine) + 1)
				}
			}
		}
		
		warnings = append(warnings, ValidationError{
			Line:        lineNum,
			Column:      colNum,
			Message:     msg,
			Severity:    "warning",
			StartLine:   lineNum,
			EndLine:     lineNum,
			StartColumn: colNum,
			EndColumn:   endCol,
		})
	}

	return warnings
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

		// Remove "validating <file>:" prefix if present
		line = strings.TrimPrefix(line, "validating ")
		if idx := strings.Index(line, ": "); idx > 0 {
			line = line[idx+2:]
		}

		// Try to extract line number from various formats
		lineNum, colNum, message := extractDockerComposeError(line, lines)

		if lineNum > 0 {
			// Calculate proper end column based on the error location
			endCol := int32(colNum + 30) // Default width
			if lineNum <= len(lines) {
				// Try to find the actual end position based on the line content
				actualLine := lines[lineNum-1]
				if int(colNum) <= len(actualLine) {
					// Find the end of the problematic token/word
					lineFromCol := actualLine[colNum-1:]
					// Look for end of word/token (space, comma, colon, bracket, etc.)
					if endIdx := strings.IndexAny(lineFromCol, " \t,:])}"); endIdx > 0 {
						endCol = int32(colNum + endIdx)
					} else if int(colNum) <= len(actualLine) {
						// End of line or end of content
						endCol = int32(len(actualLine) + 1)
					}
				}
			}
			
			errors = append(errors, ValidationError{
				Line:        int32(lineNum),
				Column:      int32(colNum),
				Message:     message,
				Severity:    "error",
				StartLine:   int32(lineNum),
				EndLine:     int32(lineNum),
				StartColumn: int32(colNum),
				EndColumn:   endCol,
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
	// Pattern 1: "yaml: line X: column Y: message" (most specific)
	re1 := regexp.MustCompile(`yaml:\s*line\s+(\d+)(?::\s*column\s+(\d+))?:\s*(.+)`)
	if matches := re1.FindStringSubmatch(errorLine); len(matches) >= 4 {
		if line, err := strconv.Atoi(matches[1]); err == nil {
			col := 1
			if matches[2] != "" {
				if parsedCol, err := strconv.Atoi(matches[2]); err == nil {
					col = parsedCol
				}
			}
			message := matches[3]
			
			// YAML parsers often report errors one line before the actual problem
			// when they encounter issues like missing commas/brackets on the next line
			// For "did not find expected" errors, check if the next line has the actual issue
			if strings.Contains(message, "did not find expected") && line < len(composeLines) {
				reportedLine := composeLines[line-1]
				nextLine := composeLines[line]
				
				// Check if the next line has an unclosed bracket/parenthesis/brace
				// If so, that's likely where the error actually is
				nextOpenBrackets := strings.Count(nextLine, "[") - strings.Count(nextLine, "]")
				nextOpenParens := strings.Count(nextLine, "(") - strings.Count(nextLine, ")")
				nextOpenBraces := strings.Count(nextLine, "{") - strings.Count(nextLine, "}")
				
				// If next line has unclosed brackets/parens/braces, error is likely on that line
				if nextOpenBrackets > 0 || nextOpenParens > 0 || nextOpenBraces > 0 {
					// Error is on the next line
					line = line + 1
				} else {
					// Check if reported line has unclosed brackets and next line is a new statement
					reportedOpenBrackets := strings.Count(reportedLine, "[") - strings.Count(reportedLine, "]")
					if reportedOpenBrackets > 0 {
						// Reported line has unclosed bracket
						// If next line starts a new key (not a continuation), error is on reported line
						// If next line looks like a continuation, check if it's the actual problem
						if !strings.HasPrefix(strings.TrimSpace(nextLine), "-") &&
						   !strings.HasPrefix(strings.TrimSpace(nextLine), "[") &&
						   strings.Contains(strings.TrimSpace(nextLine), ":") {
							// Next line is a new key/value pair - error is on reported line (missing closing bracket)
						} else {
							// Next line might be continuation - error could be on either line
							// But since Docker reported the previous line, try next line
							line = line + 1
						}
					}
				}
			}
			
			// Make message more descriptive
			message = improveErrorMessage(message, line, composeLines)
			return line, col, message
		}
	}

	// Pattern 2: "line X: message" or "line X, column Y: message"
	re2 := regexp.MustCompile(`line\s+(\d+)(?:,\s*column\s+(\d+))?:\s*(.+)`)
	if matches := re2.FindStringSubmatch(errorLine); len(matches) >= 3 {
		if line, err := strconv.Atoi(matches[1]); err == nil {
			col := 1
			if len(matches) >= 3 && matches[2] != "" {
				if parsedCol, err := strconv.Atoi(matches[2]); err == nil {
					col = parsedCol
				}
			}
			message := matches[len(matches)-1]
			
			// If we have column info from the regex but didn't extract it, try again
			// Pattern 2 sometimes has column in different format: "line 5, column 10:"
			if col == 1 && len(matches) >= 4 {
				// Check if message contains column info
				if colMatch := regexp.MustCompile(`column\s+(\d+)`).FindStringSubmatch(message); len(colMatch) >= 2 {
					if parsedCol, err := strconv.Atoi(colMatch[1]); err == nil {
						col = parsedCol
						// Remove column info from message
						message = regexp.MustCompile(`,\s*column\s+\d+`).ReplaceAllString(message, "")
					}
				}
			}
			
			// Apply same line offset logic as Pattern 1
			if strings.Contains(message, "did not find expected") && line < len(composeLines) {
				nextLine := composeLines[line]
				nextOpenBrackets := strings.Count(nextLine, "[") - strings.Count(nextLine, "]")
				nextOpenParens := strings.Count(nextLine, "(") - strings.Count(nextLine, ")")
				nextOpenBraces := strings.Count(nextLine, "{") - strings.Count(nextLine, "}")
				
				if nextOpenBrackets > 0 || nextOpenParens > 0 || nextOpenBraces > 0 {
					line = line + 1
				}
			}
			
			message = improveErrorMessage(message, line, composeLines)
			return line, col, message
		}
	}

	// Pattern 3: "additional properties '<field>' not allowed"
	// Can be at root level or under services/networks/volumes
	if strings.Contains(errorLine, "additional properties") {
		// Match two patterns:
		// 1. Root level: "additional properties 'field' not allowed"
		// 2. Nested: "section.name additional properties 'field' not allowed"
		reNested := regexp.MustCompile(`^([^.\s]+)\.([^.\s]+)\s+additional\s+properties\s+['"]([^'"]+)['"]`)
		if matches := reNested.FindStringSubmatch(errorLine); len(matches) >= 4 {
			section := matches[1]    // "services", "networks", "volumes"
			name := matches[2]        // service/network/volume name
			fieldName := matches[3]   // invalid field name
			message := fmt.Sprintf("Invalid field '%s' in %s '%s'", fieldName, section, name)
			
			// Find the field line in the compose file
			lineNum := findServiceFieldLine(fmt.Sprintf("%s.%s.%s", section, name, fieldName), composeLines)
			if lineNum > 0 {
				message = improveErrorMessage(message, lineNum, composeLines)
				return lineNum, 1, message
			}
			// Fallback: find the section/name line if field not found
			for i, line := range composeLines {
				trimmed := strings.TrimSpace(line)
				if strings.HasPrefix(trimmed, name+":") {
					message = improveErrorMessage(message, i+1, composeLines)
					return i + 1, 1, message
				}
			}
		}
		
		// Root level pattern: "additional properties 'field' not allowed"
		reRoot := regexp.MustCompile(`^additional\s+properties\s+['"]([^'"]+)['"]`)
		if matches := reRoot.FindStringSubmatch(errorLine); len(matches) >= 2 {
			fieldName := matches[1]
			message := fmt.Sprintf("Invalid top-level field '%s'", fieldName)
			lineNum := findKeyLine(composeLines, fieldName)
			if lineNum > 0 {
				message = improveErrorMessage(message, lineNum, composeLines)
				return lineNum, 1, message
			}
		}
	}

	// Pattern 4: "services.<name>.<field>: message" - try to find the field in the file
	if strings.Contains(errorLine, "services.") && !strings.Contains(errorLine, "additional properties") {
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

	// Pattern 5: Try to find referenced keys in the error message
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

// improveErrorMessage makes Docker Compose error messages more descriptive by adding context
func improveErrorMessage(message string, lineNum int, composeLines []string) string {
	if lineNum <= 0 || lineNum > len(composeLines) {
		return message
	}
	
	line := composeLines[lineNum-1]
	trimmed := strings.TrimSpace(line)
	
	// Check for "additional properties" errors and make them clearer
	if strings.Contains(message, "additional properties") {
		// Extract the field name from the error
		re := regexp.MustCompile(`['"]([^'"]+)['"]`)
		if matches := re.FindStringSubmatch(message); len(matches) >= 2 {
			fieldName := matches[1]
			//TODO: what the helly
			// Suggest common corrections for typos
			suggestions := map[string]string{
				"sevices":  "Did you mean 'services'?",
				"bild":     "Did you mean 'build'?",
				"conainer_name": "Did you mean 'container_name'?",
				"vlumes":   "Did you mean 'volumes'?",
				"network":  "Did you mean 'networks' (plural)?",
				"alwys":    "Did you mean 'always'?",
				"eee":      "Unknown field. Check if this should be 'environment', 'env_file', or another valid service field.",
			}
			
			if suggestion, ok := suggestions[fieldName]; ok {
				return fmt.Sprintf("Invalid field '%s' is not allowed. %s", fieldName, suggestion)
			}
			
			return fmt.Sprintf("Invalid field '%s' is not allowed in this context", fieldName)
		}
	}
	
	// Check for port-related errors
	if strings.Contains(message, "port") || (strings.Contains(message, "invalid") && strings.Contains(trimmed, "ports:")) {
		// Check if the line contains a port mapping
		if strings.Contains(trimmed, ":") {
			// Extract the port value (could be in quotes)
			portMatch := regexp.MustCompile(`["']?([^"']+)["']?`)
			if portParts := strings.Split(trimmed, ":"); len(portParts) >= 2 {
				// Check if port contains letters (invalid)
				if matched := portMatch.FindStringSubmatch(portParts[0]); len(matched) >= 2 {
					hostPort := strings.TrimSpace(matched[1])
					if regexp.MustCompile(`[a-zA-Z]`).MatchString(hostPort) {
						return fmt.Sprintf("Invalid port mapping '%s': port numbers must be numeric (0-65535). Found non-numeric characters.", strings.TrimSpace(trimmed))
					}
				}
			}
		}
		if strings.Contains(trimmed, "\"") && !strings.Contains(trimmed, ":") {
			return "Invalid port mapping. Port mappings must be in format 'host:container' (e.g., '8080:80') or 'host:container/protocol'"
		}
		if strings.Contains(message, "port") && strings.Contains(message, "invalid") {
			return fmt.Sprintf("Invalid port format in '%s'. Port mappings must be in format 'host:container' where both ports are numbers (0-65535), e.g., '8080:80'", trimmed)
		}
	}
	
	// Check for environment variable errors
	if strings.Contains(message, "environment") || strings.Contains(message, "env") {
		if !strings.Contains(trimmed, "=") && !strings.Contains(trimmed, ":") && trimmed != "" && !strings.HasPrefix(trimmed, "-") {
			return "Invalid environment variable format. Environment variables must be in format 'KEY=VALUE' or 'KEY: VALUE'"
		}
	}
	
	// Check for volume errors
	if strings.Contains(message, "volume") {
		if !strings.Contains(trimmed, ":") && !strings.HasPrefix(trimmed, "/") && !strings.HasPrefix(trimmed, "./") {
			return "Invalid volume format. Volumes must be in format 'host:container', './path:/path', or a named volume"
		}
	}
	
	// Check for service reference errors
	if strings.Contains(message, "depends_on") || strings.Contains(message, "service") {
		if strings.Contains(message, "not found") || strings.Contains(message, "does not exist") {
			return fmt.Sprintf("Service reference error: %s", message)
		}
	}
	
	// Check for "did not find expected" - improve context
	if strings.Contains(message, "did not find expected") {
		if strings.Contains(line, "[") && !strings.Contains(line, "]") {
			return message + " (missing closing bracket ']')"
		} else if strings.Contains(line, "(") && !strings.Contains(line, ")") {
			return message + " (missing closing parenthesis ')')"
		} else if strings.Contains(line, "{") && !strings.Contains(line, "}") {
			return message + " (missing closing brace '}')"
		} else if strings.Contains(trimmed, "image ") && !strings.Contains(trimmed, "image:") {
			return message + " (missing colon after 'image')"
		} else if strings.Contains(trimmed, " ") && !strings.Contains(trimmed, ":") && !strings.HasPrefix(trimmed, "-") {
			return message + " (check for missing colon or incorrect value format)"
		}
	}
	
	return message
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

// enhanceYAMLSyntaxError adds context to YAML syntax errors
func enhanceYAMLSyntaxError(errMsg string, lineNum int32, lineContent string) string {
	trimmed := strings.TrimSpace(lineContent)
	
	// Handle "did not find expected key" - often means invalid value or structure
	if strings.Contains(errMsg, "did not find expected key") {
		if trimmed != "" {
			// Check for common issues on this line
			if strings.Contains(trimmed, "version:") {
				// Likely invalid version format
				parts := strings.SplitN(trimmed, ":", 2)
				if len(parts) == 2 {
					versionValue := strings.TrimSpace(strings.Trim(parts[1], `"'`))
					if versionValue != "" {
						return fmt.Sprintf("Invalid version format '%s'. Expected format: 'major.minor' (e.g., '3.8', '3.9'). Found: %s", versionValue, errMsg)
					}
				}
				return fmt.Sprintf("Invalid version declaration. %s Check that the version value is properly formatted (e.g., '3.8').", errMsg)
			}
			
			// Check for missing colon (common cause of "did not find expected key")
			if strings.Contains(trimmed, " ") && !strings.Contains(trimmed, ":") {
				// Has space but no colon - might be missing colon
				firstWord := strings.Fields(trimmed)[0]
				return fmt.Sprintf("Missing colon after '%s'. YAML requires a colon to separate keys from values: '%s: value'", firstWord, firstWord)
			}
			
			// Generic but more helpful message
			return fmt.Sprintf("YAML syntax error: %s. Check the line structure - each key-value pair should be in format 'key: value'", errMsg)
		}
		return fmt.Sprintf("YAML syntax error: %s. Check that the line has proper YAML structure.", errMsg)
	}
	
	// Handle other common YAML errors
	if strings.Contains(errMsg, "cannot unmarshal") {
		if trimmed != "" {
			// Try to identify what value is causing the issue
			return fmt.Sprintf("Invalid value type: %s. Check that values match the expected data type (string, number, boolean, array, object).", errMsg)
		}
	}
	
	if strings.Contains(errMsg, "mapping values are not allowed") {
		return fmt.Sprintf("YAML syntax error: %s. This usually means there's a formatting issue with key-value pairs. Check for missing colons or incorrect indentation.", errMsg)
	}
	
	// Add line content context if available
	if trimmed != "" && len(trimmed) < 100 {
		return fmt.Sprintf("YAML syntax error: %s (at: %s)", errMsg, trimmed)
	}
	
	return fmt.Sprintf("YAML syntax error: %s", errMsg)
}

// Removed validateComposeStructure and validatePorts - we rely entirely on Docker Compose for validation

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

// validateVersion validates the version field format in Docker Compose (returns warnings)
// Docker Compose v2 doesn't require version, but we warn users about it
func validateVersion(composeYaml string) []ValidationError {
	var warnings []ValidationError
	lines := strings.Split(composeYaml, "\n")
	
	// Look for version: line
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "version:") || strings.HasPrefix(trimmed, "version ") {
			// Note: Docker Compose v2 doesn't require version field, but it's still accepted
			// We'll always show a warning that version is deprecated
			col := strings.Index(line, "version") + 1
				if col == 0 {
				col = 1
				}
				
			warnings = append(warnings, ValidationError{
					Line:        int32(i + 1),
					Column:      int32(col),
				Message:     "The 'version' field is deprecated in Docker Compose v2. It's accepted but not required. Consider removing it.",
				Severity:    "warning",
					StartLine:   int32(i + 1),
					EndLine:     int32(i + 1),
					StartColumn: int32(col),
				EndColumn:   int32(col + 7),
				})
			
			break // Only check first version declaration
		}
	}
	
	return warnings
}

// validateNoHostPortMappings checks for host port bindings in compose files
// Users must use the routing system instead of port mappings
func validateNoHostPortMappings(composeYaml string) []ValidationError {
	var errors []ValidationError
	
	// Parse YAML to check for port mappings
	var compose map[string]interface{}
	if err := yaml.Unmarshal([]byte(composeYaml), &compose); err != nil {
		// If we can't parse YAML, skip this check (YAML syntax errors will be caught elsewhere)
		return errors
	}
	
	lines := strings.Split(composeYaml, "\n")
	
	// Check services for port mappings
	if services, ok := compose["services"].(map[string]interface{}); ok {
		for serviceName, serviceData := range services {
			if service, ok := serviceData.(map[string]interface{}); ok {
				if ports, ok := service["ports"].([]interface{}); ok {
					for portIndex, port := range ports {
						// Check if port binding includes host port (format: "host:container" or object with published)
						if hasHostPortBinding(port) {
							// Find the line number for this port mapping
							lineNum, colNum := findPortMappingLine(serviceName, portIndex, ports, lines)
							
							// Create error message
							var portDesc string
							switch v := port.(type) {
							case string:
								portDesc = v
							case map[string]interface{}:
								if published, ok := v["published"].(int); ok {
									if target, ok := v["target"].(int); ok {
										portDesc = fmt.Sprintf("%d:%d", published, target)
									} else {
										portDesc = fmt.Sprintf("%d", published)
									}
								} else {
									portDesc = "port mapping"
								}
							default:
								portDesc = "port mapping"
							}
							
							message := fmt.Sprintf("Host port mappings are not supported. Found port mapping '%s' in service '%s'. Please use the routing configuration instead of port mappings in your compose file.", portDesc, serviceName)
							
							// Calculate end column
							endCol := colNum + 30
							if lineNum <= len(lines) {
								actualLine := lines[lineNum-1]
								if int(colNum) <= len(actualLine) {
									lineFromCol := actualLine[colNum-1:]
									if endIdx := strings.IndexAny(lineFromCol, " \t,\n"); endIdx > 0 {
										endCol = colNum + int32(endIdx)
									} else {
										endCol = int32(len(actualLine) + 1)
									}
								}
							}
							
							errors = append(errors, ValidationError{
								Line:        int32(lineNum),
								Column:      colNum,
								Message:     message,
								Severity:    "error",
								StartLine:   int32(lineNum),
								EndLine:     int32(lineNum),
								StartColumn: colNum,
								EndColumn:   endCol,
							})
						}
					}
				}
			}
		}
	}
	
	return errors
}

// hasHostPortBinding checks if a port binding includes a host port
func hasHostPortBinding(port interface{}) bool {
	switch v := port.(type) {
	case string:
		// String format: "host:container" or "host:container/protocol" indicates host binding
		// Just "container" or "container/protocol" without colon is container-only (allowed)
		return strings.Contains(v, ":")
	case map[string]interface{}:
		// Object format: if "published" field exists, it indicates host port binding
		if published, ok := v["published"].(int); ok && published > 0 {
			return true
		}
		// If only "target" exists without "published", it's container-only (allowed)
		return false
	default:
		return false
	}
}

// findPortMappingLine finds the line number of a port mapping in the compose file
func findPortMappingLine(serviceName string, portIndex int, ports []interface{}, lines []string) (int, int32) {
	// First, find the service line
	serviceLine := 0
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, serviceName+":") {
			serviceLine = i + 1
			break
		}
	}
	
	if serviceLine == 0 {
		return 1, 1 // Fallback
	}
	
	// Find the "ports:" line within the service
	portsLine := 0
	inService := false
	indentLevel := 0
	
	for i := serviceLine - 1; i < len(lines); i++ {
		line := lines[i]
		trimmed := strings.TrimSpace(line)
		
		// Check if we've left the service block (same or less indentation, different top-level key)
		if !inService && (strings.HasPrefix(line, " ") || strings.HasPrefix(line, "\t")) {
			inService = true
			// Determine indent level
			for _, char := range line {
				if char == ' ' || char == '\t' {
					indentLevel++
				} else {
					break
				}
			}
		}
		
		if inService {
			currentIndent := 0
			for _, char := range line {
				if char == ' ' || char == '\t' {
					currentIndent++
				} else {
					break
				}
			}
			
			// If we've left the service block
			if currentIndent <= indentLevel && trimmed != "" && strings.HasSuffix(trimmed, ":") && !strings.HasPrefix(trimmed, serviceName) {
				break
			}
			
			if strings.HasPrefix(trimmed, "ports:") {
				portsLine = i + 1
				break
			}
		}
	}
	
	if portsLine == 0 {
		return serviceLine, 1 // Fallback to service line
	}
	
	// Find the specific port mapping line (portIndex-th entry in the ports array)
	// Ports are typically in YAML array format with "- " prefix
	itemCount := -1 // Start at -1 because we'll count as we go
	
	for i := portsLine; i < len(lines); i++ {
		line := lines[i]
		trimmed := strings.TrimSpace(line)
		
		// Check if we've left the ports block
		if !strings.HasPrefix(line, " ") && !strings.HasPrefix(line, "\t") && trimmed != "" {
			break
		}
		
		// Check if this is a port entry (starts with "- " or is indented)
		if strings.HasPrefix(trimmed, "- ") || (strings.HasPrefix(line, "  ") && !strings.HasPrefix(trimmed, "ports:")) {
			itemCount++
			if itemCount == portIndex {
				// Calculate column (where the port value starts)
				col := 1
				if strings.HasPrefix(trimmed, "- ") {
					// Find where the actual port value starts after "- "
					portValue := strings.TrimPrefix(trimmed, "- ")
					col = strings.Index(line, portValue) + 1
					if col == 0 {
						col = strings.Index(line, "-") + 1
					}
				} else {
					col = strings.Index(line, strings.TrimSpace(line)) + 1
				}
				if col == 0 {
					col = 1
				}
				return i + 1, int32(col)
			}
		}
	}
	
	// Fallback: return the ports line
	col := strings.Index(lines[portsLine-1], "ports") + 1
	if col == 0 {
		col = 1
	}
	return portsLine, int32(col)
}
