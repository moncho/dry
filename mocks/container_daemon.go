package mocks

import (
	"encoding/json"
	"io"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/events"
	drydocker "github.com/moncho/dry/docker"
)

//ContainerDaemonMock mocks a ContainerDaemonMock
type ContainerDaemonMock struct {
}

//ContainerByID mock
func (_m *ContainerDaemonMock) ContainerByID(id string) *types.Container {
	return nil
}

//Containers mock
func (_m *ContainerDaemonMock) Containers(filter drydocker.ContainerFilter, mode drydocker.SortMode) []*types.Container {
	return nil
}

//DiskUsage mock
func (_m *ContainerDaemonMock) DiskUsage() (types.DiskUsage, error) {
	return types.DiskUsage{}, nil
}

// DockerEnv provides a mock function with given fields:
func (_m *ContainerDaemonMock) DockerEnv() *drydocker.Env {
	return &drydocker.Env{DockerHost: "dry.io", DockerTLSVerify: false, DockerCertPath: ""}
}

// Events provides a mock function with given fields:
func (_m *ContainerDaemonMock) Events() (<-chan events.Message, chan<- struct{}, error) {

	return nil, nil, nil
}

//EventLog mock
func (_m *ContainerDaemonMock) EventLog() *drydocker.EventLog {
	return nil
}

//History mock
func (_m *ContainerDaemonMock) History(id string) ([]types.ImageHistory, error) {
	return nil, nil
}

//ImageAt mock
func (_m *ContainerDaemonMock) ImageAt(pos int) (*types.ImageSummary, error) {
	return nil, nil
}

//Images mock
func (_m *ContainerDaemonMock) Images() ([]types.ImageSummary, error) {

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
func (_m *ContainerDaemonMock) ImagesCount() int {
	i, _ := _m.Images()
	return len(i)
}

// Info provides a mock function with given fields:
func (_m *ContainerDaemonMock) Info() (types.Info, error) {
	return types.Info{}, nil
}

// Inspect provides a mock function with given fields: id
func (_m *ContainerDaemonMock) Inspect(id string) (types.ContainerJSON, error) {
	return types.ContainerJSON{}, nil
}

// InspectImage mock
func (_m *ContainerDaemonMock) InspectImage(name string) (types.ImageInspect, error) {
	return types.ImageInspect{}, nil
}

// IsContainerRunning provides a mock function with given fields: id
func (_m *ContainerDaemonMock) IsContainerRunning(id string) bool {
	return false
}

// Kill provides a mock function with given fields: id
func (_m *ContainerDaemonMock) Kill(id string) error {
	return nil
}

// Logs provides a mock function with given fields: id
func (_m *ContainerDaemonMock) Logs(id string) io.ReadCloser {
	return nil
}

//Networks mock
func (_m *ContainerDaemonMock) Networks() ([]types.NetworkResource, error) {
	return nil, nil
}

//NetworkAt mock
func (_m *ContainerDaemonMock) NetworkAt(position int) (*types.NetworkResource, error) {
	return nil, nil
}

//NetworksCount mock
func (_m *ContainerDaemonMock) NetworksCount() int {
	return 0
}

//NetworkInspect mock
func (_m *ContainerDaemonMock) NetworkInspect(id string) (types.NetworkResource, error) {
	return types.NetworkResource{}, nil
}

// Ok mocks OK
func (_m *ContainerDaemonMock) Ok() (bool, error) {

	return false, nil
}

//OpenChannel mocks OpenChannel
func (_m *ContainerDaemonMock) OpenChannel(container *types.Container) *drydocker.StatsChannel {
	return nil
}

// Prune mocks prune command
func (_m *ContainerDaemonMock) Prune() (*drydocker.PruneReport, error) {
	return nil, nil
}

// RestartContainer provides a mock function with given fields: id
func (_m *ContainerDaemonMock) RestartContainer(id string) error {

	return nil
}

// Rm provides a mock function with given fields: id
func (_m *ContainerDaemonMock) Rm(id string) error {
	return nil
}

// Rmi mock
func (_m *ContainerDaemonMock) Rmi(id string, force bool) ([]types.ImageDelete, error) {
	return nil, nil
}

// Refresh provides a mock function with given fields: allContainers
func (_m *ContainerDaemonMock) Refresh(notify func(err error)) {
	notify(nil)
}

//RefreshImages mock
func (_m *ContainerDaemonMock) RefreshImages() error {
	return nil
}

//RefreshNetworks mock
func (_m *ContainerDaemonMock) RefreshNetworks() error {
	return nil
}

// RemoveAllStoppedContainers provides a mock function with given fields:
func (_m *ContainerDaemonMock) RemoveAllStoppedContainers() (int, error) {
	return 0, nil

}

//RemoveDanglingImages mock
func (_m *ContainerDaemonMock) RemoveDanglingImages() (int, error) {
	return 0, nil
}

//RemoveNetwork mock
func (_m *ContainerDaemonMock) RemoveNetwork(id string) error {
	return nil
}

// StopContainer provides a mock function with given fields: id
func (_m *ContainerDaemonMock) StopContainer(id string) error {
	return nil
}

// Sort provides a mock function with given fields: sortMode
func (_m *ContainerDaemonMock) Sort(sortMode drydocker.SortMode) {

}

//SortImages mock
func (_m *ContainerDaemonMock) SortImages(sortMode drydocker.SortImagesMode) {
}

//SortNetworks mock
func (_m *ContainerDaemonMock) SortNetworks(sortMode drydocker.SortNetworksMode) {
}

//Top function mock
func (_m *ContainerDaemonMock) Top(id string) (types.ContainerProcessList, error) {

	return types.ContainerProcessList{}, nil
}

// Version provides a mock function with given fields:
func (_m *ContainerDaemonMock) Version() (*types.Version, error) {

	return &types.Version{}, nil
}
