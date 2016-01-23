package mocks

import drydocker "github.com/moncho/dry/docker"
import "github.com/stretchr/testify/mock"

import "io"

import "github.com/fsouza/go-dockerclient"

//ContainerDaemonMock mocks a ContainerDaemonMock
type ContainerDaemonMock struct {
	mock.Mock
}

//Containers mocked
func (_m *ContainerDaemonMock) Containers() []docker.APIContainers {
	return make([]docker.APIContainers, 0)
}

// ContainersCount provides a mock function with given fields:
func (_m *ContainerDaemonMock) ContainersCount() int {
	ret := _m.Called()

	var r0 int
	if rf, ok := ret.Get(0).(func() int); ok {
		r0 = rf()
	} else {
		r0 = ret.Get(0).(int)
	}

	return r0
}

// ContainerIDAt provides a mock function with given fields: pos
func (_m *ContainerDaemonMock) ContainerIDAt(pos int) (string, string, error) {
	ret := _m.Called(pos)

	var r0 string
	if rf, ok := ret.Get(0).(func(int) string); ok {
		r0 = rf(pos)
	} else {
		r0 = ret.Get(0).(string)
	}

	var r1 string
	if rf, ok := ret.Get(1).(func(int) string); ok {
		r1 = rf(pos)
	} else {
		r1 = ret.Get(1).(string)
	}

	var r2 error
	if rf, ok := ret.Get(2).(func(int) error); ok {
		r2 = rf(pos)
	} else {
		r2 = ret.Error(2)
	}

	return r0, r1, r2
}

// ContainerByID provides a mock function with given fields: cid
func (_m *ContainerDaemonMock) ContainerByID(cid string) docker.APIContainers {
	ret := _m.Called(cid)

	var r0 docker.APIContainers
	if rf, ok := ret.Get(0).(func(string) docker.APIContainers); ok {
		r0 = rf(cid)
	} else {
		r0 = ret.Get(0).(docker.APIContainers)
	}

	return r0
}

// DockerEnv provides a mock function with given fields:
func (_m *ContainerDaemonMock) DockerEnv() *drydocker.DockerEnv {
	ret := _m.Called()

	var r0 *drydocker.DockerEnv
	if rf, ok := ret.Get(0).(func() *drydocker.DockerEnv); ok {
		r0 = rf()
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*drydocker.DockerEnv)
		}
	}

	return r0
}

// Events provides a mock function with given fields:
func (_m *ContainerDaemonMock) Events() (chan *docker.APIEvents, error) {
	ret := _m.Called()

	var r0 chan *docker.APIEvents
	if rf, ok := ret.Get(0).(func() chan *docker.APIEvents); ok {
		r0 = rf()
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(chan *docker.APIEvents)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func() error); ok {
		r1 = rf()
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// Info provides a mock function with given fields:
func (_m *ContainerDaemonMock) Info() (*docker.Env, error) {
	ret := _m.Called()

	var r0 *docker.Env
	if rf, ok := ret.Get(0).(func() *docker.Env); ok {
		r0 = rf()
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*docker.Env)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func() error); ok {
		r1 = rf()
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// Inspect provides a mock function with given fields: id
func (_m *ContainerDaemonMock) Inspect(id string) (*docker.Container, error) {
	ret := _m.Called(id)

	var r0 *docker.Container
	if rf, ok := ret.Get(0).(func(string) *docker.Container); ok {
		r0 = rf(id)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*docker.Container)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(string) error); ok {
		r1 = rf(id)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// IsContainerRunning provides a mock function with given fields: id
func (_m *ContainerDaemonMock) IsContainerRunning(id string) bool {
	ret := _m.Called(id)

	var r0 bool
	if rf, ok := ret.Get(0).(func(string) bool); ok {
		r0 = rf(id)
	} else {
		r0 = ret.Get(0).(bool)
	}

	return r0
}

// Kill provides a mock function with given fields: id
func (_m *ContainerDaemonMock) Kill(id string) error {
	ret := _m.Called(id)

	var r0 error
	if rf, ok := ret.Get(0).(func(string) error); ok {
		r0 = rf(id)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// Logs provides a mock function with given fields: id
func (_m *ContainerDaemonMock) Logs(id string) io.ReadCloser {
	ret := _m.Called(id)

	var r0 io.ReadCloser
	if rf, ok := ret.Get(0).(func(string) io.ReadCloser); ok {
		r0 = rf(id)
	} else {
		r0 = ret.Get(0).(io.ReadCloser)
	}

	return r0
}

// Ok provides a mock function with given fields:
func (_m *ContainerDaemonMock) Ok() (bool, error) {
	ret := _m.Called()

	var r0 bool
	if rf, ok := ret.Get(0).(func() bool); ok {
		r0 = rf()
	} else {
		r0 = ret.Get(0).(bool)
	}

	var r1 error
	if rf, ok := ret.Get(1).(func() error); ok {
		r1 = rf()
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// RestartContainer provides a mock function with given fields: id
func (_m *ContainerDaemonMock) RestartContainer(id string) error {
	ret := _m.Called(id)

	var r0 error
	if rf, ok := ret.Get(0).(func(string) error); ok {
		r0 = rf(id)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// Rm provides a mock function with given fields: id
func (_m *ContainerDaemonMock) Rm(id string) error {
	ret := _m.Called(id)

	var r0 error
	if rf, ok := ret.Get(0).(func(string) error); ok {
		r0 = rf(id)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// Refresh provides a mock function with given fields: allContainers
func (_m *ContainerDaemonMock) Refresh(allContainers bool) error {
	return nil
}

// RemoveAllStoppedContainers provides a mock function with given fields:
func (_m *ContainerDaemonMock) RemoveAllStoppedContainers() error {
	ret := _m.Called()

	var r0 error
	if rf, ok := ret.Get(0).(func() error); ok {
		r0 = rf()
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// Stats provides a mock function with given fields: id
func (_m *ContainerDaemonMock) Stats(id string) (<-chan *drydocker.Stats, chan<- bool, <-chan error) {
	ret := _m.Called(id)

	var r0 <-chan *drydocker.Stats
	if rf, ok := ret.Get(0).(func(string) <-chan *drydocker.Stats); ok {
		r0 = rf(id)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(<-chan *drydocker.Stats)
		}
	}

	var r1 chan<- bool
	if rf, ok := ret.Get(1).(func(string) chan<- bool); ok {
		r1 = rf(id)
	} else {
		if ret.Get(1) != nil {
			r1 = ret.Get(1).(chan<- bool)
		}
	}

	var r2 <-chan error
	if rf, ok := ret.Get(2).(func(string) <-chan error); ok {
		r2 = rf(id)
	} else {
		if ret.Get(2) != nil {
			r2 = ret.Get(2).(<-chan error)
		}
	}

	return r0, r1, r2
}

// StopContainer provides a mock function with given fields: id
func (_m *ContainerDaemonMock) StopContainer(id string) error {
	ret := _m.Called(id)

	var r0 error
	if rf, ok := ret.Get(0).(func(string) error); ok {
		r0 = rf(id)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// Sort provides a mock function with given fields: sortMode
func (_m *ContainerDaemonMock) Sort(sortMode drydocker.SortMode) {
	_m.Called(sortMode)
}

// StopReceivingEvents provides a mock function with given fields: eventChan
func (_m *ContainerDaemonMock) StopReceivingEvents(eventChan chan *docker.APIEvents) error {
	ret := _m.Called(eventChan)

	var r0 error
	if rf, ok := ret.Get(0).(func(chan *docker.APIEvents) error); ok {
		r0 = rf(eventChan)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// Version provides a mock function with given fields:
func (_m *ContainerDaemonMock) Version() (*drydocker.Version, error) {
	ret := _m.Called()

	var r0 *drydocker.Version
	if rf, ok := ret.Get(0).(func() *drydocker.Version); ok {
		r0 = rf()
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*drydocker.Version)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func() error); ok {
		r1 = rf()
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}
