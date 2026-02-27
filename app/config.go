package app

import "time"

// Config dry initial configuration
type Config struct {
	DockerHost         string
	DockerCertPath     string
	DockerTLSVerify    bool
	MonitorMode        bool
	MonitorRefreshRate int
	SplashDuration     time.Duration
}
