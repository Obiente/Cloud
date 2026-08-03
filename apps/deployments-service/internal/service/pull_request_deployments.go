package deployments

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"regexp"
	"sort"
	"strings"
	"time"

	"connectrpc.com/connect"
	githubclient "deployments-service/internal/github"
	"github.com/google/uuid"
	"github.com/obiente/cloud/apps/shared/pkg/auth"
	"github.com/obiente/cloud/apps/shared/pkg/database"
	"github.com/obiente/cloud/apps/shared/pkg/logger"
	"github.com/obiente/cloud/apps/shared/pkg/quota"
	deploymentsv1 "github.com/obiente/cloud/apps/shared/proto/obiente/cloud/deployments/v1"
	"google.golang.org/protobuf/types/known/timestamppb"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const (
	defaultPRDomainTemplate   = "pr-{pr}-{deployment}"
	defaultPRMaxActive        = int32(5)
	defaultPRTTLHours         = int32(72)
	defaultRestoredPRTTLHours = int32(4)
	maxPRMaxActive            = int32(50)
	maxPRTTLHours             = int32(24 * 30)
	maxRestoredPRTTLHours     = int32(72)
	maxPRFilterCount          = 100
	maxPRFilterLength         = 256
	prEnvironmentCommentMark  = "<!-- obiente-pr-environment:%s -->"
	// Version 3 provisions durable routes and provider-specific Traefik network
	// labels for every preview build strategy before deployment.
	currentPRIsolationVersion = int32(3)
	pendingPRIsolationVersion = int32(-1)
)

type pullRequestSyncSource struct {
	repositoryURL  string
	integrationID  string
	installationID int64
}

type isolationMigrationCandidate struct {
	ID               string
	IsolationVersion int32
}

func defaultPullRequestDeploymentConfig(deploymentID, organizationID string) *database.PullRequestDeploymentConfig {
	now := time.Now()
	return &database.PullRequestDeploymentConfig{
		DeploymentID: deploymentID, OrganizationID: organizationID,
		BaseBranches: "[]", IncludePaths: "[]", ExcludePaths: "[]",
		RedeployOnPush: true, CleanupOnClose: true, CommentEnabled: true,
		DeploymentStatusEnabled: true, CheckRunEnabled: true, DomainTemplate: defaultPRDomainTemplate,
		MaxActivePreviews: defaultPRMaxActive, TTLHours: defaultPRTTLHours,
		RestoredPreviewTTLHours:  defaultRestoredPRTTLHours,
		ForkPolicy:               int32(deploymentsv1.PullRequestForkPolicy_PULL_REQUEST_FORK_DENY),
		EnvironmentVariableNames: "[]", BuildArgumentNames: "[]",
		CreatedAt: now, UpdatedAt: now,
	}
}

func (s *Service) GetPullRequestDeploymentConfig(ctx context.Context, req *connect.Request[deploymentsv1.GetPullRequestDeploymentConfigRequest]) (*connect.Response[deploymentsv1.GetPullRequestDeploymentConfigResponse], error) {
	deployment, err := s.pullRequestSourceDeployment(ctx, req.Msg.GetDeploymentId(), req.Msg.GetOrganizationId(), "read")
	if err != nil {
		return nil, err
	}
	config := defaultPullRequestDeploymentConfig(deployment.ID, deployment.OrganizationID)
	err = database.DB.WithContext(ctx).Where("deployment_id = ?", deployment.ID).First(config).Error
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to load pull request deployment settings: %w", err))
	}
	return connect.NewResponse(&deploymentsv1.GetPullRequestDeploymentConfigResponse{Config: pullRequestConfigToProto(config)}), nil
}

func (s *Service) UpdatePullRequestDeploymentConfig(ctx context.Context, req *connect.Request[deploymentsv1.UpdatePullRequestDeploymentConfigRequest]) (*connect.Response[deploymentsv1.UpdatePullRequestDeploymentConfigResponse], error) {
	deployment, err := s.pullRequestSourceDeployment(ctx, req.Msg.GetDeploymentId(), req.Msg.GetOrganizationId(), "edit")
	if err != nil {
		return nil, err
	}
	if req.Msg.Config == nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("pull request deployment config is required"))
	}
	config, err := sanitizePullRequestDeploymentConfig(deployment, req.Msg.Config)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	if config.Enabled {
		if deployment.RepositoryURL == nil || normalizeGitHubRepoFullName(*deployment.RepositoryURL) == "" || deployment.GitHubIntegrationID == nil {
			return nil, connect.NewError(connect.CodeFailedPrecondition, fmt.Errorf("connect this deployment to a GitHub App repository before enabling pull request environments"))
		}
		if deployment.BuildStrategy == int32(deploymentsv1.BuildStrategy_PLAIN_COMPOSE) {
			return nil, connect.NewError(connect.CodeFailedPrecondition, fmt.Errorf("pull request environments require a repository-backed build strategy"))
		}
		if err := validateGitHubIntegrationForDeployment(ctx, deployment); err != nil {
			return nil, connect.NewError(connect.CodeFailedPrecondition, err)
		}
	}
	var previous database.PullRequestDeploymentConfig
	var previousErr error
	err = withDistributedLock(ctx, "pull-request-config:"+deployment.ID, func() error {
		previousErr = database.DB.WithContext(ctx).Where("deployment_id = ?", deployment.ID).First(&previous).Error
		if previousErr != nil && !errors.Is(previousErr, gorm.ErrRecordNotFound) {
			return fmt.Errorf("load current pull request deployment settings: %w", previousErr)
		}
		reconciliationRequired := (errors.Is(previousErr, gorm.ErrRecordNotFound) && config.Enabled) ||
			(previousErr == nil && pullRequestConfigRequiresReconciliation(&previous, config))
		if previousErr == nil {
			config.CreatedAt = previous.CreatedAt
			config.ReconciliationGeneration = previous.ReconciliationGeneration
		}
		if reconciliationRequired {
			config.ReconciliationGeneration++
		}
		config.ReconciliationPending = reconciliationRequired || (previousErr == nil && previous.ReconciliationPending)
		if reconciliationRequired {
			now := time.Now()
			config.ReconciliationAttempts = 0
			config.NextReconciliationAt = &now
			if config.Enabled {
				config.OpenPullRequestsSyncedAt = nil
			}
		} else if previousErr == nil && previous.ReconciliationPending {
			config.ReconciliationAttempts = previous.ReconciliationAttempts
			config.NextReconciliationAt = previous.NextReconciliationAt
			config.OpenPullRequestsSyncedAt = previous.OpenPullRequestsSyncedAt
		} else if previousErr == nil {
			config.OpenPullRequestsSyncedAt = previous.OpenPullRequestsSyncedAt
		}
		return database.DB.WithContext(ctx).Save(config).Error
	})
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to save pull request deployment settings: %w", err))
	}
	if config.ReconciliationPending {
		syncSource, reconcileErr := s.reconcilePullRequestDeploymentConfig(ctx, deployment, config)
		if reconcileErr != nil {
			if retryErr := schedulePullRequestReconciliationRetry(ctx, config); retryErr != nil {
				reconcileErr = errors.Join(reconcileErr, retryErr)
			}
			return nil, connect.NewError(connect.CodeUnavailable, fmt.Errorf("pull request settings were saved, but active previews could not all be reconciled: %w", reconcileErr))
		}
		completed, err := completePullRequestReconciliation(ctx, config, syncSource)
		if err != nil {
			return nil, connect.NewError(connect.CodeUnavailable, fmt.Errorf("pull request settings were reconciled, but completion could not be recorded: %w", err))
		}
		if completed {
			config.ReconciliationPending = false
			config.ReconciliationAttempts = 0
			config.NextReconciliationAt = nil
			if config.Enabled {
				now := time.Now()
				config.OpenPullRequestsSyncedAt = &now
			}
		} else if err := database.DB.WithContext(ctx).Where("deployment_id = ?", config.DeploymentID).First(config).Error; err != nil {
			return nil, connect.NewError(connect.CodeUnavailable, fmt.Errorf("pull request settings changed during reconciliation and could not be reloaded: %w", err))
		}
	}
	// Persist the disabled setting before retiring checks. Reporters that are
	// already running finish first under the per-PR reporting lock; later ones
	// then observe the disabled setting and cannot put the check back in a
	// pending state. Retrying this update also retries any failed retirement.
	if !config.CheckRunEnabled {
		if err := s.completePullRequestChecksBeforeDisable(ctx, deployment.ID); err != nil {
			return nil, connect.NewError(connect.CodeUnavailable, fmt.Errorf("GitHub check reporting was disabled, but active checks could not be completed: %w", err))
		}
	}
	if !config.DeploymentStatusEnabled {
		if err := s.completePullRequestDeploymentsBeforeDisable(ctx, deployment.ID); err != nil {
			return nil, connect.NewError(connect.CodeUnavailable, fmt.Errorf("GitHub deployment reporting was disabled, but active deployments could not be retired: %w", err))
		}
	}
	if !config.CommentEnabled {
		if err := s.deletePullRequestCommentsBeforeDisable(ctx, deployment.ID); err != nil {
			return nil, connect.NewError(connect.CodeUnavailable, fmt.Errorf("pull request comments were disabled, but maintained comments could not be removed: %w", err))
		}
	}
	if previousErr == nil && ((!previous.CheckRunEnabled && config.CheckRunEnabled) ||
		(!previous.DeploymentStatusEnabled && config.DeploymentStatusEnabled) ||
		(!previous.CommentEnabled && config.CommentEnabled)) {
		s.reportExistingPullRequestDeployments(deployment.ID)
	}
	return connect.NewResponse(&deploymentsv1.UpdatePullRequestDeploymentConfigResponse{Config: pullRequestConfigToProto(config)}), nil
}

func (s *Service) reportExistingPullRequestDeployments(sourceDeploymentID string) {
	var recordIDs []string
	if err := database.DB.Model(&database.PullRequestDeployment{}).
		Where("source_deployment_id = ?", sourceDeploymentID).
		Pluck("id", &recordIDs).Error; err != nil {
		logger.Warn("[PRDeployments] Failed to queue reports after enabling GitHub reporting for %s: %v", sourceDeploymentID, err)
		return
	}
	for _, recordID := range recordIDs {
		go s.reportPullRequestDeployment(recordID)
	}
}

func pullRequestConfigRequiresReconciliation(previous, current *database.PullRequestDeploymentConfig) bool {
	if previous == nil || current == nil {
		return false
	}
	return previous.Enabled != current.Enabled ||
		previous.BaseBranches != current.BaseBranches ||
		previous.IncludePaths != current.IncludePaths ||
		previous.ExcludePaths != current.ExcludePaths ||
		previous.DeployDrafts != current.DeployDrafts ||
		previous.CleanupOnClose != current.CleanupOnClose ||
		previous.DomainTemplate != current.DomainTemplate ||
		previous.MaxActivePreviews != current.MaxActivePreviews ||
		previous.TTLHours != current.TTLHours ||
		previous.ForkPolicy != current.ForkPolicy ||
		previous.EnvironmentVariableNames != current.EnvironmentVariableNames ||
		previous.BuildArgumentNames != current.BuildArgumentNames ||
		previous.RequireApproval != current.RequireApproval ||
		previous.ApprovalCoversUpdates != current.ApprovalCoversUpdates
}

func (s *Service) reconcilePullRequestDeploymentConfig(ctx context.Context, source *database.Deployment, config *database.PullRequestDeploymentConfig) (*pullRequestSyncSource, error) {
	var records []database.PullRequestDeployment
	if err := database.DB.WithContext(ctx).Where("source_deployment_id = ? AND closed_at IS NULL", source.ID).Order("id ASC").Find(&records).Error; err != nil {
		return nil, err
	}
	// Make cleanup durable before any external Docker or GitHub operation. If
	// this request or replica dies mid-reconciliation, the janitor will retry
	// every runtime that was still attached instead of preserving the old
	// policy for its previous (potentially very long) TTL.
	current, err := schedulePullRequestRuntimeReconciliation(ctx, config)
	if err != nil {
		return nil, fmt.Errorf("schedule durable settings reconciliation: %w", err)
	}
	if !current {
		return nil, nil
	}

	integrationID, installationID := stringValue(source.GitHubIntegrationID), int64(0)
	var syncSource *pullRequestSyncSource
	if config.Enabled {
		var err error
		syncSource, err = loadPullRequestSyncSource(ctx, source)
		if err != nil {
			return nil, err
		}
		integrationID, installationID = syncSource.integrationID, syncSource.installationID
	}

	var reconciliationErrors []error
	processed := make(map[int64]struct{}, len(records))
	for i := range records {
		record := records[i]
		reconciled := false
		superseded := false
		lockKey := fmt.Sprintf("pull-request:%s:%s:%d", source.ID, record.Repository, record.PullRequestNumber)
		configLockKey := "pull-request-config:" + source.ID
		if err := withDistributedLock(ctx, configLockKey, func() error {
			return withDistributedLock(ctx, lockKey, func() error {
				currentConfig, current, err := currentPullRequestReconciliationConfig(ctx, config)
				if err != nil {
					return err
				}
				if !current {
					superseded = true
					return nil
				}
				if err := database.DB.WithContext(ctx).Where("id = ? AND closed_at IS NULL", record.ID).First(&record).Error; err != nil {
					if errors.Is(err, gorm.ErrRecordNotFound) {
						return nil
					}
					return err
				}
				if err := s.detachPullRequestRuntimeForReconciliation(ctx, &record, source, currentConfig.Enabled, integrationID, installationID); err != nil {
					return err
				}
				if !currentConfig.Enabled {
					s.reportPullRequestDeployment(record.ID)
					reconciled = true
					return nil
				}
				payload := githubPullRequestWebhookPayload{Action: "reconcile", Number: record.PullRequestNumber}
				payload.Installation.ID = installationID
				payload.Repository.FullName = record.Repository
				if err := s.processPullRequestWebhookLocked(ctx, *currentConfig, payload); err != nil {
					return err
				}
				reconciled = true
				return nil
			})
		}); err != nil {
			reconciliationErrors = append(reconciliationErrors, fmt.Errorf("reconcile %s#%d: %w", record.Repository, record.PullRequestNumber, err))
		} else if superseded {
			return syncSource, nil
		} else if reconciled {
			processed[record.PullRequestNumber] = struct{}{}
		}
	}
	if config.Enabled {
		if err := s.backfillOpenPullRequestsForConfig(ctx, source, config, syncSource, processed); err != nil {
			reconciliationErrors = append(reconciliationErrors, err)
		}
	}
	return syncSource, errors.Join(reconciliationErrors...)
}

func schedulePullRequestRuntimeReconciliation(ctx context.Context, config *database.PullRequestDeploymentConfig) (bool, error) {
	if config == nil {
		return false, nil
	}
	current := false
	err := withDistributedLock(ctx, "pull-request-config:"+config.DeploymentID, func() error {
		currentConfig, isCurrent, err := currentPullRequestReconciliationConfig(ctx, config)
		if err != nil || !isCurrent {
			return err
		}
		now := time.Now()
		result := database.DB.WithContext(ctx).Model(&database.PullRequestDeployment{}).
			Where("source_deployment_id = ? AND closed_at IS NULL AND preview_deployment_id IS NOT NULL", currentConfig.DeploymentID).
			Updates(map[string]interface{}{"expires_at": now, "error": "Preview settings reconciliation is pending.", "updated_at": now})
		if result.Error != nil {
			return result.Error
		}
		current = true
		return nil
	})
	return current, err
}

func currentPullRequestReconciliationConfig(ctx context.Context, snapshot *database.PullRequestDeploymentConfig) (*database.PullRequestDeploymentConfig, bool, error) {
	if snapshot == nil {
		return nil, false, nil
	}
	var current database.PullRequestDeploymentConfig
	err := database.DB.WithContext(ctx).
		Where("deployment_id = ? AND reconciliation_pending = ? AND reconciliation_generation = ?", snapshot.DeploymentID, true, snapshot.ReconciliationGeneration).
		First(&current).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, err
	}
	return &current, true, nil
}

func loadPullRequestSyncSource(ctx context.Context, source *database.Deployment) (*pullRequestSyncSource, error) {
	if source == nil || source.RepositoryURL == nil || normalizeGitHubRepoFullName(*source.RepositoryURL) == "" {
		return nil, fmt.Errorf("source deployment does not have a valid GitHub repository")
	}
	integrationID := stringValue(source.GitHubIntegrationID)
	if integrationID == "" {
		return nil, fmt.Errorf("current GitHub App integration is unavailable")
	}
	var integration database.GitHubIntegration
	if err := database.DB.WithContext(ctx).Where("id = ?", integrationID).First(&integration).Error; err != nil {
		return nil, fmt.Errorf("load current GitHub App integration: %w", err)
	}
	if integration.GitHubAppInstallationID == nil || *integration.GitHubAppInstallationID <= 0 {
		return nil, fmt.Errorf("current GitHub App installation is unavailable")
	}
	return &pullRequestSyncSource{
		repositoryURL:  stringValue(source.RepositoryURL),
		integrationID:  integrationID,
		installationID: *integration.GitHubAppInstallationID,
	}, nil
}

func (s *Service) backfillOpenPullRequestsForConfig(ctx context.Context, source *database.Deployment, config *database.PullRequestDeploymentConfig, syncSource *pullRequestSyncSource, processed map[int64]struct{}) error {
	if syncSource == nil {
		return fmt.Errorf("current GitHub source is unavailable")
	}
	client, err := githubclient.NewInstallationClient(ctx, syncSource.installationID)
	if err != nil {
		return fmt.Errorf("create GitHub client for open pull request backfill: %w", err)
	}
	repository := ""
	if source.RepositoryURL != nil {
		repository = normalizeGitHubRepoFullName(*source.RepositoryURL)
	}
	if repository == "" {
		return fmt.Errorf("source deployment does not have a valid GitHub repository")
	}
	openPullRequests, err := client.ListOpenPullRequests(ctx, repository)
	if err != nil {
		return fmt.Errorf("list existing open pull requests: %w", err)
	}
	var backfillErrors []error
	for i := range openPullRequests {
		pullRequest := openPullRequests[i]
		if _, exists := processed[pullRequest.Number]; exists {
			continue
		}
		payload := pullRequestWebhookPayloadFromGitHub(repository, syncSource.installationID, &pullRequest)
		lockKey := fmt.Sprintf("pull-request:%s:%s:%d", source.ID, repository, pullRequest.Number)
		if processErr := withDistributedLock(ctx, "pull-request-source:"+source.ID, func() error {
			currentSource, err := s.repo.GetByID(ctx, source.ID)
			if err != nil {
				return err
			}
			currentSyncSource, err := loadPullRequestSyncSource(ctx, currentSource)
			if err != nil {
				return err
			}
			if !pullRequestSyncSourcesEqual(currentSyncSource, syncSource) {
				return fmt.Errorf("source repository or GitHub installation changed during open pull request discovery")
			}
			return withDistributedLock(ctx, lockKey, func() error {
				var current database.PullRequestDeploymentConfig
				if err := database.DB.WithContext(ctx).Where("deployment_id = ? AND enabled = ? AND reconciliation_generation = ?", config.DeploymentID, true, config.ReconciliationGeneration).First(&current).Error; err != nil {
					if errors.Is(err, gorm.ErrRecordNotFound) {
						return nil
					}
					return err
				}
				return s.processPullRequestWebhookLocked(ctx, current, payload)
			})
		}); processErr != nil {
			backfillErrors = append(backfillErrors, fmt.Errorf("backfill %s#%d: %w", repository, pullRequest.Number, processErr))
		}
	}
	return errors.Join(backfillErrors...)
}

func pullRequestSyncSourcesEqual(left, right *pullRequestSyncSource) bool {
	return left != nil && right != nil &&
		left.repositoryURL == right.repositoryURL &&
		left.integrationID == right.integrationID &&
		left.installationID == right.installationID
}

func pullRequestWebhookPayloadFromGitHub(repository string, installationID int64, pullRequest *githubclient.PullRequest) githubPullRequestWebhookPayload {
	payload := githubPullRequestWebhookPayload{Action: "reconcile", Number: pullRequest.Number}
	payload.Installation.ID = installationID
	payload.Repository.FullName = repository
	payload.PullRequest.Draft = pullRequest.Draft
	payload.PullRequest.Merged = pullRequest.Merged
	payload.PullRequest.Head.SHA = pullRequest.Head.SHA
	payload.PullRequest.Head.Ref = pullRequest.Head.Ref
	payload.PullRequest.Head.Repo.FullName = pullRequest.Head.Repo.FullName
	payload.PullRequest.Base.Ref = pullRequest.Base.Ref
	payload.PullRequest.Base.Repo.FullName = pullRequest.Base.Repo.FullName
	return payload
}

func (s *Service) detachPullRequestRuntimeForReconciliation(ctx context.Context, record *database.PullRequestDeployment, source *database.Deployment, enabled bool, integrationID string, installationID int64) error {
	return withDistributedLock(ctx, "organization-quota:"+source.OrganizationID, func() error {
		if record.PreviewDeploymentID != nil {
			if err := s.removePreviewDeployment(ctx, record); err != nil {
				s.markPullRequestPreviewCleanupForRetry(ctx, record.ID, "Preview removal is pending after a settings change.")
				return fmt.Errorf("remove existing preview runtime: %w", err)
			}
		}
		if installationID > 0 {
			record.GitHubIntegrationID = integrationID
			record.GitHubInstallationID = installationID
		}
		_ = s.markGitHubDeploymentInactive(ctx, record)
		_ = s.markGitHubCheckRunComplete(ctx, record, "Preview settings changed", "Obiente is re-evaluating this pull request environment.", "cancelled")

		now := time.Now()
		status := int32(deploymentsv1.PullRequestDeploymentStatus_PULL_REQUEST_DEPLOYMENT_QUEUED)
		var errorMessage *string
		if !enabled {
			status = int32(deploymentsv1.PullRequestDeploymentStatus_PULL_REQUEST_DEPLOYMENT_SKIPPED)
			message := "Pull request environments are disabled for this deployment."
			errorMessage = &message
		}
		updates := map[string]interface{}{
			"preview_deployment_id": nil,
			"environment_url":       nil,
			"active_head_sha":       nil,
			"github_deployment_id":  nil,
			"github_deployment_sha": nil,
			"github_check_run_id":   nil,
			"github_check_run_sha":  nil,
			"status":                status,
			"error":                 errorMessage,
			"updated_at":            now,
		}
		if installationID > 0 {
			updates["github_integration_id"] = integrationID
			updates["github_installation_id"] = installationID
		}
		if err := database.DB.WithContext(ctx).Model(&database.PullRequestDeployment{}).Where("id = ? AND closed_at IS NULL", record.ID).Updates(updates).Error; err != nil {
			return err
		}
		record.PreviewDeploymentID, record.EnvironmentURL, record.ActiveHeadSHA = nil, nil, nil
		record.GitHubDeploymentID, record.GitHubDeploymentSHA = nil, nil
		record.GitHubCheckRunID, record.GitHubCheckRunSHA = nil, nil
		record.Status, record.UpdatedAt = status, now
		record.Error = errorMessage
		return nil
	})
}

func (s *Service) completePullRequestChecksBeforeDisable(ctx context.Context, sourceDeploymentID string) error {
	var records []database.PullRequestDeployment
	if err := database.DB.WithContext(ctx).
		Where("source_deployment_id = ? AND closed_at IS NULL AND github_check_run_id IS NOT NULL", sourceDeploymentID).
		Find(&records).Error; err != nil {
		return err
	}
	for i := range records {
		if err := s.markGitHubCheckRunComplete(ctx, &records[i], "Preview check disabled", "GitHub check reporting was disabled in Obiente.", "neutral"); err != nil {
			return err
		}
	}
	if len(records) == 0 {
		return nil
	}
	ids := make([]string, 0, len(records))
	for i := range records {
		ids = append(ids, records[i].ID)
	}
	return database.DB.WithContext(ctx).Model(&database.PullRequestDeployment{}).Where("id IN ?", ids).
		Updates(map[string]interface{}{"github_check_run_id": nil, "github_check_run_sha": nil}).Error
}

func (s *Service) completePullRequestDeploymentsBeforeDisable(ctx context.Context, sourceDeploymentID string) error {
	var records []database.PullRequestDeployment
	if err := database.DB.WithContext(ctx).
		Where("source_deployment_id = ? AND closed_at IS NULL AND github_deployment_id IS NOT NULL", sourceDeploymentID).
		Find(&records).Error; err != nil {
		return err
	}
	for i := range records {
		if err := s.markGitHubDeploymentInactiveWithDescription(ctx, &records[i], "GitHub deployment reporting was disabled in Obiente."); err != nil {
			return err
		}
	}
	if len(records) == 0 {
		return nil
	}
	ids := make([]string, 0, len(records))
	for i := range records {
		ids = append(ids, records[i].ID)
	}
	return database.DB.WithContext(ctx).Model(&database.PullRequestDeployment{}).Where("id IN ?", ids).
		Updates(map[string]interface{}{"github_deployment_id": nil, "github_deployment_sha": nil}).Error
}

func (s *Service) deletePullRequestCommentsBeforeDisable(ctx context.Context, sourceDeploymentID string) error {
	var records []database.PullRequestDeployment
	if err := database.DB.WithContext(ctx).
		Where("source_deployment_id = ? AND github_comment_id IS NOT NULL", sourceDeploymentID).
		Find(&records).Error; err != nil {
		return err
	}
	for i := range records {
		recordID := records[i].ID
		if err := withDistributedLock(ctx, "pull-request-report:"+recordID, func() error {
			var current database.PullRequestDeployment
			if err := database.DB.WithContext(ctx).Where("id = ?", recordID).First(&current).Error; err != nil {
				return err
			}
			if current.GitHubCommentID == nil {
				return nil
			}
			client, err := githubclient.NewInstallationClient(ctx, current.GitHubInstallationID)
			if err != nil {
				return err
			}
			if err := client.DeleteIssueComment(ctx, current.Repository, *current.GitHubCommentID); err != nil {
				return err
			}
			return database.DB.WithContext(ctx).Model(&database.PullRequestDeployment{}).
				Where("id = ? AND github_comment_id = ?", current.ID, *current.GitHubCommentID).
				Update("github_comment_id", nil).Error
		}); err != nil {
			return fmt.Errorf("remove maintained comment for %s: %w", recordID, err)
		}
	}
	return nil
}

func (s *Service) ListPullRequestDeployments(ctx context.Context, req *connect.Request[deploymentsv1.ListPullRequestDeploymentsRequest]) (*connect.Response[deploymentsv1.ListPullRequestDeploymentsResponse], error) {
	deployment, err := s.pullRequestSourceDeployment(ctx, req.Msg.GetDeploymentId(), req.Msg.GetOrganizationId(), "read")
	if err != nil {
		return nil, err
	}
	query := database.DB.WithContext(ctx).Where("source_deployment_id = ? AND organization_id = ?", deployment.ID, deployment.OrganizationID)
	if !req.Msg.GetIncludeClosed() {
		query = query.Where("closed_at IS NULL")
	}
	var records []database.PullRequestDeployment
	if err := query.Order("CASE WHEN closed_at IS NULL THEN 0 ELSE 1 END").Order("updated_at DESC").Limit(100).Find(&records).Error; err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to list pull request deployments: %w", err))
	}
	items := make([]*deploymentsv1.PullRequestDeployment, 0, len(records))
	for i := range records {
		items = append(items, pullRequestDeploymentToProto(&records[i]))
	}
	return connect.NewResponse(&deploymentsv1.ListPullRequestDeploymentsResponse{Deployments: items}), nil
}

func (s *Service) ApprovePullRequestDeployment(ctx context.Context, req *connect.Request[deploymentsv1.ApprovePullRequestDeploymentRequest]) (*connect.Response[deploymentsv1.ApprovePullRequestDeploymentResponse], error) {
	record, _, err := s.authorizedPullRequestDeployment(ctx, req.Msg.GetDeploymentId(), req.Msg.GetOrganizationId(), req.Msg.GetPullRequestDeploymentId(), "edit")
	if err != nil {
		return nil, err
	}
	if record.ClosedAt != nil {
		return nil, connect.NewError(connect.CodeFailedPrecondition, fmt.Errorf("pull request environment is closed"))
	}
	allowedStatuses := []int32{
		int32(deploymentsv1.PullRequestDeploymentStatus_PULL_REQUEST_DEPLOYMENT_WAITING_APPROVAL),
		int32(deploymentsv1.PullRequestDeploymentStatus_PULL_REQUEST_DEPLOYMENT_REJECTED),
	}
	if record.ActiveHeadSHA != nil || !containsPRStatus(allowedStatuses, record.Status) {
		return nil, connect.NewError(connect.CodeFailedPrecondition, fmt.Errorf("pull request environment is not waiting for approval"))
	}
	user, err := auth.GetUserFromContext(ctx)
	if err != nil || user == nil {
		return nil, connect.NewError(connect.CodeUnauthenticated, fmt.Errorf("authenticated approver is required"))
	}
	_ = s.markGitHubCheckRunComplete(ctx, record, "Preview approved", "A maintainer approved this revision. A new check will report the build.", "neutral")
	now := time.Now()
	result := pullRequestApprovalMutation(ctx, record).
		Where("status IN ?", allowedStatuses).
		Updates(map[string]interface{}{"approved_by": user.Id, "approved_head_sha": record.HeadSHA, "approved_at": now, "github_check_run_id": nil, "github_check_run_sha": nil, "status": int32(deploymentsv1.PullRequestDeploymentStatus_PULL_REQUEST_DEPLOYMENT_QUEUED), "error": nil, "updated_at": now})
	if result.Error != nil {
		return nil, connect.NewError(connect.CodeInternal, result.Error)
	}
	if result.RowsAffected != 1 {
		return nil, connect.NewError(connect.CodeAborted, fmt.Errorf("pull request environment changed while it was being approved"))
	}
	if err := database.DB.WithContext(ctx).Where("id = ?", record.ID).First(record).Error; err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	go s.deployPullRequestEnvironment(record.ID)
	return connect.NewResponse(&deploymentsv1.ApprovePullRequestDeploymentResponse{Deployment: pullRequestDeploymentToProto(record)}), nil
}

func (s *Service) RejectPullRequestDeployment(ctx context.Context, req *connect.Request[deploymentsv1.RejectPullRequestDeploymentRequest]) (*connect.Response[deploymentsv1.RejectPullRequestDeploymentResponse], error) {
	record, _, err := s.authorizedPullRequestDeployment(ctx, req.Msg.GetDeploymentId(), req.Msg.GetOrganizationId(), req.Msg.GetPullRequestDeploymentId(), "edit")
	if err != nil {
		return nil, err
	}
	reason := strings.TrimSpace(req.Msg.GetReason())
	if len(reason) > 500 {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("rejection reason must be 500 characters or fewer"))
	}
	if reason == "" {
		reason = "A maintainer rejected this pull request environment."
	}
	now := time.Now()
	result := pullRequestApprovalMutation(ctx, record).
		Where("status = ?", int32(deploymentsv1.PullRequestDeploymentStatus_PULL_REQUEST_DEPLOYMENT_WAITING_APPROVAL)).
		Updates(map[string]interface{}{"status": int32(deploymentsv1.PullRequestDeploymentStatus_PULL_REQUEST_DEPLOYMENT_REJECTED), "error": reason, "updated_at": now})
	if result.Error != nil {
		return nil, connect.NewError(connect.CodeInternal, result.Error)
	}
	if result.RowsAffected != 1 {
		return nil, connect.NewError(connect.CodeAborted, fmt.Errorf("pull request environment changed while it was being rejected"))
	}
	if err := database.DB.WithContext(ctx).Where("id = ?", record.ID).First(record).Error; err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	go s.reportPullRequestDeployment(record.ID)
	return connect.NewResponse(&deploymentsv1.RejectPullRequestDeploymentResponse{Deployment: pullRequestDeploymentToProto(record)}), nil
}

func pullRequestApprovalMutation(ctx context.Context, record *database.PullRequestDeployment) *gorm.DB {
	return database.DB.WithContext(ctx).Model(&database.PullRequestDeployment{}).
		Where("id = ? AND head_sha = ? AND closed_at IS NULL AND active_head_sha IS NULL", record.ID, record.HeadSHA)
}

func (s *Service) RedeployPullRequestDeployment(ctx context.Context, req *connect.Request[deploymentsv1.RedeployPullRequestDeploymentRequest]) (*connect.Response[deploymentsv1.RedeployPullRequestDeploymentResponse], error) {
	record, config, err := s.authorizedPullRequestDeployment(ctx, req.Msg.GetDeploymentId(), req.Msg.GetOrganizationId(), req.Msg.GetPullRequestDeploymentId(), "edit")
	if err != nil {
		return nil, err
	}
	if requiresPullRequestApproval(record, config) && !pullRequestDeploymentApproved(record, config) {
		return nil, connect.NewError(connect.CodeFailedPrecondition, fmt.Errorf("approve the current pull request revision before redeploying"))
	}
	if record.ClosedAt != nil || record.ActiveHeadSHA != nil {
		return nil, connect.NewError(connect.CodeFailedPrecondition, fmt.Errorf("pull request environment cannot be redeployed in its current state"))
	}
	_ = s.markGitHubDeploymentInactive(ctx, record)
	_ = s.markGitHubCheckRunComplete(ctx, record, "Preview redeploying", "A maintainer requested a fresh preview build.", "cancelled")
	now := time.Now()
	result := database.DB.WithContext(ctx).Model(&database.PullRequestDeployment{}).
		Where("id = ? AND closed_at IS NULL AND active_head_sha IS NULL", record.ID).
		Updates(map[string]interface{}{"github_deployment_id": nil, "github_deployment_sha": nil, "github_check_run_id": nil, "github_check_run_sha": nil, "status": int32(deploymentsv1.PullRequestDeploymentStatus_PULL_REQUEST_DEPLOYMENT_QUEUED), "error": nil, "updated_at": now})
	if result.Error != nil {
		return nil, connect.NewError(connect.CodeInternal, result.Error)
	}
	if result.RowsAffected != 1 {
		return nil, connect.NewError(connect.CodeAborted, fmt.Errorf("pull request environment changed while it was being redeployed"))
	}
	if err := database.DB.WithContext(ctx).Where("id = ?", record.ID).First(record).Error; err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	go s.deployPullRequestEnvironment(record.ID)
	return connect.NewResponse(&deploymentsv1.RedeployPullRequestDeploymentResponse{Deployment: pullRequestDeploymentToProto(record)}), nil
}

func (s *Service) DeletePullRequestDeployment(ctx context.Context, req *connect.Request[deploymentsv1.DeletePullRequestDeploymentRequest]) (*connect.Response[deploymentsv1.DeletePullRequestDeploymentResponse], error) {
	record, _, err := s.authorizedPullRequestDeployment(ctx, req.Msg.GetDeploymentId(), req.Msg.GetOrganizationId(), req.Msg.GetPullRequestDeploymentId(), "delete")
	if err != nil {
		return nil, err
	}
	if err := s.cleanupPullRequestDeployment(ctx, record, "Removed from Obiente."); err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&deploymentsv1.DeletePullRequestDeploymentResponse{Success: true}), nil
}

func (s *Service) RestorePullRequestDeployment(ctx context.Context, req *connect.Request[deploymentsv1.RestorePullRequestDeploymentRequest]) (*connect.Response[deploymentsv1.RestorePullRequestDeploymentResponse], error) {
	record, config, err := s.authorizedPullRequestDeployment(ctx, req.Msg.GetDeploymentId(), req.Msg.GetOrganizationId(), req.Msg.GetPullRequestDeploymentId(), "edit")
	if err != nil {
		return nil, err
	}
	if !config.Enabled {
		return nil, connect.NewError(connect.CodeFailedPrecondition, fmt.Errorf("pull request environments are disabled for this deployment"))
	}
	if !pullRequestDeploymentCanRestore(record, config) {
		return nil, connect.NewError(connect.CodeFailedPrecondition, fmt.Errorf("only a previously approved revision can be restored"))
	}
	if record.ActiveHeadSHA != nil {
		return nil, connect.NewError(connect.CodeFailedPrecondition, fmt.Errorf("pull request environment is already being restored"))
	}
	now := time.Now()
	expiresAt := now.Add(time.Duration(config.RestoredPreviewTTLHours) * time.Hour)
	result := database.DB.WithContext(ctx).Model(&database.PullRequestDeployment{}).
		Where("id = ? AND merged = ? AND closed_at IS NOT NULL AND active_head_sha IS NULL", record.ID, true).
		Updates(map[string]interface{}{"preview_deployment_id": nil, "github_deployment_id": nil, "github_deployment_sha": nil, "github_check_run_id": nil, "github_check_run_sha": nil, "status": int32(deploymentsv1.PullRequestDeploymentStatus_PULL_REQUEST_DEPLOYMENT_QUEUED), "error": nil, "closed_at": nil, "restored_at": now, "expires_at": expiresAt, "updated_at": now})
	if result.Error != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to restore pull request environment: %w", result.Error))
	}
	if result.RowsAffected != 1 {
		return nil, connect.NewError(connect.CodeAborted, fmt.Errorf("pull request environment changed while it was being restored"))
	}
	if err := database.DB.WithContext(ctx).Where("id = ?", record.ID).First(record).Error; err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	go s.deployPullRequestEnvironment(record.ID)
	return connect.NewResponse(&deploymentsv1.RestorePullRequestDeploymentResponse{Deployment: pullRequestDeploymentToProto(record)}), nil
}

func (s *Service) pullRequestSourceDeployment(ctx context.Context, deploymentID, organizationID, permission string) (*database.Deployment, error) {
	if strings.TrimSpace(deploymentID) == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("deployment_id is required"))
	}
	if err := s.checkDeploymentPermission(ctx, deploymentID, permission); err != nil {
		return nil, err
	}
	deployment, err := s.repo.GetByID(ctx, deploymentID)
	if err != nil {
		return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("deployment not found"))
	}
	if organizationID != "" && deployment.OrganizationID != organizationID {
		return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("deployment not found"))
	}
	return deployment, nil
}

func (s *Service) authorizedPullRequestDeployment(ctx context.Context, deploymentID, organizationID, recordID, permission string) (*database.PullRequestDeployment, *database.PullRequestDeploymentConfig, error) {
	deployment, err := s.pullRequestSourceDeployment(ctx, deploymentID, organizationID, permission)
	if err != nil {
		return nil, nil, err
	}
	var record database.PullRequestDeployment
	if err := database.DB.WithContext(ctx).Where("id = ? AND source_deployment_id = ? AND organization_id = ?", recordID, deployment.ID, deployment.OrganizationID).First(&record).Error; err != nil {
		return nil, nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("pull request environment not found"))
	}
	config := defaultPullRequestDeploymentConfig(deployment.ID, deployment.OrganizationID)
	if err := database.DB.WithContext(ctx).Where("deployment_id = ?", deployment.ID).First(config).Error; err != nil {
		return nil, nil, connect.NewError(connect.CodeInternal, err)
	}
	return &record, config, nil
}

func sanitizePullRequestDeploymentConfig(deployment *database.Deployment, input *deploymentsv1.PullRequestDeploymentConfig) (*database.PullRequestDeploymentConfig, error) {
	baseBranches, err := sanitizePRPatterns(input.GetBaseBranches(), false)
	if err != nil {
		return nil, fmt.Errorf("invalid base branch filter: %w", err)
	}
	includePaths, err := sanitizePRPatterns(input.GetIncludePaths(), true)
	if err != nil {
		return nil, fmt.Errorf("invalid include path: %w", err)
	}
	excludePaths, err := sanitizePRPatterns(input.GetExcludePaths(), true)
	if err != nil {
		return nil, fmt.Errorf("invalid exclude path: %w", err)
	}
	envNames, err := sanitizePRVariableNames(input.GetEnvironmentVariableNames())
	if err != nil {
		return nil, err
	}
	buildNames, err := sanitizePRVariableNames(input.GetBuildArgumentNames())
	if err != nil {
		return nil, err
	}
	template := strings.TrimSpace(strings.ToLower(input.GetDomainTemplate()))
	if template == "" {
		template = defaultPRDomainTemplate
	}
	if err := validatePRDomainTemplate(template); err != nil {
		return nil, err
	}
	probe := &database.PullRequestDeployment{PullRequestNumber: int64(^uint64(0) >> 1), HeadRef: strings.Repeat("b", 20)}
	probeConfig := &database.PullRequestDeploymentConfig{DomainTemplate: template}
	if _, err := renderPRDomain(probeConfig, deployment, probe); err != nil {
		return nil, fmt.Errorf("domain template is too long for this deployment: %w", err)
	}
	maxActive := int32(input.GetMaxActivePreviews())
	if maxActive == 0 {
		maxActive = defaultPRMaxActive
	}
	if maxActive < 1 || maxActive > maxPRMaxActive {
		return nil, fmt.Errorf("max active previews must be between 1 and %d", maxPRMaxActive)
	}
	ttl := int32(input.GetTtlHours())
	if ttl == 0 {
		ttl = defaultPRTTLHours
	}
	if ttl < 1 || ttl > maxPRTTLHours {
		return nil, fmt.Errorf("preview lifetime must be between 1 and %d hours", maxPRTTLHours)
	}
	restoredTTL := int32(input.GetRestoredPreviewTtlHours())
	if restoredTTL == 0 {
		restoredTTL = defaultRestoredPRTTLHours
	}
	if restoredTTL < 1 || restoredTTL > maxRestoredPRTTLHours {
		return nil, fmt.Errorf("restored preview lifetime must be between 1 and %d hours", maxRestoredPRTTLHours)
	}
	forkPolicy := input.GetForkPolicy()
	if forkPolicy == deploymentsv1.PullRequestForkPolicy_PULL_REQUEST_FORK_POLICY_UNSPECIFIED {
		forkPolicy = deploymentsv1.PullRequestForkPolicy_PULL_REQUEST_FORK_DENY
	}
	if forkPolicy < deploymentsv1.PullRequestForkPolicy_PULL_REQUEST_FORK_DENY || forkPolicy > deploymentsv1.PullRequestForkPolicy_PULL_REQUEST_FORK_ISOLATED {
		return nil, fmt.Errorf("invalid fork policy")
	}
	now := time.Now()
	config := &database.PullRequestDeploymentConfig{
		DeploymentID: deployment.ID, OrganizationID: deployment.OrganizationID, Enabled: input.GetEnabled(),
		BaseBranches: mustJSON(baseBranches), IncludePaths: mustJSON(includePaths), ExcludePaths: mustJSON(excludePaths),
		DeployDrafts: input.GetDeployDrafts(), RedeployOnPush: input.GetRedeployOnPush(), CleanupOnClose: input.GetCleanupOnClose(),
		CommentEnabled: input.GetCommentEnabled(), DeploymentStatusEnabled: input.GetDeploymentStatusEnabled(), DomainTemplate: template,
		CheckRunEnabled:   input.GetCheckRunEnabled(),
		MaxActivePreviews: maxActive, TTLHours: ttl, RestoredPreviewTTLHours: restoredTTL, ForkPolicy: int32(forkPolicy), EnvironmentVariableNames: mustJSON(envNames), BuildArgumentNames: mustJSON(buildNames),
		RequireApproval: input.GetRequireApproval(), ApprovalCoversUpdates: input.GetApprovalCoversUpdates(), CreatedAt: now, UpdatedAt: now,
	}
	var existing database.PullRequestDeploymentConfig
	if err := database.DB.Where("deployment_id = ?", deployment.ID).First(&existing).Error; err == nil {
		config.CreatedAt = existing.CreatedAt
	}
	return config, nil
}

func sanitizePRPatterns(values []string, pathPattern bool) ([]string, error) {
	if len(values) > maxPRFilterCount {
		return nil, fmt.Errorf("no more than %d patterns are allowed", maxPRFilterCount)
	}
	result, seen := make([]string, 0, len(values)), map[string]bool{}
	for _, raw := range values {
		value := strings.TrimSpace(raw)
		if value == "" || len(value) > maxPRFilterLength || strings.ContainsAny(value, "\x00\r\n") {
			return nil, fmt.Errorf("invalid pattern %q", raw)
		}
		if pathPattern && (strings.HasPrefix(value, "/") || strings.Contains(value, "../")) {
			return nil, fmt.Errorf("path patterns must be repository-relative")
		}
		if _, err := compilePRGlob(value); err != nil {
			return nil, err
		}
		if !seen[value] {
			result, seen[value] = append(result, value), true
		}
	}
	sort.Strings(result)
	return result, nil
}

func sanitizePRVariableNames(values []string) ([]string, error) {
	if len(values) > maxPRFilterCount {
		return nil, fmt.Errorf("no more than %d variable names are allowed", maxPRFilterCount)
	}
	result, seen := make([]string, 0, len(values)), map[string]bool{}
	for _, raw := range values {
		value := strings.TrimSpace(raw)
		if !isSafeBuildArgName(value) {
			return nil, fmt.Errorf("invalid variable name %q", value)
		}
		if !seen[value] {
			result, seen[value] = append(result, value), true
		}
	}
	sort.Strings(result)
	return result, nil
}

func validatePRDomainTemplate(template string) error {
	if len(template) > 120 || !strings.Contains(template, "{pr}") || !strings.Contains(template, "{deployment}") {
		return fmt.Errorf("domain template must contain {pr} and {deployment}, and be 120 characters or fewer")
	}
	replaced := strings.NewReplacer("{pr}", "1", "{deployment}", "app", "{branch}", "branch").Replace(template)
	if strings.Contains(replaced, "{") || strings.Contains(replaced, "}") || !isDNSLabel(replaced) {
		return fmt.Errorf("domain template may use {pr}, {deployment}, and {branch}, plus lowercase letters, numbers, and dashes")
	}
	return nil
}

func isDNSLabel(value string) bool {
	if value == "" || len(value) > 63 || value[0] == '-' || value[len(value)-1] == '-' {
		return false
	}
	for _, r := range value {
		if (r < 'a' || r > 'z') && (r < '0' || r > '9') && r != '-' {
			return false
		}
	}
	return true
}

func mustJSON(values []string) string { data, _ := json.Marshal(values); return string(data) }
func parseStringList(value string) []string {
	var result []string
	_ = json.Unmarshal([]byte(value), &result)
	return result
}

func pullRequestConfigToProto(config *database.PullRequestDeploymentConfig) *deploymentsv1.PullRequestDeploymentConfig {
	return &deploymentsv1.PullRequestDeploymentConfig{
		DeploymentId: config.DeploymentID, Enabled: config.Enabled, BaseBranches: parseStringList(config.BaseBranches), IncludePaths: parseStringList(config.IncludePaths), ExcludePaths: parseStringList(config.ExcludePaths),
		DeployDrafts: config.DeployDrafts, RedeployOnPush: config.RedeployOnPush, CleanupOnClose: config.CleanupOnClose, CommentEnabled: config.CommentEnabled,
		DeploymentStatusEnabled: config.DeploymentStatusEnabled, DomainTemplate: config.DomainTemplate, MaxActivePreviews: uint32(config.MaxActivePreviews), TtlHours: uint32(config.TTLHours),
		RestoredPreviewTtlHours: uint32(config.RestoredPreviewTTLHours),
		ForkPolicy:              deploymentsv1.PullRequestForkPolicy(config.ForkPolicy), EnvironmentVariableNames: parseStringList(config.EnvironmentVariableNames), BuildArgumentNames: parseStringList(config.BuildArgumentNames),
		CreatedAt: timestamppb.New(config.CreatedAt), UpdatedAt: timestamppb.New(config.UpdatedAt), RequireApproval: config.RequireApproval, ApprovalCoversUpdates: config.ApprovalCoversUpdates, CheckRunEnabled: config.CheckRunEnabled,
	}
}

func pullRequestDeploymentToProto(record *database.PullRequestDeployment) *deploymentsv1.PullRequestDeployment {
	result := &deploymentsv1.PullRequestDeployment{Id: record.ID, SourceDeploymentId: record.SourceDeploymentID, PreviewDeploymentId: record.PreviewDeploymentID, Repository: record.Repository,
		PullRequestNumber: record.PullRequestNumber, HeadSha: record.HeadSHA, HeadRef: record.HeadRef, BaseRef: record.BaseRef, FromFork: record.FromFork,
		Status: deploymentsv1.PullRequestDeploymentStatus(record.Status), EnvironmentUrl: record.EnvironmentURL, StateUrl: pullRequestDeploymentStateURL(record), Error: record.Error, ApprovedBy: record.ApprovedBy, ApprovedHeadSha: record.ApprovedHeadSHA, Merged: record.Merged,
		ExpiresAt: timestamppb.New(record.ExpiresAt), CreatedAt: timestamppb.New(record.CreatedAt), UpdatedAt: timestamppb.New(record.UpdatedAt)}
	if record.ApprovedAt != nil {
		result.ApprovedAt = timestamppb.New(*record.ApprovedAt)
	}
	if record.ClosedAt != nil {
		result.ClosedAt = timestamppb.New(*record.ClosedAt)
	}
	return result
}

func compilePRGlob(pattern string) (*regexp.Regexp, error) {
	var out strings.Builder
	out.WriteString("^")
	for i := 0; i < len(pattern); i++ {
		switch pattern[i] {
		case '*':
			if i+1 < len(pattern) && pattern[i+1] == '*' {
				out.WriteString(".*")
				i++
			} else {
				out.WriteString("[^/]*")
			}
		case '?':
			out.WriteString("[^/]")
		default:
			out.WriteString(regexp.QuoteMeta(string(pattern[i])))
		}
	}
	out.WriteString("$")
	return regexp.Compile(out.String())
}

func matchesPRPatterns(value string, patterns []string) bool {
	for _, pattern := range patterns {
		if compiled, err := compilePRGlob(pattern); err == nil && compiled.MatchString(value) {
			return true
		}
	}
	return false
}

func containsPRStatus(statuses []int32, status int32) bool {
	for _, candidate := range statuses {
		if candidate == status {
			return true
		}
	}
	return false
}

func requiresPullRequestApproval(record *database.PullRequestDeployment, config *database.PullRequestDeploymentConfig) bool {
	return config.RequireApproval || (record.FromFork && deploymentsv1.PullRequestForkPolicy(config.ForkPolicy) == deploymentsv1.PullRequestForkPolicy_PULL_REQUEST_FORK_REQUIRE_APPROVAL)
}

func pullRequestDeploymentApproved(record *database.PullRequestDeployment, config *database.PullRequestDeploymentConfig) bool {
	if !requiresPullRequestApproval(record, config) {
		return true
	}
	if record.ApprovedAt == nil || record.ApprovedHeadSHA == nil {
		return false
	}
	return config.ApprovalCoversUpdates || *record.ApprovedHeadSHA == record.HeadSHA
}

func pullRequestDeploymentCanRestore(record *database.PullRequestDeployment, config *database.PullRequestDeploymentConfig) bool {
	if record == nil || config == nil || !record.Merged || record.ClosedAt == nil || record.ApprovedAt == nil || record.ApprovedHeadSHA == nil {
		return false
	}
	return config.ApprovalCoversUpdates || *record.ApprovedHeadSHA == record.HeadSHA
}

func renderPRDomain(config *database.PullRequestDeploymentConfig, source *database.Deployment, record *database.PullRequestDeployment) (string, error) {
	deploymentLabel := slugifyPRLabel(database.DefaultMyObienteCloudLabel(source.ID))
	branchLabel := slugifyPRLabel(record.HeadRef)
	if len(branchLabel) > 20 {
		branchLabel = strings.Trim(branchLabel[:20], "-")
	}
	label := strings.NewReplacer("{pr}", fmt.Sprintf("%d", record.PullRequestNumber), "{deployment}", deploymentLabel, "{branch}", branchLabel).Replace(config.DomainTemplate)
	if len(label) > 63 {
		return "", fmt.Errorf("domain template produced a hostname label longer than 63 characters")
	}
	if !isDNSLabel(label) {
		return "", fmt.Errorf("domain template produced an invalid hostname label")
	}
	return label + ".my.obiente.cloud", nil
}

func slugifyPRLabel(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	var out strings.Builder
	dash := false
	for _, r := range value {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			out.WriteRune(r)
			dash = false
		} else if !dash && out.Len() > 0 {
			out.WriteByte('-')
			dash = true
		}
	}
	return strings.Trim(out.String(), "-")
}

func (s *Service) deployPullRequestEnvironment(recordID string) {
	ctx, cancel := s.detachedContext(10 * time.Minute)
	defer cancel()
	var record database.PullRequestDeployment
	if err := database.DB.WithContext(ctx).Where("id = ?", recordID).First(&record).Error; err != nil {
		return
	}
	var config database.PullRequestDeploymentConfig
	if err := database.DB.WithContext(ctx).Where("deployment_id = ?", record.SourceDeploymentID).First(&config).Error; err != nil {
		s.failPullRequestDeployment(ctx, &record, err)
		return
	}
	if requiresPullRequestApproval(&record, &config) && !pullRequestDeploymentApproved(&record, &config) {
		result := database.DB.WithContext(ctx).Model(&database.PullRequestDeployment{}).
			Where("id = ? AND closed_at IS NULL AND head_sha = ? AND active_head_sha IS NULL", record.ID, record.HeadSHA).
			Updates(map[string]interface{}{"status": int32(deploymentsv1.PullRequestDeploymentStatus_PULL_REQUEST_DEPLOYMENT_WAITING_APPROVAL), "updated_at": time.Now()})
		if result.Error != nil || result.RowsAffected != 1 {
			return
		}
		s.reportPullRequestDeployment(record.ID)
		return
	}
	claim := database.DB.WithContext(ctx).Model(&database.PullRequestDeployment{}).
		Where("id = ? AND active_head_sha IS NULL AND closed_at IS NULL", record.ID).
		Updates(map[string]interface{}{"active_head_sha": record.HeadSHA, "status": int32(deploymentsv1.PullRequestDeploymentStatus_PULL_REQUEST_DEPLOYMENT_QUEUED), "updated_at": time.Now()})
	if claim.Error != nil {
		s.failPullRequestDeployment(ctx, &record, claim.Error)
		return
	}
	if claim.RowsAffected == 0 {
		s.reportPullRequestDeployment(record.ID)
		return
	}
	record.ActiveHeadSHA = &record.HeadSHA
	buildHeadSHA := record.HeadSHA
	source, err := s.repo.GetByID(ctx, record.SourceDeploymentID)
	if err != nil {
		s.failPullRequestDeployment(ctx, &record, err)
		return
	}
	preview, err := s.ensurePreviewDeployment(ctx, source, &config, &record)
	if err != nil {
		s.failPullRequestDeployment(ctx, &record, err)
		return
	}
	record.PreviewDeploymentID = &preview.ID
	record.Status = int32(deploymentsv1.PullRequestDeploymentStatus_PULL_REQUEST_DEPLOYMENT_QUEUED)
	record.Error = nil
	record.UpdatedAt = time.Now()
	updated := database.DB.WithContext(ctx).Model(&database.PullRequestDeployment{}).
		Where("id = ? AND closed_at IS NULL AND active_head_sha = ? AND head_sha = ?", record.ID, record.HeadSHA, record.HeadSHA).
		Updates(map[string]interface{}{"preview_deployment_id": preview.ID, "environment_url": record.EnvironmentURL, "status": record.Status, "error": nil, "updated_at": record.UpdatedAt})
	if updated.Error != nil {
		s.failPullRequestDeployment(ctx, &record, updated.Error)
		return
	}
	if updated.RowsAffected == 0 {
		if err := database.DB.WithContext(ctx).Where("id = ?", record.ID).First(&record).Error; err != nil || record.ClosedAt != nil || record.ActiveHeadSHA == nil || *record.ActiveHeadSHA != buildHeadSHA {
			return
		}
	}
	s.reportPullRequestDeployment(record.ID)
	systemCtx := auth.WithSystemUser(ctx)
	if _, err := s.TriggerDeployment(systemCtx, connect.NewRequest(&deploymentsv1.TriggerDeploymentRequest{DeploymentId: preview.ID, OrganizationId: preview.OrganizationID, CommitSha: &buildHeadSHA})); err != nil {
		s.failPullRequestDeployment(ctx, &record, err)
	}
}

func (s *Service) ensurePreviewDeployment(ctx context.Context, source *database.Deployment, config *database.PullRequestDeploymentConfig, record *database.PullRequestDeployment) (*database.Deployment, error) {
	var preview *database.Deployment
	err := withDistributedLock(ctx, "organization-quota:"+source.OrganizationID, func() error {
		requested := previewRequestedResources(source)
		if record.PreviewDeploymentID != nil {
			requested.ExcludeDeploymentID = *record.PreviewDeploymentID
		}
		if err := s.quotaChecker.CanAllocate(ctx, source.OrganizationID, requested); err != nil {
			return fmt.Errorf("preview quota check failed: %w", err)
		}

		if record.PreviewDeploymentID != nil {
			if existing, err := s.repo.GetByID(ctx, *record.PreviewDeploymentID); err == nil {
				if err := refreshPreviewDeployment(existing, source, config, record); err != nil {
					return err
				}
				if err := s.repo.Update(ctx, existing); err != nil {
					return err
				}
				if err := ensurePreviewRouting(ctx, existing, source); err != nil {
					return err
				}
				url, err := previewEnvironmentURL(ctx, existing)
				if err != nil {
					return err
				}
				record.EnvironmentURL = &url
				preview = existing
				return nil
			}
		}

		created := database.Deployment{ID: "deployment-" + uuid.NewString()}
		if err := refreshPreviewDeployment(&created, source, config, record); err != nil {
			return err
		}
		if err := database.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
			var lockedConfig database.PullRequestDeploymentConfig
			if lockErr := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Select("deployment_id").Where("deployment_id = ?", source.ID).First(&lockedConfig).Error; lockErr != nil {
				return lockErr
			}
			active, countErr := countActivePullRequestPreviews(tx, source.ID, record.ID)
			if countErr != nil {
				return countErr
			}
			if active >= int64(config.MaxActivePreviews) {
				return fmt.Errorf("maximum of %d active pull request environments reached", config.MaxActivePreviews)
			}
			if createErr := tx.Create(&created).Error; createErr != nil {
				return createErr
			}
			claimed := tx.Model(&database.PullRequestDeployment{}).
				Where("id = ? AND closed_at IS NULL AND active_head_sha = ?", record.ID, record.HeadSHA).
				Update("preview_deployment_id", created.ID)
			if claimed.Error != nil {
				return claimed.Error
			}
			if claimed.RowsAffected != 1 {
				return fmt.Errorf("pull request environment changed while its runtime was being created")
			}
			return nil
		}); err != nil {
			return err
		}
		if err := ensurePreviewRouting(ctx, &created, source); err != nil {
			return err
		}
		url, err := previewEnvironmentURL(ctx, &created)
		if err != nil {
			return err
		}
		record.EnvironmentURL = &url
		preview = &created
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("failed to reserve preview deployment: %w", err)
	}
	return preview, nil
}

func countActivePullRequestPreviews(db *gorm.DB, sourceDeploymentID, excludedRecordID string) (int64, error) {
	activeStatuses := []int32{
		int32(deploymentsv1.PullRequestDeploymentStatus_PULL_REQUEST_DEPLOYMENT_QUEUED),
		int32(deploymentsv1.PullRequestDeploymentStatus_PULL_REQUEST_DEPLOYMENT_BUILDING),
		int32(deploymentsv1.PullRequestDeploymentStatus_PULL_REQUEST_DEPLOYMENT_RUNNING),
	}
	var active int64
	result := db.Model(&database.PullRequestDeployment{}).
		Where(`source_deployment_id = ? AND id <> ? AND closed_at IS NULL AND preview_deployment_id IS NOT NULL
			AND (status IN ? OR EXISTS (
				SELECT 1 FROM deployment_locations
				WHERE deployment_locations.deployment_id = pull_request_deployments.preview_deployment_id
				  AND deployment_locations.status = ?
			))`, sourceDeploymentID, excludedRecordID, activeStatuses, "running").
		Count(&active)
	return active, result.Error
}

func previewEnvironmentURL(ctx context.Context, preview *database.Deployment) (string, error) {
	if preview == nil || strings.TrimSpace(preview.Domain) == "" {
		return "", fmt.Errorf("preview domain is unavailable")
	}
	result := "https://" + strings.TrimSpace(preview.Domain)
	var routing database.DeploymentRouting
	if err := database.DB.WithContext(ctx).
		Where("deployment_id = ? AND domain = ?", preview.ID, preview.Domain).
		Order("created_at ASC, id ASC").
		First(&routing).Error; err != nil {
		return "", fmt.Errorf("load preview route for its public URL: %w", err)
	}
	prefix := strings.TrimSpace(routing.PathPrefix)
	if prefix == "" || prefix == "/" {
		return result, nil
	}
	if !strings.HasPrefix(prefix, "/") {
		prefix = "/" + prefix
	}
	return result + prefix, nil
}

func ensurePreviewRouting(ctx context.Context, preview, source *database.Deployment) error {
	if preview == nil || source == nil {
		return fmt.Errorf("preview routing template is incomplete")
	}

	var sourceRoutings []database.DeploymentRouting
	if err := database.DB.WithContext(ctx).
		Where("deployment_id = ?", source.ID).
		Order("created_at ASC, id ASC").
		Find(&sourceRoutings).Error; err != nil {
		return fmt.Errorf("load source Compose routing: %w", err)
	}
	routing, err := previewRouting(preview, source, sourceRoutings)
	if err != nil {
		return err
	}
	if err := database.UpsertDeploymentRouting(routing); err != nil {
		return fmt.Errorf("provision preview routing: %w", err)
	}
	return nil
}

func previewRouting(preview, source *database.Deployment, sourceRoutings []database.DeploymentRouting) (*database.DeploymentRouting, error) {
	if preview != nil && (preview.BuildStrategy == int32(deploymentsv1.BuildStrategy_COMPOSE_REPO) || preview.BuildStrategy == int32(deploymentsv1.BuildStrategy_PLAIN_COMPOSE)) {
		return previewComposeRouting(preview, source, sourceRoutings)
	}
	return previewSingleServiceRouting(preview, source, sourceRoutings)
}

func previewSingleServiceRouting(preview, source *database.Deployment, sourceRoutings []database.DeploymentRouting) (*database.DeploymentRouting, error) {
	if preview == nil || source == nil || strings.TrimSpace(preview.Domain) == "" {
		return nil, fmt.Errorf("preview routing requires a deployment and domain")
	}
	template := preferredPreviewSourceRouting(source, sourceRoutings)
	targetPort := 0
	certResolver, middleware, pathPrefix := "letsencrypt", "{}", ""
	if template != nil {
		targetPort = template.TargetPort
		if strings.TrimSpace(template.SSLCertResolver) != "" && template.SSLCertResolver != "internal" {
			certResolver = template.SSLCertResolver
		}
		if strings.TrimSpace(template.Middleware) != "" {
			middleware = template.Middleware
		}
		pathPrefix = template.PathPrefix
	} else if source.Port != nil {
		targetPort = int(*source.Port)
	}
	if targetPort <= 0 {
		return nil, fmt.Errorf("source deployment has no routable service port")
	}
	now := time.Now()
	return &database.DeploymentRouting{
		ID:              fmt.Sprintf("route-%s-default", preview.ID),
		DeploymentID:    preview.ID,
		Domain:          preview.Domain,
		ServiceName:     "default",
		PathPrefix:      pathPrefix,
		TargetPort:      targetPort,
		Protocol:        "https",
		SSLEnabled:      true,
		SSLCertResolver: certResolver,
		Middleware:      middleware,
		CreatedAt:       now,
		UpdatedAt:       now,
	}, nil
}

func previewComposeRouting(preview, source *database.Deployment, sourceRoutings []database.DeploymentRouting) (*database.DeploymentRouting, error) {
	if preview == nil || source == nil || strings.TrimSpace(preview.Domain) == "" {
		return nil, fmt.Errorf("preview Compose routing requires a deployment and domain")
	}

	template := preferredPreviewSourceRouting(source, sourceRoutings)

	targetPort := 0
	serviceName, certResolver, middleware := "default", "letsencrypt", "{}"
	pathPrefix := ""
	if template != nil {
		targetPort = template.TargetPort
		if strings.TrimSpace(template.ServiceName) != "" {
			serviceName = template.ServiceName
		}
		if strings.TrimSpace(template.SSLCertResolver) != "" && template.SSLCertResolver != "internal" {
			certResolver = template.SSLCertResolver
		}
		if strings.TrimSpace(template.Middleware) != "" {
			middleware = template.Middleware
		}
		pathPrefix = template.PathPrefix
	} else if source.Port != nil {
		targetPort = int(*source.Port)
	}
	if targetPort <= 0 {
		return nil, fmt.Errorf("source Compose deployment has no routable service port")
	}
	if serviceName == "default" {
		resolvedServiceName, err := previewComposeServiceName(source.ComposeYaml, targetPort)
		if err != nil {
			return nil, err
		}
		serviceName = resolvedServiceName
	}

	now := time.Now()
	return &database.DeploymentRouting{
		ID:              fmt.Sprintf("route-%s-default", preview.ID),
		DeploymentID:    preview.ID,
		Domain:          preview.Domain,
		ServiceName:     serviceName,
		PathPrefix:      pathPrefix,
		TargetPort:      targetPort,
		Protocol:        "https",
		SSLEnabled:      true,
		SSLCertResolver: certResolver,
		Middleware:      middleware,
		CreatedAt:       now,
		UpdatedAt:       now,
	}, nil
}

func preferredPreviewSourceRouting(source *database.Deployment, sourceRoutings []database.DeploymentRouting) *database.DeploymentRouting {
	if source == nil {
		return nil
	}
	var template *database.DeploymentRouting
	for i := range sourceRoutings {
		candidate := &sourceRoutings[i]
		if candidate.TargetPort <= 0 {
			continue
		}
		if template == nil || (source.Domain != "" && candidate.Domain == source.Domain) {
			template = candidate
		}
		if source.Domain != "" && candidate.Domain == source.Domain {
			break
		}
	}
	return template
}

func previewComposeServiceName(composeYaml string, targetPort int) (string, error) {
	serviceNames, err := ExtractServiceNames(composeYaml)
	if err != nil {
		return "", fmt.Errorf("read source Compose services: %w", err)
	}
	actualNames := make([]string, 0, len(serviceNames))
	for _, serviceName := range serviceNames {
		if serviceName != "" && serviceName != "default" {
			actualNames = append(actualNames, serviceName)
		}
	}
	if len(actualNames) == 1 {
		return actualNames[0], nil
	}

	matchingNames := make([]string, 0, len(actualNames))
	for _, serviceName := range actualNames {
		port, portErr := ExtractServicePort(composeYaml, serviceName)
		if portErr == nil && port == targetPort {
			matchingNames = append(matchingNames, serviceName)
		}
	}
	if len(matchingNames) == 1 {
		return matchingNames[0], nil
	}
	return "", fmt.Errorf("source Compose routing must identify one target service")
}

func resolvePreviewComposeRoutingForRevision(ctx context.Context, preview *database.Deployment, composeYaml string) error {
	if preview == nil || !deploymentIsPullRequestPreview(preview) || preview.BuildStrategy != int32(deploymentsv1.BuildStrategy_COMPOSE_REPO) {
		return nil
	}
	var routings []database.DeploymentRouting
	if err := database.DB.WithContext(ctx).
		Where("deployment_id = ?", preview.ID).
		Order("created_at ASC, id ASC").
		Find(&routings).Error; err != nil {
		return fmt.Errorf("load preview routing for current revision: %w", err)
	}
	if len(routings) == 0 {
		return fmt.Errorf("preview routing is unavailable for the current revision")
	}
	for i := range routings {
		routing := &routings[i]
		serviceName, targetPort, resolveErr := resolvePreviewComposeTarget(composeYaml, routing.ServiceName, routing.TargetPort)
		if resolveErr != nil {
			return fmt.Errorf("resolve route %s against the current preview Compose file: %w", routing.ID, resolveErr)
		}
		if routing.ServiceName == serviceName && routing.TargetPort == targetPort {
			continue
		}
		routing.ServiceName = serviceName
		routing.TargetPort = targetPort
		routing.UpdatedAt = time.Now()
		if err := database.UpsertDeploymentRouting(routing); err != nil {
			return fmt.Errorf("update preview route %s for the current revision: %w", routing.ID, err)
		}
	}
	return nil
}

func resolvePreviewComposeTarget(composeYaml, previousService string, previousPort int) (string, int, error) {
	serviceNames, err := ExtractServiceNames(composeYaml)
	if err != nil {
		return "", 0, err
	}
	actualNames := make([]string, 0, len(serviceNames))
	for _, name := range serviceNames {
		if name != "" && name != "default" {
			actualNames = append(actualNames, name)
		}
	}
	for _, name := range actualNames {
		if name != previousService {
			continue
		}
		if port, portErr := ExtractServicePort(composeYaml, name); portErr == nil && port > 0 {
			return name, port, nil
		}
		if previousPort > 0 {
			return name, previousPort, nil
		}
	}
	matchingNames := make([]string, 0, len(actualNames))
	for _, name := range actualNames {
		if port, portErr := ExtractServicePort(composeYaml, name); portErr == nil && port == previousPort {
			matchingNames = append(matchingNames, name)
		}
	}
	if len(matchingNames) == 1 {
		return matchingNames[0], previousPort, nil
	}
	if len(actualNames) == 1 {
		port := previousPort
		if detected, portErr := ExtractServicePort(composeYaml, actualNames[0]); portErr == nil && detected > 0 {
			port = detected
		}
		if port > 0 {
			return actualNames[0], port, nil
		}
	}
	return "", 0, fmt.Errorf("current Compose revision does not identify one routed service and target port")
}

func refreshPreviewDeployment(preview, source *database.Deployment, config *database.PullRequestDeploymentConfig, record *database.PullRequestDeployment) error {
	if preview == nil || source == nil || config == nil || record == nil {
		return fmt.Errorf("preview template is incomplete")
	}
	id, createdAt := preview.ID, preview.CreatedAt
	existingComposeYAML := preview.ComposeYaml
	if id == "" {
		id = "deployment-" + uuid.NewString()
	}
	if createdAt.IsZero() {
		createdAt = time.Now()
	}
	*preview = *source
	preview.ID = id
	preview.Name = fmt.Sprintf("%s · PR #%d", source.Name, record.PullRequestNumber)
	preview.Branch = record.HeadRef
	preview.CustomDomains = "[]"
	preview.Groups = mustJSON([]string{"pull-request", fmt.Sprintf("pr-%d", record.PullRequestNumber)})
	preview.Environment = int32(deploymentsv1.Environment_PULL_REQUEST)
	preview.DockerfileVolumes = "[]"
	preview.Status = int32(deploymentsv1.DeploymentStatus_DEPLOYING)
	preview.HealthStatus = "pending"
	preview.Image = nil
	if preview.BuildStrategy == int32(deploymentsv1.BuildStrategy_COMPOSE_REPO) || preview.BuildStrategy == int32(deploymentsv1.BuildStrategy_PLAIN_COMPOSE) {
		if existingComposeYAML != "" {
			preview.ComposeYaml = existingComposeYAML
		}
	} else {
		preview.ComposeYaml = ""
	}
	preview.CreatedBy = "system"
	preview.CreatedAt = createdAt
	preview.LastDeployedAt = time.Now()
	preview.DeletedAt = nil
	autoDeploy := false
	preview.AutoDeploy = &autoDeploy
	domain, err := renderPRDomain(config, source, record)
	if err != nil {
		return err
	}
	preview.Domain = domain
	preview.EnvFileContent = ""
	preview.EnvVars, preview.BuildArgs = previewScopedVariables(source, config, record.FromFork)
	return nil
}

func previewRequestedResources(source *database.Deployment) quota.RequestedResources {
	replicas := 1
	if source.Replicas != nil && *source.Replicas > 0 {
		replicas = int(*source.Replicas)
	}
	memoryBytes := int64(512 * 1024 * 1024)
	if source.MemoryBytes != nil && *source.MemoryBytes > 0 {
		memoryBytes = *source.MemoryBytes
	}
	cpuShares := int64(256)
	if source.CPUShares != nil && *source.CPUShares > 0 {
		cpuShares = *source.CPUShares
	}
	return quota.RequestedResources{Replicas: replicas, MemoryBytes: memoryBytes, CPUshares: cpuShares}
}

func filterJSONVariables(raw string, allowed []string) string {
	values := parseEnvVars(raw)
	filtered := make(map[string]string, len(allowed))
	for _, name := range allowed {
		if value, ok := values[name]; ok {
			filtered[name] = value
		}
	}
	data, _ := json.Marshal(filtered)
	return string(data)
}

func previewScopedVariables(source *database.Deployment, config *database.PullRequestDeploymentConfig, fromFork bool) (string, string) {
	if source == nil || config == nil || fromFork {
		return "{}", "{}"
	}
	return filterJSONVariables(source.EnvVars, parseStringList(config.EnvironmentVariableNames)), filterJSONVariables(source.BuildArgs, parseStringList(config.BuildArgumentNames))
}

func deploymentIsPullRequestPreview(deployment *database.Deployment) bool {
	return deployment != nil && deployment.Environment == int32(deploymentsv1.Environment_PULL_REQUEST)
}

func (s *Service) failPullRequestDeployment(ctx context.Context, record *database.PullRequestDeployment, err error) {
	message := strings.TrimSpace(err.Error())
	if len(message) > 2000 {
		message = message[:2000]
	}
	var current database.PullRequestDeployment
	if loadErr := database.DB.WithContext(ctx).Where("id = ?", record.ID).First(&current).Error; loadErr != nil || current.ClosedAt != nil {
		return
	}
	expectedHeadSHA := stringValue(record.ActiveHeadSHA)
	query := database.DB.WithContext(ctx).Model(&database.PullRequestDeployment{}).Where("id = ? AND closed_at IS NULL", record.ID)
	if expectedHeadSHA == "" {
		query = query.Where("active_head_sha IS NULL")
	} else {
		if current.ActiveHeadSHA == nil || *current.ActiveHeadSHA != expectedHeadSHA {
			return
		}
		query = query.Where("active_head_sha = ?", expectedHeadSHA)
		if current.HeadSHA != expectedHeadSHA {
			result := query.Updates(map[string]interface{}{"active_head_sha": nil, "status": int32(deploymentsv1.PullRequestDeploymentStatus_PULL_REQUEST_DEPLOYMENT_QUEUED), "error": nil, "updated_at": time.Now()})
			if result.Error == nil && result.RowsAffected == 1 {
				go s.deployPullRequestEnvironment(record.ID)
			}
			return
		}
	}
	result := query.Updates(map[string]interface{}{"active_head_sha": nil, "status": int32(deploymentsv1.PullRequestDeploymentStatus_PULL_REQUEST_DEPLOYMENT_FAILED), "error": message, "updated_at": time.Now()})
	if result.Error == nil && result.RowsAffected == 1 {
		go s.reportPullRequestDeployment(record.ID)
	}
}

func (s *Service) updatePullRequestDeploymentRuntime(ctx context.Context, previewDeploymentID, buildHeadSHA string, status deploymentsv1.PullRequestDeploymentStatus, message string) {
	var record database.PullRequestDeployment
	if err := database.DB.WithContext(ctx).Select("id").Where("preview_deployment_id = ? AND closed_at IS NULL", previewDeploymentID).First(&record).Error; err != nil {
		return
	}
	terminal := status == deploymentsv1.PullRequestDeploymentStatus_PULL_REQUEST_DEPLOYMENT_RUNNING || status == deploymentsv1.PullRequestDeploymentStatus_PULL_REQUEST_DEPLOYMENT_FAILED
	updates := map[string]interface{}{"status": int32(status), "updated_at": time.Now()}
	if status == deploymentsv1.PullRequestDeploymentStatus_PULL_REQUEST_DEPLOYMENT_RUNNING {
		updates["isolation_version"] = currentPRIsolationVersion
	}
	if terminal {
		updates["active_head_sha"] = nil
	}
	if message == "" {
		updates["error"] = nil
	} else {
		updates["error"] = message
	}
	result := database.DB.WithContext(ctx).Model(&database.PullRequestDeployment{}).
		Where("id = ? AND closed_at IS NULL AND active_head_sha = ? AND head_sha = ?", record.ID, buildHeadSHA, buildHeadSHA).
		Updates(updates)
	if result.Error != nil {
		logger.Warn("[PRDeployments] Failed to persist runtime state for %s: %v", record.ID, result.Error)
		return
	}
	if result.RowsAffected == 1 {
		go s.reportPullRequestDeployment(record.ID)
		return
	}
	if !terminal {
		return
	}
	// A newer webhook may have advanced head_sha while this build was still
	// running. Retire only the matching old build lease and queue the current
	// head without writing any fields from the stale callback snapshot.
	queued := database.DB.WithContext(ctx).Model(&database.PullRequestDeployment{}).
		Where("id = ? AND closed_at IS NULL AND active_head_sha = ? AND head_sha <> ?", record.ID, buildHeadSHA, buildHeadSHA).
		Updates(map[string]interface{}{"active_head_sha": nil, "status": int32(deploymentsv1.PullRequestDeploymentStatus_PULL_REQUEST_DEPLOYMENT_QUEUED), "error": nil, "updated_at": time.Now()})
	if queued.Error != nil {
		logger.Warn("[PRDeployments] Failed to queue the current revision for %s: %v", record.ID, queued.Error)
	} else if queued.RowsAffected == 1 {
		go s.deployPullRequestEnvironment(record.ID)
	}
}

func (s *Service) cleanupPullRequestDeployment(ctx context.Context, record *database.PullRequestDeployment, reason string) error {
	if record.PreviewDeploymentID != nil {
		_, err := s.DeleteDeployment(auth.WithSystemUser(ctx), connect.NewRequest(&deploymentsv1.DeleteDeploymentRequest{DeploymentId: *record.PreviewDeploymentID, OrganizationId: record.OrganizationID}))
		if err != nil && connect.CodeOf(err) != connect.CodeNotFound {
			return fmt.Errorf("failed to remove preview deployment: %w", err)
		}
	}
	now := time.Now()
	record.Status, record.ActiveHeadSHA, record.ClosedAt, record.UpdatedAt = int32(deploymentsv1.PullRequestDeploymentStatus_PULL_REQUEST_DEPLOYMENT_CLOSED), nil, &now, now
	if reason != "" {
		record.Error = &reason
	}
	if err := database.DB.WithContext(ctx).Save(record).Error; err != nil {
		return err
	}
	go s.reportPullRequestDeployment(record.ID)
	return nil
}

func (s *Service) cleanupPullRequestDeploymentsForSource(ctx context.Context, sourceDeploymentID string) error {
	activePreviewIDs := database.DB.Model(&database.Deployment{}).Select("id").Where("deleted_at IS NULL")
	var records []database.PullRequestDeployment
	if err := database.DB.WithContext(ctx).
		Where("source_deployment_id = ? AND (closed_at IS NULL OR preview_deployment_id IN (?))", sourceDeploymentID, activePreviewIDs).
		Find(&records).Error; err != nil {
		return err
	}
	for i := range records {
		if records[i].PreviewDeploymentID != nil && *records[i].PreviewDeploymentID == sourceDeploymentID {
			return fmt.Errorf("pull request preview %s cannot reference its source as its runtime", records[i].ID)
		}
		if err := s.cleanupPullRequestDeployment(ctx, &records[i], "The source deployment was removed from Obiente."); err != nil {
			return fmt.Errorf("cleanup pull request preview %s: %w", records[i].ID, err)
		}
		// The source and its reporting configuration are about to be deleted, so
		// wait for one final GitHub update while that context is still available.
		s.reportPullRequestDeployment(records[i].ID)
	}
	return nil
}

func (s *Service) StartPullRequestDeploymentJanitor(ctx context.Context) {
	go func() {
		s.correctPullRequestPreviewEnvironments()
		s.migrateLegacyPullRequestPreviewIsolation(ctx)
		s.cleanupExpiredPullRequestDeployments(ctx)
		s.retryPendingPullRequestReconciliations(ctx)
		s.retryPendingPullRequestReports(ctx)
		cleanupTicker := time.NewTicker(15 * time.Minute)
		retryTicker := time.NewTicker(time.Minute)
		defer cleanupTicker.Stop()
		defer retryTicker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-cleanupTicker.C:
				s.cleanupExpiredPullRequestDeployments(ctx)
			case <-retryTicker.C:
				s.correctPullRequestPreviewEnvironments()
				s.migrateLegacyPullRequestPreviewIsolation(ctx)
				s.retryPendingPullRequestReconciliations(ctx)
				s.retryPendingPullRequestReports(ctx)
			}
		}
	}()
	go func() {
		s.backfillExistingOpenPullRequests(ctx)
		ticker := time.NewTicker(time.Minute)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				s.backfillExistingOpenPullRequests(ctx)
			}
		}
	}()
}

func (s *Service) correctPullRequestPreviewEnvironments() {
	if err := database.BackfillPullRequestPreviewEnvironments(database.DB); err != nil {
		logger.Warn("[PRDeployments] Failed to correct pull request preview environments during rolling update: %v", err)
	}
}

func (s *Service) migrateLegacyPullRequestPreviewIsolation(ctx context.Context) {
	const retryDelay = 5 * time.Minute
	candidates, err := loadPullRequestIsolationMigrationCandidates(ctx, time.Now().Add(-retryDelay))
	if err != nil {
		logger.Warn("[PRDeployments] Failed to load legacy preview runtimes for runtime migration: %v", err)
		return
	}
	for _, candidate := range candidates {
		if candidate.IsolationVersion >= 0 && candidate.IsolationVersion < currentPRIsolationVersion {
			marked := database.DB.WithContext(ctx).Model(&database.PullRequestDeployment{}).
				Where("id = ? AND closed_at IS NULL AND active_head_sha IS NULL AND status = ? AND isolation_version = ?", candidate.ID, int32(deploymentsv1.PullRequestDeploymentStatus_PULL_REQUEST_DEPLOYMENT_RUNNING), candidate.IsolationVersion).
				Update("isolation_version", pendingPRIsolationVersion)
			if marked.Error != nil {
				logger.Warn("[PRDeployments] Failed to mark preview %s for runtime migration: %v", candidate.ID, marked.Error)
				continue
			}
			if marked.RowsAffected != 1 {
				continue
			}
		}
		go s.deployPullRequestEnvironment(candidate.ID)
		// Rebuild one existing preview per janitor pass so a runtime migration
		// cannot start every repository build at once on a small deployment node.
		return
	}
}

func loadPullRequestIsolationMigrationCandidates(ctx context.Context, retryBefore time.Time) ([]isolationMigrationCandidate, error) {
	var candidates []isolationMigrationCandidate
	if err := database.DB.WithContext(ctx).Model(&database.PullRequestDeployment{}).
		Select("pull_request_deployments.id, pull_request_deployments.isolation_version").
		Joins("JOIN deployments AS preview ON preview.id = pull_request_deployments.preview_deployment_id").
		Joins("JOIN pull_request_deployment_configs AS config ON config.deployment_id = pull_request_deployments.source_deployment_id").
		Where("pull_request_deployments.closed_at IS NULL AND pull_request_deployments.active_head_sha IS NULL").
		Where("(pull_request_deployments.isolation_version >= ? AND pull_request_deployments.isolation_version < ? AND pull_request_deployments.status = ?) OR (pull_request_deployments.isolation_version = ? AND pull_request_deployments.status IN ? AND pull_request_deployments.updated_at <= ?)",
			int32(0), currentPRIsolationVersion, int32(deploymentsv1.PullRequestDeploymentStatus_PULL_REQUEST_DEPLOYMENT_RUNNING),
			pendingPRIsolationVersion,
			[]int32{
				int32(deploymentsv1.PullRequestDeploymentStatus_PULL_REQUEST_DEPLOYMENT_RUNNING),
				int32(deploymentsv1.PullRequestDeploymentStatus_PULL_REQUEST_DEPLOYMENT_FAILED),
			},
			retryBefore).
		Where("config.enabled = ?", true).
		Where("preview.deleted_at IS NULL AND preview.environment = ?", int32(deploymentsv1.Environment_PULL_REQUEST)).
		Order("pull_request_deployments.id ASC").Limit(25).
		Scan(&candidates).Error; err != nil {
		return nil, err
	}
	return candidates, nil
}

func (s *Service) backfillExistingOpenPullRequests(ctx context.Context) {
	lastDeploymentID := ""
	for {
		var configs []database.PullRequestDeploymentConfig
		query := database.DB.WithContext(ctx).
			Where("enabled = ? AND open_pull_requests_synced_at IS NULL", true).
			Order("deployment_id ASC").Limit(25)
		if lastDeploymentID != "" {
			query = query.Where("deployment_id > ?", lastDeploymentID)
		}
		if err := query.Find(&configs).Error; err != nil {
			logger.Warn("[PRDeployments] Failed to load existing settings for open pull request backfill: %v", err)
			return
		}
		if len(configs) == 0 {
			return
		}
		for i := range configs {
			config := configs[i]
			lastDeploymentID = config.DeploymentID
			source, err := s.repo.GetByID(ctx, config.DeploymentID)
			if err != nil {
				logger.Warn("[PRDeployments] Failed to load source %s for open pull request backfill: %v", config.DeploymentID, err)
				continue
			}
			syncSource, err := loadPullRequestSyncSource(ctx, source)
			if err != nil {
				logger.Warn("[PRDeployments] GitHub integration is unavailable for open pull request backfill on %s", config.DeploymentID)
				continue
			}
			if err := s.backfillOpenPullRequestsForConfig(ctx, source, &config, syncSource, nil); err != nil {
				logger.Warn("[PRDeployments] Existing open pull request backfill failed for %s: %v", config.DeploymentID, err)
				continue
			}
			if _, err := markOpenPullRequestsSynced(ctx, &config, syncSource); err != nil {
				logger.Warn("[PRDeployments] Failed to record open pull request backfill for %s: %v", config.DeploymentID, err)
			}
		}
		if len(configs) < 25 {
			return
		}
	}
}

func (s *Service) retryPendingPullRequestReports(ctx context.Context) {
	var recordIDs []string
	if err := database.DB.WithContext(ctx).Model(&database.PullRequestDeployment{}).
		Where("report_pending = ? AND (next_report_at IS NULL OR next_report_at <= ?)", true, time.Now()).
		Order("next_report_at ASC, id ASC").
		Limit(100).
		Pluck("id", &recordIDs).Error; err != nil {
		logger.Warn("[PRDeployments] Failed to load pending GitHub reports: %v", err)
		return
	}
	for _, recordID := range recordIDs {
		go s.reportPullRequestDeployment(recordID)
	}
}

func pullRequestSyncSourceGuard(query *gorm.DB, syncSource *pullRequestSyncSource) *gorm.DB {
	if syncSource == nil {
		return query.Where("1 = 0")
	}
	return query.Where(`EXISTS (
		SELECT 1
		FROM deployments AS source
		JOIN github_integrations AS integration ON integration.id = source.github_integration_id
		WHERE source.id = pull_request_deployment_configs.deployment_id
		  AND source.deleted_at IS NULL
		  AND source.repository_url = ?
		  AND source.github_integration_id = ?
		  AND integration.github_app_installation_id = ?
	)`, syncSource.repositoryURL, syncSource.integrationID, syncSource.installationID)
}

func markOpenPullRequestsSynced(ctx context.Context, config *database.PullRequestDeploymentConfig, syncSource *pullRequestSyncSource) (bool, error) {
	query := database.DB.WithContext(ctx).Model(&database.PullRequestDeploymentConfig{}).
		Where("deployment_id = ? AND enabled = ? AND open_pull_requests_synced_at IS NULL AND reconciliation_generation = ?", config.DeploymentID, true, config.ReconciliationGeneration)
	result := pullRequestSyncSourceGuard(query, syncSource).Update("open_pull_requests_synced_at", time.Now())
	return result.RowsAffected == 1, result.Error
}

func completePullRequestReconciliation(ctx context.Context, config *database.PullRequestDeploymentConfig, syncSource *pullRequestSyncSource) (bool, error) {
	updates := map[string]interface{}{
		"reconciliation_pending":  false,
		"reconciliation_attempts": 0,
		"next_reconciliation_at":  nil,
	}
	query := database.DB.WithContext(ctx).Model(&database.PullRequestDeploymentConfig{}).
		Where("deployment_id = ? AND reconciliation_pending = ? AND reconciliation_generation = ?", config.DeploymentID, true, config.ReconciliationGeneration)
	if config.Enabled {
		query = pullRequestSyncSourceGuard(query.Where("enabled = ?", true), syncSource)
		updates["open_pull_requests_synced_at"] = time.Now()
	}
	result := query.Updates(updates)
	return result.RowsAffected == 1, result.Error
}

func (s *Service) retryPendingPullRequestReconciliations(ctx context.Context) {
	var configs []database.PullRequestDeploymentConfig
	if err := database.DB.WithContext(ctx).
		Where("reconciliation_pending = ? AND (next_reconciliation_at IS NULL OR next_reconciliation_at <= ?)", true, time.Now()).
		Order("next_reconciliation_at ASC, deployment_id ASC").
		Limit(25).
		Find(&configs).Error; err != nil {
		logger.Warn("[PRDeployments] Failed to load pending settings reconciliations: %v", err)
		return
	}
	for i := range configs {
		config := configs[i]
		source, err := s.repo.GetByID(ctx, config.DeploymentID)
		if err != nil {
			logger.Warn("[PRDeployments] Failed to load source %s for pending settings reconciliation: %v", config.DeploymentID, err)
			if retryErr := schedulePullRequestReconciliationRetry(ctx, &config); retryErr != nil {
				logger.Warn("[PRDeployments] Failed to advance settings reconciliation retry for %s: %v", config.DeploymentID, retryErr)
			}
			continue
		}
		syncSource, reconcileErr := s.reconcilePullRequestDeploymentConfig(ctx, source, &config)
		if reconcileErr != nil {
			logger.Warn("[PRDeployments] Pending settings reconciliation failed for %s: %v", config.DeploymentID, reconcileErr)
			if retryErr := schedulePullRequestReconciliationRetry(ctx, &config); retryErr != nil {
				logger.Warn("[PRDeployments] Failed to advance settings reconciliation retry for %s: %v", config.DeploymentID, retryErr)
			}
			continue
		}
		if _, err := completePullRequestReconciliation(ctx, &config, syncSource); err != nil {
			logger.Warn("[PRDeployments] Failed to complete settings reconciliation for %s: %v", config.DeploymentID, err)
		}
	}
}

func schedulePullRequestReconciliationRetry(ctx context.Context, config *database.PullRequestDeploymentConfig) error {
	attempts := config.ReconciliationAttempts + 1
	next := time.Now().Add(pullRequestReportRetryDelay(attempts))
	result := database.DB.WithContext(ctx).Model(&database.PullRequestDeploymentConfig{}).
		Where("deployment_id = ? AND reconciliation_pending = ? AND reconciliation_generation = ?", config.DeploymentID, true, config.ReconciliationGeneration).
		Updates(map[string]interface{}{"reconciliation_attempts": attempts, "next_reconciliation_at": next, "updated_at": time.Now()})
	if result.Error == nil && result.RowsAffected == 1 {
		config.ReconciliationAttempts = attempts
		config.NextReconciliationAt = &next
	}
	return result.Error
}

func (s *Service) cleanupExpiredPullRequestDeployments(parent context.Context) {
	ctx, cancel := context.WithTimeout(parent, 10*time.Minute)
	defer cancel()
	lastID := ""
	for {
		var records []database.PullRequestDeployment
		query := database.DB.WithContext(ctx).
			Where("closed_at IS NULL AND expires_at < ?", time.Now()).
			Order("id ASC").
			Limit(100)
		if lastID != "" {
			query = query.Where("id > ?", lastID)
		}
		if err := query.Find(&records).Error; err != nil {
			logger.Warn("[PRDeployments] Failed to load expired environments: %v", err)
			return
		}
		if len(records) == 0 {
			break
		}
		for i := range records {
			record := records[i]
			lockKey := fmt.Sprintf("pull-request:%s:%s:%d", record.SourceDeploymentID, record.Repository, record.PullRequestNumber)
			if err := withDistributedLock(ctx, lockKey, func() error {
				if err := database.DB.WithContext(ctx).Where("id = ? AND closed_at IS NULL AND expires_at < ?", record.ID, time.Now()).First(&record).Error; err != nil {
					if errors.Is(err, gorm.ErrRecordNotFound) {
						return nil
					}
					return err
				}
				return s.cleanupPullRequestDeployment(ctx, &record, "Preview lifetime expired.")
			}); err != nil {
				logger.Warn("[PRDeployments] Failed to clean up %s: %v", records[i].ID, err)
			}
		}
		lastID = records[len(records)-1].ID
		if len(records) < 100 {
			break
		}
	}
	stuckBefore := time.Now().Add(-6 * time.Hour)
	var interrupted []database.PullRequestDeployment
	if err := database.DB.WithContext(ctx).Where("closed_at IS NULL AND active_head_sha IS NOT NULL AND updated_at < ?", stuckBefore).Limit(100).Find(&interrupted).Error; err != nil {
		logger.Warn("[PRDeployments] Failed to recover interrupted builds: %v", err)
		return
	}
	for i := range interrupted {
		result := database.DB.WithContext(ctx).Model(&database.PullRequestDeployment{}).
			Where("id = ? AND active_head_sha = ? AND closed_at IS NULL", interrupted[i].ID, interrupted[i].ActiveHeadSHA).
			Updates(map[string]interface{}{"active_head_sha": nil, "status": int32(deploymentsv1.PullRequestDeploymentStatus_PULL_REQUEST_DEPLOYMENT_QUEUED), "error": nil, "updated_at": time.Now()})
		if result.Error != nil {
			logger.Warn("[PRDeployments] Failed to recover interrupted build %s: %v", interrupted[i].ID, result.Error)
		} else if result.RowsAffected == 1 {
			go s.deployPullRequestEnvironment(interrupted[i].ID)
		}
	}

	// Queued rows are the durable webhook work queue. The request starts an
	// eager worker after committing the row, while this scan recovers work if a
	// replica exits between the commit and goroutine launch. The build lease in
	// deployPullRequestEnvironment makes concurrent HA scans harmless.
	var queued []database.PullRequestDeployment
	if err := database.DB.WithContext(ctx).
		Where("closed_at IS NULL AND active_head_sha IS NULL AND status = ?", int32(deploymentsv1.PullRequestDeploymentStatus_PULL_REQUEST_DEPLOYMENT_QUEUED)).
		Order("updated_at ASC").Limit(100).Find(&queued).Error; err != nil {
		logger.Warn("[PRDeployments] Failed to recover queued preview work: %v", err)
		return
	}
	for i := range queued {
		go s.deployPullRequestEnvironment(queued[i].ID)
	}
}

type githubPullRequestWebhookPayload struct {
	Action       string `json:"action"`
	Number       int64  `json:"number"`
	Installation struct {
		ID int64 `json:"id"`
	} `json:"installation"`
	Repository struct {
		FullName string `json:"full_name"`
	} `json:"repository"`
	Sender struct {
		Login string `json:"login"`
	} `json:"sender"`
	PullRequest struct {
		Draft  bool `json:"draft"`
		Merged bool `json:"merged"`
		Head   struct {
			SHA  string `json:"sha"`
			Ref  string `json:"ref"`
			Repo struct {
				FullName string `json:"full_name"`
			} `json:"repo"`
		} `json:"head"`
		Base struct {
			Ref  string `json:"ref"`
			Repo struct {
				FullName string `json:"full_name"`
			} `json:"repo"`
		} `json:"base"`
	} `json:"pull_request"`
}

func (s *Service) handleGitHubPullRequestWebhook(w http.ResponseWriter, event string, body []byte) {
	var payload githubPullRequestWebhookPayload
	if err := json.Unmarshal(body, &payload); err != nil {
		writeGitHubWebhookJSON(w, http.StatusBadRequest, githubWebhookResponse{OK: false, Event: event, Message: "invalid pull request payload"})
		return
	}
	repository := normalizeGitHubRepoFullName(payload.Repository.FullName)
	if repository == "" || payload.Number <= 0 || payload.Installation.ID <= 0 || !isGitHubCommitSHA(payload.PullRequest.Head.SHA) {
		writeGitHubWebhookJSON(w, http.StatusBadRequest, githubWebhookResponse{OK: false, Event: event, Message: "pull request payload is missing repository, installation, number, or head revision"})
		return
	}
	if !supportedPullRequestAction(payload.Action) {
		writeGitHubWebhookJSON(w, http.StatusAccepted, githubWebhookResponse{OK: true, Event: event, Repository: repository, Message: "pull request action ignored"})
		return
	}
	var configs []database.PullRequestDeploymentConfig
	err := database.DB.Table("pull_request_deployment_configs").
		Joins("JOIN deployments ON deployments.id = pull_request_deployment_configs.deployment_id").
		Joins("JOIN github_integrations ON github_integrations.id = deployments.github_integration_id").
		Where("deployments.deleted_at IS NULL").
		Where("github_integrations.github_app_installation_id = ?", payload.Installation.ID).
		Where("github_integrations.organization_id = deployments.organization_id").
		Find(&configs).Error
	if err != nil {
		logger.Error("[PRDeployments] Failed to find templates for %s#%d: %v", repository, payload.Number, err)
		writeGitHubWebhookJSON(w, http.StatusInternalServerError, githubWebhookResponse{OK: false, Event: event, Repository: repository, Message: "failed to find pull request deployment settings"})
		return
	}
	matched := make([]string, 0, len(configs))
	var processingErrors []error
	for i := range configs {
		source, err := s.repo.GetByID(rContext(), configs[i].DeploymentID)
		if err != nil || source.RepositoryURL == nil || !githubRepoURLMatchesFullName(*source.RepositoryURL, repository) {
			continue
		}
		if !configs[i].Enabled && payload.Action != "closed" {
			continue
		}
		matched = append(matched, configs[i].DeploymentID)
		if err := s.processPullRequestWebhook(configs[i], payload); err != nil {
			processingErrors = append(processingErrors, fmt.Errorf("deployment %s: %w", configs[i].DeploymentID, err))
		}
	}
	if err := errors.Join(processingErrors...); err != nil {
		logger.Error("[PRDeployments] Failed to durably process webhook state for %s#%d: %v", repository, payload.Number, err)
		writeGitHubWebhookJSON(w, http.StatusServiceUnavailable, githubWebhookResponse{OK: false, Event: event, Repository: repository, MatchedDeployments: len(matched), Triggered: matched, Message: "pull request state was not saved; GitHub should retry this delivery"})
		return
	}
	writeGitHubWebhookJSON(w, http.StatusAccepted, githubWebhookResponse{OK: true, Event: event, Repository: repository, MatchedDeployments: len(matched), Triggered: matched, Message: "pull request webhook accepted"})
}

func supportedPullRequestAction(action string) bool {
	switch action {
	case "opened", "reopened", "synchronize", "ready_for_review", "converted_to_draft", "edited", "closed":
		return true
	default:
		return false
	}
}

func rContext() context.Context { return context.Background() }

func (s *Service) processPullRequestWebhook(config database.PullRequestDeploymentConfig, payload githubPullRequestWebhookPayload) error {
	ctx, cancel := s.detachedContext(10 * time.Minute)
	defer cancel()
	repository := normalizeGitHubRepoFullName(payload.Repository.FullName)
	lockKey := fmt.Sprintf("pull-request:%s:%s:%d", config.DeploymentID, repository, payload.Number)
	return withDistributedLock(ctx, lockKey, func() error {
		var current database.PullRequestDeploymentConfig
		if err := database.DB.WithContext(ctx).Where("deployment_id = ?", config.DeploymentID).First(&current).Error; err != nil {
			return fmt.Errorf("reload pull request deployment settings: %w", err)
		}
		return s.processPullRequestWebhookLocked(ctx, current, payload)
	})
}

func (s *Service) processPullRequestWebhookLocked(ctx context.Context, config database.PullRequestDeploymentConfig, payload githubPullRequestWebhookPayload) error {
	repository := normalizeGitHubRepoFullName(payload.Repository.FullName)
	client, err := githubclient.NewInstallationClient(ctx, payload.Installation.ID)
	if err != nil {
		return fmt.Errorf("create GitHub client: %w", err)
	}
	live, err := client.GetPullRequest(ctx, repository, payload.Number)
	if err != nil {
		return fmt.Errorf("load current pull request state: %w", err)
	}
	if !pullRequestWebhookMatchesCurrentState(payload.Action, payload.PullRequest.Head.SHA, live) {
		return nil
	}
	payload.PullRequest.Draft, payload.PullRequest.Merged = live.Draft, live.Merged
	payload.PullRequest.Head.SHA, payload.PullRequest.Head.Ref, payload.PullRequest.Head.Repo.FullName = live.Head.SHA, live.Head.Ref, live.Head.Repo.FullName
	payload.PullRequest.Base.Ref, payload.PullRequest.Base.Repo.FullName = live.Base.Ref, live.Base.Repo.FullName
	reconcilingClosedPullRequest := payload.Action == "reconcile" && live.State == "closed"
	var existing database.PullRequestDeployment
	existingErr := database.DB.WithContext(ctx).Where("source_deployment_id = ? AND repository = ? AND pull_request_number = ?", config.DeploymentID, repository, payload.Number).First(&existing).Error
	if existingErr != nil && !errors.Is(existingErr, gorm.ErrRecordNotFound) {
		return fmt.Errorf("load existing pull request state: %w", existingErr)
	}
	reconcilingRestoredPreview := existingErr == nil && pullRequestReconciliationPreservesRestoredPreview(&existing, payload.Action, live.State, payload.PullRequest.Head.SHA)
	if reconcilingClosedPullRequest && !reconcilingRestoredPreview {
		payload.Action = "closed"
	}
	if payload.Action == "closed" {
		if existingErr == nil {
			// A restored merged preview intentionally remains open until its short
			// TTL expires. GitHub may redeliver the original close webhook after
			// restoration; do not tear the restored revision down again.
			if pullRequestCloseIsRestoredRedelivery(&existing, payload.PullRequest.Head.SHA) {
				return nil
			}
			existing.Merged = payload.PullRequest.Merged
			existing.UpdatedAt = time.Now()
			if config.CleanupOnClose {
				if err := s.cleanupPullRequestDeployment(ctx, &existing, "Pull request closed."); err != nil {
					return err
				}
			} else if err := database.DB.WithContext(ctx).Save(&existing).Error; err != nil {
				return err
			}
		}
		return nil
	}
	source, err := s.repo.GetByID(ctx, config.DeploymentID)
	if err != nil {
		return err
	}
	baseRef, headRef := strings.TrimSpace(payload.PullRequest.Base.Ref), strings.TrimSpace(payload.PullRequest.Head.Ref)
	fromFork := !strings.EqualFold(normalizeGitHubRepoFullName(payload.PullRequest.Head.Repo.FullName), repository)
	reason := ""
	if !config.Enabled {
		reason = "Pull request environments are disabled for this deployment."
	}
	if reason == "" {
		if patterns := parseStringList(config.BaseBranches); len(patterns) > 0 && !matchesPRPatterns(baseRef, patterns) {
			reason = "The pull request target branch is outside this preview scope."
		}
	}
	if reason == "" && payload.PullRequest.Draft && !config.DeployDrafts {
		reason = "Draft pull requests are not deployed by this template."
	}
	if reason == "" && fromFork && deploymentsv1.PullRequestForkPolicy(config.ForkPolicy) == deploymentsv1.PullRequestForkPolicy_PULL_REQUEST_FORK_DENY {
		reason = "Fork pull request previews are disabled for this deployment."
	}
	if reason == "" && (len(parseStringList(config.IncludePaths)) > 0 || len(parseStringList(config.ExcludePaths)) > 0) {
		if files, filesErr := client.ListPullRequestFiles(ctx, repository, payload.Number); filesErr != nil {
			reason = "Obiente could not inspect the pull request file scope."
		} else if !pullRequestFilesMatch(files, &config) {
			reason = "No changed files match this preview scope."
		}
	}
	now := time.Now()
	if existingErr == nil && payload.Action == "synchronize" && !config.RedeployOnPush {
		return database.DB.WithContext(ctx).Model(&database.PullRequestDeployment{}).
			Where("id = ? AND closed_at IS NULL", existing.ID).
			Updates(map[string]interface{}{
				"ignored_head_sha": payload.PullRequest.Head.SHA,
				"expires_at":       now.Add(time.Duration(config.TTLHours) * time.Hour),
				"updated_at":       now,
			}).Error
	}
	if preserveIgnoredPullRequestRevision(&existing, existingErr, &config, payload.PullRequest.Head.SHA, reason) {
		return database.DB.WithContext(ctx).Model(&database.PullRequestDeployment{}).
			Where("id = ? AND closed_at IS NULL AND ignored_head_sha = ?", existing.ID, payload.PullRequest.Head.SHA).
			Updates(map[string]interface{}{
				"base_ref":   baseRef,
				"from_fork":  fromFork,
				"draft":      payload.PullRequest.Draft,
				"expires_at": now.Add(time.Duration(config.TTLHours) * time.Hour),
				"updated_at": now,
			}).Error
	}
	record := &existing
	if existingErr != nil {
		record = &database.PullRequestDeployment{ID: "pr-deployment-" + uuid.NewString(), SourceDeploymentID: source.ID, OrganizationID: source.OrganizationID,
			GitHubIntegrationID: stringValue(source.GitHubIntegrationID), GitHubInstallationID: payload.Installation.ID, Repository: repository, PullRequestNumber: payload.Number, CreatedAt: now}
	}
	record.GitHubIntegrationID = stringValue(source.GitHubIntegrationID)
	record.GitHubInstallationID = payload.Installation.ID
	restoredAt, restoredExpiresAt := record.RestoredAt, record.ExpiresAt
	shaChanged := record.HeadSHA != "" && record.HeadSHA != payload.PullRequest.Head.SHA
	revisionUnchanged := existingErr == nil && !shaChanged
	if shaChanged && record.GitHubDeploymentID != nil {
		_ = s.markGitHubDeploymentInactive(ctx, record)
		record.GitHubDeploymentID, record.GitHubDeploymentSHA = nil, nil
	}
	if shaChanged {
		_ = s.markGitHubCheckRunSuperseded(ctx, record)
		record.GitHubCheckRunID, record.GitHubCheckRunSHA = nil, nil
	}
	record.HeadSHA, record.IgnoredHeadSHA, record.HeadRef, record.BaseRef, record.FromFork, record.Draft = payload.PullRequest.Head.SHA, nil, headRef, baseRef, fromFork, payload.PullRequest.Draft
	record.ClosedAt, record.UpdatedAt = nil, now
	if reconcilingRestoredPreview {
		record.Merged = true
		record.RestoredAt = restoredAt
		record.ExpiresAt = restoredExpiresAt
	} else {
		record.Merged = false
		record.RestoredAt = nil
		record.ExpiresAt = now.Add(time.Duration(config.TTLHours) * time.Hour)
	}
	if shaChanged && !config.ApprovalCoversUpdates {
		record.ApprovedAt, record.ApprovedBy, record.ApprovedHeadSHA = nil, nil, nil
	}
	runtimeDetached := false
	if reason != "" {
		if record.PreviewDeploymentID != nil {
			s.markPullRequestPreviewCleanupForRetry(ctx, record.ID, "Preview removal is pending retry.")
			if err := s.removePreviewDeployment(ctx, record); err != nil {
				logger.Warn("[PRDeployments] Failed to remove out-of-scope preview %s: %v", record.ID, err)
				return fmt.Errorf("remove out-of-scope preview: %w", err)
			}
			runtimeDetached = true
		}
		record.ActiveHeadSHA = nil
		record.Status, record.Error = int32(deploymentsv1.PullRequestDeploymentStatus_PULL_REQUEST_DEPLOYMENT_SKIPPED), &reason
	} else if requiresPullRequestApproval(record, &config) && !pullRequestDeploymentApproved(record, &config) {
		if record.PreviewDeploymentID != nil {
			s.markPullRequestPreviewCleanupForRetry(ctx, record.ID, "Preview removal is pending maintainer approval.")
			if err := s.removePreviewDeployment(ctx, record); err != nil {
				return fmt.Errorf("remove preview awaiting approval: %w", err)
			}
			runtimeDetached = true
		}
		record.ActiveHeadSHA = nil
		if !revisionUnchanged || !containsPRStatus([]int32{
			int32(deploymentsv1.PullRequestDeploymentStatus_PULL_REQUEST_DEPLOYMENT_WAITING_APPROVAL),
			int32(deploymentsv1.PullRequestDeploymentStatus_PULL_REQUEST_DEPLOYMENT_REJECTED),
		}, record.Status) {
			record.Status, record.Error = int32(deploymentsv1.PullRequestDeploymentStatus_PULL_REQUEST_DEPLOYMENT_WAITING_APPROVAL), nil
		}
	} else {
		if !revisionUnchanged || !preservePullRequestStateForUnchangedRevision(record.Status) {
			record.Status, record.Error = int32(deploymentsv1.PullRequestDeploymentStatus_PULL_REQUEST_DEPLOYMENT_QUEUED), nil
		}
	}
	if record.Status == int32(deploymentsv1.PullRequestDeploymentStatus_PULL_REQUEST_DEPLOYMENT_QUEUED) {
		if domain, domainErr := renderPRDomain(&config, source, record); domainErr == nil {
			url := "https://" + domain
			record.EnvironmentURL = &url
		} else {
			logger.Warn("[PRDeployments] Invalid preview hostname for %s#%d: %v", repository, payload.Number, domainErr)
			record.Status = int32(deploymentsv1.PullRequestDeploymentStatus_PULL_REQUEST_DEPLOYMENT_FAILED)
			message := "The configured preview hostname is invalid."
			record.Error = &message
		}
	} else if reason != "" || record.Status == int32(deploymentsv1.PullRequestDeploymentStatus_PULL_REQUEST_DEPLOYMENT_WAITING_APPROVAL) || record.Status == int32(deploymentsv1.PullRequestDeploymentStatus_PULL_REQUEST_DEPLOYMENT_REJECTED) {
		record.EnvironmentURL = nil
	}
	var persistErr error
	if existingErr != nil {
		persistErr = database.DB.WithContext(ctx).Create(record).Error
	} else {
		updates := map[string]interface{}{
			"github_integration_id":  record.GitHubIntegrationID,
			"github_installation_id": record.GitHubInstallationID,
			"head_sha":               record.HeadSHA,
			"ignored_head_sha":       nil,
			"head_ref":               record.HeadRef,
			"base_ref":               record.BaseRef,
			"from_fork":              record.FromFork,
			"draft":                  record.Draft,
			"merged":                 record.Merged,
			"status":                 record.Status,
			"environment_url":        record.EnvironmentURL,
			"error":                  record.Error,
			"expires_at":             record.ExpiresAt,
			"closed_at":              nil,
			"restored_at":            record.RestoredAt,
			"updated_at":             record.UpdatedAt,
		}
		if reason != "" || record.Status == int32(deploymentsv1.PullRequestDeploymentStatus_PULL_REQUEST_DEPLOYMENT_WAITING_APPROVAL) {
			updates["preview_deployment_id"] = record.PreviewDeploymentID
			updates["active_head_sha"] = nil
		}
		if runtimeDetached {
			_ = s.markGitHubDeploymentInactive(ctx, record)
			_ = s.markGitHubCheckRunComplete(ctx, record, "Preview removed", "This pull request no longer meets the current preview policy.", "cancelled")
			record.GitHubDeploymentID, record.GitHubDeploymentSHA = nil, nil
			record.GitHubCheckRunID, record.GitHubCheckRunSHA = nil, nil
			updates["github_deployment_id"] = nil
			updates["github_deployment_sha"] = nil
			updates["github_check_run_id"] = nil
			updates["github_check_run_sha"] = nil
		}
		if shaChanged {
			updates["github_deployment_id"] = nil
			updates["github_deployment_sha"] = nil
			updates["github_check_run_id"] = nil
			updates["github_check_run_sha"] = nil
			if !config.ApprovalCoversUpdates {
				updates["approved_at"] = nil
				updates["approved_by"] = nil
				updates["approved_head_sha"] = nil
			}
		}
		persistErr = database.DB.WithContext(ctx).Model(&database.PullRequestDeployment{}).Where("id = ?", record.ID).Updates(updates).Error
	}
	if persistErr != nil {
		return fmt.Errorf("persist pull request state: %w", persistErr)
	}
	if err := database.DB.WithContext(ctx).Where("id = ?", record.ID).First(record).Error; err != nil {
		return fmt.Errorf("reload pull request state: %w", err)
	}
	if record.Status == int32(deploymentsv1.PullRequestDeploymentStatus_PULL_REQUEST_DEPLOYMENT_QUEUED) {
		if record.ActiveHeadSHA == nil {
			go s.deployPullRequestEnvironment(record.ID)
		} else {
			s.reportPullRequestDeployment(record.ID)
		}
	} else {
		s.reportPullRequestDeployment(record.ID)
	}
	return nil
}

func pullRequestCloseIsRestoredRedelivery(record *database.PullRequestDeployment, headSHA string) bool {
	return record != nil && record.RestoredAt != nil && record.Merged && record.ClosedAt == nil && record.HeadSHA == headSHA
}

func pullRequestReconciliationPreservesRestoredPreview(record *database.PullRequestDeployment, action, liveState, headSHA string) bool {
	return action == "reconcile" && liveState == "closed" && pullRequestCloseIsRestoredRedelivery(record, headSHA)
}

func preserveIgnoredPullRequestRevision(record *database.PullRequestDeployment, existingErr error, config *database.PullRequestDeploymentConfig, observedHeadSHA, reason string) bool {
	if record == nil || existingErr != nil || config == nil || config.RedeployOnPush || record.ClosedAt != nil || reason != "" || record.IgnoredHeadSHA == nil || *record.IgnoredHeadSHA != observedHeadSHA {
		return false
	}
	return !containsPRStatus([]int32{
		int32(deploymentsv1.PullRequestDeploymentStatus_PULL_REQUEST_DEPLOYMENT_SKIPPED),
		int32(deploymentsv1.PullRequestDeploymentStatus_PULL_REQUEST_DEPLOYMENT_CLOSED),
	}, record.Status)
}

func preservePullRequestStateForUnchangedRevision(status int32) bool {
	return containsPRStatus([]int32{
		int32(deploymentsv1.PullRequestDeploymentStatus_PULL_REQUEST_DEPLOYMENT_QUEUED),
		int32(deploymentsv1.PullRequestDeploymentStatus_PULL_REQUEST_DEPLOYMENT_BUILDING),
		int32(deploymentsv1.PullRequestDeploymentStatus_PULL_REQUEST_DEPLOYMENT_RUNNING),
		int32(deploymentsv1.PullRequestDeploymentStatus_PULL_REQUEST_DEPLOYMENT_FAILED),
	}, status)
}

func pullRequestWebhookMatchesCurrentState(action, payloadHeadSHA string, live *githubclient.PullRequest) bool {
	if live == nil {
		return false
	}
	if action == "reconcile" {
		return live.State == "open" || live.State == "closed"
	}
	if live.Head.SHA != payloadHeadSHA {
		return false
	}
	if action == "closed" {
		return live.State == "closed"
	}
	return live.State == "open"
}

func (s *Service) removePreviewDeployment(ctx context.Context, record *database.PullRequestDeployment) error {
	if record.PreviewDeploymentID == nil {
		return nil
	}
	_, err := s.DeleteDeployment(auth.WithSystemUser(ctx), connect.NewRequest(&deploymentsv1.DeleteDeploymentRequest{DeploymentId: *record.PreviewDeploymentID, OrganizationId: record.OrganizationID}))
	if err != nil && connect.CodeOf(err) != connect.CodeNotFound {
		return err
	}
	record.PreviewDeploymentID, record.EnvironmentURL = nil, nil
	return nil
}

func (s *Service) markPullRequestPreviewCleanupForRetry(ctx context.Context, recordID, message string) {
	now := time.Now()
	if err := database.DB.WithContext(ctx).Model(&database.PullRequestDeployment{}).
		Where("id = ? AND closed_at IS NULL", recordID).
		Updates(map[string]interface{}{"expires_at": now, "error": message, "updated_at": now}).Error; err != nil {
		logger.Warn("[PRDeployments] Failed to schedule cleanup retry for %s: %v", recordID, err)
	}
}

func pullRequestFilesMatch(files []githubclient.PullRequestFile, config *database.PullRequestDeploymentConfig) bool {
	includes, excludes := parseStringList(config.IncludePaths), parseStringList(config.ExcludePaths)
	for _, file := range files {
		name := strings.TrimSpace(file.Filename)
		if name == "" || matchesPRPatterns(name, excludes) {
			continue
		}
		if len(includes) == 0 || matchesPRPatterns(name, includes) {
			return true
		}
	}
	return false
}

func stringValue(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

func (s *Service) reportPullRequestDeployment(recordID string) {
	ctx, cancel := s.detachedContext(2 * time.Minute)
	defer cancel()
	now := time.Now()
	if err := database.DB.WithContext(ctx).Model(&database.PullRequestDeployment{}).Where("id = ?", recordID).
		Updates(map[string]interface{}{"report_pending": true, "next_report_at": now}).Error; err != nil {
		logger.Warn("[PRDeployments] Failed to queue GitHub reporting for %s: %v", recordID, err)
		return
	}
	err := withDistributedLock(ctx, "pull-request-report:"+recordID, func() error {
		return s.reportPullRequestDeploymentLocked(ctx, recordID)
	})
	if err == nil {
		if clearErr := database.DB.WithContext(ctx).Model(&database.PullRequestDeployment{}).Where("id = ?", recordID).
			Updates(map[string]interface{}{"report_pending": false, "report_attempts": 0, "next_report_at": nil}).Error; clearErr != nil {
			logger.Warn("[PRDeployments] GitHub reporting completed for %s but its retry marker could not be cleared: %v", recordID, clearErr)
		}
		return
	}
	var record database.PullRequestDeployment
	if loadErr := database.DB.WithContext(ctx).Select("report_attempts").Where("id = ?", recordID).First(&record).Error; loadErr != nil {
		logger.Warn("[PRDeployments] GitHub reporting failed for %s and its retry state could not be loaded: %v (report error: %v)", recordID, loadErr, err)
		return
	}
	attempts := record.ReportAttempts + 1
	nextAttempt := time.Now().Add(pullRequestReportRetryDelay(attempts))
	if updateErr := database.DB.WithContext(ctx).Model(&database.PullRequestDeployment{}).Where("id = ?", recordID).
		Updates(map[string]interface{}{"report_pending": true, "report_attempts": attempts, "next_report_at": nextAttempt}).Error; updateErr != nil {
		logger.Warn("[PRDeployments] GitHub reporting failed for %s and its retry could not be scheduled: %v (report error: %v)", recordID, updateErr, err)
		return
	}
	logger.Warn("[PRDeployments] GitHub reporting failed for %s; retry %d scheduled for %s: %v", recordID, attempts, nextAttempt.Format(time.RFC3339), err)
}

func pullRequestReportRetryDelay(attempt int32) time.Duration {
	delay := 30 * time.Second
	for i := int32(1); i < attempt && delay < 15*time.Minute; i++ {
		delay *= 2
	}
	if delay > 15*time.Minute {
		return 15 * time.Minute
	}
	return delay
}

func (s *Service) reportPullRequestDeploymentLocked(ctx context.Context, recordID string) error {
	var record database.PullRequestDeployment
	if err := database.DB.WithContext(ctx).Where("id = ?", recordID).First(&record).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil
		}
		return err
	}
	var config database.PullRequestDeploymentConfig
	if err := database.DB.WithContext(ctx).Where("deployment_id = ?", record.SourceDeploymentID).First(&config).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil
		}
		return err
	}
	client, err := githubclient.NewInstallationClient(ctx, record.GitHubInstallationID)
	if err != nil {
		return fmt.Errorf("create GitHub client: %w", err)
	}
	var reportErrors []error
	source, _ := s.repo.GetByID(ctx, record.SourceDeploymentID)
	logURL := ""
	if record.PreviewDeploymentID != nil {
		logURL = deploymentDashboardURL(*record.PreviewDeploymentID)
	} else {
		logURL = deploymentDashboardURL(record.SourceDeploymentID)
	}
	if config.DeploymentStatusEnabled && deploymentsv1.PullRequestDeploymentStatus(record.Status) != deploymentsv1.PullRequestDeploymentStatus_PULL_REQUEST_DEPLOYMENT_SKIPPED {
		if record.GitHubDeploymentID == nil && record.ClosedAt == nil {
			environment := fmt.Sprintf("Obiente Preview / PR #%d / %s", record.PullRequestNumber, sourceName(source, record.SourceDeploymentID))
			deploymentID, createErr := client.CreateDeployment(ctx, record.Repository, record.HeadSHA, environment, "Obiente pull request environment")
			if createErr != nil {
				reportErrors = append(reportErrors, fmt.Errorf("create GitHub deployment: %w", createErr))
			} else {
				record.GitHubDeploymentID, record.GitHubDeploymentSHA = &deploymentID, &record.HeadSHA
				if err := database.DB.WithContext(ctx).Model(&record).Updates(map[string]interface{}{"github_deployment_id": deploymentID, "github_deployment_sha": record.HeadSHA}).Error; err != nil {
					reportErrors = append(reportErrors, fmt.Errorf("persist GitHub deployment: %w", err))
				}
			}
		}
		if record.GitHubDeploymentID != nil {
			state, description := githubPRDeploymentState(&record)
			environmentURL := pullRequestDeploymentPublicURL(&record)
			if err := client.CreateDeploymentStatus(ctx, record.Repository, *record.GitHubDeploymentID, state, description, environmentURL, logURL); err != nil {
				reportErrors = append(reportErrors, fmt.Errorf("update GitHub deployment: %w", err))
			}
		}
	}
	if config.CheckRunEnabled {
		check := githubPRCheckRun(&record, source, logURL)
		if record.GitHubCheckRunID == nil && record.ClosedAt == nil {
			if id, createErr := client.CreateCheckRun(ctx, record.Repository, check); createErr != nil {
				reportErrors = append(reportErrors, fmt.Errorf("create GitHub check: %w", createErr))
			} else {
				record.GitHubCheckRunID, record.GitHubCheckRunSHA = &id, &record.HeadSHA
				if err := database.DB.WithContext(ctx).Model(&record).Updates(map[string]interface{}{"github_check_run_id": id, "github_check_run_sha": record.HeadSHA}).Error; err != nil {
					reportErrors = append(reportErrors, fmt.Errorf("persist GitHub check: %w", err))
				}
			}
		} else if record.GitHubCheckRunID != nil {
			if updateErr := client.UpdateCheckRun(ctx, record.Repository, *record.GitHubCheckRunID, check); updateErr != nil {
				reportErrors = append(reportErrors, fmt.Errorf("update GitHub check: %w", updateErr))
			}
		}
	}
	if config.CommentEnabled {
		marker := fmt.Sprintf(prEnvironmentCommentMark, record.SourceDeploymentID)
		body := pullRequestDeploymentComment(marker, &record, source, logURL)
		commentID := record.GitHubCommentID
		commentLookupSucceeded := true
		if commentID == nil {
			if comment, findErr := client.FindIssueComment(ctx, record.Repository, record.PullRequestNumber, marker); findErr != nil {
				commentLookupSucceeded = false
				reportErrors = append(reportErrors, fmt.Errorf("find pull request comment: %w", findErr))
			} else if comment != nil {
				commentID = &comment.ID
			}
		}
		if commentID == nil {
			if commentLookupSucceeded {
				if id, createErr := client.CreateIssueComment(ctx, record.Repository, record.PullRequestNumber, body); createErr != nil {
					reportErrors = append(reportErrors, fmt.Errorf("create pull request comment: %w", createErr))
				} else {
					commentID = &id
				}
			}
		} else if updateErr := client.UpdateIssueComment(ctx, record.Repository, *commentID, body); updateErr != nil {
			reportErrors = append(reportErrors, fmt.Errorf("update pull request comment: %w", updateErr))
		}
		if commentID != nil && record.GitHubCommentID == nil {
			record.GitHubCommentID = commentID
			if err := database.DB.WithContext(ctx).Model(&record).Update("github_comment_id", *commentID).Error; err != nil {
				reportErrors = append(reportErrors, fmt.Errorf("persist pull request comment: %w", err))
			}
		}
	}
	return errors.Join(reportErrors...)
}

func githubPRCheckRun(record *database.PullRequestDeployment, source *database.Deployment, detailsURL string) githubclient.CheckRunUpdate {
	status, conclusion, title, summary := "queued", "", "Preview queued", "Obiente is preparing a pull request environment."
	switch deploymentsv1.PullRequestDeploymentStatus(record.Status) {
	case deploymentsv1.PullRequestDeploymentStatus_PULL_REQUEST_DEPLOYMENT_BUILDING:
		status, title, summary = "in_progress", "Preview building", "Obiente is building and starting the preview."
	case deploymentsv1.PullRequestDeploymentStatus_PULL_REQUEST_DEPLOYMENT_RUNNING:
		status, conclusion, title, summary = "completed", "success", "Preview ready", "The pull request environment is ready."
		if record.EnvironmentURL != nil {
			summary += "\n\n[Open preview](" + *record.EnvironmentURL + ")"
		}
	case deploymentsv1.PullRequestDeploymentStatus_PULL_REQUEST_DEPLOYMENT_FAILED:
		status, conclusion, title, summary = "completed", "failure", "Preview failed", "The preview did not deploy successfully. Open Obiente Cloud for build output."
	case deploymentsv1.PullRequestDeploymentStatus_PULL_REQUEST_DEPLOYMENT_SKIPPED:
		status, conclusion, title, summary = "completed", "skipped", "Preview not deployed", stringValue(record.Error)
	case deploymentsv1.PullRequestDeploymentStatus_PULL_REQUEST_DEPLOYMENT_WAITING_APPROVAL:
		status, conclusion, title, summary = "completed", "action_required", "Maintainer approval required", "Approve the current revision in Obiente before it can run."
	case deploymentsv1.PullRequestDeploymentStatus_PULL_REQUEST_DEPLOYMENT_REJECTED:
		status, conclusion, title, summary = "completed", "failure", "Preview rejected", "A maintainer rejected this preview."
	case deploymentsv1.PullRequestDeploymentStatus_PULL_REQUEST_DEPLOYMENT_CLOSED:
		status, conclusion, title, summary = "completed", "cancelled", "Preview removed", "The pull request environment has been removed."
	}
	return githubclient.CheckRunUpdate{Name: "Obiente Preview · " + sourceName(source, record.SourceDeploymentID), HeadSHA: record.HeadSHA, DetailsURL: detailsURL, ExternalID: record.ID, Status: status, Conclusion: conclusion, Title: title, Summary: summary}
}

func (s *Service) markGitHubDeploymentInactive(ctx context.Context, record *database.PullRequestDeployment) error {
	return s.markGitHubDeploymentInactiveWithDescription(ctx, record, "Superseded by a newer pull request revision.")
}

func (s *Service) markGitHubDeploymentInactiveWithDescription(ctx context.Context, record *database.PullRequestDeployment, description string) error {
	if record.GitHubDeploymentID == nil {
		return nil
	}
	return withDistributedLock(ctx, "pull-request-report:"+record.ID, func() error {
		client, err := githubclient.NewInstallationClient(ctx, record.GitHubInstallationID)
		if err != nil {
			return err
		}
		return client.CreateDeploymentStatus(ctx, record.Repository, *record.GitHubDeploymentID, "inactive", description, "", "")
	})
}

func (s *Service) markGitHubCheckRunSuperseded(ctx context.Context, record *database.PullRequestDeployment) error {
	return s.markGitHubCheckRunComplete(ctx, record, "Preview superseded", "A newer pull request revision replaced this preview.", "cancelled")
}

func (s *Service) markGitHubCheckRunComplete(ctx context.Context, record *database.PullRequestDeployment, title, summary, conclusion string) error {
	if record.GitHubCheckRunID == nil {
		return nil
	}
	return withDistributedLock(ctx, "pull-request-report:"+record.ID, func() error {
		client, err := githubclient.NewInstallationClient(ctx, record.GitHubInstallationID)
		if err != nil {
			return err
		}
		source, _ := s.repo.GetByID(ctx, record.SourceDeploymentID)
		update := githubclient.CheckRunUpdate{Name: "Obiente Preview · " + sourceName(source, record.SourceDeploymentID), DetailsURL: deploymentDashboardURL(record.SourceDeploymentID), ExternalID: record.ID, Status: "completed", Conclusion: conclusion, Title: title, Summary: summary}
		return client.UpdateCheckRun(ctx, record.Repository, *record.GitHubCheckRunID, update)
	})
}

func deploymentDashboardURL(deploymentID string) string {
	baseURL := strings.TrimRight(strings.TrimSpace(os.Getenv("DASHBOARD_URL")), "/")
	if baseURL == "" {
		return ""
	}
	return baseURL + "/deployments/" + deploymentID
}

func githubPRDeploymentState(record *database.PullRequestDeployment) (string, string) {
	switch deploymentsv1.PullRequestDeploymentStatus(record.Status) {
	case deploymentsv1.PullRequestDeploymentStatus_PULL_REQUEST_DEPLOYMENT_QUEUED:
		return "queued", "Preview is queued."
	case deploymentsv1.PullRequestDeploymentStatus_PULL_REQUEST_DEPLOYMENT_BUILDING:
		return "in_progress", "Preview is building."
	case deploymentsv1.PullRequestDeploymentStatus_PULL_REQUEST_DEPLOYMENT_RUNNING:
		return "success", "Preview is ready."
	case deploymentsv1.PullRequestDeploymentStatus_PULL_REQUEST_DEPLOYMENT_FAILED:
		return "failure", "Preview failed."
	case deploymentsv1.PullRequestDeploymentStatus_PULL_REQUEST_DEPLOYMENT_WAITING_APPROVAL:
		return "pending", "Preview is waiting for maintainer approval."
	case deploymentsv1.PullRequestDeploymentStatus_PULL_REQUEST_DEPLOYMENT_REJECTED:
		return "failure", "Preview was rejected."
	case deploymentsv1.PullRequestDeploymentStatus_PULL_REQUEST_DEPLOYMENT_CLOSED:
		return "inactive", "Preview was removed."
	default:
		return "pending", "Preview status is pending."
	}
}

func pullRequestDeploymentComment(marker string, record *database.PullRequestDeployment, source *database.Deployment, manageURL string) string {
	status := "Pending"
	switch deploymentsv1.PullRequestDeploymentStatus(record.Status) {
	case deploymentsv1.PullRequestDeploymentStatus_PULL_REQUEST_DEPLOYMENT_QUEUED:
		status = "Queued"
	case deploymentsv1.PullRequestDeploymentStatus_PULL_REQUEST_DEPLOYMENT_BUILDING:
		status = "Building"
	case deploymentsv1.PullRequestDeploymentStatus_PULL_REQUEST_DEPLOYMENT_RUNNING:
		status = "Ready"
	case deploymentsv1.PullRequestDeploymentStatus_PULL_REQUEST_DEPLOYMENT_FAILED:
		status = "Failed"
	case deploymentsv1.PullRequestDeploymentStatus_PULL_REQUEST_DEPLOYMENT_SKIPPED:
		status = "Not deployed"
	case deploymentsv1.PullRequestDeploymentStatus_PULL_REQUEST_DEPLOYMENT_CLOSED:
		status = "Removed"
	case deploymentsv1.PullRequestDeploymentStatus_PULL_REQUEST_DEPLOYMENT_WAITING_APPROVAL:
		status = "Waiting for maintainer approval"
	case deploymentsv1.PullRequestDeploymentStatus_PULL_REQUEST_DEPLOYMENT_REJECTED:
		status = "Rejected"
	}
	sha := record.HeadSHA
	if len(sha) > 12 {
		sha = sha[:12]
	}
	lines := []string{marker, "### Obiente preview", "", fmt.Sprintf("**%s** · `%s` · %s", markdownText(sourceName(source, record.SourceDeploymentID)), sha, status), ""}
	if record.EnvironmentURL != nil && record.Status == int32(deploymentsv1.PullRequestDeploymentStatus_PULL_REQUEST_DEPLOYMENT_RUNNING) {
		lines = append(lines, fmt.Sprintf("[Open preview](%s)", *record.EnvironmentURL), "")
	} else if stateURL := pullRequestDeploymentStateURL(record); stateURL != nil {
		lines = append(lines, fmt.Sprintf("[View preview status](%s)", *stateURL), "")
	}
	if record.Status == int32(deploymentsv1.PullRequestDeploymentStatus_PULL_REQUEST_DEPLOYMENT_WAITING_APPROVAL) {
		lines = append(lines, "A maintainer with access to this Obiente deployment must approve the current revision before it can run.", "")
	}
	if record.FromFork {
		lines = append(lines, "This is a fork preview. Production environment variables, build arguments, and persistent volumes are not available to it.", "")
	}
	if manageURL != "" {
		lines = append(lines, fmt.Sprintf("[View in Obiente](%s)", manageURL), "")
	}
	lines = append(lines, "<sub>Obiente updates this comment as the preview changes.</sub>")
	return strings.Join(lines, "\n")
}

func sourceName(source *database.Deployment, fallback string) string {
	if source != nil && strings.TrimSpace(source.Name) != "" {
		return source.Name
	}
	return fallback
}
func markdownText(value string) string {
	replacer := strings.NewReplacer("\\", "\\\\", "*", "\\*", "_", "\\_", "`", "\\`", "[", "\\[", "]", "\\]")
	return replacer.Replace(value)
}
