package main

import (
	"testing"

	"github.com/allank/chartea/clob"
)

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

func TestUpdateWSOrderBookMsg(t *testing.T) {
	m := InitialModel()

	snapshotMsg := wsOrderBookMsg{
		IsSnapshot: true,
		Asks: []clob.Order{
			{Price: 100.0, Volume: 1.0},
		},
		Bids: []clob.Order{
			{Price: 99.0, Volume: 1.0},
		},
	}

	newModel, _ := m.Update(snapshotMsg)
	updatedModel := newModel.(mainModel)

	if len(updatedModel.wclob.Asks()) != 1 || updatedModel.wclob.Asks()[0].Price != 100.0 {
		t.Errorf("snapshot ask update failed: got %+v", updatedModel.wclob.Asks())
	}

	deltaMsg := wsOrderBookMsg{
		IsSnapshot: false,
		Asks: []clob.Order{
			{Price: 100.0, Volume: 2.5},
		},
	}

	newModel2, _ := updatedModel.Update(deltaMsg)
	updatedModel2 := newModel2.(mainModel)

	if len(updatedModel2.wclob.Asks()) != 1 || updatedModel2.wclob.Asks()[0].Volume != 2.5 {
		t.Errorf("delta ask update failed: got %+v", updatedModel2.wclob.Asks())
	}
}

// TestUpdateWSOrderBookMsgDoesNotAffectStaticPanel guards against the
// Static Panel (rclob) contradicting CONTEXT.md's definition of Static
// Mode ("fetches the order book once via REST API on startup or manual
// refresh") by silently also tracking Realtime Mode's WebSocket feed.
func TestUpdateWSOrderBookMsgDoesNotAffectStaticPanel(t *testing.T) {
	m := InitialModel()

	// Copy before rendering/updating — Asks()/Bids() return the live
	// backing slice, so a bare assignment would alias rather than
	// snapshot the pre-update state.
	wantAsks := append([]clob.Order(nil), m.rclob.Asks()...)
	wantBids := append([]clob.Order(nil), m.rclob.Bids()...)

	snapshotMsg := wsOrderBookMsg{
		IsSnapshot: true,
		Asks: []clob.Order{
			{Price: 100.0, Volume: 1.0},
		},
		Bids: []clob.Order{
			{Price: 99.0, Volume: 1.0},
		},
	}

	newModel, _ := m.Update(snapshotMsg)
	updatedModel := newModel.(mainModel)

	assertOrdersUnchanged(t, "rclob.Asks()", updatedModel.rclob.Asks(), wantAsks)
	assertOrdersUnchanged(t, "rclob.Bids()", updatedModel.rclob.Bids(), wantBids)

	deltaMsg := wsOrderBookMsg{
		IsSnapshot: false,
		Asks: []clob.Order{
			{Price: 100.0, Volume: 2.5},
		},
	}

	newModel2, _ := updatedModel.Update(deltaMsg)
	updatedModel2 := newModel2.(mainModel)

	assertOrdersUnchanged(t, "rclob.Asks() after delta", updatedModel2.rclob.Asks(), wantAsks)
}

func assertOrdersUnchanged(t *testing.T, label string, got, want []clob.Order) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("%s: expected unchanged (%d orders), got %d orders: %+v", label, len(want), len(got), got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("%s: expected unchanged at index %d, got %+v, want %+v", label, i, got[i], want[i])
		}
	}
}
