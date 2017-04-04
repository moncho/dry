package appui

import (
	"testing"

	"github.com/moncho/dry/mocks"
)

var expectedDockerInfoWithSwarm = `  <blue>Docker Host:</>         <yellow>dry.io</>    <blue>Docker Version:</>  <yellow>1.0</>           <blue>Swarm:</>       <yellow>active</>       <blue>Is Manager:</>  <yellow>false</>  
  <blue>Cert Path:</>           <yellow></>          <blue>APIVersion:</>      <yellow>1.27</>          <blue>Cluster ID:</>  <yellow>MyClusterID</>  
  <blue>Verify Certificate:</>  <yellow>false</>     <blue>OS/Arch/Kernel:</>  <yellow>dry/amd64/42</>  <blue>Node ID:</>     <yellow>ThisNodeID</>   
`

var expectedDockerInfoNoSwarm = `  <blue>Docker Host:</>         <yellow>dry.io</>    <blue>Docker Version:</>  <yellow>1.0</>           <blue>Swarm:</>  <yellow>inactive</>  
  <blue>Cert Path:</>           <yellow></>          <blue>APIVersion:</>      <yellow>1.27</>          
  <blue>Verify Certificate:</>  <yellow>false</>     <blue>OS/Arch/Kernel:</>  <yellow>dry/amd64/42</>  
`

func TestDockerInfo(t *testing.T) {
	daemon := &mocks.DockerDaemonMock{}
	di := NewDockerInfo(daemon)

	if di == nil {
		t.Error("Docker info widget is nil")
	}
	content := di.Buffer()
	if content.Area.Dy() != di.GetHeight() {
		t.Error("Docker info widget does not have the expected height")
	}

	if len(content.CellMap) == 0 {
		t.Errorf("Docker info widget does not have the expected content: %v", content.CellMap)
	}
}

func TestNoSwarmDockerInfoContent(t *testing.T) {
	daemon := &mocks.DockerDaemonMock{}
	di := dockerInfo(daemon)

	if di == "" {
		t.Error("Docker info is empty")
	}

	if di != expectedDockerInfoNoSwarm {
		t.Errorf("Docker info output does not match. Expected: \n'%q'\n, got: \n'%q'", expectedDockerInfoNoSwarm, di)
	}
}

func TestSwarmDockerInfoContent(t *testing.T) {
	daemon := &mocks.SwarmDockerDaemon{}
	di := dockerInfo(daemon)

	if di == "" {
		t.Error("Docker info is empty")
	}

	if di != expectedDockerInfoWithSwarm {
		t.Errorf("Docker info output does not match. Expected: \n'%q'\n, got: \n'%q'", expectedDockerInfoWithSwarm, di)
	}
}
