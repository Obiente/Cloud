package deployments

import (
	"strings"
	"testing"
	"time"

	"github.com/obiente/cloud/apps/shared/pkg/database"
)

func TestGetUsableGitHubTokenUsesNonExpiringStoredToken(t *testing.T) {
	token, err := getUsableGitHubToken(&database.GitHubIntegration{
		Token: "gho_current",
	})
	if err != nil {
		t.Fatalf("expected token to be usable: %v", err)
	}
	if token != "gho_current" {
		t.Fatalf("expected stored token, got %q", token)
	}
}

func TestGetUsableGitHubTokenUsesStoredTokenWhenExpiryIsOutsideSkew(t *testing.T) {
	expiresAt := time.Now().Add(githubTokenRefreshSkew + time.Minute)

	token, err := getUsableGitHubToken(&database.GitHubIntegration{
		Token:          "gho_current",
		TokenExpiresAt: &expiresAt,
	})
	if err != nil {
		t.Fatalf("expected token to be usable: %v", err)
	}
	if token != "gho_current" {
		t.Fatalf("expected stored token, got %q", token)
	}
}

func TestGetUsableGitHubTokenRequiresRefreshTokenForExpiredToken(t *testing.T) {
	expiresAt := time.Now().Add(-time.Minute)

	_, err := getUsableGitHubToken(&database.GitHubIntegration{
		Token:          "gho_expired",
		TokenExpiresAt: &expiresAt,
	})
	if err == nil {
		t.Fatal("expected expired token without refresh token to fail")
	}
	if !strings.Contains(strings.ToLower(err.Error()), "reconnect") {
		t.Fatalf("expected reconnect error, got %v", err)
	}
}

func TestGetUsableGitHubTokenRequiresRefreshTokenForExpiringToken(t *testing.T) {
	expiresAt := time.Now().Add(githubTokenRefreshSkew - time.Minute)
	refreshToken := " "

	_, err := getUsableGitHubToken(&database.GitHubIntegration{
		Token:          "gho_expiring",
		RefreshToken:   &refreshToken,
		TokenExpiresAt: &expiresAt,
	})
	if err == nil {
		t.Fatal("expected expiring token without refresh token to fail")
	}
	if !strings.Contains(strings.ToLower(err.Error()), "reconnect") {
		t.Fatalf("expected reconnect error, got %v", err)
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
