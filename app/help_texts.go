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
	<white>F8</>        Shows Docker disk usage
	<white>F9</>        Shows the last 10 events reported by Docker
	<white>F10</>       Inspects Docker
	<white>1</>         To container list
	<white>2</>         To image list
	<white>3</>         To network list
	<white>4</>         To node list (in Swarm mode)
	<white>5</>         To service list (in Swarm mode)
	<white>m</>         Show container monitor mode
	<white>h</>         Shows this help screen
	<white>Ctrl+c</>    Quits <white>dry</> immediately
	<white>q</>         Quits <white>dry</>
	<white>esc</>       Goes back to the main screen

<yellow>Container list keybinds</>
	<white>F1</>        Cycles through containers sort modes (by Id | by Image | by Status | by Name)
	<white>F2</>        Toggles showing all containers (default shows just running)
	<white>F3</>        Filters containers by its name	
	<white>F5</>        Refreshes container list
	<white>e</>         Removes the selected container
	<white>Ctrl+e</>    Removes all stopped containers
	<white>Ctrl+k</>    Kills the selected container
	<white>l</>         Displays the logs of the selected container
	<white>Ctrl+r</>    Restarts selected container
	<white>s</>         Displays a live stream of the selected container resource usage statistics
	<white>Ctrl+t</>    Stops selected container (noop if it is not running)
	<white>Enter</>     Returns low-level information of the selected container

<yellow>Image list keybinds</>
	<white>F1</>        Cycles through images sort modes (by Repo | by Id | by Creation date | by Size)
	<white>F5</>        Refresh the image list
	<white>Ctrl+e</>    Removes the selected image
	<white>Ctrl+f</>    Forces removal of the selected image
	<white>i</>         Shows image history
	<white>Enter</>     Returns low-level information of the selected image

<yellow>Network list keybinds</>
	<white>Enter</>     Returns low-level information of the selected network

<yellow>Node list keybinds</>
	<white>Enter</>     Shows the list of taks running on the selected node

<yellow>Service list keybinds</>
	<white>Enter</>     Shows the list of taks that are part of the selected service
	<white>l</>         Displays the logs of the selected service
	<white>Ctrl+R</>    Removes the selected service
	
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
		"<b>[F1]:<darkgrey>Sort</> <b>[F2]:<darkgrey>Toggle Show Containers</> <b>[F3]:<darkgrey>Filter(By Name)</> <b>[F5]:<darkgrey>Refresh</> <blue>|</> " +
		"<b>[m]:<darkgrey>Monitor mode</> <b>[2]:<darkgrey>Images</> <b>[3]:<darkgrey>Networks</> <b>[4]:<darkgrey>Nodes</> <b>[5]:<darkgrey>Services</> <blue>|</> <b>[Enter]:<darkgrey>Commands</></>"

	monitorMapping = commonMappings +
		"<b>[m]:<darkgrey>Monitor mode</> <b>[1]:<darkgrey>Containers</> <b>[2]:<darkgrey>Images</> <b>[3]:<darkgrey>Networks</> <b>[4]:<darkgrey>Nodes</> <b>[5]:<darkgrey>Services</>"

	swarmMapping = commonMappings +
		"<b>[m]:<darkgrey>Monitor mode</> <b>[1]:<darkgrey>Containers</> <b>[2]:<darkgrey>Images</> <b>[3]:<darkgrey>Networks</> <b>[4]:<darkgrey>Nodes</> <b>[5]:<darkgrey>Services</>"

	imagesKeyMappings = commonMappings +
		"<b>[F1]:<darkgrey>Sort</> <b>[F5]:<darkgrey>Refresh</> <blue>|</> " +
		"<b>[1]:<darkgrey>Containers</> <b>[3]:<darkgrey>Networks</> <b>[4]:<darkgrey>Nodes</> <b>[5]:<darkgrey>Services</> <blue>|</>" +
		"<b>[Ctrl+D]:<darkgrey>Remove Dangling</> <b>[Ctrl+E]:<darkgrey>Remove</> <b>[Ctrl+F]:<darkgrey>Force Remove</> <b>[I]:<darkgrey>History</>"

	networkKeyMappings = commonMappings +
		"<b>[F1]:<darkgrey>Sort</> <b>[F5]:<darkgrey>Refresh</> <blue>|</> " +
		"<b>[1]:<darkgrey>Containers</> <b>[2]:<darkgrey>Images</> <b>[4]:<darkgrey>Nodes</> <b>[5]:<darkgrey>Services</> <blue>|</>" +
		"<b>[Ctrl+E]:<darkgrey>Remove</> <b>[Enter]:<darkgrey>Inspect</>"

	diskUsageKeyMappings = commonMappings +
		"<b>[1]:<darkgrey>Containers</> <b>[2]:<darkgrey>Images</><blue>|</> <b>[3]:<darkgrey>Networks</> <b>[4]:<darkgrey>Nodes</> <b>[5]:<darkgrey>Services</> <blue>|</>" +
		"<b>[p]:<darkgrey>Prune</>"

	serviceKeyMappings = swarmMapping + " <blue>|</> <b>[F1]:<darkgrey>Sort</> <b>[F5]:<darkgrey>Refresh</> <blue>|</> <b>[l]:<darkgrey>Service logs</> <b>[Ctrl+R]:<darkgrey>Remove Service</> <b>[Ctrl+S]:<darkgrey>Scale service</>"

	nodeKeyMappings = swarmMapping + " <blue>|</> <b>[F1]:<darkgrey>Sort</> <b>[F5]:<darkgrey>Refresh</> <blue>|</>  <b>[Enter]:<darkgrey>Show Node Tasks</> <b>[Ctrl+A]:<darkgrey>Set Availability</>"

	commandsMenuBar = "<b>[Esc]:<darkgrey>Back</> <b>[Up]:<darkgrey>Cursor Up</> <b>[Down]:<darkgrey>Cursor Down</> <b>[Enter]:<darkgrey>Execute Command</>"
)
