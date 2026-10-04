//! NIST prime-curve point validation for SSH ECDSA keys.
//!
//! Mirrors Go `elliptic.Unmarshal` acceptance (exact uncompressed length,
//! `0x04` prefix, coordinates below the prime, point on the curve) for
//! P-256, P-384 and P-521. Only the curve equation is evaluated — no
//! scalar arithmetic — using small generic limb routines. Curve constants
//! were transcribed from Go's `elliptic` params and are pinned by the
//! generator-point tests below: a wrong constant fails them.

/// Decode a hex string (whitespace-free, even length) into bytes.
fn unhex(hex: &str) -> Vec<u8> {
    let b = hex.as_bytes();
    assert!(b.len() % 2 == 0);
    let val = |c: u8| match c {
        b'0'..=b'9' => c - b'0',
        b'a'..=b'f' => c - b'a' + 10,
        b'A'..=b'F' => c - b'A' + 10,
        _ => panic!("bad hex"),
    };
    b.chunks_exact(2)
        .map(|c| val(c[0]) << 4 | val(c[1]))
        .collect()
}

fn bytes_to_limbs<const N: usize>(be: &[u8]) -> [u64; N] {
    let mut limbs = [0u64; N];
    for (i, chunk) in be.rchunks(8).enumerate() {
        let mut word = [0u8; 8];
        word[8 - chunk.len()..].copy_from_slice(chunk);
        limbs[i] = u64::from_be_bytes(word);
    }
    limbs
}

fn cmp_limbs<const N: usize>(a: &[u64; N], b: &[u64; N]) -> std::cmp::Ordering {
    for i in (0..N).rev() {
        if a[i] != b[i] {
            return a[i].cmp(&b[i]);
        }
    }
    std::cmp::Ordering::Equal
}

/// Schoolbook multiply: N limbs by N limbs into 2N limbs, with full
/// carry propagation (the row carry can exceed one limb).
fn mul_limbs<const N: usize>(a: &[u64; N], b: &[u64; N]) -> Vec<u64> {
    let mut out = vec![0u64; 2 * N];
    for i in 0..N {
        let mut carry = 0u128;
        for j in 0..N {
            let t = out[i + j] as u128 + a[i] as u128 * b[j] as u128 + carry;
            out[i + j] = t as u64;
            carry = t >> 64;
        }
        let mut k = i + N;
        while carry != 0 {
            debug_assert!(k < 2 * N, "product exceeds 2N limbs");
            let t = out[k] as u128 + (carry & 0xffff_ffff_ffff_ffff);
            out[k] = t as u64;
            carry = (carry >> 64) + (t >> 64);
            k += 1;
        }
    }
    out
}

fn shl1(v: &mut [u64]) {
    let mut carry = 0u64;
    for limb in v.iter_mut() {
        let next = *limb >> 63;
        *limb = (*limb << 1) | carry;
        carry = next;
    }
}

/// Subtract N limbs from an (N+1)-limb accumulator, tracking the top limb.
fn sub_full<const N: usize>(a: &mut [u64], b: &[u64; N]) {
    debug_assert_eq!(a.len(), N + 1);
    let mut borrow = 0u64;
    for i in 0..N {
        let (t, b1) = a[i].overflowing_sub(b[i]);
        let (t, b2) = t.overflowing_sub(borrow);
        a[i] = t;
        borrow = (b1 as u64) | (b2 as u64);
    }
    a[N] = a[N].wrapping_sub(borrow);
}

/// Reduce a 2N-limb value modulo an N-limb modulus (binary long division).
fn reduce<const N: usize>(num: &[u64], modulus: &[u64; N]) -> [u64; N] {
    assert_eq!(num.len(), 2 * N);
    let mut rem = vec![0u64; N + 1];
    for i in (0..2 * N * 64).rev() {
        shl1(&mut rem);
        rem[0] |= (num[i / 64] >> (i % 64)) & 1;
        // Compare rem (N+1 limbs) with modulus (N limbs).
        let low: [u64; N] = rem[..N].try_into().unwrap();
        if rem[N] != 0 || cmp_limbs::<N>(&low, modulus) != std::cmp::Ordering::Less {
            sub_full::<N>(&mut rem, modulus);
        }
    }
    debug_assert_eq!(rem[N], 0);
    rem[..N].try_into().unwrap()
}

fn add_mod<const N: usize>(a: &[u64; N], b: &[u64; N], p: &[u64; N]) -> [u64; N] {
    let mut wide = vec![0u64; N + 1];
    let mut carry = 0u128;
    for i in 0..N {
        let t = a[i] as u128 + b[i] as u128 + carry;
        wide[i] = t as u64;
        carry = t >> 64;
    }
    wide[N] = carry as u64;
    let low: [u64; N] = wide[..N].try_into().unwrap();
    if wide[N] != 0 || cmp_limbs::<N>(&low, p) != std::cmp::Ordering::Less {
        sub_full::<N>(&mut wide, p);
    }
    wide[..N].try_into().unwrap()
}

fn sub_mod<const N: usize>(a: &[u64; N], b: &[u64; N], p: &[u64; N]) -> [u64; N] {
    // (a + p - b): always in range, at most one conditional subtract.
    let mut wide = vec![0u64; N + 1];
    let mut carry = 0u128;
    for i in 0..N {
        let t = a[i] as u128 + p[i] as u128 + carry;
        wide[i] = t as u64;
        carry = t >> 64;
    }
    wide[N] = carry as u64;
    sub_full::<N>(&mut wide, b);
    // Result is a + p - b, which is < 2p; reduce once if needed.
    let low: [u64; N] = wide[..N].try_into().unwrap();
    if wide[N] != 0 || cmp_limbs::<N>(&low, p) != std::cmp::Ordering::Less {
        sub_full::<N>(&mut wide, p);
    }
    debug_assert_eq!(wide[N], 0);
    wide[..N].try_into().unwrap()
}

fn mul_mod<const N: usize>(a: &[u64; N], b: &[u64; N], p: &[u64; N]) -> [u64; N] {
    reduce::<N>(&mul_limbs(a, b), p)
}

pub struct Curve {
    pub name: &'static str,
    pub coord_len: usize,
    p_hex: &'static str,
    b_hex: &'static str,
}

pub const P256: Curve = Curve {
    name: "nistp256",
    coord_len: 32,
    p_hex: "ffffffff00000001000000000000000000000000ffffffffffffffffffffffff",
    b_hex: "5ac635d8aa3a93e7b3ebbd55769886bc651d06b0cc53b0f63bce3c3e27d2604b",
};

pub const P384: Curve = Curve {
    name: "nistp384",
    coord_len: 48,
    p_hex: "fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffeffffffff0000000000000000ffffffff",
    b_hex: "b3312fa7e23ee7e4988e056be3f82d19181d9c6efe8141120314088f5013875ac656398d8a2ed19d2a85c8edd3ec2aef",
};

pub const P521: Curve = Curve {
    name: "nistp521",
    coord_len: 66,
    p_hex: "01ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff",
    b_hex: "51953eb9618e1c9a1f929a21a0b68540eea2da725b99b315f3b8b489918ef109e156193951ec7e937b1652c0bd3bb1bf073573df883d2c34f1ef451fd46b503f00",
};

/// Validate an uncompressed point exactly like `elliptic.Unmarshal`:
/// `0x04` prefix, exact length, coordinates below p, on the curve
/// `y^2 = x^3 - 3x + b`.
pub fn valid_point(curve: &Curve, point: &[u8]) -> bool {
    if point.len() != 1 + 2 * curve.coord_len || point[0] != 0x04 {
        return false;
    }
    let (x_bytes, y_bytes) = point[1..].split_at(curve.coord_len);
    match curve.coord_len {
        32 => valid_point_n::<4>(curve, x_bytes, y_bytes),
        48 => valid_point_n::<6>(curve, x_bytes, y_bytes),
        66 => valid_point_n::<9>(curve, x_bytes, y_bytes),
        _ => false,
    }
}

fn valid_point_n<const N: usize>(curve: &Curve, x_bytes: &[u8], y_bytes: &[u8]) -> bool {
    let p = bytes_to_limbs::<N>(&unhex(curve.p_hex));
    let b = bytes_to_limbs::<N>(&unhex(curve.b_hex));
    let x = bytes_to_limbs::<N>(x_bytes);
    let y = bytes_to_limbs::<N>(y_bytes);
    if cmp_limbs(&x, &p) != std::cmp::Ordering::Less
        || cmp_limbs(&y, &p) != std::cmp::Ordering::Less
    {
        return false;
    }
    // y^2 == x^3 - 3x + b (mod p)
    let x2 = mul_mod::<N>(&x, &x, &p);
    let x3 = mul_mod::<N>(&x2, &x, &p);
    let mut three = [0u64; N];
    three[0] = 3;
    let tx = mul_mod::<N>(&three, &x, &p);
    let rhs = add_mod::<N>(&sub_mod::<N>(&x3, &tx, &p), &b, &p);
    mul_mod::<N>(&y, &y, &p) == rhs
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn constants_have_exact_lengths() {
        assert_eq!(unhex(P256.p_hex).len(), 32);
        assert_eq!(unhex(P256.b_hex).len(), 32);
        assert_eq!(unhex(P384.p_hex).len(), 48);
        assert_eq!(unhex(P384.b_hex).len(), 48);
        assert_eq!(unhex(P521.p_hex).len(), 66);
        // b < 2^520: Go's big.Int bytes strip the leading zero.
        assert_eq!(unhex(P521.b_hex).len(), 65);
    }

    #[test]
    fn generators_are_on_curve() {
        // Generator coordinates from Go's elliptic params; they satisfy the
        // equation only if p, b and the points are all transcribed exactly.
        let vectors = [
            (
                &P256,
                "6b17d1f2e12c4247f8bce6e563a440f277037d812deb33a0f4a13945d898c296",
                "4fe342e2fe1a7f9b8ee7eb4a7c0f9e162bce33576b315ececbb6406837bf51f5",
            ),
            (
                &P384,
                "aa87ca22be8b05378eb1c71ef320ad746e1d3b628ba79b9859f741e082542a385502f25dbf55296c3a545e3872760ab7",
                "3617de4a96262c6f5d9e98bf9292dc29f8f41dbd289a147ce9da3113b5f0b8c00a60b1ce1d7e819d7a431d7c90ea0e5f",
            ),
            (
                &P521,
                "c6858e06b70404e9cd9e3ecb662395b4429c648139053fb521f828af606b4d3dbaa14b5e77efe75928fe1dc127a2ffa8de3348b3c1856a429bf97e7e31c2e5bd66",
                "011839296a789a3bc0045c8a5fb42c7d1bd998f54449579b446817afbd17273e662c97ee72995ef42640c550b9013fad0761353c7086a272c24088be94769fd16650",
            ),
        ];
        for (curve, gx, gy) in vectors {
            let mut pt = vec![0x04];
            for coord in [gx, gy] {
                let raw = unhex(coord);
                // Coordinates are fixed-width big-endian; pad short values.
                pt.extend(vec![0u8; curve.coord_len - raw.len()]);
                pt.extend(raw);
            }
            assert!(valid_point(curve, &pt), "generator of {}", curve.name);
            // Flipped bytes leave the curve.
            for flip_at in [1, curve.coord_len, 2 * curve.coord_len] {
                let mut bad = pt.clone();
                bad[flip_at] ^= 0x01;
                assert!(
                    !valid_point(curve, &bad),
                    "flip at {flip_at} on {}",
                    curve.name
                );
            }
        }
    }

    #[test]
    fn malformed_points_rejected() {
        let mut good = vec![0x04];
        good.extend(vec![0u8; 64]);
        // Wrong shapes.
        assert!(!valid_point(&P256, &[]));
        assert!(!valid_point(&P256, &[0x00]));
        assert!(!valid_point(&P256, &good[..64]));
        assert!(!valid_point(&P256, &[good.clone(), vec![0]].concat()));
        let mut bad = good.clone();
        bad[0] = 0x03;
        assert!(!valid_point(&P256, &bad));
        // Coordinates at/above p.
        let mut big = vec![0x04];
        big.extend(vec![0xff; 64]);
        assert!(!valid_point(&P256, &big));
    }
}
