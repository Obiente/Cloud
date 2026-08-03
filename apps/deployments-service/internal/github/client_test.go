package github

import (
	"encoding/base64"
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
