package orchestrator

import "time"

// ContainerStats represents statistics for a container
type ContainerStats struct {
	CPUUsage           float64
	MemoryUsage        int64
	NetworkRxBytes     int64
	NetworkTxBytes     int64
	DiskReadBytes      int64
	DiskWriteBytes     int64

	// Raw values from Docker stats (used to compute CPU deltas when precpu_stats is not provided)
	RawCPUUsageTotal   uint64
	RawSystemUsage     uint64
	OnlineCPUs         uint
	// LastSeen is the timestamp when these raw stats were observed
	LastSeen           time.Time
}

