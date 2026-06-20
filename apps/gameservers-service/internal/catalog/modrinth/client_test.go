package modrinth

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestGetVersionByFileHash(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/version_file/abc123" {
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
		if got := r.URL.Query().Get("algorithm"); got != "sha1" {
			t.Fatalf("expected sha1 algorithm, got %q", got)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"id":"version-id",
			"project_id":"project-id",
			"name":"Release",
			"version_number":"1.0.0",
			"game_versions":["1.20.1"],
			"loaders":["paper"],
			"server_side":"required",
			"client_side":"unsupported",
			"date_published":"2024-01-02T03:04:05Z",
			"files":[{"hashes":{"sha1":"abc123"},"primary":true,"filename":"Plugin.jar","url":"https://example.test/plugin.jar","size":123}]
		}`))
	}))
	defer server.Close()

	client := NewClient(server.Client())
	client.baseURL = server.URL

	version, err := client.GetVersionByFileHash(context.Background(), "abc123", "sha1")
	if err != nil {
		t.Fatalf("GetVersionByFileHash returned error: %v", err)
	}
	if version.ID != "version-id" || version.ProjectID != "project-id" {
		t.Fatalf("unexpected version: %#v", version)
	}
	if len(version.Files) != 1 || version.Files[0].Filename != "Plugin.jar" {
		t.Fatalf("unexpected files: %#v", version.Files)
	}
}

func TestGetVersionByFileHashNotFound(t *testing.T) {
	server := httptest.NewServer(http.NotFoundHandler())
	defer server.Close()

	client := NewClient(server.Client())
	client.baseURL = server.URL

	_, err := client.GetVersionByFileHash(context.Background(), "missing", "sha1")
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}
