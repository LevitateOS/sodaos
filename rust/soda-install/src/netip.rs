//! Go `net/netip` subset: `ParseAddr`/`ParsePrefix` with Go's exact
//! accept/reject behavior and error text, `Is4`/`Is6`/`Is4In6`/`IsPrivate`,
//! canonical `String`, `Masked`, prefix `Contains`.
//!
//! Byte fidelity: Go strings are byte sequences, and `ParseAddr` operates on
//! bytes (a zone may hold arbitrary non-UTF-8 bytes after URL unescaping, and
//! parse errors quote the raw input). The parser core therefore runs on
//! `&[u8]`; [`parse_addr`] is a thin `&str` wrapper. Zones are unbounded and
//! unvalidated past non-emptiness, as in Go. `GODEBUG` has no `netip` knobs.

use crate::fmtx::go_quote_into;
use std::fmt;

/// Parsed IP address: 16 raw bytes plus an optional zone. IPv4 is stored
/// as the 4-byte address with [`Addr::is4`] set (never 4-in-6 mapped).
#[derive(Debug, Clone, PartialEq, Eq)]
pub struct Addr {
    bytes: [u8; 16],
    is4: bool,
    zone: Option<Vec<u8>>,
}

/// A Go `parseAddrError`/`parsePrefixError`, preformatted exactly as Go's
/// `Error()` renders it (`ParseAddr("in"): msg` with optional `(at "at")`,
/// or `netip.ParsePrefix("in"): msg`).
#[derive(Debug, Clone, PartialEq, Eq)]
pub struct NetipError {
    text: String,
}

impl NetipError {
    fn addr(input: &[u8], msg: &str, at: Option<&[u8]>) -> NetipError {
        let mut text = String::from("ParseAddr(");
        go_quote_into(&mut text, input);
        text.push_str("): ");
        text.push_str(msg);
        if let Some(at) = at {
            text.push_str(" (at ");
            go_quote_into(&mut text, at);
            text.push(')');
        }
        NetipError { text }
    }

    fn prefix(input: &str, msg: &str) -> NetipError {
        let mut text = String::from("netip.ParsePrefix(");
        go_quote_into(&mut text, input.as_bytes());
        text.push_str("): ");
        text.push_str(msg);
        NetipError { text }
    }

    /// The exact Go error text.
    pub fn text(&self) -> &str {
        &self.text
    }
}

impl fmt::Display for NetipError {
    fn fmt(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
        f.write_str(&self.text)
    }
}

impl std::error::Error for NetipError {}

/// Go `netip.ParseAddr`: the first of `.`, `:`, `%` selects the IPv4 path,
/// the IPv6 path, or the missing-address error; anything else is unparseable.
pub fn parse_addr(value: &str) -> Result<Addr, NetipError> {
    parse_addr_bytes(value.as_bytes())
}

/// Byte-oriented `ParseAddr` core. Go parses bytes, so callers holding
/// unescaped URL bytes (which may be invalid UTF-8) parse them directly.
pub fn parse_addr_bytes(input: &[u8]) -> Result<Addr, NetipError> {
    for b in input.iter() {
        match b {
            b'.' => return parse_v4(input),
            b':' => return parse_v6(input),
            b'%' => return Err(NetipError::addr(input, "missing IPv6 address", None)),
            _ => {}
        }
    }
    Err(NetipError::addr(input, "unable to parse IP", None))
}

fn parse_v4(input: &[u8]) -> Result<Addr, NetipError> {
    let mut fields = [0u8; 4];
    parse_v4_fields(input, 0, input.len(), &mut fields)?;
    let mut bytes = [0u8; 16];
    bytes[..4].copy_from_slice(&fields);
    Ok(Addr {
        bytes,
        is4: true,
        zone: None,
    })
}

/// Go `parseIPv4Fields`: strict dotted quads, no leading zeros, values
/// `<= 255`, exactly four fields. `in` is the full original input (which may
/// carry an IPv6 zone suffix when parsing an embedded tail); `off..end`
/// selects the dotted-quad slice; `at` substrings are relative to it.
fn parse_v4_fields(
    input: &[u8],
    off: usize,
    end: usize,
    fields: &mut [u8],
) -> Result<(), NetipError> {
    let s = &input[off..end];
    let mut val = 0u32;
    let mut pos = 0usize;
    let mut dig_len = 0u32;
    let mut i = 0;
    while i < s.len() {
        let b = s[i];
        if b.is_ascii_digit() {
            if dig_len == 1 && val == 0 {
                return Err(NetipError::addr(
                    input,
                    "IPv4 field has octet with leading zero",
                    None,
                ));
            }
            val = val * 10 + u32::from(b - b'0');
            dig_len += 1;
            if val > 255 {
                return Err(NetipError::addr(input, "IPv4 field has value >255", None));
            }
        } else if b == b'.' {
            if i == 0 || i == s.len() - 1 || s[i - 1] == b'.' {
                return Err(NetipError::addr(
                    input,
                    "IPv4 field must have at least one digit",
                    Some(&s[i..]),
                ));
            }
            if pos == 3 {
                return Err(NetipError::addr(input, "IPv4 address too long", None));
            }
            fields[pos] = val as u8;
            pos += 1;
            val = 0;
            dig_len = 0;
        } else {
            return Err(NetipError::addr(
                input,
                "unexpected character",
                Some(&s[i..]),
            ));
        }
        i += 1;
    }
    if pos < 3 {
        return Err(NetipError::addr(input, "IPv4 address too short", None));
    }
    fields[3] = val as u8;
    Ok(())
}

/// Go `parseIPv6`, line for line: the zone splits at the FIRST `%` and is
/// stored unvalidated past non-emptiness; a leading `::` is an ellipsis at
/// zero; then hex groups with colon rules and an optional embedded dotted
/// quad that must supply exactly the final 32 bits.
fn parse_v6(input: &[u8]) -> Result<Addr, NetipError> {
    let mut s = input;
    let mut zone: &[u8] = b"";
    if let Some(i) = input.iter().position(|b| *b == b'%') {
        s = &input[..i];
        zone = &input[i + 1..];
        if zone.is_empty() {
            return Err(NetipError::addr(
                input,
                "zone must be a non-empty string",
                None,
            ));
        }
    }

    let mut ip = [0u8; 16];
    let mut ellipsis: i32 = -1;

    if s.len() >= 2 && s[0] == b':' && s[1] == b':' {
        ellipsis = 0;
        s = &s[2..];
        if s.is_empty() {
            return Ok(v6_with_zone(ip, zone));
        }
    }

    let mut i = 0usize;
    while i < 16 {
        let mut off = 0usize;
        let mut acc = 0u32;
        while off < s.len() {
            let c = s[off];
            if c.is_ascii_digit() {
                acc = (acc << 4) + u32::from(c - b'0');
            } else if (b'a'..=b'f').contains(&c) {
                acc = (acc << 4) + u32::from(c - b'a' + 10);
            } else if (b'A'..=b'F').contains(&c) {
                acc = (acc << 4) + u32::from(c - b'A' + 10);
            } else {
                break;
            }
            if off > 3 {
                return Err(NetipError::addr(
                    input,
                    "each group must have 4 or less digits",
                    Some(s),
                ));
            }
            if acc > 0xffff {
                // Unreachable in practice (four hex digits cannot exceed
                // 0xffff), kept to mirror Go's overflow check.
                return Err(NetipError::addr(
                    input,
                    "IPv6 field has value >=2^16",
                    Some(s),
                ));
            }
            off += 1;
        }
        if off == 0 {
            return Err(NetipError::addr(
                input,
                "each colon-separated field must have at least one digit",
                Some(s),
            ));
        }

        if off < s.len() && s[off] == b'.' {
            if ellipsis < 0 && i != 12 {
                return Err(NetipError::addr(
                    input,
                    "embedded IPv4 address must replace the final 2 fields of the address",
                    Some(s),
                ));
            }
            if i + 4 > 16 {
                return Err(NetipError::addr(
                    input,
                    "too many hex fields to fit an embedded IPv4 at the end of the address",
                    Some(s),
                ));
            }
            let mut end = input.len();
            if !zone.is_empty() {
                end -= zone.len() + 1;
            }
            parse_v4_fields(input, end - s.len(), end, &mut ip[i..i + 4])?;
            s = b"";
            i += 4;
            break;
        }

        ip[i] = (acc >> 8) as u8;
        ip[i + 1] = acc as u8;
        i += 2;

        s = &s[off..];
        if s.is_empty() {
            break;
        }
        if s[0] != b':' {
            return Err(NetipError::addr(
                input,
                "unexpected character, want colon",
                Some(s),
            ));
        } else if s.len() == 1 {
            return Err(NetipError::addr(
                input,
                "colon must be followed by more characters",
                Some(s),
            ));
        }
        s = &s[1..];

        if s[0] == b':' {
            if ellipsis >= 0 {
                return Err(NetipError::addr(input, "multiple :: in address", Some(s)));
            }
            ellipsis = i as i32;
            s = &s[1..];
            if s.is_empty() {
                break;
            }
        }
    }

    if !s.is_empty() {
        return Err(NetipError::addr(
            input,
            "trailing garbage after address",
            Some(s),
        ));
    }

    if i < 16 {
        if ellipsis < 0 {
            return Err(NetipError::addr(input, "address string too short", None));
        }
        let n = 16 - i;
        let ellipsis = ellipsis as usize;
        let mut j = i;
        while j > ellipsis {
            j -= 1;
            ip[j + n] = ip[j];
        }
        ip[ellipsis..ellipsis + n].fill(0);
    } else if ellipsis >= 0 {
        return Err(NetipError::addr(
            input,
            "the :: must expand to at least one field of zeros",
            None,
        ));
    }
    Ok(v6_with_zone(ip, zone))
}

fn v6_with_zone(ip: [u8; 16], zone: &[u8]) -> Addr {
    Addr {
        bytes: ip,
        is4: false,
        zone: if zone.is_empty() {
            None
        } else {
            Some(zone.to_vec())
        },
    }
}

impl Addr {
    pub fn is4(&self) -> bool {
        self.is4
    }
    pub fn is6(&self) -> bool {
        !self.is4
    }
    /// Go `Is4In6`: `::ffff:0:0/96` exactly (zone ignored, as in Go).
    pub fn is4_in6(&self) -> bool {
        !self.is4
            && self.bytes[..10].iter().all(|b| *b == 0)
            && self.bytes[10] == 0xff
            && self.bytes[11] == 0xff
    }
    /// Raw zone bytes, empty when absent. Go zones are unvalidated byte
    /// strings and may be invalid UTF-8 after URL unescaping.
    pub fn zone(&self) -> &[u8] {
        self.zone.as_deref().unwrap_or(b"")
    }
    /// Go `IsPrivate`: 4-in-6 unmapped to v4 first, then v4 `10/8`
    /// `172.16/12` `192.168/16`, v6 `fc00::/7`.
    pub fn is_private(&self) -> bool {
        if self.is4 {
            return is_private_v4(&self.bytes[..4]);
        }
        if self.is4_in6() {
            return is_private_v4(&self.bytes[12..16]);
        }
        self.bytes[0] & 0xfe == 0xfc
    }
    /// Go `Addr.String`: canonical dotted quad, or compressed lowercase
    /// IPv6 (longest zero run of length >= 2, first wins ties; 4-in-6 keeps
    /// the dotted tail), plus the raw `%zone` bytes.
    pub fn to_string_go(&self) -> Vec<u8> {
        let mut out = Vec::new();
        if self.is4 {
            push_dot(&mut out, &self.bytes[..4]);
        } else if self.is4_in6() {
            out.extend_from_slice(b"::ffff:");
            push_dot(&mut out, &self.bytes[12..16]);
        } else {
            let mut groups = [0u16; 8];
            for (i, group) in groups.iter_mut().enumerate() {
                *group = u16::from_be_bytes([self.bytes[2 * i], self.bytes[2 * i + 1]]);
            }
            let mut best_start = 0;
            let mut best_len = 0;
            let mut i = 0;
            while i < 8 {
                if groups[i] == 0 {
                    let mut j = i;
                    while j < 8 && groups[j] == 0 {
                        j += 1;
                    }
                    if j - i > best_len {
                        best_start = i;
                        best_len = j - i;
                    }
                    i = j;
                } else {
                    i += 1;
                }
            }
            if best_len < 2 {
                best_len = 0;
            }
            let mut first = true;
            let mut i = 0;
            while i < 8 {
                if best_len > 0 && i == best_start {
                    out.extend_from_slice(b"::");
                    first = true;
                    i += best_len;
                    continue;
                }
                if !first {
                    out.push(b':');
                }
                first = false;
                push_hex(&mut out, groups[i]);
                i += 1;
            }
            if out.is_empty() {
                out.extend_from_slice(b"::");
            }
        }
        if let Some(zone) = &self.zone {
            out.push(b'%');
            out.extend_from_slice(zone);
        }
        out
    }
    fn bits(&self) -> u32 {
        if self.is4 {
            32
        } else {
            128
        }
    }
    fn masked_bytes(&self, bits: u32) -> [u8; 16] {
        let mut out = self.bytes;
        let total = if self.is4 { 32 } else { 128 };
        for i in bits..total {
            let byte = (i / 8) as usize;
            let bit = 7 - (i % 8);
            out[byte] &= !(1 << bit);
        }
        out
    }
}

fn is_private_v4(b: &[u8]) -> bool {
    b[0] == 10 || (b[0] == 172 && b[1] & 0xf0 == 16) || (b[0] == 192 && b[1] == 168)
}

fn push_dot(out: &mut Vec<u8>, b: &[u8]) {
    for (i, octet) in b.iter().enumerate() {
        if i > 0 {
            out.push(b'.');
        }
        out.extend_from_slice(octet.to_string().as_bytes());
    }
}

fn push_hex(out: &mut Vec<u8>, group: u16) {
    out.extend_from_slice(format!("{group:x}").as_bytes());
}

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

#[cfg(test)]
mod tests {
    use super::*;

    fn text(bytes: &[u8]) -> &str {
        std::str::from_utf8(bytes).unwrap()
    }

    fn addr_ok(value: &str, is4: bool, is4in6: bool, priv_: bool, expect: &str) {
        let addr = parse_addr(value).unwrap_or_else(|err| panic!("parse {value:?}: {err}"));
        assert_eq!(addr.is4(), is4, "{value:?} is4");
        assert_eq!(addr.is6(), !is4, "{value:?} is6");
        assert_eq!(addr.is4_in6(), is4in6, "{value:?} is4in6");
        assert_eq!(addr.is_private(), priv_, "{value:?} private");
        assert_eq!(text(&addr.to_string_go()), expect, "{value:?} string");
    }

    fn addr_err(value: &str, expect: &str) {
        let err = parse_addr(value).unwrap_err();
        assert_eq!(err.text(), expect, "{value:?}");
    }

    #[test]
    fn oracle_addr_vectors() {
        // Oracle: TestZZOracleNetip `ADDR` + TestZZMicro `M` lines.
        for bad in [
            "192.168.1.01",
            "1.2.3.4.5",
            "0x7f.0.0.1",
            "0177.0.0.1",
            "1.2.3",
            "1::2::3",
            "1.2.3.4%eth0",
            "%eth0",
            "",
            "10.0.0.256",
            "1.2.3.4.",
            ".1.2.3.4",
            "1.2.3.4 ",
            " 1.2.3.4",
            "010.0.0.1",
            "1.02.3.4",
            "fe80::1%",
        ] {
            assert!(parse_addr(bad).is_err(), "accepted {bad:?}");
        }
        addr_ok("192.168.1.1", true, false, true, "192.168.1.1");
        addr_ok("0.0.0.0", true, false, false, "0.0.0.0");
        addr_ok("255.255.255.255", true, false, false, "255.255.255.255");
        addr_ok("::", false, false, false, "::");
        addr_ok("::1", false, false, false, "::1");
        addr_ok("::ffff:1.2.3.4", false, true, false, "::ffff:1.2.3.4");
        addr_ok("::FFFF:1.2.3.4", false, true, false, "::ffff:1.2.3.4");
        addr_ok(
            "0:0:0:0:0:ffff:1.2.3.4",
            false,
            true,
            false,
            "::ffff:1.2.3.4",
        );
        addr_ok("64:ff9b::1.2.3.4", false, false, false, "64:ff9b::102:304");
        addr_ok("2001:db8::1", false, false, false, "2001:db8::1");
        addr_ok("fe80::1", false, false, false, "fe80::1");
        addr_ok("ff02::1", false, false, false, "ff02::1");
        addr_ok("100::1", false, false, false, "100::1");
        addr_ok("FD00::123", false, false, true, "fd00::123");
        addr_ok("fd00::123", false, false, true, "fd00::123");
        addr_ok("100.90.1.2", true, false, false, "100.90.1.2");
        addr_ok("169.254.1.2", true, false, false, "169.254.1.2");
        addr_ok(
            "2001:db8:0:1:1:1:1:1",
            false,
            false,
            false,
            "2001:db8:0:1:1:1:1:1",
        );
        addr_ok("1:0:0:2:0:0:0:3", false, false, false, "1:0:0:2::3");
        addr_ok("1:0:0:2:0:0:3:4", false, false, false, "1::2:0:0:3:4");
        addr_ok("0:0:1:2:3:4:5:6", false, false, false, "::1:2:3:4:5:6");
        addr_ok("1:2:3:4:5:6:0:0", false, false, false, "1:2:3:4:5:6::");
        addr_ok("0:1:2:3:4:5:6:7", false, false, false, "0:1:2:3:4:5:6:7");
        addr_ok("1:2:3:4:5:6:7:0", false, false, false, "1:2:3:4:5:6:7:0");
        addr_ok("::ffff:10.0.0.1", false, true, true, "::ffff:10.0.0.1");
        addr_ok("::ffff:0:1", false, true, false, "::ffff:0.0.0.1");
        addr_ok("1::", false, false, false, "1::");
        addr_ok("::1:0", false, false, false, "::1:0");
        addr_ok("64:ff9b:0:0:0:0:1:2", false, false, false, "64:ff9b::1:2");
        addr_ok("::ffff:0.0.0.0", false, true, false, "::ffff:0.0.0.0");
        addr_ok("0:0:0:0:0:0:0:1", false, false, false, "::1");
        addr_ok("0:0:0:0:0:0:13.1.68.3", false, false, false, "::d01:4403");
        for bad in [
            "1.2.3.4::5",
            "1.2.3.4::",
            "0:0:0:0:0:13.1.68.3:0",
            "1:2:3:4:5:6:7:8::",
            "1:2:3:4:5:6:7:8:9",
            "1:2:3:4:5:6:7:8:",
            ":1:2:3:4:5:6:7:8",
            "1:2:3:4::5:6:7:8",
            "1:2:3:4:5:255.255.255.255:7",
        ] {
            assert!(parse_addr(bad).is_err(), "accepted {bad:?}");
        }
        addr_ok("::1.2.3.4", false, false, false, "::102:304");
        addr_ok("1::2.3.4.5", false, false, false, "1::203:405");
        addr_ok("1:2:3:4:5:6:7::", false, false, false, "1:2:3:4:5:6:7:0");
        addr_ok("::1:2:3:4:5:6:7", false, false, false, "0:1:2:3:4:5:6:7");
        addr_ok("::ffff", false, false, false, "::ffff");
        addr_ok("ffff::", false, false, false, "ffff::");
        addr_ok(
            "1:2:3:4:5:6:255.255.255.255",
            false,
            false,
            false,
            "1:2:3:4:5:6:ffff:ffff",
        );
        let zoned = parse_addr("fe80::1%eth0").unwrap();
        assert_eq!(zoned.zone(), b"eth0");
        assert_eq!(text(&zoned.to_string_go()), "fe80::1%eth0");
    }

    #[test]
    fn addr_error_text_matches_go() {
        addr_err("", "ParseAddr(\"\"): unable to parse IP");
        addr_err("999", "ParseAddr(\"999\"): unable to parse IP");
        addr_err("%eth0", "ParseAddr(\"%eth0\"): missing IPv6 address");
        addr_err(
            "fe80::1%",
            "ParseAddr(\"fe80::1%\"): zone must be a non-empty string",
        );
        addr_err(
            "1.2.3.4.5",
            "ParseAddr(\"1.2.3.4.5\"): IPv4 address too long",
        );
        addr_err("1.2.3", "ParseAddr(\"1.2.3\"): IPv4 address too short");
        addr_err(
            "10.0.0.256",
            "ParseAddr(\"10.0.0.256\"): IPv4 field has value >255",
        );
        addr_err(
            "192.168.1.01",
            "ParseAddr(\"192.168.1.01\"): IPv4 field has octet with leading zero",
        );
        addr_err(
            "1.2.3.4.",
            "ParseAddr(\"1.2.3.4.\"): IPv4 field must have at least one digit (at \".\")",
        );
        addr_err(
            "0x7f.0.0.1",
            "ParseAddr(\"0x7f.0.0.1\"): unexpected character (at \"x7f.0.0.1\")",
        );
        addr_err(
            "1::2::3",
            "ParseAddr(\"1::2::3\"): multiple :: in address (at \":3\")",
        );
        addr_err(
            "1:2:3:4:5:6:7:8:9",
            "ParseAddr(\"1:2:3:4:5:6:7:8:9\"): trailing garbage after address (at \"9\")",
        );
        addr_err(
            "1:2:3:4:5:6:7:8:",
            "ParseAddr(\"1:2:3:4:5:6:7:8:\"): colon must be followed by more characters (at \":\")",
        );
        addr_err(
            "1:2:3:4:5:6:7",
            "ParseAddr(\"1:2:3:4:5:6:7\"): address string too short",
        );
        addr_err(
            "1:2:3:4:5:6:7:8::",
            "ParseAddr(\"1:2:3:4:5:6:7:8::\"): the :: must expand to at least one field of zeros",
        );
        addr_err(
            "12345::",
            "ParseAddr(\"12345::\"): each group must have 4 or less digits (at \"12345::\")",
        );
        addr_err(
            ":1:2:3:4:5:6:7:8",
            "ParseAddr(\":1:2:3:4:5:6:7:8\"): each colon-separated field must have at least one digit (at \":1:2:3:4:5:6:7:8\")",
        );
        addr_err(
            "1:2:3:4:5:6:7.8.9.10:11",
            "ParseAddr(\"1:2:3:4:5:6:7.8.9.10:11\"): unexpected character (at \":11\")",
        );
        addr_err(
            "1:2:3:4:5:6:7:1.2.3.4",
            "ParseAddr(\"1:2:3:4:5:6:7:1.2.3.4\"): embedded IPv4 address must replace the final 2 fields of the address (at \"1.2.3.4\")",
        );
        addr_err(
            "1::2:3:4:5:6:7:1.2.3.4",
            "ParseAddr(\"1::2:3:4:5:6:7:1.2.3.4\"): too many hex fields to fit an embedded IPv4 at the end of the address (at \"1.2.3.4\")",
        );
        // Dotted forms take the IPv4 path even with colons later.
        addr_err(
            "1.2::3",
            "ParseAddr(\"1.2::3\"): unexpected character (at \"::3\")",
        );
        // Non-UTF-8 bytes quote per byte, as in Go.
        let err = parse_addr_bytes(b"\xff").unwrap_err();
        assert_eq!(err.text(), "ParseAddr(\"\\xff\"): unable to parse IP");
    }

    #[test]
    fn zone_splits_at_first_percent_without_cap() {
        // Go splits the zone at the FIRST `%` and never length-checks it.
        let addr = parse_addr("fe80::1%a%b").unwrap();
        assert_eq!(addr.zone(), b"a%b");
        assert_eq!(text(&addr.to_string_go()), "fe80::1%a%b");
        let long = format!("fe80::1%{}", "z".repeat(100));
        let addr = parse_addr(&long).unwrap();
        assert_eq!(addr.zone().len(), 100);
        // Zones may hold bytes that are invalid UTF-8 after URL unescaping.
        let addr = parse_addr_bytes(b"fe80::1%\xff").unwrap();
        assert_eq!(addr.zone(), b"\xff");
        assert_eq!(addr.to_string_go(), b"fe80::1%\xff");
    }

    #[test]
    fn oracle_prefix_vectors() {
        // Oracle: `PREFIX` + `MP` lines.
        for bad in [
            "10.0.0.0/33",
            "10.0.0.0/-1",
            "10.0.0.0/",
            "10.0.0.0",
            "010.0.0.0/8",
            "10.0.0.0/08",
            "10.0.0.0/8 ",
            " 10.0.0.0/8",
            "10.0.0.0/+8",
            "10.0.0.0/8/8",
            "fe80::1%eth0/64",
            "10.0.0.0/00",
            "10.0.0.0/000",
            "10.0.0.0/0008",
            "fd00::/129",
            "fd00::/00",
        ] {
            assert!(parse_prefix(bad).is_err(), "accepted {bad:?}");
        }
        for (value, masked) in [
            ("10.89.0.0/24", true),
            ("10.89.0.1/24", false),
            ("0.0.0.0/0", true),
            ("fd00::/64", true),
            ("fd00::1/64", false),
            ("10.0.0.0/0", false),
            ("::/0", true),
            ("1.2.3.4/32", true),
            ("1.2.3.4/31", true),
            ("fd00::/128", true),
            ("::ffff:1.2.3.4/128", true),
            ("::ffff:1.2.3.4/96", false),
        ] {
            let prefix = parse_prefix(value).unwrap_or_else(|err| panic!("parse {value:?}: {err}"));
            assert_eq!(prefix == prefix.masked(), masked, "{value:?} masked");
        }
        let tailscale = parse_prefix("100.64.0.0/10").unwrap();
        assert!(tailscale.contains(&parse_addr("100.90.1.2").unwrap()));
        assert!(!tailscale.contains(&parse_addr("192.168.1.1").unwrap()));
        assert!(!tailscale.contains(&parse_addr("fd00::1").unwrap()));
        // Zoned addresses never match: prefixes strip zones.
        let v6 = parse_prefix("fd00::/64").unwrap();
        assert!(v6.contains(&parse_addr("fd00::1").unwrap()));
        assert!(!v6.contains(&parse_addr("fd00::1%eth0").unwrap()));
    }

    #[test]
    fn prefix_error_text_matches_go() {
        let err = parse_prefix("10.0.0.0").unwrap_err();
        assert_eq!(err.text(), "netip.ParsePrefix(\"10.0.0.0\"): no '/'");
        let err = parse_prefix("10.0.0.0/33").unwrap_err();
        assert_eq!(
            err.text(),
            "netip.ParsePrefix(\"10.0.0.0/33\"): prefix length out of range"
        );
        let err = parse_prefix("10.0.0.0/08").unwrap_err();
        assert_eq!(
            err.text(),
            "netip.ParsePrefix(\"10.0.0.0/08\"): bad bits after slash: \"08\""
        );
        let err = parse_prefix("fe80::1%eth0/64").unwrap_err();
        assert_eq!(
            err.text(),
            "netip.ParsePrefix(\"fe80::1%eth0/64\"): IPv6 zones cannot be present in a prefix"
        );
        let err = parse_prefix("999/8").unwrap_err();
        assert_eq!(
            err.text(),
            "netip.ParsePrefix(\"999/8\"): ParseAddr(\"999\"): unable to parse IP"
        );
    }
}
