use std::time::{Duration, Instant, SystemTime, UNIX_EPOCH};

use super::run::{FACTORY_CLEANUP_SECS, FACTORY_LIVE_SECS};

// ---------- RFC3339Nano deadlines ----------

pub(in crate::factory) const NANOS_PER_SEC: i128 = 1_000_000_000;

fn is_leap_year(y: i64) -> bool {
    y % 4 == 0 && (y % 100 != 0 || y % 400 == 0)
}

fn days_in_month(y: i64, m: i64) -> i64 {
    match m {
        1 | 3 | 5 | 7 | 8 | 10 | 12 => 31,
        4 | 6 | 9 | 11 => 30,
        2 if is_leap_year(y) => 29,
        2 => 28,
        _ => 0,
    }
}

/// Days from civil date to Unix epoch (Howard Hinnant's algorithm).
fn days_from_civil(y: i64, m: i64, d: i64) -> i64 {
    let y = if m <= 2 { y - 1 } else { y };
    let era = if y >= 0 { y } else { y - 399 } / 400;
    let yoe = y - era * 400;
    let mp = (m + 9) % 12;
    let doy = (153 * mp + 2) / 5 + d - 1;
    let doe = yoe * 365 + yoe / 4 - yoe / 100 + doy;
    era * 146097 + doe - 719468
}

fn two_digits(b: &[u8]) -> Option<i64> {
    if b.len() == 2 && b[0].is_ascii_digit() && b[1].is_ascii_digit() {
        Some(((b[0] - b'0') * 10 + (b[1] - b'0')) as i64)
    } else {
        None
    }
}

/// Parse a Go `time.Time` JSON value to epoch nanos. Accepts the canonical
/// shape Go emits: `YYYY-MM-DDTHH:MM:SS` with an optional 1-9 digit
/// fraction and a `Z` or numeric offset.
pub(in crate::factory) fn parse_deadline(s: &str) -> Option<i128> {
    let b = s.as_bytes();
    if b.len() < 20
        || b[4] != b'-'
        || b[7] != b'-'
        || b[10] != b'T'
        || b[13] != b':'
        || b[16] != b':'
    {
        return None;
    }
    if !b[0..4].iter().all(|c| c.is_ascii_digit()) {
        return None;
    }
    let year: i64 = s[0..4].parse().ok()?;
    if !(1..=9999).contains(&year) {
        return None;
    }
    let month = two_digits(&b[5..7])?;
    let day = two_digits(&b[8..10])?;
    let hour = two_digits(&b[11..13])?;
    let minute = two_digits(&b[14..16])?;
    let second = two_digits(&b[17..19])?;
    if !(1..=12).contains(&month) || !(1..=days_in_month(year, month)).contains(&day) {
        return None;
    }
    if hour > 23 || minute > 59 || second > 59 {
        return None;
    }
    let mut rest = &b[19..];
    let mut frac_nanos: i128 = 0;
    if rest.first() == Some(&b'.') {
        rest = &rest[1..];
        let mut digits = 0;
        let mut scale: i128 = 100_000_000;
        while rest.first().is_some_and(|c| c.is_ascii_digit()) {
            if digits >= 9 {
                return None;
            }
            frac_nanos += ((rest[0] - b'0') as i128) * scale;
            scale /= 10;
            digits += 1;
            rest = &rest[1..];
        }
        if digits == 0 {
            return None;
        }
    }
    let offset_secs: i128 = if rest == b"Z" {
        0
    } else if rest.len() == 6 && (rest[0] == b'+' || rest[0] == b'-') && rest[3] == b':' {
        let oh = two_digits(&rest[1..3])?;
        let om = two_digits(&rest[4..6])?;
        if oh > 23 || om > 59 {
            return None;
        }
        let sign = if rest[0] == b'-' { -1 } else { 1 };
        sign * (oh as i128 * 3600 + om as i128 * 60)
    } else {
        return None;
    };
    let days = days_from_civil(year, month, day) as i128;
    Some(
        (days * 86400 + hour as i128 * 3600 + minute as i128 * 60 + second as i128) * NANOS_PER_SEC
            + frac_nanos
            - offset_secs * NANOS_PER_SEC,
    )
}

/// Current wall-clock time as Unix epoch nanos.
pub(in crate::factory) fn system_nanos_now() -> i128 {
    match SystemTime::now().duration_since(UNIX_EPOCH) {
        Ok(d) => d.as_nanos() as i128,
        Err(e) => -(e.duration().as_nanos() as i128),
    }
}

/// Go zero time (`0001-01-01T00:00:00Z`) has Unix epoch nanos of the
/// proleptic civil date; anything else, malformed included, is nonzero or
/// unusable. Callers treat `None` (malformed) like a missing deadline: the
/// route decoder owns malformed-deadline reports, the factory never sees
/// them over the wire.
pub(in crate::factory) fn deadline_is_zero(text: &str) -> bool {
    match parse_deadline(text) {
        Some(nanos) => nanos == days_from_civil(1, 1, 1) as i128 * 86400 * NANOS_PER_SEC,
        None => false,
    }
}

/// Truncated seconds from now until the deadline, like
/// `int64(time.Until(deadline).Seconds())`.
pub(in crate::factory) fn seconds_until(deadline_nanos: i128) -> i64 {
    ((deadline_nanos - system_nanos_now()) / NANOS_PER_SEC)
        .clamp(i64::MIN as i128, i64::MAX as i128) as i64
}

/// Operation deadline capped by the run deadline, like
/// `context.WithDeadline(ctx, run.Deadline)`.
pub(in crate::factory) fn drive_deadline(op: Instant, deadline_nanos: i128) -> Instant {
    let delta = deadline_nanos - system_nanos_now();
    if delta <= 0 {
        return Instant::now();
    }
    let wait = Duration::from_nanos(delta.min(u64::MAX as i128) as u64);
    Instant::now().checked_add(wait).unwrap_or(op).min(op)
}

fn min_instant(a: Instant, b: Instant) -> Instant {
    if a < b {
        a
    } else {
        b
    }
}

pub(in crate::factory) fn cleanup_deadline() -> Instant {
    Instant::now() + Duration::from_secs(FACTORY_CLEANUP_SECS)
}

pub(in crate::factory) fn live_deadline(op: Instant) -> Instant {
    min_instant(op, Instant::now() + Duration::from_secs(FACTORY_LIVE_SECS))
}
