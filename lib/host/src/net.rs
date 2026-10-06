//! IP admission: port of `admitProjectIP` in
//! `internal/host/project/create.go` (netip parse + prefix containment).

use std::net::IpAddr;

/// Parse a project IP the way `netip.ParseAddr` does for podman output.
/// Zones are refused: podman never reports them, and Go would scope them
/// to an interface the daemon does not model.
pub fn parse_addr(s: &str) -> Result<IpAddr, String> {
    if s.contains('%') {
        return Err(format!("invalid IP address {s:?}"));
    }
    s.parse::<IpAddr>()
        .map_err(|_| format!("invalid IP address {s:?}"))
}

pub struct Prefix {
    addr: IpAddr,
    bits: u32,
}

/// Parse `address/bits` like `netip.ParsePrefix` (bits required).
pub fn parse_prefix(s: &str) -> Result<Prefix, String> {
    let (addr, bits) = s
        .split_once('/')
        .ok_or_else(|| format!("invalid prefix {s:?}"))?;
    let addr = parse_addr(addr)?;
    let max = match addr {
        IpAddr::V4(_) => 32,
        IpAddr::V6(_) => 128,
    };
    let bits: u32 = bits.parse().map_err(|_| format!("invalid prefix {s:?}"))?;
    if bits > max {
        return Err(format!("invalid prefix {s:?}"));
    }
    Ok(Prefix { addr, bits })
}

impl Prefix {
    pub fn contains(&self, ip: IpAddr) -> bool {
        match (self.addr, ip) {
            (IpAddr::V4(net), IpAddr::V4(addr)) => {
                let mask = if self.bits == 0 {
                    0u32
                } else {
                    !0u32 << (32 - self.bits)
                };
                (u32::from(net) & mask) == (u32::from(addr) & mask)
            }
            (IpAddr::V6(net), IpAddr::V6(addr)) => {
                let mask = if self.bits == 0 {
                    0u128
                } else {
                    !0u128 << (128 - self.bits)
                };
                (u128::from(net) & mask) == (u128::from(addr) & mask)
            }
            _ => false,
        }
    }
}

/// Admit one project IP against the configured subnet. Empty IPs are
/// admitted (stopped/no lease); anything outside the subnet is refused.
/// A malformed subnet admits nothing, matching Go's ignored parse error.
pub fn admit_ip(ip: &str, subnet: &str) -> Result<(), String> {
    if ip.is_empty() {
        return Ok(());
    }
    let addr = parse_addr(ip)?;
    let prefix =
        parse_prefix(subnet).map_err(|_| "project IP outside configured network".to_string())?;
    if !prefix.contains(addr) {
        return Err("project IP outside configured network".to_string());
    }
    Ok(())
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn admission_matches_go_cases() {
        assert!(admit_ip("", "10.0.0.0/24").is_ok());
        assert!(admit_ip("10.0.0.5", "10.0.0.0/24").is_ok());
        assert!(admit_ip("10.0.1.5", "10.0.0.0/24").is_err());
        assert!(admit_ip("not-an-ip", "10.0.0.0/24").is_err());
        assert!(admit_ip("10.0.0.5", "bogus").is_err());
        assert!(admit_ip("10.0.0.5", "10.0.0.0").is_err()); // bits required
        assert!(admit_ip("10.0.0.5", "10.0.0.0/33").is_err());
        assert!(admit_ip("fd00::5", "fd00::/64").is_ok());
        assert!(admit_ip("fd00:1::5", "fd00::/64").is_err());
        assert!(admit_ip("10.0.0.5", "fd00::/64").is_err()); // family mismatch
    }
}
