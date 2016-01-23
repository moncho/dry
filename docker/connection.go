package docker

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/fsouza/go-dockerclient"
)

func connect(client *docker.Client, env *DockerEnv) (*DockerDaemon, error) {
	containers, containersByID, err := containers(client, false)
	if err == nil {
		d := &DockerDaemon{
			client:        client,
			err:           err,
			containerByID: containersByID,
			containers:    containers,
			dockerEnv:     env,
		}
		d.Version()
		return d, nil
	}
	return nil, err
}

//ConnectToDaemon connects to a Docker daemon using environment properties.
func ConnectToDaemon() (*DockerDaemon, error) {
	client, err := docker.NewClientFromEnv()
	if err == nil {
		return connect(client, &DockerEnv{
			DockerHost:      os.Getenv("DOCKER_HOST"),
			DockerTLSVerify: GetBool(os.Getenv("DOCKER_TLS_VERIFY")),
			DockerCertPath:  os.Getenv("DOCKER_CERT_PATH")})
	}
	return nil, err
}

//ConnectToDaemonUsingEnv connects to a Docker daemon using the given properties.
func ConnectToDaemonUsingEnv(env *DockerEnv) (*DockerDaemon, error) {
	dockerHost := env.DockerHost
	//If a cert path is given it is implied that tls has to be used.
	if env.DockerCertPath != "" {
		parts := strings.SplitN(dockerHost, "://", 2)
		if len(parts) != 2 {
			return nil, fmt.Errorf("could not split %s into two parts by ://", dockerHost)
		}

		var client *docker.Client
		var err error
		//fsouza docker client decides if tls has to be verified by checking if a CACert is given
		//See https://github.com/fsouza/go-dockerclient/blob/master/client.go#L289
		if env.DockerTLSVerify {
			cert := filepath.Join(env.DockerCertPath, "cert.pem")
			key := filepath.Join(env.DockerCertPath, "key.pem")
			ca := filepath.Join(env.DockerCertPath, "ca.pem")
			client, err = docker.NewVersionedTLSClient(dockerHost, cert, key, ca, "")
		} else {
			cert, key, _, err :=
				readCertificateFiles(
					filepath.Join(env.DockerCertPath, "cert.pem"),
					filepath.Join(env.DockerCertPath, "key.pem"),
					"")
			if err == nil {
				client, err = docker.NewVersionedTLSClientFromBytes(dockerHost, cert, key, nil, "")
			}
		}
		if err == nil {
			return connect(client, env)
		}
		return nil, err
	}
	client, err := docker.NewVersionedClient(dockerHost, "")
	if err == nil {
		return connect(client, env)
	}
	return nil, err
}

//readCertificateContent reads the content of the given files. The CA file path is optional.
func readCertificateFiles(certificateFile string, keyFile, caFile string) ([]byte, []byte, []byte, error) {
	certPEMBlock, err := ioutil.ReadFile(certificateFile)
	if err != nil {
		return nil, nil, nil, err
	}
	keyPEMBlock, err := ioutil.ReadFile(keyFile)
	if err != nil {
		return nil, nil, nil, err
	}

	//caFile is optional
	if caFile != "" {
		caPEMCert, err := ioutil.ReadFile(caFile)
		if err != nil {
			return nil, nil, nil, err
		}
		return certPEMBlock, keyPEMBlock, caPEMCert, nil
	}
	return certPEMBlock, keyPEMBlock, nil, nil
}

//IsContainerRunning returns true if the given container is running
func IsContainerRunning(container docker.APIContainers) bool {
	return strings.Contains(container.Status, "Up")
}
