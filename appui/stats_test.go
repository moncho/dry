package appui

import "testing"
import drydocker "github.com/moncho/dry/docker"

func TestNewDockerStatsBufferer(t *testing.T) {
	stats := &drydocker.Stats{
		ProcessList: nil,
	}

	bufferer := NewDockerStatsBufferer(stats, 0, 0, 0, 0)
	if len(bufferer) != 2 {
		t.Error("Unexpected bufferer length")
	}
}
