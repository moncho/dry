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
	<white>Enter</>     Shows the list of tasks of the selected stack
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
