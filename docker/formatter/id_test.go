package formatter

import "testing"

func TestTruncateID(t *testing.T) {

	tests := []struct {
		name string
		args string
		want string
	}{
		{
			"Truncate id",
			"129a7f8c0fae8c3251a8df9370577d9d6b96ca6f1bea33ada677bc153d7b1867",
			"129a7f8c0fae",
		},
		{
			"Truncate SHA256 id",
			"sha256:129a7f8c0fae8c3251a8df9370577d9d6b96ca6f1bea33ada677bc153d7b1867",
			"129a7f8c0fae",
		},
		{
			"Truncate empty id",
			"",
			"",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := TruncateID(tt.args); got != tt.want {
				t.Errorf("TruncateID() = %v, want %v", got, tt.want)
			}
		})
	}
}
