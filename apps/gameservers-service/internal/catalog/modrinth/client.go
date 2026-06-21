package modrinth

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

const (
	defaultBaseURL = "https://api.modrinth.com/v2"
	defaultUA      = "obiente-cloud-gameservers-service"
	maxLimit       = 100
	maxAttempts    = 3
)

var ErrNotFound = errors.New("modrinth resource not found")

// RateLimitError indicates Modrinth asked the client to slow down.
type RateLimitError struct {
	Operation  string
	StatusCode int
	RetryAfter time.Duration
	Body       string
}

func (e *RateLimitError) Error() string {
	if e.RetryAfter > 0 {
		return fmt.Sprintf("%s rate limited: retry after %s", e.Operation, e.RetryAfter.Round(time.Second))
	}
	return fmt.Sprintf("%s rate limited", e.Operation)
}

// Client wraps the Modrinth REST API.
type Client struct {
	httpClient *http.Client
	baseURL    string
	userAgent  string
}

// NewClient creates a new Modrinth client with sane defaults.
func NewClient(httpClient *http.Client) *Client {
	if httpClient == nil {
		httpClient = &http.Client{
			Timeout: 30 * time.Second,
		}
	}

	return &Client{
		httpClient: httpClient,
		baseURL:    defaultBaseURL,
		userAgent:  defaultUA,
	}
}

// SearchParams describes project search filters.
type SearchParams struct {
	Query        string
	Limit        int
	Offset       int
	ProjectType  string
	Loaders      []string
	GameVersions []string
	Categories   []string
}

// SearchResult contains paginated search results.
type SearchResult struct {
	Projects  []Project
	TotalHits int
	Limit     int
	Offset    int
}

// Project is a simplified Modrinth project.
type Project struct {
	ID            string
	Slug          string
	Title         string
	Description   string
	ProjectType   string
	IconURL       string
	Categories    []string
	Loaders       []string
	GameVersions  []string
	Authors       []string
	Downloads     int64
	Rating        float64
	LatestVersion string
	ProjectURL    string
	SourceURL     string
	IssuesURL     string
	Body          string   // Full body/description with markdown
	Gallery       []string // Screenshot/image URLs
}

// VersionFilter describes version filtering options.
type VersionFilter struct {
	Loaders      []string
	GameVersions []string
	Limit        int
}

// Version represents a Modrinth version entity.
type Version struct {
	ID             string
	ProjectID      string
	Name           string
	VersionNumber  string
	VersionType    string
	GameVersions   []string
	Loaders        []string
	ServerSide     string
	ClientSide     string
	Changelog      string
	DatePublished  time.Time
	Files          []VersionFile
	PrimaryFileURL string
}

// VersionFile represents a downloadable artifact.
type VersionFile struct {
	Hashes   map[string]string `json:"hashes"`
	Primary  bool              `json:"primary"`
	Filename string            `json:"filename"`
	URL      string            `json:"url"`
	Size     int64             `json:"size"`
}

// SearchProjects queries Modrinth projects with applied filters.
func (c *Client) SearchProjects(ctx context.Context, params SearchParams) (*SearchResult, error) {
	limit := params.Limit
	if limit <= 0 || limit > maxLimit {
		limit = 20
	}
	if params.Offset < 0 {
		params.Offset = 0
	}

	facets := buildFacets(params)
	facetsJSON, err := json.Marshal(facets)
	if err != nil {
		return nil, fmt.Errorf("encode facets: %w", err)
	}

	values := url.Values{}
	values.Set("limit", fmt.Sprintf("%d", limit))
	values.Set("offset", fmt.Sprintf("%d", params.Offset))
	values.Set("facets", string(facetsJSON))
	if params.Query != "" {
		values.Set("query", params.Query)
	}
	values.Set("index", "relevance")

	body, err := c.get(ctx, fmt.Sprintf("%s/search?%s", c.baseURL, values.Encode()), "modrinth search")
	if err != nil {
		return nil, err
	}

	var payload searchResponse
	if err := json.Unmarshal(body, &payload); err != nil {
		return nil, fmt.Errorf("decode search response: %w", err)
	}

	results := make([]Project, 0, len(payload.Hits))
	for _, hit := range payload.Hits {
		results = append(results, mapProject(hit))
	}

	return &SearchResult{
		Projects:  results,
		TotalHits: payload.TotalHits,
		Limit:     payload.Limit,
		Offset:    payload.Offset,
	}, nil
}

// GetProject returns full project details including body and gallery.
func (c *Client) GetProject(ctx context.Context, projectID string) (*Project, error) {
	body, err := c.get(ctx, fmt.Sprintf("%s/project/%s", c.baseURL, projectID), "modrinth project")
	if err != nil {
		return nil, err
	}

	var payload projectDetailResponse
	if err := json.Unmarshal(body, &payload); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	project := mapProjectDetail(payload)
	return &project, nil
}

// GetProjectVersions returns versions for a project using optional filters.
func (c *Client) GetProjectVersions(ctx context.Context, projectID string, filter VersionFilter) ([]Version, error) {
	limit := filter.Limit
	if limit <= 0 || limit > maxLimit {
		limit = 25
	}

	values := url.Values{}
	values.Set("limit", fmt.Sprintf("%d", limit))

	if len(filter.Loaders) > 0 {
		if encoded, err := json.Marshal(filter.Loaders); err == nil {
			values.Set("loaders", string(encoded))
		}
	}
	if len(filter.GameVersions) > 0 {
		if encoded, err := json.Marshal(filter.GameVersions); err == nil {
			values.Set("game_versions", string(encoded))
		}
	}

	body, err := c.get(ctx, fmt.Sprintf("%s/project/%s/version?%s", c.baseURL, projectID, values.Encode()), "modrinth versions")
	if err != nil {
		return nil, err
	}

	var payload []versionResponse
	if err := json.Unmarshal(body, &payload); err != nil {
		return nil, fmt.Errorf("decode versions: %w", err)
	}

	versions := make([]Version, 0, len(payload))
	for _, item := range payload {
		versions = append(versions, mapVersion(item))
	}

	return versions, nil
}

// GetVersion fetches a single version by ID.
func (c *Client) GetVersion(ctx context.Context, versionID string) (*Version, error) {
	body, err := c.get(ctx, fmt.Sprintf("%s/version/%s", c.baseURL, versionID), "modrinth version")
	if err != nil {
		return nil, err
	}

	var payload versionResponse
	if err := json.Unmarshal(body, &payload); err != nil {
		return nil, fmt.Errorf("decode version: %w", err)
	}

	version := mapVersion(payload)
	return &version, nil
}

// GetVersionByFileHash returns the Modrinth version that owns a file hash.
func (c *Client) GetVersionByFileHash(ctx context.Context, fileHash, algorithm string) (*Version, error) {
	fileHash = strings.TrimSpace(fileHash)
	if fileHash == "" {
		return nil, fmt.Errorf("file hash is required")
	}
	algorithm = strings.TrimSpace(strings.ToLower(algorithm))
	if algorithm == "" {
		algorithm = "sha1"
	}

	values := url.Values{}
	values.Set("algorithm", algorithm)
	body, err := c.get(ctx, fmt.Sprintf("%s/version_file/%s?%s", c.baseURL, url.PathEscape(fileHash), values.Encode()), "modrinth version file lookup")
	if err != nil {
		var httpErr *httpError
		if errors.As(err, &httpErr) && httpErr.StatusCode == http.StatusNotFound {
			return nil, ErrNotFound
		}
		return nil, err
	}

	var payload versionResponse
	if err := json.Unmarshal(body, &payload); err != nil {
		return nil, fmt.Errorf("decode version file lookup: %w", err)
	}

	version := mapVersion(payload)
	return &version, nil
}

// --- internal helpers ---

type httpError struct {
	Operation  string
	StatusCode int
	Body       string
}

func (e *httpError) Error() string {
	return fmt.Sprintf("%s failed: status=%d body=%s", e.Operation, e.StatusCode, e.Body)
}

func (c *Client) get(ctx context.Context, endpoint, operation string) ([]byte, error) {
	var lastErr error
	for attempt := 0; attempt < maxAttempts; attempt++ {
		body, retryAfter, err := c.getOnce(ctx, endpoint, operation)
		if err == nil {
			return body, nil
		}
		lastErr = err

		var rateLimitErr *RateLimitError
		if !errors.As(err, &rateLimitErr) || attempt == maxAttempts-1 {
			return nil, err
		}
		wait := retryDelay(attempt, retryAfter)
		timer := time.NewTimer(wait)
		select {
		case <-ctx.Done():
			timer.Stop()
			return nil, ctx.Err()
		case <-timer.C:
		}
	}
	return nil, lastErr
}

func (c *Client) getOnce(ctx context.Context, endpoint, operation string) ([]byte, time.Duration, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, 0, err
	}
	req.Header.Set("User-Agent", c.userAgent)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, 0, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return nil, 0, fmt.Errorf("read response: %w", err)
		}
		return body, 0, nil
	}

	bodyBytes, _ := io.ReadAll(io.LimitReader(resp.Body, 8<<10))
	body := strings.TrimSpace(string(bodyBytes))
	retryAfter := parseRetryAfter(resp.Header.Get("Retry-After"))
	if resp.StatusCode == http.StatusTooManyRequests {
		return nil, retryAfter, &RateLimitError{
			Operation:  operation,
			StatusCode: resp.StatusCode,
			RetryAfter: retryAfter,
			Body:       body,
		}
	}
	return nil, 0, &httpError{
		Operation:  operation,
		StatusCode: resp.StatusCode,
		Body:       body,
	}
}

func retryDelay(attempt int, retryAfter time.Duration) time.Duration {
	if retryAfter > 0 {
		if retryAfter > 5*time.Second {
			return 5 * time.Second
		}
		return retryAfter
	}
	return time.Duration(attempt+1) * 500 * time.Millisecond
}

func parseRetryAfter(value string) time.Duration {
	value = strings.TrimSpace(value)
	if value == "" {
		return 0
	}
	if seconds, err := strconv.Atoi(value); err == nil {
		if seconds < 0 {
			return 0
		}
		return time.Duration(seconds) * time.Second
	}
	if retryAt, err := http.ParseTime(value); err == nil {
		if wait := time.Until(retryAt); wait > 0 {
			return wait
		}
	}
	return 0
}

type searchResponse struct {
	Hits      []projectHit `json:"hits"`
	Limit     int          `json:"limit"`
	Offset    int          `json:"offset"`
	TotalHits int          `json:"total_hits"`
}

type projectHit struct {
	ProjectID       string   `json:"project_id"`
	Slug            string   `json:"slug"`
	Title           string   `json:"title"`
	Description     string   `json:"description"`
	ProjectType     string   `json:"project_type"`
	IconURL         string   `json:"icon_url"`
	Categories      []string `json:"categories"`
	Versions        []string `json:"versions"`
	Downloads       int64    `json:"downloads"`
	Author          string   `json:"author"`
	LatestVersion   string   `json:"latest_version"`
	Published       string   `json:"date_published"`
	Modified        string   `json:"date_modified"`
	ClientSide      string   `json:"client_side"`
	ServerSide      string   `json:"server_side"`
	Followers       int64    `json:"follows"`
	DisplayLicense  string   `json:"license"`
	FeaturedGallery string   `json:"featured_gallery"`
}

type versionResponse struct {
	ID            string        `json:"id"`
	ProjectID     string        `json:"project_id"`
	Name          string        `json:"name"`
	VersionNumber string        `json:"version_number"`
	VersionType   string        `json:"version_type"`
	GameVersions  []string      `json:"game_versions"`
	Loaders       []string      `json:"loaders"`
	ServerSide    string        `json:"server_side"`
	ClientSide    string        `json:"client_side"`
	Changelog     string        `json:"changelog"`
	DatePublished string        `json:"date_published"`
	Files         []VersionFile `json:"files"`
}

type projectDetailResponse struct {
	ID            string          `json:"id"`
	Slug          string          `json:"slug"`
	Title         string          `json:"title"`
	Description   string          `json:"description"`
	Body          string          `json:"body"`
	ProjectType   string          `json:"project_type"`
	IconURL       string          `json:"icon_url"`
	Categories    []string        `json:"categories"`
	Loaders       []string        `json:"loaders"`
	GameVersions  []string        `json:"game_versions"`
	Team          json.RawMessage `json:"team"` // Can be array or string, handle dynamically
	Downloads     int64           `json:"downloads"`
	Rating        float64         `json:"rating"`
	LatestVersion string          `json:"latest_version"`
	ProjectURL    string          `json:"project_url"`
	SourceURL     string          `json:"source_url"`
	IssuesURL     string          `json:"issues_url"`
	Gallery       json.RawMessage `json:"gallery"` // Can be array of strings or objects, handle dynamically
}

type galleryItem struct {
	URL string `json:"url"`
}

type teamMember struct {
	User struct {
		Username string `json:"username"`
	} `json:"user"`
}

func buildFacets(params SearchParams) [][]string {
	facets := make([][]string, 0)

	if params.ProjectType != "" {
		facets = append(facets, []string{fmt.Sprintf("project_type:%s", params.ProjectType)})
	}

	for _, loader := range params.Loaders {
		if loader == "" {
			continue
		}
		facets = append(facets, []string{fmt.Sprintf("categories:%s", strings.ToLower(loader))})
	}

	for _, cat := range params.Categories {
		if cat == "" {
			continue
		}
		facets = append(facets, []string{fmt.Sprintf("categories:%s", strings.ToLower(cat))})
	}

	for _, version := range params.GameVersions {
		if version == "" {
			continue
		}
		facets = append(facets, []string{fmt.Sprintf("versions:%s", normalizeVersion(version))})
	}

	// Prefer server-side compatible results (OR condition: either required OR optional)
	facets = append(facets, []string{"server_side:required", "server_side:optional"})

	return facets
}

func mapProject(hit projectHit) Project {
	loaders := extractLoaders(hit.Categories)
	projectURL := ""
	if hit.Slug != "" {
		projectURL = fmt.Sprintf("https://modrinth.com/project/%s", hit.Slug)
	}

	return Project{
		ID:            hit.ProjectID,
		Slug:          hit.Slug,
		Title:         hit.Title,
		Description:   hit.Description,
		ProjectType:   hit.ProjectType,
		IconURL:       hit.IconURL,
		Categories:    hit.Categories,
		Loaders:       loaders,
		GameVersions:  hit.Versions,
		Authors:       []string{hit.Author},
		Downloads:     hit.Downloads,
		Rating:        0, // Search payload does not expose rating; keep 0 for now
		LatestVersion: hit.LatestVersion,
		ProjectURL:    projectURL,
	}
}

func mapProjectDetail(resp projectDetailResponse) Project {
	loaders := extractLoaders(resp.Categories)
	projectURL := resp.ProjectURL
	if projectURL == "" && resp.Slug != "" {
		projectURL = fmt.Sprintf("https://modrinth.com/project/%s", resp.Slug)
	}

	// Extract author names from team field (can be array or string)
	authors := make([]string, 0)
	if len(resp.Team) > 0 {
		// Try to unmarshal as array first
		var teamMembers []teamMember
		if err := json.Unmarshal(resp.Team, &teamMembers); err == nil {
			for _, member := range teamMembers {
				if member.User.Username != "" {
					authors = append(authors, member.User.Username)
				}
			}
		} else {
			// If array fails, try as string (JSON string)
			var teamStr string
			if err := json.Unmarshal(resp.Team, &teamStr); err == nil && teamStr != "" {
				// Try to unmarshal the string as JSON array
				if err := json.Unmarshal([]byte(teamStr), &teamMembers); err == nil {
					for _, member := range teamMembers {
						if member.User.Username != "" {
							authors = append(authors, member.User.Username)
						}
					}
				}
			}
		}
	}

	// Extract gallery URLs (can be array of strings or array of objects)
	gallery := make([]string, 0)
	if len(resp.Gallery) > 0 {
		// Try to unmarshal as array of strings first
		var galleryStrings []string
		if err := json.Unmarshal(resp.Gallery, &galleryStrings); err == nil {
			gallery = galleryStrings
		} else {
			// If that fails, try as array of objects with url field
			var galleryItems []galleryItem
			if err := json.Unmarshal(resp.Gallery, &galleryItems); err == nil {
				for _, item := range galleryItems {
					if item.URL != "" {
						gallery = append(gallery, item.URL)
					}
				}
			}
		}
	}

	return Project{
		ID:            resp.ID,
		Slug:          resp.Slug,
		Title:         resp.Title,
		Description:   resp.Description,
		Body:          resp.Body,
		ProjectType:   resp.ProjectType,
		IconURL:       resp.IconURL,
		Categories:    resp.Categories,
		Loaders:       loaders,
		GameVersions:  resp.GameVersions,
		Authors:       authors,
		Downloads:     resp.Downloads,
		Rating:        resp.Rating,
		LatestVersion: resp.LatestVersion,
		ProjectURL:    projectURL,
		SourceURL:     resp.SourceURL,
		IssuesURL:     resp.IssuesURL,
		Gallery:       gallery,
	}
}

func mapVersion(resp versionResponse) Version {
	published, _ := time.Parse(time.RFC3339, resp.DatePublished)

	return Version{
		ID:            resp.ID,
		ProjectID:     resp.ProjectID,
		Name:          resp.Name,
		VersionNumber: resp.VersionNumber,
		VersionType:   resp.VersionType,
		GameVersions:  resp.GameVersions,
		Loaders:       resp.Loaders,
		ServerSide:    resp.ServerSide,
		ClientSide:    resp.ClientSide,
		Changelog:     resp.Changelog,
		DatePublished: published,
		Files:         resp.Files,
	}
}

var loaderTags = map[string]struct{}{
	"fabric":    {},
	"forge":     {},
	"neoforge":  {},
	"quilt":     {},
	"paper":     {},
	"purpur":    {},
	"spigot":    {},
	"bukkit":    {},
	"folia":     {},
	"velocity":  {},
	"waterfall": {},
	"magma":     {},
	"catserver": {},
	"arclight":  {},
}

func extractLoaders(categories []string) []string {
	set := make(map[string]struct{})
	for _, cat := range categories {
		c := strings.ToLower(cat)
		if _, ok := loaderTags[c]; ok {
			set[c] = struct{}{}
		}
	}
	loaders := make([]string, 0, len(set))
	for key := range set {
		loaders = append(loaders, key)
	}
	return loaders
}

func normalizeVersion(version string) string {
	version = strings.TrimSpace(version)
	version = strings.TrimPrefix(version, "v")
	return version
}
