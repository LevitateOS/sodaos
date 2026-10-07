/// Parse the shared strict RFC3339 wire profile to Unix seconds and nanoseconds.
pub fn parse_rfc3339(s: &str) -> Option<(i64, u32)> {
    soda_wire_time::parse(s)
}
