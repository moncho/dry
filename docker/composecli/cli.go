// Package composecli runs the Docker Compose CLI plugin. It builds command
// lines and manages processes; it holds no model types and does not import
// the rest of the docker package, so the parent can wire it in without an
// import cycle.
package composecli

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

// composeCancelGrace bounds how long Close waits, once it cancels a running
// compose command, for the child's output pipes to actually close before it
// forces them shut. Without this bound, a subprocess compose spawned that
// inherited the child's stdout — a credential helper, a buildx worker — and
// that outlives the killed child holds the pipe's write end open, and Close
// never returns: it is the package's only cancellation path, so a hang here
// freezes whatever UI is waiting on it. See exec.Cmd.WaitDelay.
const composeCancelGrace = 2 * time.Second

// composeReadTimeout bounds the calls dry makes on its own: ConfigHashes and
// ResolveProject on every cycle, the Detect probe once at startup. Compose
// has no timeout of its own and the model gates each cycle on the last one
// finishing, so one unanswered call stopped every later refresh. It bounds
// the wait, not the launch or an uninterruptible child: exec watches the
// context only once the process runs, so a `docker` on a wedged mount still
// gates refreshes. Write actions get no bound; see stream.
const composeReadTimeout = 10 * time.Second

// composeConfigTimeout bounds `compose config`, which the user asks for and
// waits on. Far longer than the polled bound: nothing retries it, and a cold
// plugin, a project with many include:s or a slow remote context can each
// cost seconds before compose has done any work.
const composeConfigTimeout = 60 * time.Second

// ErrProbeTimeout reports that the `docker compose version` probe did not
// answer, as opposed to answering that the plugin is not installed. Both
// leave no engine, but only one is fixed by installing the plugin.
var ErrProbeTimeout = errors.New("docker compose version probe timed out")

// Options configures how compose is invoked.
type Options struct {
	// Host is the Docker host to target, i.e. the value of DOCKER_HOST.
	// Empty means whatever the environment already says.
	Host string
	// Extra is appended to the child environment, mainly for tests.
	Extra []string
	// ReadTimeout overrides the bound on every read-only call, Config
	// included, unless ConfigTimeout is set as well. Zero means the
	// per-call default; tests set it small.
	ReadTimeout time.Duration
	// ConfigTimeout overrides the bound on Config alone, which is the one
	// read the user waits on rather than dry polling it. Zero falls back to
	// ReadTimeout and then to composeConfigTimeout.
	ConfigTimeout time.Duration
	// CancelGrace overrides composeCancelGrace. Zero means the default;
	// tests set it small.
	CancelGrace time.Duration
}

// CLI runs `docker compose` commands.
type CLI struct {
	opts    Options
	version string
}

// Detect verifies that the compose plugin is available and records its
// version. It runs one short-lived process and must be called from a command,
// never from the UI update path. The probe is bounded like the other
// read-only calls, and nothing retries it, so the bound does not save
// detection: it turns an indefinite hang into a reported failure, which the
// model can tell the user about instead of leaving compose quietly absent
// for the session.
func Detect(opts Options) (*CLI, error) {
	c := &CLI{opts: opts}
	bound := c.timeout(composeReadTimeout)
	ctx, cancel := context.WithTimeout(context.Background(), bound)
	defer cancel()
	cmd := exec.CommandContext(ctx, "docker", "compose", "version")
	cmd.WaitDelay = c.cancelGrace()
	killed := killSwitch(cmd)
	out, err := cmd.CombinedOutput()
	// ErrWaitDelay means the command exited fine and a grandchild held the
	// pipe open, so the version is already in hand: not a failure.
	if err != nil && !errors.Is(err, exec.ErrWaitDelay) {
		if killed.Load() {
			return nil, fmt.Errorf("%w after %s", ErrProbeTimeout, bound)
		}
		return nil, fmt.Errorf("docker compose plugin not available: %w", err)
	}
	c.version = strings.TrimSpace(string(out))
	return c, nil
}

// timeout is the bound for a read-only call: the base the caller names,
// unless Options overrides it. ConfigTimeout applies to composeConfigTimeout
// only, so the polled and user-initiated bounds can be set apart; ReadTimeout
// overrides both, which is what most tests want.
func (c *CLI) timeout(base time.Duration) time.Duration {
	if base == composeConfigTimeout && c.opts.ConfigTimeout > 0 {
		return c.opts.ConfigTimeout
	}
	if c.opts.ReadTimeout > 0 {
		return c.opts.ReadTimeout
	}
	return base
}

// cancelGrace is how long a killed command's output pipes are waited on.
func (c *CLI) cancelGrace() time.Duration {
	if c.opts.CancelGrace > 0 {
		return c.opts.CancelGrace
	}
	return composeCancelGrace
}

// Version is the version string reported by the plugin.
func (c *CLI) Version() string { return c.version }

// Project is the target of a compose command: a project name, the directory
// commands resolve relative paths against, and the files that define it.
type Project struct {
	Name       string
	WorkingDir string
	Files      []string
}

// args builds the argument list for a compose verb against a project.
func (c *CLI) args(p Project, verb ...string) []string {
	args := []string{"compose"}
	if p.WorkingDir != "" {
		args = append(args, "--project-directory", p.WorkingDir)
	}
	for _, f := range p.Files {
		args = append(args, "-f", f)
	}
	if p.Name != "" {
		args = append(args, "-p", p.Name)
	}
	return append(args, verb...)
}

// env is the child environment for every invocation.
func (c *CLI) env() []string {
	env := os.Environ()
	if c.opts.Host != "" {
		env = append(env, "DOCKER_HOST="+c.opts.Host)
	}
	return append(env, c.opts.Extra...)
}

// procReader streams a process's combined output. Closing it kills the
// process and reports its exit status. A termination this package itself
// caused — Close canceling a still-running command, or WaitDelay force-
// closing pipes a lingering grandchild held open — is reported as a nil
// error rather than as a failure; a termination the process chose on its
// own, including a genuine non-zero exit, is reported as-is. A concurrent
// reader — the expected usage, since cancellation is documented as "closing
// the returned reader" while another goroutine streams from it — sees a
// clean io.EOF when Close ends the stream, never the pipe's generic
// "closed" error, which would be indistinguishable from a real failure.
type procReader struct {
	pr     *io.PipeReader
	cancel context.CancelFunc
	done   chan struct{} // closed once the process has been waited on
	err    error         // the error to report; valid once done is closed

	once   sync.Once
	result error
}

// Read implements io.Reader. This package is the only thing that ever closes
// pr's read side (in Close, below), so a Read that unblocks with
// io.ErrClosedPipe is always Close doing its job, not a real stream failure
// — report it as a clean end-of-stream instead.
func (p *procReader) Read(b []byte) (int, error) {
	n, err := p.pr.Read(b)
	if errors.Is(err, io.ErrClosedPipe) {
		err = io.EOF
	}
	return n, err
}

func (p *procReader) Close() error {
	p.once.Do(func() {
		// Cancel first so a still-running compose does not block the wait,
		// then close the reader so a writer blocked on a full, undrained
		// pipe cannot block cmd.Wait() either.
		p.cancel()
		_ = p.pr.Close()
		<-p.done
		p.result = p.err
	})
	return p.result
}

// stream starts a compose command and returns a reader over its combined
// stdout and stderr. There is no timeout on the command itself: pulls and
// builds are legitimately slow. Closing the reader cancels the process,
// bounded by composeCancelGrace (see its doc comment).
func (c *CLI) stream(ctx context.Context, p Project, verb ...string) (io.ReadCloser, error) {
	ctx, cancel := context.WithCancel(ctx)
	cmd := exec.CommandContext(ctx, "docker", c.args(p, verb...)...)
	cmd.Env = c.env()
	cmd.WaitDelay = c.cancelGrace()
	pr, pw := io.Pipe()
	cmd.Stdout = pw
	cmd.Stderr = pw
	if err := cmd.Start(); err != nil {
		cancel()
		return nil, err
	}

	pReader := &procReader{pr: pr, cancel: cancel, done: make(chan struct{})}
	go func() {
		pReader.err = classifyExit(cmd.Wait())
		_ = pw.Close() // unblocks any reader still waiting on EOF
		close(pReader.done)
	}()
	return pReader, nil
}

// classifyExit turns cmd.Wait's error into what Close should report. It
// suppresses two shapes that reflect this package ending the process itself
// rather than the process failing on its own:
//
//   - the process was terminated by a signal (ExitCode() == -1) — whether
//     from our own cancellation or from a WaitDelay-forced kill of a child
//     that ignored it;
//   - exec.ErrWaitDelay, which exec.Cmd only returns when the command
//     itself exited successfully but WaitDelay had to force-close pipes a
//     lingering grandchild (a credential helper, say) was still holding
//     open — not a failure of compose.
//
// Everything else, including a genuine non-zero exit code, is reported
// as-is. This classifies on the process's actual, final exit state rather
// than on whether or when Close happened to run, so it can't be confused by
// a Close that races a natural exit the way checking the context's
// cancellation cause could be.
func classifyExit(err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, exec.ErrWaitDelay) {
		return nil
	}
	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) && exitErr.ExitCode() == -1 {
		return nil
	}
	return err
}

// output runs a read-only compose command to completion and returns its
// stdout, bounded by the given timeout or by the caller's context, whichever
// expires first, so the bound it reports is the one that applied. WaitDelay
// caps cmd.Wait too: a grandchild holding the write end open outlives the
// child the deadline killed.
//
// The branches are ordered on whether this package killed the process
// rather than on what the context says; killSwitch is what records it.
func (c *CLI) output(ctx context.Context, bound time.Duration, p Project, verb ...string) (string, error) {
	if deadline, ok := ctx.Deadline(); ok {
		if remaining := time.Until(deadline); remaining < bound {
			bound = remaining
		}
	}
	ctx, cancel := context.WithTimeout(ctx, bound)
	defer cancel()
	cmd := exec.CommandContext(ctx, "docker", c.args(p, verb...)...)
	cmd.Env = c.env()
	cmd.WaitDelay = c.cancelGrace()
	killed := killSwitch(cmd)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	out, err := cmd.Output()
	switch {
	case err == nil:
		return string(out), nil
	case errors.Is(err, exec.ErrWaitDelay):
		// ErrWaitDelay means the command exited fine and a grandchild held
		// the pipe open, so the output is already complete: the draining
		// copy had the whole grace period. A slow bystander, not a failure.
		return string(out), nil
	case !killed.Load():
		// The process was not killed by us: either it chose its own non-zero
		// exit, and compose's diagnosis beats the exit code, or it never
		// started, which leaves nothing on stderr and falls through to err.
		if msg := strings.TrimSpace(stderr.String()); msg != "" {
			return "", fmt.Errorf("%s", msg)
		}
		return "", err
	case errors.Is(ctx.Err(), context.DeadlineExceeded):
		return "", fmt.Errorf("docker compose %s timed out after %s%s",
			strings.Join(verb, " "), bound.Round(time.Millisecond), hint(&stderr))
	default:
		return "", fmt.Errorf("docker compose %s was cancelled%s",
			strings.Join(verb, " "), hint(&stderr))
	}
}

// killSwitch wraps cmd.Cancel so the caller can tell a process this package
// killed from one that failed on its own, which the exit status cannot carry
// everywhere: a signalled process has no exit code on POSIX, but Windows
// kills with TerminateProcess(h, 1), indistinguishable from compose choosing
// to exit 1.
//
// The flag is set only when the kill found something to kill. exec calls
// Cancel whenever the context is done before the result has been handed
// over, which includes a process that finished a moment earlier, and Kill
// then reports os.ErrProcessDone; exec treats that as "not interrupted" and
// so does this. What is left is the window between a process exiting and
// being reaped, where a kill still succeeds: a failure landing there is
// reported as whichever of the deadline or the cancellation applies, with
// compose's own stderr appended by hint.
//
// Only for a Cmd from exec.CommandContext: that is what installs the Cancel
// this wraps, and Start rejects a Cancel on a Cmd built any other way.
func killSwitch(cmd *exec.Cmd) *atomic.Bool {
	var killed atomic.Bool
	kill := cmd.Cancel
	cmd.Cancel = func() error {
		err := kill()
		if !errors.Is(err, os.ErrProcessDone) {
			killed.Store(true)
		}
		return err
	}
	return &killed
}

// hint returns what the killed process had written to stderr, as a suffix
// for the message saying why it was killed. It explains where compose got
// stuck, not what went wrong: it never got to report its own verdict.
func hint(stderr *bytes.Buffer) string {
	msg := strings.TrimSpace(stderr.String())
	if msg == "" {
		return ""
	}
	return ": " + strings.ReplaceAll(msg, "\n", "; ")
}

// Up creates and starts the project, or only the named services.
func (c *CLI) Up(ctx context.Context, p Project, services ...string) (io.ReadCloser, error) {
	return c.stream(ctx, p, append([]string{"up", "-d"}, services...)...)
}

// Down stops and removes the project's containers, networks, and default volumes.
func (c *CLI) Down(ctx context.Context, p Project) (io.ReadCloser, error) {
	return c.stream(ctx, p, "down")
}

// Recreate forces recreation of one service even when its config is unchanged.
func (c *CLI) Recreate(ctx context.Context, p Project, service string) (io.ReadCloser, error) {
	return c.stream(ctx, p, "up", "-d", "--force-recreate", service)
}

// Config returns the project's rendered configuration. It gets the longer,
// user-initiated bound: the c key waits on it and nothing retries it.
func (c *CLI) Config(ctx context.Context, p Project) (string, error) {
	return c.output(ctx, c.timeout(composeConfigTimeout), p, "config")
}

// ConfigHashes returns the per-service config hash of the project's files.
// These are comparable to each container's com.docker.compose.config-hash
// label, which is how compose itself decides whether to recreate a container.
func (c *CLI) ConfigHashes(ctx context.Context, p Project) (map[string]string, error) {
	out, err := c.output(ctx, c.timeout(composeReadTimeout), p, "config", "--hash=*")
	if err != nil {
		return nil, err
	}
	return parseConfigHashes(out), nil
}

// ResolveProject asks compose for the real project defined by the given files,
// which respects a `name:` key in the file rather than guessing from the
// directory name.
func (c *CLI) ResolveProject(ctx context.Context, dir string, files []string) (Project, error) {
	probe := Project{WorkingDir: dir, Files: files}
	out, err := c.output(ctx, c.timeout(composeReadTimeout), probe, "config", "--format", "json")
	if err != nil {
		return Project{}, err
	}
	var parsed struct {
		Name string `json:"name"`
	}
	if err := json.Unmarshal([]byte(out), &parsed); err != nil {
		return Project{}, fmt.Errorf("parse compose config: %w", err)
	}
	if parsed.Name == "" {
		return Project{}, fmt.Errorf("compose config reported no project name")
	}
	probe.Name = parsed.Name
	return probe, nil
}
