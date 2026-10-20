# Baltimore Civic Pulse: 311 Requests × Vacant Building Rehab Locations

Generated: 2026-10-11T22:00:34-04:00  
Mode: Saved Open Baltimore raw data  
Parsing workers: 8  
Elapsed: 7.581s

## Executive summary

The pipeline processed **692,663 311 service requests with a neighborhood label** and **12,647 vacant-building rehab locations**. It matched records by Baltimore neighborhood name across **234 neighborhoods**. The neighborhood-level Spearman rank correlation between 2026 request volume and rehab locations is **0.770** (strong positive association). This is a descriptive comparison of two administrative datasets, not evidence that rehab activity causes service requests or vice versa.

## Five stakeholder questions

### 1. Which neighborhoods show the strongest combined signal of service demand and rehab locations?

The table ranks neighborhoods by the sum of their standardized 311 and rehab-location counts. It highlights places high on both city measures; it is not a per-resident rate.

| Neighborhood | 311 requests | Rehab locations | Non-closed 311 | Requests per 100 rehab locations |
|---|---:|---:|---:|---:|
| Belair-Edison | 24,775 | 373 | 650 | 6642.1 |
| Coldstream Homestead Montebello | 16,414 | 367 | 451 | 4472.5 |
| Broadway East | 9,466 | 445 | 292 | 2127.2 |
| McElderry Park | 8,905 | 382 | 259 | 2331.2 |
| Brooklyn | 11,883 | 253 | 332 | 4696.8 |

### 2. Where are the largest concentrations of vacant-building rehab locations?

| Neighborhood | 311 requests | Rehab locations | Non-closed 311 | Requests per 100 rehab locations |
|---|---:|---:|---:|---:|
| Broadway East | 9,466 | 445 | 292 | 2127.2 |
| McElderry Park | 8,905 | 382 | 259 | 2331.2 |
| Belair-Edison | 24,775 | 373 | 650 | 6642.1 |
| Coldstream Homestead Montebello | 16,414 | 367 | 451 | 4472.5 |
| Oliver | 7,544 | 337 | 275 | 2238.6 |

### 3. Which neighborhoods have the most non-closed 311 requests?

Non-closed means a request with a recorded status that is not marked closed, complete, resolved, or cancelled. These counts are a workload snapshot; age, service type, and agency-specific completion rules are not adjusted.

| Neighborhood | 311 requests | Rehab locations | Non-closed 311 | Requests per 100 rehab locations |
|---|---:|---:|---:|---:|
| Downtown | 13,502 | 33 | 834 | 40915.2 |
| Canton | 13,051 | 82 | 783 | 15915.9 |
| Belair-Edison | 24,775 | 373 | 650 | 6642.1 |
| Coldstream Homestead Montebello | 16,414 | 367 | 451 | 4472.5 |
| Frankford | 9,174 | 157 | 437 | 5843.3 |

### 4. Which 311 request types are more common in high-rehab neighborhoods?

High-rehab neighborhoods are the top quartile among neighborhoods with at least one rehab location (cutoff: **69 rehab locations**). Enrichment compares each request type's share in that group with its share in other matched neighborhoods; values above 1 mean relatively more common. Low counts can produce unstable ratios.

| Request type | Requests in high-rehab areas | Share of those requests | Enrichment vs. other areas |
|---|---:|---:|---:|
| SW-Boarding | 2,995 | 0.9% | 3.83x |
| SW-Cleaning | 4,370 | 1.3% | 3.71x |
| HCD-Vacant Building Squatter Encounter (Internal BCFD USE Only) | 199 | 0.1% | 3.53x |
| HCD-Vacant Building | 4,724 | 1.4% | 2.65x |
| SW-Dirty Alley | 17,505 | 5.2% | 2.52x |

### 5. Do the two indicators tend to rank neighborhoods similarly?

Spearman rank correlation = **0.770** (strong positive association), calculated over 234 neighborhoods with both measures. A positive association means that neighborhoods ranked higher on one measure tend to rank higher on the other; it does not explain why.

## Visual summaries

### Most 311 requests by neighborhood (2026)

```text
Belair-Edison             24775 ██████████████████████████████
Coldstream Homestead Mo…  16414 ████████████████████
Downtown                  13502 ████████████████
Canton                    13051 ████████████████
Brooklyn                  11883 ██████████████
Central Park Heights      10047 ████████████
Washington Village/Pigt…   9574 ████████████
Broadway East              9466 ███████████
Frankford                  9174 ███████████
Hampden                    9058 ███████████
```

### Most vacant-building rehab locations

```text
Broadway East               445 ██████████████████████████████
McElderry Park              382 ██████████████████████████
Belair-Edison               373 █████████████████████████
Coldstream Homestead Mo…    367 █████████████████████████
Oliver                      337 ███████████████████████
Central Park Heights        265 ██████████████████
Brooklyn                    253 █████████████████
Washington Village/Pigt…    248 █████████████████
Middle East                 238 ████████████████
Ellwood Park/Monument       228 ███████████████
```

## Data, method, and limitations

- **311 source:** [Baltimore City 311 Customer Service Requests 2026](https://services1.arcgis.com/UWYHeuuJISiGmgXx/ArcGIS/rest/services/311_Customer_Service_Requests_current/FeatureServer/0). The city describes this as 2026 requests, updated daily. The query filters to records with a neighborhood value.
- **Housing source:** [Vacant Building Rehabs](https://data.baltimorecity.gov/datasets/baltimore::vacant-building-rehabs), a citywide inventory of locations where vacant buildings have been rehabilitated.
- The app saves the paginated raw 311 GeoJSON responses and the raw rehab GeoJSON snapshot under `data/raw/`; its parser joins on normalized neighborhood names. Missing or unmatched labels are excluded from neighborhood-level comparisons.
- The feeds cover different time windows: 311 records are for calendar year 2026; rehab locations span the years included in the city inventory.
- The comparison is not population-normalized, does not imply property-level matches, and may reflect reporting access or data quality. Rehab records do not establish whether a building remains vacant today or whether a project is complete. Use the output to frame follow-up analysis, not as a stand-alone basis for resource allocation.
- The 311 “non-closed” count uses the recorded status field and a broad terminal-status rule; confirm against agency definitions before operational use.

## Reproduce

From this directory, run `go run .` to fetch current records, save raw pages, and write this report, `summary.json`, and `neighborhoods.csv` to `reports/`. For the bundled raw excerpt, run `go run . -offline`.
