//! Q2: is violence concentrated on blocks with liquor outlets?

use std::collections::BTreeMap;

use rayon::prelude::*;

use super::{BlockAgg, Context, Question};
use crate::model::*;
use crate::report::{dec, int, pct, times, BarChart, Block, Section, Table};
use crate::stats::percentile;

pub struct LiquorOutletsAndViolence;

/// (blocks, violent crimes, all crimes) over the blocks passing `filter`.
fn class_totals(blocks: &[&BlockAgg], filter: impl Fn(&BlockAgg) -> bool) -> (u64, u64, u64) {
    blocks.iter().filter(|b| filter(b)).fold((0, 0, 0), |acc, b| (acc.0 + 1, acc.1 + u64::from(b.violent), acc.2 + u64::from(b.crimes)))
}

impl Question for LiquorOutletsAndViolence {
    fn id(&self) -> &'static str {
        "q2"
    }

    fn run(&self, ctx: &Context<'_>) -> Section {
        let store = ctx.store;
        let active: Vec<&BlockAgg> = ctx.blocks.iter().filter(|b| b.crimes + b.vacants + b.outlets() > 0).collect();
        let none = class_totals(&active, |b| b.outlets() == 0);
        let rate = |t: (u64, u64, u64)| if t.0 > 0 { t.1 as f64 / t.0 as f64 } else { 0.0 };
        let all_rate = |t: (u64, u64, u64)| if t.0 > 0 { t.2 as f64 / t.0 as f64 } else { 0.0 };

        let classes: [(&str, (u64, u64, u64)); 5] = [
            ("Tavern / bar", class_totals(&active, |b| b.taverns > 0)),
            ("Package goods store", class_totals(&active, |b| b.package > 0)),
            ("Restaurant", class_totals(&active, |b| b.restaurants > 0)),
            ("Any licensed outlet", class_totals(&active, |b| b.outlets() > 0)),
            ("No licensed outlet", none),
        ];
        let mut class_table = Table::new(
            format!("Crime per hundred-block by licence type on the block ({})", ctx.crime_period()),
            &["Block has", "#Blocks", "#Violent crimes", "#Violent / block", "#All crimes / block", "#vs no-outlet blocks"],
        );
        let mut class_bars = BarChart::new("Violent crimes per block");
        for (label, t) in classes {
            class_table.row(vec![label.into(), int(t.0), int(t.1), dec(rate(t), 2), dec(all_rate(t), 1), times(rate(t), rate(none))]);
            class_bars.bar(label, rate(t), dec(rate(t), 2));
        }

        // Control for location: compare outlet and non-outlet blocks inside the
        // same neighborhood, so downtown is only ever compared with downtown.
        let mut per_nb: BTreeMap<NeighborhoodId, [(u64, u64); 2]> = BTreeMap::new();
        for (i, b) in ctx.blocks.iter().enumerate() {
            let nb = store.block_neighborhood.get(i).copied().unwrap_or(NO_NEIGHBORHOOD);
            if nb == NO_NEIGHBORHOOD || b.crimes + b.vacants + b.outlets() == 0 {
                continue;
            }
            let slot = &mut per_nb.entry(nb).or_default()[usize::from(b.outlets() == 0)];
            slot.0 += 1;
            slot.1 += u64::from(b.violent);
        }
        let mut ratios: Vec<f64> = per_nb
            .values()
            .filter(|[with, without]| with.0 >= 3 && without.0 >= 3 && without.1 > 0)
            .map(|[with, without]| (with.1 as f64 / with.0 as f64) / (without.1 as f64 / without.0 as f64))
            .collect();
        ratios.sort_by(f64::total_cmp);
        let median_ratio = percentile(&ratios, 50.0);
        let higher = ratios.iter().filter(|&&r| r > 1.0).count();

        // When does violence happen? 3-hour bins, licensed bar/package blocks vs the rest.
        let bins = store
            .crimes
            .par_iter()
            .filter(|c| c.offense.is_violent())
            .fold(
                || [[0u64; 8]; 2],
                |mut bins, c| {
                    if let Some(b) = ctx.blocks.get(c.block as usize) {
                        bins[usize::from(b.taverns + b.package == 0)][usize::from(c.hour / 3)] += 1;
                    }
                    bins
                },
            )
            .reduce(
                || [[0u64; 8]; 2],
                |mut a, b| {
                    for (x, y) in a.iter_mut().flatten().zip(b.iter().flatten()) {
                        *x += y;
                    }
                    a
                },
            );
        let totals = [bins[0].iter().sum::<u64>() as f64, bins[1].iter().sum::<u64>() as f64];
        let mut hours = BarChart::new("Violent crime on bar / package-store blocks by time of day (Eastern), vs all other blocks");
        for (bin, (&here, &elsewhere)) in bins[0].iter().zip(&bins[1]).enumerate() {
            let counts = [here, elsewhere];
            let share = |class: usize| if totals[class] > 0.0 { counts[class] as f64 / totals[class] * 100.0 } else { 0.0 };
            hours.bar(
                format!("{:02}:00–{:02}:59", bin * 3, bin * 3 + 2),
                share(0),
                format!("{:.1}%  (elsewhere {:.1}%)", share(0), share(1)),
            );
        }
        let late = |class: usize| pct((bins[class][7] + bins[class][0]) as f64, totals[class]);

        // Highest-violence licensed blocks, with the establishments on them.
        let mut by_block: BTreeMap<BlockId, Vec<&str>> = BTreeMap::new();
        for o in store.outlets.iter().filter(|o| o.block != NO_BLOCK) {
            by_block.entry(o.block).or_default().push(&o.name);
        }
        let mut licensed: Vec<(BlockId, &BlockAgg)> = by_block.keys().map(|&id| (id, &ctx.blocks[id as usize])).collect();
        licensed.sort_by(|a, b| b.1.violent.cmp(&a.1.violent).then_with(|| store.block(a.0).cmp(store.block(b.0))));
        let mut top = Table::new(
            format!("Top {} licensed blocks by violent crime", ctx.top_n),
            &["Block", "Neighborhood", "Licensed outlets", "#Violent", "#All crimes"],
        );
        for (id, b) in licensed.iter().take(ctx.top_n) {
            let names = &by_block[id];
            let mut listed = names.iter().take(2).copied().collect::<Vec<_>>().join("; ");
            if names.len() > 2 {
                listed.push_str(&format!(" +{}", names.len() - 2));
            }
            top.row(vec![
                store.block(*id).to_string(),
                store.neighborhood(store.block_neighborhood[*id as usize]).to_string(),
                listed,
                int(b.violent.into()),
                int(b.crimes.into()),
            ]);
        }

        let any = classes[3].1;
        let total_violent = any.1 + none.1;
        let on_crime_block = by_block.keys().filter(|&&id| ctx.blocks[id as usize].crimes > 0).count();
        let placed = store.outlets.iter().filter(|o| o.neighborhood != NO_NEIGHBORHOOD).count();
        let facts = Block::Facts {
            title: "Join quality".into(),
            items: vec![
                ("Establishments licensed in the latest licence year".into(), int(store.outlets.len() as u64)),
                ("Placed in a neighborhood via the address join".into(), pct(placed as f64, store.outlets.len() as f64)),
                ("Licensed blocks that also appear in crime data".into(), format!("{} of {}", int(on_crime_block as u64), int(by_block.len() as u64))),
                ("Neighborhoods in the within-neighborhood comparison".into(), int(ratios.len() as u64)),
            ],
        };

        let findings = vec![
            format!(
                "Blocks with a licensed outlet are {} of active blocks but the scene of {} of violent crime: {} violent crimes per block against {} elsewhere ({}).",
                pct(any.0 as f64, (any.0 + none.0) as f64),
                pct(any.1 as f64, total_violent as f64),
                dec(rate(any), 2),
                dec(rate(none), 2),
                times(rate(any), rate(none))
            ),
            format!(
                "The gap survives a like-for-like comparison: inside the same neighborhood, outlet blocks have a median {median_ratio:.1}x the violent crime of other blocks, and are higher in {higher} of {} neighborhoods.",
                ratios.len()
            ),
            {
                let mut by_type = [classes[0], classes[1], classes[2]];
                by_type.sort_by(|a, b| rate(b.1).total_cmp(&rate(a.1)));
                format!(
                    "By licence type, violent crimes per block run highest for {} ({}), then {} ({}), then {} ({}).",
                    by_type[0].0.to_lowercase(),
                    dec(rate(by_type[0].1), 2),
                    by_type[1].0.to_lowercase(),
                    dec(rate(by_type[1].1), 2),
                    by_type[2].0.to_lowercase(),
                    dec(rate(by_type[2].1), 2)
                )
            },
            format!(
                "Timing: {} of violent crime on bar and package-store blocks falls between 9 PM and 3 AM, versus {} on other blocks.",
                late(0),
                late(1)
            ),
            "Suggested action: prioritise licence-renewal reviews and joint Liquor Board / BPD inspections at the blocks listed below, starting with package goods stores.".into(),
        ];

        Section {
            id: self.id(),
            title: "Liquor outlets and block-level violence".into(),
            stakeholder: "Board of Liquor License Commissioners and Baltimore Police Department".into(),
            question: "Do blocks with liquor outlets see more violent crime than comparable blocks, and which licensed blocks stand out?".into(),
            findings,
            blocks: vec![Block::Bars(class_bars), Block::Table(class_table), Block::Bars(hours), Block::Table(top), facts],
            method: "Licence rows (one per licensee per year since 2003) are reduced to establishments licensed in the latest year. \
                     Licences carry no coordinates, so their addresses and crime locations are both normalised to hundred-block keys and hash-joined. \
                     Licence-type rows overlap when a block hosts several types. Active blocks are those appearing in crime, vacancy or licence data; block '0' is a street's unit block (numbers 1-99). \
                     Outlets cluster on busy commercial streets, so the within-neighborhood ratio is the fairer figure; neither proves causation."
                .into(),
        }
    }
}
