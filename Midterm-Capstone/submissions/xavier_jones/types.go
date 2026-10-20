package main

import "encoding/json"

const crimeURL = "https://services1.arcgis.com/UWYHeuuJISiGmgXx/arcgis/rest/services/NIBRS_GroupA_Crime_Data/FeatureServer/0"
const arrestURL = "https://baltegis.baltimorecity.gov/mapping/rest/services/GeoSpatialized_Tables/Arrest/FeatureServer/0"
const inspiration = "https://baltimore-psad.github.io/"

type Row map[string]any
type Feature struct {
	Attributes Row `json:"attributes"`
}
type Page struct {
	Features []Feature       `json:"features"`
	More     bool            `json:"exceededTransferLimit"`
	Error    json.RawMessage `json:"error,omitempty"`
}
type Source struct {
	URL, DateField, IDField, Where string
	Rows, Available                int
	Truncated                      bool
}
type Manifest struct {
	Retrieved     string
	Start, End    string
	Limit         int
	Demo          bool
	Crime, Arrest Source
}
type Cell struct {
	Name                    string
	Crimes, Arrests, Linked int
}
type Incident struct {
	When         int64
	Neighborhood string
}
type Result struct {
	CategoryByNeighborhood, WeaponByNeighborhood                                                                              map[string]map[string]int
	Neighborhoods, Months, Categories, Weapons                                                                                map[string]*Cell
	CrimeRows, ArrestRows, UniqueIncidents, UniqueArrests, MatchedIncidents                                                   int
	MissingCrimeID, MissingArrestID, MissingCrimeNeighborhood, MissingArrestNeighborhood, MissingCrimeDate, MissingArrestDate int
	NegativeLag                                                                                                               int
	Lags                                                                                                                      []float64
}
