package orchestrator

import (
	"context"
	"github.com/obiente/cloud/apps/shared/pkg/database"
)

func withDeploymentRuntimeLock(ctx context.Context, id string, operation func(context.Context) error) error {
	return database.WithResourceRuntimeLock(ctx, "deployment:"+id, operation)
}
