package swarm

import (
	"flag"
	"os"
	"path/filepath"
	"testing"

	"github.com/docker/docker/api/types/swarm"
	"github.com/moncho/dry/docker"
	"github.com/moncho/dry/mocks"
	"github.com/moncho/dry/ui"
)

var updateGoldenFiles = flag.Bool("updateGoldenFiles", false, "update .golden files")

var testService = &swarm.Service{
	Spec: swarm.ServiceSpec{
		TaskTemplate: swarm.TaskSpec{
			ContainerSpec: &swarm.ContainerSpec{
				Image: "bla",
			},
			Placement: &swarm.Placement{
				Constraints: []string{"constraint", "magic"},
			},
		},
		EndpointSpec: &swarm.EndpointSpec{},
	},
}

func TestServiceInfo(t *testing.T) {
	daemon := &mocks.SwarmDockerDaemon{}
	screen := &testScreen{
		dimensions: ui.Dimensions{Height: 14, Width: 100},
		cursor:     ui.NewCursor()}
	di := NewServiceInfoWidget(daemon, testService, screen)

	if di == nil {
		t.Error("Service info widget is nil")
	}
	content := di.Buffer()
	if content.Area.Dy() != di.GetHeight() {
		t.Error("Service info widget does not have the expected height")
	}

	if len(content.CellMap) == 0 {
		t.Errorf("Service info widget does not have the expected content: %v", content.CellMap)
	}
}

func Test_serviceInfo(t *testing.T) {
	type args struct {
		swarmClient docker.SwarmAPI
		name        string
		service     *swarm.Service
	}
	tests := []struct {
		name string
		args args
	}{
		{
			"service_info",
			args{
				&mocks.SwarmDockerDaemon{},
				"serviceName",
				testService,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			got := serviceInfo(tt.args.swarmClient, tt.args.name, tt.args.service)

			golden := filepath.Join("testdata", tt.name+".golden")
			if *updateGoldenFiles {
				os.WriteFile(golden, []byte(got), 0644)
			}
			expected, _ := os.ReadFile(golden)

			if got != string(expected) {
				t.Errorf("serviceInfo() = %v, want %v", got, expected)
			}
		})
	}
}
