package docker

import (
	"encoding/json"
	"strings"
	"time"

	"golang.org/x/net/context"

	"github.com/docker/docker/pkg/stringid"
	"github.com/docker/engine-api/client"
	"github.com/docker/engine-api/types"
)

//StatsStream creates a channel on which to receive the stats of the given container
func StatsStream(cli client.APIClient, container types.Container, streamStats bool) (<-chan *Stats, chan<- struct{}) {
	stats := make(chan *Stats)
	done := make(chan struct{})

	go func() {
		responseBody, err := cli.ContainerStats(context.Background(), container.Names[0], streamStats)
		defer responseBody.Close()
		defer close(stats)
		if err != nil {
			return
		}

		var statsJSON *types.StatsJSON
		dec := json.NewDecoder(responseBody)

		if err := dec.Decode(&statsJSON); err != nil {
			return
		}
		timer := time.NewTicker(1000 * time.Millisecond)
		for {
			select {
			case <-timer.C:
				if err := dec.Decode(&statsJSON); err != nil {
					return
				}

				stats <- buildStats(container, statsJSON)
			case <-done:
				return
			}
		}
	}()
	return stats, done
}

//buildStats builds Stats with the given information
func buildStats(container types.Container, stats *types.StatsJSON) *Stats {
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

	var memPercent = 0.0
	var cpuPercent = 0.0

	// MemoryStats.Limit will never be 0 unless the container is not running and we haven't
	// got any data from cgroup
	if stats.MemoryStats.Limit != 0 {
		memPercent = float64(stats.MemoryStats.Usage) / float64(stats.MemoryStats.Limit) * 100.0
	}

	cpuPercent = calculateCPUPercent(stats)
	blkRead, blkWrite := calculateBlockIO(stats)
	s.CPUPercentage = cpuPercent
	s.Memory = float64(stats.MemoryStats.Usage)
	s.MemoryLimit = float64(stats.MemoryStats.Limit)
	s.MemoryPercentage = memPercent
	s.NetworkRx, s.NetworkTx = calculateNetwork(stats)
	s.BlockRead = float64(blkRead)
	s.BlockWrite = float64(blkWrite)
	s.PidsCurrent = stats.PidsStats.Current
	return s
}

func calculateCPUPercent(stats *types.StatsJSON) float64 {
	previousCPU := stats.PreCPUStats.CPUUsage.TotalUsage
	previousSystem := stats.PreCPUStats.SystemUsage
	var (
		cpuPercent = 0.0
		// calculate the change for the cpu usage of the container in between readings
		cpuDelta = float64(stats.CPUStats.CPUUsage.TotalUsage - previousCPU)
		// calculate the change for the entire system between readings
		systemDelta = float64(stats.CPUStats.SystemUsage - previousSystem)
	)

	if systemDelta > 0.0 && cpuDelta > 0.0 {
		cpuPercent = (cpuDelta / systemDelta) * float64(len(stats.CPUStats.CPUUsage.PercpuUsage)) * 100.0
	}
	return cpuPercent
}

func calculateMemPercentage(stats *types.StatsJSON) float64 {
	// MemoryStats.Limit will never be 0 unless the container is not running and we havn't
	// got any data from cgroup
	if stats.MemoryStats.Limit != 0 {
		return float64(stats.MemoryStats.Usage) / float64(stats.MemoryStats.Limit) * 100.0
	}
	return 0.0
}

func calculateBlockIO(stats *types.StatsJSON) (blkRead uint64, blkWrite uint64) {
	blkio := stats.BlkioStats
	for _, bioEntry := range blkio.IoServiceBytesRecursive {
		switch strings.ToLower(bioEntry.Op) {
		case "read":
			blkRead = blkRead + bioEntry.Value
		case "write":
			blkWrite = blkWrite + bioEntry.Value
		}
	}
	return
}

func calculateNetwork(stats *types.StatsJSON) (float64, float64) {
	networks := stats.Networks
	var rx, tx float64
	for _, v := range networks {
		rx += float64(v.RxBytes)
		tx += float64(v.TxBytes)
	}
	return rx, tx

}
