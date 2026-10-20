//! Row parsers for the four CSV datasets.

use super::chunked::{CsvSchema, ParseCtx, Row, Scratch};
use crate::geo;
use crate::model::*;
use crate::normalize;

fn intern_neighborhood(ctx: &ParseCtx<'_>, raw: &str, scratch: &mut Scratch, labelled: bool) -> NeighborhoodId {
    if !normalize::neighborhood_key(raw, &mut scratch.key) {
        return NO_NEIGHBORHOOD;
    }
    ctx.neighborhoods.intern_labelled(&scratch.key, labelled.then_some(raw)) as NeighborhoodId
}

fn intern_block(ctx: &ParseCtx<'_>, raw: &str, scratch: &mut Scratch) -> BlockId {
    if normalize::block_key(raw, &mut scratch.key) {
        ctx.blocks.intern(&scratch.key)
    } else {
        NO_BLOCK
    }
}

pub struct CrimeSchema;

impl CsvSchema for CrimeSchema {
    type Record = Crime;
    const DATASET: &'static str = "NIBRS Group A Crime";
    const COLUMNS: &'static [&'static str] =
        &["CrimeDateTime", "Description", "Shooting", "Location", "Neighborhood", "Latitude", "Longitude"];

    fn parse(row: &Row<'_>, ctx: &ParseCtx<'_>, scratch: &mut Scratch) -> Option<Crime> {
        let ts = normalize::parse_us_datetime(row.get(0))?;
        Some(Crime {
            ts,
            point: geo::project_text(row.get(5), row.get(6)),
            block: intern_block(ctx, row.get(3), scratch),
            // Crime spells neighborhoods in upper case; a mixed-case label is
            // supplied by 311 or the vacant-notice data when they share the name.
            neighborhood: intern_neighborhood(ctx, row.get(4), scratch, false),
            offense: Offense::from_description(row.get(1)),
            hour: normalize::eastern_hour(ts),
            shooting: row.get(2).eq_ignore_ascii_case("Y"),
        })
    }
}

pub struct ServiceRequestSchema;

impl CsvSchema for ServiceRequestSchema {
    type Record = ServiceRequest;
    const DATASET: &'static str = "311 Service Requests 2025";
    const COLUMNS: &'static [&'static str] =
        &["SRType", "CreatedDate", "SRStatus", "CloseDate", "Address", "Neighborhood", "Latitude", "Longitude"];

    fn parse(row: &Row<'_>, ctx: &ParseCtx<'_>, scratch: &mut Scratch) -> Option<ServiceRequest> {
        let sr_type = row.get(0);
        let created = normalize::parse_us_datetime(row.get(1))?;
        let status = row.get(2);
        let close_secs = match normalize::parse_us_datetime(row.get(3)) {
            Some(closed) if starts_with_ignore_case(status, "Closed") => closed.saturating_sub(created),
            _ => NOT_CLOSED,
        };
        Some(ServiceRequest {
            close_secs,
            point: geo::project_text(row.get(6), row.get(7)),
            block: intern_block(ctx, row.get(4), scratch),
            neighborhood: intern_neighborhood(ctx, row.get(5), scratch, true),
            sr_type: ctx.sr_types.intern(sr_type) as u16,
            category: SrCategory::from_type(sr_type),
            duplicate: contains_ignore_case(status, "Duplicate"),
        })
    }
}

pub struct VacantNoticeSchema;

impl CsvSchema for VacantNoticeSchema {
    type Record = VacantNotice;
    const DATASET: &'static str = "Vacant Building Notices";
    const COLUMNS: &'static [&'static str] = &["NoticeNum", "Neighborhood", "Address"];

    fn parse(row: &Row<'_>, ctx: &ParseCtx<'_>, scratch: &mut Scratch) -> Option<VacantNotice> {
        if row.get(0).is_empty() {
            return None;
        }
        Some(VacantNotice {
            block: intern_block(ctx, row.get(2), scratch),
            neighborhood: intern_neighborhood(ctx, row.get(1), scratch, true),
        })
    }
}

pub struct LicenseSchema;

impl CsvSchema for LicenseSchema {
    type Record = LicenseRow;
    const DATASET: &'static str = "Liquor Licenses";
    const COLUMNS: &'static [&'static str] =
        &["TradeName", "LicenseYear", "LicenseStatus", "EstablishmentDesc", "AddrStreet"];

    fn parse(row: &Row<'_>, ctx: &ParseCtx<'_>, scratch: &mut Scratch) -> Option<LicenseRow> {
        let year = row.get(1).parse().ok()?;
        Some(LicenseRow {
            name: row.get(0).to_ascii_uppercase().into_boxed_str(),
            block: intern_block(ctx, row.get(4), scratch),
            year,
            kind: OutletKind::from_description(row.get(3)),
            closed: row.get(2).eq_ignore_ascii_case("Closed"),
        })
    }
}
