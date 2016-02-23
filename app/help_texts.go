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

<yellow>Basic commands</>
<u>Command</u>     <u>Description                                </u>
	<white>h</>         Shows this help screen
	<white>Crtl+c</>    Quits dry inmediately

<yellow>Container commands</>
<u>Command</u>     <u>Description                                </u>
	<white>F1</>        Cycles through containers sort modes (by Id | by Image | by Status | by Name)
	<white>F2</>        Toggles showing all containers (default shows just running)
	<white>F5</>        Refresh container list
	<white>F10</>       Inspects Docker
	<white>1</>         To image list
	<white>e</>         Removes the selected container
	<white>Crtl+e</>    Removes all stopped containers
	<white>Crtl+k</>    Kills the selected container
	<white>l</>         Displays the logs of the selected container
	<white>Ctrl+r</>         Restarts selected container
	<white>s</>         Displays a live stream of the selected container resource usage statistics
	<white>Crtl+t</>    Stops selected container (noop if it is not running)
	<white>Enter</>     Returns low-level information on the selected container
	<white>q</>         Quits <white>dry</>.
	<white>esc</>       In the main view, quits <white>dry</>. In any other view, goes back to the main view

<yellow>Image commands</>
<u>Command</u>     <u>Description                                </u>
	<white>F1</>        Cycles through images sort modes (by Repo | by Id | by Creation date | by Size)
	<white>F5</>        Refresh the image list
	<white>F10</>       Inspects Docker
	<white>1</>         To container list
	<white>Crtl+e</>    Removes the selected image
	<white>Crtl+f</>    Forces removal of the selected image
	<white>i</>         Shows image history
	<white>Enter</>     Returns low-level information on the selected image
	<white>q</>         Quits <white>dry</>.
	<white>esc</>       In the main view, quits <white>dry</>. In any other view, goes back to the main view

<yellow>Move around in container/image list</>
	<white>ArrowUp</>   Moves the cursor one line up
	<white>ArrowDown</> Moves the cursor uone line down

<yellow>Move around in logs/inspect></>
	<white>g</>         Moves the cursor to the beggining
	<white>G</>         Moves the cursor until the end
	<white>n</>         After a search, it moves forwards until the next search hit
	<white>N</>         After a search, it moves backwards until the next search hit
	<white>s</>         Searchs in the text being shown
	<white>pg up</>     Moves the cursor "screen size" lines up
	<white>pg down</>   Moves the cursor "screen size" lines down

<r> Press ESC to exit help. </r>
`

const (
	commonMappings = "<b>[H]:<darkgrey>Help</> <b>[Q]:<darkgrey>Quit</> <blue>|</> "
	inspectMapping = "<b>[Enter]:<darkgrey>Inspect</></>"
	keyMappings    = commonMappings +
		"<b>[F1]:<darkgrey>Sort</> <b>[F2]:<darkgrey>Toggle Show Containers</> <b>[F5]:<darkgrey>Refresh</> <b>[F10]:<darkgrey>Docker Info</> <blue>|</> " +
		"<b>[1]:<darkgrey>Images</> <b>[2]:<darkgrey>Networks</><blue>|</>" +
		"<b>[E]:<darkgrey>Remove</> <b>[Crtl+K]:<darkgrey>Kill</> <b>[L]:<darkgrey>Logs</> <b>[Ctrl+R]:<darkgrey>Restart</> " +
		"<b>[S]:<darkgrey>Stats</> <b>[Crtl+T]:<darkgrey>Stop</> <blue>|</>" +
		inspectMapping

	imagesKeyMappings = commonMappings +
		"<b>[F1]:<darkgrey>Sort</> <b>[F5]:<darkgrey>Refresh</> <b>[F10]:<darkgrey>Docker Info</> <blue>|</> " +
		"<b>[1]:<darkgrey>Containers</> <b>[2]:<darkgrey>Networks</> <blue>|</>" +
		"<b>[Crtl+E]:<darkgrey>Remove</> <b>[Crtl+F]:<darkgrey>Force Remove</> <b>[I]:<darkgrey>History</> <blue>|</>" +
		inspectMapping

	networkKeyMappings = commonMappings +
		"<b>[F1]:<darkgrey>Sort</> <b>[F5]:<darkgrey>Refresh</> <b>[F10]:<darkgrey>Docker Info</> <blue>|</> " +
		"<b>[1]:<darkgrey>Containers</> <b>[2]:<darkgrey>Images</><blue>|</>" +
		inspectMapping
)
