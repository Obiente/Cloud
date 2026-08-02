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
