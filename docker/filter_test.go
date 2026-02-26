package docker

import (
	"testing"

	"github.com/docker/docker/api/types/container"
)

func TestFilterByName(t *testing.T) {

	c := &Container{
		Summary: container.Summary{Names: []string{"bla"}},
	}

	filter := ContainerFilters.ByName("bla")
	if !filter(c) {
		t.Error("Filter by name is filtering out when it should not")
	}

	c = &Container{
		Summary: container.Summary{Names: []string{"123"}},
	}
	if filter(c) {
		t.Error("Filter by name is not filtering")
	}

	c = &Container{
		Summary: container.Summary{Names: []string{"123bla123"}},
	}
	if !filter(c) {
		t.Error("Filter by name is filtering out when it should not")

	}

}

func TestFilterByID(t *testing.T) {

	c := &Container{
		Summary: container.Summary{ID: "bla"},
	}

	filter := ContainerFilters.ByID("bla")
	if !filter(c) {
		t.Error("Filter by ID is filtering out when it should not")
	}

	c = &Container{
		Summary: container.Summary{ID: "123"},
	}
	if filter(c) {
		t.Error("Filter by ID is not filtering")
	}

	c = &Container{
		Summary: container.Summary{ID: "123bla123"},
	}
	if !filter(c) {
		t.Error("Filter by ID is filtering out when it should not")

	}

}
