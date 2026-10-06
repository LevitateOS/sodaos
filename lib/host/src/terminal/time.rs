// ---------- RFC 3339 deadlines (`time.Time` wire form) ----------

fn days_from_civil(year: i64, month: i64, day: i64) -> i64 {
    let y = if month <= 2 { year - 1 } else { year };
    let era = if y >= 0 { y } else { y - 399 } / 400;
    let yoe = y - era * 400;
    let mp = (month + 9) % 12;
    let doy = (153 * mp + 2) / 5 + day - 1;
    let doe = yoe * 365 + yoe / 4 - yoe / 100 + doy;
    era * 146097 + doe - 719468
}

fn days_in_month(year: i64, month: i64) -> i64 {
    match month {
        1 | 3 | 5 | 7 | 8 | 10 | 12 => 31,
        4 | 6 | 9 | 11 => 30,
        2 if year % 4 == 0 && (year % 100 != 0 || year % 400 == 0) => 29,
        2 => 28,
        _ => 0,
    }
}

fn parse_two(digits: &[u8]) -> Option<i64> {
    if digits.len() == 2 && digits.iter().all(|b| b.is_ascii_digit()) {
        Some(((digits[0] - b'0') * 10 + (digits[1] - b'0')) as i64)
    } else {
        None
    }
}

/// Strict RFC 3339 subset matching Go `time.Time` JSON decoding: uppercase
/// `T`, `Z` or numeric offset, optional fractional seconds. Returns Unix
/// seconds and nanoseconds.
pub fn parse_rfc3339(s: &str) -> Option<(i64, u32)> {
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
    if !b[0..4].iter().all(|c| c.is_ascii_digit())
        || !b[5..7].iter().all(|c| c.is_ascii_digit())
        || !b[8..10].iter().all(|c| c.is_ascii_digit())
        || !b[11..13].iter().all(|c| c.is_ascii_digit())
        || !b[14..16].iter().all(|c| c.is_ascii_digit())
        || !b[17..19].iter().all(|c| c.is_ascii_digit())
    {
        return None;
    }
    let year: i64 = s[0..4].parse().ok()?;
    let month = parse_two(&b[5..7])?;
    let day = parse_two(&b[8..10])?;
    let hour = parse_two(&b[11..13])?;
    let minute = parse_two(&b[14..16])?;
    let second = parse_two(&b[17..19])?;
    if !(1..=12).contains(&month) || day < 1 || day > days_in_month(year, month) {
        return None;
    }
    if hour > 23 || minute > 59 || second > 59 {
        return None;
    }
    let mut rest = &b[19..];
    let mut nanos: u32 = 0;
    if rest.first() == Some(&b'.') {
        rest = &rest[1..];
        let mut count = 0usize;
        let mut value: u32 = 0;
        while count < 9 && !rest.is_empty() && rest[0].is_ascii_digit() {
            value = value * 10 + u32::from(rest[0] - b'0');
            rest = &rest[1..];
            count += 1;
        }
        if count == 0 {
            return None;
        }
        // Extra fractional digits beyond nanoseconds: Go rounds; the agent
        // never emits them, so accept-and-truncate would hide corruption.
        // Reject to fail closed.
        if !rest.is_empty() && rest[0].is_ascii_digit() {
            return None;
        }
        for _ in count..9 {
            value *= 10;
        }
        nanos = value;
    }
    let offset: i64 = if rest == b"Z" {
        0
    } else if rest.len() >= 3 && (rest[0] == b'+' || rest[0] == b'-') {
        let sign = if rest[0] == b'-' { -1 } else { 1 };
        let (hours, minutes) = match rest.len() {
            3 => (parse_two(&rest[1..3])?, 0),
            5 => (parse_two(&rest[1..3])?, parse_two(&rest[3..5])?),
            6 if rest[3] == b':' => (parse_two(&rest[1..3])?, parse_two(&rest[4..6])?),
            _ => return None,
        };
        if hours > 23 || minutes > 59 {
            return None;
        }
        sign * (hours * 3600 + minutes * 60)
    } else {
        return None;
    };
    let days = days_from_civil(year, month, day);
    Some((
        days * 86400 + hour * 3600 + minute * 60 + second - offset,
        nanos,
    ))
}
