package docker

import (
	"github.com/json-iterator/go"
	"github.com/pkg/errors"
	"io"
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
			ctx, cancel := context.WithCancel(context.Background())
			containerStats, err := daemon.client.ContainerStats(ctx, container.Names[0], true)
			responseBody := containerStats.Body
			defer responseBody.Close()
			defer close(stats)
			if err != nil {
				stats <- &Stats{
					Error: errors.Wrapf(err, "Error creating stats stream for container %s", container.ID)}
				return
			}

			var statsJSON types.StatsJSON
			dec := jsoniter.NewDecoder(responseBody)

			timer := time.NewTicker(1000 * time.Millisecond)
			defer timer.Stop()
			for {
				select {
				case <-timer.C:
					if err := dec.Decode(&statsJSON); err != nil {
						if err == io.EOF {
							return
						}
						continue
					}

					if top, err := daemon.Top(ctx, container.ID); err == nil {
						stats <- buildStats(daemon.version, container, &statsJSON, &top)
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
func buildStats(version *types.Version, container *Container, stats *types.StatsJSON, topResult *container.ContainerTopOKBody) *Stats {
	s := &Stats{
		CID:         TruncateID(container.ID),
		Command:     container.Command,
		Stats:       stats,
		ProcessList: topResult,
	}

	var (
		memPercent, cpuPercent float64
		blkRead, blkWrite      uint64 // Only used on Linux
		mem, memLimit          float64
		pidsStatsCurrent       uint64
	)

	if version.Os != "windows" {

		cpuPercent = calculateCPUPercentUnix(stats)
		blkRead, blkWrite = calculateBlockIO(stats.BlkioStats)
		mem = calculateMemUsageUnixNoCache(stats.MemoryStats)
		memLimit = float64(stats.MemoryStats.Limit)
		memPercent = calculateMemPercentUnixNoCache(memLimit, mem)
		pidsStatsCurrent = stats.PidsStats.Current
	} else {
		cpuPercent = calculateCPUPercentWindows(stats)
		blkRead = stats.StorageStats.ReadSizeBytes
		blkWrite = stats.StorageStats.WriteSizeBytes
		mem = float64(stats.MemoryStats.PrivateWorkingSet)
	}

	s.CPUPercentage = cpuPercent
	s.Memory = mem
	s.MemoryLimit = memLimit
	s.MemoryPercentage = memPercent
	s.NetworkRx, s.NetworkTx = calculateNetwork(stats)
	s.BlockRead = float64(blkRead)
	s.BlockWrite = float64(blkWrite)
	s.PidsCurrent = pidsStatsCurrent
	return s
}

func calculateBlockIO(blkio types.BlkioStats) (blkRead uint64, blkWrite uint64) {
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
	if limit != 0 {
		return usedNoCache / limit * 100.0
	}
	return 0
}

func calculateCPUPercentUnix(stats *types.StatsJSON) float64 {
	previousCPU := stats.PreCPUStats.CPUUsage.TotalUsage
	previousSystem := stats.PreCPUStats.SystemUsage
	var (
		cpuPercent = 0.0
		// calculate the change for the cpu usage of the container in between readings
		cpuDelta = float64(stats.CPUStats.CPUUsage.TotalUsage) - float64(previousCPU)
		// calculate the change for the entire system between readings
		systemDelta = float64(stats.CPUStats.SystemUsage) - float64(previousSystem)
		onlineCPUs  = float64(stats.CPUStats.OnlineCPUs)
	)

	if onlineCPUs == 0.0 {
		onlineCPUs = float64(len(stats.CPUStats.CPUUsage.PercpuUsage))
	}
	if systemDelta > 0.0 && cpuDelta > 0.0 {
		cpuPercent = (cpuDelta / systemDelta) * onlineCPUs * 100.0
	}
	return cpuPercent
}

func calculateCPUPercentWindows(v *types.StatsJSON) float64 {
	// Max number of 100ns intervals between the previous time read and now
	possIntervals := uint64(v.Read.Sub(v.PreRead).Nanoseconds()) // Start with number of ns intervals
	possIntervals /= 100                                         // Convert to number of 100ns intervals
	possIntervals *= uint64(v.NumProcs)                          // Multiple by the number of processors

	// Intervals used
	intervalsUsed := v.CPUStats.CPUUsage.TotalUsage - v.PreCPUStats.CPUUsage.TotalUsage

	// Percentage avoiding divide-by-zero
	if possIntervals > 0 {
		return float64(intervalsUsed) / float64(possIntervals) * 100.0
	}
	return 0.00
}
