//! The four-stage pipeline and its builder.
//!
//! ```text
//!  1 ingest     5 files read at once, CSVs parsed by a pool      (threads + channels)
//!  2 match      learn lookup tables, attribute every record      (rayon, MatchStrategy chain)
//!  3 aggregate  reduce records to neighborhood / block tables    (rayon map-reduce + atomics)
//!  4 analyse    five questions answered at the same time         (rayon, Question trait)
//! ```

use std::path::PathBuf;
use std::time::Instant;

use rayon::prelude::*;

use crate::analysis::{self, Context, Question};
use crate::error::AppError;
use crate::ingest;
use crate::matching::{BlockAddress, MatchChain, MatchStats, NativeName, Probe, SpatialCell};
use crate::model::*;
use crate::report::{int, pct, Block, Report, Table};

pub struct PipelineBuilder {
    data_dir: PathBuf,
    threads: usize,
    camera_radius_m: f32,
    top_n: usize,
    questions: Vec<String>,
}

impl Default for PipelineBuilder {
    fn default() -> Self {
        PipelineBuilder {
            data_dir: PathBuf::from("data"),
            threads: std::thread::available_parallelism().map_or(4, |n| n.get()),
            camera_radius_m: 150.0,
            top_n: 10,
            questions: Vec::new(),
        }
    }
}

impl PipelineBuilder {
    pub fn data_dir(mut self, dir: impl Into<PathBuf>) -> Self {
        self.data_dir = dir.into();
        self
    }

    pub fn threads(mut self, threads: usize) -> Self {
        self.threads = threads;
        self
    }

    pub fn camera_radius_m(mut self, metres: f32) -> Self {
        self.camera_radius_m = metres;
        self
    }

    pub fn top_n(mut self, n: usize) -> Self {
        self.top_n = n;
        self
    }

    pub fn question(mut self, id: impl Into<String>) -> Self {
        self.questions.push(id.into());
        self
    }

    /// Validates the configuration once, so `Pipeline::run` cannot be handed a bad one.
    pub fn build(self) -> Result<Pipeline, AppError> {
        if !(1..=256).contains(&self.threads) {
            return Err(AppError::Config("--threads must be between 1 and 256".into()));
        }
        if !(10.0..=2_000.0).contains(&self.camera_radius_m) {
            return Err(AppError::Config("--camera-radius must be between 10 and 2000 metres".into()));
        }
        if !(1..=100).contains(&self.top_n) {
            return Err(AppError::Config("--top must be between 1 and 100".into()));
        }
        if !self.data_dir.is_dir() {
            return Err(AppError::Config(format!(
                "data directory '{}' not found (run scripts/fetch_data.sh, or pass --data-dir)",
                self.data_dir.display()
            )));
        }
        // A private pool keeps the thread budget explicit and leaves rayon's global pool untouched.
        let pool = rayon::ThreadPoolBuilder::new()
            .num_threads(self.threads)
            .thread_name(|i| format!("analytics-{i}"))
            .build()
            .map_err(|e| AppError::Config(format!("cannot start thread pool: {e}")))?;
        Ok(Pipeline {
            questions: analysis::questions(&self.questions)?,
            data_dir: self.data_dir,
            threads: self.threads,
            camera_radius_m: self.camera_radius_m,
            top_n: self.top_n,
            pool,
        })
    }
}

pub struct Pipeline {
    data_dir: PathBuf,
    threads: usize,
    camera_radius_m: f32,
    top_n: usize,
    questions: Vec<Box<dyn Question>>,
    pool: rayon::ThreadPool,
}

impl Pipeline {
    pub fn builder() -> PipelineBuilder {
        PipelineBuilder::default()
    }

    pub fn run(&self, progress: &dyn Fn(&str)) -> Result<Report, AppError> {
        let mut timings = Vec::new();
        let mut stage = Instant::now();
        let mut lap = |name: &str, timings: &mut Vec<(String, f64)>| {
            timings.push((name.to_string(), stage.elapsed().as_secs_f64()));
            stage = Instant::now();
        };

        progress("[1/4] ingesting 5 datasets concurrently");
        let raw = ingest::load_all(&self.data_dir, self.threads)?;
        lap("ingest", &mut timings);

        progress("[2/4] matching records across datasets");
        let (store, chain_names, match_stats) = self.pool.install(|| match_records(raw));
        lap("match", &mut timings);

        progress("[3/4] aggregating by neighborhood and block");
        let ctx = self.pool.install(|| Context::build(&store, self.camera_radius_m, self.top_n));
        lap("aggregate", &mut timings);

        progress("[4/4] answering research questions in parallel");
        let sections = self.pool.install(|| self.questions.par_iter().map(|q| q.run(&ctx)).collect());
        lap("analyse", &mut timings);
        timings.push(("total".into(), timings.iter().map(|t| t.1).sum()));

        Ok(Report {
            title: "Baltimore GodEye".into(),
            subtitle: format!(
                "Cross-dataset analysis of Open Baltimore data · crime {} · {} worker threads",
                ctx.crime_period(),
                self.threads
            ),
            overview: overview(&store, &chain_names, &match_stats),
            sections,
            timings,
        })
    }
}

/// Stage 2: learn the lookup tables from the labelled records, then attribute
/// every record in every dataset to a neighborhood.
fn match_records(raw: ingest::RawData) -> (DataStore, Vec<&'static str>, Vec<MatchStats>) {
    let ingest::RawData { mut crimes, mut requests, mut vacants, licenses, mut cameras, neighborhoods, blocks, sr_types, ingest } = raw;
    let mut outlets = ingest::active_outlets(&licenses);
    drop(licenses); // 31k heap-allocated names are no longer needed

    // The two tables are independent, so they are learned side by side.
    let (spatial, by_block) = rayon::join(
        || SpatialCell::learn(&crimes, &requests, &neighborhoods),
        || BlockAddress::learn(&crimes, &requests, &vacants, &neighborhoods, blocks.len()),
    );
    let block_neighborhood = by_block.table().to_vec();
    let chain = MatchChain::new(vec![Box::new(NativeName), Box::new(spatial), Box::new(by_block)]);

    let stats = vec![
        chain.apply(
            "NIBRS Group A Crime",
            &mut crimes,
            |c| Probe { native: c.neighborhood, point: c.point, block: c.block },
            |c, id| c.neighborhood = id,
        ),
        chain.apply(
            "311 Service Requests 2025",
            &mut requests,
            |r| Probe { native: r.neighborhood, point: r.point, block: r.block },
            |r, id| r.neighborhood = id,
        ),
        chain.apply(
            "Vacant Building Notices",
            &mut vacants,
            |v| Probe { native: v.neighborhood, point: Point::NONE, block: v.block },
            |v, id| v.neighborhood = id,
        ),
        chain.apply(
            "Liquor Licenses (active outlets)",
            &mut outlets,
            |o| Probe { native: NO_NEIGHBORHOOD, point: Point::NONE, block: o.block },
            |o, id| o.neighborhood = id,
        ),
        chain.apply(
            "CitiWatch Cameras",
            &mut cameras,
            |c| Probe { native: NO_NEIGHBORHOOD, point: c.point, block: NO_BLOCK },
            |c, id| c.neighborhood = id,
        ),
    ];
    let names = chain.strategy_names();
    let store =
        DataStore { crimes, requests, vacants, outlets, cameras, neighborhoods, blocks, sr_types, block_neighborhood, ingest };
    (store, names, stats)
}

fn overview(store: &DataStore, chain_names: &[&'static str], match_stats: &[MatchStats]) -> Vec<Block> {
    let mut inventory = Table::new(
        "Datasets ingested (raw exports from data.baltimorecity.gov)",
        &["Dataset", "File", "Format", "#Size (MB)", "#Rows read", "#Rows kept", "#Parser threads", "#Seconds"],
    );
    for s in &store.ingest {
        inventory.row(vec![
            s.dataset.into(),
            s.file.clone(),
            s.format.into(),
            format!("{:.1}", s.bytes as f64 / 1e6),
            int(s.rows),
            int(s.kept),
            s.parser_threads.to_string(),
            format!("{:.2}", s.seconds),
        ]);
    }

    let mut headers = vec!["Dataset".to_string(), "#Records".to_string()];
    headers.extend(chain_names.iter().map(|n| format!("#By {n}")));
    headers.push("#Unmatched".into());
    let header_refs: Vec<&str> = headers.iter().map(String::as_str).collect();
    let mut matching = Table::new("Cross-dataset matching: how each record was attributed to a neighborhood", &header_refs);
    for s in match_stats {
        let total: u64 = s.counts.iter().sum();
        let mut row = vec![s.dataset.to_string(), int(total)];
        row.extend(s.counts.iter().map(|&n| format!("{} ({})", int(n), pct(n as f64, total as f64))));
        matching.row(row);
    }

    let facts = Block::Facts {
        title: "Shared vocabularies built during ingestion".into(),
        items: vec![
            ("Distinct neighborhoods".into(), int(store.neighborhoods.len() as u64)),
            ("Distinct hundred-blocks (address join keys)".into(), int(store.blocks.len() as u64)),
            ("Distinct 311 request types".into(), int(store.sr_types.len() as u64)),
            ("Active liquor outlets after de-duplicating licence history".into(), int(store.outlets.len() as u64)),
            (
                "Note on unmatched 311 rows".into(),
                "information calls with no address or coordinates; they cannot be placed and are excluded from neighborhood figures".into(),
            ),
        ],
    };
    vec![Block::Table(inventory), Block::Table(matching), facts]
}
