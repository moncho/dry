package docker

import (
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/docker/docker/pkg/stringid"
	"github.com/docker/docker/pkg/units"
	"github.com/fsouza/go-dockerclient"
	"github.com/gosuri/uitable/util/strutil"
)

const (
	idHeader         = "CONTAINER ID"
	imageHeader      = "IMAGE"
	namesHeader      = "NAMES"
	commandHeader    = "COMMAND"
	createdAtHeader  = "CREATED AT"
	runningForHeader = "CREATED"
	statusHeader     = "STATUS"
	portsHeader      = "PORTS"
	sizeHeader       = "SIZE"
	labelsHeader     = "LABELS"
)

//ContainerFormatter knows how to pretty-print the information of a container
type ContainerFormatter struct {
	trunc  bool
	header []string
	c      docker.APIContainers
}

//ID prettifies the id
func (c *ContainerFormatter) ID() string {
	c.addHeader(idHeader)
	if c.trunc {
		return stringid.TruncateID(c.c.ID)
	}
	return c.c.ID
}

//Names prettifies the container name(s)
func (c *ContainerFormatter) Names() string {
	c.addHeader(namesHeader)
	names := stripNamePrefix(c.c.Names)
	if c.trunc {
		for _, name := range names {
			if len(strings.Split(name, "/")) == 1 {
				names = []string{name}
				break
			}
		}
	}
	return strings.Join(names, ",")
}

//Image prettifies the image used by the container
func (c *ContainerFormatter) Image() string {
	c.addHeader(imageHeader)
	if c.c.Image == "" {
		return "<no image>"
	}
	//No ImageID field on the go-docker API yet
	//if c.trunc {
	//if stringid.TruncateID(c.c.ImageID) == stringid.TruncateID(c.c.Image) {
	//			return stringutils.Truncate(c.c.Image, 12)
	//		}
	//}
	return c.c.Image
}

//Command prettifies the command that starts the container
func (c *ContainerFormatter) Command() string {
	c.addHeader(commandHeader)
	command := c.c.Command
	if c.trunc {
		command = strutil.Resize(command, 30)
	}
	return command
}

//CreatedAt prettifies the command that starts the container
func (c *ContainerFormatter) CreatedAt() string {
	c.addHeader(createdAtHeader)
	return DurationForHumans(c.c.Created)
}

//RunningFor prettifies the  that starts the container
func (c *ContainerFormatter) RunningFor() string {
	c.addHeader(runningForHeader)
	return DurationForHumans(c.c.Created)
}

//Ports prettifies the container port information
func (c *ContainerFormatter) Ports() string {
	c.addHeader(portsHeader)
	return displayablePorts(c.c.Ports)
}

//Status prettifies the container status
func (c *ContainerFormatter) Status() string {
	c.addHeader(statusHeader)
	return c.c.Status
}

//Size prettifies the container size
func (c *ContainerFormatter) Size() string {
	c.addHeader(sizeHeader)
	srw := units.HumanSize(float64(c.c.SizeRw))
	sf := srw

	if c.c.SizeRootFs > 0 {
		sv := units.HumanSize(float64(c.c.SizeRootFs))
		sf = fmt.Sprintf("%s (virtual %s)", srw, sv)
	}
	return sf
}

//Labels prettifies the container labels
func (c *ContainerFormatter) Labels() string {
	c.addHeader(labelsHeader)
	if c.c.Labels == nil {
		return ""
	}

	var joinLabels []string
	for k, v := range c.c.Labels {
		joinLabels = append(joinLabels, fmt.Sprintf("%s=%s", k, v))
	}
	return strings.Join(joinLabels, ",")
}

func (c *ContainerFormatter) fullHeader() string {
	if c.header == nil {
		return ""
	}
	return strings.Join(c.header, "\t")
}

func (c *ContainerFormatter) addHeader(header string) {
	if c.header == nil {
		c.header = []string{}
	}
	c.header = append(c.header, strings.ToUpper(header))
}

func stripNamePrefix(ss []string) []string {
	for i, s := range ss {
		if s[0] == '/' {
			ss[i] = s[1:]
		}
	}

	return ss
}

func displayablePorts(ports []docker.APIPort) string {
	type portGroup struct {
		first int64
		last  int64
	}
	groupMap := make(map[string]*portGroup)
	var result []string
	var hostMappings []string
	var groupMapKeys []string
	sort.Sort(byPortInfo(ports))
	for _, port := range ports {
		current := port.PrivatePort
		portKey := port.Type
		if port.IP != "" {
			if port.PublicPort != current {
				hostMappings = append(hostMappings,
					fmt.Sprintf("%s:%d->%d/%s", port.IP, port.PublicPort, port.PrivatePort, port.Type))
				continue
			}
			portKey = fmt.Sprintf("%s/%s", port.IP, port.Type)
		}
		group := groupMap[portKey]

		if group == nil {
			groupMap[portKey] = &portGroup{first: current, last: current}
			// record order that groupMap keys are created
			groupMapKeys = append(groupMapKeys, portKey)
			continue
		}
		if current == (group.last + 1) {
			group.last = current
			continue
		}

		result = append(result, formGroup(portKey, group.first, group.last))
		groupMap[portKey] = &portGroup{first: current, last: current}
	}
	for _, portKey := range groupMapKeys {
		g := groupMap[portKey]
		result = append(result, formGroup(portKey, g.first, g.last))
	}
	result = append(result, hostMappings...)
	return strings.Join(result, ", ")
}

// byPortInfo is a temporary type used to sort types.Port by its fields
type byPortInfo []docker.APIPort

func (r byPortInfo) Len() int      { return len(r) }
func (r byPortInfo) Swap(i, j int) { r[i], r[j] = r[j], r[i] }
func (r byPortInfo) Less(i, j int) bool {
	if r[i].PrivatePort != r[j].PrivatePort {
		return r[i].PrivatePort < r[j].PrivatePort
	}

	if r[i].IP != r[j].IP {
		return r[i].IP < r[j].IP
	}

	if r[i].PublicPort != r[j].PublicPort {
		return r[i].PublicPort < r[j].PublicPort
	}

	return r[i].Type < r[j].Type
}

func formGroup(key string, start, last int64) string {
	parts := strings.Split(key, "/")
	groupType := parts[0]
	var ip string
	if len(parts) > 1 {
		ip = parts[0]
		groupType = parts[1]
	}
	group := strconv.FormatInt(start, 10)
	if start != last {
		group = fmt.Sprintf("%s-%d", group, last)
	}
	if ip != "" {
		group = fmt.Sprintf("%s:%s->%s", ip, group, group)
	}
	return fmt.Sprintf("%s/%s", group, groupType)
}
