package daisy

import "testing"

func TestCheckDiskMode(t *testing.T) {
	tests := []struct {
		desc, input string
		want        bool
	}{
		{"default case", defaultDiskMode, true},
		{"ro case", diskModeRO, true},
		{"rw case", diskModeRW, true},
		{"bad mode case", "bad!", false},
	}

	for _, tt := range tests {
		got := checkDiskMode(tt.input)
		if got != tt.want {
			t.Errorf("%s: want: %t, got: %t", tt.desc, got, tt.want)
		}
	}
}
