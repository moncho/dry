package docker

import (
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/api/types/volume"
)

// PruneReport represents the result of a prune operation
type PruneReport struct {
	ContainerReport container.PruneReport
	ImagesReport    image.PruneReport
	NetworksReport  network.PruneReport
	VolumesReport   volume.PruneReport
}

// TotalSpaceReclaimed reports the total space reclaimed
func (p *PruneReport) TotalSpaceReclaimed() uint64 {
	total := p.ContainerReport.SpaceReclaimed
	total += p.ImagesReport.SpaceReclaimed
	total += p.VolumesReport.SpaceReclaimed
	return total
}
