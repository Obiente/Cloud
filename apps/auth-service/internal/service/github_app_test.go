package auth

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

const testPKCEVerifier = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789-._~"

func TestConfiguredGitHubAppCallbackURL(t *testing.T) {
	t.Setenv("DASHBOARD_URL", "https://obiente.cloud/settings?tab=integrations#github")

	got, err := configuredGitHubAppCallbackURL()
	if err != nil {
		t.Fatalf("configured callback URL: %v", err)
	}
	if got != "https://obiente.cloud/api/github/app/callback" {
		t.Fatalf("callback URL = %q", got)
	}
}

func TestConfiguredGitHubAppCallbackURLRejectsInvalidOrigin(t *testing.T) {
	t.Setenv("DASHBOARD_URL", "javascript:alert(1)")

	if _, err := configuredGitHubAppCallbackURL(); err == nil {
		t.Fatal("expected invalid DASHBOARD_URL to be rejected")
	}
}

func TestValidPKCECodeVerifier(t *testing.T) {
	if !isValidPKCECodeVerifier(testPKCEVerifier) {
		t.Fatal("expected RFC 7636 verifier to be accepted")
	}
	for _, value := range []string{
		"too-short",
		strings.Repeat("a", 129),
		strings.Repeat("a", 42) + "+",
	} {
		if isValidPKCECodeVerifier(value) {
			t.Fatalf("invalid verifier %q was accepted", value)
		}
	}
}

func TestExchangeGitHubAppUserCodeUsesPKCEAndCanonicalRedirect(t *testing.T) {
	t.Setenv("GITHUB_APP_CLIENT_ID", "client-id")
	t.Setenv("GITHUB_APP_CLIENT_SECRET", "client-secret")
	t.Setenv("DASHBOARD_URL", "https://obiente.cloud")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		if req.Method != http.MethodPost {
			t.Errorf("method = %s", req.Method)
		}
		if err := req.ParseForm(); err != nil {
			t.Errorf("parse form: %v", err)
		}
		want := url.Values{
			"client_id":     {"client-id"},
			"client_secret": {"client-secret"},
			"code":          {"one-time-code"},
			"code_verifier": {testPKCEVerifier},
			"redirect_uri":  {"https://obiente.cloud/api/github/app/callback"},
		}
		if req.Form.Encode() != want.Encode() {
			t.Errorf("form = %q, want %q", req.Form.Encode(), want.Encode())
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"access_token":"ghu_test"}`)
	}))
	defer server.Close()

	previousClient := githubAppHTTPClient
	previousURL := githubAppOAuthAccessTokenURL
	githubAppHTTPClient = server.Client()
	githubAppOAuthAccessTokenURL = server.URL
	t.Cleanup(func() {
		githubAppHTTPClient = previousClient
		githubAppOAuthAccessTokenURL = previousURL
	})

	token, err := exchangeGitHubAppUserCode(
		context.Background(),
		"one-time-code",
		testPKCEVerifier,
		"https://obiente.cloud/api/github/app/callback",
	)
	if err != nil {
		t.Fatalf("exchange code: %v", err)
	}
	if token != "ghu_test" {
		t.Fatalf("token = %q", token)
	}
}

func TestExchangeGitHubAppUserCodeRejectsRedirectMismatch(t *testing.T) {
	t.Setenv("GITHUB_APP_CLIENT_ID", "client-id")
	t.Setenv("GITHUB_APP_CLIENT_SECRET", "client-secret")
	t.Setenv("DASHBOARD_URL", "https://obiente.cloud")

	_, err := exchangeGitHubAppUserCode(
		context.Background(),
		"one-time-code",
		testPKCEVerifier,
		"https://attacker.example/api/github/app/callback",
	)
	if err == nil || !strings.Contains(err.Error(), "does not match") {
		t.Fatalf("redirect mismatch error = %v", err)
	}
}

func TestUserCanAccessGitHubInstallationUsesDirectLookup(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		if req.URL.Path != "/user/installations/42/repositories" {
			t.Errorf("path = %q", req.URL.Path)
		}
		if req.URL.Query().Get("per_page") != "1" {
			t.Errorf("per_page = %q", req.URL.Query().Get("per_page"))
		}
		if req.Header.Get("Authorization") != "Bearer ghu_test" {
			t.Errorf("authorization header = %q", req.Header.Get("Authorization"))
		}
		w.WriteHeader(http.StatusOK)
		_, _ = io.WriteString(w, `{}`)
	}))
	defer server.Close()

	previousClient := githubAppHTTPClient
	previousBaseURL := githubAppAPIBaseURL
	githubAppHTTPClient = server.Client()
	githubAppAPIBaseURL = server.URL
	t.Cleanup(func() {
		githubAppHTTPClient = previousClient
		githubAppAPIBaseURL = previousBaseURL
	})

	found, err := userCanAccessGitHubInstallation(context.Background(), "ghu_test", 42)
	if err != nil {
		t.Fatalf("lookup installation: %v", err)
	}
	if !found {
		t.Fatal("expected installation to be visible to the user")
	}
}

func TestReadGitHubAppResponseBodyRejectsOversizedResponse(t *testing.T) {
	_, err := readGitHubAppResponseBody(
		strings.NewReader(strings.Repeat("a", githubAppResponseBodyLimit+1)),
	)
	if err == nil {
		t.Fatal("expected oversized response to be rejected")
	}
}
