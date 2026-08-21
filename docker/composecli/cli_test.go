package composecli

import (
	"context"
	"errors"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

// newFakeDocker writes a fake `docker` executable that appends its arguments,
// one invocation per line, to a file, then runs the given shell body. It
// prepends the fake to PATH for the duration of the test and returns the path
// of the arguments file.
func newFakeDocker(t *testing.T, body string) string {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip("fake-binary harness needs a POSIX shell")
	}
	dir := t.TempDir()
	argsFile := filepath.Join(dir, "args")
	script := "#!/bin/sh\nprintf '%s\\n' \"$*\" >> " + argsFile + "\n" + body
	if err := os.WriteFile(filepath.Join(dir, "docker"), []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))
	return argsFile
}

func TestDetect_ReportsVersion(t *testing.T) {
	newFakeDocker(t, "echo 'Docker Compose version v5.3.0'\n")

	cli, err := Detect(Options{})
	if err != nil {
		t.Fatalf("expected detection to succeed, got %v", err)
	}
	if !strings.Contains(cli.Version(), "v5.3.0") {
		t.Fatalf("expected the reported version, got %q", cli.Version())
	}
}

func TestDetect_MissingPluginIsAnError(t *testing.T) {
	newFakeDocker(t, "echo 'docker: unknown command: docker compose' >&2\nexit 1\n")

	if _, err := Detect(Options{}); err == nil {
		t.Fatal("expected an error when the compose plugin is absent")
	}
}

func TestDetect_AsksForTheComposeVersion(t *testing.T) {
	argsFile := newFakeDocker(t, "echo v5.3.0\n")

	if _, err := Detect(Options{}); err != nil {
		t.Fatal(err)
	}
	recorded, err := os.ReadFile(argsFile)
	if err != nil {
		t.Fatal(err)
	}
	if got := strings.TrimSpace(string(recorded)); got != "compose version" {
		t.Fatalf("expected `docker compose version`, got %q", got)
	}
}

func TestArgs_ProjectTargeting(t *testing.T) {
	cli := &CLI{}
	p := Project{
		Name:       "web",
		WorkingDir: "/srv/web",
		Files:      []string{"/srv/web/compose.yaml", "/srv/web/override.yaml"},
	}

	got := strings.Join(cli.args(p, "up", "-d"), " ")
	want := "compose --project-directory /srv/web " +
		"-f /srv/web/compose.yaml -f /srv/web/override.yaml " +
		"-p web up -d"
	if got != want {
		t.Fatalf("argv mismatch\n got: %s\nwant: %s", got, want)
	}
}

func TestArgs_OmitsUnknownFields(t *testing.T) {
	cli := &CLI{}
	got := strings.Join(cli.args(Project{Name: "solo"}, "down"), " ")
	if got != "compose -p solo down" {
		t.Fatalf("expected no --project-directory or -f flags, got %q", got)
	}
}

func TestUp_StreamsOutputAndTargetsServices(t *testing.T) {
	argsFile := newFakeDocker(t, "echo 'Container web-1  Started'\n")
	cli := &CLI{}

	r, err := cli.Up(context.Background(), Project{Name: "web"}, "api")
	if err != nil {
		t.Fatal(err)
	}
	out, err := io.ReadAll(r)
	if err != nil {
		t.Fatal(err)
	}
	if err := r.Close(); err != nil {
		t.Fatalf("close: %v", err)
	}
	if !strings.Contains(string(out), "Started") {
		t.Fatalf("expected streamed output, got %q", out)
	}
	recorded, _ := os.ReadFile(argsFile)
	if got := strings.TrimSpace(string(recorded)); got != "compose -p web up -d api" {
		t.Fatalf("wrong argv: %q", got)
	}
}

func TestUp_NonZeroExitSurfacesAsCloseError(t *testing.T) {
	newFakeDocker(t, "echo 'service \"api\" has no image' >&2\nexit 1\n")
	cli := &CLI{}

	r, err := cli.Up(context.Background(), Project{Name: "web"})
	if err != nil {
		t.Fatal(err)
	}
	out, _ := io.ReadAll(r)
	closeErr := r.Close()
	if !strings.Contains(string(out), "no image") {
		t.Fatalf("expected compose stderr in the stream, got %q", out)
	}
	if closeErr == nil {
		t.Fatal("expected a non-zero exit to be reported on Close")
	}
}

// TestClose_CancelsALongRunningProcess proves cancellation on a process that
// is provably running (we read its readiness line first, rather than racing
// Close against /bin/sh even starting) and that outlives the killed child:
// the fake backgrounds a grandchild that inherits the child's stdout before
// running its own long-lived foreground command, mirroring a real compose
// subprocess (a credential helper, a buildx worker) that keeps that pipe's
// write end open after the immediate `docker` process is gone. Without a
// WaitDelay bound, Close hangs forever waiting for that fd to close; with
// one, it returns within the grace period.
func TestClose_CancelsALongRunningProcess(t *testing.T) {
	newFakeDocker(t, "echo ready\nsleep 30 &\nsleep 30\n")
	cli := &CLI{}

	r, err := cli.Up(context.Background(), Project{Name: "web"})
	if err != nil {
		t.Fatal(err)
	}
	buf := make([]byte, 64)
	n, err := r.Read(buf)
	if err != nil {
		t.Fatalf("expected to read the readiness line, got err=%v", err)
	}
	if !strings.Contains(string(buf[:n]), "ready") {
		t.Fatalf("expected the readiness line, got %q", buf[:n])
	}

	done := make(chan struct{})
	go func() { _ = r.Close(); close(done) }()
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("Close did not cancel the compose process")
	}
}

// TestClose_ConcurrentReaderObservesCleanEOF exercises the documented usage:
// one goroutine streams output while another goroutine's Close ends it. The
// pipe's read side can only ever be closed by this package's own Close, so a
// concurrent Read unblocked by it must see a plain io.EOF (io.ReadAll returns
// a nil error), never the pipe's generic "closed" error, which would be
// indistinguishable from a genuine stream failure.
func TestClose_ConcurrentReaderObservesCleanEOF(t *testing.T) {
	newFakeDocker(t, "echo ready\nsleep 30\n")
	cli := &CLI{}

	r, err := cli.Up(context.Background(), Project{Name: "web"})
	if err != nil {
		t.Fatal(err)
	}

	// Drain the readiness line synchronously so the process is known to be
	// live before another goroutine takes over reading, same as above.
	buf := make([]byte, 64)
	if _, err := r.Read(buf); err != nil {
		t.Fatalf("expected to read the readiness line, got err=%v", err)
	}

	readErr := make(chan error, 1)
	readerStarted := make(chan struct{})
	go func() {
		close(readerStarted)
		_, err := io.ReadAll(r)
		readErr <- err
	}()
	<-readerStarted
	time.Sleep(50 * time.Millisecond) // let the goroutine block inside Read

	closeErr := make(chan error, 1)
	go func() { closeErr <- r.Close() }()
	select {
	case err := <-closeErr:
		if err != nil {
			t.Fatalf("close: %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("Close did not return")
	}

	select {
	case err := <-readErr:
		if err != nil {
			t.Fatalf("expected the concurrent reader to see a clean EOF, got %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("concurrent reader never unblocked")
	}
}

// TestClassifyExit_* test the pure classification logic that decides what
// Close reports, directly against the three real process-exit shapes it must
// tell apart. Testing the process's actual exit state (rather than racing a
// live cancellation against a live exit, which is what the classifier itself
// exists to be immune to) keeps these deterministic.

func TestClassifyExit_SignalKillIsTreatedAsCancellation(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("signal semantics need a POSIX shell")
	}
	err := exec.Command("sh", "-c", "kill -KILL $$").Run()
	if err == nil {
		t.Fatal("expected the self-kill to produce a non-nil error")
	}
	if got := classifyExit(err); got != nil {
		t.Fatalf("expected a signal-killed process to classify as cancellation (nil), got %v", got)
	}
}

func TestClassifyExit_RealNonZeroExitIsReportedAsIs(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("shell semantics need a POSIX shell")
	}
	err := exec.Command("sh", "-c", "exit 7").Run()
	if err == nil {
		t.Fatal("expected a non-zero exit to produce a non-nil error")
	}
	if got := classifyExit(err); got != err {
		t.Fatalf("expected a genuine non-zero exit to be reported as-is, got %v (want %v)", got, err)
	}
}

func TestClassifyExit_WaitDelayAfterSuccessIsTreatedAsSuccess(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("shell semantics need a POSIX shell")
	}
	cmd := exec.Command("sh", "-c", "sleep 5 & exit 0")
	cmd.WaitDelay = 200 * time.Millisecond
	cmd.Stdout = io.Discard
	err := cmd.Run()
	if !errors.Is(err, exec.ErrWaitDelay) {
		t.Fatalf("expected ErrWaitDelay from the lingering grandchild, got %v", err)
	}
	if got := classifyExit(err); got != nil {
		t.Fatalf("expected a successful exit delayed only by WaitDelay to classify as success (nil), got %v", got)
	}
}

func TestClassifyExit_NilIsNil(t *testing.T) {
	if got := classifyExit(nil); got != nil {
		t.Fatalf("expected a nil wait error to classify as nil, got %v", got)
	}
}

func TestConfigHashes_ParsesTheFakeOutput(t *testing.T) {
	argsFile := newFakeDocker(t, "echo 'web abc123'\n")
	cli := &CLI{}

	got, err := cli.ConfigHashes(context.Background(), Project{Name: "web"})
	if err != nil {
		t.Fatal(err)
	}
	if got["web"] != "abc123" {
		t.Fatalf("expected the parsed hash, got %v", got)
	}
	recorded, _ := os.ReadFile(argsFile)
	if got := strings.TrimSpace(string(recorded)); got != "compose -p web config --hash=*" {
		t.Fatalf("wrong argv: %q", got)
	}
}

func TestResolveProject_UsesTheNameFromConfig(t *testing.T) {
	newFakeDocker(t, `echo '{"name":"custom-project-name","services":{"web":{}}}'`+"\n")
	cli := &CLI{}

	p, err := cli.ResolveProject(context.Background(), "/srv/app", []string{"/srv/app/compose.yaml"})
	if err != nil {
		t.Fatal(err)
	}
	if p.Name != "custom-project-name" {
		t.Fatalf("expected the resolved name, got %q", p.Name)
	}
	if p.WorkingDir != "/srv/app" || len(p.Files) != 1 {
		t.Fatalf("expected dir and files to be carried through, got %+v", p)
	}
}
