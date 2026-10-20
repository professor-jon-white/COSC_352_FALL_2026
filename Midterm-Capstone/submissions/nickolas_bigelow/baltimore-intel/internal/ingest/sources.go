package ingest

import (
	"context"
	"fmt"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"baltimore-intel/internal/model"
)

// Source is one Open Baltimore dataset. Sources are created through the
// NewSource factory so the pipeline never depends on a concrete dataset type.
type Source interface {
	Name() string
	FileName() string // file expected inside the data directory
	Load(ctx context.Context, dir string, workers int, ds *model.Dataset) (Stats, error)
}

const (
	KindCrime   = "crime"
	Kind311     = "311"
	KindLiquor  = "liquor"
	KindVacants = "vacants"
)

// Kinds lists every dataset in load order.
var Kinds = []string{KindCrime, Kind311, KindLiquor, KindVacants}

// NewSource is the factory for dataset sources.
func NewSource(kind string) (Source, error) {
	switch kind {
	case KindCrime:
		return crimeSource{}, nil
	case Kind311:
		return requestSource{}, nil
	case KindLiquor:
		return liquorSource{}, nil
	case KindVacants:
		return vacantSource{}, nil
	}
	return nil, fmt.Errorf("unknown dataset %q (want one of %v)", kind, Kinds)
}

// FileFor returns the canonical raw file name for a dataset kind.
func FileFor(kind string) string {
	s, err := NewSource(kind)
	if err != nil {
		return ""
	}
	return s.FileName()
}

func need(h Header, name string, aliases ...string) (int, error) {
	if i := h.Col(aliases...); i >= 0 {
		return i, nil
	}
	return -1, fmt.Errorf("missing required column %s (looked for %v)", name, aliases)
}

// point resolves coordinates from Latitude/Longitude, X/Y or GeoLocation.
type pointCols struct{ lat, lon, geo int }

func newPointCols(h Header) pointCols {
	return pointCols{
		lat: h.Col("Latitude", "Lat", "Y", "_lat"),
		lon: h.Col("Longitude", "Lon", "Long", "X", "_lon"),
		geo: h.Col("GeoLocation", "Location_1", "Location 1"),
	}
}

func (p pointCols) get(row []string) (float32, float32, bool) {
	lat, ok1 := ParseCoord(Get(row, p.lat))
	lon, ok2 := ParseCoord(Get(row, p.lon))
	if !(ok1 && ok2 && model.InBaltimore(lat, lon)) {
		var ok bool
		if lat, lon, ok = ParseGeoLocation(Get(row, p.geo)); !ok || !model.InBaltimore(lat, lon) {
			return 0, 0, false
		}
	}
	return float32(lat), float32(lon), true
}

// ---- Part 1 Crime --------------------------------------------------------

type crimeSource struct{}

func (crimeSource) Name() string     { return "Part 1 Crime" }
func (crimeSource) FileName() string { return "part1_crime.csv" }

// violentTerms identifies violent offences in the Description column.
var violentTerms = []string{"HOMICIDE", "SHOOTING", "ASSAULT", "ROBBERY", "RAPE", "CARJACK"}

func IsViolent(desc string) bool {
	d := strings.ToUpper(desc)
	for _, t := range violentTerms {
		if strings.Contains(d, t) {
			return true
		}
	}
	return false
}

func (s crimeSource) Load(ctx context.Context, dir string, workers int, ds *model.Dataset) (Stats, error) {
	recs, st, err := ParseCSV(ctx, s.Name(), filepath.Join(dir, s.FileName()), workers,
		func(h Header) (RowFunc[model.Crime], error) {
			tCol, err := need(h, "CrimeDateTime", "CrimeDateTime", "CrimeDate")
			if err != nil {
				return nil, err
			}
			dCol, err := need(h, "Description", "Description")
			if err != nil {
				return nil, err
			}
			timeCol := h.Col("CrimeTime") // legacy layout splits date and time
			hood := h.Col("Neighborhood")
			inc := h.Col("Total_Incidents", "Total Incidents")
			pts := newPointCols(h)
			return func(row []string) (model.Crime, bool, string) {
				ts := Get(row, tCol)
				if timeCol >= 0 && !strings.Contains(ts, ":") {
					ts = strings.TrimSpace(ts + " " + Get(row, timeCol))
				}
				t, ok := ParseTime(ts)
				if !ok {
					return model.Crime{}, false, "bad_date"
				}
				if ds.Year > 0 && YearOf(t) != ds.Year {
					return model.Crime{}, false, "other_year"
				}
				desc := Get(row, dCol)
				lat, lon, _ := pts.get(row) // records without coords still count per neighborhood
				n, err := strconv.Atoi(Get(row, inc))
				if err != nil || n < 1 {
					n = 1
				}
				return model.Crime{
					Time: t, Lat: lat, Lon: lon,
					Hood:      ds.Hoods.ID(Get(row, hood)),
					Desc:      ds.CrimeTypes.ID(desc),
					Incidents: uint16(min(n, 65535)),
					Violent:   IsViolent(desc),
				}, true, ""
			}, nil
		})
	ds.Crimes = recs
	return st, err
}

// ---- 311 Customer Service Requests --------------------------------------

type requestSource struct{}

func (requestSource) Name() string     { return "311 Service Requests" }
func (requestSource) FileName() string { return "service_requests_311.csv" }

var sanitationTerms = []string{"RAT", "RODENT", "DUMP", "DIRTY", "TRASH", "SANITATION",
	"CLEANING", "BULK", "DEBRIS", "GARBAGE", "LITTER", "ALLEY", "HIGH GRASS", "WEEDS"}

func IsSanitation(srType string) bool {
	t := strings.ToUpper(srType)
	for _, s := range sanitationTerms {
		if strings.Contains(t, s) {
			return true
		}
	}
	return false
}

func (s requestSource) Load(ctx context.Context, dir string, workers int, ds *model.Dataset) (Stats, error) {
	recs, st, err := ParseCSV(ctx, s.Name(), filepath.Join(dir, s.FileName()), workers,
		func(h Header) (RowFunc[model.ServiceRequest], error) {
			cCol, err := need(h, "CreatedDate", "CreatedDate", "Created Date")
			if err != nil {
				return nil, err
			}
			tCol, err := need(h, "SRType", "SRType", "SR Type")
			if err != nil {
				return nil, err
			}
			closeCol := h.Col("CloseDate", "Close Date")
			dueCol := h.Col("DueDate", "Due Date")
			agCol := h.Col("Agency")
			hood := h.Col("Neighborhood")
			pts := newPointCols(h)
			return func(row []string) (model.ServiceRequest, bool, string) {
				c, ok := ParseTime(Get(row, cCol))
				if !ok {
					return model.ServiceRequest{}, false, "bad_date"
				}
				if ds.Year > 0 && YearOf(c) != ds.Year {
					return model.ServiceRequest{}, false, "other_year"
				}
				typ := Get(row, tCol)
				closed, _ := ParseTime(Get(row, closeCol))
				if closed != 0 && closed < c {
					closed = 0 // data-entry error; treat as open
				}
				due, _ := ParseTime(Get(row, dueCol))
				lat, lon, _ := pts.get(row)
				return model.ServiceRequest{
					Created: c, Closed: closed, Due: due, Lat: lat, Lon: lon,
					Hood:       ds.Hoods.ID(Get(row, hood)),
					Type:       ds.SRTypes.ID(typ),
					Agency:     ds.Agencies.ID(Get(row, agCol)),
					Sanitation: IsSanitation(typ),
				}, true, ""
			}, nil
		})
	ds.Requests = recs
	return st, err
}

// ---- Liquor Licenses ----------------------------------------------------

type liquorSource struct{}

func (liquorSource) Name() string     { return "Liquor Licenses" }
func (liquorSource) FileName() string { return "liquor_licenses.csv" }

type liquorRow struct {
	model.LiquorOutlet
	number string
}

func (s liquorSource) Load(ctx context.Context, dir string, workers int, ds *model.Dataset) (Stats, error) {
	recs, st, err := ParseCSV(ctx, s.Name(), filepath.Join(dir, s.FileName()), workers,
		func(h Header) (RowFunc[liquorRow], error) {
			numCol, err := need(h, "LicenseNumber", "LicenseNumber", "License Number")
			if err != nil {
				return nil, err
			}
			name := h.Col("TradeName", "Trade Name")
			corp := h.Col("CorpName")
			addr := h.Col("AddrStreet", "Address")
			zip := h.Col("AddrZip", "Zip")
			class := h.Col("LicenseClass", "License Class")
			kind := h.Col("EstablishmentDesc", "Description")
			year := h.Col("LicenseYear")
			status := h.Col("LicenseStatus")
			pts := newPointCols(h)
			return func(row []string) (liquorRow, bool, string) {
				st := strings.ToUpper(Get(row, status))
				if strings.Contains(st, "CANCEL") || strings.Contains(st, "REVOK") || strings.Contains(st, "DENI") {
					return liquorRow{}, false, "inactive"
				}
				lat, lon, ok := pts.get(row)
				if !ok {
					return liquorRow{}, false, "no_location" // outlet joins are spatial
				}
				n := Get(row, name)
				if n == "" {
					n = Get(row, corp)
				}
				y, _ := strconv.Atoi(Get(row, year))
				return liquorRow{number: Get(row, numCol), LiquorOutlet: model.LiquorOutlet{
					Name: n, Address: Get(row, addr), Zip: Get(row, zip),
					Class: Get(row, class), Kind: Get(row, kind),
					Lat: lat, Lon: lon, Year: y,
				}}, true, ""
			}, nil
		})
	if err != nil {
		return st, err
	}
	// The licenses table has one row per license per year; keep each
	// establishment's latest record so an outlet is counted once.
	latest := make(map[string]model.LiquorOutlet, len(recs))
	for _, r := range recs {
		key := r.number
		if key == "" {
			key = fmt.Sprintf("%.5f,%.5f,%s", r.Lat, r.Lon, r.Name)
		}
		if cur, ok := latest[key]; !ok || r.Year > cur.Year {
			latest[key] = r.LiquorOutlet
		}
	}
	keys := make([]string, 0, len(latest))
	for k := range latest {
		keys = append(keys, k)
	}
	sort.Strings(keys) // deterministic order -> reproducible reports
	ds.Outlets = make([]model.LiquorOutlet, 0, len(keys))
	for _, k := range keys {
		ds.Outlets = append(ds.Outlets, latest[k])
	}
	st.Skipped["duplicate_year"] = st.Kept - int64(len(ds.Outlets))
	st.Kept = int64(len(ds.Outlets))
	return st, nil
}

// ---- Vacant Building Notices -------------------------------------------

type vacantSource struct{}

func (vacantSource) Name() string     { return "Vacant Building Notices" }
func (vacantSource) FileName() string { return "vacant_building_notices.csv" }

func (s vacantSource) Load(ctx context.Context, dir string, workers int, ds *model.Dataset) (Stats, error) {
	recs, st, err := ParseCSV(ctx, s.Name(), filepath.Join(dir, s.FileName()), workers,
		func(h Header) (RowFunc[model.VacantNotice], error) {
			hood, err := need(h, "Neighborhood", "Neighborhood")
			if err != nil {
				return nil, err
			}
			date := h.Col("DateNotice", "Date Notice")
			cancel := h.Col("DateCancel")
			abate := h.Col("DateAbate")
			typ := h.Col("HousingMarketTypology2023", "HousingMarketTypology2017", "HMT")
			pts := newPointCols(h)
			return func(row []string) (model.VacantNotice, bool, string) {
				if Get(row, cancel) != "" || Get(row, abate) != "" {
					return model.VacantNotice{}, false, "closed_notice"
				}
				id := ds.Hoods.ID(Get(row, hood))
				if id == 0 {
					return model.VacantNotice{}, false, "no_neighborhood"
				}
				d, _ := ParseTime(Get(row, date))
				lat, lon, _ := pts.get(row)
				return model.VacantNotice{Date: d, Lat: lat, Lon: lon, Hood: id,
					Typology: ds.Typologies.ID(Get(row, typ))}, true, ""
			}, nil
		})
	ds.Vacants = recs
	return st, err
}
