# Baltimore Municipal Intelligence Report

- Generated: Thu Oct 15 2026 23:33 EDT
- Sources: Open Baltimore - Part 1 Crime Data; 311 Customer Service Requests (joined on Neighborhood)
- Engine: Go go1.27.2, 16 worker goroutines per dataset, 16 CPUs

## Ingestion & Runtime Summary


| Dataset | File size | Rows read | Rows kept | Dropped | Malformed | Time |
| --- | --- | --- | --- | --- | --- | --- |
| Part 1 Crime Data | 71.7 MB | 278,239 | 222,948 | 55,291 | 0 | 176ms |
| 311 Customer Service Requests | 405.0 MB | 1,081,868 | 747,207 | 334,661 | 0 | 615ms |

- Concurrent ingest: 615ms | join/aggregate: 18ms | 5 questions: 72ms
- Neighborhoods: 290 total, 279 present in both datasets (join coverage: 99.0% of crimes, 100.0% of 311 requests)
- Heap after ingest: 187.4 MB | heap after join + GC of raw records: 3.7 MB | GC cycles: 39

## Q1. Where is combined public-safety and service burden highest?

*Stakeholder: Mayor's Office*

Research question: Which neighborhoods carry the heaviest combined load of Part 1 crime AND 311 service demand, and how concentrated is it?

- 279 neighborhoods were matched across both datasets.
- Highest combined burden: DOWNTOWN (index 78.4 of 100).
- The top 10 neighborhoods (3.6% of matched neighborhoods) generate 17.5% of crime and 17.2% of 311 demand.
- Recommendation: prioritize these neighborhoods for coordinated multi-agency investment rather than single-agency responses.

```text
Top 10 neighborhoods by composite burden index (0-100)
  DOWNTOWN                       | ████████████████████████████████████████ 78.4
  BELAIR-EDISON                  | ████████████████████████████████████████ 78.1
  FRANKFORD                      | ████████████████████████ 47.3
  BROOKLYN                       | ██████████████████████ 43.6
  CANTON                         | █████████████████████ 41.5
  SANDTOWN-WINCHESTER            | █████████████████ 33.9
  WASHINGTON VILLAGE/PIGTOWN     | ████████████████ 31.3
  COLDSTREAM HOMESTEAD MONTEB... | ███████████████ 29.8
  CENTRAL PARK HEIGHTS           | ███████████████ 29.0
  BROADWAY EAST                  | ██████████████ 28.1
```


| Rank | Neighborhood | Part 1 crimes | 311 requests | Burden index |
| --- | --- | --- | --- | --- |
| 1 | DOWNTOWN | 8,715 | 16,610 | 78.4 |
| 2 | BELAIR-EDISON | 4,890 | 29,281 | 78.1 |
| 3 | FRANKFORD | 5,238 | 10,130 | 47.3 |
| 4 | BROOKLYN | 4,380 | 10,829 | 43.6 |
| 5 | CANTON | 3,734 | 11,765 | 41.5 |
| 6 | SANDTOWN-WINCHESTER | 2,830 | 10,338 | 33.9 |
| 7 | WASHINGTON VILLAGE/PIGTOWN | 2,673 | 9,354 | 31.3 |
| 8 | COLDSTREAM HOMESTEAD MONTEBELLO | 2,167 | 10,187 | 29.8 |
| 9 | CENTRAL PARK HEIGHTS | 2,272 | 9,358 | 29.0 |
| 10 | BROADWAY EAST | 1,658 | 10,902 | 28.1 |


## Q2. Which high-crime neighborhoods are under-using 311?

*Stakeholder: Mayor's Office of Neighborhood Engagement*

Research question: Does 311 demand track crime, and which high-crime neighborhoods file unusually few 311 requests per crime (possible under-reporting)?

- Crime and 311 volume across neighborhoods have a strong positive correlation (Pearson r = 0.82).
- Citywide (matched) there are 3.39 311 requests per Part 1 crime.
- 71 neighborhoods are in the top crime quartile (>= 975 crimes); those listed below have the lowest 311 engagement relative to their crime level.
- Lowest engagement: INNER HARBOR at 0.38 requests per crime (11% of the citywide rate).
- Recommendation: target 311 awareness and community-liaison outreach here; low reporting can hide service needs.

```text
311 requests per Part 1 crime, high-crime neighborhoods (lowest = least engaged)
  INNER HARBOR            | ██████████ 0.4
  DUNBAR-BROADWAY         | ███████████████ 0.6
  CANTON INDUSTRIAL AREA  | ███████████████ 0.6
  PULASKI INDUSTRIAL AREA | ████████████████████ 0.8
  REISTERSTOWN STATION    | ████████████████████ 0.8
  OLDTOWN                 | ████████████████████████████ 1.1
  GAY STREET              | ██████████████████████████████ 1.2
  CHERRY HILL             | █████████████████████████████████ 1.3
  MOUNT VERNON            | ██████████████████████████████████████ 1.5
  ARMISTEAD GARDENS       | ████████████████████████████████████████ 1.6
```


| Neighborhood | Part 1 crimes | 311 requests | 311 per crime |
| --- | --- | --- | --- |
| INNER HARBOR | 2,755 | 1,054 | 0.38 |
| DUNBAR-BROADWAY | 1,181 | 756 | 0.64 |
| CANTON INDUSTRIAL AREA | 1,930 | 1,245 | 0.65 |
| PULASKI INDUSTRIAL AREA | 1,769 | 1,463 | 0.83 |
| REISTERSTOWN STATION | 1,530 | 1,285 | 0.84 |
| OLDTOWN | 1,568 | 1,734 | 1.11 |
| GAY STREET | 1,297 | 1,545 | 1.19 |
| CHERRY HILL | 3,323 | 4,424 | 1.33 |
| MOUNT VERNON | 2,753 | 4,259 | 1.55 |
| ARMISTEAD GARDENS | 983 | 1,528 | 1.55 |


## Q3. When do crime and service demand peak during the day?

*Stakeholder: Baltimore Police Dept. & Department of Transportation*

Research question: How do crime and 311 intake vary by hour of day, and do the peaks line up for staffing and patrol planning?

- Crime peaks at 17:00 (13,144 incidents); 311 intake peaks at 14:00 (96,625 requests).
- 37.8% of timestamped crime occurs between 20:00 and 05:00.
- 77.0% of 311 requests arrive during business hours (08:00-17:00).
- Peaks are 3 hours apart, so patrol staffing and 311/DOT crew staffing should be scheduled on different curves.

```text
Part 1 crime by hour of day (Baltimore local time)
  00:00 | ████████████████████████████████████████ 13,013
  01:00 | ███████████████████████ 7,560
  02:00 | ████████████████████ 6,695
  03:00 | █████████████████ 5,714
  04:00 | ██████████████ 4,596
  05:00 | ████████████ 4,055
  06:00 | ████████████ 3,871
  07:00 | ███████████████ 4,989
  08:00 | ██████████████████████ 7,086
  09:00 | ████████████████████████ 7,794
  10:00 | ██████████████████████████ 8,623
  11:00 | ██████████████████████████ 8,659
  12:00 | ███████████████████████████████████ 11,589
  13:00 | ████████████████████████████ 9,230
  14:00 | ██████████████████████████████ 9,839
  15:00 | █████████████████████████████████████ 12,200
  16:00 | █████████████████████████████████████ 12,322
  17:00 | ████████████████████████████████████████ 13,144
  18:00 | ████████████████████████████████████████ 13,082
  19:00 | █████████████████████████████████████ 12,248
  20:00 | ███████████████████████████████████████ 12,895
  21:00 | █████████████████████████████████████ 11,994
  22:00 | ██████████████████████████████████ 11,161
  23:00 | ████████████████████████████████ 10,589
```


```text
311 requests created by hour of day
  00:00 | ██ 4,204
  01:00 | ██ 3,630
  02:00 | █ 2,976
  03:00 | █ 2,725
  04:00 | █ 2,694
  05:00 | █ 3,431
  06:00 | ████ 10,727
  07:00 | █████████ 21,344
  08:00 | █████████████████ 40,674
  09:00 | ████████████████████ 49,141
  10:00 | ██████████████████████ 52,805
  11:00 | ████████████████████████ 56,938
  12:00 | ███████████████████████████ 65,019
  13:00 | ███████████████████████████████ 73,741
  14:00 | ████████████████████████████████████████ 96,625
  15:00 | ██████████████████████████████████ 83,272
  16:00 | ████████████████████████ 57,044
  17:00 | ███████████████ 37,139
  18:00 | ██████████ 24,012
  19:00 | ████████ 19,157
  20:00 | ██████ 14,747
  21:00 | █████ 11,602
  22:00 | ███ 8,319
  23:00 | ██ 5,241
```


## Q4. Is physical blight associated with crime?

*Stakeholder: Department of Planning / Housing & Community Development*

Research question: Do neighborhoods with more blight-related 311 requests (trash, rats, vacants, dumping) also experience more Part 1 crime?

- 51.5% of all 311 requests are blight/sanitation related.
- Blight requests and crime have a moderate positive correlation (r = 0.64).
- Neighborhoods in the highest-blight quartile average 6.5x the crime of the lowest-blight quartile.
- Caveat: correlation is not causation; both may share underlying drivers such as vacancy and disinvestment.
- Recommendation: pair code enforcement and vacant-lot remediation with public-safety strategy in Q4 neighborhoods.

```text
Average Part 1 crimes per neighborhood, by blight-request quartile
  Q1 (least blight) | ██████ 255
  Q2                | ███████████ 462
  Q3                | ███████████████████ 778
  Q4 (most blight)  | ████████████████████████████████████████ 1,660
```


| Blight quartile | Neighborhoods | Blight requests (range) | Avg crimes / neighborhood |
| --- | --- | --- | --- |
| Q1 (least blight) | 69 | 1 - 265 | 255.3 |
| Q2 | 70 | 283 - 933 | 462.3 |
| Q3 | 70 | 934 - 1,852 | 778.1 |
| Q4 (most blight) | 70 | 1,882 - 21,200 | 1660.4 |


## Q5. Are 311 requests resolved equitably across crime levels?

*Stakeholder: Department of Public Works / Office of Equity*

Research question: Do high-crime neighborhoods wait longer for their 311 requests to be closed than low-crime neighborhoods?

- 725,565 closed 311 requests had measurable resolution times.
- Highest-crime neighborhoods wait a median 22.1 h vs 20.5 h in the lowest-crime ones (8% longer).
- Recommendation: set equity-based service-level targets so resolution time does not depend on neighborhood.

```text
Median hours to close a 311 request, by neighborhood crime quartile
  Q1 (lowest crime)  | █████████████████████████████████████ 20.5 h
  Q2                 | ███████████████████████████████████████ 22.0 h
  Q3                 | ████████████████████████████████████████ 22.3 h
  Q4 (highest crime) | ████████████████████████████████████████ 22.1 h
```


| Crime quartile | Neighborhoods | Closed requests | Median hrs | 90th pct hrs |
| --- | --- | --- | --- | --- |
| Q1 (lowest crime) | 69 | 38,501 | 20.5 | 874.6 |
| Q2 | 70 | 103,924 | 22.0 | 766.5 |
| Q3 | 70 | 186,372 | 22.3 | 678.5 |
| Q4 (highest crime) | 70 | 396,768 | 22.1 | 621.5 |

