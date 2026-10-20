import json, glob, csv, statistics
from collections import Counter
G="/Users/jonwhite/Morgan/COSC_352_FALL_2026/Midterm-Capstone/grading"
roster=["Allen, Jordan","Arulogun, Obaloluwa","Basnet, Arpana","Basnet, Bishant","Bigelow, Nickolas","Brown, Mikayla","Curbeam, Michael","Dada, Daniel","Eboh, Kingsley","Efferson, Anaya","Eze, Ebubechukwu","Hall, Amari","Holder, Gabriel","Ighade, Nicholas","James, Aidan","Jones, Xavier","Joseph, Grace","Lewis, Mekhi","Madu, Jessica","Martin, Montrell","Morton, Tayven","Nwokelo, John","Olajide, Oluwatobi","Olorogun, Abimbola","Qualls, Gladys","Rasheed, Wariz","Reyes, Jayden","Shareef, Nyriq","Smith, Lyric","Stewart, Elijah","Stewart, Elizabeth","Tejan, Imran","Thomas, Tyriqe","Wanjiku, Abigail","Williams, Vernon","Wilson, Drake"]
slug=lambda n:(lambda l,f:f"{f.lower()}_{l.lower()}")(*[x.strip() for x in n.split(",")])
recs={json.load(open(p))["student"]:json.load(open(p)) for p in glob.glob(G+"/*.json")}

# --- Three-datasets criterion (/15): count of distinct Open Baltimore datasets actually ingested in the documented run
DATASETS={ # (score, note)
 "aidan_james":(15,"4 real: NIBRS crime, 311 2025, liquor licenses, vacant notices"),
 "ebubechukwu_eze":(15,"5 real: NIBRS crime, 311 2025, vacant notices, liquor licenses, CitiWatch cameras"),
 "wariz_rasheed":(15,"3 real: 311 2026, NIBRS crime, liquor licenses"),
 "nickolas_bigelow":(15,"4 fetched: Part 1 crime, 311, liquor licenses, vacant notices (licenses arrived without coordinates)"),
 "daniel_dada":(10,"3 bundled (crime, 311 2024, 311 2025) but default run uses only 2; the two 311 files are the same dataset split by year"),
 "jayden_reyes":(10,"3 sources but only 2 from Open Baltimore (arrests, gun offenders); CVI table hand-typed from a MONSE PDF"),
 "bishant_basnet":(8,"2 declared (crime, 311); neither shipped"),
 "tyriqe_thomas":(5,"2 sources, only 1 from Open Baltimore (dropout rate, one citywide row); CompStat hand-typed from PDF"),
 "nicholas_ighade":(0,"2 inline hand-typed tables, nothing from the portal"),
}
DS_DEFAULT=(8,"2 datasets (meets the original minimum, not the 3-dataset bar)")

# --- Real data (/15) overrides for synthetic cases: credit only when synthetic is in the real Open Baltimore schema
REAL={
 "nickolas_bigelow":(9,"No real data bundled and the documented demo is synthetic, but the generator uses exact Open Baltimore schemas and `make fetch` pulls genuine data (verified)"),
 "bishant_basnet":(6,"Nothing shipped; synthetic fallback uses the real crime/311 schema and the shipped report.md proves a real-data run; engine verified on real files by grader"),
 "jessica_madu":(5,"Nothing shipped; gen_sample.py mimics the real schema but with uniform random values (r=0.99 artifact); report_real_data.txt unreproducible"),
 "obaloluwa_arulogun":(3,"Generator output uses real column names but planted effects, fake addresses in wrong neighborhoods, and README misrepresents it as real exports"),
 "gabriel_holder":(1,"10 hand-typed rows in BNIA shape; admitted synthetic"),
 "elijah_stewart":(1,"14 fabricated rows with invented headers, not the Open Baltimore schema"),
 "lyric_smith":(0,"12 hand-written rows, invented schema"),
 "mikayla_brown":(0,"Datasets do not exist on Open Baltimore; 6 synthetic rows"),
 "nicholas_ighade":(0,"16 inline fake rows"),
}

# --- Novelty ranking (judged from the questions actually answered on data)
NOVELTY=[
 ("ebubechukwu_eze","CitiWatch camera coverage gaps: 41.5% of violent crime and 50.7% of shootings happened >150 m from any camera, with a map of where new cameras would see the most. Also the only block-level liquor join that controls for neighborhood."),
 ("aidan_james","Tested whether liquor-block crime is a closing-hours problem (it is not: 21.4% vs 21.3% late-night share, so 'the excess is about place, not time'), and a two-hop council-district capital-budget scorecard."),
 ("abigail_wanjiku","Streetlight requests vs nighttime outdoor crime with a share-based control, and the inverted question 'high 311, low crime' that surfaced Mount Washington and Roland Park as over-reporters (28 requests per crime vs 10 citywide)."),
 ("nickolas_bigelow","Mix-adjusted 311 wait index by agency across crime tiers (widest gap: Forestry/Rec & Parks), and day-of-week x hour patrol timing near liquor outlets."),
 ("wariz_rasheed","Volume-adjusted search for which 311 types track violent crime, and night-time (21:00-03:59) violent crime by license-density tier."),
 ("daniel_dada","Under-reporting z-score (crimes per 311 request) that flagged Inner Harbor at z=7.2, and neighborhoods whose monthly crime curve diverges from the city's (Brooklyn r=-0.27)."),
 ("jordan_allen","Lift analysis of 311 types in top-crime neighborhoods (SW-Dirty Alley Proactive 6.2x) and crime-vs-311 hour-of-day divergence for staffing."),
 ("tayven_morton","Same-request-type closure comparison across vacancy cohorts, isolating parking complaints at +12.4 days in high-vacancy areas."),
 ("michael_curbeam","Category-level correlation (sanitation r=+0.70 vs transportation +0.26) and ZIP-level code-enforcement targeting (21217, 21223, 21213)."),
 ("xavier_jones","Crime-share vs arrest-share gap by neighborhood (Downtown 3.6% of crime, 8.3% of arrests) as an enforcement-distribution signal."),
 ("obaloluwa_arulogun","Intake-channel 'digital divide' by vacancy tier (phone vs app vs web). Novel idea, but answered on synthetic data so no real finding."),
 ("tyriqe_thomas","Leave-one-out predictive test of district gun incidents. Novel method, but on a hand-typed 9-row table."),
]

cols=["concurrency","real_data","questions","datasets","runs_docs","structure"]
maxes=dict(concurrency=20,real_data=15,questions=20,datasets=15,runs_docs=15,structure=15)
rows=[]
for n in roster:
    s=slug(n); d=recs.get(s)
    if not d:
        rows.append(dict(Student=n,slug=s,sub=False,**{c:0 for c in cols},total=0,letter="F (no submission)",lang="-",late="",data="-",conc="-",runs="-",why="No submission in Canvas export.")); continue
    o=d["scores"]
    conc=round(o["concurrency"]*20/15)
    real=REAL.get(s,(o["real_data"],""))[0]
    q=round(o["stakeholder"]*20/15)
    ds=DATASETS.get(s,DS_DEFAULT)[0]
    rd=o["runs_results"]+o["readme"]
    st=o["architecture"]
    tot=conc+real+q+ds+rd+st
    letter="A" if tot>=90 else "B" if tot>=80 else "C" if tot>=70 else "D" if tot>=60 else "F"
    why=(f"Concurrency {d['concurrency_level']} ({d['concurrency_primitives'][:90]}). "
         f"Data {d['data_status']}" + (f": {REAL[s][1]}" if s in REAL else "") + ". "
         f"Datasets: {DATASETS.get(s,DS_DEFAULT)[1]}. Builds {d['builds']}, runs {d['runs']}. "
         f"{d['rationale']}")
    rows.append(dict(Student=n,slug=s,sub=True,concurrency=conc,real_data=real,questions=q,datasets=ds,runs_docs=rd,structure=st,total=tot,letter=letter,lang=d["language"],late="LATE" if d["late"] else "",data=d["data_status"],conc=d["concurrency_level"],runs=d["runs"],why=why,old=d["total"]))

with open(G+"/grades_v2.csv","w",newline="") as f:
    w=csv.writer(f); w.writerow(["Student","Language","Late","Concurrency (20)","Real Data (15)","Five Questions (20)","Three Datasets (15)","Runs Well + Documented (15)","Project Structure (15)","Total (100)","Letter","Previous total (v1 rubric)","How the grade was reached"])
    for r in rows: w.writerow([r["Student"],r["lang"],r["late"]]+[r[c] for c in cols]+[r["total"],r["letter"],r.get("old",""),r["why"]])

sub=[r for r in rows if r["sub"]]; t=[r["total"] for r in sub]
out=["# COSC 352 Midterm Capstone — Grade Report v2 (re-weighted rubric)\n",
"Rubric: Concurrency 20 · Real Data 15 · Five Stakeholder Questions 20 · Three Datasets 15 · Runs Well + Well Documented 15 · Project Structure 15 = 100. Letters: A 90+, B 80, C 70, D 60.\n",
"Mapping from verified evidence: Concurrency and Questions rescaled from the /15 evidence to /20; Real Data re-judged with partial credit only for synthetic data in the genuine Open Baltimore schema; Three Datasets is new (3+ real = 15, 2 = 8, mixed-source = 5-10, none = 0); Runs+Docs = run result (10) + README (5); Structure = architecture evidence (15). Memory management is no longer scored separately.\n",
"## High-level statistics (29 submitted of 36)\n",
f"- Mean {statistics.mean(t):.1f}, median {statistics.median(t):.1f}, std dev {statistics.pstdev(t):.1f}, min {min(t)}, max {max(t)}",
f"- Letters: {dict(sorted(Counter(r['letter'] for r in sub).items()))}",
"- Mean per criterion: "+", ".join(f"{c} {statistics.mean(r[c] for r in sub):.1f}/{maxes[c]}" for c in cols),
f"- Students with 3+ real datasets: {sum(1 for r in sub if r['datasets']==15)} of 29",
f"- Students with genuine multi-stage concurrency: {sum(1 for r in sub if r['conc']=='full')} of 29\n",
"## Novelty ranking — most novel questions answered\n"]
for i,(s,why) in enumerate(NOVELTY,1):
    r=next(r for r in rows if r["slug"]==s); out.append(f"{i}. **{r['Student']}** ({r['total']}) — {why}")
out.append("\n## Grade table\n")
hdr=["Student","Lang","Late","Conc<br>/20","Real<br>Data /15","5 Qs<br>/20","3 Data-<br>sets /15","Runs+<br>Docs /15","Struct<br>/15","**Total**","Letter","v1","How the grade was reached"]
out.append("| "+" | ".join(hdr)+" |"); out.append("|"+"---|"*len(hdr))
for r in sorted(rows,key=lambda r:-r["total"]):
    out.append("| "+" | ".join(str(x) for x in [r["Student"],r["lang"],r["late"]]+[r[c] for c in cols]+[f"**{r['total']}**",r["letter"],r.get("old",""),r["why"]])+" |")
open(G+"/GRADES_v2.md","w").write("\n".join(out))
for r in sorted(rows,key=lambda r:-r["total"]):
    if r["sub"]: print(f"{r['Student']:24} {r['concurrency']:2} {r['real_data']:2} {r['questions']:2} {r['datasets']:2} {r['runs_docs']:2} {r['structure']:2} | {r['total']:3} {r['letter']}  (v1 {r['old']})")
print("\n".join(out[4:10]))
