package github

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestGetFileEscapesPathAndBranch(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		if req.URL.EscapedPath() != "/repos/obiente/cloud/contents/docs/release%20notes.md" {
			t.Errorf("escaped path = %q", req.URL.EscapedPath())
		}
		if req.URL.Query().Get("ref") != "feature/a&b" {
			t.Errorf("ref = %q", req.URL.Query().Get("ref"))
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = fmt.Fprintf(w, `{"content":%q,"encoding":"base64","size":5}`, base64.StdEncoding.EncodeToString([]byte("hello")))
	}))
	defer server.Close()

	client := NewClient("token")
	client.baseURL = server.URL
	client.httpClient = server.Client()
	file, err := client.GetFile(t.Context(), "obiente/cloud", "feature/a&b", "docs/release notes.md")
	if err != nil {
		t.Fatalf("get file: %v", err)
	}
	if file.Content != "hello" || file.Encoding != "text" {
		t.Fatalf("file = %+v", file)
	}
}

func TestGetFilePreservesWhitespaceInPath(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		if req.URL.EscapedPath() != "/repos/obiente/cloud/contents/%20docs/release%20notes.md%20" {
			t.Errorf("escaped path = %q", req.URL.EscapedPath())
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = fmt.Fprintf(w, `{"content":%q,"encoding":"base64","size":5}`, base64.StdEncoding.EncodeToString([]byte("hello")))
	}))
	defer server.Close()

	client := NewClient("token")
	client.baseURL = server.URL
	client.httpClient = server.Client()
	if _, err := client.GetFile(t.Context(), "obiente/cloud", "main", " docs/release notes.md "); err != nil {
		t.Fatalf("get file with surrounding whitespace: %v", err)
	}
}

func TestGetFileRejectsTraversalPath(t *testing.T) {
	client := NewClient("token")
	if _, err := client.GetFile(t.Context(), "obiente/cloud", "main", "../secret"); err == nil {
		t.Fatal("expected traversal path to be rejected")
	}
	if _, err := client.ListBranches(t.Context(), "../repos"); err == nil {
		t.Fatal("expected traversal repository name to be rejected")
	}
}

func TestListBranchesFollowsPagination(t *testing.T) {
	var server *httptest.Server
	server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		if req.URL.EscapedPath() != "/repos/Obiente/nc-native/branches" {
			t.Errorf("escaped path = %q", req.URL.EscapedPath())
		}
		if req.URL.Query().Get("per_page") != "100" {
			t.Errorf("per_page = %q", req.URL.Query().Get("per_page"))
		}
		w.Header().Set("Content-Type", "application/json")
		if req.URL.Query().Get("page") == "2" {
			_, _ = fmt.Fprint(w, `[{"name":"main","commit":{"sha":"main-sha"}}]`)
			return
		}
		w.Header().Set("Link", fmt.Sprintf(`<%s/repos/Obiente/nc-native/branches?page=2&per_page=100>; rel="next"`, server.URL))
		_, _ = fmt.Fprint(w, `[{"name":"feat/example","commit":{"sha":"feature-sha"}}]`)
	}))
	defer server.Close()

	client := NewClient("token")
	client.baseURL = server.URL
	client.httpClient = server.Client()
	branches, err := client.ListBranches(t.Context(), "Obiente/nc-native")
	if err != nil {
		t.Fatalf("list branches: %v", err)
	}
	if len(branches) != 2 || branches[0].Name != "feat/example" || branches[1].Name != "main" {
		t.Fatalf("branches = %+v", branches)
	}
}

func TestListBranchesRejectsCrossOriginPagination(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Link", `<https://example.invalid/branches?page=2>; rel="next"`)
		_, _ = fmt.Fprint(w, `[]`)
	}))
	defer server.Close()

	client := NewClient("token")
	client.baseURL = server.URL
	client.httpClient = server.Client()
	if _, err := client.ListBranches(t.Context(), "Obiente/nc-native"); err == nil || !strings.Contains(err.Error(), "unexpected host") {
		t.Fatalf("expected cross-origin pagination error, got %v", err)
	}
}

func TestGetRepositoryReturnsDefaultBranch(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		if req.URL.EscapedPath() != "/repos/Obiente/nc-native" {
			t.Errorf("escaped path = %q", req.URL.EscapedPath())
		}
		_, _ = fmt.Fprint(w, `{"id":1,"name":"nc-native","full_name":"Obiente/nc-native","default_branch":"main"}`)
	}))
	defer server.Close()

	client := NewClient("token")
	client.baseURL = server.URL
	client.httpClient = server.Client()
	repository, err := client.GetRepository(t.Context(), "Obiente/nc-native")
	if err != nil {
		t.Fatalf("get repository: %v", err)
	}
	if repository.DefaultBranch != "main" {
		t.Fatalf("default branch = %q", repository.DefaultBranch)
	}
}

func TestReadGitHubAPIResponseBodyRejectsOversizedResponse(t *testing.T) {
	_, err := readGitHubAPIResponseBody(strings.NewReader(strings.Repeat("a", githubAPIResponseBodyLimit+1)))
	if err == nil {
		t.Fatal("expected oversized response to be rejected")
	}
}

func TestCreateDeploymentUsesTransientNonProductionEnvironment(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		if req.Method != http.MethodPost || req.URL.Path != "/repos/obiente/cloud/deployments" {
			t.Fatalf("unexpected request %s %s", req.Method, req.URL.Path)
		}
		var body map[string]interface{}
		if err := json.NewDecoder(req.Body).Decode(&body); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		if body["auto_merge"] != false || body["transient_environment"] != true || body["production_environment"] != false {
			t.Fatalf("unsafe deployment options: %#v", body)
		}
		contexts, ok := body["required_contexts"].([]interface{})
		if !ok || len(contexts) != 0 {
			t.Fatalf("required contexts = %#v", body["required_contexts"])
		}
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(`{"id":42}`))
	}))
	defer server.Close()
	client := NewClient("token")
	client.baseURL, client.httpClient = server.URL, server.Client()
	id, err := client.CreateDeployment(t.Context(), "obiente/cloud", strings.Repeat("a", 40), "Obiente Preview / PR #1", "preview")
	if err != nil || id != 42 {
		t.Fatalf("create deployment: id=%d err=%v", id, err)
	}
}

func TestGetPullRequestReturnsAuthoritativeState(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		if req.Method != http.MethodGet || req.URL.Path != "/repos/obiente/cloud/pulls/31" {
			t.Fatalf("unexpected request %s %s", req.Method, req.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"state":"open","draft":false,"merged":false,"head":{"sha":"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa","ref":"feature","repo":{"full_name":"obiente/cloud"}},"base":{"ref":"main","repo":{"full_name":"obiente/cloud"}}}`))
	}))
	defer server.Close()
	client := NewClient("token")
	client.baseURL, client.httpClient = server.URL, server.Client()
	pullRequest, err := client.GetPullRequest(t.Context(), "obiente/cloud", 31)
	if err != nil {
		t.Fatalf("get pull request: %v", err)
	}
	if pullRequest.State != "open" || pullRequest.Head.Ref != "feature" || pullRequest.Base.Ref != "main" {
		t.Fatalf("unexpected pull request: %#v", pullRequest)
	}
}

func TestListOpenPullRequestsPaginatesExistingPullRequests(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		if req.URL.Path != "/repos/obiente/cloud/pulls" || req.URL.Query().Get("state") != "open" {
			t.Fatalf("unexpected request %s", req.URL.String())
		}
		w.Header().Set("Content-Type", "application/json")
		if req.URL.Query().Get("page") == "1" {
			pullRequests := make([]map[string]interface{}, 100)
			for i := range pullRequests {
				pullRequests[i] = map[string]interface{}{"number": i + 1, "state": "open"}
			}
			_ = json.NewEncoder(w).Encode(pullRequests)
			return
		}
		_, _ = w.Write([]byte(`[{"number":101,"state":"open"}]`))
	}))
	defer server.Close()
	client := NewClient("token")
	client.baseURL, client.httpClient = server.URL, server.Client()
	pullRequests, err := client.ListOpenPullRequests(t.Context(), "obiente/cloud")
	if err != nil || len(pullRequests) != 101 || pullRequests[100].Number != 101 {
		t.Fatalf("list open pull requests: count=%d err=%v", len(pullRequests), err)
	}
}

func TestListOpenPullRequestsRescansMutablePagination(t *testing.T) {
	scan := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		page := req.URL.Query().Get("page")
		if page == "1" {
			scan++
		}
		w.Header().Set("Content-Type", "application/json")
		if page == "1" {
			start := 1
			if scan > 1 {
				start = 2
			}
			pullRequests := make([]map[string]interface{}, 100)
			for i := range pullRequests {
				pullRequests[i] = map[string]interface{}{"number": start + i, "state": "open"}
			}
			_ = json.NewEncoder(w).Encode(pullRequests)
			return
		}
		_, _ = w.Write([]byte(`[{"number":102,"state":"open"}]`))
	}))
	defer server.Close()

	client := NewClient("token")
	client.baseURL, client.httpClient = server.URL, server.Client()
	pullRequests, err := client.ListOpenPullRequests(t.Context(), "obiente/cloud")
	if err != nil {
		t.Fatalf("list open pull requests: %v", err)
	}
	if scan != 3 {
		t.Fatalf("pagination scans = %d, want 3", scan)
	}
	found101 := false
	for i := range pullRequests {
		if pullRequests[i].Number == 101 {
			found101 = true
		}
		if pullRequests[i].Number == 1 {
			t.Fatal("closed pull request from the unstable first scan was retained")
		}
	}
	if !found101 {
		t.Fatal("pull request shifted across the first pagination boundary was skipped")
	}
}

func TestCheckRunActionRequiredIsCompleted(t *testing.T) {
	body := checkRunBody(CheckRunUpdate{Status: "completed", Conclusion: "action_required"}, true)
	if body["status"] != "completed" || body["conclusion"] != "action_required" {
		t.Fatalf("check run body = %#v", body)
	}
}

func TestCheckRunOmitsEmptyDetailsURL(t *testing.T) {
	body := checkRunBody(CheckRunUpdate{Status: "in_progress"}, true)
	if _, exists := body["details_url"]; exists {
		t.Fatalf("empty details URL should be omitted: %#v", body)
	}
}

func TestFindIssueCommentOnlyAdoptsCurrentGitHubAppComment(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		if req.Method != http.MethodGet || req.URL.Path != "/repos/obiente/cloud/issues/31/comments" {
			t.Fatalf("unexpected request %s %s", req.Method, req.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`[
			{"id":1,"body":"<!-- obiente-preview -->","performed_via_github_app":{"id":999}},
			{"id":2,"body":"<!-- obiente-preview -->","performed_via_github_app":{"id":42}}
		]`))
	}))
	defer server.Close()

	client := NewClient("token")
	client.baseURL, client.httpClient, client.appID = server.URL, server.Client(), 42
	comment, err := client.FindIssueComment(t.Context(), "obiente/cloud", 31, "<!-- obiente-preview -->")
	if err != nil {
		t.Fatalf("find issue comment: %v", err)
	}
	if comment == nil || comment.ID != 2 {
		t.Fatalf("adopted comment = %#v", comment)
	}
}

func TestFindIssueCommentDoesNotAdoptUnownedMarker(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`[{"id":1,"body":"<!-- obiente-preview -->"}]`))
	}))
	defer server.Close()

	client := NewClient("token")
	client.baseURL, client.httpClient, client.appID = server.URL, server.Client(), 42
	comment, err := client.FindIssueComment(t.Context(), "obiente/cloud", 31, "<!-- obiente-preview -->")
	if err != nil {
		t.Fatalf("find issue comment: %v", err)
	}
	if comment != nil {
		t.Fatalf("adopted unowned comment = %#v", comment)
	}
}

func TestDeleteIssueCommentAcceptsDeletedOrAlreadyMissing(t *testing.T) {
	for _, status := range []int{http.StatusNoContent, http.StatusNotFound} {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			if req.Method != http.MethodDelete || req.URL.Path != "/repos/obiente/cloud/issues/comments/42" {
				t.Fatalf("unexpected request %s %s", req.Method, req.URL.Path)
			}
			w.WriteHeader(status)
		}))
		client := NewClient("token")
		client.baseURL, client.httpClient = server.URL, server.Client()
		if err := client.DeleteIssueComment(t.Context(), "obiente/cloud", 42); err != nil {
			server.Close()
			t.Fatalf("delete issue comment with status %d: %v", status, err)
		}
		server.Close()
	}
}
