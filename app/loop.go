package app

import (
	"sync"
	"time"

	"github.com/gdamore/tcell"
	"github.com/moncho/dry/ui"
	log "github.com/sirupsen/logrus"
)

// RenderContext encapsulates all state needed for the render loop
// This replaces the global function variables pattern for better testability
type RenderContext struct {
	dry         *Dry
	screen      *ui.Screen
	renderChan  chan struct{}
	closingLock sync.RWMutex
	wg          sync.WaitGroup
}

// Global context instance - set during RenderLoop initialization
var renderCtx *RenderContext

// Legacy global function variables - now delegate to renderCtx
// These are kept for backward compatibility during transition
var refreshScreen func() error
var refreshIfView func(v viewMode) error
var widgets *widgetRegistry

// newRenderContext creates a new render context
func newRenderContext(dry *Dry) *RenderContext {
	return &RenderContext{
		dry:        dry,
		screen:     dry.screen,
		renderChan: make(chan struct{}, 1), // Buffered to prevent blocking
	}
}

// refreshScreen queues a render request (non-blocking)
func (rc *RenderContext) refreshScreen() error {
	rc.closingLock.RLock()
	defer rc.closingLock.RUnlock()

	select {
	case rc.renderChan <- struct{}{}:
		// Successfully queued render
	default:
		// Render already pending, skip this request
		// This automatically coalesces multiple rapid refresh calls
	}
	return nil
}

// refreshIfView conditionally refreshes if we're in the specified view
func (rc *RenderContext) refreshIfView(v viewMode) error {
	if v == rc.dry.viewMode() {
		return rc.refreshScreen()
	}
	return nil
}

// startRenderLoop starts the background goroutine that performs rendering
func (rc *RenderContext) startRenderLoop() {
	rc.wg.Add(1)
	go func() {
		defer rc.wg.Done()

		for range rc.renderChan {
			if !rc.screen.Closing() {
				// Panic recovery per render
				func() {
					rc.screen.Clear()
					render(rc.dry, rc.screen)
				}()
			}
		}
	}()
}

// startMessageLoop starts the background goroutine that handles dry messages
func (rc *RenderContext) startMessageLoop() {
	dryOutputChan := rc.dry.OutputChannel()
	rc.wg.Add(1)
	go func() {
		defer rc.wg.Done()
		statusBar := widgets.MessageBar
		for dryMessage := range dryOutputChan {
			statusBar.Message(dryMessage, 10*time.Second)
			statusBar.Render()
		}
	}()
}

// handleEvent processes a single UI event
// Returns true if the event loop should exit
func (rc *RenderContext) handleEvent(event tcell.Event, handler *eventHandler) bool {
	switch ev := event.(type) {
	case *tcell.EventInterrupt:
		return true

	case *tcell.EventKey:
		// Ctrl+C or Q breaks the loop (and exits dry) no matter what
		if ev.Key() == tcell.KeyCtrlC || ev.Rune() == 'Q' {
			return true
		}
		(*handler).handle(ev, func(eh eventHandler) {
			*handler = eh
		})

	case *tcell.EventResize:
		rc.screen.Resize()
		// Refresh screen after resize
		rc.refreshScreen()
	}

	return false
}

// cleanup performs graceful shutdown
func (rc *RenderContext) cleanup() {
	log.Debug("Shutting down render loop")

	// Make refreshScreen a noop BEFORE closing channels to prevent panics
	rc.closingLock.Lock()
	refreshScreen = func() error {
		return nil
	}
	refreshIfView = func(v viewMode) error {
		return nil
	}
	rc.closingLock.Unlock()

	// Close the render channel to signal render goroutine to exit
	close(rc.renderChan)

	// Close dry to signal message loop goroutine to exit
	rc.dry.Close()

	// Wait for all goroutines to finish
	rc.wg.Wait()

	log.Debug("Render loop shutdown complete")
}

// RenderLoop runs dry
func RenderLoop(dry *Dry) {
	if ok, err := dry.Ok(); !ok {
		log.Error(err.Error())
		return
	}

	// Create render context
	ctx := newRenderContext(dry)
	renderCtx = ctx // Set global for use by other packages

	// Set up global function delegates for backward compatibility
	refreshScreen = ctx.refreshScreen
	refreshIfView = ctx.refreshIfView

	// Get terminal event channel
	termuiEvents, done := ui.EventChannel()

	// Start background goroutines
	ctx.startRenderLoop()
	ctx.startMessageLoop()

	// Initial render
	ctx.refreshScreen()

	// Main event loop
	handler := viewsToHandlers[dry.viewMode()]
loop:
	for event := range termuiEvents {
		if ctx.handleEvent(event, &handler) {
			break loop
		}
	}

	log.Debug("Exiting render loop")

	// Close terminal event channel
	close(done)

	// Clean up and wait for goroutines
	ctx.cleanup()
}
