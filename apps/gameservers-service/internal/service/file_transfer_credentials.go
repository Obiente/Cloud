package gameservers

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/obiente/cloud/apps/shared/pkg/auth"
	"github.com/obiente/cloud/apps/shared/pkg/database"

	gameserversv1 "github.com/obiente/cloud/apps/shared/proto/obiente/cloud/gameservers/v1"

	"connectrpc.com/connect"
	"google.golang.org/protobuf/types/known/timestamppb"
)

const fileTransferUsernamePrefix = "gs_"

func (s *Service) ListGameServerFileTransferCredentials(ctx context.Context, req *connect.Request[gameserversv1.ListGameServerFileTransferCredentialsRequest]) (*connect.Response[gameserversv1.ListGameServerFileTransferCredentialsResponse], error) {
	gameServerID := strings.TrimSpace(req.Msg.GetGameServerId())
	if gameServerID == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("game server ID is required"))
	}
	if err := s.checkGameServerPermission(ctx, gameServerID, auth.PermissionGameServersRead); err != nil {
		return nil, err
	}

	repo := database.NewFileTransferCredentialRepository(database.DB)
	credentials, err := repo.ListActiveByResource(ctx, database.FileTransferResourceGameServer, gameServerID, time.Now())
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to list file transfer credentials: %w", err))
	}

	return connect.NewResponse(&gameserversv1.ListGameServerFileTransferCredentialsResponse{
		Credentials: dbFileTransferCredentialsToProto(credentials, gameServerID),
		Connection:  fileTransferConnectionInfo(gameServerID),
	}), nil
}

func (s *Service) CreateGameServerFileTransferCredential(ctx context.Context, req *connect.Request[gameserversv1.CreateGameServerFileTransferCredentialRequest]) (*connect.Response[gameserversv1.CreateGameServerFileTransferCredentialResponse], error) {
	gameServerID := strings.TrimSpace(req.Msg.GetGameServerId())
	if gameServerID == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("game server ID is required"))
	}
	if err := s.checkGameServerPermission(ctx, gameServerID, auth.PermissionGameServersUpdate); err != nil {
		return nil, err
	}

	gameServer, err := s.repo.GetByID(ctx, gameServerID)
	if err != nil {
		return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("game server %s not found", gameServerID))
	}

	userInfo, err := auth.GetUserFromContext(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeUnauthenticated, fmt.Errorf("user authentication required: %w", err))
	}

	name := strings.TrimSpace(req.Msg.GetName())
	if name == "" {
		name = "SFTP credential"
	}
	if len(name) > 120 {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("credential name must be 120 characters or fewer"))
	}

	scopes := normalizeRequestedFileTransferScopes(req.Msg.GetScopes())
	secret, err := database.GenerateFileTransferSecret()
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to generate file transfer password: %w", err))
	}

	var expiresAt *time.Time
	if req.Msg.ExpiresAt != nil {
		expires := req.Msg.ExpiresAt.AsTime()
		if !expires.After(time.Now()) {
			return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("expiration must be in the future"))
		}
		expiresAt = &expires
	}

	credential := &database.FileTransferCredential{
		Name:           name,
		KeyHash:        database.HashFileTransferSecret(secret),
		UserID:         userInfo.Id,
		OrganizationID: gameServer.OrganizationID,
		ResourceType:   database.FileTransferResourceGameServer,
		ResourceID:     gameServerID,
		Scopes:         strings.Join(scopes, ","),
		ExpiresAt:      expiresAt,
	}

	repo := database.NewFileTransferCredentialRepository(database.DB)
	if err := repo.Create(ctx, credential); err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to create file transfer credential: %w", err))
	}

	return connect.NewResponse(&gameserversv1.CreateGameServerFileTransferCredentialResponse{
		Credential: dbFileTransferCredentialToProto(credential, gameServerID),
		Password:   secret,
		Connection: fileTransferConnectionInfo(gameServerID),
	}), nil
}

func (s *Service) RevokeGameServerFileTransferCredential(ctx context.Context, req *connect.Request[gameserversv1.RevokeGameServerFileTransferCredentialRequest]) (*connect.Response[gameserversv1.RevokeGameServerFileTransferCredentialResponse], error) {
	gameServerID := strings.TrimSpace(req.Msg.GetGameServerId())
	credentialID := strings.TrimSpace(req.Msg.GetCredentialId())
	if gameServerID == "" || credentialID == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("game server ID and credential ID are required"))
	}
	if err := s.checkGameServerPermission(ctx, gameServerID, auth.PermissionGameServersUpdate); err != nil {
		return nil, err
	}

	repo := database.NewFileTransferCredentialRepository(database.DB)
	if err := repo.RevokeByResource(ctx, credentialID, database.FileTransferResourceGameServer, gameServerID, time.Now()); err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to revoke file transfer credential: %w", err))
	}

	return connect.NewResponse(&gameserversv1.RevokeGameServerFileTransferCredentialResponse{Success: true}), nil
}

func normalizeRequestedFileTransferScopes(scopes []string) []string {
	normalized := database.NormalizeFileTransferScopes(strings.Join(scopes, ","))
	out := strings.Split(normalized, ",")
	if len(out) == 0 {
		return []string{database.FileTransferScopeRead}
	}
	return out
}

func dbFileTransferCredentialsToProto(credentials []*database.FileTransferCredential, gameServerID string) []*gameserversv1.GameServerFileTransferCredential {
	out := make([]*gameserversv1.GameServerFileTransferCredential, 0, len(credentials))
	for _, credential := range credentials {
		out = append(out, dbFileTransferCredentialToProto(credential, gameServerID))
	}
	return out
}

func dbFileTransferCredentialToProto(credential *database.FileTransferCredential, gameServerID string) *gameserversv1.GameServerFileTransferCredential {
	if credential == nil {
		return nil
	}
	item := &gameserversv1.GameServerFileTransferCredential{
		Id:        credential.ID,
		Name:      credential.Name,
		Username:  fileTransferUsername(gameServerID),
		Scopes:    strings.Split(database.NormalizeFileTransferScopes(credential.Scopes), ","),
		CreatedAt: timestamppb.New(credential.CreatedAt),
	}
	if credential.LastUsedAt != nil {
		item.LastUsedAt = timestamppb.New(*credential.LastUsedAt)
	}
	if credential.ExpiresAt != nil {
		item.ExpiresAt = timestamppb.New(*credential.ExpiresAt)
	}
	return item
}

func fileTransferConnectionInfo(gameServerID string) *gameserversv1.GameServerFileTransferConnectionInfo {
	username := fileTransferUsername(gameServerID)
	host := strings.TrimSpace(os.Getenv("FILE_TRANSFER_PUBLIC_HOST"))
	if host == "" {
		host = strings.TrimSpace(os.Getenv("DOMAIN"))
	}
	if host == "" {
		host = "localhost"
	}

	port := int32(2223)
	if rawPort := strings.TrimSpace(os.Getenv("FILE_TRANSFER_SFTP_PUBLIC_PORT")); rawPort != "" {
		if parsed, err := strconv.Atoi(rawPort); err == nil && parsed > 0 && parsed <= 65535 {
			port = int32(parsed)
		}
	}

	command := fmt.Sprintf("sftp -P %d %s@%s", port, username, host)
	return &gameserversv1.GameServerFileTransferConnectionInfo{
		Host:     host,
		Port:     port,
		Username: username,
		Protocol: "sftp",
		Command:  command,
	}
}

func fileTransferUsername(gameServerID string) string {
	return fileTransferUsernamePrefix + strings.TrimSpace(gameServerID)
}
