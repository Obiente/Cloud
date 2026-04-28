package deployments

import "testing"

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
