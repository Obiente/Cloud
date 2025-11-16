package gameservers

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/obiente/cloud/apps/shared/pkg/logger"

	"connectrpc.com/connect"
	gameserversv1 "github.com/obiente/cloud/apps/shared/proto/obiente/cloud/gameservers/v1"
)

// GetMinecraftPlayerUUID gets a Minecraft player UUID from their username
func (s *Service) GetMinecraftPlayerUUID(
	ctx context.Context,
	req *connect.Request[gameserversv1.GetMinecraftPlayerUUIDRequest],
) (*connect.Response[gameserversv1.GetMinecraftPlayerUUIDResponse], error) {
	username := req.Msg.Username
	if username == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("username is required"))
	}

	// Proxy request to Mojang API
	mojangURL := fmt.Sprintf("https://api.mojang.com/users/profiles/minecraft/%s", username)
	client := &http.Client{Timeout: 10 * time.Second}
	httpReq, err := http.NewRequestWithContext(ctx, "GET", mojangURL, nil)
	if err != nil {
		logger.Error("[MinecraftPlayer] Failed to create request: %v", err)
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to create request"))
	}
	httpReq.Header.Set("User-Agent", "ObienteCloud/1.0")

	resp, err := client.Do(httpReq)
	if err != nil {
		logger.Error("[MinecraftPlayer] Failed to fetch player UUID: %v", err)
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to fetch player data"))
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNoContent || resp.StatusCode == http.StatusNotFound {
		// Player not found
		return connect.NewResponse(&gameserversv1.GetMinecraftPlayerUUIDResponse{}), nil
	}

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		logger.Error("[MinecraftPlayer] Mojang API returned status %d: %s", resp.StatusCode, string(body))
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to fetch player data"))
	}

	var data struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		logger.Error("[MinecraftPlayer] Failed to decode response: %v", err)
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to parse response"))
	}

	// Format UUID with dashes (Mojang returns undashed)
	uuid := formatUUIDWithDashes(data.ID)

	return connect.NewResponse(&gameserversv1.GetMinecraftPlayerUUIDResponse{
		Uuid: &uuid,
		Name: &data.Name,
	}), nil
}

// GetMinecraftPlayerProfile gets a Minecraft player profile from their UUID
func (s *Service) GetMinecraftPlayerProfile(
	ctx context.Context,
	req *connect.Request[gameserversv1.GetMinecraftPlayerProfileRequest],
) (*connect.Response[gameserversv1.GetMinecraftPlayerProfileResponse], error) {
	uuid := req.Msg.Uuid
	if uuid == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("uuid is required"))
	}

	// Remove dashes for Mojang API (it expects undashed format)
	formattedUUID := strings.ReplaceAll(uuid, "-", "")

	// Proxy request to Mojang Session API
	mojangURL := fmt.Sprintf("https://sessionserver.mojang.com/session/minecraft/profile/%s", formattedUUID)
	client := &http.Client{Timeout: 10 * time.Second}
	httpReq, err := http.NewRequestWithContext(ctx, "GET", mojangURL, nil)
	if err != nil {
		logger.Error("[MinecraftPlayer] Failed to create request: %v", err)
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to create request"))
	}
	httpReq.Header.Set("User-Agent", "ObienteCloud/1.0")

	resp, err := client.Do(httpReq)
	if err != nil {
		logger.Error("[MinecraftPlayer] Failed to fetch player profile: %v", err)
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to fetch player profile"))
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNoContent || resp.StatusCode == http.StatusNotFound {
		// Player not found
		return connect.NewResponse(&gameserversv1.GetMinecraftPlayerProfileResponse{}), nil
	}

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		logger.Error("[MinecraftPlayer] Mojang API returned status %d: %s", resp.StatusCode, string(body))
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to fetch player profile"))
	}

	var data struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		logger.Error("[MinecraftPlayer] Failed to decode response: %v", err)
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to parse response"))
	}

	// Format UUID with dashes
	formattedUUIDWithDashes := formatUUIDWithDashes(data.ID)

	// Generate avatar URL from Crafatar
	avatarURL := fmt.Sprintf("https://crafatar.com/avatars/%s?size=32&overlay", formattedUUIDWithDashes)

	return connect.NewResponse(&gameserversv1.GetMinecraftPlayerProfileResponse{
		Uuid:      &formattedUUIDWithDashes,
		Name:      &data.Name,
		AvatarUrl: &avatarURL,
	}), nil
}

// formatUUIDWithDashes formats a UUID string to include dashes
// Input: "550e8400e29b41d4a716446655440000"
// Output: "550e8400-e29b-41d4-a716-446655440000"
func formatUUIDWithDashes(uuid string) string {
	uuid = strings.ReplaceAll(uuid, "-", "") // Remove existing dashes
	if len(uuid) != 32 {
		return uuid // Invalid UUID format, return as-is
	}
	return fmt.Sprintf("%s-%s-%s-%s-%s",
		uuid[0:8],
		uuid[8:12],
		uuid[12:16],
		uuid[16:20],
		uuid[20:32],
	)
}
