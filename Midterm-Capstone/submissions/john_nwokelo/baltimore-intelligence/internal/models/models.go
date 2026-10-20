package models

type Point [2]float64 // longitude, latitude

type Neighborhood struct {
	Name                   string
	Polygons               [][][]Point // polygons -> rings -> points; outer ring first, then holes
	MinX, MaxX, MinY, MaxY float64
}

type Request struct {
	ID, Category, Status, Neighborhood string
	Lon, Lat                           float64
	Created, Closed                    int64 // Unix milliseconds; 0 represents unknown
}

type Joined struct {
	Request      Request
	Neighborhood string
}
