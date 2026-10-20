//! Q1: do vacant buildings and violent crime occupy the same places?

use super::{vacancy_tier, Context, Question, TIER_LABELS};
use crate::report::{dec, int, pct, times, BarChart, Block, Section, Table};
use crate::stats::{pearson, spearman};

pub struct VacancyAndViolence;

impl Question for VacancyAndViolence {
    fn id(&self) -> &'static str {
        "q1"
    }

    fn run(&self, ctx: &Context<'_>) -> Section {
        let rows: Vec<_> = ctx.order.iter().map(|&id| &ctx.nb[id as usize]).collect();
        let vacants: Vec<f64> = rows.iter().map(|r| f64::from(r.vacants)).collect();
        let violent: Vec<f64> = rows.iter().map(|r| f64::from(r.violent)).collect();
        let (r, rho) = (pearson(&vacants, &violent), spearman(&vacants, &violent));

        // [neighborhoods, vacants, violent, shootings] per tier
        let mut tiers = [[0u64; 4]; 4];
        for row in &rows {
            let t = &mut tiers[vacancy_tier(row.vacants)];
            t[0] += 1;
            t[1] += u64::from(row.vacants);
            t[2] += u64::from(row.violent);
            t[3] += u64::from(row.shootings);
        }
        let total_violent: u64 = tiers.iter().map(|t| t[2]).sum();
        let total_shootings: u64 = tiers.iter().map(|t| t[3]).sum();
        let per = |sum: u64, n: u64| if n > 0 { sum as f64 / n as f64 } else { 0.0 };

        let mut tier_table = Table::new(
            format!("Violent crime by neighborhood vacancy tier ({})", ctx.crime_period()),
            &["Tier", "#Neighborhoods", "#Vacant notices", "#Violent crimes", "#Violent / nbhd", "#Shootings / nbhd", "#Share of violent"],
        );
        let mut tier_bars = BarChart::new("Average violent crimes per neighborhood, by vacancy tier");
        for (label, t) in TIER_LABELS.iter().zip(&tiers) {
            tier_table.row(vec![
                label.to_string(),
                int(t[0]),
                int(t[1]),
                int(t[2]),
                dec(per(t[2], t[0]), 0),
                dec(per(t[3], t[0]), 1),
                pct(t[2] as f64, total_violent as f64),
            ]);
            tier_bars.bar(*label, per(t[2], t[0]), dec(per(t[2], t[0]), 0));
        }

        let mut top = Table::new(
            format!("Top {} neighborhoods by open vacant-building notices", ctx.top_n),
            &["Neighborhood", "#Vacant notices", "#Violent crimes", "#Shootings", "#Violent per vacant"],
        );
        let ranked = ctx.ranked(|r| f64::from(r.vacants));
        for &id in ranked.iter().take(ctx.top_n) {
            let row = &ctx.nb[id as usize];
            top.row(vec![
                ctx.name(id).to_string(),
                int(row.vacants.into()),
                int(row.violent.into()),
                int(row.shootings.into()),
                dec(per(row.violent.into(), row.vacants.into()), 1),
            ]);
        }

        // Block level: the same comparison through the address join.
        let active: Vec<_> = ctx.blocks.iter().filter(|b| b.crimes + b.vacants + b.outlets() > 0).collect();
        let (mut with, mut with_violent, mut without_violent) = (0u64, 0u64, 0u64);
        for b in &active {
            if b.vacants > 0 {
                with += 1;
                with_violent += u64::from(b.violent);
            } else {
                without_violent += u64::from(b.violent);
            }
        }
        let without = active.len() as u64 - with;
        let (rate_with, rate_without) = (per(with_violent, with), per(without_violent, without));
        let block_facts = Block::Facts {
            title: "Block-level view (hundred-blocks joined by normalised address)".into(),
            items: vec![
                ("Blocks with at least one vacant notice".into(), format!("{} of {} ({})", int(with), int(active.len() as u64), pct(with as f64, active.len() as f64))),
                ("Their share of block-attributed violent crime".into(), pct(with_violent as f64, (with_violent + without_violent) as f64)),
                ("Violent crimes per block, with vacants".into(), dec(rate_with, 2)),
                ("Violent crimes per block, without vacants".into(), dec(rate_without, 2)),
            ],
        };

        let hi = &tiers[3];
        let n = rows.len() as u64;
        let names: Vec<&str> = ranked.iter().take(3).map(|&id| ctx.name(id)).collect();
        let findings = vec![
            format!(
                "Vacancy and violence rise together across {n} neighborhoods: Spearman rank correlation {rho:.2} (Pearson {r:.2})."
            ),
            format!(
                "The {} neighborhoods with 100+ vacant notices are {} of neighborhoods but carry {} of violent crime and {} of shootings.",
                hi[0],
                pct(hi[0] as f64, n as f64),
                pct(hi[2] as f64, total_violent as f64),
                pct(hi[3] as f64, total_shootings as f64)
            ),
            format!(
                "At street level, a block with a vacant building averages {} the violent crime of a block without one ({} vs {}).",
                times(rate_with, rate_without),
                dec(rate_with, 2),
                dec(rate_without, 2)
            ),
            format!(
                "Suggested action: concentrate demolition, receivership and stabilisation funding in {}, which hold the most vacants.",
                names.join(", ")
            ),
        ];

        Section {
            id: self.id(),
            title: "Vacant buildings and violent crime".into(),
            stakeholder: "Mayor's Office and Department of Housing & Community Development".into(),
            question: "Where do vacant buildings and violent crime overlap, and how strongly does one track the other?".into(),
            findings,
            blocks: vec![Block::Bars(tier_bars), Block::Table(tier_table), Block::Table(top), block_facts],
            method: format!(
                "Joins Vacant Building Notices to NIBRS crime by neighborhood (name, with spatial and address fallbacks) and by hundred-block. \
                 Violent crime = homicide, rape, robbery, aggravated assault, {}. Counts are not population-adjusted, and correlation is not causation.",
                ctx.crime_period()
            ),
        }
    }
}
