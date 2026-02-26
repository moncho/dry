package docker

import (
	"strconv"
	"testing"

	"github.com/docker/docker/api/types/container"
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

	checkMemoryStore(memStore.(*inMemoryContainerStore), containerCount, t)
}

func createTestContainers(numberOfContainers int) []container.Summary {
	var containers []container.Summary

	for i := 0; i < numberOfContainers; i++ {
		containers = append(containers, container.Summary{
			ID:    strconv.Itoa(i),
			Names: []string{"Name" + strconv.Itoa(i)},
		})
	}

	return containers
}

func TestMemoryStoreAddReplacesExisting(t *testing.T) {
	c := mock.ContainerAPIClientMock{}
	memStore, _ := NewDockerContainerStore(c)
	store := memStore.(*inMemoryContainerStore)

	original := store.Get("1")
	if original == nil {
		t.Fatal("expected container '1' to exist")
	}

	// Add a replacement container with the same ID but different name
	replacement := &Container{
		Summary: container.Summary{
			ID:    "1",
			Names: []string{"Replaced"},
		},
	}
	store.add(replacement)

	// Size should remain the same
	if store.Size() != containerCount {
		t.Errorf("expected %d containers, got %d", containerCount, store.Size())
	}
	// Map and slice must agree
	if len(store.c) != len(store.s) {
		t.Errorf("slice/map mismatch: slice=%d map=%d", len(store.c), len(store.s))
	}
	// Get should return the replacement
	got := store.Get("1")
	if got.Names[0] != "Replaced" {
		t.Errorf("expected name 'Replaced', got %q", got.Names[0])
	}
	// List should also contain the replacement (not the stale original)
	found := false
	for _, ct := range store.List() {
		if ct.ID == "1" {
			if ct.Names[0] != "Replaced" {
				t.Errorf("List() returned stale container: name=%q", ct.Names[0])
			}
			found = true
			break
		}
	}
	if !found {
		t.Error("container '1' not found in List()")
	}
}

func checkMemoryStore(memStore *inMemoryContainerStore, containerCount int, t *testing.T) {
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
