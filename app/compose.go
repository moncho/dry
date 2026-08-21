package app

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/moncho/dry/docker"
	"github.com/moncho/dry/docker/composecli"
)

// composeEngine is what the app needs from the compose CLI. The concrete
// implementation is *composecli.CLI; tests use a stub.
type composeEngine interface {
	Up(ctx context.Context, p composecli.Project, services ...string) (io.ReadCloser, error)
	Down(ctx context.Context, p composecli.Project) (io.ReadCloser, error)
	Recreate(ctx context.Context, p composecli.Project, service string) (io.ReadCloser, error)
	Config(ctx context.Context, p composecli.Project) (string, error)
	ConfigHashes(ctx context.Context, p composecli.Project) (map[string]string, error)
}

const composeUnavailable = "Docker Compose plugin not found; install it to manage projects from dry"

// composeProjectOf converts a discovered project into a compose command target.
func composeProjectOf(p docker.ComposeProject) composecli.Project {
	return composecli.Project{
		Name:       p.Name,
		WorkingDir: p.WorkingDir,
		Files:      p.ConfigFiles,
	}
}

// detectComposeCmd probes for the compose plugin once, off the update path.
func detectComposeCmd(env docker.Env) tea.Cmd {
	return func() tea.Msg {
		cli, err := composecli.Detect(composecli.Options{Host: env.DockerHost})
		return composeDetectedMsg{cli: cli, err: err}
	}
}

// composeUnavailableMsg reports that an action cannot run.
func composeUnavailableMsg() tea.Msg {
	return statusMessageMsg{text: composeUnavailable, expiry: 5 * time.Second}
}

// composeFilesUsable reports whether p's compose files can actually target
// compose from this host. Two things make a recorded path unusable, and both
// look identical to compose: there is no path at all (an older compose wrote
// no config_files label, or the project row predates the working-directory
// scan), or the path exists only on the daemon's filesystem —
// com.docker.compose.project.config_files describes the host the containers
// were created on, while the compose CLI runs locally, so a remote
// DOCKER_HOST, a moved file, or a deleted file all yield a path that means
// nothing here. Both cases are treated as "no files".
//
// A relative path counts as no path at all. Compose v1 recorded config_files
// as given rather than absolute, and the label is stored verbatim, so
// ConfigFiles can read "docker-compose.yml"; composecli hands that to -f
// unchanged and never sets cmd.Dir, so compose would resolve it against dry's
// own working directory — the same silent wrong-project write this guard
// exists to prevent, reached through a path that happens to stat.
func composeFilesUsable(p docker.ComposeProject) bool {
	if len(p.ConfigFiles) == 0 {
		return false
	}
	for _, f := range p.ConfigFiles {
		if !filepath.IsAbs(f) {
			return false
		}
		if _, err := os.Stat(f); err != nil {
			return false
		}
	}
	return true
}

// composeNoFilesMsg explains why an action that needs the project's compose
// file cannot run.
//
// up, recreate and config all need one. Without -f, `cli.args` emits only
// `compose -p <name> <verb>` and the child inherits dry's own working
// directory, so compose resolves whatever compose file happens to sit there —
// pressing u on one project would create *that* directory's services labelled
// with the selected project's name. A duplicate stack under a wrong name is
// worse than a refusal, so dry refuses.
//
// down is deliberately not guarded: compose removes a project by its
// container labels alone, so it does the right thing with no file at all, and
// a project dry only knows from labels must stay removable.
func composeNoFilesMsg(p docker.ComposeProject) tea.Msg {
	return statusMessageMsg{
		text:   fmt.Sprintf("Project %s has no compose file on this host", p.Name),
		expiry: 5 * time.Second,
	}
}

// composeUpCmd brings a project, or named services of it, up.
func composeUpCmd(engine composeEngine, p docker.ComposeProject, services ...string) tea.Cmd {
	return func() tea.Msg {
		if engine == nil {
			return composeUnavailableMsg()
		}
		if !composeFilesUsable(p) {
			return composeNoFilesMsg(p)
		}
		reader, err := engine.Up(context.Background(), composeProjectOf(p), services...)
		if err != nil {
			return statusMessageMsg{
				text:   fmt.Sprintf("Compose up failed: %s", err),
				expiry: 5 * time.Second,
			}
		}
		title := fmt.Sprintf("Compose up: %s", p.Name)
		if len(services) > 0 {
			title = fmt.Sprintf("Compose up: %s/%s", p.Name, services[0])
		}
		return showStreamingLessMsg{title: title, reader: reader}
	}
}

// composeResolver resolves the real project defined by compose files on disk.
// *composecli.CLI implements it; tests use a stub.
type composeResolver interface {
	ResolveProject(ctx context.Context, dir string, files []string) (composecli.Project, error)
}

// composeScanCmd folds a compose file in dir into the label-derived project
// list. A file that compose cannot resolve is ignored rather than guessed at,
// so dry never lists a project it could not actually bring up.
func composeScanCmd(resolver composeResolver, dir string, projects []docker.ProjectWithServices) tea.Cmd {
	return func() tea.Msg {
		if resolver == nil || dir == "" {
			return composeProjectsMsg{projects: projects}
		}
		files, ok := docker.ScanComposeDir(dir)
		if !ok {
			return composeProjectsMsg{projects: projects}
		}
		resolved, err := resolver.ResolveProject(context.Background(), dir, files)
		if err != nil {
			return composeProjectsMsg{projects: projects}
		}
		scanned := docker.ComposeProject{
			Name:        resolved.Name,
			WorkingDir:  resolved.WorkingDir,
			ConfigFiles: resolved.Files,
			Status:      docker.ProjectNotCreated,
		}
		return composeProjectsMsg{projects: docker.MergeScannedProject(projects, scanned)}
	}
}

// composeRecreateCmd forces recreation of one service even when its config is
// unchanged, which `up` would skip. Exposed through the palette only, because
// `up` already recreates drifted services.
func composeRecreateCmd(engine composeEngine, p docker.ComposeProject, service string) tea.Cmd {
	return func() tea.Msg {
		if engine == nil {
			return composeUnavailableMsg()
		}
		if !composeFilesUsable(p) {
			return composeNoFilesMsg(p)
		}
		reader, err := engine.Recreate(context.Background(), composeProjectOf(p), service)
		if err != nil {
			return statusMessageMsg{
				text:   fmt.Sprintf("Compose recreate failed: %s", err),
				expiry: 5 * time.Second,
			}
		}
		return showStreamingLessMsg{
			title:  fmt.Sprintf("Compose recreate: %s/%s", p.Name, service),
			reader: reader,
		}
	}
}

// composeProjectFor looks up the full project by name from the loaded
// projects list, falling back to a name-only project when it is not found
// (e.g. a service view whose parent project list has not been loaded yet).
// A project known only by name still needs a usable compose target.
func (m model) composeProjectFor(name string) docker.ComposeProject {
	if p := m.composeProjects.ProjectByName(name); p != nil {
		return *p
	}
	return docker.ComposeProject{Name: name}
}

// composeDriftCmd asks compose for each project's file hashes and compares
// them against the running containers. Projects whose files are unknown are
// skipped rather than reported as unknown, so the views render exactly as
// they did before drift existed. A ConfigHashes failure drops only that
// project's drift (an empty SYNC column, same as "not checked yet") but is
// never swallowed: every failure is joined into the returned message's err
// so the model can surface it rather than leave it undiagnosable.
func composeDriftCmd(engine composeEngine, projects []docker.ProjectWithServices, containers []*docker.Container) tea.Cmd {
	return func() tea.Msg {
		if engine == nil {
			return composeDriftMsg{}
		}
		drift := make(map[string]map[string]docker.ServiceSync)
		var errs []error
		for _, p := range projects {
			// A project whose recorded files are not readable here is
			// treated as a project with unknown files, not as a failure:
			// ConfigFiles describes the daemon host's filesystem, so a
			// remote DOCKER_HOST or a moved file would otherwise make
			// ConfigHashes fail every cycle and pin an error banner to the
			// screen for as long as dry runs.
			if !composeFilesUsable(p.Project) {
				continue
			}
			hashes, err := engine.ConfigHashes(context.Background(), composeProjectOf(p.Project))
			if err != nil {
				errs = append(errs, fmt.Errorf("%s: %w", p.Project.Name, err))
				continue
			}
			drift[p.Project.Name] = docker.CompareConfigHashes(containers, p.Project.Name, hashes)
		}
		return composeDriftMsg{drift: drift, err: errors.Join(errs...)}
	}
}

// composeConfigCmd renders a project's configuration into the less viewer.
func composeConfigCmd(engine composeEngine, p docker.ComposeProject) tea.Cmd {
	return func() tea.Msg {
		if engine == nil {
			return composeUnavailableMsg()
		}
		if !composeFilesUsable(p) {
			return composeNoFilesMsg(p)
		}
		rendered, err := engine.Config(context.Background(), composeProjectOf(p))
		if err != nil {
			return statusMessageMsg{
				text:   fmt.Sprintf("Compose config failed: %s", err),
				expiry: 5 * time.Second,
			}
		}
		return showLessMsg{
			content: rendered,
			title:   fmt.Sprintf("Compose config: %s", p.Name),
		}
	}
}

// composeDownCmd takes a project down.
func composeDownCmd(engine composeEngine, p docker.ComposeProject) tea.Cmd {
	return func() tea.Msg {
		if engine == nil {
			return composeUnavailableMsg()
		}
		reader, err := engine.Down(context.Background(), composeProjectOf(p))
		if err != nil {
			return statusMessageMsg{
				text:   fmt.Sprintf("Compose down failed: %s", err),
				expiry: 5 * time.Second,
			}
		}
		return showStreamingLessMsg{
			title:  fmt.Sprintf("Compose down: %s", p.Name),
			reader: reader,
		}
	}
}
