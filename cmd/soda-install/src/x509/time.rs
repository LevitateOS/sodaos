use super::der::{Reader, TAG_GENTIME, TAG_UTCTIME};

// ---------------------------------------------------------------------------
// Validity times.
// ---------------------------------------------------------------------------

fn is_leap_year(year: i64) -> bool {
    year % 4 == 0 && (year % 100 != 0 || year % 400 == 0)
}

fn days_in_month(year: i64, month: u32) -> u32 {
    match month {
        1 | 3 | 5 | 7 | 8 | 10 | 12 => 31,
        4 | 6 | 9 | 11 => 30,
        2 => {
            if is_leap_year(year) {
                29
            } else {
                28
            }
        }
        _ => 0,
    }
}

fn days_from_civil(year: i64, month: u32, day: u32) -> i64 {
    let y = if month <= 2 { year - 1 } else { year };
    let era = if y >= 0 { y } else { y - 399 } / 400;
    let yoe = y - era * 400;
    let mp = ((month + 9) % 12) as i64;
    let doy = (153 * mp + 2) / 5 + day as i64 - 1;
    let doe = yoe * 365 + yoe / 4 - yoe / 100 + doy;
    era * 146097 + doe - 719468
}

/// Zone suffix: `Z`, or `±hhmm` with `hh <= 24`, `mm <= 59` — exactly the
/// offsets Go's `time.Parse` accepts that survive its format round-trip.
/// Returns the offset in seconds east of UTC.
fn parse_zone(zone: &[u8]) -> Option<i64> {
    if zone == b"Z" {
        return Some(0);
    }
    if zone.len() != 5 || (zone[0] != b'+' && zone[0] != b'-') {
        return None;
    }
    if !zone[1..].iter().all(|b| b.is_ascii_digit()) {
        return None;
    }
    let hour = u32::from(zone[1] - b'0') * 10 + u32::from(zone[2] - b'0');
    let min = u32::from(zone[3] - b'0') * 10 + u32::from(zone[4] - b'0');
    if hour > 24 || min > 59 {
        return None;
    }
    let offset = (hour * 60 + min) as i64 * 60;
    if zone[0] == b'-' {
        Some(-offset)
    } else {
        Some(offset)
    }
}

/// UTCTime two-digit years: 00-49 map to 2000-2049, 50-99 to 1950-1999.
fn utc_year(yy: u32) -> i64 {
    if yy <= 49 {
        2000 + yy as i64
    } else {
        1900 + yy as i64
    }
}

fn time_fields(contents: &[u8], digits: usize) -> Option<i64> {
    let datetime = contents.get(..digits)?;
    let zone = contents.get(digits..)?;
    if !datetime.iter().all(|b| b.is_ascii_digit()) {
        return None;
    }
    let offset = parse_zone(zone)?;
    let num = |from: usize, to: usize| -> u32 {
        datetime[from..to]
            .iter()
            .fold(0u32, |v, b| v * 10 + u32::from(*b - b'0'))
    };
    // 14 digits: YYYYMMDDHHMMSS. 12: YYMMDDHHMMSS. Otherwise the
    // minute-precision UTCTime fallback YYMMDDHHMM (10 digits).
    let (year, month, day, hour, min, sec) = match digits {
        14 => (
            num(0, 4) as i64,
            num(4, 6),
            num(6, 8),
            num(8, 10),
            num(10, 12),
            num(12, 14),
        ),
        12 => (
            utc_year(num(0, 2)),
            num(2, 4),
            num(4, 6),
            num(6, 8),
            num(8, 10),
            num(10, 12),
        ),
        _ => (
            utc_year(num(0, 2)),
            num(2, 4),
            num(4, 6),
            num(6, 8),
            num(8, 10),
            0,
        ),
    };
    if !(1..=12).contains(&month) {
        return None;
    }
    if day < 1 || day > days_in_month(year, month) {
        return None;
    }
    if hour > 23 || min > 59 || sec > 59 {
        return None;
    }
    let days = days_from_civil(year, month, day);
    Some(days * 86400 + hour as i64 * 3600 + min as i64 * 60 + sec as i64 - offset)
}

fn parse_utc_time(contents: &[u8]) -> Option<i64> {
    time_fields(contents, 12).or_else(|| time_fields(contents, 10))
}

fn parse_generalized_time(contents: &[u8]) -> Option<i64> {
    time_fields(contents, 14)
}

fn parse_time(reader: &mut Reader<'_>) -> Result<i64, String> {
    if reader.peek_tag(TAG_UTCTIME) {
        let contents = reader
            .read_asn1(TAG_UTCTIME)
            .ok_or_else(|| String::from("x509: malformed UTCTime"))?;
        parse_utc_time(contents).ok_or_else(|| String::from("x509: malformed UTCTime"))
    } else if reader.peek_tag(TAG_GENTIME) {
        let contents = reader
            .read_asn1(TAG_GENTIME)
            .ok_or_else(|| String::from("x509: malformed GeneralizedTime"))?;
        parse_generalized_time(contents)
            .ok_or_else(|| String::from("x509: malformed GeneralizedTime"))
    } else {
        Err(String::from("x509: unsupported time format"))
    }
}

pub(super) fn parse_validity(contents: &[u8]) -> Result<(i64, i64), String> {
    let mut reader = Reader::new(contents);
    let not_before = parse_time(&mut reader)?;
    let not_after = parse_time(&mut reader)?;
    Ok((not_before, not_after))
}
