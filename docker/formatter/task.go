package formatter

import (
	"fmt"
	"strings"
	"time"

	"github.com/docker/distribution/reference"
	"github.com/docker/docker/api/types/swarm"
	"github.com/docker/docker/cli/command"
	"github.com/docker/docker/pkg/stringid"
	"github.com/docker/go-units"
)

const (
	maxErrLength = 30
)

//NewTaskStringer creates a TaskStringer for the given task
func NewTaskStringer(task swarm.Task, trunc bool) *TaskStringer {
	return &TaskStringer{task, trunc}
}

//TaskStringer converts to it string representation Task attributes
type TaskStringer struct {
	task  swarm.Task
	trunc bool
}

//ID Task id as a string
func (t *TaskStringer) ID() string {
	if t.trunc {
		return stringid.TruncateID(t.task.ID)
	}
	return t.task.ID
}

//Name Task name as a string
func (t *TaskStringer) Name() string {
	return t.task.ServiceID
}

//Image Task image as a string
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

//NodeID Task nodeID as a string
func (t *TaskStringer) NodeID() string {
	return t.task.NodeID
}

//DesiredState Task desired state as a string
func (t *TaskStringer) DesiredState() string {
	return command.PrettyPrint(t.task.DesiredState)
}

//CurrentState Task current state as a string
func (t *TaskStringer) CurrentState() string {
	return fmt.Sprintf("%s %s ago",
		command.PrettyPrint(t.task.Status.State),
		strings.ToLower(units.HumanDuration(time.Since(t.task.Status.Timestamp))),
	)
}

//Error Task status error as a string
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

//Ports Task ports as a string
func (t *TaskStringer) Ports() string {
	if len(t.task.Status.PortStatus.Ports) == 0 {
		return ""
	}
	ports := []string{}
	for _, pConfig := range t.task.Status.PortStatus.Ports {
		ports = append(ports, fmt.Sprintf("*:%d->%d/%s",
			pConfig.PublishedPort,
			pConfig.TargetPort,
			pConfig.Protocol,
		))
	}
	return strings.Join(ports, ",")
}
