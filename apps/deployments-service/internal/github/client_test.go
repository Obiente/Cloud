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

func TestCheckRunActionRequiredIsCompleted(t *testing.T) {
	body := checkRunBody(CheckRunUpdate{Status: "completed", Conclusion: "action_required"}, true)
	if body["status"] != "completed" || body["conclusion"] != "action_required" {
		t.Fatalf("check run body = %#v", body)
	}
}
