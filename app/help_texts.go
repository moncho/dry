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
	<white>8</>         To compose projects list
	<white>m</>         Show container monitor mode
	<white>h</>         Shows this help screen
	<white>Ctrl+c</>    Quits <white>dry</> immediately
	<white>Q</>         Quits <white>dry</>
	<white>esc</>       Goes back to the main screen
	<white>Ctrl+0</>    Cycles color theme (dark/light)
	<white>:</>         Opens the command palette
	<white>Space</>     Opens Quick Peek for the current selection
	<white>Tab</>       Moves workspace focus forward between navigator, context, and activity
	<white>Shift+Tab</> Moves workspace focus backward between navigator, context, and activity
	<white>p/P</>       Toggles workspace pin/unpin for the current preview

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
	<white>x</>         Exec a command in the selected container (default /bin/sh)
	<white>Enter</>     Opens the command menu for the selected container (includes Attach)

<yellow>Image list keybinds</>
	<white>Ctrl+d</>    Removes dangling images
	<white>Ctrl+e</>    Removes the selected image
	<white>Ctrl+f</>    Forces removal of the selected image
	<white>Ctrl+u</>    Removes unused images
	<white>i</>         Shows image history
	<white>Enter</>     Shows low-level information of the selected image

<yellow>Network list keybinds</>
	<white>Ctrl+e</>    Removes the selected network
	<white>Enter</>     Shows low-level information of the selected network

<yellow>Volume list keybinds</>
	<white>Ctrl+a</>    Removes all volumes
	<white>Ctrl+e</>    Removes the selected volume
	<white>Ctrl+f</>    Forces removal of the selected volume
	<white>Ctrl+u</>    Removes unused volumes
	<white>Enter</>     Shows low-level information of the selected volume

<yellow>Node list keybinds</>
	<white>Enter</>     Shows the list of tasks running on the selected node
	<white>i</>         Shows low-level information of the selected node
	<white>Ctrl+a</>    Cycles the selected node availability

<yellow>Service list keybinds</>
	<white>Enter</>     Shows the list of tasks that are part of the selected service
	<white>i</>         Shows low-level information of the selected service
	<white>l</>         Displays the logs of the selected service
	<white>Ctrl+R</>    Removes the selected service
	<white>Ctrl+S</>    Scales the selected service
	<white>Ctrl+U</>    Forces an update of the selected service

<yellow>Stack list keybinds</>
	<white>Enter</>     Shows the list of tasks of the selected stack
	<white>Ctrl+R</>    Removes the selected stack

<yellow>Compose Projects</>
	<white>Enter</>     Shows the services of the selected project
	<white>l</>         Displays logs for the selected project or service
	<white>F1</>        Sorts the list
	<white>F5</>        Refreshes the list
	<white>%</>         Filters the list
	<white>Ctrl+t</>    Stop project containers
	<white>Ctrl+r</>    Restart project containers
	<white>Ctrl+e</>    Remove project containers

<yellow>Compose Services</>
	<white>Esc</>       Back to projects
	<white>Enter</>     Shows low-level information of the selected service, network, or volume
	<white>l</>         Displays logs of the selected service
	<white>F1</>        Sorts the list
	<white>%</>         Filters the list
	<white>Ctrl+s</>    Start service containers
	<white>Ctrl+t</>    Stop service containers
	<white>Ctrl+r</>    Restart service containers
	<white>Ctrl+e</>    Remove service containers

<yellow>Workspace activity</>
	<white>f</>         Toggles follow mode for embedded logs

<yellow>Quick Peek</>
	<white>Space</>     Opens or closes the Quick Peek panel
	<white>ArrowUp</>   Scrolls the preview up
	<white>ArrowDown</> Scrolls the preview down
	<white>g</>         Jumps to the beginning of the preview
	<white>G</>         Jumps to the end of the preview

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
