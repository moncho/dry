package docker

import (
	"testing"

	dockerTypes "github.com/docker/docker/api/types"
)

func TestFilterByName(t *testing.T) {

	c := &dockerTypes.Container{
		Names: []string{"bla"},
	}

	filter := ContainerFilters.ByName("bla")
	if !filter(c) {
		t.Error("Filter by name is filtering out when it should not")
	}

	c = &dockerTypes.Container{
		Names: []string{"123"},
	}
	if filter(c) {
		t.Error("Filter by name is not filtering")
	}

	c = &dockerTypes.Container{
		Names: []string{"123bla123"},
	}
	if !filter(c) {
		t.Error("Filter by name is filtering out when it should not")

	}

}

func TestFilterByID(t *testing.T) {

	c := &dockerTypes.Container{
		ID: "bla",
	}

	filter := ContainerFilters.ByID("bla")
	if !filter(c) {
		t.Error("Filter by ID is filtering out when it should not")
	}

	c = &dockerTypes.Container{
		ID: "123",
	}
	if filter(c) {
		t.Error("Filter by ID is not filtering")
	}

	c = &dockerTypes.Container{
		ID: "123bla123",
	}
	if !filter(c) {
		t.Error("Filter by ID is filtering out when it should not")

	}

}
