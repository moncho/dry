package app

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"
	"unicode/utf8"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"
	"github.com/moby/moby/api/types/container"
	"github.com/moncho/dry/appui"
	appcompose "github.com/moncho/dry/appui/compose"
	"github.com/moncho/dry/docker"
	"github.com/moncho/dry/docker/composecli"
)

// stubComposeEngine records calls and returns canned results. It also
// implements composeResolver (via ResolveProject) so a single stub can drive
// the scan-then-drift sequence end to end without a second stub type.
type stubComposeEngine struct {
	upCalls          []composecli.Project
	upServices       []string
	downCalls        []composecli.Project
	recreateCalls    []composecli.Project
	recreateServices []string
	configCalls      []composecli.Project
	configOutput     string
	hashes           map[string]string
	hashesCalls      []composecli.Project
	hashesErr        error
	hangHashes       bool
	hashesDelay      time.Duration
	resolveName      string
	err              error
}

func (s *stubComposeEngine) Up(_ context.Context, p composecli.Project, services ...string) (io.ReadCloser, error) {
	if s.err != nil {
		return nil, s.err
	}
	s.upCalls = append(s.upCalls, p)
	s.upServices = append(s.upServices, services...)
	return io.NopCloser(strings.NewReader("Container started\n")), nil
}

func (s *stubComposeEngine) Down(_ context.Context, p composecli.Project) (io.ReadCloser, error) {
	if s.err != nil {
		return nil, s.err
	}
	s.downCalls = append(s.downCalls, p)
	return io.NopCloser(strings.NewReader("Container removed\n")), nil
}

func (s *stubComposeEngine) Recreate(_ context.Context, p composecli.Project, service string) (io.ReadCloser, error) {
	if s.err != nil {
		return nil, s.err
	}
	s.recreateCalls = append(s.recreateCalls, p)
	s.recreateServices = append(s.recreateServices, service)
	return io.NopCloser(strings.NewReader("recreated\n")), nil
}

func (s *stubComposeEngine) Config(_ context.Context, p composecli.Project) (string, error) {
	// Recorded, not discarded: config is the one read-only compose call,
	// and a caller that hands it an empty project renders the wrong file's
	// configuration while every assertion on the output still passes.
	s.configCalls = append(s.configCalls, p)
	return s.configOutput, s.err
}

func (s *stubComposeEngine) ConfigHashes(ctx context.Context, p composecli.Project) (map[string]string, error) {
	s.hashesCalls = append(s.hashesCalls, p)
	// hangHashes makes the stub behave like a compose that never answers:
	// it waits for the caller's context instead of returning, which is what
	// the drift cycle's budget has to survive. The fallback matters: with
	// only <-ctx.Done() here, an unbounded context turns a regression into
	// a hung package instead of a failed assertion.
	if s.hashesDelay > 0 {
		// A call that completes despite the deadline, deliberately not
		// watching ctx: a compose subprocess that finishes anyway. The
		// budget then expires between calls rather than inside one, which
		// is the only thing the pre-call check catches.
		time.Sleep(s.hashesDelay)
	}
	if s.hangHashes {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(2 * time.Second):
			return s.hashes, nil
		}
	}
	if s.hashesErr != nil {
		return nil, s.hashesErr
	}
	return s.hashes, s.err
}

// ResolveProject implements composeResolver so a stubComposeEngine can serve
// as m.composeCLI in tests that need the scan path to actually run.
func (s *stubComposeEngine) ResolveProject(_ context.Context, dir string, files []string) (composecli.Project, error) {
	if s.err != nil {
		return composecli.Project{}, s.err
	}
	name := s.resolveName
	if name == "" {
		name = "resolved"
	}
	return composecli.Project{Name: name, WorkingDir: dir, Files: files}, nil
}

// composeFileFixture writes a real compose file into a temp directory and
// returns the directory and the file path. Tests that exercise an action
// which targets a project's files need paths that actually exist: dry refuses
// to hand compose a file it cannot see, because a missing -f makes compose
// resolve the file from dry's own working directory instead.
func composeFileFixture(t *testing.T) (string, string) {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "compose.yaml")
	if err := os.WriteFile(path, []byte("services: {}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	return dir, path
}

func TestComposeProjectOf_CarriesFilesAndDir(t *testing.T) {
	p := composeProjectOf(docker.ComposeProject{
		Name:        "web",
		WorkingDir:  "/srv/web",
		ConfigFiles: []string{"/srv/web/compose.yaml"},
	})
	if p.Name != "web" || p.WorkingDir != "/srv/web" || len(p.Files) != 1 {
		t.Fatalf("conversion lost information: %+v", p)
	}
}

func TestComposeUpCmd_StreamsIntoTheViewer(t *testing.T) {
	engine := &stubComposeEngine{}
	dir, file := composeFileFixture(t)
	cmd := composeUpCmd(engine, docker.ComposeProject{
		Name:        "web",
		WorkingDir:  dir,
		ConfigFiles: []string{file},
	}, "api")

	msg := cmd()
	streaming, ok := msg.(showStreamingLessMsg)
	if !ok {
		t.Fatalf("expected showStreamingLessMsg, got %T", msg)
	}
	if streaming.reader == nil {
		t.Fatal("expected a reader to stream compose output")
	}
	_ = streaming.reader.Close()
	if len(engine.upCalls) != 1 || engine.upCalls[0].Name != "web" {
		t.Fatalf("expected Up on project web, got %+v", engine.upCalls)
	}
	if len(engine.upServices) != 1 || engine.upServices[0] != "api" {
		t.Fatalf("expected the service to be targeted, got %v", engine.upServices)
	}
}

func TestComposeUpCmd_NoEngineReportsWhy(t *testing.T) {
	cmd := composeUpCmd(nil, docker.ComposeProject{Name: "web"})

	status, ok := cmd().(statusMessageMsg)
	if !ok {
		t.Fatalf("expected a status message when no engine is available, got %T", cmd())
	}
	if !strings.Contains(status.text, "Compose plugin") {
		t.Fatalf("expected the message to name the missing plugin, got %q", status.text)
	}
}

// TestComposeUpCmd_NoFilesRefusesRatherThanGuessing guards the branch's
// worst failure: with no ConfigFiles and no WorkingDir, composecli emits
// `compose -p <name> up -d` and the child inherits dry's own working
// directory, so `u` on one project creates *dry's* directory's services
// under that project's name — a duplicate stack under a wrong name, silently.
// A project whose files dry cannot see must be refused, not guessed at.
func TestComposeUpCmd_NoFilesRefusesRatherThanGuessing(t *testing.T) {
	engine := &stubComposeEngine{}

	msg := composeUpCmd(engine, docker.ComposeProject{Name: "web"})()

	status, ok := msg.(statusMessageMsg)
	if !ok {
		t.Fatalf("expected a status message for a project with no files, got %T", msg)
	}
	if !strings.Contains(status.text, "no compose file") {
		t.Fatalf("expected the reason, got %q", status.text)
	}
	if len(engine.upCalls) != 0 {
		t.Fatalf("compose must not be invoked without a file to target, got %+v", engine.upCalls)
	}
}

// TestComposeUpCmd_FilesMissingLocallyRefuses covers the moved-file and
// brought-up-elsewhere case: ConfigFiles comes from a container label
// recording the filesystem of whichever machine ran the compose client, so
// the recorded path can be meaningless here.
// An unreadable file must be treated exactly like no file at all.
func TestComposeUpCmd_FilesMissingLocallyRefuses(t *testing.T) {
	engine := &stubComposeEngine{}
	p := docker.ComposeProject{
		Name:        "web",
		WorkingDir:  "/srv/web",
		ConfigFiles: []string{filepath.Join(t.TempDir(), "gone.yaml")},
	}

	msg := composeUpCmd(engine, p)()

	if _, ok := msg.(statusMessageMsg); !ok {
		t.Fatalf("expected a status message when the recorded file is not readable here, got %T", msg)
	}
	if len(engine.upCalls) != 0 {
		t.Fatalf("compose must not be invoked with a file dry cannot read, got %+v", engine.upCalls)
	}
}

// TestComposeUpCmd_PartiallyMissingFilesRefuses covers a multi-file project
// (a compose file plus an override) where only one file survives locally:
// compose would fail on the missing one, so the project is not a usable
// target.
func TestComposeUpCmd_PartiallyMissingFilesRefuses(t *testing.T) {
	engine := &stubComposeEngine{}
	dir, file := composeFileFixture(t)
	p := docker.ComposeProject{
		Name:        "web",
		WorkingDir:  dir,
		ConfigFiles: []string{file, filepath.Join(dir, "compose.override.yaml")},
	}

	if _, ok := composeUpCmd(engine, p)().(statusMessageMsg); !ok {
		t.Fatal("expected a status message when one of the project's files is missing")
	}
	if len(engine.upCalls) != 0 {
		t.Fatalf("compose must not be invoked with an incomplete file set, got %+v", engine.upCalls)
	}
}

// TestComposeUpCmd_RelativeFileRefusesEvenWhenItResolvesHere is the Critical's
// own failure mode reached through a path that exists. Compose v1 recorded
// config_files as given rather than absolute, and docker/compose.go stores the
// label verbatim, so a project's ConfigFiles can read "docker-compose.yml".
// composecli passes that to -f unchanged and never sets cmd.Dir, so compose
// resolves it against dry's own working directory: an os.Stat-only guard
// passes whenever dry was started in *any* directory holding a file of that
// name, and up then creates that directory's services under the selected
// project's name. The trap is specifically a relative path that does resolve
// locally, which is why this test creates the file it names.
func TestComposeUpCmd_RelativeFileRefusesEvenWhenItResolvesHere(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "docker-compose.yml"), []byte("services: {}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	// dry started in a directory that happens to hold a compose file of the
	// name the label recorded — a different project's file.
	t.Chdir(dir)
	engine := &stubComposeEngine{}
	p := docker.ComposeProject{Name: "web", ConfigFiles: []string{"docker-compose.yml"}}

	if _, err := os.Stat(p.ConfigFiles[0]); err != nil {
		t.Fatalf("precondition: the relative path must resolve in this directory: %v", err)
	}

	msg := composeUpCmd(engine, p)()

	status, ok := msg.(statusMessageMsg)
	if !ok {
		t.Fatalf("expected a status message for a relative recorded path, got %T", msg)
	}
	if !strings.Contains(status.text, "no compose file") {
		t.Fatalf("expected the reason, got %q", status.text)
	}
	if len(engine.upCalls) != 0 {
		t.Fatalf("compose must not be invoked with a path that resolves against dry's own directory, got %+v", engine.upCalls)
	}
}

func TestComposeRecreateCmd_NoFilesRefusesRatherThanGuessing(t *testing.T) {
	engine := &stubComposeEngine{}

	msg := composeRecreateCmd(engine, docker.ComposeProject{Name: "web"}, "api")()

	status, ok := msg.(statusMessageMsg)
	if !ok {
		t.Fatalf("expected a status message for a project with no files, got %T", msg)
	}
	if !strings.Contains(status.text, "no compose file") {
		t.Fatalf("expected the reason, got %q", status.text)
	}
	if len(engine.recreateCalls) != 0 {
		t.Fatalf("compose must not be invoked without a file to target, got %+v", engine.recreateCalls)
	}
}

// TestComposeDownCmd_NoFilesStillRuns documents the deliberate asymmetry:
// `down` removes a project by its container labels alone, so it needs no
// file and must keep working for a project dry only knows from labels.
func TestComposeDownCmd_NoFilesStillRuns(t *testing.T) {
	engine := &stubComposeEngine{}

	msg := composeDownCmd(engine, docker.ComposeProject{Name: "web"})()

	if _, ok := msg.(showStreamingLessMsg); !ok {
		t.Fatalf("down must not be guarded on files, got %T", msg)
	}
	if len(engine.downCalls) != 1 {
		t.Fatalf("expected Down to run for a file-less project, got %+v", engine.downCalls)
	}
}

func TestComposeDownCmd_StreamsIntoTheViewer(t *testing.T) {
	engine := &stubComposeEngine{}
	cmd := composeDownCmd(engine, docker.ComposeProject{Name: "web", WorkingDir: "/srv/web"})

	msg := cmd()
	streaming, ok := msg.(showStreamingLessMsg)
	if !ok {
		t.Fatalf("expected showStreamingLessMsg, got %T", msg)
	}
	if streaming.reader == nil {
		t.Fatal("expected a reader to stream compose output")
	}
	_ = streaming.reader.Close()
	if len(engine.downCalls) != 1 || engine.downCalls[0].Name != "web" {
		t.Fatalf("expected Down on project web, got %+v", engine.downCalls)
	}
}

func TestComposeDownCmd_NoEngineReportsWhy(t *testing.T) {
	cmd := composeDownCmd(nil, docker.ComposeProject{Name: "web"})

	status, ok := cmd().(statusMessageMsg)
	if !ok {
		t.Fatalf("expected a status message when no engine is available, got %T", cmd())
	}
	if !strings.Contains(status.text, "Compose plugin") {
		t.Fatalf("expected the message to name the missing plugin, got %q", status.text)
	}
}

// stubResolver resolves a fixed project name, recording what it was asked.
type stubResolver struct {
	name     string
	askedDir string
	err      error
}

func (s *stubResolver) ResolveProject(_ context.Context, dir string, files []string) (composecli.Project, error) {
	s.askedDir = dir
	if s.err != nil {
		return composecli.Project{}, s.err
	}
	return composecli.Project{Name: s.name, WorkingDir: dir, Files: files}, nil
}

func TestComposeScanCmd_AddsNeverStartedProject(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "compose.yaml"), []byte("services: {}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	resolver := &stubResolver{name: "custom-name"}

	msg := composeScanCmd(resolver, dir, nil, 1)()
	loaded, ok := msg.(composeProjectsMsg)
	if !ok {
		t.Fatalf("expected composeProjectsMsg, got %T", msg)
	}
	if len(loaded.projects) != 1 {
		t.Fatalf("expected the scanned project to be listed, got %d", len(loaded.projects))
	}
	p := loaded.projects[0].Project
	if p.Name != "custom-name" {
		t.Fatalf("expected the resolved name, not the directory name, got %q", p.Name)
	}
	if p.Status != docker.ProjectNotCreated {
		t.Fatalf("expected status not created, got %q", p.Status)
	}
	if resolver.askedDir != dir {
		t.Fatalf("expected the scan directory to be resolved, got %q", resolver.askedDir)
	}
}

func TestComposeScanCmd_NoComposeFileIsAPassthrough(t *testing.T) {
	existing := []docker.ProjectWithServices{{Project: docker.ComposeProject{Name: "web"}}}

	msg := composeScanCmd(&stubResolver{name: "unused"}, t.TempDir(), existing, 1)()
	loaded := msg.(composeProjectsMsg)
	if len(loaded.projects) != 1 || loaded.projects[0].Project.Name != "web" {
		t.Fatalf("expected the list to pass through unchanged, got %+v", loaded.projects)
	}
}

func TestComposeScanCmd_ResolveFailureIsAPassthrough(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "compose.yaml"), []byte("bad: ["), 0o644); err != nil {
		t.Fatal(err)
	}

	msg := composeScanCmd(&stubResolver{err: errors.New("invalid compose file")}, dir, nil, 1)()
	loaded := msg.(composeProjectsMsg)
	if len(loaded.projects) != 0 {
		t.Fatalf("an unresolvable file must not invent a project, got %+v", loaded.projects)
	}
}

// u on a project row asks first, like d, ctrl+t, ctrl+r and ctrl+e do:
// bringing up every service in a project is not a keystroke's worth of
// consequence, and running it unasked made the cursor position a safety
// mechanism it could not be.
func TestComposeProjectsView_UPromptsBeforeBringingAProjectUp(t *testing.T) {
	engine := &stubComposeEngine{}
	dir, file := composeFileFixture(t)
	m := newTestModel()
	m.view = ComposeProjects
	m.composeCLI = engine
	m.composeProjects.SetProjects([]docker.ProjectWithServices{{
		Project: docker.ComposeProject{Name: "web", WorkingDir: dir, ConfigFiles: []string{file}},
	}})

	result, _ := m.Update(tea.KeyPressMsg{Code: 'u'})
	asked := result.(model)
	if asked.overlay != overlayPrompt {
		t.Fatal("expected u on a project row to ask first")
	}
	if len(engine.upCalls) != 0 {
		t.Fatalf("expected nothing run before the answer, got %+v", engine.upCalls)
	}
	// What the prompt asks and what it carries, both read off the prompt
	// itself. Injecting the tag and id below instead would pass with u
	// wired to any tag at all, and a tag nothing dispatches on means the
	// user confirms and nothing happens.
	if q := ansi.Strip(asked.prompt.View()); !strings.Contains(q, "Bring project web up?") {
		t.Errorf("expected the prompt to name what it is about to do, got %q", q)
	}
	if tag, id := answerPrompt(t, asked); tag != "compose-project-up" || id != "web" {
		t.Fatalf("expected the project-up confirmation for web, got %q %q", tag, id)
	}

	// Confirming runs it, and it streams into the viewer.
	result, cmd := asked.Update(appui.PromptResultMsg{Confirmed: true, Tag: "compose-project-up", ID: "web"})
	if cmd == nil {
		t.Fatal("expected the confirmation to run up")
	}
	if _, ok := cmd().(showStreamingLessMsg); !ok {
		t.Fatalf("expected up to stream into the viewer, got %T", cmd())
	}
	if len(engine.upCalls) != 1 || engine.upCalls[0].Name != "web" {
		t.Fatalf("expected one Up on web, got %+v", engine.upCalls)
	}
	if got := result.(model); got.overlay == overlayPrompt {
		t.Error("expected the prompt closed after answering")
	}

	// And declining runs nothing.
	engine.upCalls = nil
	if _, cmd := asked.Update(appui.PromptResultMsg{Confirmed: false, Tag: "compose-project-up", ID: "web"}); cmd != nil {
		cmd()
	}
	if len(engine.upCalls) != 0 {
		t.Errorf("expected nothing run when declined, got %+v", engine.upCalls)
	}
}

func TestComposeProjectsView_DPromptsBeforeDown(t *testing.T) {
	m := newTestModel()
	m.view = ComposeProjects
	m.composeCLI = &stubComposeEngine{}
	m.composeProjects.SetProjects([]docker.ProjectWithServices{{
		Project: docker.ComposeProject{Name: "web"},
	}})

	result, _ := m.Update(tea.KeyPressMsg{Code: 'd'})
	m = result.(model)
	if m.overlay != overlayPrompt {
		t.Fatal("expected down to ask for confirmation first")
	}
}

// TestComposeProjectDown_ExecutesAgainstTheEngine drives the confirmation
// that "d" opens all the way through to the daemon call, the way
// appui.PromptModel actually delivers it, so the tag-string contract
// between keys_compose.go's showPrompt("compose-project-down", p.Name) and
// ops.go's case "compose-project-down" is verified end to end rather than
// by inspection. This is the branch's one new destructive action.
func TestComposeProjectDown_ExecutesAgainstTheEngine(t *testing.T) {
	engine := &stubComposeEngine{}
	m := newTestModel()
	m.composeCLI = engine
	m.composeProjects.SetProjects([]docker.ProjectWithServices{{
		Project: docker.ComposeProject{Name: "web", WorkingDir: "/srv/web"},
	}})

	_, cmd := m.Update(appui.PromptResultMsg{Confirmed: true, Tag: "compose-project-down", ID: "web"})
	if cmd == nil {
		t.Fatal("expected the confirmed prompt to produce a command")
	}
	msg := cmd()
	if _, ok := msg.(showStreamingLessMsg); !ok {
		t.Fatalf("expected down to stream into the viewer, got %T", msg)
	}
	if len(engine.downCalls) != 1 || engine.downCalls[0].Name != "web" {
		t.Fatalf("expected one Down call on project web, got %+v", engine.downCalls)
	}
}

func TestPalette_RecreateOffersItselfForAService(t *testing.T) {
	m := newTestModel()
	m.view = ComposeServices
	m.composeCLI = &stubComposeEngine{}
	m.composeServices.SetServices([]docker.ComposeService{
		{Project: "web", Name: "api"},
	}, nil, nil, "web")

	// A fresh load selects the "Services" section header, not the service
	// itself; navigate down once through the model's own key path, the way
	// a user would, before a service is actually selected.
	result, _ := m.Update(tea.KeyPressMsg{Code: tea.KeyDown})
	m = result.(model)
	if m.composeServices.SelectedService() == nil {
		t.Fatal("expected the down key to select the service row")
	}

	var found bool
	for _, a := range m.commandPaletteActions() {
		if a.ID == "compose:recreate" {
			found = true
			if !strings.Contains(a.Description, "api") {
				t.Fatalf("expected the action to name the service, got %q", a.Description)
			}
		}
	}
	if !found {
		t.Fatal("expected a forced-recreate action for the selected service")
	}
}

// TestPalette_RecreateExecutesAgainstTheEngine drives
// executePaletteAction("compose:recreate") itself, so the wiring between
// the palette and the engine is verified rather than assumed: without this,
// swapping the project and service arguments to composeRecreateCmd would
// keep every other test green.
func TestPalette_RecreateExecutesAgainstTheEngine(t *testing.T) {
	engine := &stubComposeEngine{}
	dir, file := composeFileFixture(t)
	m := newTestModel()
	m.view = ComposeServices
	m.composeCLI = engine
	// The recreate target is resolved from the project list, so the project
	// row must carry the files the action will hand to compose.
	m.composeProjects.SetProjects([]docker.ProjectWithServices{{
		Project: docker.ComposeProject{Name: "web", WorkingDir: dir, ConfigFiles: []string{file}},
	}})
	m.composeServices.SetServices([]docker.ComposeService{
		{Project: "web", Name: "api"},
	}, nil, nil, "web")
	result, _ := m.Update(tea.KeyPressMsg{Code: tea.KeyDown})
	m = result.(model)

	_, cmd := m.executePaletteAction("compose:recreate")
	if cmd == nil {
		t.Fatal("expected the recreate action to produce a command")
	}
	msg := cmd()
	if _, ok := msg.(showStreamingLessMsg); !ok {
		t.Fatalf("expected recreate to stream into the viewer, got %T", msg)
	}
	if len(engine.recreateCalls) != 1 || engine.recreateCalls[0].Name != "web" {
		t.Fatalf("expected one Recreate call on project web, got %+v", engine.recreateCalls)
	}
	if len(engine.recreateServices) != 1 || engine.recreateServices[0] != "api" {
		t.Fatalf("expected the service to be targeted, got %v", engine.recreateServices)
	}
}

func TestComposeRecreateCmd_Streams(t *testing.T) {
	engine := &stubComposeEngine{}
	dir, file := composeFileFixture(t)
	msg := composeRecreateCmd(engine, docker.ComposeProject{
		Name:        "web",
		WorkingDir:  dir,
		ConfigFiles: []string{file},
	}, "api")()
	if _, ok := msg.(showStreamingLessMsg); !ok {
		t.Fatalf("expected recreate to stream, got %T", msg)
	}
}

func TestComposeDriftCmd_ReportsPerProjectStatus(t *testing.T) {
	engine := &stubComposeEngine{hashes: map[string]string{"api": "aaa"}}
	dir, file := composeFileFixture(t)
	projects := []docker.ProjectWithServices{{
		Project: docker.ComposeProject{
			Name:        "web",
			WorkingDir:  dir,
			ConfigFiles: []string{file},
		},
	}}
	containers := []*docker.Container{}

	msg := composeDriftCmd(engine, projects, stubContainers(containers), 1)()
	drift, ok := msg.(composeDriftMsg)
	if !ok {
		t.Fatalf("expected composeDriftMsg, got %T", msg)
	}
	if drift.drift["web"]["api"] != docker.ServiceNotCreated {
		t.Fatalf("expected api to be not created, got %q", drift.drift["web"]["api"])
	}
}

func TestComposeDriftCmd_SkipsProjectsWithoutFiles(t *testing.T) {
	engine := &stubComposeEngine{hashes: map[string]string{"api": "aaa"}}
	projects := []docker.ProjectWithServices{{
		Project: docker.ComposeProject{Name: "nofiles"},
	}}

	msg := composeDriftCmd(engine, projects, stubContainers(nil), 1)()
	drift := msg.(composeDriftMsg)
	if _, ok := drift.drift["nofiles"]; ok {
		t.Fatal("expected a project with no known files to be skipped entirely")
	}
}

// TestComposeDriftCmd_SkipsProjectsWhoseFilesAreNotLocal guards the recurring
// error banner. ConfigFiles comes from a container label recording the
// filesystem of whichever machine ran the compose client: for a project
// brought up on another machine, or whose file has moved, ConfigHashes fails
// for that project on every refresh cycle and the model banners it. A file
// dry cannot read is not a failure, it is an unknown file, which drift
// already handles by skipping silently.
func TestComposeDriftCmd_SkipsProjectsWhoseFilesAreNotLocal(t *testing.T) {
	engine := &stubComposeEngine{hashes: map[string]string{"api": "aaa"}}
	projects := []docker.ProjectWithServices{{
		Project: docker.ComposeProject{
			Name:        "web",
			WorkingDir:  "/srv/web",
			ConfigFiles: []string{filepath.Join(t.TempDir(), "gone.yaml")},
		},
	}}

	msg := composeDriftCmd(engine, projects, stubContainers(nil), 1)()
	drift, ok := msg.(composeDriftMsg)
	if !ok {
		t.Fatalf("expected composeDriftMsg, got %T", msg)
	}
	if _, ok := drift.drift["web"]; ok {
		t.Fatalf("expected a project whose files are not readable here to be skipped, got %+v", drift.drift)
	}
	if len(drift.failures) != 0 {
		t.Fatalf("a file dry cannot read must not raise an error banner, got %v", drift.failures)
	}
	if len(engine.hashesCalls) != 0 {
		t.Fatalf("compose must not be asked about a file dry cannot read, got %+v", engine.hashesCalls)
	}
}

func TestComposeConfigCmd_ShowsRenderedConfig(t *testing.T) {
	engine := &stubComposeEngine{configOutput: "services:\n  api:\n    image: nginx\n"}
	dir, file := composeFileFixture(t)

	msg := composeConfigCmd(engine, docker.ComposeProject{
		Name:        "web",
		WorkingDir:  dir,
		ConfigFiles: []string{file},
	})()
	less, ok := msg.(showLessMsg)
	if !ok {
		t.Fatalf("expected showLessMsg, got %T", msg)
	}
	if !strings.Contains(less.content, "image: nginx") {
		t.Fatalf("expected the rendered config, got %q", less.content)
	}
	if !strings.Contains(less.title, "web") {
		t.Fatalf("expected the project in the title, got %q", less.title)
	}
}

func TestComposeConfigCmd_NoFilesExplainsItself(t *testing.T) {
	engine := &stubComposeEngine{configOutput: "ignored"}

	msg := composeConfigCmd(engine, docker.ComposeProject{Name: "web"})()
	status, ok := msg.(statusMessageMsg)
	if !ok {
		t.Fatalf("expected a status message when the project has no files, got %T", msg)
	}
	if !strings.Contains(status.text, "no compose file") {
		t.Fatalf("expected the reason, got %q", status.text)
	}
}

// TestComposeDriftCmd_RunsAfterScanEnrichesProject drives the resolver path
// end to end through the model: a container-derived project with no
// ConfigFiles yet (the case docker.MergeScannedProject exists to fix) goes
// through appcompose.ProjectsLoadedMsg, then the scan's composeProjectsMsg,
// and asserts the model dispatches drift against the *enriched* project.
// Before the fix, composeDriftCmd was dispatched directly from
// ProjectsLoadedMsg against the pre-scan project (still zero ConfigFiles),
// batched concurrently with composeScanCmd against the same backing slice —
// a data race, and one where drift always saw an empty ConfigFiles and so
// silently skipped every scan-enriched project forever, F5 included.
func TestComposeDriftCmd_RunsAfterScanEnrichesProject(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "compose.yaml"), []byte("services: {}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	engine := &stubComposeEngine{
		hashes:      map[string]string{"api": "aaa"},
		resolveName: "web",
	}
	m := newTestModel()
	m.view = ComposeProjects
	m.composeCLI = engine
	m.workingDir = dir

	result, cmd := m.Update(appcompose.ProjectsLoadedMsg{
		Projects: []docker.ProjectWithServices{{
			Project: docker.ComposeProject{Name: "web"}, // no ConfigFiles: labels lacked them
		}},
	})
	m = result.(model)
	if cmd == nil {
		t.Fatal("expected the scan command to be dispatched from ProjectsLoadedMsg")
	}

	scanMsg := cmd()
	projectsMsg, ok := scanMsg.(composeProjectsMsg)
	if !ok {
		t.Fatalf("expected composeProjectsMsg from the scan, got %T", scanMsg)
	}
	if len(projectsMsg.projects) != 1 || len(projectsMsg.projects[0].Project.ConfigFiles) == 0 {
		t.Fatalf("expected the scan to enrich the project with files, got %+v", projectsMsg.projects)
	}

	result, cmd = m.Update(projectsMsg)
	m = result.(model)
	if cmd == nil {
		t.Fatal("expected drift to be dispatched against the scan-enriched project")
	}

	driftMsg, ok := cmd().(composeDriftMsg)
	if !ok {
		t.Fatalf("expected composeDriftMsg, got %T", cmd())
	}
	if driftMsg.drift["web"]["api"] != docker.ServiceNotCreated {
		t.Fatalf("expected drift to be computed for the scan-enriched project, got %q", driftMsg.drift["web"]["api"])
	}
	if len(engine.hashesCalls) != 1 || len(engine.hashesCalls[0].Files) == 0 {
		t.Fatalf("expected ConfigHashes to be called with the resolved files, got %+v", engine.hashesCalls)
	}
}

// composeCycleModel returns a model sitting in the Compose Projects view
// with one container event pending, which is what the 250ms refresh window
// coalesces into a flushRefreshMsg.
func composeCycleModel(t *testing.T) model {
	t.Helper()
	m := newTestModel()
	m.view = ComposeProjects
	m.composeCLI = &stubHashesOnlyEngine{hashes: map[string]string{"api": "aaa"}}
	m.pendingRefresh[docker.ContainerSource] = true
	return m
}

// composeReloadDispatched reports whether flushRefreshMsg dispatched a
// compose project reload. With only a container event pending and the Compose
// Projects view active, that reload is the only command the refresh can
// produce, so tea.Batch hands it back directly (Batch returns nil for no
// commands and the command itself for exactly one) and running it yields the
// ProjectsLoadedMsg that starts a scan/drift cycle.
func composeReloadDispatched(t *testing.T, m model) (model, bool) {
	t.Helper()
	result, cmd := m.Update(flushRefreshMsg{})
	next := result.(model)
	if cmd == nil {
		return next, false
	}
	msg := cmd()
	if _, ok := msg.(appcompose.ProjectsLoadedMsg); !ok {
		t.Fatalf("expected the refresh to dispatch a compose reload, got %T", msg)
	}
	return next, true
}

// TestFlushRefresh_SkipsComposeReloadWhileCycleInFlight is the throttle the
// feature's own happy path needs: pressing u produces a stream of container
// events, and every ProjectsLoadedMsg dispatches `compose config --format
// json` plus one `compose config --hash=*` per project. At a 250ms coalescing
// window that spawns overlapping batches of subprocesses, each 150-400ms of
// CPU, for the whole duration of the up. Only one cycle may be in flight.
func TestFlushRefresh_SkipsComposeReloadWhileCycleInFlight(t *testing.T) {
	m := composeCycleModel(t)

	// A cycle starts: this is what a reload's ProjectsLoadedMsg does.
	result, _ := m.Update(appcompose.ProjectsLoadedMsg{
		Projects: []docker.ProjectWithServices{{
			Project: docker.ComposeProject{Name: "web"},
		}},
	})
	m = result.(model)
	m.pendingRefresh[docker.ContainerSource] = true

	if _, dispatched := composeReloadDispatched(t, m); dispatched {
		t.Fatal("a refresh during an in-flight scan/drift cycle must not start a second one")
	}
}

// TestFlushRefresh_ResumesComposeReloadAfterCycleCompletes is the other half
// of the guard: the throttle must be a throttle, not an off switch. Without
// this, the finding could be "fixed" by never refreshing the Compose view
// again.
func TestFlushRefresh_ResumesComposeReloadAfterCycleCompletes(t *testing.T) {
	m := composeCycleModel(t)

	if _, dispatched := composeReloadDispatched(t, m); !dispatched {
		t.Fatal("the first refresh must reload the compose projects")
	}

	result, _ := m.Update(appcompose.ProjectsLoadedMsg{
		Projects: []docker.ProjectWithServices{{Project: docker.ComposeProject{Name: "web"}}},
	})
	m = result.(model)
	// composeDriftMsg ends the cycle, whatever it found. It has to name the
	// cycle it belongs to: checks are tracked by generation.
	result, _ = m.Update(composeDriftMsg{gen: m.composeDriftGen})
	m = result.(model)
	m.pendingRefresh[docker.ContainerSource] = true

	if _, dispatched := composeReloadDispatched(t, m); !dispatched {
		t.Fatal("the refresh must resume once the cycle has completed")
	}
}

// TestComposeCycle_CoalescesTheSkippedReload proves the throttle drops
// nothing: the events skipped while a cycle was running are worth one reload
// once it ends. Without this the last container event of an `up` — the one
// that turns the project's STATUS from "not created" to "running" — could
// land during the final cycle and never be picked up, leaving the view
// showing pre-up state until the user pressed F5.
func TestComposeCycle_CoalescesTheSkippedReload(t *testing.T) {
	m := composeCycleModel(t)

	result, _ := m.Update(appcompose.ProjectsLoadedMsg{
		Projects: []docker.ProjectWithServices{{Project: docker.ComposeProject{Name: "web"}}},
	})
	m = result.(model)
	m.pendingRefresh[docker.ContainerSource] = true
	m, dispatched := composeReloadDispatched(t, m)
	if dispatched {
		t.Fatal("precondition: the refresh must be skipped while the cycle runs")
	}

	result, cmd := m.Update(composeDriftMsg{gen: m.composeDriftGen})
	m = result.(model)
	if cmd == nil {
		t.Fatal("expected the deferred reload to run once the cycle ended")
	}
	if _, ok := cmd().(appcompose.ProjectsLoadedMsg); !ok {
		t.Fatalf("expected the deferred compose reload, got %T", cmd())
	}
}

// TestComposeCycle_DoesNotReloadWhenNothingWasSkipped is the brake on the
// coalescing: a cycle that ended with no skipped refresh must not start
// another one, or the two would feed each other forever.
func TestComposeCycle_DoesNotReloadWhenNothingWasSkipped(t *testing.T) {
	m := composeCycleModel(t)
	m.pendingRefresh = map[docker.SourceType]bool{}

	result, _ := m.Update(appcompose.ProjectsLoadedMsg{
		Projects: []docker.ProjectWithServices{{Project: docker.ComposeProject{Name: "web"}}},
	})
	m = result.(model)

	if _, cmd := m.Update(composeDriftMsg{gen: m.composeDriftGen}); cmd != nil {
		t.Fatalf("a completed cycle must not chain into another, got a command producing %T", cmd())
	}
}

// TestComposeCycle_StreamCloseRunsTheDeferredReload covers the other skip
// reason: the refreshes dropped behind an open `up` viewer are what used to
// keep the view underneath current, so closing the viewer must run one.
func TestComposeCycle_StreamCloseRunsTheDeferredReload(t *testing.T) {
	m := composeCycleModel(t)
	reader := &stubStreamReader{}
	result, _ := m.Update(showStreamingLessMsg{title: "Compose up: web", reader: reader})
	m = result.(model)
	m.pendingRefresh[docker.ContainerSource] = true
	m, dispatched := composeReloadDispatched(t, m)
	if dispatched {
		t.Fatal("precondition: the refresh must be skipped behind the streaming viewer")
	}

	result, cmd := m.Update(appui.CloseOverlayMsg{})
	m = result.(model)
	if cmd == nil {
		t.Fatal("expected closing the streaming viewer to run the deferred reload")
	}
	if _, ok := cmd().(appcompose.ProjectsLoadedMsg); !ok {
		t.Fatalf("expected the deferred compose reload, got %T", cmd())
	}
	if !reader.closed {
		t.Fatal("expected the stream reader to be closed with the overlay")
	}
}

// TestFlushRefresh_SkipsComposeReloadWhileStreamingViewerOpen covers the
// other half of the storm: the events arrive precisely because a streamed
// `up` or `down` is running, and its output is covering the view the refresh
// would repaint. Nothing on screen can show the result, so the subprocesses
// are pure waste.
func TestFlushRefresh_SkipsComposeReloadWhileStreamingViewerOpen(t *testing.T) {
	for _, view := range []viewMode{ComposeProjects, ComposeServices} {
		m := composeCycleModel(t)
		m.view = view
		reader := &stubStreamReader{}
		result, _ := m.Update(showStreamingLessMsg{title: "Compose up: web", reader: reader})
		m = result.(model)
		m.pendingRefresh[docker.ContainerSource] = true

		if _, cmd := m.Update(flushRefreshMsg{}); cmd != nil {
			t.Fatalf("view %v: a refresh behind an open streaming viewer must not reload compose, got a command producing %T", view, cmd())
		}
	}
}

// TestComposeDemoPath_UpTargetsTheScannedFile is the feature's own demo path,
// end to end through the model: no containers anywhere, a compose file in
// dry's working directory, the project appears in the view as "not created",
// and pressing u brings it up *with* -f pointing at that file. Nothing else
// asserted the last step — TestComposeDriftCmd_RunsAfterScanEnrichesProject
// stops at drift — so an enrichment-ordering regression would have let `u`
// run against a project with no files at all, which is how one project's key
// press ends up creating another directory's services.
func TestComposeDemoPath_UpTargetsTheScannedFile(t *testing.T) {
	dir, file := composeFileFixture(t)
	engine := &stubComposeEngine{resolveName: "web"}
	m := newTestModel()
	m.view = ComposeProjects
	m.composeCLI = engine
	m.workingDir = dir

	// Nothing in docker ps: the project is invisible until the scan finds it.
	result, cmd := m.Update(appcompose.ProjectsLoadedMsg{Projects: nil})
	m = result.(model)
	if cmd == nil {
		t.Fatal("expected the working-directory scan to be dispatched")
	}
	scanned, ok := cmd().(composeProjectsMsg)
	if !ok {
		t.Fatalf("expected composeProjectsMsg from the scan, got %T", cmd())
	}

	result, _ = m.Update(scanned)
	m = result.(model)

	// The project is on screen, and it says what state it is in.
	p := m.composeProjects.SelectedProject()
	if p == nil || p.Name != "web" {
		t.Fatalf("expected the scanned project to be selected, got %+v", p)
	}
	if p.Status != docker.ProjectNotCreated {
		t.Fatalf("expected the file-only project to read as not created, got %q", p.Status)
	}
	if view := ansi.Strip(m.composeProjects.View()); !strings.Contains(view, "not created") {
		t.Fatalf("expected the view to show the project's status, got:\n%s", view)
	}

	// u brings it up once confirmed, and compose is told which file to use.
	result, _ = m.Update(tea.KeyPressMsg{Code: 'u'})
	m = result.(model)
	if m.overlay != overlayPrompt {
		t.Fatal("expected u on a project row to ask first")
	}
	// Off the prompt, not injected: the id it carries is what decides which
	// project the confirmation acts on.
	if tag, id := answerPrompt(t, m); tag != "compose-project-up" || id != "web" {
		t.Fatalf("expected the project-up confirmation for web, got %q %q", tag, id)
	}
	_, cmd = m.Update(appui.PromptResultMsg{Confirmed: true, Tag: "compose-project-up", ID: "web"})
	if cmd == nil {
		t.Fatal("expected the confirmation to run up")
	}
	if msg := cmd(); !isStreamingLess(msg) {
		t.Fatalf("expected up to stream into the viewer, got %T (%+v)", msg, msg)
	}
	if len(engine.upCalls) != 1 {
		t.Fatalf("expected exactly one Up call, got %+v", engine.upCalls)
	}
	if got := engine.upCalls[0].Files; len(got) != 1 || got[0] != file {
		t.Fatalf("expected up to target the scanned compose file %q, got %v", file, got)
	}
	if engine.upCalls[0].WorkingDir != dir {
		t.Fatalf("expected up to run against the scanned directory %q, got %q", dir, engine.upCalls[0].WorkingDir)
	}
}

// isStreamingLess reports whether msg opened the streaming viewer, closing
// the reader so the test leaves nothing running.
func isStreamingLess(msg tea.Msg) bool {
	streaming, ok := msg.(showStreamingLessMsg)
	if ok && streaming.reader != nil {
		_ = streaming.reader.Close()
	}
	return ok
}

// TestComposeDriftCmd_NoResolverDispatchesDriftDirectly covers the other
// branch of the fix: when m.composeCLI does not implement composeResolver,
// composeProjectsMsg never arrives, so ProjectsLoadedMsg must dispatch drift
// itself rather than leaving the SYNC column permanently empty.
func TestComposeDriftCmd_NoResolverDispatchesDriftDirectly(t *testing.T) {
	engine := &stubHashesOnlyEngine{hashes: map[string]string{"api": "aaa"}}
	_, file := composeFileFixture(t)
	m := newTestModel()
	m.view = ComposeProjects
	m.composeCLI = engine

	result, cmd := m.Update(appcompose.ProjectsLoadedMsg{
		Projects: []docker.ProjectWithServices{{
			Project: docker.ComposeProject{Name: "web", ConfigFiles: []string{file}},
		}},
	})
	m = result.(model)
	if cmd == nil {
		t.Fatal("expected drift to be dispatched directly when there is no resolver")
	}
	driftMsg, ok := cmd().(composeDriftMsg)
	if !ok {
		t.Fatalf("expected composeDriftMsg, got %T", cmd())
	}
	if _, ok := driftMsg.drift["web"]; !ok {
		t.Fatal("expected drift to have been computed for the project")
	}
}

// stubHashesOnlyEngine is a composeEngine that does NOT implement
// composeResolver, needed to prove ProjectsLoadedMsg's no-resolver branch
// (stubComposeEngine always implements composeResolver, so it cannot
// exercise this path).
type stubHashesOnlyEngine struct {
	hashes map[string]string
}

func (s *stubHashesOnlyEngine) Up(context.Context, composecli.Project, ...string) (io.ReadCloser, error) {
	return nil, errors.New("not implemented")
}

func (s *stubHashesOnlyEngine) Down(context.Context, composecli.Project) (io.ReadCloser, error) {
	return nil, errors.New("not implemented")
}

func (s *stubHashesOnlyEngine) Recreate(context.Context, composecli.Project, string) (io.ReadCloser, error) {
	return nil, errors.New("not implemented")
}

func (s *stubHashesOnlyEngine) Config(context.Context, composecli.Project) (string, error) {
	return "", errors.New("not implemented")
}

func (s *stubHashesOnlyEngine) ConfigHashes(_ context.Context, _ composecli.Project) (map[string]string, error) {
	return s.hashes, nil
}

// TestComposeDriftCmd_ReportsConfigHashesFailure guards finding 4: a
// ConfigHashes error must never be swallowed. It has to arrive in
// composeDriftMsg.failures keyed by its project, which is what lets merge
// keep that project's last known SYNC and the model banner the failure.
func TestComposeDriftCmd_ReportsConfigHashesFailure(t *testing.T) {
	failure := errors.New("compose config --hash failed")
	engine := &stubComposeEngine{hashesErr: failure}
	_, file := composeFileFixture(t)
	projects := []docker.ProjectWithServices{{
		Project: docker.ComposeProject{
			Name:        "web",
			ConfigFiles: []string{file},
		},
	}}

	msg := composeDriftCmd(engine, projects, stubContainers(nil), 1)()
	drift, ok := msg.(composeDriftMsg)
	if !ok {
		t.Fatalf("expected composeDriftMsg, got %T", msg)
	}
	if _, ok := drift.drift["web"]; ok {
		t.Fatalf("expected the failed project to be dropped from drift, got %+v", drift.drift)
	}
	why, reported := drift.failures["web"]
	if !reported {
		t.Fatal("expected the ConfigHashes failure to be reported, not swallowed")
	}
	if !strings.Contains(why, failure.Error()) {
		t.Fatalf("expected the failure to name the cause, got %q", why)
	}
}

// TestComposeDownCmd_TargetsByLabelWhenTheFilesAreNotHere is the point of
// down being unguarded. Compose removes a project by its container labels,
// so down is the one action that works when the recorded compose file is not
// on this host, but only if dry stops handing compose that path: `-f` on a
// file that does not exist makes compose fail on the file instead of
// removing the project, in exactly the moved-file and elsewhere case
// where removing by label is the only thing that can work.
func TestComposeDownCmd_TargetsByLabelWhenTheFilesAreNotHere(t *testing.T) {
	engine := &stubComposeEngine{}
	p := docker.ComposeProject{
		Name:        "web",
		WorkingDir:  "/srv/web",
		ConfigFiles: []string{"/srv/web/compose.yaml"},
	}

	msg := composeDownCmd(engine, p)()
	if _, ok := msg.(showStreamingLessMsg); !ok {
		t.Fatalf("expected down to run, got %T", msg)
	}
	if len(engine.downCalls) != 1 {
		t.Fatalf("expected one Down call, got %+v", engine.downCalls)
	}
	call := engine.downCalls[0]
	if call.Name != "web" {
		t.Fatalf("expected the project name to be kept, got %+v", call)
	}
	if len(call.Files) != 0 {
		t.Fatalf("expected no -f for files that are not on this host, got %v", call.Files)
	}
	if call.WorkingDir != "" {
		t.Fatalf("expected no --project-directory for a path that is not here, got %q", call.WorkingDir)
	}
}

// A project whose files are here keeps them: that is what `docker compose
// down` in the project directory does. down does not need them (it finds
// containers, networks and volumes by label), so this pins the narrower
// rule: paths are dropped only when they are unusable.
func TestComposeDownCmd_KeepsFilesThatAreHere(t *testing.T) {
	engine := &stubComposeEngine{}
	dir, file := composeFileFixture(t)

	msg := composeDownCmd(engine, docker.ComposeProject{
		Name:        "web",
		WorkingDir:  dir,
		ConfigFiles: []string{file},
	})()
	if _, ok := msg.(showStreamingLessMsg); !ok {
		t.Fatalf("expected down to run, got %T", msg)
	}
	if len(engine.downCalls) != 1 {
		t.Fatalf("expected one Down call, got %+v", engine.downCalls)
	}
	call := engine.downCalls[0]
	if len(call.Files) != 1 || call.Files[0] != file {
		t.Fatalf("expected the readable file to be passed, got %v", call.Files)
	}
	if call.WorkingDir != dir {
		t.Fatalf("expected the working directory to be passed, got %q", call.WorkingDir)
	}
}

// TestComposeDriftCmd_GivesUpOnTheWholeCycle covers the failure the per-call
// bound does not: the reason a compose call hangs, a daemon that accepted
// the connection and went quiet, hangs every project at once, and the cycle
// walks them one at a time, so the per-call bounds add up while the model
// gates every compose reload on the cycle finishing.
func TestComposeDriftCmd_GivesUpOnTheWholeCycle(t *testing.T) {
	restore := composeDriftBudget
	composeDriftBudget = 200 * time.Millisecond
	t.Cleanup(func() { composeDriftBudget = restore })

	engine := &stubComposeEngine{hangHashes: true}
	dir, file := composeFileFixture(t)
	var projects []docker.ProjectWithServices
	for _, name := range []string{"one", "two", "three", "four", "five"} {
		projects = append(projects, docker.ProjectWithServices{
			Project: docker.ComposeProject{Name: name, WorkingDir: dir, ConfigFiles: []string{file}},
		})
	}

	start := time.Now()
	msg := composeDriftCmd(engine, projects, stubContainers(nil), 1)()
	elapsed := time.Since(start)

	drift, ok := msg.(composeDriftMsg)
	if !ok {
		t.Fatalf("expected composeDriftMsg, got %T", msg)
	}
	// The expiry is recorded against every project the cycle did not reach,
	// which is the shape composeDriftMsg has: merge then keeps each of their
	// last known SYNC values, and newDriftFailures names two and counts the
	// rest rather than filling a one-line bar with one sentence per project.
	if len(drift.failures) == 0 {
		t.Fatal("expected the cycle to report that it gave up")
	}
	for name, why := range drift.failures {
		if !strings.Contains(why, "gave up") {
			t.Errorf("expected %s's failure to say the cycle gave up, got %q", name, why)
		}
		if !strings.Contains(why, "unchecked") {
			t.Errorf("expected %s's failure to say how many were left, got %q", name, why)
		}
		if got := strings.Count(why, "gave up"); got != 1 {
			t.Errorf("expected the expiry once per project, got %d copies: %q", got, why)
		}
	}
	// The hung one plus the four it never reached.
	if len(drift.failures) != 5 {
		t.Errorf("expected every project accounted for, got %d: %v", len(drift.failures), drift.failures)
	}
	// One hung call spends the budget; the rest must not each spend another.
	if len(engine.hashesCalls) > 2 {
		t.Fatalf("expected the cycle to stop after the budget ran out, got %d calls",
			len(engine.hashesCalls))
	}
	// Comfortably inside the stub's own 2s fallback, which is what would
	// otherwise absorb a regression: pass context.Background() to
	// ConfigHashes instead of ctx and the hung call returns on the fallback,
	// the loop breaks between calls on ctx.Err(), and every assertion above
	// still holds. Only the elapsed time tells the two apart.
	if elapsed > time.Second {
		t.Fatalf("expected the cycle to end with its 200ms budget, took %s", elapsed)
	}
}

// A probe that timed out is not the same as a plugin that is not installed:
// the plugin may be there, dry behaves as if it were not, and "install it"
// is the wrong advice. The timeout is reported once, at startup; a plugin
// that is simply missing stays silent, because every compose key says so
// when pressed.
func TestComposeDetected_ProbeTimeoutIsReported(t *testing.T) {
	m := newTestModel()
	_, cmd := m.Update(composeDetectedMsg{
		err: fmt.Errorf("%w after 10s", composecli.ErrProbeTimeout),
	})
	if cmd == nil {
		t.Fatal("expected a probe timeout to be reported")
	}
	status, ok := cmd().(statusMessageMsg)
	if !ok {
		t.Fatalf("expected a status message, got %T", cmd())
	}
	if !strings.Contains(status.text, "timed out") {
		t.Fatalf("expected the message to name the timeout, got %q", status.text)
	}
}

func TestComposeDetected_MissingPluginStaysSilent(t *testing.T) {
	m := newTestModel()
	_, cmd := m.Update(composeDetectedMsg{
		err: errors.New("docker compose plugin not available: exec: \"docker\": executable file not found in $PATH"),
	})
	if cmd != nil {
		t.Fatalf("expected no startup message for a missing plugin, got %T", cmd())
	}
}

// TestComposeServicesView_UBringsUpTheSelectedServiceOnAFreshView is the
// headline of the cursor fix: entering a project and pressing u, the
// documented "bring this up" key, with no navigation in between, must
// actually run compose up for the first service, not report that nothing is
// selected. The view's first row is the "Services" section header, so this
// only works because the cursor skips it.
func TestComposeServicesView_UBringsUpTheSelectedServiceOnAFreshView(t *testing.T) {
	engine := &stubComposeEngine{}
	dir, file := composeFileFixture(t)
	m := newTestModel()
	m.view = ComposeServices
	m.composeCLI = engine
	m.selectedProject = "web"
	m.composeProjects.SetProjects([]docker.ProjectWithServices{{
		Project: docker.ComposeProject{Name: "web", WorkingDir: dir, ConfigFiles: []string{file}},
	}})
	m.composeServices.SetSize(120, 40)
	m.composeServices.SetServices([]docker.ComposeService{
		{Project: "web", Name: "api"},
		{Project: "web", Name: "worker"},
	}, nil, nil, "web")

	_, cmd := m.Update(tea.KeyPressMsg{Code: 'u'})
	if cmd == nil {
		t.Fatal("expected u to produce a command")
	}
	if _, ok := cmd().(showStreamingLessMsg); !ok {
		t.Fatalf("expected up to stream into the viewer, got %T", cmd())
	}
	if len(engine.upCalls) != 1 || engine.upCalls[0].Name != "web" {
		t.Fatalf("expected one Up call on project web, got %+v", engine.upCalls)
	}
	if len(engine.upServices) != 1 || engine.upServices[0] != "api" {
		t.Fatalf("expected the first service to be targeted, got %v", engine.upServices)
	}
}

// TestComposeServicesView_KeysThatNeedAServiceSaySo covers the rows the
// Compose Services view holds that are not services. The cursor lands on a
// network by walking past the services, or immediately when the project has
// none, and six documented keys can do nothing with one. Each has to say
// so: a key that does nothing and says nothing reads as broken.
func TestComposeServicesView_KeysThatNeedAServiceSaySo(t *testing.T) {
	keys := []tea.KeyPressMsg{
		{Code: 'u'},
		{Code: 'l'},
		{Code: 's', Mod: tea.ModCtrl},
		{Code: 't', Mod: tea.ModCtrl},
		{Code: 'r', Mod: tea.ModCtrl},
		{Code: 'e', Mod: tea.ModCtrl},
	}
	for _, key := range keys {
		t.Run(key.String(), func(t *testing.T) {
			m := newTestModel()
			m.view = ComposeServices
			m.composeCLI = &stubComposeEngine{}
			m.selectedProject = "web"
			m.composeServices.SetSize(120, 40)
			m.composeServices.SetServices(nil,
				[]docker.ComposeNetwork{{Name: "web_default"}}, nil, "web")
			if m.composeServices.SelectedNetwork() == nil {
				t.Fatal("precondition: the cursor must sit on the network row")
			}

			_, cmd := m.Update(key)
			if cmd == nil {
				t.Fatalf("expected %s to explain itself on a network row", key.String())
			}
			status, ok := cmd().(statusMessageMsg)
			if !ok {
				t.Fatalf("expected a status message, got %T", cmd())
			}
			if !strings.Contains(strings.ToLower(status.text), "service") {
				t.Fatalf("expected the message to mention services, got %q", status.text)
			}
			// The row is visibly selected, so the message must not ask the
			// user to select something. u says where the whole-project
			// version lives instead of naming the row, and says it as a
			// place rather than a keystroke: in workspace mode with a
			// pinned context the first esc only clears the pin, so "esc,
			// then u" would be wrong there.
			if key.String() == "u" {
				if !strings.Contains(status.text, "projects list") {
					t.Fatalf("expected u to point at the projects list, got %q", status.text)
				}
				if strings.Contains(status.text, "esc") {
					t.Fatalf("expected no keystroke count in the message, got %q", status.text)
				}
			} else if !strings.Contains(status.text, "a network is selected") {
				t.Fatalf("expected the message to name the selected row, got %q", status.text)
			}
		})
	}
}

// enter acts on all three resource kinds, so the only row it cannot use is
// one that resolves to nothing: the section header a filter matching nothing
// else leaves, or no row at all, which is what an empty project has, since
// refreshRows emits a header only alongside items.
func TestComposeServicesView_EnterWithNothingSelectedSaysSo(t *testing.T) {
	m := newTestModel()
	m.view = ComposeServices
	m.selectedProject = "web"
	m.composeServices.SetSize(120, 40)
	m.composeServices.SetServices(nil, nil, nil, "web")

	_, cmd := m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	if cmd == nil {
		t.Fatal("expected enter to explain itself with nothing selected")
	}
	status, ok := cmd().(statusMessageMsg)
	if !ok {
		t.Fatalf("expected a status message, got %T", cmd())
	}
	if !strings.Contains(status.text, "no services, networks or volumes") {
		t.Fatalf("expected the message to say the project is empty, got %q", status.text)
	}
}

// TestComposeServicesView_OpeningAnotherProjectDropsTheOldRows guards
// against acting on the wrong project. Entering a project switches the view
// and records the project name, then loads its resources asynchronously;
// until that load lands the view would otherwise still hold the previous
// project's rows, and u, which now always finds a service under the cursor
// , would bring up a service of the project the user just left.
func TestComposeServicesView_OpeningAnotherProjectDropsTheOldRows(t *testing.T) {
	engine := &stubComposeEngine{}
	dir, file := composeFileFixture(t)
	m := newTestModel()
	m.view = ComposeProjects
	m.composeCLI = engine
	m.composeProjects.SetSize(120, 40)
	m.composeProjects.SetProjects([]docker.ProjectWithServices{
		{Project: docker.ComposeProject{Name: "alpha", WorkingDir: dir, ConfigFiles: []string{file}}},
		{Project: docker.ComposeProject{Name: "beta", WorkingDir: dir, ConfigFiles: []string{file}}},
	})

	// Open alpha and let its resources arrive.
	result, _ := m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	m = result.(model)
	result, _ = m.Update(appcompose.ServicesLoadedMsg{
		Services: []docker.ComposeService{{Project: "alpha", Name: "alpha-api"}},
		Project:  "alpha",
	})
	m = result.(model)
	if svc := m.composeServices.SelectedService(); svc == nil || svc.Name != "alpha-api" {
		t.Fatalf("precondition: expected alpha-api selected, got %+v", svc)
	}

	// Back out, move to beta, open it. Its resources have not arrived yet.
	result, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyEscape})
	m = result.(model)
	result, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyDown})
	m = result.(model)
	result, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	m = result.(model)
	if m.selectedProject != "beta" {
		t.Fatalf("precondition: expected beta to be selected, got %q", m.selectedProject)
	}

	_, cmd := m.Update(tea.KeyPressMsg{Code: 'u'})
	if cmd != nil {
		if _, ok := cmd().(statusMessageMsg); !ok {
			t.Fatalf("expected u to report nothing selected, got %T", cmd())
		}
	}
	if len(engine.upCalls) != 0 {
		t.Fatalf("u must not act on the project the user left, got %+v with services %v",
			engine.upCalls, engine.upServices)
	}

	// Once beta's resources arrive, u targets beta.
	result, _ = m.Update(appcompose.ServicesLoadedMsg{
		Services: []docker.ComposeService{{Project: "beta", Name: "beta-web"}},
		Project:  "beta",
	})
	m = result.(model)
	if _, cmd := m.Update(tea.KeyPressMsg{Code: 'u'}); cmd != nil {
		cmd()
	}
	if len(engine.upCalls) != 1 || engine.upCalls[0].Name != "beta" {
		t.Fatalf("expected one Up call on beta, got %+v", engine.upCalls)
	}
	if len(engine.upServices) != 1 || engine.upServices[0] != "beta-web" {
		t.Fatalf("expected beta-web to be targeted, got %v", engine.upServices)
	}
}

// A load that finishes for a project the user has already left must be
// dropped: two are in flight after a quick esc-and-enter, and the older,
// slower one would replace the current project's rows.
func TestComposeServicesLoaded_IgnoresAStaleProjectsLoad(t *testing.T) {
	m := newTestModel()
	m.view = ComposeServices
	m.selectedProject = "beta"
	m.composeServices.SetSize(120, 40)
	m.composeServices.SetProject("beta")
	result, _ := m.Update(appcompose.ServicesLoadedMsg{
		Services: []docker.ComposeService{{Project: "beta", Name: "beta-web"}},
		Project:  "beta",
	})
	m = result.(model)

	result, _ = m.Update(appcompose.ServicesLoadedMsg{
		Services: []docker.ComposeService{{Project: "alpha", Name: "alpha-api"}},
		Project:  "alpha",
	})
	m = result.(model)

	if svc := m.composeServices.SelectedService(); svc == nil || svc.Name != "beta-web" {
		t.Fatalf("expected beta's rows to survive the stale load, got %+v", svc)
	}
}

// TestComposeServicesView_ResizeKeepsTheSelectionOnScreen drives the resize
// through the model the way a terminal does, since that is where the
// selection went missing: the view is sized by resizeContentModels on every
// WindowSizeMsg, and a shorter window under a cursor near the bottom used to
// leave the highlighted row unrendered while every key still acted on it.
func TestComposeServicesView_ResizeKeepsTheSelectionOnScreen(t *testing.T) {
	appui.InitStyles()
	m := newTestModel()
	m.view = ComposeServices
	m.selectedProject = "web"

	services := make([]docker.ComposeService, 20)
	for i := range services {
		services[i] = docker.ComposeService{Project: "web", Name: fmt.Sprintf("svc%02d", i)}
	}
	result, _ := m.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	m = result.(model)
	m.composeServices.SetServices(services,
		[]docker.ComposeNetwork{{Name: "web_default"}},
		[]docker.ComposeVolume{{Name: "web_data"}}, "web")

	// Walk to the bottom of the list.
	for range len(services) + 6 {
		result, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyDown})
		m = result.(model)
	}
	if v := m.composeServices.SelectedVolume(); v == nil || v.Name != "web_data" {
		t.Fatalf("precondition: expected the last row selected, got %+v", v)
	}

	marker := strings.SplitN(appui.SelectedRowStyle.Render("x"), "x", 2)[0]
	for _, height := range []int{40, 34, 30, 24, 18, 12} {
		result, _ = m.Update(tea.WindowSizeMsg{Width: 120, Height: height})
		m = result.(model)
		var highlighted string
		for _, line := range strings.Split(m.composeServices.View(), "\n") {
			if strings.Contains(line, marker) {
				highlighted = ansi.Strip(line)
				break
			}
		}
		if !strings.Contains(highlighted, "web_data") {
			t.Fatalf("height %d: expected the selected row to be highlighted on screen, got %q",
				height, highlighted)
		}
	}
}

// Two resource loads for the same project can be in flight, an f5 during a
// container event, or a quick esc and re-enter, and by arrival order the
// older one's rows win, so the view shows a service the project no longer
// has until the next event.
func TestComposeServicesLoaded_IgnoresAnOlderLoadOfTheSameProject(t *testing.T) {
	m := newTestModel()
	m.view = ComposeServices
	m.selectedProject = "web"
	m.composeServices.SetSize(120, 40)

	result, _ := m.Update(appcompose.ServicesLoadedMsg{
		Project: "web",
		Gen:     2,
		Services: []docker.ComposeService{
			{Project: "web", Name: "api"},
			{Project: "web", Name: "worker"},
		},
	})
	m = result.(model)

	result, _ = m.Update(appcompose.ServicesLoadedMsg{
		Project:  "web",
		Gen:      1,
		Services: []docker.ComposeService{{Project: "web", Name: "api"}},
	})
	m = result.(model)

	view := ansi.Strip(m.composeServices.View())
	if !strings.Contains(view, "worker") {
		t.Fatalf("expected the newer load's rows to survive the older one, got:\n%s", view)
	}
	if !strings.Contains(view, "Services (2)") {
		t.Fatalf("expected the newer load's count to survive, got:\n%s", view)
	}
}

// A filter can narrow the list to a section header, the one row the cursor
// rests on that resolves to nothing. Telling the user that nothing is
// selected there contradicts the row highlighted on screen, so the message
// names the filter instead.
func TestComposeServicesView_KeysUnderAHeaderOnlyFilterNameTheFilter(t *testing.T) {
	m := newTestModel()
	m.view = ComposeServices
	m.composeCLI = &stubComposeEngine{}
	m.selectedProject = "web"
	m.composeServices.SetSize(120, 40)
	m.composeServices.SetServices([]docker.ComposeService{
		{Project: "web", Name: "api"},
		{Project: "web", Name: "worker"},
	}, nil, nil, "web")

	result, _ := m.Update(tea.KeyPressMsg{Code: '%'})
	m = result.(model)
	for _, r := range "services (2)" {
		result, _ = m.Update(tea.KeyPressMsg{Code: r, Text: string(r)})
		m = result.(model)
	}
	if m.composeServices.SelectedService() != nil {
		t.Fatalf("precondition: expected no service to match, got %+v", m.composeServices.SelectedService())
	}
	// Leave the filter committed, the way enter does.
	result, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	m = result.(model)

	_, cmd := m.Update(tea.KeyPressMsg{Code: 'l'})
	if cmd == nil {
		t.Fatal("expected l to explain itself")
	}
	status, ok := cmd().(statusMessageMsg)
	if !ok {
		t.Fatalf("expected a status message, got %T", cmd())
	}
	if !strings.Contains(status.text, "filter") {
		t.Fatalf("expected the message to name the filter, got %q", status.text)
	}
}

// c renders the open project's configuration, and the one state where it has
// no project is the one where it used to do nothing and say nothing.
func TestComposeServicesView_ConfigWithNoProjectSaysSo(t *testing.T) {
	m := newTestModel()
	m.view = ComposeServices
	m.composeCLI = &stubComposeEngine{}

	_, cmd := m.Update(tea.KeyPressMsg{Code: 'c'})
	if cmd == nil {
		t.Fatal("expected c to explain itself with no project open")
	}
	status, ok := cmd().(statusMessageMsg)
	if !ok {
		t.Fatalf("expected a status message, got %T", cmd())
	}
	if !strings.Contains(strings.ToLower(status.text), "project") {
		t.Fatalf("expected the message to name what is missing, got %q", status.text)
	}
}

// The generation that keeps an older resource load from replacing a newer
// one has to advance on every path that starts a load. A container
// operation's success is one of them, and it reloads through loadViewData,
// which is where a value receiver would have left the generation on a
// discarded copy, handing two loads the same number and making the guard
// unable to tell them apart.
func TestComposeServicesView_EveryReloadPathAdvancesTheGeneration(t *testing.T) {
	m := newTestModel()
	m.view = ComposeServices
	m.selectedProject = "web"

	seen := map[uint64]string{}
	record := func(path string, cmd tea.Cmd) {
		t.Helper()
		if cmd == nil {
			t.Fatalf("%s: expected a load", path)
		}
		msg, ok := cmd().(appcompose.ServicesLoadedMsg)
		if !ok {
			t.Fatalf("%s: expected a services load, got %T", path, cmd())
		}
		if msg.Gen == 0 {
			t.Fatalf("%s: expected a generation", path)
		}
		if other, dup := seen[msg.Gen]; dup {
			t.Fatalf("%s and %s share generation %d", other, path, msg.Gen)
		}
		seen[msg.Gen] = path
	}

	result, cmd := m.Update(tea.KeyPressMsg{Code: tea.KeyF5})
	m = result.(model)
	record("f5", cmd)

	result, cmd = m.Update(operationSuccessMsg{message: "Stop web/api: 1 targeted, 1 succeeded"})
	m = result.(model)
	record("operation success", cmd)

	result, cmd = m.Update(tea.KeyPressMsg{Code: tea.KeyF5})
	m = result.(model)
	record("f5 again", cmd)

	if len(seen) != 3 {
		t.Fatalf("expected three distinct generations, got %v", seen)
	}
}

// Entering a project clears the view, and the load is several daemon round
// trips behind: until it lands, a key that needs a service must not assert
// that the project has none.
func TestComposeServicesView_KeysDuringTheLoadSayItIsLoading(t *testing.T) {
	dir, file := composeFileFixture(t)
	m := newTestModel()
	m.view = ComposeProjects
	m.composeCLI = &stubComposeEngine{}
	m.composeProjects.SetSize(120, 40)
	m.composeProjects.SetProjects([]docker.ProjectWithServices{{
		Project: docker.ComposeProject{Name: "web", WorkingDir: dir, ConfigFiles: []string{file}},
	}})

	result, _ := m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	m = result.(model)
	if !m.composeServices.Loading() {
		t.Fatal("precondition: expected the view to be waiting for its resources")
	}

	for _, key := range []tea.KeyPressMsg{{Code: 'l'}, {Code: 's', Mod: tea.ModCtrl}, {Code: tea.KeyEnter}} {
		_, cmd := m.Update(key)
		if cmd == nil {
			t.Fatalf("%s: expected a message", key.String())
		}
		status, ok := cmd().(statusMessageMsg)
		if !ok {
			t.Fatalf("%s: expected a status message, got %T", key.String(), cmd())
		}
		if !strings.Contains(status.text, "loading") {
			t.Fatalf("%s: expected the message to say the project is still loading, got %q",
				key.String(), status.text)
		}
	}

	// Once the resources land, the same keys act.
	result, _ = m.Update(appcompose.ServicesLoadedMsg{
		Services: []docker.ComposeService{{Project: "web", Name: "api"}},
		Project:  "web",
		Gen:      m.composeServicesGen,
	})
	m = result.(model)
	if m.composeServices.Loading() {
		t.Fatal("expected the view to stop reporting a load once the resources arrived")
	}
	if svc := m.composeServices.SelectedService(); svc == nil {
		t.Fatal("expected the first service to be selected")
	}
}

// The Compose Projects view reaches "nothing selected" whenever its list is
// empty, and u, d, c, enter and l all used to return nil there: a documented
// key that does nothing and says nothing reads as broken, the same rule the
// services view follows. The message names why the list is empty, because
// "select a project" is not actionable when there is nothing to select.
func TestComposeProjectsView_KeysWithNoProjectSaySo(t *testing.T) {
	keys := []tea.KeyPressMsg{
		{Code: 'u'}, {Code: 'd'}, {Code: 'c'}, {Code: 'l'}, {Code: tea.KeyEnter},
	}
	for _, key := range keys {
		t.Run(key.String(), func(t *testing.T) {
			m := newTestModel()
			m.view = ComposeProjects
			m.composeCLI = &stubComposeEngine{}
			m.composeProjects.SetSize(120, 40)
			m.composeProjects.SetProjects(nil)
			if m.composeProjects.SelectedProject() != nil {
				t.Fatal("precondition: expected an empty project list")
			}

			_, cmd := m.Update(key)
			if cmd == nil {
				t.Fatalf("expected %s to explain itself with no project, got nil", key.String())
			}
			status, ok := cmd().(statusMessageMsg)
			if !ok {
				t.Fatalf("expected a status message, got %T", cmd())
			}
			if !strings.Contains(status.text, "none is loaded") {
				t.Errorf("expected the message to name the reason, got %q", status.text)
			}
		})
	}
}

// And with a filter hiding every project, the filter is the thing to name.
func TestComposeProjectsView_KeysUnderAFilterNameTheFilter(t *testing.T) {
	m := newTestModel()
	m.view = ComposeProjects
	m.composeCLI = &stubComposeEngine{}
	m.composeProjects.SetSize(120, 40)
	m.composeProjects.SetProjects([]docker.ProjectWithServices{
		{Project: docker.ComposeProject{Name: "web"}},
	})
	m.composeProjects.SetFilter("nothing-matches-this")

	_, cmd := m.Update(tea.KeyPressMsg{Code: 'u'})
	if cmd == nil {
		t.Fatal("expected u to explain itself under a filter that matches nothing")
	}
	status := cmd().(statusMessageMsg)
	if !strings.Contains(status.text, "filter") {
		t.Errorf("expected the filter named, got %q", status.text)
	}
}

// TestComposeDriftMsg_FailureSurfacesAsStatusMessage proves the model
// actually tells the user about a drift-check failure instead of only
// recording it in a map nothing reads. Without this, finding 4 could be
// "fixed" by filling in failures and never wiring it to anything visible.
func TestComposeDriftMsg_FailureSurfacesAsStatusMessage(t *testing.T) {
	m := newTestModel()
	failure := errors.New("compose config --hash failed: web")

	result, cmd := m.Update(composeDriftMsg{
		drift:    map[string]map[string]docker.ServiceSync{},
		failures: map[string]string{"web": failure.Error()},
	})
	_ = result.(model)
	if cmd == nil {
		t.Fatal("expected a command to surface the drift failure")
	}
	status, ok := cmd().(statusMessageMsg)
	if !ok {
		t.Fatalf("expected a statusMessageMsg, got %T", cmd())
	}
	if !strings.Contains(status.text, failure.Error()) {
		t.Fatalf("expected the status message to name the failure, got %q", status.text)
	}
}

// stubContainers is a composeContainerSource over a fixed list, so a drift
// test can hand the check the containers it should compare against.
type stubContainers []*docker.Container

func (c stubContainers) Containers([]docker.ContainerFilter, docker.SortMode) []*docker.Container {
	return c
}

// TestComposeServicesView_SyncUpdatesWhileSittingInTheView is the headline:
// SYNC is per-service drift against the compose file, and the services view
// is where a user watches it change, press u, watch "drift" become "ok".
// Drift used to be dispatched only from the projects-load path, so the
// column showed whatever was true when the project list was last loaded, for
// as long as the user stayed in the view.
func TestComposeServicesView_SyncUpdatesWhileSittingInTheView(t *testing.T) {
	dir, file := composeFileFixture(t)
	engine := &stubComposeEngine{hashes: map[string]string{"api": "new-hash"}}
	m := newTestModel()
	m.view = ComposeServices
	m.composeCLI = engine
	m.selectedProject = "web"
	m.composeProjects.SetProjects([]docker.ProjectWithServices{{
		Project: docker.ComposeProject{Name: "web", WorkingDir: dir, ConfigFiles: []string{file}},
	}})
	m.composeServices.SetSize(120, 40)

	// The containers were created from the old file: drifted.
	m.composeDrift = newComposeDriftState(map[string]map[string]docker.ServiceSync{
		"web": {"api": docker.ServiceDrifted},
	})
	m.composeServices.SetServices([]docker.ComposeService{{Project: "web", Name: "api"}}, nil, nil, "web")
	m.composeServices.SetDrift(m.composeDrift.sync)
	if view := ansi.Strip(m.composeServices.View()); !strings.Contains(view, "drift") {
		t.Fatalf("precondition: expected the drifted SYNC cell, got:\n%s", view)
	}

	// u recreates the containers, the container events reload the view.
	result, cmd := m.Update(appcompose.ServicesLoadedMsg{
		Services: []docker.ComposeService{{Project: "web", Name: "api"}},
		Project:  "web",
	})
	m = result.(model)
	if cmd == nil {
		t.Fatal("expected the reload to recompute this project's drift")
	}
	driftMsg, ok := cmd().(composeDriftMsg)
	if !ok {
		t.Fatalf("expected a composeDriftMsg, got %T", cmd())
	}
	if driftMsg.project != "web" {
		t.Fatalf("expected the message to name the project it covers, got %q", driftMsg.project)
	}
	if len(engine.hashesCalls) != 1 {
		t.Fatalf("expected one ConfigHashes call, got %d", len(engine.hashesCalls))
	}

	result, _ = m.Update(driftMsg)
	m = result.(model)
	view := ansi.Strip(m.composeServices.View())
	if strings.Contains(view, "drift") {
		t.Fatalf("expected the stale drift to be gone, got:\n%s", view)
	}
	// The mock daemon has no containers for this project, so compose's
	// hashes describe services that are not created, which is what the
	// column should now say.
	if !strings.Contains(view, "absent") {
		t.Fatalf("expected the recomputed SYNC, got:\n%s", view)
	}
}

// TestComposeServicesView_SyncReadsOkOnceTheContainersMatch is the other
// half of the story the previous test tells: with a container whose
// config-hash matches the file, the recomputed column reads "ok". Together
// they cover "press u, watch drift become ok", a recompute that returned
// nonsense would pass the first test and fail this one.
func TestComposeServicesView_SyncReadsOkOnceTheContainersMatch(t *testing.T) {
	dir, file := composeFileFixture(t)
	engine := &stubComposeEngine{hashes: map[string]string{"api": "matching-hash"}}
	m := newTestModel()
	m.view = ComposeServices
	m.composeCLI = engine
	m.selectedProject = "web"
	m.composeServices.SetSize(120, 40)
	m.composeServices.SetServices([]docker.ComposeService{{Project: "web", Name: "api"}}, nil, nil, "web")

	project := docker.ComposeProject{Name: "web", WorkingDir: dir, ConfigFiles: []string{file}}
	containers := []*docker.Container{{Summary: container.Summary{
		ID:    "abc123",
		Names: []string{"/web-api-1"},
		Labels: map[string]string{
			"com.docker.compose.project":     "web",
			"com.docker.compose.service":     "api",
			"com.docker.compose.config-hash": "matching-hash",
		},
		Status: "Up 2 minutes",
	}}}

	msg := composeServiceDriftCmd(engine, project, stubContainers(containers), 1)()
	drift, ok := msg.(composeDriftMsg)
	if !ok {
		t.Fatalf("expected a composeDriftMsg, got %T", msg)
	}
	if got := drift.drift["web"]["api"]; got != docker.ServiceInSync {
		t.Fatalf("expected the service to read as in sync, got %q", got)
	}

	result, _ := m.Update(drift)
	m = result.(model)
	if view := ansi.Strip(m.composeServices.View()); !strings.Contains(view, "ok") {
		t.Fatalf("expected the SYNC column to read ok, got:\n%s", view)
	}
}

// A single-project recompute must not blank the SYNC the projects view knows
// for every other project.
func TestComposeServicesDrift_KeepsOtherProjectsSync(t *testing.T) {
	m := newTestModel()
	m.composeDrift = newComposeDriftState(map[string]map[string]docker.ServiceSync{
		"web":   {"api": docker.ServiceDrifted},
		"other": {"db": docker.ServiceInSync},
	})

	result, _ := m.Update(composeDriftMsg{
		project: "web",
		drift: map[string]map[string]docker.ServiceSync{
			"web": {"api": docker.ServiceInSync},
		},
	})
	m = result.(model)

	if got := m.composeDrift.sync["web"]["api"]; got != docker.ServiceInSync {
		t.Fatalf("expected web's drift to be replaced, got %q", got)
	}
	if got := m.composeDrift.sync["other"]["db"]; got != docker.ServiceInSync {
		t.Fatalf("expected the other project's drift to survive, got %q", got)
	}
}

// A whole cycle replaces the map, so a project that has disappeared from the
// list takes its SYNC with it rather than lingering forever.
func TestComposeDriftMsg_WholeCycleReplacesTheMap(t *testing.T) {
	m := newTestModel()
	m.composeDrift = newComposeDriftState(map[string]map[string]docker.ServiceSync{
		"gone": {"api": docker.ServiceDrifted},
	})

	result, _ := m.Update(composeDriftMsg{
		drift: map[string]map[string]docker.ServiceSync{
			"web": {"api": docker.ServiceInSync},
		},
	})
	m = result.(model)

	if _, ok := m.composeDrift.sync["gone"]; ok {
		t.Fatal("expected a whole cycle to replace the map")
	}
	if got := m.composeDrift.sync["web"]["api"]; got != docker.ServiceInSync {
		t.Fatalf("expected the cycle's result, got %q", got)
	}
}

// A project whose files stopped being usable here loses its SYNC rather than
// keeping a stale value: an empty column means "not checked", which is the
// truth.
func TestComposeServicesDrift_UnusableFilesClearTheStaleSync(t *testing.T) {
	engine := &stubComposeEngine{hashes: map[string]string{"api": "aaa"}}
	m := newTestModel()
	m.composeDrift = newComposeDriftState(map[string]map[string]docker.ServiceSync{
		"web": {"api": docker.ServiceInSync},
	})

	// A recorded path that is not on this host, the remote-daemon and
	// moved-file case, rather than a project with no recorded path at all,
	// which the dispatcher never asks about.
	msg := composeServiceDriftCmd(engine, docker.ComposeProject{
		Name:        "web",
		WorkingDir:  "/srv/web",
		ConfigFiles: []string{filepath.Join(t.TempDir(), "gone.yaml")},
	}, stubContainers(nil), 1)()
	drift, ok := msg.(composeDriftMsg)
	if !ok {
		t.Fatalf("expected a composeDriftMsg, got %T", msg)
	}
	if len(engine.hashesCalls) != 0 {
		t.Fatalf("expected no compose call for a project with no usable files, got %+v", engine.hashesCalls)
	}
	result, _ := m.Update(drift)
	m = result.(model)
	if _, ok := m.composeDrift.sync["web"]; ok {
		t.Fatalf("expected the stale entry to be dropped, got %v", m.composeDrift.sync)
	}
}

// A cycle already in flight covers this project too, and piling a second
// batch of compose subprocesses on the first is what composeChecks and
// composeCycleRunning exist to prevent. The skipped refresh is recorded,
// not dropped, so the drift that ends the cycle drains it.
func TestComposeServicesLoaded_SkipsDriftWhileACycleIsInFlight(t *testing.T) {
	engine := &stubComposeEngine{hashes: map[string]string{"api": "aaa"}}
	dir, file := composeFileFixture(t)
	m := newTestModel()
	m.view = ComposeServices
	m.composeCLI = engine
	m.selectedProject = "web"
	m.startComposeDrift()
	m.composeProjects.SetProjects([]docker.ProjectWithServices{{
		Project: docker.ComposeProject{Name: "web", WorkingDir: dir, ConfigFiles: []string{file}},
	}})

	result, cmd := m.Update(appcompose.ServicesLoadedMsg{
		Services: []docker.ComposeService{{Project: "web", Name: "api"}},
		Project:  "web",
	})
	m = result.(model)
	if cmd != nil {
		t.Fatalf("expected no second compose batch while a cycle runs, got %T", cmd())
	}
	if len(engine.hashesCalls) != 0 {
		t.Fatalf("expected no compose call, got %+v", engine.hashesCalls)
	}
	if !m.composeRefreshPending {
		t.Fatal("expected the skipped refresh to be recorded")
	}

	// The cycle ends: the deferred reload runs, and it is a services reload
	// because that is the view the user is in.
	result, cmd = m.Update(composeDriftMsg{gen: m.composeDriftGen})
	m = result.(model)
	if cmd == nil {
		t.Fatal("expected the deferred reload to be drained")
	}
	if _, ok := cmd().(appcompose.ServicesLoadedMsg); !ok {
		t.Fatalf("expected a services reload, got %T", cmd())
	}
}

// A project the loaded list does not know, the services view reached before
// its parent list finished loading, must not have its SYNC recomputed:
// nothing here knows its compose files, so the check could only come back
// empty, and an empty per-project result clears the column.
func TestComposeServicesLoaded_UnknownProjectKeepsItsSync(t *testing.T) {
	engine := &stubComposeEngine{hashes: map[string]string{"api": "aaa"}}
	m := newTestModel()
	m.view = ComposeServices
	m.composeCLI = engine
	m.selectedProject = "web"
	m.composeDrift = newComposeDriftState(map[string]map[string]docker.ServiceSync{
		"web": {"api": docker.ServiceInSync},
	})

	result, cmd := m.Update(appcompose.ServicesLoadedMsg{
		Services: []docker.ComposeService{{Project: "web", Name: "api"}},
		Project:  "web",
	})
	m = result.(model)
	// Nothing here knows this project's files, and there is no list at all,
	// so instead of a check that could only come back empty it asks for
	// one, this view is reachable from the command palette without the
	// projects view ever having loaded it, and without the list SYNC would
	// never be computed.
	if cmd == nil {
		t.Fatal("expected the project list to be loaded")
	}
	if _, ok := cmd().(appcompose.ProjectsLoadedMsg); !ok {
		t.Fatalf("expected a projects load, got %T", cmd())
	}
	if len(engine.hashesCalls) != 0 {
		t.Fatalf("expected no compose call, got %+v", engine.hashesCalls)
	}
	if got := m.composeDrift.sync["web"]["api"]; got != docker.ServiceInSync {
		t.Fatalf("expected the known SYNC to survive, got %q", got)
	}
	if m.composeCycleRunning() {
		t.Fatal("expected no cycle to be marked in flight")
	}
}

// A project reload arriving while the services view's own check is in
// flight defers its cycle rather than starting one beside it: the cycle is
// the expensive half, a scan plus one subprocess per project, and two
// batches of compose subprocesses on top of each other is what the guard
// exists to prevent. The deferred reload is drained when the check lands.
func TestComposeDrift_AProjectReloadWaitsForACheckInFlight(t *testing.T) {
	dir, file := composeFileFixture(t)
	engine := &stubComposeEngine{hashes: map[string]string{"api": "aaa"}}
	m := newTestModel()
	m.view = ComposeServices
	m.composeCLI = engine
	m.selectedProject = "web"
	projects := []docker.ProjectWithServices{{
		Project: docker.ComposeProject{Name: "web", WorkingDir: dir, ConfigFiles: []string{file}},
	}}
	m.composeProjects.SetProjects(projects)

	// The services view starts a per-project check.
	result, cmd := m.Update(appcompose.ServicesLoadedMsg{
		Services: []docker.ComposeService{{Project: "web", Name: "api"}},
		Project:  "web",
	})
	m = result.(model)
	if cmd == nil {
		t.Fatal("precondition: expected a per-project drift check")
	}
	perProject := cmd().(composeDriftMsg)

	// The user presses esc and the project list lands while the check is
	// still in flight: no second cycle.
	m.view = ComposeProjects
	result, cmd = m.Update(appcompose.ProjectsLoadedMsg{Projects: projects})
	m = result.(model)
	if cmd != nil {
		t.Fatalf("expected no cycle beside the check in flight, got %T", cmd())
	}
	if len(m.composeChecks) != 1 {
		t.Fatalf("expected one check in flight, got %d", len(m.composeChecks))
	}
	if !m.composeRefreshPending {
		t.Fatal("expected the skipped cycle to be recorded")
	}

	// The check lands: the guard opens and the deferred reload goes out.
	result, cmd = m.Update(perProject)
	m = result.(model)
	if m.composeCycleRunning() {
		t.Fatalf("expected the guard to open, got %d in flight", len(m.composeChecks))
	}
	if cmd == nil {
		t.Fatal("expected the deferred reload to be drained")
	}
	if _, ok := cmd().(appcompose.ProjectsLoadedMsg); !ok {
		t.Fatalf("expected a projects reload, got %T", cmd())
	}
}

// A check whose message never arrives, a compose subprocess that never
// returns, must not keep the guard closed for the rest of the session,
// which is what a count with no way back down would do: every compose
// refresh, in both views, is gated on it.
func TestComposeDrift_AStaleCheckStopsClosingTheGuard(t *testing.T) {
	m := newTestModel()
	m.startComposeDrift()
	if !m.composeCycleRunning() {
		t.Fatal("precondition: expected the guard closed while a check is in flight")
	}

	// Absolute ages, not multiples of composeCycleStale: aged relative to
	// the constant, the assertions move with it and the value is pinned by
	// nothing. Three minutes must be stale and thirty seconds must not,
	// which brackets the two-minute bound from both sides.
	for gen := range m.composeChecks {
		m.composeChecks[gen] = time.Now().Add(-3 * time.Minute)
	}
	if m.composeCycleRunning() {
		t.Fatal("expected a check three minutes old to stop closing the guard")
	}
	// A fresh entry, because composeCycleRunning prunes what it ages out.
	m.composeChecks[99] = time.Now().Add(-30 * time.Second)
	if !m.composeCycleRunning() {
		t.Fatal("expected a check thirty seconds old to keep closing the guard")
	}

	// A replacement closes it again, so the opening is one dispatch wide
	// rather than a permanent free-for-all.
	m.startComposeDrift()
	if !m.composeCycleRunning() {
		t.Fatal("expected the replacement check to close the guard again")
	}
}

// A check dispatched before another one must not overwrite its result, in
// either direction: the per-project check is one subprocess against a
// cycle's one-plus-one-per-project, so the older one landing last is the
// likely ordering, and by arrival order the user would watch SYNC go to ok
// and then flip back to drift.
func TestComposeDrift_AnOlderCheckDoesNotOverwriteANewerOne(t *testing.T) {
	m := newTestModel()

	stale := composeDriftMsg{
		project: "web",
		gen:     1,
		drift:   map[string]map[string]docker.ServiceSync{"web": {"api": docker.ServiceDrifted}},
	}
	fresh := composeDriftMsg{
		gen:   2,
		drift: map[string]map[string]docker.ServiceSync{"web": {"api": docker.ServiceInSync}},
	}

	result, _ := m.Update(fresh)
	m = result.(model)
	result, _ = m.Update(stale)
	m = result.(model)
	if got := m.composeDrift.sync["web"]["api"]; got != docker.ServiceInSync {
		t.Fatalf("expected the newer result to survive the older one, got %q", got)
	}

	// The same the other way round: a newer per-project result survives an
	// older whole cycle landing after it.
	m2 := newTestModel()
	newer := composeDriftMsg{
		project: "web",
		gen:     4,
		drift:   map[string]map[string]docker.ServiceSync{"web": {"api": docker.ServiceInSync}},
	}
	older := composeDriftMsg{
		gen:   3,
		drift: map[string]map[string]docker.ServiceSync{"web": {"api": docker.ServiceDrifted}},
	}
	result, _ = m2.Update(newer)
	m2 = result.(model)
	result, _ = m2.Update(older)
	m2 = result.(model)
	if got := m2.composeDrift.sync["web"]["api"]; got != docker.ServiceInSync {
		t.Fatalf("expected the newer per-project result to survive, got %q", got)
	}
}

// A check that failed is not a report that there is no drift: the last known
// SYNC stays on screen. A compose file being edited fails every check until
// it parses again, and blanking the column each time hides what was true a
// moment ago.
func TestComposeDrift_AFailedCheckKeepsTheLastKnownSync(t *testing.T) {
	m := newTestModel()
	result, _ := m.Update(composeDriftMsg{
		gen:   1,
		drift: map[string]map[string]docker.ServiceSync{"web": {"api": docker.ServiceInSync}},
	})
	m = result.(model)

	result, cmd := m.Update(composeDriftMsg{
		project:  "web",
		gen:      2,
		failures: map[string]string{"web": "yaml: line 3: mapping values are not allowed"},
	})
	m = result.(model)
	if got := m.composeDrift.sync["web"]["api"]; got != docker.ServiceInSync {
		t.Fatalf("expected the last known SYNC to survive a failed check, got %q", got)
	}
	if cmd == nil {
		t.Fatal("expected the failure to be reported")
	}
}

// The same failure repeating must not re-report itself. A drift check runs
// on every reload, and container events keep the view reloading, so an
// event-per-cycle banner would pin the message bar for as long as the
// compose file stays broken.
func TestComposeDrift_ARepeatedFailureIsReportedOnce(t *testing.T) {
	m := newTestModel()
	failure := composeDriftMsg{
		project:  "web",
		gen:      1,
		failures: map[string]string{"web": "yaml: line 3: mapping values are not allowed"},
	}

	result, cmd := m.Update(failure)
	m = result.(model)
	if cmd == nil {
		t.Fatal("expected the first failure to be reported")
	}
	if _, ok := cmd().(statusMessageMsg); !ok {
		t.Fatalf("expected a status message, got %T", cmd())
	}

	failure.gen = 2
	result, cmd = m.Update(failure)
	m = result.(model)
	if cmd != nil {
		if msg := cmd(); msg != nil {
			if _, ok := msg.(statusMessageMsg); ok {
				t.Fatal("expected the same failure not to be reported twice")
			}
		}
	}

	// A clean cycle clears it, so the next occurrence is reported again.
	result, _ = m.Update(composeDriftMsg{gen: 3})
	m = result.(model)
	failure.gen = 4
	_, cmd = m.Update(failure)
	if cmd == nil {
		t.Fatal("expected a recurrence after a clean cycle to be reported")
	}
	if _, ok := cmd().(statusMessageMsg); !ok {
		t.Fatalf("expected a status message, got %T", cmd())
	}
}

// A load that finishes after the user has pressed esc must not spend a
// compose subprocess on a view nobody is looking at, and, worse, count as
// a cycle in flight, which defers the projects view's own reload.
func TestComposeServicesLoaded_NoDriftForAViewTheUserLeft(t *testing.T) {
	dir, file := composeFileFixture(t)
	engine := &stubComposeEngine{hashes: map[string]string{"api": "aaa"}}
	m := newTestModel()
	m.view = ComposeProjects // the user has already pressed esc
	m.composeCLI = engine
	m.selectedProject = "web"
	m.composeProjects.SetProjects([]docker.ProjectWithServices{{
		Project: docker.ComposeProject{Name: "web", WorkingDir: dir, ConfigFiles: []string{file}},
	}})

	result, cmd := m.Update(appcompose.ServicesLoadedMsg{
		Services: []docker.ComposeService{{Project: "web", Name: "api"}},
		Project:  "web",
	})
	m = result.(model)
	if cmd != nil {
		t.Fatalf("expected no drift check for a view the user left, got %T", cmd())
	}
	if len(engine.hashesCalls) != 0 {
		t.Fatalf("expected no compose call, got %+v", engine.hashesCalls)
	}
	if m.composeCycleRunning() {
		t.Fatal("expected no cycle to be counted in flight")
	}
}

// A drift result the state rejects as superseded must not raise a banner
// either. Two checks in flight means the older one can land last, and by
// message rather than by state the user would be told about a failure that
// a newer, successful check has already answered.
func TestComposeDrift_ASupersededFailureIsNotReported(t *testing.T) {
	m := newTestModel()

	// A newer cycle succeeded for web.
	result, _ := m.Update(composeDriftMsg{
		gen:   2,
		drift: map[string]map[string]docker.ServiceSync{"web": {"api": docker.ServiceInSync}},
	})
	m = result.(model)

	// The older per-project check for web lands afterwards, having failed.
	result, cmd := m.Update(composeDriftMsg{
		project:  "web",
		gen:      1,
		failures: map[string]string{"web": "yaml: line 3: mapping values are not allowed"},
	})
	m = result.(model)

	if got := m.composeDrift.sync["web"]["api"]; got != docker.ServiceInSync {
		t.Fatalf("expected the newer result to stand, got %q", got)
	}
	if cmd != nil {
		if msg := cmd(); msg != nil {
			if status, ok := msg.(statusMessageMsg); ok {
				t.Fatalf("expected no banner for a superseded failure, got %q", status.text)
			}
		}
	}
	if _, failed := m.composeDrift.fail["web"]; failed {
		t.Fatal("expected no failure recorded for a superseded check")
	}
}

// One project's success must not erase the record of another's failure, or
// ordinary navigation re-reports the same unchanged broken file on every
// trip in and out of a project.
func TestComposeDrift_AnotherProjectsSuccessKeepsTheFailureQuiet(t *testing.T) {
	m := newTestModel()
	broken := composeDriftMsg{
		gen:      1,
		drift:    map[string]map[string]docker.ServiceSync{"other": {"db": docker.ServiceInSync}},
		failures: map[string]string{"web": "yaml: line 3: mapping values are not allowed"},
	}
	result, cmd := m.Update(broken)
	m = result.(model)
	if cmd == nil {
		t.Fatal("expected the first failure to be reported")
	}

	// The services view recomputes the healthy project, twice.
	for i := range 2 {
		result, cmd = m.Update(composeDriftMsg{
			project: "other",
			gen:     uint64(2 + i),
			drift:   map[string]map[string]docker.ServiceSync{"other": {"db": docker.ServiceInSync}},
		})
		m = result.(model)
		if cmd != nil {
			if msg := cmd(); msg != nil {
				if status, ok := msg.(statusMessageMsg); ok {
					t.Fatalf("recompute %d re-reported another project's failure: %q", i+1, status.text)
				}
			}
		}
	}
	if _, failed := m.composeDrift.fail["web"]; !failed {
		t.Fatal("expected web's failure to still be on record")
	}
}

// A whole cycle that drops a project must not have it written back by an
// older cycle landing afterwards: the dropped project has no per-project
// generation left, so the state keeps the newest cycle's generation as the
// floor under all of them.
func TestComposeDrift_AnOlderCycleCannotResurrectADroppedProject(t *testing.T) {
	m := newTestModel()
	result, _ := m.Update(composeDriftMsg{
		gen:   1,
		drift: map[string]map[string]docker.ServiceSync{"gone": {"api": docker.ServiceDrifted}},
	})
	m = result.(model)

	// A newer cycle no longer reports that project at all.
	result, _ = m.Update(composeDriftMsg{
		gen:   3,
		drift: map[string]map[string]docker.ServiceSync{"web": {"api": docker.ServiceInSync}},
	})
	m = result.(model)
	if _, ok := m.composeDrift.sync["gone"]; ok {
		t.Fatal("precondition: expected the newer cycle to drop the project")
	}

	// The older cycle lands late.
	result, _ = m.Update(composeDriftMsg{
		gen:   2,
		drift: map[string]map[string]docker.ServiceSync{"gone": {"api": docker.ServiceDrifted}},
	})
	m = result.(model)
	if _, ok := m.composeDrift.sync["gone"]; ok {
		t.Fatalf("expected the dropped project to stay dropped, got %v", m.composeDrift.sync)
	}
}

// The drift check reads the container list when it runs, not when it is
// dispatched: the daemon refreshes its own store on its own goroutine when
// an event arrives, and a snapshot taken at dispatch can still hold the
// pre-recreate labels, which reads as drift for containers that already
// match.
func TestComposeDriftCmd_ReadsTheContainerListWhenItRuns(t *testing.T) {
	dir, file := composeFileFixture(t)
	engine := &stubComposeEngine{hashes: map[string]string{"api": "matching-hash"}}
	source := &countingContainers{}
	cmd := composeDriftCmd(engine, []docker.ProjectWithServices{{
		Project: docker.ComposeProject{Name: "web", WorkingDir: dir, ConfigFiles: []string{file}},
	}}, source, 1)

	if source.calls != 0 {
		t.Fatalf("expected the container list to be read when the check runs, not at dispatch (%d)", source.calls)
	}
	msg := cmd()
	if source.calls != 1 {
		t.Fatalf("expected one read, got %d", source.calls)
	}
	drift := msg.(composeDriftMsg)
	if got := drift.drift["web"]["api"]; got != docker.ServiceInSync {
		t.Fatalf("expected the containers the check read to decide, got %q", got)
	}
}

// countingContainers reports one container matching the stub engine's hash,
// and counts how often it is asked.
type countingContainers struct{ calls int }

func (c *countingContainers) Containers([]docker.ContainerFilter, docker.SortMode) []*docker.Container {
	c.calls++
	return []*docker.Container{{Summary: container.Summary{
		ID:    "abc123",
		Names: []string{"/web-api-1"},
		Labels: map[string]string{
			"com.docker.compose.project":     "web",
			"com.docker.compose.service":     "api",
			"com.docker.compose.config-hash": "matching-hash",
		},
		Status: "Up 2 minutes",
	}}}
}

// A check that has aged out and lands afterwards must not open the guard for
// the check that replaced it. Tracking each check by its generation is what
// makes that impossible; a bare count would decrement the replacement's slot.
func TestComposeDrift_AStaleCheckLandingDoesNotUnlockItsReplacement(t *testing.T) {
	m := newTestModel()
	stale := m.startComposeDrift()
	m.composeChecks[stale] = time.Now().Add(-2 * composeCycleStale)
	if m.composeCycleRunning() {
		t.Fatal("precondition: expected the aged-out check to stop closing the guard")
	}

	replacement := m.startComposeDrift()
	if !m.composeCycleRunning() {
		t.Fatal("precondition: expected the replacement to close the guard")
	}

	result, _ := m.Update(composeDriftMsg{gen: stale})
	m = result.(model)
	if !m.composeCycleRunning() {
		t.Fatal("expected the guard to stay closed while the replacement runs")
	}

	result, _ = m.Update(composeDriftMsg{gen: replacement})
	m = result.(model)
	if m.composeCycleRunning() {
		t.Fatalf("expected the guard to open once the replacement landed, got %d checks", len(m.composeChecks))
	}
}

// Two per-project checks for the same project can cross, the first one can
// age out and be replaced, and the older result must not overwrite the
// newer one, which is the flip-back-to-drift the whole design exists to
// prevent.
func TestComposeDrift_AnOlderPerProjectCheckLosesToANewerOne(t *testing.T) {
	m := newTestModel()
	result, _ := m.Update(composeDriftMsg{
		project: "web",
		gen:     2,
		drift:   map[string]map[string]docker.ServiceSync{"web": {"api": docker.ServiceInSync}},
	})
	m = result.(model)

	result, _ = m.Update(composeDriftMsg{
		project: "web",
		gen:     1,
		drift:   map[string]map[string]docker.ServiceSync{"web": {"api": docker.ServiceDrifted}},
	})
	m = result.(model)

	if got := m.composeDrift.sync["web"]["api"]; got != docker.ServiceInSync {
		t.Fatalf("expected the newer per-project result to stand, got %q", got)
	}
}

// The scan sits between a project load and its drift check, so it has to
// carry the cycle's generation across. A cycle stamped zero is discarded by
// the floor the moment any other cycle has run, which would freeze SYNC with
// no error anywhere.
func TestComposeProjectsMsg_CarriesTheCycleGeneration(t *testing.T) {
	dir, file := composeFileFixture(t)
	engine := &stubComposeEngine{resolveName: "web", hashes: map[string]string{"api": "aaa"}}
	m := newTestModel()
	m.view = ComposeProjects
	m.composeCLI = engine
	m.workingDir = dir
	projects := []docker.ProjectWithServices{{
		Project: docker.ComposeProject{Name: "web", WorkingDir: dir, ConfigFiles: []string{file}},
	}}

	result, cmd := m.Update(appcompose.ProjectsLoadedMsg{Projects: projects})
	m = result.(model)
	if cmd == nil {
		t.Fatal("expected the scan to be dispatched")
	}
	scanned, ok := cmd().(composeProjectsMsg)
	if !ok {
		t.Fatalf("expected a composeProjectsMsg, got %T", cmd())
	}
	if scanned.gen == 0 {
		t.Fatal("expected the scan to carry the cycle's generation")
	}
	if scanned.gen != m.composeDriftGen {
		t.Fatalf("expected the cycle's own generation %d, got %d", m.composeDriftGen, scanned.gen)
	}

	// And the drift it dispatches next carries it too, so the result is
	// applied rather than discarded as older than the floor.
	result, cmd = m.Update(scanned)
	m = result.(model)
	if cmd == nil {
		t.Fatal("expected drift to be dispatched after the scan")
	}
	drift := cmd().(composeDriftMsg)
	if drift.gen != scanned.gen {
		t.Fatalf("expected the drift check to keep the generation, got %d", drift.gen)
	}
	result, _ = m.Update(drift)
	m = result.(model)
	if _, ok := m.composeDrift.sync["web"]; !ok {
		t.Fatalf("expected the cycle's result to be applied, got %v", m.composeDrift.sync)
	}
}

// The cycle is the expensive half, so it does not run for a view that is not
// on screen: a project list can land after the user has switched away.
func TestComposeProjectsLoaded_NoCycleForAViewTheUserLeft(t *testing.T) {
	dir, file := composeFileFixture(t)
	engine := &stubComposeEngine{hashes: map[string]string{"api": "aaa"}}
	m := newTestModel()
	m.view = Main
	m.composeCLI = engine
	projects := []docker.ProjectWithServices{{
		Project: docker.ComposeProject{Name: "web", WorkingDir: dir, ConfigFiles: []string{file}},
	}}

	result, cmd := m.Update(appcompose.ProjectsLoadedMsg{Projects: projects})
	m = result.(model)
	if cmd != nil {
		t.Fatalf("expected no cycle for a view the user is not in, got %T", cmd())
	}
	if m.composeCycleRunning() {
		t.Fatal("expected no check to be counted in flight")
	}
	if !m.composeRefreshPending {
		t.Fatal("expected the skipped cycle to be recorded")
	}
	// The rows still land, so the palette and the workspace context have
	// them; only the drift check waits.
	if m.composeProjects.ProjectByName("web") == nil {
		t.Fatal("expected the project list to be applied")
	}
}

// The per-project check reads the container list when it runs, for the same
// reason the cycle does.
func TestComposeServiceDriftCmd_ReadsTheContainerListWhenItRuns(t *testing.T) {
	dir, file := composeFileFixture(t)
	engine := &stubComposeEngine{hashes: map[string]string{"api": "matching-hash"}}
	source := &countingContainers{}
	cmd := composeServiceDriftCmd(engine,
		docker.ComposeProject{Name: "web", WorkingDir: dir, ConfigFiles: []string{file}}, source, 1)

	if source.calls != 0 {
		t.Fatalf("expected no read at dispatch, got %d", source.calls)
	}
	drift := cmd().(composeDriftMsg)
	if source.calls != 1 {
		t.Fatalf("expected one read when the check ran, got %d", source.calls)
	}
	if got := drift.drift["web"]["api"]; got != docker.ServiceInSync {
		t.Fatalf("expected the containers the check read to decide, got %q", got)
	}
}

// Nothing repaints behind a streaming viewer, the per-project check
// included: the view it would refresh is covered, and an `up` is exactly
// when container events arrive fastest.
func TestComposeServicesLoaded_NoDriftBehindAStreamingViewer(t *testing.T) {
	dir, file := composeFileFixture(t)
	engine := &stubComposeEngine{hashes: map[string]string{"api": "aaa"}}
	m := newTestModel()
	m.view = ComposeServices
	m.composeCLI = engine
	m.selectedProject = "web"
	m.composeProjects.SetProjects([]docker.ProjectWithServices{{
		Project: docker.ComposeProject{Name: "web", WorkingDir: dir, ConfigFiles: []string{file}},
	}})

	result, _ := m.Update(showStreamingLessMsg{
		title:  "Compose up: web",
		reader: io.NopCloser(strings.NewReader("Container web-api-1  Started\n")),
	})
	m = result.(model)
	if !m.streamingViewerOpen() {
		t.Fatal("precondition: expected the streaming viewer to be open")
	}

	result, cmd := m.Update(appcompose.ServicesLoadedMsg{
		Services: []docker.ComposeService{{Project: "web", Name: "api"}},
		Project:  "web",
	})
	m = result.(model)
	if cmd != nil {
		if _, isDrift := cmd().(composeDriftMsg); isDrift {
			t.Fatal("expected no drift check behind a streaming viewer")
		}
	}
	if len(engine.hashesCalls) != 0 {
		t.Fatalf("expected no compose call, got %+v", engine.hashesCalls)
	}
	if !m.composeRefreshPending {
		t.Fatal("expected the skipped check to be recorded")
	}
}

// A list that has loaded and does not contain this project is a project
// whose containers are gone. Asking for the list again would not find it,
// and this runs on every reload, so it would spend a drift cycle per burst
// of container events on a project that no longer exists.
func TestComposeServicesLoaded_AMissingProjectDoesNotReloadTheList(t *testing.T) {
	dir, file := composeFileFixture(t)
	engine := &stubComposeEngine{hashes: map[string]string{"api": "aaa"}}
	m := newTestModel()
	m.view = ComposeServices
	m.composeCLI = engine
	m.selectedProject = "gone"
	// The list has loaded; it just does not have this project.
	m.composeProjects.SetProjects([]docker.ProjectWithServices{{
		Project: docker.ComposeProject{Name: "web", WorkingDir: dir, ConfigFiles: []string{file}},
	}})
	m.composeDrift = newComposeDriftState(map[string]map[string]docker.ServiceSync{
		"gone": {"api": docker.ServiceInSync},
	})

	result, cmd := m.Update(appcompose.ServicesLoadedMsg{
		Services: []docker.ComposeService{{Project: "gone", Name: "api"}},
		Project:  "gone",
	})
	m = result.(model)
	if cmd != nil {
		t.Fatalf("expected nothing to be dispatched for a project that is gone, got %T", cmd())
	}
	if len(engine.hashesCalls) != 0 {
		t.Fatalf("expected no compose call, got %+v", engine.hashesCalls)
	}
	if got := m.composeDrift.sync["gone"]["api"]; got != docker.ServiceInSync {
		t.Fatalf("expected the last known SYNC to survive, got %q", got)
	}
}

// The banner's length must not grow with the number of failures, with how
// much compose had to say, or with how long a project's name is: the message
// bar is one line in the layout's height budget, and a banner that grows
// without limit takes the footer's row.
func TestNewDriftFailures_HasACeilingRegardlessOfHowMuchFailed(t *testing.T) {
	big := map[string]string{}
	for _, name := range []string{"alpha", "beta", "gamma", "delta", "epsilon", "zeta", "eta", "theta", "iota", "kappa"} {
		big[strings.Repeat(name, 12)] = strings.Repeat("compose said something very long ", 6)
	}

	msg := newDriftFailures(nil, big)

	if !strings.Contains(msg, "(+8 more)") {
		t.Errorf("expected the count of the projects it did not name, got %q", msg)
	}
	// Cells, not bytes: the clip is by cell and the bar is one terminal
	// line, and a CJK project name costs three bytes a cell, so a byte
	// bound both mismeasures and fails on input the sibling subtest below
	// says is real. An absolute number, deliberately not derived from the
	// constants it guards, which would move with them and assert nothing:
	// two entries of a 30-cell name and a 60-cell reason, plus the prefix
	// and the "(+N more)" suffix: 28 + 2*(30 + 2 + 60) + 10 = 226.
	const ceiling = 230
	if got := ansi.StringWidth(msg); got > ceiling {
		t.Errorf("banner is %d cells, over the %d-cell ceiling: %q", got, ceiling, msg)
	}
	if strings.Count(msg, ": ") > 3 {
		t.Errorf("expected at most two projects named, got %q", msg)
	}
}

// Two failures are named in a stable order, so the banner does not shuffle
// between ticks while the same two projects keep failing.
func TestNewDriftFailures_NamesTwoFailuresInAStableOrder(t *testing.T) {
	for range 20 {
		msg := newDriftFailures(nil, map[string]string{"zeta": "boom", "alpha": "bang"})
		if msg != "Compose drift check failed: alpha: bang; zeta: boom" {
			t.Fatalf("unstable or unsorted: %q", msg)
		}
	}
}

// compose's stderr is untrusted text on its way to the terminal. Escape
// sequences would be re-interpreted; a carriage return or a backspace would
// make lipgloss render the one-line bar as two lines and push the footer off
// the screen. And the clip is by cells, so a multi-byte rune is never cut in
// half, which a byte slice does.
func TestNewDriftFailures_SanitisesComposeOutput(t *testing.T) {
	cases := []struct {
		name, why string
		wantNo    string
	}{
		// The visible sequence, not the ESC byte: unicode.IsControl removes
		// ESC on its own, so asserting the byte left ansi.Strip unguarded.
		{"escape", "\x1b[31mred\x1b[0m", "[31m"},
		{"carriage return", "boom\rHACKED", "\r"},
		{"backspace", "boom\bx", "\b"},
		{"bell", "boom\a", "\a"},
		{"tab", "boom\tmore", "\t"},
		{"newline", "line one\nline two", "\n"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			msg := newDriftFailures(nil, map[string]string{"web": tc.why})
			if strings.Contains(msg, tc.wantNo) {
				t.Errorf("expected %q to be removed, got %q", tc.wantNo, msg)
			}
			if !strings.Contains(msg, "boom") && !strings.Contains(msg, "red") && !strings.Contains(msg, "line one") {
				t.Errorf("sanitising ate the message: %q", msg)
			}
		})
	}

	// Two of these are separators rather than noise, and the control filter
	// would drop them silently and run the words together.
	t.Run("tabs become a space", func(t *testing.T) {
		msg := newDriftFailures(nil, map[string]string{"web": "boom\tmore"})
		if want := "Compose drift check failed: web: boom more"; msg != want {
			t.Errorf("got %q, want %q", msg, want)
		}
	})

	// A newline is a separator, not noise: the control filter would drop it
	// silently and run the two lines together.
	t.Run("newlines become separators", func(t *testing.T) {
		msg := newDriftFailures(nil, map[string]string{"web": "line one\nline two"})
		if want := "Compose drift check failed: web: line one; line two"; msg != want {
			t.Errorf("got %q, want %q", msg, want)
		}
	})

	t.Run("a wide-character banner stays inside the cell ceiling", func(t *testing.T) {
		wide := map[string]string{}
		for i := range 10 {
			wide[strings.Repeat("プロジェクト", 6)+string(rune('a'+i))] = strings.Repeat("コンポーズが言った", 12)
		}
		msg := newDriftFailures(nil, wide)
		// The same ceiling as the ASCII case, which is the point of
		// measuring cells: 220 here against 224 there, where a byte
		// bound would have read 316 and failed.
		if got := ansi.StringWidth(msg); got > 230 {
			t.Errorf("banner is %d cells: %q", got, msg)
		}
		if !utf8.ValidString(msg) {
			t.Errorf("clip cut a rune in half: %q", msg)
		}
	})

	t.Run("clips on a rune boundary", func(t *testing.T) {
		msg := newDriftFailures(nil, map[string]string{
			"web": strings.Repeat("x", driftFailureReason-1) + "é and more",
		})
		if !utf8.ValidString(msg) {
			t.Errorf("clip cut a rune in half: %q", msg)
		}
	})
}

// One failure is named in full, so the common case reads as a sentence and
// not as a count.
func TestNewDriftFailures_NamesASingleFailureOutright(t *testing.T) {
	msg := newDriftFailures(nil, map[string]string{"web": "no such file"})
	if msg != "Compose drift check failed: web: no such file" {
		t.Errorf("unexpected message: %q", msg)
	}
}

// A failed check has to record its generation, not just its reason. Without
// that, the project's generation stays where the last successful check left
// it, and a slower check that was already in flight when the failure landed
// overwrites the failure with a value from before it: SYNC flips back to the
// state composeDriftState exists to make unreachable, and the failure record
// is erased with it.
func TestComposeDriftState_AFailureIsNotOverwrittenByAnOlderCheck(t *testing.T) {
	state := newComposeDriftState(nil)

	// gen 2 succeeds, gen 5 fails, then gen 3 lands late.
	state = state.merge(composeDriftMsg{
		project: "web", gen: 2,
		drift: map[string]map[string]docker.ServiceSync{"web": {"api": docker.ServiceInSync}},
	})
	state = state.merge(composeDriftMsg{
		project: "web", gen: 5,
		failures: map[string]string{"web": "compose config --hash failed"},
	})
	state = state.merge(composeDriftMsg{
		project: "web", gen: 3,
		drift: map[string]map[string]docker.ServiceSync{"web": {"api": docker.ServiceDrifted}},
	})

	if got := state.sync["web"]["api"]; got != docker.ServiceInSync {
		t.Errorf("an older check overwrote the value the failure preserved: SYNC = %q, want %q",
			got, docker.ServiceInSync)
	}
	if state.fail["web"] == "" {
		t.Error("the failure record was erased by an older check")
	}
}

// 2b: the views have to render the merged state, not the arriving message.
// A per-project result carries that project only, so handing msg.drift
// straight to SetDrift blanks every other project's SYNC, which is the flip
// composeDriftState exists to make unreachable. Asserting on m.composeDrift
// alone cannot see it: the bug is in what reaches the models.
func TestComposeDriftMsg_TheViewsRenderTheMergedState(t *testing.T) {
	m := newTestModel()
	m.view = ComposeProjects
	m.composeProjects.SetSize(160, 40)
	m.composeProjects.SetProjects([]docker.ProjectWithServices{
		{Project: docker.ComposeProject{Name: "web"}, Services: []docker.ComposeService{{Project: "web", Name: "api"}}},
		{Project: docker.ComposeProject{Name: "db"}, Services: []docker.ComposeService{{Project: "db", Name: "pg"}}},
	})

	// A whole cycle establishes both projects.
	result, _ := m.Update(composeDriftMsg{gen: 1, drift: map[string]map[string]docker.ServiceSync{
		"web": {"api": docker.ServiceInSync},
		"db":  {"pg": docker.ServiceInSync},
	}})
	m = result.(model)

	// Then one project's own recompute lands.
	result, _ = m.Update(composeDriftMsg{project: "web", gen: 2, drift: map[string]map[string]docker.ServiceSync{
		"web": {"api": docker.ServiceDrifted},
	}})
	m = result.(model)

	view := ansi.Strip(m.composeProjects.View())
	if !strings.Contains(view, "drift") {
		t.Errorf("expected the recomputed project to show drift:\n%s", view)
	}
	if !strings.Contains(view, "ok") {
		t.Errorf("expected the other project to keep its SYNC:\n%s", view)
	}
}

// 2d: every path out of composeScanCmd has to stamp the generation, not just
// the one that finds a file. A zero generation is never in composeChecks, so
// the delete on arrival is a no-op, the guard stays closed, and every compose
// refresh is gated until the check ages out two minutes later. The two paths
// that return early are the common ones: no resolver, and no compose file in
// the working directory.
func TestComposeScanCmd_EveryPathCarriesTheGeneration(t *testing.T) {
	withFile := t.TempDir()
	if err := os.WriteFile(filepath.Join(withFile, "compose.yaml"),
		[]byte("services:\n  web:\n    image: alpine\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	cases := []struct {
		name     string
		resolver composeResolver
		dir      string
	}{
		{"no resolver", nil, withFile},
		{"no compose file in the directory", &stubComposeEngine{}, t.TempDir()},
		{"the resolve fails", &stubComposeEngine{err: errors.New("boom")}, withFile},
		{"the resolve succeeds", &stubComposeEngine{}, withFile},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			msg, ok := composeScanCmd(tc.resolver, tc.dir, nil, 7)().(composeProjectsMsg)
			if !ok {
				t.Fatalf("expected a composeProjectsMsg")
			}
			if msg.gen != 7 {
				t.Errorf("expected the cycle's generation 7, got %d", msg.gen)
			}
		})
	}
}

// 2e: the drift cycle has to run for the services view too. It is reachable
// from the command palette without the projects view ever loading, so a guard
// that only knows about Compose Projects means SYNC never fills in there.
func TestComposeViewActive_CoversBothComposeViews(t *testing.T) {
	m := newTestModel()
	for view, want := range map[viewMode]bool{
		ComposeProjects: true, ComposeServices: true, Main: false, Images: false,
	} {
		m.view = view
		if got := m.composeViewActive(); got != want {
			t.Errorf("composeViewActive() for view %v = %v, want %v", view, got, want)
		}
	}
}

// 2j: a whole cycle reports on every project, so one it does not mention has
// gone from the list and must lose its entry, the failure record included.
// Without that, a project that failed once and then disappeared keeps its
// fail entry forever, and newDriftFailures compares against a project that
// is no longer there.
func TestComposeDriftState_AWholeCycleForgetsAProjectThatLeft(t *testing.T) {
	state := newComposeDriftState(nil)
	state = state.merge(composeDriftMsg{project: "web", gen: 1,
		failures: map[string]string{"web": "compose config --hash failed"}})
	if state.fail["web"] == "" {
		t.Fatal("precondition: expected the failure recorded")
	}

	// A later cycle covering only another project.
	state = state.merge(composeDriftMsg{gen: 2, drift: map[string]map[string]docker.ServiceSync{
		"db": {"pg": docker.ServiceInSync},
	}})

	if why, ok := state.fail["web"]; ok {
		t.Errorf("expected the departed project's failure forgotten, still have %q", why)
	}
	if _, ok := state.sync["web"]; ok {
		t.Error("expected the departed project's SYNC forgotten")
	}
	if state.sync["db"]["pg"] != docker.ServiceInSync {
		t.Errorf("expected the reported project applied, got %q", state.sync["db"]["pg"])
	}
}

// A project whose only entry is a failure still has to be considered by a
// whole cycle, so an older cycle cannot quietly drop it. The failure came
// from a newer per-project check, which knows more than the cycle does, and
// a project with no SYNC entry is reachable only this way: its first check
// failed, so there was never a value to keep.
func TestComposeDriftState_AnOlderCycleKeepsANewerFailure(t *testing.T) {
	state := newComposeDriftState(nil)
	state = state.merge(composeDriftMsg{project: "web", gen: 5,
		failures: map[string]string{"web": "compose config --hash failed"}})
	if _, ok := state.sync["web"]; ok {
		t.Fatal("precondition: a first-check failure leaves no SYNC to keep")
	}

	// A cycle that started before that check, arriving after it.
	state = state.merge(composeDriftMsg{gen: 3, drift: map[string]map[string]docker.ServiceSync{
		"db": {"pg": docker.ServiceInSync},
	}})

	if state.fail["web"] == "" {
		t.Error("expected the newer failure kept against an older cycle")
	}
	if got := state.gen["web"]; got != 5 {
		t.Errorf("expected the failure to keep its generation 5, got %d", got)
	}
}

// The budget has to stop the walk between calls as well as inside one. Five
// projects whose checks each outlast the whole budget and return anyway, as
// a compose subprocess that ignores cancellation would: the second starts
// only if nothing looks at the clock between them.
func TestComposeDriftCmd_TheBudgetStopsTheWalkBetweenCalls(t *testing.T) {
	restore := composeDriftBudget
	composeDriftBudget = 200 * time.Millisecond
	t.Cleanup(func() { composeDriftBudget = restore })

	engine := &stubComposeEngine{
		hashesDelay: 250 * time.Millisecond,
		hashes:      map[string]string{"api": "aaa"},
	}
	dir, file := composeFileFixture(t)
	var projects []docker.ProjectWithServices
	for _, name := range []string{"one", "two", "three", "four", "five"} {
		projects = append(projects, docker.ProjectWithServices{
			Project: docker.ComposeProject{Name: name, WorkingDir: dir, ConfigFiles: []string{file}},
		})
	}

	msg := composeDriftCmd(engine, projects, stubContainers(nil), 1)().(composeDriftMsg)

	if len(engine.hashesCalls) != 1 {
		t.Errorf("expected the walk to stop after the first call spent the budget, got %d calls",
			len(engine.hashesCalls))
	}
	if len(drift(msg)) == 0 {
		t.Error("expected the projects it did reach to report drift")
	}
	if len(msg.failures) == 0 {
		t.Fatal("expected the unreached projects to be reported")
	}
	for name, why := range msg.failures {
		if !strings.Contains(why, "gave up") {
			t.Errorf("expected %s to say the cycle gave up, got %q", name, why)
		}
	}
}

// drift is msg.drift, named so the assertion above reads as a sentence.
func drift(msg composeDriftMsg) map[string]map[string]docker.ServiceSync { return msg.drift }

// Every container key on a service that has none; see
// composeNoContainersCmd for why they may not prompt.
func TestComposeServicesView_ContainerKeysOnAServiceWithNoContainers(t *testing.T) {
	keys := map[string]struct {
		key    tea.KeyPressMsg
		action string
	}{
		"enter":   {tea.KeyPressMsg{Code: tea.KeyEnter}, "Inspect"},
		"logs":    {tea.KeyPressMsg{Code: 'l'}, "Logs"},
		"start":   {tea.KeyPressMsg{Code: 's', Mod: tea.ModCtrl}, "Start"},
		"stop":    {tea.KeyPressMsg{Code: 't', Mod: tea.ModCtrl}, "Stop"},
		"restart": {tea.KeyPressMsg{Code: 'r', Mod: tea.ModCtrl}, "Restart"},
		"remove":  {tea.KeyPressMsg{Code: 'e', Mod: tea.ModCtrl}, "Remove"},
	}
	for name, tc := range keys {
		key := tc.key
		t.Run(name, func(t *testing.T) {
			m := newTestModel()
			m.view = ComposeServices
			m.composeCLI = &stubComposeEngine{}
			m.selectedProject = "web"
			m.composeServices.SetSize(120, 40)
			m.composeServices.SetServices(nil, nil, nil, "web")
			m.composeServices.SetDrift(map[string]map[string]docker.ServiceSync{
				"web": {"cache": docker.ServiceNotCreated},
			})
			svc := m.composeServices.SelectedService()
			if svc == nil || svc.Containers != 0 {
				t.Fatalf("precondition: expected a service with no containers, got %+v", svc)
			}

			result, cmd := m.Update(key)
			if cmd == nil {
				t.Fatalf("expected %s to explain itself, got nil", name)
			}
			status, ok := cmd().(statusMessageMsg)
			if !ok {
				t.Fatalf("expected a status message, got %T", cmd())
			}
			// The action and the service by name: with only the shared
			// half asserted, every message could read "Logs" and name a
			// service that does not exist and these five would still pass.
			if want := tc.action + " needs a container, and cache has none: u brings it up"; status.text != want {
				t.Errorf("expected %q, got %q", want, status.text)
			}
			// And no confirmation prompt was opened: the point is that the
			// user is not asked to confirm a no-op.
			if got := result.(model); got.overlay == overlayPrompt {
				t.Errorf("expected no prompt for an action that cannot act")
			}
		})
	}
}

// u on the same row is the one key that does apply, and it must still work.
func TestComposeServicesView_UBringsUpAServiceWithNoContainers(t *testing.T) {
	engine := &stubComposeEngine{}
	dir, file := composeFileFixture(t)
	m := newTestModel()
	m.view = ComposeServices
	m.composeCLI = engine
	m.selectedProject = "web"
	// up reads the compose file, so the project needs usable files. The
	// fixture's file has no services of its own; cache reaches the view
	// through the drift map this test injects, as it would from a real
	// check.
	m.composeProjects.SetProjects([]docker.ProjectWithServices{{
		Project: docker.ComposeProject{Name: "web", WorkingDir: dir, ConfigFiles: []string{file}},
	}})
	m.composeServices.SetSize(120, 40)
	m.composeServices.SetServices(nil, nil, nil, "web")
	m.composeServices.SetDrift(map[string]map[string]docker.ServiceSync{
		"web": {"cache": docker.ServiceNotCreated},
	})

	_, cmd := m.Update(tea.KeyPressMsg{Code: 'u'})
	if cmd == nil {
		t.Fatal("expected u to act on the not-created service")
	}
	cmd()
	if len(engine.upCalls) != 1 {
		t.Fatalf("expected one Up call, got %+v", engine.upCalls)
	}
	if len(engine.upServices) != 1 || engine.upServices[0] != "cache" {
		t.Errorf("expected up to target the service alone, got %v", engine.upServices)
	}
}

// The command palette reaches the same actions, so it needs the same answer:
// it used to prompt "Remove service cache containers?" on a row whose
// service has none.
func TestComposePalette_ContainerActionsOnAServiceWithNoContainers(t *testing.T) {
	for id, action := range map[string]string{
		"compose-service:logs":    "Logs",
		"compose-service:start":   "Start",
		"compose-service:stop":    "Stop",
		"compose-service:restart": "Restart",
		"compose-service:rm":      "Remove",
	} {
		t.Run(id, func(t *testing.T) {
			m := newTestModel()
			m.view = ComposeServices
			m.composeCLI = &stubComposeEngine{}
			m.selectedProject = "web"
			m.composeServices.SetSize(120, 40)
			m.composeServices.SetServices(nil, nil, nil, "web")
			m.composeServices.SetDrift(map[string]map[string]docker.ServiceSync{
				"web": {"cache": docker.ServiceNotCreated},
			})

			result, cmd := m.executePaletteAction(id)
			if cmd == nil {
				t.Fatalf("expected %s to explain itself, got nil", id)
			}
			status, ok := cmd().(statusMessageMsg)
			if !ok {
				t.Fatalf("expected a status message, got %T", cmd())
			}
			if want := action + " needs a container, and cache has none: u brings it up"; status.text != want {
				t.Errorf("expected %q, got %q", want, status.text)
			}
			if got := result.(model); got.overlay == overlayPrompt {
				t.Error("expected no prompt for an action that cannot act")
			}
		})
	}
}

// The Compose Projects view lists the same rows, so its container keys need
// the same answer. Nothing here selected a not-created row in that view
// before, which is how its palette twin stayed unguarded through a review.
func TestComposeProjectsView_ContainerKeysOnAServiceWithNoContainers(t *testing.T) {
	cases := map[string]struct {
		key    tea.KeyPressMsg
		action string
	}{
		"logs":    {tea.KeyPressMsg{Code: 'l'}, "Logs"},
		"enter":   {tea.KeyPressMsg{Code: tea.KeyEnter}, "Inspect"},
		"stop":    {tea.KeyPressMsg{Code: 't', Mod: tea.ModCtrl}, "Stop"},
		"restart": {tea.KeyPressMsg{Code: 'r', Mod: tea.ModCtrl}, "Restart"},
		"remove":  {tea.KeyPressMsg{Code: 'e', Mod: tea.ModCtrl}, "Remove"},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			m := newTestModel()
			m.view = ComposeProjects
			m.composeCLI = &stubComposeEngine{}
			m.composeProjects.SetSize(120, 40)
			m.composeProjects.SetProjects([]docker.ProjectWithServices{
				{Project: docker.ComposeProject{Name: "web"}},
			})
			m.composeProjects.SetDrift(map[string]map[string]docker.ServiceSync{
				"web": {"cache": docker.ServiceNotCreated},
			})
			// Move onto the service row, below the project header.
			result, _ := m.Update(tea.KeyPressMsg{Code: tea.KeyDown})
			m = result.(model)
			if svc := m.composeProjects.SelectedService(); svc == nil || svc.Containers != 0 {
				t.Fatalf("precondition: expected a service with no containers, got %+v", svc)
			}

			result, cmd := m.Update(tc.key)
			if cmd == nil {
				t.Fatalf("expected %s to explain itself, got nil", name)
			}
			status, ok := cmd().(statusMessageMsg)
			if !ok {
				t.Fatalf("expected a status message, got %T", cmd())
			}
			if want := tc.action + " needs a container, and cache has none: u brings it up"; status.text != want {
				t.Errorf("expected %q, got %q", want, status.text)
			}
			if got := result.(model); got.overlay == overlayPrompt {
				t.Error("expected no prompt for an action that cannot act")
			}
		})
	}
}

// The palette reaches the projects view's service actions too, and those
// are the entries that were missed while the guard was written out per site
// rather than shared.
func TestComposePalette_ProjectsViewContainerActions(t *testing.T) {
	for id, action := range map[string]string{
		"compose-project-service:logs":    "Logs",
		"compose-project-service:inspect": "Inspect",
	} {
		t.Run(id, func(t *testing.T) {
			m := newTestModel()
			m.view = ComposeProjects
			m.composeCLI = &stubComposeEngine{}
			m.composeProjects.SetSize(120, 40)
			m.composeProjects.SetProjects([]docker.ProjectWithServices{
				{Project: docker.ComposeProject{Name: "web"}},
			})
			m.composeProjects.SetDrift(map[string]map[string]docker.ServiceSync{
				"web": {"cache": docker.ServiceNotCreated},
			})
			result, _ := m.Update(tea.KeyPressMsg{Code: tea.KeyDown})
			m = result.(model)

			_, cmd := m.executePaletteAction(id)
			if cmd == nil {
				t.Fatalf("expected %s to explain itself, got nil", id)
			}
			status, ok := cmd().(statusMessageMsg)
			if !ok {
				t.Fatalf("expected a status message, got %T", cmd())
			}
			if want := action + " needs a container, and cache has none: u brings it up"; status.text != want {
				t.Errorf("expected %q, got %q", want, status.text)
			}
		})
	}
}

// Every palette action the two Compose views offer for a service must
// either refuse on a service with no containers or be one of the entries
// that deliberately act without one. The set is derived from
// commandPaletteActions rather than hand-written, so an action added later
// without a guard fails here instead of waiting for someone to notice: the
// guard was missed once per review round, in a different view each time.
//
// The listing is taken with a service that has containers selected, since
// the palette no longer offers the container-requiring entries on a row
// with none. Reading the list off the not-created row instead would leave
// this with nothing to check.
func TestComposePalette_EveryServiceActionHandlesAServiceWithNoContainers(t *testing.T) {
	actsWithoutContainers := map[string]bool{
		"compose:recreate":           true,
		"compose-service:up":         true,
		"compose-project-service:up": true,
	}
	drift := map[string]map[string]docker.ServiceSync{"web": {"cache": docker.ServiceNotCreated}}
	running := []docker.ComposeService{{Project: "web", Name: "api", Containers: 2, Running: 2}}

	for _, view := range []viewMode{ComposeProjects, ComposeServices} {
		// One model with a service that has containers, to read the full
		// list of entries off, and one on the not-created row to run them
		// against.
		listing := newComposeServiceSelectionModel(t, view, running, drift, "api")
		onEmpty := newComposeServiceSelectionModel(t, view, running, drift, "cache")

		var offered []string
		for _, a := range listing.commandPaletteActions() {
			// The compose service actions only: switch:services is a view
			// switch, and compose-project:* acts on the project.
			if strings.HasPrefix(a.ID, "compose-service:") || strings.HasPrefix(a.ID, "compose-project-service:") || a.ID == "compose:recreate" {
				offered = append(offered, a.ID)
			}
		}
		if len(offered) < 4 {
			t.Fatalf("view %v: expected the palette to offer the service actions, got %v", view, offered)
		}

		for _, id := range offered {
			if actsWithoutContainers[id] {
				continue
			}
			result, cmd := onEmpty.executePaletteAction(id)
			if cmd == nil {
				t.Errorf("view %v: %s did nothing on a service with no containers", view, id)
				continue
			}
			status, ok := cmd().(statusMessageMsg)
			if !ok {
				t.Errorf("view %v: %s acted on a service with no containers, got %T", view, id, cmd())
				continue
			}
			if !strings.Contains(status.text, "has none") {
				t.Errorf("view %v: %s said %q, expected it to say the service has none", view, id, status.text)
			}
			if got := result.(model); got.overlay == overlayPrompt {
				t.Errorf("view %v: %s prompted for an action that cannot act", view, id)
			}
		}

		// And the palette does not offer what it would refuse: on the
		// not-created row only the entries that create the service are
		// listed.
		for _, a := range onEmpty.commandPaletteActions() {
			if !strings.HasPrefix(a.ID, "compose-service:") && !strings.HasPrefix(a.ID, "compose-project-service:") && a.ID != "compose:recreate" {
				continue
			}
			if !actsWithoutContainers[a.ID] {
				t.Errorf("view %v: the palette offers %s on a service with no containers, which it refuses", view, a.ID)
			}
		}
	}
}

// newComposeServiceSelectionModel builds a model in one of the two Compose
// views with the named service under the cursor.
func newComposeServiceSelectionModel(t *testing.T, view viewMode, services []docker.ComposeService,
	drift map[string]map[string]docker.ServiceSync, want string) model {
	t.Helper()
	m := newTestModel()
	m.view = view
	m.composeCLI = &stubComposeEngine{}
	m.selectedProject = "web"
	switch view {
	case ComposeProjects:
		m.composeProjects.SetSize(120, 40)
		m.composeProjects.SetProjects([]docker.ProjectWithServices{{
			Project:  docker.ComposeProject{Name: "web"},
			Services: services,
		}})
		m.composeProjects.SetDrift(drift)
		for range 10 {
			if svc := m.composeProjects.SelectedService(); svc != nil && svc.Name == want {
				return m
			}
			next, _ := m.Update(tea.KeyPressMsg{Code: tea.KeyDown})
			m = next.(model)
		}
	case ComposeServices:
		m.composeServices.SetSize(120, 40)
		m.composeServices.SetServices(services, nil, nil, "web")
		m.composeServices.SetDrift(drift)
		for range 10 {
			if svc := m.composeServices.SelectedService(); svc != nil && svc.Name == want {
				return m
			}
			next, _ := m.Update(tea.KeyPressMsg{Code: tea.KeyDown})
			m = next.(model)
		}
	}
	t.Fatalf("view %v: no service row named %s", view, want)
	return m
}

// The footer is where a key is discovered, and it is truncated to the
// terminal width, so a binding added at the end of a long list is
// advertised in the source and nowhere on screen. Both compose footers have
// to show every key that acts on the selected row at a width a reader
// actually has: u, which every refusal on a not-created row names, and d,
// which takes a whole project down.
func TestComposeFooters_AdvertiseEveryRowAction(t *testing.T) {
	for _, tc := range []struct {
		view    viewMode
		name    string
		actions []string
	}{
		{ComposeProjects, "projects", []string{"services", "up", "down", "logs", "stop", "restart", "rm"}},
		{ComposeServices, "services", []string{"back", "inspect", "up", "logs", "start", "stop", "restart", "rm"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			m := newTestModel()
			m.view = tc.view
			m.width = 140
			m.height = 40

			footer := ansi.Strip(m.renderFooter())
			for _, action := range tc.actions {
				if !strings.Contains(footer, action) {
					t.Errorf("expected %q in the 140-column footer, got %q", action, footer)
				}
			}
		})
	}
}

// And u specifically answers to u, since the message tells the user to
// press it. The help screen documented it while the footer did not.
func TestComposeFooters_AdvertiseU(t *testing.T) {
	for _, tc := range []struct {
		view viewMode
		name string
	}{
		{ComposeProjects, "projects"},
		{ComposeServices, "services"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			m := newTestModel()
			m.view = tc.view

			var found bool
			for _, b := range m.viewFooterBindings() {
				if b.Help().Key != "u" {
					continue
				}
				found = true
				if b.Help().Desc != "up" {
					t.Errorf("expected u described as up, got %q", b.Help().Desc)
				}
				if !slices.Contains(b.Keys(), "u") {
					t.Errorf("expected the binding to answer to u, got %v", b.Keys())
				}
			}
			if !found {
				t.Fatal("expected u among the footer bindings")
			}

			// And it survives the render: the strip is truncated to the
			// terminal width, so a binding added at the end of a long list
			// is advertised in the source and nowhere on screen.
			m.width = 140
			m.height = 40
			footer := m.renderFooter()
			if !strings.Contains(footer, "up") {
				t.Errorf("expected u in the rendered footer, got %q", footer)
			}
		})
	}
}

// A lifecycle key in the Compose Projects view acts on the row under the
// cursor. All three took SelectedProject(), which returns the parent for a
// service row, so ctrl+e on a service prompted to remove every container in
// its project, while enter, l and u on the same row acted per service.
func TestComposeProjectsView_LifecycleKeysTargetTheSelectedRow(t *testing.T) {
	cases := map[string]struct {
		key         tea.KeyPressMsg
		serviceText string
		serviceTag  string
		projectText string
		projectTag  string
	}{
		"stop": {
			tea.KeyPressMsg{Code: 't', Mod: tea.ModCtrl},
			"Stop service web/api?", "compose-stop",
			"Stop project web?", "compose-project-stop",
		},
		"restart": {
			tea.KeyPressMsg{Code: 'r', Mod: tea.ModCtrl},
			"Restart service web/api?", "compose-restart",
			"Restart project web?", "compose-project-restart",
		},
		"remove": {
			tea.KeyPressMsg{Code: 'e', Mod: tea.ModCtrl},
			"Remove service web/api containers?", "compose-rm",
			"Remove project web containers?", "compose-project-rm",
		},
	}
	projects := []docker.ProjectWithServices{{
		Project:  docker.ComposeProject{Name: "web", Containers: 2},
		Services: []docker.ComposeService{{Project: "web", Name: "api", Containers: 2, Running: 2}},
	}}

	for name, tc := range cases {
		t.Run(name+"/service row", func(t *testing.T) {
			m := newTestModel()
			m.view = ComposeProjects
			m.composeCLI = &stubComposeEngine{}
			m.composeProjects.SetSize(120, 40)
			m.composeProjects.SetProjects(projects)
			down, _ := m.Update(tea.KeyPressMsg{Code: tea.KeyDown})
			m = down.(model)
			if svc := m.composeProjects.SelectedService(); svc == nil || svc.Name != "api" {
				t.Fatalf("precondition: expected web/api selected, got %+v", svc)
			}

			result, _ := m.Update(tc.key)
			got := result.(model)
			if got.overlay != overlayPrompt {
				t.Fatalf("expected a prompt, got overlay %v", got.overlay)
			}
			if q := ansi.Strip(got.prompt.View()); !strings.Contains(q, tc.serviceText) {
				t.Errorf("expected the prompt to ask %q, got %q", tc.serviceText, q)
			}
			// The wording and the tag are separate: a prompt worded per
			// service that carries the project tag still stops the project,
			// since the tag alone decides which daemon call runs.
			tag, id := answerPrompt(t, got)
			if tag != tc.serviceTag {
				t.Errorf("expected tag %q, got %q", tc.serviceTag, tag)
			}
			if id != composeServiceID("web", "api") {
				t.Errorf("expected the id for web's api service, got %q", id)
			}
		})

		t.Run(name+"/project header", func(t *testing.T) {
			m := newTestModel()
			m.view = ComposeProjects
			m.composeCLI = &stubComposeEngine{}
			m.composeProjects.SetSize(120, 40)
			m.composeProjects.SetProjects(projects)
			if p := m.composeProjects.SelectedProject(); p == nil || p.Name != "web" {
				t.Fatalf("precondition: expected web's header selected, got %+v", p)
			}

			result, _ := m.Update(tc.key)
			got := result.(model)
			if got.overlay != overlayPrompt {
				t.Fatalf("expected a prompt, got overlay %v", got.overlay)
			}
			if q := ansi.Strip(got.prompt.View()); !strings.Contains(q, tc.projectText) {
				t.Errorf("expected the prompt to ask %q, got %q", tc.projectText, q)
			}
			tag, id := answerPrompt(t, got)
			if tag != tc.projectTag {
				t.Errorf("expected tag %q, got %q", tc.projectTag, tag)
			}
			if id != "web" {
				t.Errorf("expected the project id web, got %q", id)
			}
		})
	}
}

// answerPrompt says yes to the open prompt and reports which operation the
// prompt was carrying. The tag and id are private to PromptModel, and they
// are what the confirmation dispatches on, so the y keypress is the only
// way to read them.
func answerPrompt(t *testing.T, m model) (tag, id string) {
	t.Helper()
	_, cmd := m.prompt.Update(tea.KeyPressMsg{Code: 'y'})
	if cmd == nil {
		t.Fatal("expected the prompt to answer a y keypress")
	}
	res, ok := cmd().(appui.PromptResultMsg)
	if !ok {
		t.Fatalf("expected a prompt result, got %T", cmd())
	}
	if !res.Confirmed {
		t.Fatal("expected y to confirm")
	}
	return res.Tag, res.ID
}

// u on a not-created row in the Compose Projects view brings up that one
// service. It is the key every other action's message names, and it is the
// key with the most to lose from a mistargeted row: gating it on the row
// having containers, the way the other keys are gated, made it fall through
// to the project branch, so u on a service row prompted for the whole
// project instead.
func TestComposeProjectsView_UBringsUpAServiceWithNoContainers(t *testing.T) {
	engine := &stubComposeEngine{}
	dir, file := composeFileFixture(t)
	m := newTestModel()
	m.view = ComposeProjects
	m.composeCLI = engine
	m.composeProjects.SetSize(120, 40)
	m.composeProjects.SetProjects([]docker.ProjectWithServices{{
		Project: docker.ComposeProject{Name: "web", WorkingDir: dir, ConfigFiles: []string{file}},
	}})
	m.composeProjects.SetDrift(map[string]map[string]docker.ServiceSync{
		"web": {"cache": docker.ServiceNotCreated},
	})
	down, _ := m.Update(tea.KeyPressMsg{Code: tea.KeyDown})
	m = down.(model)
	if svc := m.composeProjects.SelectedService(); svc == nil || svc.Name != "cache" {
		t.Fatalf("precondition: expected web/cache selected, got %+v", svc)
	}

	result, cmd := m.Update(tea.KeyPressMsg{Code: 'u'})
	if got := result.(model); got.overlay == overlayPrompt {
		t.Fatal("expected u on a service row to act without asking about a project")
	}
	if cmd == nil {
		t.Fatal("expected u to bring the service up")
	}
	cmd()
	if len(engine.upCalls) != 1 {
		t.Fatalf("expected one up call, got %d", len(engine.upCalls))
	}
	if len(engine.upServices) != 1 || engine.upServices[0] != "cache" {
		t.Errorf("expected up to target cache alone, got %v", engine.upServices)
	}
}

// Every refusal on a not-created row names u, so the palette has to offer
// it: the palette is the surface a user reaches for when they do not know
// the key, and it listed six actions that refuse and nothing that acts.
func TestComposePalette_OffersUpOnAServiceWithNoContainers(t *testing.T) {
	dir, file := composeFileFixture(t)
	drift := map[string]map[string]docker.ServiceSync{"web": {"cache": docker.ServiceNotCreated}}
	project := docker.ComposeProject{Name: "web", WorkingDir: dir, ConfigFiles: []string{file}}

	for _, tc := range []struct {
		view viewMode
		id   string
		name string
	}{
		{ComposeProjects, "compose-project-service:up", "projects"},
		{ComposeServices, "compose-service:up", "services"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			engine := &stubComposeEngine{}
			m := newTestModel()
			m.view = tc.view
			m.composeCLI = engine
			m.selectedProject = "web"
			m.composeProjects.SetSize(120, 40)
			m.composeProjects.SetProjects([]docker.ProjectWithServices{{Project: project}})
			m.composeProjects.SetDrift(drift)
			switch tc.view {
			case ComposeProjects:
				down, _ := m.Update(tea.KeyPressMsg{Code: tea.KeyDown})
				m = down.(model)
			case ComposeServices:
				m.composeServices.SetSize(120, 40)
				m.composeServices.SetServices(nil, nil, nil, "web")
				m.composeServices.SetDrift(drift)
			}

			var offered bool
			for _, a := range m.commandPaletteActions() {
				if a.ID == tc.id {
					offered = true
					if a.Title != "Up" {
						t.Errorf("expected the entry titled Up, got %q", a.Title)
					}
				}
			}
			if !offered {
				t.Fatalf("expected the palette to offer %s", tc.id)
			}

			_, cmd := m.executePaletteAction(tc.id)
			if cmd == nil {
				t.Fatal("expected the entry to bring the service up")
			}
			cmd()
			if len(engine.upServices) != 1 || engine.upServices[0] != "cache" {
				t.Errorf("expected up to target cache alone, got %v", engine.upServices)
			}
		})
	}
}

// The palette's project-level Up asks the same question the key asks. The
// two surfaces reaching the same action must agree, or the palette becomes
// the way to skip a confirmation.
func TestComposePalette_ProjectUpAsksFirst(t *testing.T) {
	engine := &stubComposeEngine{}
	dir, file := composeFileFixture(t)
	m := newTestModel()
	m.view = ComposeProjects
	m.composeCLI = engine
	m.composeProjects.SetSize(120, 40)
	m.composeProjects.SetProjects([]docker.ProjectWithServices{{
		Project: docker.ComposeProject{Name: "web", WorkingDir: dir, ConfigFiles: []string{file}},
	}})
	if p := m.composeProjects.SelectedProject(); p == nil {
		t.Fatal("precondition: expected web's header selected")
	}

	// Listed as well as handled: an entry the palette never offers is
	// unreachable, and the handler alone passes a test that calls it.
	var offered bool
	for _, a := range m.commandPaletteActions() {
		if a.ID == "compose-project:up" {
			offered = true
			if a.Title != "Up" {
				t.Errorf("expected the entry titled Up, got %q", a.Title)
			}
		}
	}
	if !offered {
		t.Fatal("expected the palette to offer compose-project:up")
	}

	result, _ := m.executePaletteAction("compose-project:up")
	asked := result.(model)
	if asked.overlay != overlayPrompt {
		t.Fatalf("expected a prompt, got overlay %v", asked.overlay)
	}
	if q := ansi.Strip(asked.prompt.View()); !strings.Contains(q, "Bring project web up?") {
		t.Errorf("expected the same question the key asks, got %q", q)
	}
	if len(engine.upCalls) != 0 {
		t.Errorf("expected nothing brought up before the answer, got %v", engine.upCalls)
	}
	tag, id := answerPrompt(t, asked)
	if tag != "compose-project-up" || id != "web" {
		t.Errorf("expected the project-up confirmation for web, got %q %q", tag, id)
	}
}

// A confirmation outlives the keypress that raised it, carrying only the
// prompt's id, so the id has to name one service unambiguously. Project and
// service names both come from container labels anything can set: joined by
// a slash, project "a" service "b/c" and project "a/b" service "c" are the
// same id and the answer acts on whichever one ops.go splits out.
func TestComposeServiceID_SurvivesNamesWithSlashes(t *testing.T) {
	cases := [][2]string{
		{"a", "b/c"},
		{"a/b", "c"},
		{"web", "api"},
		{"web/", "/api"},
	}
	seen := map[string][2]string{}
	for _, c := range cases {
		id := composeServiceID(c[0], c[1])
		if other, clash := seen[id]; clash {
			t.Errorf("%v and %v share the id %q", c, other, id)
		}
		seen[id] = c

		project, service := splitComposeServiceID(id)
		if project != c[0] || service != c[1] {
			t.Errorf("expected %q/%q back, got %q/%q", c[0], c[1], project, service)
		}
	}
}

// And the whole way through: a lifecycle key on a service of a project whose
// name contains a slash confirms into an action on that same service.
func TestComposeProjectsView_LifecycleOnASlashedProjectName(t *testing.T) {
	m := newTestModel()
	m.view = ComposeProjects
	m.composeCLI = &stubComposeEngine{}
	m.composeProjects.SetSize(120, 40)
	m.composeProjects.SetProjects([]docker.ProjectWithServices{{
		Project:  docker.ComposeProject{Name: "a/b", Containers: 1},
		Services: []docker.ComposeService{{Project: "a/b", Name: "c", Containers: 1, Running: 1}},
	}})
	down, _ := m.Update(tea.KeyPressMsg{Code: tea.KeyDown})
	m = down.(model)
	if svc := m.composeProjects.SelectedService(); svc == nil || svc.Name != "c" {
		t.Fatalf("precondition: expected a/b's service c selected, got %+v", svc)
	}

	result, _ := m.Update(tea.KeyPressMsg{Code: 't', Mod: tea.ModCtrl})
	if q := ansi.Strip(result.(model).prompt.View()); !strings.Contains(q, "Stop service a/b/c?") {
		t.Errorf("expected the prompt to name the project and the service, got %q", q)
	}
	tag, id := answerPrompt(t, result.(model))
	if tag != "compose-stop" {
		t.Errorf("expected the per-service stop, got %q", tag)
	}
	project, service := splitComposeServiceID(id)
	if project != "a/b" || service != "c" {
		t.Errorf("expected a/b and c back out of the confirmation, got %q and %q", project, service)
	}
}

// The help screen is the first entry in every footer, so it is where a user
// checks what a key does. It described ctrl+t, ctrl+r and ctrl+e in the
// Compose Projects view as project-level long after the code stopped
// treating them that way, which is worse than saying nothing.
func TestComposeHelp_DescribesWhatTheLifecycleKeysTarget(t *testing.T) {
	help := Help()
	start := strings.Index(help, "Compose Projects")
	end := strings.Index(help, "Compose Services")
	if start < 0 || end < start {
		t.Fatal("expected a Compose Projects section before the Compose Services one")
	}
	section := help[start:end]

	for _, key := range []string{"Ctrl+t", "Ctrl+r", "Ctrl+e", "u", "d"} {
		// The keybinding lines only: the section also has prose, which
		// mentions some of these keys in passing.
		var line string
		for _, l := range strings.Split(section, "\n") {
			if strings.HasPrefix(l, "\t<white>"+key+"</>") {
				line = l
			}
		}
		if line == "" {
			t.Errorf("no help line for %s in the Compose Projects section", key)
			continue
		}
		// Both halves: what the key does to the selected service, and
		// what it does to a project. d is the exception that proves the
		// rule, since it has no per-service form and has to say so.
		if !strings.Contains(line, "service") {
			t.Errorf("%s: expected the service target named, got %q", key, strings.TrimSpace(line))
		}
		if !strings.Contains(line, "project") {
			t.Errorf("%s: expected the project target named, got %q", key, strings.TrimSpace(line))
		}
	}
}

// The gate's boundary is one container, not two. Every fixture that expects
// an action to run gives its service two, so narrowing the gate to
// Containers > 1 would refuse every single-replica service in dry, which is
// most of them, and no test would notice.
func TestContainerAction_OneContainerIsEnough(t *testing.T) {
	for _, n := range []int{1, 2} {
		svc, why := containerAction(&docker.ComposeService{Project: "web", Name: "api", Containers: n}, "Stop")
		if why != nil {
			t.Errorf("with %d container(s), expected the action to run, got a refusal", n)
		}
		if svc == nil || svc.Containers != n {
			t.Errorf("with %d container(s), expected the service back, got %+v", n, svc)
		}
	}

	// And zero is the only count that refuses.
	svc, why := containerAction(&docker.ComposeService{Project: "web", Name: "cache"}, "Stop")
	if why == nil {
		t.Fatal("with no containers, expected a refusal")
	}
	if svc != nil {
		t.Errorf("expected no service to act on, got %+v", svc)
	}
	status, ok := why().(statusMessageMsg)
	if !ok {
		t.Fatalf("expected a status message, got %T", why())
	}
	if want := "Stop needs a container, and cache has none: u brings it up"; status.text != want {
		t.Errorf("expected %q, got %q", want, status.text)
	}

	// A nil selection is not a refusal: no service is selected at all, and
	// each caller words that for itself.
	if svc, why := containerAction(nil, "Stop"); svc != nil || why != nil {
		t.Errorf("expected nothing for a nil selection, got %+v and %v", svc, why != nil)
	}
}

// With no projects at all, a lifecycle key has no row to act on and says
// so. The branch is reachable from a host with no compose projects, which
// is the state the empty view was added for.
func TestComposeProjectsView_LifecycleKeysWithNoProjects(t *testing.T) {
	for name, key := range map[string]tea.KeyPressMsg{
		"stop":    {Code: 't', Mod: tea.ModCtrl},
		"restart": {Code: 'r', Mod: tea.ModCtrl},
		"remove":  {Code: 'e', Mod: tea.ModCtrl},
	} {
		t.Run(name, func(t *testing.T) {
			m := newTestModel()
			m.view = ComposeProjects
			m.composeCLI = &stubComposeEngine{}
			m.composeProjects.SetSize(120, 40)
			m.composeProjects.SetProjects(nil)

			result, cmd := m.Update(key)
			if cmd == nil {
				t.Fatalf("expected %s to explain itself, got nil", name)
			}
			status, ok := cmd().(statusMessageMsg)
			if !ok {
				t.Fatalf("expected a status message, got %T", cmd())
			}
			if !strings.Contains(status.text, "needs a project") {
				t.Errorf("expected the message to say it needs a project, got %q", status.text)
			}
			if got := result.(model); got.overlay == overlayPrompt {
				t.Error("expected no prompt with nothing to act on")
			}
		})
	}
}

// The panel's count is for the project it is asked about. In the Compose
// Services view only the project that view is showing can come from the
// services model; any other project's count still comes from the list that
// knows about it.
func TestWorkspacePanel_DefinedCountIsPerProject(t *testing.T) {
	m := newTestModel()
	m.view = ComposeServices
	m.selectedProject = "web"
	m.composeProjects.SetSize(120, 40)
	m.composeProjects.SetProjects([]docker.ProjectWithServices{
		{
			Project:  docker.ComposeProject{Name: "web", Services: 1},
			Services: []docker.ComposeService{{Project: "web", Name: "api", Containers: 1}},
		},
		{
			Project:  docker.ComposeProject{Name: "other", Services: 1},
			Services: []docker.ComposeService{{Project: "other", Name: "db", Containers: 1}},
		},
	})
	m.composeProjects.SetDrift(map[string]map[string]docker.ServiceSync{
		"web":   {"cache": docker.ServiceNotCreated},
		"other": {"redis": docker.ServiceNotCreated, "queue": docker.ServiceNotCreated},
	})
	m.composeServices.SetSize(120, 40)
	m.composeServices.SetServices(
		[]docker.ComposeService{{Project: "web", Name: "api", Containers: 1}}, nil, nil, "web")
	m.composeServices.SetDrift(map[string]map[string]docker.ServiceSync{
		"web": {"cache": docker.ServiceNotCreated},
	})

	// web is what this view shows, so its count is the services model's.
	if got := m.definedServiceCount("web"); got != 2 {
		t.Errorf("expected web's two rows from the view showing them, got %d", got)
	}
	// other is not, so its count comes from the projects list.
	if got := m.definedServiceCount("other"); got != 3 {
		t.Errorf("expected other's three rows from the projects list, got %d", got)
	}
}

// d is the one lifecycle key with no per-service form, since compose down
// removes the project's networks along with its containers. On a service
// row it says so and names the project's row, rather than prompting to take
// the parent project down, which is what the other three keys used to do.
func TestComposeProjectsView_DownOnAServiceRowSaysWhereItLives(t *testing.T) {
	m := newTestModel()
	m.view = ComposeProjects
	m.composeCLI = &stubComposeEngine{}
	m.composeProjects.SetSize(120, 40)
	m.composeProjects.SetProjects([]docker.ProjectWithServices{{
		Project: docker.ComposeProject{Name: "web", Containers: 2},
		Services: []docker.ComposeService{
			{Project: "web", Name: "api", Containers: 1, Running: 1},
			{Project: "web", Name: "db", Containers: 1, Running: 1},
		},
	}})
	// The second service, where the row above is another service and not
	// the project's header.
	for range 2 {
		next, _ := m.Update(tea.KeyPressMsg{Code: tea.KeyDown})
		m = next.(model)
	}
	if svc := m.composeProjects.SelectedService(); svc == nil || svc.Name != "db" {
		t.Fatalf("precondition: expected web/db selected, got %+v", svc)
	}

	result, cmd := m.Update(tea.KeyPressMsg{Code: 'd'})
	if got := result.(model); got.overlay == overlayPrompt {
		t.Fatal("expected no prompt to take a project down from a service row")
	}
	if cmd == nil {
		t.Fatal("expected d to explain itself, got nil")
	}
	status, ok := cmd().(statusMessageMsg)
	if !ok {
		t.Fatalf("expected a status message, got %T", cmd())
	}
	want := "Down takes a whole project, not the service db: press d on the web row"
	if status.text != want {
		t.Errorf("expected %q, got %q", want, status.text)
	}

	// Under a filter the project's row can be hidden, so the message stops
	// pointing at a row the reader cannot see.
	filtered := m
	filtered.composeProjects.SetFilter("db")
	if svc := filtered.composeProjects.SelectedService(); svc == nil || svc.Name != "db" {
		t.Fatalf("precondition: expected the filter to leave web/db selected, got %+v", svc)
	}
	_, cmd = filtered.Update(tea.KeyPressMsg{Code: 'd'})
	if cmd == nil {
		t.Fatal("expected d under a filter to explain itself")
	}
	if status, ok := cmd().(statusMessageMsg); !ok {
		t.Fatalf("expected a status message, got %T", cmd())
	} else if want := "Down takes a whole project, not the service db: clear the filter to reach web"; status.text != want {
		t.Errorf("expected %q, got %q", want, status.text)
	}

	// And on the project's own row it still asks and still means the
	// project.
	up, _ := m.Update(tea.KeyPressMsg{Code: tea.KeyUp})
	m = up.(model)
	up, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyUp})
	m = up.(model)
	if p := m.composeProjects.SelectedProject(); p == nil || m.composeProjects.SelectedService() != nil {
		t.Fatalf("precondition: expected web's header selected, got %+v", p)
	}
	asked, _ := m.Update(tea.KeyPressMsg{Code: 'd'})
	if got := asked.(model); got.overlay != overlayPrompt {
		t.Fatal("expected d on the project's row to ask")
	}
	if q := ansi.Strip(asked.(model).prompt.View()); !strings.Contains(q, "Take project web down?") {
		t.Errorf("expected the prompt to name the project, got %q", q)
	}
	if tag, id := answerPrompt(t, asked.(model)); tag != "compose-project-down" || id != "web" {
		t.Errorf("expected the project-down confirmation for web, got %q %q", tag, id)
	}
}

// The Compose Projects view lists every project at once, so a prompt that
// names only the service is ambiguous the moment two projects have a
// service of the same name: the id is right, but what the user reads before
// answering is not.
func TestComposeProjectsView_LifecyclePromptsNameTheProject(t *testing.T) {
	m := newTestModel()
	m.view = ComposeProjects
	m.composeCLI = &stubComposeEngine{}
	m.composeProjects.SetSize(120, 40)
	m.composeProjects.SetProjects([]docker.ProjectWithServices{
		{
			Project:  docker.ComposeProject{Name: "alpha", Containers: 1},
			Services: []docker.ComposeService{{Project: "alpha", Name: "web", Containers: 1, Running: 1}},
		},
		{
			Project:  docker.ComposeProject{Name: "beta", Containers: 1},
			Services: []docker.ComposeService{{Project: "beta", Name: "web", Containers: 1, Running: 1}},
		},
	})

	questions := map[string]string{}
	for _, want := range []string{"alpha", "beta"} {
		for range 4 {
			if svc := m.composeProjects.SelectedService(); svc != nil && svc.Project == want {
				break
			}
			next, _ := m.Update(tea.KeyPressMsg{Code: tea.KeyDown})
			m = next.(model)
		}
		svc := m.composeProjects.SelectedService()
		if svc == nil || svc.Project != want {
			t.Fatalf("precondition: expected %s's web selected, got %+v", want, svc)
		}
		asked, _ := m.Update(tea.KeyPressMsg{Code: 't', Mod: tea.ModCtrl})
		questions[want] = ansi.Strip(asked.(model).prompt.View())
	}

	if questions["alpha"] == questions["beta"] {
		t.Errorf("the two projects' web services ask the same question: %q", questions["alpha"])
	}
	for project, q := range questions {
		if !strings.Contains(q, project+"/web") {
			t.Errorf("expected %s's prompt to name the project, got %q", project, q)
		}
	}
}

// Every project-level key has a palette entry, since the palette is where
// a user who does not know the keys looks. d and c had none: the palette
// could bring a project up and never take it down.
func TestComposePalette_OffersEveryProjectAction(t *testing.T) {
	dir, file := composeFileFixture(t)
	engine := &stubComposeEngine{}
	m := newTestModel()
	m.view = ComposeProjects
	m.composeCLI = engine
	m.composeProjects.SetSize(120, 40)
	m.composeProjects.SetProjects([]docker.ProjectWithServices{{
		Project: docker.ComposeProject{Name: "web", Containers: 1, WorkingDir: dir, ConfigFiles: []string{file}},
	}})
	if p := m.composeProjects.SelectedProject(); p == nil {
		t.Fatal("precondition: expected web's header selected")
	}

	titles := map[string]string{}
	for _, a := range m.commandPaletteActions() {
		if strings.HasPrefix(a.ID, "compose-project:") {
			titles[a.ID] = a.Title
		}
	}
	for id, title := range map[string]string{
		"compose-project:open":    "Open Resources",
		"compose-project:logs":    "Logs",
		"compose-project:up":      "Up",
		"compose-project:down":    "Down",
		"compose-project:config":  "Config",
		"compose-project:stop":    "Stop",
		"compose-project:restart": "Restart",
		"compose-project:rm":      "Remove Containers",
	} {
		if got, ok := titles[id]; !ok {
			t.Errorf("expected the palette to offer %s", id)
		} else if got != title {
			t.Errorf("expected %s titled %q, got %q", id, title, got)
		}
	}

	// Down asks the same question the key asks, and carries the same tag.
	asked, _ := m.executePaletteAction("compose-project:down")
	if got := asked.(model); got.overlay != overlayPrompt {
		t.Fatal("expected the palette's Down to ask first")
	}
	if q := ansi.Strip(asked.(model).prompt.View()); !strings.Contains(q, "Take project web down?") {
		t.Errorf("expected the same question the key asks, got %q", q)
	}
	if tag, id := answerPrompt(t, asked.(model)); tag != "compose-project-down" || id != "web" {
		t.Errorf("expected the project-down confirmation for web, got %q %q", tag, id)
	}

	// And Config runs against the project's own files, since a config
	// rendered from an empty project is another project's configuration.
	if _, cmd := m.executePaletteAction("compose-project:config"); cmd == nil {
		t.Fatal("expected the palette's Config to run")
	} else {
		cmd()
	}
	if len(engine.configCalls) != 1 {
		t.Fatalf("expected one config call, got %+v", engine.configCalls)
	}
	if got := engine.configCalls[0]; got.Name != "web" || got.WorkingDir != dir ||
		len(got.Files) != 1 || got.Files[0] != file {
		t.Errorf("expected config to run against web at %s with %s, got %+v", dir, file, got)
	}
}

// The palette's per-service lifecycle entries in the Compose Projects view
// must reach the same action the key reaches: the same prompt, the same
// tag, and an id whose halves are the right way round. Swapping them reads
// identically on screen and removes a service called after the project.
func TestComposePalette_ProjectsViewLifecycleMatchesTheKeys(t *testing.T) {
	cases := map[string]struct {
		id   string
		key  tea.KeyPressMsg
		tag  string
		text string
	}{
		"stop": {"compose-project-service:stop", tea.KeyPressMsg{Code: 't', Mod: tea.ModCtrl},
			"compose-stop", "Stop service web/api?"},
		"restart": {"compose-project-service:restart", tea.KeyPressMsg{Code: 'r', Mod: tea.ModCtrl},
			"compose-restart", "Restart service web/api?"},
		"rm": {"compose-project-service:rm", tea.KeyPressMsg{Code: 'e', Mod: tea.ModCtrl},
			"compose-rm", "Remove service web/api containers?"},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			build := func() model {
				m := newTestModel()
				m.view = ComposeProjects
				m.composeCLI = &stubComposeEngine{}
				m.composeProjects.SetSize(120, 40)
				m.composeProjects.SetProjects([]docker.ProjectWithServices{{
					Project:  docker.ComposeProject{Name: "web", Containers: 2},
					Services: []docker.ComposeService{{Project: "web", Name: "api", Containers: 2, Running: 2}},
				}})
				down, _ := m.Update(tea.KeyPressMsg{Code: tea.KeyDown})
				return down.(model)
			}

			m := build()
			if svc := m.composeProjects.SelectedService(); svc == nil || svc.Name != "api" {
				t.Fatalf("precondition: expected web/api selected, got %+v", svc)
			}
			viaPalette, _ := m.executePaletteAction(tc.id)
			asked := viaPalette.(model)
			if asked.overlay != overlayPrompt {
				t.Fatalf("expected %s to ask, got overlay %v", tc.id, asked.overlay)
			}
			if q := ansi.Strip(asked.prompt.View()); !strings.Contains(q, tc.text) {
				t.Errorf("expected %q, got %q", tc.text, q)
			}
			tag, id := answerPrompt(t, asked)
			if tag != tc.tag {
				t.Errorf("expected tag %q, got %q", tc.tag, tag)
			}
			// The halves the right way round, read back the way ops.go
			// reads them.
			project, service := splitComposeServiceID(id)
			if project != "web" || service != "api" {
				t.Errorf("expected project web and service api, got %q and %q", project, service)
			}

			// And the key on the same row produces the same thing.
			viaKey, _ := build().Update(tc.key)
			keyAsked := viaKey.(model)
			keyTag, keyID := answerPrompt(t, keyAsked)
			if keyTag != tag || keyID != id {
				t.Errorf("the key confirms %q/%q where the palette confirms %q/%q", keyTag, keyID, tag, id)
			}
			if ansi.Strip(keyAsked.prompt.View()) != ansi.Strip(asked.prompt.View()) {
				t.Errorf("the key asks %q where the palette asks %q",
					ansi.Strip(keyAsked.prompt.View()), ansi.Strip(asked.prompt.View()))
			}
		})
	}
}

// c renders the project's configuration, so it has to hand compose the
// project's own working directory and files. Given a bare name instead,
// compose resolves whatever file sits in dry's directory and renders that,
// which reads as the selected project's configuration.
func TestComposeConfig_RunsAgainstTheProjectsOwnFiles(t *testing.T) {
	dir, file := composeFileFixture(t)
	project := docker.ComposeProject{Name: "web", WorkingDir: dir, ConfigFiles: []string{file}}

	for _, tc := range []struct {
		view viewMode
		name string
	}{
		{ComposeProjects, "projects"},
		{ComposeServices, "services"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			engine := &stubComposeEngine{configOutput: "services:\n  api:\n    image: nginx\n"}
			m := newTestModel()
			m.view = tc.view
			m.composeCLI = engine
			m.selectedProject = "web"
			m.composeProjects.SetSize(120, 40)
			m.composeProjects.SetProjects([]docker.ProjectWithServices{{Project: project}})

			_, cmd := m.Update(tea.KeyPressMsg{Code: 'c'})
			if cmd == nil {
				t.Fatal("expected c to render the configuration")
			}
			cmd()
			if len(engine.configCalls) != 1 {
				t.Fatalf("expected one config call, got %+v", engine.configCalls)
			}
			got := engine.configCalls[0]
			if got.Name != "web" || got.WorkingDir != dir || len(got.Files) != 1 || got.Files[0] != file {
				t.Errorf("expected config against web at %s with %s, got %+v", dir, file, got)
			}
		})
	}
}

// The palette's per-service entries are listed with a service selected, so
// their empty branch needs the selection to have moved while the palette
// was open: a filter typed into the projects list can hide every service
// row. The message then has to name the filter, since the reader can see a
// project and no service under it.
func TestComposePalette_ProjectsViewLifecycleWithNoServiceRow(t *testing.T) {
	for name, filter := range map[string]string{"filtered": "no-such-service", "empty": ""} {
		t.Run(name, func(t *testing.T) {
			m := newTestModel()
			m.view = ComposeProjects
			m.composeCLI = &stubComposeEngine{}
			m.composeProjects.SetSize(120, 40)
			m.composeProjects.SetProjects([]docker.ProjectWithServices{{
				Project: docker.ComposeProject{Name: "web"},
			}})
			if filter != "" {
				m.composeProjects.SetFilter(filter)
			}
			if svc := m.composeProjects.SelectedService(); svc != nil {
				t.Fatalf("precondition: expected no service row, got %+v", svc)
			}

			result, cmd := m.executePaletteAction("compose-project-service:stop")
			if cmd == nil {
				t.Fatal("expected the entry to explain itself, got nil")
			}
			status, ok := cmd().(statusMessageMsg)
			if !ok {
				t.Fatalf("expected a status message, got %T", cmd())
			}
			if !strings.Contains(status.text, "needs a service") {
				t.Errorf("expected the message to say it needs a service, got %q", status.text)
			}
			// Named, not guessed: a filter is the reason a project with
			// services shows none, and the reader has to know to clear it.
			if filter != "" && !strings.Contains(status.text, "filter") {
				t.Errorf("expected the filter named, got %q", status.text)
			}
			if filter == "" && strings.Contains(status.text, "filter") {
				t.Errorf("expected no filter mentioned with none set, got %q", status.text)
			}
			if got := result.(model); got.overlay == overlayPrompt {
				t.Error("expected no prompt with nothing to act on")
			}
		})
	}
}
