package compose

import (
	"sort"

	"github.com/moncho/dry/docker"
)

// mergeByName is a project's services in one name-ordered run: the ones
// with containers and the ones only the compose file knows about. The
// daemon already returns both projects and services sorted by name, so
// appending the file-only ones afterwards left two alphabetical runs in a
// list whose column says NAME. Name order is also the tiebreak once the
// view sorts the run on some other column.
func mergeByName(running, notCreated []docker.ComposeService) []docker.ComposeService {
	if len(notCreated) == 0 {
		return running
	}
	out := make([]docker.ComposeService, 0, len(running)+len(notCreated))
	out = append(out, running...)
	out = append(out, notCreated...)
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out
}

// notCreatedServices are the project's services that the compose file
// defines and no container exists for. Their names come from the drift
// check, which reports them because it compares the file's hashes against
// the containers; nothing else in these views knows they exist.
func notCreatedServices(project string, running []docker.ComposeService, sync map[string]docker.ServiceSync) []docker.ComposeService {
	have := make(map[string]struct{}, len(running))
	for _, svc := range running {
		have[svc.Name] = struct{}{}
	}
	var out []docker.ComposeService
	for name, status := range sync {
		if status != docker.ServiceNotCreated {
			continue
		}
		if _, ok := have[name]; ok {
			continue
		}
		// A nameless key would render a row with a blank name that every
		// key would then act on. The drift source drops these, so this is
		// belt and braces on a map dry does not own.
		if name == "" {
			continue
		}
		out = append(out, docker.ComposeService{Project: project, Name: name})
	}
	// Stable order, appended after the running services: a map iteration
	// would shuffle them between refreshes.
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out
}
