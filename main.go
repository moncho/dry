package main

import (
	"fmt"
	"os"
	"runtime/debug"
	"strconv"

	"net/http"
	_ "net/http/pprof"

	tea "charm.land/bubbletea/v2"
	"github.com/jessevdk/go-flags"
	"github.com/moncho/dry/app"
	"github.com/moncho/dry/docker"
	"github.com/moncho/dry/version"
	log "github.com/sirupsen/logrus"
)

// options dry's flags
type options struct {
	Description bool   `short:"d" long:"description" description:"Shows the description"`
	MonitorMode string `short:"m" long:"monitor" description:"Starts in monitor mode, given value (if any) is the refresh rate" optional:"yes" optional-value:"500"`
	// enable profiling
	Profile bool `short:"p" long:"profile" description:"Enable profiling"`
	Version bool `short:"v" long:"version" description:"Dry version"`
	//Docker-related properties
	DockerHost      string `short:"H" long:"docker_host" description:"Docker Host"`
	DockerCertPath  string `short:"c" long:"docker_certpath" description:"Docker cert path"`
	DockerTLSVerify string `short:"t" long:"docker_tls" description:"Docker TLS verify"`
}

func config(opts options) (app.Config, error) {
	var cfg app.Config
	if opts.DockerHost == "" {
		if os.Getenv("DOCKER_HOST") == "" {
			log.Printf(
				"No DOCKER_HOST env variable found and no Host parameter was given, connecting to %s",
				docker.DefaultDockerHost)
			cfg.DockerHost = docker.DefaultDockerHost
		} else {
			cfg.DockerHost = os.Getenv("DOCKER_HOST")
			cfg.DockerTLSVerify = docker.GetBool(os.Getenv("DOCKER_TLS_VERIFY"))
			cfg.DockerCertPath = os.Getenv("DOCKER_CERT_PATH")
		}
	} else {
		cfg.DockerHost = opts.DockerHost
		cfg.DockerTLSVerify = docker.GetBool(opts.DockerTLSVerify)
		cfg.DockerCertPath = opts.DockerCertPath
	}

	if opts.MonitorMode != "" {
		cfg.MonitorMode = true
		refreshRate, err := strconv.Atoi(opts.MonitorMode)
		if err != nil {
			return cfg, fmt.Errorf("invalid refresh rate %s: %w", opts.MonitorMode, err)
		}
		cfg.MonitorRefreshRate = refreshRate
	}
	return cfg, nil
}

func main() {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("Dry panicked: %v", r)
			log.Error(string(debug.Stack()))
			os.Exit(1)
		}
	}()

	// parse flags
	var opts options
	parser := flags.NewParser(&opts, flags.Default)
	_, err := parser.Parse()
	if err != nil {
		flagError := err.(*flags.Error)
		if flagError.Type == flags.ErrHelp {
			return
		}
		if flagError.Type == flags.ErrUnknownFlag {
			log.Print("Use --help to view all available options.")
			return
		}
		log.Printf("Could not parse flags: %s", err)
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
	// Start profiling (if required)
	if opts.Profile {
		go func() {
			log.Fatal(http.ListenAndServe("localhost:6060", nil))
		}()
	}

	cfg, err := config(opts)
	if err != nil {
		log.Println(err.Error())
		return
	}

	m := app.NewModel(cfg)
	p := tea.NewProgram(m)

	if _, err := p.Run(); err != nil {
		log.Printf("Error running dry: %s", err)
		os.Exit(1)
	}
}
