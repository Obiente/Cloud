package pricing

import (
	"fmt"
	"os"
	"strconv"
)

// PricingModel defines the cost structure for resource usage
type PricingModel struct {
	// CPU cost per core-second in dollars
	CPUCostPerCoreSecond float64
	// Memory cost per byte-second in dollars
	MemoryCostPerByteSecond float64
	// Bandwidth cost per byte in dollars
	BandwidthCostPerByte float64
	// Storage cost per byte-month in dollars
	StorageCostPerByteMonth float64
}

	var (
		// Default pricing model (fallback if env vars not set)
		// Competitive VPS-like pricing: ~$5/month for 1GB RAM + 1 CPU core running 24/7
		// Storage pricing is higher to reflect limited capacity (not a storage provider)
		defaultPricing = PricingModel{
			CPUCostPerCoreSecond:    0.000000761,        // $0.000000761 per core-second = $0.00274 per core-hour = $2.00/month for 1 core 24/7
			MemoryCostPerByteSecond: 0.000000000000001063, // $0.000000000000001063 per byte-second = $0.00411 per GB-hour = $3.00/month for 1GB 24/7
			BandwidthCostPerByte:    0.000000000009313, // $0.000000000009313 per byte = $0.01 per GB (very cheap, like VPS)
			StorageCostPerByteMonth: 0.000000000186264, // $0.000000000186264 per byte-month = $0.20 per GB-month (higher due to limited capacity)
		}

	// Singleton instance
	globalPricing *PricingModel
)

// GetPricing returns the global pricing model, initializing it from env vars if needed
func GetPricing() *PricingModel {
	if globalPricing == nil {
		globalPricing = loadPricingFromEnv()
	}
	return globalPricing
}

// loadPricingFromEnv loads pricing from environment variables with defaults
func loadPricingFromEnv() *PricingModel {
	pricing := defaultPricing

	// Load from environment variables if set
	if cpuCostStr := os.Getenv("PRICING_CPU_COST_PER_CORE_SECOND"); cpuCostStr != "" {
		if val, err := strconv.ParseFloat(cpuCostStr, 64); err == nil {
			pricing.CPUCostPerCoreSecond = val
		}
	}

	if memCostStr := os.Getenv("PRICING_MEMORY_COST_PER_BYTE_SECOND"); memCostStr != "" {
		if val, err := strconv.ParseFloat(memCostStr, 64); err == nil {
			pricing.MemoryCostPerByteSecond = val
		}
	}

	if bwCostStr := os.Getenv("PRICING_BANDWIDTH_COST_PER_BYTE"); bwCostStr != "" {
		if val, err := strconv.ParseFloat(bwCostStr, 64); err == nil {
			pricing.BandwidthCostPerByte = val
		}
	}

	if storageCostStr := os.Getenv("PRICING_STORAGE_COST_PER_BYTE_MONTH"); storageCostStr != "" {
		if val, err := strconv.ParseFloat(storageCostStr, 64); err == nil {
			pricing.StorageCostPerByteMonth = val
		}
	}

	return &pricing
}

// CalculateCPUCost calculates CPU cost in cents
// cpuCoreSeconds: total CPU core-seconds used
func (p *PricingModel) CalculateCPUCost(cpuCoreSeconds int64) int64 {
	return int64(float64(cpuCoreSeconds) * p.CPUCostPerCoreSecond * 100)
}

// CalculateMemoryCost calculates memory cost in cents
// memoryByteSeconds: total memory byte-seconds used
func (p *PricingModel) CalculateMemoryCost(memoryByteSeconds int64) int64 {
	return int64(float64(memoryByteSeconds) * p.MemoryCostPerByteSecond * 100)
}

// CalculateBandwidthCost calculates bandwidth cost in cents
// bandwidthBytes: total bandwidth bytes (rx + tx)
func (p *PricingModel) CalculateBandwidthCost(bandwidthBytes int64) int64 {
	return int64(float64(bandwidthBytes) * p.BandwidthCostPerByte * 100)
}

// CalculateStorageCost calculates storage cost in cents
// storageBytes: total storage bytes
func (p *PricingModel) CalculateStorageCost(storageBytes int64) int64 {
	return int64(float64(storageBytes) * p.StorageCostPerByteMonth * 100)
}

// CalculateTotalCost calculates total cost in cents for all resources
func (p *PricingModel) CalculateTotalCost(cpuCoreSeconds, memoryByteSeconds, bandwidthBytes, storageBytes int64) int64 {
	cpuCost := p.CalculateCPUCost(cpuCoreSeconds)
	memoryCost := p.CalculateMemoryCost(memoryByteSeconds)
	bandwidthCost := p.CalculateBandwidthCost(bandwidthBytes)
	storageCost := p.CalculateStorageCost(storageBytes)
	return cpuCost + memoryCost + bandwidthCost + storageCost
}

// GetPricingInfo returns a human-readable description of the pricing model
func (p *PricingModel) GetPricingInfo() string {
	return fmt.Sprintf(
		"CPU: $%.6f/core-second, Memory: $%.10f/byte-second, Bandwidth: $%.10f/byte, Storage: $%.10f/byte-month",
		p.CPUCostPerCoreSecond,
		p.MemoryCostPerByteSecond,
		p.BandwidthCostPerByte,
		p.StorageCostPerByteMonth,
	)
}
