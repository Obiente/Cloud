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

func TestDockerBuildImageTagIncludesExactRevision(t *testing.T) {
	first := strings.Repeat("a", 40)
	second := strings.Repeat("b", 40)
	firstTag := dockerBuildImageTag("feat/preview", first)
	secondTag := dockerBuildImageTag("feat/preview", second)
	valid := regexp.MustCompile(`^[A-Za-z0-9_][A-Za-z0-9_.-]{0,127}$`)
	if firstTag == secondTag {
		t.Fatal("different revisions reused the same Docker tag")
	}
	if !strings.HasSuffix(firstTag, "-"+first) || !valid.MatchString(firstTag) {
		t.Fatalf("revision tag is invalid or does not retain the commit: %q", firstTag)
	}
	longTag := dockerBuildImageTag(strings.Repeat("feature/", 40), strings.ToUpper(first))
	if len(longTag) > 128 || !strings.HasSuffix(longTag, "-"+first) || !valid.MatchString(longTag) {
		t.Fatalf("long revision tag is invalid: %q", longTag)
	}
}

func TestDockerBuildImageTagKeepsManualBuildTagsStable(t *testing.T) {
	want := dockerImageTag("main")
	for _, commit := range []string{"", "not-a-commit", strings.Repeat("a", 39)} {
		if got := dockerBuildImageTag("main", commit); got != want {
			t.Fatalf("manual build tag changed for commit %q: got %q, want %q", commit, got, want)
		}
	}
}
