use serde::{Deserialize, Deserializer, Serialize, Serializer};

/// Unix time with nanoseconds, serialized as canonical UTC RFC3339.
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

    pub fn add_hours(self, hours: i64) -> Option<UnixTime> {
        Some(UnixTime {
            sec: self.sec.checked_add(hours.checked_mul(3600)?)?,
            nanos: self.nanos,
        })
    }

    pub fn as_system_time(self) -> Option<std::time::SystemTime> {
        let total = i128::from(self.sec)
            .checked_mul(1_000_000_000)?
            .checked_add(i128::from(self.nanos))?;
        let abs = total.unsigned_abs();
        let duration = std::time::Duration::new(
            u64::try_from(abs / 1_000_000_000).ok()?,
            (abs % 1_000_000_000) as u32,
        );
        if total >= 0 {
            std::time::UNIX_EPOCH.checked_add(duration)
        } else {
            std::time::UNIX_EPOCH.checked_sub(duration)
        }
    }
}

impl Serialize for UnixTime {
    fn serialize<S: Serializer>(&self, ser: S) -> Result<S::Ok, S::Error> {
        let text = soda_wire_time::format(self.sec, self.nanos)
            .ok_or_else(|| serde::ser::Error::custom("timestamp outside RFC3339 range"))?;
        ser.serialize_str(&text)
    }
}

impl<'de> Deserialize<'de> for UnixTime {
    fn deserialize<D: Deserializer<'de>>(de: D) -> Result<UnixTime, D::Error> {
        let text = String::deserialize(de)?;
        let (sec, nanos) = soda_wire_time::parse(&text)
            .ok_or_else(|| serde::de::Error::custom("invalid RFC3339 timestamp"))?;
        Ok(UnixTime { sec, nanos })
    }
}

pub fn parse_rfc3339_nano(text: &str) -> Result<(i64, u32), String> {
    soda_wire_time::parse(text).ok_or_else(|| format!("invalid timestamp {text:?}"))
}

pub fn format_rfc3339_nano(sec: i64, nanos: u32) -> Option<String> {
    soda_wire_time::format(sec, nanos)
}
