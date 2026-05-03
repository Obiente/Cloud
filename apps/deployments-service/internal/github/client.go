package github

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

type Client struct {
	token      string
	baseURL    string
	httpClient *http.Client
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

type GitHubHook struct {
	ID     int64 `json:"id"`
	Active bool  `json:"active"`
	Config struct {
		URL         string `json:"url"`
		ContentType string `json:"content_type"`
	} `json:"config"`
	Events []string `json:"events"`
}

type CreateHookRequest struct {
	Name   string            `json:"name"`
	Active bool              `json:"active"`
	Events []string          `json:"events"`
	Config map[string]string `json:"config"`
}

func (c *Client) ListRepos(ctx context.Context, page, perPage int) ([]GitHubRepo, error) {
	url := fmt.Sprintf("%s/user/repos?page=%d&per_page=%d&sort=updated", c.baseURL, page, perPage)
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
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

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		// Check for authentication errors (401/403) which indicate expired/revoked tokens
		if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
			return nil, fmt.Errorf("github authentication failed (token may be expired or revoked): %d - %s", resp.StatusCode, string(body))
		}
		return nil, fmt.Errorf("github API error: %d - %s", resp.StatusCode, string(body))
	}

	var repos []GitHubRepo
	if err := json.NewDecoder(resp.Body).Decode(&repos); err != nil {
		return nil, fmt.Errorf("failed to decode repos: %w", err)
	}

	return repos, nil
}

func (c *Client) ListInstallationRepos(ctx context.Context, page, perPage int) ([]GitHubRepo, error) {
	url := fmt.Sprintf("%s/installation/repositories?page=%d&per_page=%d", c.baseURL, page, perPage)
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
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

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
			return nil, fmt.Errorf("github app authentication failed (installation may be suspended, revoked, or missing repository access): %d - %s", resp.StatusCode, string(body))
		}
		return nil, fmt.Errorf("github API error: %d - %s", resp.StatusCode, string(body))
	}

	var payload struct {
		Repositories []GitHubRepo `json:"repositories"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return nil, fmt.Errorf("failed to decode installation repos: %w", err)
	}

	return payload.Repositories, nil
}

func (c *Client) ListBranches(ctx context.Context, repoFullName string) ([]GitHubBranch, error) {
	url := fmt.Sprintf("%s/repos/%s/branches", c.baseURL, repoFullName)
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
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

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		// Check for authentication errors (401/403) which indicate expired/revoked tokens
		if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
			return nil, fmt.Errorf("github authentication failed (token may be expired or revoked): %d - %s", resp.StatusCode, string(body))
		}
		return nil, fmt.Errorf("github API error: %d - %s", resp.StatusCode, string(body))
	}

	var branches []GitHubBranch
	if err := json.NewDecoder(resp.Body).Decode(&branches); err != nil {
		return nil, fmt.Errorf("failed to decode branches: %w", err)
	}

	return branches, nil
}

func (c *Client) GetFile(ctx context.Context, repoFullName, branch, path string) (*GitHubFileContent, error) {
	url := fmt.Sprintf("%s/repos/%s/contents/%s?ref=%s", c.baseURL, repoFullName, path, branch)
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
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

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		// Check for authentication errors (401/403) which indicate expired/revoked tokens
		if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
			return nil, fmt.Errorf("github authentication failed (token may be expired or revoked): %d - %s", resp.StatusCode, string(body))
		}
		return nil, fmt.Errorf("github API error: %d - %s", resp.StatusCode, string(body))
	}

	var fileContent GitHubFileContent
	if err := json.NewDecoder(resp.Body).Decode(&fileContent); err != nil {
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

func (c *Client) ListHooks(ctx context.Context, repoFullName string) ([]GitHubHook, error) {
	url := fmt.Sprintf("%s/repos/%s/hooks", c.baseURL, repoFullName)
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
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

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
			return nil, formatGitHubWebhookAuthError(resp.StatusCode, body)
		}
		return nil, fmt.Errorf("github API error: %d - %s", resp.StatusCode, string(body))
	}

	var hooks []GitHubHook
	if err := json.NewDecoder(resp.Body).Decode(&hooks); err != nil {
		return nil, fmt.Errorf("failed to decode hooks: %w", err)
	}

	return hooks, nil
}

func (c *Client) CreateHook(ctx context.Context, repoFullName string, hook CreateHookRequest) (*GitHubHook, error) {
	payload, err := json.Marshal(hook)
	if err != nil {
		return nil, err
	}

	url := fmt.Sprintf("%s/repos/%s/hooks", c.baseURL, repoFullName)
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(payload))
	if err != nil {
		return nil, err
	}

	if c.token != "" {
		req.Header.Set("Authorization", "Bearer "+c.token)
	}
	req.Header.Set("Accept", "application/vnd.github.v3+json")
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("github API request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
			return nil, formatGitHubWebhookAuthError(resp.StatusCode, body)
		}
		return nil, fmt.Errorf("github API error: %d - %s", resp.StatusCode, string(body))
	}

	var created GitHubHook
	if err := json.NewDecoder(resp.Body).Decode(&created); err != nil {
		return nil, fmt.Errorf("failed to decode hook: %w", err)
	}

	return &created, nil
}

func (c *Client) UpdateHook(ctx context.Context, repoFullName string, hookID int64, hook CreateHookRequest) (*GitHubHook, error) {
	payload, err := json.Marshal(hook)
	if err != nil {
		return nil, err
	}

	url := fmt.Sprintf("%s/repos/%s/hooks/%d", c.baseURL, repoFullName, hookID)
	req, err := http.NewRequestWithContext(ctx, "PATCH", url, bytes.NewReader(payload))
	if err != nil {
		return nil, err
	}

	if c.token != "" {
		req.Header.Set("Authorization", "Bearer "+c.token)
	}
	req.Header.Set("Accept", "application/vnd.github.v3+json")
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("github API request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
			return nil, formatGitHubWebhookAuthError(resp.StatusCode, body)
		}
		return nil, fmt.Errorf("github API error: %d - %s", resp.StatusCode, string(body))
	}

	var updated GitHubHook
	if err := json.NewDecoder(resp.Body).Decode(&updated); err != nil {
		return nil, fmt.Errorf("failed to decode hook: %w", err)
	}

	return &updated, nil
}

func formatGitHubWebhookAuthError(statusCode int, body []byte) error {
	bodyText := strings.TrimSpace(string(body))
	lowerBody := strings.ToLower(bodyText)
	if strings.Contains(lowerBody, "resource not accessible by integration") {
		return fmt.Errorf("github webhook permission denied: the connected GitHub account can access this repository but cannot manage its webhooks. Make sure the OAuth app is approved for the organization, the connection includes admin:repo_hook, and the GitHub user has Admin access to the repository: %d - %s", statusCode, bodyText)
	}

	return fmt.Errorf("github authentication failed (token may be expired, revoked, missing admin:repo_hook scope, or missing repository Admin access): %d - %s", statusCode, bodyText)
}
