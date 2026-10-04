//! Minimal time helpers (TEMPORARY: the sibling's canonical `timex.rs`
//! replaces this file at merge — other modules use only `parse_iso_deadline`
//! and `now_secs`).

/// Wall-clock seconds, like `int(time.time())`.
pub fn now_secs() -> i64 {
    unsafe { libc::time(std::ptr::null_mut()) as i64 }
}

/// Strict ISO-8601 deadline: `YYYY-MM-DDTHH:MM:SS` plus `Z` or a numeric
/// zone (`+HH:MM`, `-HH:MM`, `+HHMM`, `-HHMM`). Returns epoch seconds, or
/// `None` for any malformed or out-of-range value.
pub fn parse_iso_deadline(s: &str) -> Option<i64> {
    let bytes = s.as_bytes();
    if bytes.len() < 20 {
        return None;
    }
    // Fixed `YYYY-MM-DDTHH:MM:SS` prefix.
    if bytes[4] != b'-'
        || bytes[7] != b'-'
        || bytes[10] != b'T'
        || bytes[13] != b':'
        || bytes[16] != b':'
    {
        return None;
    }
    let digits = |lo: usize, hi: usize| -> Option<i64> {
        let mut value: i64 = 0;
        for b in &bytes[lo..hi] {
            if !b.is_ascii_digit() {
                return None;
            }
            value = value * 10 + (b - b'0') as i64;
        }
        Some(value)
    };
    let year = digits(0, 4)?;
    let month = digits(5, 7)?;
    let day = digits(8, 10)?;
    let hour = digits(11, 13)?;
    let minute = digits(14, 16)?;
    let second = digits(17, 19)?;
    if !(1..=12).contains(&month) || hour > 23 || minute > 59 || second > 59 {
        return None;
    }
    let leap = (year % 4 == 0 && year % 100 != 0) || year % 400 == 0;
    let dim = match month {
        1 | 3 | 5 | 7 | 8 | 10 | 12 => 31,
        4 | 6 | 9 | 11 => 30,
        _ => {
            if leap {
                29
            } else {
                28
            }
        }
    };
    if day < 1 || day > dim {
        return None;
    }
    // Zone suffix.
    let zone = &s[19..];
    let offset: i64 = if zone == "Z" {
        0
    } else if zone.len() == 6
        && (zone.starts_with('+') || zone.starts_with('-'))
        && zone.as_bytes()[3] == b':'
    {
        let sign = if zone.starts_with('-') { -1 } else { 1 };
        let zh: i64 = zone[1..3].parse().ok()?;
        let zm: i64 = zone[4..6].parse().ok()?;
        if zh > 23 || zm > 59 {
            return None;
        }
        sign * (zh * 3600 + zm * 60)
    } else if zone.len() == 5 && (zone.starts_with('+') || zone.starts_with('-')) {
        let sign = if zone.starts_with('-') { -1 } else { 1 };
        let zh: i64 = zone[1..3].parse().ok()?;
        let zm: i64 = zone[3..5].parse().ok()?;
        if zh > 23 || zm > 59 {
            return None;
        }
        sign * (zh * 3600 + zm * 60)
    } else {
        return None;
    };
    // Days from civil (Howard Hinnant) → epoch seconds, UTC sensed.
    let y = if month <= 2 { year - 1 } else { year };
    let era = y.div_euclid(400);
    let yoe = y - era * 400;
    let mp = (month + 9) % 12;
    let doy = (153 * mp + 2) / 5 + day - 1;
    let doe = yoe * 365 + yoe / 4 - yoe / 100 + doy;
    let days = era * 146097 + doe - 719468;
    Some(days * 86400 + hour * 3600 + minute * 60 + second - offset)
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn deadline_vectors() {
        assert_eq!(parse_iso_deadline("1970-01-01T00:00:00Z"), Some(0));
        assert_eq!(parse_iso_deadline("2030-01-01T00:00:00Z"), Some(1893456000));
        assert_eq!(
            parse_iso_deadline("2030-01-01T02:00:00+02:00"),
            Some(1893456000)
        );
        assert_eq!(
            parse_iso_deadline("2029-12-31T19:00:00-0500"),
            Some(1893456000)
        );
        assert_eq!(parse_iso_deadline("2024-02-29T12:00:00Z"), Some(1709208000));
        // Rejections.
        assert_eq!(parse_iso_deadline("2023-02-29T12:00:00Z"), None);
        assert_eq!(parse_iso_deadline("2030-13-01T00:00:00Z"), None);
        assert_eq!(parse_iso_deadline("2030-01-01T24:00:00Z"), None);
        assert_eq!(parse_iso_deadline("2030-01-01T00:00:61Z"), None);
        assert_eq!(parse_iso_deadline("2030-01-01 00:00:00Z"), None);
        assert_eq!(parse_iso_deadline("2030-01-01T00:00:00"), None);
        assert_eq!(parse_iso_deadline("2030-01-01T00:00:00+24:00"), None);
        assert_eq!(parse_iso_deadline("2030-01-01T00:00:00.000Z"), None);
        assert_eq!(parse_iso_deadline("not-a-date"), None);
        assert_eq!(parse_iso_deadline(""), None);
    }

    #[test]
    fn clock_sane() {
        let now = now_secs();
        assert!(now > 1_700_000_000 && now < 9_000_000_000);
    }
}
