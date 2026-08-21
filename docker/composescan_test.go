package docker

import (
	"os"
	"path/filepath"
	"testing"
)

func TestScanComposeDir_PrefersComposeYaml(t *testing.T) {
	dir := t.TempDir()
	for _, name := range []string{"docker-compose.yml", "compose.yaml"} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte("services: {}\n"), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	files, ok := ScanComposeDir(dir)
	if !ok {
		t.Fatal("expected a compose file to be found")
	}
	if len(files) != 1 || filepath.Base(files[0]) != "compose.yaml" {
		t.Fatalf("expected compose.yaml to win, got %v", files)
	}
}

func TestScanComposeDir_NoFile(t *testing.T) {
	if _, ok := ScanComposeDir(t.TempDir()); ok {
		t.Fatal("expected no compose file in an empty directory")
	}
}

func TestMergeScannedProject_LabelDerivedWins(t *testing.T) {
	existing := []ProjectWithServices{{
		Project: ComposeProject{Name: "web", Containers: 2, Status: ProjectRunning},
	}}
	scanned := ComposeProject{Name: "web", ConfigFiles: []string{"/srv/web/compose.yaml"}, Status: ProjectNotCreated}

	merged := MergeScannedProject(existing, scanned)
	if len(merged) != 1 {
		t.Fatalf("expected no duplicate project, got %d", len(merged))
	}
	if merged[0].Project.Status != ProjectRunning {
		t.Fatalf("expected the label-derived project to win, got %q", merged[0].Project.Status)
	}
	if len(merged[0].Project.ConfigFiles) != 1 {
		t.Fatalf("expected the scan to fill in missing files, got %v", merged[0].Project.ConfigFiles)
	}
}

// TestMergeScannedProject_DoesNotMutateItsInput is the fix for a concurrent
// write: the caller hands this function the very slice the Compose view model
// is already holding (app/model.go passes msg.Projects to composeScanCmd
// after SetProjects stored it), and the Update goroutine reads those elements
// on every u/d/palette keypress via ProjectByName/SelectedProject. Writing
// into the caller's backing array from the scan goroutine is a data race on a
// slice header, which can tear rather than merely misread. -race does not
// catch it because no test presses a key while a scan is in flight, so the
// invariant is asserted directly instead: the input the caller still holds is
// never written to.
func TestMergeScannedProject_DoesNotMutateItsInput(t *testing.T) {
	input := []ProjectWithServices{
		{Project: ComposeProject{Name: "web", Containers: 2, Status: ProjectRunning}},
		{Project: ComposeProject{Name: "other", Containers: 1, Status: ProjectRunning}},
	}
	scanned := ComposeProject{
		Name:        "web",
		ConfigFiles: []string{"/srv/web/compose.yaml"},
		WorkingDir:  "/srv/web",
		Status:      ProjectNotCreated,
	}

	merged := MergeScannedProject(input, scanned)

	if len(input[0].Project.ConfigFiles) != 0 {
		t.Fatalf("the caller's slice was mutated: ConfigFiles = %v", input[0].Project.ConfigFiles)
	}
	if input[0].Project.WorkingDir != "" {
		t.Fatalf("the caller's slice was mutated: WorkingDir = %q", input[0].Project.WorkingDir)
	}
	if len(merged[0].Project.ConfigFiles) != 1 || merged[0].Project.WorkingDir != "/srv/web" {
		t.Fatalf("the returned slice must carry the enrichment, got %+v", merged[0].Project)
	}
	if &merged[0] == &input[0] {
		t.Fatal("the returned slice must not share its backing array with the input")
	}
}

// TestMergeScannedProject_AppendDoesNotMutateItsInput covers the other
// branch: appending a scanned project must not write into spare capacity the
// caller's slice still owns, which would publish a new element into a slice
// another goroutine is iterating.
func TestMergeScannedProject_AppendDoesNotMutateItsInput(t *testing.T) {
	input := make([]ProjectWithServices, 1, 4) // spare capacity: append writes in place
	input[0] = ProjectWithServices{Project: ComposeProject{Name: "web"}}

	merged := MergeScannedProject(input, ComposeProject{Name: "fresh", Status: ProjectNotCreated})

	if len(merged) != 2 {
		t.Fatalf("expected the scanned project to be appended, got %+v", merged)
	}
	if got := input[:cap(input)][1].Project.Name; got != "" {
		t.Fatalf("the append wrote into the caller's backing array: %q", got)
	}
}

func TestMergeScannedProject_AddsNotCreatedProject(t *testing.T) {
	scanned := ComposeProject{Name: "fresh", ConfigFiles: []string{"/srv/fresh/compose.yaml"}, Status: ProjectNotCreated}

	merged := MergeScannedProject(nil, scanned)
	if len(merged) != 1 || merged[0].Project.Name != "fresh" {
		t.Fatalf("expected the scanned project to be listed, got %+v", merged)
	}
	if merged[0].Project.Status != ProjectNotCreated {
		t.Fatalf("expected status not created, got %q", merged[0].Project.Status)
	}
}
