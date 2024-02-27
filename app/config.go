package app

import (
	"github.com/moncho/dry/docker"
)

// Config dry initial configuration
type Config struct {
	DockerHost         string
	DockerCertPath     string
	DockerTLSVerify    bool
	MonitorMode        bool
	MonitorRefreshRate int
}

func (c Config) dockerEnv() docker.Env {
	env := docker.NewEnv()
	env.DockerHost = c.DockerHost
	env.DockerTLSVerify = c.DockerTLSVerify
	env.DockerCertPath = c.DockerCertPath
	return env
}
