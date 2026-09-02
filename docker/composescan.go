package docker

import (
	"os"
	"path/filepath"
	"slices"
)

// composeFileNames are the file names compose recognises, in the order
// compose itself prefers them. The legacy name prefers .yml and the current
// one .yaml, which looks like a typo and is not: get them backwards and, in
// a directory holding both legacy names, dry acts on a different file from
// the one `docker compose up` there would. Compose warns which of several it
// picked, and between docker-compose.yml and docker-compose.yaml it takes
// the .yml.
var composeFileNames = []string{
	"compose.yaml",
	"compose.yml",
	"docker-compose.yml",
	"docker-compose.yaml",
}

// ScanComposeDir looks for a compose file in the given directory and returns
// the single highest-precedence match. It reports false when the directory
// holds no compose file.
func ScanComposeDir(dir string) ([]string, bool) {
	if dir == "" {
		return nil, false
	}
	for _, name := range composeFileNames {
		path := filepath.Join(dir, name)
		if info, err := os.Stat(path); err == nil && !info.IsDir() {
			return []string{path}, true
		}
	}
	return nil, false
}

// MergeScannedProject folds a project discovered by scanning a directory into
// the label-derived list. A project that already has containers wins, because
// its status and counts are real; the scan only fills in file paths it was
// missing. A project with no containers is appended.
//
// This runs on a command goroutine while the caller still holds the slice it
// passed in: the Compose view model stores it, and the Update goroutine reads
// its elements on any keypress that resolves the selected project. So every
// write goes to a copy. The elements are values, so cloning the slice is
// enough to make the caller's memory read-only here; nothing this function
// touches reaches through a pointer into the original. The clone also gives
// the append an array of its own, instead of writing a new element into
// spare capacity the caller's slice still owns.
func MergeScannedProject(projects []ProjectWithServices, scanned ComposeProject) []ProjectWithServices {
	if scanned.Name == "" {
		return projects
	}
	merged := slices.Clone(projects)
	for i := range merged {
		if merged[i].Project.Name != scanned.Name {
			continue
		}
		if len(merged[i].Project.ConfigFiles) == 0 {
			merged[i].Project.ConfigFiles = scanned.ConfigFiles
		}
		if merged[i].Project.WorkingDir == "" {
			merged[i].Project.WorkingDir = scanned.WorkingDir
		}
		return merged
	}
	return append(merged, ProjectWithServices{Project: scanned})
}
