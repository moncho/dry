package docker

import (
	"context"
	"fmt"
	"net"
	"net/url"
	"os"
	osuser "os/user"
	"path/filepath"
	"strings"
	"time"

	"github.com/docker/cli/opts"
	"github.com/kevinburke/ssh_config"
	homedir "github.com/mitchellh/go-homedir"
	"github.com/moby/moby/client"
	drytls "github.com/moncho/dry/tls"
	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/agent"
	"golang.org/x/crypto/ssh/knownhosts"
)

const (
	// DefaultConnectionTimeout is the timeout for connecting with the Docker daemon
	DefaultConnectionTimeout = 32 * time.Second
	// defaultRemoteDockerSocket is the Docker socket path used on the remote
	// host when the ssh:// URL does not name one.
	defaultRemoteDockerSocket = "/var/run/docker.sock"
	// envSkipHostKeyCheck disables SSH host key verification when set to "1".
	// Verification against known_hosts is the default.
	envSkipHostKeyCheck = "DRY_SSH_INSECURE_SKIP_HOST_KEY_CHECK"
)

var defaultDockerPath string

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

// ConnectToDaemon connects to a Docker daemon using the given properties.
func ConnectToDaemon(env Env) (*DockerDaemon, error) {
	host, err := getServerHost(env)
	if err != nil {
		return nil, fmt.Errorf("invalid host: %w", err)
	}
	var options *drytls.Options
	// If a path to certificates is given use the path to read certificates from
	if dockerCertPath := env.DockerCertPath; dockerCertPath != "" {
		options = &drytls.Options{
			CAFile:             filepath.Join(dockerCertPath, "ca.pem"),
			CertFile:           filepath.Join(dockerCertPath, "cert.pem"),
			KeyFile:            filepath.Join(dockerCertPath, "key.pem"),
			InsecureSkipVerify: !env.DockerTLSVerify,
		}
	} else if env.DockerTLSVerify {
		// No cert path is given but TLS verify is set, default location for
		// docker certs will be used.
		// See https://docs.docker.com/engine/security/https/#secure-by-default
		// Fixes #23
		options = &drytls.Options{
			CAFile:             filepath.Join(defaultDockerPath, "ca.pem"),
			CertFile:           filepath.Join(defaultDockerPath, "cert.pem"),
			KeyFile:            filepath.Join(defaultDockerPath, "key.pem"),
			InsecureSkipVerify: !env.DockerTLSVerify,
		}
		env.DockerCertPath = defaultDockerPath
	}

	var clientOpts []client.Opt
	if options != nil {
		clientOpts = append(clientOpts, client.WithTLSClientConfig(options.CAFile, options.CertFile, options.KeyFile))
	}

	if host != "" && strings.HasPrefix(host, "ssh") {
		// if it starts with ssh, its an ssh connection, and we need to handle this specially
		// github.com/docker/docker does not handle ssh, as an upgrade to go-connections need to be made
		// see https://github.com/docker/go-connections/pull/39
		hostURL, err := url.Parse(host)
		if err != nil {
			return nil, err
		}

		endpoint, err := resolveSSHEndpoint(hostURL)
		if err != nil {
			return nil, err
		}
		pass, _ := hostURL.User.Password()
		sshConfig, err := configureSSHTransport(endpoint, pass)
		if err != nil {
			return nil, err
		}
		clientOpts = append(clientOpts, client.WithDialContext(
			func(ctx context.Context, network, addr string) (net.Conn, error) {
				return connectSSHTransport(endpoint, sshConfig)
			}))
	} else if host != "" {
		// default uses the docker library to connect to hosts
		clientOpts = append(clientOpts, client.WithHost(host))
	}

	apiClient, err := client.New(clientOpts...)
	if err != nil {
		return nil, fmt.Errorf("create Docker client: %w", err)
	}
	return connect(apiClient, env)
}

// sshEndpoint describes a Docker daemon reachable over SSH.
type sshEndpoint struct {
	alias  string // host as written in DOCKER_HOST, used for ssh_config lookups
	addr   string // resolved host:port to dial
	user   string
	socket string // Docker socket path on the remote host
}

// resolveSSHEndpoint turns an ssh:// DOCKER_HOST URL into a dialable
// endpoint, applying the same defaults the ssh command would: Hostname,
// Port, and User from ssh_config, port 22, the current OS user, and the
// default Docker socket path. This makes the canonical ssh://user@host
// form work instead of requiring ssh://user@host:22/var/run/docker.sock.
func resolveSSHEndpoint(hostURL *url.URL) (sshEndpoint, error) {
	alias := hostURL.Hostname()
	if alias == "" {
		return sshEndpoint{}, fmt.Errorf("DOCKER_HOST %q has no host", hostURL)
	}

	host := alias
	if configured := ssh_config.Get(alias, "Hostname"); configured != "" {
		host = configured
	}
	port := hostURL.Port()
	if port == "" {
		port = ssh_config.Get(alias, "Port")
	}
	if port == "" {
		port = "22"
	}
	user := hostURL.User.Username()
	if user == "" {
		user = ssh_config.Get(alias, "User")
	}
	if user == "" {
		if current, err := osuser.Current(); err == nil {
			user = current.Username
		}
	}
	socket := hostURL.Path
	if socket == "" {
		socket = defaultRemoteDockerSocket
	}

	return sshEndpoint{
		alias:  alias,
		addr:   net.JoinHostPort(host, port),
		user:   user,
		socket: socket,
	}, nil
}

func configureSSHTransport(endpoint sshEndpoint, pass string) (*ssh.ClientConfig, error) {
	home, err := homedir.Dir()
	if err != nil {
		return nil, err
	}

	methods, foundIdentityFile, err := identityFileAuthMethods(ssh_config.GetAll(endpoint.alias, "IdentityFile"), home)
	if err != nil {
		return nil, err
	}

	if !foundIdentityFile {
		fallback, err := defaultKeyAuthMethods(filepath.Join(home, ".ssh"))
		if err != nil {
			return nil, err
		}
		methods = append(methods, fallback...)
	}
	if agentMethod, ok := sshAgentAuthMethod(); ok {
		methods = append(methods, agentMethod)
	}
	if pass != "" {
		methods = append(methods, ssh.Password(pass))
	}
	if len(methods) == 0 {
		return nil, fmt.Errorf(
			"no usable SSH auth methods for %s: configure an IdentityFile in ~/.ssh/config, create an ~/.ssh/id_* key, run an SSH agent, or set a password in DOCKER_HOST",
			endpoint.alias)
	}

	hostKeyCallback, err := hostKeyVerification(home)
	if err != nil {
		return nil, err
	}

	return &ssh.ClientConfig{
		User:            endpoint.user,
		Auth:            methods,
		HostKeyCallback: hostKeyCallback,
		Timeout:         DefaultConnectionTimeout,
	}, nil
}

// identityFileAuthMethods loads the private keys configured for the host in
// ssh_config. Reported files that do not exist are skipped (ssh_config
// returns defaults like ~/.ssh/identity even when nothing is configured); a
// file that exists but cannot be read is an error; a key that cannot be
// parsed (e.g. passphrase protected) is skipped so other methods still work.
func identityFileAuthMethods(files []string, home string) ([]ssh.AuthMethod, bool, error) {
	var methods []ssh.AuthMethod
	found := false
	for _, file := range files {
		path := sshKeyPath(file, home)
		if _, err := os.Stat(path); err != nil {
			continue
		}
		found = true
		pk, err := os.ReadFile(path)
		if err != nil {
			return nil, found, fmt.Errorf("read SSH identity file %s: %w", path, err)
		}
		signer, err := ssh.ParsePrivateKey(pk)
		if err != nil {
			// Passphrase-protected or unparseable key material: skip it,
			// the agent or other methods may still authenticate.
			continue
		}
		methods = append(methods, ssh.PublicKeys(signer))
	}
	return methods, found, nil
}

// defaultKeyAuthMethods scans ~/.ssh for id_* private keys, skipping any
// that cannot be read or parsed.
func defaultKeyAuthMethods(sshDir string) ([]ssh.AuthMethod, error) {
	entries, err := os.ReadDir(sshDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var methods []ssh.AuthMethod
	for _, entry := range entries {
		name := entry.Name()
		if !strings.HasPrefix(name, "id_") || strings.HasSuffix(name, ".pub") {
			continue
		}
		method, err := readPk(filepath.Join(sshDir, name))
		if err != nil {
			continue
		}
		methods = append(methods, method)
	}
	return methods, nil
}

// sshAgentAuthMethod exposes the keys held by a running SSH agent, which
// covers passphrase-protected keys that cannot be parsed from disk.
func sshAgentAuthMethod() (ssh.AuthMethod, bool) {
	sock := os.Getenv("SSH_AUTH_SOCK")
	if sock == "" {
		return nil, false
	}
	conn, err := net.Dial("unix", sock)
	if err != nil {
		return nil, false
	}
	return ssh.PublicKeysCallback(agent.NewClient(conn).Signers), true
}

// sshKeyPath resolves an identity file path the way ssh does: ~ expands to
// the home directory, absolute paths are used as given, and relative paths
// are taken relative to the home directory.
func sshKeyPath(path, home string) string {
	if strings.HasPrefix(path, "~") {
		if expanded, err := homedir.Expand(path); err == nil {
			return expanded
		}
	}
	if filepath.IsAbs(path) {
		return path
	}
	return filepath.Join(home, path)
}

func readPk(path string) (ssh.AuthMethod, error) {
	pk, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	signer, err := ssh.ParsePrivateKey(pk)
	if err != nil {
		return nil, err
	}
	return ssh.PublicKeys(signer), nil
}

// hostKeyVerification returns a host key callback backed by the standard
// known_hosts files. Verification can be disabled explicitly by setting
// DRY_SSH_INSECURE_SKIP_HOST_KEY_CHECK=1; it is never disabled silently.
func hostKeyVerification(home string) (ssh.HostKeyCallback, error) {
	if os.Getenv(envSkipHostKeyCheck) == "1" {
		return ssh.InsecureIgnoreHostKey(), nil // #nosec G106 -- explicit opt-out
	}
	var files []string
	for _, path := range []string{
		filepath.Join(home, ".ssh", "known_hosts"),
		"/etc/ssh/ssh_known_hosts",
	} {
		if _, err := os.Stat(path); err == nil {
			files = append(files, path)
		}
	}
	return hostKeyCallbackFromFiles(files)
}

func hostKeyCallbackFromFiles(files []string) (ssh.HostKeyCallback, error) {
	if len(files) == 0 {
		return nil, fmt.Errorf(
			"no known_hosts file found: connect once with ssh to record the host key, or set %s=1 to skip verification",
			envSkipHostKeyCheck)
	}
	callback, err := knownhosts.New(files...)
	if err != nil {
		return nil, fmt.Errorf("parse known_hosts: %w", err)
	}
	return func(hostname string, remote net.Addr, key ssh.PublicKey) error {
		if err := callback(hostname, remote, key); err != nil {
			return fmt.Errorf(
				"SSH host key verification for %s failed: %w (connect once with ssh to record the key, or set %s=1 to skip verification)",
				hostname, err, envSkipHostKeyCheck)
		}
		return nil
	}, nil
}

func connectSSHTransport(endpoint sshEndpoint, sshConfig *ssh.ClientConfig) (net.Conn, error) {
	remoteConn, err := net.Dial("tcp", endpoint.addr)
	if err != nil {
		return nil, err
	}

	// The address is what the host key callback verifies against, so it must
	// be the real dial target, never empty.
	ncc, chans, reqs, err := ssh.NewClientConn(remoteConn, endpoint.addr, sshConfig)
	if err != nil {
		_ = remoteConn.Close()
		return nil, err
	}

	sClient := ssh.NewClient(ncc, chans, reqs)
	c, err := sClient.Dial("unix", endpoint.socket)
	if err != nil {
		_ = sClient.Close()
		return nil, err
	}

	return c, nil
}
