package orchestrator

import "testing"

func TestTraefikHostRule(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name   string
		domain string
		want   string
	}{
		{
			name:   "exact domain",
			domain: "example.com",
			want:   "Host(`example.com`)",
		},
		{
			name:   "wildcard domain",
			domain: "*.example.com",
			want:   "HostRegexp(`{subdomain:[^.]+}.example.com`)",
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if got := traefikHostRule(tc.domain); got != tc.want {
				t.Fatalf("traefikHostRule(%q) = %q, want %q", tc.domain, got, tc.want)
			}
		})
	}
}

func TestTraefikRouterPriority(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name   string
		domain string
		want   string
	}{
		{
			name:   "exact domain priority",
			domain: "example.com",
			want:   "200",
		},
		{
			name:   "wildcard domain priority",
			domain: "*.example.com",
			want:   "100",
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if got := traefikRouterPriority(tc.domain); got != tc.want {
				t.Fatalf("traefikRouterPriority(%q) = %q, want %q", tc.domain, got, tc.want)
			}
		})
	}
}
