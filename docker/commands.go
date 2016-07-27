package docker

//Command represents a docker command
type Command int

const (

	//INSPECT Inspect command
	INSPECT Command = iota
	//KILL kill command
	KILL
	//LOGS logs command
	LOGS
	//RM remove command
	RM
	//RESTART restart command
	RESTART
	//STATS stats command
	STATS
	//STOP stop command
	STOP
)

//ContainerCommands is the list of container commands
var ContainerCommands = []CommandDescription{
	CommandDescription{INSPECT, " Display container low-level information"},
	CommandDescription{KILL, " Kill container"},
	CommandDescription{LOGS, " Fetch logs"},
	CommandDescription{RM, " Remove container"},
	CommandDescription{RESTART, " Restart"},
	CommandDescription{STATS, " Stats + Top"},
	CommandDescription{STOP, " Stop"},
}

//CommandDescriptions lists command descriptions in the same order
//as they are found in ContainerCommands
var CommandDescriptions = justDescriptions(ContainerCommands)

//CommandDescription describes docker commands
type CommandDescription struct {
	Command     Command
	Description string
}

//justDescriptions extract from the given list of commands a copy
//of the descriptions
func justDescriptions(commands []CommandDescription) []string {
	commandsLen := len(commands)
	descriptions := make([]string, commandsLen)
	for i := 0; i < commandsLen; i++ {
		descriptions[i] = commands[i].Description
	}
	return descriptions
}
