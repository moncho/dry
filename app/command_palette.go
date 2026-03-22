package app

import (
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"
	"github.com/moncho/dry/appui"
	"github.com/moncho/dry/docker"
)

type paletteAction struct {
	Group       string
	ID          string
	Title       string
	Description string
	Search      string
}

func (m model) openCommandPalette() (tea.Model, tea.Cmd) {
	actions := m.commandPaletteActions()
	items := make([]appui.CommandPaletteItem, len(actions))
	for i, action := range actions {
		items[i] = appui.CommandPaletteItem{
			ID:          action.ID,
			Group:       action.Group,
			Title:       action.Title,
			Description: action.Description,
			Search:      action.Search,
		}
	}

	palette, cmd := appui.NewCommandPaletteModel(items)
	palette.SetSize(m.width, m.height)
	m.commandPalette = palette
	m.overlay = overlayCommandPalette
	return m, cmd
}

func (m model) commandPaletteActions() []paletteAction {
	actions := make([]paletteAction, 0, 40)
	add := func(group, id, title, desc, search string) {
		actions = append(actions, paletteAction{
			Group:       group,
			ID:          id,
			Title:       title,
			Description: desc,
			Search:      strings.TrimSpace(group + " " + search),
		})
	}

	switch m.view {
	case Main:
		if c := m.containers.SelectedContainer(); c != nil {
			label := paletteContainerLabel(c)
			add("Container", "container:commands", "Command Menu", label, "menu attach exec inspect")
			add("Container", "container:inspect", "Inspect", label, "inspect details")
			add("Container", "container:logs", "Logs", label, "logs output")
			add("Container", "container:stats", "Stats", label, "stats top usage")
			add("Container", "container:exec", "Exec Shell", label, "exec shell terminal")
			add("Container", "container:restart", "Restart", label, "restart")
			add("Container", "container:stop", "Stop", label, "stop")
			add("Container", "container:kill", "Kill", label, "kill")
			add("Container", "container:rm", "Remove", label, "rm delete")
		}
		add("Containers", "containers:rm-stopped", "Remove All Stopped", "", "prune stopped remove")
	case Images:
		if img := m.images.SelectedImage(); img != nil {
			label := shortID(img.ID)
			add("Image", "image:inspect", "Inspect", label, "inspect details")
			add("Image", "image:history", "History", label, "history layers")
			add("Image", "image:rm", "Remove", label, "remove delete")
			add("Image", "image:rm-force", "Force Remove", label, "force remove delete")
		}
		add("Images", "images:rm-dangling", "Remove Dangling", "", "dangling cleanup")
		add("Images", "images:rm-unused", "Remove Unused", "", "unused cleanup")
	case Networks:
		if n := m.networks.SelectedNetwork(); n != nil {
			add("Network", "network:inspect", "Inspect", n.Name, "inspect details")
			add("Network", "network:rm", "Remove", n.Name, "remove delete")
		}
	case Volumes:
		if v := m.volumes.SelectedVolume(); v != nil {
			add("Volume", "volume:inspect", "Inspect", v.Name, "inspect details")
			add("Volume", "volume:rm", "Remove", v.Name, "remove delete")
			add("Volume", "volume:rm-force", "Force Remove", v.Name, "force remove delete")
		}
		add("Volumes", "volumes:rm-all", "Remove All", "", "remove all")
		add("Volumes", "volumes:rm-unused", "Remove Unused", "", "prune unused")
	case Monitor:
		if s := m.monitor.SelectedStats(); s != nil {
			add("Monitor", "monitor:logs", "Open Container Logs", s.CID, "container logs")
			add("Monitor", "monitor:stats", "Open Container Stats", s.CID, "container stats")
			add("Monitor", "monitor:exec", "Exec Shell", s.CID, "container exec shell")
		}
	case Nodes:
		if n := m.nodes.SelectedNode(); n != nil {
			label := n.Description.Hostname
			if label == "" {
				label = shortID(n.ID)
			}
			add("Node", "node:tasks", "Show Tasks", label, "tasks")
			add("Node", "node:inspect", "Inspect", label, "inspect details")
			add("Node", "node:availability", "Cycle Availability", label, "availability drain pause active")
		}
	case Services:
		if s := m.services.SelectedService(); s != nil {
			add("Service", "service:tasks", "Show Tasks", s.Spec.Name, "tasks")
			add("Service", "service:inspect", "Inspect", s.Spec.Name, "inspect details")
			add("Service", "service:logs", "Logs", s.Spec.Name, "logs output")
			add("Service", "service:scale", "Scale", s.Spec.Name, "scale replicas")
			add("Service", "service:update", "Force Update", s.Spec.Name, "update rollout")
			add("Service", "service:rm", "Remove", s.Spec.Name, "remove delete")
		}
	case Stacks:
		if s := m.stacks.SelectedStack(); s != nil {
			add("Stack", "stack:tasks", "Show Tasks", s.Name, "tasks")
			add("Stack", "stack:rm", "Remove", s.Name, "remove delete")
		}
	case ComposeProjects:
		if svc := m.composeProjects.SelectedService(); svc != nil {
			label := svc.Project + "/" + svc.Name
			add("Compose Service", "compose-project-service:inspect", "Inspect", label, "inspect")
			add("Compose Service", "compose-project-service:logs", "Logs", label, "logs")
		}
		if p := m.composeProjects.SelectedProject(); p != nil {
			add("Compose Project", "compose-project:open", "Open Resources", p.Name, "open services")
			add("Compose Project", "compose-project:logs", "Logs", p.Name, "logs")
			add("Compose Project", "compose-project:stop", "Stop", p.Name, "stop")
			add("Compose Project", "compose-project:restart", "Restart", p.Name, "restart")
			add("Compose Project", "compose-project:rm", "Remove Containers", p.Name, "remove rm")
		}
	case ComposeServices:
		if svc := m.composeServices.SelectedService(); svc != nil {
			label := svc.Project + "/" + svc.Name
			add("Compose Service", "compose-service:inspect", "Inspect", label, "inspect")
			add("Compose Service", "compose-service:logs", "Logs", label, "logs")
			add("Compose Service", "compose-service:start", "Start", label, "start")
			add("Compose Service", "compose-service:stop", "Stop", label, "stop")
			add("Compose Service", "compose-service:restart", "Restart", label, "restart")
			add("Compose Service", "compose-service:rm", "Remove Containers", label, "remove rm")
		}
		if n := m.composeServices.SelectedNetwork(); n != nil {
			add("Compose Network", "compose-network:inspect", "Inspect", n.Name, "inspect")
		}
		if v := m.composeServices.SelectedVolume(); v != nil {
			add("Compose Volume", "compose-volume:inspect", "Inspect", v.Name, "inspect")
		}
	}

	if m.workspaceEnabled() {
		if m.pinnedContext != nil {
			add("Workspace", "workspace:pin-toggle", "Unpin Preview", "", "unpin unlock")
		} else if _, ok := m.currentWorkspacePreview(); ok {
			add("Workspace", "workspace:pin-toggle", "Pin Current Preview", "", "pin lock")
		}
		if action, ok := m.workspaceOpenInspectAction(); ok {
			actions = append(actions, action)
		}
		if action, ok := m.workspaceOpenLogsAction(); ok {
			actions = append(actions, action)
		}
	}

	add("Docker", "global:help", "Open Help", "", "help docs")
	add("Docker", "global:events", "Show Events", "", "events")
	add("Docker", "global:info", "Show Info", "", "info")
	add("Docker", "global:disk-usage", "Show Disk Usage", "", "disk usage")
	add("Docker", "global:prune", "Prune Unused Resources", "", "prune cleanup unused")
	add("Theme", "global:theme", "Cycle Color Theme", "", "color dark light")

	if m.view != Main {
		add("Go To", "switch:main", "Containers", "", "switch containers main")
	}
	if m.view != Images {
		add("Go To", "switch:images", "Images", "", "switch images")
	}
	if m.view != Networks {
		add("Go To", "switch:networks", "Networks", "", "switch networks")
	}
	if m.view != Volumes {
		add("Go To", "switch:volumes", "Volumes", "", "switch volumes")
	}
	if m.view != Monitor {
		add("Go To", "switch:monitor", "Monitor", "", "switch monitor stats")
	}
	if m.swarmMode && m.view != Nodes {
		add("Go To", "switch:nodes", "Nodes", "", "switch nodes swarm")
	}
	if m.swarmMode && m.view != Services {
		add("Go To", "switch:services", "Services", "", "switch services swarm")
	}
	if m.swarmMode && m.view != Stacks {
		add("Go To", "switch:stacks", "Stacks", "", "switch stacks swarm")
	}
	if m.view != ComposeProjects {
		add("Go To", "switch:compose-projects", "Compose Projects", "", "switch compose projects")
	}
	if m.view != DiskUsage {
		add("Go To", "switch:disk-usage", "Disk Usage", "", "switch disk usage")
	}

	return actions
}

func (m model) executePaletteAction(id string) (tea.Model, tea.Cmd) {
	switch id {
	case "global:help":
		return m, showHelpCmd()
	case "global:events":
		if m.daemon != nil {
			return m, showDockerEventsCmd(m.daemon)
		}
		return m, nil
	case "global:info":
		if m.daemon != nil {
			return m, showDockerInfoCmd(m.daemon)
		}
		return m, nil
	case "global:disk-usage", "switch:disk-usage":
		return m.switchView(DiskUsage)
	case "global:prune":
		return m.showPrompt("Prune all unused Docker resources?", "prune", ""), nil
	case "global:theme":
		m.rotateTheme()
		return m, nil
	case "workspace:pin-toggle":
		return m.toggleWorkspacePin()
	case "workspace:open-inspect":
		return m.executeWorkspaceOpenInspect()
	case "workspace:open-logs":
		return m.executeWorkspaceOpenLogs()
	case "switch:main":
		return m.switchView(Main)
	case "switch:images":
		return m.switchView(Images)
	case "switch:networks":
		return m.switchView(Networks)
	case "switch:volumes":
		return m.switchView(Volumes)
	case "switch:monitor":
		return m.switchView(Monitor)
	case "switch:nodes":
		return m.switchView(Nodes)
	case "switch:services":
		return m.switchView(Services)
	case "switch:stacks":
		return m.switchView(Stacks)
	case "switch:compose-projects":
		return m.switchView(ComposeProjects)
	case "containers:rm-stopped":
		return m.showPrompt("Remove all stopped containers?", "rm-all-stopped", ""), nil
	case "images:rm-dangling":
		return m.showPrompt("Remove dangling images?", "rmi-dangling", ""), nil
	case "images:rm-unused":
		return m.showPrompt("Remove unused images?", "rmi-unused", ""), nil
	case "volumes:rm-all":
		return m.showPrompt("Remove all volumes?", "vol-rm-all", ""), nil
	case "volumes:rm-unused":
		return m.showPrompt("Remove unused volumes?", "vol-prune", ""), nil
	case "container:commands":
		if c := m.containers.SelectedContainer(); c != nil {
			m.containerMenu = appui.NewContainerMenuModel(c)
			m.containerMenu.SetSize(m.width, m.height)
			m.overlay = overlayContainerMenu
		}
		return m, nil
	case "container:inspect":
		if c := m.containers.SelectedContainer(); c != nil {
			return m.executeMenuCommand(c.ID, docker.INSPECT)
		}
	case "container:logs":
		if c := m.containers.SelectedContainer(); c != nil {
			return m.executeMenuCommand(c.ID, docker.LOGS)
		}
	case "container:stats":
		if c := m.containers.SelectedContainer(); c != nil {
			return m.executeMenuCommand(c.ID, docker.STATS)
		}
	case "container:exec":
		if c := m.containers.SelectedContainer(); c != nil {
			return m.executeMenuCommand(c.ID, docker.EXEC)
		}
	case "container:restart":
		if c := m.containers.SelectedContainer(); c != nil {
			return m.executeMenuCommand(c.ID, docker.RESTART)
		}
	case "container:stop":
		if c := m.containers.SelectedContainer(); c != nil {
			return m.executeMenuCommand(c.ID, docker.STOP)
		}
	case "container:kill":
		if c := m.containers.SelectedContainer(); c != nil {
			return m.executeMenuCommand(c.ID, docker.KILL)
		}
	case "container:rm":
		if c := m.containers.SelectedContainer(); c != nil {
			return m.executeMenuCommand(c.ID, docker.RM)
		}
	case "image:inspect":
		if img := m.images.SelectedImage(); img != nil {
			return m, inspectImageCmd(m.daemon, img.ID)
		}
	case "image:history":
		if img := m.images.SelectedImage(); img != nil {
			return m, showImageHistoryCmd(m.daemon, img.ID)
		}
	case "image:rm":
		if img := m.images.SelectedImage(); img != nil {
			return m.showPrompt(
				fmt.Sprintf("Remove image %s?", docker.TruncateID(docker.ImageID(img.ID))),
				"rmi", img.ID,
			), nil
		}
	case "image:rm-force":
		if img := m.images.SelectedImage(); img != nil {
			return m.showPrompt(
				fmt.Sprintf("Force remove image %s?", docker.TruncateID(docker.ImageID(img.ID))),
				"rmi-force", img.ID,
			), nil
		}
	case "network:inspect":
		if n := m.networks.SelectedNetwork(); n != nil {
			return m, inspectNetworkCmd(m.daemon, n.ID)
		}
	case "network:rm":
		if n := m.networks.SelectedNetwork(); n != nil {
			return m.showPrompt(fmt.Sprintf("Remove network %s?", n.Name), "net-rm", n.ID), nil
		}
	case "volume:inspect":
		if v := m.volumes.SelectedVolume(); v != nil {
			return m, inspectVolumeCmd(m.daemon, v.Name)
		}
	case "volume:rm":
		if v := m.volumes.SelectedVolume(); v != nil {
			return m.showPrompt(fmt.Sprintf("Remove volume %s?", v.Name), "vol-rm", v.Name), nil
		}
	case "volume:rm-force":
		if v := m.volumes.SelectedVolume(); v != nil {
			return m.showPrompt(fmt.Sprintf("Force remove volume %s?", v.Name), "vol-rm-force", v.Name), nil
		}
	case "monitor:logs":
		if s := m.monitor.SelectedStats(); s != nil {
			return m, showContainerLogsCmd(m.daemon, s.CID)
		}
	case "monitor:stats":
		if s := m.monitor.SelectedStats(); s != nil {
			return m, showContainerStatsCmd(m.daemon, s.CID)
		}
	case "monitor:exec":
		if s := m.monitor.SelectedStats(); s != nil {
			var cmd tea.Cmd
			m.inputPrompt, cmd = appui.NewInputPromptModelWithLimit(
				fmt.Sprintf("Exec in %s:", shortID(s.CID)),
				"/bin/sh", "exec", s.CID, 120,
			)
			m.inputPrompt.SetSize(m.width, m.height)
			m.overlay = overlayInputPrompt
			return m, cmd
		}
	case "node:tasks":
		if n := m.nodes.SelectedNode(); n != nil {
			m.previousView = m.view
			m.view = Tasks
			return m, loadNodeTasksCmd(m.daemon, n.ID)
		}
	case "node:inspect":
		if n := m.nodes.SelectedNode(); n != nil {
			return m, inspectNodeCmd(m.daemon, n.ID)
		}
	case "node:availability":
		if n := m.nodes.SelectedNode(); n != nil {
			return m, m.cycleNodeAvailability(n.ID)
		}
	case "service:tasks":
		if s := m.services.SelectedService(); s != nil {
			m.previousView = m.view
			m.view = ServiceTasks
			return m, loadServiceTasksCmd(m.daemon, s.ID)
		}
	case "service:inspect":
		if s := m.services.SelectedService(); s != nil {
			return m, inspectServiceCmd(m.daemon, s.ID)
		}
	case "service:logs":
		if s := m.services.SelectedService(); s != nil {
			return m, showServiceLogsCmd(m.daemon, s.ID)
		}
	case "service:rm":
		if s := m.services.SelectedService(); s != nil {
			return m.showPrompt(fmt.Sprintf("Remove service %s?", s.Spec.Name), "service-rm", s.ID), nil
		}
	case "service:scale":
		if s := m.services.SelectedService(); s != nil {
			var cmd tea.Cmd
			m.inputPrompt, cmd = appui.NewInputPromptModel(
				fmt.Sprintf("Scale service %s to replicas:", s.Spec.Name),
				"number", "service-scale", s.ID,
			)
			m.inputPrompt.SetSize(m.width, m.height)
			m.overlay = overlayInputPrompt
			return m, cmd
		}
	case "service:update":
		if s := m.services.SelectedService(); s != nil {
			return m.showPrompt(fmt.Sprintf("Force update service %s?", s.Spec.Name), "service-update", s.ID), nil
		}
	case "stack:tasks":
		if s := m.stacks.SelectedStack(); s != nil {
			m.previousView = m.view
			m.view = StackTasks
			return m, loadStackTasksCmd(m.daemon, s.Name)
		}
	case "stack:rm":
		if s := m.stacks.SelectedStack(); s != nil {
			return m.showPrompt(fmt.Sprintf("Remove stack %s?", s.Name), "stack-rm", s.Name), nil
		}
	case "compose-project-service:inspect":
		if svc := m.composeProjects.SelectedService(); svc != nil {
			return m, inspectComposeServiceCmd(m.daemon, svc.Project, svc.Name)
		}
	case "compose-project-service:logs":
		if svc := m.composeProjects.SelectedService(); svc != nil {
			return m, showComposeLogsCmd(m.daemon, svc.Project, svc.Name)
		}
	case "compose-project:open":
		if p := m.composeProjects.SelectedProject(); p != nil {
			m.previousView = m.view
			m.view = ComposeServices
			m.selectedProject = p.Name
			return m, loadComposeServicesCmd(m.daemon, p.Name)
		}
	case "compose-project:logs":
		if p := m.composeProjects.SelectedProject(); p != nil {
			return m, showComposeLogsCmd(m.daemon, p.Name, "")
		}
	case "compose-project:stop":
		if p := m.composeProjects.SelectedProject(); p != nil {
			return m.showPrompt(fmt.Sprintf("Stop project %s?", p.Name), "compose-project-stop", p.Name), nil
		}
	case "compose-project:restart":
		if p := m.composeProjects.SelectedProject(); p != nil {
			return m.showPrompt(fmt.Sprintf("Restart project %s?", p.Name), "compose-project-restart", p.Name), nil
		}
	case "compose-project:rm":
		if p := m.composeProjects.SelectedProject(); p != nil {
			return m.showPrompt(fmt.Sprintf("Remove project %s containers?", p.Name), "compose-project-rm", p.Name), nil
		}
	case "compose-service:inspect":
		if svc := m.composeServices.SelectedService(); svc != nil {
			return m, inspectComposeServiceCmd(m.daemon, svc.Project, svc.Name)
		}
	case "compose-service:logs":
		if svc := m.composeServices.SelectedService(); svc != nil {
			return m, showComposeLogsCmd(m.daemon, svc.Project, svc.Name)
		}
	case "compose-service:start":
		if svc := m.composeServices.SelectedService(); svc != nil {
			return m.showPrompt(fmt.Sprintf("Start service %s?", svc.Name), "compose-start", svc.Project+"/"+svc.Name), nil
		}
	case "compose-service:stop":
		if svc := m.composeServices.SelectedService(); svc != nil {
			return m.showPrompt(fmt.Sprintf("Stop service %s?", svc.Name), "compose-stop", svc.Project+"/"+svc.Name), nil
		}
	case "compose-service:restart":
		if svc := m.composeServices.SelectedService(); svc != nil {
			return m.showPrompt(fmt.Sprintf("Restart service %s?", svc.Name), "compose-restart", svc.Project+"/"+svc.Name), nil
		}
	case "compose-service:rm":
		if svc := m.composeServices.SelectedService(); svc != nil {
			return m.showPrompt(fmt.Sprintf("Remove service %s containers?", svc.Name), "compose-rm", svc.Project+"/"+svc.Name), nil
		}
	case "compose-network:inspect":
		if n := m.composeServices.SelectedNetwork(); n != nil {
			return m, inspectNetworkCmd(m.daemon, n.Name)
		}
	case "compose-volume:inspect":
		if v := m.composeServices.SelectedVolume(); v != nil {
			return m, inspectVolumeCmd(m.daemon, v.Name)
		}
	}
	return m, nil
}

func paletteContainerLabel(c *docker.Container) string {
	if c == nil {
		return ""
	}
	if len(c.Names) > 0 {
		return strings.TrimPrefix(c.Names[0], "/")
	}
	return shortID(c.ID)
}

func (m model) workspacePaletteContext() (*workspaceContext, bool) {
	if m.pinnedContext != nil {
		return m.pinnedContext, true
	}
	ctx, ok := m.currentWorkspacePreview()
	if !ok {
		return nil, false
	}
	return &ctx, true
}

func (m model) workspaceOpenInspectAction() (paletteAction, bool) {
	ctx, ok := m.workspacePaletteContext()
	if !ok || ctx == nil {
		return paletteAction{}, false
	}
	switch {
	case ctx.containerID != "":
		return paletteAction{Group: "Workspace", ID: "workspace:open-inspect", Title: "Open Full Inspect", Description: ctx.title, Search: "workspace open full inspect container"}, true
	case ctx.imageID != "":
		return paletteAction{Group: "Workspace", ID: "workspace:open-inspect", Title: "Open Full Inspect", Description: ctx.title, Search: "workspace open full inspect image"}, true
	case ctx.networkID != "":
		return paletteAction{Group: "Workspace", ID: "workspace:open-inspect", Title: "Open Full Inspect", Description: ctx.title, Search: "workspace open full inspect network"}, true
	case ctx.volumeName != "":
		return paletteAction{Group: "Workspace", ID: "workspace:open-inspect", Title: "Open Full Inspect", Description: ctx.title, Search: "workspace open full inspect volume"}, true
	case ctx.nodeID != "":
		return paletteAction{Group: "Workspace", ID: "workspace:open-inspect", Title: "Open Full Inspect", Description: ctx.title, Search: "workspace open full inspect node"}, true
	case ctx.serviceID != "":
		return paletteAction{Group: "Workspace", ID: "workspace:open-inspect", Title: "Open Full Inspect", Description: ctx.title, Search: "workspace open full inspect service"}, true
	case ctx.taskID != "":
		return paletteAction{Group: "Workspace", ID: "workspace:open-inspect", Title: "Open Full Inspect", Description: ctx.title, Search: "workspace open full inspect task"}, true
	case ctx.project != "" && ctx.service != "":
		return paletteAction{Group: "Workspace", ID: "workspace:open-inspect", Title: "Open Full Inspect", Description: ctx.title, Search: "workspace open full inspect compose service"}, true
	case ctx.project != "":
		return paletteAction{Group: "Workspace", ID: "workspace:open-inspect", Title: "Open Project Resources", Description: ctx.title, Search: "workspace open compose project resources"}, true
	}
	return paletteAction{}, false
}

func (m model) workspaceOpenLogsAction() (paletteAction, bool) {
	ctx, ok := m.workspacePaletteContext()
	if !ok || ctx == nil {
		return paletteAction{}, false
	}
	switch {
	case ctx.containerID != "":
		return paletteAction{Group: "Workspace", ID: "workspace:open-logs", Title: "Open Full Logs", Description: ctx.title, Search: "workspace open full logs container"}, true
	case ctx.monitorCID != "":
		return paletteAction{Group: "Workspace", ID: "workspace:open-logs", Title: "Open Full Logs", Description: ctx.title, Search: "workspace open full logs monitor container"}, true
	case ctx.project != "" && ctx.service != "":
		return paletteAction{Group: "Workspace", ID: "workspace:open-logs", Title: "Open Full Logs", Description: ctx.title, Search: "workspace open full logs compose service"}, true
	case ctx.project != "":
		return paletteAction{Group: "Workspace", ID: "workspace:open-logs", Title: "Open Full Logs", Description: ctx.title, Search: "workspace open full logs compose project"}, true
	case ctx.serviceID != "":
		return paletteAction{Group: "Workspace", ID: "workspace:open-logs", Title: "Open Full Logs", Description: ctx.title, Search: "workspace open full logs service"}, true
	}
	return paletteAction{}, false
}

func (m model) executeWorkspaceOpenInspect() (tea.Model, tea.Cmd) {
	ctx, ok := m.workspacePaletteContext()
	if !ok || ctx == nil {
		return m, nil
	}
	switch {
	case ctx.containerID != "":
		return m, inspectContainerCmd(m.daemon, ctx.containerID)
	case ctx.imageID != "":
		return m, inspectImageCmd(m.daemon, ctx.imageID)
	case ctx.networkID != "":
		return m, inspectNetworkCmd(m.daemon, ctx.networkID)
	case ctx.volumeName != "":
		return m, inspectVolumeCmd(m.daemon, ctx.volumeName)
	case ctx.nodeID != "":
		return m, inspectNodeCmd(m.daemon, ctx.nodeID)
	case ctx.serviceID != "":
		return m, inspectServiceCmd(m.daemon, ctx.serviceID)
	case ctx.taskID != "":
		return m, inspectTaskCmd(m.daemon, ctx.taskID)
	case ctx.project != "" && ctx.service != "":
		return m, inspectComposeServiceCmd(m.daemon, ctx.project, ctx.service)
	case ctx.project != "":
		m.previousView = m.view
		m.view = ComposeServices
		m.selectedProject = ctx.project
		return m, loadComposeServicesCmd(m.daemon, ctx.project)
	}
	return m, nil
}

func (m model) executeWorkspaceOpenLogs() (tea.Model, tea.Cmd) {
	ctx, ok := m.workspacePaletteContext()
	if !ok || ctx == nil {
		return m, nil
	}
	switch {
	case ctx.containerID != "":
		return m, showContainerLogsCmd(m.daemon, ctx.containerID)
	case ctx.monitorCID != "":
		return m, showContainerLogsCmd(m.daemon, ctx.monitorCID)
	case ctx.project != "" && ctx.service != "":
		return m, showComposeLogsCmd(m.daemon, ctx.project, ctx.service)
	case ctx.project != "":
		return m, showComposeLogsCmd(m.daemon, ctx.project, "")
	case ctx.serviceID != "":
		return m, showServiceLogsCmd(m.daemon, ctx.serviceID)
	}
	return m, nil
}
