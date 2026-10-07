//! Strict RFC3339 wire timestamps shared by Soda protocol adapters.
//!
//! The lexical gate deliberately narrows `time`'s RFC3339 parser to the shape
//! emitted by current Go wire values. Calendar and timestamp arithmetic are
//! delegated to `time`.

use time::format_description::well_known::Rfc3339;
use time::{OffsetDateTime, UtcOffset};

/// Parse a complete timestamp to Unix seconds and normalized nanoseconds.
pub fn parse(text: &str) -> Option<(i64, u32)> {
    if !wire_shape(text) {
        return None;
    }
    let value = OffsetDateTime::parse(text, &Rfc3339)
        .ok()?
        .checked_to_offset(UtcOffset::UTC)?;
    if !(1..=9999).contains(&value.year()) {
        return None;
    }
    Some((value.unix_timestamp(), value.nanosecond()))
}

/// Parse a timestamp as signed Unix nanoseconds.
pub fn parse_nanos(text: &str) -> Option<i128> {
    let (seconds, nanos) = parse(text)?;
    i128::from(seconds)
        .checked_mul(1_000_000_000)?
        .checked_add(i128::from(nanos))
}

/// Format a normalized or overfull nanosecond value as canonical UTC RFC3339.
/// Nanoseconds carry into seconds, and the result is limited to years 1..=9999.
pub fn format(seconds: i64, nanos: u32) -> Option<String> {
    let total = i128::from(seconds)
        .checked_mul(1_000_000_000)?
        .checked_add(i128::from(nanos))?;
    let value = OffsetDateTime::from_unix_timestamp_nanos(total)
        .ok()?
        .checked_to_offset(UtcOffset::UTC)?;
    if !(1..=9999).contains(&value.year()) {
        return None;
    }
    value.format(&Rfc3339).ok()
}

/// Format a UTC filename stem as `YYYYMMDDTHHMMSSZ`.
pub fn compact_utc(seconds: i64) -> Option<String> {
    let value = OffsetDateTime::from_unix_timestamp(seconds)
        .ok()?
        .checked_to_offset(UtcOffset::UTC)?;
    if !(1..=9999).contains(&value.year()) {
        return None;
    }
    Some(format!(
        "{:04}{:02}{:02}T{:02}{:02}{:02}Z",
        value.year(),
        value.month() as u8,
        value.day(),
        value.hour(),
        value.minute(),
        value.second()
    ))
}

fn wire_shape(text: &str) -> bool {
    let b = text.as_bytes();
    if b.len() < 20
        || b[4] != b'-'
        || b[7] != b'-'
        || b[10] != b'T'
        || b[13] != b':'
        || b[16] != b':'
        || !digits(&b[0..4])
        || !digits(&b[5..7])
        || !digits(&b[8..10])
        || !digits(&b[11..13])
        || !digits(&b[14..16])
        || !digits(&b[17..19])
    {
        return false;
    }
    if b[0..4] == *b"0000" || &b[17..19] > b"59".as_slice() {
        return false;
    }
    let mut rest = &b[19..];
    if rest.first() == Some(&b'.') {
        rest = &rest[1..];
        let count = rest.iter().take_while(|c| c.is_ascii_digit()).count();
        if !(1..=9).contains(&count) {
            return false;
        }
        rest = &rest[count..];
    }
    if rest == b"Z" {
        return true;
    }
    rest.len() == 6
        && (rest[0] == b'+' || rest[0] == b'-')
        && rest[3] == b':'
        && digits(&rest[1..3])
        && digits(&rest[4..6])
}

fn digits(bytes: &[u8]) -> bool {
    bytes.iter().all(u8::is_ascii_digit)
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn strict_profile_and_calendar_boundaries() {
        for good in [
            "0001-01-01T00:00:00Z",
            "2024-02-29T23:59:59.123456789+02:30",
            "9999-12-31T23:59:59+00:30",
        ] {
            assert!(parse(good).is_some(), "{good}");
        }
        for bad in [
            "0000-01-01T00:00:00Z",
            "2026-01-01 00:00:00Z",
            "2026-01-01t00:00:00Z",
            "2026-01-01T00:00:00z",
            "2026-01-01T00:00:60Z",
            "2026-02-30T00:00:00Z",
            "2026-01-01T00:00:00+24:00",
            "2026-01-01T00:00:00+01:60",
            "2026-01-01T00:00:00.1234567890Z",
            "2026-01-01T00:00:00+0100",
        ] {
            assert!(parse(bad).is_none(), "{bad}");
        }
        assert!(parse("0001-01-01T00:00:00+23:59").is_none());
        assert!(parse("9999-12-31T23:59:59-23:59").is_none());
    }

    #[test]
    fn canonical_format_normalizes_and_trims() {
        let sec = parse("2026-10-04T20:30:05.1200+02:00").unwrap().0;
        assert_eq!(
            format(sec, 120_000_000).as_deref(),
            Some("2026-10-04T18:30:05.12Z")
        );
        assert_eq!(
            format(sec, 1_000_000_000).as_deref(),
            Some("2026-10-04T18:30:06Z")
        );
        assert_eq!(format(sec, 0).as_deref(), Some("2026-10-04T18:30:05Z"));
        assert_eq!(format(i64::MIN, 0), None);
        assert_eq!(compact_utc(sec).as_deref(), Some("20261004T183005Z"));
    }
}
