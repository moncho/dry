package docker

import (
	"reflect"
	"strings"
	"testing"

	"github.com/moby/moby/api/types/container"
	"github.com/moby/moby/api/types/network"
)

func Test_containerConfigBuilder_build(t *testing.T) {
	type fields struct {
		image   string
		command string
		ports   map[string]struct{}
	}
	tests := []struct {
		name    string
		fields  fields
		want    container.Config
		want1   container.HostConfig
		wantErr bool
	}{
		{
			"no information given to the builder -> empty configs",
			fields{
				"",
				"",
				make(map[string]struct{}),
			},
			container.Config{},
			container.HostConfig{},
			false,
		},
		{
			"image and command given to the builder -> no errors",
			fields{
				"image",
				"command",
				make(map[string]struct{}),
			},
			container.Config{
				Image: "image",
				Cmd:   strings.Split("command", " "),
			},
			container.HostConfig{},
			false,
		},
		{
			"ports are set -> port configuration is created",
			fields{
				"image",
				"command",
				map[string]struct{}{
					"8080": {},
				},
			},
			container.Config{
				Image: "image",
				Cmd:   strings.Split("command", " "),
				ExposedPorts: network.PortSet{
					network.MustParsePort("8080"): {},
				},
			},
			container.HostConfig{
				PortBindings: network.PortMap{
					network.MustParsePort("8080/tcp"): {{
						HostPort: "8080",
					}},
				},
			},
			false,
		},
		{
			"invalid port set -> error is reported",
			fields{
				"image",
				"command",
				map[string]struct{}{
					"asd": {},
				},
			},
			container.Config{
				Image: "image",
				Cmd:   strings.Split("command", " "),
			},
			container.HostConfig{},
			true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cc := newCCB()
			cc = cc.command(tt.fields.command).
				image(tt.fields.image).
				ports(tt.fields.ports)
			got, got1, err := cc.build()
			if (err != nil) != tt.wantErr {
				t.Errorf("containerConfigBuilder.build() error = %v, wantErr %v", err, tt.wantErr)
				return
			} else if tt.wantErr == true {
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("containerConfigBuilder.build() config = %v, want %v", got, tt.want)
			}
			if !reflect.DeepEqual(got1, tt.want1) {
				t.Errorf("containerConfigBuilder.build() hostConfig = %v, want %v", got1, tt.want1)
			}
		})
	}
}
