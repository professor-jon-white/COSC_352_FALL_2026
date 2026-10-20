# Baltimore Municipal Intelligence Engine (BMIE)

A high-performance data analytics tool built in **Rust** that looks at Baltimore police misconduct settlements and crime incident records, processes them super fast using multi-threading, and shows the insights through both a terminal CLI and a clean web dashboard.

---

## What This Project Does

City leaders need a clear way to see how police misconduct and settlement payouts break down across different districts. This app takes two official Open Baltimore datasets (Gun Trace Task Force settlements and BPD GIS incidents), links them together safely, and answers 5 big questions:
1. **District Costs:** Which police districts cost the city the most money in settlements?
2. **Unit Hotspots:** Which police units and shifts have the highest concentration of matched incidents and payouts?
3. **Timelines:** How long does it actually take from when misconduct is reported to when disciplinary action happens?
4. **Repeat Officers:** Which officer IDs show up multiple times, and how much do they cost compared to the department average?
5. **District Risk Index:** A calculated score for each district combining financial liability and incident volume to help leaders allocate resources better.

---

## How It's Built (The Tech Stack)

* **Rust:** The whole core engine is written in Rust so it's crazy fast, memory-safe, and doesn't crash or lag.
* **Concurrency (Tokio & Rayon):** Instead of processing data one row at a time, it uses **Tokio** to handle background tasks and **Rayon** to split the heavy lifting across multiple CPU cores at once.
* **Clean Code Patterns:** 
  * **Pipeline:** Data flows smoothly step-by-step from raw files $\rightarrow$ parsing $\rightarrow$ matching $\rightarrow$ analysis.
  * **Builder:** Lets you easily filter results down to a specific district (like Western or Central).
  * **Strategy:** Keeps the math for each of the 5 research questions neatly separated into its own modular block.
* **Visual Dashboards:** Gives you cool ASCII charts right in your terminal when you run commands, plus a companion `index.html` web page for a clean visual view.

---

## Project Layout

```text
baltimore_intelligence/
├── Cargo.toml          # Project config and package dependencies
├── run.sh              # Quick-start script to build and run the app
├── index.html          # Visual web dashboard for city leaders
├── data/               # The normalized CSV files
│   ├── gttf.csv
│   └── incidents.csv
└── src/
    ├── main.rs         # Starting point (bridges async Tokio with Rayon)
    ├── models.rs       # Data structures for settlements and incidents
    ├── concurrency.rs  # Shared multi-threading helpers
    ├── ingestion.rs    # Loads and parses CSVs in parallel
    ├── pipeline.rs     # Matches records together safely without copying heavy data
    ├── analysis.rs     # The logic and math for all 5 research questions
    └── ui.rs           # Renders the terminal output and ASCII charts
    