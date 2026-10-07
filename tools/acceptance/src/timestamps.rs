use crate::error::Error;

/// Current UTC time in canonical RFC3339Nano form.
pub fn now_rfc3339_nano() -> String {
    let elapsed = std::time::SystemTime::now()
        .duration_since(std::time::UNIX_EPOCH)
        .unwrap_or_default();
    soda_wire_time::format(elapsed.as_secs() as i64, elapsed.subsec_nanos())
        .expect("current clock timestamp outside supported RFC3339 range")
}

/// Validate an admitted strict RFC3339 wire timestamp.
pub fn validate_rfc3339(text: &str) -> Result<(), Error> {
    soda_wire_time::parse(text)
        .map(|_| ())
        .ok_or_else(|| Error::msg(format!("invalid timestamp: {text:?}")))
}

#[cfg(test)]
mod tests;
