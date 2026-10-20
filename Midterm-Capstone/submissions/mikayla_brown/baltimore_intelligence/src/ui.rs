use crate::ingestion::Sources;
use crate::models::{Metric, Report};

fn section(title: &str, rows: &[Metric], money: bool) {
    println!("\n{title}\n{}", "─".repeat(title.chars().count()));

    if rows.is_empty() {
        println!("  No records matched.");
        return;
    }

    for row in rows {
        let value = if money {
            format!("${:.0}", row.value)
        } else {
            format!("{:.1}", row.value)
        };
        let bars = "█".repeat((row.value.abs().sqrt().round() as usize).clamp(1, 30));

        println!(
            "  {:<22} {:>10}  {}\n  {}",
            row.label, value, bars, row.detail
        );
    }
}

pub fn render(
    report: &Report,
    sources: &Sources,
    settlements: usize,
    incidents: usize,
) {
    println!(
        "\x1b[1;36mBaltimore Municipal Intelligence Engine\x1b[0m\nRecords: {settlements} settlements | {incidents} incidents\nSettlement source: {}\nIncident source: {}",
        sources.settlements, sources.incidents
    );

    section("Q1 · Settlement cost by district", &report.q1, true);
    section(
        "Q2 · Units/shifts in matched records (association, not correlation)",
        &report.q2,
        false,
    );
    section("Q3 · Reporting and discipline timeline", &report.q3, false);
    section("Q4 · Recurring officer identifiers", &report.q4, false);
    section("Q5 · District resource allocation risk index", &report.q5, false);

    println!("\nData limitations\n{}", report.caveat);
}
