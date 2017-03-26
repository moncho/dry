package docker

import (
	"strconv"
	"testing"

	dockerTypes "github.com/docker/docker/api/types"
	"github.com/moncho/dry/docker/mock"
)

var testContainers = createTestContainers(10)
var containerCount = len(testContainers)
var hundredContainers = createTestContainers(100)

func BenchmarkMemoryStoreContainerCreation(b *testing.B) {
	c := mock.ContainerAPIClientMock{Containers: hundredContainers}
	for i := 0; i < b.N; i++ {
		NewDockerContainerStore(c)
	}
}

func BenchmarkMemoryStoreContainerListing(b *testing.B) {
	c := mock.ContainerAPIClientMock{Containers: hundredContainers}
	memStore, _ := NewDockerContainerStore(c)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		memStore.List()
	}
}

func TestMemoryStoreCreation(t *testing.T) {
	c := mock.ContainerAPIClientMock{}
	memStore, _ := NewDockerContainerStore(c)
	if memStore == nil {
		t.Error("Memstore is nil")
	}
	if memStore.Size() != containerCount {
		t.Errorf("Memstore does not contain the expected the number of elements, expected: %d, got: %d", containerCount, memStore.Size())
	}

	if memStore.Get("1") == nil {
		t.Error("Memstore does not contain expected container")
	}

	if memStore.Get("11") != nil {
		t.Error("Memstore contains an unexpected container")
	}

	checkMemoryStore(memStore, containerCount, t)
}

func createTestContainers(numberOfContainers int) []dockerTypes.Container {
	var containers []dockerTypes.Container

	for i := 0; i < numberOfContainers; i++ {
		containers = append(containers, dockerTypes.Container{
			ID:    strconv.Itoa(i),
			Names: []string{"Name" + strconv.Itoa(i)},
		})
	}

	return containers
}

func checkMemoryStore(memStore *DockerContainerStore, containerCount int, t *testing.T) {
	containers := memStore.List()
	if memStore.Size() != containerCount {
		t.Errorf("Memstore does not contain the expected the number of elements, expected: %d, got: %d", containerCount, memStore.Size())
	}

	if len(containers) != containerCount {
		t.Errorf("Element list from memstore does not contain the expected the number of elements, expected: %d, got: %d", containerCount, len(containers))
	}

	if len(memStore.c) != len(memStore.s) {
		t.Error("Memstore internal state is incorrect")
	}
}
