mod analysis;
mod concurrency;
mod ingestion;
mod models;
mod pipeline;
mod ui;

use anyhow::Result;
use clap::Parser;
use std::path::PathBuf;

#[derive(Debug, Parser)]
#[command(
    name = "bmie",
    about = "Baltimore Municipal Intelligence Engine (offline-first demo)"
)]
struct Args {
    /// Directory containing gttf.csv and incidents.csv
    #[arg(long, default_value = "data")]
    data_dir: PathBuf,
    /// Filter results to a district (for example Western or Central)
    #[arg(long)]
    district: Option<String>,
    /// Optional URLs of normalized CSV files; local assets remain the fallback
    #[arg(long)]
    settlements_url: Option<String>,
    #[arg(long)]
    incidents_url: Option<String>,
}

#[tokio::main]
async fn main() -> Result<()> {
    let args = Args::parse();
    let Args {
        data_dir,
        district,
        settlements_url,
        incidents_url,
    } = args;

    let (settlements, incidents, sources, report) = tokio::task::spawn_blocking(move || {
        let (settlements, incidents, sources) = ingestion::load_all(
            &data_dir,
            settlements_url.as_deref(),
            incidents_url.as_deref(),
        )?;
        let config = analysis::QueryConfig::builder().district(district).build();
        let joined = pipeline::join(&settlements, &incidents, &config);
        let report = analysis::run_all(&settlements, &incidents, &joined, &config);

        Ok::<_, anyhow::Error>((settlements, incidents, sources, report))
    })
    .await??;

    ui::render(&report, &sources, settlements.len(), incidents.len());
    Ok(())
}
