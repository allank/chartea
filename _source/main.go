package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/gorilla/websocket"
)

const (
	restAPIBaseURL = "https://api.kraken.com/0/public"
	wsAPIBaseURL   = "wss://ws.kraken.com/"
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

func main() {
	if len(os.Args) < 2 {
		log.Fatalf("Usage: %s <market_pair> (e.g., BTC/USD)", os.Args[0])
	}
	// marketPair := strings.ToUpper(os.Args[1])
	marketPair := os.Args[1]

	isTokenized := false
	fmt.Println("Fetching asset information...")

	// First, check if the pair is a crypto asset
	fmt.Println("Checking for crypto asset...")
	cryptoPairs, err := getAssetPairs("currency")
	if err != nil {
		log.Fatalf("Error fetching crypto asset pairs: %v", err)
	}

	pairInfo, found := findPair(cryptoPairs, marketPair)
	var allPairs map[string]AssetPairInfo

	if found {
		fmt.Println("Found as a crypto asset.")
		allPairs = cryptoPairs
	} else {
		// If not found, check if it is a tokenized asset
		fmt.Println("Not found as a crypto asset, checking for tokenized asset...")
		tokenizedPairs, err := getAssetPairs("tokenized_asset")
		if err != nil {
			log.Fatalf("Error fetching tokenized asset pairs: %v", err)
		}

		pairInfo, found = findPair(tokenizedPairs, marketPair)
		if !found {
			log.Fatalf("Market pair '%s' not found as a crypto or tokenized asset.", marketPair)
		}
		fmt.Println("Found as a tokenized asset.")
		isTokenized = true
		allPairs = tokenizedPairs
	}

	fmt.Printf("Found pair: %s, Asset Class: %s, Base: %s, Quote: %s\n\n",
		pairInfo.WSName, pairInfo.AssetClass, pairInfo.Base, pairInfo.Quote)

	// fmt.Printf("%+v", pairInfo)
	// --- REST API Order Book ---
	fmt.Println("--- REST API Order Book ---")
	// Use the original key from the map for the REST call, which might be different from wsname
	var restPairKey string
	for key, pi := range allPairs {
		if pi.WSName == pairInfo.WSName {
			restPairKey = key
			break
		}
	}

	orderBook, err := getRestOrderBook(restPairKey, isTokenized)
	if err != nil {
		log.Fatalf("Error getting REST order book: %v", err)
	}

	fmt.Println("Bids:")
	for _, bid := range orderBook.Bids {
		fmt.Printf("  Price: %s, Volume: %s\n", bid[0].(string), bid[1].(string))
	}
	fmt.Println("Asks:")
	for _, ask := range orderBook.Asks {
		fmt.Printf("  Price: %s, Volume: %s\n", ask[0].(string), ask[1].(string))
	}
	fmt.Println("------------------------------------")

	// --- WebSocket API Connection ---
	fmt.Printf("\nConnecting to WebSocket for real-time data on %s...\n", pairInfo.WSName)
	fmt.Println("Press CTRL+C to exit.")

	conn, _, err := websocket.DefaultDialer.Dial(wsAPIBaseURL, nil)
	if err != nil {
		log.Fatalf("WebSocket dial error: %v", err)
	}
	defer conn.Close()

	// Subscribe to the order book
	subscription := map[string]interface{}{
		"event": "subscribe",
		"pair":  []string{pairInfo.WSName},
		"subscription": map[string]string{
			"name": "book",
		},
	}
	if err := conn.WriteJSON(subscription); err != nil {
		log.Fatalf("WebSocket subscription failed: %v", err)
	}

	// Handle graceful shutdown
	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt, syscall.SIGTERM)
	done := make(chan struct{})

	var initialSnapshotReceived = false

	go func() {
		defer close(done)
		for {
			_, message, err := conn.ReadMessage()
			if err != nil {
				log.Println("WebSocket read error:", err)
				return
			}

			// We expect array messages for book data
			if strings.HasPrefix(string(message), "[") {
				var bookData []interface{}
				if err := json.Unmarshal(message, &bookData); err != nil {
					log.Printf("Error unmarshaling websocket message: %v", err)
					continue
				}

				// The book data is inside a map within the received array. We loop to find it.
				for _, item := range bookData {
					if data, ok := item.(map[string]interface{}); ok {

						// ADDED: Logic to handle the initial full order book snapshot.
						// The snapshot uses "as" and "bs" keys for asks and bids.
						if !initialSnapshotReceived {
							// Check if both 'as' and 'bs' keys exist to confirm it's a snapshot.
							if bids, bidsOk := data["bs"].([]interface{}); bidsOk {
								if asks, asksOk := data["as"].([]interface{}); asksOk {
									fmt.Println("\n--- WebSocket Initial Order Book (Top 5) ---")
									fmt.Println("Bids:")
									// Loop through and print the top 5 bids from the snapshot.
									for i, bid := range bids {
										if i >= 5 {
											break
										}
										if bidData, ok := bid.([]interface{}); ok && len(bidData) >= 2 {
											fmt.Printf("  Price: %s, Volume: %s\n", bidData[0], bidData[1])
										}
									}
									fmt.Println("Asks:")
									// Loop through and print the top 5 asks from the snapshot.
									for i, ask := range asks {
										if i >= 5 {
											break
										}
										if askData, ok := ask.([]interface{}); ok && len(askData) >= 2 {
											fmt.Printf("  Price: %s, Volume: %s\n", askData[0], askData[1])
										}
									}
									fmt.Println("------------------------------------------")
									// Set the flag to true so we don't process a snapshot again.
									initialSnapshotReceived = true
									break // Snapshot processed, no need to check other items in this message.
								}
							}
						}

						// Updates use "a" and "b" keys.
						isUpdate := false
						if _, ok := data["a"]; ok {
							isUpdate = true
						}
						if _, ok := data["b"]; ok {
							isUpdate = true
						}

						if isUpdate && initialSnapshotReceived {
							fmt.Println("\n--- WebSocket Order Book Update ---")
							if bids, ok := data["b"].([]interface{}); ok && len(bids) > 0 {
								fmt.Println("Top Bid Update:")
								if topBid, ok := bids[0].([]interface{}); ok {
									fmt.Printf("  Price: %s, Volume: %s\n", topBid[0], topBid[1])
								}
							}
							if asks, ok := data["a"].([]interface{}); ok && len(asks) > 0 {
								fmt.Println("Top Ask Update:")
								if topAsk, ok := asks[0].([]interface{}); ok {
									fmt.Printf("  Price: %s, Volume: %s\n", topAsk[0], topAsk[1])
								}
							}
						}
					}
				}
			} else {
				fmt.Printf("WebSocket Info: %s\n", message)
			}
		}
	}()

	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-done:
			return
		case <-ticker.C:
			// Send a ping to keep the connection alive
			err := conn.WriteMessage(websocket.PingMessage, nil)
			if err != nil {
				log.Println("write ping:", err)
				return
			}
		case <-interrupt:
			log.Println("Interrupt received, closing connection.")
			// Cleanly close the connection by sending a close message
			err := conn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
			if err != nil {
				log.Println("write close:", err)
				return
			}
			select {
			case <-done:
			case <-time.After(time.Second):
			}
			return
		}
	}
}
