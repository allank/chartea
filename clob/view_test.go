package clob_test

import (
	"strings"
	"testing"

	"github.com/allank/chartea/clob"
)

func TestViewWithOptionsDoesNotMutateOrderBook(t *testing.T) {
	m := clob.New()
	m.ApplySnapshot(
		[]clob.Order{{Price: 100.0, Volume: 1.0}, {Price: 105.0, Volume: 1.0}, {Price: 110.0, Volume: 1.0}},
		[]clob.Order{{Price: 90.0, Volume: 1.0}, {Price: 92.0, Volume: 1.0}, {Price: 95.0, Volume: 1.0}},
	)

	// Copy the canonical order before rendering — Asks()/Bids() return the
	// live backing slice, so a bare assignment here would alias the model's
	// state rather than capture a snapshot to compare against.
	wantAsks := append([]clob.Order(nil), m.Asks()...)
	wantBids := append([]clob.Order(nil), m.Bids()...)

	// Render in both orientations, since each drove a different render-time
	// sort direction before this change.
	m.Orientation = clob.Horizontal
	m.ViewWithOptions(clob.ViewOptions{Width: 40, Height: 20})
	m.Orientation = clob.Vertical
	m.ViewWithOptions(clob.ViewOptions{Width: 40, Height: 20})

	assertOrders(t, "Asks() after rendering", m.Asks(), wantAsks)
	assertOrders(t, "Bids() after rendering", m.Bids(), wantBids)
}

func TestViewWithOptionsHorizontalOrdering(t *testing.T) {
	m := clob.New()
	m.Orientation = clob.Horizontal
	m.ApplySnapshot(
		[]clob.Order{{Price: 100.0, Volume: 1.0}, {Price: 105.0, Volume: 1.0}, {Price: 110.0, Volume: 1.0}},
		[]clob.Order{{Price: 90.0, Volume: 1.0}, {Price: 92.0, Volume: 1.0}, {Price: 95.0, Volume: 1.0}},
	)

	out := m.ViewWithOptions(clob.ViewOptions{Width: 40, Height: 20})

	// Best ask (lowest price) first, best bid (highest price) first.
	assertBefore(t, out, "100.00", "105.00")
	assertBefore(t, out, "105.00", "110.00")
	assertBefore(t, out, "95.00", "92.00")
	assertBefore(t, out, "92.00", "90.00")
}

func TestViewWithOptionsVerticalOrdering(t *testing.T) {
	m := clob.New()
	m.Orientation = clob.Vertical
	m.ApplySnapshot(
		[]clob.Order{{Price: 100.0, Volume: 1.0}, {Price: 105.0, Volume: 1.0}, {Price: 110.0, Volume: 1.0}},
		[]clob.Order{{Price: 90.0, Volume: 1.0}, {Price: 92.0, Volume: 1.0}, {Price: 95.0, Volume: 1.0}},
	)

	out := m.ViewWithOptions(clob.ViewOptions{Width: 40, Height: 20})

	// Asks render worst-to-best top-to-bottom (best ask at the bottom,
	// nearest the spread); bids render best-to-worst (best bid at the top).
	assertBefore(t, out, "110.00", "105.00")
	assertBefore(t, out, "105.00", "100.00")
	assertBefore(t, out, "95.00", "92.00")
	assertBefore(t, out, "92.00", "90.00")

	// Spread sits between the asks and bids blocks and reflects best
	// ask (100.00) minus best bid (95.00).
	assertBefore(t, out, "100.00", "Spread: 5.00")
	assertBefore(t, out, "Spread: 5.00", "95.00")
}

func TestViewWithOptionsVerticalAlignRightOrdering(t *testing.T) {
	m := clob.New()
	m.Orientation = clob.Vertical
	m.Alignment = clob.AlignRight
	m.ApplySnapshot(
		[]clob.Order{{Price: 100.0, Volume: 1.0}, {Price: 105.0, Volume: 1.0}, {Price: 110.0, Volume: 1.0}},
		[]clob.Order{{Price: 90.0, Volume: 1.0}, {Price: 92.0, Volume: 1.0}, {Price: 95.0, Volume: 1.0}},
	)

	out := m.ViewWithOptions(clob.ViewOptions{Width: 40, Height: 20})

	// Row ordering is governed by truncateOrders, not Alignment, so it
	// matches the AlignLeft case: worst-to-best asks top-to-bottom,
	// best-to-worst bids.
	assertBefore(t, out, "110.00", "105.00")
	assertBefore(t, out, "105.00", "100.00")
	assertBefore(t, out, "95.00", "92.00")
	assertBefore(t, out, "92.00", "90.00")
}

// assertBefore fails if first does not appear in out, or appears after last.
func assertBefore(t *testing.T, out, first, last string) {
	t.Helper()
	i := strings.Index(out, first)
	j := strings.Index(out, last)
	if i == -1 {
		t.Fatalf("expected %q to appear in output, got:\n%s", first, out)
	}
	if j == -1 {
		t.Fatalf("expected %q to appear in output, got:\n%s", last, out)
	}
	if i >= j {
		t.Errorf("expected %q (at %d) to appear before %q (at %d), got:\n%s", first, i, last, j, out)
	}
}
