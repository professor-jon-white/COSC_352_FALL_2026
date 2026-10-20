use crate::{
    analysis::QueryConfig,
    concurrency::parallel_map,
    models::{Incident, JoinedRecord, Settlement},
};
use std::collections::HashMap;

/// Joins by the stable officer identifier. Each matching settlement/incident
/// pair is retained; absent identity linkage is never inferred from names.
pub fn join<'a>(
    settlements: &'a [Settlement],
    incidents: &'a [Incident],
    config: &QueryConfig,
) -> Vec<JoinedRecord<'a>> {
    let mut index: HashMap<&str, Vec<&Incident>> = HashMap::new();

    for incident in incidents {
        index
            .entry(&incident.officer_id)
            .or_default()
            .push(incident);
    }

    let partials = parallel_map(settlements, |settlement| {
        if !config.matches_district(&settlement.district) {
            return Vec::new();
        }

        index
            .get(settlement.officer_id.as_str())
            .into_iter()
            .flatten()
            .filter(|incident| config.matches_district(&incident.district))
            .map(|incident| JoinedRecord {
                settlement,
                incident,
            })
            .collect::<Vec<_>>()
    });

    partials.into_iter().flatten().collect()
}
