use serde::{Deserialize, Serialize};

#[derive(Debug, Clone, Deserialize, Serialize)]
pub struct Settlement {
    pub officer_id: String,
    pub officer_name: String,
    pub amount_usd: f64,
    pub settlement_date: String,
    pub unit: String,
    pub district: String,
    pub case_type: String,
    pub report_year: i32,
    pub discipline_year: Option<i32>,
}

#[derive(Debug, Clone, Deserialize, Serialize)]
pub struct Incident {
    pub incident_id: String,
    pub officer_id: String,
    pub incident_date: String,
    pub district: String,
    pub unit: String,
    pub shift: String,
    pub crime_type: String,
    pub latitude: f64,
    pub longitude: f64,
}

#[derive(Debug, Clone)]
pub struct JoinedRecord<'a> {
    pub settlement: &'a Settlement,
    pub incident: &'a Incident,
}

#[derive(Debug, Clone)]
pub struct Metric {
    pub label: String,
    pub value: f64,
    pub detail: String,
}

#[derive(Debug, Default)]
pub struct Report {
    pub q1: Vec<Metric>,
    pub q2: Vec<Metric>,
    pub q3: Vec<Metric>,
    pub q4: Vec<Metric>,
    pub q5: Vec<Metric>,
    pub caveat: String,
}
