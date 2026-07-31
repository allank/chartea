package clob

import (
	"fmt"
	"math"
	"sort"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"charm.land/lipgloss/v2/compat"
)

// Orientation defines the orientation of the order book.
type Orientation int

const (
	// Horizontal displays the order book with bids and asks side-by-side.
	// Best Bid and Best Ask are at the top of the book.
	Horizontal Orientation = iota
	// Vertical displays the order book with asks above bids.
	// Best Ask is at the bottom of the asks, best Bid is at the top of the bids
	// with the spread shown between best bid and best ask
	Vertical
)

// Alignment defines the alignment of the volume bar in vertical view.
type Alignment int

const (
	// AlignLeft aligns the volume bar to the left, price is on the right
	AlignLeft Alignment = iota
	// AlignRight aligns the volume bar to the right, price is on the left
	AlignRight
)

// Side determines which portion of the order book ViewWithOptions renders.
type Side int

const (
	// Both renders bids and asks together. This is the zero value, so an
	// unset Sides field preserves prior behavior.
	Both Side = iota
	// BidsOnly renders only the bid side, using the full available space.
	BidsOnly
	// AsksOnly renders only the ask side, using the full available space.
	AsksOnly
)

// ViewOptions allows you to specify the dimensions of the CLOB view.
type ViewOptions struct {
	Width  int
	Height int
}

// Model represents the state of the CLOB component.
type Model struct {
	width  int
	height int

	// OrderBook is the data for the order book.
	OrderBook

	// Orientation determines whether the order book is displayed vertically or horizontally.
	Orientation Orientation

	// Alignment determines, for a vertical layout, whether the volume bar is aligned to the left or right.
	Alignment Alignment

	// Sides determines whether ViewWithOptions renders both sides, bids
	// only, or asks only. The zero value is Both.
	Sides Side

	// Spacing is the space between the bid and ask columns.
	Spacing int

	// Precision for price and volume.
	PricePrecision  int
	VolumePrecision int

	// VolumeFormatter, if set, formats volume values for display instead of
	// the default "%.<VolumePrecision>f" formatting. This lets callers plug
	// in their own formatting (e.g. github.com/dustin/go-humanize) without
	// chartea depending on it directly.
	VolumeFormatter func(float64) string

	// Styles
	StyleOffBar lipgloss.Style
	StyleOnBid  lipgloss.Style
	StyleOnAsk  lipgloss.Style
}

// OrderBook represents the full order book.
type OrderBook struct {
	bids []Order
	asks []Order
}

// Order represents a single order in the book.
type Order struct {
	Volume float64
	Price  float64
}

// Bids returns the current bid levels, sorted descending by price.
func (b *OrderBook) Bids() []Order {
	return b.bids
}

// Asks returns the current ask levels, sorted ascending by price.
func (b *OrderBook) Asks() []Order {
	return b.asks
}

// ApplySnapshot replaces the book's contents with the given levels. Zero-volume
// entries in the input are dropped, and both sides are sorted (asks ascending,
// bids descending) before replacing the prior contents.
func (b *OrderBook) ApplySnapshot(asks, bids []Order) {
	b.asks = normalizeLevels(asks, false)
	b.bids = normalizeLevels(bids, true)
}

// ApplyDelta merges the given levels into the book by price: a level matching
// an existing price updates its volume, a zero-volume update removes that
// price level, and an update for an unmatched price with non-zero volume is
// inserted. Both sides are re-sorted after merging (asks ascending, bids
// descending).
func (b *OrderBook) ApplyDelta(asks, bids []Order) {
	b.asks = mergeLevels(b.asks, asks, false)
	b.bids = mergeLevels(b.bids, bids, true)
}

// GroupedBids returns the current bids aggregated into price Buckets of the
// given increment, rounded down, with volume summed within each Bucket.
// increment <= 0 returns the current bids unchanged. This is a pure read:
// it does not mutate the book, so Bids() always retains full granularity.
func (b *OrderBook) GroupedBids(increment float64) []Order {
	return groupLevels(b.bids, increment, false)
}

// GroupedAsks returns the current asks aggregated into price Buckets of the
// given increment, rounded up, with volume summed within each Bucket.
// increment <= 0 returns the current asks unchanged. This is a pure read:
// it does not mutate the book, so Asks() always retains full granularity.
func (b *OrderBook) GroupedAsks(increment float64) []Order {
	return groupLevels(b.asks, increment, true)
}

// groupLevels aggregates already-sorted levels into price buckets of the
// given increment (rounded up when roundUp is true, down otherwise),
// summing volume within each bucket. Bucket identity is computed as an
// integer index rather than by comparing bucketed float prices directly,
// avoiding floating-point round-trip drift that could otherwise split one
// bucket into two. Because levels are already sorted and floor/ceil are
// monotonic, equal buckets are always adjacent, so a single forward pass
// with no map is sufficient.
func groupLevels(levels []Order, increment float64, roundUp bool) []Order {
	if increment <= 0 || len(levels) == 0 {
		return levels
	}
	grouped := make([]Order, 0, len(levels))
	var lastBucket int64
	for i, o := range levels {
		bucket := bucketIndex(o.Price, increment, roundUp)
		if i > 0 && bucket == lastBucket {
			grouped[len(grouped)-1].Volume += o.Volume
		} else {
			grouped = append(grouped, Order{Price: bucketPrice(bucket, increment), Volume: o.Volume})
			lastBucket = bucket
		}
	}
	return grouped
}

// bucketIndex returns the integer bucket a price falls into for the given
// increment (rounded up when roundUp is true, down otherwise).
func bucketIndex(price, increment float64, roundUp bool) int64 {
	if roundUp {
		return int64(math.Ceil(price / increment))
	}
	return int64(math.Floor(price / increment))
}

// bucketPrice converts a bucket index back into a price. float64(bucket) *
// increment can carry trailing float64 noise for increments that aren't
// exactly representable in binary (e.g. 0.0001). Rounding to a fixed
// number of decimal places would remove that noise but risks colliding
// two genuinely distinct buckets onto the same displayed price for
// increments finer than that fixed precision — so the rounding unit
// instead scales with increment itself, staying far finer than the gap
// between adjacent buckets (always exactly one increment apart) while
// still coarse enough to absorb the multiplication's float64 noise.
func bucketPrice(bucket int64, increment float64) float64 {
	unit := increment / 1e6
	return math.Round(float64(bucket)*increment/unit) * unit
}

// mergeLevels merges updates into an existing set of price levels and sorts
// the result (descending by price when desc is true, ascending otherwise).
func mergeLevels(levels, updates []Order, desc bool) []Order {
	for _, update := range updates {
		found := false
		for i, existing := range levels {
			if existing.Price == update.Price {
				if update.Volume == 0 {
					levels = append(levels[:i], levels[i+1:]...)
				} else {
					levels[i].Volume = update.Volume
				}
				found = true
				break
			}
		}
		if !found && update.Volume != 0 {
			levels = append(levels, update)
		}
	}
	sortLevels(levels, desc)
	return levels
}

// normalizeLevels drops zero-volume entries and sorts the remaining levels
// (descending by price when desc is true, ascending otherwise).
func normalizeLevels(levels []Order, desc bool) []Order {
	normalized := make([]Order, 0, len(levels))
	for _, o := range levels {
		if o.Volume != 0 {
			normalized = append(normalized, o)
		}
	}
	sortLevels(normalized, desc)
	return normalized
}

// sortLevels sorts levels by price (descending when desc is true, ascending
// otherwise).
func sortLevels(levels []Order, desc bool) {
	sort.Slice(levels, func(i, j int) bool {
		if desc {
			return levels[i].Price > levels[j].Price
		}
		return levels[i].Price < levels[j].Price
	})
}

// New creates a new CLOB model with default styles.
func New() Model {
	return Model{
		Spacing:         1,
		PricePrecision:  2,
		VolumePrecision: 2,
		StyleOffBar: lipgloss.NewStyle().
			Foreground(compat.AdaptiveColor{Light: lipgloss.Color("232"), Dark: lipgloss.Color("188")}),
		StyleOnBid: lipgloss.NewStyle().
			Foreground(lipgloss.Color("188")).
			Background(lipgloss.Color("34")),
		StyleOnAsk: lipgloss.NewStyle().
			Foreground(lipgloss.Color("188")).
			Background(lipgloss.Color("124")),
	}
}

// Init initializes the CLOB model.
func (m Model) Init() tea.Cmd {
	return nil
}

// Update handles messages for the CLOB model.
func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
	}
	return m, nil
}

// View renders the CLOB, taking up the full width and height of the model.
func (m *Model) View() string {
	if m.width <= 0 {
		return "Initializing..."
	}
	return m.ViewWithOptions(ViewOptions{Width: m.width, Height: m.height})
}

// ViewWithOptions renders the CLOB with the given options.
func (m *Model) ViewWithOptions(opts ViewOptions) string {
	if opts.Width <= 0 {
		return "Initializing..."
	}

	switch m.Orientation {
	case Vertical:
		// Vertical orientation renders both sides volume-first for
		// AlignLeft, price-first for AlignRight — for either side alone or
		// both together.
		volumeFirst := m.Alignment == AlignLeft

		switch m.Sides {
		case BidsOnly:
			// A single side gets the full height: there's no spread row to
			// reserve a line for when the other side isn't shown.
			bids, _ := m.truncateOrders(opts.Height)
			maxVolume := m.calculateMaxVolume(bids, nil)
			bidView := m.renderSide(bids, m.StyleOnBid, opts.Width, maxVolume, volumeFirst)
			return place(opts, bidView)
		case AsksOnly:
			_, asks := m.truncateOrders(opts.Height)
			maxVolume := m.calculateMaxVolume(nil, asks)
			askView := m.renderSide(asks, m.StyleOnAsk, opts.Width, maxVolume, volumeFirst)
			return place(opts, askView)
		}

		// Both: truncate the bids and asks, accounting for the spread row.
		bids, asks := m.truncateOrders((opts.Height - 1) / 2)

		// Find the maximum volume in the order book to scale the bars correctly.
		maxVolume := m.calculateMaxVolume(bids, asks)

		askView := m.renderSide(asks, m.StyleOnAsk, opts.Width, maxVolume, volumeFirst)
		spreadView := m.renderSpread(opts.Width)
		bidView := m.renderSide(bids, m.StyleOnBid, opts.Width, maxVolume, volumeFirst)

		bookPanel := lipgloss.JoinVertical(lipgloss.Left, askView, spreadView, bidView)
		return place(opts, bookPanel)
	case Horizontal:
		// Truncate the bids and asks if a height is specified. Horizontal
		// orientation always renders bids price-first, asks volume-first —
		// for either side alone or both together.
		bids, asks := m.truncateOrders(opts.Height)
		const bidsVolumeFirst, asksVolumeFirst = false, true

		switch m.Sides {
		case BidsOnly:
			// A single side gets the full width: no spacer, no second column.
			maxVolume := m.calculateMaxVolume(bids, nil)
			bidView := m.renderSide(bids, m.StyleOnBid, opts.Width, maxVolume, bidsVolumeFirst)
			return place(opts, bidView)
		case AsksOnly:
			maxVolume := m.calculateMaxVolume(nil, asks)
			askView := m.renderSide(asks, m.StyleOnAsk, opts.Width, maxVolume, asksVolumeFirst)
			return place(opts, askView)
		}

		// Both: calculate the width of each column.
		columnWidth := (opts.Width - m.Spacing) / 2

		// Find the maximum volume in the order book to scale the bars correctly.
		maxVolume := m.calculateMaxVolume(bids, asks)
		bidView := m.renderSide(bids, m.StyleOnBid, columnWidth, maxVolume, bidsVolumeFirst)
		askView := m.renderSide(asks, m.StyleOnAsk, columnWidth, maxVolume, asksVolumeFirst)

		// Create a spacer between the two columns.
		spacer := lipgloss.NewStyle().Width(m.Spacing).Render("")

		// Join the bid, spacer, and ask views horizontally.
		bookPanel := lipgloss.JoinHorizontal(lipgloss.Top, bidView, spacer, askView)
		return place(opts, bookPanel)
	}
	return ""
}

// place centers a rendered book panel within the given dimensions.
func place(opts ViewOptions, panel string) string {
	return lipgloss.Place(
		opts.Width,
		opts.Height,
		lipgloss.Center,
		lipgloss.Center,
		panel,
	)
}

// renderSpread renders the spread between the best bid and ask.
func (m *Model) renderSpread(width int) string {
	if len(m.asks) == 0 || len(m.bids) == 0 {
		return ""
	}
	bestAsk := m.asks[0].Price
	bestBid := m.bids[0].Price
	spread := bestAsk - bestBid
	priceFormat := fmt.Sprintf("Spread: %%.%df", m.PricePrecision)
	spreadString := fmt.Sprintf(priceFormat, spread)
	align := lipgloss.Left
	if m.Alignment == AlignLeft {
		align = lipgloss.Right
	}
	return lipgloss.NewStyle().Width(width).Align(align).Render(m.StyleOffBar.Render(spreadString))
}

// renderSide renders one side of the order book (a column of price/volume
// rows) using onStyle for the filled portion of each row's volume bar.
//
// Every row reduces to one of two shapes, chosen by volumeFirst:
//   - volume-first: "volume  price", bar filled from the left (on-segment
//     first), joined left.
//   - price-first: "price  volume", bar filled from the right (off-segment
//     first), joined right.
//
// Horizontal orientation always renders bids price-first and asks
// volume-first; Vertical orientation renders both sides volume-first for
// AlignLeft and price-first for AlignRight. This is the single
// implementation behind what were previously four near-identical
// functions — renderVerticalAsks's AlignRight-equivalent path used to skip
// the .Width() calls the other three carried, leaving its bar segments
// unconstrained; that inconsistency can't recur now that there's only one
// code path.
func (m *Model) renderSide(orders []Order, onStyle lipgloss.Style, width int, maxVolume float64, volumeFirst bool) string {
	rows := make([]string, 0, len(orders))
	priceFormat := fmt.Sprintf("%%.%df", m.PricePrecision)
	volumeFormat := fmt.Sprintf("%%.%df", m.VolumePrecision)

	for _, o := range orders {
		priceString := fmt.Sprintf(priceFormat, o.Price)
		volumeString := fmt.Sprintf(volumeFormat, o.Volume)
		if m.VolumeFormatter != nil {
			volumeString = m.VolumeFormatter(o.Volume)
		}

		padding := width - len(priceString) - len(volumeString)
		if padding < 0 {
			padding = 0
		}

		var output string
		if volumeFirst {
			output = fmt.Sprintf("%s%s%s", volumeString, strings.Repeat(" ", padding), priceString)
		} else {
			output = fmt.Sprintf("%s%s%s", priceString, strings.Repeat(" ", padding), volumeString)
		}

		onLen := int(float64(width) * (o.Volume / maxVolume))
		offLen := width - onLen

		var bar string
		if volumeFirst {
			onStr := onStyle.Width(onLen).Render(output[:onLen])
			offStr := m.StyleOffBar.Width(offLen).Render(output[onLen:])
			bar = lipgloss.JoinHorizontal(lipgloss.Left, onStr, offStr)
		} else {
			offStr := m.StyleOffBar.Width(offLen).Render(output[:offLen])
			onStr := onStyle.Width(onLen).Render(output[offLen:])
			bar = lipgloss.JoinHorizontal(lipgloss.Right, offStr, onStr)
		}
		rows = append(rows, bar)
	}
	return lipgloss.JoinVertical(lipgloss.Left, rows...)
}

// truncateOrders truncates the bids and asks to the given height and
// returns them in display order for the current Orientation. Bids are
// always best-first (descending); asks are best-first (ascending) for
// Horizontal, matching the book's canonical order, or reversed for
// Vertical, which displays the worst ask at the top and the best ask
// at the bottom, nearest the spread.
func (m *Model) truncateOrders(height int) ([]Order, []Order) {
	bids := m.bids
	asks := m.asks
	if height > 0 {
		if len(bids) > height {
			bids = bids[:height]
		}
		if len(asks) > height {
			asks = asks[:height]
		}
	}
	if m.Orientation == Vertical {
		// Bids are stored best-first (descending), which already matches
		// Vertical's "best bid at the top" convention. Asks are stored
		// best-first (ascending), but Vertical wants the worst ask at the
		// top and the best ask at the bottom, nearest the spread — so the
		// kept subset is reversed for display. A copy is required here:
		// asks still shares a backing array with the book's canonical
		// slice, and reversing in place would corrupt that order for the
		// next render.
		asks = reverseOrders(asks)
	}
	return bids, asks
}

// reverseOrders returns a copy of orders in reverse order, leaving the
// input slice (and any array it shares) untouched.
func reverseOrders(orders []Order) []Order {
	reversed := make([]Order, len(orders))
	for i, o := range orders {
		reversed[len(orders)-1-i] = o
	}
	return reversed
}

// calculateMaxVolume finds the maximum volume in the given orders.
func (m *Model) calculateMaxVolume(bids, asks []Order) float64 {
	maxVolume := 0.0
	for _, o := range asks {
		if o.Volume > maxVolume {
			maxVolume = o.Volume
		}
	}
	for _, o := range bids {
		if o.Volume > maxVolume {
			maxVolume = o.Volume
		}
	}
	return maxVolume
}
