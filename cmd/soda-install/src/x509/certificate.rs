use super::algorithms::PublicKeyAlgorithm;
use super::der::{
    oid_to_string, right_align, Reader, CTX0_CONS, CTX1_PRIM, CTX2_PRIM, CTX3_CONS, TAG_SEQ,
};
use super::extensions::{parse_extension, process_extensions, ProcessedExtensions};
use super::name_constraints::quoted;
use super::names::{parse_ai, parse_name};
use super::public_key::{
    is_negative, magnitude, parse_public_key, public_key_algorithm_from_oid,
    signature_algorithm_from_ai,
};
use super::time::parse_validity;
use super::types::Certificate;

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
