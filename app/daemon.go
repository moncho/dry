package app

import "github.com/moncho/dry/docker"

// dockerDaemon is the app's aggregate of the docker sub-interfaces it wires
// into views and commands. The app is the only legitimate aggregator: every
// other consumer, from the appui models to the command constructors, depends
// on one of the narrow sub-interfaces instead. *docker.DockerDaemon satisfies
// this implicitly.
type dockerDaemon interface {
	docker.ContainerAPI
	docker.ImageAPI
	docker.NetworkAPI
	docker.VolumesAPI
	docker.SwarmAPI
	docker.ComposeAPI
	docker.ComposeActionsAPI
	docker.ContainerRuntime
	docker.SystemAPI
}
