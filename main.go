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
	logo = `     _
    | |
  __| |  ____  _   _
 / _  | / ___)| | | |
( (_| || |    | |_| |
 \____||_|     \__  |
               (____/
`
	loader1 = "Connecting to the Docker host.  "
	loader2 = "Connecting to the Docker host.. "
	loader3 = "Connecting to the Docker host..."
)

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
		} else {
			dockerEnv = &docker.DockerEnv{
				DockerHost:      os.Getenv("DOCKER_HOST"),
				DockerTLSVerify: docker.GetBool(os.Getenv("DOCKER_TLS_VERIFY")),
				DockerCertPath:  os.Getenv("DOCKER_CERT_PATH")}
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

func showLoadingScreen(screen *ui.Screen, dockerEnv *docker.DockerEnv, stop <-chan struct{}) {
	screen.RenderAtColumn(screen.Width/2-10, 0, ui.Yellow(logo))
	screen.RenderLine(2, 10, fmt.Sprintf("<blue>Version:</> %s", ui.White(version.VERSION)))
	if dockerEnv != nil {
		screen.RenderLine(2, 11, fmt.Sprintf("<blue>Docker Host:</> %s", ui.White(dockerEnv.DockerHost)))
	} else {
		screen.RenderLine(2, 11, ui.White("No Docker host"))
	}
	go func() {
		var loadingMessage = loader1
		var rotator = 1
		loaderTimer := time.NewTicker(1 * time.Second)
		for {
			select {
			case <-loaderTimer.C:
				screen.RenderLine(screen.Width/2-12, 14, ui.White(loadingMessage))
				screen.Flush()
				if rotator == 1 {
					loadingMessage = loader2
					rotator = 2
				} else if rotator == 2 {
					loadingMessage = loader3
					rotator = 3
				} else {
					loadingMessage = loader1
					rotator = 1
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
