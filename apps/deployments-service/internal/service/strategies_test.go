package deployments

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	deploymentsv1 "github.com/obiente/cloud/apps/shared/proto/obiente/cloud/deployments/v1"
)

func TestCleanRelativeRepoPath(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    string
		wantErr bool
	}{
		{name: "empty defaults to repo root", input: "", want: "."},
		{name: "dot defaults to repo root", input: ".", want: "."},
		{name: "trims and cleans subdir", input: " apps/web/../api ", want: "apps/api"},
		{name: "rejects absolute path", input: "/etc/passwd", wantErr: true},
		{name: "rejects parent traversal", input: "../secret", wantErr: true},
		{name: "rejects cleaned parent traversal", input: "apps/../../secret", wantErr: true},
		{name: "rejects null byte", input: "apps/api\x00Dockerfile", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := cleanRelativeRepoPath(tt.input)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("cleanRelativeRepoPath(%q) returned nil error", tt.input)
				}
				return
			}
			if err != nil {
				t.Fatalf("cleanRelativeRepoPath(%q) returned error: %v", tt.input, err)
			}
			if got != tt.want {
				t.Fatalf("cleanRelativeRepoPath(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestBuildStrategyAutoDetectMatrix(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name     string
		files    map[string]string
		want     deploymentsv1.BuildStrategy
		wantType deploymentsv1.DeploymentType
	}{
		{
			name: "compose repo wins over dockerfile",
			files: map[string]string{
				"compose.yml": `services:
  web:
    image: nginx:latest
`,
				"Dockerfile": "FROM nginx:alpine\n",
			},
			want:     deploymentsv1.BuildStrategy_COMPOSE_REPO,
			wantType: deploymentsv1.DeploymentType_DOCKER,
		},
		{
			name: "dockerfile wins over language indicators",
			files: map[string]string{
				"Dockerfile":   "FROM node:22-alpine\n",
				"package.json": `{"scripts":{"start":"node server.js"}}`,
			},
			want:     deploymentsv1.BuildStrategy_DOCKERFILE,
			wantType: deploymentsv1.DeploymentType_DOCKER,
		},
		{
			name: "plain static html is static",
			files: map[string]string{
				"index.html": "<h1>hello</h1>\n",
			},
			want:     deploymentsv1.BuildStrategy_STATIC_SITE,
			wantType: deploymentsv1.DeploymentType_STATIC,
		},
		{
			name: "node app uses railpack",
			files: map[string]string{
				"package.json": `{"scripts":{"start":"node server.js"}}`,
				"server.js":    "console.log('ok')\n",
			},
			want:     deploymentsv1.BuildStrategy_RAILPACK,
			wantType: deploymentsv1.DeploymentType_NODE,
		},
		{
			name: "go app uses railpack",
			files: map[string]string{
				"go.mod":  "module example.com/app\n\ngo 1.25\n",
				"main.go": "package main\nfunc main() {}\n",
			},
			want:     deploymentsv1.BuildStrategy_RAILPACK,
			wantType: deploymentsv1.DeploymentType_GO,
		},
		{
			name: "python app uses railpack",
			files: map[string]string{
				"requirements.txt": "fastapi\nuvicorn\n",
				"main.py":          "print('ok')\n",
			},
			want:     deploymentsv1.BuildStrategy_RAILPACK,
			wantType: deploymentsv1.DeploymentType_PYTHON,
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			repoPath := t.TempDir()
			for name, contents := range tc.files {
				writeFixtureFile(t, repoPath, name, contents)
			}

			registry := NewBuildStrategyRegistry()
			got, err := registry.AutoDetect(context.Background(), repoPath)
			if err != nil {
				t.Fatalf("AutoDetect returned error: %v", err)
			}
			if got != tc.want {
				t.Fatalf("AutoDetect() = %v, want %v", got, tc.want)
			}

			gotType := registry.InferDeploymentType(context.Background(), got, repoPath)
			if gotType != tc.wantType {
				t.Fatalf("InferDeploymentType(%v) = %v, want %v", got, gotType, tc.wantType)
			}
		})
	}
}

func writeFixtureFile(t *testing.T, root, name, contents string) {
	t.Helper()

	path := filepath.Join(root, name)
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatalf("failed to create fixture directory for %s: %v", name, err)
	}
	if err := os.WriteFile(path, []byte(contents), 0644); err != nil {
		t.Fatalf("failed to write fixture file %s: %v", name, err)
	}
}
