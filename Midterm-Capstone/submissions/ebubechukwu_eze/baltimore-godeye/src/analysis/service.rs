//! Q4: is 311 blight response as fast where vacancy is worst?

use rayon::prelude::*;

use super::{vacancy_tier, Context, Question, TIER_LABELS};
use crate::model::*;
use crate::report::{dec, int, pct, BarChart, Block, Section, Table};
use crate::stats::{percentile, spearman};

pub struct BlightResponse;

fn days(secs: u32) -> f64 {
    f64::from(secs) / 86_400.0
}

impl Question for BlightResponse {
    fn id(&self) -> &'static str {
        "q4"
    }

    fn run(&self, ctx: &Context<'_>) -> Section {
        let store = ctx.store;
        let blight = || store.requests.par_iter().filter(|r| r.category == SrCategory::Blight && !r.duplicate);

        // Closure times per neighborhood, sorted so each neighborhood is a contiguous run.
        let mut closed: Vec<(NeighborhoodId, u32)> = blight()
            .filter(|r| r.close_secs != NOT_CLOSED && r.neighborhood != NO_NEIGHBORHOOD)
            .map(|r| (r.neighborhood, r.close_secs))
            .collect();
        closed.par_sort_unstable();
        let mut open_by_tier = [0u64; 4];
        for r in store.requests.iter().filter(|r| {
            r.category == SrCategory::Blight && !r.duplicate && r.close_secs == NOT_CLOSED && r.neighborhood != NO_NEIGHBORHOOD
        }) {
            open_by_tier[vacancy_tier(ctx.nb[r.neighborhood as usize].vacants)] += 1;
        }

        let mut tier_times: [Vec<u32>; 4] = Default::default();
        let mut nb_median = vec![0u32; store.neighborhoods.len()];
        for run in closed.chunk_by(|a, b| a.0 == b.0) {
            let id = run[0].0;
            let secs: Vec<u32> = run.iter().map(|r| r.1).collect();
            nb_median[id as usize] = percentile(&secs, 50.0);
            tier_times[vacancy_tier(ctx.nb[id as usize].vacants)].extend(secs);
        }
        tier_times.iter_mut().for_each(|t| t.sort_unstable());

        let mut tier_table = Table::new(
            "Resident-reported blight requests (2025) by neighborhood vacancy tier",
            &["Tier", "#Neighborhoods", "#Requests", "#Requests / nbhd", "#Median days to close", "#90th pct days", "#Still open"],
        );
        let mut tier_bars = BarChart::new("Median days to close a blight request, by vacancy tier");
        let mut nb_per_tier = [0u64; 4];
        for &id in &ctx.order {
            nb_per_tier[vacancy_tier(ctx.nb[id as usize].vacants)] += 1;
        }
        for t in 0..4 {
            let total = tier_times[t].len() as u64 + open_by_tier[t];
            let median = days(percentile(&tier_times[t], 50.0));
            tier_table.row(vec![
                TIER_LABELS[t].into(),
                int(nb_per_tier[t]),
                int(total),
                int(total.checked_div(nb_per_tier[t]).unwrap_or(0)),
                dec(median, 1),
                dec(days(percentile(&tier_times[t], 90.0)), 1),
                pct(open_by_tier[t] as f64, total as f64),
            ]);
            tier_bars.bar(TIER_LABELS[t], median, format!("{median:.1} days"));
        }

        // By request type, city-wide (includes requests with no neighborhood).
        let mut by_type: Vec<(u16, u32)> = blight().map(|r| (r.sr_type, r.close_secs)).collect();
        by_type.par_sort_unstable();
        let mut types: Vec<(usize, u16, f64, f64, usize)> = by_type
            .chunk_by(|a, b| a.0 == b.0)
            .map(|run| {
                // NOT_CLOSED is u32::MAX, so open requests sort to the end of the run.
                let done = run.partition_point(|r| r.1 != NOT_CLOSED);
                let secs: Vec<u32> = run[..done].iter().map(|r| r.1).collect();
                (run.len(), run[0].0, days(percentile(&secs, 50.0)), days(percentile(&secs, 90.0)), run.len() - done)
            })
            .collect();
        types.sort_by(|a, b| b.0.cmp(&a.0).then_with(|| store.sr_types[a.1 as usize].cmp(&store.sr_types[b.1 as usize])));
        let mut type_table = Table::new(
            "Blight request types, city-wide",
            &["Request type", "#Requests", "#Median days to close", "#90th pct days", "#Still open"],
        );
        for (n, sr_type, median, p90, open) in types.iter().take(ctx.top_n) {
            type_table.row(vec![
                store.sr_types[*sr_type as usize].to_string(),
                int(*n as u64),
                dec(*median, 1),
                dec(*p90, 1),
                pct(*open as f64, *n as f64),
            ]);
        }

        let mut all_closed: Vec<u32> = by_type.iter().filter(|r| r.1 != NOT_CLOSED).map(|r| r.1).collect();
        all_closed.par_sort_unstable();
        let still_open = by_type.len() - all_closed.len();
        let mut speed = BarChart::new("How long blight requests take to close, city-wide");
        let edges = [1.0, 3.0, 7.0, 14.0, 30.0];
        let labels = ["under 1 day", "1–3 days", "3–7 days", "7–14 days", "14–30 days", "over 30 days"];
        let mut buckets = [0u64; 6];
        for &(_, secs) in by_type.iter().filter(|r| r.1 != NOT_CLOSED) {
            buckets[edges.iter().position(|&e| days(secs) < e).unwrap_or(5)] += 1;
        }
        for (label, n) in labels.iter().zip(buckets) {
            speed.bar(*label, n as f64, format!("{} ({})", int(n), pct(n as f64, by_type.len() as f64)));
        }
        speed.bar("still open", still_open as f64, format!("{} ({})", int(still_open as u64), pct(still_open as f64, by_type.len() as f64)));

        let ranked = ctx.ranked(|r| f64::from(r.blight));
        let mut top = Table::new(
            format!("Top {} neighborhoods by blight requests", ctx.top_n),
            &["Neighborhood", "#Blight requests", "#Median days to close", "#Vacant notices", "#Requests per vacant"],
        );
        for &id in ranked.iter().take(ctx.top_n) {
            let row = &ctx.nb[id as usize];
            top.row(vec![
                ctx.name(id).to_string(),
                int(row.blight.into()),
                dec(days(nb_median[id as usize]), 1),
                int(row.vacants.into()),
                if row.vacants > 0 { dec(f64::from(row.blight) / f64::from(row.vacants), 1) } else { "n/a".into() },
            ]);
        }

        let blight_counts: Vec<f64> = ctx.order.iter().map(|&id| f64::from(ctx.nb[id as usize].blight)).collect();
        let vacants: Vec<f64> = ctx.order.iter().map(|&id| f64::from(ctx.nb[id as usize].vacants)).collect();
        let rho = spearman(&blight_counts, &vacants);
        let median_of = |t: usize| days(percentile(&tier_times[t], 50.0));
        let per_nb = |t: usize| {
            if nb_per_tier[t] > 0 {
                (tier_times[t].len() as u64 + open_by_tier[t]) as f64 / nb_per_tier[t] as f64
            } else {
                0.0
            }
        };
        let gap = median_of(3) - median_of(0);
        let verdict = if gap.abs() < 0.5 {
            "about the same speed as".to_string()
        } else if gap > 0.0 {
            format!("{gap:.1} days slower than")
        } else {
            format!("{:.1} days faster than", -gap)
        };
        let slowest = types.iter().take(ctx.top_n).max_by(|a, b| a.2.total_cmp(&b.2));
        let mut findings = vec![
            format!(
                "Residents filed {} blight requests in 2025; the city-wide median time to close was {:.1} days and {} were still open.",
                int(by_type.len() as u64),
                days(percentile(&all_closed, 50.0)),
                pct(still_open as f64, by_type.len() as f64)
            ),
            format!(
                "Demand follows vacancy (Spearman {rho:.2}): a neighborhood with 100+ vacants files {} blight requests a year on average, against {} where there are none.",
                int(per_nb(3) as u64),
                int(per_nb(0) as u64)
            ),
            format!(
                "Equity check: the median blight request in 100+ vacant neighborhoods closes in {:.1} days, {verdict} neighborhoods with no vacants ({:.1} days).",
                median_of(3),
                median_of(0)
            ),
        ];
        if let Some((_, sr_type, median, p90, _)) = slowest {
            findings.push(format!(
                "Suggested action: '{}' is the slowest high-volume type (median {median:.1} days, 90th percentile {p90:.1}); it is the first place to add crews or review the closure standard.",
                store.sr_types[*sr_type as usize]
            ));
        }

        Section {
            id: self.id(),
            title: "311 blight response and equity".into(),
            stakeholder: "Department of Public Works and Department of Housing & Community Development".into(),
            question: "Where is resident-reported blight concentrated, and do high-vacancy neighborhoods wait longer for it to be resolved?".into(),
            findings,
            blocks: vec![Block::Bars(tier_bars), Block::Table(tier_table), Block::Bars(speed), Block::Table(type_table), Block::Table(top)],
            method: "Joins 2025 311 requests to Vacant Building Notices by neighborhood. Blight = resident-reported sanitation, dirty street/alley, illegal dumping, \
                     vacant building, boarding, cleaning, rodent, high grass, structure maintenance and graffiti requests; duplicates, call-centre information requests and \
                     city-initiated proactive work orders are excluded. Time to close = CloseDate minus CreatedDate for closed requests; \
                     'closed' is the agency's own status and may not mean the problem was fixed."
                .into(),
        }
    }
}
