package deployments

import (
	"regexp"
	"strings"
	"testing"
)

func TestDockerImageTagPreservesValidTags(t *testing.T) {
	for _, tag := range []string{"main", "release-1.2.3", "build_42"} {
		if got := dockerImageTag(tag); got != tag {
			t.Fatalf("valid tag %q changed to %q", tag, got)
		}
	}
}

func TestDockerImageTagSanitizesGitRefsDeterministically(t *testing.T) {
	valid := regexp.MustCompile(`^[A-Za-z0-9_][A-Za-z0-9_.-]{0,127}$`)
	branch := "feat/selective-virtual-folders"
	got := dockerImageTag(branch)
	if !valid.MatchString(got) || got != dockerImageTag(branch) || !strings.HasPrefix(got, "feat-selective-virtual-folders-") {
		t.Fatalf("unsafe or unstable Docker tag for %q: %q", branch, got)
	}
	if got == dockerImageTag("feat-selective-virtual-folders") {
		t.Fatal("distinct Git refs collapsed to the same Docker tag")
	}
	long := dockerImageTag(strings.Repeat("feature/", 40))
	if !valid.MatchString(long) || len(long) > 128 {
		t.Fatalf("long ref produced invalid Docker tag %q", long)
	}
}
