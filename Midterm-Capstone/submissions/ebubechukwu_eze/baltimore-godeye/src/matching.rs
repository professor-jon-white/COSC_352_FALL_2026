//! Stage 2: cross-dataset matching.
//!
//! Every analysis needs records from different datasets attributed to the same
//! neighborhood, but only some rows arrive with one: 20% of crime rows have a
//! blank neighborhood, cameras have only coordinates, and liquor licences have
//! only a street address. Each way of recovering the neighborhood is a
//! `MatchStrategy`; a `MatchChain` tries them from most to least precise.
//! The reference tables the strategies consult are themselves learned from the
//! other datasets (e.g. 311 rows label the grid cells that crime rows fall in).

use std::collections::HashMap;

use rayon::prelude::*;

use crate::geo::{cell_of, Cell};
use crate::model::*;

/// The facts a record can offer for matching.
#[derive(Clone, Copy)]
pub struct Probe {
    pub native: NeighborhoodId,
    pub point: Point,
    pub block: BlockId,
}

pub trait MatchStrategy: Sync {
    fn name(&self) -> &'static str;
    fn resolve(&self, probe: &Probe) -> Option<NeighborhoodId>;
}

/// The record already names its neighborhood.
pub struct NativeName;

impl MatchStrategy for NativeName {
    fn name(&self) -> &'static str {
        "native name"
    }
    fn resolve(&self, probe: &Probe) -> Option<NeighborhoodId> {
        (probe.native != NO_NEIGHBORHOOD).then_some(probe.native)
    }
}

type Votes<K> = HashMap<(K, NeighborhoodId), u32>;

fn tally<K: Eq + std::hash::Hash + Send + Copy>(keys: impl ParallelIterator<Item = (K, NeighborhoodId)>) -> Votes<K> {
    keys.fold(HashMap::new, |mut votes: Votes<K>, key| {
        *votes.entry(key).or_insert(0) += 1;
        votes
    })
    .reduce(HashMap::new, |mut a, b| {
        for (key, n) in b {
            *a.entry(key).or_insert(0) += n;
        }
        a
    })
}

/// Majority neighborhood per key; ties go to the alphabetically first name so
/// the outcome never depends on thread scheduling.
fn majority<K: Eq + std::hash::Hash + Copy>(votes: Votes<K>, names: &[Box<str>]) -> HashMap<K, NeighborhoodId> {
    let mut best: HashMap<K, (u32, NeighborhoodId)> = HashMap::new();
    for ((key, id), n) in votes {
        let entry = best.entry(key).or_insert((n, id));
        if n > entry.0 || (n == entry.0 && names[id as usize] < names[entry.1 as usize]) {
            *entry = (n, id);
        }
    }
    best.into_iter().map(|(key, (_, id))| (key, id)).collect()
}

/// Point-in-area lookup without polygons: the city is cut into 100 m cells and
/// each cell is labelled with the neighborhood most of its labelled points carry.
pub struct SpatialCell {
    cells: HashMap<Cell, NeighborhoodId>,
}

impl SpatialCell {
    pub const CELL_M: f32 = 100.0;

    pub fn learn(crimes: &[Crime], requests: &[ServiceRequest], names: &[Box<str>]) -> Self {
        let labelled = |point: Point, id: NeighborhoodId| {
            (point.is_some() && id != NO_NEIGHBORHOOD).then(|| (cell_of(point, Self::CELL_M), id))
        };
        let votes = tally(
            crimes
                .par_iter()
                .filter_map(|c| labelled(c.point, c.neighborhood))
                .chain(requests.par_iter().filter_map(|r| labelled(r.point, r.neighborhood))),
        );
        SpatialCell { cells: majority(votes, names) }
    }
}

impl MatchStrategy for SpatialCell {
    fn name(&self) -> &'static str {
        "spatial cell"
    }
    fn resolve(&self, probe: &Probe) -> Option<NeighborhoodId> {
        if !probe.point.is_some() {
            return None;
        }
        let (cx, cy) = cell_of(probe.point, Self::CELL_M);
        if let Some(&id) = self.cells.get(&(cx, cy)) {
            return Some(id);
        }
        // An unlabelled cell (a park, a rail yard) borrows from its 8 neighbours,
        // nearest first, in a fixed order.
        const RING: [(i32, i32); 8] = [(0, 1), (1, 0), (0, -1), (-1, 0), (1, 1), (1, -1), (-1, -1), (-1, 1)];
        RING.iter().find_map(|(dx, dy)| self.cells.get(&(cx + dx, cy + dy)).copied())
    }
}

/// Address join: the neighborhood most records on the same hundred-block carry.
pub struct BlockAddress {
    by_block: Vec<NeighborhoodId>,
}

impl BlockAddress {
    pub fn learn(
        crimes: &[Crime],
        requests: &[ServiceRequest],
        vacants: &[VacantNotice],
        names: &[Box<str>],
        block_count: usize,
    ) -> Self {
        let labelled =
            |block: BlockId, id: NeighborhoodId| (block != NO_BLOCK && id != NO_NEIGHBORHOOD).then_some((block, id));
        let votes = tally(
            crimes
                .par_iter()
                .filter_map(|c| labelled(c.block, c.neighborhood))
                .chain(requests.par_iter().filter_map(|r| labelled(r.block, r.neighborhood)))
                .chain(vacants.par_iter().filter_map(|v| labelled(v.block, v.neighborhood))),
        );
        let mut by_block = vec![NO_NEIGHBORHOOD; block_count];
        for (block, id) in majority(votes, names) {
            by_block[block as usize] = id;
        }
        BlockAddress { by_block }
    }

    pub fn table(&self) -> &[NeighborhoodId] {
        &self.by_block
    }
}

impl MatchStrategy for BlockAddress {
    fn name(&self) -> &'static str {
        "block address"
    }
    fn resolve(&self, probe: &Probe) -> Option<NeighborhoodId> {
        let id = *self.by_block.get(probe.block as usize)?;
        (id != NO_NEIGHBORHOOD).then_some(id)
    }
}

/// How many records of one dataset each strategy resolved.
pub struct MatchStats {
    pub dataset: &'static str,
    /// One count per strategy in chain order, then the unmatched count.
    pub counts: Vec<u64>,
}

pub struct MatchChain {
    strategies: Vec<Box<dyn MatchStrategy>>,
}

impl MatchChain {
    pub fn new(strategies: Vec<Box<dyn MatchStrategy>>) -> Self {
        MatchChain { strategies }
    }

    pub fn strategy_names(&self) -> Vec<&'static str> {
        self.strategies.iter().map(|s| s.name()).collect()
    }

    /// Returns the neighborhood and the index of the strategy that found it
    /// (`strategies.len()` when none did).
    pub fn resolve(&self, probe: &Probe) -> (NeighborhoodId, usize) {
        self.strategies
            .iter()
            .enumerate()
            .find_map(|(i, s)| s.resolve(probe).map(|id| (id, i)))
            .unwrap_or((NO_NEIGHBORHOOD, self.strategies.len()))
    }

    /// Fills in the neighborhood of every record in parallel. Each rayon task
    /// mutates a disjoint slice of `records`, so no synchronisation is needed.
    pub fn apply<T: Send>(
        &self,
        dataset: &'static str,
        records: &mut [T],
        probe: impl Fn(&T) -> Probe + Sync,
        assign: impl Fn(&mut T, NeighborhoodId) + Sync,
    ) -> MatchStats {
        let slots = self.strategies.len() + 1;
        let counts = records
            .par_iter_mut()
            .fold(
                || vec![0u64; slots],
                |mut counts, record| {
                    let (id, which) = self.resolve(&probe(record));
                    assign(record, id);
                    counts[which] += 1;
                    counts
                },
            )
            .reduce(|| vec![0u64; slots], |a, b| a.iter().zip(&b).map(|(x, y)| x + y).collect());
        MatchStats { dataset, counts }
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    fn names() -> Vec<Box<str>> {
        vec!["Alpha".into(), "Beta".into()]
    }

    fn crime(x: f32, y: f32, block: BlockId, neighborhood: NeighborhoodId) -> Crime {
        Crime { ts: 0, point: Point { x, y }, block, neighborhood, offense: Offense::Other, hour: 0, shooting: false }
    }

    #[test]
    fn chain_falls_back_from_name_to_space_to_address() {
        let crimes = vec![
            crime(10.0, 10.0, 0, 0),
            crime(20.0, 20.0, 0, 0),
            crime(30.0, 30.0, 0, 1), // outvoted in the cell and on the block
            crime(5_000.0, 5_000.0, 1, 1),
        ];
        let chain = MatchChain::new(vec![
            Box::new(NativeName),
            Box::new(SpatialCell::learn(&crimes, &[], &names())),
            Box::new(BlockAddress::learn(&crimes, &[], &[], &names(), 2)),
        ]);
        let probe = |native, x: f32, block| Probe { native, point: Point { x, y: x }, block };

        assert_eq!(chain.resolve(&probe(1, 10.0, 0)), (1, 0));
        assert_eq!(chain.resolve(&probe(NO_NEIGHBORHOOD, 50.0, NO_BLOCK)), (0, 1));
        assert_eq!(chain.resolve(&probe(NO_NEIGHBORHOOD, 150.0, NO_BLOCK)), (0, 1)); // neighbour cell
        assert_eq!(chain.resolve(&probe(NO_NEIGHBORHOOD, f32::NAN, 1)), (1, 2));
        assert_eq!(chain.resolve(&probe(NO_NEIGHBORHOOD, f32::NAN, NO_BLOCK)), (NO_NEIGHBORHOOD, 3));
    }

    #[test]
    fn apply_updates_records_and_counts_each_strategy() {
        let mut crimes =
            vec![crime(10.0, 10.0, 0, 0), crime(12.0, 12.0, NO_BLOCK, NO_NEIGHBORHOOD), crime(f32::NAN, f32::NAN, NO_BLOCK, NO_NEIGHBORHOOD)];
        let spatial = SpatialCell::learn(&crimes, &[], &names());
        let chain = MatchChain::new(vec![Box::new(NativeName), Box::new(spatial)]);
        let stats = chain.apply(
            "crime",
            &mut crimes,
            |c| Probe { native: c.neighborhood, point: c.point, block: c.block },
            |c, id| c.neighborhood = id,
        );
        assert_eq!(stats.counts, vec![1, 1, 1]);
        assert_eq!(crimes[1].neighborhood, 0);
        assert_eq!(crimes[2].neighborhood, NO_NEIGHBORHOOD);
    }
}
