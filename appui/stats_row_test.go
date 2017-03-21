package appui

import (
	"testing"

	"github.com/docker/docker/api/types"
	"github.com/moncho/dry/docker"
)

func TestStatsRow(t *testing.T) {
	container := &types.Container{ID: "CID", Names: []string{"Name"}, Status: "Never worked"}
	sc := &docker.StatsChannel{Container: container}

	row := NewContainerStatsRow(sc)
	if row == nil {
		t.Error("Stats row was not created")
	}
	if row.container != container {
		t.Error("Stats row does not hold a reference to the container.")
	}

	if len(row.columns) != 7 {
		t.Errorf("Stats row does not have the expected number of columns: %d.", len(row.columns))
	}

	if row.ID.Text != container.ID {
		t.Errorf("ID widget does not contain the container ID. Expected: %s, got: %s.", container.ID, row.ID.Text)
	}

	if row.Name.Text != container.Names[0] {
		t.Errorf("Name widget does not contain the container name. Expected: %s, got: %s.", container.Names[0], row.Name.Text)
	}

	if row.CPU.Label != "-" {
		t.Errorf("CPU widget does not contain the default value. Expected: %s, got: %s.", "-", row.CPU.Label)
	}
	if row.Memory.Label != "-" {
		t.Errorf("Memory widget does not contain the default value. Expected: %s, got: %s.", "-", row.Memory.Label)
	}
	if row.Net.Text != "-" {
		t.Errorf("CPU widget does not contain the default value. Expected: %s, got: %s.", "-", row.Net.Text)
	}
	if row.Block.Text != "-" {
		t.Errorf("Block widget does not contain the default value. Expected: %s, got: %s.", "-", row.Block.Text)
	}
	if row.Pids.Text != "-" {
		t.Errorf("CPU widget does not contain the default value. Expected: %s, got: %s.", "-", row.Pids.Text)
	}
}
