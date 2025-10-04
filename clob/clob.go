package clob

import (
	"fmt"
	"sort"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
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
	return m.ViewWithOptions(ViewOptions{Width: m.width, Height: m.height})
}

// ViewWithOptions renders the CLOB with the given options.
func (m *Model) ViewWithOptions(opts ViewOptions) string {
	if opts.Width == 0 {
		return "Initializing..."
	}

	// Sort the bids and asks before rendering.
	m.sortBids()
	m.sortAsks()

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

// sortBids sorts the bids in descending order by price.
func (m *Model) sortBids() {
	sort.Slice(m.Bids, func(i, j int) bool {
		return m.Bids[i].Price > m.Bids[j].Price
	})
}

// sortAsks sorts the asks in ascending order by price.
func (m *Model) sortAsks() {
	sort.Slice(m.Asks, func(i, j int) bool {
		return m.Asks[i].Price < m.Asks[j].Price
	})
}

// truncateOrders truncates the bids and asks to the given height.
func (m *Model) truncateOrders(height int) ([]Order, []Order) {
	bids := m.Bids
	asks := m.Asks
	if height > 0 {
		if len(bids) > height {
			bids = bids[:height]
		}
		if len(asks) > height {
			asks = asks[:height]
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
	rows := []string{}
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
	rows := []string{}
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