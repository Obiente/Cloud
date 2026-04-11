package orchestrator

import (
	"reflect"
	"strings"
	"testing"

	"github.com/obiente/cloud/apps/shared/pkg/database"
	"gopkg.in/yaml.v3"
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

func TestAddTraefikNetworkToRoutedServicesPreservesSwarmNetworkAliases(t *testing.T) {
	t.Setenv("ENABLE_SWARM", "true")

	dm := &DeploymentManager{}
	composeYaml := strings.TrimSpace(`
services:
  directus:
    image: directus/directus:11
    networks:
      deployment-123:
        aliases:
          - directus
      obiente-network:
        aliases:
          - directus
networks:
  deployment-123: {}
  obiente-network:
    external: true
    name: obiente_obiente-network
`)

	routings := []database.DeploymentRouting{{ServiceName: "directus"}}
	gotYaml, err := dm.addTraefikNetworkToRoutedServices(composeYaml, routings)
	if err != nil {
		t.Fatalf("addTraefikNetworkToRoutedServices failed: %v", err)
	}

	var compose map[string]interface{}
	if err := yaml.Unmarshal([]byte(gotYaml), &compose); err != nil {
		t.Fatalf("unmarshal result: %v", err)
	}

	services := compose["services"].(map[string]interface{})
	directus := services["directus"].(map[string]interface{})
	networks := directus["networks"].(map[string]interface{})

	obienteNetwork, ok := networks["obiente-network"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected obiente-network to remain map config, got %T", networks["obiente-network"])
	}

	aliases, ok := obienteNetwork["aliases"].([]interface{})
	if !ok {
		t.Fatalf("expected obiente-network aliases to be preserved, got %T", obienteNetwork["aliases"])
	}

	if !reflect.DeepEqual(aliases, []interface{}{"directus"}) {
		t.Fatalf("expected aliases to remain intact, got %#v", aliases)
	}

	deploymentNetwork, ok := networks["deployment-123"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected deployment network to remain map config, got %T", networks["deployment-123"])
	}

	deploymentAliases, ok := deploymentNetwork["aliases"].([]interface{})
	if !ok {
		t.Fatalf("expected deployment network aliases to be preserved, got %T", deploymentNetwork["aliases"])
	}

	if !reflect.DeepEqual(deploymentAliases, []interface{}{"directus"}) {
		t.Fatalf("expected deployment aliases to remain intact, got %#v", deploymentAliases)
	}
}
