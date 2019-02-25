package docker

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"io/ioutil"
	"strings"
	"sync"
	"testing"

	"github.com/docker/docker/api/types"
	containertypes "github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
	"go.uber.org/goleak"
)

type statsClientMock struct {
	client.ContainerAPIClient
	statsBody io.ReadCloser
}

func (s statsClientMock) ContainerStats(ctx context.Context, id string, stream bool) (types.ContainerStats, error) {
	return types.ContainerStats{
		Body: s.statsBody,
	}, nil
}

func (s statsClientMock) ContainerTop(ctx context.Context, container string, arguments []string) (containertypes.ContainerTopOKBody, error) {
	return containertypes.ContainerTopOKBody{}, nil
}

func TestStatsChannel_cancellingContextClosesResources(t *testing.T) {

	sc := StatsChannel{
		Container: &Container{
			types.Container{
				ID:    "1234",
				Names: []string{"1234"},
			},
			types.ContainerJSON{},
		},
		version: &types.Version{
			Os: "Not windows",
		},
		client: statsClientMock{
			statsBody: ioutil.NopCloser(strings.NewReader("")),
		},
		refresh: make(chan struct{})}
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

func TestStatsChannel_refreshingPublishesStats(t *testing.T) {

	sc := StatsChannel{
		Container: &Container{
			types.Container{
				ID:    "1234",
				Names: []string{"1234"},
			},
			types.ContainerJSON{},
		},
		version: &types.Version{
			Os: "Not windows",
		},
		client: statsClientMock{
			statsBody: ioutil.NopCloser(strings.NewReader(asJson(types.StatsJSON{}))),
		},
		//Using a buffered chan in the test so Start goroutine receives the refresh
		//signal
		refresh: make(chan struct{}, 1)}
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
	sc.Refresh()
	wg.Wait()
	cancel()
}

func TestStatsChannel_noErrors_goroutineExistsOnCtxCancel(t *testing.T) {
	defer goleak.VerifyNoLeaks(t)
	sc := StatsChannel{
		Container: &Container{
			types.Container{
				ID:    "1234",
				Names: []string{"1234"},
			},
			types.ContainerJSON{},
		},
		version: &types.Version{
			Os: "Not windows",
		},
		client: statsClientMock{
			statsBody: ioutil.NopCloser(strings.NewReader(asJson(types.StatsJSON{}))),
		},
		//Using a buffered chan in the test so Start goroutine receives the refresh
		//signal
		refresh: make(chan struct{}, 1)}
	ctx, cancel := context.WithCancel(context.Background())
	sc.Refresh()
	sc.Start(ctx)
	cancel()
}

func TestStatsChannel_errorBuildingStats_goroutineExistsOnCtxCancel(t *testing.T) {
	defer goleak.VerifyNoLeaks(t)
	sc := StatsChannel{
		Container: &Container{
			types.Container{
				ID:    "1234",
				Names: []string{"1234"},
			},
			types.ContainerJSON{},
		},
		version: &types.Version{
			Os: "Not windows",
		},
		client: statsClientMock{
			//Empty reader results in EOF error
			statsBody: ioutil.NopCloser(strings.NewReader("")),
		},
		//Using a buffered chan in the test so Start goroutine receives the refresh
		//signal
		refresh: make(chan struct{}, 1)}
	ctx, cancel := context.WithCancel(context.Background())
	sc.Refresh()
	sc.Start(ctx)
	cancel()
}

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

func TestCalculateCPUPercentUnix(t *testing.T) {

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
			result := calculateCPUPercentUnix(test.stats)
			if test.expected != result {
				t.Errorf("Error calculating CPU percent, expected: %f, got: %f ", test.expected, result)
			}
		})
	}

}

func asJson(stats types.StatsJSON) string {
	var buffer bytes.Buffer
	enc := json.NewEncoder(&buffer)
	enc.Encode(stats)

	return buffer.String()
}
