package docker

import (
	"strings"

	"github.com/docker/docker/pkg/stringid"
	godocker "github.com/fsouza/go-dockerclient"
)

//BuildStats builds Stats with the given information
func BuildStats(container godocker.APIContainers, stats *godocker.Stats) *Stats {
	s := &Stats{
		CID:     stringid.TruncateID(container.ID),
		Command: container.Command,
		Stats:   stats,
	}
	s.CPUPercentage = calculateCPUPercent(stats)
	br, bw := calculateBlockIO(stats)
	s.BlockRead = float64(br)
	s.BlockWrite = float64(bw)
	s.Memory = float64(stats.MemoryStats.Usage)
	s.MemoryLimit = float64(stats.MemoryStats.Limit)
	s.MemoryPercentage = calculateMemPercentage(stats)
	s.NetworkRx, s.NetworkTx = calculateNetwork(stats)
	return s
}

func calculateCPUPercent(stats *godocker.Stats) float64 {
	previousCPU := stats.PreCPUStats.CPUUsage.TotalUsage
	previousSystem := stats.PreCPUStats.SystemCPUUsage
	var (
		cpuPercent = 0.0
		// calculate the change for the cpu usage of the container in between readings
		cpuDelta = float64(stats.CPUStats.CPUUsage.TotalUsage - previousCPU)
		// calculate the change for the entire system between readings
		systemDelta = float64(stats.CPUStats.SystemCPUUsage - previousSystem)
	)

	if systemDelta > 0.0 && cpuDelta > 0.0 {
		cpuPercent = (cpuDelta / systemDelta) * float64(len(stats.CPUStats.CPUUsage.PercpuUsage)) * 100.0
	}
	return cpuPercent
}

func calculateMemPercentage(stats *godocker.Stats) float64 {
	// MemoryStats.Limit will never be 0 unless the container is not running and we havn't
	// got any data from cgroup
	if stats.MemoryStats.Limit != 0 {
		return float64(stats.MemoryStats.Usage) / float64(stats.MemoryStats.Limit) * 100.0
	}
	return 0.0
}

func calculateBlockIO(stats *godocker.Stats) (blkRead uint64, blkWrite uint64) {
	blkio := stats.BlkioStats
	for _, bioEntry := range blkio.IOServiceBytesRecursive {
		switch strings.ToLower(bioEntry.Op) {
		case "read":
			blkRead = blkRead + bioEntry.Value
		case "write":
			blkWrite = blkWrite + bioEntry.Value
		}
	}
	return
}

func calculateNetwork(stats *godocker.Stats) (float64, float64) {
	networks := stats.Networks
	var rx, tx float64
	for _, v := range networks {
		rx += float64(v.RxBytes)
		tx += float64(v.TxBytes)
	}
	return rx, tx

}
