use super::config_wire::go_quoted;

// parse_prefix mirrors netip.ParsePrefix accept/reject behavior with
// Go-identical error strings for the observed input shapes.
pub(crate) fn parse_prefix(s: &str) -> Result<(), String> {
    let quoted = go_quoted(s);
    let fail = |detail: String| format!("netip.ParsePrefix({quoted}): {detail}");
    let slash = match s.rfind('/') {
        Some(i) => i,
        None => return Err(fail(String::from("no '/'"))),
    };
    let (ip, bits) = (&s[..slash], &s[slash + 1..]);
    let quoted_ip = go_quoted(ip);
    let (is_v6, zone) =
        parse_addr(ip).map_err(|detail| fail(format!("ParseAddr({quoted_ip}): {detail}")))?;
    if zone {
        return Err(fail(String::from(
            "IPv6 zones cannot be present in a prefix",
        )));
    }
    // strconv.Atoi accepts signs and leading zeroes, but prefixes do not:
    // multi-character bits must start with 1-9, then parse as digits.
    if bits.len() > 1 && (bits.as_bytes()[0] < b'1' || bits.as_bytes()[0] > b'9') {
        return Err(fail(format!("bad bits after slash: {}", go_quoted(bits))));
    }
    if bits.is_empty() || !bits.bytes().all(|b| b.is_ascii_digit()) {
        return Err(fail(format!("bad bits after slash: {}", go_quoted(bits))));
    }
    let max: i64 = if is_v6 { 128 } else { 32 };
    match bits.parse::<i64>() {
        // Unparseable magnitudes are bad bits; parsed excess is out of range.
        Err(_) => Err(fail(format!("bad bits after slash: {}", go_quoted(bits)))),
        Ok(n) if n <= max => Ok(()),
        Ok(_) => Err(fail(String::from("prefix length out of range"))),
    }
}

// parse_addr mirrors netip.ParseAddr dispatch on the first of '.', ':',
// '%', returning (is_v6, has_zone).
fn parse_addr(ip: &str) -> Result<(bool, bool), String> {
    for b in ip.bytes() {
        match b {
            b'.' => {
                parse_ipv4(ip)?;
                return Ok((false, false));
            }
            b':' => {
                let zone = parse_ipv6(ip)?;
                return Ok((true, zone));
            }
            b'%' => return Err(String::from("missing IPv6 address")),
            _ => {}
        }
    }
    Err(String::from("unable to parse IP"))
}

// parse_ipv4_fields is a line-for-line port of netip's scanner: errors
// fire left to right, and `at` is the unparsed remainder of the slice.
fn parse_ipv4_fields(s: &str) -> Result<(), String> {
    let b = s.as_bytes();
    let mut val: u32 = 0;
    let mut pos = 0;
    let mut dig_len = 0;
    for i in 0..b.len() {
        if b[i].is_ascii_digit() {
            if dig_len == 1 && val == 0 {
                return Err(String::from("IPv4 field has octet with leading zero"));
            }
            val = val * 10 + (b[i] - b'0') as u32;
            dig_len += 1;
            if val > 255 {
                return Err(String::from("IPv4 field has value >255"));
            }
        } else if b[i] == b'.' {
            if i == 0 || i == b.len() - 1 || b[i - 1] == b'.' {
                return Err(format!(
                    "IPv4 field must have at least one digit (at {})",
                    go_quoted(&s[i..])
                ));
            }
            if pos == 3 {
                return Err(String::from("IPv4 address too long"));
            }
            pos += 1;
            val = 0;
            dig_len = 0;
        } else {
            return Err(format!("unexpected character (at {})", go_quoted(&s[i..])));
        }
    }
    if pos < 3 {
        return Err(String::from("IPv4 address too short"));
    }
    Ok(())
}

fn parse_ipv4(ip: &str) -> Result<(), String> {
    parse_ipv4_fields(ip)
}

// parse_ipv6 ports netip's progressive loop, returning has_zone. `at`
// reports the unparsed remainder, never the whole input.
fn parse_ipv6(input: &str) -> Result<bool, String> {
    let mut s = input;
    let mut has_zone = false;
    if let Some(i) = s.find('%') {
        if s[i + 1..].is_empty() {
            return Err(String::from("zone must be a non-empty string"));
        }
        has_zone = true;
        s = &s[..i];
    }
    let mut ellipsis = false;
    if s.len() >= 2 && s.starts_with("::") {
        ellipsis = true;
        s = &s[2..];
        if s.is_empty() {
            return Ok(has_zone);
        }
    }
    let mut i = 0;
    while i < 16 {
        let sb = s.as_bytes();
        let mut off = 0;
        while off < sb.len() && sb[off].is_ascii_hexdigit() {
            off += 1;
        }
        if off > 4 {
            return Err(format!(
                "each group must have 4 or less digits (at {})",
                go_quoted(s)
            ));
        }
        if off == 0 {
            return Err(format!(
                "each colon-separated field must have at least one digit (at {})",
                go_quoted(s)
            ));
        }
        if off < sb.len() && sb[off] == b'.' {
            if !ellipsis && i != 12 {
                return Err(format!(
                    "embedded IPv4 address must replace the final 2 fields of the address (at {})",
                    go_quoted(s)
                ));
            }
            if i + 4 > 16 {
                return Err(format!(
                    "too many hex fields to fit an embedded IPv4 at the end of the address (at {})",
                    go_quoted(s)
                ));
            }
            parse_ipv4_fields(s)?;
            s = "";
            i += 4;
            break;
        }
        i += 2;
        s = &s[off..];
        if s.is_empty() {
            break;
        }
        if !s.starts_with(':') {
            return Err(format!(
                "unexpected character, want colon (at {})",
                go_quoted(s)
            ));
        } else if s.len() == 1 {
            return Err(format!(
                "colon must be followed by more characters (at {})",
                go_quoted(s)
            ));
        }
        s = &s[1..];
        if s.starts_with(':') {
            if ellipsis {
                return Err(format!("multiple :: in address (at {})", go_quoted(s)));
            }
            ellipsis = true;
            s = &s[1..];
            if s.is_empty() {
                break;
            }
        }
    }
    if !s.is_empty() {
        return Err(format!(
            "trailing garbage after address (at {})",
            go_quoted(s)
        ));
    }
    if i < 16 {
        if !ellipsis {
            return Err(String::from("address string too short"));
        }
    } else if ellipsis {
        return Err(String::from(
            "the :: must expand to at least one field of zeros",
        ));
    }
    Ok(has_zone)
}

#[cfg(test)]
#[path = "network_tests.rs"]
mod network_tests;
