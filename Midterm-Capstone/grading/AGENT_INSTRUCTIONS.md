You are grading student submissions for COSC 352 Midterm Capstone. Read grading/RUBRIC.md first (absolute path:
/Users/jonwhite/Morgan/COSC_352_FALL_2026/Midterm-Capstone/grading/RUBRIC.md) and the assignment text at
/Users/jonwhite/Morgan/COSC_352_FALL_2026/Midterm-Capstone/assignment-requirments.txt.

Each student's extracted submission is in /Users/jonwhite/Morgan/COSC_352_FALL_2026/Midterm-Capstone/submissions/<firstname_lastname>/.
Toolchains available: go 1.24 (/opt/homebrew/bin/go), cargo/rustc 1.88, scala, sbt, clojure CLI (no lein), java 22, python3.
Network access is available if a program fetches data at runtime (note that it did so).

For EACH assigned student do ALL of the following, in order:
1. Inventory: list files, language, LOC per source file (wc -l), presence of README/run script/tests/data files, data sizes.
2. Read the README and ALL source files fully (they are small). Identify: datasets used, where data came from, concurrency
   primitives and where used (file:line), memory/resource management evidence, design patterns actually realized,
   the 5 stakeholder questions and the stakeholders named, output format/visualization.
3. Verify data realness: head -3 each data file, check column headers against Open Baltimore schemas, look at a few
   records (addresses, neighborhoods like "Canton", "Sandtown-Winchester", real dates). Look for mock/sample generators
   and whether they are the default path. Report row counts (wc -l).
4. Build exactly as README says (go build ./..., cargo build --release, sbt compile, etc.) with a 5 min timeout. Record
   success/failure and exact errors. Also run `go vet ./...` / `cargo check` style checks and `go test ./...` if tests exist
   (3 min timeout) and note results. For Go: run with `go build -race` if feasible, and if runtime allows, run the
   program once with the -race binary to detect data races (skip if the data is huge; note it).
5. Run exactly as README says (run.sh / make / go run / cargo run) with a 10 min timeout (use `timeout 600` or
   the Bash tool timeout). Capture stdout/stderr to the scratchpad. Record: did it run, wall time (use `time`), peak
   memory if easy (`/usr/bin/time -l` on macOS gives max RSS), and the 5 answers it printed. If it needs the user to
   download data first and the data is NOT present, report that, then (only then) try the documented sample/mock path
   and note that real data was absent. If it serves a web UI, curl the page(s) and capture them; kill the server afterward.
6. Write the overview: style, approach, code structure, maintainability, originality. Assess the value to hypothetical
   stakeholders (would the Mayor's Office / DOT / Planning actually learn something actionable? Are the joins meaningful
   or just coincidental counting?). Be specific and honest. Note anything that looks copy-pasted/generic or AI-boilerplate
   that doesn't match the code.
7. Score each rubric criterion with a one-line justification each. NEVER change student code or data files.

OUTPUT (required, per student):
A) /Users/jonwhite/Morgan/COSC_352_FALL_2026/Midterm-Capstone/grading/<firstname_lastname>.md — the full written report
   with sections: Summary line (total/100, letter), Inventory, Datasets & Data Realness, Build Results (commands + output),
   Test/Vet/Race Results, Run Results (commands, timing, memory, captured output incl. the 5 answers), Concurrency Analysis,
   Memory/Resource Analysis, Architecture & Code Quality, Stakeholder Questions & Value Assessment, Issues, Score Table
   with justification per criterion, Final Grade Rationale paragraph.
B) /Users/jonwhite/Morgan/COSC_352_FALL_2026/Midterm-Capstone/grading/<firstname_lastname>.json — EXACT schema:
{
 "student": "firstname_lastname",
 "display_name": "Firstname Lastname",
 "late": false,
 "language": "Go|Rust|Scala|Clojure|Python|None",
 "datasets": ["..."],
 "data_status": "real_bundled|real_fetched_runtime|sample_only|simulated|absent",
 "data_rows": {"file": rows},
 "builds": "yes|no|partial",
 "build_error": "" ,
 "tests": "pass|fail|none",
 "race_detector": "clean|races|not_run",
 "runs": "yes|no|partial",
 "run_wall_seconds": 0,
 "run_max_rss_mb": 0,
 "concurrency_primitives": "...",
 "concurrency_level": "full|partial|token|none",
 "design_patterns": "...",
 "num_questions": 5,
 "questions": ["..."],
 "stakeholders": ["..."],
 "visualization": "ascii|html|web_ui|table_only|none",
 "loc": 0,
 "num_source_files": 0,
 "has_tests": true,
 "scores": {"build":0,"real_data":0,"concurrency":0,"memory":0,"architecture":0,"stakeholder":0,"runs_results":0,"readme":0},
 "total": 0,
 "letter": "A|B|C|D|F",
 "overview": "2-4 sentence description of style/approach/structure/maintainability",
 "stakeholder_value": "2-3 sentences",
 "issues": ["..."],
 "rationale": "2-4 sentences explaining how the grade was reached"
}
Make sure total == sum of scores and the JSON is valid (verify with python3 -m json.tool).
When finished, reply with a compact summary table (student, total, builds, runs, data_status, concurrency_level) and the top issue per student.
