package clob_test

import (
	"testing"

	"github.com/allank/chartea/clob"
)

func TestGroupedAsks(t *testing.T) {
	t.Run("sums multiple raw orders into the same bucket, rounding up", func(t *testing.T) {
		var book clob.OrderBook
		book.ApplySnapshot(
			[]clob.Order{
				{Price: 77214.0, Volume: 1.0},
				{Price: 77213.0, Volume: 2.0}, // same bucket as 77214 (both round up to 77220)
				{Price: 77206.0, Volume: 3.0}, // different bucket (rounds up to 77210)
			},
			nil,
		)

		got := book.GroupedAsks(10)
		want := []clob.Order{
			{Price: 77210.0, Volume: 3.0},
			{Price: 77220.0, Volume: 3.0},
		}
		assertOrders(t, "GroupedAsks(10)", got, want)
	})

	t.Run("an order exactly on a bucket boundary does not shift", func(t *testing.T) {
		var book clob.OrderBook
		book.ApplySnapshot([]clob.Order{{Price: 77220.0, Volume: 1.0}}, nil)

		got := book.GroupedAsks(10)
		want := []clob.Order{{Price: 77220.0, Volume: 1.0}}
		assertOrders(t, "GroupedAsks(10)", got, want)
	})

	t.Run("increment <= 0 returns the book unchanged", func(t *testing.T) {
		var book clob.OrderBook
		book.ApplySnapshot([]clob.Order{{Price: 100.0, Volume: 1.0}, {Price: 105.0, Volume: 2.0}}, nil)

		for _, increment := range []float64{0, -10} {
			got := book.GroupedAsks(increment)
			assertOrders(t, "GroupedAsks(<=0)", got, book.Asks())
		}
	})

	t.Run("fine-grained increment merges close prices into one bucket without spurious splitting", func(t *testing.T) {
		var book clob.OrderBook
		book.ApplySnapshot(
			[]clob.Order{
				{Price: 1.00051, Volume: 1.0},
				{Price: 1.00052, Volume: 2.0},
				{Price: 1.00053, Volume: 3.0}, // all three round up to 1.0006
				{Price: 1.00049, Volume: 4.0}, // rounds up to 1.0005, distinct bucket
			},
			nil,
		)

		got := book.GroupedAsks(0.0001)
		want := []clob.Order{
			{Price: 1.0005, Volume: 4.0},
			{Price: 1.0006, Volume: 6.0},
		}
		assertOrders(t, "GroupedAsks(0.0001)", got, want)
	})
}

func TestGroupedBids(t *testing.T) {
	t.Run("sums multiple raw orders into the same bucket, rounding down", func(t *testing.T) {
		var book clob.OrderBook
		book.ApplySnapshot(
			nil,
			[]clob.Order{
				{Price: 77214.0, Volume: 1.0},
				{Price: 77213.0, Volume: 2.0}, // same bucket as 77214 (both round down to 77210)
				{Price: 77206.0, Volume: 3.0}, // different bucket (rounds down to 77200)
			},
		)

		got := book.GroupedBids(10)
		want := []clob.Order{
			{Price: 77210.0, Volume: 3.0},
			{Price: 77200.0, Volume: 3.0},
		}
		assertOrders(t, "GroupedBids(10)", got, want)
	})

	t.Run("an order exactly on a bucket boundary does not shift", func(t *testing.T) {
		var book clob.OrderBook
		book.ApplySnapshot(nil, []clob.Order{{Price: 77210.0, Volume: 1.0}})

		got := book.GroupedBids(10)
		want := []clob.Order{{Price: 77210.0, Volume: 1.0}}
		assertOrders(t, "GroupedBids(10)", got, want)
	})

	t.Run("increment <= 0 returns the book unchanged", func(t *testing.T) {
		var book clob.OrderBook
		book.ApplySnapshot(nil, []clob.Order{{Price: 95.0, Volume: 1.0}, {Price: 90.0, Volume: 2.0}})

		for _, increment := range []float64{0, -10} {
			got := book.GroupedBids(increment)
			assertOrders(t, "GroupedBids(<=0)", got, book.Bids())
		}
	})

	t.Run("fine-grained increment merges close prices into one bucket without spurious splitting", func(t *testing.T) {
		var book clob.OrderBook
		book.ApplySnapshot(
			nil,
			[]clob.Order{
				{Price: 0.99951, Volume: 1.0},
				{Price: 0.99952, Volume: 2.0},
				{Price: 0.99953, Volume: 3.0}, // all three round down to 0.9995
				{Price: 0.99961, Volume: 4.0}, // rounds down to 0.9996, distinct bucket
			},
		)

		got := book.GroupedBids(0.0001)
		want := []clob.Order{
			{Price: 0.9996, Volume: 4.0},
			{Price: 0.9995, Volume: 6.0},
		}
		assertOrders(t, "GroupedBids(0.0001)", got, want)
	})
}

func TestGroupedAsksVeryFineIncrementDoesNotCollideAdjacentBuckets(t *testing.T) {
	// At an increment finer than the display-rounding a naive fixed-precision
	// implementation might use, two genuinely distinct buckets must not be
	// rounded onto the same displayed price -- that would silently corrupt
	// Price as a distinguishing key between them.
	var book clob.OrderBook
	book.ApplySnapshot(
		[]clob.Order{
			{Price: 1.000510000, Volume: 1.0}, // buckets to 1.000510000 at increment 1e-9
			{Price: 1.000510002, Volume: 2.0}, // buckets to 1.000510002, an adjacent but distinct bucket
		},
		nil,
	)

	got := book.GroupedAsks(1e-9)
	if len(got) != 2 {
		t.Fatalf("expected 2 distinct buckets, got %d: %+v", len(got), got)
	}
	if got[0].Price == got[1].Price {
		t.Errorf("adjacent buckets collided onto the same displayed price: %+v", got)
	}
}

func TestGroupedBidsAsksDoNotMutateOrderBook(t *testing.T) {
	var book clob.OrderBook
	book.ApplySnapshot(
		[]clob.Order{{Price: 100.0, Volume: 1.0}, {Price: 105.0, Volume: 1.0}},
		[]clob.Order{{Price: 95.0, Volume: 1.0}, {Price: 90.0, Volume: 1.0}},
	)

	// Copy the canonical order before grouping — Asks()/Bids() return the
	// live backing slice, so a bare assignment here would alias rather
	// than capture a snapshot to compare against.
	wantAsks := append([]clob.Order(nil), book.Asks()...)
	wantBids := append([]clob.Order(nil), book.Bids()...)

	book.GroupedAsks(10)
	book.GroupedBids(10)

	assertOrders(t, "Asks() after grouping", book.Asks(), wantAsks)
	assertOrders(t, "Bids() after grouping", book.Bids(), wantBids)
}
