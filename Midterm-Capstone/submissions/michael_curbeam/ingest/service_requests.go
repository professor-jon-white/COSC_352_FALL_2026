package ingest

import (
	"context"
	"strings"
)

// Open Baltimore "311 Customer Service Requests" (current year) layer.
const serviceRequestsURL = "https://services1.arcgis.com/UWYHeuuJISiGmgXx/ArcGIS/rest/services/311_Customer_Service_Requests_current/FeatureServer/0"

// maxServiceRequests caps the download: the full layer has ~900k rows.
// Only the newest records that carry a neighborhood are fetched.
const maxServiceRequests = 20000

// FetchServiceRequests downloads recent 311 requests that have a neighborhood.
func FetchServiceRequests(ctx context.Context) ([]ServiceRequest, error) {
	out, err := fetchLayer[ServiceRequest](ctx, arcgisQuery{
		LayerURL:   serviceRequestsURL,
		Where:      "Neighborhood IS NOT NULL",
		OrderBy:    "CreatedDate DESC",
		PageSize:   2000,
		MaxRecords: maxServiceRequests,
	})
	if err != nil {
		return nil, err
	}
	for i := range out {
		out[i].Agency = strings.TrimSpace(out[i].Agency) // source pads to fixed width
	}
	return out, nil
}
