package clob

import (
	"fmt"
	"sort"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
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

	// Spacing is the space between the bid and ask columns.
	Spacing int

	// Precision for price and volume.
	PricePrecision  int
	VolumePrecision int

	// Styles
	StyleOffBar lipgloss.Style
	StyleOnBid  lipgloss.Style
	StyleOnAsk  lipgloss.Style
}

// OrderBook represents the full order book.
type OrderBook struct {
	Bids []Order
	Asks []Order
}

// Order represents a single order in the book.
type Order struct {
	Volume float64
	Price  float64
}

// New creates a new CLOB model with default styles.
func New() Model {
	return Model{
		Spacing:         1,
		PricePrecision:  2,
		VolumePrecision: 2,
		StyleOffBar: lipgloss.NewStyle().
			Foreground(lipgloss.AdaptiveColor{Light: "232", Dark: "188"}),
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
		// Sort the bids and asks before rendering.
		m.sortBids(true)
		m.sortAsks(true)

		// Truncate the bids and asks if a height is specified.
		// Account for the spread when using Vertical orientation
		bids, asks := m.truncateOrders((opts.Height - 1) / 2)

		// Find the maximum volume in the order book to scale the bars correctly.
		maxVolume := m.calculateMaxVolume(bids, asks)

		// Render the bid and ask sides of the book.
		askView := m.renderVerticalAsks(asks, opts.Width, maxVolume)
		spreadView := m.renderSpread(opts.Width)
		bidView := m.renderVerticalBids(bids, opts.Width, maxVolume)

		bookPanel := lipgloss.JoinVertical(lipgloss.Left, askView, spreadView, bidView)
		// bookPanel := lipgloss.JoinVertical(lipgloss.Left, askView)

		// Place the book panel in the center of the available space.
		return lipgloss.Place(
			opts.Width,
			opts.Height,
			lipgloss.Center,
			lipgloss.Center,
			bookPanel,
		)
	case Horizontal:
		// Sort the bids and asks before rendering.
		m.sortBids(true)
		m.sortAsks(false)

		// Truncate the bids and asks if a height is specified.
		bids, asks := m.truncateOrders(opts.Height)

		// Calculate the width of each column.
		columnWidth := (opts.Width - m.Spacing) / 2

		// Find the maximum volume in the order book to scale the bars correctly.
		maxVolume := m.calculateMaxVolume(bids, asks)
		// Render the bid and ask sides of the book.
		bidView := m.renderBids(bids, columnWidth, maxVolume)
		askView := m.renderAsks(asks, columnWidth, maxVolume)

		// Create a spacer between the two columns.
		spacer := lipgloss.NewStyle().Width(m.Spacing).Render("")

		// Join the bid, spacer, and ask views horizontally.
		bookPanel := lipgloss.JoinHorizontal(lipgloss.Top, bidView, spacer, askView)

		// Place the book panel in the center of the available space.
		return lipgloss.Place(
			opts.Width,
			opts.Height,
			lipgloss.Center,
			lipgloss.Center,
			bookPanel,
		)
	}
	return ""
}

// renderSpread renders the spread between the best bid and ask.
func (m *Model) renderSpread(width int) string {
	if len(m.Asks) == 0 || len(m.Bids) == 0 {
		return ""
	}
	bestAsk := m.Asks[len(m.Asks)-1].Price
	bestBid := m.Bids[0].Price
	spread := bestAsk - bestBid
	priceFormat := fmt.Sprintf("Spread: %%.%df", m.PricePrecision)
	spreadString := fmt.Sprintf(priceFormat, spread)
	align := lipgloss.Left
	if m.Alignment == AlignLeft {
		align = lipgloss.Right
	}
	return lipgloss.NewStyle().Width(width).Align(align).Render(m.StyleOffBar.Render(spreadString))
}

// renderVerticalBids renders the bid side of the order book for vertical orientation.
func (m *Model) renderVerticalBids(orders []Order, width int, maxVolume float64) string {
	rows := make([]string, 0, len(orders))
	priceFormat := fmt.Sprintf("%%.%df", m.PricePrecision)
	volumeFormat := fmt.Sprintf("%%.%df", m.VolumePrecision)

	for _, o := range orders {
		priceString := fmt.Sprintf(priceFormat, o.Price)
		volumeString := fmt.Sprintf(volumeFormat, o.Volume)

		padding := width - len(priceString) - len(volumeString)
		if padding < 0 {
			padding = 0
		}

		var output string
		if m.Alignment == AlignLeft {
			output = fmt.Sprintf("%s%s%s", volumeString, strings.Repeat(" ", padding), priceString)
		} else {
			output = fmt.Sprintf("%s%s%s", priceString, strings.Repeat(" ", padding), volumeString)
		}

		onLen := int(float64(width) * (o.Volume / maxVolume))
		offLen := width - onLen

		var bar string
		if m.Alignment == AlignLeft {
			onStr := m.StyleOnBid.Width(onLen).Render(output[:onLen])
			offStr := m.StyleOffBar.Width(offLen).Render(output[onLen:])
			bar = lipgloss.JoinHorizontal(lipgloss.Left, onStr, offStr)
		} else {
			offStr := m.StyleOffBar.Width(offLen).Render(output[:offLen])
			onStr := m.StyleOnBid.Width(onLen).Render(output[offLen:])
			bar = lipgloss.JoinHorizontal(lipgloss.Right, offStr, onStr)
		}
		rows = append(rows, bar)
	}
	return lipgloss.JoinVertical(lipgloss.Left, rows...)
}

// renderVerticalAsks renders the ask side of the order book for vertical orientation.
func (m *Model) renderVerticalAsks(orders []Order, width int, maxVolume float64) string {
	rows := make([]string, 0, len(orders))
	priceFormat := fmt.Sprintf("%%.%df", m.PricePrecision)
	volumeFormat := fmt.Sprintf("%%.%df", m.VolumePrecision)

	for _, o := range orders {
		priceString := fmt.Sprintf(priceFormat, o.Price)
		volumeString := fmt.Sprintf(volumeFormat, o.Volume)

		padding := width - len(priceString) - len(volumeString)
		if padding < 0 {
			padding = 0
		}

		var output string
		if m.Alignment == AlignLeft {
			output = fmt.Sprintf("%s%s%s", volumeString, strings.Repeat(" ", padding), priceString)
		} else {
			output = fmt.Sprintf("%s%s%s", priceString, strings.Repeat(" ", padding), volumeString)
		}

		onLen := int(float64(width) * (o.Volume / maxVolume))
		offLen := width - onLen

		var bar string
		if m.Alignment == AlignLeft {
			onStr := m.StyleOnAsk.Width(onLen).Render(output[:onLen])
			offStr := m.StyleOffBar.Width(offLen).Render(output[onLen:])
			bar = lipgloss.JoinHorizontal(lipgloss.Left, onStr, offStr)
		} else {
			offStr := m.StyleOffBar.Render(output[:offLen])
			onStr := m.StyleOnAsk.Render(output[offLen:])
			bar = lipgloss.JoinHorizontal(lipgloss.Right, offStr, onStr)
		}
		rows = append(rows, bar)
	}
	return lipgloss.JoinVertical(lipgloss.Left, rows...)
}

// sortBids sorts the bids in descending order by price.
func (m *Model) sortBids(desc bool) {
	sort.Slice(m.Bids, func(i, j int) bool {
		if desc {
			return m.Bids[i].Price > m.Bids[j].Price
		}
		return m.Bids[i].Price < m.Bids[j].Price
	})
}

// sortAsks sorts the asks in ascending order by price.
func (m *Model) sortAsks(desc bool) {
	sort.Slice(m.Asks, func(i, j int) bool {
		if desc {
			return m.Asks[i].Price > m.Asks[j].Price
		}
		return m.Asks[i].Price < m.Asks[j].Price
	})
}

// truncateOrders truncates the bids and asks to the given height.
func (m *Model) truncateOrders(height int) ([]Order, []Order) {
	bids := m.Bids
	asks := m.Asks
	if height > 0 {
		switch m.Orientation {
		case Vertical:
			if len(bids) > height {
				bids = bids[:height]
			}
			if len(asks) > height {
				asks = asks[len(asks)-height:]
			}
		case Horizontal:
			if len(bids) > height {
				bids = bids[:height]
			}
			if len(asks) > height {
				asks = asks[:height]
			}
		}
	}
	return bids, asks
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

// renderBids renders the bid side of the order book.
func (m *Model) renderBids(orders []Order, width int, maxVolume float64) string {
	rows := make([]string, 0, len(orders))
	priceFormat := fmt.Sprintf("%%.%df", m.PricePrecision)
	volumeFormat := fmt.Sprintf("%%.%df", m.VolumePrecision)

	for _, o := range orders {
		priceString := fmt.Sprintf(priceFormat, o.Price)
		volumeString := fmt.Sprintf(volumeFormat, o.Volume)

		padding := width - len(priceString) - len(volumeString)
		if padding < 0 {
			padding = 0
		}
		output := fmt.Sprintf("%s%s%s", priceString, strings.Repeat(" ", padding), volumeString)

		onLen := int(float64(width) * (o.Volume / maxVolume))
		offLen := width - onLen

		offStr := m.StyleOffBar.Width(offLen).Render(output[:offLen])
		onStr := m.StyleOnBid.Width(onLen).Render(output[offLen:])

		bar := lipgloss.JoinHorizontal(lipgloss.Right, offStr, onStr)
		rows = append(rows, bar)
	}
	return lipgloss.JoinVertical(lipgloss.Left, rows...)
}

// renderAsks renders the ask side of the order book.
func (m *Model) renderAsks(orders []Order, width int, maxVolume float64) string {
	rows := make([]string, 0, len(orders))
	priceFormat := fmt.Sprintf("%%.%df", m.PricePrecision)
	volumeFormat := fmt.Sprintf("%%.%df", m.VolumePrecision)

	for _, o := range orders {
		priceString := fmt.Sprintf(priceFormat, o.Price)
		volumeString := fmt.Sprintf(volumeFormat, o.Volume)

		padding := width - len(priceString) - len(volumeString)
		if padding < 0 {
			padding = 0
		}
		output := fmt.Sprintf("%s%s%s", volumeString, strings.Repeat(" ", padding), priceString)

		onLen := int(float64(width) * (o.Volume / maxVolume))
		offLen := width - onLen

		onStr := m.StyleOnAsk.Width(onLen).Render(output[:onLen])
		offStr := m.StyleOffBar.Width(offLen).Render(output[onLen:])

		bar := lipgloss.JoinHorizontal(lipgloss.Left, onStr, offStr)
		rows = append(rows, bar)
	}
	return lipgloss.JoinVertical(lipgloss.Left, rows...)
}
