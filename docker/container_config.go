package docker

import (
	"strings"

	"github.com/moby/moby/api/types/container"
	"github.com/moby/moby/api/types/network"
)

type containerConfigBuilder struct {
	config     container.Config
	hostConfig container.HostConfig
	err        error
}

func newCCB() *containerConfigBuilder {
	return &containerConfigBuilder{
		config:     container.Config{},
		hostConfig: container.HostConfig{},
	}
}

func (cc *containerConfigBuilder) build() (container.Config, container.HostConfig, error) {
	return cc.config, cc.hostConfig, cc.err
}

func (cc *containerConfigBuilder) image(image string) *containerConfigBuilder {
	cc.config.Image = image
	return cc
}

func (cc *containerConfigBuilder) command(command string) *containerConfigBuilder {
	if command != "" {
		splitCommand := strings.Split(command, " ")
		if len(splitCommand) > 0 {
			cc.config.Cmd = splitCommand
		}
	}
	return cc
}

func (cc *containerConfigBuilder) ports(portSet map[string]struct{}) *containerConfigBuilder {
	if len(portSet) > 0 {
		exposedPorts := make(network.PortSet)
		bindings := make(network.PortMap)
		for rawPort := range portSet {
			pr, err := network.ParsePortRange(rawPort)
			if err != nil {
				cc.err = err
				break
			}
			for p := range pr.All() {
				exposedPorts[p] = struct{}{}
				bindings[p] = []network.PortBinding{{
					HostPort: p.Port(),
				}}
			}
		}
		cc.config.ExposedPorts = exposedPorts
		cc.hostConfig.PortBindings = bindings

	}
	return cc
}
