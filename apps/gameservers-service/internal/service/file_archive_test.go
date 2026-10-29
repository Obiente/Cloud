package gameservers

import (
	"context"
	"errors"
	"testing"

	"github.com/obiente/cloud/apps/shared/pkg/docker"
)

type failingArchiveReader struct{ nested, fail bool }

var errArchiveFixture = errors.New("file became unreadable")

func (f failingArchiveReader) ContainerListFiles(_ context.Context, _, directory string) ([]docker.FileInfo, error) {
	if directory == "/root/nested" {
		if f.fail {
			return nil, errArchiveFixture
		}
		return []docker.FileInfo{{Path: "/root/nested/record"}}, nil
	}
	if f.nested {
		return []docker.FileInfo{{Path: "/root/nested", IsDirectory: true}}, nil
	}
	return []docker.FileInfo{{Path: "/root/record"}}, nil
}
func (f failingArchiveReader) ContainerReadFile(context.Context, string, string) ([]byte, error) {
	if f.fail {
		return nil, errArchiveFixture
	}
	return []byte("complete file"), nil
}

func TestArchiveCollectionPropagatesReadErrors(t *testing.T) {
	for _, nested := range []bool{false, true} {
		files := make(map[string][]byte)
		var count int32
		err := collectFilesFromContainerDir(failingArchiveReader{nested: nested, fail: true}, context.Background(), "fixture", "/root", "/root", files, &count, false, "root")
		if !errors.Is(err, errArchiveFixture) {
			t.Fatalf("archive succeeded despite missing contents: %v", err)
		}
		if count != 0 || len(files) != 0 {
			t.Fatalf("unexpected collected contents: %v", files)
		}
		err = collectFilesFromContainerDir(failingArchiveReader{nested: nested}, context.Background(), "fixture", "/root", "/root", files, &count, false, "root")
		if err != nil || count != 1 {
			t.Fatalf("complete archive rejected: %v %d", err, count)
		}
		name := "record"
		if nested {
			name = "nested/record"
		}
		if string(files[name]) != "complete file" {
			t.Fatalf("wrong archive member: %v", files)
		}
	}
}
