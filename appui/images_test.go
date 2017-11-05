package appui

import (
	"testing"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/image"

	"github.com/moncho/dry/mocks"
	"github.com/moncho/dry/ui"
)

func TestImagesToShowSmallScreen(t *testing.T) {
	_ = "breakpoint"
	daemon := &mocks.DockerDaemonMock{}
	imagesLen := daemon.ImagesCount()
	if imagesLen != 5 {
		t.Errorf("Daemon has %d images, expected %d", imagesLen, 5)
	}

	cursor := ui.NewCursor()
	ui.ActiveScreen = &ui.Screen{
		Dimensions: &ui.Dimensions{Height: 15, Width: 100},
		Cursor:     cursor}

	renderer := NewDockerImagesWidget(daemon, 0)

	if err := renderer.Mount(); err != nil {
		t.Errorf("There was an error mounting the widget %v", err)
	}
	ui.ActiveScreen.Cursor.ScrollTo(0)
	renderer.prepareForRendering()

	images := renderer.visibleRows()
	if len(images) != 4 {
		t.Errorf("Images renderer is showing %d images, expected %d", len(images), 4)
	}

	if images[0].ID.Text != "8dfafdbc3a40" {
		t.Errorf("First image rendered is %s, expected %s. Cursor: %d", images[0].ID.Text, "8dfafdbc3a40", cursor.Position())
	}

	if images[2].ID.Text != "26380e1ca356" {
		t.Errorf("Last image rendered is %s, expected %s. Cursor: %d", images[2].ID.Text, "26380e1ca356", cursor.Position())
	}
	cursor.ScrollTo(4)
	renderer.prepareForRendering()
	images = renderer.visibleRows()
	if len(images) != 4 {
		t.Errorf("Images renderer is showing %d images, expected %d", len(images), 4)
	}
	if images[0].ID.Text != "541a0f4efc6f" {
		t.Errorf("First image rendered is %s, expected %s. Cursor: %d", images[0].ID.Text, "541a0f4efc6f", cursor.Position())
	}

	if images[3].ID.Text != "a3d6e836e86a" {
		t.Errorf("Last image rendered is %s, expected %s. Cursor: %d", images[3].ID.Text, "a3d6e836e86a", cursor.Position())
	}
}

func TestImagesToShow(t *testing.T) {
	_ = "breakpoint"
	daemon := &mocks.DockerDaemonMock{}
	imagesLen := daemon.ImagesCount()
	if imagesLen != 5 {
		t.Errorf("Daemon has %d images, expected %d", imagesLen, 3)
	}

	cursor := ui.NewCursor()

	ui.ActiveScreen = &ui.Screen{Dimensions: &ui.Dimensions{Height: 20, Width: 100},
		Cursor: cursor}
	renderer := NewDockerImagesWidget(daemon, 0)
	if err := renderer.Mount(); err != nil {
		t.Errorf("There was an error mounting the widget %v", err)
	}

	renderer.prepareForRendering()
	images := renderer.visibleRows()
	if len(images) != 5 {
		t.Errorf("Images renderer is showing %d images, expected %d", len(images), 5)
	}
	if images[0].ID.Text != "8dfafdbc3a40" {
		t.Errorf("First image rendered is %s, expected %s", images[0].ID.Text, "8dfafdbc3a40")
	}
	cursor.ScrollTo(3)
	images = renderer.visibleRows()
	if len(images) != 5 {
		t.Errorf("Images renderer is showing %d images, expected %d", len(images), 5)
	}
	if images[0].ID.Text != "8dfafdbc3a40" {
		t.Errorf("First image rendered is %s, expected %s", images[0].ID.Text, "8dfafdbc3a40")
	}

	if images[4].ID.Text != "a3d6e836e86a" {
		t.Errorf("Last image rendered is %s, expected %s", images[4].ID.Text, "a3d6e836e86a")
	}
}

func TestImagesToShowNoImages(t *testing.T) {
	renderer := NewDockerImagesWidget(noopImageAPI{}, 0)

	renderer.Mount()

	images := renderer.visibleRows()
	if len(images) != 0 {
		t.Error("Unexpected number of image rows, it should be 0")
	}

}

type noopImageAPI struct {
}

func (i noopImageAPI) History(id string) ([]image.HistoryResponseItem, error) {
	return []image.HistoryResponseItem{}, nil
}

func (i noopImageAPI) ImageByID(id string) (types.ImageSummary, error) {
	return types.ImageSummary{}, nil

}
func (i noopImageAPI) Images() ([]types.ImageSummary, error) {
	return []types.ImageSummary{}, nil
}
func (i noopImageAPI) ImagesCount() int {
	return 0
}
func (i noopImageAPI) RunImage(image types.ImageSummary, command string) error {
	return nil
}
