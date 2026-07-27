package main

import (
	"encoding/json"
	"log"

	tea "charm.land/bubbletea/v2"
	"github.com/allank/chartea/clob"
	"github.com/gorilla/websocket"
)

// WSSubscriptionRequest represents the payload to subscribe to a channel.
type WSSubscriptionRequest struct {
	Method string               `json:"method"`
	Params WSSubscriptionParams `json:"params"`
	ReqID  int                  `json:"req_id,omitempty"`
}

type WSSubscriptionParams struct {
	Channel  string   `json:"channel"`
	Symbol   []string `json:"symbol"`
	Depth    int      `json:"depth,omitempty"`
	Snapshot bool     `json:"snapshot"`
}

// WSBookMessage represents the outer envelope of a message from the book channel.
type WSBookMessage struct {
	Channel string       `json:"channel"`
	Type    string       `json:"type"`
	Data    []WSBookData `json:"data"`
}

// WSBookData contains the actual orderbook data.
type WSBookData struct {
	Symbol   string         `json:"symbol"`
	Asks     []WSPriceLevel `json:"asks"`
	Bids     []WSPriceLevel `json:"bids"`
	Checksum int64          `json:"checksum"`
}

// WSPriceLevel is a price and quantity pair.
type WSPriceLevel struct {
	Price float64 `json:"price"`
	Qty   float64 `json:"qty"`
}

// wsOrderBookMsg is dispatched to Bubble Tea when a new websocket frame arrives.
type wsOrderBookMsg struct {
	IsSnapshot bool
	Asks       []clob.Order
	Bids       []clob.Order
}

// connectWebSocket dials Kraken's v2 websocket and drives a read loop
// that sends updates to the Bubble Tea program `p`.
func connectWebSocket(p *tea.Program, symbol string) {
	u := "wss://ws.kraken.com/v2"
	c, _, err := websocket.DefaultDialer.Dial(u, nil)
	if err != nil {
		log.Fatal("dial:", err)
	}

	subscribeMsg := WSSubscriptionRequest{
		Method: "subscribe",
		Params: WSSubscriptionParams{
			Channel:  "book",
			Symbol:   []string{symbol},
			Depth:    25,
			Snapshot: true,
		},
		ReqID: 1,
	}

	err = c.WriteJSON(subscribeMsg)
	if err != nil {
		log.Println("write:", err)
		return
	}

	go func() {
		defer c.Close()
		for {
			_, message, err := c.ReadMessage()
			if err != nil {
				log.Println("read:", err)
				return
			}

			// Try to parse it as a book message
			var bookMsg WSBookMessage
			if err := json.Unmarshal(message, &bookMsg); err != nil {
				continue // likely a heartbeat, status, or unrelated message
			}

			if bookMsg.Channel != "book" || len(bookMsg.Data) == 0 {
				continue
			}

			// We have a book message. Dispatch to Bubble Tea.
			data := bookMsg.Data[0]
			msgOut := wsOrderBookMsg{
				IsSnapshot: bookMsg.Type == "snapshot",
				Asks:       make([]clob.Order, len(data.Asks)),
				Bids:       make([]clob.Order, len(data.Bids)),
			}

			for i, ask := range data.Asks {
				msgOut.Asks[i] = clob.Order{Price: ask.Price, Volume: ask.Qty}
			}
			for i, bid := range data.Bids {
				msgOut.Bids[i] = clob.Order{Price: bid.Price, Volume: bid.Qty}
			}

			p.Send(msgOut)
		}
	}()
}
