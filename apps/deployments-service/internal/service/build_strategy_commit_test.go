package deployments

import (
	"context"
	"encoding/base64"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestGitHubCloneAuthenticationIsEphemeral(t *testing.T) {
	token := "installation-secret"
	cmd := gitCommandWithAuthentication(context.Background(), "", "https://github.com/Obiente/Cloud.git", token, "fetch", "origin")
	if strings.Contains(strings.Join(cmd.Args, " "), token) {
		t.Fatal("git command arguments exposed the installation token")
	}
	wantHeader := "GIT_CONFIG_VALUE_0=Authorization: Basic " + base64.StdEncoding.EncodeToString([]byte("x-access-token:"+token))
	foundHeader := false
	for _, value := range cmd.Env {
		if value == wantHeader {
			foundHeader = true
		}
	}
	if !foundHeader {
		t.Fatal("git command did not receive the ephemeral authorization header")
	}
}

func TestCloneRepositoryChecksOutRequestedCommit(t *testing.T) {
	sourceDir := t.TempDir()
	runGitForTest(t, sourceDir, "init", "--initial-branch", "main")
	runGitForTest(t, sourceDir, "config", "user.name", "Obiente Test")
	runGitForTest(t, sourceDir, "config", "user.email", "test@obiente.org")

	trackedFile := filepath.Join(sourceDir, "version.txt")
	if err := os.WriteFile(trackedFile, []byte("first\n"), 0o600); err != nil {
		t.Fatalf("write first revision: %v", err)
	}
	runGitForTest(t, sourceDir, "add", "version.txt")
	runGitForTest(t, sourceDir, "commit", "-m", "first")
	firstCommit := gitOutputForTest(t, sourceDir, "rev-parse", "HEAD")

	if err := os.WriteFile(trackedFile, []byte("second\n"), 0o600); err != nil {
		t.Fatalf("write second revision: %v", err)
	}
	runGitForTest(t, sourceDir, "commit", "-am", "second")

	destination := filepath.Join(t.TempDir(), "checkout")
	if err := cloneRepository(context.Background(), sourceDir, "main", firstCommit, destination, ""); err != nil {
		t.Fatalf("clone requested commit: %v", err)
	}

	content, err := os.ReadFile(filepath.Join(destination, "version.txt"))
	if err != nil {
		t.Fatalf("read cloned revision: %v", err)
	}
	if got := string(content); got != "first\n" {
		t.Fatalf("cloned content = %q, want first revision", got)
	}
	if got := gitOutputForTest(t, destination, "rev-parse", "HEAD"); got != firstCommit {
		t.Fatalf("cloned HEAD = %s, want %s", got, firstCommit)
	}
}

func runGitForTest(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git %s: %v\n%s", strings.Join(args, " "), err, output)
	}
}

func gitOutputForTest(t *testing.T, dir string, args ...string) string {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %s: %v\n%s", strings.Join(args, " "), err, output)
	}
	return strings.TrimSpace(string(output))
}
