package main

import (
	"bufio"
	"io"
	"os"
	"time"

	"net/http"

	log "github.com/Sirupsen/logrus"
	"github.com/jessevdk/go-flags"
	"github.com/moncho/dry/app"
	"github.com/moncho/dry/ui"
	"github.com/nsf/termbox-go"
)
import _ "net/http/pprof"

// command line flags variable
var opts struct {
	// enable profiling
	Profile bool `short:"p" long:"profile" description:"Enable profiling"`
}

//-----------------------------------------------------------------------------
func mainLoop(dry *app.Dry, screen *ui.Screen) {

	if ok, _ := dry.Ok(); !ok {
		return
	}

	keyboardQueue := make(chan termbox.Event)
	timestampQueue := time.NewTicker(1 * time.Second)

	viewClosed := make(chan bool)
	keyboardQueueForView := make(chan termbox.Event)

	defer close(keyboardQueue)
	defer close(viewClosed)
	defer close(keyboardQueueForView)

	go func() {
		for {
			keyboardQueue <- termbox.PollEvent()
		}
	}()

	app.Render(dry, screen)
	//belongs outside the loop
	var streamMode = false
	var viewMode = false
loop:
	for {
		//Used for refresh-forcing events outside dry
		var refresh = false
		select {
		case <-timestampQueue.C:
			if !streamMode {
				timestamp := time.Now().Format(`3:04:05pm PST`)
				screen.RenderLine(0, 0, `<right><white>`+timestamp+`</></right>`)
				screen.Flush()
			}
		case <-viewClosed:
			viewMode = false
			streamMode = false
			dry.ShowDockerHostInfo()
		case event := <-keyboardQueue:
			switch event.Type {
			case termbox.EventKey:
				if !dry.State.ShowingHelp && !viewMode {
					if event.Key == termbox.KeyEsc || event.Ch == 'q' || event.Ch == 'Q' {
						break loop
					} else if event.Key == termbox.KeyArrowUp { //cursor up
						screen.MoveCursorUp()
						refresh = true
					} else if event.Key == termbox.KeyArrowDown { // cursor down
						screen.MoveCursorDown()
						refresh = true
					} else if event.Key == termbox.KeyF1 { //sort
						dry.Sort()
						dry.Refresh()
					} else if event.Key == termbox.KeyF2 { //show all containers
						dry.ToggleShowAllContainers()
					} else if event.Key == termbox.KeyF5 { // refresh
						dry.Refresh()
					} else if event.Ch == '?' || event.Ch == 'h' || event.Ch == 'H' { //help
						dry.ShowHelp()
					} else if event.Ch == 'e' || event.Ch == 'E' { //remove
						dry.Rm(screen.CursorPosition())
					} else if event.Ch == 'k' || event.Ch == 'K' { //kill
						dry.Kill(screen.CursorPosition())
					} else if event.Ch == 'l' || event.Ch == 'L' { //logs
						if logs, err := dry.Logs(screen.CursorPosition()); err == nil {
							viewMode = true
							streamMode = true
							stream(screen, logs)
						}
					} else if event.Ch == 'r' || event.Ch == 'R' { //start
						dry.StartContainer(screen.CursorPosition())
					} else if event.Ch == 's' || event.Ch == 'S' { //stats
						dry.Stats(screen.CursorPosition())
						viewMode = true
						go overlayView(dry, screen, keyboardQueueForView, viewClosed)
					} else if event.Ch == 't' || event.Ch == 'T' { //stop
						dry.StopContainer(screen.CursorPosition())
					} else if event.Key == termbox.KeyEnter { //inspect
						dry.Inspect(screen.CursorPosition())
						viewMode = true
						go overlayView(dry, screen, keyboardQueueForView, viewClosed)

					}
				} else if viewMode {
					//The view handles the event
					keyboardQueueForView <- event
				} else if dry.State.ShowingHelp {
					dry.ShowDockerHostInfo()
				}

			case termbox.EventResize:
				screen.Resize()
				refresh = true
			}
		}
		if !streamMode && (refresh || dry.Changed()) {
			screen.Clear()
			app.Render(dry, screen)
		}
	}

	log.Info("something broke the loop")
}

func stream(screen *ui.Screen, stream io.Reader) {
	log.Info("[stream] Showing stream")
	screen.Clear()
	screen.Sync()
	go func() {
		//io.Copy(os.Stdout, stream)
		scanner := bufio.NewScanner(stream)

		screen.RenderLine(0, 0, "ESC to go back")
		for i := 1; scanner.Scan(); i++ {
			screen.RenderLine(0, i, scanner.Text())
		}
		if err := scanner.Err(); err != nil {
			screen.RenderLine(0, 0, "There was an error with the scanner")
		}
	}()

	log.Debugf("[stream] End of stdout")
}

func overlayView(dry *app.Dry, screen *ui.Screen, keyboardQueue chan termbox.Event, done chan bool) {
	screen.Clear()
	v := ui.NewView("", 0, 0, screen.Width, screen.Height)
	v.Highlight = true
	app.Write(dry, v)
	err := v.Render()
	if err != nil {
		log.Panicf("Alarm!!! %s", err)
	}
	screen.Flush()
loop:
	for {
		select {
		case event := <-keyboardQueue:
			switch event.Type {
			case termbox.EventKey:
				screen.Clear()
				if event.Key == termbox.KeyEsc {
					break loop
				} else if event.Key == termbox.KeyArrowDown { //cursor up
					v.CursorDown()
					x, y := v.Origin()
					cx, cy := v.Cursor()
				} else if event.Key == termbox.KeyArrowUp { // cursor down
					v.CursorUp()
				} else if event.Ch == 'g' { //to the top of the view
					v.MoveCursorToTop()
				} else if event.Ch == 'G' { //to the bottom of the view
					v.MoveCursorToBottom()
				}

				v.Render()
				screen.Flush()
			}
		}
	}
	screen.Clear()
	done <- true
}

//-----------------------------------------------------------------------------
func main() {
	log.Infof("Launching dry")
	// parse flags
	_, err := flags.Parse(&opts)
	if err != nil {
		flagError := err.(*flags.Error)
		if flagError.Type == flags.ErrHelp {
			return
		}
		if flagError.Type == flags.ErrUnknownFlag {
			log.Error("Use --help to view all available options.")
			return
		}
		log.Errorf("Error parsing flags: %s\n", err)
		return
	}
	if os.Getenv("DOCKER_HOST") == "" {
		log.Error("No DOCKER_HOST environment variable found.")
		return
	}
	// Start profiling (if required)
	if opts.Profile {
		go func() {
			log.Info(http.ListenAndServe("localhost:6060", nil))
		}()
	}
	_ = "breakpoint"

	screen := ui.NewScreen()
	app := app.NewDryApp(screen)
	mainLoop(app, screen)
	screen.Close()
	log.Info("Bye")
}
