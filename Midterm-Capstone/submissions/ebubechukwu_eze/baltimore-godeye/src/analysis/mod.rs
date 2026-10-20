//! Stages 3 and 4: aggregate the matched records, then answer the questions.
//!
//! `Context::build` reduces ~1.2 million records to two small tables (one row
//! per neighborhood, one per hundred-block) in parallel. Each research
//! question is a `Question` implementation that reads the shared, immutable
//! context; the pipeline runs all of them at the same time.

mod cameras;
mod citymap;
mod liquor;
mod priority;
mod service;
mod vacancy;

use std::sync::atomic::{AtomicU32, Ordering};

use rayon::prelude::*;

use crate::error::AppError;
use crate::geo::SpatialGrid;
use crate::model::*;
use crate::normalize::month_label;
use crate::report::Section;

/// Per-neighborhood totals across all five datasets.
#[derive(Clone, Default, Debug)]
pub struct NbAgg {
    pub crimes: u32,
    pub violent: u32,
    pub shootings: u32,
    /// "12m" = the final 12 months of the crime data.
    pub violent_12m: u32,
    pub shootings_12m: u32,
    /// Violent crimes farther than the camera radius from every camera.
    pub uncovered_12m: u32,
    pub vacants: u32,
    /// 311 requests needing field work (no info calls, no duplicates).
    pub requests: u32,
    pub blight: u32,
    pub outlets: u32,
    pub cameras: u32,
}

impl NbAgg {
    fn merge(&mut self, o: &NbAgg) {
        self.crimes += o.crimes;
        self.violent += o.violent;
        self.shootings += o.shootings;
        self.violent_12m += o.violent_12m;
        self.shootings_12m += o.shootings_12m;
        self.uncovered_12m += o.uncovered_12m;
        self.vacants += o.vacants;
        self.requests += o.requests;
        self.blight += o.blight;
        self.outlets += o.outlets;
        self.cameras += o.cameras;
    }
}

/// Per-hundred-block totals: the result of the address join.
#[derive(Clone, Default, Debug)]
pub struct BlockAgg {
    pub crimes: u32,
    pub violent: u32,
    pub vacants: u32,
    pub taverns: u16,
    pub package: u16,
    pub restaurants: u16,
    pub other_outlets: u16,
}

impl BlockAgg {
    pub fn outlets(&self) -> u32 {
        u32::from(self.taverns) + u32::from(self.package) + u32::from(self.restaurants) + u32::from(self.other_outlets)
    }
}

pub struct Context<'a> {
    pub store: &'a DataStore,
    pub camera_radius_m: f32,
    pub top_n: usize,
    pub nb: Vec<NbAgg>,
    pub blocks: Vec<BlockAgg>,
    pub cameras: SpatialGrid,
    /// Neighborhoods with any activity, in name order. Analyses iterate in
    /// this order so floating-point sums are identical from run to run.
    pub order: Vec<NeighborhoodId>,
    pub first_ts: u32,
    pub last_ts: u32,
    /// Start of the final 12 months of crime data.
    pub recent_cutoff: u32,
}

fn merge_tables(mut a: Vec<NbAgg>, b: Vec<NbAgg>) -> Vec<NbAgg> {
    for (x, y) in a.iter_mut().zip(&b) {
        x.merge(y);
    }
    a
}

impl<'a> Context<'a> {
    pub fn build(store: &'a DataStore, camera_radius_m: f32, top_n: usize) -> Self {
        let nb_len = store.neighborhoods.len();
        let cameras = SpatialGrid::build(store.cameras.iter().map(|c| c.point), camera_radius_m.max(50.0));
        let (first_ts, last_ts) = store
            .crimes
            .par_iter()
            .map(|c| (c.ts, c.ts))
            .reduce(|| (u32::MAX, 0), |a, b| (a.0.min(b.0), a.1.max(b.1)));
        let recent_cutoff = last_ts.saturating_sub(365 * 86_400);

        // Block counters are shared atomics: every worker increments them
        // lock-free, which avoids giving each task its own block-sized table.
        let atomic_table = || (0..store.blocks.len()).map(|_| AtomicU32::new(0)).collect::<Vec<_>>();
        let (block_crimes, block_violent) = (atomic_table(), atomic_table());

        // Neighborhood totals use map-reduce: each chunk fills a private
        // table, and tables are merged pairwise.
        let empty = || vec![NbAgg::default(); nb_len];
        let from_crimes = store
            .crimes
            .par_chunks(16_384)
            .map(|chunk| {
                let mut table = empty();
                for c in chunk {
                    let violent = c.offense.is_violent();
                    if let Some(counter) = block_crimes.get(c.block as usize) {
                        counter.fetch_add(1, Ordering::Relaxed);
                        if violent {
                            block_violent[c.block as usize].fetch_add(1, Ordering::Relaxed);
                        }
                    }
                    let Some(row) = table.get_mut(c.neighborhood as usize) else { continue };
                    let recent = c.ts >= recent_cutoff;
                    row.crimes += 1;
                    row.violent += u32::from(violent);
                    row.shootings += u32::from(c.shooting);
                    row.violent_12m += u32::from(violent && recent);
                    row.shootings_12m += u32::from(c.shooting && recent);
                    if violent && recent && c.point.is_some() && !cameras.any_within(c.point, camera_radius_m) {
                        row.uncovered_12m += 1;
                    }
                }
                table
            })
            .reduce(empty, merge_tables);

        let from_requests = store
            .requests
            .par_chunks(16_384)
            .map(|chunk| {
                let mut table = empty();
                for r in chunk.iter().filter(|r| !r.duplicate) {
                    let Some(row) = table.get_mut(r.neighborhood as usize) else { continue };
                    row.requests += u32::from(matches!(r.category, SrCategory::Blight | SrCategory::Other));
                    row.blight += u32::from(r.category == SrCategory::Blight);
                }
                table
            })
            .reduce(empty, merge_tables);

        let mut nb = merge_tables(from_crimes, from_requests);
        let mut blocks: Vec<BlockAgg> = block_crimes
            .into_iter()
            .zip(block_violent)
            .map(|(crimes, violent)| BlockAgg {
                crimes: crimes.into_inner(),
                violent: violent.into_inner(),
                ..BlockAgg::default()
            })
            .collect();

        // The remaining datasets are a few thousand rows: a plain loop is cheapest.
        for v in &store.vacants {
            if let Some(row) = nb.get_mut(v.neighborhood as usize) {
                row.vacants += 1;
            }
            if let Some(block) = blocks.get_mut(v.block as usize) {
                block.vacants += 1;
            }
        }
        for o in &store.outlets {
            if let Some(row) = nb.get_mut(o.neighborhood as usize) {
                row.outlets += 1;
            }
            if let Some(block) = blocks.get_mut(o.block as usize) {
                match o.kind {
                    OutletKind::Tavern => block.taverns += 1,
                    OutletKind::PackageGoods => block.package += 1,
                    OutletKind::Restaurant => block.restaurants += 1,
                    OutletKind::Other => block.other_outlets += 1,
                }
            }
        }
        for c in &store.cameras {
            if let Some(row) = nb.get_mut(c.neighborhood as usize) {
                row.cameras += 1;
            }
        }

        let mut order: Vec<NeighborhoodId> = (0..nb_len as NeighborhoodId)
            .filter(|&id| {
                let row = &nb[id as usize];
                row.crimes + row.vacants + row.requests > 0
            })
            .collect();
        order.sort_by(|&a, &b| store.neighborhood(a).cmp(store.neighborhood(b)));

        Context { store, camera_radius_m, top_n, nb, blocks, cameras, order, first_ts, last_ts, recent_cutoff }
    }

    pub fn name(&self, id: NeighborhoodId) -> &str {
        self.store.neighborhood(id)
    }

    /// "Jan 2022 – Oct 2026"
    pub fn crime_period(&self) -> String {
        format!("{} – {}", month_label(self.first_ts), month_label(self.last_ts))
    }

    pub fn recent_period(&self) -> String {
        format!("{} – {}", month_label(self.recent_cutoff), month_label(self.last_ts))
    }

    /// Neighborhood ids ranked by `key` descending; ties in name order.
    pub fn ranked(&self, key: impl Fn(&NbAgg) -> f64) -> Vec<NeighborhoodId> {
        let mut ids = self.order.clone();
        ids.sort_by(|&a, &b| key(&self.nb[b as usize]).total_cmp(&key(&self.nb[a as usize])));
        ids
    }
}

/// Neighborhoods grouped by how many open vacant-building notices they hold.
pub const TIER_LABELS: [&str; 4] = ["No vacants", "1–24 vacants", "25–99 vacants", "100+ vacants"];

pub fn vacancy_tier(vacants: u32) -> usize {
    match vacants {
        0 => 0,
        1..=24 => 1,
        25..=99 => 2,
        _ => 3,
    }
}

/// One research question for one stakeholder.
pub trait Question: Send + Sync {
    fn id(&self) -> &'static str;
    fn run(&self, ctx: &Context<'_>) -> Section;
}

/// Factory for the question set; an empty selection means "all five".
pub fn questions(selected: &[String]) -> Result<Vec<Box<dyn Question>>, AppError> {
    let all: Vec<Box<dyn Question>> = vec![
        Box::new(vacancy::VacancyAndViolence),
        Box::new(liquor::LiquorOutletsAndViolence),
        Box::new(cameras::CameraCoverage),
        Box::new(service::BlightResponse),
        Box::new(priority::PriorityIndex),
    ];
    if selected.is_empty() {
        return Ok(all);
    }
    for want in selected {
        if !all.iter().any(|q| q.id().eq_ignore_ascii_case(want)) {
            return Err(AppError::Config(format!("unknown question '{want}' (expected q1, q2, q3, q4 or q5)")));
        }
    }
    Ok(all.into_iter().filter(|q| selected.iter().any(|s| s.eq_ignore_ascii_case(q.id()))).collect())
}

#[cfg(test)]
pub mod testing {
    use super::*;
    use crate::geo;

    /// A tiny hand-built city: two neighborhoods, three blocks, one camera.
    pub fn store() -> DataStore {
        let at = |lat: f64, lon: f64| geo::project(lat, lon);
        let crime = |ts, point, block, neighborhood, offense, shooting| Crime {
            ts,
            point,
            block,
            neighborhood,
            offense,
            hour: 23,
            shooting,
        };
        let year = 365 * 86_400;
        let request = |close_secs, neighborhood, category, duplicate| ServiceRequest {
            close_secs,
            point: Point::NONE,
            block: 0,
            neighborhood,
            sr_type: 0,
            category,
            duplicate,
        };
        DataStore {
            crimes: vec![
                crime(3 * year, at(39.3000, -76.6000), 0, 0, Offense::Homicide, true), // beside the camera
                crime(3 * year, at(39.3200, -76.6000), 1, 0, Offense::Robbery, false), // 2 km from it
                crime(3 * year, at(39.3200, -76.6001), 1, 0, Offense::AggAssault, false),
                crime(year, at(39.3200, -76.6000), 1, 0, Offense::Robbery, false), // older than 12 months
                crime(3 * year, at(39.3500, -76.6500), 2, 1, Offense::Larceny, false),
            ],
            requests: vec![
                request(86_400, 0, SrCategory::Blight, false),
                request(3 * 86_400, 0, SrCategory::Blight, false),
                request(NOT_CLOSED, 0, SrCategory::Blight, false),
                request(86_400, 0, SrCategory::Blight, true),
                request(60, 1, SrCategory::Other, false),
                request(60, 1, SrCategory::Info, false),
            ],
            vacants: vec![VacantNotice { block: 1, neighborhood: 0 }; 30],
            outlets: vec![LiquorOutlet { name: "CORNER BAR".into(), block: 1, kind: OutletKind::Tavern, neighborhood: 0 }],
            cameras: vec![Camera { point: at(39.3000, -76.6000), neighborhood: 0 }],
            neighborhoods: vec!["Alpha".into(), "Beta".into()],
            blocks: vec!["100 A ST".into(), "200 B ST".into(), "300 C ST".into()],
            sr_types: vec!["SW-Dirty Alley".into()],
            block_neighborhood: vec![0, 0, 1],
            ingest: Vec::new(),
        }
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn context_joins_all_datasets_per_neighborhood_and_block() {
        let store = testing::store();
        let ctx = Context::build(&store, 150.0, 5);
        let alpha = &ctx.nb[0];
        assert_eq!((alpha.crimes, alpha.violent, alpha.shootings), (4, 4, 1));
        assert_eq!((alpha.violent_12m, alpha.uncovered_12m), (3, 2));
        assert_eq!((alpha.vacants, alpha.outlets, alpha.cameras), (30, 1, 1));
        assert_eq!((alpha.requests, alpha.blight), (3, 3));
        let beta = &ctx.nb[1];
        assert_eq!((beta.crimes, beta.violent, beta.requests, beta.blight), (1, 0, 1, 0));
        assert_eq!((ctx.blocks[1].crimes, ctx.blocks[1].violent, ctx.blocks[1].vacants, ctx.blocks[1].taverns), (3, 3, 30, 1));
        assert_eq!(ctx.order, vec![0, 1]);
        assert_eq!(vacancy_tier(alpha.vacants), 2);
    }

    #[test]
    fn every_question_produces_a_section() {
        let store = testing::store();
        let ctx = Context::build(&store, 150.0, 5);
        let all = questions(&[]).unwrap();
        assert_eq!(all.len(), 5);
        for q in &all {
            let section = q.run(&ctx);
            assert_eq!(section.id, q.id());
            assert!(!section.findings.is_empty() && !section.blocks.is_empty(), "{} is empty", q.id());
            let maps = section.blocks.iter().filter(|b| matches!(b, crate::report::Block::Map(_))).count();
            assert_eq!(maps, usize::from(matches!(q.id(), "q3" | "q5")), "{} map count", q.id());
        }
        assert_eq!(questions(&["Q3".to_string()]).unwrap().len(), 1);
        assert!(questions(&["q9".to_string()]).is_err());
    }
}
