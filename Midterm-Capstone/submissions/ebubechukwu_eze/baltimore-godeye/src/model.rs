//! Compact, typed records shared by every pipeline stage.
//!
//! Raw rows are wide and string-heavy (a 311 row is ~400 bytes of text). Each
//! record here is a fixed-size `Copy` struct of 16-28 bytes: text that repeats
//! (neighborhood, block address, request type) is interned to an integer id,
//! timestamps are `u32` Unix seconds, and coordinates are `f32` metres in a
//! local plane. A million rows therefore live in one contiguous ~25 MB `Vec`
//! with no per-row heap allocation.

use std::collections::HashMap;
use std::sync::RwLock;

pub type NeighborhoodId = u16;
pub type BlockId = u32;
pub const NO_NEIGHBORHOOD: NeighborhoodId = NeighborhoodId::MAX;
pub const NO_BLOCK: BlockId = BlockId::MAX;

/// Position in metres east/north of a fixed origin south-west of the city.
/// NaN marks "no usable coordinates" so the struct stays 8 bytes.
#[derive(Clone, Copy, Debug)]
pub struct Point {
    pub x: f32,
    pub y: f32,
}

impl Point {
    pub const NONE: Point = Point { x: f32::NAN, y: f32::NAN };

    pub fn is_some(self) -> bool {
        !self.x.is_nan()
    }
}

#[derive(Clone, Copy, Debug, PartialEq, Eq)]
#[repr(u8)]
pub enum Offense {
    Homicide,
    Rape,
    Robbery,
    AggAssault,
    CommonAssault,
    Burglary,
    Larceny,
    AutoTheft,
    Vandalism,
    Other,
}

impl Offense {
    pub fn from_description(desc: &str) -> Offense {
        let d = desc.trim();
        if d.eq_ignore_ascii_case("HOMICIDE") {
            Offense::Homicide
        } else if d.eq_ignore_ascii_case("RAPE") {
            Offense::Rape
        } else if starts_with_ignore_case(d, "ROBBERY") {
            Offense::Robbery
        } else if starts_with_ignore_case(d, "AGG. ASSAULT") {
            Offense::AggAssault
        } else if d.eq_ignore_ascii_case("COMMON ASSAULT") {
            Offense::CommonAssault
        } else if d.eq_ignore_ascii_case("BURGLARY") {
            Offense::Burglary
        } else if d.eq_ignore_ascii_case("AUTO THEFT") {
            Offense::AutoTheft
        } else if starts_with_ignore_case(d, "LARCENY") || d.eq_ignore_ascii_case("SHOPLIFTING") {
            Offense::Larceny
        } else if d.eq_ignore_ascii_case("VANDALISM") {
            Offense::Vandalism
        } else {
            Offense::Other
        }
    }

    /// Homicide, rape, robbery and aggravated assault: the FBI violent-crime set.
    pub fn is_violent(self) -> bool {
        matches!(self, Offense::Homicide | Offense::Rape | Offense::Robbery | Offense::AggAssault)
    }
}

pub fn starts_with_ignore_case(s: &str, prefix: &str) -> bool {
    s.len() >= prefix.len() && s.as_bytes()[..prefix.len()].eq_ignore_ascii_case(prefix.as_bytes())
}

pub fn contains_ignore_case(haystack: &str, needle: &str) -> bool {
    let (h, n) = (haystack.as_bytes(), needle.as_bytes());
    n.is_empty() || h.windows(n.len()).any(|w| w.eq_ignore_ascii_case(n))
}

#[derive(Clone, Copy, Debug)]
pub struct Crime {
    pub ts: u32,
    pub point: Point,
    pub block: BlockId,
    pub neighborhood: NeighborhoodId,
    pub offense: Offense,
    /// Hour of day in US Eastern time (the portal exports UTC).
    pub hour: u8,
    pub shooting: bool,
}

#[derive(Clone, Copy, Debug, PartialEq, Eq)]
#[repr(u8)]
pub enum SrCategory {
    /// Resident-reported blight: dumping, dirty streets/alleys, rats, vacant
    /// buildings, boarding, high grass, unsanitary property.
    Blight,
    /// Call-centre information requests with no field work.
    Info,
    /// Work orders the city opened itself (e.g. proactive rat abatement).
    Proactive,
    Other,
}

impl SrCategory {
    pub fn from_type(sr_type: &str) -> SrCategory {
        const BLIGHT: [&str; 12] = [
            "Sanitation Property",
            "Dirty Alley",
            "Dirty Street",
            "Illegal Dumping",
            "HCD-Vacant Building",
            "SW-Cleaning",
            "SW-Boarding",
            "Rat Rubout",
            "HCD-Rodents",
            "SW-HGW",
            "Maintenance Structure",
            "Graffiti",
        ];
        if contains_ignore_case(sr_type, "Proactive")
            || contains_ignore_case(sr_type, "USE ONLY")
            || contains_ignore_case(sr_type, "creation only")
        {
            SrCategory::Proactive
        } else if starts_with_ignore_case(sr_type, "ECC-") {
            SrCategory::Info
        } else if BLIGHT.iter().any(|k| contains_ignore_case(sr_type, k)) {
            SrCategory::Blight
        } else {
            SrCategory::Other
        }
    }
}

pub const NOT_CLOSED: u32 = u32::MAX;

#[derive(Clone, Copy, Debug)]
pub struct ServiceRequest {
    /// Seconds from creation to closure, or `NOT_CLOSED`.
    pub close_secs: u32,
    pub point: Point,
    pub block: BlockId,
    pub neighborhood: NeighborhoodId,
    pub sr_type: u16,
    pub category: SrCategory,
    pub duplicate: bool,
}

#[derive(Clone, Copy, Debug)]
pub struct VacantNotice {
    pub block: BlockId,
    pub neighborhood: NeighborhoodId,
}

#[derive(Clone, Copy, Debug, PartialEq, Eq)]
#[repr(u8)]
pub enum OutletKind {
    Tavern,
    PackageGoods,
    Restaurant,
    Other,
}

impl OutletKind {
    pub fn from_description(desc: &str) -> OutletKind {
        if starts_with_ignore_case(desc, "Tavern") || starts_with_ignore_case(desc, "Adult") {
            OutletKind::Tavern
        } else if starts_with_ignore_case(desc, "Package") {
            OutletKind::PackageGoods
        } else if starts_with_ignore_case(desc, "Restaurant") {
            OutletKind::Restaurant
        } else {
            OutletKind::Other
        }
    }
}

/// One row of the licence history (one per licensee per licence year).
#[derive(Clone, Debug)]
pub struct LicenseRow {
    pub name: Box<str>,
    pub block: BlockId,
    pub year: u16,
    pub kind: OutletKind,
    pub closed: bool,
}

/// A currently licensed establishment, after de-duplicating the history.
#[derive(Clone, Debug)]
pub struct LiquorOutlet {
    pub name: Box<str>,
    pub block: BlockId,
    pub kind: OutletKind,
    pub neighborhood: NeighborhoodId,
}

#[derive(Clone, Copy, Debug)]
pub struct Camera {
    pub point: Point,
    pub neighborhood: NeighborhoodId,
}

/// Thread-safe string interner: maps a normalised key to a dense integer id.
///
/// Parser threads share one interner per vocabulary. Lookups of keys that
/// already exist (the overwhelmingly common case once the first chunks are
/// parsed) take only the read lock and allocate nothing; the write lock is
/// taken only to insert a new key. A key can also carry a human-readable
/// label (e.g. "Belair-Edison" for key "BELAIR-EDISON").
pub struct Interner {
    inner: RwLock<Inner>,
}

#[derive(Default)]
struct Inner {
    ids: HashMap<Box<str>, u32>,
    labels: Vec<Box<str>>,
    labelled: Vec<bool>,
}

impl Interner {
    pub fn new() -> Self {
        Interner { inner: RwLock::new(Inner::default()) }
    }

    pub fn intern(&self, key: &str) -> u32 {
        self.intern_labelled(key, None)
    }

    pub fn intern_labelled(&self, key: &str, label: Option<&str>) -> u32 {
        {
            let guard = self.inner.read().unwrap_or_else(|e| e.into_inner());
            if let Some(&id) = guard.ids.get(key) {
                // When several spellings of a label race, the smallest wins,
                // so the result does not depend on thread scheduling.
                let keep = match label {
                    None => true,
                    Some(l) => guard.labelled[id as usize] && &*guard.labels[id as usize] <= l,
                };
                if keep {
                    return id;
                }
            }
        }
        let mut guard = self.inner.write().unwrap_or_else(|e| e.into_inner());
        if let Some(&id) = guard.ids.get(key) {
            if let Some(l) = label {
                let i = id as usize;
                if !guard.labelled[i] || l < &*guard.labels[i] {
                    guard.labels[i] = l.into();
                    guard.labelled[i] = true;
                }
            }
            return id;
        }
        let id = guard.labels.len() as u32;
        guard.ids.insert(key.into(), id);
        guard.labels.push(label.unwrap_or(key).into());
        guard.labelled.push(label.is_some());
        id
    }

    /// Consumes the interner once ingestion is done; later stages index the
    /// returned vector directly with no locking.
    pub fn into_labels(self) -> Vec<Box<str>> {
        self.inner.into_inner().unwrap_or_else(|e| e.into_inner()).labels
    }
}

/// How one input file was ingested, for the report's data inventory.
#[derive(Clone, Debug)]
pub struct IngestStats {
    pub dataset: &'static str,
    pub file: String,
    pub format: &'static str,
    pub bytes: u64,
    pub rows: u64,
    pub kept: u64,
    pub parser_threads: usize,
    pub seconds: f64,
}

/// Everything the analysis stage reads. Immutable after the matching stage,
/// so it is shared across threads by plain `&` reference with no locks.
pub struct DataStore {
    pub crimes: Vec<Crime>,
    pub requests: Vec<ServiceRequest>,
    pub vacants: Vec<VacantNotice>,
    pub outlets: Vec<LiquorOutlet>,
    pub cameras: Vec<Camera>,
    pub neighborhoods: Vec<Box<str>>,
    pub blocks: Vec<Box<str>>,
    pub sr_types: Vec<Box<str>>,
    /// Majority neighborhood of each hundred-block (`NO_NEIGHBORHOOD` if unknown).
    pub block_neighborhood: Vec<NeighborhoodId>,
    pub ingest: Vec<IngestStats>,
}

impl DataStore {
    pub fn neighborhood(&self, id: NeighborhoodId) -> &str {
        self.neighborhoods.get(id as usize).map_or("(unknown)", |s| s)
    }

    pub fn block(&self, id: BlockId) -> &str {
        self.blocks.get(id as usize).map_or("(unknown)", |s| s)
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn interner_is_stable_and_prefers_labels() {
        let i = Interner::new();
        let a = i.intern("CANTON");
        let b = i.intern_labelled("CANTON", Some("Canton"));
        let c = i.intern("DOWNTOWN");
        assert_eq!(a, b);
        assert_ne!(a, c);
        let labels = i.into_labels();
        assert_eq!(&*labels[a as usize], "Canton");
        assert_eq!(&*labels[c as usize], "DOWNTOWN");
    }

    #[test]
    fn interner_is_safe_across_threads() {
        let i = Interner::new();
        std::thread::scope(|s| {
            for _ in 0..8 {
                s.spawn(|| {
                    for n in 0..500 {
                        i.intern(&format!("KEY{}", n % 50));
                    }
                });
            }
        });
        assert_eq!(i.into_labels().len(), 50);
    }

    #[test]
    fn classifies_offenses_and_requests() {
        assert!(Offense::from_description("ROBBERY - CARJACKING").is_violent());
        assert!(Offense::from_description("AGG. ASSAULT").is_violent());
        assert!(!Offense::from_description("LARCENY FROM AUTO").is_violent());
        assert_eq!(SrCategory::from_type("SW-Dirty Alley"), SrCategory::Blight);
        assert_eq!(SrCategory::from_type("SW-Rat Rubout Proactive"), SrCategory::Proactive);
        assert_eq!(SrCategory::from_type("ECC-Information Request"), SrCategory::Info);
        assert_eq!(SrCategory::from_type("TRM-Potholes"), SrCategory::Other);
    }
}
