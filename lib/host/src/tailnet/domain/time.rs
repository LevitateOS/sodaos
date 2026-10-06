use crate::json;

fn digits(b: &[u8], lo: usize, hi: usize) -> Option<i64> {
    if hi > b.len() {
        return None;
    }
    let mut v: i64 = 0;
    for &c in &b[lo..hi] {
        if !c.is_ascii_digit() {
            return None;
        }
        v = v * 10 + (c - b'0') as i64;
    }
    Some(v)
}

/// Proleptic-Gregorian days from 0001-01-01 (negative for year 0 dates).
fn days_since_year_one(year: i64, month: i64, day: i64) -> i64 {
    let y = if month <= 2 { year - 1 } else { year };
    let era = y.div_euclid(400);
    let yoe = y.rem_euclid(400);
    let mp = (month + 9).rem_euclid(12);
    let doy = (153 * mp + 2) / 5 + day - 1;
    let doe = yoe * 365 + yoe / 4 - yoe / 100 + doy;
    era * 146097 + doe - 719468 + 719162
}

/// Strict `time.Parse(time.RFC3339Nano)`: `None` for invalid input, else
/// `Some(is_zero)` where zero is the instant 0001-01-01T00:00:00Z.
/// Grammar: `YYYY-MM-DDTHH:MM:SS[.f+](Z|±HH:MM)` with a real calendar
/// date, H/HH 00-23, MM/SS 00-59, 1+ fraction digits (Go keeps nanosecond
/// precision and ignores the rest), offset HH 00-24 and MM 00-60
/// (Go's per-field offset bounds, verified by probe).
pub fn parse_rfc3339_nano(s: &str) -> Option<bool> {
    let b = s.as_bytes();
    if b.len() < 19 {
        return None;
    }
    let year = digits(b, 0, 4)?;
    if b.get(4) != Some(&b'-') {
        return None;
    }
    let month = digits(b, 5, 7)?;
    if b.get(7) != Some(&b'-') {
        return None;
    }
    let day = digits(b, 8, 10)?;
    if b.get(10) != Some(&b'T') {
        return None;
    }
    // Go's hour layout takes 1-2 digits; every other field is fixed width.
    let mut hpos = 11;
    while hpos < b.len() && b[hpos].is_ascii_digit() && hpos - 11 < 2 {
        hpos += 1;
    }
    if hpos == 11 || b.get(hpos) != Some(&b':') {
        return None;
    }
    let hour = digits(b, 11, hpos)?;
    let minute = digits(b, hpos + 1, hpos + 3)?;
    if b.get(hpos + 3) != Some(&b':') {
        return None;
    }
    let second = digits(b, hpos + 4, hpos + 6)?;
    let mut pos = hpos + 6;
    let mut nanos: i64 = 0;
    if b.get(pos) == Some(&b'.') {
        pos += 1;
        let start = pos;
        while pos < b.len() && b[pos].is_ascii_digit() {
            pos += 1;
        }
        let len = pos - start;
        if len < 1 {
            return None;
        }
        let mut scale = 100_000_000i64;
        for &c in &b[start..start + len.min(9)] {
            nanos += (c - b'0') as i64 * scale;
            scale /= 10;
        }
    }
    let rest = b.get(pos..)?;
    let offset_secs: i64 = if rest.len() == 1 && rest[0] == b'Z' {
        0
    } else if rest.len() == 6
        && (rest[0] == b'+' || rest[0] == b'-')
        && rest[3] == b':'
        && rest[1].is_ascii_digit()
        && rest[2].is_ascii_digit()
        && rest[4].is_ascii_digit()
        && rest[5].is_ascii_digit()
    {
        let hh = ((rest[1] - b'0') as i64) * 10 + (rest[2] - b'0') as i64;
        let mm = ((rest[4] - b'0') as i64) * 10 + (rest[5] - b'0') as i64;
        if hh > 24 || mm > 60 {
            return None;
        }
        let total = hh * 3600 + mm * 60;
        if rest[0] == b'-' {
            -total
        } else {
            total
        }
    } else {
        return None;
    };
    if !(1..=12).contains(&month) || hour > 23 || minute > 59 || second > 59 {
        return None;
    }
    let leap = year % 4 == 0 && (year % 100 != 0 || year % 400 == 0);
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
    let abs = days_since_year_one(year, month, day) * 86_400 + hour * 3600 + minute * 60 + second
        - offset_secs;
    Some(abs == 0 && nanos == 0)
}

/// `encoding/json` string escaping with surrounding double quotes.
pub fn go_escape(s: &str) -> String {
    json::quote(s)
}
