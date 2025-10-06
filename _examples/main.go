package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"math"
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/allank/chartea/clob"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

var market string

var (
	orderBookCache   *OrderBook
	isTokenizedCache bool
)

func fetchOrderBook(marketPair string, forceRefetch bool) (*OrderBook, bool, error) {
	if forceRefetch {
		orderBookCache = nil
	}
	if orderBookCache != nil {
		return orderBookCache, isTokenizedCache, nil
	}

	// First, check if the pair is a crypto asset
	cryptoPairs, err := getAssetPairs("currency")
	if err != nil {
		return nil, false, fmt.Errorf("Error fetching crypto asset pairs: %v", err)
	}

	pairInfo, found := findPair(cryptoPairs, marketPair)
	var allPairs map[string]AssetPairInfo
	isTokenized := false

	if found {
		allPairs = cryptoPairs
	} else {
		// If not found, check if it is a tokenized asset
		tokenizedPairs, err := getAssetPairs("tokenized_asset")
		if err != nil {
			return nil, false, fmt.Errorf("Error fetching tokenized asset pairs: %v", err)
		}

		pairInfo, found = findPair(tokenizedPairs, marketPair)
		if !found {
			return nil, false, fmt.Errorf("Market pair '%s' not found as a crypto or tokenized asset.", marketPair)
		}
		isTokenized = true
		allPairs = tokenizedPairs
	}

	var restPairKey string
	for key, pi := range allPairs {
		if pi.WSName == pairInfo.WSName {
			restPairKey = key
			break
		}
	}

	orderBook, err := getRestOrderBook(restPairKey, isTokenized)
	if err != nil {
		return nil, false, fmt.Errorf("Error getting REST order book: %v", err)
	}

	orderBookCache = orderBook
	isTokenizedCache = isTokenized

	return orderBook, isTokenized, nil
}

const (
	restAPIBaseURL = "https://api.kraken.com/0/public"
)

// Structs for unmarshaling Kraken REST API responses
type AssetPairsResponse struct {
	Error  []string                 `json:"error"`
	Result map[string]AssetPairInfo `json:"result"`
}

type AssetPairInfo struct {
	WSName     string `json:"wsname"`
	Base       string `json:"base"`
	Quote      string `json:"quote"`
	AssetClass string // Custom field to store asset class
}

type OrderBookResponse struct {
	Error  []string             `json:"error"`
	Result map[string]OrderBook `json:"result"`
}

type OrderBook struct {
	Asks [][]interface{} `json:"asks"`
	Bids [][]interface{} `json:"bids"`
}

// getAssetPairs fetches asset pairs for a given asset class from Kraken.
func getAssetPairs(assetClass string) (map[string]AssetPairInfo, error) {
	url := fmt.Sprintf("%s/AssetPairs", restAPIBaseURL)
	if assetClass != "" {
		url = fmt.Sprintf("%s?aclass_base=%s", url, assetClass)
	}

	resp, err := http.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to get asset pairs: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("bad status from Kraken API: %s", resp.Status)
	}

	var assetPairsResponse AssetPairsResponse
	if err := json.NewDecoder(resp.Body).Decode(&assetPairsResponse); err != nil {
		return nil, fmt.Errorf("failed to decode asset pairs response: %w", err)
	}

	if len(assetPairsResponse.Error) > 0 {
		return nil, fmt.Errorf("kraken API error: %v", assetPairsResponse.Error)
	}

	// Add the asset class to each pair info
	for key, pair := range assetPairsResponse.Result {
		pair.AssetClass = assetClass
		assetPairsResponse.Result[key] = pair
	}

	return assetPairsResponse.Result, nil
}

// findPair searches for a given market pair in the combined list of asset pairs.
func findPair(allPairs map[string]AssetPairInfo, marketPair string) (AssetPairInfo, bool) {
	// Kraken API might use XBT for BTC, so we check for that common case
	marketPair = strings.ToUpper(marketPair)
	normalizedPair := strings.Replace(marketPair, "BTC", "XBT", -1)

	for _, pairInfo := range allPairs {
		// Use WSName for matching as it's used in WebSocket subscriptions
		pairUpper := strings.ToUpper(pairInfo.WSName)
		if pairUpper == marketPair || pairUpper == normalizedPair {
			return pairInfo, true
		}
	}
	return AssetPairInfo{}, false
}

// getRestOrderBook fetches the order book for a given pair via the REST API.
func getRestOrderBook(pair string, isTokenized bool) (*OrderBook, error) {
	url := fmt.Sprintf("%s/Depth?pair=%s", restAPIBaseURL, pair)
	if isTokenized {
		url = fmt.Sprintf("%s&asset_class=tokenized_asset", url)
	}
	resp, err := http.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to get order book: %w", err)
	}
	defer resp.Body.Close()
	var orderBookResponse OrderBookResponse
	if err := json.NewDecoder(resp.Body).Decode(&orderBookResponse); err != nil {
		return nil, fmt.Errorf("failed to decode order book response: %w", err)
	}

	if len(orderBookResponse.Error) > 0 {
		return nil, fmt.Errorf("kraken API error on order book fetch: %v", orderBookResponse.Error)
	}

	// The result map has one key which is the pair name
	for _, book := range orderBookResponse.Result {
		return &book, nil
	}

	return nil, fmt.Errorf("order book not found in response for pair %s", pair)
}

func init() {
}
type refetchMsg struct{}

// mainModel represents the state of our TUI application.
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
	statusBar := lipgloss.JoinHorizontal(lipgloss.Center, statusRefreshKey, statusRefreshVal, "  ", statusAlignKey, statusAlignVal, "  ", statusQuitKey, statusQuitVal)

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
