package app

//ShortHelp is a short description of dry
const ShortHelp = `
dry

Connects to a Docker daemon if environment variable DOCKER_HOST is present
then shows the list of containers and allows some interaction with them.

`

const help = `
<white>dry</>

Connects to a Docker daemon if environment variable DOCKER_HOST is present
then shows the list of containers and allows some interaction with them.

<u>Command</u>    <u>Description                                </u>
	<white>F1</>       Cycles through containers sort modes (by Id | by Image | by Status | by Name)
	<white>F2</>       Toggles showing all containers (default shows just running)
	<white>F5</>       Refresh container list
	<white>k</>        Kills the selected container
	<white>l</>        Displays the logs of the selected container
	<white>r</>        Restarts selected container (noop if it is already running)
	<white>s</>        Displays a live stream of the selected container resource usage statistics
	<white>t</>        Stops selected container (noop if it is not running)
	<white>Enter</>    Returns low-level information on the selected container
	<white>q</>        Quits <white>dry</>.
	<white>esc</>      In the main view, quits <white>dry</>. In any other view, goes back to the main view


## Moving around in logs/inspect
	<white>g</>    		 Moves the cursor to the beggining
	<white>G</>   		 Moves the cursor until the end
	<white>pg up</>		 Moves the cursor "screen size" lines up
	<white>pg down</>	 Moves the cursor "screen size" lines down

<r> Press any key to continue </r>
`

const keyMappings = "<b>[H]:<darkgrey>Help</> <b>[Q]:<darkgrey>Quit</> <blue>|</> " +
	"<b>[F1]:<darkgrey>Sort</> <b>[F2]:<darkgrey>Toggle Show Containers</> <b>[F5]:<darkgrey>Refresh</> <blue>|</> " +
	"<b>[E]:<darkgrey>Remove</> <b>[K]:<darkgrey>Kill</> <b>[L]:<darkgrey>Logs</> <b>[R]:<darkgrey>Restart</> " +
	"<b>[S]:<darkgrey>Stats</> <b>[T]:<darkgrey>Stop</> <blue>|</>" +
	"<b>[Enter]:<darkgrey>Inspect</>"
