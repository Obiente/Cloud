package github

import (
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

const githubAPIResponseBodyLimit = 4 << 20

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
	requestURL := fmt.Sprintf("%s/repos/%s/branches", strings.TrimRight(c.baseURL, "/"), repositoryPath)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, requestURL, nil)
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

	var branches []GitHubBranch
	if err := json.Unmarshal(body, &branches); err != nil {
		return nil, fmt.Errorf("failed to decode branches: %w", err)
	}

	return branches, nil
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
	parts := strings.Split(strings.TrimSpace(path), "/")
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
