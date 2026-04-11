package orchestrator

import (
	"reflect"
	"testing"
)

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

func TestNormalizeServiceNetworksFromList(t *testing.T) {
	service := map[string]interface{}{
		"networks": []interface{}{"deployment-123", "obiente-network"},
	}

	got := normalizeServiceNetworks(service)
	want := map[string]interface{}{
		"deployment-123": nil,
		"obiente-network": nil,
	}

	if !reflect.DeepEqual(got, want) {
		t.Fatalf("normalizeServiceNetworks mismatch\nwant: %#v\ngot:  %#v", want, got)
	}
}

func TestMergeNetworkAliasesAddsAlias(t *testing.T) {
	got := mergeNetworkAliases(nil, "cache")
	want := map[string]interface{}{
		"aliases": []interface{}{"cache"},
	}

	if !reflect.DeepEqual(got, want) {
		t.Fatalf("mergeNetworkAliases mismatch\nwant: %#v\ngot:  %#v", want, got)
	}
}

func TestMergeNetworkAliasesPreservesExistingAliases(t *testing.T) {
	input := map[string]interface{}{
		"aliases": []interface{}{"cache", "cache.internal"},
	}

	got := mergeNetworkAliases(input, "cache")
	want := map[string]interface{}{
		"aliases": []interface{}{"cache", "cache.internal"},
	}

	if !reflect.DeepEqual(got, want) {
		t.Fatalf("mergeNetworkAliases should preserve aliases\nwant: %#v\ngot:  %#v", want, got)
	}
}
