package appui

import (
	"testing"

	"github.com/docker/docker/api/types"
	"github.com/moncho/dry/mocks"
	"github.com/moncho/dry/ui"
)

func TestImagesToShowSmallScreen(t *testing.T) {
	daemon := &mocks.DockerDaemonMock{}
	imagesLen := daemon.ImagesCount()
	if imagesLen != 5 {
		t.Errorf("Daemon has %d images, expected %d", imagesLen, 5)
	}

	cursor := ui.NewCursor()
	// to test scrolling all images but one fit on the screen
	screen := &testScreen{
		y1: imagesLen + widgetHeaderLength - 1, x1: 40,
		cursor: cursor}

	renderer := NewDockerImagesWidget(daemon.Images, screen)

	if err := renderer.Mount(); err != nil {
		t.Errorf("There was an error mounting the widget %v", err)
	}
	screen.Cursor().ScrollTo(0)
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

	screen := &testScreen{
		y1: 20, x1: 100,
		cursor: cursor}
	renderer := NewDockerImagesWidget(daemon.Images, screen)
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

	imageFunc := func() ([]types.ImageSummary, error) {
		return []types.ImageSummary{}, nil
	}
	renderer := NewDockerImagesWidget(imageFunc, &testScreen{})

	renderer.Mount()

	images := renderer.visibleRows()
	if len(images) != 0 {
		t.Error("Unexpected number of image rows, it should be 0")
	}
}
