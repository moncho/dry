package docker

import (
	"testing"

	"github.com/docker/docker/api/types"
)

func TestCalculateMemUsageUnixNoCache(t *testing.T) {
	stats := types.MemoryStats{Usage: 500, Stats: map[string]uint64{"cache": 400}}
	result := calculateMemUsageUnixNoCache(stats)
	if 100.0 != result {
		t.Errorf("Error calculating Unix mem usage, expected: %f, got: %f ", 100.0, result)
	}
}

func TestCalculateMemPercentUnixNoCache(t *testing.T) {

	tests := []struct {
		name     string
		limit    float64
		used     float64
		expected float64
	}{
		{
			"Limit is set",
			100.0,
			70.0,
			70.0,
		},
		{
			"No limit, no cgroup data",
			0.0,
			70.0,
			0.0,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result := calculateMemPercentUnixNoCache(test.limit, test.used)
			if test.expected != result {
				t.Errorf("Error calculating Unix mem percent, expected: %f, got: %f ", test.expected, result)
			}
		})
	}

}

func TestCalculateCPUPercent(t *testing.T) {

	tests := []struct {
		name     string
		stats    *types.StatsJSON
		expected float64
	}{
		{
			"CPU percent calculation",
			&types.StatsJSON{
				Stats: types.Stats{
					CPUStats: types.CPUStats{
						CPUUsage: types.CPUUsage{
							TotalUsage:  700,
							PercpuUsage: []uint64{0},
						},
						SystemUsage: 1000,
					},
					PreCPUStats: types.CPUStats{
						CPUUsage: types.CPUUsage{
							TotalUsage: 0,
						},
						SystemUsage: 0,
					},
				},
			},
			70.0,
		},
		{
			"CPU percent calculation, missing data",
			&types.StatsJSON{
				Stats: types.Stats{},
			},
			0,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result := calculateCPUPercent(test.stats)
			if test.expected != result {
				t.Errorf("Error calculating CPU percent, expected: %f, got: %f ", test.expected, result)
			}
		})
	}

}
