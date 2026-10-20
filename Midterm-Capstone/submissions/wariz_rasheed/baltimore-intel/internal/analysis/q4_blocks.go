package analysis

import (
	"fmt"
	"sort"

	"baltimoreintel/internal/model"
	"baltimoreintel/internal/report"
)

// LicensedBlocks (licenses x crime x 311, joined at street-block level): do
// blocks with a liquor license see more crime than blocks without one?
type LicensedBlocks struct{}

// ID implements Analysis.
func (LicensedBlocks) ID() string { return "Q4" }

type blockGroup struct {
	Label                    string
	Blocks                   int
	Crimes, Violent, Night   int
}

func (g *blockGroup) add(b *model.BlockAgg) {
	g.Blocks++
	g.Crimes += b.Crimes
	g.Violent += b.Violent
	g.Night += b.NightViolent
}

func (g *blockGroup) avg(total int) float64 {
	if g.Blocks == 0 {
		return 0
	}
	return float64(total) / float64(g.Blocks)
}

// minKindBlocks is the fewest blocks a license type needs to be listed.
const minKindBlocks = 5

// Run implements Analysis.
func (LicensedBlocks) Run(d *model.Dataset) report.Result {
	res := report.Result{
		ID:       "Q4",
		Title:    "Are licensed blocks more violent than unlicensed blocks?",
		Audience: "City Planning / Board of Liquor License Commissioners",
		Question: "Joining the three datasets at the level of individual street blocks: how much violent crime " +
			"does the average block with a liquor license see compared with a block without one, and which license types stand out?",
	}
	none := &blockGroup{Label: "No liquor license"}
	licensed := &blockGroup{Label: "Any liquor license"}
	kinds := make(map[string]*blockGroup)
	for _, b := range d.Blocks {
		if len(b.Kinds) == 0 {
			none.add(b)
			continue
		}
		licensed.add(b)
		seen := make(map[string]bool, len(b.Kinds))
		for _, k := range b.Kinds {
			if seen[k] {
				continue
			}
			seen[k] = true
			g := kinds[k]
			if g == nil {
				g = &blockGroup{Label: k}
				kinds[k] = g
			}
			g.add(b)
		}
	}
	if none.Blocks == 0 || licensed.Blocks == 0 {
		res.Finding = "Not enough street blocks with and without licenses in the analysis window to compare."
		return res
	}

	var list []*blockGroup
	for _, g := range kinds {
		if g.Blocks >= minKindBlocks {
			list = append(list, g)
		}
	}
	sort.Slice(list, func(i, j int) bool {
		ai, aj := list[i].avg(list[i].Violent), list[j].avg(list[j].Violent)
		if ai != aj {
			return ai > aj
		}
		return list[i].Label < list[j].Label
	})

	base := none.avg(none.Violent)
	ratio := func(g *blockGroup) string {
		if base == 0 {
			return "n/a"
		}
		return f1(g.avg(g.Violent)/base) + "x"
	}

	groups := append([]*blockGroup{none, licensed}, list...)
	chart := report.Chart{Title: "Average violent crimes per street block"}
	table := report.Table{
		Title:   "Crime per street block by liquor-license status and type",
		Headers: []string{"Block group", "Blocks", "Avg crimes", "Avg violent crimes", "Avg night violent crimes", "Violent vs unlicensed"},
	}
	for _, g := range groups {
		label := g.Label
		if g != none && g != licensed {
			label = pretty(g.Label)
		}
		chart.Bars = append(chart.Bars, report.Bar{
			Label: trunc(label, 30),
			Value: g.avg(g.Violent),
			Text:  f2(g.avg(g.Violent)),
		})
		table.Rows = append(table.Rows, []string{
			label, commas(g.Blocks), f2(g.avg(g.Crimes)), f2(g.avg(g.Violent)), f2(g.avg(g.Night)), ratio(g),
		})
	}

	res.Finding = fmt.Sprintf("The %s street blocks with a liquor license average %s violent crimes per block, "+
		"versus %s on the %s blocks without one (%s as many).",
		commas(licensed.Blocks), f2(licensed.avg(licensed.Violent)), f2(none.avg(none.Violent)), commas(none.Blocks), ratio(licensed))
	if len(list) > 0 {
		res.Finding += fmt.Sprintf(" The license type with the highest average is \"%s\" (%s violent crimes per block).",
			pretty(list[0].Label), f2(list[0].avg(list[0].Violent)))
	}
	res.Charts = []report.Chart{chart}
	res.Tables = []report.Table{table}
	res.Notes = []string{
		"A block is a 100-block of one street, built by normalising the address in each dataset; " +
			"this is the only key all three datasets share.",
		"The block universe is every block that appears in the crime, 311 or license data inside the window. " +
			"Blocks with no crime, no 311 request and no license are not in any file, so averages for unlicensed blocks are somewhat overstated.",
		fmt.Sprintf("License types with fewer than %d blocks are not listed. A block with several license types counts once in each.", minKindBlocks),
		"Licensed blocks are mostly commercial corridors with more foot traffic and more reporting, so the gap does not show that licenses cause crime.",
	}
	return res
}
