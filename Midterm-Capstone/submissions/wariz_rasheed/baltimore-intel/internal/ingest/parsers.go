package ingest

import (
	"strings"
	"time"
	_ "time/tzdata" // embed the tz database so America/New_York works on any machine

	"baltimoreintel/internal/geo"
	"baltimoreintel/internal/model"
)

// Timestamp layouts used by the portal exports.
const (
	crimeLayout = "1/2/2006 3:04:05 PM"      // e.g. "10/11/2026 5:00:00 AM" (UTC)
	srLayout    = "2006/01/02 15:04:05-07"   // e.g. "2026/01/03 14:29:00+00"
)

var eastern = loadEastern()

func loadEastern() *time.Location {
	loc, err := time.LoadLocation("America/New_York")
	if err != nil {
		return time.FixedZone("EST", -5*3600)
	}
	return loc
}

func parseSRTime(s string) (time.Time, bool) {
	if s == "" {
		return time.Time{}, false
	}
	t, err := time.Parse(srLayout, s)
	if err != nil {
		return time.Time{}, false
	}
	return t.UTC(), true
}

func isViolent(desc string) bool {
	return desc == "HOMICIDE" || desc == "AGG. ASSAULT" || desc == "RAPE" ||
		strings.HasPrefix(desc, "ROBBERY")
}

// CrimeParser builds the parser for the NIBRS Group A crime export.
//
// Timestamps in this export are UTC. Records whose local time is exactly
// midnight are treated as "time unknown" (the source appears to use midnight
// as a default), so they never count toward the night-time analysis.
func CrimeParser(h Header) (RowParser[model.Crime], error) {
	x := &indexer{h: h}
	iWhen := x.get("CrimeDateTime")
	iDesc := x.get("Description")
	iHood := x.get("Neighborhood")
	iLoc := x.get("Location")
	iShoot := x.get("Shooting")
	if x.err != nil {
		return nil, x.err
	}
	return func(rec []string, in *Interner) (model.Crime, bool) {
		var c model.Crime
		t, err := time.ParseInLocation(crimeLayout, field(rec, iWhen), time.UTC)
		if err != nil {
			return c, false
		}
		desc := strings.ToUpper(field(rec, iDesc))
		c.When = t
		c.Offense = in.Intern(desc)
		c.Violent = isViolent(desc)
		c.Shooting = field(rec, iShoot) == "Y"
		local := t.In(eastern)
		if local.Hour() == 0 && local.Minute() == 0 && local.Second() == 0 {
			c.LocalHour = -1
		} else {
			c.LocalHour = local.Hour()
		}
		c.Hood = in.Intern(geo.CanonHood(field(rec, iHood)))
		c.Block = in.Intern(geo.BlockKey(field(rec, iLoc)))
		return c, true
	}, nil
}

// RequestParser builds the parser for the 311 Customer Service Requests export.
func RequestParser(h Header) (RowParser[model.Request], error) {
	x := &indexer{h: h}
	iCreated := x.get("CreatedDate")
	iClosed := x.get("CloseDate")
	iType := x.get("SRType")
	iStatus := x.get("SRStatus")
	iAddr := x.get("Address")
	iHood := x.get("Neighborhood")
	if x.err != nil {
		return nil, x.err
	}
	iDeleted := -1
	if i, ok := h["IsDeleted"]; ok {
		iDeleted = i
	}
	return func(rec []string, in *Interner) (model.Request, bool) {
		var r model.Request
		if iDeleted >= 0 && field(rec, iDeleted) == "1" {
			return r, false
		}
		created, ok := parseSRTime(field(rec, iCreated))
		if !ok {
			return r, false
		}
		typ := field(rec, iType)
		if typ == "" {
			return r, false
		}
		prefix := typ
		if i := strings.IndexByte(typ, '-'); i > 0 {
			prefix = typ[:i]
		}
		r.Created = created
		r.Type = in.Intern(typ)
		r.Prefix = in.Intern(prefix)
		closed, hasClosed := parseSRTime(field(rec, iClosed))
		if hasClosed {
			r.Closed = closed
		}
		r.Completed = hasClosed && !closed.Before(created) && field(rec, iStatus) == "Closed"
		r.Hood = in.Intern(geo.CanonHood(field(rec, iHood)))
		r.Block = in.Intern(geo.BlockKey(field(rec, iAddr)))
		return r, true
	}, nil
}

// LicenseParser builds the parser for the liquor-license export.
func LicenseParser(h Header) (RowParser[model.License], error) {
	x := &indexer{h: h}
	iKey := x.get("LLKey")
	iStart := x.get("LicenseDate")
	iEnd := x.get("LicenseEndDate")
	iStatus := x.get("LicenseStatus")
	iKind := x.get("EstablishmentDesc")
	iAddr := x.get("AddrStreet")
	if x.err != nil {
		return nil, x.err
	}
	return func(rec []string, in *Interner) (model.License, bool) {
		var l model.License
		key := field(rec, iKey)
		if key == "" {
			return l, false
		}
		start, ok1 := parseSRTime(field(rec, iStart))
		end, ok2 := parseSRTime(field(rec, iEnd))
		if !ok1 || !ok2 {
			return l, false
		}
		kind := strings.ToUpper(field(rec, iKind))
		kind = strings.TrimSuffix(kind, " LICENSE")
		kind = strings.Join(strings.Fields(kind), " ")
		if kind == "" {
			kind = "UNSPECIFIED"
		}
		l.Key = key
		l.Start = start
		l.End = end
		l.Kind = in.Intern(kind)
		l.Active = !strings.EqualFold(field(rec, iStatus), "Closed")
		l.Block = in.Intern(geo.BlockKey(field(rec, iAddr)))
		return l, true
	}, nil
}
