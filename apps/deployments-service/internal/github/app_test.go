package github

import (
	"strings"
	"testing"
)

func TestReadGitHubAppResponseBodyRejectsOversizedResponse(t *testing.T) {
	_, err := readGitHubAppResponseBody(strings.NewReader(strings.Repeat("a", githubAppResponseBodyLimit+1)))
	if err == nil {
		t.Fatal("expected oversized response to be rejected")
	}
}

func TestConfiguredGitHubAppIDRequiresPositiveInteger(t *testing.T) {
	for _, value := range []string{"", "not-a-number", "0", "-1"} {
		t.Run(value, func(t *testing.T) {
			t.Setenv("GITHUB_APP_ID", value)
			if _, err := configuredGitHubAppID(); err == nil {
				t.Fatalf("GITHUB_APP_ID %q should be rejected", value)
			}
		})
	}
	t.Setenv("GITHUB_APP_ID", "42")
	if got, err := configuredGitHubAppID(); err != nil || got != 42 {
		t.Fatalf("configuredGitHubAppID() = %d, %v", got, err)
	}
}
