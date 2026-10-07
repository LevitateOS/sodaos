//! Shared wire timestamp parsing and wall-clock timestamps used by the CLI.

/// System wall clock as whole seconds since the Unix epoch.
pub fn now_secs() -> i64 {
    match std::time::SystemTime::now().duration_since(std::time::UNIX_EPOCH) {
        Ok(d) => d.as_secs() as i64,
        Err(e) => -(e.duration().as_secs() as i64),
    }
}

/// Parse the broker's RFC3339 deadline representation.
///
/// Pre-epoch deadlines are outside the terminal's admission window.
pub fn parse_iso_deadline(s: &str) -> Option<i64> {
    let (seconds, _) = soda_wire_time::parse(s)?;
    (seconds >= 0).then_some(seconds)
}

#[cfg(test)]
#[path = "timex_tests.rs"]
mod timex_tests;
