package deployments

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/obiente/cloud/apps/shared/pkg/database"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
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

func TestIsGitHubCommitSHA(t *testing.T) {
	tests := map[string]bool{
		strings.Repeat("a", 40): true,
		strings.Repeat("B", 64): true,
		strings.Repeat("0", 40): false,
		"not-a-commit":          false,
		strings.Repeat("g", 40): false,
	}

	for input, want := range tests {
		if got := isGitHubCommitSHA(input); got != want {
			t.Fatalf("isGitHubCommitSHA(%q) = %t, want %t", input, got, want)
		}
	}
}

func TestHandleGitHubPushWebhookIgnoresDeletedBranch(t *testing.T) {
	payload := githubWebhookPushPayload{
		Ref:     "refs/heads/feature/retired",
		After:   strings.Repeat("0", 40),
		Deleted: true,
	}
	payload.Installation.ID = 42
	payload.Repository.FullName = "obiente/cloud"
	body, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("marshal push payload: %v", err)
	}

	recorder := httptest.NewRecorder()
	(&Service{}).handleGitHubPushWebhook(recorder, "push", body)

	if recorder.Code != http.StatusAccepted {
		t.Fatalf("status = %d, want %d; body: %s", recorder.Code, http.StatusAccepted, recorder.Body.String())
	}
	var response githubWebhookResponse
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if response.Message != "deleted branch ignored" || response.MatchedDeployments != 0 {
		t.Fatalf("unexpected response: %+v", response)
	}
}

func TestFindDeploymentsForGitHubPushBindsInstallationAndWorkspace(t *testing.T) {
	db := newGitHubWebhookTestDB(t)
	autoDeploy := true
	disabled := false
	repoURL := "https://github.com/Obiente/Cloud.git"
	integrationOne := "integration-one"
	integrationTwo := "integration-two"

	orgOne := "org-one"
	orgTwo := "org-two"
	installOne := int64(101)
	installTwo := int64(202)
	for _, integration := range []database.GitHubIntegration{
		{ID: integrationOne, OrganizationID: &orgOne, AuthType: "github_app", GitHubAppInstallationID: &installOne},
		{ID: integrationTwo, OrganizationID: &orgTwo, AuthType: "github_app", GitHubAppInstallationID: &installTwo},
	} {
		if err := db.Create(&integration).Error; err != nil {
			t.Fatalf("create integration %s: %v", integration.ID, err)
		}
	}

	deployments := []database.Deployment{
		{ID: "matching", OrganizationID: orgOne, RepositoryURL: &repoURL, Branch: "main", GitHubIntegrationID: &integrationOne, AutoDeploy: &autoDeploy},
		{ID: "wrong-installation", OrganizationID: orgTwo, RepositoryURL: &repoURL, Branch: "main", GitHubIntegrationID: &integrationTwo, AutoDeploy: &autoDeploy},
		{ID: "wrong-workspace", OrganizationID: orgTwo, RepositoryURL: &repoURL, Branch: "main", GitHubIntegrationID: &integrationOne, AutoDeploy: &autoDeploy},
		{ID: "disabled", OrganizationID: orgOne, RepositoryURL: &repoURL, Branch: "main", GitHubIntegrationID: &integrationOne, AutoDeploy: &disabled},
	}
	for i := range deployments {
		if err := db.Create(&deployments[i]).Error; err != nil {
			t.Fatalf("create deployment %s: %v", deployments[i].ID, err)
		}
	}

	matches, err := findDeploymentsForGitHubPush("obiente/cloud", "main", installOne)
	if err != nil {
		t.Fatalf("find matching deployments: %v", err)
	}
	if len(matches) != 1 || matches[0].ID != "matching" {
		t.Fatalf("matches = %+v, want only matching deployment", matches)
	}

	matches, err = findDeploymentsForGitHubPush("obiente/cloud", "main", 999)
	if err != nil {
		t.Fatalf("find unknown installation: %v", err)
	}
	if len(matches) != 0 {
		t.Fatalf("unknown installation matched deployments: %+v", matches)
	}
}

func TestEnsureGitHubWebhookRejectsAnotherWorkspaceIntegration(t *testing.T) {
	db := newGitHubWebhookTestDB(t)
	t.Setenv("GITHUB_WEBHOOK_SECRET", "test-secret")

	integrationID := "integration-other-workspace"
	integrationOrgID := "org-two"
	installationID := int64(202)
	integration := database.GitHubIntegration{
		ID:                      integrationID,
		OrganizationID:          &integrationOrgID,
		AuthType:                "github_app",
		GitHubAppInstallationID: &installationID,
	}
	if err := db.Create(&integration).Error; err != nil {
		t.Fatalf("create integration: %v", err)
	}

	repoURL := "https://github.com/obiente/cloud"
	deployment := &database.Deployment{
		ID:                  "deployment-one",
		OrganizationID:      "org-one",
		RepositoryURL:       &repoURL,
		GitHubIntegrationID: &integrationID,
	}

	err := (&Service{}).ensureGitHubWebhookForDeployment(t.Context(), deployment)
	if err == nil || !strings.Contains(err.Error(), "does not belong") {
		t.Fatalf("error = %v, want workspace ownership failure", err)
	}
}

func newGitHubWebhookTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	dsn := fmt.Sprintf("file:%s?mode=memory&cache=shared", strings.ReplaceAll(t.Name(), "/", "-"))
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite db: %v", err)
	}
	if err := db.AutoMigrate(&database.Deployment{}, &database.GitHubIntegration{}); err != nil {
		t.Fatalf("migrate sqlite db: %v", err)
	}

	previousDB := database.DB
	database.DB = db
	t.Cleanup(func() {
		database.DB = previousDB
	})
	return db
}
