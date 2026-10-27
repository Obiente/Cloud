package orchestrator

import (
	"context"
	"fmt"
	"github.com/obiente/cloud/apps/shared/pkg/database"
	deploymentsv1 "github.com/obiente/cloud/apps/shared/proto/obiente/cloud/deployments/v1"
)

func (dm *DeploymentManager) ReconcileDeploymentIntentLocked(ctx context.Context, id string) error {
	var dep database.Deployment
	if err := database.DB.WithContext(ctx).Where("id = ? AND deleted_at IS NULL", id).First(&dep).Error; err != nil {
		return err
	}
	if dep.Status == int32(deploymentsv1.DeploymentStatus_BUILDING) || dep.Status == int32(deploymentsv1.DeploymentStatus_DEPLOYING) {
		return nil
	}
	locations, err := database.GetAllDeploymentLocations(id)
	if err != nil {
		return err
	}
	owner, err := database.GetDeploymentVolumeNode(ctx, id)
	if err != nil {
		return err
	}
	if owner == "" && len(locations) > 0 {
		owner = locations[0].NodeID
	}
	if owner == "" {
		target, err := dm.SelectTargetNode(ctx, "")
		if err != nil {
			return err
		}
		owner = target.ID
	}
	// Volumes remain on their owning node. A different node must not recreate empty data.
	if owner != dm.nodeID {
		return nil
	}
	switch dep.DesiredState {
	case "running":
		if err := dm.StartDeployment(ctx, id); err != nil {
			return err
		}
		return database.DB.WithContext(ctx).Model(&database.Deployment{}).Where("id = ? AND desired_state = 'running'", id).Update("status", int32(deploymentsv1.DeploymentStatus_RUNNING)).Error
	case "stopped":
		if dep.ComposeYaml != "" {
			err = dm.StopComposeDeployment(ctx, id)
		} else {
			err = dm.StopDeployment(ctx, id)
		}
		if err != nil {
			return err
		}
		return database.DB.WithContext(ctx).Model(&database.Deployment{}).Where("id = ? AND desired_state = 'stopped'", id).Update("status", int32(deploymentsv1.DeploymentStatus_STOPPED)).Error
	default:
		return fmt.Errorf("deployment has no recovery intent")
	}
}

func (dm *DeploymentManager) ReconcileDeploymentIntent(ctx context.Context, id string) error {
	return withDeploymentRuntimeLock(ctx, id, func(locked context.Context) error { return dm.ReconcileDeploymentIntentLocked(locked, id) })
}

// Read durable intent under the runtime lock before materializing a queued
// operation. Empty intent remains compatible with legacy records.
func deploymentIntentAllows(ctx context.Context, id, desired string) error {
	if database.DB == nil {
		return nil
	}
	var dep database.Deployment
	if err := database.DB.WithContext(ctx).Select("desired_state").Where("id = ? AND deleted_at IS NULL", id).First(&dep).Error; err != nil {
		return err
	}
	if dep.DesiredState != "" && dep.DesiredState != desired {
		return fmt.Errorf("deployment operation superseded by %s intent", dep.DesiredState)
	}
	return nil
}
