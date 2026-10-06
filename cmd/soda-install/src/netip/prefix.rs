use super::address::{parse_addr, Addr};
use super::NetipError;

use crate::fmtx::go_quote_into;

/// Parsed `address/bits` prefix. Zones are rejected, as in Go.
#[derive(Debug, Clone, PartialEq, Eq)]
pub struct Prefix {
    addr: Addr,
    bits: u32,
}

/// Go `netip.ParsePrefix`: the bits split at the LAST `/`; the address must
/// parse and carry no zone; bits reject signs, leading zeros, and overflow
/// exactly like Go's `Atoi` gate; out-of-family lengths fail.
pub fn parse_prefix(value: &str) -> Result<Prefix, NetipError> {
    let slash = match value.rfind('/') {
        Some(i) => i,
        None => return Err(NetipError::prefix(value, "no '/'")),
    };
    let addr = parse_addr(&value[..slash]).map_err(|err| NetipError::prefix(value, err.text()))?;
    if !addr.zone().is_empty() {
        return Err(NetipError::prefix(
            value,
            "IPv6 zones cannot be present in a prefix",
        ));
    }
    let bits_str = &value[slash + 1..];
    let bits_bytes = bits_str.as_bytes();
    if bits_bytes.len() > 1 && (bits_bytes[0] < b'1' || bits_bytes[0] > b'9') {
        return Err(NetipError::prefix(value, &bad_bits_text(bits_str)));
    }
    let mut bits: u32 = 0;
    if bits_bytes.is_empty() {
        return Err(NetipError::prefix(value, &bad_bits_text(bits_str)));
    }
    for b in bits_bytes {
        if !b.is_ascii_digit() {
            return Err(NetipError::prefix(value, &bad_bits_text(bits_str)));
        }
        bits = match bits
            .checked_mul(10)
            .and_then(|v| v.checked_add(u32::from(*b - b'0')))
        {
            Some(v) => v,
            None => return Err(NetipError::prefix(value, &bad_bits_text(bits_str))),
        };
    }
    if bits > addr.bits() {
        return Err(NetipError::prefix(value, "prefix length out of range"));
    }
    Ok(Prefix { addr, bits })
}

fn bad_bits_text(bits: &str) -> String {
    let mut quoted = String::new();
    go_quote_into(&mut quoted, bits.as_bytes());
    format!("bad bits after slash: {quoted}")
}

impl Prefix {
    pub fn addr(&self) -> Addr {
        self.addr.clone()
    }

    /// Go `Prefix.Masked`: host bits zeroed.
    pub fn masked(&self) -> Prefix {
        let mut addr = self.addr.clone();
        addr.bytes = self.addr.masked_bytes(self.bits);
        Prefix {
            addr,
            bits: self.bits,
        }
    }
    /// Go `Prefix.Contains`: zoned addresses never match (prefixes strip
    /// zones), families must agree, masked bytes must agree.
    pub fn contains(&self, addr: &Addr) -> bool {
        if !addr.zone().is_empty() {
            return false;
        }
        if self.addr.is4 != addr.is4 {
            return false;
        }
        let mut masked = addr.clone();
        masked.bytes = addr.masked_bytes(self.bits);
        masked.bytes == self.masked().addr.bytes
    }
}
