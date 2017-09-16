package docker

import (
	"context"
	"reflect"
	"sync"
	"testing"

	"github.com/docker/docker/api/types/events"
)

var noop = func(ctx context.Context, event events.Message) error { return nil }

func TestEventListeners_RegisterCallbacks(t *testing.T) {
	type fields struct {
		actions map[SourceType][]EventCallback
	}
	type args struct {
		source SourceType
		action EventCallback
	}
	tests := []struct {
		name     string
		args     []args
		expected fields
	}{
		{
			"Add one callback for a source type",
			[]args{
				args{
					ContainerSource,
					noop,
				},
			},
			fields{
				map[SourceType][]EventCallback{
					ContainerSource: []EventCallback{noop},
				},
			},
		},

		{
			"Add two callbacks for a source type",
			[]args{
				args{
					ContainerSource,
					noop,
				},
				args{
					ContainerSource,
					noop,
				},
			},
			fields{
				map[SourceType][]EventCallback{
					ContainerSource: []EventCallback{
						noop, noop,
					},
				},
			},
		},
		{
			"Add one callback for two source types",
			[]args{
				args{
					ContainerSource,
					noop,
				},
				args{
					ImageSource,
					noop,
				},
			},
			fields{
				map[SourceType][]EventCallback{
					ContainerSource: []EventCallback{
						noop,
					},
					ImageSource: []EventCallback{
						noop,
					},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			el := &registry{actions: make(map[SourceType][]EventCallback)}
			for _, args := range tt.args {
				el.Register(args.source, args.action)
			}

			if !eq(el.actions, tt.expected.actions) {
				t.Errorf("Adding events listener fails. Expected: %v, got: %v", tt.expected.actions, el.actions)
			}
		})
	}
}

//Checks if both map are equal, by checking the length, keys and number of actions per key
func eq(a, b map[SourceType][]EventCallback) bool {
	if len(a) != len(b) {
		return false
	}

	for k, v := range a {
		if w, ok := b[k]; !ok || len(v) != len(w) {
			return false
		}
	}

	return true
}

func Test_notifyCallbacks(t *testing.T) {

	var callback = func(
		wg *sync.WaitGroup,
		invocations map[SourceType]int) EventCallback {
		return func(ctx context.Context, event events.Message) error {
			defer wg.Done()
			t := SourceType(event.Type)
			invocations[t] = invocations[t] + 1
			return nil
		}
	}
	type args struct {
		r        *registry
		sources  []SourceType
		messages []events.Message
	}
	tests := []struct {
		name string
		args args
		want map[SourceType]int
	}{
		{
			"No messages, no calls to callbacks",
			args{
				r:        &registry{actions: make(map[SourceType][]EventCallback)},
				sources:  []SourceType{ContainerSource},
				messages: []events.Message{},
			},
			map[SourceType]int{},
		},
		{
			"One message, one callback called",
			args{
				r:       &registry{actions: make(map[SourceType][]EventCallback)},
				sources: []SourceType{ContainerSource},
				messages: []events.Message{
					events.Message{Type: "container"},
				},
			},
			map[SourceType]int{
				ContainerSource: 1,
			},
		},
		{
			"Two messages, two callbacks called",
			args{
				r:       &registry{actions: make(map[SourceType][]EventCallback)},
				sources: []SourceType{ContainerSource},
				messages: []events.Message{
					events.Message{Type: "container"},
					events.Message{Type: "container"},
				},
			},
			map[SourceType]int{
				ContainerSource: 2,
			},
		},
		{
			"Two messages of one type, one message of another type, three callbacks called",
			args{
				r:       &registry{actions: make(map[SourceType][]EventCallback)},
				sources: []SourceType{ContainerSource, ImageSource},
				messages: []events.Message{
					events.Message{Type: "container"},
					events.Message{Type: "container"},
					events.Message{Type: "image"},
				},
			},
			map[SourceType]int{
				ContainerSource: 2,
				ImageSource:     1,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			notify := notifyCallbacks(tt.args.r)
			invocations := make(map[SourceType]int)
			var wg sync.WaitGroup

			for _, s := range tt.args.sources {
				tt.args.r.Register(s, callback(&wg, invocations))
			}
			for _, m := range tt.args.messages {
				wg.Add(1)
				notify(context.Background(), m)
			}
			wg.Wait()
			if !reflect.DeepEqual(invocations, tt.want) {
				t.Errorf("notifyCallbacks() = %v, want %v", invocations, tt.want)
			}
		})
	}
}
