use super::der::{Reader, TAG_SEQ, TAG_SET};

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
pub(super) fn parse_name(element: &[u8]) -> Result<(), String> {
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
pub(super) struct AlgorithmIdentifier<'a> {
    pub(super) oid: &'a [u8],
    pub(super) params: &'a [u8],
}

pub(super) fn parse_ai(contents: &[u8]) -> Result<AlgorithmIdentifier<'_>, String> {
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
