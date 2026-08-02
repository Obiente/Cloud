package deployments

import (
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	githubclient "deployments-service/internal/github"
	"github.com/obiente/cloud/apps/shared/pkg/database"
	deploymentsv1 "github.com/obiente/cloud/apps/shared/proto/obiente/cloud/deployments/v1"
)

func TestPullRequestApprovalIsBoundToHeadSHA(t *testing.T) {
	approvedSHA := "1111111111111111111111111111111111111111"
	now := time.Now()
	record := &database.PullRequestDeployment{HeadSHA: approvedSHA, ApprovedHeadSHA: &approvedSHA, ApprovedAt: &now}
	config := &database.PullRequestDeploymentConfig{RequireApproval: true}
	if !pullRequestDeploymentApproved(record, config) {
		t.Fatal("current approved revision should be deployable")
	}
	record.HeadSHA = "2222222222222222222222222222222222222222"
	if pullRequestDeploymentApproved(record, config) {
		t.Fatal("a new revision must invalidate head-scoped approval")
	}
	config.ApprovalCoversUpdates = true
	if !pullRequestDeploymentApproved(record, config) {
		t.Fatal("explicit whole-PR approval should cover later revisions")
	}
}

func TestForkPreviewNeverReceivesTemplateValues(t *testing.T) {
	source := &database.Deployment{EnvVars: `{"PUBLIC":"yes","SECRET":"no"}`, BuildArgs: `{"MODE":"preview","TOKEN":"no"}`}
	config := &database.PullRequestDeploymentConfig{EnvironmentVariableNames: `["PUBLIC","SECRET"]`, BuildArgumentNames: `["MODE","TOKEN"]`}
	env, args := previewScopedVariables(source, config, true)
	if env != "{}" || args != "{}" {
		t.Fatalf("fork preview inherited values: env=%s args=%s", env, args)
	}
	env, args = previewScopedVariables(source, config, false)
	if env != `{"PUBLIC":"yes","SECRET":"no"}` || args != `{"MODE":"preview","TOKEN":"no"}` {
		t.Fatalf("trusted preview did not receive its explicit allowlist: env=%s args=%s", env, args)
	}
}

func TestPullRequestPathScope(t *testing.T) {
	config := &database.PullRequestDeploymentConfig{IncludePaths: `["apps/web/**"]`, ExcludePaths: `["apps/web/docs/**"]`}
	if !pullRequestFilesMatch([]githubclient.PullRequestFile{{Filename: "apps/web/src/main.ts"}}, config) {
		t.Fatal("matching application change should deploy")
	}
	if pullRequestFilesMatch([]githubclient.PullRequestFile{{Filename: "apps/web/docs/readme.md"}, {Filename: "services/api/main.go"}}, config) {
		t.Fatal("excluded or unrelated changes should not deploy")
	}
}

func TestPullRequestDomainTemplateValidation(t *testing.T) {
	if err := validatePRDomainTemplate("pr-{pr}-{deployment}"); err != nil {
		t.Fatalf("valid template rejected: %v", err)
	}
	for _, invalid := range []string{"preview", "{pr}.example.com", "PR-{pr}", "-{pr}"} {
		if err := validatePRDomainTemplate(invalid); err == nil {
			t.Fatalf("invalid template %q accepted", invalid)
		}
	}
}

func TestForkApprovalPolicy(t *testing.T) {
	record := &database.PullRequestDeployment{FromFork: true}
	config := &database.PullRequestDeploymentConfig{ForkPolicy: int32(deploymentsv1.PullRequestForkPolicy_PULL_REQUEST_FORK_REQUIRE_APPROVAL)}
	if !requiresPullRequestApproval(record, config) {
		t.Fatal("fork policy should require approval")
	}
}

func TestMergedPreviewRestoreRequiresRecordedApproval(t *testing.T) {
	now := time.Now()
	sha := strings.Repeat("a", 40)
	record := &database.PullRequestDeployment{Merged: true, ClosedAt: &now, ApprovedAt: &now, ApprovedHeadSHA: &sha, HeadSHA: sha}
	config := &database.PullRequestDeploymentConfig{}
	if !pullRequestDeploymentCanRestore(record, config) {
		t.Fatal("approved merged revision should be restorable")
	}
	record.HeadSHA = strings.Repeat("b", 40)
	if pullRequestDeploymentCanRestore(record, config) {
		t.Fatal("a different unapproved revision must not be restorable")
	}
	config.ApprovalCoversUpdates = true
	if !pullRequestDeploymentCanRestore(record, config) {
		t.Fatal("explicit whole-PR approval should permit the recorded merged revision")
	}
	record.Merged = false
	if pullRequestDeploymentCanRestore(record, config) {
		t.Fatal("a closed unmerged pull request must not be restorable")
	}
}

func TestPreviewStateCopyDoesNotExposeStoredErrors(t *testing.T) {
	privateError := "clone failed with token secret-value"
	record := &database.PullRequestDeployment{Status: int32(deploymentsv1.PullRequestDeploymentStatus_PULL_REQUEST_DEPLOYMENT_FAILED), Error: &privateError}
	title, detail, transient := previewStateCopy(record)
	if transient || title == "" || detail == "" {
		t.Fatal("failed state should be a stable, useful page")
	}
	if strings.Contains(title+detail, privateError) || strings.Contains(title+detail, "secret-value") {
		t.Fatal("public state copy exposed the stored build error")
	}
}

func TestPreviewStateURLUsesExplicitOriginOrProductionDomain(t *testing.T) {
	record := &database.PullRequestDeployment{ID: "pr-deployment-test-id"}
	t.Setenv("PREVIEW_STATUS_BASE_URL", "https://preview-state.example.test/")
	if got := stringValue(pullRequestDeploymentStateURL(record)); got != "https://preview-state.example.test/previews/pr-deployment-test-id" {
		t.Fatalf("unexpected explicit state URL: %q", got)
	}
	t.Setenv("PREVIEW_STATUS_BASE_URL", "")
	t.Setenv("DOMAIN", "obiente.cloud")
	if got := stringValue(pullRequestDeploymentStateURL(record)); got != "https://obiente.cloud/previews/pr-deployment-test-id" {
		t.Fatalf("unexpected derived state URL: %q", got)
	}
	t.Setenv("PREVIEW_STATUS_BASE_URL", "javascript:alert(1)")
	if got := pullRequestDeploymentStateURL(record); got != nil {
		t.Fatalf("unsafe state URL should be rejected: %q", *got)
	}
}

func TestPreviewStateSecurityHeaders(t *testing.T) {
	recorder := httptest.NewRecorder()
	setPreviewStateHeaders(recorder)
	for header, want := range map[string]string{
		"Cache-Control":          "no-store",
		"Referrer-Policy":        "no-referrer",
		"X-Content-Type-Options": "nosniff",
		"X-Frame-Options":        "DENY",
		"X-Robots-Tag":           "noindex, nofollow, noarchive",
	} {
		if got := recorder.Header().Get(header); got != want {
			t.Errorf("%s = %q, want %q", header, got, want)
		}
	}
	if csp := recorder.Header().Get("Content-Security-Policy"); !strings.Contains(csp, "frame-ancestors 'none'") || !strings.Contains(csp, "default-src 'none'") {
		t.Fatalf("unexpected content security policy: %q", csp)
	}
}
