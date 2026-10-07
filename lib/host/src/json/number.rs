/// Existing caller string policy: remove one leading `+`, then use Rust's
/// decimal parser with the signed destination width.
pub fn parse_go_int64(s: &str) -> Option<i64> {
    s.strip_prefix('+').unwrap_or(s).parse::<i64>().ok()
}

/// The same caller normalization with an unsigned destination width.
pub fn parse_go_uint32(s: &str) -> Option<u32> {
    s.strip_prefix('+').unwrap_or(s).parse::<u32>().ok()
}
