use super::der::{Reader, CTX0_CONS, CTX1_CONS, TAG_SEQ};
use super::Extension;

use crate::fmtx::{go_quote_into, hex_lower};
use crate::netip;
use crate::urlx;

fn is_ia5_string(value: &[u8]) -> bool {
    value.iter().all(|b| *b <= 0x7f)
}

fn ia5_error(value: &[u8]) -> String {
    let mut quoted = String::new();
    go_quote_into(&mut quoted, value);
    format!("x509: {quoted} cannot be encoded as an IA5String")
}

pub(super) fn quoted(value: &[u8]) -> String {
    let mut out = String::new();
    go_quote_into(&mut out, value);
    out
}

/// Go `domainNameValid`, byte for byte: empty passes, trailing dots fail,
/// labels must be non-empty printable ASCII.
fn domain_name_valid(value: &[u8], constraint: bool) -> bool {
    if value.is_empty() {
        return true;
    }
    if value[value.len() - 1] == b'.' {
        return false;
    }
    let mut s = value;
    if constraint && s[0] == b'.' {
        s = &s[1..];
    }
    let mut last_dot: Option<usize> = None;
    let mut i = 0;
    while i <= s.len() {
        if i < s.len() && (s[i] < 33 || s[i] > 126) {
            return false;
        }
        if i == s.len() || s[i] == b'.' {
            let label_len = i - last_dot.map(|d| d + 1).unwrap_or(0);
            if label_len == 0 {
                return false;
            }
            last_dot = Some(i);
        }
        i += 1;
    }
    true
}

pub(super) fn parse_san(value: &[u8]) -> Result<bool, String> {
    let mut reader = Reader::new(value);
    let contents = reader
        .read_asn1(TAG_SEQ)
        .ok_or_else(|| String::from("x509: invalid subject alternative names"))?;
    let mut reader = Reader::new(contents);
    let mut parsed_any = false;
    while !reader.is_empty() {
        let (tag, data) = reader
            .read_any()
            .ok_or_else(|| String::from("x509: invalid subject alternative name"))?;
        match tag ^ 0x80 {
            1 => {
                if !is_ia5_string(data) {
                    return Err(String::from("x509: SAN rfc822Name is malformed"));
                }
                parsed_any = true;
            }
            2 => {
                if !is_ia5_string(data) {
                    return Err(String::from("x509: SAN dNSName is malformed"));
                }
                parsed_any = true;
            }
            6 => {
                if !is_ia5_string(data) {
                    return Err(String::from(
                        "x509: SAN uniformResourceIdentifier is malformed",
                    ));
                }
                let uri_str = std::str::from_utf8(data).map_err(|_| {
                    String::from("x509: SAN uniformResourceIdentifier is malformed")
                })?;
                let uri = urlx::parse(uri_str).map_err(|err| {
                    format!("x509: cannot parse URI {}: {}", quoted(data), err.text())
                })?;
                if !uri.host.is_empty() && !domain_name_valid(&uri.host, false) {
                    return Err(format!(
                        "x509: cannot parse URI {}: invalid domain",
                        quoted(data)
                    ));
                }
                parsed_any = true;
            }
            7 => {
                if data.len() != 4 && data.len() != 16 {
                    return Err(format!(
                        "x509: cannot parse IP address of length {}",
                        data.len()
                    ));
                }
                parsed_any = true;
            }
            _ => {}
        }
    }
    Ok(parsed_any)
}

fn is_valid_ip_mask(mask: &[u8]) -> bool {
    let mut seen_zero = false;
    for b in mask {
        if seen_zero {
            if *b != 0 {
                return false;
            }
            continue;
        }
        match b {
            0x00 | 0x80 | 0xc0 | 0xe0 | 0xf0 | 0xf8 | 0xfc | 0xfe => seen_zero = true,
            0xff => {}
            _ => return false,
        }
    }
    true
}

fn domain_to_reverse_labels(domain: &[u8]) -> bool {
    let mut rest = domain;
    let mut labels = Vec::new();
    while !rest.is_empty() {
        match rest.iter().rposition(|b| *b == b'.') {
            None => {
                labels.push(rest);
                rest = b"";
            }
            Some(i) => {
                labels.push(&rest[i + 1..]);
                rest = &rest[..i];
                if i == 0 {
                    labels.push(b"".as_slice());
                }
            }
        }
    }
    if labels.first().is_some_and(|label| label.is_empty()) {
        return false;
    }
    for label in labels {
        if label.is_empty() {
            return false;
        }
        if label.iter().any(|b| *b < 33 || *b > 126) {
            return false;
        }
    }
    true
}

/// Go `parseRFC2821Mailbox`, acceptance only.
pub(super) fn parse_rfc2821_mailbox(input: &[u8]) -> bool {
    if input.is_empty() {
        return false;
    }
    let mut rest = input;
    let mut local: Vec<u8> = Vec::new();
    if rest[0] == b'"' {
        rest = &rest[1..];
        loop {
            if rest.is_empty() {
                return false;
            }
            let c = rest[0];
            rest = &rest[1..];
            if c == b'"' {
                break;
            } else if c == b'\\' {
                if rest.is_empty() {
                    return false;
                }
                let e = rest[0];
                if e == 11 || e == 12 || (1..=9).contains(&e) || (14..=127).contains(&e) {
                    local.push(e);
                    rest = &rest[1..];
                } else {
                    return false;
                }
            } else if c == 11
                || c == 12
                || c == 32
                || c == 33
                || c == 127
                || (1..=8).contains(&c)
                || (14..=31).contains(&c)
                || (35..=91).contains(&c)
                || (93..=126).contains(&c)
            {
                local.push(c);
            } else {
                return false;
            }
        }
    } else {
        loop {
            if rest.is_empty() {
                break;
            }
            let c = rest[0];
            if c == b'\\' {
                rest = &rest[1..];
                if rest.is_empty() {
                    return false;
                }
                local.push(rest[0]);
                rest = &rest[1..];
            } else if c.is_ascii_alphanumeric()
                || matches!(
                    c,
                    b'!' | b'#'
                        | b'$'
                        | b'%'
                        | b'&'
                        | b'\''
                        | b'*'
                        | b'+'
                        | b'-'
                        | b'/'
                        | b'='
                        | b'?'
                        | b'^'
                        | b'_'
                        | b'`'
                        | b'{'
                        | b'|'
                        | b'}'
                        | b'~'
                        | b'.'
                )
            {
                local.push(c);
                rest = &rest[1..];
            } else {
                break;
            }
        }
        if local.is_empty() {
            return false;
        }
        if local[0] == b'.' || local[local.len() - 1] == b'.' {
            return false;
        }
        if local.windows(2).any(|w| w == b"..") {
            return false;
        }
    }
    if rest.is_empty() || rest[0] != b'@' {
        return false;
    }
    domain_to_reverse_labels(&rest[1..])
}

fn name_constraint_values(subtrees: &[u8]) -> Result<bool, String> {
    let mut reader = Reader::new(subtrees);
    let mut unhandled = false;
    while !reader.is_empty() {
        let seq = reader
            .read_asn1(TAG_SEQ)
            .ok_or_else(|| String::from("x509: invalid NameConstraints extension"))?;
        let mut seq = Reader::new(seq);
        let (tag, value) = seq
            .read_any()
            .ok_or_else(|| String::from("x509: invalid NameConstraints extension"))?;
        match tag {
            0x82 => {
                if !is_ia5_string(value) {
                    return Err(format!(
                        "x509: invalid constraint value: {}",
                        ia5_error(value)
                    ));
                }
                if !domain_name_valid(value, true) {
                    return Err(format!(
                        "x509: failed to parse dnsName constraint {}",
                        quoted(value)
                    ));
                }
            }
            0x87 => {
                let mask = match value.len() {
                    8 => &value[4..],
                    32 => &value[16..],
                    len => {
                        return Err(format!(
                            "x509: IP constraint contained value of length {len}"
                        ));
                    }
                };
                if !is_valid_ip_mask(mask) {
                    return Err(format!(
                        "x509: IP constraint contained invalid mask {}",
                        hex_lower(mask)
                    ));
                }
            }
            0x81 => {
                if !is_ia5_string(value) {
                    return Err(format!(
                        "x509: invalid constraint value: {}",
                        ia5_error(value)
                    ));
                }
                if value.contains(&b'@') {
                    if !parse_rfc2821_mailbox(value) {
                        return Err(format!(
                            "x509: failed to parse rfc822Name constraint {}",
                            quoted(value)
                        ));
                    }
                } else if !domain_name_valid(value, true) {
                    return Err(format!(
                        "x509: failed to parse rfc822Name constraint {}",
                        quoted(value)
                    ));
                }
            }
            0x86 => {
                if !is_ia5_string(value) {
                    return Err(format!(
                        "x509: invalid constraint value: {}",
                        ia5_error(value)
                    ));
                }
                let is_ip = netip::parse_addr_bytes(value)
                    .map(|addr| addr.zone().is_empty())
                    .unwrap_or(false);
                if is_ip {
                    return Err(format!(
                        "x509: failed to parse URI constraint {}: cannot be IP address",
                        quoted(value)
                    ));
                }
                if !domain_name_valid(value, true) {
                    return Err(format!(
                        "x509: failed to parse URI constraint {}",
                        quoted(value)
                    ));
                }
            }
            _ => unhandled = true,
        }
    }
    Ok(unhandled)
}

pub(super) fn parse_name_constraints(ext: &Extension<'_>) -> Result<bool, String> {
    let invalid = || String::from("x509: invalid NameConstraints extension");
    let mut outer = Reader::new(ext.value);
    let toplevel = outer.read_asn1(TAG_SEQ).ok_or_else(invalid)?;
    if !outer.is_empty() {
        return Err(invalid());
    }
    let mut toplevel = Reader::new(toplevel);
    let permitted = toplevel.read_optional(CTX0_CONS).ok_or_else(invalid)?;
    let excluded = toplevel.read_optional(CTX1_CONS).ok_or_else(invalid)?;
    if !toplevel.is_empty() {
        return Err(invalid());
    }
    let permitted_len = permitted.map(|s| s.len()).unwrap_or(0);
    let excluded_len = excluded.map(|s| s.len()).unwrap_or(0);
    if permitted.is_none() && excluded.is_none() || permitted_len == 0 && excluded_len == 0 {
        return Err(String::from("x509: empty name constraints extension"));
    }
    let mut unhandled = false;
    if let Some(permitted) = permitted {
        unhandled |= name_constraint_values(permitted)?;
    }
    if let Some(excluded) = excluded {
        unhandled |= name_constraint_values(excluded)?;
    }
    let _ = ext.critical;
    Ok(unhandled)
}
