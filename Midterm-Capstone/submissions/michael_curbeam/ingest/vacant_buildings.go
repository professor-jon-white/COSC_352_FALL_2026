package ingest

import "context"

// DHCD "Vacant Building Notice - Open" layer (~11.5k rows).
const vacantBuildingsURL = "https://egisdata.baltimorecity.gov/egis/rest/services/Housing/DHCD_Open_Baltimore_Datasets/FeatureServer/1"

// FetchVacantBuildings downloads all open vacant building notices.
func FetchVacantBuildings(ctx context.Context) ([]VacantBuilding, error) {
	return fetchLayer[VacantBuilding](ctx, arcgisQuery{
		LayerURL: vacantBuildingsURL,
		Where:    "1=1",
		OrderBy:  "OBJECTID",
		PageSize: 1000,
	})
}
