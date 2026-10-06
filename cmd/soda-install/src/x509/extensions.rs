use super::der::{
    bit_at, oid_to_string, Reader, CTX0_CONS, CTX0_PRIM, CTX1_PRIM, CTX6_PRIM, OID_AIA,
    TAG_BOOLEAN, TAG_INTEGER, TAG_OCTET, TAG_OID, TAG_SEQ,
};
use super::name_constraints::{parse_name_constraints, parse_san};

pub(super) struct Extension<'a> {
    pub(super) id: &'a [u8],
    pub(super) critical: bool,
    pub(super) value: &'a [u8],
}

pub(super) fn parse_extension(contents: &[u8]) -> Result<Extension<'_>, String> {
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

pub(super) struct ProcessedExtensions {
    pub(super) key_usage: u16,
    pub(super) is_ca: bool,
    pub(super) basic_constraints_valid: bool,
    pub(super) max_path_len: i64,
    pub(super) max_path_len_zero: bool,
    pub(super) unhandled_critical: Vec<String>,
}

pub(super) fn process_extensions(
    extensions: &[Extension<'_>],
) -> Result<ProcessedExtensions, String> {
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
