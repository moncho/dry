package docker

import (
	"sort"
	"testing"

	"github.com/docker/docker/api/types/swarm"
)

func TestSortServices(t *testing.T) {
	type args struct {
		services []swarm.Service
		mode     SortMode
	}
	tests := []struct {
		name string
		args args
	}{
		{
			"SortByServiceName",
			args{
				[]swarm.Service{
					{
						Spec: swarm.ServiceSpec{
							Annotations: swarm.Annotations{Name: "a"},
						},
					},
					{
						Spec: swarm.ServiceSpec{
							Annotations: swarm.Annotations{Name: "c"},
						},
					},
					{
						Spec: swarm.ServiceSpec{
							Annotations: swarm.Annotations{Name: "b"},
						},
					},
				},
				SortByServiceName,
			},
		},
		{
			"SortByServiceImage",
			args{
				[]swarm.Service{
					{
						Spec: swarm.ServiceSpec{
							Annotations: swarm.Annotations{Name: "b"},
							TaskTemplate: swarm.TaskSpec{
								ContainerSpec: &swarm.ContainerSpec{
									Image: "b",
								},
							},
						},
					},
					{
						Spec: swarm.ServiceSpec{
							Annotations: swarm.Annotations{Name: "c"},
							TaskTemplate: swarm.TaskSpec{
								ContainerSpec: &swarm.ContainerSpec{
									Image: "c",
								},
							},
						},
					},
					{
						Spec: swarm.ServiceSpec{
							Annotations: swarm.Annotations{Name: "a"},
							TaskTemplate: swarm.TaskSpec{
								ContainerSpec: &swarm.ContainerSpec{
									Image: "a",
								},
							},
						},
					},
				},
				SortByServiceImage,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			services := tt.args.services
			SortServices(services, tt.args.mode)

			if !sort.IsSorted(toSortInterface(services, tt.args.mode)) {
				t.Errorf("Unexpected service ordering %v", services)
			}
		})
	}
}

func toSortInterface(services []swarm.Service, mode SortMode) sort.Interface {

	switch mode {
	case SortByServiceName:
		return servicesByName{services}
	case SortByServiceImage:
		return servicesByImage{services}
	default:
		return nil
	}

}
