//! Shared concurrency helpers. Rayon uses a bounded worker pool and parallel
//! iterators while preserving deterministic result ordering where applicable.
use rayon::prelude::*;

pub fn parallel_map<'a, T: Sync + 'a, U: Send + 'a, F: Fn(&'a T) -> U + Sync + Send>(
    items: &'a [T],
    f: F,
) -> Vec<U> {
    items.par_iter().map(f).collect()
}
