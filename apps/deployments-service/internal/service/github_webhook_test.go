package deployments

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"testing"
)

func TestVerifyGitHubWebhookSignature(t *testing.T) {
	body := []byte(`{"zen":"Keep it logically awesome."}`)
	secret := "top-secret"
	t.Setenv("GITHUB_WEBHOOK_SECRET", secret)

	mac := hmac.New(sha256.New, []byte(secret))
	_, _ = mac.Write(body)
	signature := "sha256=" + hex.EncodeToString(mac.Sum(nil))

	if err := verifyGitHubWebhookSignature(body, signature); err != nil {
		t.Fatalf("expected signature to verify: %v", err)
	}

	if err := verifyGitHubWebhookSignature(body, "sha256=deadbeef"); err == nil {
		t.Fatal("expected invalid signature to fail")
	}
}

func TestNormalizeGitHubRepoFullName(t *testing.T) {
	tests := map[string]string{
		"owner/repo":                        "owner/repo",
		"OWNER/Repo":                        "owner/repo",
		"https://github.com/owner/repo":     "owner/repo",
		"https://github.com/owner/repo.git": "owner/repo",
		"git@github.com:owner/repo.git":     "owner/repo",
		"https://example.com/owner/repo":    "",
	}

	for input, want := range tests {
		if got := normalizeGitHubRepoFullName(input); got != want {
			t.Fatalf("normalizeGitHubRepoFullName(%q) = %q, want %q", input, got, want)
		}
	}
}

func TestBranchFromGitHubRef(t *testing.T) {
	if got := branchFromGitHubRef("refs/heads/main"); got != "main" {
		t.Fatalf("expected main branch, got %q", got)
	}

	if got := branchFromGitHubRef("refs/tags/v1.0.0"); got != "" {
		t.Fatalf("expected tag ref to be ignored, got %q", got)
	}
}
