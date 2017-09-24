package app

import "testing"

func Test_viewMode_isMainScreen(t *testing.T) {
	tests := []struct {
		name string
		v    viewMode
		want bool
	}{
		{
			"main screen",
			Main,
			true,
		},
		{
			"non main screen",
			HelpMode,
			false,
		},
	}
	for _, tt := range tests {
		if got := tt.v.isMainScreen(); got != tt.want {
			t.Errorf("%q. viewMode.isMainScreen() = %v, want %v", tt.name, got, tt.want)
		}
	}
}
