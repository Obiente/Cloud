package deployments

import (
	"strings"
	"testing"

	"github.com/obiente/cloud/apps/shared/pkg/database"
)

func TestGetUsableGitHubTokenRejectsLegacyIntegration(t *testing.T) {
	_, err := getUsableGitHubToken(&database.GitHubIntegration{
		AuthType: "oauth",
		Token:    "gho_current",
	})
	if err == nil {
		t.Fatal("expected legacy integration to fail")
	}
	if !strings.Contains(strings.ToLower(err.Error()), "github app") {
		t.Fatalf("expected GitHub App reinstall guidance, got %v", err)
	}
}

func TestGetUsableGitHubTokenRequiresAppInstallationID(t *testing.T) {
	_, err := getUsableGitHubToken(&database.GitHubIntegration{
		AuthType: "github_app",
	})
	if err == nil {
		t.Fatal("expected missing installation ID to fail")
	}
	if !strings.Contains(strings.ToLower(err.Error()), "installation id") {
		t.Fatalf("expected installation ID error, got %v", err)
	}
}

func TestGetUsableGitHubTokenRejectsNilIntegration(t *testing.T) {
	_, err := getUsableGitHubToken(nil)
	if err == nil {
		t.Fatal("expected nil integration to fail")
	}
}

func TestIsMissingGitHubIntegrationError(t *testing.T) {
	if !isMissingGitHubIntegrationError(assertErr("no GitHub integration found for user or organization")) {
		t.Fatal("expected missing integration error to match")
	}
	if isMissingGitHubIntegrationError(assertErr("GitHub token expired or could not be refreshed")) {
		t.Fatal("expected token refresh error not to match missing integration")
	}
}

type assertErr string

func (e assertErr) Error() string {
	return string(e)
}
