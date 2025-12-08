package databases

import (
	"context"

	"github.com/obiente/cloud/apps/shared/pkg/auth"
	"github.com/obiente/cloud/apps/shared/pkg/database"
	"github.com/obiente/cloud/apps/shared/pkg/services/common"

	databasesv1connect "github.com/obiente/cloud/apps/shared/proto/obiente/cloud/databases/v1/databasesv1connect"

	"connectrpc.com/connect"
)

type Service struct {
	databasesv1connect.UnimplementedDatabaseServiceHandler
	permissionChecker *auth.PermissionChecker
	repo              *database.DatabaseRepository
	connRepo          *database.DatabaseConnectionRepository
	backupRepo        *database.DatabaseBackupRepository
}

func NewService(
	repo *database.DatabaseRepository,
	connRepo *database.DatabaseConnectionRepository,
	backupRepo *database.DatabaseBackupRepository,
) *Service {
	return &Service{
		permissionChecker: auth.NewPermissionChecker(),
		repo:              repo,
		connRepo:          connRepo,
		backupRepo:        backupRepo,
	}
}

// ensureAuthenticated ensures the user is authenticated for streaming RPCs
func (s *Service) ensureAuthenticated(ctx context.Context, req connect.AnyRequest) (context.Context, error) {
	return common.EnsureAuthenticated(ctx, req)
}

// checkDatabasePermission verifies user permissions for a database
func (s *Service) checkDatabasePermission(ctx context.Context, databaseID string, permission string) error {
	return auth.CheckResourcePermissionWithError(ctx, s.permissionChecker, "database", databaseID, permission)
}

// checkOrganizationPermission verifies user has access to an organization
func (s *Service) checkOrganizationPermission(ctx context.Context, organizationID string) error {
	return auth.CheckScopedPermissionWithError(ctx, s.permissionChecker, organizationID, auth.ScopedPermission{
		Permission: auth.PermissionOrganizationRead,
	})
}

