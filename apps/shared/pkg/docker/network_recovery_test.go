package docker

import (
	"context"
	"errors"
	"github.com/moby/moby/api/types/container"
	"github.com/moby/moby/api/types/network"
	"github.com/moby/moby/client"
	"net/netip"
	"testing"
)

type networkRecoveryAPI struct {
	client.APIClient
	inspect               container.InspectResponse
	disconnects, connects int
	connectError          error
}

func (f *networkRecoveryAPI) NetworkInspect(context.Context, string, client.NetworkInspectOptions) (client.NetworkInspectResult, error) {
	return client.NetworkInspectResult{Network: network.Inspect{Network: network.Network{ID: "current-network"}}}, nil
}
func (f *networkRecoveryAPI) ContainerInspect(context.Context, string, client.ContainerInspectOptions) (client.ContainerInspectResult, error) {
	return client.ContainerInspectResult{Container: f.inspect}, nil
}
func (f *networkRecoveryAPI) NetworkDisconnect(context.Context, string, client.NetworkDisconnectOptions) (client.NetworkDisconnectResult, error) {
	f.disconnects++
	return client.NetworkDisconnectResult{}, nil
}
func (f *networkRecoveryAPI) NetworkConnect(_ context.Context, _ string, options client.NetworkConnectOptions) (client.NetworkConnectResult, error) {
	f.connects++
	if f.connectError != nil {
		return client.NetworkConnectResult{}, f.connectError
	}
	endpoint := &network.EndpointSettings{NetworkID: "current-network", EndpointID: "endpoint", IPAddress: netip.MustParseAddr("192.0.2.5")}
	if options.EndpointConfig != nil {
		endpoint.Aliases = options.EndpointConfig.Aliases
	}
	f.inspect.NetworkSettings = &container.NetworkSettings{Networks: map[string]*network.EndpointSettings{"test-network": endpoint}}
	return client.NetworkConnectResult{}, nil
}
func TestNetworkRecovery(t *testing.T) {
	for _, tc := range []struct {
		name, id    string
		running, ip bool
		repairs     int
	}{
		{"healthy", "current-network", true, true, 0},
		{"recreated overlay", "old-network", true, true, 1},
		{"missing address", "current-network", true, false, 1},
		{"stopped container stale overlay", "old-network", false, false, 1},
		{"stopped container valid attachment", "current-network", false, false, 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			endpoint := &network.EndpointSettings{NetworkID: tc.id, EndpointID: "endpoint", Aliases: []string{"db-test"}}
			if tc.ip {
				endpoint.IPAddress = netip.MustParseAddr("192.0.2.4")
			}
			f := &networkRecoveryAPI{inspect: container.InspectResponse{State: &container.State{Running: tc.running}, NetworkSettings: &container.NetworkSettings{Networks: map[string]*network.EndpointSettings{"test-network": endpoint}}}}
			c := &Client{api: f}
			if err := c.EnsureContainerNetwork(context.Background(), "container-test", "test-network"); err != nil {
				t.Fatal(err)
			}
			if f.connects != tc.repairs || f.disconnects != tc.repairs {
				t.Fatalf("unexpected repairs: %d/%d", f.connects, f.disconnects)
			}
			if err := c.EnsureContainerNetwork(context.Background(), "container-test", "test-network"); err != nil {
				t.Fatal(err)
			}
			if f.connects != tc.repairs {
				t.Fatal("healthy endpoint was reattached")
			}
		})
	}
}
func TestNetworkRecoveryRetriesAfterDockerFailure(t *testing.T) {
	f := &networkRecoveryAPI{inspect: container.InspectResponse{State: &container.State{Running: true}}, connectError: errors.New("daemon unavailable")}
	c := &Client{api: f}
	if err := c.EnsureContainerNetwork(context.Background(), "container-test", "test-network"); err == nil {
		t.Fatal("failure was hidden")
	}
	f.connectError = nil
	if err := c.EnsureContainerNetwork(context.Background(), "container-test", "test-network"); err != nil {
		t.Fatal(err)
	}
}

func TestStoppedOverlayAttachmentUsesNetworkIDUntilStart(t *testing.T) {
	inspect := container.InspectResponse{State: &container.State{}, NetworkSettings: &container.NetworkSettings{Networks: map[string]*network.EndpointSettings{"current-network": {}}}}
	if !networkEndpointUsable(inspect, "current-network", "test-network") {
		t.Fatal("pending overlay attachment was not recognized")
	}
	if networkEndpointUsable(inspect, "replacement-network", "test-network") {
		t.Fatal("old overlay identity was accepted")
	}
	inspect.State.Running = true
	if networkEndpointUsable(inspect, "current-network", "test-network") {
		t.Fatal("running container without usable endpoint was accepted")
	}
}
