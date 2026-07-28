package clob_test

import (
	"testing"

	"github.com/allank/chartea/clob"
)

func TestApplyDelta(t *testing.T) {
	t.Run("insert, update volume, remove level, and sort asks ascending", func(t *testing.T) {
		var book clob.OrderBook
		book.ApplyDelta(
			[]clob.Order{
				{Price: 100.0, Volume: 1.0},
				{Price: 105.0, Volume: 2.0},
				{Price: 110.0, Volume: 3.0},
			},
			nil,
		)

		book.ApplyDelta(
			[]clob.Order{
				{Price: 105.0, Volume: 5.0}, // Update volume
				{Price: 100.0, Volume: 0.0}, // Remove level
				{Price: 98.0, Volume: 1.5},  // Insert new level (should be first when sorted ascending)
			},
			nil,
		)

		got := book.Asks()
		want := []clob.Order{
			{Price: 98.0, Volume: 1.5},
			{Price: 105.0, Volume: 5.0},
			{Price: 110.0, Volume: 3.0},
		}
		assertOrders(t, "Asks()", got, want)
	})

	t.Run("insert, update, and sort bids descending", func(t *testing.T) {
		var book clob.OrderBook
		book.ApplyDelta(nil, []clob.Order{
			{Price: 95.0, Volume: 1.0},
			{Price: 90.0, Volume: 2.0},
		})

		book.ApplyDelta(nil, []clob.Order{
			{Price: 97.0, Volume: 0.5}, // Insert higher bid (should be first when sorted descending)
			{Price: 90.0, Volume: 0.0}, // Remove bid
		})

		got := book.Bids()
		want := []clob.Order{
			{Price: 97.0, Volume: 0.5},
			{Price: 95.0, Volume: 1.0},
		}
		assertOrders(t, "Bids()", got, want)
	})

	t.Run("a single call updates both sides independently", func(t *testing.T) {
		var book clob.OrderBook
		book.ApplyDelta(
			[]clob.Order{{Price: 101.0, Volume: 1.0}},
			[]clob.Order{{Price: 99.0, Volume: 1.0}},
		)

		assertOrders(t, "Asks()", book.Asks(), []clob.Order{{Price: 101.0, Volume: 1.0}})
		assertOrders(t, "Bids()", book.Bids(), []clob.Order{{Price: 99.0, Volume: 1.0}})
	})
}

func TestApplySnapshot(t *testing.T) {
	t.Run("replaces prior contents entirely", func(t *testing.T) {
		var book clob.OrderBook
		book.ApplyDelta(
			[]clob.Order{{Price: 100.0, Volume: 1.0}, {Price: 105.0, Volume: 1.0}},
			[]clob.Order{{Price: 95.0, Volume: 1.0}, {Price: 90.0, Volume: 1.0}},
		)

		book.ApplySnapshot(
			[]clob.Order{{Price: 102.0, Volume: 2.0}, {Price: 108.0, Volume: 1.0}},
			[]clob.Order{{Price: 97.0, Volume: 2.0}},
		)

		assertOrders(t, "Asks()", book.Asks(), []clob.Order{
			{Price: 102.0, Volume: 2.0},
			{Price: 108.0, Volume: 1.0},
		})
		assertOrders(t, "Bids()", book.Bids(), []clob.Order{
			{Price: 97.0, Volume: 2.0},
		})
	})

	t.Run("drops zero-volume entries and sorts both sides", func(t *testing.T) {
		var book clob.OrderBook
		book.ApplySnapshot(
			[]clob.Order{
				{Price: 110.0, Volume: 1.0},
				{Price: 100.0, Volume: 0.0}, // Should be dropped
				{Price: 105.0, Volume: 2.0},
			},
			[]clob.Order{
				{Price: 90.0, Volume: 1.0},
				{Price: 95.0, Volume: 0.0}, // Should be dropped
				{Price: 92.0, Volume: 2.0},
			},
		)

		assertOrders(t, "Asks()", book.Asks(), []clob.Order{
			{Price: 105.0, Volume: 2.0},
			{Price: 110.0, Volume: 1.0},
		})
		assertOrders(t, "Bids()", book.Bids(), []clob.Order{
			{Price: 92.0, Volume: 2.0},
			{Price: 90.0, Volume: 1.0},
		})
	})
}

func assertOrders(t *testing.T, label string, got, want []clob.Order) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("%s: got %d orders %+v, want %d orders %+v", label, len(got), got, len(want), want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("%s: at index %d got %+v, want %+v", label, i, got[i], want[i])
		}
	}
}
