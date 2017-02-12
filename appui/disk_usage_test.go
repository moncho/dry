package appui

import (
	"testing"

	"github.com/docker/docker/api/types"
	"github.com/moncho/dry/docker"
)

const (
	dockerInfoValue = `
Docker Host: test
`
	diskUsageOutput = `
Docker Host: test



<green>TYPE           TOTAL                 ACTIVE                SIZE                  RECLAIMABLE</>

Images              0                   0                   0 B                 0 B
Containers          0                   0                   0 B                 0 B
Local Volumes       0                   0                   0 B                 0 B


Deleted containers: 0 
Deleted images: 0 
Deleted networks: 0 
Deleted volumes: 0 
Total reclaimed space: 0 B 

`

	screenHeight = 20
)

func TestDiskUsageRendererCreation(t *testing.T) {

	r := NewDockerDiskUsageRenderer(dockerInfoValue, screenHeight)

	if r == nil {
		t.Error("DiskUsageRenderer was not created")
	}

	if r.dockerInfo != dockerInfoValue || r.height != screenHeight {
		t.Errorf("DiskUsageRenderer was not initialized correctly: %v", r)
	}
}

func TestDiskUsageRendererRendering(t *testing.T) {
	r := NewDockerDiskUsageRenderer(dockerInfoValue, screenHeight)

	du := &types.DiskUsage{}
	pr := &docker.PruneReport{}
	r.PrepareToRender(du, pr)

	if r.Render() != diskUsageOutput {
		t.Errorf("DiskUsageRenderer render output does not match. Expected: \n'%q'\n, got: \n'%q'", diskUsageOutput, r.Render())
	}
}
