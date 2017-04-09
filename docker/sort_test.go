package docker

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestSortById(t *testing.T) {
	c, error := containersToSort()
	if error != nil {
		t.Error("Could not create container list")
	}
	SortContainers(c, SortByContainerID)
	if c[0].ID != "6dfafdbc3a40" {
		t.Errorf("Sorting by CID did not work. Sorted to: %s", strings.Join(containersAsString(c), ","))

	} else if c[1].ID != "7dfafdbc3a40" {
		t.Errorf("Sorting by CID did not work. Sorted to: %s", strings.Join(containersAsString(c), ","))

	}
}

func TestSortByName(t *testing.T) {
	c, error := containersToSort()
	if error != nil {
		t.Error("Could not create container list")
	}
	SortContainers(c, SortByName)
	if c[0].ID != "6dfafdbc3a40" {
		t.Errorf("Sorting by Name did not work. Sorted to: %s", strings.Join(containersAsString(c), ","))

	} else if c[1].ID != "7dfafdbc3a40" {
		t.Errorf("Sorting by Name did not work. Sorted to: %s", strings.Join(containersAsString(c), ","))

	}
}

func TestSortByImage(t *testing.T) {
	c, error := containersToSort()
	if error != nil {
		t.Error("Could not create container list")
	}
	SortContainers(c, SortByImage)
	if c[0].ID != "6dfafdbc3a40" {
		t.Errorf("Sorting by image did not work. Sorted to: %s", strings.Join(containersAsString(c), ","))

	} else if c[1].ID != "7dfafdbc3a40" {
		t.Errorf("Sorting by image did not work. Sorted to: %s", strings.Join(containersAsString(c), ","))
	}
}

func TestSortByStatus(t *testing.T) {
	c, error := containersToSort()
	if error != nil {
		t.Error("Could not create container list")
	}
	SortContainers(c, SortByStatus)
	if c[0].ID != "6dfafdbc3a40" {
		t.Errorf("Sorting by status did not work. Sorted to: %s", strings.Join(containersAsString(c), ","))

	} else if c[1].ID != "7dfafdbc3a40" {
		t.Errorf("Sorting by status did not work. Sorted to: %s", strings.Join(containersAsString(c), ","))
	}
}

func containersToSort() ([]*Container, error) {
	jsonContainers := `[
     {
             "Id": "8dfafdbc3a40",
             "Image": "base:latest",
             "Command": "echo 1",
             "Created": 1367854155,
						 "Status": "Up 8 hours",
						 "Names":["8dfafdbc3a40"]
     },
     {
             "Id": "7dfafdbc3a40",
             "Image": "base:latest",
             "Command": "echo 222222",
             "Created": 1367854155,
             "Status": "Exit 0",
						 "Names":["7dfafdbc3a40"]
     },
     {
             "Id": "6dfafdbc3a40",
             "Image": "base:latest",
             "Command": "echo 3333333333333333",
             "Created": 1367854154,
						 "Status": "Exit 0",
						 "Names": ["6dfafdbc3a40"]
     }
]`
	var containers []*Container
	err := json.Unmarshal([]byte(jsonContainers), &containers)
	return containers, err
}
func containersAsString(containers []*Container) []string {
	result := make([]string, len(containers))
	for i, c := range containers {
		result[i] = c.ID
	}
	return result
}
