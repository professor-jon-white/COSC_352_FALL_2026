package model

import "time"

type CrimeRecord struct {
	When         time.Time
	Description  string
	Neighborhood string // normalized join key
}

type ServiceRequest struct {
	Created      time.Time
	Type         string
	Agency       string
	Neighborhood string // normalized join key
}
