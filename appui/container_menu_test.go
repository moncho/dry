package appui

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/moncho/dry/ui/termui"
)

func TestContainerMenuWidget_OnEvent(t *testing.T) {
	type fields struct {
		rows          []*Row
		selectedIndex int
	}
	type args struct {
		event EventCommand
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr error
	}{
		{
			name: "happy path",
			fields: fields{
				rows: []*Row{
					{
						ParColumns: []*termui.ParColumn{
							{},
						},
					},
					{
						ParColumns: []*termui.ParColumn{
							{},
						},
					},
				},
				selectedIndex: 1,
			},
			args: args{
				event: func(string) error { return nil },
			},
			wantErr: nil,
		},
		{
			name:    "no rows in menu",
			wantErr: invalidRow{selected: 0, max: 0},
		},
		{
			name: "selected index is out of range",
			fields: fields{
				rows: []*Row{
					{},
				},
				selectedIndex: 2,
			},
			wantErr: invalidRow{selected: 2, max: 1},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &ContainerMenuWidget{
				rows:          tt.fields.rows,
				selectedIndex: tt.fields.selectedIndex,
			}
			err := s.OnEvent(tt.args.event)
			if diff := cmp.Diff(err, tt.wantErr, cmpopts.EquateErrors()); diff != "" {
				t.Errorf("ContainerMenuWidget.OnEvent() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
