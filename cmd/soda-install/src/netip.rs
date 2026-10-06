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

pub use self::address::{parse_addr, parse_addr_bytes, Addr, NetipError};

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

mod address;
mod address_format;

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
