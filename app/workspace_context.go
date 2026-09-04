package app

// Workspace context construction: per-kind summary builders and formatting helpers.
// Moved verbatim from model.go.

import (
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/docker/go-units"
	dockercontainer "github.com/moby/moby/api/types/container"
	"github.com/moby/moby/api/types/image"
	"github.com/moby/moby/api/types/network"
	"github.com/moby/moby/api/types/swarm"
	"github.com/moby/moby/api/types/volume"
	"github.com/moncho/dry/appui"
	"github.com/moncho/dry/docker"
)

type workspaceContextKind int

const (
	workspaceContextNone workspaceContextKind = iota
	workspaceContextContainer
	workspaceContextComposeProject
	workspaceContextComposeService
	workspaceContextMonitor
)

type workspaceContext struct {
	kind              workspaceContextKind
	title             string
	subtitle          string
	lines             []string
	containerID       string
	imageID           string
	monitorCID        string
	networkID         string
	nodeID            string
	project           string
	service           string
	serviceID         string
	stackName         string
	taskID            string
	volumeName        string
	monitorCPU        float64
	monitorMem        float64
	monitorMax        float64
	monitorPct        float64
	monitorCPUHistory []appui.MonitorPoint
	monitorMemHistory []appui.MonitorPoint
}

type monitorContainerLookup interface {
	ContainerByID(id string) *docker.Container
}

// identity returns a stable string identifying the target of this context so
// the cursor's current selection can be compared against the pinned one.
func (c workspaceContext) identity() string {
	return fmt.Sprintf("%d|%s|%s|%s|%s|%s|%s|%s|%s|%s|%s/%s",
		c.kind, c.containerID, c.imageID, c.monitorCID, c.networkID,
		c.nodeID, c.serviceID, c.stackName, c.taskID, c.volumeName,
		c.project, c.service)
}

func workspaceContextFromContainer(c *docker.Container) workspaceContext {
	name := shortID(c.ID)
	if len(c.Names) > 0 {
		name = strings.TrimPrefix(c.Names[0], "/")
	}
	lines := []string{
		fmt.Sprintf("id: %s", shortID(c.ID)),
		fmt.Sprintf("status: %s", c.Status),
		fmt.Sprintf("image: %s", c.Image),
	}
	if c.Created > 0 {
		lines = append(lines, fmt.Sprintf("created: %s", workspaceFormatUnix(c.Created)))
	}
	if c.Detail.State != nil {
		if status := c.Detail.State.Status; status != "" {
			lines = append(lines, fmt.Sprintf("state: %s", status))
		}
		if c.Detail.State.Health != nil && c.Detail.State.Health.Status != "" {
			lines = append(lines, fmt.Sprintf("health: %s", c.Detail.State.Health.Status))
		}
		if started := workspaceFormatTimestamp(c.Detail.State.StartedAt); started != "" {
			lines = append(lines, fmt.Sprintf("started: %s", started))
		}
		if finished := workspaceFormatTimestamp(c.Detail.State.FinishedAt); finished != "" {
			lines = append(lines, fmt.Sprintf("finished: %s", finished))
		}
	}
	if c.Detail.RestartCount > 0 {
		lines = append(lines, fmt.Sprintf("restarts: %d", c.Detail.RestartCount))
	}
	if project := c.Labels["com.docker.compose.project"]; project != "" {
		lines = append(lines, fmt.Sprintf("compose project: %s", project))
	}
	if service := c.Labels["com.docker.compose.service"]; service != "" {
		lines = append(lines, fmt.Sprintf("compose service: %s", service))
	}
	if c.Command != "" {
		lines = append(lines, fmt.Sprintf("command: %s", c.Command))
	}
	if len(c.Ports) > 0 {
		lines = append(lines, fmt.Sprintf("ports: %s", workspaceContainerPorts(c)))
	}
	if c.Detail.Config != nil {
		if c.Detail.Config.User != "" {
			lines = append(lines, fmt.Sprintf("user: %s", c.Detail.Config.User))
		}
		if c.Detail.Config.WorkingDir != "" {
			lines = append(lines, fmt.Sprintf("workdir: %s", c.Detail.Config.WorkingDir))
		}
		if envs := len(c.Detail.Config.Env); envs > 0 {
			lines = append(lines, fmt.Sprintf("env: %d vars", envs))
		}
	}
	if mounts := len(c.Detail.Mounts); mounts > 0 {
		lines = append(lines, fmt.Sprintf("mounts: %d", mounts))
		if targets := workspaceContainerMountTargets(c); targets != "" {
			lines = append(lines, fmt.Sprintf("mount targets: %s", targets))
		}
	}
	if networks := workspaceContainerNetworkCount(c); networks > 0 {
		lines = append(lines, fmt.Sprintf("networks: %d", networks))
		if names := workspaceContainerNetworkNames(c); names != "" {
			lines = append(lines, fmt.Sprintf("network names: %s", names))
		}
	}
	if labels := len(c.Labels); labels > 0 {
		lines = append(lines, fmt.Sprintf("labels: %d", labels))
		lines = append(lines, fmt.Sprintf("label keys: %s", workspaceMapKeys(c.Labels, 6)))
	}
	return workspaceContext{
		kind:        workspaceContextContainer,
		title:       name,
		subtitle:    "Container",
		lines:       lines,
		containerID: c.ID,
	}
}

// defined is how many service rows the projects view shows, which counts the
// ones nothing runs and p.Services, taken from containers, does not. It can
// also be lower, or zero: the panel is reachable with the projects list
// unloaded, and the two numbers then come from different sources. Only a
// higher count says anything, so only a higher count is shown.
func workspaceContextFromComposeProject(p docker.ComposeProject, defined int) workspaceContext {
	services := fmt.Sprintf("services: %d", p.Services)
	if defined > p.Services {
		// "3 of 4 in the file", not "3 of 4 defined": the line sits above
		// "containers: 4", where "defined" reads as the smaller number's
		// qualifier rather than the larger one's.
		services = fmt.Sprintf("services: %d of %d in the file", p.Services, defined)
	}
	lines := []string{
		services,
		fmt.Sprintf("containers: %d", p.Containers),
		fmt.Sprintf("running: %d", p.Running),
		fmt.Sprintf("exited: %d", p.Exited),
		fmt.Sprintf("health: %s", workspaceRunningHealth(p.Running, p.Containers)),
	}
	if p.Containers > 0 {
		lines = append(lines, fmt.Sprintf("status ratio: %d/%d running", p.Running, p.Containers))
	}
	return workspaceContext{
		kind:     workspaceContextComposeProject,
		title:    p.Name,
		subtitle: "Compose Project",
		lines:    lines,
		project:  p.Name,
	}
}

func workspaceContextFromComposeService(s docker.ComposeService) workspaceContext {
	lines := []string{
		fmt.Sprintf("project: %s", s.Project),
		fmt.Sprintf("service: %s", s.Name),
		fmt.Sprintf("containers: %d", s.Containers),
		fmt.Sprintf("running: %d", s.Running),
		fmt.Sprintf("exited: %d", s.Exited),
	}
	if s.Containers > 0 {
		lines = append(lines, fmt.Sprintf("status ratio: %d/%d running", s.Running, s.Containers))
	} else {
		// A service row with no containers is one the compose file defines
		// and nothing runs, since every other row is built from a
		// container. "containers: 0" on its own reads as a fault, so the
		// panel names the state, in the SYNC column's own word so the two
		// do not describe one thing twice.
		lines = append(lines, "state: absent, u brings it up")
	}
	if s.Image != "" {
		lines = append(lines, fmt.Sprintf("image: %s", s.Image))
	}
	if s.Health != "" {
		lines = append(lines, fmt.Sprintf("health: %s", s.Health))
	}
	if s.Ports != "" {
		lines = append(lines, fmt.Sprintf("ports: %s", s.Ports))
	}
	// The summary describes containers, so a service with none has nothing
	// for it to describe: "health summary: empty" under "not created" reads
	// as a second and worse diagnosis of the same thing.
	if s.Containers > 0 {
		lines = append(lines, fmt.Sprintf("health summary: %s", workspaceRunningHealth(s.Running, s.Containers)))
	}
	return workspaceContext{
		kind:     workspaceContextComposeService,
		title:    s.Name,
		subtitle: "Compose Service",
		lines:    lines,
		project:  s.Project,
		service:  s.Name,
	}
}

func workspaceContextFromImage(img image.Summary) workspaceContext {
	title := docker.TruncateID(docker.ImageID(img.ID))
	subtitle := "Image"
	if len(img.RepoTags) > 0 && img.RepoTags[0] != "<none>:<none>" {
		title = img.RepoTags[0]
	}
	lines := []string{
		fmt.Sprintf("id: %s", docker.TruncateID(docker.ImageID(img.ID))),
	}
	if img.ParentID != "" {
		lines = append(lines, fmt.Sprintf("parent: %s", docker.TruncateID(docker.ImageID(img.ParentID))))
	}
	if len(img.RepoTags) > 0 {
		lines = append(lines, fmt.Sprintf("tag count: %d", len(img.RepoTags)))
		lines = append(lines, fmt.Sprintf("tags: %s", strings.Join(img.RepoTags, ", ")))
	}
	if len(img.RepoDigests) > 0 {
		lines = append(lines, fmt.Sprintf("digests: %d", len(img.RepoDigests)))
		lines = append(lines, fmt.Sprintf("digest refs: %s", workspaceJoinLimited(img.RepoDigests, 3)))
	}
	if len(img.Manifests) > 0 {
		lines = append(lines, fmt.Sprintf("manifests: %d", len(img.Manifests)))
	}
	if len(img.Labels) > 0 {
		lines = append(lines, fmt.Sprintf("labels: %d", len(img.Labels)))
		lines = append(lines, fmt.Sprintf("label keys: %s", workspaceMapKeys(img.Labels, 6)))
	}
	lines = append(lines,
		fmt.Sprintf("created: %s", workspaceFormatUnix(img.Created)),
		fmt.Sprintf("size: %s", units.BytesSize(float64(img.Size))),
	)
	if img.SharedSize > 0 {
		lines = append(lines, fmt.Sprintf("shared size: %s", units.BytesSize(float64(img.SharedSize))))
	}
	if img.Containers > 0 {
		lines = append(lines, fmt.Sprintf("used by: %d containers", img.Containers))
	}
	return workspaceContext{
		title:    title,
		subtitle: subtitle,
		lines:    lines,
		imageID:  img.ID,
	}
}

func workspaceContextFromNetwork(n network.Inspect) workspaceContext {
	title := n.Name
	if title == "" {
		title = docker.TruncateID(n.ID)
	}
	lines := []string{
		fmt.Sprintf("id: %s", docker.TruncateID(n.ID)),
		fmt.Sprintf("driver: %s", n.Driver),
		fmt.Sprintf("scope: %s", n.Scope),
		fmt.Sprintf("containers: %d", len(n.Containers)),
	}
	if !n.Created.IsZero() {
		lines = append(lines, fmt.Sprintf("created: %s", workspaceFormatLocalTime(n.Created)))
	}
	if n.IPAM.Driver != "" {
		lines = append(lines, fmt.Sprintf("ipam: %s", n.IPAM.Driver))
	}
	if len(n.IPAM.Config) > 0 && n.IPAM.Config[0].Subnet.IsValid() {
		lines = append(lines, "subnet: "+n.IPAM.Config[0].Subnet.String())
	}
	if len(n.IPAM.Config) > 0 && n.IPAM.Config[0].Gateway.IsValid() {
		lines = append(lines, "gateway: "+n.IPAM.Config[0].Gateway.String())
	}
	if n.Internal {
		lines = append(lines, "internal: true")
	}
	if n.Attachable {
		lines = append(lines, "attachable: true")
	}
	if n.Ingress {
		lines = append(lines, "ingress: true")
	}
	if n.ConfigOnly {
		lines = append(lines, "config only: true")
	}
	if !n.EnableIPv4 {
		lines = append(lines, "ipv4: disabled")
	}
	if n.EnableIPv6 {
		lines = append(lines, "ipv6: enabled")
	}
	if services := len(n.Services); services > 0 {
		lines = append(lines, fmt.Sprintf("services: %d", services))
	}
	if options := len(n.Options); options > 0 {
		lines = append(lines, fmt.Sprintf("options: %d", options))
		lines = append(lines, fmt.Sprintf("option keys: %s", workspaceMapKeys(n.Options, 6)))
	}
	if labels := len(n.Labels); labels > 0 {
		lines = append(lines, fmt.Sprintf("labels: %d", labels))
		lines = append(lines, fmt.Sprintf("label keys: %s", workspaceMapKeys(n.Labels, 6)))
	}
	if attached := workspaceAttachedNetworkNames(n); attached != "" {
		lines = append(lines, fmt.Sprintf("attached: %s", attached))
	}
	return workspaceContext{
		title:     title,
		subtitle:  "Network",
		lines:     lines,
		networkID: n.ID,
	}
}

func workspaceContextFromVolume(v *volume.Volume) workspaceContext {
	if v == nil {
		return workspaceContext{}
	}
	lines := []string{
		fmt.Sprintf("name: %s", v.Name),
		fmt.Sprintf("driver: %s", v.Driver),
		fmt.Sprintf("mountpoint: %s", v.Mountpoint),
	}
	if v.CreatedAt != "" {
		lines = append(lines, fmt.Sprintf("created: %s", v.CreatedAt))
	}
	if v.Scope != "" {
		lines = append(lines, fmt.Sprintf("scope: %s", v.Scope))
	}
	if len(v.Labels) > 0 {
		lines = append(lines, fmt.Sprintf("labels: %d", len(v.Labels)))
		lines = append(lines, fmt.Sprintf("label keys: %s", workspaceMapKeys(v.Labels, 6)))
	}
	if len(v.Options) > 0 {
		lines = append(lines, fmt.Sprintf("options: %d", len(v.Options)))
		lines = append(lines, fmt.Sprintf("option keys: %s", workspaceMapKeys(v.Options, 6)))
	}
	return workspaceContext{
		title:      v.Name,
		subtitle:   "Volume",
		lines:      lines,
		volumeName: v.Name,
	}
}

// statsContainerID returns the ID to use for daemon lookups from a Stats
// entry: the daemon store is keyed by the full container ID, so the truncated
// s.CID never matches it.
func statsContainerID(s *docker.Stats) string {
	if s.ID != "" {
		return s.ID
	}
	return s.CID
}

// workspaceMonitorTarget builds only the identity and title of a monitor
// context: no stat lines, no series copies, no daemon lookup. It runs once
// per render while pinned, so it must stay cheap — and it must agree with
// workspaceContextFromStats on identity and title (locked by
// TestWorkspaceMonitorTargetMatchesFullContext).
func workspaceMonitorTarget(s *docker.Stats) workspaceContext {
	if s == nil {
		return workspaceContext{}
	}
	title := s.Name
	if title == "" {
		title = s.CID
	}
	return workspaceContext{
		kind:       workspaceContextMonitor,
		title:      title,
		subtitle:   "Monitor",
		monitorCID: statsContainerID(s),
	}
}

func workspaceContextFromStats(s *docker.Stats, lookup monitorContainerLookup, series appui.MonitorSeries) workspaceContext {
	if s == nil {
		return workspaceContext{}
	}
	id := statsContainerID(s)
	title := s.Name
	if title == "" {
		title = s.CID
	}
	lines := []string{
		fmt.Sprintf("container: %s", s.CID),
		fmt.Sprintf("command: %s", s.Command),
		fmt.Sprintf("net io: %s / %s", units.BytesSize(s.NetworkRx), units.BytesSize(s.NetworkTx)),
		fmt.Sprintf("block io: %s / %s", units.BytesSize(s.BlockRead), units.BytesSize(s.BlockWrite)),
		fmt.Sprintf("pids: %d", s.PidsCurrent),
	}
	if s.Name != "" {
		lines = append([]string{fmt.Sprintf("name: %s", s.Name)}, lines...)
	}
	if c := workspaceMonitorContainer(lookup, id); c != nil {
		if name := workspaceContainerPrimaryName(c); name != "" && s.Name == "" {
			title = name
			lines = append([]string{fmt.Sprintf("name: %s", name)}, lines...)
		}
		if c.Status != "" {
			lines = append(lines, fmt.Sprintf("status: %s", c.Status))
		}
		if c.Image != "" {
			lines = append(lines, fmt.Sprintf("image: %s", c.Image))
		}
		if c.Detail.State != nil &&
			c.Detail.State.Health != nil && c.Detail.State.Health.Status != "" {
			lines = append(lines, fmt.Sprintf("health: %s", c.Detail.State.Health.Status))
		}
		if c.Detail.RestartCount > 0 {
			lines = append(lines, fmt.Sprintf("restarts: %d", c.Detail.RestartCount))
		}
		if ports := workspaceContainerPorts(c); ports != "" {
			lines = append(lines, fmt.Sprintf("ports: %s", ports))
		}
		if c.Detail.Config != nil {
			if c.Detail.Config.User != "" {
				lines = append(lines, fmt.Sprintf("user: %s", c.Detail.Config.User))
			}
			if c.Detail.Config.WorkingDir != "" {
				lines = append(lines, fmt.Sprintf("workdir: %s", c.Detail.Config.WorkingDir))
			}
		}
		if project := c.Labels["com.docker.compose.project"]; project != "" {
			lines = append(lines, fmt.Sprintf("compose project: %s", project))
		}
		if service := c.Labels["com.docker.compose.service"]; service != "" {
			lines = append(lines, fmt.Sprintf("compose service: %s", service))
		}
		if networks := workspaceContainerNetworkNames(c); networks != "" {
			lines = append(lines, fmt.Sprintf("network names: %s", networks))
		}
		if mounts := workspaceContainerMountTargets(c); mounts != "" {
			lines = append(lines, fmt.Sprintf("mount targets: %s", mounts))
		}
		if labels := len(c.Labels); labels > 0 {
			lines = append(lines, fmt.Sprintf("labels: %d", labels))
		}
	}
	lines = append(lines, workspaceDockerStatsLines(s.Stats)...)
	return workspaceContext{
		kind:              workspaceContextMonitor,
		title:             title,
		subtitle:          "Monitor",
		lines:             lines,
		monitorCID:        id,
		monitorCPU:        s.CPUPercentage,
		monitorMem:        s.Memory,
		monitorMax:        s.MemoryLimit,
		monitorPct:        s.MemoryPercentage,
		monitorCPUHistory: append([]appui.MonitorPoint(nil), series.CPU...),
		monitorMemHistory: append([]appui.MonitorPoint(nil), series.Memory...),
	}
}

func workspaceMonitorContainer(lookup monitorContainerLookup, id string) *docker.Container {
	if lookup == nil {
		return nil
	}
	return lookup.ContainerByID(id)
}

func workspaceContainerPrimaryName(c *docker.Container) string {
	if c == nil || len(c.Names) == 0 {
		return ""
	}
	return strings.TrimPrefix(c.Names[0], "/")
}

func workspaceContainerPorts(c *docker.Container) string {
	if c == nil || len(c.Ports) == 0 {
		return ""
	}
	var ports []string
	for _, p := range c.Ports {
		if p.PublicPort != 0 {
			ports = append(ports, fmt.Sprintf("%d->%d/%s", p.PublicPort, p.PrivatePort, p.Type))
			continue
		}
		ports = append(ports, fmt.Sprintf("%d/%s", p.PrivatePort, p.Type))
	}
	return strings.Join(ports, ", ")
}

func workspaceContextFromNode(n swarm.Node) workspaceContext {
	lines := []string{
		fmt.Sprintf("id: %s", docker.TruncateID(n.ID)),
		fmt.Sprintf("hostname: %s", n.Description.Hostname),
		fmt.Sprintf("role: %s", n.Spec.Role),
		fmt.Sprintf("availability: %s", n.Spec.Availability),
		fmt.Sprintf("status: %s", n.Status.State),
		fmt.Sprintf("cpu: %d", n.Description.Resources.NanoCPUs/1e9),
		fmt.Sprintf("memory: %s", units.BytesSize(float64(n.Description.Resources.MemoryBytes))),
	}
	if n.Description.Engine.EngineVersion != "" {
		lines = append(lines, fmt.Sprintf("engine: %s", n.Description.Engine.EngineVersion))
	}
	if n.Description.Platform.OS != "" || n.Description.Platform.Architecture != "" {
		lines = append(lines, fmt.Sprintf("platform: %s/%s", n.Description.Platform.OS, n.Description.Platform.Architecture))
	}
	if n.ManagerStatus != nil {
		lines = append(lines, fmt.Sprintf("manager: %s", n.ManagerStatus.Reachability))
		if n.ManagerStatus.Leader {
			lines = append(lines, "leader: true")
		}
		if n.ManagerStatus.Addr != "" {
			lines = append(lines, fmt.Sprintf("manager addr: %s", n.ManagerStatus.Addr))
		}
	}
	if len(n.Spec.Labels) > 0 {
		lines = append(lines, fmt.Sprintf("labels: %d", len(n.Spec.Labels)))
		lines = append(lines, fmt.Sprintf("label keys: %s", workspaceMapKeys(n.Spec.Labels, 6)))
	}
	return workspaceContext{
		title:    n.Description.Hostname,
		subtitle: "Node",
		lines:    lines,
		nodeID:   n.ID,
	}
}

func workspaceContextFromSwarmService(s swarm.Service) workspaceContext {
	mode := "global"
	replicas := "global"
	if s.Spec.Mode.Replicated != nil {
		mode = "replicated"
		if s.Spec.Mode.Replicated.Replicas != nil {
			replicas = fmt.Sprintf("%d", *s.Spec.Mode.Replicated.Replicas)
		} else {
			replicas = "0"
		}
	}
	imageRef := ""
	if s.Spec.TaskTemplate.ContainerSpec != nil {
		imageRef = s.Spec.TaskTemplate.ContainerSpec.Image
	}
	lines := []string{
		fmt.Sprintf("id: %s", docker.TruncateID(s.ID)),
		fmt.Sprintf("mode: %s", mode),
		fmt.Sprintf("replicas: %s", replicas),
	}
	if s.ServiceStatus != nil {
		lines = append(lines, fmt.Sprintf("tasks: %d/%d running", s.ServiceStatus.RunningTasks, s.ServiceStatus.DesiredTasks))
	}
	if imageRef != "" {
		lines = append(lines, fmt.Sprintf("image: %s", imageRef))
	}
	if spec := s.Spec.TaskTemplate.ContainerSpec; spec != nil {
		if len(spec.Command) > 0 {
			lines = append(lines, fmt.Sprintf("command: %s", workspaceJoinLimited(spec.Command, 4)))
		}
		if spec.User != "" {
			lines = append(lines, fmt.Sprintf("user: %s", spec.User))
		}
		if spec.Dir != "" {
			lines = append(lines, fmt.Sprintf("workdir: %s", spec.Dir))
		}
		if len(spec.Env) > 0 {
			lines = append(lines, fmt.Sprintf("env: %d vars", len(spec.Env)))
		}
		if len(spec.Mounts) > 0 {
			lines = append(lines, fmt.Sprintf("mounts: %d", len(spec.Mounts)))
		}
		if len(spec.Secrets) > 0 {
			lines = append(lines, fmt.Sprintf("secrets: %d", len(spec.Secrets)))
		}
		if len(spec.Configs) > 0 {
			lines = append(lines, fmt.Sprintf("configs: %d", len(spec.Configs)))
		}
		if len(spec.Labels) > 0 {
			lines = append(lines, fmt.Sprintf("container labels: %d", len(spec.Labels)))
		}
	}
	if len(s.Endpoint.Ports) > 0 {
		lines = append(lines, fmt.Sprintf("ports: %s", workspaceFormatSwarmPorts(s.Endpoint.Ports)))
	}
	if len(s.Spec.TaskTemplate.Networks) > 0 {
		lines = append(lines, fmt.Sprintf("networks: %d", len(s.Spec.TaskTemplate.Networks)))
	}
	if s.Spec.TaskTemplate.Placement != nil && len(s.Spec.TaskTemplate.Placement.Constraints) > 0 {
		lines = append(lines, fmt.Sprintf("constraints: %s", workspaceJoinLimited(s.Spec.TaskTemplate.Placement.Constraints, 4)))
	}
	if s.UpdateStatus != nil {
		lines = append(lines, fmt.Sprintf("update: %s", s.UpdateStatus.State))
		if s.UpdateStatus.Message != "" {
			lines = append(lines, fmt.Sprintf("update message: %s", s.UpdateStatus.Message))
		}
	}
	if s.Spec.UpdateConfig != nil {
		lines = append(lines, fmt.Sprintf("update policy: %s", s.Spec.UpdateConfig.FailureAction))
	}
	if s.Spec.RollbackConfig != nil {
		lines = append(lines, fmt.Sprintf("rollback policy: %s", s.Spec.RollbackConfig.FailureAction))
	}
	if len(s.Spec.Labels) > 0 {
		lines = append(lines, fmt.Sprintf("labels: %d", len(s.Spec.Labels)))
		lines = append(lines, fmt.Sprintf("label keys: %s", workspaceMapKeys(s.Spec.Labels, 6)))
	}
	return workspaceContext{
		title:     s.Spec.Name,
		subtitle:  "Swarm Service",
		lines:     lines,
		serviceID: s.ID,
	}
}

func workspaceContextFromStack(s docker.Stack) workspaceContext {
	lines := []string{
		fmt.Sprintf("orchestrator: %s", s.Orchestrator),
		fmt.Sprintf("services: %d", s.Services),
		fmt.Sprintf("networks: %d", s.Networks),
		fmt.Sprintf("configs: %d", s.Configs),
		fmt.Sprintf("secrets: %d", s.Secrets),
	}
	if s.Services > 0 {
		lines = append(lines, fmt.Sprintf("network ratio: %d services / %d networks", s.Services, workspaceMaxInt(s.Networks, 1)))
	}
	return workspaceContext{
		title:     s.Name,
		subtitle:  "Stack",
		stackName: s.Name,
		lines:     lines,
	}
}

func workspaceContextFromTask(t swarm.Task) workspaceContext {
	title := docker.TruncateID(t.ID)
	if t.Spec.ContainerSpec != nil && t.Spec.ContainerSpec.Hostname != "" {
		title = t.Spec.ContainerSpec.Hostname
	}
	lines := []string{
		fmt.Sprintf("id: %s", docker.TruncateID(t.ID)),
		fmt.Sprintf("desired: %s", t.DesiredState),
		fmt.Sprintf("current: %s", t.Status.State),
	}
	if !t.Status.Timestamp.IsZero() {
		lines = append(lines, fmt.Sprintf("updated: %s", workspaceFormatLocalTime(t.Status.Timestamp)))
	}
	if t.Slot != 0 {
		lines = append(lines, fmt.Sprintf("slot: %d", t.Slot))
	}
	if t.ServiceID != "" {
		lines = append(lines, fmt.Sprintf("service: %s", docker.TruncateID(t.ServiceID)))
	}
	if t.NodeID != "" {
		lines = append(lines, fmt.Sprintf("node: %s", docker.TruncateID(t.NodeID)))
	}
	if t.Spec.ContainerSpec != nil {
		if t.Spec.ContainerSpec.Image != "" {
			lines = append(lines, fmt.Sprintf("image: %s", t.Spec.ContainerSpec.Image))
		}
		if len(t.Spec.ContainerSpec.Command) > 0 {
			lines = append(lines, fmt.Sprintf("command: %s", workspaceJoinLimited(t.Spec.ContainerSpec.Command, 4)))
		}
	}
	if t.Status.ContainerStatus != nil {
		if t.Status.ContainerStatus.ContainerID != "" {
			lines = append(lines, fmt.Sprintf("container: %s", docker.TruncateID(t.Status.ContainerStatus.ContainerID)))
		}
		if t.Status.ContainerStatus.ExitCode != 0 {
			lines = append(lines, fmt.Sprintf("exit code: %d", t.Status.ContainerStatus.ExitCode))
		}
	}
	if len(t.NetworksAttachments) > 0 {
		lines = append(lines, fmt.Sprintf("networks: %d", len(t.NetworksAttachments)))
	}
	if len(t.Status.PortStatus.Ports) > 0 {
		lines = append(lines, fmt.Sprintf("ports: %s", workspaceFormatSwarmPorts(t.Status.PortStatus.Ports)))
	}
	if t.Status.Message != "" {
		lines = append(lines, fmt.Sprintf("message: %s", t.Status.Message))
	}
	if t.Status.Err != "" {
		lines = append(lines, fmt.Sprintf("error: %s", t.Status.Err))
	}
	return workspaceContext{
		title:    title,
		subtitle: "Task",
		lines:    lines,
		taskID:   t.ID,
	}
}

func workspaceContainerNetworkCount(c *docker.Container) int {
	if c == nil || c.Detail.NetworkSettings == nil {
		return 0
	}
	return len(c.Detail.NetworkSettings.Networks)
}

func workspaceRunningHealth(running, total int) string {
	if total == 0 {
		return "empty"
	}
	if running == total {
		return "healthy"
	}
	if running == 0 {
		return "stopped"
	}
	return "degraded"
}

func workspaceAttachedNetworkNames(n network.Inspect) string {
	names := make([]string, 0, len(n.Containers))
	for _, endpoint := range n.Containers {
		if endpoint.Name != "" {
			names = append(names, endpoint.Name)
		}
	}
	if len(names) == 0 {
		return ""
	}
	slices.Sort(names)
	if len(names) > 3 {
		return strings.Join(names[:3], ", ") + fmt.Sprintf(" +%d", len(names)-3)
	}
	return strings.Join(names, ", ")
}

func workspaceContainerNetworkNames(c *docker.Container) string {
	if c == nil || c.Detail.NetworkSettings == nil || len(c.Detail.NetworkSettings.Networks) == 0 {
		return ""
	}
	names := make([]string, 0, len(c.Detail.NetworkSettings.Networks))
	for name := range c.Detail.NetworkSettings.Networks {
		names = append(names, name)
	}
	slices.Sort(names)
	return workspaceJoinLimited(names, 6)
}

func workspaceContainerMountTargets(c *docker.Container) string {
	if c == nil || len(c.Detail.Mounts) == 0 {
		return ""
	}
	targets := make([]string, 0, len(c.Detail.Mounts))
	for _, mount := range c.Detail.Mounts {
		if mount.Destination != "" {
			targets = append(targets, mount.Destination)
		}
	}
	return workspaceJoinLimited(targets, 6)
}

func workspaceDockerStatsLines(stats *dockercontainer.StatsResponse) []string {
	if stats == nil {
		return nil
	}
	var lines []string
	if stats.Name != "" {
		lines = append(lines, fmt.Sprintf("stats.name: %s", strings.TrimPrefix(stats.Name, "/")))
	}
	if stats.ID != "" {
		lines = append(lines, fmt.Sprintf("stats.id: %s", stats.ID))
	}
	if !stats.Read.IsZero() {
		lines = append(lines, fmt.Sprintf("stats.read: %s", stats.Read.Local().Format("2006-01-02 15:04:05")))
	}
	if !stats.PreRead.IsZero() {
		lines = append(lines, fmt.Sprintf("stats.preread: %s", stats.PreRead.Local().Format("2006-01-02 15:04:05")))
	}
	lines = append(lines, workspacePidsStatsLines(stats.PidsStats)...)
	lines = append(lines, workspaceCPUStatsLines("cpu_stats", stats.CPUStats)...)
	lines = append(lines, workspaceCPUStatsLines("precpu_stats", stats.PreCPUStats)...)
	lines = append(lines, workspaceMemoryStatsLines(stats.MemoryStats)...)
	lines = append(lines, workspaceStorageStatsLines(stats.StorageStats)...)
	lines = append(lines, workspaceBlkioStatsLines(stats.BlkioStats)...)
	lines = append(lines, workspaceNetworkStatsLines(stats.Networks)...)
	return lines
}

func workspacePidsStatsLines(stats dockercontainer.PidsStats) []string {
	return []string{
		fmt.Sprintf("pids_stats.current: %d", stats.Current),
		fmt.Sprintf("pids_stats.limit: %d", stats.Limit),
	}
}

func workspaceCPUStatsLines(prefix string, stats dockercontainer.CPUStats) []string {
	lines := []string{
		fmt.Sprintf("%s.cpu_usage.total_usage: %s", prefix, workspaceFormatNanos(stats.CPUUsage.TotalUsage)),
		fmt.Sprintf("%s.cpu_usage.percpu_usage: %s", prefix, workspaceFormatUintSlice(stats.CPUUsage.PercpuUsage, 8)),
		fmt.Sprintf("%s.cpu_usage.usage_in_kernelmode: %s", prefix, workspaceFormatNanos(stats.CPUUsage.UsageInKernelmode)),
		fmt.Sprintf("%s.cpu_usage.usage_in_usermode: %s", prefix, workspaceFormatNanos(stats.CPUUsage.UsageInUsermode)),
		fmt.Sprintf("%s.system_cpu_usage: %s", prefix, workspaceFormatNanos(stats.SystemUsage)),
		fmt.Sprintf("%s.online_cpus: %d", prefix, stats.OnlineCPUs),
		fmt.Sprintf("%s.throttling.periods: %d", prefix, stats.ThrottlingData.Periods),
		fmt.Sprintf("%s.throttling.throttled_periods: %d", prefix, stats.ThrottlingData.ThrottledPeriods),
		fmt.Sprintf("%s.throttling.throttled_time: %s", prefix, workspaceFormatNanos(stats.ThrottlingData.ThrottledTime)),
	}
	return lines
}

func workspaceMemoryStatsLines(stats dockercontainer.MemoryStats) []string {
	lines := []string{
		fmt.Sprintf("memory_stats.usage: %s", workspaceFormatBytesValue(stats.Usage)),
		fmt.Sprintf("memory_stats.max_usage: %s", workspaceFormatBytesValue(stats.MaxUsage)),
		fmt.Sprintf("memory_stats.limit: %s", workspaceFormatBytesValue(stats.Limit)),
		fmt.Sprintf("memory_stats.failcnt: %d", stats.Failcnt),
		fmt.Sprintf("memory_stats.commitbytes: %s", workspaceFormatBytesValue(stats.Commit)),
		fmt.Sprintf("memory_stats.commitpeakbytes: %s", workspaceFormatBytesValue(stats.CommitPeak)),
		fmt.Sprintf("memory_stats.privateworkingset: %s", workspaceFormatBytesValue(stats.PrivateWorkingSet)),
	}
	if len(stats.Stats) > 0 {
		keys := make([]string, 0, len(stats.Stats))
		for key := range stats.Stats {
			keys = append(keys, key)
		}
		slices.Sort(keys)
		for _, key := range keys {
			lines = append(lines, fmt.Sprintf("memory_stats.stats.%s: %s", key, workspaceFormatBytesValue(stats.Stats[key])))
		}
	}
	return lines
}

func workspaceStorageStatsLines(stats dockercontainer.StorageStats) []string {
	return []string{
		fmt.Sprintf("storage_stats.read_count_normalized: %d", stats.ReadCountNormalized),
		fmt.Sprintf("storage_stats.read_size_bytes: %s", workspaceFormatBytesValue(stats.ReadSizeBytes)),
		fmt.Sprintf("storage_stats.write_count_normalized: %d", stats.WriteCountNormalized),
		fmt.Sprintf("storage_stats.write_size_bytes: %s", workspaceFormatBytesValue(stats.WriteSizeBytes)),
	}
}

func workspaceBlkioStatsLines(stats dockercontainer.BlkioStats) []string {
	var lines []string
	lines = append(lines, workspaceBlkioEntryLines("blkio_stats.io_service_bytes_recursive", stats.IoServiceBytesRecursive, true)...)
	lines = append(lines, workspaceBlkioEntryLines("blkio_stats.io_serviced_recursive", stats.IoServicedRecursive, false)...)
	lines = append(lines, workspaceBlkioEntryLines("blkio_stats.io_queue_recursive", stats.IoQueuedRecursive, false)...)
	lines = append(lines, workspaceBlkioEntryLines("blkio_stats.io_service_time_recursive", stats.IoServiceTimeRecursive, false)...)
	lines = append(lines, workspaceBlkioEntryLines("blkio_stats.io_wait_time_recursive", stats.IoWaitTimeRecursive, false)...)
	lines = append(lines, workspaceBlkioEntryLines("blkio_stats.io_merged_recursive", stats.IoMergedRecursive, false)...)
	lines = append(lines, workspaceBlkioEntryLines("blkio_stats.io_time_recursive", stats.IoTimeRecursive, false)...)
	lines = append(lines, workspaceBlkioEntryLines("blkio_stats.sectors_recursive", stats.SectorsRecursive, false)...)
	return lines
}

func workspaceBlkioEntryLines(prefix string, entries []dockercontainer.BlkioStatEntry, bytesValue bool) []string {
	if len(entries) == 0 {
		return []string{fmt.Sprintf("%s: none", prefix)}
	}
	lines := make([]string, 0, len(entries))
	for _, entry := range entries {
		key := fmt.Sprintf("%s.%d:%d.%s", prefix, entry.Major, entry.Minor, strings.ToLower(entry.Op))
		if bytesValue {
			lines = append(lines, fmt.Sprintf("%s: %s", key, workspaceFormatBytesValue(entry.Value)))
			continue
		}
		lines = append(lines, fmt.Sprintf("%s: %d", key, entry.Value))
	}
	return lines
}

func workspaceNetworkStatsLines(networks map[string]dockercontainer.NetworkStats) []string {
	if len(networks) == 0 {
		return []string{"networks: none"}
	}
	names := make([]string, 0, len(networks))
	for name := range networks {
		names = append(names, name)
	}
	slices.Sort(names)
	lines := make([]string, 0, len(names)*10)
	for _, name := range names {
		stats := networks[name]
		prefix := fmt.Sprintf("networks.%s", name)
		lines = append(lines,
			fmt.Sprintf("%s.rx_bytes: %s", prefix, workspaceFormatBytesValue(stats.RxBytes)),
			fmt.Sprintf("%s.rx_packets: %d", prefix, stats.RxPackets),
			fmt.Sprintf("%s.rx_errors: %d", prefix, stats.RxErrors),
			fmt.Sprintf("%s.rx_dropped: %d", prefix, stats.RxDropped),
			fmt.Sprintf("%s.tx_bytes: %s", prefix, workspaceFormatBytesValue(stats.TxBytes)),
			fmt.Sprintf("%s.tx_packets: %d", prefix, stats.TxPackets),
			fmt.Sprintf("%s.tx_errors: %d", prefix, stats.TxErrors),
			fmt.Sprintf("%s.tx_dropped: %d", prefix, stats.TxDropped),
		)
		if stats.EndpointID != "" {
			lines = append(lines, fmt.Sprintf("%s.endpoint_id: %s", prefix, stats.EndpointID))
		}
		if stats.InstanceID != "" {
			lines = append(lines, fmt.Sprintf("%s.instance_id: %s", prefix, stats.InstanceID))
		}
	}
	return lines
}

func workspaceJoinLimited(items []string, limit int) string {
	if len(items) == 0 {
		return ""
	}
	filtered := make([]string, 0, len(items))
	for _, item := range items {
		if strings.TrimSpace(item) != "" {
			filtered = append(filtered, item)
		}
	}
	if len(filtered) == 0 {
		return ""
	}
	if limit <= 0 || len(filtered) <= limit {
		return strings.Join(filtered, ", ")
	}
	return strings.Join(filtered[:limit], ", ") + fmt.Sprintf(" +%d", len(filtered)-limit)
}

func workspaceFormatSwarmPorts(ports []swarm.PortConfig) string {
	if len(ports) == 0 {
		return ""
	}
	values := make([]string, 0, len(ports))
	for _, port := range ports {
		if port.PublishedPort != 0 {
			values = append(values, fmt.Sprintf("%d->%d/%s", port.PublishedPort, port.TargetPort, port.Protocol))
		} else {
			values = append(values, fmt.Sprintf("%d/%s", port.TargetPort, port.Protocol))
		}
	}
	return workspaceJoinLimited(values, 6)
}

func workspaceFormatTimestamp(value string) string {
	if value == "" || strings.HasPrefix(value, "0001-01-01") {
		return ""
	}
	t, err := time.Parse(time.RFC3339Nano, value)
	if err != nil {
		return value
	}
	// Docker emits UTC timestamps; render in local time to stay consistent
	// with workspaceFormatUnix so e.g. "started" never appears before "created".
	return workspaceFormatLocalTime(t)
}

// workspaceFormatLocalTime renders a Docker-decoded (UTC) time.Time in local
// time. Every context-pane timestamp must go through this or
// workspaceFormatTimestamp so no pane mixes UTC and local renderings.
func workspaceFormatLocalTime(t time.Time) string {
	return t.Local().Format("2006-01-02 15:04")
}

func workspaceFormatUnix(ts int64) string {
	return time.Unix(ts, 0).Format("2006-01-02 15:04")
}

func workspaceFormatBytesValue(v uint64) string {
	return fmt.Sprintf("%d (%s)", v, units.BytesSize(float64(v)))
}

func workspaceFormatNanos(v uint64) string {
	return fmt.Sprintf("%d (%s)", v, time.Duration(v))
}

func workspaceFormatUintSlice(values []uint64, limit int) string {
	if len(values) == 0 {
		return "none"
	}
	parts := make([]string, 0, min(len(values), limit))
	for i, value := range values {
		if limit > 0 && i >= limit {
			parts = append(parts, fmt.Sprintf("+%d", len(values)-limit))
			break
		}
		parts = append(parts, fmt.Sprintf("%d", value))
	}
	return strings.Join(parts, ", ")
}

func workspaceMaxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}
