package clob

import (
	"strings"
	"testing"
)

// TestRenderSideWrapsOverflowingContent exercises renderSide with a
// price/volume pair too long to fit the given width, forcing the row's
// padding calculation to clamp to zero. lipgloss's Style.Width(n)
// constrains oversized content by wrapping it onto an extra line, whereas
// Render() alone leaves it as one unconstrained line — verified directly
// against this package's lipgloss dependency before writing this test.
//
// Before renderSide existed, the four separate row renderers applied
// Width() inconsistently: renderVerticalAsks's price-first (AlignRight)
// branch skipped it, while the other three carried it. Since renderSide is
// now the single implementation behind all four call sites, this asserts
// Width() is applied for both the volume-first and price-first shapes,
// closing that gap.
//
// This is a same-package (white-box) test against a private function
// rather than through the public ViewWithOptions API: the wrapping this
// test detects gets absorbed into lipgloss.Place()'s height-padding when
// rendered through the full panel, making the effect unobservable (and the
// assertion fragile) from the outside. Testing renderSide directly is the
// precise, low-fragility way to verify this specific property.
func TestRenderSideWrapsOverflowingContent(t *testing.T) {
	m := New()
	m.PricePrecision = 2
	m.VolumePrecision = 2
	// maxVolume is double the order's volume so onLen/offLen split the
	// width roughly in half (5/5) rather than going fully on or off —
	// a 0-width target doesn't reproduce the bug, since lipgloss treats
	// Width(0) as unconstrained rather than "wrap immediately."
	orders := []Order{{Price: 123456789.99, Volume: 1.0}}
	maxVolume := 2.0
	width := 10

	for _, volumeFirst := range []bool{true, false} {
		name := "price-first"
		if volumeFirst {
			name = "volume-first"
		}
		t.Run(name, func(t *testing.T) {
			out := m.renderSide(orders, m.StyleOnAsk, width, maxVolume, volumeFirst)
			if !strings.Contains(out, "\n") {
				t.Errorf("expected overflowing row to wrap onto an extra line (proving .Width() was applied), got a single unconstrained line:\n%q", out)
			}
		})
	}
}

// TestRenderSideUsesVolumeFormatter verifies that a non-nil VolumeFormatter
// overrides VolumePrecision-based formatting entirely.
func TestRenderSideUsesVolumeFormatter(t *testing.T) {
	m := New()
	m.VolumePrecision = 2
	m.VolumeFormatter = func(v float64) string { return "many" }
	orders := []Order{{Price: 100, Volume: 1234.5}}

	out := m.renderSide(orders, m.StyleOnAsk, 20, 1234.5, true)

	if !strings.Contains(out, "many") {
		t.Errorf("expected VolumeFormatter output %q in rendered row, got:\n%q", "many", out)
	}
	if strings.Contains(out, "1234.50") {
		t.Errorf("expected VolumeFormatter to override VolumePrecision formatting, but found the default-formatted volume in:\n%q", out)
	}
}
