package clob_test

import (
	"strings"
	"testing"

	"github.com/allank/chartea/clob"
)

func newSideTestModel(orientation clob.Orientation, sides clob.Side) clob.Model {
	m := clob.New()
	m.Orientation = orientation
	m.Sides = sides
	m.ApplySnapshot(
		[]clob.Order{{Price: 100.0, Volume: 1.0}, {Price: 105.0, Volume: 1.0}},
		[]clob.Order{{Price: 90.0, Volume: 1.0}, {Price: 92.0, Volume: 1.0}},
	)
	return m
}

func TestViewWithOptionsHorizontalSingleSide(t *testing.T) {
	t.Run("BidsOnly shows only bid prices", func(t *testing.T) {
		m := newSideTestModel(clob.Horizontal, clob.BidsOnly)
		out := m.ViewWithOptions(clob.ViewOptions{Width: 40, Height: 20})

		assertContains(t, out, "90.00")
		assertContains(t, out, "92.00")
		assertNotContains(t, out, "100.00")
		assertNotContains(t, out, "105.00")
	})

	t.Run("AsksOnly shows only ask prices", func(t *testing.T) {
		m := newSideTestModel(clob.Horizontal, clob.AsksOnly)
		out := m.ViewWithOptions(clob.ViewOptions{Width: 40, Height: 20})

		assertContains(t, out, "100.00")
		assertContains(t, out, "105.00")
		assertNotContains(t, out, "90.00")
		assertNotContains(t, out, "92.00")
	})
}

func TestViewWithOptionsVerticalSingleSide(t *testing.T) {
	t.Run("BidsOnly shows only bid prices, no spread line", func(t *testing.T) {
		m := newSideTestModel(clob.Vertical, clob.BidsOnly)
		out := m.ViewWithOptions(clob.ViewOptions{Width: 40, Height: 20})

		assertContains(t, out, "90.00")
		assertContains(t, out, "92.00")
		assertNotContains(t, out, "100.00")
		assertNotContains(t, out, "105.00")
		assertNotContains(t, out, "Spread")
	})

	t.Run("AsksOnly shows only ask prices, no spread line, preserves worst-at-top/best-at-bottom ordering", func(t *testing.T) {
		m := newSideTestModel(clob.Vertical, clob.AsksOnly)
		out := m.ViewWithOptions(clob.ViewOptions{Width: 40, Height: 20})

		assertContains(t, out, "100.00")
		assertContains(t, out, "105.00")
		assertNotContains(t, out, "90.00")
		assertNotContains(t, out, "92.00")
		assertNotContains(t, out, "Spread")

		// Same worst-at-top/best-at-bottom convention as both-sides Vertical
		// — unaffected by single-side selection.
		assertBefore(t, out, "105.00", "100.00")
	})
}

func TestViewWithOptionsBothSidesUnchanged(t *testing.T) {
	// Both is the zero value of Side, so an unset Sides field (the default
	// for every existing Model) must behave exactly as before this feature.
	m := newSideTestModel(clob.Horizontal, clob.Both)
	out := m.ViewWithOptions(clob.ViewOptions{Width: 40, Height: 20})

	assertContains(t, out, "90.00")
	assertContains(t, out, "92.00")
	assertContains(t, out, "100.00")
	assertContains(t, out, "105.00")
}

func assertContains(t *testing.T, out, substr string) {
	t.Helper()
	if !strings.Contains(out, substr) {
		t.Errorf("expected %q to appear in output, got:\n%s", substr, out)
	}
}

func assertNotContains(t *testing.T, out, substr string) {
	t.Helper()
	if strings.Contains(out, substr) {
		t.Errorf("expected %q not to appear in output, got:\n%s", substr, out)
	}
}
