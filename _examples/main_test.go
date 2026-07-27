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

func TestMergeOrderBook(t *testing.T) {
	t.Run("insert, update volume, remove level, and sort asks ascending", func(t *testing.T) {
		book := []clob.Order{
			{Price: 100.0, Volume: 1.0},
			{Price: 105.0, Volume: 2.0},
			{Price: 110.0, Volume: 3.0},
		}

		updates := []clob.Order{
			{Price: 105.0, Volume: 5.0}, // Update volume
			{Price: 100.0, Volume: 0.0}, // Remove level
			{Price: 98.0, Volume: 1.5},  // Insert new level (should be first when sorted ascending)
		}

		mergeOrderBook(&book, updates, false)

		if len(book) != 3 {
			t.Fatalf("expected 3 orders after merge, got %d", len(book))
		}

		expected := []clob.Order{
			{Price: 98.0, Volume: 1.5},
			{Price: 105.0, Volume: 5.0},
			{Price: 110.0, Volume: 3.0},
		}

		for i, o := range book {
			if o.Price != expected[i].Price || o.Volume != expected[i].Volume {
				t.Errorf("at index %d got {Price: %f, Volume: %f}, want {Price: %f, Volume: %f}",
					i, o.Price, o.Volume, expected[i].Price, expected[i].Volume)
			}
		}
	})

	t.Run("insert, update, and sort bids descending", func(t *testing.T) {
		book := []clob.Order{
			{Price: 95.0, Volume: 1.0},
			{Price: 90.0, Volume: 2.0},
		}

		updates := []clob.Order{
			{Price: 97.0, Volume: 0.5}, // Insert higher bid (should be first when sorted descending)
			{Price: 90.0, Volume: 0.0}, // Remove bid
		}

		mergeOrderBook(&book, updates, true)

		if len(book) != 2 {
			t.Fatalf("expected 2 orders after merge, got %d", len(book))
		}

		expected := []clob.Order{
			{Price: 97.0, Volume: 0.5},
			{Price: 95.0, Volume: 1.0},
		}

		for i, o := range book {
			if o.Price != expected[i].Price || o.Volume != expected[i].Volume {
				t.Errorf("at index %d got {Price: %f, Volume: %f}, want {Price: %f, Volume: %f}",
					i, o.Price, o.Volume, expected[i].Price, expected[i].Volume)
			}
		}
	})
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

	if len(updatedModel.wclob.Asks) != 1 || updatedModel.wclob.Asks[0].Price != 100.0 {
		t.Errorf("snapshot ask update failed: got %+v", updatedModel.wclob.Asks)
	}

	deltaMsg := wsOrderBookMsg{
		IsSnapshot: false,
		Asks: []clob.Order{
			{Price: 100.0, Volume: 2.5},
		},
	}

	newModel2, _ := updatedModel.Update(deltaMsg)
	updatedModel2 := newModel2.(mainModel)

	if len(updatedModel2.wclob.Asks) != 1 || updatedModel2.wclob.Asks[0].Volume != 2.5 {
		t.Errorf("delta ask update failed: got %+v", updatedModel2.wclob.Asks)
	}
}
