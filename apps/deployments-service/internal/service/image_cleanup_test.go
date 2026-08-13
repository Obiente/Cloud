package deployments

import (
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
