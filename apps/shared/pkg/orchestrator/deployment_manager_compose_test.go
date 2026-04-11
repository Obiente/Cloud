package orchestrator

import (
	"reflect"
	"testing"
)

func TestComposeUpArgs(t *testing.T) {
	got := composeUpArgs("deploy-123", "/tmp/deploy/docker-compose.yml")
	want := []string{
		"compose",
		"-p",
		"deploy-123",
		"-f",
		"/tmp/deploy/docker-compose.yml",
		"up",
		"-d",
		"--build",
		"--pull",
		"always",
		"--force-recreate",
		"--remove-orphans",
	}

	if !reflect.DeepEqual(got, want) {
		t.Fatalf("composeUpArgs mismatch\nwant: %#v\ngot:  %#v", want, got)
	}
}

func TestStackDeployArgs(t *testing.T) {
	got := stackDeployArgs("deploy-123", "/tmp/deploy/docker-compose.yml")
	want := []string{
		"stack",
		"deploy",
		"-c",
		"/tmp/deploy/docker-compose.yml",
		"--with-registry-auth=true",
		"--resolve-image",
		"always",
		"deploy-123",
	}

	if !reflect.DeepEqual(got, want) {
		t.Fatalf("stackDeployArgs mismatch\nwant: %#v\ngot:  %#v", want, got)
	}
}