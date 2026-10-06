// ---------- wire primitives ----------

pub fn read_string(buf: &[u8]) -> Result<(&[u8], &[u8]), ()> {
    if buf.len() < 4 {
        return Err(());
    }
    let len = u32::from_be_bytes([buf[0], buf[1], buf[2], buf[3]]) as usize;
    if buf.len() < 4 + len {
        return Err(());
    }
    Ok((&buf[4..4 + len], &buf[4 + len..]))
}

pub fn read_u32(buf: &[u8]) -> Result<(u32, &[u8]), ()> {
    if buf.len() < 4 {
        return Err(());
    }
    Ok((
        u32::from_be_bytes([buf[0], buf[1], buf[2], buf[3]]),
        &buf[4..],
    ))
}

pub fn read_u64(buf: &[u8]) -> Result<(u64, &[u8]), ()> {
    if buf.len() < 8 {
        return Err(());
    }
    Ok((u64::from_be_bytes(buf[..8].try_into().unwrap()), &buf[8..]))
}

pub fn put_string(out: &mut Vec<u8>, data: &[u8]) {
    out.extend_from_slice(&(data.len() as u32).to_be_bytes());
    out.extend_from_slice(data);
}

/// Signed mpint: two's-complement parse, minimal magnitude.
#[derive(Debug, Clone, PartialEq, Eq)]
pub struct Mpint {
    pub negative: bool,
    pub mag: Vec<u8>,
}

fn strip_zeros(bytes: &[u8]) -> Vec<u8> {
    let i = bytes.iter().position(|&b| b != 0).unwrap_or(bytes.len());
    bytes[i..].to_vec()
}

pub fn parse_mpint(buf: &[u8]) -> Result<(Mpint, &[u8]), ()> {
    let (raw, rest) = read_string(buf)?;
    if raw.is_empty() {
        return Ok((
            Mpint {
                negative: false,
                mag: Vec::new(),
            },
            rest,
        ));
    }
    if raw[0] & 0x80 != 0 {
        // Negative: two's complement magnitude.
        let mut mag: Vec<u8> = raw.iter().map(|b| !b).collect();
        let mut carry = 1u16;
        for b in mag.iter_mut().rev() {
            let t = *b as u16 + carry;
            *b = t as u8;
            carry = t >> 8;
        }
        Ok((
            Mpint {
                negative: true,
                mag: strip_zeros(&mag),
            },
            rest,
        ))
    } else {
        Ok((
            Mpint {
                negative: false,
                mag: strip_zeros(raw),
            },
            rest,
        ))
    }
}

/// Bit length of the absolute value, like big.Int.BitLen.
pub fn mpint_bitlen(m: &Mpint) -> usize {
    if m.mag.is_empty() {
        return 0;
    }
    (m.mag.len() - 1) * 8 + (8 - m.mag[0].leading_zeros() as usize)
}

/// Signed comparison.
pub fn mpint_cmp(a: &Mpint, b: &Mpint) -> std::cmp::Ordering {
    match (a.negative, b.negative) {
        (true, false) => {
            if a.mag.is_empty() && b.mag.is_empty() {
                return std::cmp::Ordering::Equal;
            }
            // Negative (nonzero) is always less; -0 equals 0, but -0 cannot
            // arise: strip_zeros of all-0xff complement... note -0 parses
            // from [0x80]: complement+1 = [0x80] -> mag [0x80], negative.
            // Go: -0? big.Int can't hold -0; parseInt of [0x80] gives -128.
            // Our (neg,[0x80]) compares by magnitude below correctly.
            if a.mag.is_empty() {
                return std::cmp::Ordering::Equal;
            }
            std::cmp::Ordering::Less
        }
        (false, true) => {
            if b.mag.is_empty() {
                return std::cmp::Ordering::Equal;
            }
            std::cmp::Ordering::Greater
        }
        (false, false) | (true, true) => {
            let ord = mpint_bitlen(a)
                .cmp(&mpint_bitlen(b))
                .then_with(|| a.mag.cmp(&b.mag));
            if a.negative {
                ord.reverse()
            } else {
                ord
            }
        }
    }
}

pub fn mpint_to_i64(m: &Mpint) -> Option<i64> {
    if m.mag.len() > 8 {
        return None;
    }
    let mut v: i64 = 0;
    for &b in &m.mag {
        v = v.checked_mul(256)?.checked_add(b as i64)?;
    }
    if m.negative {
        v.checked_neg()
    } else {
        Some(v)
    }
}

pub fn marshal_mpint(out: &mut Vec<u8>, m: &Mpint) {
    let mut bytes = m.mag.clone();
    if bytes.is_empty() {
        // Zero is the zero-length string.
    } else if !m.negative {
        if bytes[0] & 0x80 != 0 {
            bytes.insert(0, 0x00);
        }
    } else {
        // Two's complement, minimal with the sign bit set.
        let mut carry = 1u16;
        for b in bytes.iter_mut().rev() {
            let t = (*b as u16 ^ 0xff) + carry;
            *b = t as u8;
            carry = t >> 8;
        }
        let mut twos = if carry != 0 { vec![1] } else { Vec::new() };
        twos.extend(bytes);
        // Strip redundant 0xff sign bytes, keeping the sign bit set.
        let mut i = 0;
        while twos.len() - i > 1 && twos[i] == 0xff && twos[i + 1] & 0x80 != 0 {
            i += 1;
        }
        bytes = twos[i..].to_vec();
        debug_assert!(!bytes.is_empty() && bytes[0] & 0x80 != 0);
    }
    put_string(out, &bytes);
}
