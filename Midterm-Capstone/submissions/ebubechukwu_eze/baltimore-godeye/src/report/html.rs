//! Self-contained HTML renderer (inline CSS, no scripts, no network requests).

use std::fmt::Write;

use super::{BarChart, Block, MapGrid, Renderer, Report, Table};

pub struct HtmlRenderer;

fn esc(text: &str) -> String {
    let mut out = String::with_capacity(text.len());
    for c in text.chars() {
        match c {
            '&' => out.push_str("&amp;"),
            '<' => out.push_str("&lt;"),
            '>' => out.push_str("&gt;"),
            '"' => out.push_str("&quot;"),
            _ => out.push(c),
        }
    }
    out
}

const STYLE: &str = r#"
:root { --l0:#eceff3; --l1:#f6d5c4; --l2:#ec9f7f; --l3:#d4593f; --l4:#8f1d1d; --mark:#f2c230; --bg:#f6f7f9; --card:#fff; --ink:#1b2430; --muted:#5d6b7c; --line:#dfe4ea; --accent:#0b6e8f; --bar:#2a8fb0; --note:#fff8e1; }
@media (prefers-color-scheme: dark) {
  :root { --l0:#232b35; --l1:#4a2f2a; --l2:#7d3a2c; --l3:#b8492f; --l4:#ef6a4a; --mark:#f2c230; --bg:#11161c; --card:#1a212a; --ink:#e6ebf1; --muted:#9aa7b6; --line:#2c3642; --accent:#5cc0e0; --bar:#3d9fc0; --note:#2a2616; }
}
* { box-sizing: border-box; }
body { margin:0; background:var(--bg); color:var(--ink); font:15px/1.5 -apple-system, "Segoe UI", Helvetica, Arial, sans-serif; }
main { max-width:1040px; margin:0 auto; padding:24px 16px 64px; }
header h1 { margin:0 0 4px; font-size:28px; }
header p { margin:0; color:var(--muted); }
nav { display:flex; flex-wrap:wrap; gap:8px; margin:20px 0; }
nav a { padding:6px 12px; border:1px solid var(--line); border-radius:999px; background:var(--card); color:var(--accent); text-decoration:none; font-size:14px; }
section { background:var(--card); border:1px solid var(--line); border-radius:10px; padding:20px 24px; margin:20px 0; }
section h2 { margin:0; font-size:21px; }
.who { color:var(--muted); font-size:13px; text-transform:uppercase; letter-spacing:.05em; margin-bottom:4px; }
.question { font-size:16px; color:var(--muted); margin:8px 0 16px; }
.findings { background:var(--note); border-left:4px solid var(--accent); border-radius:4px; padding:12px 16px 12px 34px; margin:0 0 8px; }
.findings li { margin:4px 0; }
h3 { font-size:15px; margin:24px 0 8px; }
.scroll { overflow-x:auto; }
table { border-collapse:collapse; width:100%; font-size:14px; font-variant-numeric:tabular-nums; }
th, td { padding:6px 10px; border-bottom:1px solid var(--line); text-align:left; white-space:nowrap; }
th { color:var(--muted); font-weight:600; border-bottom:2px solid var(--line); }
.num { text-align:right; }
.bars { display:grid; grid-template-columns:minmax(90px, max-content) 1fr max-content; gap:6px 10px; align-items:center; font-size:14px; }
.track { height:14px; }
.fill { height:100%; min-width:2px; background:var(--bar); border-radius:2px; }
.val { font-variant-numeric:tabular-nums; color:var(--muted); }
dl { display:grid; grid-template-columns:max-content 1fr; gap:4px 20px; margin:0; font-size:14px; }
dt { color:var(--muted); } dd { margin:0; font-weight:600; }
.map { display:flex; flex-wrap:wrap; gap:16px 28px; align-items:flex-start; }
.map svg { width:100%; max-width:440px; height:auto; }
.map .l0 { fill:var(--l0); } .map .l1 { fill:var(--l1); } .map .l2 { fill:var(--l2); } .map .l3 { fill:var(--l3); } .map .l4 { fill:var(--l4); }
.map .pin circle { fill:var(--mark); stroke:#1b2430; stroke-width:.08; } .map .pin text { font:700 1.1px sans-serif; fill:#1b2430; text-anchor:middle; dominant-baseline:central; }
.legend { font-size:14px; } .legend div { display:flex; gap:8px; align-items:center; margin:3px 0; }
.sw { width:14px; height:14px; border-radius:2px; flex:none; } .num-pin { width:18px; height:18px; border-radius:50%; background:var(--mark); color:#1b2430; font-weight:700; font-size:12px; display:grid; place-items:center; flex:none; }
.method, footer { color:var(--muted); font-size:13px; margin-top:20px; }
"#;

fn table(t: &Table, out: &mut String) {
    let _ = write!(out, "<h3>{}</h3><div class=\"scroll\"><table><thead><tr>", esc(&t.title));
    let class = |i: usize| if t.numeric[i] { " class=\"num\"" } else { "" };
    for (i, header) in t.headers.iter().enumerate() {
        let _ = write!(out, "<th{}>{}</th>", class(i), esc(header));
    }
    out.push_str("</tr></thead><tbody>");
    for row in &t.rows {
        out.push_str("<tr>");
        for (i, cell) in row.iter().enumerate().take(t.headers.len()) {
            let _ = write!(out, "<td{}>{}</td>", class(i), esc(cell));
        }
        out.push_str("</tr>");
    }
    out.push_str("</tbody></table></div>\n");
}

fn bars(chart: &BarChart, out: &mut String) {
    let max = chart.bars.iter().map(|b| b.value).fold(0.0, f64::max);
    let _ = write!(out, "<h3>{}</h3><div class=\"bars\">", esc(&chart.title));
    for b in &chart.bars {
        let share = if max > 0.0 { (b.value / max * 100.0).clamp(0.0, 100.0) } else { 0.0 };
        let _ = write!(
            out,
            "<div>{}</div><div class=\"track\"><div class=\"fill\" style=\"width:{share:.1}%\"></div></div><div class=\"val\">{}</div>",
            esc(&b.label),
            esc(&b.display)
        );
    }
    out.push_str("</div>\n");
}

/// The map as an SVG in grid units (one unit per square), cropped to the city.
fn map(m: &MapGrid, out: &mut String) {
    let inside = |c: usize, r: usize| m.cells[r * m.cols + c].is_some();
    let rows: Vec<usize> = (0..m.rows).filter(|&r| (0..m.cols).any(|c| inside(c, r))).collect();
    let cols: Vec<usize> = (0..m.cols).filter(|&c| (0..m.rows).any(|r| inside(c, r))).collect();
    let (Some(&top), Some(&bottom), Some(&left), Some(&right)) = (rows.first(), rows.last(), cols.first(), cols.last()) else {
        return;
    };
    let _ = write!(
        out,
        "<h3>{}</h3><div class=\"map\"><svg viewBox=\"{left} {top} {} {}\" role=\"img\" aria-label=\"{}\">",
        esc(&m.title),
        right - left + 1,
        bottom - top + 1,
        esc(&m.title)
    );
    for r in top..=bottom {
        for c in left..=right {
            if let Some(level) = m.cells[r * m.cols + c] {
                let _ = write!(out, "<rect class=\"l{}\" x=\"{c}\" y=\"{r}\" width=\"1.02\" height=\"1.02\"/>", level.min(4));
            }
        }
    }
    for (i, p) in m.markers.iter().enumerate() {
        if m.markers[..i].iter().any(|q| (q.col, q.row) == (p.col, p.row)) {
            continue; // a higher-ranked marker already holds this square
        }
        let _ = write!(
            out,
            "<g class=\"pin\"><title>{}</title><circle cx=\"{}.5\" cy=\"{}.5\" r=\".85\"/><text x=\"{}.5\" y=\"{}.5\">{}</text></g>",
            esc(&p.label), p.col, p.row, p.col, p.row, esc(&p.tag)
        );
    }
    out.push_str("</svg><div class=\"legend\">");
    for (level, label) in m.used_levels() {
        let _ = write!(out, "<div><span class=\"sw\" style=\"background:var(--l{level})\"></span>{}</div>", esc(label));
    }
    for p in &m.markers {
        let _ = write!(out, "<div><span class=\"num-pin\">{}</span>{}</div>", esc(&p.tag), esc(&p.label));
    }
    let _ = writeln!(out, "<p class=\"method\">{}</p></div></div>", esc(&m.note));
}

fn block(b: &Block, out: &mut String) {
    match b {
        Block::Facts { title, items } => {
            let _ = write!(out, "<h3>{}</h3><dl>", esc(title));
            for (key, value) in items {
                let _ = write!(out, "<dt>{}</dt><dd>{}</dd>", esc(key), esc(value));
            }
            out.push_str("</dl>\n");
        }
        Block::Table(t) => table(t, out),
        Block::Bars(chart) => bars(chart, out),
        Block::Map(m) => map(m, out),
    }
}

impl Renderer for HtmlRenderer {
    fn extension(&self) -> &'static str {
        "html"
    }

    fn render(&self, report: &Report) -> String {
        let mut out = String::with_capacity(64 * 1024);
        let _ = write!(
            out,
            "<!doctype html>\n<html lang=\"en\"><head><meta charset=\"utf-8\">\
             <meta name=\"viewport\" content=\"width=device-width, initial-scale=1\">\
             <title>{}</title><style>{STYLE}</style></head><body><main>\n\
             <header><h1>{}</h1><p>{}</p></header>\n<nav><a href=\"#data\">Data</a>",
            esc(&report.title),
            esc(&report.title),
            esc(&report.subtitle)
        );
        for s in &report.sections {
            let _ = write!(out, "<a href=\"#{}\">{}: {}</a>", s.id, s.id.to_uppercase(), esc(&s.title));
        }
        out.push_str("</nav>\n<section id=\"data\"><h2>Data and pipeline</h2>\n");
        for b in &report.overview {
            block(b, &mut out);
        }
        out.push_str("</section>\n");
        for s in &report.sections {
            let _ = write!(
                out,
                "<section id=\"{}\"><div class=\"who\">{} · for {}</div><h2>{}</h2><p class=\"question\">{}</p><ul class=\"findings\">",
                s.id,
                s.id.to_uppercase(),
                esc(&s.stakeholder),
                esc(&s.title),
                esc(&s.question)
            );
            for finding in &s.findings {
                let _ = write!(out, "<li>{}</li>", esc(finding));
            }
            out.push_str("</ul>\n");
            for b in &s.blocks {
                block(b, &mut out);
            }
            let _ = writeln!(out, "<p class=\"method\"><strong>Method.</strong> {}</p></section>", esc(&s.method));
        }
        let timings: Vec<String> = report.timings.iter().map(|(stage, s)| format!("{} {s:.2}s", esc(stage))).collect();
        let _ = write!(out, "<footer>Pipeline timings: {}</footer>\n</main></body></html>\n", timings.join(" · "));
        out
    }
}
