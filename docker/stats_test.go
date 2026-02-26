package docker

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"strings"
	"sync"
	"testing"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
	"go.uber.org/goleak"
)

type statsClientMock struct {
	client.ContainerAPIClient
	statsBody io.ReadCloser
	statsErr  error
}

func (s statsClientMock) ContainerStats(ctx context.Context, id string, stream bool) (container.StatsResponseReader, error) {
	return container.StatsResponseReader{
		Body: s.statsBody,
	}, s.statsErr
}

func (s statsClientMock) ContainerTop(ctx context.Context, ctr string, arguments []string) (container.TopResponse, error) {
	return container.TopResponse{}, nil
}

func TestStatsChannel_cancellingContextClosesResources(t *testing.T) {

	sc := StatsChannel{
		Container: &Container{
			Summary: container.Summary{
				ID:    "1234",
				Names: []string{"1234"},
			},
		},
		version: &types.Version{
			Os: "Not windows",
		},
		client: statsClientMock{
			statsBody: io.NopCloser(strings.NewReader("")),
		}}
	ctx, cancel := context.WithCancel(context.Background())
	stats := sc.Start(ctx)
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		<-stats
		wg.Done()
	}()
	cancel()
	wg.Wait()
}

func TestStatsChannel_statsArePublished(t *testing.T) {

	sc := StatsChannel{
		Container: &Container{
			Summary: container.Summary{
				ID:    "1234",
				Names: []string{"1234"},
			},
		},
		version: &types.Version{
			Os: "Not windows",
		},
		client: statsClientMock{
			statsBody: io.NopCloser(strings.NewReader(asJSON(container.StatsResponse{}))),
		}}
	ctx, cancel := context.WithCancel(context.Background())
	stats := sc.Start(ctx)
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		s := <-stats
		if s.Error != nil {
			t.Errorf("Expected no errors, got: %s \n", s.Error.Error())
		}
		wg.Done()
	}()
	wg.Wait()
	cancel()
}

func TestStatsChannel_noErrors_goroutineExitsOnCtxCancel(t *testing.T) {
	defer goleak.VerifyNone(t)
	sc := StatsChannel{
		Container: &Container{
			Summary: container.Summary{
				ID:    "1234",
				Names: []string{"1234"},
			},
		},
		version: &types.Version{
			Os: "Not windows",
		},
		client: statsClientMock{
			statsBody: io.NopCloser(strings.NewReader(asJSON(container.StatsResponse{}))),
		}}
	ctx, cancel := context.WithCancel(context.Background())
	sc.Start(ctx)
	cancel()
}

func TestStatsChannel_errorBuildingStats_goroutineExitsOnCtxCancel(t *testing.T) {
	defer goleak.VerifyNone(t)
	sc := StatsChannel{
		Container: &Container{
			Summary: container.Summary{
				ID:    "1234",
				Names: []string{"1234"},
			},
		},
		version: &types.Version{
			Os: "Not windows",
		},
		client: statsClientMock{
			//Empty reader results in EOF error
			statsBody: io.NopCloser(strings.NewReader("")),
		}}
	ctx, cancel := context.WithCancel(context.Background())
	sc.Start(ctx)
	cancel()
}

func TestStatsChannel_errorOpeningStream_goroutineExits(t *testing.T) {
	defer goleak.VerifyNone(t)
	sc := StatsChannel{
		Container: &Container{
			Summary: container.Summary{
				ID:    "1234",
				Names: []string{"1234"},
			},
		},
		version: &types.Version{
			Os: "Not windows",
		},
		client: statsClientMock{
			statsErr: errors.New("No stats for you, my friend"),
		}}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	sc.Start(ctx)
}

func TestCalculateMemUsageUnixNoCache(t *testing.T) {
	stats := container.MemoryStats{Usage: 500, Stats: map[string]uint64{"cache": 400}}
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

func TestCalculateCPUPercentUnix(t *testing.T) {

	tests := []struct {
		name     string
		stats    *container.StatsResponse
		expected float64
	}{
		{
			"CPU percent calculation",
			&container.StatsResponse{
				CPUStats: container.CPUStats{
					CPUUsage: container.CPUUsage{
						TotalUsage:  700,
						PercpuUsage: []uint64{0},
					},
					SystemUsage: 1000,
				},
				PreCPUStats: container.CPUStats{
					CPUUsage: container.CPUUsage{
						TotalUsage: 0,
					},
					SystemUsage: 0,
				},
			},
			70.0,
		},
		{
			"CPU percent calculation, missing data",
			&container.StatsResponse{},
			0,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result := calculateCPUPercentUnix(test.stats)
			if test.expected != result {
				t.Errorf("Error calculating CPU percent, expected: %f, got: %f ", test.expected, result)
			}
		})
	}

}

func asJSON(stats container.StatsResponse) string {
	var buffer bytes.Buffer
	enc := json.NewEncoder(&buffer)
	enc.Encode(stats)

	return buffer.String()
}
