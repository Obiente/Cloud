package deployments

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/obiente/cloud/apps/shared/pkg/database"
)

func TestRetainedRevisionImagesKeepsCurrentAndRollback(t *testing.T) {
	current := "registry.example/obiente/deploy-1:main-" + strings.Repeat("c", 40)
	previous := "registry.example/obiente/deploy-1:main-" + strings.Repeat("b", 40)
	old := "registry.example/obiente/deploy-1:main-" + strings.Repeat("a", 40)
	failed := "registry.example/obiente/deploy-1:main-" + strings.Repeat("d", 64)
	active := "registry.example/obiente/deploy-1:main-" + strings.Repeat("e", 40)
	runtimeFailed := "registry.example/obiente/deploy-1:main-" + strings.Repeat("f", 40)
	manual := "registry.example/obiente/deploy-1:main"
	runtimeError := "container startup failed"
	builds := []*database.BuildHistory{
		{Status: 2, ImageName: &active},
		{Status: 4, ImageName: &failed},
		{Status: 3, ImageName: &runtimeFailed, Error: &runtimeError},
		{Status: 3, ImageName: &current},
		{Status: 3, ImageName: &previous},
		{Status: 3, ImageName: &old},
		{Status: 3, ImageName: &manual},
	}

	kept, obsolete := retainedRevisionImages(builds, current)
	if _, ok := kept[current]; !ok {
		t.Fatal("current image was not retained")
	}
	if _, ok := kept[previous]; !ok {
		t.Fatal("rollback image was not retained")
	}
	if _, ok := kept[active]; !ok {
		t.Fatal("active build image was not retained")
	}
	if _, ok := kept[manual]; !ok {
		t.Fatal("stable mutable tag was not retained")
	}
	wantObsolete := map[string]bool{old: true, failed: true, runtimeFailed: true}
	if len(obsolete) != len(wantObsolete) {
		t.Fatalf("obsolete images = %v, want %v", obsolete, wantObsolete)
	}
	for _, image := range obsolete {
		if !wantObsolete[image] {
			t.Fatalf("unexpected obsolete image %q", image)
		}
	}
}

func TestSplitRegistryImage(t *testing.T) {
	image := "registry.example:5000/obiente/deploy-1:feature-" + strings.Repeat("a", 64)
	repository, tag, ok := splitRegistryImage("https://registry.example:5000", image)
	if !ok || repository != "obiente/deploy-1" || tag != strings.TrimPrefix(image, "registry.example:5000/obiente/deploy-1:") {
		t.Fatalf("unexpected registry image split: repository=%q tag=%q ok=%v", repository, tag, ok)
	}
}

func TestCleanupCallerMustOwnLiveImage(t *testing.T) {
	revisionA := "registry.example/obiente/deploy-1:main-" + strings.Repeat("a", 40)
	revisionB := "registry.example/obiente/deploy-1:main-" + strings.Repeat("b", 40)
	revisionC := "registry.example/obiente/deploy-1:main-" + strings.Repeat("c", 40)

	if cleanupCallerOwnsLiveImage(revisionA, &revisionC) {
		t.Fatal("delayed cleanup for revision A accepted newer live revision C")
	}
	if !cleanupCallerOwnsLiveImage("  "+revisionC+"  ", &revisionC) {
		t.Fatal("cleanup for the live revision was rejected")
	}
	if cleanupCallerOwnsLiveImage(revisionC, nil) {
		t.Fatal("cleanup without a live deployment image was accepted")
	}

	builds := []*database.BuildHistory{
		{Status: 3, ImageName: &revisionC},
		{Status: 3, ImageName: &revisionB},
		{Status: 3, ImageName: &revisionA},
	}
	kept, obsolete := retainedRevisionImages(builds, revisionC)
	if _, ok := kept[revisionB]; !ok {
		t.Fatal("live revision C did not retain revision B for rollback")
	}
	if len(obsolete) != 1 || obsolete[0] != revisionA {
		t.Fatalf("obsolete images = %v, want only delayed caller revision A", obsolete)
	}
}

func TestActiveRevisionImagesProtectsCommitBeforeImageExists(t *testing.T) {
	commitSHA := strings.Repeat("a", 40)
	builds := []*database.BuildHistory{
		{
			Status:       2,
			DeploymentID: "deploy-1",
			Branch:       "feature/reused",
			CommitSHA:    &commitSHA,
			ImageName:    nil,
		},
	}
	protected := activeRevisionImages(builds, "deploy-1", "https://registry.example:5000")
	tag := dockerBuildImageTag("feature/reused", commitSHA)
	for _, image := range []string{
		"obiente/deploy-1:" + tag,
		"registry.example:5000/obiente/deploy-1:" + tag,
	} {
		if _, ok := protected[image]; !ok {
			t.Fatalf("active revision image %q was not protected", image)
		}
	}
}

func TestProtectRegistryImageDigestsIncludesNewActiveRevision(t *testing.T) {
	const digest = "sha256:output-identical"
	server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		if request.Method != http.MethodHead {
			t.Fatalf("method = %s, want HEAD", request.Method)
		}
		response.Header().Set("Docker-Content-Digest", digest)
		response.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	registryHost := strings.TrimPrefix(server.URL, "http://")
	activeImage := registryHost + "/obiente/deploy-1:main-" + strings.Repeat("b", 40)
	protectedDigests := make(map[string]map[string]struct{})
	unsafeRepositories := make(map[string]struct{})
	protectRegistryImageDigests(
		context.Background(),
		server.Client(),
		server.URL,
		map[string]struct{}{activeImage: {}},
		protectedDigests,
		unsafeRepositories,
	)

	if _, protected := protectedDigests["obiente/deploy-1"][digest]; !protected {
		t.Fatalf("newly active manifest digest %q was not protected", digest)
	}
	if len(unsafeRepositories) != 0 {
		t.Fatalf("unexpected unsafe repositories: %v", unsafeRepositories)
	}
}
