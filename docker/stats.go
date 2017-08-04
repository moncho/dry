package docker

import (
	"encoding/json"
	"strings"
	"time"

	"golang.org/x/net/context"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
)

//StatsChannel is a container and its stats channel.
//If the container is not running stats and done channel are nil.
type StatsChannel struct {
	Container *Container
	Stats     <-chan *Stats
	Done      chan<- struct{}
}

//NewStatsChannel creates a channel on which to receive the runtime stats of the given container
func NewStatsChannel(daemon *DockerDaemon, container *Container) *StatsChannel {
	if IsContainerRunning(container) {
		stats := make(chan *Stats)
		done := make(chan struct{})

		go func() {
			cli := daemon.client
			ctx, cancel := context.WithCancel(context.Background())
			containerStats, err := cli.ContainerStats(ctx, container.Names[0], true)
			responseBody := containerStats.Body
			defer responseBody.Close()
			defer close(stats)
			if err != nil {
				return
			}

			var statsJSON *types.StatsJSON
			dec := json.NewDecoder(responseBody)

			timer := time.NewTicker(1000 * time.Millisecond)
			for {
				select {
				case <-timer.C:
					if err := dec.Decode(&statsJSON); err != nil {
						return
					}
					if statsJSON != nil {
						top, _ := daemon.Top(container.ID)
						stats <- buildStats(container, statsJSON, &top)
					}
				case <-ctx.Done():
					return
				case <-done:
					cancel()
					return
				}
			}
		}()

		return &StatsChannel{container, stats, done}
	}
	return &StatsChannel{Container: container}

}

//buildStats builds Stats with the given information
func buildStats(container *Container, stats *types.StatsJSON, topResult *container.ContainerTopOKBody) *Stats {
	s := &Stats{
		CID:         TruncateID(container.ID),
		Command:     container.Command,
		Stats:       stats,
		ProcessList: topResult,
	}
	s.CPUPercentage = calculateCPUPercent(stats)
	br, bw := calculateBlockIO(stats)
	s.BlockRead = float64(br)
	s.BlockWrite = float64(bw)
	s.NetworkRx, s.NetworkTx = calculateNetwork(stats)

	mem := calculateMemUsageUnixNoCache(stats.MemoryStats)
	memLimit := float64(stats.MemoryStats.Limit)
	cpuPercent := calculateCPUPercent(stats)
	blkRead, blkWrite := calculateBlockIO(stats)
	s.CPUPercentage = cpuPercent
	s.Memory = mem
	s.MemoryLimit = memLimit
	s.MemoryPercentage = calculateMemPercentUnixNoCache(memLimit, mem)
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

func calculateMemUsageUnixNoCache(mem types.MemoryStats) float64 {
	return float64(mem.Usage - mem.Stats["cache"])
}

func calculateMemPercentUnixNoCache(limit float64, usedNoCache float64) float64 {
	// MemoryStats.Limit will never be 0 unless the container is not running and we haven't
	// got any data from cgroup
	if limit != 0 {
		return usedNoCache / limit * 100.0
	}
	return 0
}
