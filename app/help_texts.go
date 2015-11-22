package app

const help = `
<white>dry</>

Connects to a Docker daemon if environment variable DOCKER_HOST (and DOCKER_TLS_VERIFY, and DOCKER_CERT_PATH) is present
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
	<white>q</>        Quits mop.
	<white>esc</>      Ditto.
<r> Press any key to continue </r>
`

const keyMappings = "H<b><white>elp</></b> Q<b><white>uit</></b> <blue>|</> " +
	"F1:<b><white>Sort</></b> F2:<b><white>Toggle Show Containers</></b> F5:<b><white>Refresh</></b> <blue>|</> " +
	"<b><white>R</></b>e<b><white>move</></b> K<b><white>ill</></b> L<b><white>ogs</></b> R<b><white>estart</></b> S<b><white>tats</></b> <b><white>S</></b>t<b><white>op</></b>"
