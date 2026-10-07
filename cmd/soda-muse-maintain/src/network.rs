use std::net::IpAddr;

/// Parse and range-check an advertised prefix using the standard IP grammar.
/// Host bits are accepted because callers validate the subnet separately.
pub(crate) fn parse_prefix(value: &str) -> Result<(), String> {
    let (address, bits) = value
        .rsplit_once('/')
        .ok_or_else(|| format!("invalid IP prefix {value:?}: missing slash"))?;
    if address.contains('%') {
        return Err(format!(
            "invalid IP prefix {value:?}: zones are not allowed"
        ));
    }
    let addr = address
        .parse::<IpAddr>()
        .map_err(|_| format!("invalid IP prefix {value:?}: invalid address"))?;
    if bits.is_empty()
        || (bits.len() > 1 && bits.starts_with('0'))
        || !bits.bytes().all(|b| b.is_ascii_digit())
    {
        return Err(format!(
            "invalid IP prefix {value:?}: invalid prefix length"
        ));
    }
    let bits = bits
        .parse::<u32>()
        .map_err(|_| format!("invalid IP prefix {value:?}: invalid prefix length"))?;
    let max = if addr.is_ipv4() { 32 } else { 128 };
    if bits > max {
        return Err(format!(
            "invalid IP prefix {value:?}: prefix length out of range"
        ));
    }
    Ok(())
}
