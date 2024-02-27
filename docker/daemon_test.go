package docker

import (
	"context"
	"errors"
	"testing"
	"time"

	dockerTypes "github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/api/types/image"
	dockerAPI "github.com/docker/docker/client"
)

type imageAPIClientMock struct {
	dockerAPI.APIClient
	err           error
	imagesDeleted []image.DeleteResponse
}

func (c imageAPIClientMock) ImagesPrune(ctx context.Context, f filters.Args) (dockerTypes.ImagesPruneReport, error) {
	return dockerTypes.ImagesPruneReport{
		ImagesDeleted: c.imagesDeleted,
	}, c.err
}

type timedOutImageAPIClientMock struct {
	dockerAPI.APIClient
}

func (c timedOutImageAPIClientMock) ImagesPrune(ctx context.Context, f filters.Args) (dockerTypes.ImagesPruneReport, error) {
	ctx, cancel := context.WithTimeout(ctx, 500*time.Millisecond)
	defer cancel()
	select {
	case <-time.After(600 * time.Millisecond):
		return dockerTypes.ImagesPruneReport{}, nil
	case <-ctx.Done(): //This should always happen
		return dockerTypes.ImagesPruneReport{}, ctx.Err()
	}
}
func TestDockerDaemon_RemoveUnusedImages(t *testing.T) {
	type fields struct {
		client dockerAPI.APIClient
	}
	tests := []struct {
		name    string
		fields  fields
		want    int
		wantErr bool
	}{
		{
			"Remove unused images, 1 image deleted, no errors",
			fields{
				client: imageAPIClientMock{
					imagesDeleted: []image.DeleteResponse{
						{},
					},
				},
			},
			1,
			false,
		},
		{
			"Remove unused images fails",
			fields{
				client: imageAPIClientMock{
					err: errors.New("Not today"),
				},
			},
			0,
			true,
		},
		{
			"Remove unused imagess timeout",
			fields{
				client: timedOutImageAPIClientMock{},
			},
			0,
			true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			daemon := &DockerDaemon{
				client: tt.fields.client,
			}
			got, err := daemon.RemoveUnusedImages()
			if (err != nil) != tt.wantErr {
				t.Errorf("DockerDaemon.RemoveUnusedImages() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("DockerDaemon.RemoveUnusedImages() = %v, want %v", got, tt.want)
			}
		})
	}
}
