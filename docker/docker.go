package docker

import (
	"fmt"
	"io"
	"os"

	log "github.com/Sirupsen/logrus"
	"github.com/fsouza/go-dockerclient"
)

//DockerDaemon knows how to talk to the Docker daemon
type DockerDaemon struct {
	client     *docker.Client         //client used to to connect to the Docker daemon
	Containers []docker.APIContainers // Array of containers.
	err        error                  // Error, if any.
	connected  bool
	DockerEnv  *DockerEnv
}

//DockerEnv are the Docker-related environment variables defined
type DockerEnv struct {
	DockerHost      string
	DockerTLSVerify bool
	DockerCertPath  string
}

//ContainersCount returns the number of containers found.
func (daemon *DockerDaemon) ContainersCount() int {
	return len(daemon.Containers)
}

//ContainerIDAt returns the container ID of the container found at the given
//position from the list of containers.
func (daemon *DockerDaemon) ContainerIDAt(position int) (string, error) {
	if position >= len(daemon.Containers) {
		return "", fmt.Errorf("Invalid position %d", position)
	}
	return daemon.Containers[position].ID, nil
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
func (daemon *DockerDaemon) Logs(id string) io.Reader {
	r, w := io.Pipe()
	options := docker.AttachToContainerOptions{
		Container:    id,
		OutputStream: w,
		ErrorStream:  w,
		Stream:       true,
		Stdout:       true,
		Stderr:       true,
		Logs:         true,
	}
	log.Infof("Attaching to container: %s", id)
	go daemon.client.AttachToContainer(options)
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
func (daemon *DockerDaemon) Rm(id string) bool {
	opts := docker.RemoveContainerOptions{
		ID: id,
	}
	return daemon.client.RemoveContainer(opts) == nil
}

//Stats shows resource usage statistics of the container with the given id
func (daemon *DockerDaemon) Stats(id string) (<-chan *Stats, chan bool) {
	statsFromDocker := make(chan *docker.Stats)
	stats := make(chan *Stats)
	done := make(chan bool, 1)
	options := docker.StatsOptions{
		ID:      id,
		Stream:  false,
		Timeout: 0, //1 * time.Second,
		Stats:   statsFromDocker,
		Done:    done,
	}
	//log.Infof("Showing stats of container: %s", id)

	go func(done chan bool) {
		if err := daemon.client.Stats(options); err != nil {
			//log.Errorf("Error gettings statistics for id %s, error: %s", id, err.Error())
			done <- true
		}
	}(done)
	go func(stats chan *Stats, done chan bool) {
		select {
		case s := <-statsFromDocker:
			stats <- BuildStats(id, s)
		case <-done:
			close(stats)
			close(done)
		}
	}(stats, done)
	return stats, done
}

//StopContainer stops the container with the given id
func (daemon *DockerDaemon) StopContainer(id string) error {
	//fixme: timeout to stop a container
	return daemon.client.StopContainer(id, 10)
}

//Refresh the container list
func (daemon *DockerDaemon) Refresh(allContainers bool) error {
	containers, err :=
		daemon.client.ListContainers(docker.ListContainersOptions{All: allContainers})
	if err == nil {
		daemon.Containers = containers
	}
	return err
}

//ConnectToDaemon connects with the Docker daemon.
//if allContainers is true, then a list of both running and non-running containers
//is retrieved.
func ConnectToDaemon() *DockerDaemon {
	client, error := docker.NewClientFromEnv()

	containers, _ := client.ListContainers(docker.ListContainersOptions{All: false})
	return &DockerDaemon{
		client:     client,
		err:        error,
		Containers: containers,
		DockerEnv: &DockerEnv{
			os.Getenv("DOCKER_HOST"),
			os.Getenv("DOCKER_TLS_VERIFY") != "",
			os.Getenv("DOCKER_CERT_PATH"),
		}}
}
