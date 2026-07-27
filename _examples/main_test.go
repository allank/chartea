package main

import "testing"

func TestGetModeLabel(t *testing.T) {
	tests := []struct {
		name     string
		market   string
		realtime bool
		expected string
	}{
		{
			name:     "default mock mode when no market and not realtime",
			market:   "",
			realtime: false,
			expected: "[Mock Mode]",
		},
		{
			name:     "static mode when market provided without realtime",
			market:   "BTC/USD",
			realtime: false,
			expected: "[Static Mode]",
		},
		{
			name:     "realtime mode when market and realtime provided",
			market:   "BTC/USD",
			realtime: true,
			expected: "[Realtime Mode]",
		},
		{
			name:     "realtime mode when realtime provided without market",
			market:   "",
			realtime: true,
			expected: "[Realtime Mode]",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := getModeLabel(tt.market, tt.realtime)
			if got != tt.expected {
				t.Errorf("getModeLabel(%q, %v) = %q; want %q", tt.market, tt.realtime, got, tt.expected)
			}
		})
	}
}
