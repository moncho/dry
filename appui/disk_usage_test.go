package appui

import (
	"flag"
	"io/ioutil"
	"path/filepath"
	"testing"

	"github.com/docker/docker/api/types"
	"github.com/moncho/dry/docker"
)

const (
	screenHeight = 20
)

var update = flag.Bool("update", false, "update .golden files")

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
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			r := NewDockerDiskUsageRenderer(screenHeight)

			r.PrepareToRender(tt.args.diskUsage, tt.args.pruneReport)
			actual := r.Render()

			golden := filepath.Join("testdata", tt.name+".golden")
			if *update {
				ioutil.WriteFile(golden, []byte(actual), 0644)
			}
			expected, _ := ioutil.ReadFile(golden)
			if string(expected) != actual {
				t.Errorf("DockerDiskUsageRenderer.Render(), got: \n%v\nWant: \n%s", actual, expected)
			}
		})
	}
}
