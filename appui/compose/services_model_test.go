package compose

import (
	"strings"
	"testing"

	"github.com/moncho/dry/docker"
)

// TestServicesModel_SetDrift_RendersSyncColumn is the services-view sibling
// of the projects-view drift rendering test: a duplicate colorSync/SYNC
// column exists in this file's newServiceRow, so a bug in only this model
// would go uncaught if only the projects model were tested.
func TestServicesModel_SetDrift_RendersSyncColumn(t *testing.T) {
	m := NewServicesModel()
	m.SetSize(120, 40)
	m.SetServices([]docker.ComposeService{
		{Project: "web", Name: "api"},
		{Project: "web", Name: "worker"},
	}, nil, nil, "web")
	m.SetDrift(map[string]map[string]docker.ServiceSync{
		"web": {
			"api":    docker.ServiceNotCreated,
			"worker": docker.ServiceInSync,
		},
	})

	var apiCols, workerCols []string
	for _, row := range m.table.FilteredRows() {
		r, ok := row.(serviceRow)
		if !ok {
			continue
		}
		switch r.service.Name {
		case "api":
			apiCols = r.Columns()
		case "worker":
			workerCols = r.Columns()
		}
	}
	if apiCols == nil || workerCols == nil {
		t.Fatalf("expected both service rows to be present, got api=%v worker=%v", apiCols, workerCols)
	}

	const syncCol = 6 // NAME, CONTAINERS, RUNNING, EXITED, IMAGE/DRIVER, HEALTH/SCOPE, SYNC, PORTS
	if !strings.Contains(apiCols[syncCol], "none") {
		t.Fatalf("expected the not-created service's SYNC cell to render \"none\", got %q", apiCols[syncCol])
	}
	if !strings.Contains(workerCols[syncCol], "ok") {
		t.Fatalf("expected the in-sync service's SYNC cell to render \"ok\", got %q", workerCols[syncCol])
	}
}

// TestServicesModel_SetServices_NetworkAndVolumeRowsHaveEmptySyncCell guards
// finding-adjacent column alignment: network and volume rows must carry an
// empty SYNC cell in the same position as the header, or their remaining
// columns (PORTS in particular) shift left relative to the header.
func TestServicesModel_SetServices_NetworkAndVolumeRowsHaveEmptySyncCell(t *testing.T) {
	m := NewServicesModel()
	m.SetSize(120, 40)
	m.SetServices(nil,
		[]docker.ComposeNetwork{{Name: "web_default", Driver: "bridge", Scope: "local"}},
		[]docker.ComposeVolume{{Name: "web_data", Driver: "local"}},
		"web")

	var netCols, volCols []string
	for _, row := range m.table.FilteredRows() {
		switch r := row.(type) {
		case networkRow:
			netCols = r.Columns()
		case volumeRow:
			volCols = r.Columns()
		}
	}
	if netCols == nil || volCols == nil {
		t.Fatalf("expected both a network and a volume row, got net=%v vol=%v", netCols, volCols)
	}
	const syncCol = 6
	if netCols[syncCol] != "" {
		t.Fatalf("expected the network row's SYNC cell to be empty, got %q", netCols[syncCol])
	}
	if volCols[syncCol] != "" {
		t.Fatalf("expected the volume row's SYNC cell to be empty, got %q", volCols[syncCol])
	}
	// Ports column (index 7) must still be the last column for both.
	if len(netCols) != 8 || len(volCols) != 8 {
		t.Fatalf("expected 8 columns (NAME..PORTS) on both rows, got net=%d vol=%d", len(netCols), len(volCols))
	}
}
