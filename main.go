package main

import (
	"fmt"
	"io"
	"os"
	"sync"
	"time"

	"net/http"

	log "github.com/Sirupsen/logrus"
	"github.com/jessevdk/go-flags"
	"github.com/moncho/dry/app"
	"github.com/moncho/dry/docker"
	"github.com/moncho/dry/ui"
	"github.com/moncho/dry/version"
	"github.com/nsf/termbox-go"
)
import _ "net/http/pprof"

// command line flags variable
var opts struct {
	Description bool `short:"a" long:"description" description:"Dry description"`
	// enable profiling
	Profile bool `short:"p" long:"profile" description:"Enable profiling"`
	Version bool `short:"v" long:"version" description:"Dry version"`
	//Docker-related properties
	DockerHost       string `short:"d" long:"docker_host" description:"Docker Host"`
	DockerCertPath   string `short:"c" long:"docker_certpath" description:"Docker cert path"`
	DockerTLSVerifiy bool   `short:"t" long:"docker_tls" description:"Docker TLS verify"`
}

//-----------------------------------------------------------------------------
func mainScreen(dry *app.Dry, screen *ui.Screen) {

	if ok, _ := dry.Ok(); !ok {
		return
	}

	keyboardQueue := make(chan termbox.Event)
	timestampQueue := time.NewTicker(1 * time.Second)

	viewClosed := make(chan bool, 1)
	keyboardQueueForView := make(chan termbox.Event)

	defer timestampQueue.Stop()
	defer close(keyboardQueueForView)
	defer close(viewClosed)
	defer close(keyboardQueue)

	go func() {
		for {
			keyboardQueue <- termbox.PollEvent()
		}
	}()

	app.Render(dry, screen)
	//belongs outside the loop
	var viewMode = false
loop:
	for {
		//Used for refresh-forcing events outside dry
		var refresh = false
		select {
		case <-timestampQueue.C:
			if !viewMode {
				timestamp := time.Now().Format(`3:04:05pm PST`)
				screen.RenderLine(0, 0, `<right><white>`+timestamp+`</></right>`)
				screen.Flush()
			}
		case <-viewClosed:
			viewMode = false
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
							go stream(screen, logs, keyboardQueueForView, viewClosed)
						}
					} else if event.Ch == 'r' || event.Ch == 'R' { //start
						dry.StartContainer(screen.CursorPosition())
					} else if event.Ch == 's' || event.Ch == 'S' { //stats
						done, err := dry.Stats(screen.CursorPosition())
						if err == nil {
							viewMode = true
							go autorefresh(dry, screen, keyboardQueueForView, viewClosed, done)
						} else {
							log.Warn(err)
						}
					} else if event.Ch == 't' || event.Ch == 'T' { //stop
						dry.StopContainer(screen.CursorPosition())
					} else if event.Key == termbox.KeyEnter { //inspect
						dry.Inspect(screen.CursorPosition())
						viewMode = true
						go less(dry, screen, keyboardQueueForView, viewClosed)

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
		if refresh || dry.Changed() {
			screen.Clear()
			app.Render(dry, screen)
		}
	}

	log.Info("something broke the loop")
}

func stream(screen *ui.Screen, stream io.ReadCloser, keyboardQueue chan termbox.Event, done chan bool) {
	screen.Clear()
	screen.Sync()
	v := ui.NewView("", 0, 0, screen.Width, screen.Height, true)
	go func() {
		io.Copy(v, stream)
	}()
	screen.RenderLine(0, 0, "Use the arrow keys, 'g' and 'G' to move through the log, escape key to go back.")
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
	stream.Close()
	termbox.HideCursor()
	screen.Clear()
	screen.Sync()
	done <- true
}

//autorefresh view that autorefreshes its content every second
func autorefresh(dry *app.Dry, screen *ui.Screen, keyboardQueue chan termbox.Event, done chan<- bool, doneStats chan<- bool) {
	defer func() { doneStats <- true }()
	defer func() { done <- true }()
	screen.Clear()
	v := ui.NewMarkupView("", 0, 0, screen.Width, screen.Height, false)
	//used to coordinate rendering betwen the ticker
	//and the exit event
	var mutex = &sync.Mutex{}
	app.Write(dry, v)
	err := v.Render()
	if err != nil {
		log.Panicf("Alarm!!! %s", err)
	}
	screen.Flush()
	//the ticker is created after the first render
	timestampQueue := time.NewTicker(1000 * time.Millisecond)

loop:
	for {
		select {
		case event := <-keyboardQueue:
			switch event.Type {
			case termbox.EventKey:
				if event.Key == termbox.KeyEsc {
					//the lock is acquired and the time-based refresh queue is stopped
					//before breaking the loop
					mutex.Lock()
					timestampQueue.Stop()
					break loop
				}
			}
		case <-timestampQueue.C:
			{
				mutex.Lock()
				v.Clear()
				app.Write(dry, v)
				v.Render()
				screen.Flush()
				mutex.Unlock()
			}
		}
	}
	//cleanup before exiting, the screen is cleared and the lock released
	termbox.HideCursor()
	screen.Clear()
	screen.Sync()
	mutex.Unlock()
}

//less shows dry output in a "less" emulator
func less(dry *app.Dry, screen *ui.Screen, keyboardQueue chan termbox.Event, done chan bool) {
	screen.Clear()
	v := ui.NewView("", 0, 0, screen.Width, screen.Height, true)
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
	termbox.HideCursor()
	screen.Clear()
	screen.Sync()

	done <- true
}

//-----------------------------------------------------------------------------

func newApp(screen *ui.Screen, dockerEnv *docker.DockerEnv) (*app.Dry, error) {
	if dockerEnv == nil {
		return app.NewDryApp(screen)
	}
	return app.NewDryAppWithDockerEnv(screen, dockerEnv)
}

func main() {
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
	if opts.Description {
		fmt.Print(app.ShortHelp)
		return
	}
	if opts.Version {
		fmt.Printf("dry version %s, build %s", version.VERSION, version.GITCOMMIT)
		return
	}
	log.Info("Launching dry")
	var dockerEnv *docker.DockerEnv
	if opts.DockerHost == "" {
		if os.Getenv("DOCKER_HOST") == "" {
			log.Error("No DOCKER_HOST environment variable found.")
			return
		}
	} else {
		dockerEnv = &docker.DockerEnv{
			DockerHost:      opts.DockerHost,
			DockerTLSVerify: opts.DockerTLSVerifiy,
			DockerCertPath:  opts.DockerCertPath,
		}
	}

	// Start profiling (if required)
	if opts.Profile {
		go func() {
			log.Info(http.ListenAndServe("localhost:6060", nil))
		}()
	}
	screen := ui.NewScreen()
	newApp(screen, dockerEnv)
	app, err := newApp(screen, dockerEnv)
	if err == nil {
		mainScreen(app, screen)
	} else {
		log.Error(err)
	}
	screen.Close()
	log.Info("Bye")
}
