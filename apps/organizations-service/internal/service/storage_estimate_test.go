package organizations

import (
	organizationsv1 "github.com/obiente/cloud/apps/shared/proto/obiente/cloud/organizations/v1"
	"testing"
)

func TestUnavailableStorageDoesNotPublishAPartialPrice(t *testing.T) {
	storageCost := int64(400)
	metrics := &organizationsv1.UsageMetrics{StorageBytes: 8192, StorageCostCents: &storageCost, EstimatedCostCents: 1000, CpuCoreSeconds: 3600}
	setStorageEstimateAvailability(metrics, false)
	if !metrics.StorageMeasurementUnavailable || !metrics.CostEstimateUnavailable || metrics.StorageCostCents != nil || metrics.EstimatedCostCents != 0 || metrics.StorageBytes != 0 || metrics.CpuCoreSeconds != 3600 {
		t.Fatal("unknown storage appeared as a valid price or destroyed independent usage")
	}
	metrics.StorageBytes = 8192
	metrics.StorageCostCents = &storageCost
	metrics.EstimatedCostCents = 1000
	setStorageEstimateAvailability(metrics, true)
	if metrics.StorageMeasurementUnavailable || metrics.CostEstimateUnavailable || metrics.EstimatedCostCents != 1000 {
		t.Fatal("recovered measurement did not restore the estimate")
	}
}
