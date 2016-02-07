package main

import (
	"fmt"
	"os"

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
	dry, err := newApp(screen, dockerEnv)
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
