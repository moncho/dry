package docker

import "errors"

// Command represents a docker command
type Command int

const (
	//HISTORY Image history command
	HISTORY Command = iota
	//INSPECT Inspect command
	INSPECT
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

// ContainerCommands is the list of container commands
var ContainerCommands = []CommandDescription{
	{LOGS, "Fetch logs"},
	{INSPECT, "Inspect container"},
	{KILL, "Kill container"},
	{RM, "Remove container"},
	{RESTART, "Restart"},
	{HISTORY, "Show image history"},
	{STATS, "Stats + Top"},
	{STOP, "Stop"},
}

// CommandDescriptions lists command descriptions in the same order
// as they are found in ContainerCommands
var CommandDescriptions = justDescriptions(ContainerCommands)

// CommandDescription describes docker commands
type CommandDescription struct {
	Command     Command
	Description string
}

// CommandFromDescription returns the command with the given description, if any
func CommandFromDescription(d string) (Command, error) {
	for _, cc := range ContainerCommands {
		if cc.Description == d {
			return cc.Command, nil
		}
	}

	return -1, errors.New("Command for description not found")
}

// justDescriptions extract from the given list of commands a copy
// of the descriptions
func justDescriptions(commands []CommandDescription) []string {
	commandsLen := len(commands)
	descriptions := make([]string, commandsLen)
	for i := 0; i < commandsLen; i++ {
		descriptions[i] = commands[i].Description
	}
	return descriptions
}
