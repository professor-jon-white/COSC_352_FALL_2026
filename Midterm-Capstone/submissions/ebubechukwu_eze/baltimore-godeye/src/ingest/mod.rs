//! Stage 1: read every raw file concurrently and turn it into typed records.

pub mod chunked;
pub mod schemas;

use std::collections::HashMap;
use std::fs::File;
use std::io::BufReader;
use std::path::Path;
use std::thread;
use std::time::Instant;

use crate::error::AppError;
use crate::geo;
use crate::model::*;
use chunked::{ingest_csv, ParseCtx};
use schemas::*;

pub const CRIME_FILE: &str = "crime_nibrs.csv";
pub const REQUESTS_FILE: &str = "service_requests_311_2025.csv";
pub const VACANTS_FILE: &str = "vacant_building_notices.csv";
pub const LICENSES_FILE: &str = "liquor_licenses.csv";
pub const CAMERAS_FILE: &str = "citiwatch_cameras.geojson";

/// Parsed but not yet cross-referenced data.
pub struct RawData {
    pub crimes: Vec<Crime>,
    pub requests: Vec<ServiceRequest>,
    pub vacants: Vec<VacantNotice>,
    pub licenses: Vec<LicenseRow>,
    pub cameras: Vec<Camera>,
    pub neighborhoods: Vec<Box<str>>,
    pub blocks: Vec<Box<str>>,
    pub sr_types: Vec<Box<str>>,
    pub ingest: Vec<IngestStats>,
}

/// Splits the thread budget across the CSV files in proportion to their size,
/// so the 380 MB 311 export gets most of the parsers and the 1 MB files get one.
fn parser_budget(sizes: &[u64], threads: usize) -> Vec<usize> {
    let total: u64 = sizes.iter().sum::<u64>().max(1);
    sizes.iter().map(|&s| ((threads as u64 * s + total / 2) / total).max(1) as usize).collect()
}

fn load_cameras(path: &Path) -> Result<(Vec<Camera>, IngestStats), AppError> {
    let started = Instant::now();
    let file = File::open(path).map_err(|e| AppError::io(path, e))?;
    let bytes = file.metadata().map_err(|e| AppError::io(path, e))?.len();
    let doc: serde_json::Value = serde_json::from_reader(BufReader::new(file))
        .map_err(|source| AppError::Json { path: path.to_path_buf(), source })?;
    let features = doc["features"].as_array().ok_or_else(|| AppError::Malformed {
        path: path.to_path_buf(),
        reason: "GeoJSON has no 'features' array".into(),
    })?;
    let cameras: Vec<Camera> = features
        .iter()
        .filter_map(|feature| {
            let coords = feature["geometry"]["coordinates"].as_array()?;
            let point = geo::project(coords.get(1)?.as_f64()?, coords.first()?.as_f64()?);
            point.is_some().then_some(Camera { point, neighborhood: NO_NEIGHBORHOOD })
        })
        .collect();
    let stats = IngestStats {
        dataset: "CitiWatch Cameras",
        file: CAMERAS_FILE.into(),
        format: "GeoJSON",
        bytes,
        rows: features.len() as u64,
        kept: cameras.len() as u64,
        parser_threads: 1,
        seconds: started.elapsed().as_secs_f64(),
    };
    Ok((cameras, stats))
}

pub fn load_all(data_dir: &Path, threads: usize) -> Result<RawData, AppError> {
    let csv_files = [CRIME_FILE, REQUESTS_FILE, VACANTS_FILE, LICENSES_FILE];
    let paths = csv_files.map(|f| data_dir.join(f));
    let sizes: Vec<u64> = paths.iter().map(|p| p.metadata().map_or(0, |m| m.len())).collect();
    let budget = parser_budget(&sizes, threads);
    let camera_path = data_dir.join(CAMERAS_FILE);

    let (neighborhoods, blocks, sr_types) = (Interner::new(), Interner::new(), Interner::new());
    let ctx = ParseCtx { neighborhoods: &neighborhoods, blocks: &blocks, sr_types: &sr_types };
    let ctx = &ctx;

    // All five files are read at the same time. Scoped threads may borrow the
    // interners and paths from this stack frame; the borrow checker proves the
    // threads are joined before those values are dropped.
    let (crimes, requests, vacants, licenses, cameras) = thread::scope(|scope| {
        let crimes = scope.spawn(|| ingest_csv::<CrimeSchema>(&paths[0], ctx, budget[0]));
        let requests = scope.spawn(|| ingest_csv::<ServiceRequestSchema>(&paths[1], ctx, budget[1]));
        let vacants = scope.spawn(|| ingest_csv::<VacantNoticeSchema>(&paths[2], ctx, budget[2]));
        let licenses = scope.spawn(|| ingest_csv::<LicenseSchema>(&paths[3], ctx, budget[3]));
        let cameras = scope.spawn(|| load_cameras(&camera_path));
        let panic = |_| AppError::WorkerPanic("ingestion");
        (
            crimes.join().map_err(panic),
            requests.join().map_err(panic),
            vacants.join().map_err(panic),
            licenses.join().map_err(panic),
            cameras.join().map_err(panic),
        )
    });
    let (crimes, s1) = crimes??;
    let (requests, s2) = requests??;
    let (vacants, s3) = vacants??;
    let (licenses, s4) = licenses??;
    let (cameras, s5) = cameras??;

    let mut neighborhoods = neighborhoods.into_labels();
    for label in &mut neighborhoods {
        if !label.chars().any(char::is_lowercase) {
            *label = crate::normalize::title_case(label).into_boxed_str();
        }
    }
    Ok(RawData {
        crimes,
        requests,
        vacants,
        licenses,
        cameras,
        neighborhoods,
        blocks: blocks.into_labels(),
        sr_types: sr_types.into_labels(),
        ingest: vec![s1, s2, s3, s4, s5],
    })
}

/// The licence file is a 24-year history with one row per licensee per year.
/// Reduce it to the establishments holding a licence in the latest year.
pub fn active_outlets(licenses: &[LicenseRow]) -> Vec<LiquorOutlet> {
    let latest = licenses.iter().map(|l| l.year).max().unwrap_or(0);
    let mut unique: HashMap<(&str, BlockId), &LicenseRow> = HashMap::new();
    for row in licenses.iter().filter(|l| l.year == latest && !l.closed) {
        unique.entry((&row.name, row.block)).or_insert(row);
    }
    let mut outlets: Vec<LiquorOutlet> = unique
        .into_values()
        .map(|row| LiquorOutlet {
            name: row.name.clone(),
            block: row.block,
            kind: row.kind,
            neighborhood: NO_NEIGHBORHOOD,
        })
        .collect();
    outlets.sort_by(|a, b| (&a.name, a.block).cmp(&(&b.name, b.block)));
    outlets
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn budget_follows_file_size_but_never_starves() {
        assert_eq!(parser_budget(&[75, 380, 1, 7], 10), vec![2, 8, 1, 1]);
        assert_eq!(parser_budget(&[0, 0], 4), vec![1, 1]);
    }

    #[test]
    fn outlets_keep_only_latest_open_licences() {
        let row = |name: &str, block, year, closed| LicenseRow {
            name: name.into(),
            block,
            year,
            kind: OutletKind::Tavern,
            closed,
        };
        let licenses = vec![
            row("OLD BAR", 1, 2019, false),
            row("CORNER BAR", 2, 2026, false),
            row("CORNER BAR", 2, 2026, false), // second licensee, same place
            row("SHUT BAR", 3, 2026, true),
        ];
        let outlets = active_outlets(&licenses);
        assert_eq!(outlets.len(), 1);
        assert_eq!(&*outlets[0].name, "CORNER BAR");
    }
}
