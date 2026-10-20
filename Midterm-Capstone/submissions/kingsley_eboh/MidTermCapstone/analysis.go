package main

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"unicode"
)

type NeighborhoodSummary struct {
	Name                 string  `json:"neighborhood"`
	Requests             int     `json:"requests_2026"`
	Unresolved           int     `json:"nonclosed_311_requests"`
	RehabSites           int     `json:"vacant_building_rehab_locations"`
	RequestsPer100Rehabs float64 `json:"requests_per_100_rehab_sites"`
	UnresolvedRate       float64 `json:"nonclosed_rate"`
	JointIndex           float64 `json:"joint_index"`
}
type TypeSummary struct {
	Name       string  `json:"request_type"`
	Count      int     `json:"count"`
	Share      float64 `json:"share"`
	Enrichment float64 `json:"high_rehab_area_enrichment"`
}
type Analysis struct {
	GeneratedAt                 string                `json:"generated_at"`
	Elapsed                     string                `json:"elapsed"`
	Workers                     int                   `json:"workers"`
	OfflineExample              bool                  `json:"offline_example"`
	RawReplay                   bool                  `json:"replayed_saved_raw_data"`
	RequestCount                int                   `json:"request_count"`
	RehabCount                  int                   `json:"vacant_building_rehab_count"`
	UnresolvedCount             int                   `json:"nonclosed_311_request_count"`
	MatchedNeighborhoods        int                   `json:"matched_neighborhoods"`
	RequestNeighborhoodCoverage int                   `json:"request_records_with_neighborhood"`
	RehabNeighborhoodCoverage   int                   `json:"vacant_records_with_neighborhood"`
	Spearman                    float64               `json:"spearman_rank_correlation"`
	SpearmanLabel               string                `json:"spearman_interpretation"`
	HighRehabCutoff             int                   `json:"high_rehab_sites_cutoff"`
	Neighborhoods               []NeighborhoodSummary `json:"neighborhoods"`
	TopRequestTypes             []TypeSummary         `json:"top_request_types"`
	EnrichedTypes               []TypeSummary         `json:"high_rehab_area_request_types"`
}
type neighborhoodWork struct {
	Name                         string
	Requests, Unresolved, Vacant int
	Types                        map[string]int
}
type accumulator struct {
	areas                                                            map[string]*neighborhoodWork
	requestCount, vacantCount, unresolved, requestKnown, vacantKnown int
	allTypes                                                         map[string]int
}

func newAccumulator() *accumulator {
	return &accumulator{areas: map[string]*neighborhoodWork{}, allTypes: map[string]int{}}
}
func (a *accumulator) area(name string) *neighborhoodWork {
	key := normalizeNeighborhood(name)
	if key == "" {
		return nil
	}
	if a.areas[key] == nil {
		a.areas[key] = &neighborhoodWork{Name: name, Types: map[string]int{}}
	}
	return a.areas[key]
}
func (a *accumulator) add(r record) {
	area := a.area(r.Neighborhood)
	if r.Source == "requests" {
		a.requestCount++
		if isUnresolved(r.Status) {
			a.unresolved++
		}
		if area != nil {
			a.requestKnown++
			area.Requests++
			if isUnresolved(r.Status) {
				area.Unresolved++
			}
			typ := r.RequestType
			if typ == "" {
				typ = "Unspecified"
			}
			area.Types[typ]++
			a.allTypes[typ]++
		}
	} else {
		a.vacantCount++
		if area != nil {
			a.vacantKnown++
			area.Vacant++
		}
	}
}
func isUnresolved(status string) bool {
	s := strings.ToLower(strings.TrimSpace(status))
	if s == "" {
		return false
	}
	return !strings.Contains(s, "closed") && !strings.Contains(s, "complete") && !strings.Contains(s, "resolved") && !strings.Contains(s, "cancel")
}
func normalizeNeighborhood(s string) string {
	var b strings.Builder
	space := false
	for _, r := range strings.ToLower(strings.TrimSpace(s)) {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			b.WriteRune(r)
			space = false
		} else if !space && b.Len() > 0 {
			b.WriteByte(' ')
			space = true
		}
	}
	return strings.TrimSpace(b.String())
}

func (a *accumulator) finish() Analysis {
	rows := make([]NeighborhoodSummary, 0, len(a.areas))
	sharedReq, sharedVac := map[string]int{}, map[string]int{}
	positiveVac := []int{}
	for key, w := range a.areas {
		if w.Vacant > 0 {
			positiveVac = append(positiveVac, w.Vacant)
		}
		if w.Requests > 0 && w.Vacant > 0 {
			sharedReq[key] = w.Requests
			sharedVac[key] = w.Vacant
		}
		if w.Requests == 0 && w.Vacant == 0 {
			continue
		}
		row := NeighborhoodSummary{Name: w.Name, Requests: w.Requests, Unresolved: w.Unresolved, RehabSites: w.Vacant}
		if w.Vacant > 0 {
			row.RequestsPer100Rehabs = float64(w.Requests) * 100 / float64(w.Vacant)
		}
		if w.Requests > 0 {
			row.UnresolvedRate = float64(w.Unresolved) * 100 / float64(w.Requests)
		}
		rows = append(rows, row)
	}
	spearman := spearman(sharedReq, sharedVac)
	meanR, sdR := meanSD(rows, func(n NeighborhoodSummary) float64 { return float64(n.Requests) })
	meanV, sdV := meanSD(rows, func(n NeighborhoodSummary) float64 { return float64(n.RehabSites) })
	for i := range rows {
		if sdR > 0 {
			rows[i].JointIndex += (float64(rows[i].Requests) - meanR) / sdR
		}
		if sdV > 0 {
			rows[i].JointIndex += (float64(rows[i].RehabSites) - meanV) / sdV
		}
	}
	sort.Slice(rows, func(i, j int) bool {
		if rows[i].JointIndex == rows[j].JointIndex {
			if rows[i].RehabSites == rows[j].RehabSites {
				return rows[i].Requests > rows[j].Requests
			}
			return rows[i].RehabSites > rows[j].RehabSites
		}
		return rows[i].JointIndex > rows[j].JointIndex
	})
	cutoff := 0
	if len(positiveVac) > 0 {
		sort.Ints(positiveVac)
		idx := int(math.Floor(float64(len(positiveVac)-1) * 0.75))
		cutoff = positiveVac[idx]
	}
	enriched := a.enrichedTypes(cutoff)
	topTypes := make([]TypeSummary, 0, len(a.allTypes))
	for name, count := range a.allTypes {
		share := 0.0
		if a.requestKnown > 0 {
			share = float64(count) * 100 / float64(a.requestKnown)
		}
		topTypes = append(topTypes, TypeSummary{Name: name, Count: count, Share: share})
	}
	sort.Slice(topTypes, func(i, j int) bool { return topTypes[i].Count > topTypes[j].Count })
	label := "insufficient overlap to assess"
	if len(sharedReq) >= 3 {
		switch {
		case spearman >= 0.6:
			label = "strong positive association"
		case spearman >= 0.25:
			label = "moderate positive association"
		case spearman <= -0.6:
			label = "strong negative association"
		case spearman <= -0.25:
			label = "moderate negative association"
		default:
			label = "weak or no monotonic association"
		}
	}
	return Analysis{RequestCount: a.requestCount, RehabCount: a.vacantCount, UnresolvedCount: a.unresolved, MatchedNeighborhoods: len(sharedReq), RequestNeighborhoodCoverage: a.requestKnown, RehabNeighborhoodCoverage: a.vacantKnown, Spearman: spearman, SpearmanLabel: label, HighRehabCutoff: cutoff, Neighborhoods: rows, TopRequestTypes: firstN(topTypes, 10), EnrichedTypes: firstN(enriched, 5)}
}
func (a *accumulator) enrichedTypes(cutoff int) []TypeSummary {
	high, low := map[string]int{}, map[string]int{}
	highTotal, lowTotal := 0, 0
	for _, w := range a.areas {
		if w.Requests == 0 || w.Vacant == 0 {
			continue
		}
		target := low
		if cutoff > 0 && w.Vacant >= cutoff {
			target = high
		}
		for name, count := range w.Types {
			target[name] += count
		}
		if cutoff > 0 && w.Vacant >= cutoff {
			highTotal += w.Requests
		} else {
			lowTotal += w.Requests
		}
	}
	items := []TypeSummary{}
	for name, count := range high {
		if highTotal == 0 || lowTotal == 0 || count < 100 {
			continue
		}
		other := low[name]
		shareHigh := float64(count) / float64(highTotal)
		shareLow := float64(other) / float64(lowTotal)
		enrichment := 0.0
		if shareLow > 0 {
			enrichment = shareHigh / shareLow
		}
		items = append(items, TypeSummary{Name: name, Count: count, Share: shareHigh * 100, Enrichment: enrichment})
	}
	sort.Slice(items, func(i, j int) bool { return items[i].Enrichment > items[j].Enrichment })
	return items
}
func meanSD(rows []NeighborhoodSummary, value func(NeighborhoodSummary) float64) (float64, float64) {
	if len(rows) == 0 {
		return 0, 0
	}
	m := 0.0
	for _, r := range rows {
		m += value(r)
	}
	m /= float64(len(rows))
	v := 0.0
	for _, r := range rows {
		d := value(r) - m
		v += d * d
	}
	return m, math.Sqrt(v / float64(len(rows)))
}
func spearman(x, y map[string]int) float64 {
	keys := []string{}
	for k := range x {
		if _, ok := y[k]; ok {
			keys = append(keys, k)
		}
	}
	if len(keys) < 2 {
		return 0
	}
	rx, ry := rank(keys, x), rank(keys, y)
	mx, my := 0.0, 0.0
	for _, k := range keys {
		mx += rx[k]
		my += ry[k]
	}
	mx /= float64(len(keys))
	my /= float64(len(keys))
	num, dx, dy := 0.0, 0.0, 0.0
	for _, k := range keys {
		u, v := rx[k]-mx, ry[k]-my
		num += u * v
		dx += u * u
		dy += v * v
	}
	if dx == 0 || dy == 0 {
		return 0
	}
	return num / math.Sqrt(dx*dy)
}
func rank(keys []string, vals map[string]int) map[string]float64 {
	sorted := append([]string(nil), keys...)
	sort.Slice(sorted, func(i, j int) bool { return vals[sorted[i]] < vals[sorted[j]] })
	out := map[string]float64{}
	for i := 0; i < len(sorted); {
		j := i + 1
		for j < len(sorted) && vals[sorted[j]] == vals[sorted[i]] {
			j++
		}
		r := (float64(i+1) + float64(j)) / 2
		for k := i; k < j; k++ {
			out[sorted[k]] = r
		}
		i = j
	}
	return out
}

func WriteOutputs(dir string, a Analysis) error {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	j, err := json.MarshalIndent(a, "", "  ")
	if err != nil {
		return err
	}
	if err = os.WriteFile(filepath.Join(dir, "summary.json"), append(j, '\n'), 0o644); err != nil {
		return err
	}
	if err = writeCSV(filepath.Join(dir, "neighborhoods.csv"), a.Neighborhoods); err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(dir, "analysis.md"), []byte(renderReport(a)), 0o644)
}
func writeCSV(path string, rows []NeighborhoodSummary) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	w := csv.NewWriter(f)
	defer w.Flush()
	if err = w.Write([]string{"neighborhood", "311_requests_2026", "nonclosed_311_requests", "vacant_building_rehab_locations", "311_requests_per_100_rehab_sites", "nonclosed_rate_percent", "joint_index"}); err != nil {
		return err
	}
	for _, r := range rows {
		if err = w.Write([]string{r.Name, fmt.Sprint(r.Requests), fmt.Sprint(r.Unresolved), fmt.Sprint(r.RehabSites), fmt.Sprintf("%.2f", r.RequestsPer100Rehabs), fmt.Sprintf("%.2f", r.UnresolvedRate), fmt.Sprintf("%.3f", r.JointIndex)}); err != nil {
			return err
		}
	}
	w.Flush()
	return w.Error()
}
