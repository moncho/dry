package swarm

import (
	"testing"

	"github.com/docker/docker/api/types/swarm"
	"github.com/moncho/dry/mocks"
)

var expectedServiceInfo = `  <blue>Service Name:</>  <yellow>serviceName</>  <blue>Image:</>  <yellow>bla</>  
  <blue>Service Mode:</>  <yellow></>  <blue>Labels:</>       <yellow></>                  <blue>Created at:</>  <yellow>01 Jan 01 00:00 UTC</>  
  <blue>Replicas:</>      <yellow></>  <blue>Constraints:</>  <yellow>constraint,magic</>  <blue>Updated at:</>  <yellow>01 Jan 01 00:00 UTC</>  
  <blue>Networks:</>      <yellow></>  <blue>Ports:</>        <yellow></>                  
`
var testService = &swarm.Service{
	Spec: swarm.ServiceSpec{
		TaskTemplate: swarm.TaskSpec{
			ContainerSpec: swarm.ContainerSpec{
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
	di := NewServiceInfoWidget(daemon, testService, 0)

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

func TestSwarmDockerInfoContent(t *testing.T) {
	daemon := &mocks.SwarmDockerDaemon{}
	info := serviceInfo(daemon, "serviceName", testService)

	if info == "" {
		t.Error("Docker info is empty")
	}
	if info != expectedServiceInfo {
		t.Errorf("Service info output does not match. Expected: \n'%q'\n, got: \n'%q'", expectedServiceInfo, info)
	}

}
