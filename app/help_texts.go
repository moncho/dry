package app

import "github.com/moncho/dry/version"

// ShortHelp is a short description of dry
const ShortHelp = `
dry

A tool to interact with a Docker Daemon from the terminal.
`

// Help returns the full help text with version info.
func Help() string {
	return `<white>dry version ` + version.VERSION + `, build ` + version.GITCOMMIT + `</>
A tool to interact with a Docker Daemon from the terminal.

Visit <blue>http://moncho.github.io/dry/</> for more information.

<yellow>Global keybinds</>
	<white>F7</>        Toggles showing Docker daemon information
	<white>F8</>        Shows Docker disk usage
	<white>F9</>        Shows the last 10 events reported by Docker
	<white>F10</>       Inspects Docker
	<white>1</>         To container list
	<white>2</>         To image list
	<white>3</>         To network list
	<white>4</>         To volumes list
	<white>5</>         To node list (in Swarm mode)
	<white>6</>         To service list (in Swarm mode)
	<white>7</>         To stack list (in Swarm mode)
	<white>m</>         Show container monitor mode
	<white>h</>         Shows this help screen
	<white>Ctrl+c</>    Quits <white>dry</> immediately
	<white>Q</>         Quits <white>dry</>
	<white>esc</>       Goes back to the main screen

<yellow>Global list keybinds</>	
	<white>F1</>        Cycles through sort modes
	<white>F5</>        Refreshes the list
	<white>%</>         Filter

<yellow>Container list keybinds</>
	<white>F2</>        Toggles showing all containers (default shows just running)
	<white>e</>         Removes the selected container
	<white>Ctrl+e</>    Removes all stopped containers
	<white>Ctrl+k</>    Kills the selected container
	<white>l</>         Displays the logs of the selected container
	<white>Ctrl+r</>    Restarts selected container
	<white>s</>         Displays resource usage statistics of the selected container
	<white>Ctrl+t</>    Stops selected container (noop if it is not running)
	<white>Enter</>     Opens the command menu for the selected container

<yellow>Image list keybinds</>
	<white>Ctrl+d</>    Removes dangling images
	<white>Ctrl+e</>    Removes the selected image
	<white>Ctrl+f</>    Forces removal of the selected image
	<white>Ctrl+u</>    Removes unused images
	<white>i</>         Shows image history
	<white>Enter</>     Shows low-level information of the selected image

<yellow>Network list keybinds</>
	<white>Enter</>     Shows low-level information of the selected network

<yellow>Node list keybinds</>
	<white>Enter</>     Shows the list of tasks running on the selected node

<yellow>Service list keybinds</>
	<white>Enter</>     Shows the list of tasks that are part of the selected service
	<white>l</>         Displays the logs of the selected service
	<white>Ctrl+R</>    Removes the selected service
	<white>Ctrl+S</>    Scales the selected service
	<white>Ctrl+U</>    Forces an update of the selected service

<yellow>Stack list keybinds</>
	<white>Enter</>     Shows the list of services of the selected stack
	<white>Ctrl+R</>    Removes the selected stack
	
<yellow>Move around in lists</>
	<white>ArrowUp</>   Moves the cursor one line up
	<white>ArrowDown</> Moves the cursor one line down
	<white>g</>         Moves the cursor to the beginning of the list
	<white>G</>         Moves the cursor to the end of the list

<yellow>Move around in logs/inspect buffers</>
	<white>/</>         Searches for a pattern
	<white>F</>         Only show lines that matches a pattern
	<white>f</>         Toggles follow mode (auto-scroll to bottom)
	<white>g</>         Moves the cursor to the beginning
	<white>G</>         Moves the cursor until the end
	<white>n</>         After a search, it moves forwards to the next search hit
	<white>N</>         After a search, it moves backwards to the previous search hit
	<white>pg up</>     Moves the cursor "screen size" lines up
	<white>pg down</>   Moves the cursor "screen size" lines down

<r> Press ESC to exit help. </r>
`
}

const (
	commonMappings = "<cyan>h</> <grey>help</>  <cyan>q</> <grey>quit</>  <darkgrey>·</> "
	keyMappings    = commonMappings +
		"<cyan>F1</> <grey>sort</>  <cyan>F2</> <grey>all/running</>  <cyan>F5</> <grey>refresh</>  <cyan>%</> <grey>filter</>  <darkgrey>·</> " +
		"<cyan>m</> <grey>monitor</>  <cyan>2</> <grey>images</>  <cyan>3</> <grey>nets</>  <cyan>4</> <grey>vols</>  <cyan>5</> <grey>nodes</>  <cyan>6</> <grey>svcs</>  <cyan>7</> <grey>stacks</>  <darkgrey>·</> <cyan>enter</> <grey>commands</>"

	monitorMapping = commonMappings +
		"<cyan>m</> <grey>monitor</>  <cyan>1</> <grey>containers</>  <cyan>2</> <grey>images</>  <cyan>3</> <grey>nets</>  <cyan>4</> <grey>vols</>  <cyan>5</> <grey>nodes</>  <cyan>6</> <grey>svcs</>  <cyan>7</> <grey>stacks</>"

	swarmMapping = commonMappings +
		"<cyan>m</> <grey>monitor</>  <cyan>1</> <grey>containers</>  <cyan>2</> <grey>images</>  <cyan>3</> <grey>nets</>  <cyan>4</> <grey>vols</>  <cyan>5</> <grey>nodes</>  <cyan>6</> <grey>svcs</>  <cyan>7</> <grey>stacks</>"

	imagesKeyMappings = commonMappings +
		"<cyan>F1</> <grey>sort</>  <cyan>F5</> <grey>refresh</>  <darkgrey>·</> " +
		"<cyan>1</> <grey>containers</>  <cyan>3</> <grey>nets</>  <cyan>4</> <grey>vols</>  <cyan>5</> <grey>nodes</>  <cyan>6</> <grey>svcs</>  <cyan>7</> <grey>stacks</>  <darkgrey>·</> " +
		"<cyan>^d</> <grey>rm dangling</>  <cyan>^e</> <grey>rm</>  <cyan>^f</> <grey>force rm</>  <cyan>^u</> <grey>rm unused</>  <cyan>i</> <grey>history</>"

	networkKeyMappings = commonMappings +
		"<cyan>F1</> <grey>sort</>  <cyan>F5</> <grey>refresh</>  <darkgrey>·</> " +
		"<cyan>1</> <grey>containers</>  <cyan>2</> <grey>images</>  <cyan>4</> <grey>vols</>  <cyan>5</> <grey>nodes</>  <cyan>6</> <grey>svcs</>  <cyan>7</> <grey>stacks</>  <darkgrey>·</> " +
		"<cyan>^e</> <grey>rm</>  <cyan>enter</> <grey>inspect</>"

	volumesKeyMappings = commonMappings +
		"<cyan>F1</> <grey>sort</>  <cyan>F5</> <grey>refresh</>  <darkgrey>·</> " +
		"<cyan>1</> <grey>containers</>  <cyan>2</> <grey>images</>  <cyan>3</> <grey>nets</>  <cyan>5</> <grey>nodes</>  <cyan>6</> <grey>svcs</>  <cyan>7</> <grey>stacks</>  <darkgrey>·</> " +
		"<cyan>^a</> <grey>rm all</>  <cyan>^e</> <grey>rm</>  <cyan>^f</> <grey>force rm</>  <cyan>^u</> <grey>rm unused</>  <cyan>enter</> <grey>inspect</>"

	diskUsageKeyMappings = commonMappings +
		"<cyan>1</> <grey>containers</>  <cyan>2</> <grey>images</>  <cyan>3</> <grey>nets</>  <cyan>4</> <grey>vols</>  <cyan>5</> <grey>nodes</>  <cyan>6</> <grey>svcs</>  <cyan>7</> <grey>stacks</>  <darkgrey>·</> " +
		"<cyan>p</> <grey>prune</>"

	serviceKeyMappings = swarmMapping + " <darkgrey>·</> <cyan>F1</> <grey>sort</>  <cyan>F5</> <grey>refresh</>  <cyan>%</> <grey>filter</>  <darkgrey>·</> <cyan>l</> <grey>logs</>  <cyan>^r</> <grey>rm</>  <cyan>^s</> <grey>scale</>  <cyan>^u</> <grey>update</>"

	stackKeyMappings = swarmMapping + " <darkgrey>·</> <cyan>F1</> <grey>sort</>  <cyan>F5</> <grey>refresh</>  <cyan>%</> <grey>filter</>  <darkgrey>·</> <cyan>^r</> <grey>rm stack</>"

	nodeKeyMappings = swarmMapping + " <darkgrey>·</> <cyan>F1</> <grey>sort</>  <cyan>F5</> <grey>refresh</>  <darkgrey>·</> <cyan>enter</> <grey>node tasks</>  <cyan>^a</> <grey>availability</>"

	commandsMenuBar = "<cyan>esc</> <grey>back</>  <cyan>↑↓</> <grey>navigate</>  <cyan>enter</> <grey>execute</>"
)
