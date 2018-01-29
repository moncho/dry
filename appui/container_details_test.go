package appui

import (
	"testing"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/network"
	"github.com/moncho/dry/docker"
)

func TestNewContainerDetailsWidget(t *testing.T) {
	type args struct {
		container *docker.Container
		y         int
	}
	tests := []struct {
		name string
		args args
	}{
		{
			"a widget with container details can be created",
			args{
				&docker.Container{
					Container: types.Container{
						Names: []string{"container1"},
						NetworkSettings: &types.SummaryNetworkSettings{
							Networks: make(map[string]*network.EndpointSettings),
						},
					},
					ContainerJSON: types.ContainerJSON{
						NetworkSettings: &types.NetworkSettings{
							Networks: make(map[string]*network.EndpointSettings),
						},
					},
				},
				0,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := NewContainerDetailsWidget(tt.args.container, tt.args.y)

			if w == nil {
				t.Error("ContainerDetailsWidget was not created")
			}
		})
	}
}
