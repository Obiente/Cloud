package deployments

import (
	"regexp"
	"strings"
)

// sanitizeHealthcheckCommand removes dangerous characters and patterns from healthcheck commands
// to prevent command injection while allowing common healthcheck patterns
func sanitizeHealthcheckCommand(cmd string) string {
	// Remove leading/trailing whitespace
	cmd = strings.TrimSpace(cmd)

	// Disallow command chaining, piping, and redirection
	dangerousPatterns := []string{
		";", "|", "&", ">", "<", "`", "$(",
		"\n", "\r", "$(", "${",
	}

	for _, pattern := range dangerousPatterns {
		if strings.Contains(cmd, pattern) {
			// Return empty to disable custom healthcheck if injection detected
			return ""
		}
	}

	// Only allow alphanumeric, spaces, dashes, underscores, slashes, colons, dots, and common healthcheck tools
	validPattern := regexp.MustCompile(`^[a-zA-Z0-9\s\-_/.:'"\[\]]+$`)
	if !validPattern.MatchString(cmd) {
		return ""
	}

	// Limit length to prevent abuse
	if len(cmd) > 500 {
		cmd = cmd[:500]
	}

	return cmd
}
