use crate::models::{Incident, Settlement};
use anyhow::{Context, Result};
use rayon::join;
use std::{fs, path::Path, process::Command};

#[derive(Debug)]
pub struct Sources {
    pub settlements: String,
    pub incidents: String,
}

fn fetch(url: &str) -> Result<Vec<u8>> {
    let output = Command::new("curl")
        .args([
            "--fail",
            "--silent",
            "--show-error",
            "--location",
            "--max-time",
            "20",
            url,
        ])
        .output()
        .with_context(|| "could not start curl for optional live CSV download")?;
    anyhow::ensure!(
        output.status.success(),
        "live download failed: {}",
        String::from_utf8_lossy(&output.stderr)
    );
    Ok(output.stdout)
}

fn load_dataset<T: for<'de> serde::Deserialize<'de>>(
    path: &Path,
    url: Option<&str>,
    label: &str,
) -> Result<(Vec<T>, String)> {
    if let Some(url) = url {
        if let Ok(bytes) = fetch(url) {
            if !bytes.is_empty() {
                if let Ok(records) = parse_csv(&bytes, label) {
                    if !records.is_empty() {
                        return Ok((records, format!("live URL {url}")));
                    }
                }
            }
        }
    }

    let bytes = fs::read(path).with_context(|| format!("reading {}", path.display()))?;
    let records = parse_csv(&bytes, label)?;
    Ok((records, format!("local {}", path.display())))
}

fn parse_csv<T: for<'de> serde::Deserialize<'de>>(
    bytes: &[u8],
    label: &str,
) -> Result<Vec<T>> {
    let mut reader = csv::ReaderBuilder::new()
        .trim(csv::Trim::All)
        .from_reader(bytes);
    reader
        .deserialize()
        .collect::<std::result::Result<Vec<_>, _>>()
        .with_context(|| {
            format!("parsing {label}; expected the documented normalized CSV columns")
        })
}

pub fn load_all(
    dir: &Path,
    settlement_url: Option<&str>,
    incident_url: Option<&str>,
) -> Result<(Vec<Settlement>, Vec<Incident>, Sources)> {
    let gttf = dir.join("gttf.csv");
    let gis = dir.join("incidents.csv");

    let (settlements_result, incidents_result) = join(
        || load_dataset::<Settlement>(&gttf, settlement_url, "settlements"),
        || load_dataset::<Incident>(&gis, incident_url, "incidents"),
    );

    let (settlements, settlements_source) = settlements_result?;
    let (incidents, incidents_source) = incidents_result?;

    anyhow::ensure!(
        !settlements.is_empty() && !incidents.is_empty(),
        "input datasets must each contain at least one record"
    );

    Ok((
        settlements,
        incidents,
        Sources {
            settlements: settlements_source,
            incidents: incidents_source,
        },
    ))
}
