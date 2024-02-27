package formatter

import (
	"fmt"
	"strings"
	"time"

	"github.com/distribution/reference"
	"github.com/docker/docker/api/types/swarm"
	"github.com/docker/go-units"
	"github.com/moncho/dry/docker"
)

const (
	maxErrLength = 30
)

// NewTaskStringer creates a TaskStringer for the given task
func NewTaskStringer(api docker.SwarmAPI, task swarm.Task, trunc bool) *TaskStringer {
	return &TaskStringer{api, task, trunc}
}

// TaskStringer converts to it string representation Task attributes
type TaskStringer struct {
	api   docker.SwarmAPI
	task  swarm.Task
	trunc bool
}

// ID Task id as a string
func (t *TaskStringer) ID() string {
	if t.trunc {
		return TruncateID(t.task.ID)
	}
	return t.task.ID
}

// Name Task name as a string
func (t *TaskStringer) Name() string {

	if serviceName, err := t.api.ResolveService(t.task.ServiceID); err == nil {
		name := ""
		if t.task.Slot != 0 {
			name = fmt.Sprintf("%v.%v", serviceName, t.task.Slot)
		} else {
			name = fmt.Sprintf("%v.%v", serviceName, t.task.NodeID)
		}
		return name
	}
	return ""

}

// Image Task image as a string
func (t *TaskStringer) Image() string {
	image := t.task.Spec.ContainerSpec.Image
	if t.trunc {
		ref, err := reference.ParseNormalizedNamed(image)
		if err == nil {
			// update image string for display, (strips any digest)
			if nt, ok := ref.(reference.NamedTagged); ok {
				if namedTagged, err := reference.WithTag(reference.TrimNamed(nt), nt.Tag()); err == nil {
					image = reference.FamiliarString(namedTagged)
				}
			}
		}
	}
	return image
}

// NodeID Task nodeID as a string
func (t *TaskStringer) NodeID() string {
	if name, err := t.api.ResolveNode(t.task.NodeID); err == nil {
		return name
	}
	return ""
}

// DesiredState Task desired state as a string
func (t *TaskStringer) DesiredState() string {
	return PrettyPrint(t.task.DesiredState)
}

// CurrentState Task current state as a string
func (t *TaskStringer) CurrentState() string {
	return fmt.Sprintf("%s %s ago",
		PrettyPrint(t.task.Status.State),
		strings.ToLower(units.HumanDuration(time.Since(t.task.Status.Timestamp))),
	)
}

// Error Task status error as a string
func (t *TaskStringer) Error() string {
	// Trim and quote the error message.
	taskErr := t.task.Status.Err
	if t.trunc && len(taskErr) > maxErrLength {
		taskErr = fmt.Sprintf("%s…", taskErr[:maxErrLength-1])
	}
	if len(taskErr) > 0 {
		taskErr = fmt.Sprintf("\"%s\"", taskErr)
	}
	return taskErr
}

// Ports Task ports as a string
func (t *TaskStringer) Ports() string {
	if len(t.task.Status.PortStatus.Ports) == 0 {
		return ""
	}
	return FormatPorts(t.task.Status.PortStatus.Ports)
}
