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

fn fixed_prefix(b: &[u8]) -> bool {
    b.len() >= 20
        && b[4] == b'-'
        && b[7] == b'-'
        && b[10] == b'T'
        && b[13] == b':'
        && b[16] == b':'
        && digits(&b[0..4])
        && digits(&b[5..7])
        && digits(&b[8..10])
        && digits(&b[11..13])
        && digits(&b[14..16])
        && digits(&b[17..19])
        && b[0..4] != *b"0000"
        && &b[17..19] <= b"59".as_slice()
}

fn skip_fraction(rest: &[u8]) -> Option<&[u8]> {
    if rest.first() != Some(&b'.') {
        return Some(rest);
    }
    let rest = &rest[1..];
    let count = rest.iter().take_while(|c| c.is_ascii_digit()).count();
    if !(1..=9).contains(&count) {
        return None;
    }
    Some(&rest[count..])
}

fn zone_suffix(rest: &[u8]) -> bool {
    if rest == b"Z" {
        return true;
    }
    rest.len() == 6
        && (rest[0] == b'+' || rest[0] == b'-')
        && rest[3] == b':'
        && digits(&rest[1..3])
        && digits(&rest[4..6])
}

fn wire_shape(text: &str) -> bool {
    let b = text.as_bytes();
    if !fixed_prefix(b) {
        return false;
    }
    let Some(rest) = skip_fraction(&b[19..]) else {
        return false;
    };
    zone_suffix(rest)
}

fn digits(bytes: &[u8]) -> bool {
    bytes.iter().all(u8::is_ascii_digit)
}
