package appui

import (
	"fmt"
	"strconv"
	"testing"

	"github.com/docker/docker/api/types"
	units "github.com/docker/go-units"
	"github.com/moncho/dry/docker"
	drytermui "github.com/moncho/dry/ui/termui"
)

func TestStatsRow(t *testing.T) {

	container := &docker.Container{
		Container:     types.Container{ID: "CID", Names: []string{"Name"}, Status: "Never worked"},
		ContainerJSON: types.ContainerJSON{}}

	row := NewContainerStatsRow(container, NewMonitorTableHeader())
	if row == nil {
		t.Error("Stats row was not created")
	}
	if row.container != container {
		t.Error("Stats row does not hold a reference to the container.")
	}

	if len(row.Columns) != 9 {
		t.Errorf("Stats row does not have the expected number of columns. Got: %d, expected 9.", len(row.Columns))
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
		t.Errorf("Net widget does not contain the default value. Expected: %s, got: %s.", "-", row.Net.Text)
	}
	if row.Block.Text != "-" {
		t.Errorf("Block widget does not contain the default value. Expected: %s, got: %s.", "-", row.Block.Text)
	}
	if row.Pids.Text != "0" {
		t.Errorf("Pids widget does not contain the default value. Expected: %s, got: %s.", "-", row.Pids.Text)
	}
	if row.Uptime.Text != "-" {
		t.Errorf("Uptime widget does not contain the default value. Expected: %s, got: %s.", "-", row.Uptime.Text)
	}
}

func TestContainerStatsRow_Update(t *testing.T) {
	type fields struct {
		Status *drytermui.ParColumn
		Name   *drytermui.ParColumn
		ID     *drytermui.ParColumn
		CPU    *drytermui.GaugeColumn
		Memory *drytermui.GaugeColumn
		Net    *drytermui.ParColumn
		Block  *drytermui.ParColumn
		Pids   *drytermui.ParColumn
		Uptime *drytermui.ParColumn
	}
	type args struct {
		container *docker.Container
		stats     *docker.Stats
	}
	tests := []struct {
		name   string
		fields fields
		args   args
	}{
		{
			"Update row, row has the expected values",
			fields{
				Status: drytermui.NewParColumn(""),
				Name:   drytermui.NewParColumn(""),
				ID:     drytermui.NewParColumn(""),
				CPU:    &drytermui.GaugeColumn{},
				Memory: &drytermui.GaugeColumn{},
				Net:    drytermui.NewParColumn(""),
				Block:  drytermui.NewParColumn(""),
				Pids:   drytermui.NewParColumn(""),
				Uptime: drytermui.NewParColumn(""),
			},
			args{
				container: &docker.Container{
					Container: types.Container{ID: "CID", Names: []string{"Name"}, Status: "Never worked"},
					ContainerJSON: types.ContainerJSON{
						ContainerJSONBase: &types.ContainerJSONBase{
							State: &types.ContainerState{
								Status: "Running I guess",
							}},
					}},
				stats: &docker.Stats{
					PidsCurrent:   3,
					NetworkRx:     1.15,
					NetworkTx:     2.34,
					CPUPercentage: 45.356,
				},
			},
		},
		{
			"Update row, no stats are passed, row does not change",
			fields{
				Status: drytermui.NewParColumn(""),
				Name:   drytermui.NewParColumn(""),
				ID:     drytermui.NewParColumn(""),
				CPU:    &drytermui.GaugeColumn{},
				Memory: &drytermui.GaugeColumn{},
				Net:    drytermui.NewParColumn(""),
				Block:  drytermui.NewParColumn(""),
				Pids:   drytermui.NewParColumn(""),
				Uptime: drytermui.NewParColumn(""),
			},
			args{
				container: &docker.Container{
					Container: types.Container{ID: "CID", Names: []string{"Name"}, Status: "Never worked"},
					ContainerJSON: types.ContainerJSON{
						ContainerJSONBase: &types.ContainerJSONBase{
							State: &types.ContainerState{
								Status: "Running I guess",
							}},
					}},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			row := &ContainerStatsRow{
				Status:    tt.fields.Status,
				Name:      tt.fields.Name,
				ID:        tt.fields.ID,
				CPU:       tt.fields.CPU,
				Memory:    tt.fields.Memory,
				Net:       tt.fields.Net,
				Block:     tt.fields.Block,
				Pids:      tt.fields.Pids,
				Uptime:    tt.fields.Uptime,
				container: tt.args.container,
			}
			stats := tt.args.stats
			row.Update(stats)
			if stats != nil {
				if row.Pids.Text != strconv.Itoa(int(stats.PidsCurrent)) {
					t.Errorf("Unexpected pids. Got %s, expected %d", row.Pids.Text, stats.PidsCurrent)
				}
				net := fmt.Sprintf("%s / %s", units.BytesSize(stats.NetworkRx), units.BytesSize(stats.NetworkTx))
				if row.Net.Text != net {
					t.Errorf("Unexpected network information. Got %s, expected %s", row.Net.Text, net)
				}

				cpu := fmt.Sprintf("%.2f%%", stats.CPUPercentage)
				if row.CPU.Label != cpu {
					t.Errorf("Unexpected CPU information. Got %s, expected %s", row.CPU.Label, cpu)
				}
			}
		})
	}
}
