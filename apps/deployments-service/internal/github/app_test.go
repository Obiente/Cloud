package github

import (
	"strings"
	"testing"
)

func TestReadGitHubAppResponseBodyRejectsOversizedResponse(t *testing.T) {
	_, err := readGitHubAppResponseBody(strings.NewReader(strings.Repeat("a", githubAppResponseBodyLimit+1)))
	if err == nil {
		t.Fatal("expected oversized response to be rejected")
	}
}
