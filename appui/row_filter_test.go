package appui

import (
	"reflect"
	"testing"
)

func TestRowFilter_ByPattern(t *testing.T) {
	type args struct {
		pattern string
	}
	tests := []struct {
		name string
		rf   RowFilter
		args args
		want RowFilter
	}{
	// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.rf.ByPattern(tt.args.pattern); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("RowFilter.ByPattern() = %v, want %v", got, tt.want)
			}
		})
	}
}
