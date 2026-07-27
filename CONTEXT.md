# Chartea Domain Context

Terminal UI component library and examples for rendering Central Limit Order Books (CLOB).

## Language

**Order Book**:
The aggregated set of buy (bids) and sell (asks) orders for a trading pair.
_Avoid_: Market depth, book state

**Static Mode**:
A data source mode that fetches the order book once via REST API on startup or manual refresh.
_Avoid_: REST mode, snapshot-only mode, pull mode

**Realtime Mode**:
A data source mode that streams continuous order book updates over a WebSocket connection.
_Avoid_: Live mode, WebSocket mode, push mode

**Mock Mode**:
A data source mode using synthetic generated orders for offline testing and visualization without external network dependencies.
_Avoid_: Fake mode, dummy mode
