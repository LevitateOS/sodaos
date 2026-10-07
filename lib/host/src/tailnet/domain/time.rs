use crate::json;

/// Strict shared RFC3339Nano profile; report whether the instant is Go zero time.
pub fn parse_rfc3339_nano(s: &str) -> Option<bool> {
    soda_wire_time::parse_nanos(s).map(|nanos| nanos == -62_135_596_800_000_000_000)
}

/// `encoding/json` string escaping with surrounding double quotes.
pub fn go_escape(s: &str) -> String {
    json::quote(s)
}
