package docker

import "testing"

func TestCommandFromDescription(t *testing.T) {
	type args struct {
		d string
	}
	tests := []struct {
		name    string
		args    args
		want    Command
		wantErr bool
	}{
		{
			"known description returns expected command",
			args{
				ContainerCommands[0].Description,
			},
			ContainerCommands[0].Command,
			false,
		},
		{
			"invalid description returns error",
			args{
				"yup, I dont know",
			},
			-1,
			true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := CommandFromDescription(tt.args.d)
			if (err != nil) != tt.wantErr {
				t.Errorf("CommandFromDescription() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("CommandFromDescription() = %v, want %v", got, tt.want)
			}
		})
	}
}
