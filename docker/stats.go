package docker

import (
	"context"
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
	jsoniter "github.com/json-iterator/go"
)

// StatsChannel manages the stats channel of a container
type StatsChannel struct {
	Container *Container
	version   *types.Version
	client    client.ContainerAPIClient
}

// Start starts sending stats to the channel returned
func (s *StatsChannel) Start(ctx context.Context) <-chan *Stats {
	stats := make(chan *Stats)

	go func() {
		defer close(stats)
		containerStats, err := s.client.ContainerStats(ctx, s.Container.Names[0], true)

		if err != nil {
			nonBlockingSend(stats, &Stats{
				Error: fmt.Errorf("create stats stream for container %s: %w", s.Container.ID, err)})
			return
		}

		responseBody := containerStats.Body
		defer responseBody.Close()

		var statsJSON types.StatsJSON
		dec := jsoniter.NewDecoder(responseBody)
	loop:
		for {
			select {
			default:
				if err := dec.Decode(&statsJSON); err != nil {
					if err == io.EOF {
						nonBlockingSend(stats, &Stats{
							Error: fmt.Errorf("end of stats stream reached for container %s", s.Container.ID)})
					} else {
						nonBlockingSend(stats, &Stats{
							Error: fmt.Errorf("read stats for container %s: %w", s.Container.ID, err)})
					}
					break loop
				}

				top, err := s.client.ContainerTop(ctx, s.Container.ID, nil)

				if err != nil {
					nonBlockingSend(stats, &Stats{
						Error: fmt.Errorf("retrieve top info for container %s: %w", s.Container.ID, err)})
					break loop
				}
				nonBlockingSend(stats, buildStats(s.version, s.Container, &statsJSON, &top))
			case <-ctx.Done():
				break loop
			}
		}

	}()
	return stats
}

// newStatsChannel creates a ready to use stats channel
func newStatsChannel(version *types.Version, client client.ContainerAPIClient, container *Container) (*StatsChannel, error) {
	if container == nil {
		return nil, errors.New("Container cannot be null")
	} else if !IsContainerRunning(container) {
		return nil, fmt.Errorf("Container %s is not running", container.ID)
	}
	return &StatsChannel{
		client:    client,
		Container: container,
		version:   version,
	}, nil

}

// buildStats builds Stats with the given information
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
			blkRead += bioEntry.Value
		case "write":
			blkWrite += bioEntry.Value
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

func nonBlockingSend(stats chan<- *Stats, s *Stats) {
	select {
	case stats <- s:
	default:
	}
}
