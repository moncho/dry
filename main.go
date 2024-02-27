package main

import (
	"context"
	"fmt"
	"os"
	"runtime/debug"
	"strconv"
	"time"

	"net/http"
	_ "net/http/pprof"

	"github.com/jessevdk/go-flags"
	"github.com/moncho/dry/app"
	"github.com/moncho/dry/appui"
	"github.com/moncho/dry/docker"
	"github.com/moncho/dry/ui"
	"github.com/moncho/dry/version"
	log "github.com/sirupsen/logrus"
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
	cheese     = "<white>made with \U0001F499  (and go) by</> <blue>moncho</>"
	connecting = "\U0001f433  Trying to connect to the Docker Host \U0001f433"
)

var loadMessage = []string{docker.Whale0,
	docker.Whale1,
	docker.Whale2,
	docker.Whale3,
	docker.Whale4,
	docker.Whale5,
	docker.Whale6,
	docker.Whale7,
	docker.Whale}

// options dry's flags
type options struct {
	Description bool   `short:"d" long:"description" description:"Shows the description"`
	MonitorMode string `short:"m" long:"monitor" description:"Starts in monitor mode, given value (if any) is the refresh rate" optional:"yes" optional-value:"500"`
	// enable profiling
	Profile bool `short:"p" long:"profile" description:"Enable profiling"`
	Version bool `short:"v" long:"version" description:"Dry version"`
	//Docker-related properties
	DockerHost       string `short:"H" long:"docker_host" description:"Docker Host"`
	DockerCertPath   string `short:"c" long:"docker_certpath" description:"Docker cert path"`
	DockerTLSVerifiy string `short:"t" long:"docker_tls" description:"Docker TLS verify"`
	//Whale
	Whale uint `short:"w" long:"whale" description:"Show whale for w seconds"`
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
		cfg.DockerTLSVerify = docker.GetBool(opts.DockerTLSVerifiy)
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

func showLoadingScreen(ctx context.Context, screen *ui.Screen, cfg app.Config) {
	screen.Clear()
	midscreen := screen.Dimensions().Width / 2
	height := screen.Dimensions().Height
	screen.RenderAtColumn(midscreen-len(connecting)/2, 1, ui.White(connecting))
	screen.RenderLine(2, height-2, fmt.Sprintf("<blue>Dry Version:</> %s", ui.White(version.VERSION)))
	if cfg.DockerHost != "" {
		screen.RenderLine(2, height-1, fmt.Sprintf("<blue>Docker Host:</> %s", ui.White(cfg.DockerHost)))
	} else {
		screen.RenderLine(2, height-1, ui.White("No Docker host"))
	}

	//20 is a safe aproximation for the length of interpreted characters from the message
	screen.RenderLine(ui.ActiveScreen.Dimensions().Width-len(cheese)+20, height-1, cheese)
	screen.Flush()
	go func() {
		rotorPos := 0
		forward := true
		ticker := time.NewTicker(250 * time.Millisecond)

		for {
			select {
			case <-ticker.C:
				loadingMessage := loadMessage[rotorPos]
				screen.RenderAtColumn(midscreen-19, height/2-6, ui.Cyan(loadingMessage))
				screen.Flush()
				if rotorPos == len(loadMessage)-1 {
					forward = false
				} else if rotorPos == 0 {
					forward = true
				}
				if forward {
					rotorPos++
				} else {
					rotorPos--
				}

			case <-ctx.Done():
				return
			}
		}
	}()
}
func main() {
	defer func() {
		if r := recover(); r != nil {
			log.Printf(
				"Dry panicked: %v", r)
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
	screen, err := ui.NewScreen(appui.DryTheme)
	if err != nil {
		log.Printf("Dry could not start: %s", err)
		return
	}
	cfg, err := config(opts)
	if err != nil {
		log.Println(err.Error())
		return
	}

	start := time.Now()
	ctx, cancel := context.WithCancel(context.Background())
	showLoadingScreen(ctx, screen, cfg)

	dry, err := app.NewDry(screen, cfg)

	//if asked to, show whale to bablat a bit longer
	if opts.Whale > 0 {
		showWale, _ := time.ParseDuration(fmt.Sprintf("%ds", opts.Whale))
		time.Sleep(showWale - time.Since(start))
	}
	//dry has loaded, stopping the loading screen
	cancel()
	if err != nil {
		screen.Close()
		log.Printf("Dry could not start: %s", err)
		return
	}
	app.RenderLoop(dry)
	screen.Close()
}
