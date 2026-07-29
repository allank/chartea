# Chartea Domain Context

Terminal UI component library and examples for rendering Central Limit Order Books (CLOB).

## Language

### Order Book

**Order Book**:
The aggregated set of buy (bids) and sell (asks) orders for a trading pair.
_Avoid_: Market depth, book state

**Order**:
A single price/volume pair — the fundamental unit of an Order Book, representing the total volume resting at one price.
_Avoid_: Price level, level, entry

**Bid**:
An Order held in the Order Book's buy-side collection. Order itself carries no field marking it a Bid — bid-ness is purely positional, determined by which collection holds it.
_Avoid_: Buy order

**Ask**:
An Order held in the Order Book's sell-side collection, with the same positional-only distinction as Bid.
_Avoid_: Sell order, offer

**Side**:
Which portion of the Order Book an operation or view applies to: Bids, Asks, or Both. Both means no restriction to one side, not a third kind of side.
_Avoid_: Direction, mode (Mode already means data source mode — see below)

### Grouping

**Grouping**:
Aggregating adjacent Orders into coarser price Buckets by a chosen Increment, to show liquidity at a lower resolution than the raw book.
_Avoid_: Precision, binning

**Increment**:
The bucket size used when Grouping — e.g. 10 or 0.0001. Supplied by the caller per grouping request; the Order Book has no opinion on which increments are valid for a given pair.
_Avoid_: Precision (collides with `PricePrecision`/`VolumePrecision`, which mean decimal display precision, not bucket size), tick size

**Bucket**:
A single aggregated price level produced by Grouping: one or more Orders whose prices round to the same Increment boundary — down for Bids, up for Asks — with volumes summed.
_Avoid_: Group, level

### Data Source Modes

**Static Mode**:
A data source mode that fetches the order book once via REST API on startup or manual refresh.
_Avoid_: REST mode, snapshot-only mode, pull mode

**Realtime Mode**:
A data source mode that streams continuous order book updates over a WebSocket connection.
_Avoid_: Live mode, WebSocket mode, push mode

**Mock Mode**:
A data source mode using synthetic generated orders for offline testing and visualization without external network dependencies.
_Avoid_: Fake mode, dummy mode
