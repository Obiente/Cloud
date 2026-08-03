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

const (
	githubAPIResponseBodyLimit = 4 << 20
	githubBranchesPerPage      = 100
	githubBranchesMaxPages     = 100
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
