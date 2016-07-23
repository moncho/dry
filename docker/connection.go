package docker

import (
	"crypto/tls"
	"net"
	"net/http"
	"path/filepath"
	"time"

	"github.com/docker/docker/opts"
	"github.com/docker/engine-api/client"
	"github.com/docker/go-connections/sockets"
	drytls "github.com/moncho/dry/tls"
)

const (
	//DefaultConnectionTimeout is the timeout for connecting with the Docker daemon
	DefaultConnectionTimeout = 32 * time.Second
)

func connect(client client.APIClient, env *DockerEnv) (*DockerDaemon, error) {
	containers, containersByID, err := containers(client, false)
	if err == nil {
		images, err := images(client, defaultImageListOptions)
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
				d.eventLog = NewEventLog()
				d.Version()
				return d, nil
			}
		}
		return nil, err

	}
	return nil, err
}

func getServerHost(env *DockerEnv) (string, error) {

	host := env.DockerHost
	if host == "" {
		host = DefaultDockerHost
	}

	return opts.ParseHost(env.DockerCertPath != "", host)
}

func newHTTPClient(host string, config *tls.Config) (*http.Client, error) {
	if config == nil {
		// let the api client configure the default transport.
		return nil, nil
	}

	proto, addr, _, err := client.ParseHost(host)
	if err != nil {
		return nil, err
	}

	tr := &http.Transport{
		TLSClientConfig: config,
		Dial: func(network, addr string) (net.Conn, error) {
			return net.DialTimeout(proto, addr, DefaultConnectionTimeout)
		},
	}

	sockets.ConfigureTransport(tr, proto, addr)

	return &http.Client{
		Transport: tr,
	}, nil
}

//ConnectToDaemon connects to a Docker daemon using the given properties.
func ConnectToDaemon(env *DockerEnv) (*DockerDaemon, error) {

	host, err := getServerHost(env)
	if err != nil {
		return nil, err
	}
	var tlsConfig *tls.Config
	if dockerCertPath := env.DockerCertPath; dockerCertPath != "" {
		options := drytls.Options{
			CAFile:             filepath.Join(dockerCertPath, "ca.pem"),
			CertFile:           filepath.Join(dockerCertPath, "cert.pem"),
			KeyFile:            filepath.Join(dockerCertPath, "key.pem"),
			InsecureSkipVerify: env.DockerTLSVerify,
		}
		tlsConfig, err = drytls.Client(options)
		if err != nil {
			return nil, err
		}
	}
	httpClient, err := newHTTPClient(host, tlsConfig)
	if err != nil {
		return nil, err
	}

	client, err := client.NewClient(host, env.DockerAPIVersion, httpClient, nil)
	if err == nil {
		return connect(client, env)
	}
	return nil, err
}
