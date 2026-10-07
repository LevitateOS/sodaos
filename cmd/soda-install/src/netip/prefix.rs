use super::address::{parse_addr, Addr, NetipError};

#[derive(Debug, Clone, PartialEq, Eq)]
pub struct Prefix {
    addr: Addr,
    bits: u32,
}

pub fn parse_prefix(value: &str) -> Result<Prefix, NetipError> {
    let (address, bits) = value
        .rsplit_once('/')
        .ok_or_else(|| NetipError::prefix(value, "missing slash"))?;
    let addr = parse_addr(address).map_err(|_| NetipError::prefix(value, "invalid address"))?;
    if !addr.zone().is_empty() {
        return Err(NetipError::prefix(
            value,
            "zones are not allowed in prefixes",
        ));
    }
    if bits.is_empty()
        || (bits.len() > 1 && bits.starts_with('0'))
        || !bits.bytes().all(|b| b.is_ascii_digit())
    {
        return Err(NetipError::prefix(value, "invalid prefix length"));
    }
    let bits = bits
        .parse::<u32>()
        .map_err(|_| NetipError::prefix(value, "invalid prefix length"))?;
    if bits > addr.bits() {
        return Err(NetipError::prefix(value, "prefix length out of range"));
    }
    Ok(Prefix { addr, bits })
}

impl Prefix {
    pub fn addr(&self) -> Addr {
        self.addr.clone()
    }
    pub fn masked(&self) -> Prefix {
        let mut addr = self.addr.clone();
        let bytes = addr.masked_bytes(self.bits);
        addr.ip = match addr.ip {
            std::net::IpAddr::V4(_) => std::net::IpAddr::V4(std::net::Ipv4Addr::from(
                <[u8; 4]>::try_from(&bytes[..4]).expect("IPv4 mask"),
            )),
            std::net::IpAddr::V6(_) => std::net::IpAddr::V6(std::net::Ipv6Addr::from(bytes)),
        };
        Prefix {
            addr,
            bits: self.bits,
        }
    }
    pub fn contains(&self, addr: &Addr) -> bool {
        if !addr.zone().is_empty() || self.addr.is4() != addr.is4() {
            return false;
        }
        {
            let mut masked = addr.clone();
            let bytes = addr.masked_bytes(self.bits);
            masked.ip = match addr.ip {
                std::net::IpAddr::V4(_) => std::net::IpAddr::V4(std::net::Ipv4Addr::from(
                    <[u8; 4]>::try_from(&bytes[..4]).expect("IPv4 mask"),
                )),
                std::net::IpAddr::V6(_) => std::net::IpAddr::V6(std::net::Ipv6Addr::from(bytes)),
            };
            masked == self.masked().addr
        }
    }
}
