//! Plain-text renderer: box-drawn tables and horizontal bar charts.

use std::fmt::Write;

use super::{BarChart, Block, MapGrid, Renderer, Report, Table};

pub struct TerminalRenderer {
    pub color: bool,
}

const WIDTH: usize = 100;
const BAR_WIDTH: usize = 40;
const MAX_CELL: usize = 44;

fn width(s: &str) -> usize {
    s.chars().count()
}

fn clip(s: &str, max: usize) -> String {
    if width(s) <= max {
        s.to_string()
    } else {
        let mut out: String = s.chars().take(max - 1).collect();
        out.push('…');
        out
    }
}

/// Greedy word wrap with a hanging indent.
fn wrap(text: &str, first: &str, rest: &str, out: &mut String) {
    let mut line = String::from(first);
    let mut empty = true;
    for word in text.split_whitespace() {
        if !empty && width(&line) + 1 + width(word) > WIDTH {
            let _ = writeln!(out, "{line}");
            line = String::from(rest);
            empty = true;
        }
        if !empty {
            line.push(' ');
        }
        line.push_str(word);
        empty = false;
    }
    let _ = writeln!(out, "{line}");
}

/// A bar of `fraction * BAR_WIDTH` cells, using eighth-blocks for the remainder.
fn bar(fraction: f64) -> String {
    const PARTIAL: [char; 8] = [' ', '▏', '▎', '▍', '▌', '▋', '▊', '▉'];
    let eighths = (fraction.clamp(0.0, 1.0) * (BAR_WIDTH * 8) as f64).round() as usize;
    let mut out = "█".repeat(eighths / 8);
    if eighths % 8 > 0 {
        out.push(PARTIAL[eighths % 8]);
    }
    out
}

impl TerminalRenderer {
    fn paint(&self, code: &str, text: &str) -> String {
        if self.color {
            format!("\x1b[{code}m{text}\x1b[0m")
        } else {
            text.to_string()
        }
    }

    fn table(&self, table: &Table, out: &mut String) {
        let cols = table.headers.len();
        let mut widths: Vec<usize> = table.headers.iter().map(|h| width(h)).collect();
        for row in &table.rows {
            for (i, cell) in row.iter().enumerate().take(cols) {
                widths[i] = widths[i].max(width(cell).min(MAX_CELL));
            }
        }
        let rule = |left: &str, mid: &str, right: &str| {
            let cells: Vec<String> = widths.iter().map(|w| "─".repeat(w + 2)).collect();
            format!("  {left}{}{right}", cells.join(mid))
        };
        let line = |cells: &[String], bold: bool| {
            let mut s = String::from("  │");
            for (i, cell) in cells.iter().enumerate().take(cols) {
                let cell = clip(cell, MAX_CELL);
                let pad = " ".repeat(widths[i] - width(&cell));
                let text = if bold { self.paint("1", &cell) } else { cell };
                if table.numeric[i] {
                    let _ = write!(s, " {pad}{text} │");
                } else {
                    let _ = write!(s, " {text}{pad} │");
                }
            }
            s
        };
        let _ = writeln!(out, "\n  {}", self.paint("1", &table.title));
        let _ = writeln!(out, "{}", rule("┌", "┬", "┐"));
        let _ = writeln!(out, "{}", line(&table.headers, true));
        let _ = writeln!(out, "{}", rule("├", "┼", "┤"));
        for row in &table.rows {
            let _ = writeln!(out, "{}", line(row, false));
        }
        let _ = writeln!(out, "{}", rule("└", "┴", "┘"));
    }

    fn bars(&self, chart: &BarChart, out: &mut String) {
        let label_width = chart.bars.iter().map(|b| width(&b.label).min(34)).max().unwrap_or(0);
        let max = chart.bars.iter().map(|b| b.value).fold(0.0, f64::max);
        let _ = writeln!(out, "\n  {}", self.paint("1", &chart.title));
        for b in &chart.bars {
            let label = clip(&b.label, 34);
            let pad = " ".repeat(label_width - width(&label));
            let fill = bar(if max > 0.0 { b.value / max } else { 0.0 });
            let _ = writeln!(out, "  {label}{pad} │{} {}", self.paint("36", &fill), b.display);
        }
    }

    /// Two characters per square, shaded with block elements; markers are
    /// drawn over their square in bold. Empty rows and columns around the
    /// city are trimmed.
    fn map(&self, map: &MapGrid, out: &mut String) {
        const SHADES: [&str; MapGrid::LEVELS] = [" ·", "░░", "▒▒", "▓▓", "██"];
        let inside = |c: usize, r: usize| map.cells[r * map.cols + c].is_some();
        let rows: Vec<usize> = (0..map.rows).filter(|&r| (0..map.cols).any(|c| inside(c, r))).collect();
        let cols: Vec<usize> = (0..map.cols).filter(|&c| (0..map.rows).any(|r| inside(c, r))).collect();
        let (Some(&top), Some(&bottom), Some(&left), Some(&right)) = (rows.first(), rows.last(), cols.first(), cols.last()) else {
            return;
        };
        let _ = writeln!(out, "\n  {}", self.paint("1", &map.title));
        for r in top..=bottom {
            let mut line = String::from("    ");
            for c in left..=right {
                if let Some(m) = map.markers.iter().find(|m| m.col == c && m.row == r) {
                    line.push_str(&self.paint("1;30;43", &format!("{:>2}", clip(&m.tag, 2))));
                    continue;
                }
                match map.cells[r * map.cols + c] {
                    None => line.push_str("  "),
                    Some(0) => line.push_str(&self.paint("2", SHADES[0])),
                    Some(level) => line.push_str(&self.paint("31", SHADES[usize::from(level).min(4)])),
                }
            }
            if r == top {
                line.push_str("   N ↑");
            }
            let _ = writeln!(out, "{}", line.trim_end());
        }
        let _ = writeln!(out);
        for (level, label) in map.used_levels() {
            let swatch = if level == 0 { self.paint("2", SHADES[0]) } else { self.paint("31", SHADES[level.min(4)]) };
            let _ = writeln!(out, "    {swatch}  {label}");
        }
        for m in &map.markers {
            let _ = writeln!(out, "    {}  {}", self.paint("1;30;43", &format!("{:>2}", clip(&m.tag, 2))), m.label);
        }
        wrap(&map.note, "    ", "    ", out);
    }

    fn block(&self, block: &Block, out: &mut String) {
        match block {
            Block::Facts { title, items } => {
                let key_width = items.iter().map(|(k, _)| width(k)).max().unwrap_or(0);
                let _ = writeln!(out, "\n  {}", self.paint("1", title));
                for (key, value) in items {
                    let dots = ".".repeat(key_width - width(key) + 3);
                    let _ = writeln!(out, "  {key} {} {value}", self.paint("2", &dots));
                }
            }
            Block::Table(table) => self.table(table, out),
            Block::Bars(chart) => self.bars(chart, out),
            Block::Map(map) => self.map(map, out),
        }
    }
}

impl Renderer for TerminalRenderer {
    fn extension(&self) -> &'static str {
        "txt"
    }

    fn render(&self, report: &Report) -> String {
        let mut out = String::new();
        let heavy = "═".repeat(WIDTH);
        let _ = writeln!(out, "{heavy}\n  {}\n  {}\n{heavy}", self.paint("1;36", &report.title), report.subtitle);
        for block in &report.overview {
            self.block(block, &mut out);
        }
        for section in &report.sections {
            let _ = writeln!(out, "\n{}", "━".repeat(WIDTH));
            let heading = format!("{}  {}", section.id.to_uppercase(), section.title);
            let _ = writeln!(out, "  {}", self.paint("1;36", &heading));
            let _ = writeln!(out, "  For: {}", section.stakeholder);
            let _ = writeln!(out, "{}", "━".repeat(WIDTH));
            wrap(&section.question, "  Question: ", "            ", &mut out);
            let _ = writeln!(out, "\n  {}", self.paint("1;33", "Key findings"));
            for finding in &section.findings {
                wrap(finding, "   • ", "     ", &mut out);
            }
            for block in &section.blocks {
                self.block(block, &mut out);
            }
            let _ = writeln!(out);
            wrap(&format!("Method: {}", section.method), "  ", "  ", &mut out);
        }
        let _ = writeln!(out, "\n{}", "─".repeat(WIDTH));
        let timings: Vec<String> = report.timings.iter().map(|(stage, s)| format!("{stage} {s:.2}s")).collect();
        wrap(&format!("Pipeline timings: {}", timings.join(" | ")), "  ", "  ", &mut out);
        out
    }
}
