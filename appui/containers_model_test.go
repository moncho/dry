package appui

import (
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/moncho/dry/docker"
	dockerTypes "github.com/docker/docker/api/types"
)

func makeTestContainers(n int) []*docker.Container {
	containers := make([]*docker.Container, n)
	for i := range n {
		containers[i] = &docker.Container{
			Container: dockerTypes.Container{
				ID:    "abcdef" + string(rune('0'+i)) + "12345",
				Names: []string{"container-" + string(rune('a'+i))},
				Image: "image-" + string(rune('a'+i)),
			},
		}
	}
	return containers
}

func TestContainersModel_SortModeCycleSyncsTableColumn(t *testing.T) {
	m := NewContainersModel()
	m.SetSize(120, 30)
	m.SetContainers(makeTestContainers(3))

	// Columns: [0]=indicator, [1]=CONTAINER, [2]=IMAGE, [3]=COMMAND,
	//          [4]=STATUS, [5]=PORTS, [6]=NAMES
	//
	// Docker SortMode: NoSort=0, SortByContainerID=1, SortByImage=2,
	//                  SortByStatus=3, SortByName=4

	// Initial sort mode is SortByContainerID (set in NewContainersModel)
	if m.sortMode != docker.SortByContainerID {
		t.Fatalf("expected initial sortMode SortByContainerID, got %d", m.sortMode)
	}

	// Press F1 to advance: SortByContainerID → SortByImage
	m, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyF1})
	if m.sortMode != docker.SortByImage {
		t.Fatalf("expected SortByImage after 1st F1, got %d", m.sortMode)
	}
	if m.table.SortField() != 2 {
		t.Fatalf("expected table sort field 2 (IMAGE), got %d", m.table.SortField())
	}

	// F1 again: SortByImage → SortByStatus
	m, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyF1})
	if m.sortMode != docker.SortByStatus {
		t.Fatalf("expected SortByStatus after 2nd F1, got %d", m.sortMode)
	}
	if m.table.SortField() != 4 {
		t.Fatalf("expected table sort field 4 (STATUS), got %d", m.table.SortField())
	}

	// F1 again: SortByStatus → SortByName
	m, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyF1})
	if m.sortMode != docker.SortByName {
		t.Fatalf("expected SortByName after 3rd F1, got %d", m.sortMode)
	}
	if m.table.SortField() != 6 {
		t.Fatalf("expected table sort field 6 (NAMES), got %d", m.table.SortField())
	}

	// F1 again: SortByName → NoSort (wraps)
	m, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyF1})
	if m.sortMode != docker.NoSort {
		t.Fatalf("expected NoSort after 4th F1, got %d", m.sortMode)
	}
	if m.table.SortField() != -1 {
		t.Fatalf("expected table sort field -1 (no indicator) for NoSort, got %d", m.table.SortField())
	}

	// F1 again: NoSort → SortByContainerID (wraps back)
	m, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyF1})
	if m.sortMode != docker.SortByContainerID {
		t.Fatalf("expected SortByContainerID after 5th F1, got %d", m.sortMode)
	}
	if m.table.SortField() != 1 {
		t.Fatalf("expected table sort field 1 (CONTAINER), got %d", m.table.SortField())
	}
}

func TestContainersModel_SetAndSelect(t *testing.T) {
	m := NewContainersModel()
	m.SetSize(120, 30)

	containers := makeTestContainers(5)
	m.SetContainers(containers)

	sel := m.SelectedContainer()
	if sel == nil {
		t.Fatal("expected non-nil selected container")
	}
	if sel.ID != containers[0].ID {
		t.Fatalf("expected first container ID %q, got %q", containers[0].ID, sel.ID)
	}

	// Navigate down
	m, _ = m.Update(tea.KeyPressMsg{Code: 'j'})
	sel = m.SelectedContainer()
	if sel.ID != containers[1].ID {
		t.Fatalf("expected second container after j, got %q", sel.ID)
	}
}

func TestContainersModel_EmptySelected(t *testing.T) {
	m := NewContainersModel()
	m.SetSize(120, 30)

	if m.SelectedContainer() != nil {
		t.Fatal("expected nil selected container for empty model")
	}
}

