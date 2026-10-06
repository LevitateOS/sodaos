use std::fmt;

use crate::fmtx::go_quote_into;

/// Parsed IP address: 16 raw bytes plus an optional zone. IPv4 is stored
/// as the 4-byte address with [`Addr::is4`] set (never 4-in-6 mapped).
#[derive(Debug, Clone, PartialEq, Eq)]
pub struct Addr {
    pub(super) bytes: [u8; 16],
    pub(super) is4: bool,
    pub(super) zone: Option<Vec<u8>>,
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

    pub(super) fn prefix(input: &str, msg: &str) -> NetipError {
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
