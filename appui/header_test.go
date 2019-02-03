package appui

import (
	"github.com/moncho/dry/ui/termui"
	"testing"
)

func TestWidgetHeader(t *testing.T) {
	type args struct {
		keys   []string
		values []string
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			"WidgetHeader content is valid",
			args{
				keys: []string{
					"1", "2", "3",
				},
				values: []string{
					"111", "222", "333",
				},
			},
			"1: 111 | 2: 222 | 3: 333",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			header := NewWidgetHeader()
			for i, key := range tt.args.keys {
				header.HeaderEntry(key, tt.args.values[i])
			}
			got, err := termui.String(header)

			if err != nil {
				t.Fatal(err)
			}
			if got != tt.want {
				t.Errorf("WidgetHeader got = '%s', want '%s'", got, tt.want)
			}
		})
	}
}
