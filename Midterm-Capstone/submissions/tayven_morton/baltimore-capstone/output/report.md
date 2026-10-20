# Baltimore Neighborhood Service and Vacancy Analyzer

311 period: October 1 through December 31, 2025 (UTC). Vacancy data is a later open-notice snapshot; see data/manifest.json for extraction time.

Accepted requests: 168569. Distinct vacancy properties with neighborhood: 11481.

Important coverage warning: 75051 of 243620 raw requests (30.81%) lack a neighborhood and are excluded rather than assigned a guessed location. This can bias comparisons.

Sources: [Baltimore 311 requests](https://services1.arcgis.com/UWYHeuuJISiGmgXx/arcgis/rest/services/311_Customer_Service_Requests_2025/FeatureServer/0) and [Open vacant-building notices](https://egisdata.baltimorecity.gov/egis/rest/services/Housing/DHCD_Open_Baltimore_Datasets/FeatureServer/1). All findings below are computed from the bundled extracts of these sources.

Analysis universe: 213 neighborhoods appearing in both datasets; 66 dataset-only neighborhoods excluded from comparisons. Zero notice rows do not establish zero real-world vacancy.

## Q1: Where are both request volume and vacancy counts concentrated?

The following neighborhoods are at or above the 75th-percentile midrank on both measures. This is a descriptive count comparison, not a population-adjusted rate.

| Neighborhood | Requests | Vacancy properties |
|---|---:|---:|
| BROADWAY EAST | 2192 | 719 |
| CARROLLTON RIDGE | 2175 | 743 |
| SANDTOWN-WINCHESTER | 2170 | 587 |
| CENTRAL PARK HEIGHTS | 2191 | 316 |
| HARLEM PARK | 1697 | 439 |
| COLDSTREAM HOMESTEAD MONTEBELLO | 2758 | 234 |
| OLIVER | 1659 | 329 |
| BROOKLYN | 2547 | 143 |
| EAST BALTIMORE MIDWAY | 1502 | 314 |
| BELAIR-EDISON | 7237 | 75 |
| BEREA | 1604 | 133 |
| UPTON | 1350 | 251 |
| MCELDERRY PARK | 1677 | 119 |
| WASHINGTON VILLAGE/PIGTOWN | 2285 | 89 |
| PENROSE/FAYETTE STREET OUTREACH | 986 | 317 |
| DOWNTOWN | 3639 | 50 |
| MOUNT CLARE | 939 | 289 |
| PARK CIRCLE | 1013 | 168 |
| CURTIS BAY | 1326 | 121 |
| FRANKLIN SQUARE | 973 | 235 |
| MONDAWMIN | 895 | 277 |
| ALLENDALE | 1517 | 70 |
| ELLWOOD PARK/MONUMENT | 999 | 101 |
| ARLINGTON | 982 | 94 |
| CHERRY HILL | 1206 | 51 |

## Q2: How does the mix of service needs differ?

Low-count cohort: 75 neighborhoods, 30619 requests. High-count cohort: 71 neighborhoods, 79202 requests. Cohorts use lower/upper vacancy-count thirds; ties stay together, so group sizes can differ.

Snapshot cutoffs: low = at most 5 vacancy properties; high = more than 26 vacancy properties. Both cohorts are restricted to matched neighborhoods.

Top five request types in each cohort (share of all requests in that cohort):

### Low vacancy-count cohort

| Type | Requests | Share |
|---|---:|---:|
| SW-RAT RUBOUT PROACTIVE | 5534 | 18.07% |
| TRS-PARKING COMPLAINT | 3243 | 10.59% |
| ECC-VEHICLE LOOK UP | 1359 | 4.44% |
| SW-BULK SCHEDULED-WEEKDAY | 1011 | 3.30% |
| SW-RECYCLING | 780 | 2.55% |

### High vacancy-count cohort

| Type | Requests | Share |
|---|---:|---:|
| SW-RAT RUBOUT PROACTIVE | 16623 | 20.99% |
| TRS-PARKING COMPLAINT | 5437 | 6.86% |
| SW-DIRTY ALLEY | 4155 | 5.25% |
| HCD-SANITATION PROPERTY | 3387 | 4.28% |
| SW-HGW | 3152 | 3.98% |

## Q3: Does recorded closure time differ for comparable requests?

Only requests with a status beginning CLOSED, non-null dates and nonnegative duration qualify. Transferred closures are included, so this is NOT time to successful resolution. Open cases are excluded and may bias these medians downward. Comparison requires at least 20 qualifying closures per type in EACH cohort.

| Request type | Low n | Low median days | High n | High median days | High minus low days |
|---|---:|---:|---:|---:|---:|
| SW-RAT RUBOUT PROACTIVE | 5439 | 0.00 | 16354 | 0.00 | +0.00 |
| TRS-PARKING COMPLAINT | 3238 | 9.81 | 5434 | 22.16 | +12.35 |
| SW-DIRTY ALLEY | 397 | 2.77 | 4154 | 2.26 | -0.52 |
| HCD-SANITATION PROPERTY | 737 | 4.29 | 3386 | 3.72 | -0.57 |
| SW-HGW | 66 | 8.29 | 3151 | 7.80 | -0.49 |
| ECC-VEHICLE LOOK UP | 1359 | 0.00 | 1544 | 0.00 | +0.00 |
| SW-CITY TRASH CAN OR RECYCLING CART CONCERN | 613 | 14.09 | 2143 | 13.68 | -0.41 |
| SW-RECYCLING | 777 | 0.80 | 1948 | 0.79 | -0.00 |
| SW-DIRTY STREET | 518 | 2.22 | 2151 | 2.25 | +0.03 |
| SW-MIXED REFUSE | 752 | 0.77 | 1895 | 0.78 | +0.02 |

Up to ten most frequent eligible types are shown. No statistical-significance or causal claim is made.

## Q4: How do monthly request volumes change by vacancy cohort?

Totals describe cohort workload; requests per neighborhood adjust for differing group sizes, not population.

| Month | Low total | Low per neighborhood | High total | High per neighborhood |
|---|---:|---:|---:|---:|
| 2025-10 | 11317 | 150.9 | 30032 | 423.0 |
| 2025-11 | 9599 | 128.0 | 23454 | 330.3 |
| 2025-12 | 9703 | 129.4 | 25716 | 362.2 |

```text
2025-10 Low  ###########                    11317
2025-10 High ############################## 30032
2025-11 Low  ##########                     9599
2025-11 High #######################        23454
2025-12 Low  ##########                     9703
2025-12 High ##########################     25716
```

Low cohort October-to-December change: -14.26%.

High cohort October-to-December change: -14.37%.

## Q5: Which neighborhoods should staff review first?

Screening score = 50% vacancy-count percentile + 50% request-count percentile, among matched neighborhoods. Equal values receive equal midrank percentiles; alphabetic order breaks display ties. The score is a student-designed review aid, not an official funding formula.

| Neighborhood | Score / 100 | Requests | Vacancy properties | Top request type |
|---|---:|---:|---:|---|
| BROADWAY EAST | 97.89 | 2192 | 719 | SW-HGW |
| CARROLLTON RIDGE | 97.65 | 2175 | 743 | SW-CLEANING |
| SANDTOWN-WINCHESTER | 96.95 | 2170 | 587 | SW-RAT RUBOUT PROACTIVE |
| CENTRAL PARK HEIGHTS | 96.24 | 2191 | 316 | SW-RAT RUBOUT PROACTIVE |
| HARLEM PARK | 96.01 | 1697 | 439 | SW-RAT RUBOUT PROACTIVE |

Suggested action: review the listed neighborhoods jointly with housing and the agencies responsible for their dominant request categories. Validate conditions on the ground before assigning resources.

## Data quality and limits

311: raw=243620; accepted=168569; malformed=0; missing ID=0; duplicate ID/property=0; missing neighborhood=75051; missing creation date=0; outside period=0; deleted=0; present but unusable closure=3040.

Vacancies: raw=11481; accepted=11481; malformed=0; missing ID=0; duplicate ID/property=0; missing neighborhood=0; missing creation date=0; outside period=0; deleted=0; present but unusable closure=0.

Exclusion counts are mutually exclusive first-failure counts, except unusable closure (an accepted-record flag). Vacancy deduplication uses parcel ID with address fallback; requests use service-request number. Missing fields, reporting behavior, property size, population and neighborhood spelling can affect comparisons.

These data cannot establish causation. The later vacancy snapshot is not a reconstruction of 2025 vacancies. No population or total-property denominator is available in these two inputs. Current closure status may reflect updates after the request period. Dataset-only neighborhoods are excluded rather than assumed to have zero demand or vacancy.

Requests can include proactive agency activity, not just resident complaints. Their volume is a workload indicator, not a direct count of unmet needs.

Dataset-only neighborhoods:
- ARMISTEAD GARDENS (336 requests; 0 vacancy properties)
- BALTIMORE PENINSULA (407 requests; 0 vacancy properties)
- BARRE CIRCLE (133 requests; 0 vacancy properties)
- BELLONA-GITTINGS (118 requests; 0 vacancy properties)
- BEVERLY HILLS (358 requests; 0 vacancy properties)
- BLYTHEWOOD (11 requests; 0 vacancy properties)
- CANTON INDUSTRIAL AREA (247 requests; 0 vacancy properties)
- CARROLL PARK (110 requests; 0 vacancy properties)
- CHINQUAPIN PARK (404 requests; 0 vacancy properties)
- COLDSPRING (245 requests; 0 vacancy properties)
- CROSS KEYS (180 requests; 0 vacancy properties)
- CURTIS BAY INDUSTRIAL AREA (18 requests; 0 vacancy properties)
- DICKEYVILLE (55 requests; 0 vacancy properties)
- DOWNTOWN WEST (118 requests; 0 vacancy properties)
- DRUID HILL PARK (176 requests; 0 vacancy properties)
- DUNDALK MARINE TERMINAL (2 requests; 0 vacancy properties)
- EASTWOOD (329 requests; 0 vacancy properties)
- EVESHAM PARK (133 requests; 0 vacancy properties)
- FOREST PARK GOLF COURSE (42 requests; 0 vacancy properties)
- GREENMOUNT CEMETERY (21 requests; 0 vacancy properties)
- GWYNNS FALLS/LEAKIN PARK (103 requests; 0 vacancy properties)
- HARBOR POINT (38 requests; 0 vacancy properties)
- HAWKINS POINT (50 requests; 0 vacancy properties)
- HERITAGE CROSSING (225 requests; 0 vacancy properties)
- HERRING RUN PARK (30 requests; 0 vacancy properties)
- HOES HEIGHTS (298 requests; 0 vacancy properties)
- HOLABIRD INDUSTRIAL PARK (77 requests; 0 vacancy properties)
- HOPKINS BAYVIEW (120 requests; 0 vacancy properties)
- INNER HARBOR (248 requests; 0 vacancy properties)
- JOHNS HOPKINS HOMEWOOD (114 requests; 0 vacancy properties)
- KEY HIGHWAY (31 requests; 0 vacancy properties)
- LAKE EVESHAM (100 requests; 0 vacancy properties)
- LOCUST POINT INDUSTRIAL AREA (58 requests; 0 vacancy properties)
- LOWER HERRING RUN PARK (8 requests; 0 vacancy properties)
- LOYOLA/NOTRE DAME (84 requests; 0 vacancy properties)
- MEDFIELD (711 requests; 0 vacancy properties)
- MIDDLE BRANCH/REEDBIRD PARKS (73 requests; 0 vacancy properties)
- MONTEBELLO (86 requests; 0 vacancy properties)
- MORGAN PARK (53 requests; 0 vacancy properties)
- MORGAN STATE UNIVERSITY (49 requests; 0 vacancy properties)
- MT PLEASANT PARK (33 requests; 0 vacancy properties)
- NORTH ROLAND PARK/POPLAR HILL (226 requests; 0 vacancy properties)
- O'DONNELL HEIGHTS (95 requests; 0 vacancy properties)
- OAKLEE (109 requests; 0 vacancy properties)
- ORCHARD RIDGE (95 requests; 0 vacancy properties)
- ORIGINAL NORTHWOOD (182 requests; 0 vacancy properties)
- OTTERBEIN (321 requests; 0 vacancy properties)
- OVERLEA (142 requests; 0 vacancy properties)
- PATTERSON PARK (148 requests; 0 vacancy properties)
- PENN-FALLSWAY (88 requests; 0 vacancy properties)
- PERKINS (40 requests; 0 vacancy properties)
- PERRING LOCH (760 requests; 0 vacancy properties)
- PURNELL (67 requests; 0 vacancy properties)
- SABINA-MATTFELDT (163 requests; 0 vacancy properties)
- SETON BUSINESS PARK (71 requests; 0 vacancy properties)
- SPRING GARDEN INDUSTRIAL AREA (10 requests; 0 vacancy properties)
- STADIUM/ENTERTAINMENT AREA (116 requests; 0 vacancy properties)
- TAYLOR HEIGHTS (60 requests; 0 vacancy properties)
- THE ORCHARDS (105 requests; 0 vacancy properties)
- TUSCANY-CANTERBURY (201 requests; 0 vacancy properties)
- UPLANDS (388 requests; 0 vacancy properties)
- VILLAGES OF HOMELAND (10 requests; 0 vacancy properties)
- WAKEFIELD (91 requests; 0 vacancy properties)
- WOODBOURNE HEIGHTS (211 requests; 0 vacancy properties)
- WYNDHURST (183 requests; 0 vacancy properties)
- YORK-HOMELAND (39 requests; 0 vacancy properties)
