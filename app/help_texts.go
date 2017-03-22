package app

import (
	"fmt"

	"github.com/moncho/dry/version"
)

//ShortHelp is a short description of dry
const ShortHelp = `
dry

Connects to a Docker daemon if environment variable DOCKER_HOST is present
then shows the list of containers and allows to execute Docker commands on them.

`

var help = `
<white>dry ` + fmt.Sprintf("version %s, build %s", version.VERSION, version.GITCOMMIT) + `</>` +
	`
Connects to a Docker daemon, shows the list of containers and allows to execute Docker commands on them.

Visit <blue>http://moncho.github.io/dry/</> for more information.

<yellow>Global keybinds</>
	<white>F8</>        Shows Docker disk usage
	<white>F9</>        Shows the last 10 events reported by Docker
	<white>F10</>       Inspects Docker
	<white>1</>         To container list
	<white>2</>         To image list
	<white>3</>         To network list
	<white>m</>         To container monitor mode
	<white>h</>         Shows this help screen
	<white>Crtl+c</>    Quits <white>dry</> inmediately
	<white>q</>         Quits <white>dry</>
	<white>esc</>       Goes back to the main screen


<yellow>Container list keybinds</>
	<white>F1</>        Cycles through containers sort modes (by Id | by Image | by Status | by Name)
	<white>F2</>        Toggles showing all containers (default shows just running)
	<white>F3</>        Filters containers by its name	
	<white>F5</>        Refreshes container list
	<white>e</>         Removes the selected container
	<white>Crtl+e</>    Removes all stopped containers
	<white>Crtl+k</>    Kills the selected container
	<white>l</>         Displays the logs of the selected container
	<white>Ctrl+r</>    Restarts selected container
	<white>s</>         Displays a live stream of the selected container resource usage statistics
	<white>Crtl+t</>    Stops selected container (noop if it is not running)
	<white>Enter</>     Returns low-level information of the selected container

<yellow>Image list keybinds</>
	<white>F1</>        Cycles through images sort modes (by Repo | by Id | by Creation date | by Size)
	<white>F5</>        Refresh the image list
	<white>Crtl+e</>    Removes the selected image
	<white>Crtl+f</>    Forces removal of the selected image
	<white>i</>         Shows image history
	<white>Enter</>     Returns low-level information of the selected image

	<yellow>Network list keybinds</>
	<white>Enter</>     Returns low-level information of the selected network


<yellow>Move around in container/image/network lists</>
	<white>ArrowUp</>   Moves the cursor one line up
	<white>ArrowDown</> Moves the cursor one line down

<yellow>Move around in logs/inspect buffers</>
	<white>g</>         Moves the cursor to the beginning
	<white>G</>         Moves the cursor until the end
	<white>n</>         After a search, it moves forwards to the next search hit
	<white>N</>         After a search, it moves backwards to the previous search hit
	<white>s</>         Searches in the text being shown
	<white>pg up</>     Moves the cursor "screen size" lines up
	<white>pg down</>   Moves the cursor "screen size" lines down

<r> Press ESC to exit help. </r>
`

const (
	commonMappings = "<b>[H]:<darkgrey>Help</> <b>[Q]:<darkgrey>Quit</> <blue>|</> "
	keyMappings    = commonMappings +
		"<b>[F1]:<darkgrey>Sort</> <b>[F2]:<darkgrey>Toggle Show Containers</> <b>[F3]:<darkgrey>Filter(By Name)</> <b>[F5]:<darkgrey>Refresh</> <blue>|</> " +
		"<b>[m]:<darkgrey>Monitor mode</> <b>[2]:<darkgrey>Images</> <b>[3]:<darkgrey>Networks</> <blue>|</> <b>[Enter]:<darkgrey>Commands</></>"

	monitorMapping = commonMappings +
		"<b>[1]:<darkgrey>Containers</> <b>[2]:<darkgrey>Images</> <b>[3]:<darkgrey>Networks</> <blue>"

	imagesKeyMappings = commonMappings +
		"<b>[F1]:<darkgrey>Sort</> <b>[F5]:<darkgrey>Refresh</> <blue>|</> " +
		"<b>[1]:<darkgrey>Containers</> <b>[3]:<darkgrey>Networks</> <blue>|</>" +
		"<b>[Crtl+D]:<darkgrey>Remove Dangling</> <b>[Crtl+E]:<darkgrey>Remove</> <b>[Crtl+F]:<darkgrey>Force Remove</> <b>[I]:<darkgrey>History</>"

	networkKeyMappings = commonMappings +
		"<b>[F1]:<darkgrey>Sort</> <b>[F5]:<darkgrey>Refresh</> <blue>|</> " +
		"<b>[1]:<darkgrey>Containers</> <b>[2]:<darkgrey>Images</><blue>|</>" +
		"<b>[Crtl+E]:<darkgrey>Remove</> <b>[Enter]:<darkgrey>Inspect</>"

	diskUsageKeyMappings = commonMappings +
		"<b>[1]:<darkgrey>Containers</> <b>[2]:<darkgrey>Images</><blue>|</> <b>[3]:<darkgrey>Networks</> <blue>|</>" +
		"<b>[p]:<darkgrey>Prune</>"

	commandsMenuBar = "<b>[Esc]:<darkgrey>Back</> <b>[Up]:<darkgrey>Cursor Up</> <b>[Down]:<darkgrey>Cursor Down</> <b>[Intro]:<darkgrey>Execute Command</>"
)
