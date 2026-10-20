import csv, json
G="/Users/jonwhite/Morgan/COSC_352_FALL_2026/Midterm-Capstone/grading"
S="/Users/jonwhite/Morgan/COSC_352_FALL_2026/Midterm-Capstone/submissions"
slug=lambda n:(lambda l,f:f"{f.lower()}_{l.lower()}")(*[x.strip() for x in n.split(",")])
v1={json.load(open(f"{G}/{s}.json"))["student"]:json.load(open(f"{G}/{s}.json")) for s in [slug(r["Student"]) for r in csv.DictReader(open(G+"/grades_v3.csv")) if int(r["Raw v2 Total"])>0]}
EXPL="""## What each column means

| Column | Max | How it was scored |
|---|---|---|
| Concurrency | 20 | Native concurrency primitives (goroutines/channels, Rust threads/rayon, etc.) used across parse, join and aggregate stages with correct synchronization. Full multi-stage worker pools = 20. Concurrency in one real stage (usually ingest only) = about 13. Token concurrency (one goroutine per file) = about 7. Verified by reading the code and by running the Go race detector where feasible. |
| Real Data | 15 | At least two genuine Open Baltimore datasets downloaded onto the computer and used by the documented run. Real data bundled = 15. Real data fetched at runtime but not bundled = 10. Synthetic data earns partial credit only if it uses the exact Open Baltimore schema; invented schemas or hand-typed rows = 0 to 1. Verified by inspecting headers and records against the live portal. |
| Five Questions | 20 | Five distinct research questions for named municipal stakeholders that genuinely cross datasets, with formatted summaries or visualizations. Points are lost for single-dataset questions, trivial counts, unnamed stakeholders, degenerate answers, or no visualization. |
| Three Datasets | 15 | Number of distinct Open Baltimore datasets actually ingested in the documented run. Three or more = 15. Two = 8. Mixed sources where one is not from the portal = 5 to 10. None = 0. |
| Runs Well + Documented | 15 | Run result (10): builds and executes end to end on the included or downloaded data with sensible output. README (5): explicit macOS compile, execute and verify instructions that match reality. Nothing in the submission was modified to make it run. |
| Project Structure | 15 | Separation of ingestion, join, analysis and output; design patterns actually realized in code; package layout; tests; maintainability. |
| Raw Total | 100 | Sum of the six columns. |
| Curved Grade | 100 | Class curve with a 70 floor, piecewise-linear on the raw total. Anchors: 70 printout only, 75 minimal synthetic data, 80 weak concurrency and weak questions, 85 real data but minimal concurrency or questions, 88 to 93 real data with full concurrency on two datasets, 100 for the top projects with three or more real datasets. |

The grader built each project from source exactly as the README described, ran it with timeouts, captured output, and read every source file. Full build logs, run output, race-detector results and per-criterion justifications are in the detailed report referenced below.
"""
for r in csv.DictReader(open(G+"/grades_v3.csv")):
    raw=int(r["Raw v2 Total"])
    if raw==0: continue
    s=slug(r["Student"]); d=v1[s]
    md=f"""# Midterm Capstone Grade — {r['Student'].split(', ')[1]} {r['Student'].split(', ')[0]}

**Raw score: {raw}/100    Curved grade: {r['Curved Grade']}/100**{'    (submitted LATE; noted, not deducted)' if r['Late'] else ''}

Language: {r['Language']}. Builds: {d['builds']}. Runs: {d['runs']}. Data status: {d['data_status']}. Concurrency level: {d['concurrency_level']}. Datasets found: {len(d['datasets'])}. Questions answered: {d['num_questions']}. Visualization: {d['visualization']}. Tests: {d['tests']}. Race detector: {d['race_detector']}.

## Score breakdown

| Column | Score | Max |
|---|---|---|
| Concurrency | {r['Concurrency (20)']} | 20 |
| Real Data | {r['Real Data (15)']} | 15 |
| Five Questions | {r['Five Questions (20)']} | 20 |
| Three Datasets | {r['Three Datasets (15)']} | 15 |
| Runs Well + Documented | {r['Runs Well + Documented (15)']} | 15 |
| Project Structure | {r['Project Structure (15)']} | 15 |
| **Raw Total** | **{raw}** | 100 |
| **Curved Grade** | **{r['Curved Grade']}** | 100 |

## How this grade was reached

{r['Description']}

### Datasets identified
{chr(10).join('- '+x for x in d['datasets'])}

### Concurrency primitives found
{d['concurrency_primitives']}

### Design patterns found
{d['design_patterns']}

### Questions answered
{chr(10).join(f'{i}. {q}' for i,q in enumerate(d['questions'],1))}

Stakeholders named: {', '.join(d['stakeholders']) if d['stakeholders'] else 'none'}

### Overview
{d['overview']}

### Stakeholder value
{d['stakeholder_value']}

### Issues noted
{chr(10).join('- '+x for x in d['issues'])}

{EXPL}
Detailed grader report with build logs and captured program output: `grading/{s}.md` in the course grading folder.
"""
    open(f"{S}/{s}/GRADE.md","w").write(md); print("wrote",s)
