package docker

//Stack contains information about a Docker Swarm stack
type Stack struct {
	Name     string
	Services int
	Networks int
	Configs  int
	Secrets  int
}
