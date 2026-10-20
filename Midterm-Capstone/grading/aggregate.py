import json, glob, csv, statistics, os
G="/Users/jonwhite/Morgan/COSC_352_FALL_2026/Midterm-Capstone/grading"
roster=["Allen, Jordan","Arulogun, Obaloluwa","Basnet, Arpana","Basnet, Bishant","Bigelow, Nickolas","Brown, Mikayla","Curbeam, Michael","Dada, Daniel","Eboh, Kingsley","Efferson, Anaya","Eze, Ebubechukwu","Hall, Amari","Holder, Gabriel","Ighade, Nicholas","James, Aidan","Jones, Xavier","Joseph, Grace","Lewis, Mekhi","Madu, Jessica","Martin, Montrell","Morton, Tayven","Nwokelo, John","Olajide, Oluwatobi","Olorogun, Abimbola","Qualls, Gladys","Rasheed, Wariz","Reyes, Jayden","Shareef, Nyriq","Smith, Lyric","Stewart, Elijah","Stewart, Elizabeth","Tejan, Imran","Thomas, Tyriqe","Wanjiku, Abigail","Williams, Vernon","Wilson, Drake"]
def slug(n): l,f=[x.strip() for x in n.split(",")]; return f"{f.lower()}_{l.lower()}"
recs={}
for p in glob.glob(G+"/*.json"):
    d=json.load(open(p)); recs[d["student"]]=d
cols=["build","real_data","concurrency","memory","architecture","stakeholder","runs_results","readme"]
maxes={"build":15,"real_data":15,"concurrency":15,"memory":10,"architecture":15,"stakeholder":15,"runs_results":10,"readme":5}
rows=[]
for n in roster:
    s=slug(n); d=recs.get(s)
    if not d:
        rows.append({"Student":n,"slug":s,"submitted":False,**{c:0 for c in cols},"total":0,"letter":"F (no submission)","lang":"-","builds":"-","runs":"-","data":"-","conc":"-","late":"","rationale":"No submission found in Canvas export.","overview":"","issues":"No submission.","value":""})
        continue
    sc=d["scores"]; tot=sum(sc.values())
    rows.append({"Student":n,"slug":s,"submitted":True,**sc,"total":tot,"letter":d["letter"],"lang":d["language"],"builds":d["builds"],"runs":d["runs"],"data":d["data_status"],"conc":d["concurrency_level"],"late":"LATE" if d.get("late") else "","rationale":d["rationale"],"overview":d["overview"],"issues":" | ".join(d["issues"]),"value":d["stakeholder_value"]})
with open(G+"/grades.csv","w",newline="") as f:
    w=csv.writer(f); w.writerow(["Student","Language","Late","Builds","Runs","Data","Concurrency"]+[f"{c} ({maxes[c]})" for c in cols]+["Total (100)","Letter","Rationale","Overview","Stakeholder Value","Issues"])
    for r in rows: w.writerow([r["Student"],r["lang"],r["late"],r["builds"],r["runs"],r["data"],r["conc"]]+[r[c] for c in cols]+[r["total"],r["letter"],r["rationale"],r["overview"],r["value"],r["issues"]])
sub=[r for r in rows if r["submitted"]]
tots=[r["total"] for r in sub]
out=[]
out.append("# COSC 352 Midterm Capstone — Grade Report\n")
out.append(f"Roster: {len(roster)} students. Submitted: {len(sub)}. No submission: {len(rows)-len(sub)}.\n")
out.append("## High-level statistics (submitted projects only)\n")
out.append(f"- Mean {statistics.mean(tots):.1f}, median {statistics.median(tots):.1f}, std dev {statistics.pstdev(tots):.1f}, min {min(tots)}, max {max(tots)}")
from collections import Counter
out.append(f"- Letter distribution: {dict(sorted(Counter(r['letter'][0] for r in sub).items()))}")
out.append(f"- Language: {dict(Counter(r['lang'] for r in sub))}")
out.append(f"- Builds: {dict(Counter(r['builds'] for r in sub))}; Runs: {dict(Counter(r['runs'] for r in sub))}")
out.append(f"- Data status: {dict(Counter(r['data'] for r in sub))}")
out.append(f"- Concurrency level: {dict(Counter(r['conc'] for r in sub))}")
out.append(f"- Late submissions: {sum(1 for r in sub if r['late'])}")
out.append("- Mean per criterion: " + ", ".join(f"{c} {statistics.mean(r[c] for r in sub):.1f}/{maxes[c]}" for c in cols) + "\n")
out.append("## Grade table\n")
hdr=["Student","Lang","Late","Builds","Runs","Data","Conc."]+[f"{c}<br>/{maxes[c]}" for c in cols]+["**Total**","Letter","How the grade was reached"]
out.append("| "+" | ".join(hdr)+" |"); out.append("|"+"---|"*len(hdr))
for r in rows:
    out.append("| "+" | ".join(str(x) for x in [r["Student"],r["lang"],r["late"],r["builds"],r["runs"],r["data"],r["conc"]]+[r[c] for c in cols]+[f"**{r['total']}**",r["letter"],r["rationale"]])+" |")
out.append("\n## Per-student overview, stakeholder value, and issues\n")
for r in rows:
    out.append(f"### {r['Student']} — {r['total']}/100 ({r['letter']}) {r['late']}")
    if r["submitted"]:
        out.append(f"- **Overview:** {r['overview']}")
        out.append(f"- **Stakeholder value:** {r['value']}")
        out.append(f"- **Issues:** {r['issues']}")
        out.append(f"- Full report: `grading/{r['slug']}.md`\n")
    else: out.append("- No submission found.\n")
open(G+"/GRADES.md","w").write("\n".join(out))
print(f"{len(sub)} graded; wrote GRADES.md and grades.csv")
