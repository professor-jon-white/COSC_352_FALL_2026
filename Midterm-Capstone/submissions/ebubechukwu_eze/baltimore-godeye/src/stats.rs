//! Small statistics helpers used by the analyses.

pub fn mean(values: &[f64]) -> f64 {
    if values.is_empty() {
        0.0
    } else {
        values.iter().sum::<f64>() / values.len() as f64
    }
}

/// Pearson correlation coefficient; 0 when either series is constant.
pub fn pearson(x: &[f64], y: &[f64]) -> f64 {
    let n = x.len().min(y.len());
    if n < 2 {
        return 0.0;
    }
    let (mx, my) = (mean(&x[..n]), mean(&y[..n]));
    let (mut sxy, mut sxx, mut syy) = (0.0, 0.0, 0.0);
    for i in 0..n {
        let (dx, dy) = (x[i] - mx, y[i] - my);
        sxy += dx * dy;
        sxx += dx * dx;
        syy += dy * dy;
    }
    if sxx == 0.0 || syy == 0.0 {
        0.0
    } else {
        sxy / (sxx * syy).sqrt()
    }
}

/// Ranks with ties sharing their average rank.
fn ranks(values: &[f64]) -> Vec<f64> {
    let mut order: Vec<usize> = (0..values.len()).collect();
    order.sort_by(|&a, &b| values[a].total_cmp(&values[b]));
    let mut out = vec![0.0; values.len()];
    let mut i = 0;
    while i < order.len() {
        let mut j = i;
        while j + 1 < order.len() && values[order[j + 1]] == values[order[i]] {
            j += 1;
        }
        let rank = (i + j) as f64 / 2.0 + 1.0;
        for &k in &order[i..=j] {
            out[k] = rank;
        }
        i = j + 1;
    }
    out
}

/// Spearman rank correlation: robust to the heavy right tail of count data.
pub fn spearman(x: &[f64], y: &[f64]) -> f64 {
    pearson(&ranks(x), &ranks(y))
}

/// Percentile (0-100) of an ascending-sorted slice, by nearest rank.
pub fn percentile<T: Copy + Default>(sorted: &[T], pct: f64) -> T {
    if sorted.is_empty() {
        return T::default();
    }
    let rank = (pct / 100.0 * sorted.len() as f64).ceil() as usize;
    sorted[rank.clamp(1, sorted.len()) - 1]
}

/// Rescales values to 0-100 between their minimum and maximum.
pub fn min_max_scale(values: &[f64]) -> Vec<f64> {
    let lo = values.iter().copied().fold(f64::INFINITY, f64::min);
    let hi = values.iter().copied().fold(f64::NEG_INFINITY, f64::max);
    values.iter().map(|v| if hi > lo { (v - lo) / (hi - lo) * 100.0 } else { 0.0 }).collect()
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn correlations() {
        let x = [1.0, 2.0, 3.0, 4.0];
        assert!((pearson(&x, &[2.0, 4.0, 6.0, 8.0]) - 1.0).abs() < 1e-12);
        assert!((pearson(&x, &[8.0, 6.0, 4.0, 2.0]) + 1.0).abs() < 1e-12);
        assert_eq!(pearson(&x, &[5.0, 5.0, 5.0, 5.0]), 0.0);
        // Monotonic but not linear: Spearman is exactly 1, Pearson is not.
        let y = [1.0, 10.0, 100.0, 1000.0];
        assert!((spearman(&x, &y) - 1.0).abs() < 1e-12);
        assert!(pearson(&x, &y) < 0.95);
        assert_eq!(ranks(&[10.0, 20.0, 10.0]), vec![1.5, 3.0, 1.5]);
    }

    #[test]
    fn percentiles_and_scaling() {
        let v = [1, 2, 3, 4, 5, 6, 7, 8, 9, 10];
        assert_eq!(percentile(&v, 50.0), 5);
        assert_eq!(percentile(&v, 90.0), 9);
        assert_eq!(percentile(&v, 100.0), 10);
        assert_eq!(percentile::<u32>(&[], 50.0), 0);
        assert_eq!(min_max_scale(&[2.0, 4.0, 6.0]), vec![0.0, 50.0, 100.0]);
    }
}
