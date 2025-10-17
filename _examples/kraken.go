package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
)

const (
	restAPIBaseURL = "https://api.kraken.com/0/public"
)

type OrderBook struct {
	Asks [][]interface{} `json:"asks"`
	Bids [][]interface{} `json:"bids"`
}

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
