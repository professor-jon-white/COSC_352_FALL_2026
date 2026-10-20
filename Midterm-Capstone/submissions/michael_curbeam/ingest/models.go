package ingest

// ServiceRequest is one Baltimore 311 record (Open Baltimore
// "311 Customer Service Requests" ArcGIS layer). Dates are epoch milliseconds;
// null values decode to the zero value.
type ServiceRequest struct {
	RowID             int64  `json:"RowID"`
	ServiceRequestNum string `json:"ServiceRequestNum"`
	SRType            string `json:"SRType"`
	MethodReceived    string `json:"MethodReceived"`
	CreatedDate       int64  `json:"CreatedDate"`
	SRStatus          string `json:"SRStatus"`
	StatusDate        int64  `json:"StatusDate"`
	DueDate           int64  `json:"DueDate"`
	CloseDate         int64  `json:"CloseDate"`
	Agency            string `json:"Agency"`
	Outcome           string `json:"Outcome"`
	Address           string `json:"Address"`
	ZipCode           string `json:"ZipCode"`
	Neighborhood      string `json:"Neighborhood"`
	CouncilDistrict   string `json:"CouncilDistrict"`
	PoliceDistrict    string `json:"PoliceDistrict"`
	Latitude          string `json:"Latitude"`
	Longitude         string `json:"Longitude"`
}

// VacantBuilding is one open Vacant Building Notice (DHCD "Vacant Building
// Notice - Open" ArcGIS layer).
type VacantBuilding struct {
	ObjectID     int64  `json:"OBJECTID"`
	NoticeNum    string `json:"NoticeNum"`
	DateNotice   int64  `json:"DateNotice"`
	DateCancel   int64  `json:"DateCancel"`
	DateAbate    int64  `json:"DateAbate"`
	NoticeType   string `json:"NT"`
	OwnerAbbr    string `json:"OWNER_ABBR"`
	MarketTypo   string `json:"HousingMarketTypology2023"`
	CouncilDist  int    `json:"Council_District"`
	Neighborhood string `json:"Neighborhood"`
	BlockLot     string `json:"BLOCKLOT"`
	Address      string `json:"Address"`
}

// Datasets bundles the raw results of the ingest stage.
type Datasets struct {
	ServiceRequests []ServiceRequest
	VacantBuildings []VacantBuilding
}
