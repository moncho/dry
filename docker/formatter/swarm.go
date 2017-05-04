package formatter

import (
	"fmt"
	"strings"

	"github.com/docker/docker/api/types/swarm"
)

//FormatPorts returns the string representation of the given PortConfig
func FormatPorts(ports []swarm.PortConfig) string {
	result := []string{}
	for _, pConfig := range ports {
		result = append(result, fmt.Sprintf("*:%d->%d/%s",
			pConfig.PublishedPort,
			pConfig.TargetPort,
			pConfig.Protocol,
		))
	}
	return strings.Join(result, ",")
}
