// ---------------------------------------------------------------------------
// DER reader: cryptobyte read-path strictness.
// ---------------------------------------------------------------------------

/// Byte cursor over DER input. Slices returned borrow the original input.
pub(super) struct Reader<'a> {
    b: &'a [u8],
}

impl<'a> Reader<'a> {
    pub(super) fn new(b: &'a [u8]) -> Reader<'a> {
        Reader { b }
    }

    pub(super) fn is_empty(&self) -> bool {
        self.b.is_empty()
    }

    /// Read one TLV: low-tag-number form only, DER-minimal lengths, no
    /// indefinite form, long form capped at four length octets. Returns
    /// `(tag, contents, full_element)`.
    fn read_tlv(&mut self) -> Option<(u8, &'a [u8], &'a [u8])> {
        let input: &'a [u8] = self.b;
        if input.len() < 2 {
            return None;
        }
        let (tag, len_byte) = (input[0], input[1]);
        if tag & 0x1f == 0x1f {
            return None;
        }
        let (content_len, header_len) = if len_byte & 0x80 == 0 {
            (u32::from(len_byte), 2usize)
        } else {
            let len_len = (len_byte & 0x7f) as usize;
            if len_len == 0 || len_len > 4 || input.len() < 2 + len_len {
                return None;
            }
            let mut len32 = 0u32;
            for b in &input[2..2 + len_len] {
                len32 = (len32 << 8) | u32::from(*b);
            }
            if len32 < 128 || input[2] == 0 {
                return None;
            }
            let header_len = 2 + len_len;
            let total = (header_len as u32).checked_add(len32)?;
            if total > input.len() as u32 {
                return None;
            }
            (len32, header_len)
        };
        let total = header_len + content_len as usize;
        if input.len() < total {
            return None;
        }
        self.b = &input[total..];
        Some((tag, &input[header_len..total], &input[..total]))
    }

    pub(super) fn read_asn1(&mut self, tag: u8) -> Option<&'a [u8]> {
        let (got, contents, _) = self.read_tlv()?;
        if got != tag {
            return None;
        }
        Some(contents)
    }

    pub(super) fn read_element(&mut self, tag: u8) -> Option<&'a [u8]> {
        let (got, _, element) = self.read_tlv()?;
        if got != tag {
            return None;
        }
        Some(element)
    }

    pub(super) fn read_any(&mut self) -> Option<(u8, &'a [u8])> {
        let (tag, contents, _) = self.read_tlv()?;
        Some((tag, contents))
    }

    pub(super) fn read_any_element(&mut self) -> Option<(u8, &'a [u8])> {
        let (tag, _, element) = self.read_tlv()?;
        Some((tag, element))
    }

    pub(super) fn peek_tag(&self, tag: u8) -> bool {
        !self.b.is_empty() && self.b[0] == tag
    }

    /// Optional explicitly-tagged element: `None` when present but malformed.
    pub(super) fn read_optional(&mut self, tag: u8) -> Option<Option<&'a [u8]>> {
        if !self.peek_tag(tag) {
            return Some(None);
        }
        self.read_asn1(tag).map(Some)
    }

    pub(super) fn skip_optional(&mut self, tag: u8) -> bool {
        if !self.peek_tag(tag) {
            return true;
        }
        self.read_asn1(tag).is_some()
    }

    pub(super) fn read_boolean(&mut self) -> Option<bool> {
        let bytes = self.read_asn1(0x01)?;
        if bytes.len() != 1 {
            return None;
        }
        match bytes[0] {
            0 => Some(false),
            0xff => Some(true),
            _ => None,
        }
    }

    /// INTEGER contents with minimal-encoding enforcement.
    pub(super) fn read_integer_bytes(&mut self) -> Option<&'a [u8]> {
        let bytes = self.read_asn1(0x02)?;
        if !check_integer(bytes) {
            return None;
        }
        Some(bytes)
    }

    pub(super) fn read_int64(&mut self) -> Option<i64> {
        asn1_signed(self.read_integer_bytes()?)
    }

    pub(super) fn read_uint64(&mut self) -> Option<u64> {
        asn1_unsigned(self.read_integer_bytes()?)
    }

    pub(super) fn read_int64_with_tag(&mut self, tag: u8) -> Option<i64> {
        let bytes = self.read_asn1(tag)?;
        if !check_integer(bytes) {
            return None;
        }
        asn1_signed(bytes)
    }

    /// BIT STRING with padding-bit checks. Returns `(bytes, bit_length)`.
    pub(super) fn read_bitstring(&mut self) -> Option<(&'a [u8], usize)> {
        let bytes = self.read_asn1(TAG_BITSTRING)?;
        if bytes.is_empty() {
            return None;
        }
        let pad = bytes[0];
        let value = &bytes[1..];
        if pad > 7 || (value.is_empty() && pad != 0) {
            return None;
        }
        if !value.is_empty() && value[value.len() - 1] & ((1u16 << pad) - 1) as u8 != 0 {
            return None;
        }
        Some((value, value.len() * 8 - usize::from(pad)))
    }

    /// OBJECT IDENTIFIER contents with base-128 validation.
    pub(super) fn read_oid(&mut self) -> Option<&'a [u8]> {
        let bytes = self.read_asn1(0x06)?;
        if !valid_oid(bytes) {
            return None;
        }
        Some(bytes)
    }
}

fn check_integer(bytes: &[u8]) -> bool {
    if bytes.is_empty() {
        return false;
    }
    if bytes.len() == 1 {
        return true;
    }
    if bytes[0] == 0 && bytes[1] & 0x80 == 0 || bytes[0] == 0xff && bytes[1] & 0x80 == 0x80 {
        return false;
    }
    true
}

pub(super) fn asn1_signed(bytes: &[u8]) -> Option<i64> {
    if bytes.len() > 8 {
        return None;
    }
    let mut v = 0u64;
    for b in bytes {
        v = (v << 8) | u64::from(*b);
    }
    let shift = 64 - bytes.len() * 8;
    Some(((v << shift) as i64) >> shift)
}

fn asn1_unsigned(bytes: &[u8]) -> Option<u64> {
    if bytes.len() > 9 || (bytes.len() == 9 && bytes[0] != 0) {
        return None;
    }
    if bytes[0] & 0x80 != 0 {
        return None;
    }
    let mut v = 0u64;
    for b in bytes {
        v = (v << 8) | u64::from(*b);
    }
    Some(v)
}

fn valid_oid(content: &[u8]) -> bool {
    if content.is_empty() {
        return false;
    }
    let mut i = 0;
    while i < content.len() {
        let mut ret = 0u32;
        let mut j = 0;
        loop {
            if i >= content.len() {
                return false;
            }
            if j == 5 || ret >= (1 << 24) {
                return false;
            }
            ret <<= 7;
            let b = content[i];
            i += 1;
            j += 1;
            if j == 1 && b == 0x80 {
                return false;
            }
            ret |= u32::from(b & 0x7f);
            if b & 0x80 == 0 {
                break;
            }
        }
    }
    true
}

/// Dotted OID string from validated DER contents, as Go's
/// `asn1.ObjectIdentifier.String` renders the parsed arcs.
pub(super) fn oid_to_string(content: &[u8]) -> String {
    let mut values = Vec::new();
    let mut i = 0;
    while i < content.len() {
        let mut v = 0u64;
        loop {
            let b = content[i];
            i += 1;
            v = (v << 7) | u64::from(b & 0x7f);
            if b & 0x80 == 0 {
                break;
            }
        }
        values.push(v);
    }
    let mut out = String::new();
    let first = values[0];
    if first < 80 {
        out.push_str(&format!("{}.{}", first / 40, first % 40));
    } else {
        out.push_str(&format!("2.{}", first - 80));
    }
    for v in values.iter().skip(1) {
        out.push_str(&format!(".{v}"));
    }
    out
}

/// Go `asn1.BitString.RightAlign`: whole bytes pass through; otherwise the
/// padding bits move to the front.
pub(super) fn right_align(bytes: &[u8], bit_len: usize) -> Vec<u8> {
    let shift = 8 - bit_len % 8;
    if shift == 8 || bytes.is_empty() {
        return bytes.to_vec();
    }
    let shift = shift as u32;
    let mut out = vec![0u8; bytes.len()];
    out[0] = bytes[0] >> shift;
    for (i, b) in bytes.iter().enumerate().skip(1) {
        out[i] = bytes[i - 1] << (8 - shift) | (b >> shift);
    }
    out
}

/// Go `asn1.BitString.At`: MSB-first bit test, zero past the end.
pub(super) fn bit_at(bytes: &[u8], bit_len: usize, i: usize) -> u8 {
    if i >= bit_len {
        return 0;
    }
    (bytes[i / 8] >> (7 - (i % 8) as u32)) & 1
}

// ---------------------------------------------------------------------------
// ASN.1 tags and OID constants (DER content octets).
// ---------------------------------------------------------------------------

pub(super) const TAG_SEQ: u8 = 0x30;
pub(super) const TAG_SET: u8 = 0x31;
pub(super) const TAG_OID: u8 = 0x06;
pub(super) const TAG_OCTET: u8 = 0x04;
pub(super) const TAG_BOOLEAN: u8 = 0x01;
pub(super) const TAG_INTEGER: u8 = 0x02;
const TAG_BITSTRING: u8 = 0x03;
pub(super) const TAG_UTCTIME: u8 = 0x17;
pub(super) const TAG_GENTIME: u8 = 0x18;

pub(super) const CTX0_CONS: u8 = 0xa0;
pub(super) const CTX1_CONS: u8 = 0xa1;
pub(super) const CTX2_CONS: u8 = 0xa2;
pub(super) const CTX3_CONS: u8 = 0xa3;
pub(super) const CTX0_PRIM: u8 = 0x80;
pub(super) const CTX1_PRIM: u8 = 0x81;
pub(super) const CTX2_PRIM: u8 = 0x82;
pub(super) const CTX6_PRIM: u8 = 0x86;

pub(super) const OID_MD5_RSA: &[u8] = &[0x2a, 0x86, 0x48, 0x86, 0xf7, 0x0d, 0x01, 0x01, 0x04];
pub(super) const OID_SHA1_RSA: &[u8] = &[0x2a, 0x86, 0x48, 0x86, 0xf7, 0x0d, 0x01, 0x01, 0x05];
pub(super) const OID_SHA256_RSA: &[u8] = &[0x2a, 0x86, 0x48, 0x86, 0xf7, 0x0d, 0x01, 0x01, 0x0b];
pub(super) const OID_SHA384_RSA: &[u8] = &[0x2a, 0x86, 0x48, 0x86, 0xf7, 0x0d, 0x01, 0x01, 0x0c];
pub(super) const OID_SHA512_RSA: &[u8] = &[0x2a, 0x86, 0x48, 0x86, 0xf7, 0x0d, 0x01, 0x01, 0x0d];
pub(super) const OID_RSAPSS: &[u8] = &[0x2a, 0x86, 0x48, 0x86, 0xf7, 0x0d, 0x01, 0x01, 0x0a];
pub(super) const OID_DSA_SHA1: &[u8] = &[0x2a, 0x86, 0x48, 0xce, 0x38, 0x04, 0x03];
pub(super) const OID_DSA_SHA256: &[u8] = &[0x60, 0x86, 0x48, 0x01, 0x65, 0x03, 0x04, 0x03, 0x02];
pub(super) const OID_ECDSA_SHA1: &[u8] = &[0x2a, 0x86, 0x48, 0xce, 0x3d, 0x04, 0x01];
pub(super) const OID_ECDSA_SHA256: &[u8] = &[0x2a, 0x86, 0x48, 0xce, 0x3d, 0x04, 0x03, 0x02];
pub(super) const OID_ECDSA_SHA384: &[u8] = &[0x2a, 0x86, 0x48, 0xce, 0x3d, 0x04, 0x03, 0x03];
pub(super) const OID_ECDSA_SHA512: &[u8] = &[0x2a, 0x86, 0x48, 0xce, 0x3d, 0x04, 0x03, 0x04];
pub(super) const OID_ED25519_SIG: &[u8] = &[0x2b, 0x65, 0x70];
pub(super) const OID_ISO_SHA1_RSA: &[u8] = &[0x2b, 0x0e, 0x03, 0x02, 0x1d];

pub(super) const OID_SHA256: &[u8] = &[0x60, 0x86, 0x48, 0x01, 0x65, 0x03, 0x04, 0x02, 0x01];
pub(super) const OID_SHA384: &[u8] = &[0x60, 0x86, 0x48, 0x01, 0x65, 0x03, 0x04, 0x02, 0x02];
pub(super) const OID_SHA512: &[u8] = &[0x60, 0x86, 0x48, 0x01, 0x65, 0x03, 0x04, 0x02, 0x03];
pub(super) const OID_MGF1: &[u8] = &[0x2a, 0x86, 0x48, 0x86, 0xf7, 0x0d, 0x01, 0x01, 0x08];

pub(super) const OID_PUB_RSA: &[u8] = &[0x2a, 0x86, 0x48, 0x86, 0xf7, 0x0d, 0x01, 0x01, 0x01];
pub(super) const OID_PUB_DSA: &[u8] = &[0x2a, 0x86, 0x48, 0xce, 0x38, 0x04, 0x01];
pub(super) const OID_PUB_ECDSA: &[u8] = &[0x2a, 0x86, 0x48, 0xce, 0x3d, 0x02, 0x01];
pub(super) const OID_PUB_ED25519: &[u8] = &[0x2b, 0x65, 0x70];

pub(super) const OID_CURVE_P224: &[u8] = &[0x2b, 0x81, 0x04, 0x00, 0x21];
pub(super) const OID_CURVE_P256: &[u8] = &[0x2a, 0x86, 0x48, 0xce, 0x3d, 0x03, 0x01, 0x07];
pub(super) const OID_CURVE_P384: &[u8] = &[0x2b, 0x81, 0x04, 0x00, 0x22];
pub(super) const OID_CURVE_P521: &[u8] = &[0x2b, 0x81, 0x04, 0x00, 0x23];

pub(super) const OID_AIA: &[u8] = &[0x2b, 0x06, 0x01, 0x05, 0x05, 0x07, 0x01, 0x01];
