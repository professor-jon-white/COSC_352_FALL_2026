# Code Walkthrough

Use this guide to understand and explain the implementation. Start with the README for setup and data limitations.

## Follow one record

- **Read:** `internal/ingest.Load` scans one JSONL feature and gives it a sequence number.
- **Parse:** A worker unmarshals its JSON into `model.Feature` and normalizes its neighborhood.
- **Validate:** The dispatcher restores source order, checks required fields and date range, and excludes duplicates.
- **Aggregate:** A hash assigns its neighborhood to one aggregation worker, which updates only its own map.
- **Join:** `analysis.Join` combines each neighborhood's request statistics with its distinct vacancy-property count.
- **Analyze:** Percentiles, vacancy cohorts, category totals and closure medians answer the five research questions.
- **Present:** `report.Write` creates readable results; `report.CSV` emits the summary table.

## Why these choices matter

- **Aggregate before joining:** Otherwise one neighborhood's many requests and many vacancy notices could form a many-to-many join and inflate counts.
- **Use owned maps:** Go maps are not safe for unsynchronized concurrent writes; assigning one writer to each map avoids that problem.
- **Restore input order:** Without it, parallel parsing could change which duplicate record wins.
- **Bound the queues:** A fast reader cannot indefinitely outrun slower parsing or aggregation.
- **Compare the same request types:** Comparing a mix of parking requests with a mix of trash requests could mistake different work types for a difference in closure performance.
- **Keep unmatched rows visible:** A failed match is not proof of zero vacancy or zero demand.

## Questions you should be able to answer

- Why is `BLOCKLOT` preferred over notice number for counting properties?
- What does a channel do, and who closes each channel?
- Which maps are written by one goroutine, and which are read only?
- What data still grows with the number of records despite streaming?
- Why do one-worker and four-worker outputs match exactly?
- How are the low/high cohorts selected, and how are ties handled?
- Why is the closure metric not a measure of successful resolution?
- Why can a current vacancy snapshot not establish past vacancy conditions?
- How could missing neighborhood values bias the findings?
- Why should city staff validate the priority list before using it?

## Reasonable future improvements

Consider adding population or property denominators, historical vacancy snapshots, finer-grained status/outcome validation, and a documented spatial lookup for requests with coordinates but no neighborhood. Those would change the analytical scope and require additional validation; they are not silently assumed in this version.
