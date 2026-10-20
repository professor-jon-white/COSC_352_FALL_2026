# Baltimore Public Safety: Crime and Arrest Workload

Generated: 2026-10-15T02:55:58Z

Snapshot retrieved: 2026-10-15T02:52:32Z. Source query interval: 2025-01-01 inclusive to 2025-04-01 exclusive.

Crime rows: 12142 of 12142 available (truncated=false). Arrest rows: 4183 of 4183 available (truncated=false). Positive limits select a non-random source-ID prefix.

Inspired by the [Public Safety Accountability Dashboard](https://baltimore-psad.github.io/), using two raw Open Baltimore datasets. Independent class project; its definitions and totals do not reproduce the official dashboard.

**Coverage:** 12142 crime rows, 10648 distinct nonempty crime incident IDs; 4183 arrest rows, 4010 distinct nonempty arrest numbers. No person-level or case-level linkage is assumed.

**Geographic coverage:** 100.0% of crime rows and 31.1% of arrest rows have a neighborhood. Geographic comparisons describe only the geocoded portion; missing arrest neighborhoods can materially change the picture.

## 1. Which neighborhoods have the largest recorded crime workload, and how does arrest activity compare?

For citywide coordination: full outer join of neighborhood aggregates; top 15 by crime rows. The two location fields describe different activities. Counts are records, not unique people or population-adjusted risk.

| Group | Crime records | Arrest records |
|---|---:|---:|
| DOWNTOWN | 440 | 108 |
| FRANKFORD | 302 | 25 |
| BROOKLYN | 274 | 44 |
| BELAIR-EDISON | 268 | 23 |
| MOUNT VERNON | 245 | 26 |
| CANTON | 217 | 7 |
| HAMPDEN | 179 | 4 |
| CHERRY HILL | 171 | 11 |
| CANTON INDUSTRIAL AREA | 165 | 2 |
| SANDTOWN-WINCHESTER | 158 | 29 |
| PATTERSON PARK NEIGHBORHOOD | 141 | 8 |
| HAMILTON HILLS | 130 | 8 |
| UPTON | 125 | 26 |
| HIGHLANDTOWN | 123 | 6 |
| FELLS POINT | 121 | 13 |

## 2. How do monthly crime and arrest workloads change?

For staffing review: full outer join of America/New_York calendar-month aggregates. The source date filter uses ArcGIS SQL timestamps; month labels use returned Unix timestamps converted to Eastern time. A zero means no downloaded rows in that cell, not absence of real activity.

| Group | Crime records | Arrest records |
|---|---:|---:|
| 2024-12 | 33 | 0 |
| 2025-01 | 3930 | 1400 |
| 2025-02 | 3719 | 1328 |
| 2025-03 | 4460 | 1455 |

## 3. What offense mix is recorded in the neighborhoods with the most arrest activity?

For investigative resource planning: select the five known neighborhoods with the most arrest rows, join to crime by neighborhood, and compare crime Description shares with all other known neighborhoods. This describes contextual workload; arrests are not attributed to these offenses.

Selected neighborhoods: BROOKLYN, CARROLLTON RIDGE, CENTRAL PARK HEIGHTS, DOWNTOWN, SANDTOWN-WINCHESTER. Their arrest records: 234. Crime rows with known neighborhood: 1080 selected, 11059 elsewhere.

| Source category | Selected crime rows | Selected mix | Elsewhere crime rows | Elsewhere mix |
|---|---:|---:|---:|---:|
| COMMON ASSAULT | 236 | 21.9% | 1905 | 17.2% |
| VANDALISM | 175 | 16.2% | 1567 | 14.2% |
| LARCENY | 95 | 8.8% | 1122 | 10.1% |
| AUTO THEFT | 58 | 5.4% | 1035 | 9.4% |
| AGG. ASSAULT | 120 | 11.1% | 932 | 8.4% |
| SHOPLIFTING | 54 | 5.0% | 846 | 7.6% |
| LARCENY FROM AUTO | 68 | 6.3% | 758 | 6.9% |
| BURGLARY | 67 | 6.2% | 682 | 6.2% |
| ROBBERY | 71 | 6.6% | 508 | 4.6% |
| FRAUD | 34 | 3.1% | 513 | 4.6% |
| LARCENY OF MOTOR VEHICLE PARTS OR ACCESSORIES | 34 | 3.1% | 473 | 4.3% |
| INTIMIDATION | 22 | 2.0% | 307 | 2.8% |
| ROBBERY - COMMERCIAL | 12 | 1.1% | 107 | 1.0% |
| ROBBERY - CARJACKING | 7 | 0.6% | 82 | 0.7% |
| RAPE | 5 | 0.5% | 59 | 0.5% |

## 4. Where are crime-record and arrest-record shares most different?

For reviewing geographic workload distribution: compare each neighborhood's share of crime rows with its share of arrest rows, using known-neighborhood rows only. Sort by absolute percentage-point difference. Positive means crime-record share is larger. The gap is not a clearance rate, policing adequacy measure, or performance score.

Known-neighborhood denominators: 12139 crime rows; 1302 arrest rows.

| Neighborhood | Crime share | Arrest share | Difference (points) |
|---|---:|---:|---:|
| DOWNTOWN | 3.6% | 8.3% | -4.7 |
| CARROLLTON RIDGE | 0.8% | 2.1% | -1.3 |
| CANTON | 1.8% | 0.5% | +1.2 |
| CANTON INDUSTRIAL AREA | 1.4% | 0.2% | +1.2 |
| HAMPDEN | 1.5% | 0.3% | +1.2 |
| BROOKLYN | 2.3% | 3.4% | -1.1 |
| CENTRAL PARK HEIGHTS | 0.9% | 2.0% | -1.0 |
| SHIPLEY HILL | 0.3% | 1.3% | -1.0 |
| MONDAWMIN | 1.0% | 2.0% | -1.0 |
| WESTPORT | 0.3% | 1.3% | -1.0 |

```text
Share difference: + crime share larger; - arrest share larger
One # represents approximately one percentage point.
DOWNTOWN                   -4.7 -#####
CARROLLTON RIDGE           -1.3 -#
CANTON                     +1.2 +#
CANTON INDUSTRIAL AREA     +1.2 +#
HAMPDEN                    +1.2 +#
BROOKLYN                   -1.1 -#
CENTRAL PARK HEIGHTS       -1.0 -#
SHIPLEY HILL               -1.0 -#
MONDAWMIN                  -1.0 -#
WESTPORT                   -1.0 -#
```

## 5. What weapon profiles and arrest workload occur in neighborhoods with the most crime records?

For violence-prevention program planning: select the five known neighborhoods with the most crime rows, join their arrest totals by neighborhood, and compare source Weapon shares with other known neighborhoods. Weapon labels are not inferred or collapsed into a firearm classification. The geographic association does not link an arrest to a weapon event.

Selected neighborhoods: BELAIR-EDISON, BROOKLYN, DOWNTOWN, FRANKFORD, MOUNT VERNON. Their arrest records: 226. Crime rows with known neighborhood: 1529 selected, 10610 elsewhere.

| Source category | Selected crime rows | Selected mix | Elsewhere crime rows | Elsewhere mix |
|---|---:|---:|---:|---:|
| UNKNOWN | 958 | 62.7% | 6878 | 64.8% |
| PERSONAL_WEAPONS | 382 | 25.0% | 2380 | 22.4% |
| HANDGUN | 50 | 3.3% | 352 | 3.3% |
| KNIFE_CUTTING_INSTRUMENT | 42 | 2.7% | 254 | 2.4% |
| FIREARM | 16 | 1.0% | 224 | 2.1% |
| BLUNT_OBJECT | 27 | 1.8% | 209 | 2.0% |
| OTHER | 29 | 1.9% | 203 | 1.9% |
| ASPHYXIATION | 10 | 0.7% | 66 | 0.6% |
| MOTOR_VEHICLE_VESSEL | 12 | 0.8% | 28 | 0.3% |
| OTHER_FIREARM | 0 | 0.0% | 4 | 0.0% |
| RIFLE | 0 | 0.0% | 4 | 0.0% |
| AUTOMATIC_FIREARM | 1 | 0.1% | 2 | 0.0% |
| FIRE_INCENDIARY_DEVICE | 1 | 0.1% | 2 | 0.0% |
| EXPLOSIVES | 0 | 0.0% | 2 | 0.0% |
| SHOTGUN | 1 | 0.1% | 1 | 0.0% |

## Data quality and interpretation

| Check | Crime rows | Arrest rows |
|---|---:|---:|
| Missing incident ID | 0 | 3509 |
| Missing neighborhood | 3 | 2881 |
| Invalid/missing date | 0 | 0 |

Exact normalized ID overlaps found: 0. This is a diagnostic only: the sources use different identifier conventions and many arrest IDs are absent. All five analyses use aggregate neighborhood or time joins.

Neighborhood names are uppercased and whitespace-normalized. No spatial or fuzzy joins are inferred. UNKNOWN neighborhoods appear in overall workload tables but are excluded from geographic selections and share denominators. Crime rows may represent multiple offenses or victims, and arrest rows multiple charges; neither is a count of unique people. Arrest geography is not necessarily crime geography. Missing neighborhoods and different source coverage strongly limit comparisons. Source categories and time boundaries need review before policy use. These are descriptive workload summaries, not causal effects, official clearance rates, or population-adjusted risk measures. Records can change during pagination. BPD warns of reporting-system translation issues from May 2021; consult its documentation.

## Sources

- Inspiration: https://baltimore-psad.github.io/
- Crime: https://services1.arcgis.com/UWYHeuuJISiGmgXx/arcgis/rest/services/NIBRS_GroupA_Crime_Data/FeatureServer/0
- Arrests: https://baltegis.baltimorecity.gov/mapping/rest/services/GeoSpatialized_Tables/Arrest/FeatureServer/0
- BPD definitions and reporting caveat: https://www.baltimorepolice.org/crime-stats/open-data
