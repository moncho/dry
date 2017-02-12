package docker

import (
	"strconv"
	"testing"

	dockerTypes "github.com/docker/docker/api/types"
)

var testContainers = createTestContainers(10)
var containerCount = len(testContainers)
var hundredContainers = createTestContainers(100)

func BenchmarkMemoryStoreContainerCreation(b *testing.B) {
	for i := 0; i < b.N; i++ {
		NewMemoryStoreWithContainers(hundredContainers)
	}
}

func BenchmarkMemoryStoreContainerListing(b *testing.B) {
	memStore := NewMemoryStoreWithContainers(hundredContainers)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		memStore.List()
	}
}

func TestMemoryStoreCreation(t *testing.T) {

	memStore := NewMemoryStoreWithContainers(testContainers)
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

func TestMemoryStoreAddContainer(t *testing.T) {
	memStore := NewMemoryStoreWithContainers(testContainers)
	memStore.Add(&dockerTypes.Container{
		ID: "11",
	})

	checkMemoryStore(memStore, containerCount+1, t)
}

func TestMemoryStoreAddDuplicatedContainer(t *testing.T) {
	containerCount := containerCount
	memStore := NewMemoryStoreWithContainers(testContainers)
	memStore.Add(&dockerTypes.Container{
		ID: "10",
	})

	memStore.Add(&dockerTypes.Container{
		ID: "10",
	})
	containerCount++
	containers := memStore.List()

	checkMemoryStore(memStore, containerCount, t)

	for i, c := range containers {
		t.Log(c.ID)
		if strconv.Itoa(i) != c.ID {
			t.Errorf("Expected ID %d, got %s", i, c.ID)
		}
	}
}

func TestFilter(t *testing.T) {
	memStore := NewMemoryStoreWithContainers(testContainers)
	filter := func(c *dockerTypes.Container) bool {
		return c.ID == "1"
	}
	filtered := memStore.Filter(filter)
	checkMemoryStore(memStore, containerCount, t)

	if len(filtered) != 1 || filtered[0].ID != "1" {
		t.Errorf("Filter did not work, got %v", filtered)
	}
}

func TestStoreFilterByName(t *testing.T) {
	memStore := NewMemoryStoreWithContainers(testContainers)
	filter := ContainerFilters.ByName("1")
	filtered := memStore.Filter(filter)
	checkMemoryStore(memStore, containerCount, t)

	if len(filtered) != 1 || filtered[0].Names[0] != "Name1" {
		t.Errorf("Filter did not work, got %v", filtered)
	}

	filtered = memStore.Filter(ContainerFilters.ByName("2"))
	if len(filtered) != 1 || filtered[0].Names[0] != "Name2" {
		t.Errorf("Filter did not work, got %v", filtered)
	}
}

func TestContainerAt(t *testing.T) {
	memStore := NewMemoryStoreWithContainers(testContainers)

	container := memStore.At(0)
	checkMemoryStore(memStore, containerCount, t)

	if container == nil || container.ID != "0" {
		t.Errorf("Container at did not work, got %v", container)
	}
}

func createTestContainers(numberOfContainers int) []*dockerTypes.Container {
	var containers []*dockerTypes.Container

	for i := 0; i < numberOfContainers; i++ {
		containers = append(containers, &dockerTypes.Container{
			ID:    strconv.Itoa(i),
			Names: []string{"Name" + strconv.Itoa(i)},
		})
	}

	return containers
}

func checkMemoryStore(memStore *ContainerStore, containerCount int, t *testing.T) {
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
