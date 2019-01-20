package app

import "testing"

func Test_curateLogsDuration(t *testing.T) {
	type args struct {
		s string
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			"-2",
			args{
				"-2",
			},
			"2",
		},
		{
			"2",
			args{
				"2",
			},
			"2",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := curateLogsDuration(tt.args.s); got != tt.want {
				t.Errorf("curateLogsDuration() = %v, want %v", got, tt.want)
			}
		})
	}
}
