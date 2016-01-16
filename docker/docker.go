package docker

import (
	"errors"
	"io"
	"strings"
	"sync"

	"github.com/docker/docker/pkg/stringid"
	"github.com/fsouza/go-dockerclient"
)

const (
	//DefaultDockerHost is used as a default docker host to connect to
	//if no other value is given.
	DefaultDockerHost = "unix:///var/run/docker.sock"
)

//DockerDaemon knows how to talk to the Docker daemon
type DockerDaemon struct {
	client        *docker.Client                  //client used to to connect to the Docker daemon
	containerByID map[string]docker.APIContainers // Containers by their id
	Containers    []docker.APIContainers
	err           error // Errors, if any.
	connected     bool
	DockerEnv     *DockerEnv
}

//DockerEnv are the Docker-related environment variables defined
type DockerEnv struct {
	DockerHost      string
	DockerTLSVerify bool //tls must be verified
	DockerCertPath  string
}

//ContainersCount returns the number of containers found.
func (daemon *DockerDaemon) ContainersCount() int {
	return len(daemon.containerByID)
}

//ContainerIDAt returns the container ID of the container found at the given
//position.
func (daemon *DockerDaemon) ContainerIDAt(pos int) (string, string, error) {
	if pos >= len(daemon.Containers) {
		return "", "", errors.New("Position is higher than number of containers")
	}
	return daemon.Containers[pos].ID, stringid.TruncateID(daemon.Containers[pos].ID), nil
}

//ContainerByID returns the container with the given ID
func (daemon *DockerDaemon) ContainerByID(cid string) docker.APIContainers {
	return daemon.containerByID[cid]
}

//Info returns system-wide information about the Docker server.
func (daemon *DockerDaemon) Info() (*docker.Env, error) {
	return daemon.client.Info()
}

//Inspect the container with the given id
func (daemon *DockerDaemon) Inspect(id string) (*docker.Container, error) {
	return daemon.client.InspectContainer(id)
}

//Kill the container with the given id
func (daemon *DockerDaemon) Kill(id string) error {
	opts := docker.KillContainerOptions{
		ID: id,
	}
	return daemon.client.KillContainer(opts)
}

//Logs shows the logs of the container with the given id
func (daemon *DockerDaemon) Logs(id string) io.ReadCloser {
	r, w := io.Pipe()
	options := docker.LogsOptions{
		Container:    id,
		OutputStream: w,
		ErrorStream:  w,
		Follow:       true,
		Stdout:       true,
		Stderr:       true,
	}

	go daemon.client.Logs(options)
	return r
}

//Ok is true if connecting to the Docker daemon went fine
func (daemon *DockerDaemon) Ok() (bool, error) {
	return daemon.err == nil, daemon.err
}

//RestartContainer restarts the container with the given id
func (daemon *DockerDaemon) RestartContainer(id string) error {
	//fixme: timeout to start a container
	return daemon.client.RestartContainer(id, 10)
}

//Rm removes the container with the given id
func (daemon *DockerDaemon) Rm(id string) error {
	opts := docker.RemoveContainerOptions{
		ID: id,
	}
	return daemon.client.RemoveContainer(opts)
}

//RemoveAllStoppedContainers removes all stopped containers
func (daemon *DockerDaemon) RemoveAllStoppedContainers() error {
	containers, _, err := containers(daemon.client, true)

	errs := make(chan error, 1)
	defer close(errs)
	if err == nil {
		var wg sync.WaitGroup
		for _, container := range containers {
			if !IsContainerRunning(container) {
				wg.Add(1)
				go func(id string) {
					defer wg.Done()
					err := daemon.Rm(id)
					if err != nil {
						select {
						case errs <- err:
						default:
						}
					}
				}(container.ID)
			}
		}
		wg.Wait()
		select {
		case e := <-errs:
			return e
		default:
		}
	}
	return err
}

//Stats shows resource usage statistics of the container with the given id
func (daemon *DockerDaemon) Stats(id string) (<-chan *Stats, chan<- bool, <-chan error) {
	statsFromDocker := make(chan *docker.Stats)
	stats := make(chan *Stats)
	dockerDone := make(chan bool, 1)
	done := make(chan bool, 1)
	errorC := make(chan error, 1)

	go func() {
		options := docker.StatsOptions{
			ID:     id,
			Stream: true,
			Stats:  statsFromDocker,
			Done:   dockerDone,
		}
		if err := daemon.client.Stats(options); err != nil {
			errorC <- err
		}
	}()
	go func() {
		for {
			select {
			case s := <-statsFromDocker:
				if s != nil {
					stats <- BuildStats(daemon.containerByID[id], s)
				}
			case <-done:
				dockerDone <- true
				//statsFromDocker is closed by the Docker client
				//close(stats)
				//close(done)
				return
			}
		}
	}()
	return stats, done, errorC
}

//StopContainer stops the container with the given id
func (daemon *DockerDaemon) StopContainer(id string) error {
	//fixme: timeout to stop a container
	return daemon.client.StopContainer(id, 10)
}

//Refresh the container list
func (daemon *DockerDaemon) Refresh(allContainers bool) error {

	containers, containerByID, err := containers(daemon.client, allContainers)

	if err == nil {
		daemon.containerByID = containerByID
		daemon.Containers = containers
	}
	return err
}

//Sort the list of containers by the given mode
func (daemon *DockerDaemon) Sort(sortMode SortMode) {
	SortContainers(daemon.Containers, sortMode)
}

func containers(client *docker.Client, allContainers bool) ([]docker.APIContainers, map[string]docker.APIContainers, error) {
	containers, err := client.ListContainers(docker.ListContainersOptions{All: allContainers})
	if err == nil {
		var cmap = make(map[string]docker.APIContainers)

		for _, c := range containers {
			cmap[c.ID] = c
		}
		return containers, cmap, nil
	}
	return nil, nil, err
}

//GetBool returns false if the given string looks like you mean
//false. Func doesnt belong here.
func GetBool(key string) (value bool) {
	s := strings.ToLower(strings.Trim(key, " "))
	if s == "" || s == "0" || s == "no" || s == "false" || s == "none" {
		return false
	}
	return true
}
