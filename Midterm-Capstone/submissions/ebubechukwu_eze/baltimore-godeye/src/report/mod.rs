//! The report model and its renderers.
//!
//! Analyses produce a format-neutral `Report` (facts, tables, bar charts).
//! Each `Renderer` turns the same model into one output format, so adding a
//! format never touches analysis code.

pub mod html;
pub mod terminal;

use serde::Serialize;

#[derive(Serialize)]
pub struct Report {
    pub title: String,
    pub subtitle: String,
    pub overview: Vec<Block>,
    pub sections: Vec<Section>,
    /// (stage, seconds)
    pub timings: Vec<(String, f64)>,
}

#[derive(Serialize)]
pub struct Section {
    pub id: &'static str,
    pub title: String,
    pub stakeholder: String,
    pub question: String,
    pub findings: Vec<String>,
    pub blocks: Vec<Block>,
    pub method: String,
}

#[derive(Serialize)]
#[serde(tag = "type", rename_all = "lowercase")]
pub enum Block {
    Facts { title: String, items: Vec<(String, String)> },
    Table(Table),
    Bars(BarChart),
    Map(MapGrid),
}

/// A shaded grid map of the city with numbered markers on top.
#[derive(Serialize)]
pub struct MapGrid {
    pub title: String,
    pub cols: usize,
    pub rows: usize,
    /// Row-major, row 0 at the north edge. `None` is outside the city;
    /// `Some(0..=4)` is a shade level from none to highest.
    pub cells: Vec<Option<u8>>,
    /// Meaning of each shade level, indexed by level.
    pub levels: Vec<String>,
    pub markers: Vec<MapMarker>,
    pub note: String,
}

#[derive(Serialize)]
pub struct MapMarker {
    pub col: usize,
    pub row: usize,
    /// At most two characters, drawn over the cell.
    pub tag: String,
    pub label: String,
}

impl MapGrid {
    pub const LEVELS: usize = 5;

    /// Legend entries for the shade levels that at least one square uses.
    pub fn used_levels(&self) -> impl Iterator<Item = (usize, &str)> {
        self.levels
            .iter()
            .enumerate()
            .filter(|(level, _)| self.cells.iter().any(|c| c.map(usize::from) == Some(*level)))
            .map(|(level, label)| (level, label.as_str()))
    }
}

#[derive(Serialize)]
pub struct Table {
    pub title: String,
    pub headers: Vec<String>,
    /// Right-align (numeric) flag per column.
    pub numeric: Vec<bool>,
    pub rows: Vec<Vec<String>>,
}

impl Table {
    /// Headers starting with '#' are numeric (right-aligned) columns.
    pub fn new(title: impl Into<String>, headers: &[&str]) -> Self {
        Table {
            title: title.into(),
            headers: headers.iter().map(|h| h.trim_start_matches('#').to_string()).collect(),
            numeric: headers.iter().map(|h| h.starts_with('#')).collect(),
            rows: Vec::new(),
        }
    }

    pub fn row(&mut self, cells: Vec<String>) {
        debug_assert_eq!(cells.len(), self.headers.len());
        self.rows.push(cells);
    }
}

#[derive(Serialize)]
pub struct BarChart {
    pub title: String,
    pub bars: Vec<Bar>,
}

#[derive(Serialize)]
pub struct Bar {
    pub label: String,
    pub value: f64,
    pub display: String,
}

impl BarChart {
    pub fn new(title: impl Into<String>) -> Self {
        BarChart { title: title.into(), bars: Vec::new() }
    }

    pub fn bar(&mut self, label: impl Into<String>, value: f64, display: impl Into<String>) {
        self.bars.push(Bar { label: label.into(), value, display: display.into() });
    }
}

pub trait Renderer {
    fn extension(&self) -> &'static str;
    fn render(&self, report: &Report) -> String;
}

#[derive(Clone, Copy, Debug, PartialEq, Eq)]
pub enum Format {
    Terminal { color: bool },
    Html,
    Json,
}

/// Factory: the caller names a format and gets back something that renders it.
pub fn renderer(format: Format) -> Box<dyn Renderer> {
    match format {
        Format::Terminal { color } => Box::new(terminal::TerminalRenderer { color }),
        Format::Html => Box::new(html::HtmlRenderer),
        Format::Json => Box::new(JsonRenderer),
    }
}

struct JsonRenderer;

impl Renderer for JsonRenderer {
    fn extension(&self) -> &'static str {
        "json"
    }
    fn render(&self, report: &Report) -> String {
        serde_json::to_string_pretty(report).unwrap_or_else(|e| format!("{{\"error\": \"{e}\"}}"))
    }
}

/// 1234567 -> "1,234,567"
pub fn int(n: u64) -> String {
    let digits = n.to_string();
    let mut out = String::with_capacity(digits.len() + digits.len() / 3);
    for (i, c) in digits.chars().enumerate() {
        if i > 0 && (digits.len() - i) % 3 == 0 {
            out.push(',');
        }
        out.push(c);
    }
    out
}

pub fn dec(value: f64, places: usize) -> String {
    format!("{value:.places$}")
}

/// "42.5%" for part/whole, or "n/a" when the whole is zero.
pub fn pct(part: f64, whole: f64) -> String {
    if whole > 0.0 {
        format!("{:.1}%", part / whole * 100.0)
    } else {
        "n/a".into()
    }
}

/// "2.3x" ratio, or "n/a" when the denominator is zero.
pub fn times(a: f64, b: f64) -> String {
    if b > 0.0 {
        format!("{:.1}x", a / b)
    } else {
        "n/a".into()
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn number_formatting() {
        assert_eq!(int(0), "0");
        assert_eq!(int(999), "999");
        assert_eq!(int(1_000), "1,000");
        assert_eq!(int(1_234_567), "1,234,567");
        assert_eq!(pct(1.0, 8.0), "12.5%");
        assert_eq!(pct(1.0, 0.0), "n/a");
        assert_eq!(times(3.0, 2.0), "1.5x");
    }

    #[test]
    fn every_format_renders_the_same_report() {
        let mut table = Table::new("T", &["Name", "#Count"]);
        table.row(vec!["Canton <b>".into(), "12".into()]);
        let mut bars = BarChart::new("B");
        bars.bar("Canton", 12.0, "12");
        let report = Report {
            title: "Title".into(),
            subtitle: "Sub".into(),
            overview: vec![Block::Facts { title: "F".into(), items: vec![("k".into(), "v".into())] }],
            sections: vec![Section {
                id: "q1",
                title: "Section".into(),
                stakeholder: "Mayor".into(),
                question: "Why?".into(),
                findings: vec!["Because.".into()],
                blocks: vec![
                    Block::Table(table),
                    Block::Bars(bars),
                    Block::Map(MapGrid {
                        title: "M".into(),
                        cols: 3,
                        rows: 2,
                        cells: vec![None, Some(0), Some(4), None, Some(2), None],
                        levels: ["none", "low", "mid", "high", "top"].map(String::from).to_vec(),
                        markers: vec![MapMarker { col: 2, row: 0, tag: "1".into(), label: "Site 1".into() }],
                        note: "North is up.".into(),
                    }),
                ],
                method: "Counted.".into(),
            }],
            timings: vec![("ingest".into(), 0.5)],
        };
        let text = renderer(Format::Terminal { color: false }).render(&report);
        assert!(text.contains("Canton <b>") && text.contains("Because.") && !text.contains('\x1b'));
        // The map is cropped to its occupied columns: " · 1" then "▒▒".
        assert!(text.contains("     · 1   N ↑") && text.contains("    ▒▒\n"), "{text}");
        assert!(text.contains("top") && !text.contains("high"), "unused levels stay out of the legend");
        let html = renderer(Format::Html).render(&report);
        assert!(html.contains("Canton &lt;b&gt;") && html.contains("<table"));
        assert!(html.contains("<svg viewBox=\"1 0 2 2\"") && html.contains("class=\"l4\""));
        let json: serde_json::Value = serde_json::from_str(&renderer(Format::Json).render(&report)).unwrap();
        assert_eq!(json["sections"][0]["blocks"][0]["rows"][0][1], "12");
        assert_eq!(json["sections"][0]["blocks"][2]["cells"][2], 4);
    }
}
