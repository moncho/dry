package docker

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/pem"
	"net"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/knownhosts"
)

// testHostAlias is a name that cannot appear in the test runner's ssh_config,
// so ssh_config lookups fall back to defaults.
const testHostAlias = "dry-ssh-test-host.invalid"

func TestResolveSSHEndpoint_CanonicalFormGetsDefaults(t *testing.T) {
	// The canonical DOCKER_HOST=ssh://user@host form must resolve to a
	// dialable endpoint: port 22 and the default remote Docker socket.
	hostURL, err := url.Parse("ssh://someone@" + testHostAlias)
	if err != nil {
		t.Fatal(err)
	}
	endpoint, err := resolveSSHEndpoint(hostURL)
	if err != nil {
		t.Fatal(err)
	}
	if endpoint.addr != testHostAlias+":22" {
		t.Fatalf("expected default port 22, got addr %q", endpoint.addr)
	}
	if endpoint.socket != defaultRemoteDockerSocket {
		t.Fatalf("expected default remote socket, got %q", endpoint.socket)
	}
	if endpoint.user != "someone" {
		t.Fatalf("expected URL user, got %q", endpoint.user)
	}
	if endpoint.alias != testHostAlias {
		t.Fatalf("expected alias %q, got %q", testHostAlias, endpoint.alias)
	}
}

func TestResolveSSHEndpoint_ExplicitValuesWin(t *testing.T) {
	hostURL, err := url.Parse("ssh://root@" + testHostAlias + ":2222/run/user/1000/docker.sock")
	if err != nil {
		t.Fatal(err)
	}
	endpoint, err := resolveSSHEndpoint(hostURL)
	if err != nil {
		t.Fatal(err)
	}
	if endpoint.addr != testHostAlias+":2222" {
		t.Fatalf("expected explicit port, got addr %q", endpoint.addr)
	}
	if endpoint.socket != "/run/user/1000/docker.sock" {
		t.Fatalf("expected explicit socket, got %q", endpoint.socket)
	}
}

func TestResolveSSHEndpoint_MissingHost(t *testing.T) {
	hostURL, err := url.Parse("ssh://")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := resolveSSHEndpoint(hostURL); err == nil {
		t.Fatal("expected an error for a DOCKER_HOST without a host")
	}
}

func TestSSHKeyPath(t *testing.T) {
	home := t.TempDir()
	if got := sshKeyPath("/abs/path/key", home); got != "/abs/path/key" {
		t.Fatalf("absolute path must be used as given, got %q", got)
	}
	if got := sshKeyPath("work_key", home); got != filepath.Join(home, "work_key") {
		t.Fatalf("relative path must resolve under home, got %q", got)
	}
	got := sshKeyPath("~/keys/k", home)
	if strings.HasPrefix(got, "~") || !strings.HasSuffix(got, filepath.Join("keys", "k")) {
		t.Fatalf("~ must be expanded, got %q", got)
	}
}

func writeTestKey(t *testing.T, dir, name string) (string, ssh.PublicKey) {
	t.Helper()
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	block, err := ssh.MarshalPrivateKey(priv, "")
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, pem.EncodeToMemory(block), 0o600); err != nil {
		t.Fatal(err)
	}
	sshPub, err := ssh.NewPublicKey(pub)
	if err != nil {
		t.Fatal(err)
	}
	return path, sshPub
}

func TestIdentityFileAuthMethods_AbsolutePathIsNotMangled(t *testing.T) {
	// Regression: an absolute IdentityFile used to be joined onto the home
	// directory (/home/me + /abs/key -> /home/me/abs/key), and the resulting
	// read failure returned nil, nil, wiping every accumulated auth method.
	dir := t.TempDir()
	keyPath, _ := writeTestKey(t, dir, "work_key")

	methods, found, err := identityFileAuthMethods([]string{keyPath}, t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if !found {
		t.Fatal("expected the configured identity file to be found")
	}
	if len(methods) != 1 {
		t.Fatalf("expected 1 auth method from the absolute identity file, got %d", len(methods))
	}
}

func TestIdentityFileAuthMethods_UnparseableKeyIsSkipped(t *testing.T) {
	dir := t.TempDir()
	goodPath, _ := writeTestKey(t, dir, "good_key")
	badPath := filepath.Join(dir, "bad_key")
	if err := os.WriteFile(badPath, []byte("not a private key"), 0o600); err != nil {
		t.Fatal(err)
	}

	methods, found, err := identityFileAuthMethods([]string{badPath, goodPath}, t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if !found {
		t.Fatal("expected identity files to be found")
	}
	if len(methods) != 1 {
		t.Fatalf("expected the unparseable key to be skipped and the good one kept, got %d methods", len(methods))
	}
}

func TestIdentityFileAuthMethods_UnreadableKeyIsAnError(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("file permissions are not enforced for root")
	}
	dir := t.TempDir()
	keyPath, _ := writeTestKey(t, dir, "locked_key")
	if err := os.Chmod(keyPath, 0o000); err != nil {
		t.Fatal(err)
	}

	if _, _, err := identityFileAuthMethods([]string{keyPath}, t.TempDir()); err == nil {
		t.Fatal("expected an error for an existing but unreadable identity file")
	}
}

func TestIdentityFileAuthMethods_MissingFilesAreSkipped(t *testing.T) {
	methods, found, err := identityFileAuthMethods([]string{"~/.ssh/identity"}, t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if found {
		t.Fatal("a nonexistent default identity file must not count as found")
	}
	if len(methods) != 0 {
		t.Fatalf("expected no methods, got %d", len(methods))
	}
}

func TestHostKeyCallbackFromFiles_NoFilesIsAnError(t *testing.T) {
	_, err := hostKeyCallbackFromFiles(nil)
	if err == nil {
		t.Fatal("expected an error when no known_hosts file exists")
	}
	if !strings.Contains(err.Error(), envSkipHostKeyCheck) {
		t.Fatalf("expected the error to name the opt-out variable, got %v", err)
	}
}

func TestHostKeyCallbackFromFiles_VerifiesAgainstKnownHosts(t *testing.T) {
	dir := t.TempDir()
	_, hostKey := writeTestKey(t, dir, "host_key")
	_, otherKey := writeTestKey(t, dir, "other_key")

	knownHosts := filepath.Join(dir, "known_hosts")
	line := knownhosts.Line([]string{"example.test:22"}, hostKey)
	if err := os.WriteFile(knownHosts, []byte(line+"\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	callback, err := hostKeyCallbackFromFiles([]string{knownHosts})
	if err != nil {
		t.Fatal(err)
	}

	remote := &net.TCPAddr{IP: net.ParseIP("192.0.2.1"), Port: 22}
	if err := callback("example.test:22", remote, hostKey); err != nil {
		t.Fatalf("expected the recorded host key to verify, got %v", err)
	}
	if err := callback("example.test:22", remote, otherKey); err == nil {
		t.Fatal("expected a mismatched host key to be rejected")
	} else if !strings.Contains(err.Error(), envSkipHostKeyCheck) {
		t.Fatalf("expected rejection to explain the opt-out, got %v", err)
	}
	if err := callback("unknown.test:22", remote, hostKey); err == nil {
		t.Fatal("expected an unknown host to be rejected")
	}
}

func TestHostKeyVerification_ExplicitOptOut(t *testing.T) {
	t.Setenv(envSkipHostKeyCheck, "1")
	callback, err := hostKeyVerification(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	_, anyKey := writeTestKey(t, dir, "any_key")
	remote := &net.TCPAddr{IP: net.ParseIP("192.0.2.1"), Port: 22}
	if err := callback("anything:22", remote, anyKey); err != nil {
		t.Fatalf("expected the explicit opt-out to accept any key, got %v", err)
	}
}

func TestHostKeyVerification_DefaultRequiresKnownHosts(t *testing.T) {
	// With no known_hosts anywhere under the given home and no opt-out,
	// verification must fail closed instead of silently accepting any host.
	if os.Getenv(envSkipHostKeyCheck) != "" {
		t.Skip("opt-out set in the environment")
	}
	if _, err := os.Stat("/etc/ssh/ssh_known_hosts"); err == nil {
		t.Skip("system known_hosts exists; the empty-home case cannot be isolated")
	}
	if _, err := hostKeyVerification(t.TempDir()); err == nil {
		t.Fatal("expected an error when no known_hosts exists and no opt-out is set")
	}
}
