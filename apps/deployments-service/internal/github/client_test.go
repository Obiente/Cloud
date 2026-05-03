package github

import (
	"strings"
	"testing"
)

func TestFormatGitHubWebhookAuthErrorExplainsIntegrationPermission(t *testing.T) {
	err := formatGitHubWebhookAuthError(403, []byte(`{"message":"Resource not accessible by integration"}`))
	message := strings.ToLower(err.Error())

	if !strings.Contains(message, "webhook permission denied") {
		t.Fatalf("expected webhook permission error, got %v", err)
	}
	if !strings.Contains(message, "admin access") {
		t.Fatalf("expected admin access guidance, got %v", err)
	}
}
