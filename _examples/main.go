package main

import (
	"flag"
	"log"
	"math"
	"os"
	"strconv"

	"github.com/allank/chartea/clob"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

var market string

var (
	orderBookCache   *OrderBook
	isTokenizedCache bool
)

func init() {
}

type refetchMsg struct{}

type mainModel struct {
	rclob   clob.Model
	wclob   clob.Model
	width   int
	height  int
	loading bool
}

func parseOrderBook(orderBook *OrderBook) ([]clob.Order, []clob.Order) {
	asks := make([]clob.Order, len(orderBook.Asks))
	for i, ask := range orderBook.Asks {
		price, _ := strconv.ParseFloat(ask[0].(string), 64)
		volume, _ := strconv.ParseFloat(ask[1].(string), 64)
		asks[i] = clob.Order{Price: price, Volume: volume}
	}

	bids := make([]clob.Order, len(orderBook.Bids))
	for i, bid := range orderBook.Bids {
		price, _ := strconv.ParseFloat(bid[0].(string), 64)
		volume, _ := strconv.ParseFloat(bid[1].(string), 64)
		bids[i] = clob.Order{Price: price, Volume: volume}
	}

	return asks, bids
}

// InitialModel creates the initial state of the application model.
func InitialModel() mainModel {
	m := mainModel{
		rclob: clob.New(),
		wclob: clob.New(),
	}
	if market != "" {
		orderBook, _, err := fetchOrderBook(market, false)
		if err != nil {
			log.Fatalf("could not fetch order book: %v", err)
		}
		asks, bids := parseOrderBook(orderBook)
		m.rclob.Asks = asks
		m.rclob.Bids = bids
	} else {
		m.rclob.Asks = mockAsks()
		m.rclob.Bids = mockBids()
	}
	// Set VolumePrecision
	m.rclob.VolumePrecision = 8
	// Override default styles
	m.rclob.StyleOnBid = lipgloss.NewStyle().
		Foreground(lipgloss.Color("228")).
		Background(lipgloss.Color("28"))
	m.rclob.StyleOnAsk = lipgloss.NewStyle().
		Foreground(lipgloss.Color("228")).
		Background(lipgloss.Color("197"))
	m.wclob.Asks = mockAsks()
	m.wclob.Bids = mockBids()
	// Set VolumePrecision
	m.wclob.VolumePrecision = 8
	m.wclob.Orientation = clob.Vertical
	m.wclob.StyleOnBid = lipgloss.NewStyle().
		Foreground(lipgloss.Color("228")).
		Background(lipgloss.Color("28"))
	m.wclob.StyleOnAsk = lipgloss.NewStyle().
		Foreground(lipgloss.Color("228")).
		Background(lipgloss.Color("197"))

	return m
}

// Init is the first command that is run when the program starts.
func (m mainModel) Init() tea.Cmd {
	return nil
}

// Update handles all incoming messages and updates the model accordingly.
func (m mainModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			return m, tea.Quit
		case "r":
			m.loading = true
			return m, func() tea.Msg {
				return refetchMsg{}
			}
		case "v":
			if m.rclob.Orientation == clob.Vertical {
				m.rclob.Orientation = clob.Horizontal
			} else {
				m.rclob.Orientation = clob.Vertical
			}
		case "a":
			if m.wclob.Alignment == clob.AlignLeft {
				m.wclob.Alignment = clob.AlignRight
			} else {
				m.wclob.Alignment = clob.AlignLeft
			}
		}
	case refetchMsg:
		m.loading = false
		if market != "" {
			orderBook, _, err := fetchOrderBook(market, true)
			if err != nil {
				// Handle error appropriately, maybe set an error message in the model
			} else {
				asks, bids := parseOrderBook(orderBook)
				m.rclob.Asks = asks
				m.rclob.Bids = bids
			}
		}
		return m, nil
	}

	var cmd tea.Cmd
	m.rclob, cmd = m.rclob.Update(msg)
	return m, cmd
}

// View renders the UI based on the current model state.
func (m mainModel) View() string {
	// Panel
	panelStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("229")).
		Padding(1, 2)

	restPanelWidth := int(math.Floor(float64(m.width / 2)))
	wsPanelWidth := int(math.Floor(float64(m.width / 2)))
	panelHeight := m.height - 1

	// The available size for the rendering of the order book needs to take into account
	// the frame border and padding for the panel it is being shown inside of
	availRWidth := restPanelWidth - (panelStyle.GetHorizontalFrameSize() * 2)
	availWWidth := wsPanelWidth - (panelStyle.GetHorizontalFrameSize() * 2)
	availHeight := panelHeight - panelStyle.GetVerticalFrameSize()

	// REST Panel
	var restPanelContent string
	if m.loading {
		restPanelContent = "Loading..."
	} else {
		restPanelContent = m.rclob.ViewWithOptions(clob.ViewOptions{Width: availRWidth, Height: availHeight})
	}
	restPanel := panelStyle.
		Width(restPanelWidth - panelStyle.GetHorizontalFrameSize()).
		Height(panelHeight - panelStyle.GetVerticalFrameSize()).
		Render(restPanelContent)

	// Right Panel
	wsPanel := panelStyle.
		Width(wsPanelWidth - panelStyle.GetHorizontalFrameSize()).
		Height(panelHeight - panelStyle.GetVerticalFrameSize()).
		Render(m.wclob.ViewWithOptions(clob.ViewOptions{Width: availWWidth, Height: availHeight}))

	panels := lipgloss.JoinHorizontal(lipgloss.Top, restPanel, wsPanel)

	// Status Bar
	StatusBarContentStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("240"))

	StatusBarInfoStyle := lipgloss.NewStyle().
		Inherit(StatusBarContentStyle).
		Bold(true).
		Foreground(lipgloss.Color("255"))

	statusRefreshKey := StatusBarInfoStyle.Render("r:")
	statusRefreshVal := StatusBarContentStyle.Render(" refresh REST order book")
	statusAlignKey := StatusBarInfoStyle.Render("a:")
	statusAlignVal := StatusBarContentStyle.Render(" toggle vertical alignment")
	statusQuitKey := StatusBarInfoStyle.Render(" q:")
	statusQuitVal := StatusBarContentStyle.Render(" quit")
	statusMarket := ""
	if market != "" {
		marketType := "(Crypto)"
		if isTokenizedCache {
			marketType = "(Tokenized Equity)"
		}
		statusMarket = lipgloss.JoinHorizontal(
			lipgloss.Center,
			StatusBarInfoStyle.Render(market),
			" ",
			marketType,
			"  | ",
		)
	}
	statusBar := lipgloss.JoinHorizontal(lipgloss.Center, statusMarket, statusRefreshKey, statusRefreshVal, "  ", statusAlignKey, statusAlignVal, "  ", statusQuitKey, statusQuitVal)
	mainLayout := lipgloss.JoinVertical(
		lipgloss.Left,
		panels,
		statusBar,
	)
	return mainLayout
}

func main() {
	flag.StringVar(&market, "market", "", "the market pair to fetch")
	flag.Parse()
	p := tea.NewProgram(InitialModel(), tea.WithAltScreen())

	if _, err := p.Run(); err != nil {
		log.Fatalf("Alas, there's been an error: %v", err)
		os.Exit(1)
	}
}
