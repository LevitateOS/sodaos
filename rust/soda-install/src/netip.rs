//! Go `net/netip` subset: strict `ParseAddr`/`ParsePrefix`, `Is4`/`Is6`/
//! `Is4In6`/`IsPrivate`, canonical `String`, `Masked`, prefix `Contains`.

/// Parsed IP address: 16 raw bytes plus an optional zone. IPv4 is stored
/// as the 4-byte address with [`Addr::is4`] set (never 4-in-6 mapped).
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub struct Addr {
    bytes: [u8; 16],
    is4: bool,
    zone: Option<Zone>,
}

#[derive(Debug, Clone, Copy, PartialEq, Eq)]
struct Zone {
    bytes: [u8; 64],
    len: u8,
}

impl Zone {
    fn as_str(&self) -> &str {
        std::str::from_utf8(&self.bytes[..self.len as usize]).unwrap_or("")
    }
}

/// Go `netip.ParseAddr`: strict dotted quads (no leading zeros), strict
/// IPv6 with one `::`, optional `%zone` on IPv6 only (non-empty).
pub fn parse_addr(value: &str) -> Result<Addr, ()> {
    if value.is_empty() {
        return Err(());
    }
    if let Some(percent) = value.find('%') {
        let (head, zone) = (&value[..percent], &value[percent + 1..]);
        if zone.is_empty() || zone.len() > 64 {
            return Err(());
        }
        let mut addr = parse_no_zone(head)?;
        if addr.is4 {
            return Err(());
        }
        let mut bytes = [0u8; 64];
        bytes[..zone.len()].copy_from_slice(zone.as_bytes());
        addr.zone = Some(Zone { bytes, len: zone.len() as u8 });
        return Ok(addr);
    }
    parse_no_zone(value)
}

fn parse_no_zone(value: &str) -> Result<Addr, ()> {
    if value.is_empty() {
        return Err(());
    }
    if value.contains(':') {
        parse_v6(value)
    } else {
        parse_v4(value)
    }
}

fn parse_v4(value: &str) -> Result<Addr, ()> {
    let parts: Vec<&str> = value.split('.').collect();
    if parts.len() != 4 {
        return Err(());
    }
    let mut bytes = [0u8; 16];
    for (i, part) in parts.iter().enumerate() {
        if part.is_empty() || part.len() > 3 || !part.bytes().all(|b| b.is_ascii_digit()) {
            return Err(());
        }
        if part.len() > 1 && part.starts_with('0') {
            return Err(());
        }
        let n: u32 = part.parse().map_err(|_| ())?;
        if n > 255 {
            return Err(());
        }
        bytes[i] = n as u8;
    }
    Ok(Addr { bytes, is4: true, zone: None })
}

fn parse_v6(value: &str) -> Result<Addr, ()> {
    // Split around a single `::`, with an optional dotted-quad tail.
    let (head, tail) = match value.split_once("::") {
        Some((h, t)) => {
            if t.contains("::") {
                return Err(());
            }
            (h, Some(t))
        }
        None => (value, None),
    };
    let parse_side = |side: &str, allow_v4: bool| -> Result<(Vec<u16>, bool), ()> {
        if side.is_empty() {
            return Ok((Vec::new(), false));
        }
        let groups: Vec<&str> = side.split(':').collect();
        let mut out = Vec::new();
        for (i, group) in groups.iter().enumerate() {
            let last = i + 1 == groups.len();
            if last && allow_v4 && group.contains('.') {
                let v4 = parse_v4(group)?;
                out.push(u16::from_be_bytes([v4.bytes[0], v4.bytes[1]]));
                out.push(u16::from_be_bytes([v4.bytes[2], v4.bytes[3]]));
                return Ok((out, true));
            }
            if group.is_empty() || group.len() > 4 || !group.bytes().all(|b| b.is_ascii_hexdigit()) {
                return Err(());
            }
            out.push(u16::from_str_radix(group, 16).map_err(|_| ())?);
        }
        Ok((out, false))
    };
    let (mut groups, head_v4) = parse_side(head, true)?;
    if head_v4 && tail.is_some() {
        // A dotted quad before `::` is rejected; as the tail of the full
        // form it fills the last two groups.
        return Err(());
    }
    let tail_groups = match tail {
        Some(t) => {
            let (g, _) = parse_side(t, true)?;
            g
        }
        None => Vec::new(),
    };
    if tail.is_none() {
        if groups.len() != 8 {
            return Err(());
        }
    } else {
        if groups.len() + tail_groups.len() > 8 {
            return Err(());
        }
        // `::` must compress at least one group... except Go also accepts
        // `1:2:3:4:5:6:7::` style? No: with 8 groups total and `::`, Go
        // rejects (`1:2:3:4:5:6:7:8::` is invalid). Enforce compression.
        if groups.len() + tail_groups.len() >= 8 {
            return Err(());
        }
        let zeros = 8 - groups.len() - tail_groups.len();
        groups.extend(std::iter::repeat(0).take(zeros));
        groups.extend(tail_groups);
    }
    let mut bytes = [0u8; 16];
    for (i, g) in groups.iter().enumerate() {
        bytes[2 * i..2 * i + 2].copy_from_slice(&g.to_be_bytes());
    }
    Ok(Addr { bytes, is4: false, zone: None })
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
        !self.is4 && self.bytes[..10].iter().all(|b| *b == 0) && self.bytes[10] == 0xff && self.bytes[11] == 0xff
    }
    pub fn zone(&self) -> &str {
        self.zone.as_ref().map(Zone::as_str).unwrap_or("")
    }
    /// Go `IsPrivate`: v4 `10/8 172.16/12 192.168/16`, v6 `fc00::/7`,
    /// 4-in-6 unmapped to v4 first.
    pub fn is_private(&self) -> bool {
        if self.is4 {
            let b = &self.bytes[..4];
            return b[0] == 10
                || (b[0] == 172 && b[1] & 0xf0 == 16)
                || (b[0] == 192 && b[1] == 168);
        }
        if self.is4_in6() {
            let b = &self.bytes[12..16];
            return b[0] == 10
                || (b[0] == 172 && b[1] & 0xf0 == 16)
                || (b[0] == 192 && b[1] == 168);
        }
        self.bytes[0] & 0xfe == 0xfc
    }
    /// Go `Addr.String`: canonical dotted quad, or compressed lowercase
    /// IPv6 (longest zero run of length ≥ 2, first wins ties; 4-in-6 keeps
    /// the dotted tail), plus `%zone`.
    pub fn to_string_go(&self) -> String {
        let mut out = String::new();
        if self.is4 {
            out.push_str(&format!("{}.{}.{}.{}", self.bytes[0], self.bytes[1], self.bytes[2], self.bytes[3]));
        } else if self.is4_in6() {
            let b = &self.bytes[12..16];
            out.push_str(&format!("::ffff:{}.{}.{}.{}", b[0], b[1], b[2], b[3]));
        } else {
            let mut groups = [0u16; 8];
            for i in 0..8 {
                groups[i] = u16::from_be_bytes([self.bytes[2 * i], self.bytes[2 * i + 1]]);
            }
            // Longest zero run of length ≥ 2; first wins ties.
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
                    out.push_str("::");
                    first = true;
                    i += best_len;
                    continue;
                }
                if !first {
                    out.push(':');
                }
                first = false;
                out.push_str(&format!("{:x}", groups[i]));
                i += 1;
            }
            if out.is_empty() {
                out.push_str("::");
            }
        }
        if let Some(zone) = &self.zone {
            out.push('%');
            out.push_str(zone.as_str());
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
            let byte = if self.is4 { (i / 8) as usize } else { (i / 8) as usize };
            let bit = 7 - (i % 8);
            out[byte] &= !(1 << bit);
        }
        out
    }
}

/// Parsed `address/bits` prefix. Zones are rejected, as in Go.
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub struct Prefix {
    addr: Addr,
    bits: u32,
}

/// Go `netip.ParsePrefix`: strict address, strict bits (no leading zeros,
/// no sign, no spaces, within family size).
pub fn parse_prefix(value: &str) -> Result<Prefix, ()> {
    let (head, bits) = value.split_once('/').ok_or(())?;
    if bits.is_empty() || !bits.bytes().all(|b| b.is_ascii_digit()) {
        return Err(());
    }
    if bits.len() > 1 && bits.starts_with('0') {
        return Err(());
    }
    let addr = parse_addr(head)?;
    if !addr.zone().is_empty() {
        return Err(());
    }
    let bits: u32 = bits.parse().map_err(|_| ())?;
    if bits > addr.bits() {
        return Err(());
    }
    Ok(Prefix { addr, bits })
}

impl Prefix {
    pub fn addr(&self) -> Addr {
        self.addr
    }
    pub fn bits(&self) -> u32 {
        self.bits
    }
    /// Go `Prefix.Masked`: host bits zeroed.
    pub fn masked(&self) -> Prefix {
        let mut addr = self.addr;
        addr.bytes = self.addr.masked_bytes(self.bits);
        Prefix { addr, bits: self.bits }
    }
    /// Go `Prefix.Contains`: same family and masked equality.
    pub fn contains(&self, addr: Addr) -> bool {
        if self.addr.is4 != addr.is4 {
            return false;
        }
        let mut masked = addr;
        masked.bytes = addr.masked_bytes(self.bits);
        masked.bytes == self.masked().addr.bytes
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    fn addr_ok(value: &str, is4: bool, is4in6: bool, priv_: bool, text: &str) {
        let addr = parse_addr(value).unwrap_or_else(|_| panic!("parse {value:?}"));
        assert_eq!(addr.is4(), is4, "{value:?} is4");
        assert_eq!(addr.is6(), !is4, "{value:?} is6");
        assert_eq!(addr.is4_in6(), is4in6, "{value:?} is4in6");
        assert_eq!(addr.is_private(), priv_, "{value:?} private");
        assert_eq!(addr.to_string_go(), text, "{value:?} string");
    }

    #[test]
    fn oracle_addr_vectors() {
        // Oracle: TestZZOracleNetip `ADDR` + TestZZMicro `M` lines.
        for bad in [
            "192.168.1.01", "1.2.3.4.5", "0x7f.0.0.1", "0177.0.0.1", "1.2.3", "1::2::3",
            "1.2.3.4%eth0", "%eth0", "", "10.0.0.256", "1.2.3.4.", ".1.2.3.4", "1.2.3.4 ",
            " 1.2.3.4", "010.0.0.1", "1.02.3.4", "fe80::1%",
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
        addr_ok("0:0:0:0:0:ffff:1.2.3.4", false, true, false, "::ffff:1.2.3.4");
        addr_ok("64:ff9b::1.2.3.4", false, false, false, "64:ff9b::102:304");
        addr_ok("2001:db8::1", false, false, false, "2001:db8::1");
        addr_ok("fe80::1", false, false, false, "fe80::1");
        addr_ok("ff02::1", false, false, false, "ff02::1");
        addr_ok("100::1", false, false, false, "100::1");
        addr_ok("FD00::123", false, false, true, "fd00::123");
        addr_ok("fd00::123", false, false, true, "fd00::123");
        addr_ok("100.90.1.2", true, false, false, "100.90.1.2");
        addr_ok("169.254.1.2", true, false, false, "169.254.1.2");
        addr_ok("2001:db8:0:1:1:1:1:1", false, false, false, "2001:db8:0:1:1:1:1:1");
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
        addr_ok("1:2:3:4:5:6:255.255.255.255", false, false, false, "1:2:3:4:5:6:ffff:ffff");
        let zoned = parse_addr("fe80::1%eth0").unwrap();
        assert_eq!(zoned.zone(), "eth0");
        assert_eq!(zoned.to_string_go(), "fe80::1%eth0");
    }

    #[test]
    fn oracle_prefix_vectors() {
        // Oracle: `PREFIX` + `MP` lines.
        for bad in [
            "10.0.0.0/33", "10.0.0.0/-1", "10.0.0.0/", "10.0.0.0", "010.0.0.0/8",
            "10.0.0.0/08", "10.0.0.0/8 ", " 10.0.0.0/8", "10.0.0.0/+8", "10.0.0.0/8/8",
            "fe80::1%eth0/64", "10.0.0.0/00", "10.0.0.0/000", "10.0.0.0/0008",
            "fd00::/129", "fd00::/00",
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
            let prefix = parse_prefix(value).unwrap_or_else(|_| panic!("parse {value:?}"));
            assert_eq!(prefix == prefix.masked(), masked, "{value:?} masked");
        }
        let tailscale = parse_prefix("100.64.0.0/10").unwrap();
        assert!(tailscale.contains(parse_addr("100.90.1.2").unwrap()));
        assert!(!tailscale.contains(parse_addr("192.168.1.1").unwrap()));
        assert!(!tailscale.contains(parse_addr("fd00::1").unwrap()));
    }
}
