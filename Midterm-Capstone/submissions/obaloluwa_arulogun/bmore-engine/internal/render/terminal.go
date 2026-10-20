package render

import (
"fmt"
"io"
"math"
"strings"

"bmore-engine/internal/model"
)

func RenderTerminal(w io.Writer, rep model.AnalysisReport) {
line := strings.Repeat("=", 88)
subline := strings.Repeat("-", 88)

fmt.Fprintf(w, "\n%s\n", line)
fmt.Fprintf(w, " BALTIMORE CITY MULTI-PARADIGMATIC DATA PROCESSING ENGINE — STAKEHOLDER BRIEFING\n")
fmt.Fprintf(w, "%s\n", line)

t := rep.Telemetry
fmt.Fprintf(w, "\n[SYSTEM TELEMETRY & MEMORY RESOURCE PROFILE]\n")
fmt.Fprintf(w, "  Worker Pool Threads : %d goroutines (Native CSP Pipeline)\n", t.WorkersUsed)
fmt.Fprintf(w, "  Records Processed   : %d (311 Requests) | %d (Vacant Building Notices)\n",
t.Total311Parsed, t.TotalVacantParsed)
fmt.Fprintf(w, "  Stage Latency       : Ingest=%v | Join=%v | Analysis=%v\n",
t.IngestDuration.Round(t.IngestDuration/100+1),
t.JoinDuration.Round(t.JoinDuration/100+1),
t.AnalysisDuration.Round(t.AnalysisDuration/100+1))
fmt.Fprintf(w, "  Heap Memory Footprint: Active=%.2f MB | Cumulative=%.2f MB | OS Sys=%.2f MB | GC Cycles=%d\n",
t.AllocMB, t.TotalAllocMB, t.SysMB, t.NumGC)

fmt.Fprintf(w, "\n%s\n", subline)
fmt.Fprintf(w, " Q1 [DHCD & DPW]: ILLEGAL DUMPING MULTIPLIER IN VACANT BUILDING CORRIDORS\n")
fmt.Fprintf(w, " Research Question: Which neighborhoods experience the highest concentration of\n")
fmt.Fprintf(w, " sanitation/illegal dumping 311 requests relative to registered vacant structures?\n")
fmt.Fprintf(w, "%s\n", subline)
fmt.Fprintf(w, " %-24s | %7s | %11s | %9s | %-24s\n",
"NEIGHBORHOOD", "VACANTS", "DUMPING 311", "RATIO", "BLIGHT INTENSITY CHART")
fmt.Fprintf(w, " %s\n", strings.Repeat("-", 84))

maxRatio := 0.1
for _, p := range rep.Q1BlightHotspots {
r := float64(p.Dumping311Count) / float64(p.VacantCount)
if r > maxRatio {
maxRatio = r
}
}
for _, p := range rep.Q1BlightHotspots {
ratio := float64(p.Dumping311Count) / float64(p.VacantCount)
bar := asciiBar(ratio, maxRatio, 22)
fmt.Fprintf(w, " %-24s | %7d | %11d | %8.2fx | %s\n",
truncate(p.Name, 24), p.VacantCount, p.Dumping311Count, ratio, bar)
}

fmt.Fprintf(w, "\n%s\n", subline)
fmt.Fprintf(w, " Q2 [MAYOR'S CITISTAT]: AGENCY SLA RESOLUTION BOTTLENECKS BY VACANCY DENSITY\n")
fmt.Fprintf(w, " Research Question: How does mean 311 resolution time (hours) compare between\n")
fmt.Fprintf(w, " high-vacancy neighborhoods vs. low-vacancy neighborhoods across city agencies?\n")
fmt.Fprintf(w, "%s\n", subline)
fmt.Fprintf(w, " %-20s | %15s | %15s | %10s | %-16s\n",
"AGENCY", "HIGH-VAC (HRS)", "LOW-VAC (HRS)", "SLA DELTA", "DISPARITY VISUAL")
fmt.Fprintf(w, " %s\n", strings.Repeat("-", 84))

for _, a := range rep.Q2AgencySLAs {
delta := a.HighVacancyAvgHours - a.LowVacancyAvgHours
sign := "+"
if delta < 0 {
sign = ""
}
bar := asciiBar(math.Abs(a.HighVacancyAvgHours), 160.0, 16)
fmt.Fprintf(w, " %-20s | %12.1f hr | %12.1f hr | %s%7.1f h | %s\n",
truncate(a.AgencyName, 20), a.HighVacancyAvgHours, a.LowVacancyAvgHours, sign, delta, bar)
}

fmt.Fprintf(w, "\n%s\n", subline)
fmt.Fprintf(w, " Q3 [DEPT OF TRANSPORTATION]: HAVERSINE SPATIAL PROXIMITY HAZARD JOIN (<= 150m)\n")
fmt.Fprintf(w, " Research Question: Which street/infrastructure hazards (potholes, dark streetlights,\n")
fmt.Fprintf(w, " sidewalk damage) occur within 150m of condemned vacant buildings?\n")
fmt.Fprintf(w, " Total Matched Compounded Hazards Found: %d\n", rep.Q3TotalHazardCount)
fmt.Fprintf(w, "%s\n", subline)
fmt.Fprintf(w, " %-12s | %-20s | %-22s | %-16s | %8s\n",
"311 RECORD", "HAZARD TYPE", "NEIGHBORHOOD", "VACANT NOTICE", "DISTANCE")
fmt.Fprintf(w, " %s\n", strings.Repeat("-", 84))

for _, m := range rep.Q3SpatialHazards {
fmt.Fprintf(w, " %-12s | %-20s | %-22s | %-16s | %6.1f m\n",
truncate(m.RequestID, 12),
truncate(m.SRType, 20),
truncate(m.Neighborhood, 22),
truncate(m.VacantNoticeID, 16),
m.DistanceMeters)
}

fmt.Fprintf(w, "\n%s\n", subline)
fmt.Fprintf(w, " Q4 [CITY COUNCIL APPROPRIATIONS]: DISTRICT COMPOSITE MUNICIPAL BURDEN INDEX\n")
fmt.Fprintf(w, " Research Question: How should capital remediation funds be weighted across City\n")
fmt.Fprintf(w, " Council Districts based on combined vacant property count and open 311 backlog?\n")
fmt.Fprintf(w, "%s\n", subline)
fmt.Fprintf(w, " %-12s | %10s | %12s | %11s | %-26s\n",
"DISTRICT", "VACANTS", "OPEN BACKLOG", "BURDEN IDX", "APPROPRIATION PRIORITY")
fmt.Fprintf(w, " %s\n", strings.Repeat("-", 84))

maxBurden := 1.0
for _, b := range rep.Q4DistrictBurdens {
if b.CompositeBurden > maxBurden {
maxBurden = b.CompositeBurden
}
}
for _, b := range rep.Q4DistrictBurdens {
bar := asciiBar(b.CompositeBurden, maxBurden, 24)
fmt.Fprintf(w, " District %-3d | %10d | %12d | %11.1f | %s\n",
b.DistrictID, b.VacantNotices, b.Open311Backlog, b.CompositeBurden, bar)
}

fmt.Fprintf(w, "\n%s\n", subline)
fmt.Fprintf(w, " Q5 [DEPT OF PLANNING]: INTAKE CHANNEL EQUITY & DIGITAL DIVIDE BY VACANCY TIER\n")
fmt.Fprintf(w, " Research Question: Does reliance on phone vs. mobile app 311 reporting shift across\n")
fmt.Fprintf(w, " neighborhood vacancy tiers, and how does it correlate with ticket closure rates?\n")
fmt.Fprintf(w, "%s\n", subline)
fmt.Fprintf(w, " %-36s | %6s | %9s | %9s | %8s | %8s\n",
"VACANCY STRATIFICATION TIER", "HOODS", "MOBILE %", "PHONE %", "WEB %", "CLOSED %")
fmt.Fprintf(w, " %s\n", strings.Repeat("-", 84))

for _, c := range rep.Q5ChannelEquity {
fmt.Fprintf(w, " %-36s | %6d | %8.1f%% | %8.1f%% | %7.1f%% | %7.1f%%\n",
truncate(c.TierName, 36), c.Neighborhoods, c.MobileAppPct, c.PhoneCallPct, c.WebPortalPct, c.ClosureRatePct)
}
fmt.Fprintf(w, "%s\n\n", line)
}

func asciiBar(val, maxVal float64, width int) string {
if maxVal <= 0 || val <= 0 {
return ""
}
filled := int(math.Round((val / maxVal) * float64(width)))
if filled > width {
filled = width
}
if filled == 0 && val > 0 {
filled = 1
}
return strings.Repeat("#", filled) + strings.Repeat(".", width-filled)
}

func truncate(s string, max int) string {
if len(s) <= max {
return s
}
return s[:max-1] + "~"
}
