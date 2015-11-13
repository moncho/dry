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
func mainLoop(screen *ui.Screen) {
	app := screen.App
	if ok, _ := app.Ok(); !ok {
		return
	}

	keyboardQueue := make(chan termbox.Event)
	timestampQueue := time.NewTicker(1 * time.Second)

	go func() {
		for {
			keyboardQueue <- termbox.PollEvent()
		}
	}()

	screen.Render(app.Render())
	//belongs outside the loop
	var streamMode = false
	var readOnlyMode = false
loop:
	for {
		var refresh = false
		select {

		case event := <-keyboardQueue:
			switch event.Type {
			case termbox.EventKey:
				if !app.State.ShowingHelp && !readOnlyMode {
					if event.Key == termbox.KeyEsc || event.Ch == 'q' || event.Ch == 'Q' {
						break loop
					} else if event.Key == termbox.KeyF1 { //sort
						app.Sort()
						app.Refresh()
					} else if event.Key == termbox.KeyF2 { //show all containers
						app.ToggleShowAllContainers()
					} else if event.Key == termbox.KeyF5 { // refresh
						app.Refresh()
					} else if event.Ch == '?' || event.Ch == 'h' || event.Ch == 'H' { //help
						app.ShowHelp()
					} else if event.Ch == 'e' || event.Ch == 'E' { //remove
						app.Rm(screen.CursorPosition())
					} else if event.Ch == 'k' || event.Ch == 'K' { //kill
						app.Kill(screen.CursorPosition())
					} else if event.Ch == 'l' || event.Ch == 'L' { //logs
						if logs, err := app.Logs(screen.CursorPosition()); err == nil {
							readOnlyMode = true
							streamMode = true
							stream(screen, logs)
						}
					} else if event.Ch == 'r' || event.Ch == 'R' { //start
						app.StartContainer(screen.CursorPosition())
					} else if event.Ch == 's' || event.Ch == 'S' { //stats
						app.Stats(screen.CursorPosition())
						readOnlyMode = true
						refresh = true
					} else if event.Ch == 't' || event.Ch == 'T' { //stop
						app.StopContainer(screen.CursorPosition())
					} else if event.Key == termbox.KeyArrowUp { //cursor up
						screen.MoveCursorUp()
						refresh = true
					} else if event.Key == termbox.KeyArrowDown { // cursor down
						screen.MoveCursorDown()
						refresh = true
					}
				} else if app.State.ShowingHelp {
					app.ShowDockerInfo()
				} else if readOnlyMode && event.Key == termbox.KeyEsc {
					readOnlyMode = false
					streamMode = false
					refresh = true
				}

			case termbox.EventResize:
				screen.Resize()
				refresh = true
			}

		case <-timestampQueue.C:
			if !streamMode {
				timestamp := time.Now().Format(`3:04:05pm PST`)
				screen.RenderLine(0, 0, `<right><white>`+timestamp+`</></right>`)
			}
		}
		if !streamMode && (refresh || app.Changed()) {
			_ = "breakpoint"
			screen.Clear().Render(app.Render())
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
	app := app.NewDryApp()
	screen := ui.NewScreen(app)
	mainLoop(screen)
	screen.Close()
	log.Info("Bye")
}
