use crate::models::{Incident, JoinedRecord, Metric, Report, Settlement};
use std::collections::{BTreeMap, HashMap, HashSet};

#[derive(Debug, Default)]
pub struct QueryConfig {
    district: Option<String>,
}

pub struct QueryBuilder {
    district: Option<String>,
}

impl QueryConfig {
    pub fn builder() -> QueryBuilder {
        QueryBuilder { district: None }
    }

    pub fn matches_district(&self, district: &str) -> bool {
        self.district
            .as_deref()
            .map_or(true, |d| d.eq_ignore_ascii_case(district))
    }
}

impl QueryBuilder {
    pub fn district(mut self, district: Option<String>) -> Self {
        self.district = district;
        self
    }

    pub fn build(self) -> QueryConfig {
        QueryConfig {
            district: self.district,
        }
    }
}

trait AnalysisStrategy {
    fn calculate(
        &self,
        settlements: &[Settlement],
        incidents: &[Incident],
        joined: &[JoinedRecord<'_>],
        config: &QueryConfig,
    ) -> Vec<Metric>;
}

struct DistrictCost;
struct UnitHotspots;
struct Timeline;
struct Recurrence;
struct Risk;

impl AnalysisStrategy for DistrictCost {
    fn calculate(
        &self,
        settlements: &[Settlement],
        incidents: &[Incident],
        _: &[JoinedRecord<'_>],
        config: &QueryConfig,
    ) -> Vec<Metric> {
        let mut costs: BTreeMap<&str, f64> = BTreeMap::new();

        for settlement in settlements
            .iter()
            .filter(|row| config.matches_district(&row.district))
        {
            *costs.entry(&settlement.district).or_default() += settlement.amount_usd;
        }

        let mut counts: HashMap<&str, usize> = HashMap::new();

        for incident in incidents
            .iter()
            .filter(|row| config.matches_district(&row.district))
        {
            *counts.entry(&incident.district).or_default() += 1;
        }

        for district in counts.keys() {
            costs.entry(district).or_default();
        }

        costs
            .into_iter()
            .map(|(district, amount)| Metric {
                label: district.into(),
                value: amount,
                detail: format!(
                    "${amount:.0} settlements; {} incidents",
                    counts.get(district).copied().unwrap_or(0)
                ),
            })
            .collect()
    }
}

impl AnalysisStrategy for UnitHotspots {
    fn calculate(
        &self,
        _: &[Settlement],
        _: &[Incident],
        joined: &[JoinedRecord<'_>],
        _: &QueryConfig,
    ) -> Vec<Metric> {
        let mut groups: HashMap<String, (usize, f64, HashSet<String>)> = HashMap::new();

        for row in joined {
            let group = groups
                .entry(format!("{} / {}", row.incident.unit, row.incident.shift))
                .or_default();
            group.0 += 1;
            group.1 += row.settlement.amount_usd;
            group.2.insert(row.incident.incident_id.clone());
        }

        let mut metrics: Vec<_> = groups
            .into_iter()
            .map(|(label, (pairs, payout, incident_ids))| Metric {
                label,
                value: pairs as f64,
                detail: format!(
                    "{pairs} matched pairs, {} distinct incidents, ${payout:.0} associated payout",
                    incident_ids.len()
                ),
            })
            .collect();

        metrics.sort_by(|a, b| b.value.total_cmp(&a.value));
        metrics.truncate(8);
        metrics
    }
}

impl AnalysisStrategy for Timeline {
    fn calculate(
        &self,
        settlements: &[Settlement],
        _: &[Incident],
        _: &[JoinedRecord<'_>],
        config: &QueryConfig,
    ) -> Vec<Metric> {
        let mut years: BTreeMap<i32, (usize, usize)> = BTreeMap::new();

        for settlement in settlements
            .iter()
            .filter(|row| config.matches_district(&row.district))
        {
            years.entry(settlement.report_year).or_default().0 += 1;

            if let Some(year) = settlement.discipline_year {
                years.entry(year).or_default().1 += 1;
            }
        }

        years
            .into_iter()
            .map(|(year, (reports, actions))| Metric {
                label: year.to_string(),
                value: reports as f64,
                detail: format!(
                    "{reports} modeled misconduct reports; {actions} modeled disciplinary actions"
                ),
            })
            .collect()
    }
}

impl AnalysisStrategy for Recurrence {
    fn calculate(
        &self,
        settlements: &[Settlement],
        incidents: &[Incident],
        _: &[JoinedRecord<'_>],
        config: &QueryConfig,
    ) -> Vec<Metric> {
        let mut counts: HashMap<&str, (usize, f64)> = HashMap::new();

        for settlement in settlements
            .iter()
            .filter(|row| config.matches_district(&row.district))
        {
            let entry = counts.entry(&settlement.officer_id).or_default();
            entry.0 += 1;
            entry.1 += settlement.amount_usd;
        }

        for incident in incidents
            .iter()
            .filter(|row| config.matches_district(&row.district))
        {
            counts.entry(&incident.officer_id).or_default().0 += 1;
        }

        let total_cost: f64 = counts.values().map(|value| value.1).sum();
        let mean_cost = if counts.is_empty() {
            0.0
        } else {
            total_cost / counts.len() as f64
        };

        let mut metrics: Vec<_> = counts
            .into_iter()
            .filter(|(_, value)| value.0 > 1)
            .map(|(officer_id, (records, cost))| Metric {
                label: officer_id.into(),
                value: records as f64,
                detail: format!(
                    "{records} source records; ${cost:.0} modeled settlement cost; {:.1}x mean officer cost",
                    if mean_cost > 0.0 {
                        cost / mean_cost
                    } else {
                        0.0
                    }
                ),
            })
            .collect();

        metrics.sort_by(|a, b| b.value.total_cmp(&a.value));
        metrics.truncate(10);
        metrics
    }
}

impl AnalysisStrategy for Risk {
    fn calculate(
        &self,
        settlements: &[Settlement],
        incidents: &[Incident],
        _: &[JoinedRecord<'_>],
        config: &QueryConfig,
    ) -> Vec<Metric> {
        let mut districts: HashMap<&str, (f64, usize)> = HashMap::new();

        for settlement in settlements
            .iter()
            .filter(|row| config.matches_district(&row.district))
        {
            districts
                .entry(&settlement.district)
                .or_default()
                .0 += settlement.amount_usd;
        }

        for incident in incidents
            .iter()
            .filter(|row| config.matches_district(&row.district))
        {
            districts.entry(&incident.district).or_default().1 += 1;
        }

        let max_cost = districts
            .values()
            .map(|value| value.0)
            .fold(0.0, f64::max)
            .max(1.0);
        let max_incidents = districts
            .values()
            .map(|value| value.1)
            .max()
            .unwrap_or(1)
            .max(1) as f64;

        let mut metrics: Vec<_> = districts
            .into_iter()
            .map(|(district, (cost, incidents))| {
                let score = 100.0
                    * (0.6 * cost / max_cost + 0.4 * incidents as f64 / max_incidents);

                Metric {
                    label: district.into(),
                    value: score,
                    detail: format!("{score:.1}/100; ${cost:.0} liability, {incidents} incidents"),
                }
            })
            .collect();

        metrics.sort_by(|a, b| b.value.total_cmp(&a.value));
        metrics
    }
}

pub fn run_all(
    settlements: &[Settlement],
    incidents: &[Incident],
    joined: &[JoinedRecord<'_>],
    config: &QueryConfig,
) -> Report {
    let strategies: [&dyn AnalysisStrategy; 5] = [
        &DistrictCost,
        &UnitHotspots,
        &Timeline,
        &Recurrence,
        &Risk,
    ];

    Report {
        q1: strategies[0].calculate(settlements, incidents, joined, config),
        q2: strategies[1].calculate(settlements, incidents, joined, config),
        q3: strategies[2].calculate(settlements, incidents, joined, config),
        q4: strategies[3].calculate(settlements, incidents, joined, config),
        q5: strategies[4].calculate(settlements, incidents, joined, config),
        caveat: "Demonstration inputs are synthetic. Statistics are descriptive and do not establish causal relationships. A joined row uses officer_id supplied in both files; district, unit, and calendar-year comparisons depend on the supplied normalized fields.".into(),
    }
}
