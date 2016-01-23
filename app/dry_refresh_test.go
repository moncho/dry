package app

import (
	"sync"
	"testing"
	"time"

	"github.com/moncho/dry/docker"
	"github.com/moncho/dry/mocks"
)

func TestDryRefresh(t *testing.T) {
	dry := newDryForTest()
	firstRefreshMark := dry.lastRefresh
	if !firstRefreshMark.Before(time.Now()) {
		t.Error("dry initial refresh mark is not correct")
	}
	dry.Refresh()
	lastRefresh := dry.lastRefresh
	if !firstRefreshMark.Before(lastRefresh) {
		t.Error("dry was not refreshed")
	}
	//This should not refresh since dry was just refreshed
	dry.tryRefresh()
	if !lastRefresh.Equal(dry.lastRefresh) {
		t.Error("dry was refreshed when it should not")
	}
	//but refresh always works
	dry.Refresh()
	if !lastRefresh.Before(dry.lastRefresh) {
		t.Error("dry was not refreshed when it should had been")
	}

}

func newDryForTest() *Dry {
	dry := &Dry{}
	dry.State = &State{
		changed:              true,
		Paused:               false,
		showingAllContainers: false,
		SortMode:             docker.SortByContainerID,
		viewMode:             Main,
	}
	dry.dockerDaemon = new(mocks.ContainerDaemonMock)
	dry.refreshTimeMutex = &sync.Mutex{}
	dry.resetTimer()
	return dry
}
