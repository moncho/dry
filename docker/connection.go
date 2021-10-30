package docker

import (
	"context"
	"crypto/tls"
	"golang.org/x/crypto/ssh"
	"io/ioutil"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/docker/cli/opts"
	"github.com/docker/docker/client"
	"github.com/docker/go-connections/sockets"
	"github.com/kevinburke/ssh_config"
	homedir "github.com/mitchellh/go-homedir"
	drytls "github.com/moncho/dry/tls"
	"github.com/moncho/dry/version"
	"github.com/pkg/errors"
)

const (
	//DefaultConnectionTimeout is the timeout for connecting with the Docker daemon
	DefaultConnectionTimeout = 32 * time.Second
)

var defaultDockerPath string

var headers = map[string]string{
	"User-Agent": "dry/" + version.VERSION,
}

func init() {
	defaultDockerPath, _ = homedir.Expand("~/.docker")
}
func connect(client client.APIClient, env Env) (*DockerDaemon, error) {
	store, err := NewDockerContainerStore(client)
	if err != nil {
		return nil, err
	}
	d := &DockerDaemon{
		client:    client,
		err:       err,
		s:         store,
		dockerEnv: env,
		resolver:  newResolver(client, false),
	}
	if err := d.init(); err != nil {
		return nil, err
	}
	return d, nil
}

func getServerHost(env Env) (string, error) {

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

	url, err := client.ParseHostURL(host)
	if err != nil {
		return nil, err
	}
	transport := &http.Transport{
		TLSClientConfig: config,
		Dial: func(network, addr string) (net.Conn, error) {
			return net.DialTimeout(url.Scheme, url.Host, DefaultConnectionTimeout)
		},
	}

	if err = sockets.ConfigureTransport(transport, url.Scheme, url.Host); err != nil {
		return nil, err
	}

	return &http.Client{
		Transport:     transport,
		CheckRedirect: client.CheckRedirect,
	}, nil
}

//ConnectToDaemon connects to a Docker daemon using the given properties.
func ConnectToDaemon(env Env) (*DockerDaemon, error) {

	host, err := getServerHost(env)
	if err != nil {
		return nil, errors.Wrap(err, "Invalid Host")
	}
	var options *drytls.Options
	//If a path to certificates is given use the path to read certificates from
	if dockerCertPath := env.DockerCertPath; dockerCertPath != "" {
		options = &drytls.Options{
			CAFile:             filepath.Join(dockerCertPath, "ca.pem"),
			CertFile:           filepath.Join(dockerCertPath, "cert.pem"),
			KeyFile:            filepath.Join(dockerCertPath, "key.pem"),
			InsecureSkipVerify: env.DockerTLSVerify,
		}
	} else if env.DockerTLSVerify {
		//No cert path is given but TLS verify is set, default location for
		//docker certs will be used.
		//See https://docs.docker.com/engine/security/https/#secure-by-default
		//Fixes #23
		options = &drytls.Options{
			CAFile:             filepath.Join(defaultDockerPath, "ca.pem"),
			CertFile:           filepath.Join(defaultDockerPath, "cert.pem"),
			KeyFile:            filepath.Join(defaultDockerPath, "key.pem"),
			InsecureSkipVerify: env.DockerTLSVerify,
		}
		env.DockerCertPath = defaultDockerPath
	}

	var opt []client.Opt
	if options != nil {
		opt = append(opt, client.WithTLSClientConfig(options.CAFile, options.CertFile, options.KeyFile))
	}

	if host != "" && strings.Index(host, "ssh") == 0 {
		//if it starts with ssh, its an ssh connection, and we need to handle this specially
		//github.com/docker/docker does not handle ssh, as an upgrade to go-connections need to be made
		//see https://github.com/docker/go-connections/pull/39
		url, err := url.Parse(host)
		if err != nil {
			return nil, err
		}

		pass, _ := url.User.Password()
		sshConfig, err := configureSshTransport(url.Host, url.User.Username(), pass)
		if err != nil {
			return nil, err
		}
		opt = append(opt, client.WithDialContext(
			func(ctx context.Context, network, addr string) (net.Conn, error) {
				return connectSshTransport(url.Host, url.Path, sshConfig)
			}))
	} else if host != "" {
		//default uses the docker library to connect to hosts
		opt = append(opt, client.WithHost(host))
	}

	client, err := client.NewClientWithOpts(opt...)
	if err != nil {
		return nil, errors.Wrap(err, "Error creating client")
	}
	return connect(client, env)

}

func configureSshTransport(host string, user string, pass string) (*ssh.ClientConfig, error) {
	dirname, err := homedir.Dir()
	if err != nil {
		return nil, err
	}

	var methods []ssh.AuthMethod

	foundIdentityFile := false
	files := ssh_config.GetAll(host, "IdentityFile")
	for _, v := range files {
		//see https://github.com/docker/go-connections/pull/39#issuecomment-312765226
		if _, err := os.Stat(v); err == nil {
			methods, err = readPk(v, methods, dirname)
			if err != nil {
				return nil, err
			}
			foundIdentityFile = true
		}
	}

	if !foundIdentityFile {
		pkFilenames, err := ioutil.ReadDir(dirname + "/.ssh/")
		if err != nil {
			return nil, err
		}

		for _, pkFilename := range pkFilenames {
			if strings.Index(pkFilename.Name(), "id_") == 0 && !strings.HasSuffix(pkFilename.Name(), ".pub") {
				methods, err = readPk(pkFilename.Name(), methods, dirname+"/.ssh/")
				if err != nil {
					return nil, err
				}
			}
		}
	}
	if pass != "" {
		methods = append(methods, []ssh.AuthMethod{ssh.Password(pass)}...)
	}

	return &ssh.ClientConfig{
		User:            user,
		Auth:            methods,
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	}, nil
}

func readPk(pkFilename string, auth []ssh.AuthMethod, dirname string) ([]ssh.AuthMethod, error) {
	pk, err := ioutil.ReadFile(dirname + pkFilename)
	if err != nil {
		return nil, nil
	}
	signer, err := ssh.ParsePrivateKey(pk)
	if err != nil {
		return nil, err
	}
	auth = append(auth, []ssh.AuthMethod{ssh.PublicKeys(signer)}...)
	return auth, nil
}

func connectSshTransport(host string, path string, sshConfig *ssh.ClientConfig) (net.Conn, error) {
	remoteConn, err := net.Dial("tcp", host)
	if err != nil {
		return nil, err
	}

	ncc, chans, reqs, err := ssh.NewClientConn(remoteConn, "", sshConfig)

	if err != nil {
		return nil, err
	}

	sClient := ssh.NewClient(ncc, chans, reqs)
	c, err := sClient.Dial("unix", path)
	if err != nil {
		return nil, err
	}

	return c, nil
}
