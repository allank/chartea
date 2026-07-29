# Grouped price Buckets are represented as Order, not a distinct type

`OrderBook.GroupedBids`/`GroupedAsks` (see #17) return `[]Order` — the same type used for raw, ungrouped price levels — rather than a new `Bucket` type carrying extra metadata such as how many raw Orders were merged into it, or the bucket's price range.

We chose this so grouped output is a drop-in replacement anywhere `[]Order` is already consumed (rendering, further processing), with no downstream code needing to branch on type. The trade-off: a Bucket's `Volume` is a sum with no record of its constituent levels — anything wanting per-level detail within a bucket must work from `Bids()`/`Asks()` directly, not the grouped view.

This trade-off is only acceptable because Grouping is non-destructive: `GroupedBids`/`GroupedAsks` are pure reads that never mutate or replace the OrderBook's underlying `bids`/`asks`. Full granularity is always retained, so re-grouping at a finer Increment — or dropping grouping entirely — is always just another call away, never blocked by having previously grouped at a coarser one. If a future change made grouping destructive (e.g. grouping in place, or caching a grouped result as the book's new canonical state), this decision would need revisiting.
