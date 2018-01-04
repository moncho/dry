package docker

import (
	"strings"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/strslice"
	"github.com/docker/go-connections/nat"
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
		splittedCommand := strings.Split(command, " ")
		if len(splittedCommand) > 0 {
			cc.config.Cmd = strslice.StrSlice(splittedCommand)
		}
	}
	return cc
}

func (cc *containerConfigBuilder) ports(portSet nat.PortSet) *containerConfigBuilder {
	if len(portSet) > 0 {
		cc.config.ExposedPorts = portSet
		bindings := make(map[nat.Port][]nat.PortBinding)
		for rawPort := range portSet {
			portMappings, err := nat.ParsePortSpec(rawPort.Port())
			if err != nil {
				cc.err = err
				break
			}

			for _, portMapping := range portMappings {
				port := portMapping.Port
				bindings[port] = append(bindings[port], nat.PortBinding{
					HostPort: rawPort.Port(),
				})
			}
		}
		cc.hostConfig.PortBindings = bindings

	}
	return cc
}
