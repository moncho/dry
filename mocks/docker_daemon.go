package mocks

import (
	"context"
	"encoding/json"
	"io"
	"strconv"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/events"
	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/api/types/swarm"
	"github.com/docker/docker/api/types/volume"
	drydocker "github.com/moncho/dry/docker"
)

// DockerDaemonMock mocks a DockerDaemon
type DockerDaemonMock struct {
}

// ContainerByID mock
func (_m *DockerDaemonMock) ContainerByID(id string) *drydocker.Container {
	return nil
}

// Containers mock
func (_m *DockerDaemonMock) Containers(filters []drydocker.ContainerFilter, mode drydocker.SortMode) []*drydocker.Container {

	var containers []*drydocker.Container
	for index := 0; index < 10; index++ {
		containers = append(containers, &drydocker.Container{
			Container: types.Container{ID: strconv.Itoa(index), Names: []string{"Name"},
				Status: "Up and running"},
		})
	}
	for index := 0; index < 10; index++ {
		containers = append(containers, &drydocker.Container{
			Container: types.Container{ID: strconv.Itoa(index), Names: []string{"Name"},
				Status: "Never worked"},
		})
	}
	for _, filter := range filters {
		containers = filter.Apply(containers)
	}
	return containers
}

// DiskUsage mock
func (_m *DockerDaemonMock) DiskUsage() (types.DiskUsage, error) {
	return types.DiskUsage{}, nil
}

// DockerEnv provides a mock function with given fields:
func (_m *DockerDaemonMock) DockerEnv() drydocker.Env {
	return drydocker.Env{DockerHost: "dry.io", DockerTLSVerify: false, DockerCertPath: ""}
}

// Events provides a mock function with given fields:
func (_m *DockerDaemonMock) Events() (<-chan events.Message, chan<- struct{}, error) {

	return nil, nil, nil
}

// EventLog mock
func (_m *DockerDaemonMock) EventLog() *drydocker.EventLog {
	return nil
}

// History mock
func (_m *DockerDaemonMock) History(id string) ([]image.HistoryResponseItem, error) {
	return nil, nil
}

// ImageByID mock
func (_m *DockerDaemonMock) ImageByID(id string) (types.ImageSummary, error) {
	return types.ImageSummary{}, nil
}

// Images mock
func (_m *DockerDaemonMock) Images() ([]types.ImageSummary, error) {

	imagesJSON := `[
		 {
						 "ID": "8dfafdbc3a40",
						  "RepoTags":["dry/dry:1"],
							"Created": 1367854155,
							"Size": 100,
							"VirtualSize": 1000,
							"ParentID": "7dfafdbc3a40",
							"RepoDigests": ["8dfafdbc3a40"]
		 },
		 {
						 "ID": "541a0f4efc6f",
						  "RepoTags":["dry/dry:2"],
							"Created": 1367854155,
							"Size": 100,
							"VirtualSize": 1000,
							"ParentID": "7dfafdbc3a40",
							"RepoDigests": ["541a0f4efc6f"]
		 },
		 {
						 "ID": "26380e1ca356",
						  "RepoTags":["dry/dry:3"],
							"Created": 1367854155,
							"Size": 100,
							"VirtualSize": 1000,
							"ParentID": "7dfafdbc3a40",
							"RepoDigests": ["26380e1ca356"]
		 },
		 {
						 "ID": "a3d6e836e86a",
						  "RepoTags":["examplevotingapp_result-app:latest"],
							"Created": 1367844155,
							"Size": 0,
							"VirtualSize": 54543,
							"ParentID": "7dfafdbc3a40",
							"RepoDigests": ["a3d6e836e86a"]
		 },
		 {
						 "ID": "03b4557ad7b9",
						  "RepoTags":["dry/dry:4"],
							"Created": 136123255,
							"Size": 43985,
							"VirtualSize": 1343400,
							"ParentID": "7dfafdbc3a40",
							"RepoDigests": ["03b4557ad7b9"]
		 }
	]`
	var images []types.ImageSummary
	err := json.Unmarshal([]byte(imagesJSON), &images)
	return images, err
}

// ImagesCount mock
func (_m *DockerDaemonMock) ImagesCount() int {
	i, _ := _m.Images()
	return len(i)
}

// Info provides a mock function with given fields:
func (_m *DockerDaemonMock) Info() (types.Info, error) {
	swarmInfo := swarm.Info{LocalNodeState: swarm.LocalNodeStateInactive}
	return types.Info{
		Name:     "test",
		NCPU:     2,
		MemTotal: 1024,
		Swarm:    swarmInfo}, nil
}

// Inspect provides a mock function with given fields: id
func (_m *DockerDaemonMock) Inspect(id string) (types.ContainerJSON, error) {
	return types.ContainerJSON{}, nil
}

// InspectImage mock
func (_m *DockerDaemonMock) InspectImage(name string) (types.ImageInspect, error) {
	return types.ImageInspect{}, nil
}

// IsContainerRunning provides a mock function with given fields: id
func (_m *DockerDaemonMock) IsContainerRunning(id string) bool {
	return false
}

// Kill provides a mock function with given fields: id
func (_m *DockerDaemonMock) Kill(id string) error {
	return nil
}

// Logs provides a mock function with given fields: id
func (_m *DockerDaemonMock) Logs(id, since string, ts bool) (io.ReadCloser, error) {
	return nil, nil
}

// Networks mock
func (_m *DockerDaemonMock) Networks() ([]types.NetworkResource, error) {
	return nil, nil
}

// NetworkAt mock
func (_m *DockerDaemonMock) NetworkAt(position int) (*types.NetworkResource, error) {
	return nil, nil
}

// NetworksCount mock
func (_m *DockerDaemonMock) NetworksCount() int {
	return 0
}

// NetworkInspect mock
func (_m *DockerDaemonMock) NetworkInspect(id string) (types.NetworkResource, error) {
	return types.NetworkResource{}, nil
}

// Node mock
func (_m *DockerDaemonMock) Node(id string) (*swarm.Node, error) {
	return nil, nil
}

// NodeChangeAvailability mock
func (_m *DockerDaemonMock) NodeChangeAvailability(nodeID string, availability swarm.NodeAvailability) error {
	return nil
}

// Nodes mock
func (_m *DockerDaemonMock) Nodes() ([]swarm.Node, error) {
	return nil, nil
}

// NodeTasks mock
func (_m *DockerDaemonMock) NodeTasks(nodeID string) ([]swarm.Task, error) {
	return nil, nil
}

// Ok mocks OK
func (_m *DockerDaemonMock) Ok() (bool, error) {

	return false, nil
}

// StatsChannel mocks StatsChannel
func (_m *DockerDaemonMock) StatsChannel(container *drydocker.Container) (*drydocker.StatsChannel, error) {
	return nil, nil
}

// Prune mocks prune command
func (_m *DockerDaemonMock) Prune() (*drydocker.PruneReport, error) {
	return nil, nil
}

// RestartContainer provides a mock function with given fields: id
func (_m *DockerDaemonMock) RestartContainer(id string) error {

	return nil
}

// Rm provides a mock function with given fields: id
func (_m *DockerDaemonMock) Rm(id string) error {
	return nil
}

// Rmi mock
func (_m *DockerDaemonMock) Rmi(id string, force bool) ([]types.ImageDeleteResponseItem, error) {
	return nil, nil
}

// Refresh provides a mock function with given fields: allContainers
func (_m *DockerDaemonMock) Refresh(notify func(err error)) {
	notify(nil)
}

// RefreshImages mock
func (_m *DockerDaemonMock) RefreshImages() error {
	return nil
}

// RefreshNetworks mock
func (_m *DockerDaemonMock) RefreshNetworks() error {
	return nil
}

// RemoveAllStoppedContainers provides a mock function with given fields:
func (_m *DockerDaemonMock) RemoveAllStoppedContainers() (int, error) {
	return 0, nil

}

// RemoveDanglingImages mock
func (_m *DockerDaemonMock) RemoveDanglingImages() (int, error) {
	return 0, nil
}

// RemoveNetwork mock
func (_m *DockerDaemonMock) RemoveNetwork(id string) error {
	return nil
}

// RemoveUnusedImages mock
func (_m *DockerDaemonMock) RemoveUnusedImages() (int, error) {
	return 0, nil
}

// ResolveNode mock
func (_m *DockerDaemonMock) ResolveNode(id string) (string, error) {
	return "", nil
}

// ResolveService mock
func (_m *DockerDaemonMock) ResolveService(id string) (string, error) {
	return "", nil
}

// RunImage mock
func (_m *DockerDaemonMock) RunImage(image types.ImageSummary, command string) error {
	return nil
}

// Service mock
func (_m *DockerDaemonMock) Service(id string) (*swarm.Service, error) {
	return nil, nil
}

// ServiceLogs mock
func (_m *DockerDaemonMock) ServiceLogs(id, since string, ts bool) (io.ReadCloser, error) {
	return nil, nil
}

// Services mock
func (_m *DockerDaemonMock) Services() ([]swarm.Service, error) {
	return nil, nil
}

// ServiceRemove mock
func (_m *DockerDaemonMock) ServiceRemove(id string) error {
	return nil
}

// ServiceScale mock
func (_m *DockerDaemonMock) ServiceScale(id string, scale uint64) error {
	return nil
}

// ServiceTasks mock
func (_m *DockerDaemonMock) ServiceTasks(services ...string) ([]swarm.Task, error) {
	return nil, nil
}

// ServiceUpdate forces an update of the given service
func (_m *DockerDaemonMock) ServiceUpdate(id string) error {
	return nil
}

// StopContainer provides a mock function with given fields: id
func (_m *DockerDaemonMock) StopContainer(id string) error {
	return nil
}

// Sort provides a mock function with given fields: sortMode
func (_m *DockerDaemonMock) Sort(sortMode drydocker.SortMode) {

}

// SortImages mock
func (_m *DockerDaemonMock) SortImages(sortMode drydocker.SortMode) {
}

// SortNetworks mock
func (_m *DockerDaemonMock) SortNetworks(sortMode drydocker.SortMode) {
}

// Stacks mock
func (_m *DockerDaemonMock) Stacks() ([]drydocker.Stack, error) {
	return nil, nil
}

// StackConfigs mock
func (_m *DockerDaemonMock) StackConfigs(stack string) ([]swarm.Config, error) {
	return nil, nil
}

// StackNetworks mock
func (_m *DockerDaemonMock) StackNetworks(stack string) ([]types.NetworkResource, error) {
	return nil, nil
}

// StackSecrets mock
func (_m *DockerDaemonMock) StackSecrets(stack string) ([]swarm.Secret, error) {
	return nil, nil
}

// StackRemove mock
func (_m *DockerDaemonMock) StackRemove(stack string) error {
	return nil
}

// StackServices mock
func (_m *DockerDaemonMock) StackServices(stack string) ([]swarm.Service, error) {
	return nil, nil
}

// StackTasks empty mock
func (_m *DockerDaemonMock) StackTasks(stack string) ([]swarm.Task, error) {
	return nil, nil
}

// Task empty mock
func (_m *DockerDaemonMock) Task(id string) (swarm.Task, error) {
	return swarm.Task{}, nil
}

// Top function mock
func (_m *DockerDaemonMock) Top(ctx context.Context, id string) (container.ContainerTopOKBody, error) {

	return container.ContainerTopOKBody{}, nil
}

// Version provides a mock function with given fields:
func (_m *DockerDaemonMock) Version() (*types.Version, error) {
	return &types.Version{
		Version:       "1.0",
		APIVersion:    "1.27",
		Os:            "dry",
		Arch:          "amd64",
		KernelVersion: "42",
	}, nil
}

// VolumeInspect mock
func (_m *DockerDaemonMock) VolumeInspect(ctx context.Context, volumeID string) (volume.Volume, error) {
	return volume.Volume{}, nil
}

// VolumeList mock
func (_m *DockerDaemonMock) VolumeList(ctx context.Context) ([]*volume.Volume, error) {
	return nil, nil
}

// VolumePrune mock
func (_m *DockerDaemonMock) VolumePrune(ctx context.Context) (int, error) {
	return 0, nil
}

// VolumeRemove mock
func (_m *DockerDaemonMock) VolumeRemove(ctx context.Context, volumeID string, force bool) error {
	return nil
}

// VolumeRemoveAll mock
func (_m *DockerDaemonMock) VolumeRemoveAll(ctx context.Context) (int, error) {
	return 0, nil
}
