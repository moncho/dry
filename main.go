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

//dryOptions represents command line flags variables
type dryOptions struct {
	Description bool `short:"d" long:"description" description:"Dry description"`
	// enable profiling
	Profile bool `short:"p" long:"profile" description:"Enable profiling"`
	Version bool `short:"v" long:"version" description:"Dry version"`
	//Docker-related properties
	DockerHost       string `short:"H" long:"docker_host" description:"Docker Host"`
	DockerCertPath   string `short:"c" long:"docker_certpath" description:"Docker cert path"`
	DockerTLSVerifiy string `short:"t" long:"docker_tls" description:"Docker TLS verify"`
}

//-----------------------------------------------------------------------------
func mainScreen(dry *app.Dry, screen *ui.Screen) {

	if ok, _ := dry.Ok(); !ok {
		return
	}

	keyboardQueue, done := ui.EventChannel()
	timestampQueue := time.NewTicker(1 * time.Second)

	viewClosed := make(chan struct{}, 1)
	keyboardQueueForView := make(chan termbox.Event)
	dryOutputChan := dry.OuputChannel()

	defer timestampQueue.Stop()
	defer close(done)
	defer close(keyboardQueueForView)
	defer close(viewClosed)

	go func() {
		for {
			dryMessage := <-dryOutputChan
			screen.RenderLine(0, 0, dryMessage)
		}
	}()

	app.Render(dry, screen)
	//belongs outside the loop
	var viewMode = false

loop:
	for {
		//Used for refresh-forcing events happening outside dry
		var refresh = false
		select {
		case <-timestampQueue.C:
			if !viewMode {
				timestamp := time.Now().Format(`15:04:05`)
				screen.RenderLine(0, 0, `<right><white>`+timestamp+`</></right>`)
				screen.Flush()
			}
		case <-viewClosed:
			viewMode = false
			dry.ShowContainers()
		case event := <-keyboardQueue:
			switch event.Type {
			case termbox.EventKey:
				if !viewMode {
					if event.Key == termbox.KeyEsc || event.Ch == 'q' || event.Ch == 'Q' {
						break loop
					} else if event.Key == termbox.KeyArrowUp { //cursor up
						screen.ScrollCursorUp()
						refresh = true
					} else if event.Key == termbox.KeyArrowDown { // cursor down
						screen.ScrollCursorDown()
						refresh = true
					} else if event.Key == termbox.KeyF1 { //sort
						dry.Sort()
						dry.Refresh()
					} else if event.Key == termbox.KeyF2 { //show all containers
						dry.ToggleShowAllContainers()
					} else if event.Key == termbox.KeyF5 { // refresh
						dry.Refresh()
					} else if event.Key == termbox.KeyF10 { // docker info
						dry.ShowInfo()
						viewMode = true
						go less(dry, screen, keyboardQueueForView, viewClosed)
					} else if event.Ch == '?' || event.Ch == 'h' || event.Ch == 'H' { //help
						viewMode = true
						dry.ShowHelp()
						go less(dry, screen, keyboardQueueForView, viewClosed)
					} else if event.Ch == 'e' || event.Ch == 'E' { //remove
						dry.Rm(screen.CursorPosition())
					} else if event.Key == termbox.KeyCtrlE { //remove all stopped
						dry.RemoveAllStoppedContainers()
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
						done, errC, err := dry.Stats(screen.CursorPosition())
						if err == nil {
							viewMode = true
							go autorefresh(dry, screen, keyboardQueueForView, viewClosed, done, errC)
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

	log.Debug("something broke the loop")
}

func stream(screen *ui.Screen, stream io.ReadCloser, keyboardQueue chan termbox.Event, done chan<- struct{}) {
	screen.Clear()
	screen.Sync()
	v := ui.NewLess()
	go func() {
		io.Copy(v, stream)
	}()
	if err := v.Focus(keyboardQueue); err != nil {
		ui.ShowErrorMessage(screen, keyboardQueue, err)
	}

	stream.Close()
	termbox.HideCursor()
	screen.Clear()
	screen.Sync()
	done <- struct{}{}
}

//autorefresh view that autorefreshes its content every second
func autorefresh(dry *app.Dry, screen *ui.Screen, keyboardQueue chan termbox.Event, done chan<- struct{}, doneStats chan<- bool, errC <-chan error) {
	screen.Clear()
	v := ui.NewMarkupView("", 0, 0, screen.Width, screen.Height, false)
	//used to coordinate rendering betwen the ticker
	//and the exit event
	var mutex = &sync.Mutex{}
	app.Write(dry, v)
	err := v.Render()
	if err != nil {
		ui.ShowErrorMessage(screen, keyboardQueue, err)
	}
	screen.Flush()
	//the ticker is created after the first render
	timestampQueue := time.NewTicker(1000 * time.Millisecond)

loop:
	for {
		select {
		case <-errC:
			{
				mutex.Lock()
				timestampQueue.Stop()
				break loop
			}
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
	doneStats <- true
	done <- struct{}{}
}

//less shows dry output in a "less" emulator
func less(dry *app.Dry, screen *ui.Screen, keyboardQueue chan termbox.Event, done chan struct{}) {
	screen.Clear()
	v := ui.NewLess()
	v.MarkupSupport()
	go app.Write(dry, v)
	if err := v.Focus(keyboardQueue); err != nil {
		ui.ShowErrorMessage(screen, keyboardQueue, err)
	}
	termbox.HideCursor()
	screen.Clear()
	screen.Sync()

	done <- struct{}{}
}

//-----------------------------------------------------------------------------

func newApp(screen *ui.Screen, dockerEnv *docker.DockerEnv) (*app.Dry, error) {
	if dockerEnv == nil {
		return app.NewDryApp(screen)
	}
	return app.NewDryAppWithDockerEnv(screen, dockerEnv)
}

func newDockerEnv(opts dryOptions) *docker.DockerEnv {
	var dockerEnv *docker.DockerEnv
	if opts.DockerHost == "" {
		if os.Getenv("DOCKER_HOST") == "" {
			log.Info(
				fmt.Sprintf(
					"No DOCKER_HOST environment variable found and no Host parameter was given, trying %s",
					docker.DefaultDockerHost))
			dockerEnv = &docker.DockerEnv{
				DockerHost: docker.DefaultDockerHost,
			}
		}
	} else {
		dockerEnv = &docker.DockerEnv{
			DockerHost:      opts.DockerHost,
			DockerTLSVerify: docker.GetBool(opts.DockerTLSVerifiy),
			DockerCertPath:  opts.DockerCertPath,
		}
	}
	return dockerEnv
}

func main() {
	loggerWithVersion := log.WithFields(log.Fields{
		"version": version.VERSION,
		"build":   version.GITCOMMIT,
	})
	defer func() {
		if r := recover(); r != nil {
			loggerWithVersion.Fatal(r)
			os.Exit(1)
		}
		log.Info("Bye")
	}()
	// parse flags
	var opts dryOptions
	var parser = flags.NewParser(&opts, flags.Default)
	_, err := parser.Parse()
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
	dockerEnv := newDockerEnv(opts)

	dockerContextLogger := loggerWithVersion.WithFields(log.Fields{
		"env": dockerEnv,
	})

	// Start profiling (if required)
	if opts.Profile {
		go func() {
			log.Info(http.ListenAndServe("localhost:6060", nil))
		}()
	}
	screen := ui.NewScreen()
	defer screen.Close()
	app, err := newApp(screen, dockerEnv)
	if err == nil {
		mainScreen(app, screen)
		app.Close()
	} else {
		dockerContextLogger.Errorf("There was an error launching dry. %s", err)
	}
}
