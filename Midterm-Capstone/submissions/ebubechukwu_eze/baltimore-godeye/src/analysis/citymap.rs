//! The grid behind the city maps in Q3 and Q5.
//!
//! Baltimore City is cut into a fixed 400 m grid (40 x 49 cells). The terminal
//! draws each cell two characters wide, which makes it close to square on
//! screen because a terminal glyph is about twice as tall as it is wide. The
//! city's outline is not stored anywhere: a cell is "in the city" when the
//! geocoded crimes and 311 requests in it carry a neighborhood, so the shape
//! emerges from the data itself.

use rayon::prelude::*;

use crate::model::*;
use crate::report::{int, MapGrid, MapMarker};
use crate::stats::percentile;

pub const CELL_M: f32 = 400.0;
pub const COLS: usize = 40;
pub const ROWS: usize = 49;
/// South-west corner of the frame in projected metres (about 39.197 N, 76.713 W).
const X0: f32 = 7_500.0;
const Y0: f32 = 5_200.0;

/// Row-major cell index (row 0 = north edge) of a point inside the frame.
pub fn locate(p: Point) -> Option<usize> {
    if !p.is_some() {
        return None;
    }
    let col = ((p.x - X0) / CELL_M).floor();
    let row_from_south = ((p.y - Y0) / CELL_M).floor();
    if !(0.0..COLS as f32).contains(&col) || !(0.0..ROWS as f32).contains(&row_from_south) {
        return None;
    }
    Some((ROWS - 1 - row_from_south as usize) * COLS + col as usize)
}

/// Counts the points of `items` per cell, in parallel: each task fills a
/// private 1,960-entry table and the tables are summed pairwise.
pub fn count_cells<T: Sync>(items: &[T], point: impl Fn(&T) -> Option<Point> + Sync) -> Vec<u32> {
    let empty = || vec![0u32; COLS * ROWS];
    items
        .par_chunks(16_384)
        .map(|chunk| {
            let mut table = empty();
            for cell in chunk.iter().filter_map(&point).filter_map(locate) {
                table[cell] += 1;
            }
            table
        })
        .reduce(empty, |mut a, b| {
            for (x, y) in a.iter_mut().zip(b) {
                *x += y;
            }
            a
        })
}

/// Squares inside the city: those whose records carry a neighborhood. Stray
/// geocodes just over the city line have none, so they fall outside.
pub fn city_mask(owner: &[NeighborhoodId]) -> Vec<bool> {
    owner.iter().map(|&id| id != NO_NEIGHBORHOOD).collect()
}

/// Shades each in-city cell 0 (nothing) to 4 by the quartiles of the non-zero
/// counts, and returns the legend text for each level.
pub fn shade_counts(counts: &[u32], mask: &[bool], unit: &str) -> (Vec<Option<u8>>, Vec<String>) {
    let mut nonzero: Vec<u32> = counts.iter().zip(mask).filter(|(&n, &m)| m && n > 0).map(|(&n, _)| n).collect();
    nonzero.sort_unstable();
    let cuts = [percentile(&nonzero, 25.0), percentile(&nonzero, 50.0), percentile(&nonzero, 75.0)];
    let cells = counts
        .iter()
        .zip(mask)
        .map(|(&n, &inside)| {
            inside.then(|| match n {
                0 => 0,
                n if n <= cuts[0] => 1,
                n if n <= cuts[1] => 2,
                n if n <= cuts[2] => 3,
                _ => 4,
            })
        })
        .collect();
    let max = nonzero.last().copied().unwrap_or(0);
    let range = |lo: u32, hi: u32| if lo >= hi { format!("{} {unit}", int(hi.into())) } else { format!("{}–{} {unit}", int(lo.into()), int(hi.into())) };
    let levels = vec![
        format!("no {unit}"),
        range(1, cuts[0]),
        range(cuts[0] + 1, cuts[1]),
        range(cuts[1] + 1, cuts[2]),
        range(cuts[2] + 1, max),
    ];
    (cells, levels)
}

/// Majority neighborhood of each cell, voted by every matched crime and 311
/// request in it. Each vote is packed into one `u32` (cell, neighborhood) key
/// and the keys are sorted in parallel, so equal votes sit in runs and are
/// counted in one pass with no per-thread tables. Ties go to the
/// alphabetically first name, so the result never depends on scheduling.
pub fn cell_neighborhoods(store: &DataStore) -> Vec<NeighborhoodId> {
    let nb_count = store.neighborhoods.len().max(1) as u32;
    let key = |p: Point, id: NeighborhoodId| {
        let cell = locate(p)?;
        (id != NO_NEIGHBORHOOD).then(|| cell as u32 * nb_count + u32::from(id))
    };
    let mut keys: Vec<u32> = store
        .crimes
        .par_iter()
        .filter_map(|c| key(c.point, c.neighborhood))
        .chain(store.requests.par_iter().filter_map(|r| key(r.point, r.neighborhood)))
        .collect();
    keys.par_sort_unstable();

    let mut owner = vec![NO_NEIGHBORHOOD; COLS * ROWS];
    let mut votes = vec![0usize; COLS * ROWS];
    for run in keys.chunk_by(|a, b| a == b) {
        let cell = (run[0] / nb_count) as usize;
        let id = (run[0] % nb_count) as NeighborhoodId;
        if run.len() > votes[cell] || (run.len() == votes[cell] && store.neighborhood(id) < store.neighborhood(owner[cell])) {
            votes[cell] = run.len();
            owner[cell] = id;
        }
    }
    owner
}

/// Builds a marker at a row-major cell index.
pub fn marker(cell: usize, tag: impl Into<String>, label: impl Into<String>) -> MapMarker {
    MapMarker { col: cell % COLS, row: cell / COLS, tag: tag.into(), label: label.into() }
}

pub fn grid(title: String, cells: Vec<Option<u8>>, levels: Vec<String>, markers: Vec<MapMarker>, note: String) -> MapGrid {
    debug_assert_eq!(levels.len(), MapGrid::LEVELS);
    MapGrid { title, cols: COLS, rows: ROWS, cells, levels, markers, note }
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::geo::project;

    #[test]
    fn frame_covers_the_city_with_north_up() {
        let city_hall = locate(project(39.2909, -76.6108)).expect("City Hall is on the map");
        let north = locate(project(39.3700, -76.6108)).expect("the northern city line is on the map");
        assert!(north < city_hall, "north should be drawn above downtown");
        assert_eq!(north % COLS, city_hall % COLS);
        assert!(locate(project(39.41, -76.61)).is_none());
        assert!(locate(Point::NONE).is_none());
    }

    #[test]
    fn shading_uses_quartiles_of_non_empty_cells() {
        let counts = vec![0, 1, 2, 3, 4, 100];
        let mask = vec![true, true, true, true, true, false];
        let (cells, levels) = shade_counts(&counts, &mask, "crimes");
        assert_eq!(cells, vec![Some(0), Some(1), Some(2), Some(3), Some(4), None]);
        assert_eq!(levels[0], "no crimes");
        assert_eq!(levels[4], "4 crimes");
    }
}
