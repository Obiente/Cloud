package deployments

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"reflect"
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

func TestPullRequestPreviewIdentityUsesProtectedEnvironment(t *testing.T) {
	preview := &database.Deployment{Environment: int32(deploymentsv1.Environment_PULL_REQUEST), Groups: `[]`}
	if !deploymentIsPullRequestPreview(preview) {
		t.Fatal("system-managed PR environment was not treated as an isolated preview")
	}
	preview.Environment = int32(deploymentsv1.Environment_PRODUCTION)
	preview.Groups = `["pull-request"]`
	if deploymentIsPullRequestPreview(preview) {
		t.Fatal("mutable deployment group granted pull request preview isolation identity")
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

func TestStalePullRequestWebhookDoesNotOverrideCurrentGitHubState(t *testing.T) {
	live := &githubclient.PullRequest{State: "closed"}
	live.Head.SHA = strings.Repeat("b", 40)
	if pullRequestWebhookMatchesCurrentState("synchronize", strings.Repeat("a", 40), live) {
		t.Fatal("an older synchronize event must not replace the current revision")
	}
	if pullRequestWebhookMatchesCurrentState("opened", live.Head.SHA, live) {
		t.Fatal("an open event must not reopen a pull request that GitHub reports closed")
	}
	if !pullRequestWebhookMatchesCurrentState("closed", live.Head.SHA, live) {
		t.Fatal("the current close event should be accepted")
	}
}

func TestEditedPullRequestWebhookReevaluatesBaseBranchScope(t *testing.T) {
	if !supportedPullRequestAction("edited") {
		t.Fatal("retargeting a pull request must re-evaluate preview scope")
	}
}

func TestUnchangedPullRequestWebhookPreservesExistingRuntimeState(t *testing.T) {
	for _, status := range []deploymentsv1.PullRequestDeploymentStatus{
		deploymentsv1.PullRequestDeploymentStatus_PULL_REQUEST_DEPLOYMENT_QUEUED,
		deploymentsv1.PullRequestDeploymentStatus_PULL_REQUEST_DEPLOYMENT_BUILDING,
		deploymentsv1.PullRequestDeploymentStatus_PULL_REQUEST_DEPLOYMENT_RUNNING,
		deploymentsv1.PullRequestDeploymentStatus_PULL_REQUEST_DEPLOYMENT_FAILED,
	} {
		if !preservePullRequestStateForUnchangedRevision(int32(status)) {
			t.Fatalf("unchanged revision should preserve %s", status)
		}
	}
	for _, status := range []deploymentsv1.PullRequestDeploymentStatus{
		deploymentsv1.PullRequestDeploymentStatus_PULL_REQUEST_DEPLOYMENT_SKIPPED,
		deploymentsv1.PullRequestDeploymentStatus_PULL_REQUEST_DEPLOYMENT_WAITING_APPROVAL,
		deploymentsv1.PullRequestDeploymentStatus_PULL_REQUEST_DEPLOYMENT_REJECTED,
		deploymentsv1.PullRequestDeploymentStatus_PULL_REQUEST_DEPLOYMENT_CLOSED,
	} {
		if preservePullRequestStateForUnchangedRevision(int32(status)) {
			t.Fatalf("eligibility transition should not preserve %s", status)
		}
	}
}

func TestIgnoredPushRevisionIsNotBuiltByLaterStateOnlyEvent(t *testing.T) {
	ignored := strings.Repeat("b", 40)
	record := &database.PullRequestDeployment{
		HeadSHA:        strings.Repeat("a", 40),
		IgnoredHeadSHA: &ignored,
		Status:         int32(deploymentsv1.PullRequestDeploymentStatus_PULL_REQUEST_DEPLOYMENT_RUNNING),
	}
	config := &database.PullRequestDeploymentConfig{RedeployOnPush: false}
	if !preserveIgnoredPullRequestRevision(record, nil, config, ignored, "") {
		t.Fatal("a later state-only event must not deploy a revision ignored by RedeployOnPush=false")
	}
	if preserveIgnoredPullRequestRevision(record, nil, config, ignored, "the pull request is out of scope") {
		t.Fatal("an eligibility change must still be allowed to remove an ignored revision's runtime")
	}
	record.Status = int32(deploymentsv1.PullRequestDeploymentStatus_PULL_REQUEST_DEPLOYMENT_SKIPPED)
	if preserveIgnoredPullRequestRevision(record, nil, config, ignored, "") {
		t.Fatal("a newly eligible skipped preview must be allowed to deploy the current revision")
	}
}

func TestApprovalMutationRejectsAStaleReviewedHead(t *testing.T) {
	db := newDeploymentServiceTestDB(t)
	oldHead, newHead := strings.Repeat("a", 40), strings.Repeat("b", 40)
	record := database.PullRequestDeployment{
		ID: "pr-stale-maintainer-action", SourceDeploymentID: "source", OrganizationID: "org",
		Repository: "obiente/cloud", PullRequestNumber: 31, HeadSHA: newHead, HeadRef: "feature", BaseRef: "main",
		Status: int32(deploymentsv1.PullRequestDeploymentStatus_PULL_REQUEST_DEPLOYMENT_WAITING_APPROVAL), ExpiresAt: time.Now().Add(time.Hour),
	}
	if err := db.Create(&record).Error; err != nil {
		t.Fatalf("seed pull request deployment: %v", err)
	}
	stale := record
	stale.HeadSHA = oldHead
	result := pullRequestApprovalMutation(t.Context(), &stale).
		Where("status = ?", int32(deploymentsv1.PullRequestDeploymentStatus_PULL_REQUEST_DEPLOYMENT_WAITING_APPROVAL)).
		Update("status", int32(deploymentsv1.PullRequestDeploymentStatus_PULL_REQUEST_DEPLOYMENT_REJECTED))
	if result.Error != nil {
		t.Fatalf("apply stale approval mutation: %v", result.Error)
	}
	if result.RowsAffected != 0 {
		t.Fatal("a maintainer action for an older reviewed head changed the current revision")
	}
}

func TestPullRequestConfigReconciliationTracksRuntimePolicies(t *testing.T) {
	previous := &database.PullRequestDeploymentConfig{Enabled: true, ForkPolicy: int32(deploymentsv1.PullRequestForkPolicy_PULL_REQUEST_FORK_ISOLATED), BaseBranches: `["main"]`, CommentEnabled: true}
	current := *previous
	current.CommentEnabled = false
	if pullRequestConfigRequiresReconciliation(previous, &current) {
		t.Fatal("comment-only settings should not restart active previews")
	}
	current.ForkPolicy = int32(deploymentsv1.PullRequestForkPolicy_PULL_REQUEST_FORK_DENY)
	if !pullRequestConfigRequiresReconciliation(previous, &current) {
		t.Fatal("tightening the fork policy must reconcile active previews")
	}
	current = *previous
	current.Enabled = false
	if !pullRequestConfigRequiresReconciliation(previous, &current) {
		t.Fatal("disabling previews must reconcile active runtimes")
	}
}

func TestStaleReconciliationCannotCompleteOrRescheduleNewGeneration(t *testing.T) {
	db := newDeploymentServiceTestDB(t)
	config := database.PullRequestDeploymentConfig{
		DeploymentID: "source-generation", OrganizationID: "org", ReconciliationPending: true,
		ReconciliationGeneration: 2, BaseBranches: `[]`, IncludePaths: `[]`, ExcludePaths: `[]`,
		EnvironmentVariableNames: `[]`, BuildArgumentNames: `[]`, DomainTemplate: defaultPRDomainTemplate,
	}
	if err := db.Create(&config).Error; err != nil {
		t.Fatalf("seed pull request config: %v", err)
	}
	stale := config
	stale.ReconciliationGeneration = 1
	if completed, err := completePullRequestReconciliation(t.Context(), &stale, nil); err != nil {
		t.Fatalf("complete stale reconciliation: %v", err)
	} else if completed {
		t.Fatal("stale reconciliation completed a newer settings generation")
	}
	if err := schedulePullRequestReconciliationRetry(t.Context(), &stale); err != nil {
		t.Fatalf("schedule stale reconciliation retry: %v", err)
	}
	var got database.PullRequestDeploymentConfig
	if err := db.First(&got, "deployment_id = ?", config.DeploymentID).Error; err != nil {
		t.Fatalf("reload pull request config: %v", err)
	}
	if !got.ReconciliationPending || got.ReconciliationAttempts != 0 || got.ReconciliationGeneration != 2 {
		t.Fatalf("stale worker changed the current settings generation: %#v", got)
	}
}

func TestStaleReconciliationCannotDetachExistingRuntime(t *testing.T) {
	db := newDeploymentServiceTestDB(t)
	now := time.Now()
	config := database.PullRequestDeploymentConfig{
		DeploymentID: "source-runtime-generation", OrganizationID: "org", Enabled: true, ReconciliationPending: true,
		ReconciliationGeneration: 2, BaseBranches: `[]`, IncludePaths: `[]`, ExcludePaths: `[]`,
		EnvironmentVariableNames: `[]`, BuildArgumentNames: `[]`, DomainTemplate: defaultPRDomainTemplate,
	}
	previewID := "preview-runtime-generation"
	record := database.PullRequestDeployment{
		ID: "pr-runtime-generation", SourceDeploymentID: config.DeploymentID, PreviewDeploymentID: &previewID, OrganizationID: "org",
		Repository: "obiente/cloud", PullRequestNumber: 7, HeadSHA: strings.Repeat("a", 40), HeadRef: "feature", BaseRef: "main",
		Status: int32(deploymentsv1.PullRequestDeploymentStatus_PULL_REQUEST_DEPLOYMENT_RUNNING), ExpiresAt: now.Add(time.Hour),
	}
	if err := db.Create(&config).Error; err != nil {
		t.Fatalf("seed pull request config: %v", err)
	}
	if err := db.Create(&record).Error; err != nil {
		t.Fatalf("seed pull request deployment: %v", err)
	}

	stale := config
	stale.ReconciliationGeneration = 1
	if current, err := schedulePullRequestRuntimeReconciliation(t.Context(), &stale); err != nil {
		t.Fatalf("schedule stale runtime reconciliation: %v", err)
	} else if current {
		t.Fatal("stale settings generation was accepted for runtime reconciliation")
	}
	var unchanged database.PullRequestDeployment
	if err := db.First(&unchanged, "id = ?", record.ID).Error; err != nil {
		t.Fatalf("reload pull request deployment: %v", err)
	}
	if !unchanged.ExpiresAt.After(now) || unchanged.Error != nil {
		t.Fatalf("stale settings generation detached the current runtime: %#v", unchanged)
	}

	if current, err := schedulePullRequestRuntimeReconciliation(t.Context(), &config); err != nil {
		t.Fatalf("schedule current runtime reconciliation: %v", err)
	} else if !current {
		t.Fatal("current settings generation was rejected")
	}
	var scheduled database.PullRequestDeployment
	if err := db.First(&scheduled, "id = ?", record.ID).Error; err != nil {
		t.Fatalf("reload scheduled pull request deployment: %v", err)
	}
	if scheduled.ExpiresAt.After(time.Now()) || stringValue(scheduled.Error) != "Preview settings reconciliation is pending." {
		t.Fatalf("current settings generation did not schedule runtime reconciliation: %#v", scheduled)
	}
}

func TestOpenPullRequestSyncMarkerRequiresUnchangedSource(t *testing.T) {
	db := newDeploymentServiceTestDB(t)
	repository, replacement := "https://github.com/obiente/cloud", "https://github.com/obiente/website"
	integrationID, installationID := "integration-sync", int64(42)
	autoDeploy := false
	source := testDeployment("source-sync", "Source", "org", "owner", time.Now(), &autoDeploy)
	source.RepositoryURL, source.GitHubIntegrationID = &repository, &integrationID
	integration := database.GitHubIntegration{ID: integrationID, GitHubAppInstallationID: &installationID}
	config := database.PullRequestDeploymentConfig{
		DeploymentID: source.ID, OrganizationID: source.OrganizationID, Enabled: true, ReconciliationGeneration: 1,
		BaseBranches: `[]`, IncludePaths: `[]`, ExcludePaths: `[]`, EnvironmentVariableNames: `[]`, BuildArgumentNames: `[]`, DomainTemplate: defaultPRDomainTemplate,
	}
	if err := db.Create(&integration).Error; err != nil {
		t.Fatalf("seed GitHub integration: %v", err)
	}
	if err := db.Create(source).Error; err != nil {
		t.Fatalf("seed source deployment: %v", err)
	}
	if err := db.Create(&config).Error; err != nil {
		t.Fatalf("seed pull request config: %v", err)
	}
	snapshot := &pullRequestSyncSource{repositoryURL: repository, integrationID: integrationID, installationID: installationID}
	if err := db.Model(&database.Deployment{}).Where("id = ?", source.ID).Update("repository_url", replacement).Error; err != nil {
		t.Fatalf("change source repository: %v", err)
	}
	if marked, err := markOpenPullRequestsSynced(t.Context(), &config, snapshot); err != nil {
		t.Fatalf("mark open pull requests synced: %v", err)
	} else if marked {
		t.Fatal("old repository scan marked the replacement repository as synchronized")
	}
	var got database.PullRequestDeploymentConfig
	if err := db.First(&got, "deployment_id = ?", config.DeploymentID).Error; err != nil {
		t.Fatalf("reload pull request config: %v", err)
	}
	if got.OpenPullRequestsSyncedAt != nil {
		t.Fatal("source-changing race persisted an invalid sync marker")
	}
}

func TestActivePreviewCountExcludesFailedBuildWithoutRuntime(t *testing.T) {
	db := newDeploymentServiceTestDB(t)
	if err := db.AutoMigrate(&database.DeploymentLocation{}); err != nil {
		t.Fatalf("migrate deployment locations: %v", err)
	}
	failedPreview, livePreview, buildingPreview := "preview-failed", "preview-live", "preview-building"
	records := []database.PullRequestDeployment{
		{ID: "pr-failed", SourceDeploymentID: "source-count", PreviewDeploymentID: &failedPreview, OrganizationID: "org", Repository: "obiente/cloud", PullRequestNumber: 1, HeadSHA: strings.Repeat("a", 40), HeadRef: "failed", BaseRef: "main", Status: int32(deploymentsv1.PullRequestDeploymentStatus_PULL_REQUEST_DEPLOYMENT_FAILED), ExpiresAt: time.Now().Add(time.Hour)},
		{ID: "pr-live", SourceDeploymentID: "source-count", PreviewDeploymentID: &livePreview, OrganizationID: "org", Repository: "obiente/cloud", PullRequestNumber: 2, HeadSHA: strings.Repeat("b", 40), HeadRef: "live", BaseRef: "main", Status: int32(deploymentsv1.PullRequestDeploymentStatus_PULL_REQUEST_DEPLOYMENT_FAILED), ExpiresAt: time.Now().Add(time.Hour)},
		{ID: "pr-building", SourceDeploymentID: "source-count", PreviewDeploymentID: &buildingPreview, OrganizationID: "org", Repository: "obiente/cloud", PullRequestNumber: 3, HeadSHA: strings.Repeat("c", 40), HeadRef: "building", BaseRef: "main", Status: int32(deploymentsv1.PullRequestDeploymentStatus_PULL_REQUEST_DEPLOYMENT_BUILDING), ExpiresAt: time.Now().Add(time.Hour)},
	}
	if err := db.Create(&records).Error; err != nil {
		t.Fatalf("seed pull request deployments: %v", err)
	}
	location := database.DeploymentLocation{ID: "location-live", DeploymentID: livePreview, NodeID: "node", ContainerID: "container-live", Status: "running"}
	if err := db.Create(&location).Error; err != nil {
		t.Fatalf("seed live deployment location: %v", err)
	}
	active, err := countActivePullRequestPreviews(db, "source-count", "new-record")
	if err != nil {
		t.Fatalf("count active pull request previews: %v", err)
	}
	if active != 2 {
		t.Fatalf("active pull request previews = %d, want live failed revision plus in-progress build", active)
	}
}

func TestDisablingPullRequestConfigDetachesExistingRuntime(t *testing.T) {
	db := newDeploymentServiceTestDB(t)
	background, cancel := context.WithCancel(context.Background())
	cancel()
	service := NewService(background, database.NewDeploymentRepository(db, nil), nil, nil)
	now := time.Now()
	autoDeploy := false
	preview := testDeployment("preview-disable", "Preview", "org", "system", now, &autoDeploy)
	preview.Groups = `["pull-request"]`
	preview.Environment = int32(deploymentsv1.Environment_PULL_REQUEST)
	if err := db.Create(preview).Error; err != nil {
		t.Fatalf("seed preview deployment: %v", err)
	}
	head := strings.Repeat("a", 40)
	record := database.PullRequestDeployment{
		ID: "pr-disable", SourceDeploymentID: "source-disable", PreviewDeploymentID: &preview.ID, OrganizationID: "org",
		GitHubIntegrationID: "integration", GitHubInstallationID: 42, Repository: "obiente/cloud", PullRequestNumber: 31,
		HeadSHA: head, ActiveHeadSHA: &head, HeadRef: "feature", BaseRef: "main",
		Status: int32(deploymentsv1.PullRequestDeploymentStatus_PULL_REQUEST_DEPLOYMENT_RUNNING), ExpiresAt: now.Add(time.Hour),
	}
	if err := db.Create(&record).Error; err != nil {
		t.Fatalf("seed pull request deployment: %v", err)
	}
	source := &database.Deployment{ID: record.SourceDeploymentID, OrganizationID: record.OrganizationID}
	if err := service.detachPullRequestRuntimeForReconciliation(t.Context(), &record, source, false, "", 0); err != nil {
		t.Fatalf("detach disabled preview: %v", err)
	}
	var got database.PullRequestDeployment
	if err := db.First(&got, "id = ?", record.ID).Error; err != nil {
		t.Fatalf("reload pull request deployment: %v", err)
	}
	if got.PreviewDeploymentID != nil || got.ActiveHeadSHA != nil || got.Status != int32(deploymentsv1.PullRequestDeploymentStatus_PULL_REQUEST_DEPLOYMENT_SKIPPED) {
		t.Fatalf("disabled preview remained active: %#v", got)
	}
	var deletedPreview database.Deployment
	if err := db.Unscoped().First(&deletedPreview, "id = ?", preview.ID).Error; err != nil {
		t.Fatalf("reload deleted preview: %v", err)
	}
	if deletedPreview.DeletedAt == nil {
		t.Fatal("disabled preview runtime was not deleted")
	}
}

func TestStaleRuntimeCallbackQueuesCurrentHeadWithoutOverwritingState(t *testing.T) {
	db := newDeploymentServiceTestDB(t)
	background, cancel := context.WithCancel(context.Background())
	cancel()
	service := NewService(background, database.NewDeploymentRepository(db, nil), nil, nil)
	oldHead, newHead := strings.Repeat("a", 40), strings.Repeat("b", 40)
	deploymentID, checkID := int64(71), int64(72)
	approvedAt := time.Now()
	record := database.PullRequestDeployment{
		ID: "pr-runtime-race", SourceDeploymentID: "source", PreviewDeploymentID: stringPointer("preview"), OrganizationID: "org",
		GitHubIntegrationID: "integration", GitHubInstallationID: 42, Repository: "obiente/cloud", PullRequestNumber: 31,
		HeadSHA: newHead, ActiveHeadSHA: &oldHead, HeadRef: "feature", BaseRef: "main",
		Status: int32(deploymentsv1.PullRequestDeploymentStatus_PULL_REQUEST_DEPLOYMENT_BUILDING), GitHubDeploymentID: &deploymentID,
		GitHubCheckRunID: &checkID, ApprovedHeadSHA: &newHead, ApprovedAt: &approvedAt, ExpiresAt: time.Now().Add(time.Hour),
	}
	if err := db.Create(&record).Error; err != nil {
		t.Fatalf("seed pull request deployment: %v", err)
	}

	service.updatePullRequestDeploymentRuntime(t.Context(), "preview", oldHead, deploymentsv1.PullRequestDeploymentStatus_PULL_REQUEST_DEPLOYMENT_RUNNING, "")
	var got database.PullRequestDeployment
	if err := db.First(&got, "id = ?", record.ID).Error; err != nil {
		t.Fatalf("reload pull request deployment: %v", err)
	}
	if got.HeadSHA != newHead || got.ActiveHeadSHA != nil || got.Status != int32(deploymentsv1.PullRequestDeploymentStatus_PULL_REQUEST_DEPLOYMENT_QUEUED) {
		t.Fatalf("stale callback corrupted current runtime state: %#v", got)
	}
	if got.GitHubDeploymentID == nil || *got.GitHubDeploymentID != deploymentID || got.GitHubCheckRunID == nil || *got.GitHubCheckRunID != checkID || got.ApprovedHeadSHA == nil || *got.ApprovedHeadSHA != newHead {
		t.Fatalf("stale callback overwrote unrelated current-revision fields: %#v", got)
	}
}

func TestSuccessfulPreviewRuntimeRecordsIsolationMigration(t *testing.T) {
	db := newDeploymentServiceTestDB(t)
	background, cancel := context.WithCancel(context.Background())
	cancel()
	service := NewService(background, database.NewDeploymentRepository(db, nil), nil, nil)
	head := strings.Repeat("a", 40)
	record := database.PullRequestDeployment{
		ID: "pr-isolation-version", SourceDeploymentID: "source", PreviewDeploymentID: stringPointer("preview-isolation"), OrganizationID: "org",
		GitHubIntegrationID: "integration", GitHubInstallationID: 42, Repository: "obiente/cloud", PullRequestNumber: 32,
		HeadSHA: head, ActiveHeadSHA: &head, HeadRef: "feature", BaseRef: "main",
		Status: int32(deploymentsv1.PullRequestDeploymentStatus_PULL_REQUEST_DEPLOYMENT_BUILDING), IsolationVersion: pendingPRIsolationVersion, ExpiresAt: time.Now().Add(time.Hour),
	}
	if err := db.Create(&record).Error; err != nil {
		t.Fatalf("seed pull request deployment: %v", err)
	}
	service.updatePullRequestDeploymentRuntime(t.Context(), "preview-isolation", head, deploymentsv1.PullRequestDeploymentStatus_PULL_REQUEST_DEPLOYMENT_RUNNING, "")
	var got database.PullRequestDeployment
	if err := db.First(&got, "id = ?", record.ID).Error; err != nil {
		t.Fatalf("reload pull request deployment: %v", err)
	}
	if got.IsolationVersion != currentPRIsolationVersion || got.Status != int32(deploymentsv1.PullRequestDeploymentStatus_PULL_REQUEST_DEPLOYMENT_RUNNING) {
		t.Fatalf("successful preview did not record current isolation version: %#v", got)
	}
}

func TestIsolationMigrationRetriesOnlyMarkedFailedPreviewsAfterDelay(t *testing.T) {
	db := newDeploymentServiceTestDB(t)
	now := time.Now().UTC()
	config := defaultPullRequestDeploymentConfig("source-isolation", "org")
	config.Enabled = true
	if err := db.Create(config).Error; err != nil {
		t.Fatalf("seed pull request config: %v", err)
	}

	type candidateSeed struct {
		id               string
		status           deploymentsv1.PullRequestDeploymentStatus
		isolationVersion int32
		updatedAt        time.Time
	}
	seeds := []candidateSeed{
		{id: "retry-failed", status: deploymentsv1.PullRequestDeploymentStatus_PULL_REQUEST_DEPLOYMENT_FAILED, isolationVersion: pendingPRIsolationVersion, updatedAt: now.Add(-10 * time.Minute)},
		{id: "fresh-failed", status: deploymentsv1.PullRequestDeploymentStatus_PULL_REQUEST_DEPLOYMENT_FAILED, isolationVersion: pendingPRIsolationVersion, updatedAt: now},
		{id: "legacy-running", status: deploymentsv1.PullRequestDeploymentStatus_PULL_REQUEST_DEPLOYMENT_RUNNING, isolationVersion: 0, updatedAt: now},
		{id: "previous-running", status: deploymentsv1.PullRequestDeploymentStatus_PULL_REQUEST_DEPLOYMENT_RUNNING, isolationVersion: currentPRIsolationVersion - 1, updatedAt: now},
		{id: "current-running", status: deploymentsv1.PullRequestDeploymentStatus_PULL_REQUEST_DEPLOYMENT_RUNNING, isolationVersion: currentPRIsolationVersion, updatedAt: now},
		{id: "ordinary-failed", status: deploymentsv1.PullRequestDeploymentStatus_PULL_REQUEST_DEPLOYMENT_FAILED, isolationVersion: 0, updatedAt: now.Add(-10 * time.Minute)},
	}
	for index, seed := range seeds {
		previewID := "preview-" + seed.id
		preview := testDeployment(previewID, previewID, "org", "system", now, nil)
		preview.Environment = int32(deploymentsv1.Environment_PULL_REQUEST)
		if err := db.Create(preview).Error; err != nil {
			t.Fatalf("seed preview deployment %s: %v", previewID, err)
		}
		record := &database.PullRequestDeployment{
			ID: seed.id, SourceDeploymentID: config.DeploymentID, PreviewDeploymentID: &previewID, OrganizationID: "org",
			GitHubIntegrationID: "integration", GitHubInstallationID: 42, Repository: "obiente/cloud", PullRequestNumber: int64(index + 1),
			HeadSHA: strings.Repeat("a", 40), HeadRef: "feature", BaseRef: "main", Status: int32(seed.status),
			IsolationVersion: seed.isolationVersion, ExpiresAt: now.Add(time.Hour), CreatedAt: now.Add(-time.Hour), UpdatedAt: seed.updatedAt,
		}
		if err := db.Create(record).Error; err != nil {
			t.Fatalf("seed pull request deployment %s: %v", seed.id, err)
		}
	}

	candidates, err := loadPullRequestIsolationMigrationCandidates(t.Context(), now.Add(-5*time.Minute))
	if err != nil {
		t.Fatalf("load isolation migration candidates: %v", err)
	}
	got := make([]string, 0, len(candidates))
	for _, candidate := range candidates {
		got = append(got, candidate.ID)
	}
	want := []string{"legacy-running", "previous-running", "retry-failed"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("isolation migration candidates = %v, want %v", got, want)
	}
}

func TestPullRequestSyncSourceComparisonIncludesRepositoryAndInstallation(t *testing.T) {
	t.Parallel()

	base := &pullRequestSyncSource{repositoryURL: "Obiente/Cloud", integrationID: "integration-1", installationID: 42}
	if !pullRequestSyncSourcesEqual(base, &pullRequestSyncSource{repositoryURL: "Obiente/Cloud", integrationID: "integration-1", installationID: 42}) {
		t.Fatal("identical pull request sources should match")
	}
	for _, changed := range []*pullRequestSyncSource{
		{repositoryURL: "Obiente/Other", integrationID: "integration-1", installationID: 42},
		{repositoryURL: "Obiente/Cloud", integrationID: "integration-2", installationID: 42},
		{repositoryURL: "Obiente/Cloud", integrationID: "integration-1", installationID: 43},
		nil,
	} {
		if pullRequestSyncSourcesEqual(base, changed) {
			t.Fatalf("changed pull request source matched: %#v", changed)
		}
	}
}

func TestExpiredPreviewCleanupAdvancesBeyondFirstBatch(t *testing.T) {
	db := newDeploymentServiceTestDB(t)
	background, cancel := context.WithCancel(context.Background())
	cancel()
	service := NewService(background, database.NewDeploymentRepository(db, nil), nil, nil)
	expiredAt := time.Now().Add(-time.Hour)
	records := make([]database.PullRequestDeployment, 0, 101)
	for index := 0; index < 101; index++ {
		records = append(records, database.PullRequestDeployment{
			ID: fmt.Sprintf("pr-expired-%03d", index), SourceDeploymentID: "source", OrganizationID: "org",
			Repository: "obiente/cloud", PullRequestNumber: int64(index + 1), HeadSHA: strings.Repeat("a", 40),
			HeadRef: "feature", BaseRef: "main", Status: int32(deploymentsv1.PullRequestDeploymentStatus_PULL_REQUEST_DEPLOYMENT_RUNNING), ExpiresAt: expiredAt,
		})
	}
	if err := db.Create(&records).Error; err != nil {
		t.Fatalf("seed expired previews: %v", err)
	}
	service.cleanupExpiredPullRequestDeployments(t.Context())
	var open int64
	if err := db.Model(&database.PullRequestDeployment{}).Where("closed_at IS NULL").Count(&open).Error; err != nil {
		t.Fatalf("count open previews: %v", err)
	}
	if open != 0 {
		t.Fatalf("expired cleanup left %d previews open", open)
	}
}

func stringPointer(value string) *string { return &value }

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

func TestPullRequestDomainRenderingRejectsUniquenessTruncation(t *testing.T) {
	config := &database.PullRequestDeploymentConfig{DomainTemplate: "{deployment}-aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa-{pr}"}
	source := &database.Deployment{ID: "deployment-with-a-long-stable-identifier"}
	record := &database.PullRequestDeployment{PullRequestNumber: 42, HeadRef: "feature"}
	if _, err := renderPRDomain(config, source, record); err == nil {
		t.Fatal("overlong hostname should be rejected instead of truncating its PR suffix")
	}
}

func TestForkApprovalPolicy(t *testing.T) {
	record := &database.PullRequestDeployment{FromFork: true}
	config := &database.PullRequestDeploymentConfig{ForkPolicy: int32(deploymentsv1.PullRequestForkPolicy_PULL_REQUEST_FORK_REQUIRE_APPROVAL)}
	if !requiresPullRequestApproval(record, config) {
		t.Fatal("fork policy should require approval")
	}
}

func TestPreviewRequestedResourcesMatchSourceRuntime(t *testing.T) {
	replicas := int32(3)
	memory := int64(768 * 1024 * 1024)
	cpu := int64(1536)
	requested := previewRequestedResources(&database.Deployment{Replicas: &replicas, MemoryBytes: &memory, CPUShares: &cpu})
	if requested.Replicas != 3 || requested.MemoryBytes != memory || requested.CPUshares != cpu {
		t.Fatalf("unexpected preview reservation: %#v", requested)
	}
}

func TestRefreshingPreviewReappliesCurrentVariableAllowlist(t *testing.T) {
	buildCommand := "pnpm build"
	port := int32(8080)
	source := &database.Deployment{ID: "source", Name: "App", Environment: int32(deploymentsv1.Environment_PRODUCTION), BuildStrategy: int32(deploymentsv1.BuildStrategy_NIXPACKS), BuildCommand: &buildCommand, Port: &port, EnvVars: `{"PUBLIC":"current","REMOVED_SECRET":"old"}`, BuildArgs: `{"SAFE":"current","REMOVED_TOKEN":"old"}`}
	preview := &database.Deployment{ID: "preview", BuildStrategy: int32(deploymentsv1.BuildStrategy_DOCKERFILE), EnvVars: `{"REMOVED_SECRET":"old"}`, BuildArgs: `{"REMOVED_TOKEN":"old"}`, EnvFileContent: "SECRET=old", DockerfileVolumes: `["/secret"]`}
	config := &database.PullRequestDeploymentConfig{DomainTemplate: "pr-{pr}-{deployment}", EnvironmentVariableNames: `["PUBLIC"]`, BuildArgumentNames: `["SAFE"]`}
	record := &database.PullRequestDeployment{PullRequestNumber: 31, HeadRef: "updated-head"}

	if err := refreshPreviewDeployment(preview, source, config, record); err != nil {
		t.Fatalf("refresh preview: %v", err)
	}
	if preview.EnvVars != `{"PUBLIC":"current"}` || preview.BuildArgs != `{"SAFE":"current"}` {
		t.Fatalf("preview retained values outside the current allowlist: env=%s build=%s", preview.EnvVars, preview.BuildArgs)
	}
	if preview.EnvFileContent != "" || preview.DockerfileVolumes != "[]" {
		t.Fatal("preview retained unscoped source configuration")
	}
	if preview.ID != "preview" || preview.BuildStrategy != source.BuildStrategy || preview.BuildCommand == nil || *preview.BuildCommand != buildCommand || preview.Port == nil || *preview.Port != port {
		t.Fatalf("preview did not refresh the current source template: %#v", preview)
	}
	if preview.Environment != int32(deploymentsv1.Environment_PULL_REQUEST) {
		t.Fatalf("preview inherited source environment %d instead of using the PR environment", preview.Environment)
	}
}

func TestPreviewComposeRoutingUsesPreviewDomainAndSourceService(t *testing.T) {
	port := int32(4321)
	source := &database.Deployment{ID: "source", Domain: "app.example.test", Port: &port}
	preview := &database.Deployment{ID: "preview", Domain: "pr-31-app.example.test"}
	routing, err := previewComposeRouting(preview, source, []database.DeploymentRouting{
		{ID: "secondary", Domain: "api.example.test", ServiceName: "api", TargetPort: 9000, Protocol: "http"},
		{ID: "primary", Domain: source.Domain, ServiceName: "web", PathPrefix: "/", TargetPort: 8080, Protocol: "http", SSLCertResolver: "letsencrypt", Middleware: "{}"},
	})
	if err != nil {
		t.Fatalf("create preview routing: %v", err)
	}
	if routing.DeploymentID != preview.ID || routing.Domain != preview.Domain || routing.ServiceName != "web" || routing.TargetPort != 8080 {
		t.Fatalf("unexpected preview routing: %#v", routing)
	}
	if !routing.SSLEnabled || routing.SSLCertResolver != "letsencrypt" {
		t.Fatalf("preview routing does not enable managed HTTPS: %#v", routing)
	}
	if routing.Protocol != "https" {
		t.Fatalf("preview routing does not use the HTTPS entrypoint: %#v", routing)
	}
}

func TestPreviewDockerfileRoutingUsesPreviewDomainAndSourcePort(t *testing.T) {
	port := int32(80)
	source := &database.Deployment{
		ID: "source", Domain: "app.example.test", Port: &port,
		BuildStrategy: int32(deploymentsv1.BuildStrategy_DOCKERFILE),
	}
	preview := &database.Deployment{
		ID: "preview", Domain: "pr-31-app.example.test",
		BuildStrategy: int32(deploymentsv1.BuildStrategy_DOCKERFILE),
	}
	routing, err := previewRouting(preview, source, []database.DeploymentRouting{{
		ID: "primary", Domain: source.Domain, ServiceName: "default", PathPrefix: "/docs",
		TargetPort: 80, Protocol: "https", SSLEnabled: true, SSLCertResolver: "letsencrypt", Middleware: "{}",
	}})
	if err != nil {
		t.Fatalf("create Dockerfile preview routing: %v", err)
	}
	if routing.DeploymentID != preview.ID || routing.Domain != preview.Domain || routing.ServiceName != "default" || routing.TargetPort != 80 {
		t.Fatalf("unexpected Dockerfile preview routing: %#v", routing)
	}
	if routing.PathPrefix != "/docs" || routing.Protocol != "https" || !routing.SSLEnabled {
		t.Fatalf("Dockerfile preview routing did not preserve the public route: %#v", routing)
	}
}

func TestPreviewComposeRoutingRequiresRoutablePort(t *testing.T) {
	_, err := previewComposeRouting(
		&database.Deployment{ID: "preview", Domain: "pr-31-app.example.test"},
		&database.Deployment{ID: "source"},
		nil,
	)
	if err == nil {
		t.Fatal("preview Compose routing should fail closed without a source service port")
	}
}

func TestPreviewComposeRoutingResolvesDefaultToActualService(t *testing.T) {
	port := int32(8080)
	source := &database.Deployment{ID: "source", Port: &port, ComposeYaml: "services:\n  web:\n    image: nginx\n    ports:\n      - '8080'\n"}
	routing, err := previewComposeRouting(
		&database.Deployment{ID: "preview", Domain: "pr-31-app.example.test"},
		source,
		[]database.DeploymentRouting{{ServiceName: "default", TargetPort: 8080}},
	)
	if err != nil {
		t.Fatalf("create preview routing: %v", err)
	}
	if routing.ServiceName != "web" {
		t.Fatalf("default source routing was not resolved to the Compose service: %#v", routing)
	}
}

func TestPreviewEnvironmentURLIncludesRoutingPathPrefix(t *testing.T) {
	db := newDeploymentServiceTestDB(t)
	if err := db.AutoMigrate(&database.DeploymentRouting{}); err != nil {
		t.Fatalf("migrate deployment routing: %v", err)
	}
	preview := &database.Deployment{ID: "preview-path", Domain: "pr-31.example.test", BuildStrategy: int32(deploymentsv1.BuildStrategy_DOCKERFILE)}
	routing := database.DeploymentRouting{ID: "route-preview-path-default", DeploymentID: preview.ID, Domain: preview.Domain, ServiceName: "web", PathPrefix: "/api", TargetPort: 8080, CreatedAt: time.Now(), UpdatedAt: time.Now()}
	if err := db.Create(&routing).Error; err != nil {
		t.Fatalf("seed preview routing: %v", err)
	}
	got, err := previewEnvironmentURL(t.Context(), preview)
	if err != nil {
		t.Fatalf("build preview URL: %v", err)
	}
	if got != "https://pr-31.example.test/api" {
		t.Fatalf("preview URL omitted its route prefix: %q", got)
	}
}

func TestPreviewComposeTargetTracksRenamedServiceAndPort(t *testing.T) {
	composeYAML := "services:\n  frontend:\n    image: nginx\n    ports:\n      - '4321'\n"
	serviceName, targetPort, err := resolvePreviewComposeTarget(composeYAML, "web", 8080)
	if err != nil {
		t.Fatalf("resolve current Compose target: %v", err)
	}
	if serviceName != "frontend" || targetPort != 4321 {
		t.Fatalf("current Compose target was not selected: service=%q port=%d", serviceName, targetPort)
	}
}

func TestComposePreviewEnvironmentIsInjectedLiterally(t *testing.T) {
	composeYAML := "services:\n  web:\n    image: nginx\n    environment:\n      EXISTING: kept\n"
	got, err := injectComposeServiceEnvironment(composeYAML, map[string]string{"PUBLIC_URL": "https://example.test/$path"})
	if err != nil {
		t.Fatalf("inject Compose environment: %v", err)
	}
	if !strings.Contains(got, "EXISTING: kept") || !strings.Contains(got, "PUBLIC_URL: https://example.test/$path") {
		t.Fatalf("allowlisted Compose environment was not preserved literally:\n%s", got)
	}
	validation := composeValidationEnvironment(map[string]string{"PUBLIC_URL": "https://example.test/${MISSING?must-exist}"})
	if validation["PUBLIC_URL"] != "https://example.test/$${MISSING?must-exist}" {
		t.Fatalf("Compose validation value was not escaped exactly once: %q", validation["PUBLIC_URL"])
	}
}

func TestPullRequestReportRetryBackoffIsBounded(t *testing.T) {
	if got := pullRequestReportRetryDelay(1); got != 30*time.Second {
		t.Fatalf("unexpected first report retry delay: %s", got)
	}
	if got := pullRequestReportRetryDelay(20); got != 15*time.Minute {
		t.Fatalf("report retry delay was not capped: %s", got)
	}
}

func TestPullRequestBooleanSettingsDoNotHaveORMDefaults(t *testing.T) {
	typeOfConfig := reflect.TypeOf(database.PullRequestDeploymentConfig{})
	for _, name := range []string{"RedeployOnPush", "CleanupOnClose", "CommentEnabled", "DeploymentStatusEnabled", "CheckRunEnabled"} {
		field, ok := typeOfConfig.FieldByName(name)
		if !ok {
			t.Fatalf("missing config field %s", name)
		}
		if strings.Contains(field.Tag.Get("gorm"), "default:") {
			t.Fatalf("%s has a GORM default that can override an explicitly supplied false value", name)
		}
	}
}

func TestPullRequestGitHubColumnsArePinned(t *testing.T) {
	typeOfDeployment := reflect.TypeOf(database.PullRequestDeployment{})
	wantColumns := map[string]string{
		"GitHubIntegrationID":  "github_integration_id",
		"GitHubInstallationID": "github_installation_id",
		"GitHubDeploymentID":   "github_deployment_id",
		"GitHubDeploymentSHA":  "github_deployment_sha",
		"GitHubCommentID":      "github_comment_id",
		"GitHubCheckRunID":     "github_check_run_id",
		"GitHubCheckRunSHA":    "github_check_run_sha",
	}
	for fieldName, columnName := range wantColumns {
		field, ok := typeOfDeployment.FieldByName(fieldName)
		if !ok {
			t.Fatalf("missing pull request deployment field %s", fieldName)
		}
		if !strings.Contains(field.Tag.Get("gorm"), "column:"+columnName) {
			t.Fatalf("%s is not pinned to %s", fieldName, columnName)
		}
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

func TestRestoredMergedPreviewIgnoresDuplicateCloseDelivery(t *testing.T) {
	now := time.Now()
	record := &database.PullRequestDeployment{Merged: true, RestoredAt: &now, HeadSHA: strings.Repeat("a", 40)}
	if !pullRequestCloseIsRestoredRedelivery(record, record.HeadSHA) {
		t.Fatal("the original close delivery should not remove a restored merged preview")
	}
	if pullRequestCloseIsRestoredRedelivery(record, strings.Repeat("b", 40)) {
		t.Fatal("a close delivery for a different revision must not be ignored")
	}
	record.RestoredAt = nil
	if pullRequestCloseIsRestoredRedelivery(record, record.HeadSHA) {
		t.Fatal("a normal open preview must still process close deliveries")
	}
}

func TestInternalReconciliationPreservesRestoredMergedPreview(t *testing.T) {
	now := time.Now()
	record := &database.PullRequestDeployment{Merged: true, RestoredAt: &now, HeadSHA: strings.Repeat("a", 40)}
	if !pullRequestReconciliationPreservesRestoredPreview(record, "reconcile", "closed", record.HeadSHA) {
		t.Fatal("internal settings reconciliation should preserve an active restored preview")
	}
	if pullRequestReconciliationPreservesRestoredPreview(record, "closed", "closed", record.HeadSHA) {
		t.Fatal("an external close delivery must remain distinguishable from internal reconciliation")
	}
}

func TestStaleBuildControlRequiresExpiredHeartbeat(t *testing.T) {
	now := time.Now()
	recent := now.Add(-deploymentBuildHeartbeatInterval)
	control := &database.DeploymentBuildControl{BuildToken: "active", HeartbeatAt: &recent, UpdatedAt: recent}
	if deploymentBuildControlIsStale(control, now) {
		t.Fatal("a recently heartbeating build was considered abandoned")
	}
	stale := now.Add(-deploymentBuildHeartbeatTimeout - time.Second)
	control.HeartbeatAt = &stale
	if !deploymentBuildControlIsStale(control, now) {
		t.Fatal("an abandoned build token was not considered stale")
	}
	legacyRecent := now.Add(-deploymentBuildHeartbeatTimeout - time.Second)
	control.HeartbeatAt = nil
	control.UpdatedAt = legacyRecent
	if deploymentBuildControlIsStale(control, now) {
		t.Fatal("a pre-heartbeat build owner was stolen during the rolling-upgrade grace period")
	}
	legacyStale := now.Add(-deploymentBuildLegacyOwnerTimeout - time.Second)
	control.UpdatedAt = legacyStale
	if !deploymentBuildControlIsStale(control, now) {
		t.Fatal("an abandoned pre-heartbeat build owner was never reclaimed")
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

func TestPreviewHostFallbackServesRunningPreviewOutage(t *testing.T) {
	db := newDeploymentServiceTestDB(t)
	host := "pr-31-deployment-123.my.obiente.cloud"
	environmentURL := "https://" + host + "/docs"
	record := database.PullRequestDeployment{
		ID: "pr-deployment-host-fallback", SourceDeploymentID: "source", OrganizationID: "org",
		GitHubIntegrationID: "integration", GitHubInstallationID: 42, Repository: "obiente/cloud", PullRequestNumber: 31,
		HeadSHA: strings.Repeat("a", 40), HeadRef: "feature", BaseRef: "main",
		Status:         int32(deploymentsv1.PullRequestDeploymentStatus_PULL_REQUEST_DEPLOYMENT_RUNNING),
		EnvironmentURL: &environmentURL, ExpiresAt: time.Now().Add(time.Hour),
	}
	if err := db.Create(&record).Error; err != nil {
		t.Fatalf("seed pull request deployment: %v", err)
	}
	request := httptest.NewRequest(http.MethodGet, "https://"+host+"/unavailable", nil)
	request.Host = host
	recorder := httptest.NewRecorder()
	if handled := (&Service{}).HandlePullRequestPreviewHostFallback(recorder, request); !handled {
		t.Fatal("preview host fallback did not handle a known preview domain")
	}
	response := recorder.Result()
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		t.Fatalf("preview host fallback status = %d, want 200", response.StatusCode)
	}
	if response.Header.Get("Retry-After") != "10" {
		t.Fatalf("preview host fallback did not advertise a retry: %#v", response.Header)
	}
	if location := response.Header.Get("Location"); location != "" {
		t.Fatalf("preview host fallback redirected into a loop: %q", location)
	}
	if body := recorder.Body.String(); !strings.Contains(body, "Preview temporarily unavailable") {
		t.Fatalf("preview host fallback rendered unexpected body: %s", body)
	}
}

func TestPreviewHostFallbackIgnoresUnrelatedOrUnknownHosts(t *testing.T) {
	newDeploymentServiceTestDB(t)
	for _, host := range []string{"obiente.cloud", "pr-31-unknown.my.obiente.cloud"} {
		request := httptest.NewRequest(http.MethodGet, "https://"+host+"/", nil)
		request.Host = host
		if handled := (&Service{}).HandlePullRequestPreviewHostFallback(httptest.NewRecorder(), request); handled {
			t.Errorf("preview host fallback handled unrelated host %q", host)
		}
	}
}

func TestDeploymentDashboardURLRequiresConfiguredOrigin(t *testing.T) {
	t.Setenv("DASHBOARD_URL", "")
	if got := deploymentDashboardURL("deployment-1"); got != "" {
		t.Fatalf("missing dashboard origin produced relative URL %q", got)
	}
	t.Setenv("DASHBOARD_URL", "https://dashboard.obiente.cloud/")
	if got := deploymentDashboardURL("deployment-1"); got != "https://dashboard.obiente.cloud/deployments/deployment-1" {
		t.Fatalf("dashboard URL = %q", got)
	}
}
