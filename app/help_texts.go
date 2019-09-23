package app

import (
	"fmt"

	"github.com/moncho/dry/version"
)

//ShortHelp is a short description of dry
const ShortHelp = `
dry

A tool to interact with a Docker Daemon from the terminal. 
`

var help = `
<white>dry ` + fmt.Sprintf("version %s, build %s", version.VERSION, version.GITCOMMIT) + `</>` +
	`
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
	<white>s</>         Displays a live stream of the selected container resource usage statistics
	<white>Ctrl+t</>    Stops selected container (noop if it is not running)
	<white>Enter</>     Shows low-level information of the selected container

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
	<white>g</>         Moves the cursor to the beginning
	<white>G</>         Moves the cursor until the end
	<white>n</>         After a search, it moves forwards to the next search hit
	<white>N</>         After a search, it moves backwards to the previous search hit
	<white>pg up</>     Moves the cursor "screen size" lines up
	<white>pg down</>   Moves the cursor "screen size" lines down

<r> Press ESC to exit help. </r>
`

const (
	commonMappings = "<b>[H]:<darkgrey>Help</> <b>[Q]:<darkgrey>Quit</> <blue>|</> "
	keyMappings    = commonMappings +
		"<b>[F1]:<darkgrey>Sort</> <b>[F2]:<darkgrey>Toggle Show Containers</> <b>[F5]:<darkgrey>Refresh</> <b>[%]:<darkgrey>Filter</> <blue>|</> " +
		"<b>[m]:<darkgrey>Monitor mode</> <b>[2]:<darkgrey>Images</> <b>[3]:<darkgrey>Networks</> <b>[4]:<darkgrey>Volumes</> <b>[5]:<darkgrey>Nodes</> <b>[6]:<darkgrey>Services</> <b>[7]:<darkgrey>Stacks</> <blue>|</> <b>[Enter]:<darkgrey>Commands</></>"

	monitorMapping = commonMappings +
		"<b>[m]:<darkgrey>Monitor mode</> <b>[1]:<darkgrey>Containers</> <b>[2]:<darkgrey>Images</> <b>[3]:<darkgrey>Networks</> <b>[4]:<darkgrey>Volumes</> <b>[5]:<darkgrey>Nodes</> <b>[6]:<darkgrey>Services</> <b>[7]:<darkgrey>Stacks</> <blue>|</> <b>[s]:<darkgrey>Set refresh rate</></>"

	swarmMapping = commonMappings +
		"<b>[m]:<darkgrey>Monitor mode</> <b>[1]:<darkgrey>Containers</> <b>[2]:<darkgrey>Images</> <b>[3]:<darkgrey>Networks</> <b>[4]:<darkgrey>Volumes</> <b>[5]:<darkgrey>Nodes</> <b>[6]:<darkgrey>Services</> <b>[7]:<darkgrey>Stacks</>"

	imagesKeyMappings = commonMappings +
		"<b>[F1]:<darkgrey>Sort</> <b>[F5]:<darkgrey>Refresh</> <blue>|</> " +
		"<b>[1]:<darkgrey>Containers</> <b>[3]:<darkgrey>Networks</> <b>[4]:<darkgrey>Volumes</> <b>[5]:<darkgrey>Nodes</> <b>[6]:<darkgrey>Services</> <b>[7]:<darkgrey>Stacks</> <blue>|</>" +
		"<b>[Ctrl+D]:<darkgrey>Remove Dangling</> <b>[Ctrl+E]:<darkgrey>Remove</> <b>[Ctrl+F]:<darkgrey>Force Remove</> <b>[Ctrl+U]:<darkgrey>Remove Unused</> <b>[I]:<darkgrey>History</>"

	networkKeyMappings = commonMappings +
		"<b>[F1]:<darkgrey>Sort</> <b>[F5]:<darkgrey>Refresh</> <blue>|</> " +
		"<b>[1]:<darkgrey>Containers</> <b>[2]:<darkgrey>Images</> <b>[4]:<darkgrey>Volumes</> <b>[5]:<darkgrey>Nodes</> <b>[6]:<darkgrey>Services</> <b>[7]:<darkgrey>Stacks</> <blue>|</>" +
		"<b>[Ctrl+E]:<darkgrey>Remove</> <b>[Enter]:<darkgrey>Inspect</>"

	volumesKeyMappings = commonMappings +
		"<b>[F1]:<darkgrey>Sort</> <b>[F5]:<darkgrey>Refresh</> <blue>|</> " +
		"<b>[1]:<darkgrey>Containers</> <b>[2]:<darkgrey>Images</> <b>[3]:<darkgrey>Networks</> <b>[5]:<darkgrey>Nodes</> <b>[6]:<darkgrey>Services</> <b>[7]:<darkgrey>Stacks</> <blue>|</>" +
		"<b>[Ctrl+A]:<darkgrey>Remove All</> <b>[Ctrl+E]:<darkgrey>Remove</> <b>[Ctrl+F]:<darkgrey>Force Remove</> <b>[Ctrl+U]:<darkgrey>Remove Unused</> <b>[Enter]:<darkgrey>Inspect</>"

	diskUsageKeyMappings = commonMappings +
		"<b>[1]:<darkgrey>Containers</> <b>[2]:<darkgrey>Images</><blue>|</> <b>[3]:<darkgrey>Networks</> <b>[4]:<darkgrey>Volumes</> <b>[5]:<darkgrey>Nodes</> <b>[6]:<darkgrey>Services</> <b>[7]:<darkgrey>Stacks</> <blue>|</>" +
		"<b>[p]:<darkgrey>Prune</>"

	serviceKeyMappings = swarmMapping + "<blue>|</> <b>[F1]:<darkgrey>Sort</> <b>[F5]:<darkgrey>Refresh</> <b>[%]:<darkgrey>Filter</> <blue>|</> <b>[l]:<darkgrey>Service logs</> <b>[Ctrl+R]:<darkgrey>Remove Service</> <b>[Ctrl+S]:<darkgrey>Scale service</><b>[Ctrl+U]:<darkgrey>Update service</>"

	stackKeyMappings = swarmMapping + "<blue>|</> <b>[F1]:<darkgrey>Sort</> <b>[F5]:<darkgrey>Refresh</> <b>[%]:<darkgrey>Filter</> <blue>|</> <b>[Ctrl+R]:<darkgrey>Remove Stack</>"

	nodeKeyMappings = swarmMapping + " <blue>|</> <b>[F1]:<darkgrey>Sort</> <b>[F5]:<darkgrey>Refresh</> <blue>|</>  <b>[Enter]:<darkgrey>Show Node Tasks</> <b>[Ctrl+A]:<darkgrey>Set Availability</>"

	commandsMenuBar = "<b>[Esc]:<darkgrey>Back</> <b>[Up]:<darkgrey>Cursor Up</> <b>[Down]:<darkgrey>Cursor Down</> <b>[Enter]:<darkgrey>Execute Command</>"
)
