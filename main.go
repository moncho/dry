package main

import (
	"fmt"
	"os"
	"time"

	"net/http"
	_ "net/http/pprof"

	log "github.com/Sirupsen/logrus"
	"github.com/jessevdk/go-flags"
	"github.com/moncho/dry/app"
	"github.com/moncho/dry/docker"
	"github.com/moncho/dry/ui"
	"github.com/moncho/dry/version"
	"github.com/nsf/termbox-go"
)

const (
	banner = `     _
    | |
  __| |  ____  _   _
 / _  | / ___)| | | |
( (_| || |    | |_| |
 \____||_|     \__  |
               (____/
`
)

var loadMessage = []string{"Connecting to the Docker host.  ",
	"Connecting to the Docker host.. ",
	"Connecting to the Docker host..."}

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

func newApp(screen *ui.Screen, dockerEnv *docker.DockerEnv) (*app.Dry, error) {
	return app.NewDryApp(screen, dockerEnv)
}

func newDockerEnv(opts dryOptions) *docker.DockerEnv {
	dockerEnv := &docker.DockerEnv{}
	if opts.DockerHost == "" {
		if os.Getenv("DOCKER_HOST") == "" {
			log.Info(
				fmt.Sprintf(
					"No DOCKER_HOST environment variable found and no Host parameter was given, trying %s",
					docker.DefaultDockerHost))
			dockerEnv.DockerHost = docker.DefaultDockerHost
		} else {
			dockerEnv.DockerHost = os.Getenv("DOCKER_HOST")
			dockerEnv.DockerTLSVerify = docker.GetBool(os.Getenv("DOCKER_TLS_VERIFY"))
			dockerEnv.DockerCertPath = os.Getenv("DOCKER_CERT_PATH")
		}
	} else {
		dockerEnv.DockerHost = opts.DockerHost
		dockerEnv.DockerTLSVerify = docker.GetBool(opts.DockerTLSVerifiy)
		dockerEnv.DockerCertPath = opts.DockerCertPath
	}
	return dockerEnv
}

func showLoadingScreen(screen *ui.Screen, dockerEnv *docker.DockerEnv, stop <-chan struct{}) {
	screen.RenderAtColumn(screen.Width/2-10, 0, ui.Yellow(banner))
	screen.RenderLine(2, 10, fmt.Sprintf("<blue>Version:</> %s", ui.White(version.VERSION)))
	if dockerEnv != nil {
		screen.RenderLine(2, 11, fmt.Sprintf("<blue>Docker Host:</> %s", ui.White(dockerEnv.DockerHost)))
	} else {
		screen.RenderLine(2, 11, ui.White("No Docker host"))
	}
	go func() {
		var rotorPos = 0
		timer := time.NewTicker(1 * time.Second)
		for {
			select {
			case <-timer.C:
				loadingMessage := loadMessage[rotorPos]
				screen.RenderLine(screen.Width/2-12, 14, ui.White(loadingMessage))
				screen.Flush()
				if rotorPos < len(loadMessage)-1 {
					rotorPos++
				} else {
					rotorPos = 0
				}
			case <-stop:
				return
			}
		}
	}()
}
func main() {
	running := false
	defer func() {
		if r := recover(); r != nil {
			termbox.Close()
			log.WithField("error", r).Error(
				"Dry panicked")
			log.Info("Bye")
			os.Exit(1)
		} else if running {
			log.Info("Bye")
		}
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
		fmt.Printf("dry version %s, build %s\n", version.VERSION, version.GITCOMMIT)
		return
	}
	log.Info("Launching dry")
	dockerEnv := newDockerEnv(opts)

	// Start profiling (if required)
	if opts.Profile {
		go func() {
			log.Info(http.ListenAndServe("localhost:6060", nil))
		}()
	}
	screen := ui.NewScreen()
	running = true

	//Loading screen
	stopLoadScreen := make(chan struct{}, 1)
	showLoadingScreen(screen, dockerEnv, stopLoadScreen)

	//newApp will load dry and try to establish the connection with the docker daemon
	dry, err := newApp(screen, dockerEnv)
	//dry has loaded, loading screen should not be shown
	close(stopLoadScreen)
	if err == nil {
		app.RenderLoop(dry, screen)
		dry.Close()
		screen.Close()
	} else {
		screen.Close()
		log.WithField("error", err).Error(
			"There was an error launching dry")
	}
}
