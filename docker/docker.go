package docker

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/fsouza/go-dockerclient"
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
	DockerTLSVerify bool
	DockerCertPath  string
}

//ContainersCount returns the number of containers found.
func (daemon *DockerDaemon) ContainersCount() int {
	return len(daemon.containerByID)
}

//ContainerIDAt returns the container ID of the container found at the given
//position.
func (daemon *DockerDaemon) ContainerIDAt(pos int) (string, error) {
	if pos >= len(daemon.Containers) {
		return "", errors.New("Position is higher than number of containers")
	}
	return daemon.Containers[pos].ID, nil
}

//ContainerByID returns the container with the given ID
func (daemon *DockerDaemon) ContainerByID(cid string) docker.APIContainers {
	return daemon.containerByID[cid]
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
	options := docker.AttachToContainerOptions{
		Container:    id,
		OutputStream: w,
		ErrorStream:  w,
		Stream:       true,
		Stdout:       true,
		Stderr:       true,
		Logs:         true,
	}
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
	_ = "breakpoint"
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

func connect(client *docker.Client, err error) (*DockerDaemon, error) {
	if err == nil {
		containers, containersByID, err := containers(client, false)
		return &DockerDaemon{
			client:        client,
			err:           err,
			containerByID: containersByID,
			Containers:    containers,
			DockerEnv: &DockerEnv{
				os.Getenv("DOCKER_HOST"),
				os.Getenv("DOCKER_TLS_VERIFY") != "",
				os.Getenv("DOCKER_CERT_PATH"),
			}}, nil
	}
	return nil, err
}

//ConnectToDaemon connects to a Docker daemon using environment properties.
func ConnectToDaemon() (*DockerDaemon, error) {
	return connect(docker.NewClientFromEnv())
}

//ConnectToGivenDaemon connects to a Docker daemon using the given properties.
func ConnectToGivenDaemon(env *DockerEnv) (*DockerDaemon, error) {
	dockerHost := env.DockerHost
	if env.DockerTLSVerify {
		parts := strings.SplitN(dockerHost, "://", 2)
		if len(parts) != 2 {
			return nil, fmt.Errorf("could not split %s into two parts by ://", dockerHost)
		}
		cert := filepath.Join(env.DockerCertPath, "cert.pem")
		key := filepath.Join(env.DockerCertPath, "key.pem")
		ca := filepath.Join(env.DockerCertPath, "ca.pem")

		return connect(docker.NewVersionedTLSClient(dockerHost, cert, key, ca, ""))
	}
	return connect(docker.NewVersionedClient(dockerHost, ""))

}
