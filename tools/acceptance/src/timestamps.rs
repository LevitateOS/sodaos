use crate::error::Error;

/// Current UTC time in Go's `time.RFC3339Nano` shape: trailing zero
/// fractional digits are trimmed, and a whole second has no fraction.
pub fn now_rfc3339_nano() -> String {
    let now = std::time::SystemTime::now();
    let elapsed = now
        .duration_since(std::time::UNIX_EPOCH)
        .unwrap_or_default();
    format_unix_nano(elapsed.as_secs() as i64, elapsed.subsec_nanos())
}

fn format_unix_nano(secs: i64, nanos: u32) -> String {
    let days = secs.div_euclid(86_400);
    let time = secs.rem_euclid(86_400);
    let (year, month, day) = civil_from_days(days);
    let (hour, minute, second) = (time / 3_600, (time % 3_600) / 60, time % 60);
    let mut out = format!("{year:04}-{month:02}-{day:02}T{hour:02}:{minute:02}:{second:02}");
    if nanos != 0 {
        let mut frac = format!("{nanos:09}");
        while frac.ends_with('0') {
            frac.pop();
        }
        out.push('.');
        out.push_str(&frac);
    }
    out.push('Z');
    out
}

/// Days since the Unix epoch to civil date (Hinnant's algorithm).
fn civil_from_days(days: i64) -> (i64, u32, u32) {
    let z = days + 719_468;
    let era = z.div_euclid(146_097);
    let doe = z.rem_euclid(146_097);
    let yoe = (doe - doe / 1_460 + doe / 36_524 - doe / 146_096) / 365;
    let year = yoe + era * 400;
    let doy = doe - (365 * yoe + yoe / 4 - yoe / 100);
    let mp = (5 * doy + 2) / 153;
    let day = (doy - (153 * mp + 2) / 5 + 1) as u32;
    let month = (if mp < 10 { mp + 3 } else { mp - 9 }) as u32;
    (if month <= 2 { year + 1 } else { year }, month, day)
}

/// Validate strict RFC3339 timestamps, like Go's `time.Time` JSON decode:
/// real calendar dates, `T` separator, second precision with an optional
/// 1-9 digit fraction, and `Z` or a numeric offset.
pub fn validate_rfc3339(text: &str) -> Result<(), Error> {
    let err = || Error::msg(format!("invalid timestamp: {text:?}"));
    let (date, rest) = text.split_once('T').ok_or_else(err)?;
    let year: i64 = date
        .get(0..4)
        .and_then(|s| s.parse().ok())
        .ok_or_else(err)?;
    let month: u32 = date
        .get(5..7)
        .and_then(|s| s.parse().ok())
        .ok_or_else(err)?;
    let day: u32 = date
        .get(8..10)
        .and_then(|s| s.parse().ok())
        .ok_or_else(err)?;
    if date.len() != 10 || date.as_bytes()[4] != b'-' || date.as_bytes()[7] != b'-' {
        return Err(err());
    }
    if !(1..=12).contains(&month) || day == 0 || day > days_in_month(year, month) {
        return Err(err());
    }
    let (clock, zone) = split_zone(rest).ok_or_else(err)?;
    let hour: u32 = clock
        .get(0..2)
        .and_then(|s| s.parse().ok())
        .ok_or_else(err)?;
    let minute: u32 = clock
        .get(3..5)
        .and_then(|s| s.parse().ok())
        .ok_or_else(err)?;
    let second: u32 = clock
        .get(6..8)
        .and_then(|s| s.parse().ok())
        .ok_or_else(err)?;
    if clock.len() < 8 || clock.as_bytes()[2] != b':' || clock.as_bytes()[5] != b':' {
        return Err(err());
    }
    if hour > 23 || minute > 59 || second > 59 {
        return Err(err());
    }
    if clock.len() > 8 {
        let frac = clock.get(8..).ok_or_else(err)?;
        if !frac.starts_with('.')
            || frac.len() < 2
            || frac.len() > 10
            || !frac[1..].bytes().all(|b| b.is_ascii_digit())
        {
            return Err(err());
        }
    }
    match zone {
        "Z" => Ok(()),
        _ => {
            let sign = zone.as_bytes().first().ok_or_else(err)?;
            if *sign != b'+' && *sign != b'-' {
                return Err(err());
            }
            let hour: u32 = zone
                .get(1..3)
                .and_then(|s| s.parse().ok())
                .ok_or_else(err)?;
            let minute: u32 = zone
                .get(4..6)
                .and_then(|s| s.parse().ok())
                .ok_or_else(err)?;
            if zone.len() != 6 || zone.as_bytes()[3] != b':' || hour > 23 || minute > 59 {
                return Err(err());
            }
            Ok(())
        }
    }
}

fn split_zone(rest: &str) -> Option<(&str, &str)> {
    if let Some(clock) = rest.strip_suffix('Z') {
        return Some((clock, "Z"));
    }
    let bytes = rest.as_bytes();
    for i in (0..bytes.len()).rev() {
        if bytes[i] == b'+' || (bytes[i] == b'-' && i > 7) {
            return Some((&rest[..i], &rest[i..]));
        }
    }
    None
}

fn days_in_month(year: i64, month: u32) -> u32 {
    match month {
        1 | 3 | 5 | 7 | 8 | 10 | 12 => 31,
        4 | 6 | 9 | 11 => 30,
        _ => {
            if year % 4 == 0 && (year % 100 != 0 || year % 400 == 0) {
                29
            } else {
                28
            }
        }
    }
}

#[cfg(test)]
mod tests;
