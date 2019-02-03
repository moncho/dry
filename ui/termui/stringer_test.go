package termui

import (
	"testing"
)

func TestStringer(t *testing.T) {
	type args struct {
		b bufferer
	}
	tests := []struct {
		name    string
		args    args
		want    string
		wantErr bool
	}{
		{
			"",
			args{
				NewParColumn("bla"),
			},
			"bla ",
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := String(tt.args.b)
			if (err != nil) != tt.wantErr {
				t.Errorf("String() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("String() = '%v', want '%v'", got, tt.want)
			}
		})
	}
}
