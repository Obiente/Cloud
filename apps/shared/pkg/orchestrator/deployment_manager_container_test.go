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
