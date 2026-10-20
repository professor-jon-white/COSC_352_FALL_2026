//! Q3: how much violent crime happens out of sight of CitiWatch cameras?

use rayon::prelude::*;

use super::{citymap, Context, Question};
use crate::geo::{cell_of, unproject, Cell};
use crate::model::*;
use crate::report::{int, pct, BarChart, Block, Section, Table};

pub struct CameraCoverage;

const BAND_EDGES: [f32; 4] = [75.0, 150.0, 300.0, 600.0];
const HOT_CELL_M: f32 = 250.0;

#[derive(Clone, Copy, Default)]
struct Coverage {
    bands: [u64; 5],
    violent: [u64; 2],
    shootings: [u64; 2],
    homicides: [u64; 2],
}

impl Coverage {
    fn merge(mut self, o: Coverage) -> Coverage {
        for (a, b) in self.bands.iter_mut().zip(o.bands) {
            *a += b;
        }
        for i in 0..2 {
            self.violent[i] += o.violent[i];
            self.shootings[i] += o.shootings[i];
            self.homicides[i] += o.homicides[i];
        }
        self
    }
}

/// Most frequent value; ties go to the smallest, for determinism.
fn mode<T: Ord + Copy>(values: &mut [T]) -> Option<T> {
    values.sort_unstable();
    let mut best: Option<(usize, T)> = None;
    for run in values.chunk_by(|a, b| a == b) {
        if best.is_none_or(|(n, _)| run.len() > n) {
            best = Some((run.len(), run[0]));
        }
    }
    best.map(|(_, v)| v)
}

impl Question for CameraCoverage {
    fn id(&self) -> &'static str {
        "q3"
    }

    fn run(&self, ctx: &Context<'_>) -> Section {
        let store = ctx.store;
        let radius = ctx.camera_radius_m;

        // Index 0 = within the radius of a camera, 1 = outside it.
        let cov = store
            .crimes
            .par_iter()
            .filter(|c| c.point.is_some() && (c.offense.is_violent() || c.shooting))
            .fold(Coverage::default, |mut cov, c| {
                let nearest = ctx.cameras.nearest_within(c.point, 600.0_f32.max(radius));
                let outside = usize::from(nearest.is_none_or(|d| d > radius));
                if c.offense.is_violent() {
                    let band = nearest.map_or(4, |d| BAND_EDGES.iter().position(|&edge| d <= edge).unwrap_or(4));
                    cov.bands[band] += 1;
                    cov.violent[outside] += 1;
                }
                cov.shootings[outside] += u64::from(c.shooting);
                cov.homicides[outside] += u64::from(c.offense == Offense::Homicide);
                cov
            })
            .reduce(Coverage::default, Coverage::merge);

        let total_violent: u64 = cov.bands.iter().sum();
        let mut bands = BarChart::new(format!("Distance from each violent crime to the nearest camera ({})", ctx.crime_period()));
        let labels = ["0–75 m", "75–150 m", "150–300 m", "300–600 m", "over 600 m"];
        for (label, &n) in labels.iter().zip(&cov.bands) {
            bands.bar(*label, n as f64, format!("{} ({})", int(n), pct(n as f64, total_violent as f64)));
        }
        let share = |pair: [u64; 2]| pct(pair[0] as f64, (pair[0] + pair[1]) as f64);
        let facts = Block::Facts {
            title: format!("Share of incidents within {radius:.0} m of a camera"),
            items: vec![
                ("Cameras in the network".into(), int(store.cameras.len() as u64)),
                ("Violent crimes".into(), format!("{} of {}", share(cov.violent), int(cov.violent[0] + cov.violent[1]))),
                ("Shootings".into(), format!("{} of {}", share(cov.shootings), int(cov.shootings[0] + cov.shootings[1]))),
                ("Homicides".into(), format!("{} of {}", share(cov.homicides), int(cov.homicides[0] + cov.homicides[1]))),
            ],
        };

        let ranked = ctx.ranked(|r| f64::from(r.uncovered_12m));
        let mut gaps = Table::new(
            format!("Top {} neighborhoods by violent crime outside camera range ({})", ctx.top_n, ctx.recent_period()),
            &["Neighborhood", "#Cameras", "#Violent crimes", "#Outside range", "#Share outside", "#Shootings"],
        );
        for &id in ranked.iter().take(ctx.top_n) {
            let row = &ctx.nb[id as usize];
            gaps.row(vec![
                ctx.name(id).to_string(),
                int(row.cameras.into()),
                int(row.violent_12m.into()),
                int(row.uncovered_12m.into()),
                pct(row.uncovered_12m.into(), row.violent_12m.into()),
                int(row.shootings_12m.into()),
            ]);
        }

        // Candidate sites: 250 m cells with the most recent out-of-range violence.
        let mut uncovered: Vec<(Cell, &Crime)> = store
            .crimes
            .par_iter()
            .filter(|c| {
                c.ts >= ctx.recent_cutoff
                    && c.offense.is_violent()
                    && c.point.is_some()
                    && !ctx.cameras.any_within(c.point, radius)
            })
            .map(|c| (cell_of(c.point, HOT_CELL_M), c))
            .collect();
        uncovered.sort_unstable_by_key(|(cell, c)| (*cell, c.block, c.ts));
        let mut cells: Vec<(usize, u64, Cell, BlockId, NeighborhoodId)> = uncovered
            .chunk_by(|a, b| a.0 == b.0)
            .map(|group| {
                let shootings = group.iter().filter(|(_, c)| c.shooting).count() as u64;
                let mut blocks: Vec<BlockId> = group.iter().map(|(_, c)| c.block).filter(|&b| b != NO_BLOCK).collect();
                let mut nbs: Vec<NeighborhoodId> = group.iter().map(|(_, c)| c.neighborhood).collect();
                (group.len(), shootings, group[0].0, mode(&mut blocks).unwrap_or(NO_BLOCK), mode(&mut nbs).unwrap_or(NO_NEIGHBORHOOD))
            })
            .collect();
        cells.sort_by(|a, b| (b.0, b.1).cmp(&(a.0, a.1)).then(a.2.cmp(&b.2)));
        let mut sites = Table::new(
            format!("Candidate camera sites: {HOT_CELL_M:.0} m cells with the most out-of-range violent crime ({})", ctx.recent_period()),
            &["#Rank", "Busiest block in cell", "Neighborhood", "Cell centre (lat, lon)", "#Violent crimes", "#Shootings"],
        );
        for (rank, (count, shootings, cell, block, nb)) in cells.iter().take(ctx.top_n).enumerate() {
            let (lat, lon) = unproject((cell.0 as f32 + 0.5) * HOT_CELL_M, (cell.1 as f32 + 0.5) * HOT_CELL_M);
            sites.row(vec![
                (rank + 1).to_string(),
                store.block(*block).to_string(),
                store.neighborhood(*nb).to_string(),
                format!("{lat:.4}, {lon:.4}"),
                int(*count as u64),
                int(*shootings),
            ]);
        }

        // City map: out-of-range violent crime per 400 m cell, sites numbered on top.
        let mask = citymap::city_mask(&citymap::cell_neighborhoods(store));
        let counts = citymap::count_cells(&uncovered, |(_, c)| Some(c.point));
        let (shades, levels) = citymap::shade_counts(&counts, &mask, "violent crimes");
        // Sites closer than 400 m can share a square; the higher rank is drawn
        // and the legend still lists every site.
        let markers = cells
            .iter()
            .take(ctx.top_n.min(9))
            .enumerate()
            .filter_map(|(rank, (_, _, cell, block, _))| {
                let centre = Point { x: (cell.0 as f32 + 0.5) * HOT_CELL_M, y: (cell.1 as f32 + 0.5) * HOT_CELL_M };
                citymap::locate(centre).map(|at| (at, rank + 1, store.block(*block)))
            })
            .collect::<Vec<_>>();
        let markers = markers
            .iter()
            .map(|&(at, rank, block)| {
                let shared = markers.iter().find(|m| m.0 == at && m.1 < rank).map(|m| format!(" (drawn as {})", m.1));
                citymap::marker(at, rank.to_string(), format!("Site {rank}: {block}{}", shared.unwrap_or_default()))
            })
            .collect();
        let map = citymap::grid(
            format!("Map: violent crime more than {radius:.0} m from any camera, per 400 m square ({})", ctx.recent_period()),
            shades,
            levels,
            markers,
            "North is up; each square is 400 m across. Numbers are the candidate camera sites ranked in the table below.".into(),
        );

        let blind: Vec<&str> = ranked
            .iter()
            .filter(|&&id| ctx.nb[id as usize].cameras == 0 && ctx.nb[id as usize].violent_12m >= 50)
            .map(|&id| ctx.name(id))
            .collect();
        let top_cells: u64 = cells.iter().take(ctx.top_n).map(|c| c.0 as u64).sum();
        let mut findings = vec![
            format!(
                "{} of violent crime, {} of shootings and {} of homicides since {} happened within {radius:.0} m of a CitiWatch camera; the rest was out of range.",
                share(cov.violent),
                share(cov.shootings),
                share(cov.homicides),
                crate::normalize::month_label(ctx.first_ts)
            ),
            format!(
                "{} of violent crimes were more than 600 m from any camera.",
                pct(cov.bands[4] as f64, total_violent as f64)
            ),
        ];
        if !blind.is_empty() {
            let shown: Vec<&str> = blind.iter().take(6).copied().collect();
            findings.push(format!(
                "{} neighborhoods had 50+ violent crimes in the last 12 months and no camera at all, led by {}.",
                blind.len(),
                shown.join(", ")
            ));
        }
        findings.push(format!(
            "Suggested action: the {} candidate cells below saw {} out-of-range violent crimes in 12 months; one camera per cell would bring them into view.",
            cells.len().min(ctx.top_n),
            int(top_cells)
        ));

        Section {
            id: self.id(),
            title: "CitiWatch camera coverage gaps".into(),
            stakeholder: "Baltimore Police Department and Mayor's Office of Neighborhood Safety and Engagement".into(),
            question: "How much violent crime happens beyond the reach of the camera network, and where would new cameras see the most?".into(),
            findings,
            blocks: vec![facts, Block::Bars(bands), Block::Map(map), Block::Table(gaps), Block::Table(sites)],
            method: format!(
                "Spatial join: camera points (GeoJSON) are indexed in a uniform grid and each geocoded crime is tested against the cameras in the surrounding cells. \
                 'In range' means within {radius:.0} m (set with --camera-radius); real sight lines depend on buildings and camera type. \
                 Neighborhood and candidate-site tables use the latest 12 months of crime."
            ),
        }
    }
}
