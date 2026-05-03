package deployments

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	githubclient "deployments-service/internal/github"
	"github.com/obiente/cloud/apps/shared/pkg/auth"
	"github.com/obiente/cloud/apps/shared/pkg/database"
	"github.com/obiente/cloud/apps/shared/pkg/logger"
	"github.com/obiente/cloud/apps/shared/pkg/notifications"

	deploymentsv1 "github.com/obiente/cloud/apps/shared/proto/obiente/cloud/deployments/v1"
	notificationsv1 "github.com/obiente/cloud/apps/shared/proto/obiente/cloud/notifications/v1"

	"connectrpc.com/connect"
)

const githubWebhookMaxBodyBytes = 1 << 20 // 1 MiB

type githubWebhookPushPayload struct {
	Ref        string `json:"ref"`
	After      string `json:"after"`
	Repository struct {
		FullName string `json:"full_name"`
		HTMLURL  string `json:"html_url"`
		CloneURL string `json:"clone_url"`
	} `json:"repository"`
	HeadCommit *struct {
		ID      string `json:"id"`
		Message string `json:"message"`
	} `json:"head_commit"`
	Sender *struct {
		Login string `json:"login"`
	} `json:"sender"`
}

type githubWebhookResponse struct {
	OK                 bool     `json:"ok"`
	Event              string   `json:"event"`
	Repository         string   `json:"repository,omitempty"`
	Branch             string   `json:"branch,omitempty"`
	MatchedDeployments int      `json:"matched_deployments,omitempty"`
	Triggered          []string `json:"triggered,omitempty"`
	Message            string   `json:"message,omitempty"`
}

// HandleGitHubWebhook receives GitHub webhooks and triggers deployments linked to
// the pushed repository and branch. Deployments opt into this path by storing a
// github_integration_id alongside their repository_url.
func (s *Service) HandleGitHubWebhook(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeGitHubWebhookJSON(w, http.StatusMethodNotAllowed, githubWebhookResponse{
			OK:      false,
			Message: "method not allowed",
		})
		return
	}

	event := r.Header.Get("X-GitHub-Event")
	if event == "" {
		writeGitHubWebhookJSON(w, http.StatusBadRequest, githubWebhookResponse{
			OK:      false,
			Message: "missing X-GitHub-Event header",
		})
		return
	}

	body, err := io.ReadAll(http.MaxBytesReader(w, r.Body, githubWebhookMaxBodyBytes))
	if err != nil {
		writeGitHubWebhookJSON(w, http.StatusRequestEntityTooLarge, githubWebhookResponse{
			OK:      false,
			Event:   event,
			Message: "webhook payload is too large",
		})
		return
	}

	if err := verifyGitHubWebhookSignature(body, r.Header.Get("X-Hub-Signature-256")); err != nil {
		logger.Warn("[GitHubWebhook] Signature verification failed: %v", err)
		writeGitHubWebhookJSON(w, http.StatusUnauthorized, githubWebhookResponse{
			OK:      false,
			Event:   event,
			Message: "invalid webhook signature",
		})
		return
	}

	switch event {
	case "ping":
		writeGitHubWebhookJSON(w, http.StatusOK, githubWebhookResponse{
			OK:      true,
			Event:   event,
			Message: "pong",
		})
	case "push":
		s.handleGitHubPushWebhook(w, event, body)
	default:
		writeGitHubWebhookJSON(w, http.StatusAccepted, githubWebhookResponse{
			OK:      true,
			Event:   event,
			Message: "event ignored",
		})
	}
}

func (s *Service) handleGitHubPushWebhook(w http.ResponseWriter, event string, body []byte) {
	var payload githubWebhookPushPayload
	if err := json.Unmarshal(body, &payload); err != nil {
		writeGitHubWebhookJSON(w, http.StatusBadRequest, githubWebhookResponse{
			OK:      false,
			Event:   event,
			Message: "invalid push payload",
		})
		return
	}

	branch := branchFromGitHubRef(payload.Ref)
	repoFullName := strings.TrimSpace(payload.Repository.FullName)
	if repoFullName == "" || branch == "" {
		writeGitHubWebhookJSON(w, http.StatusBadRequest, githubWebhookResponse{
			OK:      false,
			Event:   event,
			Message: "push payload is missing repository or branch",
		})
		return
	}

	matchingDeployments, err := findDeploymentsForGitHubPush(repoFullName, branch)
	if err != nil {
		logger.Error("[GitHubWebhook] Failed to find deployments for %s/%s: %v", repoFullName, branch, err)
		writeGitHubWebhookJSON(w, http.StatusInternalServerError, githubWebhookResponse{
			OK:         false,
			Event:      event,
			Repository: repoFullName,
			Branch:     branch,
			Message:    "failed to find matching deployments",
		})
		return
	}

	triggered := make([]string, 0, len(matchingDeployments))
	for _, deployment := range matchingDeployments {
		deploymentID := deployment.ID
		triggered = append(triggered, deploymentID)

		go s.triggerDeploymentFromGitHubPush(deploymentID, repoFullName, branch, payload.After, payloadSender(payload))
	}

	writeGitHubWebhookJSON(w, http.StatusAccepted, githubWebhookResponse{
		OK:                 true,
		Event:              event,
		Repository:         repoFullName,
		Branch:             branch,
		MatchedDeployments: len(matchingDeployments),
		Triggered:          triggered,
		Message:            "push webhook accepted",
	})
}

func (s *Service) triggerDeploymentFromGitHubPush(deploymentID, repoFullName, branch, commitSHA, sender string) {
	ctx, cancel := s.detachedContext(5 * time.Minute)
	defer cancel()

	deployment, err := s.repo.GetByID(ctx, deploymentID)
	if err != nil {
		logger.Error("[GitHubWebhook] Failed to load deployment %s for %s@%s (%s) by %s: %v", deploymentID, repoFullName, branch, commitSHA, sender, err)
		return
	}
	if deployment.AutoDeploy != nil && !*deployment.AutoDeploy {
		logger.Info("[GitHubWebhook] Auto-deploy disabled for deployment %s; skipping %s@%s (%s) by %s", deploymentID, repoFullName, branch, commitSHA, sender)
		return
	}

	s.notifyGitHubAutoDeployStarted(ctx, deployment, repoFullName, branch, commitSHA, sender)

	ctx = auth.WithSystemUser(ctx)
	_, err = s.TriggerDeployment(ctx, connect.NewRequest(&deploymentsv1.TriggerDeploymentRequest{
		DeploymentId: deploymentID,
	}))
	if err != nil {
		logger.Error("[GitHubWebhook] Failed to trigger deployment %s for %s@%s (%s) by %s: %v", deploymentID, repoFullName, branch, commitSHA, sender, err)
		return
	}

	logger.Info("[GitHubWebhook] Triggered deployment %s for %s@%s (%s) by %s", deploymentID, repoFullName, branch, commitSHA, sender)
}

func (s *Service) notifyGitHubAutoDeployStarted(ctx context.Context, deployment *database.Deployment, repoFullName, branch, commitSHA, sender string) {
	deploymentName := deployment.Name
	if deploymentName == "" {
		deploymentName = deployment.Domain
	}
	if deploymentName == "" {
		deploymentName = deployment.ID
	}

	shortSHA := commitSHA
	if len(shortSHA) > 12 {
		shortSHA = shortSHA[:12]
	}
	if shortSHA == "" {
		shortSHA = "latest commit"
	}

	actionURL := fmt.Sprintf("/deployments/%s", deployment.ID)
	actionLabel := "View Deployment"
	metadata := map[string]string{
		"deployment_id": deployment.ID,
		"repository":    repoFullName,
		"branch":        branch,
		"commit_sha":    commitSHA,
		"sender":        sender,
		"trigger":       "github_push",
	}

	if err := notifications.CreateNotificationForOrganization(
		ctx,
		deployment.OrganizationID,
		notificationsv1.NotificationType_NOTIFICATION_TYPE_DEPLOYMENT,
		notificationsv1.NotificationSeverity_NOTIFICATION_SEVERITY_LOW,
		"Auto Deploy Started",
		fmt.Sprintf("%s is deploying %s from %s@%s after a GitHub push by %s.", deploymentName, shortSHA, repoFullName, branch, sender),
		&actionURL,
		&actionLabel,
		metadata,
		nil,
	); err != nil {
		logger.Warn("[GitHubWebhook] Failed to create auto-deploy notification for deployment %s: %v", deployment.ID, err)
	}
}

func (s *Service) ensureGitHubWebhookForDeployment(ctx context.Context, deployment *database.Deployment) error {
	if deployment == nil || deployment.RepositoryURL == nil || deployment.GitHubIntegrationID == nil {
		return nil
	}

	repoFullName := normalizeGitHubRepoFullName(*deployment.RepositoryURL)
	if repoFullName == "" {
		return nil
	}

	webhookURL, err := resolveGitHubWebhookURL()
	if err != nil {
		return err
	}

	webhookSecret := os.Getenv("GITHUB_WEBHOOK_SECRET")
	if webhookSecret == "" {
		return fmt.Errorf("GITHUB_WEBHOOK_SECRET is required to enable automatic GitHub deployments")
	}

	githubToken, err := s.getGitHubToken(ctx, deployment.OrganizationID, *deployment.GitHubIntegrationID)
	if err != nil {
		return fmt.Errorf("failed to get GitHub integration token: %w", err)
	}

	var integration database.GitHubIntegration
	if err := database.DB.Where("id = ?", *deployment.GitHubIntegrationID).First(&integration).Error; err == nil && githubIntegrationUsesApp(integration) {
		logger.Info("[GitHubWebhook] GitHub App installation manages webhooks for %s; skipping repository webhook creation", repoFullName)
		return nil
	}

	client := githubclient.NewClient(githubToken)
	hooks, err := client.ListHooks(ctx, repoFullName)
	if err != nil {
		return fmt.Errorf("failed to list GitHub webhooks for %s: %w", repoFullName, err)
	}

	hookRequest := githubclient.CreateHookRequest{
		Name:   "web",
		Active: true,
		Events: []string{"push"},
		Config: map[string]string{
			"url":          webhookURL,
			"content_type": "json",
			"secret":       webhookSecret,
		},
	}

	for _, hook := range hooks {
		if strings.EqualFold(strings.TrimSpace(hook.Config.URL), webhookURL) {
			if _, err := client.UpdateHook(ctx, repoFullName, hook.ID, hookRequest); err != nil {
				return fmt.Errorf("failed to update GitHub webhook for %s: %w", repoFullName, err)
			}
			logger.Info("[GitHubWebhook] Webhook already existed and was refreshed for %s -> %s", repoFullName, webhookURL)
			return nil
		}
	}

	_, err = client.CreateHook(ctx, repoFullName, hookRequest)
	if err != nil {
		return fmt.Errorf("failed to create GitHub webhook for %s: %w", repoFullName, err)
	}

	logger.Info("[GitHubWebhook] Created webhook for %s -> %s", repoFullName, webhookURL)
	return nil
}

func resolveGitHubWebhookURL() (string, error) {
	if explicitURL := strings.TrimSpace(os.Getenv("GITHUB_WEBHOOK_URL")); explicitURL != "" {
		return strings.TrimRight(explicitURL, "/"), nil
	}

	apiURL := strings.TrimSpace(os.Getenv("API_URL"))
	if apiURL == "" {
		apiURL = strings.TrimSpace(os.Getenv("NUXT_PUBLIC_API_HOST"))
	}
	if apiURL == "" {
		return "", fmt.Errorf("GITHUB_WEBHOOK_URL or API_URL must be set to a public API URL")
	}

	return strings.TrimRight(apiURL, "/") + "/webhooks/github", nil
}

func verifyGitHubWebhookSignature(body []byte, signatureHeader string) error {
	secret := os.Getenv("GITHUB_WEBHOOK_SECRET")
	if secret == "" {
		if os.Getenv("DISABLE_AUTH") == "true" {
			logger.Warn("[GitHubWebhook] GITHUB_WEBHOOK_SECRET is not set; accepting webhook because DISABLE_AUTH=true")
			return nil
		}
		return fmt.Errorf("GITHUB_WEBHOOK_SECRET is required")
	}

	const prefix = "sha256="
	if !strings.HasPrefix(signatureHeader, prefix) {
		return fmt.Errorf("missing sha256 signature")
	}

	providedSignature, err := hex.DecodeString(strings.TrimPrefix(signatureHeader, prefix))
	if err != nil {
		return fmt.Errorf("invalid signature encoding: %w", err)
	}

	mac := hmac.New(sha256.New, []byte(secret))
	_, _ = mac.Write(body)
	expectedSignature := mac.Sum(nil)

	if !hmac.Equal(providedSignature, expectedSignature) {
		return fmt.Errorf("signature mismatch")
	}

	return nil
}

func findDeploymentsForGitHubPush(repoFullName, branch string) ([]database.Deployment, error) {
	var deployments []database.Deployment
	if err := database.DB.
		Where("deleted_at IS NULL").
		Where("github_integration_id IS NOT NULL AND github_integration_id <> ''").
		Where("(auto_deploy IS NULL OR auto_deploy = ?)", true).
		Where("branch = ?", branch).
		Find(&deployments).Error; err != nil {
		return nil, err
	}

	matches := make([]database.Deployment, 0, len(deployments))
	for _, deployment := range deployments {
		if deployment.RepositoryURL == nil {
			continue
		}
		if githubRepoURLMatchesFullName(*deployment.RepositoryURL, repoFullName) {
			matches = append(matches, deployment)
		}
	}

	return matches, nil
}

func githubRepoURLMatchesFullName(repoURL, repoFullName string) bool {
	normalizedRepo := normalizeGitHubRepoFullName(repoFullName)
	if normalizedRepo == "" {
		return false
	}
	return normalizeGitHubRepoFullName(repoURL) == normalizedRepo
}

func normalizeGitHubRepoFullName(value string) string {
	value = strings.TrimSpace(value)
	value = strings.TrimSuffix(value, ".git")
	value = strings.TrimSuffix(value, "/")

	switch {
	case strings.HasPrefix(value, "https://github.com/"):
		value = strings.TrimPrefix(value, "https://github.com/")
	case strings.HasPrefix(value, "http://github.com/"):
		value = strings.TrimPrefix(value, "http://github.com/")
	case strings.HasPrefix(value, "git@github.com:"):
		value = strings.TrimPrefix(value, "git@github.com:")
	}

	parts := strings.Split(value, "/")
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return ""
	}

	return strings.ToLower(parts[0] + "/" + parts[1])
}

func branchFromGitHubRef(ref string) string {
	const branchPrefix = "refs/heads/"
	if !strings.HasPrefix(ref, branchPrefix) {
		return ""
	}
	return strings.TrimPrefix(ref, branchPrefix)
}

func payloadSender(payload githubWebhookPushPayload) string {
	if payload.Sender == nil {
		return "github"
	}
	if payload.Sender.Login == "" {
		return "github"
	}
	return payload.Sender.Login
}

func writeGitHubWebhookJSON(w http.ResponseWriter, status int, payload githubWebhookResponse) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}
