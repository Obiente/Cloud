package docker

import (
	"context"
	"fmt"
	"github.com/containerd/errdefs"
	"github.com/moby/moby/api/types/container"
	"github.com/moby/moby/api/types/network"
	"github.com/moby/moby/client"
)

func networkEndpoint(inspect container.InspectResponse, networkID, name string) *network.EndpointSettings {
	if inspect.NetworkSettings == nil {
		return nil
	}
	if endpoint := inspect.NetworkSettings.Networks[name]; endpoint != nil {
		return endpoint
	}
	// Docker keys pending attachments to stopped overlay containers by ID.
	return inspect.NetworkSettings.Networks[networkID]
}

func networkEndpointUsable(inspect container.InspectResponse, networkID, name string) bool {
	endpoint := networkEndpoint(inspect, networkID, name)
	if endpoint == nil || inspect.State == nil {
		return false
	}
	if endpoint.NetworkID != networkID {
		// An ID-keyed pending attachment identifies the current network even before
		// Docker allocates its endpoint. Never accept this for a running container.
		return !inspect.State.Running && endpoint.NetworkID == "" && inspect.NetworkSettings.Networks[networkID] == endpoint
	}
	return !inspect.State.Running || (endpoint.EndpointID != "" && (endpoint.IPAddress.IsValid() || endpoint.GlobalIPv6Address.IsValid()))
}

// EnsureContainerNetwork repairs stale overlay identities, including stopped containers.
// It never recreates a container or changes its volumes or intended running state.
func (c *Client) EnsureContainerNetwork(ctx context.Context, containerID, name string) error {
	if c == nil || c.api == nil {
		return ErrUninitialized
	}
	desired, err := c.api.NetworkInspect(ctx, name, client.NetworkInspectOptions{})
	if err != nil {
		return fmt.Errorf("inspect network %s: %w", name, err)
	}
	if desired.Network.ID == "" {
		return fmt.Errorf("network %s has no identity", name)
	}
	inspect, err := c.ContainerInspect(ctx, containerID)
	if err != nil {
		return err
	}
	if networkEndpointUsable(inspect, desired.Network.ID, name) {
		return nil
	}
	endpoint := networkEndpoint(inspect, desired.Network.ID, name)
	if endpoint != nil {
		oldID := endpoint.NetworkID
		if oldID == "" {
			oldID = name
		}
		_, err = c.api.NetworkDisconnect(ctx, oldID, client.NetworkDisconnectOptions{Container: containerID, Force: true})
		if errdefs.IsNotFound(err) && oldID != name {
			_, err = c.api.NetworkDisconnect(ctx, name, client.NetworkDisconnectOptions{Container: containerID, Force: true})
		}
		if err != nil && !errdefs.IsNotFound(err) {
			return fmt.Errorf("disconnect stale network: %w", err)
		}
	}
	var config *network.EndpointSettings
	if endpoint != nil {
		config = &network.EndpointSettings{Aliases: endpoint.Aliases}
	}
	_, err = c.api.NetworkConnect(ctx, desired.Network.ID, client.NetworkConnectOptions{Container: containerID, EndpointConfig: config})
	if err != nil {
		return fmt.Errorf("connect replacement network: %w", err)
	}
	inspect, err = c.ContainerInspect(ctx, containerID)
	if err != nil {
		return err
	}
	// Docker defers NetworkID/IP allocation for a stopped container until start.
	// A successful connect to the inspected ID is authoritative for this new
	// attachment; the next running check still requires the actual current ID/IP.
	if inspect.State != nil && !inspect.State.Running && inspect.NetworkSettings != nil {
		attached := networkEndpoint(inspect, desired.Network.ID, name)
		if attached != nil && attached.NetworkID == "" {
			return nil
		}
	}
	if !networkEndpointUsable(inspect, desired.Network.ID, name) {
		return fmt.Errorf("network %s endpoint is not ready", name)
	}
	return nil
}
