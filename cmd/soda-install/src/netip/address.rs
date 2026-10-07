use std::net::{IpAddr, Ipv4Addr, Ipv6Addr};

/// IP value with the optional IPv6 zone kept separately for installer policy.
#[derive(Debug, Clone, PartialEq, Eq)]
pub struct Addr {
    pub(super) ip: IpAddr,
    pub(super) zone: Option<Vec<u8>>,
}

#[derive(Debug, Clone, PartialEq, Eq)]
pub struct NetipError {
    text: String,
}

impl NetipError {
    pub(super) fn addr_bytes(_input: &[u8]) -> Self {
        Self {
            text: String::from("invalid IP address"),
        }
    }
    pub(super) fn prefix(input: &str, detail: &str) -> Self {
        Self {
            text: format!("invalid IP prefix {input:?}: {detail}"),
        }
    }
    pub fn text(&self) -> &str {
        &self.text
    }
}
impl std::fmt::Display for NetipError {
    fn fmt(&self, f: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
        f.write_str(&self.text)
    }
}
impl std::error::Error for NetipError {}

pub fn parse_addr(value: &str) -> Result<Addr, NetipError> {
    parse_addr_bytes(value.as_bytes())
}

pub fn parse_addr_bytes(value: &[u8]) -> Result<Addr, NetipError> {
    let (head_bytes, zone) = match value.iter().position(|b| *b == b'%') {
        Some(i) => {
            let zone = &value[i + 1..];
            if zone.is_empty() {
                return Err(NetipError::addr_bytes(value));
            }
            (&value[..i], Some(zone.to_vec()))
        }
        None => (value, None),
    };
    let head = std::str::from_utf8(head_bytes).map_err(|_| NetipError::addr_bytes(value))?;
    let ip = if head.contains(':') {
        IpAddr::V6(
            head.parse::<Ipv6Addr>()
                .map_err(|_| NetipError::addr_bytes(value))?,
        )
    } else if zone.is_none() {
        IpAddr::V4(
            head.parse::<Ipv4Addr>()
                .map_err(|_| NetipError::addr_bytes(value))?,
        )
    } else {
        return Err(NetipError::addr_bytes(value));
    };
    Ok(Addr { ip, zone })
}
