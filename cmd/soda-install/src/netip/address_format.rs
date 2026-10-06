use super::address::Addr;

impl Addr {
    pub fn is4(&self) -> bool {
        self.is4
    }
    pub fn is6(&self) -> bool {
        !self.is4
    }
    /// Go `Is4In6`: `::ffff:0:0/96` exactly (zone ignored, as in Go).
    pub fn is4_in6(&self) -> bool {
        !self.is4
            && self.bytes[..10].iter().all(|b| *b == 0)
            && self.bytes[10] == 0xff
            && self.bytes[11] == 0xff
    }
    /// Raw zone bytes, empty when absent. Go zones are unvalidated byte
    /// strings and may be invalid UTF-8 after URL unescaping.
    pub fn zone(&self) -> &[u8] {
        self.zone.as_deref().unwrap_or(b"")
    }
    /// Go `IsPrivate`: 4-in-6 unmapped to v4 first, then v4 `10/8`
    /// `172.16/12` `192.168/16`, v6 `fc00::/7`.
    pub fn is_private(&self) -> bool {
        if self.is4 {
            return is_private_v4(&self.bytes[..4]);
        }
        if self.is4_in6() {
            return is_private_v4(&self.bytes[12..16]);
        }
        self.bytes[0] & 0xfe == 0xfc
    }
    /// Go `Addr.String`: canonical dotted quad, or compressed lowercase
    /// IPv6 (longest zero run of length >= 2, first wins ties; 4-in-6 keeps
    /// the dotted tail), plus the raw `%zone` bytes.
    pub fn to_string_go(&self) -> Vec<u8> {
        let mut out = Vec::new();
        if self.is4 {
            push_dot(&mut out, &self.bytes[..4]);
        } else if self.is4_in6() {
            out.extend_from_slice(b"::ffff:");
            push_dot(&mut out, &self.bytes[12..16]);
        } else {
            let mut groups = [0u16; 8];
            for (i, group) in groups.iter_mut().enumerate() {
                *group = u16::from_be_bytes([self.bytes[2 * i], self.bytes[2 * i + 1]]);
            }
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
                    out.extend_from_slice(b"::");
                    first = true;
                    i += best_len;
                    continue;
                }
                if !first {
                    out.push(b':');
                }
                first = false;
                push_hex(&mut out, groups[i]);
                i += 1;
            }
            if out.is_empty() {
                out.extend_from_slice(b"::");
            }
        }
        if let Some(zone) = &self.zone {
            out.push(b'%');
            out.extend_from_slice(zone);
        }
        out
    }
    pub(super) fn bits(&self) -> u32 {
        if self.is4 {
            32
        } else {
            128
        }
    }
    pub(super) fn masked_bytes(&self, bits: u32) -> [u8; 16] {
        let mut out = self.bytes;
        let total = if self.is4 { 32 } else { 128 };
        for i in bits..total {
            let byte = (i / 8) as usize;
            let bit = 7 - (i % 8);
            out[byte] &= !(1 << bit);
        }
        out
    }
}

fn is_private_v4(b: &[u8]) -> bool {
    b[0] == 10 || (b[0] == 172 && b[1] & 0xf0 == 16) || (b[0] == 192 && b[1] == 168)
}

fn push_dot(out: &mut Vec<u8>, b: &[u8]) {
    for (i, octet) in b.iter().enumerate() {
        if i > 0 {
            out.push(b'.');
        }
        out.extend_from_slice(octet.to_string().as_bytes());
    }
}

fn push_hex(out: &mut Vec<u8>, group: u16) {
    out.extend_from_slice(format!("{group:x}").as_bytes());
}
