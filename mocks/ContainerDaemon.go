package mocks

import (
	"encoding/json"

	drydocker "github.com/moncho/dry/docker"
)

import "io"

import "github.com/fsouza/go-dockerclient"

//ContainerDaemonMock mocks a ContainerDaemonMock
type ContainerDaemonMock struct {
}

//Containers mocked
func (_m *ContainerDaemonMock) Containers() []docker.APIContainers {
	return make([]docker.APIContainers, 0)
}

// ContainersCount provides a mock function with given fields:
func (_m *ContainerDaemonMock) ContainersCount() int {
	return 0
}

// ContainerIDAt provides a mock function with given fields: pos
func (_m *ContainerDaemonMock) ContainerIDAt(pos int) (string, string, error) {
	return "", "", nil
}

// ContainerByID provides a mock function with given fields: cid
func (_m *ContainerDaemonMock) ContainerByID(cid string) docker.APIContainers {
	return docker.APIContainers{}
}

// DockerEnv provides a mock function with given fields:
func (_m *ContainerDaemonMock) DockerEnv() *drydocker.DockerEnv {
	return &drydocker.DockerEnv{DockerHost: "dry.io", DockerTLSVerify: false, DockerCertPath: ""}
}

// Events provides a mock function with given fields:
func (_m *ContainerDaemonMock) Events() (chan *docker.APIEvents, error) {

	return nil, nil
}

//History mock
func (_m *ContainerDaemonMock) History(id string) ([]docker.ImageHistory, error) {
	return nil, nil
}

//ImageAt mock
func (_m *ContainerDaemonMock) ImageAt(pos int) (*docker.APIImages, error) {
	return nil, nil
}

//Images mock
func (_m *ContainerDaemonMock) Images() ([]docker.APIImages, error) {

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
	var images []docker.APIImages
	err := json.Unmarshal([]byte(imagesJSON), &images)
	return images, err
}

//ImagesCount mock
func (_m *ContainerDaemonMock) ImagesCount() int {
	i, _ := _m.Images()
	return len(i)
}

// Info provides a mock function with given fields:
func (_m *ContainerDaemonMock) Info() (*docker.Env, error) {
	return nil, nil
}

// Inspect provides a mock function with given fields: id
func (_m *ContainerDaemonMock) Inspect(id string) (*docker.Container, error) {
	return nil, nil
}

// InspectImage mock
func (_m *ContainerDaemonMock) InspectImage(name string) (*docker.Image, error) {
	return nil, nil
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

// Ok provides a mock function with given fields:
func (_m *ContainerDaemonMock) Ok() (bool, error) {

	return false, nil
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
func (_m *ContainerDaemonMock) Rmi(id string) error {
	return nil
}

// Refresh provides a mock function with given fields: allContainers
func (_m *ContainerDaemonMock) Refresh(allContainers bool) error {
	return nil
}

//RefreshImages mock
func (_m *ContainerDaemonMock) RefreshImages() error {
	return nil
}

// RemoveAllStoppedContainers provides a mock function with given fields:
func (_m *ContainerDaemonMock) RemoveAllStoppedContainers() error {
	return nil

}

// Stats provides a mock function with given fields: id
func (_m *ContainerDaemonMock) Stats(id string) (<-chan *drydocker.Stats, chan<- bool, <-chan error) {

	return nil, nil, nil
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

// StopEventChannel provides a mock function with given fields: eventChan
func (_m *ContainerDaemonMock) StopEventChannel(eventChan chan *docker.APIEvents) error {

	return nil
}

//Top function mock
func (_m *ContainerDaemonMock) Top(id string) (docker.TopResult, error) {

	return docker.TopResult{}, nil
}

// Version provides a mock function with given fields:
func (_m *ContainerDaemonMock) Version() (*drydocker.Version, error) {

	return &drydocker.Version{}, nil
}
