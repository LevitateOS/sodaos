// RFC3339Nano time codec, extracted from wire.rs (A05.M).
use serde::{Deserialize, Deserializer, Serialize, Serializer};
/// Unix time with nanoseconds, serialized exactly like Go time.Time
/// (RFC3339Nano, UTC `Z`, trailing fractional zeros trimmed).
#[derive(Debug, Clone, Copy, Default, PartialEq, Eq, PartialOrd, Ord)]
pub struct UnixTime {
    pub sec: i64,
    pub nanos: u32,
}

impl UnixTime {
    pub fn now() -> UnixTime {
        let duration = std::time::SystemTime::now()
            .duration_since(std::time::UNIX_EPOCH)
            .unwrap_or_default();
        UnixTime {
            sec: duration.as_secs() as i64,
            nanos: duration.subsec_nanos(),
        }
    }

    pub fn add_hours(self, hours: i64) -> UnixTime {
        UnixTime {
            sec: self.sec + hours * 3600,
            nanos: self.nanos,
        }
    }

    pub fn as_system_time(self) -> std::time::SystemTime {
        if self.sec >= 0 {
            std::time::UNIX_EPOCH + std::time::Duration::new(self.sec as u64, self.nanos)
        } else {
            std::time::UNIX_EPOCH - std::time::Duration::new((-self.sec) as u64, 0)
                + std::time::Duration::from_nanos(self.nanos as u64)
        }
    }
}

impl Serialize for UnixTime {
    fn serialize<S: Serializer>(&self, ser: S) -> Result<S::Ok, S::Error> {
        ser.serialize_str(&format_rfc3339_nano(self.sec, self.nanos))
    }
}

impl<'de> Deserialize<'de> for UnixTime {
    fn deserialize<D: Deserializer<'de>>(de: D) -> Result<UnixTime, D::Error> {
        let text = String::deserialize(de)?;
        let (sec, nanos) = parse_rfc3339_nano(&text).map_err(serde::de::Error::custom)?;
        Ok(UnixTime { sec, nanos })
    }
}

fn is_leap(year: i64) -> bool {
    year % 4 == 0 && (year % 100 != 0 || year % 400 == 0)
}

fn days_in_month(year: i64, month: u32) -> u32 {
    match month {
        1 | 3 | 5 | 7 | 8 | 10 | 12 => 31,
        4 | 6 | 9 | 11 => 30,
        2 if is_leap(year) => 29,
        2 => 28,
        _ => 0,
    }
}

// Howard Hinnant's days-from-civil algorithm; valid for all civil dates.
fn days_from_civil(year: i64, month: u32, day: u32) -> i64 {
    let y = if month <= 2 { year - 1 } else { year };
    let era = y.div_euclid(400);
    let yoe = y.rem_euclid(400) as u64;
    let mp = (month as u64 + 9) % 12;
    let doy = (153 * mp + 2) / 5 + day as u64 - 1;
    let doe = yoe * 365 + yoe / 4 - yoe / 100 + doy;
    (era * 146097 + doe as i64) - 719468
}

fn civil_from_days(days: i64) -> (i64, u32, u32) {
    let z = days + 719468;
    let era = z.div_euclid(146097);
    let doe = z.rem_euclid(146097) as u64;
    let yoe = (doe - doe / 1460 + doe / 36524 - doe / 146096) / 365;
    let mut year = yoe as i64 + era * 400;
    let doy = doe - (365 * yoe + yoe / 4 - yoe / 100);
    let mp = (5 * doy + 2) / 153;
    let day = (doy - (153 * mp + 2) / 5 + 1) as u32;
    let month = (if mp < 10 { mp + 3 } else { mp - 9 }) as u32;
    if month <= 2 {
        year += 1;
    }
    (year, month, day)
}

pub fn parse_rfc3339_nano(text: &str) -> Result<(i64, u32), String> {
    // `2006-01-02T15:04:05(.999999999)?(Z|±07:00|±0700|±07)` like Go's
    // RFC3339Nano. Exactly four year digits and two for every other field.
    let err = || format!("invalid timestamp {text:?}");
    if text.len() < 20 {
        return Err(err());
    }
    let date = text.as_bytes();
    if date[4] != b'-'
        || date[7] != b'-'
        || date[10] != b'T'
        || date[13] != b':'
        || date[16] != b':'
    {
        return Err(err());
    }
    let num = |from: usize, to: usize| -> Result<i64, String> {
        text[from..to].parse::<i64>().map_err(|_| err())
    };
    let year = num(0, 4)?;
    let month = num(5, 7)?;
    let day = num(8, 10)?;
    let hour = num(11, 13)?;
    let minute = num(14, 16)?;
    let second = num(17, 19)?;
    if !(1..=12).contains(&month)
        || day < 1
        || day > days_in_month(year, month as u32) as i64
        || hour > 23
        || minute > 59
        || second > 59
    {
        return Err(err());
    }
    let mut rest = &text[19..];
    let mut nanos: u32 = 0;
    if let Some(fraction) = rest.strip_prefix('.') {
        let digits = fraction.bytes().take_while(u8::is_ascii_digit).count();
        if digits == 0 || digits > 9 {
            return Err(err());
        }
        let mut scale = 100_000_000u32;
        for b in fraction.bytes().take(digits) {
            nanos += (b - b'0') as u32 * scale;
            scale /= 10;
        }
        rest = &fraction[digits..];
    }
    let mut offset: i64 = 0;
    if rest == "Z" {
        rest = "";
    } else if rest.len() >= 3 && (rest.as_bytes()[0] == b'+' || rest.as_bytes()[0] == b'-') {
        let sign = if rest.as_bytes()[0] == b'-' { -1 } else { 1 };
        let hour: i64 = rest[1..3].parse().map_err(|_| err())?;
        let mut minute: i64 = 0;
        rest = &rest[3..];
        if let Some(tail) = rest.strip_prefix(':') {
            if tail.len() != 2 {
                return Err(err());
            }
            minute = tail.parse().map_err(|_| err())?;
            rest = "";
        } else if rest.len() == 2 {
            minute = rest.parse().map_err(|_| err())?;
            rest = "";
        } else if !rest.is_empty() {
            return Err(err());
        }
        if hour > 23 || minute > 59 {
            return Err(err());
        }
        offset = sign * (hour * 3600 + minute * 60);
    }
    if !rest.is_empty() {
        return Err(err());
    }
    let days = days_from_civil(year, month as u32, day as u32);
    let sec = days * 86400 + hour * 3600 + minute * 60 + second - offset;
    Ok((sec, nanos))
}

pub fn format_rfc3339_nano(sec: i64, nanos: u32) -> String {
    let days = sec.div_euclid(86400);
    let clock = sec.rem_euclid(86400);
    let (year, month, day) = civil_from_days(days);
    let mut out = format!(
        "{:04}-{:02}-{:02}T{:02}:{:02}:{:02}",
        year,
        month,
        day,
        clock / 3600,
        clock % 3600 / 60,
        clock % 60
    );
    if nanos != 0 {
        let mut fraction = format!("{nanos:09}");
        while fraction.ends_with('0') {
            fraction.pop();
        }
        out.push('.');
        out.push_str(&fraction);
    }
    out.push('Z');
    out
}
