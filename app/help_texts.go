package app

const help = `
<white>dry</>

Connects to a Docker daemon if environment variable DOCKER_HOST is present
then shows the list of containers and allows some interaction with them.

<u>Command</u>    <u>Description                                </u>
	<white>F1</>       Cycles through containers sort modes (by Id | by Image | by Status | by Name)
	<white>F2</>       Toggles showing all containers (default shows just running)
	<white>F5</>       Refresh container list
	<white>k</>        Kills the selected container
	<white>l</>        Fetch the logs of the selected container
	<white>r</>        Restarts selected container (noop if it is already running)
	<white>s</>        Displays a live stream of the selected container resource usage statistics
	<white>t</>        Stops selected container (noop if it is not running)
	<white>Enter</>    Returns low-level information on the selected container
	<white>q</>        Quits mop.
	<white>esc</>      Ditto.
<r> Press any key to continue </r>
`

const keyMappings = "H:<b><white>Help</></b> Q:<b><white>Quit</></b> <blue>|</> " +
	"F1:<b><white>Sort</></b> F2:<b><white>Toggle Show Containers</></b> F5:<b><white>Refresh</></b> <blue>|</> " +
	"E:<b><white>Remove</></b> K:<b><white>Kill</></b> L:<b><white>Logs</></b> R:<b><white>Restart</></b> " +
	"S:<b><white>Stats</></b> T:<b><white>Stop</></b> <blue>|</>" +
	"Intro:<b><white>Inspect</></b>"
