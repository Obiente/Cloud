package deployments

import (
	"reflect"
	"strings"
	"testing"

	deploymentsv1 "github.com/obiente/cloud/apps/shared/proto/obiente/cloud/deployments/v1"
)

func TestSanitizeBuildArgsRejectsUnsafeInput(t *testing.T) {
	t.Parallel()

	valid, err := sanitizeBuildArgs(map[string]string{
		"NODE_VERSION": "22",
		"_TOKEN":       "abc;$PATH",
		"APP_ENV":      "production",
	})
	if err != nil {
		t.Fatalf("sanitizeBuildArgs returned error for valid args: %v", err)
	}
	if valid["NODE_VERSION"] != "22" || valid["_TOKEN"] != "abc;$PATH" || valid["APP_ENV"] != "production" {
		t.Fatalf("sanitizeBuildArgs returned unexpected args: %#v", valid)
	}

	cases := []struct {
		name string
		args map[string]string
	}{
		{name: "empty key", args: map[string]string{"": "value"}},
		{name: "leading digit", args: map[string]string{"1BAD": "value"}},
		{name: "docker flag injection", args: map[string]string{"--progress": "plain"}},
		{name: "path traversal", args: map[string]string{"../../SECRET": "value"}},
		{name: "shell metacharacter", args: map[string]string{"BAD;rm": "value"}},
		{name: "space", args: map[string]string{"BAD ARG": "value"}},
		{name: "equals", args: map[string]string{"BAD=ARG": "value"}},
		{name: "null byte value", args: map[string]string{"SAFE_ARG": "bad\x00value"}},
		{name: "newline value", args: map[string]string{"SAFE_ARG": "bad\nvalue"}},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if _, err := sanitizeBuildArgs(tc.args); err == nil {
				t.Fatalf("sanitizeBuildArgs(%#v) returned nil error", tc.args)
			}
		})
	}
}

func TestDockerBuildCommandArgsKeepsBuildArgValuesAtomic(t *testing.T) {
	t.Parallel()

	got, err := dockerBuildCommandArgs("/tmp/repo", "registry.example/app:latest", "deploy/Dockerfile", map[string]string{
		"SAFE_ARG": "value; touch /tmp/owned && echo $HOME",
	})
	if err != nil {
		t.Fatalf("dockerBuildCommandArgs returned error: %v", err)
	}
	want := []string{
		"build",
		"-t",
		"registry.example/app:latest",
		"-f",
		"/tmp/repo/deploy/Dockerfile",
		"--build-arg",
		"SAFE_ARG=value; touch /tmp/owned && echo $HOME",
		"/tmp/repo",
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("dockerBuildCommandArgs() = %#v, want %#v", got, want)
	}

	if _, err := dockerBuildCommandArgs("/tmp/repo", "image", "", map[string]string{"--add-host": "host.docker.internal:host-gateway"}); err == nil {
		t.Fatal("dockerBuildCommandArgs returned nil error for unsafe build arg key")
	}
}

func TestSanitizeDockerfileVolumesRejectsUnsafeInput(t *testing.T) {
	t.Parallel()

	valid, err := sanitizeDockerfileVolumes([]*deploymentsv1.DockerfileVolume{
		{Name: "data", MountPath: "/data"},
		{Name: "uploads.v1", MountPath: "/app/uploads/", ReadOnly: true},
	})
	if err != nil {
		t.Fatalf("sanitizeDockerfileVolumes returned error for valid volumes: %v", err)
	}
	if got := valid[1]["mount_path"]; got != "/app/uploads" {
		t.Fatalf("sanitizeDockerfileVolumes cleaned mount path = %q, want /app/uploads", got)
	}

	longName := strings.Repeat("a", 65)
	cases := []struct {
		name    string
		volumes []*deploymentsv1.DockerfileVolume
	}{
		{name: "empty name", volumes: []*deploymentsv1.DockerfileVolume{{Name: "", MountPath: "/data"}}},
		{name: "path traversal name", volumes: []*deploymentsv1.DockerfileVolume{{Name: "../host", MountPath: "/data"}}},
		{name: "slash name", volumes: []*deploymentsv1.DockerfileVolume{{Name: "data/slash", MountPath: "/data"}}},
		{name: "overlong name", volumes: []*deploymentsv1.DockerfileVolume{{Name: longName, MountPath: "/data"}}},
		{name: "relative mount", volumes: []*deploymentsv1.DockerfileVolume{{Name: "data", MountPath: "data"}}},
		{name: "root mount", volumes: []*deploymentsv1.DockerfileVolume{{Name: "data", MountPath: "/"}}},
		{name: "proc mount", volumes: []*deploymentsv1.DockerfileVolume{{Name: "data", MountPath: "/proc/self"}}},
		{name: "sys mount", volumes: []*deploymentsv1.DockerfileVolume{{Name: "data", MountPath: "/sys/kernel"}}},
		{name: "dev mount", volumes: []*deploymentsv1.DockerfileVolume{{Name: "data", MountPath: "/dev/shm"}}},
		{name: "cleaned sensitive mount", volumes: []*deploymentsv1.DockerfileVolume{{Name: "data", MountPath: "/app/../proc/self"}}},
		{name: "docker socket", volumes: []*deploymentsv1.DockerfileVolume{{Name: "data", MountPath: "/var/run/docker.sock"}}},
		{name: "colon mount", volumes: []*deploymentsv1.DockerfileVolume{{Name: "data", MountPath: "/data:rw"}}},
		{name: "null byte mount", volumes: []*deploymentsv1.DockerfileVolume{{Name: "data", MountPath: "/data\x00evil"}}},
		{name: "duplicate cleaned mount", volumes: []*deploymentsv1.DockerfileVolume{
			{Name: "data", MountPath: "/data"},
			{Name: "other", MountPath: "/data/"},
		}},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if _, err := sanitizeDockerfileVolumes(tc.volumes); err == nil {
				t.Fatalf("sanitizeDockerfileVolumes(%#v) returned nil error", tc.volumes)
			}
		})
	}
}
