//! ISO-8601 deadline parsing with CPython `datetime.fromisoformat` semantics.
//!
//! [`parse_iso_deadline`] mirrors `int(datetime.fromisoformat(
//! s.replace('Z', '+00:00')).timestamp())` over pure integer math
//! ([`days_from_civil`], Howard Hinnant's algorithm; no chrono, no float).
//! Every vector and rejection below was baked against local CPython 3.12.
//!
//! Grammar (all verified against CPython 3.12):
//!
//! - date: `YYYY-MM-DD` or basic `YYYYMMDD` (year 1..=9999, proleptic
//!   Gregorian; Feb 29 only on leap years).
//! - exactly one separator char (any char: `T`, `t`, space, ...).
//! - time: `HH`, `HH:MM`, `HH:MM:SS`, `HHMM`, or `HHMMSS`, with an optional
//!   fraction (`.` or `,` plus zero or more digits) meaning sub-second.
//! - offset (REQUIRED): `Z` (via the same literal pre-replace as the Python
//!   call) or `±HH`, `±HHMM`, `±HH:MM`, each with optional `:SS`/seconds and
//!   an optional fraction (`.`/`,` plus one or more digits) that CPython
//!   parses and then ignores. Offset minutes/seconds past 59 are accepted
//!   exactly like CPython; only the `|total| < 24h` bound is enforced.
//!
//! The result truncates toward zero like Python `int()` on the float stamp.
//!
//! Deliberate deviations from CPython (all tested as `None`):
//!
//! - naive inputs (no offset) are rejected: CPython resolves those in the
//!   machine-local zone, which would make results host-dependent.
//! - week dates (`2025-W27-5`) are rejected (deadlines never use them).
//! - basic date+time with no separator at all is rejected (CPython accepts
//!   it under an undocumented split).

/// System wall clock as whole seconds since the Unix epoch (same contract as
/// `sys::now_secs`; duplicated so this module has no cross-module coupling).
pub fn now_secs() -> i64 {
    match std::time::SystemTime::now().duration_since(std::time::UNIX_EPOCH) {
        Ok(d) => d.as_secs() as i64,
        Err(e) => -(e.duration().as_secs() as i64),
    }
}

/// Days since 1970-01-01 for a proleptic Gregorian date.
fn days_from_civil(y: i64, m: i64, d: i64) -> i64 {
    let y = if m <= 2 { y - 1 } else { y };
    let era = if y >= 0 { y } else { y - 399 } / 400;
    let yoe = y - era * 400;
    let mp = (m + 9) % 12;
    let doy = (153 * mp + 2) / 5 + d - 1;
    let doe = yoe * 365 + yoe / 4 - yoe / 100 + doy;
    era * 146097 + doe - 719468
}

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

fn two_digits(b: &[u8], at: usize) -> Option<i64> {
    if at + 2 > b.len() || !b[at].is_ascii_digit() || !b[at + 1].is_ascii_digit() {
        return None;
    }
    Some((i64::from(b[at] - b'0')) * 10 + i64::from(b[at + 1] - b'0'))
}

/// Parse `YYYY-MM-DD` or `YYYYMMDD`; returns the date plus the rest after it.
fn parse_date(text: &str) -> Option<(i64, i64, i64, &str)> {
    let b = text.as_bytes();
    let extended = b.len() >= 10
        && b[4] == b'-'
        && b[7] == b'-'
        && b[0..4].iter().all(|c| c.is_ascii_digit())
        && b[5..7].iter().all(|c| c.is_ascii_digit())
        && b[8..10].iter().all(|c| c.is_ascii_digit());
    let (year, month, day, rest) = if extended {
        let year = b[0..4]
            .iter()
            .fold(0i64, |n, c| n * 10 + i64::from(c - b'0'));
        let month = two_digits(b, 5)?;
        let day = two_digits(b, 8)?;
        (year, month, day, &text[10..])
    } else if b.len() >= 8 && b[0..8].iter().all(|c| c.is_ascii_digit()) {
        let year = b[0..4]
            .iter()
            .fold(0i64, |n, c| n * 10 + i64::from(c - b'0'));
        let month = two_digits(b, 4)?;
        let day = two_digits(b, 6)?;
        (year, month, day, &text[8..])
    } else {
        return None;
    };
    if !(1..=9999).contains(&year) || !(1..=12).contains(&month) {
        return None;
    }
    if day < 1 || day > days_in_month(year, month) {
        return None;
    }
    Some((year, month, day, rest))
}

/// Parse the time plus the required offset; returns
/// (seconds-of-day, fraction-nonzero, offset-seconds, bytes-consumed).
fn parse_time_offset(b: &[u8]) -> Option<(i64, bool, i64, usize)> {
    let hh = two_digits(b, 0)?;
    if hh > 23 {
        return None;
    }
    let mut i = 2usize;
    let mut mm = 0i64;
    let mut ss = 0i64;
    if i < b.len() && b[i] == b':' {
        i += 1;
        mm = two_digits(b, i)?;
        if mm > 59 {
            return None;
        }
        i += 2;
        if i < b.len() && b[i] == b':' {
            i += 1;
            ss = two_digits(b, i)?;
            if ss > 59 {
                return None;
            }
            i += 2;
        }
    } else if two_digits(b, i).is_some() {
        mm = two_digits(b, i)?;
        if mm > 59 {
            return None;
        }
        i += 2;
        if two_digits(b, i).is_some() {
            ss = two_digits(b, i)?;
            if ss > 59 {
                return None;
            }
            i += 2;
        }
    }
    // Optional fraction: '.' or ',' plus zero or more digits (CPython
    // accepts a bare trailing '.'). Only nonzero-ness matters: it decides
    // the truncate-toward-zero step for negative stamps.
    let mut frac_nonzero = false;
    if i < b.len() && (b[i] == b'.' || b[i] == b',') {
        i += 1;
        while i < b.len() && b[i].is_ascii_digit() {
            if b[i] != b'0' {
                frac_nonzero = true;
            }
            i += 1;
        }
    }
    let (offset, used) = parse_offset(&b[i..])?;
    Some((hh * 3600 + mm * 60 + ss, frac_nonzero, offset, i + used))
}

/// Parse `±HH[[:]MM[[:]SS]][.fraction]`; the fraction is parsed and ignored
/// like CPython. Returns (offset-seconds, bytes-consumed).
fn parse_offset(b: &[u8]) -> Option<(i64, usize)> {
    if b.is_empty() {
        return None;
    }
    let sign = match b[0] {
        b'+' => 1i64,
        b'-' => -1i64,
        _ => return None,
    };
    let hh = two_digits(b, 1)?;
    let mut i = 3usize;
    let mut mm = 0i64;
    let mut ss = 0i64;
    if i < b.len() && b[i] == b':' {
        i += 1;
        mm = two_digits(b, i)?;
        i += 2;
        if i < b.len() && b[i] == b':' {
            i += 1;
            ss = two_digits(b, i)?;
            i += 2;
        }
    } else if two_digits(b, i).is_some() {
        mm = two_digits(b, i)?;
        i += 2;
        if two_digits(b, i).is_some() {
            ss = two_digits(b, i)?;
            i += 2;
        }
    }
    // Offset fractions require at least one digit (a bare '.' is an error
    // here, unlike in the time part) and are ignored once parsed.
    if i < b.len() && (b[i] == b'.' || b[i] == b',') {
        i += 1;
        let start = i;
        while i < b.len() && b[i].is_ascii_digit() {
            i += 1;
        }
        if i == start {
            return None;
        }
    }
    // No per-component range check (CPython accepts e.g. +05:75); only the
    // strict 24h bound on the total.
    let total = hh * 3600 + mm * 60 + ss;
    if total >= 86400 {
        return None;
    }
    Some((sign * total, i))
}

/// `int(datetime.fromisoformat(s.replace('Z', '+00:00')).timestamp())`,
/// or `None` for malformed input (and for the documented deviations above).
pub fn parse_iso_deadline(s: &str) -> Option<i64> {
    let text = s.replace('Z', "+00:00");
    let (year, month, day, rest) = parse_date(&text)?;
    let mut chars = rest.chars();
    chars.next()?; // exactly one separator char
    let time_part = chars.as_str();
    let (tod, frac_nonzero, offset, used) = parse_time_offset(time_part.as_bytes())?;
    if used != time_part.len() {
        return None;
    }
    let mut out = days_from_civil(year, month, day) * 86400 + tod - offset;
    if frac_nonzero && out < 0 {
        out += 1; // truncate toward zero, like int() on the float stamp
    }
    Some(out)
}

#[cfg(test)]
#[path = "timex_tests.rs"]
mod timex_tests;
