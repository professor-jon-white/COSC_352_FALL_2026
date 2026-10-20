//! Q5: one ranked list that combines all five datasets.

use super::{citymap, Context, Question};
use crate::report::{dec, int, BarChart, Block, Section, Table};
use crate::stats::{min_max_scale, percentile};

pub struct PriorityIndex;

struct Component {
    name: &'static str,
    weight: f64,
    lead: &'static str,
}

const COMPONENTS: [Component; 6] = [
    Component { name: "Violent crime (12 mo)", weight: 0.25, lead: "Police (violence)" },
    Component { name: "Shootings (12 mo)", weight: 0.15, lead: "Police (violence)" },
    Component { name: "Vacant building notices", weight: 0.25, lead: "Housing (vacants)" },
    Component { name: "311 blight requests", weight: 0.15, lead: "Public Works (blight)" },
    Component { name: "Violent crime outside camera range", weight: 0.10, lead: "CitiWatch (cameras)" },
    Component { name: "Liquor outlets", weight: 0.10, lead: "Liquor Board (outlets)" },
];

impl Question for PriorityIndex {
    fn id(&self) -> &'static str {
        "q5"
    }

    fn run(&self, ctx: &Context<'_>) -> Section {
        let raw: [Vec<f64>; 6] = [
            ctx.order.iter().map(|&id| f64::from(ctx.nb[id as usize].violent_12m)).collect(),
            ctx.order.iter().map(|&id| f64::from(ctx.nb[id as usize].shootings_12m)).collect(),
            ctx.order.iter().map(|&id| f64::from(ctx.nb[id as usize].vacants)).collect(),
            ctx.order.iter().map(|&id| f64::from(ctx.nb[id as usize].blight)).collect(),
            ctx.order.iter().map(|&id| f64::from(ctx.nb[id as usize].uncovered_12m)).collect(),
            ctx.order.iter().map(|&id| f64::from(ctx.nb[id as usize].outlets)).collect(),
        ];
        let scaled: Vec<Vec<f64>> = raw.iter().map(|values| min_max_scale(values)).collect();

        // (index score, position in ctx.order, lead agency)
        let mut scores: Vec<(f64, usize, &'static str)> = (0..ctx.order.len())
            .map(|i| {
                let mut by_lead: Vec<(&'static str, f64)> = Vec::new();
                let mut total = 0.0;
                for (component, values) in COMPONENTS.iter().zip(&scaled) {
                    let points = component.weight * values[i];
                    total += points;
                    match by_lead.iter_mut().find(|(lead, _)| *lead == component.lead) {
                        Some(entry) => entry.1 += points,
                        None => by_lead.push((component.lead, points)),
                    }
                }
                let lead = by_lead.iter().fold(("", f64::MIN), |best, &(l, p)| if p > best.1 { (l, p) } else { best }).0;
                (total, i, lead)
            })
            .collect();
        scores.sort_by(|a, b| b.0.total_cmp(&a.0).then(a.1.cmp(&b.1)));

        let mut table = Table::new(
            format!("Top {} neighborhoods by Neighborhood Priority Index (0–100)", ctx.top_n),
            &["#Rank", "Neighborhood", "#Index", "#Violent 12mo", "#Shootings 12mo", "#Vacants", "#Blight 311", "#Out of camera range", "#Liquor outlets", "Largest driver"],
        );
        let mut bars = BarChart::new("Neighborhood Priority Index");
        for (rank, &(score, i, lead)) in scores.iter().take(ctx.top_n).enumerate() {
            let id = ctx.order[i];
            let row = &ctx.nb[id as usize];
            table.row(vec![
                (rank + 1).to_string(),
                ctx.name(id).to_string(),
                dec(score, 1),
                int(row.violent_12m.into()),
                int(row.shootings_12m.into()),
                int(row.vacants.into()),
                int(row.blight.into()),
                int(row.uncovered_12m.into()),
                int(row.outlets.into()),
                lead.to_string(),
            ]);
            bars.bar(ctx.name(id), score, dec(score, 1));
        }

        let map = priority_map(ctx, &scores);

        let mut weights = Table::new("How the index is built", &["Component", "#Weight", "Lead agency if it dominates"]);
        for c in &COMPONENTS {
            weights.row(vec![c.name.into(), format!("{:.0}%", c.weight * 100.0), c.lead.into()]);
        }

        let top = &scores[..scores.len().min(ctx.top_n)];
        let mut leads: Vec<(&str, usize)> = Vec::new();
        for &(_, _, lead) in top {
            match leads.iter_mut().find(|(l, _)| *l == lead) {
                Some(entry) => entry.1 += 1,
                None => leads.push((lead, 1)),
            }
        }
        leads.sort_by(|a, b| b.1.cmp(&a.1).then(a.0.cmp(b.0)));
        let lead_text: Vec<String> = leads.iter().map(|(lead, n)| format!("{lead} in {n}")).collect();
        let names: Vec<&str> = top.iter().take(5).map(|&(_, i, _)| ctx.name(ctx.order[i])).collect();
        let sum = |pick: &dyn Fn(&super::NbAgg) -> u32, ids: &mut dyn Iterator<Item = usize>| -> u64 {
            ids.map(|i| u64::from(pick(&ctx.nb[ctx.order[i] as usize]))).sum()
        };
        let top_shootings = sum(&|r| r.shootings_12m, &mut top.iter().map(|s| s.1));
        let all_shootings = sum(&|r| r.shootings_12m, &mut (0..ctx.order.len()));
        let top_vacants = sum(&|r| r.vacants, &mut top.iter().map(|s| s.1));
        let all_vacants = sum(&|r| r.vacants, &mut (0..ctx.order.len()));
        let findings = vec![
            format!("Highest combined need: {}.", names.join(", ")),
            format!(
                "The top {} of {} neighborhoods hold {} of the city's vacant-building notices and {} of the last year's shootings.",
                top.len(),
                ctx.order.len(),
                crate::report::pct(top_vacants as f64, all_vacants as f64),
                crate::report::pct(top_shootings as f64, all_shootings as f64)
            ),
            format!("The largest driver among the top {} is: {}.", top.len(), lead_text.join(", ")),
            "Suggested action: use the 'largest driver' column to assign a lead agency per neighborhood, and fund the top of this list first when budgets force a choice.".into(),
        ];

        Section {
            id: self.id(),
            title: "Neighborhood Priority Index".into(),
            stakeholder: "Mayor's Office and Department of Planning".into(),
            question: "Taking violence, vacancy, blight, camera coverage and liquor outlets together, which neighborhoods need coordinated investment first?".into(),
            findings,
            blocks: vec![Block::Bars(bars), Block::Map(map), Block::Table(table), Block::Table(weights)],
            method: format!(
                "Each component is rescaled to 0–100 across neighborhoods (min–max) and combined with the weights shown. Crime components use {}; \
                 311 uses 2025; vacants, cameras and licences are current. The weights are a policy choice, not a statistical result, and the inputs are raw counts, \
                 so large neighborhoods score higher than a per-resident index would rank them.",
                ctx.recent_period()
            ),
        }
    }
}

/// Choropleth: every square takes the index quartile of the neighborhood that
/// most of its records belong to; the top five are numbered at their centre.
fn priority_map(ctx: &Context<'_>, scores: &[(f64, usize, &'static str)]) -> crate::report::MapGrid {
    let mut score_of = vec![f64::NAN; ctx.store.neighborhoods.len()];
    for &(score, i, _) in scores {
        score_of[ctx.order[i] as usize] = score;
    }
    let mut sorted: Vec<f64> = scores.iter().map(|s| s.0).collect();
    sorted.sort_by(f64::total_cmp);
    let cuts = [percentile(&sorted, 25.0), percentile(&sorted, 50.0), percentile(&sorted, 75.0)];

    let owner = citymap::cell_neighborhoods(ctx.store);
    let mask = citymap::city_mask(&owner);
    let shades = mask
        .iter()
        .zip(&owner)
        .map(|(&inside, &id)| {
            let score = score_of.get(id as usize).copied().unwrap_or(f64::NAN);
            inside.then(|| match score {
                s if s.is_nan() => 0,
                s if s <= cuts[0] => 1,
                s if s <= cuts[1] => 2,
                s if s <= cuts[2] => 3,
                _ => 4,
            })
        })
        .collect();
    let levels = vec![
        "no index (no records)".to_string(),
        format!("bottom quarter (index ≤ {})", dec(cuts[0], 1)),
        format!("second quarter (≤ {})", dec(cuts[1], 1)),
        format!("third quarter (≤ {})", dec(cuts[2], 1)),
        format!("top quarter (> {})", dec(cuts[2], 1)),
    ];

    // Marker for each of the top five: the neighborhood's square nearest the
    // mean position of all its squares.
    let markers = scores
        .iter()
        .take(5)
        .enumerate()
        .filter_map(|(rank, &(_, i, _))| {
            let id = ctx.order[i];
            let own: Vec<usize> = (0..owner.len()).filter(|&c| owner[c] == id).collect();
            if own.is_empty() {
                return None;
            }
            let col = |c: usize| (c % citymap::COLS) as f64;
            let row = |c: usize| (c / citymap::COLS) as f64;
            let n = own.len() as f64;
            let (mc, mr) = own.iter().fold((0.0, 0.0), |(a, b), &c| (a + col(c), b + row(c)));
            let (mc, mr) = (mc / n, mr / n);
            let centre = own.iter().copied().min_by(|&a, &b| {
                let d = |c: usize| (col(c) - mc).powi(2) + (row(c) - mr).powi(2);
                d(a).total_cmp(&d(b)).then(a.cmp(&b))
            })?;
            Some(citymap::marker(centre, (rank + 1).to_string(), format!("#{} {}", rank + 1, ctx.name(id))))
        })
        .collect();

    citymap::grid(
        "Map: Neighborhood Priority Index by quarter, per 400 m square".into(),
        shades,
        levels,
        markers,
        "North is up; each square is 400 m across and takes the index of the neighborhood most of its records belong to. Numbers mark the top five.".into(),
    )
}
