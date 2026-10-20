//! Planar projection and a uniform-grid spatial index.
//!
//! Baltimore spans ~20 km, so an equirectangular projection around the city's
//! latitude is accurate to well under a metre and turns every distance test
//! into plain Euclidean arithmetic on `f32` metres.

use std::collections::HashMap;

use crate::model::Point;

const ORIGIN_LAT: f64 = 39.15;
const ORIGIN_LON: f64 = -76.80;
const METRES_PER_DEG_LAT: f64 = 111_320.0;
/// cos(39.29 deg) * 111 320 m: the length of one degree of longitude downtown.
const METRES_PER_DEG_LON: f64 = 86_157.0;

/// Projects WGS84 degrees to local metres; anything outside a generous city
/// bounding box (bad geocodes, the 0,0 placeholder) becomes `Point::NONE`.
pub fn project(lat: f64, lon: f64) -> Point {
    if !(39.15..=39.42).contains(&lat) || !(-76.80..=-76.45).contains(&lon) {
        return Point::NONE;
    }
    Point {
        x: ((lon - ORIGIN_LON) * METRES_PER_DEG_LON) as f32,
        y: ((lat - ORIGIN_LAT) * METRES_PER_DEG_LAT) as f32,
    }
}

pub fn project_text(lat: &str, lon: &str) -> Point {
    match (lat.parse::<f64>(), lon.parse::<f64>()) {
        (Ok(lat), Ok(lon)) => project(lat, lon),
        _ => Point::NONE,
    }
}

/// Inverse of `project`, for printing a location a person can look up.
pub fn unproject(x: f32, y: f32) -> (f64, f64) {
    (ORIGIN_LAT + f64::from(y) / METRES_PER_DEG_LAT, ORIGIN_LON + f64::from(x) / METRES_PER_DEG_LON)
}

pub type Cell = (i32, i32);

pub fn cell_of(p: Point, cell_m: f32) -> Cell {
    ((p.x / cell_m).floor() as i32, (p.y / cell_m).floor() as i32)
}

/// Points bucketed into square cells. A radius query only visits the cells the
/// radius can reach, so "is any camera within 150 m" costs a handful of hash
/// lookups instead of a scan over every camera.
pub struct SpatialGrid {
    cell_m: f32,
    cells: HashMap<Cell, Vec<Point>>,
}

impl SpatialGrid {
    pub fn build(points: impl Iterator<Item = Point>, cell_m: f32) -> Self {
        let mut cells: HashMap<Cell, Vec<Point>> = HashMap::new();
        for p in points.filter(|p| p.is_some()) {
            cells.entry(cell_of(p, cell_m)).or_default().push(p);
        }
        SpatialGrid { cell_m, cells }
    }

    /// Distance in metres to the closest indexed point no farther than `max_m`.
    pub fn nearest_within(&self, p: Point, max_m: f32) -> Option<f32> {
        if !p.is_some() {
            return None;
        }
        let reach = (max_m / self.cell_m).ceil() as i32;
        let (cx, cy) = cell_of(p, self.cell_m);
        let mut best = max_m * max_m;
        let mut found = false;
        for dx in -reach..=reach {
            for dy in -reach..=reach {
                let Some(bucket) = self.cells.get(&(cx + dx, cy + dy)) else { continue };
                for q in bucket {
                    let d2 = (q.x - p.x).powi(2) + (q.y - p.y).powi(2);
                    if d2 <= best {
                        best = d2;
                        found = true;
                    }
                }
            }
        }
        found.then(|| best.sqrt())
    }

    pub fn any_within(&self, p: Point, radius_m: f32) -> bool {
        self.nearest_within(p, radius_m).is_some()
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn projection_measures_real_distances() {
        // City Hall to the Washington Monument is roughly 1.0 km.
        let a = project(39.2909, -76.6108);
        let b = project(39.2976, -76.6157);
        let d = ((a.x - b.x).powi(2) + (a.y - b.y).powi(2)).sqrt();
        assert!((800.0..1_000.0).contains(&d), "distance was {d}");
        let (lat, lon) = unproject(a.x, a.y);
        assert!((lat - 39.2909).abs() < 1e-4 && (lon + 76.6108).abs() < 1e-4);
    }

    #[test]
    fn rejects_coordinates_outside_the_city() {
        assert!(!project(0.0, 0.0).is_some());
        assert!(!project_text("", "").is_some());
        assert!(project_text("39.29", "-76.61").is_some());
    }

    #[test]
    fn grid_finds_neighbours_across_cell_borders() {
        let grid = SpatialGrid::build([Point { x: 149.0, y: 10.0 }].into_iter(), 150.0);
        // The query point sits in the next cell over but only 2 m away.
        assert!(grid.any_within(Point { x: 151.0, y: 10.0 }, 5.0));
        assert!(!grid.any_within(Point { x: 400.0, y: 10.0 }, 150.0));
        let d = grid.nearest_within(Point { x: 449.0, y: 10.0 }, 600.0).unwrap();
        assert!((d - 300.0).abs() < 0.01);
        assert!(!grid.any_within(Point::NONE, 1_000.0));
    }
}
