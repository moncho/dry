package docker

import "github.com/docker/docker/api/types"

// PruneReport represents the result of a prune operation
type PruneReport struct {
	ContainerReport types.ContainersPruneReport
	ImagesReport    types.ImagesPruneReport
	NetworksReport  types.NetworksPruneReport
	VolumesReport   types.VolumesPruneReport
}

// TotalSpaceReclaimed reports the total space reclaimed
func (p *PruneReport) TotalSpaceReclaimed() uint64 {
	total := p.ContainerReport.SpaceReclaimed
	total += p.ImagesReport.SpaceReclaimed
	total += p.VolumesReport.SpaceReclaimed
	return total
}
