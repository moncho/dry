package mocks

import drydocker "github.com/moncho/dry/docker"

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
	return nil
}

// Events provides a mock function with given fields:
func (_m *ContainerDaemonMock) Events() (chan *docker.APIEvents, error) {

	return nil, nil
}

// Info provides a mock function with given fields:
func (_m *ContainerDaemonMock) Info() (*docker.Env, error) {
	return nil, nil
}

// Inspect provides a mock function with given fields: id
func (_m *ContainerDaemonMock) Inspect(id string) (*docker.Container, error) {
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

// Refresh provides a mock function with given fields: allContainers
func (_m *ContainerDaemonMock) Refresh(allContainers bool) error {
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

	return nil, nil
}
