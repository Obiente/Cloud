package service

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/obiente/cloud/apps/shared/pkg/database"
)

type Permission string

const (
	PermissionRead  Permission = "read"
	PermissionWrite Permission = "write"
)

type Session struct {
	CredentialID   string
	UserID         string
	OrganizationID string
	ResourceType   string
	ResourceID     string
	RootPath       string
	Permissions    []Permission
}

type Authenticator struct {
	credentials *database.FileTransferCredentialRepository
	gameServers *database.GameServerRepository
	volumeRoot  string
}

func NewAuthenticator(volumeRoot string) *Authenticator {
	if volumeRoot == "" {
		volumeRoot = "/var/lib/obiente/volumes"
	}
	return &Authenticator{
		credentials: database.NewFileTransferCredentialRepository(database.DB),
		gameServers: database.NewGameServerRepository(database.DB, database.RedisClient),
		volumeRoot:  volumeRoot,
	}
}

func (a *Authenticator) Authenticate(ctx context.Context, secret string) (*Session, error) {
	credential, err := a.credentials.GetActiveBySecret(ctx, secret, time.Now())
	if err != nil {
		return nil, fmt.Errorf("invalid credentials")
	}

	resourceType := database.NormalizeFileTransferResourceType(credential.ResourceType)
	var root string
	switch resourceType {
	case database.FileTransferResourceGameServer:
		root, err = a.resolveGameServerRoot(ctx, credential)
	default:
		err = fmt.Errorf("unsupported resource type %q", credential.ResourceType)
	}
	if err != nil {
		return nil, err
	}

	permissions := make([]Permission, 0, 2)
	if database.FileTransferCredentialHasScope(credential.Scopes, database.FileTransferScopeRead) {
		permissions = append(permissions, PermissionRead)
	}
	if database.FileTransferCredentialHasScope(credential.Scopes, database.FileTransferScopeWrite) {
		permissions = append(permissions, PermissionWrite)
	}
	if len(permissions) == 0 {
		return nil, fmt.Errorf("credential has no file transfer permissions")
	}

	go func() {
		touchCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = a.credentials.TouchLastUsed(touchCtx, credential.ID, time.Now())
	}()

	return &Session{
		CredentialID:   credential.ID,
		UserID:         credential.UserID,
		OrganizationID: credential.OrganizationID,
		ResourceType:   resourceType,
		ResourceID:     credential.ResourceID,
		RootPath:       root,
		Permissions:    permissions,
	}, nil
}

func (a *Authenticator) resolveGameServerRoot(ctx context.Context, credential *database.FileTransferCredential) (string, error) {
	gameServer, err := a.gameServers.GetByID(ctx, credential.ResourceID)
	if err != nil {
		return "", fmt.Errorf("game server not found")
	}
	if gameServer.OrganizationID != credential.OrganizationID {
		return "", fmt.Errorf("game server does not belong to credential organization")
	}

	root := filepath.Join(a.volumeRoot, fmt.Sprintf("gameserver-%s-data", gameServer.ID))
	info, err := os.Stat(root)
	if err != nil {
		return "", fmt.Errorf("gameserver volume is not available on this node")
	}
	if !info.IsDir() {
		return "", fmt.Errorf("gameserver volume path is not a directory")
	}
	return root, nil
}

func hasPermission(permissions []Permission, required Permission) bool {
	for _, permission := range permissions {
		if permission == required {
			return true
		}
	}
	return false
}
