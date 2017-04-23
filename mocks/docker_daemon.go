package mocks

import (
	"encoding/json"
	"io"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/events"
	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/api/types/swarm"
	drydocker "github.com/moncho/dry/docker"
)

//DockerDaemonMock mocks a DockerDaemon
type DockerDaemonMock struct {
}

//ContainerByID mock
func (_m *DockerDaemonMock) ContainerByID(id string) *drydocker.Container {
	return nil
}

//Containers mock
func (_m *DockerDaemonMock) Containers(filter drydocker.ContainerFilter, mode drydocker.SortMode) []*drydocker.Container {
	return nil
}

//DiskUsage mock
func (_m *DockerDaemonMock) DiskUsage() (types.DiskUsage, error) {
	return types.DiskUsage{}, nil
}

// DockerEnv provides a mock function with given fields:
func (_m *DockerDaemonMock) DockerEnv() *drydocker.Env {
	return &drydocker.Env{DockerHost: "dry.io", DockerTLSVerify: false, DockerCertPath: ""}
}

// Events provides a mock function with given fields:
func (_m *DockerDaemonMock) Events() (<-chan events.Message, chan<- struct{}, error) {

	return nil, nil, nil
}

//EventLog mock
func (_m *DockerDaemonMock) EventLog() *drydocker.EventLog {
	return nil
}

//History mock
func (_m *DockerDaemonMock) History(id string) ([]image.HistoryResponseItem, error) {
	return nil, nil
}

//ImageAt mock
func (_m *DockerDaemonMock) ImageAt(pos int) (*types.ImageSummary, error) {
	return nil, nil
}

//Images mock
func (_m *DockerDaemonMock) Images() ([]types.ImageSummary, error) {

	imagesJSON := `[
		 {
						 "ID": "8dfafdbc3a40",
						  "RepoTags":["dry/dry:latest"],
							"Created": 1367854155,
							"Size": 100,
							"VirtualSize": 1000,
							"ParentID": "7dfafdbc3a40",
							"RepoDigests": ["8dfafdbc3a40"]
		 },
		 {
						 "ID": "541a0f4efc6f",
						  "RepoTags":["dry/dry2:latest"],
							"Created": 1367854155,
							"Size": 100,
							"VirtualSize": 1000,
							"ParentID": "7dfafdbc3a40",
							"RepoDigests": ["541a0f4efc6f"]
		 },
		 {
						 "ID": "26380e1ca356",
						  "RepoTags":["dry/dry3:latest"],
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
						  "RepoTags":["dry/dry5:latest"],
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

//ImagesCount mock
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
func (_m *DockerDaemonMock) Logs(id string) io.ReadCloser {
	return nil
}

//Networks mock
func (_m *DockerDaemonMock) Networks() ([]types.NetworkResource, error) {
	return nil, nil
}

//NetworkAt mock
func (_m *DockerDaemonMock) NetworkAt(position int) (*types.NetworkResource, error) {
	return nil, nil
}

//NetworksCount mock
func (_m *DockerDaemonMock) NetworksCount() int {
	return 0
}

//NetworkInspect mock
func (_m *DockerDaemonMock) NetworkInspect(id string) (types.NetworkResource, error) {
	return types.NetworkResource{}, nil
}

//Node mock
func (_m *DockerDaemonMock) Node(id string) (*swarm.Node, error) {
	return nil, nil
}

//Nodes mock
func (_m *DockerDaemonMock) Nodes() ([]swarm.Node, error) {
	return nil, nil
}

// Ok mocks OK
func (_m *DockerDaemonMock) Ok() (bool, error) {

	return false, nil
}

//OpenChannel mocks OpenChannel
func (_m *DockerDaemonMock) OpenChannel(container *drydocker.Container) *drydocker.StatsChannel {
	return nil
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

//RefreshImages mock
func (_m *DockerDaemonMock) RefreshImages() error {
	return nil
}

//RefreshNetworks mock
func (_m *DockerDaemonMock) RefreshNetworks() error {
	return nil
}

// RemoveAllStoppedContainers provides a mock function with given fields:
func (_m *DockerDaemonMock) RemoveAllStoppedContainers() (int, error) {
	return 0, nil

}

//RemoveDanglingImages mock
func (_m *DockerDaemonMock) RemoveDanglingImages() (int, error) {
	return 0, nil
}

//RemoveNetwork mock
func (_m *DockerDaemonMock) RemoveNetwork(id string) error {
	return nil
}

// StopContainer provides a mock function with given fields: id
func (_m *DockerDaemonMock) StopContainer(id string) error {
	return nil
}

// Sort provides a mock function with given fields: sortMode
func (_m *DockerDaemonMock) Sort(sortMode drydocker.SortMode) {

}

//SortImages mock
func (_m *DockerDaemonMock) SortImages(sortMode drydocker.SortImagesMode) {
}

//SortNetworks mock
func (_m *DockerDaemonMock) SortNetworks(sortMode drydocker.SortNetworksMode) {
}

//Tasks mock
func (_m *DockerDaemonMock) Tasks(nodeID string) ([]swarm.Task, error) {
	return nil, nil
}

//Top function mock
func (_m *DockerDaemonMock) Top(id string) (container.ContainerTopOKBody, error) {

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
