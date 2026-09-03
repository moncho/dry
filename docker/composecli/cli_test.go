package composecli

import (
	"bytes"
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

	_, err := Detect(Options{})
	if err == nil {
		t.Fatal("expected an error when the compose plugin is absent")
	}
	// And it must not be reported as a timeout: both leave no engine, but
	// only one is fixed by installing the plugin, and telling a user their
	// probe timed out when docker answered immediately sends them nowhere.
	if errors.Is(err, ErrProbeTimeout) {
		t.Fatalf("expected a missing plugin, not a probe timeout: %v", err)
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
	newFakeDocker(t, "echo ready\nsleep 5 &\nsleep 5\n")
	cli := &CLI{opts: Options{CancelGrace: 300 * time.Millisecond}}

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
	newFakeDocker(t, "echo ready\nsleep 5\n")
	cli := &CLI{opts: Options{CancelGrace: 300 * time.Millisecond}}

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

// --- Read-only calls are bounded ------------------------------------------
//
// ConfigHashes and ResolveProject run on dry's compose refresh cycle, and
// the model gates the next cycle on the current one finishing, so a compose
// that never returns used to gate refreshes for the rest of the session.
// Config is not polled, the c key waits on it, but it is bounded too.
// Each of these drives a `docker` that outlives its bound and asserts the
// call comes back on its own, naming the timeout.

// hangingDocker installs a `docker` that outlives every bound in these
// tests, and a CLI whose read-only calls are bounded in milliseconds.
func hangingDocker(t *testing.T) *CLI {
	t.Helper()
	// sleep 5 rather than a long sleep: a killed child still outlives the
	// bound by far, and the process is gone soon after the test instead of
	// lingering as an orphan for the rest of the run.
	newFakeDocker(t, "sleep 5\n")
	return &CLI{opts: Options{
		ReadTimeout: 200 * time.Millisecond,
		CancelGrace: 300 * time.Millisecond,
	}}
}

func TestConfig_TimesOutRatherThanHanging(t *testing.T) {
	cli := hangingDocker(t)

	start := time.Now()
	out, err := cli.Config(context.Background(), Project{Name: "web"})
	if err == nil {
		t.Fatalf("expected a timeout error, got output %q", out)
	}
	if !strings.Contains(err.Error(), "timed out") {
		t.Fatalf("expected the error to say it timed out, got %v", err)
	}
	if elapsed := time.Since(start); elapsed > 5*time.Second {
		t.Fatalf("expected the call to return promptly, took %s", elapsed)
	}
}

// The message has to name the compose call that hung, because it surfaces in
// dry's status bar with no other context: "docker compose config --hash=*
// timed out after 200ms".
func TestConfigHashes_TimeoutNamesTheVerb(t *testing.T) {
	cli := hangingDocker(t)

	_, err := cli.ConfigHashes(context.Background(), Project{Name: "web"})
	if err == nil {
		t.Fatal("expected a timeout error")
	}
	if !strings.Contains(err.Error(), "config --hash=*") {
		t.Fatalf("expected the error to name the compose call, got %v", err)
	}
	if !strings.Contains(err.Error(), "200ms") {
		t.Fatalf("expected the error to name the bound, got %v", err)
	}
}

func TestResolveProject_TimesOutRatherThanHanging(t *testing.T) {
	cli := hangingDocker(t)

	if _, err := cli.ResolveProject(context.Background(), "/srv/app", []string{"/srv/app/compose.yaml"}); err == nil {
		t.Fatal("expected a timeout error")
	} else if !strings.Contains(err.Error(), "timed out") {
		t.Fatalf("expected the error to say it timed out, got %v", err)
	}
}

// Detect runs once at startup. A `docker` that hangs there would otherwise
// leave compose support undetected for the whole session, with the keys
// silently reporting the plugin as missing and nothing saying why.
func TestDetect_TimesOutRatherThanHanging(t *testing.T) {
	newFakeDocker(t, "sleep 5\n")

	start := time.Now()
	_, err := Detect(Options{
		ReadTimeout: 200 * time.Millisecond,
		CancelGrace: 300 * time.Millisecond,
	})
	if err == nil {
		t.Fatal("expected a timeout error")
	}
	if !errors.Is(err, ErrProbeTimeout) {
		t.Fatalf("expected the error to identify the probe timeout, got %v", err)
	}
	if elapsed := time.Since(start); elapsed > 5*time.Second {
		t.Fatalf("expected detection to give up promptly, took %s", elapsed)
	}
}

// The write actions must NOT inherit the bound: a pull or a build is
// legitimately slower than any read timeout, streams its progress into the
// viewer, and is cancelled by closing it. A stream that outlives the read
// timeout by an order of magnitude keeps streaming.
func TestUp_IsNotBoundedByTheReadTimeout(t *testing.T) {
	// A second line well past the read timeout: the point is that the
	// stream is still delivering output then, which a Close returning nil
	// does not show, since classifyExit maps a deadline-killed process to
	// nil as well.
	newFakeDocker(t, "echo ready\nsleep 1\necho still-here\nsleep 5\n")
	cli := &CLI{opts: Options{
		ReadTimeout: 50 * time.Millisecond,
		CancelGrace: 300 * time.Millisecond,
	}}

	r, err := cli.Up(context.Background(), Project{Name: "web"})
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = r.Close() }()

	buf := make([]byte, 64)
	n, err := r.Read(buf)
	if err != nil {
		t.Fatalf("expected to read the readiness line, got err=%v", err)
	}
	if !strings.Contains(string(buf[:n]), "ready") {
		t.Fatalf("expected the readiness line, got %q", buf[:n])
	}

	// Twenty times the read timeout later, the stream is still delivering.
	deadline := time.Now().Add(4 * time.Second)
	for {
		n, err := r.Read(buf)
		if err != nil {
			t.Fatalf("stream ended after the read timeout: %v", err)
		}
		if strings.Contains(string(buf[:n]), "still-here") {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("never saw the line written after the read timeout")
		}
	}
	if err := r.Close(); err != nil {
		t.Fatalf("expected up to still be running past the read timeout, got %v", err)
	}
}

// A grandchild that inherited the child's stdout and outlived it holds the
// pipe's write end open, so cmd.Wait blocks after the child is gone. The
// WaitDelay bound ends that wait, and the result is still the command's
// complete output, because exec reports ErrWaitDelay only for a command that
// itself exited successfully, by which time the draining copy has had the
// whole grace period to finish. Reporting a failure there would blank SYNC
// and pin an error banner to the screen every refresh cycle over a call that
// worked, which is what a lingering credential helper would do.
func TestConfigHashes_LingeringGrandchildStillReturnsItsResult(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("fake-binary harness needs a POSIX shell")
	}
	newFakeDocker(t, "sleep 5 &\necho 'web abc123'\n")
	cli := &CLI{opts: Options{CancelGrace: 300 * time.Millisecond}}

	start := time.Now()
	hashes, err := cli.ConfigHashes(context.Background(), Project{Name: "web"})
	if err != nil {
		t.Fatalf("expected the complete output despite the held-open pipe, got %v", err)
	}
	if hashes["web"] != "abc123" {
		t.Fatalf("expected the parsed hash, got %v", hashes)
	}
	if elapsed := time.Since(start); elapsed > 5*time.Second {
		t.Fatalf("expected the WaitDelay bound to end the wait, took %s", elapsed)
	}
}

// TestDetect_LingeringGrandchildStillDetects is the same shape on the probe:
// a successful `docker compose version` whose output pipe a bystander held
// open must not disable compose for the whole session.
func TestDetect_LingeringGrandchildStillDetects(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("fake-binary harness needs a POSIX shell")
	}
	newFakeDocker(t, "sleep 5 &\necho 'Docker Compose version v5.3.0'\n")

	start := time.Now()
	cli, err := Detect(Options{CancelGrace: 300 * time.Millisecond})
	if err != nil {
		t.Fatalf("expected detection to succeed despite the held-open pipe, got %v", err)
	}
	if !strings.Contains(cli.Version(), "v5.3.0") {
		t.Fatalf("expected the reported version, got %q", cli.Version())
	}
	// The bound is what makes this quick: without it CombinedOutput waits
	// for the grandchild to close the pipe, which is the whole five seconds.
	if elapsed := time.Since(start); elapsed > 3*time.Second {
		t.Fatalf("expected the WaitDelay bound to end the wait, took %s", elapsed)
	}
}

// A compose that fails on its own has a real diagnosis on stderr, and
// reporting that as a timeout would hide a broken compose file behind a
// message inviting the user to wait longer. Which branch runs is decided by
// whether this package killed the process, recorded by killSwitch, not by
// the clock. This pins the ordinary case, where compose fails and nothing
// killed it; a failure landing between the process exiting and being reaped
// is still reported as a timeout, with its stderr appended.
func TestOutput_ARealFailureReportsComposesDiagnosis(t *testing.T) {
	newFakeDocker(t, "echo 'service \"web\" has neither an image nor a build context' >&2\nexit 1\n")
	cli := &CLI{opts: Options{
		ReadTimeout: 3 * time.Second,
		CancelGrace: 300 * time.Millisecond,
	}}

	_, err := cli.ConfigHashes(context.Background(), Project{Name: "web"})
	if err == nil {
		t.Fatal("expected the compose failure to be reported")
	}
	// Neither branch this package owns may claim it: the process exited on
	// its own terms, so the diagnosis is compose's, not ours.
	for _, ours := range []string{"timed out", "was cancelled"} {
		if strings.Contains(err.Error(), ours) {
			t.Fatalf("expected compose's own diagnosis, got %q: %v", ours, err)
		}
	}
	if !strings.Contains(err.Error(), "neither an image") {
		t.Fatalf("expected compose's stderr, got %v", err)
	}
}

// A process dry killed never got to report its own verdict, so the error is
// the deadline or the cancellation, whichever applied. Whatever it had
// written to stderr by then says where it got stuck, and both branches carry
// that along as a hint, on one line.
//
// A process dry kills still reports what compose had said before it died.
// The fake docker writes to stderr and only then creates the ready file, so
// the cancel below cannot land before that write: no sleep is timing this.
// This is the cancellation branch; the deadline branch carries the same
// stderr through the same hint call, asserted below.
func TestOutput_AKilledProcessCarriesWhatComposeHadSaid(t *testing.T) {
	dir := t.TempDir()
	ready := filepath.Join(dir, "ready")
	newFakeDocker(t, "echo 'resolving image tag' >&2\ntouch "+ready+"\nsleep 5\n")
	cli := &CLI{opts: Options{CancelGrace: 300 * time.Millisecond}}

	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		for {
			if _, err := os.Stat(ready); err == nil {
				cancel()
				return
			}
			time.Sleep(time.Millisecond)
		}
	}()
	_, err := cli.ConfigHashes(ctx, Project{Name: "web"})
	if err == nil {
		t.Fatal("expected the killed call to report an error")
	}
	if !strings.Contains(err.Error(), "resolving image tag") {
		t.Fatalf("expected what compose had said, got %v", err)
	}
	if strings.Contains(err.Error(), "\n") {
		t.Fatalf("expected a single-line error, got %q", err.Error())
	}
}

// hint is what carries that text, and it is the piece that has to keep the
// message on one line: the status bar is one line in the layout's budget.
func TestHint(t *testing.T) {
	cases := []struct{ name, stderr, want string }{
		{"empty", "", ""},
		{"whitespace only", " \n\t ", ""},
		{"one line", "no such file", ": no such file"},
		{"trims the trailing newline", "no such file\n", ": no such file"},
		{"flattens", "line one\nline two", ": line one; line two"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var buf bytes.Buffer
			buf.WriteString(tc.stderr)
			if got := hint(&buf); got != tc.want {
				t.Errorf("hint(%q) = %q, want %q", tc.stderr, got, tc.want)
			}
		})
	}
}

// killSwitch is the discriminator that decides between "compose failed" and
// "we killed it". It replaced an exit-status test, which cannot carry that
// distinction on every platform: a signalled process has no exit code on
// POSIX, but Windows kills with TerminateProcess(h, 1), indistinguishable
// from compose choosing to exit 1. The flag is set where the kill is
// requested rather than read back off the corpse, so the mechanism is
// platform-independent even though these tests need a POSIX shell to drive
// it and skip without one.

func TestKillSwitch_FlipsWhenTheContextKillsTheProcess(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cmd := exec.CommandContext(ctx, "sleep", "5")
	killed := killSwitch(cmd)
	if err := cmd.Start(); err != nil {
		t.Skipf("no sleep binary here: %v", err)
	}
	if killed.Load() {
		t.Fatal("expected the flag to be clear before the cancel")
	}
	cancel()
	_ = cmd.Wait()
	if !killed.Load() {
		t.Fatal("expected the flag to be set by the context kill")
	}
}

// exec calls Cancel for a context that fires after the process has already
// finished, and Kill then reports ErrProcessDone. Counting that as a kill is
// what made a compose failure racing the deadline report as a timeout.
func TestKillSwitch_StaysClearWhenTheProcessAlreadyFinished(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	cmd := exec.CommandContext(ctx, "sh", "-c", "exit 1")
	killed := killSwitch(cmd)
	err := cmd.Run()
	if err != nil && errors.As(err, new(*exec.Error)) {
		t.Skipf("no POSIX shell here: %v", err)
	}
	if err == nil {
		t.Fatal("expected the non-zero exit to produce an error")
	}
	// What exec does when the context fires late.
	if err := cmd.Cancel(); !errors.Is(err, os.ErrProcessDone) {
		t.Fatalf("expected ErrProcessDone from a finished process, got %v", err)
	}
	if killed.Load() {
		t.Fatal("a kill that found nothing to kill must not count as one")
	}
}

// CommandContext installs its own Cancel; killSwitch must not drop it, or
// the deadline would stop killing anything.
func TestKillSwitch_KeepsTheCancelItWraps(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()
	cmd := exec.CommandContext(ctx, "sleep", "5")
	killed := killSwitch(cmd)
	start := time.Now()
	err := cmd.Run()
	if err != nil && errors.As(err, new(*exec.Error)) {
		t.Skipf("no sleep binary here: %v", err)
	}
	if err == nil {
		t.Fatal("expected the killed process to report an error")
	}
	if elapsed := time.Since(start); elapsed > 2*time.Second {
		t.Fatalf("the process outlived its deadline by %s: the wrapped Cancel was dropped", elapsed)
	}
	if !killed.Load() {
		t.Fatal("expected the flag to be set")
	}
}

// A caller that cancels its context gets a sentence, not the raw
// "signal: killed" the exec package reports for a process dry killed.
func TestOutput_ParentCancellationSaysSo(t *testing.T) {
	newFakeDocker(t, "sleep 5\n")
	cli := &CLI{opts: Options{CancelGrace: 300 * time.Millisecond}}

	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		time.Sleep(100 * time.Millisecond)
		cancel()
	}()
	_, err := cli.ConfigHashes(ctx, Project{Name: "web"})
	if err == nil {
		t.Fatal("expected the cancelled call to report an error")
	}
	if !strings.Contains(err.Error(), "cancelled") {
		t.Fatalf("expected the error to say it was cancelled, got %v", err)
	}
	if strings.Contains(err.Error(), "signal") {
		t.Fatalf("expected no raw signal text, got %v", err)
	}
}

// When the caller's own deadline is the one that fires, the reported bound
// has to be the caller's: printing this package's default would name a
// number that never applied.
func TestOutput_ReportsTheBoundThatActuallyApplied(t *testing.T) {
	newFakeDocker(t, "sleep 5\n")
	cli := &CLI{opts: Options{CancelGrace: 300 * time.Millisecond}}

	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()
	_, err := cli.ConfigHashes(ctx, Project{Name: "web"})
	if err == nil {
		t.Fatal("expected a timeout error")
	}
	if !strings.Contains(err.Error(), "timed out") {
		t.Fatalf("expected a timeout, got %v", err)
	}
	// The caller's bound, not the package default. A narrow range rather
	// than the literal "200ms": output clamps to time.Until(deadline) on
	// entry, so scheduling between the caller's WithTimeout and that line
	// can round it to 199ms. Wide enough for that, tight enough that a
	// halved or tripled bound fails.
	bound := reportedBound(t, err.Error())
	if bound < 190*time.Millisecond || bound > 200*time.Millisecond {
		t.Fatalf("expected the caller's 200ms bound, got %s from %v", bound, err)
	}
	if bound == composeReadTimeout {
		t.Fatalf("expected the caller's bound, not the package default, got %v", err)
	}
}

// The polled calls and the user-initiated one are bounded differently on
// purpose: ConfigHashes runs on every refresh cycle and is retried, while
// `config` is a key the user pressed and waited on.
func TestTimeouts_PolledAndUserInitiatedDiffer(t *testing.T) {
	if composeConfigTimeout <= composeReadTimeout {
		t.Fatalf("expected the user-initiated bound to be the longer one, got config=%s read=%s",
			composeConfigTimeout, composeReadTimeout)
	}
	cli := &CLI{}
	if got := cli.timeout(composeConfigTimeout); got != composeConfigTimeout {
		t.Fatalf("expected the given default, got %s", got)
	}
	override := &CLI{opts: Options{ReadTimeout: 5 * time.Millisecond}}
	for _, base := range []time.Duration{composeReadTimeout, composeConfigTimeout} {
		if got := override.timeout(base); got != 5*time.Millisecond {
			t.Fatalf("expected the override to win for base %s, got %s", base, got)
		}
	}
}

// Comparing the constants proves nothing about which call uses which: with
// only ReadTimeout to set, both collapse to one value and swapping the two
// bases at their call sites goes unnoticed. ConfigTimeout separates them, so
// this is the test that fails if Config is ever given the polled bound.
func TestConfig_UsesTheUserInitiatedBoundAndConfigHashesDoesNot(t *testing.T) {
	hangingDocker(t)
	cli := &CLI{opts: Options{
		ReadTimeout:   50 * time.Millisecond,
		ConfigTimeout: 900 * time.Millisecond,
		CancelGrace:   50 * time.Millisecond,
	}}

	start := time.Now()
	if _, err := cli.ConfigHashes(context.Background(), Project{Name: "web"}); err == nil {
		t.Fatal("expected ConfigHashes to time out")
	}
	polled := time.Since(start)

	start = time.Now()
	if _, err := cli.Config(context.Background(), Project{Name: "web"}); err == nil {
		t.Fatal("expected Config to time out")
	}
	user := time.Since(start)

	if polled > 400*time.Millisecond {
		t.Errorf("ConfigHashes took %s: it used the user-initiated bound", polled)
	}
	if user < 500*time.Millisecond {
		t.Errorf("Config returned after %s: it used the polled bound", user)
	}

	// ResolveProject runs on the same polled cycle as ConfigHashes, so it
	// takes the same bound. Given the user-initiated one it would gate every
	// refresh for a minute, which is the failure this whole branch is about.
	start = time.Now()
	dir := t.TempDir()
	if _, err := cli.ResolveProject(context.Background(), dir, []string{dir + "/compose.yaml"}); err == nil {
		t.Fatal("expected ResolveProject to time out")
	}
	if scan := time.Since(start); scan > 400*time.Millisecond {
		t.Errorf("ResolveProject took %s: it used the user-initiated bound", scan)
	}
}

// The deadline branch has to carry compose's stderr too: a bare "timed out
// after 10s" tells the user nothing about where compose got stuck. The bound
// is a full second against a child that writes at once and then blocks, and
// the ready file is checked so a host slow enough to invalidate the premise
// skips instead of failing.
func TestOutput_ATimeoutCarriesWhatComposeHadSaid(t *testing.T) {
	dir := t.TempDir()
	ready := filepath.Join(dir, "ready")
	newFakeDocker(t, "echo 'resolving image tag' >&2\ntouch "+ready+"\nsleep 5\n")
	cli := &CLI{opts: Options{
		ReadTimeout: time.Second,
		CancelGrace: 300 * time.Millisecond,
	}}

	_, err := cli.ConfigHashes(context.Background(), Project{Name: "web"})
	if err == nil {
		t.Fatal("expected a timeout error")
	}
	if _, statErr := os.Stat(ready); statErr != nil {
		t.Skipf("the child never got to write in a second: %v", statErr)
	}
	if !strings.Contains(err.Error(), "timed out") {
		t.Fatalf("expected the deadline branch, got %v", err)
	}
	if !strings.Contains(err.Error(), "resolving image tag") {
		t.Fatalf("expected what compose had said, got %v", err)
	}
	if strings.Contains(err.Error(), "\n") {
		t.Fatalf("expected a single-line error, got %q", err.Error())
	}
}

// reportedBound pulls the duration out of a "timed out after 1.5s" message,
// so a test can assert which bound applied without pinning its exact
// rounding.
func reportedBound(t *testing.T, msg string) time.Duration {
	t.Helper()
	const marker = "timed out after "
	i := strings.Index(msg, marker)
	if i < 0 {
		t.Fatalf("expected a reported bound in %q", msg)
	}
	rest := msg[i+len(marker):]
	if j := strings.IndexAny(rest, " :"); j >= 0 {
		rest = rest[:j]
	}
	d, err := time.ParseDuration(rest)
	if err != nil {
		t.Fatalf("unparseable bound %q in %q: %v", rest, msg, err)
	}
	return d
}

// Options.CancelGrace is what makes the lingering-grandchild tests finish in
// milliseconds rather than seconds, so it needs a test of its own: without
// it every call that hits a held-open pipe waits the package default.
func TestCancelGrace_OverridesTheDefault(t *testing.T) {
	newFakeDocker(t, "sleep 5 &\necho '{}'\n")
	cli := &CLI{opts: Options{
		ReadTimeout: 2 * time.Second,
		CancelGrace: 100 * time.Millisecond,
	}}

	start := time.Now()
	if _, err := cli.ConfigHashes(context.Background(), Project{Name: "web"}); err != nil {
		t.Fatalf("expected the call to return its output, got %v", err)
	}
	// The default is composeCancelGrace, two seconds; the override is a
	// tenth of one, and the grandchild holds the pipe for five.
	if elapsed := time.Since(start); elapsed > time.Second {
		t.Fatalf("expected the 100ms grace to end the wait, took %s", elapsed)
	}
}
