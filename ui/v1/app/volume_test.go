package app

import "testing"

func TestCalcVolumeDeltaScalesWithMaxVolume(t *testing.T) {
	tests := []struct {
		name       string
		maxVolume  int
		stepPercent int
		want       int
	}{
		{"20% of librespot max", 65535, 20, 13107},
		{"20% of 100", 100, 20, 20},
		{"negative step decrements", 65535, -20, -13107},
		{"5% matches old hardcoded value", 65535, 5, 3276},
		{"0% produces no change", 65535, 0, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := calcVolumeDelta(tt.maxVolume, tt.stepPercent)
			if got != tt.want {
				t.Fatalf("calcVolumeDelta(%d, %d) = %d, want %d", tt.maxVolume, tt.stepPercent, got, tt.want)
			}
		})
	}
}
