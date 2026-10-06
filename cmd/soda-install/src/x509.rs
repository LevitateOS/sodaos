//! Go `crypto/x509` certificate subset used by the installer: DER parsing
//! with Go's exact accept/reject behavior and error text, plus
//! `CheckSignatureFrom` for self-signed local CA certificates.
//!
//! Design: a hand-rolled DER reader mirroring `golang.org/x/crypto/cryptobyte`
//! read strictness, with the parse/verify flow ported line for line from Go
//! 1.26.7 (`crypto/x509/parser.go`, `x509.go`). `x509-cert` was evaluated and
//! rejected: its `Rfc5280` profile enforces a 21-byte serial cap and a 1970
//! UTCTime floor that Go does not, and every parse error string would need
//! hand-mapping. Signature math delegates to `rsa`, `p224`/`p256`/`p384`/
//! `p521`, `ed25519-dalek`, and `sha2`.
//!
//! Scope: `setup` needs `IsCA`, `BasicConstraintsValid`, `KeyUsage`,
//! `Version`, the public key, the signature algorithm, `RawTBSCertificate`,
//! `Signature`, and `Raw`. Every other extension is still parsed and
//! validated (malformed values fail exactly as in Go) but its decoded value
//! is dropped. `Verify`, CRLs, CSRs, and creation APIs are not ported:
//! nothing in the installer uses them.
//!
//! `GODEBUG` escape hatches (`x509negativeserial`) are not honored: they are
//! Go-toolchain rollout controls, not installer behavior. Defaults match.
//!
//! Ed25519 verification uses `ed25519-dalek`'s `verify` (canonical `S`,
//! cofactored equation, no small-order checks), which is equivalent by
//! construction to Go's `verifyWithDom`. RSA verification failures map to
//! Go's `crypto/rsa: verification error`; Go returns that same error for
//! every RSA verification failure.

pub use self::algorithms::{PublicKeyAlgorithm, SignatureAlgorithm, KEY_USAGE_CERT_SIGN};
#[cfg(test)]
pub use self::algorithms::{
    KEY_USAGE_DECIPHER_ONLY, KEY_USAGE_DIGITAL_SIGNATURE, KEY_USAGE_KEY_ENCIPHERMENT,
};
use self::der::*;
use self::time::parse_validity;
pub use self::types::{Certificate, PublicKeyData};
use crate::fmtx::{go_quote_into, hex_lower};
use crate::netip;
use crate::urlx;

const NULL_BYTES: &[u8] = &[0x05, 0x00];

// ---------------------------------------------------------------------------
// Names, algorithm identifiers, validity, extensions.
// ---------------------------------------------------------------------------

fn is_printable(b: u8) -> bool {
    b.is_ascii_lowercase()
        || b.is_ascii_uppercase()
        || b.is_ascii_digit()
        || (0x27..=0x29).contains(&b)
        || (0x2b..=0x2f).contains(&b)
        || matches!(b, b' ' | b':' | b'=' | b'?' | b'*' | b'&')
}

fn parse_asn1_string(tag: u8, value: &[u8]) -> Result<(), String> {
    match tag {
        20 => Ok(()),
        19 => {
            if value.iter().all(|b| is_printable(*b)) {
                Ok(())
            } else {
                Err(String::from("invalid PrintableString"))
            }
        }
        12 => {
            if std::str::from_utf8(value).is_ok() {
                Ok(())
            } else {
                Err(String::from("invalid UTF-8 string"))
            }
        }
        30 => {
            if !value.len().is_multiple_of(2) {
                return Err(String::from("invalid BMPString"));
            }
            let mut units = value;
            if units.len() >= 2 && units[units.len() - 1] == 0 && units[units.len() - 2] == 0 {
                units = &units[..units.len() - 2];
            }
            let mut i = 0;
            while i < units.len() {
                let point = u16::from_be_bytes([units[i], units[i + 1]]);
                if point == 0xfffe
                    || point == 0xffff
                    || (0xfdd0..=0xfdef).contains(&point)
                    || (0xd800..=0xdfff).contains(&point)
                {
                    return Err(String::from("invalid BMPString"));
                }
                i += 2;
            }
            Ok(())
        }
        22 => {
            if value.iter().all(|b| *b <= 0x7f) {
                Ok(())
            } else {
                Err(String::from("invalid IA5String"))
            }
        }
        18 => {
            if value.iter().all(|b| b.is_ascii_digit() || *b == b' ') {
                Ok(())
            } else {
                Err(String::from("invalid NumericString"))
            }
        }
        _ => Err(format!("unsupported string type: {tag}")),
    }
}

/// Go `parseName`: validates the DER name; decoded values are dropped
/// (the installer never reads issuer/subject attributes).
fn parse_name(element: &[u8]) -> Result<(), String> {
    let mut raw = Reader::new(element);
    let contents = raw
        .read_asn1(TAG_SEQ)
        .ok_or_else(|| String::from("x509: invalid RDNSequence"))?;
    let mut raw = Reader::new(contents);
    while !raw.is_empty() {
        let set = raw
            .read_asn1(TAG_SET)
            .ok_or_else(|| String::from("x509: invalid RDNSequence"))?;
        let mut set = Reader::new(set);
        while !set.is_empty() {
            let attr = set
                .read_asn1(TAG_SEQ)
                .ok_or_else(|| String::from("x509: invalid RDNSequence: invalid attribute"))?;
            let mut attr = Reader::new(attr);
            attr.read_oid()
                .ok_or_else(|| String::from("x509: invalid RDNSequence: invalid attribute type"))?;
            let (value_tag, value) = attr.read_any().ok_or_else(|| {
                String::from("x509: invalid RDNSequence: invalid attribute value")
            })?;
            parse_asn1_string(value_tag, value).map_err(|err| {
                format!("x509: invalid RDNSequence: invalid attribute value: {err}")
            })?;
        }
    }
    Ok(())
}

/// Parsed algorithm identifier: OID contents plus the raw parameters element
/// (empty when absent). Trailing data after the parameters is ignored.
struct AlgorithmIdentifier<'a> {
    oid: &'a [u8],
    params: &'a [u8],
}

fn parse_ai(contents: &[u8]) -> Result<AlgorithmIdentifier<'_>, String> {
    let mut reader = Reader::new(contents);
    let oid = reader
        .read_oid()
        .ok_or_else(|| String::from("x509: malformed OID"))?;
    if reader.is_empty() {
        return Ok(AlgorithmIdentifier { oid, params: b"" });
    }
    let (_, params) = reader
        .read_any_element()
        .ok_or_else(|| String::from("x509: malformed parameters"))?;
    Ok(AlgorithmIdentifier { oid, params })
}

struct Extension<'a> {
    id: &'a [u8],
    critical: bool,
    value: &'a [u8],
}

fn parse_extension(contents: &[u8]) -> Result<Extension<'_>, String> {
    let mut reader = Reader::new(contents);
    let id = reader
        .read_oid()
        .ok_or_else(|| String::from("x509: malformed extension OID field"))?;
    let mut critical = false;
    if reader.peek_tag(TAG_BOOLEAN) {
        critical = reader
            .read_boolean()
            .ok_or_else(|| String::from("x509: malformed extension critical field"))?;
    }
    let value = reader
        .read_asn1(TAG_OCTET)
        .ok_or_else(|| String::from("x509: malformed extension value field"))?;
    Ok(Extension {
        id,
        critical,
        value,
    })
}

// ---------------------------------------------------------------------------
// Extension values.
// ---------------------------------------------------------------------------

fn parse_key_usage(value: &[u8]) -> Result<u16, String> {
    let mut reader = Reader::new(value);
    let (bytes, bit_len) = reader
        .read_bitstring()
        .ok_or_else(|| String::from("x509: invalid key usage"))?;
    let mut usage = 0u16;
    for i in 0..9 {
        if bit_at(bytes, bit_len, i) != 0 {
            usage |= 1 << i;
        }
    }
    Ok(usage)
}

fn parse_basic_constraints(value: &[u8]) -> Result<(bool, i64), String> {
    let mut reader = Reader::new(value);
    let contents = reader
        .read_asn1(TAG_SEQ)
        .ok_or_else(|| String::from("x509: invalid basic constraints"))?;
    let mut reader = Reader::new(contents);
    let mut is_ca = false;
    if reader.peek_tag(TAG_BOOLEAN) {
        is_ca = reader
            .read_boolean()
            .ok_or_else(|| String::from("x509: invalid basic constraints"))?;
    }
    let mut max_path_len = -1i64;
    if reader.peek_tag(TAG_INTEGER) {
        let raw = reader
            .read_uint64()
            .ok_or_else(|| String::from("x509: invalid basic constraints"))?;
        if raw > i64::MAX as u64 {
            return Err(String::from("x509: invalid basic constraints"));
        }
        max_path_len = raw as i64;
    }
    Ok((is_ca, max_path_len))
}

fn is_ia5_string(value: &[u8]) -> bool {
    value.iter().all(|b| *b <= 0x7f)
}

fn ia5_error(value: &[u8]) -> String {
    let mut quoted = String::new();
    go_quote_into(&mut quoted, value);
    format!("x509: {quoted} cannot be encoded as an IA5String")
}

fn quoted(value: &[u8]) -> String {
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

fn parse_san(value: &[u8]) -> Result<bool, String> {
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
fn parse_rfc2821_mailbox(input: &[u8]) -> bool {
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

fn parse_name_constraints(ext: &Extension<'_>) -> Result<bool, String> {
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

fn parse_crl_distribution_points(value: &[u8]) -> Result<(), String> {
    let mut reader = Reader::new(value);
    let contents = reader
        .read_asn1(TAG_SEQ)
        .ok_or_else(|| String::from("x509: invalid CRL distribution points"))?;
    let mut reader = Reader::new(contents);
    while !reader.is_empty() {
        let dp = reader
            .read_asn1(TAG_SEQ)
            .ok_or_else(|| String::from("x509: invalid CRL distribution point"))?;
        let mut dp = Reader::new(dp);
        let name = dp
            .read_optional(CTX0_CONS)
            .ok_or_else(|| String::from("x509: invalid CRL distribution point"))?;
        let Some(name) = name else {
            continue;
        };
        let mut name = Reader::new(name);
        let full = name
            .read_asn1(CTX0_CONS)
            .ok_or_else(|| String::from("x509: invalid CRL distribution point"))?;
        let mut full = Reader::new(full);
        while full.peek_tag(CTX6_PRIM) {
            full.read_asn1(CTX6_PRIM)
                .ok_or_else(|| String::from("x509: invalid CRL distribution point"))?;
        }
    }
    Ok(())
}

fn parse_authority_key_id(ext: &Extension<'_>) -> Result<(), String> {
    if ext.critical {
        return Err(String::from(
            "x509: authority key identifier incorrectly marked critical",
        ));
    }
    let mut reader = Reader::new(ext.value);
    let contents = reader
        .read_asn1(TAG_SEQ)
        .ok_or_else(|| String::from("x509: invalid authority key identifier"))?;
    let mut reader = Reader::new(contents);
    if reader.peek_tag(CTX0_PRIM) {
        reader
            .read_asn1(CTX0_PRIM)
            .ok_or_else(|| String::from("x509: invalid authority key identifier"))?;
    }
    Ok(())
}

fn parse_policy_constraints(value: &[u8]) -> Result<(), String> {
    let mut reader = Reader::new(value);
    let contents = reader
        .read_asn1(TAG_SEQ)
        .ok_or_else(|| String::from("x509: invalid policy constraints extension"))?;
    let mut reader = Reader::new(contents);
    if reader.peek_tag(CTX0_PRIM) {
        reader
            .read_int64_with_tag(CTX0_PRIM)
            .ok_or_else(|| String::from("x509: invalid policy constraints extension"))?;
    }
    if reader.peek_tag(CTX1_PRIM) {
        reader
            .read_int64_with_tag(CTX1_PRIM)
            .ok_or_else(|| String::from("x509: invalid policy constraints extension"))?;
    }
    Ok(())
}

fn parse_ext_key_usage(value: &[u8]) -> Result<(), String> {
    let mut reader = Reader::new(value);
    let contents = reader
        .read_asn1(TAG_SEQ)
        .ok_or_else(|| String::from("x509: invalid extended key usages"))?;
    let mut reader = Reader::new(contents);
    while !reader.is_empty() {
        reader
            .read_oid()
            .ok_or_else(|| String::from("x509: invalid extended key usages"))?;
    }
    Ok(())
}

fn parse_subject_key_id(ext: &Extension<'_>) -> Result<(), String> {
    if ext.critical {
        return Err(String::from(
            "x509: subject key identifier incorrectly marked critical",
        ));
    }
    let mut reader = Reader::new(ext.value);
    reader
        .read_asn1(TAG_OCTET)
        .ok_or_else(|| String::from("x509: invalid subject key identifier"))?;
    Ok(())
}

/// Go `newOIDFromDER` validation: non-empty, final byte terminates a
/// subidentifier, no `0x80` subidentifier opener.
fn valid_policy_oid(bytes: &[u8]) -> bool {
    if bytes.is_empty() || bytes[bytes.len() - 1] & 0x80 != 0 {
        return false;
    }
    let mut start = 0;
    for (i, b) in bytes.iter().enumerate() {
        if i == start && *b == 0x80 {
            return false;
        }
        if b & 0x80 == 0 {
            start = i + 1;
        }
    }
    true
}

fn parse_certificate_policies(value: &[u8]) -> Result<(), String> {
    let mut reader = Reader::new(value);
    let contents = reader
        .read_asn1(TAG_SEQ)
        .ok_or_else(|| String::from("x509: invalid certificate policies"))?;
    let mut reader = Reader::new(contents);
    let mut seen: Vec<&[u8]> = Vec::new();
    while !reader.is_empty() {
        let policy = reader
            .read_asn1(TAG_SEQ)
            .ok_or_else(|| String::from("x509: invalid certificate policies"))?;
        let mut policy = Reader::new(policy);
        let oid = policy
            .read_asn1(TAG_OID)
            .ok_or_else(|| String::from("x509: invalid certificate policies"))?;
        if seen.contains(&oid) || !valid_policy_oid(oid) {
            return Err(String::from("x509: invalid certificate policies"));
        }
        seen.push(oid);
    }
    Ok(())
}

fn parse_policy_mappings(value: &[u8]) -> Result<(), String> {
    let mut reader = Reader::new(value);
    let contents = reader
        .read_asn1(TAG_SEQ)
        .ok_or_else(|| String::from("x509: invalid policy mappings extension"))?;
    let mut reader = Reader::new(contents);
    while !reader.is_empty() {
        let mapping = reader
            .read_asn1(TAG_SEQ)
            .ok_or_else(|| String::from("x509: invalid policy mappings extension"))?;
        let mut mapping = Reader::new(mapping);
        let ok = mapping.read_asn1(TAG_OID).is_some() && mapping.read_asn1(TAG_OID).is_some();
        if !ok {
            return Err(String::from("x509: invalid policy mappings extension"));
        }
    }
    Ok(())
}

fn parse_inhibit_any_policy(value: &[u8]) -> Result<(), String> {
    let mut reader = Reader::new(value);
    reader
        .read_int64()
        .ok_or_else(|| String::from("x509: invalid inhibit any policy extension"))?;
    Ok(())
}

fn parse_authority_info_access(ext: &Extension<'_>) -> Result<(), String> {
    if ext.critical {
        return Err(String::from(
            "x509: authority info access incorrectly marked critical",
        ));
    }
    let mut reader = Reader::new(ext.value);
    let contents = reader
        .read_asn1(TAG_SEQ)
        .ok_or_else(|| String::from("x509: invalid authority info access"))?;
    let mut reader = Reader::new(contents);
    while !reader.is_empty() {
        let entry = reader
            .read_asn1(TAG_SEQ)
            .ok_or_else(|| String::from("x509: invalid authority info access"))?;
        let mut entry = Reader::new(entry);
        entry
            .read_oid()
            .ok_or_else(|| String::from("x509: invalid authority info access"))?;
        if !entry.peek_tag(CTX6_PRIM) {
            continue;
        }
        entry
            .read_asn1(CTX6_PRIM)
            .ok_or_else(|| String::from("x509: invalid authority info access"))?;
    }
    Ok(())
}

struct ProcessedExtensions {
    key_usage: u16,
    is_ca: bool,
    basic_constraints_valid: bool,
    max_path_len: i64,
    max_path_len_zero: bool,
    unhandled_critical: Vec<String>,
}

fn process_extensions(extensions: &[Extension<'_>]) -> Result<ProcessedExtensions, String> {
    let mut out = ProcessedExtensions {
        key_usage: 0,
        is_ca: false,
        basic_constraints_valid: false,
        max_path_len: 0,
        max_path_len_zero: false,
        unhandled_critical: Vec::new(),
    };
    for ext in extensions {
        let mut unhandled = false;
        if ext.id.len() == 3 && ext.id[0] == 0x55 && ext.id[1] == 0x1d {
            match ext.id[2] {
                15 => out.key_usage = parse_key_usage(ext.value)?,
                19 => {
                    let (is_ca, max_path_len) = parse_basic_constraints(ext.value)?;
                    out.is_ca = is_ca;
                    out.max_path_len = max_path_len;
                    out.basic_constraints_valid = true;
                    out.max_path_len_zero = max_path_len == 0;
                }
                17 => {
                    if !parse_san(ext.value)? {
                        unhandled = true;
                    }
                }
                30 => unhandled = parse_name_constraints(ext)?,
                31 => parse_crl_distribution_points(ext.value)?,
                35 => parse_authority_key_id(ext)?,
                36 => parse_policy_constraints(ext.value)?,
                37 => parse_ext_key_usage(ext.value)?,
                14 => parse_subject_key_id(ext)?,
                32 => parse_certificate_policies(ext.value)?,
                33 => parse_policy_mappings(ext.value)?,
                54 => parse_inhibit_any_policy(ext.value)?,
                _ => unhandled = true,
            }
        } else if ext.id == OID_AIA {
            parse_authority_info_access(ext)?;
        } else {
            unhandled = true;
        }
        if ext.critical && unhandled {
            out.unhandled_critical.push(oid_to_string(ext.id));
        }
    }
    Ok(out)
}

// ---------------------------------------------------------------------------
// Signature algorithm mapping.
// ---------------------------------------------------------------------------

/// Lenient `encoding/asn1` INTEGER into `i64`: two's complement, 1-8 bytes.
fn lenient_asn1_int(bytes: &[u8]) -> Option<i64> {
    if bytes.is_empty() || bytes.len() > 8 {
        return None;
    }
    asn1_signed(bytes)
}

/// Explicitly-tagged inner `AlgorithmIdentifier` (PSS hash/MGF buckets).
fn parse_inner_ai(bytes: &[u8]) -> Option<AlgorithmIdentifier<'_>> {
    let mut inner = Reader::new(bytes);
    let seq = inner.read_asn1(TAG_SEQ)?;
    if !inner.is_empty() {
        return None;
    }
    parse_ai(seq).ok()
}

/// Go `getSignatureAlgorithmFromAI` for RSA-PSS parameters: the parameters
/// must unmarshal as `{hash, mgf, saltLength, trailer}` with MGF1 matching
/// the message hash, salt length equal to the hash length, and default
/// trailer. Anything else yields `Unknown`.
fn pss_algorithm(params: &[u8]) -> SignatureAlgorithm {
    let mut outer = Reader::new(params);
    let contents = match outer.read_asn1(TAG_SEQ) {
        Some(c) if outer.is_empty() => c,
        _ => return SignatureAlgorithm::Unknown,
    };
    let mut reader = Reader::new(contents);
    let hash = match reader.read_asn1(CTX0_CONS) {
        Some(c) => c,
        None => return SignatureAlgorithm::Unknown,
    };
    let mgf = match reader.read_asn1(CTX1_CONS) {
        Some(c) => c,
        None => return SignatureAlgorithm::Unknown,
    };
    let salt = match reader.read_asn1(CTX2_CONS) {
        Some(c) => c,
        None => return SignatureAlgorithm::Unknown,
    };
    let mut trailer: i64 = 1;
    if reader.peek_tag(CTX3_CONS) {
        let bytes = match reader.read_asn1(CTX3_CONS) {
            Some(c) => c,
            None => return SignatureAlgorithm::Unknown,
        };
        let mut inner = Reader::new(bytes);
        let raw = match inner.read_asn1(TAG_INTEGER) {
            Some(c) if inner.is_empty() => c,
            _ => return SignatureAlgorithm::Unknown,
        };
        trailer = match lenient_asn1_int(raw) {
            Some(v) => v,
            None => return SignatureAlgorithm::Unknown,
        };
    }
    if !reader.is_empty() {
        return SignatureAlgorithm::Unknown;
    }
    let hash_ai = match parse_inner_ai(hash) {
        Some(ai) => ai,
        None => return SignatureAlgorithm::Unknown,
    };
    let mgf_ai = match parse_inner_ai(mgf) {
        Some(ai) => ai,
        None => return SignatureAlgorithm::Unknown,
    };
    let mut salt_reader = Reader::new(salt);
    let salt_raw = match salt_reader.read_asn1(TAG_INTEGER) {
        Some(c) if salt_reader.is_empty() => c,
        _ => return SignatureAlgorithm::Unknown,
    };
    let salt_len = match lenient_asn1_int(salt_raw) {
        Some(v) => v,
        None => return SignatureAlgorithm::Unknown,
    };
    let mgf_hash = match parse_inner_ai(mgf_ai.params) {
        Some(ai) => ai,
        None => return SignatureAlgorithm::Unknown,
    };
    let null_or_absent = |params: &[u8]| params.is_empty() || params == NULL_BYTES;
    if !null_or_absent(hash_ai.params)
        || mgf_ai.oid != OID_MGF1
        || mgf_hash.oid != hash_ai.oid
        || !null_or_absent(mgf_hash.params)
        || trailer != 1
    {
        return SignatureAlgorithm::Unknown;
    }
    if hash_ai.oid == OID_SHA256 && salt_len == 32 {
        SignatureAlgorithm::Sha256Pss
    } else if hash_ai.oid == OID_SHA384 && salt_len == 48 {
        SignatureAlgorithm::Sha384Pss
    } else if hash_ai.oid == OID_SHA512 && salt_len == 64 {
        SignatureAlgorithm::Sha512Pss
    } else {
        SignatureAlgorithm::Unknown
    }
}

/// Go `getSignatureAlgorithmFromAI`. RSA/ECDSA OIDs map regardless of
/// parameters; Ed25519 requires absent parameters; PSS parses its bucket.
fn signature_algorithm_from_ai(ai: &AlgorithmIdentifier<'_>) -> SignatureAlgorithm {
    if ai.oid == OID_ED25519_SIG {
        if !ai.params.is_empty() {
            return SignatureAlgorithm::Unknown;
        }
        return SignatureAlgorithm::Ed25519;
    }
    if ai.oid == OID_RSAPSS {
        return pss_algorithm(ai.params);
    }
    if ai.oid == OID_MD5_RSA {
        SignatureAlgorithm::Md5Rsa
    } else if ai.oid == OID_SHA1_RSA || ai.oid == OID_ISO_SHA1_RSA {
        SignatureAlgorithm::Sha1Rsa
    } else if ai.oid == OID_SHA256_RSA {
        SignatureAlgorithm::Sha256Rsa
    } else if ai.oid == OID_SHA384_RSA {
        SignatureAlgorithm::Sha384Rsa
    } else if ai.oid == OID_SHA512_RSA {
        SignatureAlgorithm::Sha512Rsa
    } else if ai.oid == OID_DSA_SHA1 {
        SignatureAlgorithm::DsaSha1
    } else if ai.oid == OID_DSA_SHA256 {
        SignatureAlgorithm::DsaSha256
    } else if ai.oid == OID_ECDSA_SHA1 {
        SignatureAlgorithm::EcdsaSha1
    } else if ai.oid == OID_ECDSA_SHA256 {
        SignatureAlgorithm::EcdsaSha256
    } else if ai.oid == OID_ECDSA_SHA384 {
        SignatureAlgorithm::EcdsaSha384
    } else if ai.oid == OID_ECDSA_SHA512 {
        SignatureAlgorithm::EcdsaSha512
    } else {
        SignatureAlgorithm::Unknown
    }
}

fn public_key_algorithm_from_oid(oid: &[u8]) -> PublicKeyAlgorithm {
    if oid == OID_PUB_RSA {
        PublicKeyAlgorithm::Rsa
    } else if oid == OID_PUB_DSA {
        PublicKeyAlgorithm::Dsa
    } else if oid == OID_PUB_ECDSA {
        PublicKeyAlgorithm::Ecdsa
    } else if oid == OID_PUB_ED25519 {
        PublicKeyAlgorithm::Ed25519
    } else {
        PublicKeyAlgorithm::Unknown
    }
}

// ---------------------------------------------------------------------------
// Public keys.
// ---------------------------------------------------------------------------

/// Field primes, big-endian, for Go's coordinate-range error. Boundary
/// behavior is covered by differential tests against the Go oracle.
const P224_PRIME: [u8; 28] = [
    0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff,
    0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x01,
];
const P256_PRIME: [u8; 32] = [
    0xff, 0xff, 0xff, 0xff, 0x00, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
    0x00, 0x00, 0x00, 0x00, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff,
];
const P384_PRIME: [u8; 48] = [
    0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff,
    0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xfe, 0xff, 0xff, 0xff, 0xff,
    0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff,
];
const P521_PRIME: [u8; 66] = [
    0x01, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff,
    0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff,
    0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff,
    0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff,
    0xff, 0xff,
];

/// Go's P-256 coordinate error is platform dependent (assembly wording on
/// amd64/arm64/ppc64le/s390x, fiat wording elsewhere, `purego` excluded).
#[cfg(any(
    target_arch = "x86_64",
    target_arch = "aarch64",
    target_arch = "powerpc64",
    target_arch = "s390x"
))]
const P256_ELEMENT_ERROR: &str = "invalid P256 element encoding";
#[cfg(not(any(
    target_arch = "x86_64",
    target_arch = "aarch64",
    target_arch = "powerpc64",
    target_arch = "s390x"
)))]
const P256_ELEMENT_ERROR: &str = "invalid P256Element encoding";

fn magnitude(bytes: &[u8]) -> &[u8] {
    if bytes.len() > 1 && bytes[0] == 0 {
        &bytes[1..]
    } else {
        bytes
    }
}

fn is_negative(bytes: &[u8]) -> bool {
    !bytes.is_empty() && bytes[0] & 0x80 != 0
}

fn is_zero(bytes: &[u8]) -> bool {
    bytes.iter().all(|b| *b == 0)
}

fn parse_rsa_key(params: &[u8], data: &[u8]) -> Result<PublicKeyData, String> {
    if params != NULL_BYTES {
        return Err(String::from("x509: RSA key missing NULL parameters"));
    }
    let mut reader = Reader::new(data);
    let contents = reader
        .read_asn1(TAG_SEQ)
        .ok_or_else(|| String::from("x509: invalid RSA public key"))?;
    let mut reader = Reader::new(contents);
    let n = reader
        .read_integer_bytes()
        .ok_or_else(|| String::from("x509: invalid RSA modulus"))?;
    let e = reader
        .read_integer_bytes()
        .ok_or_else(|| String::from("x509: invalid RSA public exponent"))?;
    let e = asn1_signed(e).ok_or_else(|| String::from("x509: invalid RSA public exponent"))?;
    if is_negative(n) || is_zero(n) {
        return Err(String::from("x509: RSA modulus is not a positive number"));
    }
    if e <= 0 {
        return Err(String::from(
            "x509: RSA public exponent is not a positive number",
        ));
    }
    let key = rsa::RsaPublicKey::new_unchecked(
        rsa::BigUint::from_bytes_be(magnitude(n)),
        rsa::BigUint::from(e as u64),
    );
    Ok(PublicKeyData::Rsa(key))
}

fn parse_dsa_key(params: &[u8], data: &[u8]) -> Result<PublicKeyData, String> {
    let mut reader = Reader::new(data);
    let y = reader
        .read_integer_bytes()
        .ok_or_else(|| String::from("x509: invalid DSA public key"))?;
    let mut params = Reader::new(params);
    let contents = params
        .read_asn1(TAG_SEQ)
        .ok_or_else(|| String::from("x509: invalid DSA parameters"))?;
    let mut params = Reader::new(contents);
    let p = params
        .read_integer_bytes()
        .ok_or_else(|| String::from("x509: invalid DSA parameters"))?;
    let q = params
        .read_integer_bytes()
        .ok_or_else(|| String::from("x509: invalid DSA parameters"))?;
    let g = params
        .read_integer_bytes()
        .ok_or_else(|| String::from("x509: invalid DSA parameters"))?;
    for value in [y, p, q, g] {
        if is_negative(value) || is_zero(value) {
            return Err(String::from("x509: zero or negative DSA parameter"));
        }
    }
    Ok(PublicKeyData::Dsa {
        y: magnitude(y).to_vec(),
        p: magnitude(p).to_vec(),
        q: magnitude(q).to_vec(),
        g: magnitude(g).to_vec(),
    })
}

/// Validated curve point length/prefix/range shared by all four curves.
/// Returns the coordinate field length on success.
fn check_ec_point(
    data: &[u8],
    prime: &[u8],
    point_error: &str,
    element_error: &str,
) -> Result<usize, String> {
    if data.is_empty() || data[0] != 4 {
        return Err(String::from("ecdsa: invalid uncompressed public key"));
    }
    let field = prime.len();
    if data.len() != 1 + 2 * field {
        return Err(String::from(point_error));
    }
    let (x, y) = (&data[1..1 + field], &data[1 + field..]);
    if x >= prime || y >= prime {
        return Err(String::from(element_error));
    }
    Ok(field)
}

fn parse_ecdsa_key(params: &[u8], data: &[u8]) -> Result<PublicKeyData, String> {
    let mut reader = Reader::new(params);
    let curve = reader
        .read_oid()
        .ok_or_else(|| String::from("x509: invalid ECDSA parameters"))?;
    if curve == OID_CURVE_P224 {
        check_ec_point(
            data,
            &P224_PRIME,
            "invalid P224 point encoding",
            "invalid P224Element encoding",
        )?;
        let key = ecdsa::VerifyingKey::<p224::NistP224>::from_sec1_bytes(data)
            .map_err(|_| String::from("P224 point not on curve"))?;
        Ok(PublicKeyData::Ecdsa224(key))
    } else if curve == OID_CURVE_P256 {
        check_ec_point(
            data,
            &P256_PRIME,
            "invalid P256 point encoding",
            P256_ELEMENT_ERROR,
        )?;
        let key = ecdsa::VerifyingKey::<p256::NistP256>::from_sec1_bytes(data)
            .map_err(|_| String::from("P256 point not on curve"))?;
        Ok(PublicKeyData::Ecdsa256(key))
    } else if curve == OID_CURVE_P384 {
        check_ec_point(
            data,
            &P384_PRIME,
            "invalid P384 point encoding",
            "invalid P384Element encoding",
        )?;
        let key = ecdsa::VerifyingKey::<p384::NistP384>::from_sec1_bytes(data)
            .map_err(|_| String::from("P384 point not on curve"))?;
        Ok(PublicKeyData::Ecdsa384(key))
    } else if curve == OID_CURVE_P521 {
        check_ec_point(
            data,
            &P521_PRIME,
            "invalid P521 point encoding",
            "invalid P521Element encoding",
        )?;
        let key = ecdsa::VerifyingKey::<p521::NistP521>::from_sec1_bytes(data)
            .map_err(|_| String::from("P521 point not on curve"))?;
        Ok(PublicKeyData::Ecdsa521(key))
    } else {
        Err(String::from("x509: unsupported elliptic curve"))
    }
}

fn parse_ed25519_key(params: &[u8], data: &[u8]) -> Result<PublicKeyData, String> {
    if !params.is_empty() {
        return Err(String::from(
            "x509: Ed25519 key encoded with illegal parameters",
        ));
    }
    if data.len() != 32 {
        return Err(String::from("x509: wrong Ed25519 public key size"));
    }
    let mut key = [0u8; 32];
    key.copy_from_slice(data);
    Ok(PublicKeyData::Ed25519(key))
}

fn parse_public_key(
    algorithm: PublicKeyAlgorithm,
    params: &[u8],
    data: &[u8],
) -> Result<PublicKeyData, String> {
    match algorithm {
        PublicKeyAlgorithm::Rsa => parse_rsa_key(params, data),
        PublicKeyAlgorithm::Dsa => parse_dsa_key(params, data),
        PublicKeyAlgorithm::Ecdsa => parse_ecdsa_key(params, data),
        PublicKeyAlgorithm::Ed25519 => parse_ed25519_key(params, data),
        PublicKeyAlgorithm::Unknown => Err(String::from("x509: unknown public key algorithm")),
    }
}

// ---------------------------------------------------------------------------
// ParseCertificate.
// ---------------------------------------------------------------------------

/// Go `x509.ParseCertificate`: structural parse with Go's exact errors.
/// Trailing bytes at every level are ignored, as in Go.
pub fn parse_certificate(der: &[u8]) -> Result<Certificate, String> {
    let mut input = Reader::new(der);
    let raw = input
        .read_element(TAG_SEQ)
        .ok_or_else(|| String::from("x509: malformed certificate"))?;
    let mut input = Reader::new(raw);
    let contents = input
        .read_asn1(TAG_SEQ)
        .ok_or_else(|| String::from("x509: malformed certificate"))?;
    let mut input = Reader::new(contents);

    let tbs_element = input
        .read_element(TAG_SEQ)
        .ok_or_else(|| String::from("x509: malformed tbs certificate"))?;
    let mut tbs_reader = Reader::new(tbs_element);
    let tbs_contents = tbs_reader
        .read_asn1(TAG_SEQ)
        .ok_or_else(|| String::from("x509: malformed tbs certificate"))?;
    let mut tbs = Reader::new(tbs_contents);

    let version_contents = tbs
        .read_optional(CTX0_CONS)
        .ok_or_else(|| String::from("x509: malformed version"))?;
    let mut version = 0i64;
    if let Some(contents) = version_contents {
        let mut version_reader = Reader::new(contents);
        version = version_reader
            .read_int64()
            .filter(|_| version_reader.is_empty())
            .ok_or_else(|| String::from("x509: malformed version"))?;
    }
    if version < 0 {
        return Err(String::from("x509: malformed version"));
    }
    version += 1;
    if version > 3 {
        return Err(String::from("x509: invalid version"));
    }

    let serial = tbs
        .read_integer_bytes()
        .ok_or_else(|| String::from("x509: malformed serial number"))?;
    if is_negative(serial) {
        return Err(String::from("x509: negative serial number"));
    }

    let inner_ai = tbs
        .read_asn1(TAG_SEQ)
        .ok_or_else(|| String::from("x509: malformed signature algorithm identifier"))?;
    let outer_ai = input
        .read_asn1(TAG_SEQ)
        .ok_or_else(|| String::from("x509: malformed algorithm identifier"))?;
    if inner_ai != outer_ai {
        return Err(String::from(
            "x509: inner and outer signature algorithm identifiers don't match",
        ));
    }
    let sig_ai = parse_ai(inner_ai)?;
    let signature_algorithm = signature_algorithm_from_ai(&sig_ai);

    let issuer = tbs
        .read_element(TAG_SEQ)
        .ok_or_else(|| String::from("x509: malformed issuer"))?;
    parse_name(issuer)?;

    let validity = tbs
        .read_asn1(TAG_SEQ)
        .ok_or_else(|| String::from("x509: malformed validity"))?;
    let (not_before, not_after) = parse_validity(validity)?;

    // Go reports the subject with the issuer's error text; keep the quirk.
    let subject = tbs
        .read_element(TAG_SEQ)
        .ok_or_else(|| String::from("x509: malformed issuer"))?;
    parse_name(subject)?;

    let spki = tbs
        .read_element(TAG_SEQ)
        .ok_or_else(|| String::from("x509: malformed spki"))?;
    let mut spki_reader = Reader::new(spki);
    let spki_contents = spki_reader
        .read_asn1(TAG_SEQ)
        .ok_or_else(|| String::from("x509: malformed spki"))?;
    let mut spki_reader = Reader::new(spki_contents);
    let pk_ai_contents = spki_reader
        .read_asn1(TAG_SEQ)
        .ok_or_else(|| String::from("x509: malformed public key algorithm identifier"))?;
    let pk_ai = parse_ai(pk_ai_contents)?;
    let public_key_algorithm = public_key_algorithm_from_oid(pk_ai.oid);
    let (spk_bytes, spk_bits) = spki_reader
        .read_bitstring()
        .ok_or_else(|| String::from("x509: malformed subjectPublicKey"))?;
    let key_data = right_align(spk_bytes, spk_bits);
    let public_key = if public_key_algorithm != PublicKeyAlgorithm::Unknown {
        Some(parse_public_key(
            public_key_algorithm,
            pk_ai.params,
            &key_data,
        )?)
    } else {
        None
    };

    let mut extensions = ProcessedExtensions {
        key_usage: 0,
        is_ca: false,
        basic_constraints_valid: false,
        max_path_len: 0,
        max_path_len_zero: false,
        unhandled_critical: Vec::new(),
    };
    if version > 1 {
        if !tbs.skip_optional(CTX1_PRIM) {
            return Err(String::from("x509: malformed issuerUniqueID"));
        }
        if !tbs.skip_optional(CTX2_PRIM) {
            return Err(String::from("x509: malformed subjectUniqueID"));
        }
        if version == 3 {
            let present = tbs
                .read_optional(CTX3_CONS)
                .ok_or_else(|| String::from("x509: malformed extensions"))?;
            if let Some(contents) = present {
                let mut ext_reader = Reader::new(contents);
                let seq = ext_reader
                    .read_asn1(TAG_SEQ)
                    .ok_or_else(|| String::from("x509: malformed extensions"))?;
                let mut ext_reader = Reader::new(seq);
                let mut parsed = Vec::new();
                let mut seen: Vec<&[u8]> = Vec::new();
                while !ext_reader.is_empty() {
                    let ext_contents = ext_reader
                        .read_asn1(TAG_SEQ)
                        .ok_or_else(|| String::from("x509: malformed extension"))?;
                    let ext = parse_extension(ext_contents)?;
                    if seen.contains(&ext.id) {
                        // Go quotes the dotted OID string, not the raw bytes.
                        let dotted = oid_to_string(ext.id);
                        return Err(format!(
                            "x509: certificate contains duplicate extension with OID {}",
                            quoted(dotted.as_bytes())
                        ));
                    }
                    seen.push(ext.id);
                    parsed.push(ext);
                }
                extensions = process_extensions(&parsed)?;
            }
        }
    }

    let (sig_bytes, sig_bits) = input
        .read_bitstring()
        .ok_or_else(|| String::from("x509: malformed signature"))?;

    Ok(Certificate {
        raw: raw.to_vec(),
        raw_tbs: tbs_element.to_vec(),
        raw_subject_public_key_info: spki.to_vec(),
        raw_subject: subject.to_vec(),
        raw_issuer: issuer.to_vec(),
        version: version as u8,
        serial: magnitude(serial).to_vec(),
        signature_algorithm,
        public_key_algorithm,
        public_key,
        signature: right_align(sig_bytes, sig_bits),
        is_ca: extensions.is_ca,
        basic_constraints_valid: extensions.basic_constraints_valid,
        max_path_len: extensions.max_path_len,
        max_path_len_zero: extensions.max_path_len_zero,
        key_usage: extensions.key_usage,
        not_before,
        not_after,
        unhandled_critical: extensions.unhandled_critical,
    })
}

// ---------------------------------------------------------------------------
// CheckSignatureFrom.
// ---------------------------------------------------------------------------

const ERR_UNSUPPORTED_ALGORITHM: &str = "x509: cannot verify signature: algorithm unimplemented";
const ERR_CONSTRAINT_VIOLATION: &str =
    "x509: invalid signature: parent certificate cannot sign this kind of certificate";
const ERR_RSA_VERIFICATION: &str = "crypto/rsa: verification error";
const ERR_ECDSA_FAILURE: &str = "x509: ECDSA verification failure";
const ERR_ED25519_FAILURE: &str = "x509: Ed25519 verification failure";

fn insecure_algorithm_error(algo: SignatureAlgorithm) -> String {
    format!(
        "x509: cannot verify signature: insecure algorithm {}",
        algo.name()
    )
}

fn mismatch_error(expected: PublicKeyAlgorithm, got: &str) -> String {
    format!(
        "x509: signature algorithm specifies an {} public key, but have public key of type {}",
        expected.name(),
        got
    )
}

#[derive(Clone, Copy, PartialEq, Eq)]
enum Hash {
    Md5,
    Sha1,
    Sha256,
    Sha384,
    Sha512,
    None,
}

/// Go's `signatureAlgorithmDetails` row for verification: hash, expected
/// public key algorithm, and the RSA-PSS flag.
fn algorithm_details(algo: SignatureAlgorithm) -> (Hash, PublicKeyAlgorithm, bool) {
    match algo {
        SignatureAlgorithm::Md5Rsa => (Hash::Md5, PublicKeyAlgorithm::Rsa, false),
        SignatureAlgorithm::Sha1Rsa => (Hash::Sha1, PublicKeyAlgorithm::Rsa, false),
        SignatureAlgorithm::Sha256Rsa => (Hash::Sha256, PublicKeyAlgorithm::Rsa, false),
        SignatureAlgorithm::Sha384Rsa => (Hash::Sha384, PublicKeyAlgorithm::Rsa, false),
        SignatureAlgorithm::Sha512Rsa => (Hash::Sha512, PublicKeyAlgorithm::Rsa, false),
        SignatureAlgorithm::Sha256Pss => (Hash::Sha256, PublicKeyAlgorithm::Rsa, true),
        SignatureAlgorithm::Sha384Pss => (Hash::Sha384, PublicKeyAlgorithm::Rsa, true),
        SignatureAlgorithm::Sha512Pss => (Hash::Sha512, PublicKeyAlgorithm::Rsa, true),
        SignatureAlgorithm::DsaSha1 => (Hash::Sha1, PublicKeyAlgorithm::Dsa, false),
        SignatureAlgorithm::DsaSha256 => (Hash::Sha256, PublicKeyAlgorithm::Dsa, false),
        SignatureAlgorithm::EcdsaSha1 => (Hash::Sha1, PublicKeyAlgorithm::Ecdsa, false),
        SignatureAlgorithm::EcdsaSha256 => (Hash::Sha256, PublicKeyAlgorithm::Ecdsa, false),
        SignatureAlgorithm::EcdsaSha384 => (Hash::Sha384, PublicKeyAlgorithm::Ecdsa, false),
        SignatureAlgorithm::EcdsaSha512 => (Hash::Sha512, PublicKeyAlgorithm::Ecdsa, false),
        SignatureAlgorithm::Ed25519 => (Hash::None, PublicKeyAlgorithm::Ed25519, false),
        SignatureAlgorithm::Unknown => (Hash::None, PublicKeyAlgorithm::Unknown, false),
    }
}

fn digest(hash: Hash, signed: &[u8]) -> Vec<u8> {
    use sha2::Digest as _;
    match hash {
        Hash::Sha256 => sha2::Sha256::digest(signed).to_vec(),
        Hash::Sha384 => sha2::Sha384::digest(signed).to_vec(),
        Hash::Sha512 => sha2::Sha512::digest(signed).to_vec(),
        _ => signed.to_vec(),
    }
}

/// Go `ecdsa.VerifyASN1` lenient signature parse: `SEQUENCE { r, s }` with
/// `encoding/asn1` INTEGER semantics (non-minimal accepted, two's
/// complement, empty rejected, exactly two elements). Returns big-endian
/// magnitudes, or `None` for anything Go rejects (including non-positive
/// values, which fail Go's range check downstream).
fn parse_ecdsa_signature(sig: &[u8]) -> Option<(Vec<u8>, Vec<u8>)> {
    let mut outer = Reader::new(sig);
    let contents = outer.read_asn1(TAG_SEQ)?;
    if !outer.is_empty() {
        return None;
    }
    let mut reader = Reader::new(contents);
    let r = reader.read_asn1(TAG_INTEGER)?;
    let s = reader.read_asn1(TAG_INTEGER)?;
    if !reader.is_empty() || r.is_empty() || s.is_empty() {
        return None;
    }
    let to_magnitude = |raw: &[u8]| -> Option<Vec<u8>> {
        if raw[0] & 0x80 != 0 {
            // Negative: Go's range check rejects it downstream.
            return None;
        }
        let mut start = 0;
        while start + 1 < raw.len() && raw[start] == 0 {
            start += 1;
        }
        let mag = &raw[start..];
        if mag.iter().all(|b| *b == 0) {
            return None;
        }
        Some(mag.to_vec())
    };
    Some((to_magnitude(r)?, to_magnitude(s)?))
}

/// Left-pad a magnitude to the field length; `None` when it cannot fit
/// (at least `2^fieldlen`, hence `>= n`, which Go rejects).
fn pad_scalar(mag: &[u8], field: usize) -> Option<Vec<u8>> {
    if mag.len() > field {
        return None;
    }
    let mut out = vec![0u8; field];
    out[field - mag.len()..].copy_from_slice(mag);
    Some(out)
}

fn verify_rsa(
    key: &rsa::RsaPublicKey,
    expected: PublicKeyAlgorithm,
    hash: Hash,
    pss: bool,
    hashed: &[u8],
    signature: &[u8],
) -> Result<(), String> {
    if expected != PublicKeyAlgorithm::Rsa {
        return Err(mismatch_error(expected, "*rsa.PublicKey"));
    }
    let ok = match (hash, pss) {
        (Hash::Sha256, false) => {
            key.verify(rsa::Pkcs1v15Sign::new::<sha2::Sha256>(), hashed, signature)
        }
        (Hash::Sha384, false) => {
            key.verify(rsa::Pkcs1v15Sign::new::<sha2::Sha384>(), hashed, signature)
        }
        (Hash::Sha512, false) => {
            key.verify(rsa::Pkcs1v15Sign::new::<sha2::Sha512>(), hashed, signature)
        }
        // Go verifies PSS with salt length equal to the hash length.
        (Hash::Sha256, true) => key.verify(rsa::pss::Pss::new::<sha2::Sha256>(), hashed, signature),
        (Hash::Sha384, true) => key.verify(rsa::pss::Pss::new::<sha2::Sha384>(), hashed, signature),
        (Hash::Sha512, true) => key.verify(rsa::pss::Pss::new::<sha2::Sha512>(), hashed, signature),
        _ => return Err(String::from(ERR_UNSUPPORTED_ALGORITHM)),
    };
    ok.map_err(|_| String::from(ERR_RSA_VERIFICATION))
}

fn verify_ecdsa_256(
    key: &ecdsa::VerifyingKey<p256::NistP256>,
    expected: PublicKeyAlgorithm,
    hashed: &[u8],
    signature: &[u8],
) -> Result<(), String> {
    use signature::hazmat::PrehashVerifier as _;
    if expected != PublicKeyAlgorithm::Ecdsa {
        return Err(mismatch_error(expected, "*ecdsa.PublicKey"));
    }
    let (r, s) = parse_ecdsa_signature(signature).ok_or_else(|| String::from(ERR_ECDSA_FAILURE))?;
    let mut fixed = Vec::with_capacity(64);
    fixed.extend_from_slice(&pad_scalar(&r, 32).ok_or_else(|| String::from(ERR_ECDSA_FAILURE))?);
    fixed.extend_from_slice(&pad_scalar(&s, 32).ok_or_else(|| String::from(ERR_ECDSA_FAILURE))?);
    let sig = ecdsa::Signature::<p256::NistP256>::from_slice(&fixed)
        .map_err(|_| String::from(ERR_ECDSA_FAILURE))?;
    key.verify_prehash(hashed, &sig)
        .map_err(|_| String::from(ERR_ECDSA_FAILURE))
}

fn verify_ecdsa_384(
    key: &ecdsa::VerifyingKey<p384::NistP384>,
    expected: PublicKeyAlgorithm,
    hashed: &[u8],
    signature: &[u8],
) -> Result<(), String> {
    use signature::hazmat::PrehashVerifier as _;
    if expected != PublicKeyAlgorithm::Ecdsa {
        return Err(mismatch_error(expected, "*ecdsa.PublicKey"));
    }
    let (r, s) = parse_ecdsa_signature(signature).ok_or_else(|| String::from(ERR_ECDSA_FAILURE))?;
    let mut fixed = Vec::with_capacity(96);
    fixed.extend_from_slice(&pad_scalar(&r, 48).ok_or_else(|| String::from(ERR_ECDSA_FAILURE))?);
    fixed.extend_from_slice(&pad_scalar(&s, 48).ok_or_else(|| String::from(ERR_ECDSA_FAILURE))?);
    let sig = ecdsa::Signature::<p384::NistP384>::from_slice(&fixed)
        .map_err(|_| String::from(ERR_ECDSA_FAILURE))?;
    key.verify_prehash(hashed, &sig)
        .map_err(|_| String::from(ERR_ECDSA_FAILURE))
}

fn verify_ecdsa_521(
    key: &ecdsa::VerifyingKey<p521::NistP521>,
    expected: PublicKeyAlgorithm,
    hashed: &[u8],
    signature: &[u8],
) -> Result<(), String> {
    use signature::hazmat::PrehashVerifier as _;
    if expected != PublicKeyAlgorithm::Ecdsa {
        return Err(mismatch_error(expected, "*ecdsa.PublicKey"));
    }
    let (r, s) = parse_ecdsa_signature(signature).ok_or_else(|| String::from(ERR_ECDSA_FAILURE))?;
    let mut fixed = Vec::with_capacity(132);
    fixed.extend_from_slice(&pad_scalar(&r, 66).ok_or_else(|| String::from(ERR_ECDSA_FAILURE))?);
    fixed.extend_from_slice(&pad_scalar(&s, 66).ok_or_else(|| String::from(ERR_ECDSA_FAILURE))?);
    let sig = ecdsa::Signature::<p521::NistP521>::from_slice(&fixed)
        .map_err(|_| String::from(ERR_ECDSA_FAILURE))?;
    key.verify_prehash(hashed, &sig)
        .map_err(|_| String::from(ERR_ECDSA_FAILURE))
}

fn verify_ecdsa_224(
    key: &ecdsa::VerifyingKey<p224::NistP224>,
    expected: PublicKeyAlgorithm,
    hashed: &[u8],
    signature: &[u8],
) -> Result<(), String> {
    use signature::hazmat::PrehashVerifier as _;
    if expected != PublicKeyAlgorithm::Ecdsa {
        return Err(mismatch_error(expected, "*ecdsa.PublicKey"));
    }
    let (r, s) = parse_ecdsa_signature(signature).ok_or_else(|| String::from(ERR_ECDSA_FAILURE))?;
    let mut fixed = Vec::with_capacity(56);
    fixed.extend_from_slice(&pad_scalar(&r, 28).ok_or_else(|| String::from(ERR_ECDSA_FAILURE))?);
    fixed.extend_from_slice(&pad_scalar(&s, 28).ok_or_else(|| String::from(ERR_ECDSA_FAILURE))?);
    let sig = ecdsa::Signature::<p224::NistP224>::from_slice(&fixed)
        .map_err(|_| String::from(ERR_ECDSA_FAILURE))?;
    key.verify_prehash(hashed, &sig)
        .map_err(|_| String::from(ERR_ECDSA_FAILURE))
}

fn verify_ed25519(
    key: &[u8; 32],
    expected: PublicKeyAlgorithm,
    signed: &[u8],
    signature: &[u8],
) -> Result<(), String> {
    use ed25519_dalek::Verifier as _;
    if expected != PublicKeyAlgorithm::Ed25519 {
        return Err(mismatch_error(expected, "ed25519.PublicKey"));
    }
    let vk = ed25519_dalek::VerifyingKey::from_bytes(key)
        .map_err(|_| String::from(ERR_ED25519_FAILURE))?;
    let sig = ed25519_dalek::Signature::try_from(signature)
        .map_err(|_| String::from(ERR_ED25519_FAILURE))?;
    vk.verify(signed, &sig)
        .map_err(|_| String::from(ERR_ED25519_FAILURE))
}

/// Go `(*Certificate).CheckSignatureFrom`: parent constraint checks, then
/// `checkSignature` with SHA-1 disallowed.
pub fn check_signature_from(child: &Certificate, parent: &Certificate) -> Result<(), String> {
    if parent.version == 3 && !parent.basic_constraints_valid
        || parent.basic_constraints_valid && !parent.is_ca
    {
        return Err(String::from(ERR_CONSTRAINT_VIOLATION));
    }
    if parent.key_usage != 0 && parent.key_usage & KEY_USAGE_CERT_SIGN == 0 {
        return Err(String::from(ERR_CONSTRAINT_VIOLATION));
    }
    if parent.public_key_algorithm == PublicKeyAlgorithm::Unknown {
        return Err(String::from(ERR_UNSUPPORTED_ALGORITHM));
    }
    let (hash, expected, pss) = algorithm_details(child.signature_algorithm);
    match hash {
        Hash::Md5 => return Err(insecure_algorithm_error(child.signature_algorithm)),
        Hash::Sha1 => return Err(insecure_algorithm_error(child.signature_algorithm)),
        Hash::None if expected != PublicKeyAlgorithm::Ed25519 => {
            return Err(String::from(ERR_UNSUPPORTED_ALGORITHM))
        }
        _ => {}
    }
    let hashed;
    let signed: &[u8] = if hash == Hash::None {
        &child.raw_tbs
    } else {
        hashed = digest(hash, &child.raw_tbs);
        &hashed
    };
    let key = match parent.public_key.as_ref() {
        Some(key) => key,
        None => return Err(String::from(ERR_UNSUPPORTED_ALGORITHM)),
    };
    match key {
        PublicKeyData::Rsa(key) => verify_rsa(key, expected, hash, pss, signed, &child.signature),
        PublicKeyData::Ecdsa224(key) => verify_ecdsa_224(key, expected, signed, &child.signature),
        PublicKeyData::Ecdsa256(key) => verify_ecdsa_256(key, expected, signed, &child.signature),
        PublicKeyData::Ecdsa384(key) => verify_ecdsa_384(key, expected, signed, &child.signature),
        PublicKeyData::Ecdsa521(key) => verify_ecdsa_521(key, expected, signed, &child.signature),
        PublicKeyData::Ed25519(key) => verify_ed25519(key, expected, signed, &child.signature),
        // Go's key-type switch has no DSA arm: DSA keys always fail here
        // (after the hash gate above).
        PublicKeyData::Dsa { .. } => Err(String::from(ERR_UNSUPPORTED_ALGORITHM)),
    }
}

mod algorithms;
mod der;
mod time;
mod types;

#[cfg(test)]
mod tests {
    use super::*;

    fn tlv(tag: u8, contents: &[u8]) -> Vec<u8> {
        let mut out = vec![tag];
        if contents.len() < 128 {
            out.push(contents.len() as u8);
        } else {
            let mut len = contents.len();
            let mut bytes = Vec::new();
            while len > 0 {
                bytes.push((len & 0xff) as u8);
                len >>= 8;
            }
            out.push(0x80 | bytes.len() as u8);
            bytes.reverse();
            out.extend_from_slice(&bytes);
        }
        out.extend_from_slice(contents);
        out
    }

    fn seq(contents: &[u8]) -> Vec<u8> {
        tlv(0x30, contents)
    }

    fn concat(parts: &[Vec<u8>]) -> Vec<u8> {
        let mut out = Vec::new();
        for part in parts {
            out.extend_from_slice(part);
        }
        out
    }

    fn int_raw(bytes: &[u8]) -> Vec<u8> {
        tlv(0x02, bytes)
    }

    fn int_small(v: u64) -> Vec<u8> {
        if v == 0 {
            return tlv(0x02, &[0]);
        }
        let mut bytes = Vec::new();
        let mut x = v;
        while x > 0 {
            bytes.push((x & 0xff) as u8);
            x >>= 8;
        }
        bytes.reverse();
        if bytes[0] & 0x80 != 0 {
            bytes.insert(0, 0);
        }
        tlv(0x02, &bytes)
    }

    fn oid(body: &[u8]) -> Vec<u8> {
        tlv(0x06, body)
    }

    fn null() -> Vec<u8> {
        vec![0x05, 0x00]
    }

    fn bitstring(payload: &[u8]) -> Vec<u8> {
        let mut contents = vec![0x00];
        contents.extend_from_slice(payload);
        tlv(0x03, &contents)
    }

    fn utctime(s: &str) -> Vec<u8> {
        tlv(0x17, s.as_bytes())
    }

    fn gentime(s: &str) -> Vec<u8> {
        tlv(0x18, s.as_bytes())
    }

    fn octet(contents: &[u8]) -> Vec<u8> {
        tlv(0x04, contents)
    }

    fn boolean(v: bool) -> Vec<u8> {
        tlv(0x01, &[if v { 0xff } else { 0 }])
    }

    const OID_SHA256_RSA: &[u8] = &[0x2A, 0x86, 0x48, 0x86, 0xF7, 0x0D, 0x01, 0x01, 0x0B];
    const OID_SHA1_RSA: &[u8] = &[0x2A, 0x86, 0x48, 0x86, 0xF7, 0x0D, 0x01, 0x01, 0x05];
    const OID_MD5_RSA: &[u8] = &[0x2A, 0x86, 0x48, 0x86, 0xF7, 0x0D, 0x01, 0x01, 0x04];
    const OID_RSA_ENC: &[u8] = &[0x2A, 0x86, 0x48, 0x86, 0xF7, 0x0D, 0x01, 0x01, 0x01];
    const OID_PSS: &[u8] = &[0x2A, 0x86, 0x48, 0x86, 0xF7, 0x0D, 0x01, 0x01, 0x0A];
    const OID_MGF1: &[u8] = &[0x2A, 0x86, 0x48, 0x86, 0xF7, 0x0D, 0x01, 0x01, 0x08];
    const OID_SHA256: &[u8] = &[0x60, 0x86, 0x48, 0x01, 0x65, 0x03, 0x04, 0x02, 0x01];
    const OID_SHA384: &[u8] = &[0x60, 0x86, 0x48, 0x01, 0x65, 0x03, 0x04, 0x02, 0x02];
    const OID_SHA512: &[u8] = &[0x60, 0x86, 0x48, 0x01, 0x65, 0x03, 0x04, 0x02, 0x03];
    const OID_ECDSA_SHA256: &[u8] = &[0x2A, 0x86, 0x48, 0xCE, 0x3D, 0x04, 0x03, 0x02];
    const OID_ECDSA_SHA1: &[u8] = &[0x2A, 0x86, 0x48, 0xCE, 0x3D, 0x04, 0x01];
    const OID_EC_PUB: &[u8] = &[0x2A, 0x86, 0x48, 0xCE, 0x3D, 0x02, 0x01];
    const OID_P256: &[u8] = &[0x2A, 0x86, 0x48, 0xCE, 0x3D, 0x03, 0x01, 0x07];
    const OID_ED25519: &[u8] = &[0x2B, 0x65, 0x70];
    const OID_DSA: &[u8] = &[0x2A, 0x86, 0x48, 0xCE, 0x38, 0x04, 0x01];
    const OID_DSA_SHA256: &[u8] = &[0x60, 0x86, 0x48, 0x01, 0x65, 0x03, 0x04, 0x03, 0x02];
    const OID_ISO_SHA1_RSA: &[u8] = &[0x2B, 0x0E, 0x03, 0x02, 0x1D];
    const OID_KEY_USAGE: &[u8] = &[0x55, 0x1D, 0x0F];
    const OID_BASIC_CONSTRAINTS: &[u8] = &[0x55, 0x1D, 0x13];

    /// Full SEQ element for an AlgorithmIdentifier with optional params.
    fn ai_element(oid_body: &[u8], params: Option<&[u8]>) -> Vec<u8> {
        let mut contents = oid(oid_body);
        if let Some(p) = params {
            contents.extend_from_slice(p);
        }
        seq(&contents)
    }

    fn rsa_spki(n: &[u8], e: &[u8], params: Option<&[u8]>) -> Vec<u8> {
        let pk_alg = ai_element(OID_RSA_ENC, params);
        let key = seq(&concat(&[int_raw(n), int_raw(e)]));
        seq(&concat(&[pk_alg, bitstring(&key)]))
    }

    fn std_spki() -> Vec<u8> {
        rsa_spki(
            &[0x00, 0xC0, 0xFF, 0xEE],
            &[0x01, 0x00, 0x01],
            Some(&null()),
        )
    }

    fn std_validity() -> Vec<u8> {
        seq(&concat(&[
            utctime("700101000000Z"),
            utctime("700102000000Z"),
        ]))
    }

    /// The six standard TBS parts: serial, inner AI, issuer, validity,
    /// subject, SPKI. Callers may prefix a version or suffix unique IDs.
    fn std_parts() -> Vec<Vec<u8>> {
        vec![
            int_small(1),
            ai_element(OID_SHA256_RSA, Some(&null())),
            seq(&[]),
            std_validity(),
            seq(&[]),
            std_spki(),
        ]
    }

    fn cert_from_tbs_parts(parts: &[Vec<u8>], outer_ai: &[u8], sig: &[u8]) -> Vec<u8> {
        let tbs = seq(&concat(parts));
        seq(&concat(&[tbs, outer_ai.to_vec(), bitstring(sig)]))
    }

    fn std_cert() -> Vec<u8> {
        cert_from_tbs_parts(
            &std_parts(),
            &ai_element(OID_SHA256_RSA, Some(&null())),
            &[0xAB, 0xCD],
        )
    }

    fn err_text(result: Result<Certificate, String>) -> String {
        result.expect_err("expected parse failure")
    }

    #[test]
    fn parse_minimal_v1_rsa_cert() {
        let bytes = std_cert();
        let cert = parse_certificate(&bytes).expect("parse std cert");
        assert_eq!(cert.version, 1);
        assert_eq!(cert.serial, vec![0x01]);
        assert_eq!(cert.signature_algorithm, SignatureAlgorithm::Sha256Rsa);
        assert_eq!(cert.public_key_algorithm, PublicKeyAlgorithm::Rsa);
        assert!(matches!(cert.public_key, Some(PublicKeyData::Rsa(_))));
        assert_eq!(cert.signature, vec![0xAB, 0xCD]);
        assert_eq!(cert.key_usage, 0);
        assert!(!cert.is_ca);
        assert!(!cert.basic_constraints_valid);
        assert_eq!(cert.max_path_len, 0);
        assert!(!cert.max_path_len_zero);
        assert_eq!(cert.not_before, 0);
        assert_eq!(cert.not_after, 86400);
        assert!(cert.unhandled_critical.is_empty());
        assert_eq!(cert.raw, bytes);
        assert_eq!(cert.raw_tbs, seq(&concat(&std_parts())));
        assert_eq!(cert.raw_subject, vec![0x30, 0x00]);
        assert_eq!(cert.raw_issuer, vec![0x30, 0x00]);
        assert_eq!(cert.raw_subject_public_key_info, std_spki());
    }

    #[test]
    fn parse_ignores_trailing_bytes() {
        let mut bytes = std_cert();
        bytes.extend_from_slice(&[0x00, 0xFF, 0x04]);
        let cert = parse_certificate(&bytes).expect("trailing ignored");
        assert_eq!(cert.version, 1);
    }

    #[test]
    fn parse_structural_error_texts() {
        // Empty and wrong outer tag.
        assert_eq!(
            err_text(parse_certificate(&[])),
            "x509: malformed certificate"
        );
        assert_eq!(
            err_text(parse_certificate(&[0x31, 0x00])),
            "x509: malformed certificate"
        );
        // Truncated outer length.
        assert_eq!(
            err_text(parse_certificate(&[0x30, 0x05, 0x30, 0x00])),
            "x509: malformed certificate"
        );
        // TBS not a SEQUENCE.
        let bad_tbs = seq(&concat(&[int_small(1), int_small(2), bitstring(&[1])]));
        assert_eq!(
            err_text(parse_certificate(&bad_tbs)),
            "x509: malformed tbs certificate"
        );

        let outer = ai_element(OID_SHA256_RSA, Some(&null()));
        let mutant = |parts: &[Vec<u8>]| {
            err_text(parse_certificate(&cert_from_tbs_parts(parts, &outer, &[1])))
        };

        // Version element with non-INTEGER contents.
        let mut parts = std_parts();
        parts.insert(0, tlv(0xA0, &[0x04, 0x01, 0x00]));
        assert_eq!(mutant(&parts), "x509: malformed version");
        // Version 4 (field value 3).
        let mut parts = std_parts();
        parts.insert(0, tlv(0xA0, &int_small(3)));
        assert_eq!(mutant(&parts), "x509: invalid version");
        // Negative version.
        let mut parts = std_parts();
        parts.insert(0, tlv(0xA0, &int_raw(&[0xFF])));
        assert_eq!(mutant(&parts), "x509: malformed version");

        // Serial not an INTEGER.
        let mut parts = std_parts();
        parts[0] = seq(&[]);
        assert_eq!(mutant(&parts), "x509: malformed serial number");
        // Negative serial.
        let mut parts = std_parts();
        parts[0] = int_raw(&[0xFF]);
        assert_eq!(mutant(&parts), "x509: negative serial number");

        // Inner AI not a SEQUENCE.
        let mut parts = std_parts();
        parts[1] = int_small(1);
        assert_eq!(
            mutant(&parts),
            "x509: malformed signature algorithm identifier"
        );

        // Issuer not a SEQUENCE.
        let mut parts = std_parts();
        parts[2] = octet(&[1]);
        assert_eq!(mutant(&parts), "x509: malformed issuer");

        // Validity not a SEQUENCE.
        let mut parts = std_parts();
        parts[3] = int_small(1);
        assert_eq!(mutant(&parts), "x509: malformed validity");
        // Malformed notBefore (UTCTime too short).
        let mut parts = std_parts();
        parts[3] = seq(&concat(&[
            utctime("70010100000Z"),
            utctime("700102000000Z"),
        ]));
        assert_eq!(mutant(&parts), "x509: malformed UTCTime");
        // Malformed notAfter (bad zone).
        let mut parts = std_parts();
        parts[3] = seq(&concat(&[
            utctime("700101000000Z"),
            utctime("700102000000X"),
        ]));
        assert_eq!(mutant(&parts), "x509: malformed UTCTime");

        // Subject uses the issuer's error text (Go quirk).
        let mut parts = std_parts();
        parts[4] = int_small(1);
        assert_eq!(mutant(&parts), "x509: malformed issuer");

        // SPKI not a SEQUENCE.
        let mut parts = std_parts();
        parts[5] = int_small(1);
        assert_eq!(mutant(&parts), "x509: malformed spki");
        // SPKI algorithm identifier not a SEQUENCE.
        let mut parts = std_parts();
        parts[5] = seq(&concat(&[int_small(1), bitstring(&[1, 2, 3])]));
        assert_eq!(
            mutant(&parts),
            "x509: malformed public key algorithm identifier"
        );
        // SPKI key not a BIT STRING.
        let mut parts = std_parts();
        parts[5] = seq(&concat(&[
            ai_element(OID_RSA_ENC, Some(&null())),
            octet(&[1, 2, 3]),
        ]));
        assert_eq!(mutant(&parts), "x509: malformed subjectPublicKey");

        // Outer AI not a SEQUENCE.
        let bad_outer = seq(&concat(&[
            seq(&concat(&std_parts())),
            int_small(1),
            bitstring(&[1]),
        ]));
        assert_eq!(
            err_text(parse_certificate(&bad_outer)),
            "x509: malformed algorithm identifier"
        );

        // Inner/outer AI mismatch.
        let mismatch =
            cert_from_tbs_parts(&std_parts(), &ai_element(OID_SHA1_RSA, Some(&null())), &[1]);
        assert_eq!(
            err_text(parse_certificate(&mismatch)),
            "x509: inner and outer signature algorithm identifiers don't match"
        );

        // Signature not a BIT STRING.
        let bad_sig = seq(&concat(&[
            seq(&concat(&std_parts())),
            ai_element(OID_SHA256_RSA, Some(&null())),
            octet(&[1]),
        ]));
        assert_eq!(
            err_text(parse_certificate(&bad_sig)),
            "x509: malformed signature"
        );
    }

    #[test]
    fn parse_unknown_public_key_algorithm_yields_none() {
        let mut parts = std_parts();
        parts[5] = seq(&concat(&[
            ai_element(&[0x2A, 0x03], None),
            bitstring(&[1, 2, 3]),
        ]));
        let cert = parse_certificate(&cert_from_tbs_parts(
            &parts,
            &ai_element(OID_SHA256_RSA, Some(&null())),
            &[1],
        ))
        .expect("unknown pk algorithm parses");
        assert_eq!(cert.public_key_algorithm, PublicKeyAlgorithm::Unknown);
        assert!(cert.public_key.is_none());
    }

    #[test]
    fn parse_unknown_signature_algorithm_yields_unknown() {
        let ai = ai_element(&[0x2A, 0x03], None);
        let cert = parse_certificate(&cert_from_tbs_parts(
            &{
                let mut parts = std_parts();
                parts[1] = ai.clone();
                parts
            },
            &ai,
            &[1],
        ))
        .expect("unknown sig algorithm parses");
        assert_eq!(cert.signature_algorithm, SignatureAlgorithm::Unknown);
    }

    #[test]
    fn parse_versions_and_unique_ids() {
        let outer = ai_element(OID_SHA256_RSA, Some(&null()));
        // v2 without unique IDs.
        let mut parts = std_parts();
        parts.insert(0, tlv(0xA0, &int_small(1)));
        let cert =
            parse_certificate(&cert_from_tbs_parts(&parts, &outer, &[1])).expect("v2 parses");
        assert_eq!(cert.version, 2);
        // v2 with well-formed unique IDs (skipped).
        let mut parts = parts.clone();
        parts.push(tlv(0x81, &[0x00, 0xAA]));
        parts.push(tlv(0x82, &[0x00, 0xBB]));
        let cert = parse_certificate(&cert_from_tbs_parts(&parts, &outer, &[1]))
            .expect("v2 with unique IDs parses");
        assert_eq!(cert.version, 2);
        // v3 without extensions.
        let mut parts = std_parts();
        parts.insert(0, tlv(0xA0, &int_small(2)));
        let cert = parse_certificate(&cert_from_tbs_parts(&parts, &outer, &[1]))
            .expect("v3 without extensions parses");
        assert_eq!(cert.version, 3);
        assert!(!cert.basic_constraints_valid);
        // v3 with malformed extensions wrapper.
        let mut parts = std_parts();
        parts.insert(0, tlv(0xA0, &int_small(2)));
        parts.push(tlv(0xA3, &[0x04, 0x01, 0x00]));
        assert_eq!(
            err_text(parse_certificate(&cert_from_tbs_parts(
                &parts,
                &outer,
                &[1]
            ))),
            "x509: malformed extensions"
        );
    }

    fn spki_with_key(pk_alg: &[u8], key_der: &[u8]) -> Vec<u8> {
        seq(&concat(&[pk_alg.to_vec(), bitstring(key_der)]))
    }

    fn key_err(spki: &[u8]) -> String {
        let mut parts = std_parts();
        parts[5] = spki.to_vec();
        err_text(parse_certificate(&cert_from_tbs_parts(
            &parts,
            &ai_element(OID_SHA256_RSA, Some(&null())),
            &[1],
        )))
    }

    #[test]
    fn parse_rsa_key_errors() {
        let key = |n: &[u8], e: &[u8]| seq(&concat(&[int_raw(n), int_raw(e)]));
        let rsa = ai_element(OID_RSA_ENC, Some(&null()));
        // Missing NULL parameters (absent and wrong).
        assert_eq!(
            key_err(&spki_with_key(
                &ai_element(OID_RSA_ENC, None),
                &key(&[1], &[1])
            )),
            "x509: RSA key missing NULL parameters"
        );
        assert_eq!(
            key_err(&spki_with_key(
                &ai_element(OID_RSA_ENC, Some(&octet(&[]))),
                &key(&[1], &[1])
            )),
            "x509: RSA key missing NULL parameters"
        );
        // Key not a SEQUENCE.
        assert_eq!(
            key_err(&spki_with_key(&rsa, &int_small(1))),
            "x509: invalid RSA public key"
        );
        // Modulus not an INTEGER.
        assert_eq!(
            key_err(&spki_with_key(
                &rsa,
                &seq(&concat(&[octet(&[1]), int_small(1)]))
            )),
            "x509: invalid RSA modulus"
        );
        // Exponent missing.
        assert_eq!(
            key_err(&spki_with_key(&rsa, &seq(&int_small(17)))),
            "x509: invalid RSA public exponent"
        );
        // Non-positive modulus.
        assert_eq!(
            key_err(&spki_with_key(&rsa, &key(&[0xFF], &[1]))),
            "x509: RSA modulus is not a positive number"
        );
        assert_eq!(
            key_err(&spki_with_key(&rsa, &key(&[0x00], &[1]))),
            "x509: RSA modulus is not a positive number"
        );
        // Non-positive exponent.
        assert_eq!(
            key_err(&spki_with_key(&rsa, &key(&[0x11], &[0x00]))),
            "x509: RSA public exponent is not a positive number"
        );
        assert_eq!(
            key_err(&spki_with_key(&rsa, &key(&[0x11], &[0xFF]))),
            "x509: RSA public exponent is not a positive number"
        );
        // Exponent too large for int64 is still an invalid exponent.
        assert_eq!(
            key_err(&spki_with_key(&rsa, &key(&[0x11], &[0x01; 9]))),
            "x509: invalid RSA public exponent"
        );
    }

    fn dsa_spki(y: &[u8], params: &[u8]) -> Vec<u8> {
        spki_with_key(&ai_element(OID_DSA, Some(params)), &int_raw(y))
    }

    fn dsa_params(p: &[u8], q: &[u8], g: &[u8]) -> Vec<u8> {
        seq(&concat(&[int_raw(p), int_raw(q), int_raw(g)]))
    }

    #[test]
    fn parse_dsa_key() {
        // Happy path stores magnitudes.
        let mut parts = std_parts();
        parts[5] = dsa_spki(&[0x00, 0x80], &dsa_params(&[0x11], &[0x13], &[0x17]));
        let cert = parse_certificate(&cert_from_tbs_parts(
            &parts,
            &ai_element(OID_SHA256_RSA, Some(&null())),
            &[1],
        ))
        .expect("dsa parses");
        assert_eq!(cert.public_key_algorithm, PublicKeyAlgorithm::Dsa);
        match cert.public_key {
            Some(PublicKeyData::Dsa { y, p, q, g }) => {
                assert_eq!(y, vec![0x80]);
                assert_eq!(p, vec![0x11]);
                assert_eq!(q, vec![0x13]);
                assert_eq!(g, vec![0x17]);
            }
            _ => panic!("expected DSA key"),
        }
        // Y not an INTEGER.
        assert_eq!(
            key_err(&spki_with_key(
                &ai_element(OID_DSA, Some(&dsa_params(&[0x11], &[0x13], &[0x17]))),
                &octet(&[1])
            )),
            "x509: invalid DSA public key"
        );
        // Params not a SEQUENCE.
        assert_eq!(
            key_err(&dsa_spki(&[0x11], &int_small(1))),
            "x509: invalid DSA parameters"
        );
        // Q missing.
        assert_eq!(
            key_err(&dsa_spki(&[0x11], &seq(&int_raw(&[0x11])))),
            "x509: invalid DSA parameters"
        );
        // Zero and negative parameters.
        assert_eq!(
            key_err(&dsa_spki(&[0x00], &dsa_params(&[0x11], &[0x13], &[0x17]))),
            "x509: zero or negative DSA parameter"
        );
        assert_eq!(
            key_err(&dsa_spki(&[0x11], &dsa_params(&[0x11], &[0xFF], &[0x17]))),
            "x509: zero or negative DSA parameter"
        );
    }

    fn ec_spki(curve_body: &[u8], point: &[u8]) -> Vec<u8> {
        spki_with_key(&ai_element(OID_EC_PUB, Some(&oid(curve_body))), point)
    }

    #[test]
    fn parse_ecdsa_key_errors() {
        // Params not an OID.
        assert_eq!(
            key_err(&spki_with_key(
                &ai_element(OID_EC_PUB, Some(&null())),
                &[0x04, 0x01, 0x02]
            )),
            "x509: invalid ECDSA parameters"
        );
        // Unsupported curve.
        assert_eq!(
            key_err(&ec_spki(&[0x2A, 0x03], &[0x04, 0x01, 0x02])),
            "x509: unsupported elliptic curve"
        );
        // Empty and compressed points.
        assert_eq!(
            key_err(&ec_spki(OID_P256, &[])),
            "ecdsa: invalid uncompressed public key"
        );
        let mut compressed = vec![0x02];
        compressed.extend_from_slice(&[0x11; 32]);
        assert_eq!(
            key_err(&ec_spki(OID_P256, &compressed)),
            "ecdsa: invalid uncompressed public key"
        );
        // Wrong length.
        let mut short = vec![0x04];
        short.extend_from_slice(&[0x11; 63]);
        assert_eq!(
            key_err(&ec_spki(OID_P256, &short)),
            "invalid P256 point encoding"
        );
        // X equal to the field prime is out of range.
        let mut point = vec![0x04];
        point.extend_from_slice(&P256_PRIME);
        point.extend_from_slice(&[0x01; 32]);
        assert_eq!(key_err(&ec_spki(OID_P256, &point)), P256_ELEMENT_ERROR);
        // Y equal to the field prime is out of range.
        let mut point = vec![0x04];
        point.extend_from_slice(&[0x01; 32]);
        point.extend_from_slice(&P256_PRIME);
        assert_eq!(key_err(&ec_spki(OID_P256, &point)), P256_ELEMENT_ERROR);
        // In-range but off-curve.
        let mut point = vec![0x04];
        point.extend_from_slice(&[0x00; 31]);
        point.push(0x01);
        point.extend_from_slice(&[0x00; 31]);
        point.push(0x01);
        assert_eq!(
            key_err(&ec_spki(OID_P256, &point)),
            "P256 point not on curve"
        );
    }

    #[test]
    fn parse_ecdsa_p256_generator() {
        // The P-256 base point is the standard on-curve vector.
        let gx = [
            0x6B, 0x17, 0xD1, 0xF2, 0xE1, 0x2C, 0x42, 0x47, 0xF8, 0xBC, 0xE6, 0xE5, 0x63, 0xA4,
            0x40, 0xF2, 0x77, 0x03, 0x7D, 0x81, 0x2D, 0xEB, 0x33, 0xA0, 0xF4, 0xA1, 0x39, 0x45,
            0xD8, 0x98, 0xC2, 0x96,
        ];
        let gy = [
            0x4F, 0xE3, 0x42, 0xE2, 0xFE, 0x1A, 0x7F, 0x9B, 0x8E, 0xE7, 0xEB, 0x4A, 0x7C, 0x0F,
            0x9E, 0x16, 0x2B, 0xCE, 0x33, 0x57, 0x6B, 0x31, 0x5E, 0xCE, 0xCB, 0xB6, 0x40, 0x68,
            0x37, 0xBF, 0x51, 0xF5,
        ];
        let mut point = vec![0x04];
        point.extend_from_slice(&gx);
        point.extend_from_slice(&gy);
        let mut parts = std_parts();
        parts[5] = ec_spki(OID_P256, &point);
        let cert = parse_certificate(&cert_from_tbs_parts(
            &parts,
            &ai_element(OID_SHA256_RSA, Some(&null())),
            &[1],
        ))
        .expect("p256 generator parses");
        assert_eq!(cert.public_key_algorithm, PublicKeyAlgorithm::Ecdsa);
        assert!(matches!(cert.public_key, Some(PublicKeyData::Ecdsa256(_))));
    }

    #[test]
    fn parse_ed25519_key() {
        // Happy path.
        let mut parts = std_parts();
        parts[5] = spki_with_key(&ai_element(OID_ED25519, None), &[0x42; 32]);
        let cert = parse_certificate(&cert_from_tbs_parts(
            &parts,
            &ai_element(OID_SHA256_RSA, Some(&null())),
            &[1],
        ))
        .expect("ed25519 parses");
        assert_eq!(cert.public_key_algorithm, PublicKeyAlgorithm::Ed25519);
        assert!(matches!(
            cert.public_key,
            Some(PublicKeyData::Ed25519(k)) if k == [0x42; 32]
        ));
        // Parameters must be absent (even NULL is illegal).
        assert_eq!(
            key_err(&spki_with_key(
                &ai_element(OID_ED25519, Some(&null())),
                &[0x42; 32]
            )),
            "x509: Ed25519 key encoded with illegal parameters"
        );
        // Wrong size.
        assert_eq!(
            key_err(&spki_with_key(&ai_element(OID_ED25519, None), &[0x42; 31])),
            "x509: wrong Ed25519 public key size"
        );
        assert_eq!(
            key_err(&spki_with_key(&ai_element(OID_ED25519, None), &[0x42; 33])),
            "x509: wrong Ed25519 public key size"
        );
    }

    fn sig_alg(oid_body: &[u8], params: Option<&[u8]>) -> SignatureAlgorithm {
        let ai = ai_element(oid_body, params);
        let mut reader = Reader::new(&ai);
        let contents = reader.read_asn1(0x30).expect("ai seq");
        let parsed = parse_ai(contents).expect("parse ai");
        signature_algorithm_from_ai(&parsed)
    }

    #[test]
    fn signature_algorithm_mapping() {
        // RSA/ECDSA OIDs map regardless of parameters.
        assert_eq!(
            sig_alg(OID_SHA256_RSA, Some(&null())),
            SignatureAlgorithm::Sha256Rsa
        );
        assert_eq!(sig_alg(OID_SHA256_RSA, None), SignatureAlgorithm::Sha256Rsa);
        assert_eq!(sig_alg(OID_SHA1_RSA, None), SignatureAlgorithm::Sha1Rsa);
        assert_eq!(
            sig_alg(OID_ISO_SHA1_RSA, Some(&null())),
            SignatureAlgorithm::Sha1Rsa
        );
        assert_eq!(sig_alg(OID_MD5_RSA, None), SignatureAlgorithm::Md5Rsa);
        assert_eq!(
            sig_alg(OID_ECDSA_SHA256, None),
            SignatureAlgorithm::EcdsaSha256
        );
        assert_eq!(sig_alg(OID_ECDSA_SHA1, None), SignatureAlgorithm::EcdsaSha1);
        assert_eq!(sig_alg(OID_DSA_SHA256, None), SignatureAlgorithm::DsaSha256);
        // Ed25519 requires absent parameters.
        assert_eq!(sig_alg(OID_ED25519, None), SignatureAlgorithm::Ed25519);
        assert_eq!(
            sig_alg(OID_ED25519, Some(&null())),
            SignatureAlgorithm::Unknown
        );
        // Unknown OIDs map to Unknown.
        assert_eq!(sig_alg(&[0x2A, 0x03], None), SignatureAlgorithm::Unknown);
    }

    fn pss_params(hash: &[u8], mgf_hash: &[u8], salt: u64, trailer: Option<u64>) -> Vec<u8> {
        let hash_ai = ai_element(hash, None);
        let mgf_inner = ai_element(mgf_hash, None);
        let mgf_ai = ai_element(OID_MGF1, Some(&mgf_inner));
        let mut contents = tlv(0xA0, &hash_ai);
        contents.extend_from_slice(&tlv(0xA1, &mgf_ai));
        contents.extend_from_slice(&tlv(0xA2, &int_small(salt)));
        if let Some(t) = trailer {
            contents.extend_from_slice(&tlv(0xA3, &int_small(t)));
        }
        seq(&contents)
    }

    #[test]
    fn pss_algorithm_buckets() {
        // The three standard buckets.
        assert_eq!(
            sig_alg(OID_PSS, Some(&pss_params(OID_SHA256, OID_SHA256, 32, None))),
            SignatureAlgorithm::Sha256Pss
        );
        assert_eq!(
            sig_alg(OID_PSS, Some(&pss_params(OID_SHA384, OID_SHA384, 48, None))),
            SignatureAlgorithm::Sha384Pss
        );
        assert_eq!(
            sig_alg(OID_PSS, Some(&pss_params(OID_SHA512, OID_SHA512, 64, None))),
            SignatureAlgorithm::Sha512Pss
        );
        // Explicit default trailer is fine.
        assert_eq!(
            sig_alg(
                OID_PSS,
                Some(&pss_params(OID_SHA256, OID_SHA256, 32, Some(1)))
            ),
            SignatureAlgorithm::Sha256Pss
        );
        // Everything else collapses to Unknown.
        assert_eq!(
            sig_alg(OID_PSS, Some(&pss_params(OID_SHA256, OID_SHA256, 20, None))),
            SignatureAlgorithm::Unknown
        );
        assert_eq!(
            sig_alg(OID_PSS, Some(&pss_params(OID_SHA256, OID_SHA384, 32, None))),
            SignatureAlgorithm::Unknown
        );
        assert_eq!(
            sig_alg(
                OID_PSS,
                Some(&pss_params(OID_SHA256, OID_SHA256, 32, Some(2)))
            ),
            SignatureAlgorithm::Unknown
        );
        assert_eq!(
            sig_alg(OID_PSS, Some(&pss_params(OID_SHA384, OID_SHA384, 32, None))),
            SignatureAlgorithm::Unknown
        );
        // Hash parameters may be explicit NULL.
        let hash_ai = ai_element(OID_SHA256, Some(&null()));
        let mgf_inner = ai_element(OID_SHA256, None);
        let mgf_ai = ai_element(OID_MGF1, Some(&mgf_inner));
        let mut contents = tlv(0xA0, &hash_ai);
        contents.extend_from_slice(&tlv(0xA1, &mgf_ai));
        contents.extend_from_slice(&tlv(0xA2, &int_small(32)));
        assert_eq!(
            sig_alg(OID_PSS, Some(&seq(&contents))),
            SignatureAlgorithm::Sha256Pss
        );
        // Garbage parameters.
        assert_eq!(
            sig_alg(OID_PSS, Some(&octet(&[1, 2, 3]))),
            SignatureAlgorithm::Unknown
        );
        assert_eq!(sig_alg(OID_PSS, None), SignatureAlgorithm::Unknown);
    }

    fn validity_time(nb: &[u8], na: &[u8]) -> (i64, i64) {
        let mut parts = std_parts();
        parts[3] = seq(&concat(&[nb.to_vec(), na.to_vec()]));
        let cert = parse_certificate(&cert_from_tbs_parts(
            &parts,
            &ai_element(OID_SHA256_RSA, Some(&null())),
            &[1],
        ))
        .expect("validity parses");
        (cert.not_before, cert.not_after)
    }

    #[test]
    fn validity_time_values() {
        // UTCTime year mapping and epoch.
        assert_eq!(
            validity_time(&utctime("700101000000Z"), &utctime("700102000000Z")),
            (0, 86400)
        );
        assert_eq!(
            validity_time(&utctime("500101000000Z"), &utctime("700101000000Z")).0,
            -631152000
        );
        assert_eq!(
            validity_time(&utctime("490101000000Z"), &utctime("700101000000Z")).0,
            2493072000
        );
        // GeneralizedTime.
        assert_eq!(
            validity_time(&gentime("19700101000000Z"), &utctime("700101000000Z")).0,
            0
        );
        // Numeric zones shift to UTC.
        assert_eq!(
            validity_time(&utctime("700101000000+0130"), &utctime("700101000000Z")).0,
            -5400
        );
        assert_eq!(
            validity_time(&utctime("700101030000-0130"), &utctime("700101000000Z")).0,
            16200
        );
        // Leap day.
        assert_eq!(
            validity_time(&utctime("000229120000Z"), &utctime("700101000000Z")).0,
            951825600
        );
    }

    #[test]
    fn validity_time_rejections() {
        // Month 13.
        let mut parts = std_parts();
        parts[3] = seq(&concat(&[
            utctime("701301000000Z"),
            utctime("700102000000Z"),
        ]));
        assert_eq!(
            err_text(parse_certificate(&cert_from_tbs_parts(
                &parts,
                &ai_element(OID_SHA256_RSA, Some(&null())),
                &[1]
            ))),
            "x509: malformed UTCTime"
        );
        // Hour 24.
        parts[3] = seq(&concat(&[
            utctime("700101240000Z"),
            utctime("700102000000Z"),
        ]));
        assert_eq!(
            err_text(parse_certificate(&cert_from_tbs_parts(
                &parts,
                &ai_element(OID_SHA256_RSA, Some(&null())),
                &[1]
            ))),
            "x509: malformed UTCTime"
        );
        // GeneralizedTime wrong length.
        parts[3] = seq(&concat(&[
            gentime("1970010100000Z"),
            utctime("700102000000Z"),
        ]));
        assert_eq!(
            err_text(parse_certificate(&cert_from_tbs_parts(
                &parts,
                &ai_element(OID_SHA256_RSA, Some(&null())),
                &[1]
            ))),
            "x509: malformed GeneralizedTime"
        );
        // Non-UTCTime/GeneralizedTime tag.
        parts[3] = seq(&concat(&[
            octet(b"700101000000Z"),
            utctime("700102000000Z"),
        ]));
        assert_eq!(
            err_text(parse_certificate(&cert_from_tbs_parts(
                &parts,
                &ai_element(OID_SHA256_RSA, Some(&null())),
                &[1]
            ))),
            "x509: unsupported time format"
        );
    }

    #[test]
    fn mailbox_acceptance() {
        assert!(parse_rfc2821_mailbox(b"user@example.com"));
        assert!(parse_rfc2821_mailbox(b"a@b"));
        assert!(parse_rfc2821_mailbox(b"\"a b\"@c"));
        assert!(parse_rfc2821_mailbox(b"a\\ b@c"));
        assert!(!parse_rfc2821_mailbox(b""));
        assert!(!parse_rfc2821_mailbox(b"a"));
        assert!(!parse_rfc2821_mailbox(b"@b"));
        // Go quirk: an empty domain passes domainToReverseLabels, so "a@" is valid.
        assert!(parse_rfc2821_mailbox(b"a@"));
        // Go quirk: '@' (64) is a legal domain char, so "a@b@c" is valid.
        assert!(parse_rfc2821_mailbox(b"a@b@c"));
        assert!(!parse_rfc2821_mailbox(b"\"a@c"));
        assert!(!parse_rfc2821_mailbox(b"\"a\"b@c"));
        assert!(!parse_rfc2821_mailbox(b".a@b"));
        assert!(!parse_rfc2821_mailbox(b"a.@b"));
        assert!(!parse_rfc2821_mailbox(b"a..b@c"));
        assert!(!parse_rfc2821_mailbox(b"a b@c"));
        assert!(!parse_rfc2821_mailbox(b"a@b "));
    }

    #[test]
    fn ecdsa_signature_lenient_parse() {
        // Minimal valid.
        let sig = seq(&concat(&[int_small(1), int_small(2)]));
        assert_eq!(parse_ecdsa_signature(&sig), Some((vec![0x01], vec![0x02])));
        // Non-minimal integers are accepted (encoding/asn1 semantics).
        let sig = seq(&concat(&[int_raw(&[0x00, 0x00, 0x01]), int_small(2)]));
        assert_eq!(parse_ecdsa_signature(&sig), Some((vec![0x01], vec![0x02])));
        // Negative, zero, and empty values are rejected.
        let sig = seq(&concat(&[int_raw(&[0xFF]), int_small(2)]));
        assert_eq!(parse_ecdsa_signature(&sig), None);
        let sig = seq(&concat(&[int_small(1), int_raw(&[0x00])]));
        assert_eq!(parse_ecdsa_signature(&sig), None);
        let sig = seq(&concat(&[tlv(0x02, &[]), int_small(2)]));
        assert_eq!(parse_ecdsa_signature(&sig), None);
        // Wrong arity, trailing bytes, wrong tag.
        let sig = seq(&concat(&[int_small(1), int_small(2), int_small(3)]));
        assert_eq!(parse_ecdsa_signature(&sig), None);
        let mut sig = seq(&concat(&[int_small(1), int_small(2)]));
        sig.push(0x00);
        assert_eq!(parse_ecdsa_signature(&sig), None);
        assert_eq!(parse_ecdsa_signature(&[0x04, 0x02, 0x01, 0x02]), None);
    }

    fn extension(oid_body: &[u8], critical: bool, value: &[u8]) -> Vec<u8> {
        let mut contents = oid(oid_body);
        if critical {
            contents.extend_from_slice(&boolean(true));
        }
        contents.extend_from_slice(&octet(value));
        seq(&contents)
    }

    fn v3_parts(exts: &[Vec<u8>]) -> Vec<Vec<u8>> {
        let mut parts = std_parts();
        parts.insert(0, tlv(0xA0, &int_small(2)));
        parts.push(tlv(0xA3, &seq(&concat(exts))));
        parts
    }

    fn v3_cert(exts: &[Vec<u8>]) -> Vec<u8> {
        cert_from_tbs_parts(
            &v3_parts(exts),
            &ai_element(OID_SHA256_RSA, Some(&null())),
            &[1],
        )
    }

    fn ku_value(bits: &[u8], unused: u8) -> Vec<u8> {
        let mut contents = vec![unused];
        contents.extend_from_slice(bits);
        tlv(0x03, &contents)
    }

    #[test]
    fn extensions_key_usage_bits() {
        // certSign only: bit 5 of the first byte, one unused bit.
        let cert = parse_certificate(&v3_cert(&[extension(
            OID_KEY_USAGE,
            false,
            &ku_value(&[0x04], 1),
        )]))
        .expect("ku parses");
        assert_eq!(cert.key_usage, KEY_USAGE_CERT_SIGN);
        // digitalSignature + keyEncipherment: bits 0 and 2.
        let cert = parse_certificate(&v3_cert(&[extension(
            OID_KEY_USAGE,
            true,
            &ku_value(&[0xA0], 1),
        )]))
        .expect("ku parses");
        assert_eq!(
            cert.key_usage,
            KEY_USAGE_DIGITAL_SIGNATURE | KEY_USAGE_KEY_ENCIPHERMENT
        );
        // decipherOnly lives in the second byte (bit 8).
        let cert = parse_certificate(&v3_cert(&[extension(
            OID_KEY_USAGE,
            false,
            &ku_value(&[0x00, 0x80], 7),
        )]))
        .expect("ku parses");
        assert_eq!(cert.key_usage, KEY_USAGE_DECIPHER_ONLY);
        // Non-BIT-STRING value is invalid.
        assert_eq!(
            err_text(parse_certificate(&v3_cert(&[extension(
                OID_KEY_USAGE,
                false,
                &octet(&[1])
            )]))),
            "x509: invalid key usage"
        );
    }

    #[test]
    fn extensions_basic_constraints() {
        // CA with no path length: -1, zero flag clear.
        let cert = parse_certificate(&v3_cert(&[extension(
            OID_BASIC_CONSTRAINTS,
            true,
            &seq(&boolean(true)),
        )]))
        .expect("bc parses");
        assert!(cert.is_ca);
        assert!(cert.basic_constraints_valid);
        assert_eq!(cert.max_path_len, -1);
        assert!(!cert.max_path_len_zero);
        // Explicit zero path length sets the zero flag.
        let cert = parse_certificate(&v3_cert(&[extension(
            OID_BASIC_CONSTRAINTS,
            true,
            &seq(&concat(&[boolean(true), int_small(0)])),
        )]))
        .expect("bc parses");
        assert!(cert.is_ca);
        assert_eq!(cert.max_path_len, 0);
        assert!(cert.max_path_len_zero);
        // Non-zero path length.
        let cert = parse_certificate(&v3_cert(&[extension(
            OID_BASIC_CONSTRAINTS,
            true,
            &seq(&concat(&[boolean(true), int_small(3)])),
        )]))
        .expect("bc parses");
        assert_eq!(cert.max_path_len, 3);
        assert!(!cert.max_path_len_zero);
        // Empty constraints: not a CA.
        let cert = parse_certificate(&v3_cert(&[extension(
            OID_BASIC_CONSTRAINTS,
            true,
            &seq(&[]),
        )]))
        .expect("bc parses");
        assert!(!cert.is_ca);
        assert!(cert.basic_constraints_valid);
        // Non-SEQUENCE value is invalid.
        assert_eq!(
            err_text(parse_certificate(&v3_cert(&[extension(
                OID_BASIC_CONSTRAINTS,
                true,
                &octet(&[1])
            )]))),
            "x509: invalid basic constraints"
        );
    }

    #[test]
    fn extensions_unhandled_and_duplicates() {
        // Critical unknown extension is recorded dotted.
        let cert = parse_certificate(&v3_cert(&[extension(&[0x2A, 0x03], true, &[1, 2])]))
            .expect("unknown critical parses");
        assert_eq!(cert.unhandled_critical, vec!["1.2.3".to_string()]);
        // Non-critical unknown extension is ignored.
        let cert = parse_certificate(&v3_cert(&[extension(&[0x2A, 0x03], false, &[1, 2])]))
            .expect("unknown non-critical parses");
        assert!(cert.unhandled_critical.is_empty());
        // Duplicate OID quotes the dotted string (Go %q).
        let ku = extension(OID_KEY_USAGE, false, &ku_value(&[0x04], 1));
        assert_eq!(
            err_text(parse_certificate(&v3_cert(&[ku.clone(), ku]))),
            "x509: certificate contains duplicate extension with OID \"2.5.29.15\""
        );
        // Non-SEQUENCE extension element.
        assert_eq!(
            err_text(parse_certificate(&v3_cert(&[int_small(1)]))),
            "x509: malformed extension"
        );
        // Extension fields.
        assert_eq!(
            err_text(parse_certificate(&v3_cert(&[seq(&concat(&[
                octet(&[1]),
                octet(&[2])
            ]))]))),
            "x509: malformed extension OID field"
        );
        assert_eq!(
            err_text(parse_certificate(&v3_cert(&[seq(&concat(&[
                oid(&[0x2A, 0x03]),
                boolean(true),
                int_small(1)
            ]))]))),
            "x509: malformed extension value field"
        );
    }

    /// v3 CA parent with the given SPKI and extra extensions.
    fn ca_cert(spki: &[u8], extra: &[Vec<u8>]) -> Certificate {
        let mut exts = vec![extension(OID_BASIC_CONSTRAINTS, true, &seq(&boolean(true)))];
        exts.extend_from_slice(extra);
        let mut parts = v3_parts(&exts);
        parts[6] = spki.to_vec();
        parse_certificate(&cert_from_tbs_parts(
            &parts,
            &ai_element(OID_SHA256_RSA, Some(&null())),
            &[1],
        ))
        .expect("ca parses")
    }

    fn child_with_alg(sig_oid: &[u8], params: Option<&[u8]>) -> Certificate {
        let ai = ai_element(sig_oid, params);
        let mut parts = std_parts();
        parts[1] = ai.clone();
        parse_certificate(&cert_from_tbs_parts(&parts, &ai, &[0xDE, 0xAD])).expect("child parses")
    }

    #[test]
    fn check_signature_parent_constraints() {
        let child = child_with_alg(OID_SHA256_RSA, Some(&null()));
        // v3 parent without basic constraints.
        let parent = parse_certificate(&v3_cert(&[])).expect("parent parses");
        assert_eq!(
            check_signature_from(&child, &parent),
            Err(String::from(
                "x509: invalid signature: parent certificate cannot sign this kind of certificate"
            ))
        );
        // Basic constraints present but not a CA.
        let parent = parse_certificate(&v3_cert(&[extension(
            OID_BASIC_CONSTRAINTS,
            true,
            &seq(&boolean(false)),
        )]))
        .expect("parent parses");
        assert_eq!(
            check_signature_from(&child, &parent),
            Err(String::from(
                "x509: invalid signature: parent certificate cannot sign this kind of certificate"
            ))
        );
        // Key usage without certSign.
        let parent = ca_cert(
            &std_spki(),
            &[extension(OID_KEY_USAGE, true, &ku_value(&[0x80], 1))],
        );
        assert_eq!(
            check_signature_from(&child, &parent),
            Err(String::from(
                "x509: invalid signature: parent certificate cannot sign this kind of certificate"
            ))
        );
        // Unknown parent public key algorithm.
        let mut parts = v3_parts(&[extension(OID_BASIC_CONSTRAINTS, true, &seq(&boolean(true)))]);
        parts[6] = seq(&concat(&[ai_element(&[0x2A, 0x03], None), bitstring(&[1])]));
        let parent = parse_certificate(&cert_from_tbs_parts(
            &parts,
            &ai_element(OID_SHA256_RSA, Some(&null())),
            &[1],
        ))
        .expect("parent parses");
        assert_eq!(
            check_signature_from(&child, &parent),
            Err(String::from(
                "x509: cannot verify signature: algorithm unimplemented"
            ))
        );
    }

    #[test]
    fn check_signature_algorithm_gates() {
        let parent = ca_cert(&std_spki(), &[]);
        // Insecure hashes name Go's algorithm string.
        for (oid_body, name) in [
            (OID_MD5_RSA, "MD5-RSA"),
            (OID_SHA1_RSA, "SHA1-RSA"),
            (OID_ECDSA_SHA1, "ECDSA-SHA1"),
        ] {
            let child = child_with_alg(oid_body, Some(&null()));
            assert_eq!(
                check_signature_from(&child, &parent),
                Err(format!(
                    "x509: cannot verify signature: insecure algorithm {name}"
                )),
                "{name} must be rejected"
            );
        }
        // Unknown child algorithm.
        let child = child_with_alg(&[0x2A, 0x03], None);
        assert_eq!(
            check_signature_from(&child, &parent),
            Err(String::from(
                "x509: cannot verify signature: algorithm unimplemented"
            ))
        );
    }

    #[test]
    fn check_signature_key_mismatches() {
        let rsa_parent = ca_cert(&std_spki(), &[]);
        // RSA key with an ECDSA signature algorithm.
        let child = child_with_alg(OID_ECDSA_SHA256, None);
        assert_eq!(
            check_signature_from(&child, &rsa_parent),
            Err(String::from(
                "x509: signature algorithm specifies an ECDSA public key, but have public key of type *rsa.PublicKey"
            ))
        );
        // RSA key with a DSA signature algorithm (Go's "an DSA" quirk: the
        // DSA-SHA256 hash is secure, so dispatch reaches the key check).
        let child = child_with_alg(OID_DSA_SHA256, None);
        assert_eq!(
            check_signature_from(&child, &rsa_parent),
            Err(String::from(
                "x509: signature algorithm specifies an DSA public key, but have public key of type *rsa.PublicKey"
            ))
        );
        // ECDSA (P-256 generator) key with an RSA signature algorithm.
        let mut point = vec![0x04];
        point.extend_from_slice(&[
            0x6B, 0x17, 0xD1, 0xF2, 0xE1, 0x2C, 0x42, 0x47, 0xF8, 0xBC, 0xE6, 0xE5, 0x63, 0xA4,
            0x40, 0xF2, 0x77, 0x03, 0x7D, 0x81, 0x2D, 0xEB, 0x33, 0xA0, 0xF4, 0xA1, 0x39, 0x45,
            0xD8, 0x98, 0xC2, 0x96,
        ]);
        point.extend_from_slice(&[
            0x4F, 0xE3, 0x42, 0xE2, 0xFE, 0x1A, 0x7F, 0x9B, 0x8E, 0xE7, 0xEB, 0x4A, 0x7C, 0x0F,
            0x9E, 0x16, 0x2B, 0xCE, 0x33, 0x57, 0x6B, 0x31, 0x5E, 0xCE, 0xCB, 0xB6, 0x40, 0x68,
            0x37, 0xBF, 0x51, 0xF5,
        ]);
        let ec_parent = ca_cert(&ec_spki(OID_P256, &point), &[]);
        let child = child_with_alg(OID_SHA256_RSA, Some(&null()));
        assert_eq!(
            check_signature_from(&child, &ec_parent),
            Err(String::from(
                "x509: signature algorithm specifies an RSA public key, but have public key of type *ecdsa.PublicKey"
            ))
        );
        // DSA keys have no verify arm in Go: always unimplemented.
        let dsa_parent = ca_cert(
            &dsa_spki(&[0x11], &dsa_params(&[0x17], &[0x13], &[0x05])),
            &[],
        );
        assert_eq!(
            check_signature_from(&child, &dsa_parent),
            Err(String::from(
                "x509: cannot verify signature: algorithm unimplemented"
            ))
        );
        // RSA key with garbage signature fails verification.
        assert_eq!(
            check_signature_from(&child, &rsa_parent),
            Err(String::from("crypto/rsa: verification error"))
        );
    }

    #[test]
    fn check_signature_ed25519_self_signed_round_trip() {
        use ed25519_dalek::Signer as _;
        let signing = ed25519_dalek::SigningKey::from_bytes(&[
            0x9D, 0x61, 0xB1, 0x9D, 0xEF, 0xFD, 0x5A, 0x60, 0xBA, 0x84, 0x4A, 0xF4, 0x92, 0x2E,
            0xC4, 0x44, 0x48, 0xC8, 0x58, 0x07, 0x31, 0x11, 0xED, 0xD3, 0xAD, 0x45, 0x8B, 0x22,
            0x7E, 0x4E, 0x4B, 0x63,
        ]);
        let verifying = signing.verifying_key();
        let spki = spki_with_key(&ai_element(OID_ED25519, None), verifying.as_bytes());
        let ai = ai_element(OID_ED25519, None);
        let mut parts = v3_parts(&[extension(OID_BASIC_CONSTRAINTS, true, &seq(&boolean(true)))]);
        parts[2] = ai.clone();
        parts[6] = spki;
        let tbs = seq(&concat(&parts));
        let sig = signing.sign(&tbs);
        let cert = parse_certificate(&seq(&concat(&[tbs, ai, bitstring(&sig.to_bytes())])))
            .expect("self-signed parses");
        assert_eq!(cert.signature_algorithm, SignatureAlgorithm::Ed25519);
        assert_eq!(cert.public_key_algorithm, PublicKeyAlgorithm::Ed25519);
        assert!(cert.is_ca);
        check_signature_from(&cert, &cert).expect("self-signed verifies");
        // A tampered signature fails.
        let mut bad = sig.to_bytes();
        bad[0] ^= 0x01;
        let mut parts = v3_parts(&[extension(OID_BASIC_CONSTRAINTS, true, &seq(&boolean(true)))]);
        parts[2] = ai_element(OID_ED25519, None);
        parts[6] = spki_with_key(&ai_element(OID_ED25519, None), verifying.as_bytes());
        let tbs = seq(&concat(&parts));
        let tampered = parse_certificate(&seq(&concat(&[
            tbs,
            ai_element(OID_ED25519, None),
            bitstring(&bad),
        ])))
        .expect("tampered parses");
        assert_eq!(
            check_signature_from(&tampered, &cert),
            Err(String::from("x509: Ed25519 verification failure"))
        );
    }
}
