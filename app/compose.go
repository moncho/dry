package app

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
	"unicode"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"
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
// scan), or the path is not on this machine: config_files records the
// filesystem of whichever machine ran the compose client, not the daemon's,
// so a project brought up on another machine, or a file since moved, yields
// a path that means nothing here. Both cases are treated as "no files".
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
// down is deliberately never refused: compose removes a project by its
// container labels alone, so it does the right thing with no file at all,
// and a project dry only knows from labels must stay removable.
// composeDownCmd does check the paths, but to drop them, not to refuse.
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
func composeScanCmd(resolver composeResolver, dir string, projects []docker.ProjectWithServices, gen uint64) tea.Cmd {
	return func() tea.Msg {
		if resolver == nil || dir == "" {
			return composeProjectsMsg{projects: projects, gen: gen}
		}
		files, ok := docker.ScanComposeDir(dir)
		if !ok {
			return composeProjectsMsg{projects: projects, gen: gen}
		}
		resolved, err := resolver.ResolveProject(context.Background(), dir, files)
		if err != nil {
			return composeProjectsMsg{projects: projects, gen: gen}
		}
		scanned := docker.ComposeProject{
			Name:        resolved.Name,
			WorkingDir:  resolved.WorkingDir,
			ConfigFiles: resolved.Files,
			Status:      docker.ProjectNotCreated,
		}
		return composeProjectsMsg{projects: docker.MergeScannedProject(projects, scanned), gen: gen}
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

// composeDriftBudget bounds the drift walk, not just the calls in it: the
// per-call bounds add up over a project list. It does not bound the model's
// whole cycle, which starts with composeScanCmd under its own
// composeReadTimeout. A var so tests can shorten it.
var composeDriftBudget = 30 * time.Second

// openComposeServices switches to the Compose Services view for a project
// and starts loading its resources, clearing the view on the way in: the
// switch lands before the load does.
func (m model) openComposeServices(project string) (tea.Model, tea.Cmd) {
	m.previousView = m.view
	m.view = ComposeServices
	m.selectedProject = project
	m.composeServices.SetProject(project)
	load := m.loadComposeServices(project)
	return m, load
}

// loadComposeServices starts a resource load for a project, stamped with a
// generation so an older load finishing later cannot replace a newer one's
// rows. Two are in flight after an f5 during a container event. It mutates
// m, so callers assign the command to a variable before returning it rather
// than relying on evaluation order inside a return statement.
func (m *model) loadComposeServices(project string) tea.Cmd {
	m.composeServicesGen++
	return loadComposeServicesCmd(m.daemon, project, m.composeServicesGen)
}

// composeServiceDrift recomputes the drift of the project whose resources
// just loaded, so SYNC keeps up while the user sits in the Compose Services
// view. It is skipped while any check is running, rather than only one that
// covers this project, so a burst cannot stack subprocesses; the skipped
// refresh is recorded and drained, so SYNC is not left stale by the skip.
func (m *model) composeServiceDrift(project string) tea.Cmd {
	if project == "" {
		return nil
	}
	// Only for the view on screen: a load can finish after esc, and a check
	// dispatched then defers the reload the user is actually waiting on.
	if m.view != ComposeServices {
		return nil
	}
	// An unknown project could only come back empty, which would clear its
	// SYNC. With no list at all, ask for one: this view is reachable from the
	// palette without the projects view. With a list that lacks the project,
	// its containers are gone and asking again per event finds nothing.
	p := m.composeProjects.ProjectByName(project)
	if p == nil {
		if m.composeProjects.ProjectCount() == 0 {
			return loadComposeProjectsCmd(m.daemon)
		}
		return nil
	}
	if m.composeCycleRunning() || m.streamingViewerOpen() {
		m.composeRefreshPending = true
		return nil
	}
	return composeServiceDriftCmd(m.composeCLI, *p, m.daemon, m.startComposeDrift())
}

// composeCycleStale bounds how long one unfinished drift check keeps the
// guard closed. Every dispatch is matched by exactly one composeDriftMsg,
// but a compose subprocess that never returns would otherwise gate every
// refresh for the session, so past this bound the check is forgotten.
const composeCycleStale = 2 * time.Minute

// startComposeDrift records that a drift check is starting and returns the
// generation to stamp it with.
func (m *model) startComposeDrift() uint64 {
	m.composeDriftGen++
	if m.composeChecks == nil {
		m.composeChecks = make(map[uint64]time.Time, 2)
	}
	m.composeChecks[m.composeDriftGen] = time.Now()
	return m.composeDriftGen
}

// composeCycleRunning reports whether any compose drift check is in flight,
// forgetting the ones that have aged out on the way past. Checks are
// tracked individually because two dispatchers write them, and a single
// flag let whichever landed first unlock the guard while the other's
// subprocesses were still running.
func (m *model) composeCycleRunning() bool {
	for gen, at := range m.composeChecks {
		if time.Since(at) >= composeCycleStale {
			delete(m.composeChecks, gen)
		}
	}
	return len(m.composeChecks) > 0
}

// The banner's ceiling: how many failing projects it names, how much of each
// project's reason it keeps, and how much of a project's name. All three are
// needed for the length to be bounded, since project names come from a
// container label. Keeping the result to one line is the message bar's job,
// not this function's; nothing here knows the terminal width.
const (
	driftFailureNames  = 2
	driftFailureReason = 60
	driftFailureName   = 30
)

// barSafe makes compose's stderr fit to put in a one-line message bar, and
// ansi.Strip alone covers only the first of three hazards. An escape
// sequence the terminal would re-interpret as styling. A newline lipgloss
// renders as a second row, taking the footer's. And a carriage return or
// backspace it passes through at zero width, so the padding arithmetic is
// satisfied while the terminal repaints over the row; compose writes those
// for progress. A tab lipgloss expands to four cells while ansi.StringWidth
// counts none, so one space in its place keeps the width honest.
func barSafe(s string) string {
	s = ansi.Strip(strings.ReplaceAll(s, "\n", "; "))
	return strings.Map(func(r rune) rune {
		switch {
		case r == '\t':
			return ' '
		case r != ' ' && unicode.IsControl(r):
			return -1
		}
		return r
	}, s)
}

// newDriftFailures is the message for the failures that are new in after, or
// "" when nothing failed that was not already failing. Stable order, and
// bounded: see driftFailureNames.
func newDriftFailures(before, after map[string]string) string {
	var fresh []string
	for name, why := range after {
		if before[name] != why {
			// Truncate by cells, not bytes: compose's stderr is UTF-8 and a
			// byte slice can cut a rune in half.
			clean := ansi.Truncate(barSafe(why), driftFailureReason, "…")
			label := ansi.Truncate(barSafe(name), driftFailureName, "…")
			fresh = append(fresh, label+": "+clean)
		}
	}
	if len(fresh) == 0 {
		return ""
	}
	sort.Strings(fresh)
	msg := "Compose drift check failed: "
	if len(fresh) <= driftFailureNames {
		return msg + strings.Join(fresh, "; ")
	}
	return fmt.Sprintf("%s%s (+%d more)", msg,
		strings.Join(fresh[:driftFailureNames], "; "), len(fresh)-driftFailureNames)
}

// composeViewActive reports whether one of the compose views is on screen.
func (m model) composeViewActive() bool {
	return m.view == ComposeProjects || m.view == ComposeServices
}

// composeDriftCmd asks compose for each project's file hashes and compares
// them against the running containers. Projects whose files are unknown are
// skipped rather than reported as unknown, so the views render as they did
// before drift existed. A ConfigHashes failure is reported per project in
// the message's failures map, and merge keeps that project's last known
// SYNC rather than blanking it.
func composeDriftCmd(engine composeEngine, projects []docker.ProjectWithServices, source composeContainerSource, gen uint64) tea.Cmd {
	return func() tea.Msg {
		if engine == nil {
			return composeDriftMsg{gen: gen}
		}
		ctx, cancel := context.WithTimeout(context.Background(), composeDriftBudget)
		defer cancel()
		drift := make(map[string]map[string]docker.ServiceSync)
		failures := make(map[string]string)
		// gaveUp records the expiry against every project from i on, which
		// is the shape composeDriftMsg has and the more useful one: merge
		// keeps each of their last known SYNC values, and newDriftFailures
		// names two and counts the rest instead of putting one sentence per
		// project into a one-line bar.
		gaveUp := func(i int) {
			why := fmt.Sprintf("drift check gave up after %s, %d of %d projects unchecked",
				composeDriftBudget, len(projects)-i, len(projects))
			for _, rest := range projects[i:] {
				failures[rest.Project.Name] = why
			}
		}
		for i, p := range projects {
			if ctx.Err() != nil {
				gaveUp(i)
				break
			}
			// Containers read per project, not once per cycle: subprocesses
			// take 150-400ms each, so a snapshot from the top would be
			// seconds stale by the last project. projectDrift skips a
			// project whose files are not readable here.
			sync, err := projectDrift(ctx, engine, p.Project,
				source.Containers(nil, docker.NoSort))
			if err != nil {
				if ctx.Err() != nil {
					// The budget expired inside this call, so the error is
					// the context's. Report the budget instead: "context
					// deadline exceeded" in a status bar tells the user
					// nothing they can act on.
					gaveUp(i)
					break
				}
				failures[p.Project.Name] = err.Error()
				continue
			}
			if sync != nil {
				drift[p.Project.Name] = sync
			}
		}
		return composeDriftMsg{gen: gen, drift: drift, failures: failures}
	}
}

// composeContainerSource is the container list a drift check compares the
// compose files against, read when the check runs rather than when it is
// dispatched: the daemon refreshes its store on its own goroutine, so a
// snapshot from dispatch can still hold pre-recreate labels.
type composeContainerSource interface {
	Containers(filters []docker.ContainerFilter, mode docker.SortMode) []*docker.Container
}

// composeServiceDriftCmd recomputes one project's drift, which is what
// keeps SYNC moving in the Compose Services view: drift used to be
// dispatched only from the projects-load path. The message names the
// project so the model merges it rather than replacing every other's.
func composeServiceDriftCmd(engine composeEngine, p docker.ComposeProject, source composeContainerSource, gen uint64) tea.Cmd {
	return func() tea.Msg {
		msg := composeDriftMsg{project: p.Name, gen: gen}
		if engine == nil {
			return msg
		}
		sync, err := projectDrift(context.Background(), engine, p, source.Containers(nil, docker.NoSort))
		if err != nil {
			msg.failures = map[string]string{p.Name: err.Error()}
			return msg
		}
		if sync != nil {
			msg.drift = map[string]map[string]docker.ServiceSync{p.Name: sync}
		}
		return msg
	}
}

// projectDrift compares one project's file hashes against its running
// containers. A nil result with a nil error means the project was skipped
// because its recorded files are not usable here: unknown files, not a
// failure, or ConfigHashes would fail every cycle and pin a banner.
func projectDrift(ctx context.Context, engine composeEngine, p docker.ComposeProject, containers []*docker.Container) (map[string]docker.ServiceSync, error) {
	if !composeFilesUsable(p) {
		return nil, nil
	}
	hashes, err := engine.ConfigHashes(ctx, composeProjectOf(p))
	if err != nil {
		return nil, err
	}
	return docker.CompareConfigHashes(containers, p.Name, hashes), nil
}

// composeDriftState is the SYNC status the compose views render, plus the
// generation each project's entry came from. Two producers write it, a whole
// cycle and the services view's own recompute. A per-project check is one
// subprocess against a cycle's one-per-project, so the older cycle finishing
// last is the expected ordering, not an exotic one, and merging by arrival
// would let a pre-`up` result overwrite a post-`up` one: press u, watch SYNC
// go to ok, watch it flip back. The generation makes that unreachable.
type composeDriftState struct {
	sync map[string]map[string]docker.ServiceSync
	gen  map[string]uint64
	// fail says why each project's last check did not complete. The banner
	// comes from changes to this map rather than from an arriving message, so
	// a result the state rejects as superseded raises no banner, and a
	// failure that keeps failing raises one and then stops.
	fail map[string]string
	// floor is the generation of the newest whole cycle applied. A cycle
	// reports on every project, so nothing older can add information about
	// any of them, including the ones it dropped: those have no per-project
	// generation left to compare against, so a stale cycle would re-add them.
	floor uint64
}

// newComposeDriftState builds a state from a plain SYNC map. Only tests use
// it: production reaches the state through merge.
func newComposeDriftState(sync map[string]map[string]docker.ServiceSync) composeDriftState {
	return composeDriftState{
		sync: sync,
		gen:  make(map[string]uint64, len(sync)),
		fail: make(map[string]string),
	}
}

// merge folds a drift result into the state. An entry is replaced only by
// a check at least as new as the one it holds. A failed check leaves the
// last known value, since "the check failed" is not "no drift": a file
// mid-edit fails until it parses. A whole cycle drops the projects it did
// not report, but never an entry a newer per-project check just wrote.
func (s composeDriftState) merge(msg composeDriftMsg) composeDriftState {
	// Superseded: a newer whole cycle has reported on every project, or a
	// newer check has reported on this one.
	if msg.gen < s.floor || (msg.project != "" && msg.gen < s.gen[msg.project]) {
		return s
	}

	out := composeDriftState{
		sync:  make(map[string]map[string]docker.ServiceSync, len(s.sync)+1),
		gen:   make(map[string]uint64, len(s.gen)+1),
		fail:  make(map[string]string, len(s.fail)+1),
		floor: s.floor,
	}
	carry := func(name string) {
		if sync, ok := s.sync[name]; ok {
			out.sync[name] = sync
		}
		if gen, ok := s.gen[name]; ok {
			out.gen[name] = gen
		}
		if why, ok := s.fail[name]; ok {
			out.fail[name] = why
		}
	}
	// apply takes what msg says about one project. A failure keeps the last
	// known value and records the attempt; a project it reported nothing
	// about loses its entry, which is how empty SYNC means "not checked".
	apply := func(name string) {
		if why, failed := msg.failures[name]; failed {
			if sync, ok := s.sync[name]; ok {
				out.sync[name] = sync
			}
			out.fail[name] = why
			out.gen[name] = msg.gen
			return
		}
		if sync, ok := msg.drift[name]; ok {
			out.sync[name] = sync
			out.gen[name] = msg.gen
		}
	}

	if msg.project != "" {
		for name := range s.sync {
			if name != msg.project {
				carry(name)
			}
		}
		for name := range s.fail {
			if name != msg.project {
				carry(name)
			}
		}
		apply(msg.project)
		return out
	}

	out.floor = msg.gen
	names := make(map[string]struct{}, len(s.sync)+len(msg.drift))
	for name := range s.sync {
		names[name] = struct{}{}
	}
	for name := range s.fail {
		names[name] = struct{}{}
	}
	for name := range msg.drift {
		names[name] = struct{}{}
	}
	for name := range msg.failures {
		names[name] = struct{}{}
	}
	for name := range names {
		if msg.gen < s.gen[name] {
			carry(name)
			continue
		}
		apply(name)
	}
	return out
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

// composeDownCmd takes a project down by label when the recorded paths are
// not usable here. -f on a path that is not on this machine fails the
// command outright ("open /srv/app/compose.yaml: no such file or
// directory") in exactly the case where by-label is the only thing that can
// work; --project-directory does not fail, and is dropped along with the
// files only to keep the invocation purely label-based. See
// composeNoFilesMsg for why down is never refused.
func composeDownCmd(engine composeEngine, p docker.ComposeProject) tea.Cmd {
	return func() tea.Msg {
		if engine == nil {
			return composeUnavailableMsg()
		}
		target := composeProjectOf(p)
		if !composeFilesUsable(p) {
			target.Files = nil
			target.WorkingDir = ""
		}
		reader, err := engine.Down(context.Background(), target)
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
