package docker

import (
	"net/http"
	"path/filepath"
	"strings"

	"github.com/docker/engine-api/client"
	dockerTypes "github.com/docker/engine-api/types"
	"github.com/docker/go-connections/tlsconfig"
)

func connect(client client.APIClient, env *DockerEnv) (*DockerDaemon, error) {
	containers, containersByID, err := containers(client, false)
	if err == nil {
		images, err := images(client)
		if err == nil {
			networks, err := networks(client)
			if err == nil {
				d := &DockerDaemon{
					client:        client,
					err:           err,
					containerByID: containersByID,
					containers:    containers,
					images:        images,
					networks:      networks,
					dockerEnv:     env,
				}
				d.Version()
				return d, nil
			}
		}
		return nil, err

	}
	return nil, err
}

//ConnectToDaemon connects to a Docker daemon using the given properties.
func ConnectToDaemon(env *DockerEnv) (*DockerDaemon, error) {
	var httpClient *http.Client
	if dockerCertPath := env.DockerCertPath; dockerCertPath != "" {
		options := tlsconfig.Options{
			CAFile:             filepath.Join(dockerCertPath, "ca.pem"),
			CertFile:           filepath.Join(dockerCertPath, "cert.pem"),
			KeyFile:            filepath.Join(dockerCertPath, "key.pem"),
			InsecureSkipVerify: env.DockerTLSVerify,
		}
		tlsc, err := tlsconfig.Client(options)
		if err != nil {
			return nil, err
		}

		httpClient = &http.Client{
			Transport: &http.Transport{
				TLSClientConfig: tlsc,
			},
		}
	}

	host := env.DockerHost
	if host == "" {
		host = DefaultDockerHost
	}
	client, err := client.NewClient(host, env.DockerAPIVersion, httpClient, nil)
	if err == nil {
		return connect(client, env)
	}
	return nil, err
}

//IsContainerRunning returns true if the given container is running
func IsContainerRunning(container dockerTypes.Container) bool {
	return strings.Contains(container.Status, "Up")
}
