package docker

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/moby/moby/api/types/image"
	"github.com/moby/moby/client"
)

type imageAPIClientMock struct {
	client.APIClient
	err           error
	imagesDeleted []image.DeleteResponse
}

func (c imageAPIClientMock) ImagePrune(context.Context, client.ImagePruneOptions) (client.ImagePruneResult, error) {
	return client.ImagePruneResult{Report: image.PruneReport{
		ImagesDeleted: c.imagesDeleted,
	}}, c.err
}

type timedOutImageAPIClientMock struct {
	client.APIClient
}

func (c timedOutImageAPIClientMock) ImagePrune(ctx context.Context, _ client.ImagePruneOptions) (client.ImagePruneResult, error) {
	ctx, cancel := context.WithTimeout(ctx, 500*time.Millisecond)
	defer cancel()
	select {
	case <-time.After(600 * time.Millisecond):
		return client.ImagePruneResult{}, nil
	case <-ctx.Done(): // This should always happen
		return client.ImagePruneResult{}, ctx.Err()
	}
}

func TestDockerDaemon_RemoveUnusedImages(t *testing.T) {
	type fields struct {
		client client.APIClient
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
