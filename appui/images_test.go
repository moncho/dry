package appui

import (
	"testing"

	"github.com/moncho/dry/docker"
	"github.com/moncho/dry/mocks"
	"github.com/moncho/dry/ui"
)

func TestImagesToShowSmallScreen(t *testing.T) {
	_ = "breakpoint"
	daemon := &mocks.ContainerDaemonMock{}
	imagesLen := daemon.ImagesCount()
	if imagesLen != 5 {
		t.Errorf("Daemon has %d images, expected %d", imagesLen, 3)
	}

	cursor := &ui.Cursor{}
	renderer := NewDockerImagesRenderer(daemon, 14)
	imagesFromDaemon, _ := daemon.Images()
	renderer.PrepareForRender(NewDockerImageRenderData(
		imagesFromDaemon, cursor.Position(), docker.NoSortImages))

	images := renderer.imagesToShow()
	if len(images) != 3 {
		t.Errorf("Images renderer is showing %d images, expected %d", len(images), 3)
	}
	if images[0].ID != "8dfafdbc3a40" {
		t.Errorf("First image rendered is %s, expected %s. Cursor: %d", images[0].ID, "8dfafdbc3a40", cursor.Position())
	}

	if images[2].ID != "26380e1ca356" {
		t.Errorf("Last image rendered is %s, expected %s. Cursor: %d", images[2].ID, "26380e1ca356", cursor.Position())
	}
	cursor.ScrollTo(3)
	renderer.PrepareForRender(NewDockerImageRenderData(
		imagesFromDaemon, cursor.Position(), docker.NoSortImages))
	images = renderer.imagesToShow()
	if len(images) != 3 {
		t.Errorf("Images renderer is showing %d images, expected %d", len(images), 3)
	}
	if images[0].ID != "541a0f4efc6f" {
		t.Errorf("First image rendered is %s, expected %s. Cursor: %d", images[0].ID, "541a0f4efc6f", cursor.Position())
	}

	if images[2].ID != "a3d6e836e86a" {
		t.Errorf("Last image rendered is %s, expected %s. Cursor: %d", images[2].ID, "a3d6e836e86a", cursor.Position())
	}
}

func TestImagesToShow(t *testing.T) {
	_ = "breakpoint"
	daemon := &mocks.ContainerDaemonMock{}
	imagesLen := daemon.ImagesCount()
	if imagesLen != 5 {
		t.Errorf("Daemon has %d images, expected %d", imagesLen, 3)
	}

	cursor := &ui.Cursor{}
	renderer := NewDockerImagesRenderer(daemon, 20)

	imagesFromDaemon, _ := daemon.Images()
	renderer.PrepareForRender(NewDockerImageRenderData(
		imagesFromDaemon, cursor.Position(), docker.NoSortImages))

	images := renderer.imagesToShow()
	if len(images) != 5 {
		t.Errorf("Images renderer is showing %d images, expected %d", len(images), 5)
	}
	cursor.ScrollTo(3)
	images = renderer.imagesToShow()
	if len(images) != 5 {
		t.Errorf("Images renderer is showing %d images, expected %d", len(images), 5)
	}
	if images[0].ID != "8dfafdbc3a40" {
		t.Errorf("First image rendered is %s, expected %s", images[0].ID, "8dfafdbc3a40")
	}

	if images[4].ID != "03b4557ad7b9" {
		t.Errorf("Last image rendered is %s, expected %s", images[4].ID, "03b4557ad7b9")
	}
}
