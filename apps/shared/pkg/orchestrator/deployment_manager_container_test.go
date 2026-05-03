package orchestrator

import "testing"

func TestSwarmMemoryReservation(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name       string
		limitBytes int64
		want       int64
	}{
		{
			name:       "zero limit",
			limitBytes: 0,
			want:       0,
		},
		{
			name:       "small limit uses floor",
			limitBytes: 256 * 1024 * 1024,
			want:       32 * 1024 * 1024,
		},
		{
			name:       "default two gigabyte limit reserves five percent",
			limitBytes: 2 * 1024 * 1024 * 1024,
			want:       107374182,
		},
		{
			name:       "large limit uses cap",
			limitBytes: 8 * 1024 * 1024 * 1024,
			want:       128 * 1024 * 1024,
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if got := swarmMemoryReservation(tc.limitBytes); got != tc.want {
				t.Fatalf("swarmMemoryReservation(%d) = %d, want %d", tc.limitBytes, got, tc.want)
			}
		})
	}
}

func TestSwarmCPUReservation(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name       string
		limitCores float64
		want       float64
	}{
		{
			name:       "zero limit",
			limitCores: 0,
			want:       0,
		},
		{
			name:       "small limit uses floor",
			limitCores: 0.25,
			want:       0.01,
		},
		{
			name:       "default two core limit reserves five hundredths",
			limitCores: 2,
			want:       0.05,
		},
		{
			name:       "large limit uses cap",
			limitCores: 8,
			want:       0.10,
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if got := swarmCPUReservation(tc.limitCores); got != tc.want {
				t.Fatalf("swarmCPUReservation(%f) = %f, want %f", tc.limitCores, got, tc.want)
			}
		})
	}
}

func TestSwarmDisableHealthcheckArgs(t *testing.T) {
	t.Parallel()

	got := swarmDisableHealthcheckArgs()
	if len(got) != 1 || got[0] != "--no-healthcheck" {
		t.Fatalf("swarmDisableHealthcheckArgs() = %#v, want [--no-healthcheck]", got)
	}
}

func TestDockerfileVolumeSanitizers(t *testing.T) {
	t.Parallel()

	validNames := []string{"data", "uploads.v1", "cache-dir", "_private"}
	for _, name := range validNames {
		name := name
		t.Run("valid name "+name, func(t *testing.T) {
			t.Parallel()
			if got := sanitizeVolumeName(name); got != name {
				t.Fatalf("sanitizeVolumeName(%q) = %q, want %q", name, got, name)
			}
		})
	}

	invalidNames := []string{"", ".", "..", "../host", "data/slash", "bad name", "bad:mode", "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"}
	for _, name := range invalidNames {
		name := name
		t.Run("invalid name "+name, func(t *testing.T) {
			t.Parallel()
			if got := sanitizeVolumeName(name); got != "" {
				t.Fatalf("sanitizeVolumeName(%q) = %q, want empty", name, got)
			}
		})
	}

	validMounts := map[string]string{
		"/data":          "/data",
		" /app/uploads/": "/app/uploads",
	}
	for input, want := range validMounts {
		input, want := input, want
		t.Run("valid mount "+input, func(t *testing.T) {
			t.Parallel()
			if got := sanitizeContainerMountPath(input); got != want {
				t.Fatalf("sanitizeContainerMountPath(%q) = %q, want %q", input, got, want)
			}
		})
	}

	invalidMounts := []string{
		"",
		"data",
		"/",
		"/proc",
		"/proc/self",
		"/sys/kernel",
		"/dev/shm",
		"/app/../proc/self",
		"/var/run/docker.sock",
		"/run/docker.sock",
		"/data:rw",
		"/data\x00evil",
	}
	for _, mount := range invalidMounts {
		mount := mount
		t.Run("invalid mount "+mount, func(t *testing.T) {
			t.Parallel()
			if got := sanitizeContainerMountPath(mount); got != "" {
				t.Fatalf("sanitizeContainerMountPath(%q) = %q, want empty", mount, got)
			}
		})
	}
}
