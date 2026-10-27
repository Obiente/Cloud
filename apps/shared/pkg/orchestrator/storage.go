package orchestrator

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/moby/moby/client"
	"github.com/obiente/cloud/apps/shared/pkg/database"
	"github.com/obiente/cloud/apps/shared/pkg/docker"
	"gorm.io/gorm"
	"time"
)

func MeasureDeploymentStorage(ctx context.Context, node, id string) error {
	var ids []string
	if err := database.DB.WithContext(ctx).Model(&database.DeploymentLocation{}).Where("deployment_id = ? AND node_id = ? AND status IN ?", id, node, []string{"running", "stopped"}).Pluck("container_id", &ids).Error; err != nil {
		return err
	}
	var previous database.DeploymentStorageMeasurement
	if err := database.DB.WithContext(ctx).Where("deployment_id = ? AND node_id = ?", id, node).First(&previous).Error; err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return err
	}
	var retained []docker.StorageMount
	if previous.Mounts != "" {
		if err := json.Unmarshal([]byte(previous.Mounts), &retained); err != nil {
			return err
		}
	}
	info, measureErr := docker.CalculateStorageWithRetainedMounts(ctx, ids, retained, database.DeploymentStorageInventoryMatches(previous, ids))
	var mounts []string
	if info != nil && info.InventoryCaptured {
		if info.Mounts == nil {
			info.Mounts = []docker.StorageMount{}
		}
		encoded, err := json.Marshal(info.Mounts)
		if err != nil {
			return err
		}
		mounts = []string{string(encoded)}
	}
	bytes := int64(0)
	if measureErr == nil {
		bytes = info.TotalStorage
	}
	// Persist failure even when the probe exhausted its deadline.
	persistCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 5*time.Second)
	defer cancel()
	return errors.Join(measureErr, database.RecordDeploymentStorage(persistCtx, database.DB, id, node, ids, bytes, measureErr, mounts...))
}

func (dm *DeploymentManager) captureStorageBeforeStop(ctx context.Context, id string) {
	if database.DB == nil {
		return
	}
	captureCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	var locations []database.DeploymentLocation
	if err := database.DB.WithContext(captureCtx).Where("deployment_id = ? AND node_id = ? AND status IN ?", id, dm.nodeID, []string{"running", "stopped"}).Find(&locations).Error; err != nil || len(locations) == 0 {
		return
	}
	var ids []string
	for _, location := range locations {
		ids = append(ids, location.ContainerID)
	}
	var previous database.DeploymentStorageMeasurement
	if err := database.DB.WithContext(captureCtx).Where("deployment_id = ? AND node_id = ?", id, dm.nodeID).First(&previous).Error; err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return
	}
	if dm.stoppedStorageInventoryUnchanged(captureCtx, locations, previous) {
		return
	}
	mounts, err := docker.CaptureStorageMounts(captureCtx, ids)
	if err != nil {
		return
	}
	// Preserve storage retained from earlier revisions as well as current mounts.
	if previous.Mounts != "" {
		var retained []docker.StorageMount
		if err := json.Unmarshal([]byte(previous.Mounts), &retained); err != nil {
			return
		}
		mounts = append(mounts, retained...)
	}
	encoded, err := json.Marshal(mounts)
	if err != nil {
		return
	}
	// A later probe measures these mounts. A metadata capture is never billable.
	_ = database.RecordDeploymentStorage(captureCtx, database.DB, id, dm.nodeID, ids, 0, errors.New("measurement pending after stop"), string(encoded))
}

// Repeated stop reconciliation must not invalidate a measurement made after the
// first stop. Container identities also fix their mount configuration.
func (dm *DeploymentManager) stoppedStorageInventoryUnchanged(ctx context.Context, locations []database.DeploymentLocation, previous database.DeploymentStorageMeasurement) bool {
	if previous.Mounts == "" || previous.Error != "" || dm.dockerClient == nil || len(locations) == 0 {
		return false
	}
	var ids []string
	for _, location := range locations {
		if location.Status != "stopped" {
			return false
		}
		ids = append(ids, location.ContainerID)
	}
	if !database.DeploymentStorageInventoryMatches(previous, ids) {
		return false
	}
	for _, id := range ids {
		info, err := dm.dockerClient.ContainerInspect(ctx, id, client.ContainerInspectOptions{})
		if err != nil || info.Container.State == nil || info.Container.State.Running || info.Container.State.Restarting || info.Container.State.Paused {
			return false
		}
	}
	return true
}
