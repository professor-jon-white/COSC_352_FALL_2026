//! Reformatting of raw text fields into join keys and typed values.
//!
//! The datasets describe the same places in different dialects:
//! crime says `1600 E PRATT ST`, liquor licences say `1601-03 PRATT STREET EAST`,
//! 311 says `1611 E PRATT ST, Baltimore City, 21231`. `block_key` reduces all
//! three to the hundred-block key `1600 E PRATT ST`. Every function writes
//! into a caller-supplied buffer so the hot parsing loops do not allocate.

/// Upper-cases and collapses whitespace. Returns false for an empty name.
pub fn neighborhood_key(raw: &str, out: &mut String) -> bool {
    out.clear();
    for token in raw.split_ascii_whitespace() {
        if !out.is_empty() {
            out.push(' ');
        }
        out.extend(token.chars().map(|c| c.to_ascii_uppercase()));
    }
    !out.is_empty()
}

/// "BUTCHER'S HILL" -> "Butcher's Hill"; used when no dataset supplied a
/// mixed-case spelling of a neighborhood.
pub fn title_case(s: &str) -> String {
    let mut out = String::with_capacity(s.len());
    let mut start = true;
    for c in s.chars() {
        if start {
            out.extend(c.to_uppercase());
        } else {
            out.extend(c.to_lowercase());
        }
        start = matches!(c, ' ' | '-' | '/' | '(');
    }
    out
}

fn direction(token: &str) -> Option<char> {
    const WORDS: [(&str, char); 4] = [("NORTH", 'N'), ("SOUTH", 'S'), ("EAST", 'E'), ("WEST", 'W')];
    if token.len() == 1 {
        let c = token.as_bytes()[0].to_ascii_uppercase() as char;
        return matches!(c, 'N' | 'S' | 'E' | 'W').then_some(c);
    }
    WORDS.iter().find(|(w, _)| token.eq_ignore_ascii_case(w)).map(|&(_, c)| c)
}

fn street_type(token: &str) -> Option<&'static str> {
    const TYPES: [(&str, &str); 19] = [
        ("STREET", "ST"),
        ("AVENUE", "AVE"),
        ("BOULEVARD", "BLVD"),
        ("ROAD", "RD"),
        ("DRIVE", "DR"),
        ("LANE", "LN"),
        ("PLACE", "PL"),
        ("COURT", "CT"),
        ("HIGHWAY", "HWY"),
        ("PARKWAY", "PKWY"),
        ("TERRACE", "TER"),
        ("CIRCLE", "CIR"),
        ("SQUARE", "SQ"),
        ("ALLEY", "ALY"),
        ("ST", "ST"),
        ("AVE", "AVE"),
        ("BLVD", "BLVD"),
        ("RD", "RD"),
        ("DR", "DR"),
    ];
    TYPES.iter().find(|(long, _)| token.eq_ignore_ascii_case(long)).map(|&(_, short)| short)
}

/// Normalises a street address to its hundred-block: `<block> [dir] <name> <type>`.
/// Returns false when the text has no house number or no street name.
pub fn block_key(raw: &str, out: &mut String) -> bool {
    out.clear();
    let street = raw.split(',').next().unwrap_or("");
    let mut words = street.split_ascii_whitespace();
    let Some(first) = words.next() else { return false };

    // "1301-03" and "53-55" are ranges on one block; only the leading number matters.
    let digits = first.bytes().take_while(u8::is_ascii_digit).count();
    if digits == 0 || digits > 6 {
        return false;
    }
    let Ok(number) = first[..digits].parse::<u32>() else { return false };

    let mut tokens: [&str; 8] = [""; 8];
    let mut n = 0;
    for word in words {
        let word = word.trim_matches('.');
        let is_unit = ["APT", "UNIT", "SUITE", "STE", "REAR"].iter().any(|u| word.eq_ignore_ascii_case(u));
        if is_unit || word.starts_with('#') || n == tokens.len() {
            break;
        }
        if word.is_empty() || word == "1/2" {
            continue;
        }
        tokens[n] = word;
        n += 1;
    }
    if n == 0 {
        return false;
    }

    // Liquor licences put the direction last ("PRATT STREET EAST"); everything
    // else puts it first ("E PRATT ST"). A lone direction word that is itself
    // the street name ("NORTH AVE") must be left alone.
    let mut dir = None;
    let (mut lo, mut hi) = (0, n);
    if n >= 3 && direction(tokens[n - 1]).is_some() {
        dir = direction(tokens[n - 1]);
        hi -= 1;
    } else if n >= 2
        && direction(tokens[0]).is_some()
        && (tokens[0].len() == 1 || (n >= 3 && street_type(tokens[1]).is_none()))
    {
        dir = direction(tokens[0]);
        lo += 1;
    }

    use std::fmt::Write;
    let _ = write!(out, "{}", number / 100 * 100);
    if let Some(d) = dir {
        out.push(' ');
        out.push(d);
    }
    for (i, token) in tokens[lo..hi].iter().enumerate() {
        out.push(' ');
        let is_last = lo + i + 1 == hi;
        match street_type(token).filter(|_| is_last && hi - lo > 1) {
            Some(short) => out.push_str(short),
            None => out.extend(token.chars().map(|c| c.to_ascii_uppercase())),
        }
    }
    true
}

/// Days since 1970-01-01 for a proleptic Gregorian date (Howard Hinnant's algorithm).
fn days_from_civil(y: i64, m: i64, d: i64) -> i64 {
    let y = if m <= 2 { y - 1 } else { y };
    let era = if y >= 0 { y } else { y - 399 } / 400;
    let yoe = y - era * 400;
    let doy = (153 * (if m > 2 { m - 3 } else { m + 9 }) + 2) / 5 + d - 1;
    let doe = yoe * 365 + yoe / 4 - yoe / 100 + doy;
    era * 146_097 + doe - 719_468
}

/// Inverse of `days_from_civil`: (year, month, day).
fn civil_from_days(days: i64) -> (i64, i64, i64) {
    let z = days + 719_468;
    let era = if z >= 0 { z } else { z - 146_096 } / 146_097;
    let doe = z - era * 146_097;
    let yoe = (doe - doe / 1_460 + doe / 36_524 - doe / 146_096) / 365;
    let doy = doe - (365 * yoe + yoe / 4 - yoe / 100);
    let mp = (5 * doy + 2) / 153;
    let day = doy - (153 * mp + 2) / 5 + 1;
    let month = if mp < 10 { mp + 3 } else { mp - 9 };
    (yoe + era * 400 + i64::from(month <= 2), month, day)
}

/// "Oct 2026" for a UTC timestamp; used to label the period the data covers.
pub fn month_label(ts: u32) -> String {
    const MONTHS: [&str; 12] = ["Jan", "Feb", "Mar", "Apr", "May", "Jun", "Jul", "Aug", "Sep", "Oct", "Nov", "Dec"];
    let (y, m, _) = civil_from_days(i64::from(ts) / 86_400);
    format!("{} {y}", MONTHS[(m - 1) as usize])
}

fn to_unix(y: i64, mo: i64, d: i64, h: i64, mi: i64, s: i64) -> Option<u32> {
    if !(1..=12).contains(&mo) || !(1..=31).contains(&d) || h > 23 || mi > 59 || s > 60 {
        return None;
    }
    u32::try_from(days_from_civil(y, mo, d) * 86_400 + h * 3_600 + mi * 60 + s).ok()
}

fn numbers<const N: usize>(text: &str, sep: char) -> Option<[i64; N]> {
    let mut out = [0i64; N];
    let mut parts = text.split(sep);
    for slot in &mut out {
        *slot = parts.next()?.trim().parse().ok()?;
    }
    Some(out)
}

/// Parses `10/4/2026 8:10:00 PM` (the ArcGIS CSV export format) as UTC seconds.
pub fn parse_us_datetime(text: &str) -> Option<u32> {
    let mut parts = text.split_ascii_whitespace();
    let [mo, d, y] = numbers::<3>(parts.next()?, '/')?;
    let (mut h, mut mi, mut s) = (0, 0, 0);
    if let Some(time) = parts.next() {
        [h, mi, s] = numbers::<3>(time, ':')?;
        match parts.next() {
            Some(half) if half.eq_ignore_ascii_case("PM") => h = h % 12 + 12,
            Some(half) if half.eq_ignore_ascii_case("AM") => h %= 12,
            _ => {}
        }
    }
    to_unix(y, mo, d, h, mi, s)
}

/// Seconds between the first Sunday on/after day `from` of a month and the epoch.
fn sunday_on_or_after(y: i64, m: i64, from: i64) -> i64 {
    let days = days_from_civil(y, m, from);
    let weekday = (days + 4).rem_euclid(7); // 0 = Sunday
    (days + (7 - weekday) % 7) * 86_400
}

/// Hour of day (0-23) in US Eastern time for a UTC timestamp, honouring DST
/// (second Sunday of March 2:00 EST to first Sunday of November 2:00 EDT).
pub fn eastern_hour(ts: u32) -> u8 {
    let ts = i64::from(ts);
    let (year, _, _) = civil_from_days(ts.div_euclid(86_400));
    let dst_start = sunday_on_or_after(year, 3, 8) + 7 * 3_600;
    let dst_end = sunday_on_or_after(year, 11, 1) + 6 * 3_600;
    let offset = if (dst_start..dst_end).contains(&ts) { 4 } else { 5 };
    ((ts - offset * 3_600).rem_euclid(86_400) / 3_600) as u8
}

#[cfg(test)]
mod tests {
    use super::*;

    fn block(raw: &str) -> Option<String> {
        let mut s = String::new();
        block_key(raw, &mut s).then_some(s)
    }

    #[test]
    fn block_keys_agree_across_dataset_dialects() {
        let expected = Some("1600 E PRATT ST".to_string());
        assert_eq!(block("1600 E PRATT ST"), expected);
        assert_eq!(block("1601-03 PRATT STREET EAST"), expected);
        assert_eq!(block("1611 E PRATT ST, Baltimore City, 21231"), expected);
        assert_eq!(block("1650 east pratt street"), expected);
    }

    #[test]
    fn block_keys_handle_awkward_streets() {
        assert_eq!(block("2301 NORTH AVENUE WEST").as_deref(), Some("2300 W NORTH AVE"));
        assert_eq!(block("2300 W NORTH AVE").as_deref(), Some("2300 W NORTH AVE"));
        assert_eq!(block("100 23RD STREET EAST").as_deref(), Some("100 E 23RD ST"));
        assert_eq!(block("1700 N BROADWAY").as_deref(), Some("1700 N BROADWAY"));
        assert_eq!(block("4 N LAKEWOOD AVE").as_deref(), Some("0 N LAKEWOOD AVE"));
        assert_eq!(block("6412-16 FRANKFORD AVENUE").as_deref(), Some("6400 FRANKFORD AVE"));
        assert_eq!(block("919 BENNETT PL").as_deref(), Some("900 BENNETT PL"));
        assert_eq!(block("EUTAW AND FAYETTE"), None);
        assert_eq!(block(""), None);
    }

    #[test]
    fn neighborhood_keys_ignore_case_and_spacing() {
        let (mut a, mut b) = (String::new(), String::new());
        assert!(neighborhood_key("Belair-Edison", &mut a));
        assert!(neighborhood_key("  BELAIR-EDISON ", &mut b));
        assert_eq!(a, b);
        assert!(!neighborhood_key("   ", &mut a));
        assert_eq!(title_case("WASHINGTON VILLAGE/PIGTOWN"), "Washington Village/Pigtown");
    }

    #[test]
    fn parses_export_timestamps() {
        assert_eq!(parse_us_datetime("1/1/1970 12:00:00 AM"), Some(0));
        assert_eq!(parse_us_datetime("1/1/2000 12:00:00 PM"), Some(946_728_000));
        assert_eq!(parse_us_datetime("10/4/2026 8:10:00 PM"), Some(1_791_144_600));
        assert_eq!(parse_us_datetime("not a date"), None);
        assert_eq!(parse_us_datetime("13/40/2026 1:00:00 AM"), None);
    }

    #[test]
    fn converts_utc_to_eastern_with_dst() {
        // 2026-07-01 16:00 UTC is noon EDT; 2026-01-15 17:00 UTC is noon EST.
        assert_eq!(eastern_hour(parse_us_datetime("7/1/2026 4:00:00 PM").unwrap()), 12);
        assert_eq!(eastern_hour(parse_us_datetime("1/15/2026 5:00:00 PM").unwrap()), 12);
        // DST began 2026-03-08 07:00 UTC.
        assert_eq!(eastern_hour(parse_us_datetime("3/8/2026 6:59:00 AM").unwrap()), 1);
        assert_eq!(eastern_hour(parse_us_datetime("3/8/2026 7:00:00 AM").unwrap()), 3);
        assert_eq!(civil_from_days(days_from_civil(2024, 2, 29)), (2024, 2, 29));
        assert_eq!(civil_from_days(days_from_civil(2025, 12, 31)), (2025, 12, 31));
        assert_eq!(month_label(parse_us_datetime("10/4/2026 8:10:00 PM").unwrap()), "Oct 2026");
    }
}
