package app

import "charm.land/bubbles/v2/key"

type globalKeyMap struct {
	Quit         key.Binding
	Help         key.Binding
	Containers   key.Binding
	Images       key.Binding
	Networks     key.Binding
	Volumes      key.Binding
	Nodes        key.Binding
	Services     key.Binding
	Stacks       key.Binding
	Monitor      key.Binding
	ToggleHeader key.Binding
	DiskUsage    key.Binding
	Events       key.Binding
	DockerInfo   key.Binding
}

var globalKeys = globalKeyMap{
	Quit: key.NewBinding(
		key.WithKeys("Q", "ctrl+c"),
		key.WithHelp("Q", "quit"),
	),
	Help: key.NewBinding(
		key.WithKeys("?", "h", "H"),
		key.WithHelp("h", "help"),
	),
	Containers: key.NewBinding(
		key.WithKeys("1"),
		key.WithHelp("1", "containers"),
	),
	Images: key.NewBinding(
		key.WithKeys("2"),
		key.WithHelp("2", "images"),
	),
	Networks: key.NewBinding(
		key.WithKeys("3"),
		key.WithHelp("3", "networks"),
	),
	Volumes: key.NewBinding(
		key.WithKeys("4"),
		key.WithHelp("4", "volumes"),
	),
	Nodes: key.NewBinding(
		key.WithKeys("5"),
		key.WithHelp("5", "nodes"),
	),
	Services: key.NewBinding(
		key.WithKeys("6"),
		key.WithHelp("6", "services"),
	),
	Stacks: key.NewBinding(
		key.WithKeys("7"),
		key.WithHelp("7", "stacks"),
	),
	Monitor: key.NewBinding(
		key.WithKeys("m", "M"),
		key.WithHelp("m", "monitor"),
	),
	ToggleHeader: key.NewBinding(
		key.WithKeys("f7"),
		key.WithHelp("F7", "toggle header"),
	),
	DiskUsage: key.NewBinding(
		key.WithKeys("f8"),
		key.WithHelp("F8", "disk usage"),
	),
	Events: key.NewBinding(
		key.WithKeys("f9"),
		key.WithHelp("F9", "events"),
	),
	DockerInfo: key.NewBinding(
		key.WithKeys("f10"),
		key.WithHelp("F10", "docker info"),
	),
}

// ---------------------------------------------------------------------------
// Per-view KeyMaps — provide ShortHelp/FullHelp for the footer key hints.
// ---------------------------------------------------------------------------

// --- containers (Main) ------------------------------------------------

type containerKeyMap struct {
	Help, Quit                                       key.Binding
	Sort, AllRunning, Refresh, Filter                key.Binding
	Monitor, Images, Nets, Vols, Nodes, Svcs, Stacks key.Binding
	Commands                                         key.Binding
	Logs, Stats, Rm, RmStopped, Kill, Restart, Stop  key.Binding
}

var containerKeys = containerKeyMap{
	Help:       key.NewBinding(key.WithKeys("h"), key.WithHelp("h", "help")),
	Quit:       key.NewBinding(key.WithKeys("Q"), key.WithHelp("q", "quit")),
	Sort:       key.NewBinding(key.WithKeys("f1"), key.WithHelp("F1", "sort")),
	AllRunning: key.NewBinding(key.WithKeys("f2"), key.WithHelp("F2", "all/running")),
	Refresh:    key.NewBinding(key.WithKeys("f5"), key.WithHelp("F5", "refresh")),
	Filter:     key.NewBinding(key.WithKeys("%"), key.WithHelp("%", "filter")),
	Monitor:    key.NewBinding(key.WithKeys("m"), key.WithHelp("m", "monitor")),
	Images:     key.NewBinding(key.WithKeys("2"), key.WithHelp("2", "images")),
	Nets:       key.NewBinding(key.WithKeys("3"), key.WithHelp("3", "nets")),
	Vols:       key.NewBinding(key.WithKeys("4"), key.WithHelp("4", "vols")),
	Nodes:      key.NewBinding(key.WithKeys("5"), key.WithHelp("5", "nodes")),
	Svcs:       key.NewBinding(key.WithKeys("6"), key.WithHelp("6", "svcs")),
	Stacks:     key.NewBinding(key.WithKeys("7"), key.WithHelp("7", "stacks")),
	Commands:   key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "commands")),
	Logs:       key.NewBinding(key.WithKeys("l"), key.WithHelp("l", "logs")),
	Stats:      key.NewBinding(key.WithKeys("s"), key.WithHelp("s", "stats")),
	Rm:         key.NewBinding(key.WithKeys("e"), key.WithHelp("e", "rm")),
	RmStopped:  key.NewBinding(key.WithKeys("ctrl+e"), key.WithHelp("^e", "rm stopped")),
	Kill:       key.NewBinding(key.WithKeys("ctrl+k"), key.WithHelp("^k", "kill")),
	Restart:    key.NewBinding(key.WithKeys("ctrl+r"), key.WithHelp("^r", "restart")),
	Stop:       key.NewBinding(key.WithKeys("ctrl+t"), key.WithHelp("^t", "stop")),
}

func (k containerKeyMap) ShortHelp() []key.Binding {
	return []key.Binding{
		k.Help, k.Quit,
		k.Sort, k.AllRunning, k.Refresh, k.Filter,
		k.Monitor, k.Images, k.Nets, k.Vols, k.Nodes, k.Svcs, k.Stacks,
		k.Commands, k.Logs, k.Stats,
		k.Rm, k.RmStopped, k.Kill, k.Restart, k.Stop,
	}
}

func (k containerKeyMap) FullHelp() [][]key.Binding { return [][]key.Binding{k.ShortHelp()} }

// --- monitor ----------------------------------------------------------

type monitorKeyMap struct {
	Help, Quit                                                   key.Binding
	Sort                                                         key.Binding
	Monitor, Containers, Images, Nets, Vols, Nodes, Svcs, Stacks key.Binding
}

var monitorKeys = monitorKeyMap{
	Help:       key.NewBinding(key.WithKeys("h"), key.WithHelp("h", "help")),
	Quit:       key.NewBinding(key.WithKeys("Q"), key.WithHelp("q", "quit")),
	Sort:       key.NewBinding(key.WithKeys("f1"), key.WithHelp("F1", "sort")),
	Monitor:    key.NewBinding(key.WithKeys("m"), key.WithHelp("m", "monitor")),
	Containers: key.NewBinding(key.WithKeys("1"), key.WithHelp("1", "containers")),
	Images:     key.NewBinding(key.WithKeys("2"), key.WithHelp("2", "images")),
	Nets:       key.NewBinding(key.WithKeys("3"), key.WithHelp("3", "nets")),
	Vols:       key.NewBinding(key.WithKeys("4"), key.WithHelp("4", "vols")),
	Nodes:      key.NewBinding(key.WithKeys("5"), key.WithHelp("5", "nodes")),
	Svcs:       key.NewBinding(key.WithKeys("6"), key.WithHelp("6", "svcs")),
	Stacks:     key.NewBinding(key.WithKeys("7"), key.WithHelp("7", "stacks")),
}

func (k monitorKeyMap) ShortHelp() []key.Binding {
	return []key.Binding{
		k.Help, k.Quit, k.Sort,
		k.Monitor, k.Containers, k.Images, k.Nets, k.Vols, k.Nodes, k.Svcs, k.Stacks,
	}
}

func (k monitorKeyMap) FullHelp() [][]key.Binding { return [][]key.Binding{k.ShortHelp()} }

// --- images -----------------------------------------------------------

type imagesKeyMap struct {
	Help, Quit                                  key.Binding
	Sort, Refresh, Filter                       key.Binding
	Containers, Nets, Vols, Nodes, Svcs, Stacks key.Binding
	Inspect                                     key.Binding
	RmDangling, Rm, ForceRm, RmUnused, History  key.Binding
}

var imagesKeys = imagesKeyMap{
	Help:       key.NewBinding(key.WithKeys("h"), key.WithHelp("h", "help")),
	Quit:       key.NewBinding(key.WithKeys("Q"), key.WithHelp("q", "quit")),
	Sort:       key.NewBinding(key.WithKeys("f1"), key.WithHelp("F1", "sort")),
	Refresh:    key.NewBinding(key.WithKeys("f5"), key.WithHelp("F5", "refresh")),
	Filter:     key.NewBinding(key.WithKeys("%"), key.WithHelp("%", "filter")),
	Containers: key.NewBinding(key.WithKeys("1"), key.WithHelp("1", "containers")),
	Nets:       key.NewBinding(key.WithKeys("3"), key.WithHelp("3", "nets")),
	Vols:       key.NewBinding(key.WithKeys("4"), key.WithHelp("4", "vols")),
	Nodes:      key.NewBinding(key.WithKeys("5"), key.WithHelp("5", "nodes")),
	Svcs:       key.NewBinding(key.WithKeys("6"), key.WithHelp("6", "svcs")),
	Stacks:     key.NewBinding(key.WithKeys("7"), key.WithHelp("7", "stacks")),
	Inspect:    key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "inspect")),
	RmDangling: key.NewBinding(key.WithKeys("ctrl+d"), key.WithHelp("^d", "rm dangling")),
	Rm:         key.NewBinding(key.WithKeys("ctrl+e"), key.WithHelp("^e", "rm")),
	ForceRm:    key.NewBinding(key.WithKeys("ctrl+f"), key.WithHelp("^f", "force rm")),
	RmUnused:   key.NewBinding(key.WithKeys("ctrl+u"), key.WithHelp("^u", "rm unused")),
	History:    key.NewBinding(key.WithKeys("i"), key.WithHelp("i", "history")),
}

func (k imagesKeyMap) ShortHelp() []key.Binding {
	return []key.Binding{
		k.Help, k.Quit, k.Sort, k.Refresh, k.Filter,
		k.Containers, k.Nets, k.Vols, k.Nodes, k.Svcs, k.Stacks,
		k.Inspect, k.RmDangling, k.Rm, k.ForceRm, k.RmUnused, k.History,
	}
}

func (k imagesKeyMap) FullHelp() [][]key.Binding { return [][]key.Binding{k.ShortHelp()} }

// --- networks ---------------------------------------------------------

type networksKeyMap struct {
	Help, Quit                                    key.Binding
	Sort, Refresh, Filter                         key.Binding
	Containers, Images, Vols, Nodes, Svcs, Stacks key.Binding
	Rm, Inspect                                   key.Binding
}

var networksKeys = networksKeyMap{
	Help:       key.NewBinding(key.WithKeys("h"), key.WithHelp("h", "help")),
	Quit:       key.NewBinding(key.WithKeys("Q"), key.WithHelp("q", "quit")),
	Sort:       key.NewBinding(key.WithKeys("f1"), key.WithHelp("F1", "sort")),
	Refresh:    key.NewBinding(key.WithKeys("f5"), key.WithHelp("F5", "refresh")),
	Filter:     key.NewBinding(key.WithKeys("%"), key.WithHelp("%", "filter")),
	Containers: key.NewBinding(key.WithKeys("1"), key.WithHelp("1", "containers")),
	Images:     key.NewBinding(key.WithKeys("2"), key.WithHelp("2", "images")),
	Vols:       key.NewBinding(key.WithKeys("4"), key.WithHelp("4", "vols")),
	Nodes:      key.NewBinding(key.WithKeys("5"), key.WithHelp("5", "nodes")),
	Svcs:       key.NewBinding(key.WithKeys("6"), key.WithHelp("6", "svcs")),
	Stacks:     key.NewBinding(key.WithKeys("7"), key.WithHelp("7", "stacks")),
	Rm:         key.NewBinding(key.WithKeys("ctrl+e"), key.WithHelp("^e", "rm")),
	Inspect:    key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "inspect")),
}

func (k networksKeyMap) ShortHelp() []key.Binding {
	return []key.Binding{
		k.Help, k.Quit, k.Sort, k.Refresh, k.Filter,
		k.Containers, k.Images, k.Vols, k.Nodes, k.Svcs, k.Stacks,
		k.Rm, k.Inspect,
	}
}

func (k networksKeyMap) FullHelp() [][]key.Binding { return [][]key.Binding{k.ShortHelp()} }

// --- volumes ----------------------------------------------------------

type volumesKeyMap struct {
	Help, Quit                                    key.Binding
	Sort, Refresh, Filter                         key.Binding
	Containers, Images, Nets, Nodes, Svcs, Stacks key.Binding
	RmAll, Rm, ForceRm, RmUnused, Inspect         key.Binding
}

var volumesKeys = volumesKeyMap{
	Help:       key.NewBinding(key.WithKeys("h"), key.WithHelp("h", "help")),
	Quit:       key.NewBinding(key.WithKeys("Q"), key.WithHelp("q", "quit")),
	Sort:       key.NewBinding(key.WithKeys("f1"), key.WithHelp("F1", "sort")),
	Refresh:    key.NewBinding(key.WithKeys("f5"), key.WithHelp("F5", "refresh")),
	Filter:     key.NewBinding(key.WithKeys("%"), key.WithHelp("%", "filter")),
	Containers: key.NewBinding(key.WithKeys("1"), key.WithHelp("1", "containers")),
	Images:     key.NewBinding(key.WithKeys("2"), key.WithHelp("2", "images")),
	Nets:       key.NewBinding(key.WithKeys("3"), key.WithHelp("3", "nets")),
	Nodes:      key.NewBinding(key.WithKeys("5"), key.WithHelp("5", "nodes")),
	Svcs:       key.NewBinding(key.WithKeys("6"), key.WithHelp("6", "svcs")),
	Stacks:     key.NewBinding(key.WithKeys("7"), key.WithHelp("7", "stacks")),
	RmAll:      key.NewBinding(key.WithKeys("ctrl+a"), key.WithHelp("^a", "rm all")),
	Rm:         key.NewBinding(key.WithKeys("ctrl+e"), key.WithHelp("^e", "rm")),
	ForceRm:    key.NewBinding(key.WithKeys("ctrl+f"), key.WithHelp("^f", "force rm")),
	RmUnused:   key.NewBinding(key.WithKeys("ctrl+u"), key.WithHelp("^u", "rm unused")),
	Inspect:    key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "inspect")),
}

func (k volumesKeyMap) ShortHelp() []key.Binding {
	return []key.Binding{
		k.Help, k.Quit, k.Sort, k.Refresh, k.Filter,
		k.Containers, k.Images, k.Nets, k.Nodes, k.Svcs, k.Stacks,
		k.RmAll, k.Rm, k.ForceRm, k.RmUnused, k.Inspect,
	}
}

func (k volumesKeyMap) FullHelp() [][]key.Binding { return [][]key.Binding{k.ShortHelp()} }

// --- disk usage -------------------------------------------------------

type diskUsageKeyMap struct {
	Help, Quit                                          key.Binding
	Containers, Images, Nets, Vols, Nodes, Svcs, Stacks key.Binding
	Prune                                               key.Binding
}

var diskUsageKeys = diskUsageKeyMap{
	Help:       key.NewBinding(key.WithKeys("h"), key.WithHelp("h", "help")),
	Quit:       key.NewBinding(key.WithKeys("Q"), key.WithHelp("q", "quit")),
	Containers: key.NewBinding(key.WithKeys("1"), key.WithHelp("1", "containers")),
	Images:     key.NewBinding(key.WithKeys("2"), key.WithHelp("2", "images")),
	Nets:       key.NewBinding(key.WithKeys("3"), key.WithHelp("3", "nets")),
	Vols:       key.NewBinding(key.WithKeys("4"), key.WithHelp("4", "vols")),
	Nodes:      key.NewBinding(key.WithKeys("5"), key.WithHelp("5", "nodes")),
	Svcs:       key.NewBinding(key.WithKeys("6"), key.WithHelp("6", "svcs")),
	Stacks:     key.NewBinding(key.WithKeys("7"), key.WithHelp("7", "stacks")),
	Prune:      key.NewBinding(key.WithKeys("p"), key.WithHelp("p", "prune")),
}

func (k diskUsageKeyMap) ShortHelp() []key.Binding {
	return []key.Binding{
		k.Help, k.Quit,
		k.Containers, k.Images, k.Nets, k.Vols, k.Nodes, k.Svcs, k.Stacks,
		k.Prune,
	}
}

func (k diskUsageKeyMap) FullHelp() [][]key.Binding { return [][]key.Binding{k.ShortHelp()} }

// --- services ---------------------------------------------------------

type servicesKeyMap struct {
	Help, Quit                                                   key.Binding
	Sort, Refresh, Filter                                        key.Binding
	Monitor, Containers, Images, Nets, Vols, Nodes, Svcs, Stacks key.Binding
	Tasks, Inspect, Logs, Rm, Scale, Update                      key.Binding
}

var servicesKeys = servicesKeyMap{
	Help:       key.NewBinding(key.WithKeys("h"), key.WithHelp("h", "help")),
	Quit:       key.NewBinding(key.WithKeys("Q"), key.WithHelp("q", "quit")),
	Monitor:    key.NewBinding(key.WithKeys("m"), key.WithHelp("m", "monitor")),
	Containers: key.NewBinding(key.WithKeys("1"), key.WithHelp("1", "containers")),
	Images:     key.NewBinding(key.WithKeys("2"), key.WithHelp("2", "images")),
	Nets:       key.NewBinding(key.WithKeys("3"), key.WithHelp("3", "nets")),
	Vols:       key.NewBinding(key.WithKeys("4"), key.WithHelp("4", "vols")),
	Nodes:      key.NewBinding(key.WithKeys("5"), key.WithHelp("5", "nodes")),
	Svcs:       key.NewBinding(key.WithKeys("6"), key.WithHelp("6", "svcs")),
	Stacks:     key.NewBinding(key.WithKeys("7"), key.WithHelp("7", "stacks")),
	Sort:       key.NewBinding(key.WithKeys("f1"), key.WithHelp("F1", "sort")),
	Refresh:    key.NewBinding(key.WithKeys("f5"), key.WithHelp("F5", "refresh")),
	Filter:     key.NewBinding(key.WithKeys("%"), key.WithHelp("%", "filter")),
	Tasks:      key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "tasks")),
	Inspect:    key.NewBinding(key.WithKeys("i"), key.WithHelp("i", "inspect")),
	Logs:       key.NewBinding(key.WithKeys("l"), key.WithHelp("l", "logs")),
	Rm:         key.NewBinding(key.WithKeys("ctrl+r"), key.WithHelp("^r", "rm")),
	Scale:      key.NewBinding(key.WithKeys("ctrl+s"), key.WithHelp("^s", "scale")),
	Update:     key.NewBinding(key.WithKeys("ctrl+u"), key.WithHelp("^u", "update")),
}

func (k servicesKeyMap) ShortHelp() []key.Binding {
	return []key.Binding{
		k.Help, k.Quit,
		k.Monitor, k.Containers, k.Images, k.Nets, k.Vols, k.Nodes, k.Svcs, k.Stacks,
		k.Sort, k.Refresh, k.Filter,
		k.Tasks, k.Inspect, k.Logs, k.Rm, k.Scale, k.Update,
	}
}

func (k servicesKeyMap) FullHelp() [][]key.Binding { return [][]key.Binding{k.ShortHelp()} }

// --- stacks -----------------------------------------------------------

type stacksKeyMap struct {
	Help, Quit                                                   key.Binding
	Sort, Refresh, Filter                                        key.Binding
	Monitor, Containers, Images, Nets, Vols, Nodes, Svcs, Stacks key.Binding
	Tasks, Rm                                                    key.Binding
}

var stacksKeys = stacksKeyMap{
	Help:       key.NewBinding(key.WithKeys("h"), key.WithHelp("h", "help")),
	Quit:       key.NewBinding(key.WithKeys("Q"), key.WithHelp("q", "quit")),
	Monitor:    key.NewBinding(key.WithKeys("m"), key.WithHelp("m", "monitor")),
	Containers: key.NewBinding(key.WithKeys("1"), key.WithHelp("1", "containers")),
	Images:     key.NewBinding(key.WithKeys("2"), key.WithHelp("2", "images")),
	Nets:       key.NewBinding(key.WithKeys("3"), key.WithHelp("3", "nets")),
	Vols:       key.NewBinding(key.WithKeys("4"), key.WithHelp("4", "vols")),
	Nodes:      key.NewBinding(key.WithKeys("5"), key.WithHelp("5", "nodes")),
	Svcs:       key.NewBinding(key.WithKeys("6"), key.WithHelp("6", "svcs")),
	Stacks:     key.NewBinding(key.WithKeys("7"), key.WithHelp("7", "stacks")),
	Sort:       key.NewBinding(key.WithKeys("f1"), key.WithHelp("F1", "sort")),
	Refresh:    key.NewBinding(key.WithKeys("f5"), key.WithHelp("F5", "refresh")),
	Filter:     key.NewBinding(key.WithKeys("%"), key.WithHelp("%", "filter")),
	Tasks:      key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "tasks")),
	Rm:         key.NewBinding(key.WithKeys("ctrl+r"), key.WithHelp("^r", "rm stack")),
}

func (k stacksKeyMap) ShortHelp() []key.Binding {
	return []key.Binding{
		k.Help, k.Quit,
		k.Monitor, k.Containers, k.Images, k.Nets, k.Vols, k.Nodes, k.Svcs, k.Stacks,
		k.Sort, k.Refresh, k.Filter,
		k.Tasks, k.Rm,
	}
}

func (k stacksKeyMap) FullHelp() [][]key.Binding { return [][]key.Binding{k.ShortHelp()} }

// --- nodes ------------------------------------------------------------

type nodesKeyMap struct {
	Help, Quit                                                   key.Binding
	Sort, Refresh, Filter                                        key.Binding
	Monitor, Containers, Images, Nets, Vols, Nodes, Svcs, Stacks key.Binding
	Tasks, Inspect, Availability                                 key.Binding
}

var nodesKeys = nodesKeyMap{
	Help:         key.NewBinding(key.WithKeys("h"), key.WithHelp("h", "help")),
	Quit:         key.NewBinding(key.WithKeys("Q"), key.WithHelp("q", "quit")),
	Monitor:      key.NewBinding(key.WithKeys("m"), key.WithHelp("m", "monitor")),
	Containers:   key.NewBinding(key.WithKeys("1"), key.WithHelp("1", "containers")),
	Images:       key.NewBinding(key.WithKeys("2"), key.WithHelp("2", "images")),
	Nets:         key.NewBinding(key.WithKeys("3"), key.WithHelp("3", "nets")),
	Vols:         key.NewBinding(key.WithKeys("4"), key.WithHelp("4", "vols")),
	Nodes:        key.NewBinding(key.WithKeys("5"), key.WithHelp("5", "nodes")),
	Svcs:         key.NewBinding(key.WithKeys("6"), key.WithHelp("6", "svcs")),
	Stacks:       key.NewBinding(key.WithKeys("7"), key.WithHelp("7", "stacks")),
	Sort:         key.NewBinding(key.WithKeys("f1"), key.WithHelp("F1", "sort")),
	Refresh:      key.NewBinding(key.WithKeys("f5"), key.WithHelp("F5", "refresh")),
	Filter:       key.NewBinding(key.WithKeys("%"), key.WithHelp("%", "filter")),
	Tasks:        key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "node tasks")),
	Inspect:      key.NewBinding(key.WithKeys("i"), key.WithHelp("i", "inspect")),
	Availability: key.NewBinding(key.WithKeys("ctrl+a"), key.WithHelp("^a", "availability")),
}

func (k nodesKeyMap) ShortHelp() []key.Binding {
	return []key.Binding{
		k.Help, k.Quit,
		k.Monitor, k.Containers, k.Images, k.Nets, k.Vols, k.Nodes, k.Svcs, k.Stacks,
		k.Sort, k.Refresh, k.Filter,
		k.Tasks, k.Inspect, k.Availability,
	}
}

func (k nodesKeyMap) FullHelp() [][]key.Binding { return [][]key.Binding{k.ShortHelp()} }

// --- tasks ------------------------------------------------------------

type tasksKeyMap struct {
	Help, Quit key.Binding
	Sort       key.Binding
	Back       key.Binding
}

var tasksKeys = tasksKeyMap{
	Help: key.NewBinding(key.WithKeys("h"), key.WithHelp("h", "help")),
	Quit: key.NewBinding(key.WithKeys("Q"), key.WithHelp("q", "quit")),
	Sort: key.NewBinding(key.WithKeys("f1"), key.WithHelp("F1", "sort")),
	Back: key.NewBinding(key.WithKeys("esc"), key.WithHelp("esc", "back")),
}

func (k tasksKeyMap) ShortHelp() []key.Binding {
	return []key.Binding{k.Help, k.Quit, k.Sort, k.Back}
}

func (k tasksKeyMap) FullHelp() [][]key.Binding { return [][]key.Binding{k.ShortHelp()} }
