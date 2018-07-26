package swarm

import (
	"strconv"
	"testing"

	"github.com/moncho/dry/docker"
)

func TestStackRow(t *testing.T) {
	type args struct {
		stack docker.Stack
	}
	tests := []struct {
		name string
		args args
	}{
		{
			"stack row test",
			args{
				docker.Stack{
					Name:         "My name is my name",
					Services:     5,
					Orchestrator: "Swarm",
					Networks:     4,
					Configs:      2,
					Secrets:      300,
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stack := tt.args.stack
			row := NewStackRow(stack, nil)

			if row.Name.Text != stack.Name {
				t.Errorf("Unexpected Name. Got %s, expected %s", row.Name.Text, stack.Name)
			}
			if row.Services.Text != strconv.Itoa(stack.Services) {
				t.Errorf("Unexpected Services. Got %s, expected %d", row.Services.Text, stack.Services)
			}
			if row.Orchestrator.Text != stack.Orchestrator {
				t.Errorf("Unexpected Orchestrator. Got %s, expected %s", row.Orchestrator.Text, stack.Orchestrator)
			}
			if row.Networks.Text != strconv.Itoa(stack.Networks) {
				t.Errorf("Unexpected Networks. Got %s, expected %d", row.Networks.Text, stack.Networks)
			}
			if row.Configs.Text != strconv.Itoa(stack.Configs) {
				t.Errorf("Unexpected Configs. Got %s, expected %d", row.Configs.Text, stack.Configs)
			}
			if row.Secrets.Text != strconv.Itoa(stack.Secrets) {
				t.Errorf("Unexpected Secrets. Got %s, expected %d", row.Secrets.Text, stack.Secrets)
			}
		})
	}
}
