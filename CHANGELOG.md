# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [0.2.0] - 2026-07-29

Non-breaking: adds new API, no existing public API removed or changed.

### Added

- `OrderBook.GroupedBids(increment)` / `GroupedAsks(increment)` aggregate the book into price Buckets of a given Increment — bids rounded down, asks rounded up, volumes summed within each Bucket. See [README: Grouping](README.md#grouping).
- `Model.Sides` (`Side`: `Both`/`BidsOnly`/`AsksOnly`) renders only bids, only asks, or both. The default (`Both`) is unchanged from prior behavior. See [README: Sides](README.md#sides).

### Fixed

- The example's Static Panel no longer receives Realtime Mode WebSocket updates, matching `CONTEXT.md`'s definition of Static Mode (REST fetch on startup or manual refresh only).
- The README's Usage and Dimensions examples now use the Bubble Tea v2 API (`charm.land/bubbletea/v2`) this module actually depends on, rather than an outdated v1-style API that didn't compile.

## [0.1.0] - 2026-07-28

First tagged release.

### Breaking Changes

- `OrderBook.Bids` and `OrderBook.Asks` are no longer public fields. Read the book via the new `Bids()`/`Asks()` accessor methods; populate or update it via `ApplySnapshot`/`ApplyDelta` instead of assigning the fields directly. See [README: The order book](README.md#the-order-book) for migration guidance.

### Added

- `OrderBook.ApplySnapshot(asks, bids []Order)` replaces the book's contents wholesale — for an initial load or a full snapshot from an exchange.
- `OrderBook.ApplyDelta(asks, bids []Order)` merges incremental updates into the book by price — for streaming L2 updates.
- `OrderBook.Bids()` / `OrderBook.Asks()` accessor methods, returning the book sorted (bids descending, asks ascending) by price.

### Fixed

- `ViewWithOptions` no longer sorts the order book as a side effect of rendering — the sort/dedup invariant is now established once, on write, by `ApplySnapshot`/`ApplyDelta`.
- Vertical orientation with right-aligned volume bars no longer skips the row-rendering width constraint the other three bid/ask × orientation combinations already applied. The four near-identical row renderers are now a single implementation.

### Removed

- The stale `_source/main.go` prototype, a pre-Bubble Tea implementation with its own duplicate `OrderBook` and Kraken response type definitions.
