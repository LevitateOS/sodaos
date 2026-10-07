use super::address::Addr;
use std::net::{IpAddr, Ipv4Addr};

impl Addr {
    pub fn is4(&self) -> bool {
        self.ip.is_ipv4()
    }
    pub fn is6(&self) -> bool {
        self.ip.is_ipv6()
    }
    pub fn is4_in6(&self) -> bool {
        matches!(self.ip, IpAddr::V6(ip) if ip.to_ipv4_mapped().is_some())
    }
    pub fn zone(&self) -> &[u8] {
        self.zone.as_deref().unwrap_or_default()
    }
    pub fn is_private(&self) -> bool {
        match self.ip {
            IpAddr::V4(ip) => is_private_v4(ip),
            IpAddr::V6(ip) => ip
                .to_ipv4_mapped()
                .map(is_private_v4)
                .unwrap_or_else(|| ip.octets()[0] & 0xfe == 0xfc),
        }
    }
    pub fn to_string_go(&self) -> Vec<u8> {
        let value = self.ip.to_string();
        let mut out = value.into_bytes();
        if let Some(zone) = &self.zone {
            out.push(b'%');
            out.extend_from_slice(zone);
        }
        out
    }
    pub(super) fn bits(&self) -> u32 {
        if self.is4() {
            32
        } else {
            128
        }
    }
    pub(super) fn masked_bytes(&self, bits: u32) -> [u8; 16] {
        let mut out = [0; 16];
        match self.ip {
            IpAddr::V4(ip) => out[..4].copy_from_slice(&mask(ip.octets(), bits)),
            IpAddr::V6(ip) => out.copy_from_slice(&mask(ip.octets(), bits)),
        }
        out
    }
}

fn is_private_v4(ip: Ipv4Addr) -> bool {
    let [a, b, _, _] = ip.octets();
    a == 10 || (a == 172 && b & 0xf0 == 16) || (a == 192 && b == 168)
}

fn mask<const N: usize>(mut bytes: [u8; N], bits: u32) -> [u8; N] {
    let total = N as u32 * 8;
    let full = (bits / 8) as usize;
    let rem = bits % 8;
    if rem != 0 && full < N {
        bytes[full] &= u8::MAX << (8 - rem);
    }
    let start = full + usize::from(rem != 0);
    bytes[start..].fill(0);
    debug_assert!(bits <= total);
    bytes
}
