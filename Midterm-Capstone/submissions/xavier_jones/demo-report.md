# Baltimore Public Safety: Crime and Arrest Workload

Generated: 2026-10-15T02:55:58Z

**SYNTHETIC DEMO: invented records, not Baltimore findings.**

Inspired by the [Public Safety Accountability Dashboard](https://baltimore-psad.github.io/), using two raw Open Baltimore datasets. Independent class project; its definitions and totals do not reproduce the official dashboard.

**Coverage:** 60 crime rows, 60 distinct nonempty crime incident IDs; 40 arrest rows, 40 distinct nonempty arrest numbers. No person-level or case-level linkage is assumed.

**Geographic coverage:** 100.0% of crime rows and 100.0% of arrest rows have a neighborhood. Geographic comparisons describe only the geocoded portion; missing arrest neighborhoods can materially change the picture.

## 1. Which neighborhoods have the largest recorded crime workload, and how does arrest activity compare?

For citywide coordination: full outer join of neighborhood aggregates; top 15 by crime rows. The two location fields describe different activities. Counts are records, not unique people or population-adjusted risk.

| Group | Crime records | Arrest records |
|---|---:|---:|
| DEMO EAST | 20 | 20 |
| DEMO NORTH | 20 | 0 |
| DEMO SOUTH | 20 | 20 |

## 2. How do monthly crime and arrest workloads change?

For staffing review: full outer join of America/New_York calendar-month aggregates. The source date filter uses ArcGIS SQL timestamps; month labels use returned Unix timestamps converted to Eastern time. A zero means no downloaded rows in that cell, not absence of real activity.

| Group | Crime records | Arrest records |
|---|---:|---:|
| 2025-01 | 31 | 20 |
| 2025-02 | 28 | 18 |
| 2025-03 | 1 | 2 |

## 3. What offense mix is recorded in the neighborhoods with the most arrest activity?

For investigative resource planning: select the five known neighborhoods with the most arrest rows, join to crime by neighborhood, and compare crime Description shares with all other known neighborhoods. This describes contextual workload; arrests are not attributed to these offenses.

Selected neighborhoods: DEMO EAST, DEMO SOUTH. Their arrest records: 40. Crime rows with known neighborhood: 40 selected, 20 elsewhere.

| Source category | Selected crime rows | Selected mix | Elsewhere crime rows | Elsewhere mix |
|---|---:|---:|---:|---:|
| ASSAULT | 20 | 50.0% | 0 | 0.0% |
| BURGLARY | 20 | 50.0% | 0 | 0.0% |
| ROBBERY | 0 | 0.0% | 20 | 100.0% |

## 4. Where are crime-record and arrest-record shares most different?

For reviewing geographic workload distribution: compare each neighborhood's share of crime rows with its share of arrest rows, using known-neighborhood rows only. Sort by absolute percentage-point difference. Positive means crime-record share is larger. The gap is not a clearance rate, policing adequacy measure, or performance score.

Known-neighborhood denominators: 60 crime rows; 40 arrest rows.

| Neighborhood | Crime share | Arrest share | Difference (points) |
|---|---:|---:|---:|
| DEMO NORTH | 33.3% | 0.0% | +33.3 |
| DEMO EAST | 33.3% | 50.0% | -16.7 |
| DEMO SOUTH | 33.3% | 50.0% | -16.7 |

```text
Share difference: + crime share larger; - arrest share larger
One # represents approximately one percentage point.
DEMO NORTH                 +33.3 +#################################
DEMO EAST                  -16.7 -#################
DEMO SOUTH                 -16.7 -#################
```

## 5. What weapon profiles and arrest workload occur in neighborhoods with the most crime records?

For violence-prevention program planning: select the five known neighborhoods with the most crime rows, join their arrest totals by neighborhood, and compare source Weapon shares with other known neighborhoods. Weapon labels are not inferred or collapsed into a firearm classification. The geographic association does not link an arrest to a weapon event.

Selected neighborhoods: DEMO EAST, DEMO NORTH, DEMO SOUTH. Their arrest records: 40. Crime rows with known neighborhood: 60 selected, 0 elsewhere.

| Source category | Selected crime rows | Selected mix | Elsewhere crime rows | Elsewhere mix |
|---|---:|---:|---:|---:|
| HANDGUN | 20 | 33.3% | 0 | 0.0% |
| NONE | 20 | 33.3% | 0 | 0.0% |
| UNKNOWN | 20 | 33.3% | 0 | 0.0% |

## Data quality and interpretation

| Check | Crime rows | Arrest rows |
|---|---:|---:|
| Missing incident ID | 0 | 0 |
| Missing neighborhood | 0 | 0 |
| Invalid/missing date | 0 | 0 |

Exact normalized ID overlaps found: 40. This is a diagnostic only: the sources use different identifier conventions and many arrest IDs are absent. All five analyses use aggregate neighborhood or time joins.

Neighborhood names are uppercased and whitespace-normalized. No spatial or fuzzy joins are inferred. UNKNOWN neighborhoods appear in overall workload tables but are excluded from geographic selections and share denominators. Crime rows may represent multiple offenses or victims, and arrest rows multiple charges; neither is a count of unique people. Arrest geography is not necessarily crime geography. Missing neighborhoods and different source coverage strongly limit comparisons. Source categories and time boundaries need review before policy use. These are descriptive workload summaries, not causal effects, official clearance rates, or population-adjusted risk measures. Records can change during pagination. BPD warns of reporting-system translation issues from May 2021; consult its documentation.

## Sources

- Inspiration: https://baltimore-psad.github.io/
- Crime: synthetic
- Arrests: synthetic
- BPD definitions and reporting caveat: https://www.baltimorepolice.org/crime-stats/open-data
