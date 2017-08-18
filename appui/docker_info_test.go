package appui

import (
	"testing"

	"github.com/moncho/dry/mocks"
)

var expectedDockerInfoWithSwarm = `  <blue>Docker Host:</>         <yellow>dry.io</>    <blue>Docker Version:</>  <yellow>1.0</>           <blue>Hostname:</>  <yellow>test</>  <blue>Swarm:</>      <yellow>active</>   
  <blue>Cert Path:</>           <yellow></>          <blue>APIVersion:</>      <yellow>1.27</>          <blue>CPU:</>       <yellow>2</>     <blue>Node role:</>  <yellow>manager</>  
  <blue>Verify Certificate:</>  <yellow>false</>     <blue>OS/Arch/Kernel:</>  <yellow>dry/amd64/42</>  <blue>Memory:</>    <yellow>1KiB</>  <blue>Nodes:</>      <yellow>0</>        
`

var expectedDockerInfoNoSwarm = `  <blue>Docker Host:</>         <yellow>dry.io</>    <blue>Docker Version:</>  <yellow>1.0</>           <blue>Hostname:</>  <yellow>test</>  <blue>Swarm:</>  <yellow>inactive</>  
  <blue>Cert Path:</>           <yellow></>          <blue>APIVersion:</>      <yellow>1.27</>          <blue>CPU:</>       <yellow>2</>     
  <blue>Verify Certificate:</>  <yellow>false</>     <blue>OS/Arch/Kernel:</>  <yellow>dry/amd64/42</>  <blue>Memory:</>    <yellow>1KiB</>  
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
