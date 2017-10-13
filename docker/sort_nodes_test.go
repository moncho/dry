package docker

import (
	"testing"

	"github.com/docker/docker/api/types/swarm"
)

func TestSortNodes(t *testing.T) {
	type args struct {
		nodes []swarm.Node
		mode  SortMode
	}
	tests := []struct {
		name string
		args args
	}{
		{
			"Sort by name ",
			args{
				[]swarm.Node{
					{
						Description: swarm.NodeDescription{
							Hostname: "b",
						},
					},
					{
						Description: swarm.NodeDescription{
							Hostname: "a",
						},
					},
				},
				SortByNodeName,
			},
		},
		{
			"SortByNodeRole",
			args{
				[]swarm.Node{
					{
						Description: swarm.NodeDescription{
							Hostname: "b",
						},
						Spec: swarm.NodeSpec{
							Role: "b",
						},
					},
					{
						Description: swarm.NodeDescription{
							Hostname: "a",
						},
						Spec: swarm.NodeSpec{
							Role: "a",
						},
					},
				},
				SortByNodeRole,
			},
		},
		{
			"SortByNodeCPU ",
			args{
				[]swarm.Node{
					{
						Description: swarm.NodeDescription{
							Hostname:  "b",
							Resources: swarm.Resources{NanoCPUs: 2},
						},
					},
					{
						Description: swarm.NodeDescription{
							Hostname:  "a",
							Resources: swarm.Resources{NanoCPUs: 1},
						},
					},
				},
				SortByNodeCPU,
			},
		},
		{
			"SortByNodeMem ",
			args{
				[]swarm.Node{
					{
						Description: swarm.NodeDescription{
							Hostname:  "b",
							Resources: swarm.Resources{MemoryBytes: 2},
						},
					},
					{
						Description: swarm.NodeDescription{
							Hostname:  "a",
							Resources: swarm.Resources{MemoryBytes: 1},
						},
					},
				},
				SortByNodeMem,
			},
		},
		{
			"SortByNodeStatus ",
			args{
				[]swarm.Node{
					{
						Description: swarm.NodeDescription{
							Hostname: "b",
						},
						Status: swarm.NodeStatus{
							State: "b",
						},
					},
					{
						Description: swarm.NodeDescription{
							Hostname: "a",
						},
						Status: swarm.NodeStatus{
							State: "a",
						},
					},
				},
				SortByNodeStatus,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			nodes := tt.args.nodes
			SortNodes(nodes, tt.args.mode)
			if nodes[0].Description.Hostname != "a" {
				t.Errorf("Unexpected node as first element %v", nodes[0])
			}
			//Hostname is used to check ordering for any testcase
			if nodes[0].Description.Hostname > nodes[1].Description.Hostname {
				t.Errorf("Unexpected node ordering %v", nodes)
			}
		})
	}
}
