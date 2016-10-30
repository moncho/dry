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
	"github.com/moncho/dry/version"
	"github.com/pkg/errors"
)

const (
	//DefaultConnectionTimeout is the timeout for connecting with the Docker daemon
	DefaultConnectionTimeout = 32 * time.Second
)

var headers = map[string]string{
	"User-Agent": "dry/" + version.VERSION,
}

func connect(client client.APIClient, env *DockerEnv) (*DockerDaemon, error) {
	containers, err := containers(client, false)
	if err == nil {
		images, errI := images(client, defaultImageListOptions)
		if errI == nil {
			networks, errN := networks(client)
			if errN == nil {
				d := &DockerDaemon{
					client:         client,
					err:            err,
					containerStore: NewMemoryStoreWithContainers(containers),
					images:         images,
					networks:       networks,
					dockerEnv:      env,
				}
				d.eventLog = NewEventLog()
				d.Version()
				return d, nil
			}
		}
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
		return nil, errors.Wrap(err, "Invalid Host")
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
			return nil, errors.Wrap(err, "TLS setup error")
		}
	}
	httpClient, err := newHTTPClient(host, tlsConfig)
	if err != nil {
		return nil, errors.Wrap(err, "HttpClient creation error")
	}

	client, err := client.NewClient(host, env.DockerAPIVersion, httpClient, headers)
	if err == nil {
		return connect(client, env)
	}
	return nil, errors.Wrap(err, "Error creating client")
}
