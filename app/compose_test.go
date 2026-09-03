package app

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"
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
	configOutput     string
	hashes           map[string]string
	hashesCalls      []composecli.Project
	hashesErr        error
	hangHashes       bool
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

func (s *stubComposeEngine) Config(_ context.Context, _ composecli.Project) (string, error) {
	return s.configOutput, s.err
}

func (s *stubComposeEngine) ConfigHashes(ctx context.Context, p composecli.Project) (map[string]string, error) {
	s.hashesCalls = append(s.hashesCalls, p)
	// hangHashes makes the stub behave like a compose that never answers:
	// it waits for the caller's context instead of returning, which is what
	// the drift cycle's budget has to survive. The fallback matters: with
	// only <-ctx.Done() here, an unbounded context turns a regression into
	// a hung package instead of a failed assertion.
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

	msg := composeScanCmd(resolver, dir, nil)()
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

	msg := composeScanCmd(&stubResolver{name: "unused"}, t.TempDir(), existing)()
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

	msg := composeScanCmd(&stubResolver{err: errors.New("invalid compose file")}, dir, nil)()
	loaded := msg.(composeProjectsMsg)
	if len(loaded.projects) != 0 {
		t.Fatalf("an unresolvable file must not invent a project, got %+v", loaded.projects)
	}
}

func TestComposeProjectsView_UBringsTheProjectUp(t *testing.T) {
	engine := &stubComposeEngine{}
	dir, file := composeFileFixture(t)
	m := newTestModel()
	m.view = ComposeProjects
	m.composeCLI = engine
	m.composeProjects.SetProjects([]docker.ProjectWithServices{{
		Project: docker.ComposeProject{Name: "web", WorkingDir: dir, ConfigFiles: []string{file}},
	}})

	_, cmd := m.Update(tea.KeyPressMsg{Code: 'u'})
	if cmd == nil {
		t.Fatal("expected u to produce a command")
	}
	if _, ok := cmd().(showStreamingLessMsg); !ok {
		t.Fatalf("expected up to stream into the viewer, got %T", cmd())
	}
	if len(engine.upCalls) != 1 {
		t.Fatalf("expected one Up call, got %d", len(engine.upCalls))
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

	msg := composeDriftCmd(engine, projects, containers)()
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

	msg := composeDriftCmd(engine, projects, nil)()
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

	msg := composeDriftCmd(engine, projects, nil)()
	drift, ok := msg.(composeDriftMsg)
	if !ok {
		t.Fatalf("expected composeDriftMsg, got %T", msg)
	}
	if _, ok := drift.drift["web"]; ok {
		t.Fatalf("expected a project whose files are not readable here to be skipped, got %+v", drift.drift)
	}
	if drift.err != nil {
		t.Fatalf("a file dry cannot read must not raise an error banner, got %q", drift.err)
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
	// composeDriftMsg ends the cycle, whatever it found.
	result, _ = m.Update(composeDriftMsg{})
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

	result, cmd := m.Update(composeDriftMsg{})
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

	if _, cmd := m.Update(composeDriftMsg{}); cmd != nil {
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

	// u brings it up, and compose is told which file to use.
	_, cmd = m.Update(tea.KeyPressMsg{Code: 'u'})
	if cmd == nil {
		t.Fatal("expected u to produce a command")
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
// ConfigHashes error must never be swallowed. The project is dropped from
// the drift map (its SYNC column stays empty, same as "not checked yet"),
// but the error must still reach composeDriftMsg.err so the model can
// surface it rather than have it vanish.
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

	msg := composeDriftCmd(engine, projects, nil)()
	drift, ok := msg.(composeDriftMsg)
	if !ok {
		t.Fatalf("expected composeDriftMsg, got %T", msg)
	}
	if _, ok := drift.drift["web"]; ok {
		t.Fatalf("expected the failed project to be dropped from drift, got %+v", drift.drift)
	}
	if drift.err == nil {
		t.Fatal("expected the ConfigHashes failure to be reported, not swallowed")
	}
	if !strings.Contains(drift.err.Error(), "web") || !strings.Contains(drift.err.Error(), failure.Error()) {
		t.Fatalf("expected the error to name the project and the cause, got %q", drift.err)
	}
}

// TestComposeDriftMsg_ErrSurfacesAsStatusMessage proves the model actually
// tells the user about a drift-check failure instead of only logging it in
// a struct field nothing reads. Without this, finding 4 could be "fixed" by
// populating composeDriftMsg.err and never wiring it to anything visible.
func TestComposeDriftMsg_ErrSurfacesAsStatusMessage(t *testing.T) {
	m := newTestModel()
	failure := errors.New("compose config --hash failed: web")

	result, cmd := m.Update(composeDriftMsg{
		drift: map[string]map[string]docker.ServiceSync{},
		err:   failure,
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
	msg := composeDriftCmd(engine, projects, nil)()
	elapsed := time.Since(start)

	drift, ok := msg.(composeDriftMsg)
	if !ok {
		t.Fatalf("expected composeDriftMsg, got %T", msg)
	}
	if drift.err == nil {
		t.Fatal("expected the cycle to report that it gave up")
	}
	if !strings.Contains(drift.err.Error(), "gave up") {
		t.Fatalf("expected the error to say the cycle gave up, got %v", drift.err)
	}
	if !strings.Contains(drift.err.Error(), "unchecked") {
		t.Fatalf("expected the error to say how many projects were left, got %v", drift.err)
	}
	// Once, not once per remaining project: the loop breaks rather than
	// continuing, or a one-line message bar gets four copies of the same
	// sentence.
	if got := strings.Count(drift.err.Error(), "gave up"); got != 1 {
		t.Fatalf("expected the expiry reported once, got %d copies: %v", got, drift.err)
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
