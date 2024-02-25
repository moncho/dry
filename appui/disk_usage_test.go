package appui

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/moncho/dry/docker"
)

const (
	screenHeight = 20
)

func TestDiskUsageRendererCreation(t *testing.T) {

	r := NewDockerDiskUsageRenderer(screenHeight)

	if r == nil {
		t.Error("DiskUsageRenderer was not created")
	}

	if r.height != screenHeight {
		t.Errorf("DiskUsageRenderer was not initialized correctly: %v", r)
	}
}

func TestDockerDiskUsageRenderer_Render(t *testing.T) {
	type args struct {
		diskUsage   *types.DiskUsage
		pruneReport *docker.PruneReport
		timeStamp   string
	}
	tests := []struct {
		name string
		args args
	}{
		{
			"DiskUsageTest_noPruneReport",
			args{
				diskUsage: &types.DiskUsage{},
			},
		},
		{
			"DiskUsageTest",
			args{
				diskUsage:   &types.DiskUsage{},
				pruneReport: &docker.PruneReport{},
				timeStamp:   "1970-Jan-01",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			r := NewDockerDiskUsageRenderer(screenHeight)

			r.PrepareToRender(tt.args.diskUsage, tt.args.pruneReport)
			//Last prune timestamp is manually set for testing
			timeStamp, _ := time.Parse("2006-Jan-02", tt.args.timeStamp)
			r.lastPrune = timeStamp
			actual := r.String()

			golden := filepath.Join("testdata", tt.name+".golden")
			if *update {
				os.WriteFile(golden, []byte(actual), 0644)
			}
			expected, _ := os.ReadFile(golden)
			if string(expected) != actual {
				t.Errorf("DockerDiskUsageRenderer.Render(), got: \n%v\nWant: \n%s", actual, expected)
			}
		})
	}
}
