package github

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const (
	githubAPIResponseBodyLimit = 4 << 20
	githubBranchesPerPage      = 100
	githubBranchesMaxPages     = 100
)

type Client struct {
	token      string
	baseURL    string
	httpClient *http.Client
	appID      int64
}

func NewClient(token string) *Client {
	return &Client{
		token:   token,
		baseURL: "https://api.github.com",
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

type GitHubRepo struct {
	ID            int64  `json:"id"`
	Name          string `json:"name"`
	FullName      string `json:"full_name"`
	Description   string `json:"description"`
	URL           string `json:"html_url"`
	IsPrivate     bool   `json:"private"`
	DefaultBranch string `json:"default_branch"`
}

type GitHubBranch struct {
	Name   string `json:"name"`
	Commit struct {
		SHA string `json:"sha"`
	} `json:"commit"`
}

type GitHubFileContent struct {
	Content  string `json:"content"`
	Encoding string `json:"encoding"`
	Size     int64  `json:"size"`
}

type PullRequestFile struct {
	Filename string `json:"filename"`
}

type PullRequest struct {
	State  string `json:"state"`
	Draft  bool   `json:"draft"`
	Merged bool   `json:"merged"`
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
}

type Deployment struct {
	ID int64 `json:"id"`
}

type IssueComment struct {
	ID                    int64  `json:"id"`
	Body                  string `json:"body"`
	PerformedViaGitHubApp *struct {
		ID int64 `json:"id"`
	} `json:"performed_via_github_app"`
}

type CheckRun struct {
	ID int64 `json:"id"`
}

type CheckRunUpdate struct {
	Name       string
	HeadSHA    string
	DetailsURL string
	ExternalID string
	Status     string
	Conclusion string
	Title      string
	Summary    string
}

func (c *Client) CreateCheckRun(ctx context.Context, repoFullName string, update CheckRunUpdate) (int64, error) {
	repositoryPath, err := escapedGitHubRepositoryPath(repoFullName)
	if err != nil {
		return 0, err
	}
	var checkRun CheckRun
	requestURL := fmt.Sprintf("%s/repos/%s/check-runs", strings.TrimRight(c.baseURL, "/"), repositoryPath)
	if err := c.doJSON(ctx, http.MethodPost, requestURL, checkRunBody(update, true), http.StatusCreated, &checkRun); err != nil {
		return 0, err
	}
	if checkRun.ID <= 0 {
		return 0, fmt.Errorf("GitHub check run response did not include an ID")
	}
	return checkRun.ID, nil
}

func (c *Client) UpdateCheckRun(ctx context.Context, repoFullName string, checkRunID int64, update CheckRunUpdate) error {
	repositoryPath, err := escapedGitHubRepositoryPath(repoFullName)
	if err != nil {
		return err
	}
	if checkRunID <= 0 {
		return fmt.Errorf("GitHub check run ID is required")
	}
	requestURL := fmt.Sprintf("%s/repos/%s/check-runs/%d", strings.TrimRight(c.baseURL, "/"), repositoryPath, checkRunID)
	return c.doJSON(ctx, http.MethodPatch, requestURL, checkRunBody(update, false), http.StatusOK, nil)
}

func checkRunBody(update CheckRunUpdate, create bool) map[string]interface{} {
	body := map[string]interface{}{
		"name":        update.Name,
		"external_id": update.ExternalID,
		"status":      update.Status,
		"output":      map[string]string{"title": update.Title, "summary": update.Summary},
	}
	if update.DetailsURL != "" {
		body["details_url"] = update.DetailsURL
	}
	if create {
		body["head_sha"] = update.HeadSHA
	}
	if update.Conclusion != "" {
		body["conclusion"] = update.Conclusion
	}
	return body
}

func (c *Client) ListPullRequestFiles(ctx context.Context, repoFullName string, pullRequestNumber int64) ([]PullRequestFile, error) {
	repositoryPath, err := escapedGitHubRepositoryPath(repoFullName)
	if err != nil {
		return nil, err
	}
	if pullRequestNumber <= 0 {
		return nil, fmt.Errorf("pull request number is required")
	}

	files := make([]PullRequestFile, 0)
	for page := 1; page <= 30; page++ {
		requestURL := fmt.Sprintf("%s/repos/%s/pulls/%d/files?per_page=100&page=%d", strings.TrimRight(c.baseURL, "/"), repositoryPath, pullRequestNumber, page)
		var batch []PullRequestFile
		if err := c.doJSON(ctx, http.MethodGet, requestURL, nil, http.StatusOK, &batch); err != nil {
			return nil, err
		}
		files = append(files, batch...)
		if len(batch) < 100 {
			return files, nil
		}
	}
	return nil, fmt.Errorf("pull request contains more than 3000 files")
}

func (c *Client) GetPullRequest(ctx context.Context, repoFullName string, pullRequestNumber int64) (*PullRequest, error) {
	repositoryPath, err := escapedGitHubRepositoryPath(repoFullName)
	if err != nil {
		return nil, err
	}
	if pullRequestNumber <= 0 {
		return nil, fmt.Errorf("pull request number is required")
	}
	var pullRequest PullRequest
	requestURL := fmt.Sprintf("%s/repos/%s/pulls/%d", strings.TrimRight(c.baseURL, "/"), repositoryPath, pullRequestNumber)
	if err := c.doJSON(ctx, http.MethodGet, requestURL, nil, http.StatusOK, &pullRequest); err != nil {
		return nil, err
	}
	return &pullRequest, nil
}

func (c *Client) CreateDeployment(ctx context.Context, repoFullName, ref, environment, description string) (int64, error) {
	repositoryPath, err := escapedGitHubRepositoryPath(repoFullName)
	if err != nil {
		return 0, err
	}
	body := map[string]interface{}{
		"ref":                    ref,
		"environment":            environment,
		"description":            description,
		"auto_merge":             false,
		"required_contexts":      []string{},
		"transient_environment":  true,
		"production_environment": false,
	}
	var deployment Deployment
	requestURL := fmt.Sprintf("%s/repos/%s/deployments", strings.TrimRight(c.baseURL, "/"), repositoryPath)
	if err := c.doJSON(ctx, http.MethodPost, requestURL, body, http.StatusCreated, &deployment); err != nil {
		return 0, err
	}
	if deployment.ID <= 0 {
		return 0, fmt.Errorf("GitHub deployment response did not include an ID")
	}
	return deployment.ID, nil
}

func (c *Client) CreateDeploymentStatus(ctx context.Context, repoFullName string, deploymentID int64, state, description, environmentURL, logURL string) error {
	repositoryPath, err := escapedGitHubRepositoryPath(repoFullName)
	if err != nil {
		return err
	}
	if deploymentID <= 0 {
		return fmt.Errorf("GitHub deployment ID is required")
	}
	body := map[string]interface{}{
		"state":         state,
		"description":   description,
		"auto_inactive": false,
	}
	if environmentURL != "" {
		body["environment_url"] = environmentURL
	}
	if logURL != "" {
		body["log_url"] = logURL
	}
	requestURL := fmt.Sprintf("%s/repos/%s/deployments/%d/statuses", strings.TrimRight(c.baseURL, "/"), repositoryPath, deploymentID)
	return c.doJSON(ctx, http.MethodPost, requestURL, body, http.StatusCreated, nil)
}

func (c *Client) FindIssueComment(ctx context.Context, repoFullName string, issueNumber int64, marker string) (*IssueComment, error) {
	repositoryPath, err := escapedGitHubRepositoryPath(repoFullName)
	if err != nil {
		return nil, err
	}
	if issueNumber <= 0 || marker == "" {
		return nil, fmt.Errorf("issue number and comment marker are required")
	}
	for page := 1; page <= 10; page++ {
		requestURL := fmt.Sprintf("%s/repos/%s/issues/%d/comments?per_page=100&page=%d", strings.TrimRight(c.baseURL, "/"), repositoryPath, issueNumber, page)
		var comments []IssueComment
		if err := c.doJSON(ctx, http.MethodGet, requestURL, nil, http.StatusOK, &comments); err != nil {
			return nil, err
		}
		for i := range comments {
			if strings.Contains(comments[i].Body, marker) && comments[i].PerformedViaGitHubApp != nil && comments[i].PerformedViaGitHubApp.ID == c.appID && c.appID > 0 {
				return &comments[i], nil
			}
		}
		if len(comments) < 100 {
			return nil, nil
		}
	}
	return nil, fmt.Errorf("could not safely search more than 1000 issue comments")
}

func (c *Client) CreateIssueComment(ctx context.Context, repoFullName string, issueNumber int64, body string) (int64, error) {
	repositoryPath, err := escapedGitHubRepositoryPath(repoFullName)
	if err != nil {
		return 0, err
	}
	var comment IssueComment
	requestURL := fmt.Sprintf("%s/repos/%s/issues/%d/comments", strings.TrimRight(c.baseURL, "/"), repositoryPath, issueNumber)
	if err := c.doJSON(ctx, http.MethodPost, requestURL, map[string]string{"body": body}, http.StatusCreated, &comment); err != nil {
		return 0, err
	}
	return comment.ID, nil
}

func (c *Client) UpdateIssueComment(ctx context.Context, repoFullName string, commentID int64, body string) error {
	repositoryPath, err := escapedGitHubRepositoryPath(repoFullName)
	if err != nil {
		return err
	}
	if commentID <= 0 {
		return fmt.Errorf("GitHub comment ID is required")
	}
	requestURL := fmt.Sprintf("%s/repos/%s/issues/comments/%d", strings.TrimRight(c.baseURL, "/"), repositoryPath, commentID)
	return c.doJSON(ctx, http.MethodPatch, requestURL, map[string]string{"body": body}, http.StatusOK, nil)
}

func (c *Client) DeleteIssueComment(ctx context.Context, repoFullName string, commentID int64) error {
	repositoryPath, err := escapedGitHubRepositoryPath(repoFullName)
	if err != nil {
		return err
	}
	if commentID <= 0 {
		return fmt.Errorf("GitHub comment ID is required")
	}
	requestURL := fmt.Sprintf("%s/repos/%s/issues/comments/%d", strings.TrimRight(c.baseURL, "/"), repositoryPath, commentID)
	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, requestURL, nil)
	if err != nil {
		return err
	}
	if c.token != "" {
		req.Header.Set("Authorization", "Bearer "+c.token)
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("GitHub API request failed: %w", err)
	}
	defer resp.Body.Close()
	body, err := readGitHubAPIResponseBody(resp.Body)
	if err != nil {
		return err
	}
	if resp.StatusCode == http.StatusNoContent || resp.StatusCode == http.StatusNotFound {
		return nil
	}
	return fmt.Errorf("GitHub API %s %s failed: %d - %s", http.MethodDelete, requestURL, resp.StatusCode, strings.TrimSpace(string(body)))
}

func (c *Client) doJSON(ctx context.Context, method, requestURL string, payload interface{}, expectedStatus int, target interface{}) error {
	var bodyReader io.Reader
	if payload != nil {
		body, err := json.Marshal(payload)
		if err != nil {
			return fmt.Errorf("failed to encode GitHub API request: %w", err)
		}
		bodyReader = bytes.NewReader(body)
	}
	req, err := http.NewRequestWithContext(ctx, method, requestURL, bodyReader)
	if err != nil {
		return err
	}
	if c.token != "" {
		req.Header.Set("Authorization", "Bearer "+c.token)
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")
	if payload != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("GitHub API request failed: %w", err)
	}
	defer resp.Body.Close()
	body, err := readGitHubAPIResponseBody(resp.Body)
	if err != nil {
		return err
	}
	if resp.StatusCode != expectedStatus {
		return fmt.Errorf("GitHub API %s %s failed: %d - %s", method, requestURL, resp.StatusCode, strings.TrimSpace(string(body)))
	}
	if target != nil && len(body) > 0 {
		if err := json.Unmarshal(body, target); err != nil {
			return fmt.Errorf("failed to decode GitHub API response: %w", err)
		}
	}
	return nil
}

func (c *Client) ListInstallationRepos(ctx context.Context, page, perPage int) ([]GitHubRepo, error) {
	requestURL, err := url.Parse(strings.TrimRight(c.baseURL, "/") + "/installation/repositories")
	if err != nil {
		return nil, fmt.Errorf("invalid GitHub API base URL: %w", err)
	}
	query := requestURL.Query()
	query.Set("page", fmt.Sprintf("%d", page))
	query.Set("per_page", fmt.Sprintf("%d", perPage))
	requestURL.RawQuery = query.Encode()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, requestURL.String(), nil)
	if err != nil {
		return nil, err
	}

	if c.token != "" {
		req.Header.Set("Authorization", "Bearer "+c.token)
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("github API request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := readGitHubAPIResponseBody(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read GitHub API response: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
			return nil, fmt.Errorf("github app authentication failed (installation may be suspended, revoked, or missing repository access): %d - %s", resp.StatusCode, string(body))
		}
		return nil, fmt.Errorf("github API error: %d - %s", resp.StatusCode, string(body))
	}

	var payload struct {
		Repositories []GitHubRepo `json:"repositories"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		return nil, fmt.Errorf("failed to decode installation repos: %w", err)
	}

	return payload.Repositories, nil
}

func (c *Client) ListBranches(ctx context.Context, repoFullName string) ([]GitHubBranch, error) {
	repositoryPath, err := escapedGitHubRepositoryPath(repoFullName)
	if err != nil {
		return nil, err
	}
	requestURL, err := url.Parse(fmt.Sprintf("%s/repos/%s/branches", strings.TrimRight(c.baseURL, "/"), repositoryPath))
	if err != nil {
		return nil, fmt.Errorf("invalid GitHub API URL: %w", err)
	}
	query := requestURL.Query()
	query.Set("per_page", fmt.Sprintf("%d", githubBranchesPerPage))
	requestURL.RawQuery = query.Encode()

	branches := make([]GitHubBranch, 0, githubBranchesPerPage)
	for page := 1; requestURL != nil; page++ {
		if page > githubBranchesMaxPages {
			return nil, fmt.Errorf("repository branch list exceeds the supported limit of %d", githubBranchesPerPage*githubBranchesMaxPages)
		}

		pageBranches, nextURL, err := c.listBranchesPage(ctx, requestURL)
		if err != nil {
			return nil, err
		}
		branches = append(branches, pageBranches...)
		requestURL = nextURL
	}

	return branches, nil
}

func (c *Client) listBranchesPage(ctx context.Context, requestURL *url.URL) ([]GitHubBranch, *url.URL, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, requestURL.String(), nil)
	if err != nil {
		return nil, nil, err
	}

	if c.token != "" {
		req.Header.Set("Authorization", "Bearer "+c.token)
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, nil, fmt.Errorf("github API request failed: %w", err)
	}
	body, readErr := readGitHubAPIResponseBody(resp.Body)
	closeErr := resp.Body.Close()
	if readErr != nil {
		return nil, nil, fmt.Errorf("failed to read GitHub API response: %w", readErr)
	}
	if closeErr != nil {
		return nil, nil, fmt.Errorf("failed to close GitHub API response: %w", closeErr)
	}
	if resp.StatusCode != http.StatusOK {
		if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
			return nil, nil, fmt.Errorf("github authentication failed (token may be expired or revoked): %d - %s", resp.StatusCode, string(body))
		}
		return nil, nil, fmt.Errorf("github API error: %d - %s", resp.StatusCode, string(body))
	}

	var branches []GitHubBranch
	if err := json.Unmarshal(body, &branches); err != nil {
		return nil, nil, fmt.Errorf("failed to decode branches: %w", err)
	}

	nextURL, err := c.githubPaginationURL(resp.Header.Get("Link"), "next")
	if err != nil {
		return nil, nil, err
	}
	return branches, nextURL, nil
}

func (c *Client) githubPaginationURL(linkHeader, relation string) (*url.URL, error) {
	for _, link := range strings.Split(linkHeader, ",") {
		parts := strings.Split(link, ";")
		if len(parts) < 2 {
			continue
		}
		matchesRelation := false
		for _, parameter := range parts[1:] {
			if strings.TrimSpace(parameter) == fmt.Sprintf(`rel="%s"`, relation) {
				matchesRelation = true
				break
			}
		}
		if !matchesRelation {
			continue
		}

		rawURL := strings.TrimSpace(parts[0])
		if len(rawURL) < 2 || rawURL[0] != '<' || rawURL[len(rawURL)-1] != '>' {
			return nil, fmt.Errorf("invalid GitHub pagination link")
		}
		nextURL, err := url.Parse(rawURL[1 : len(rawURL)-1])
		if err != nil {
			return nil, fmt.Errorf("invalid GitHub pagination URL: %w", err)
		}
		baseURL, err := url.Parse(c.baseURL)
		if err != nil {
			return nil, fmt.Errorf("invalid GitHub API base URL: %w", err)
		}
		if nextURL.Scheme != baseURL.Scheme || nextURL.Host != baseURL.Host {
			return nil, fmt.Errorf("GitHub pagination URL points to an unexpected host")
		}
		return nextURL, nil
	}
	return nil, nil
}

func (c *Client) GetRepository(ctx context.Context, repoFullName string) (*GitHubRepo, error) {
	repositoryPath, err := escapedGitHubRepositoryPath(repoFullName)
	if err != nil {
		return nil, err
	}
	requestURL := fmt.Sprintf("%s/repos/%s", strings.TrimRight(c.baseURL, "/"), repositoryPath)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, requestURL, nil)
	if err != nil {
		return nil, err
	}
	if c.token != "" {
		req.Header.Set("Authorization", "Bearer "+c.token)
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("github API request failed: %w", err)
	}
	defer resp.Body.Close()
	body, err := readGitHubAPIResponseBody(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read GitHub API response: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
			return nil, fmt.Errorf("github authentication failed (token may be expired or revoked): %d - %s", resp.StatusCode, string(body))
		}
		return nil, fmt.Errorf("github API error: %d - %s", resp.StatusCode, string(body))
	}

	var repository GitHubRepo
	if err := json.Unmarshal(body, &repository); err != nil {
		return nil, fmt.Errorf("failed to decode repository: %w", err)
	}
	return &repository, nil
}

func (c *Client) GetFile(ctx context.Context, repoFullName, branch, path string) (*GitHubFileContent, error) {
	repositoryPath, err := escapedGitHubRepositoryPath(repoFullName)
	if err != nil {
		return nil, err
	}
	contentPath, err := escapedGitHubContentPath(path)
	if err != nil {
		return nil, err
	}
	requestURL, err := url.Parse(fmt.Sprintf("%s/repos/%s/contents/%s", strings.TrimRight(c.baseURL, "/"), repositoryPath, contentPath))
	if err != nil {
		return nil, fmt.Errorf("invalid GitHub API URL: %w", err)
	}
	query := requestURL.Query()
	query.Set("ref", branch)
	requestURL.RawQuery = query.Encode()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, requestURL.String(), nil)
	if err != nil {
		return nil, err
	}

	if c.token != "" {
		req.Header.Set("Authorization", "Bearer "+c.token)
	}
	req.Header.Set("Accept", "application/vnd.github.v3+json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("github API request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := readGitHubAPIResponseBody(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read GitHub API response: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		// Check for authentication errors (401/403) which indicate expired/revoked tokens
		if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
			return nil, fmt.Errorf("github authentication failed (token may be expired or revoked): %d - %s", resp.StatusCode, string(body))
		}
		return nil, fmt.Errorf("github API error: %d - %s", resp.StatusCode, string(body))
	}

	var fileContent GitHubFileContent
	if err := json.Unmarshal(body, &fileContent); err != nil {
		return nil, fmt.Errorf("failed to decode file: %w", err)
	}

	// Decode base64 content if needed
	if fileContent.Encoding == "base64" {
		decoded, err := base64.StdEncoding.DecodeString(fileContent.Content)
		if err != nil {
			return nil, fmt.Errorf("failed to decode base64: %w", err)
		}
		fileContent.Content = string(decoded)
		fileContent.Encoding = "text"
	}

	return &fileContent, nil
}

func escapedGitHubRepositoryPath(repoFullName string) (string, error) {
	parts := strings.Split(strings.TrimSpace(repoFullName), "/")
	if len(parts) != 2 || !isSafeGitHubPathSegment(parts[0]) || !isSafeGitHubPathSegment(parts[1]) {
		return "", fmt.Errorf("invalid GitHub repository name %q", repoFullName)
	}
	return url.PathEscape(parts[0]) + "/" + url.PathEscape(parts[1]), nil
}

func isSafeGitHubPathSegment(value string) bool {
	return value != "" && value != "." && value != ".." && !strings.ContainsAny(value, "\x00\r\n")
}

func escapedGitHubContentPath(path string) (string, error) {
	parts := strings.Split(path, "/")
	if len(parts) == 0 {
		return "", fmt.Errorf("GitHub file path is required")
	}
	escaped := make([]string, 0, len(parts))
	for _, part := range parts {
		if part == "" || part == "." || part == ".." {
			return "", fmt.Errorf("invalid GitHub file path %q", path)
		}
		escaped = append(escaped, url.PathEscape(part))
	}
	return strings.Join(escaped, "/"), nil
}

func readGitHubAPIResponseBody(body io.Reader) ([]byte, error) {
	data, err := io.ReadAll(io.LimitReader(body, githubAPIResponseBodyLimit+1))
	if err != nil {
		return nil, err
	}
	if len(data) > githubAPIResponseBodyLimit {
		return nil, fmt.Errorf("response exceeds %d bytes", githubAPIResponseBodyLimit)
	}
	return data, nil
}
