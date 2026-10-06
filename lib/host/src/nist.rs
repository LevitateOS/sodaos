//! NIST prime-curve point validation for SSH ECDSA keys.
//!
//! The SSH wire contract accepts only exact-length uncompressed SEC1 points.
//! Typed curve parsers then validate that the coordinates form a point on the
//! named curve and provide its canonical uncompressed encoding.

use p256::elliptic_curve::sec1::ToEncodedPoint;

pub struct Curve {
    pub name: &'static str,
    coord_len: usize,
}

pub const P256: Curve = Curve {
    name: "nistp256",
    coord_len: 32,
};

pub const P384: Curve = Curve {
    name: "nistp384",
    coord_len: 48,
};

pub const P521: Curve = Curve {
    name: "nistp521",
    coord_len: 66,
};

/// Parse an exact-length uncompressed point and return its canonical encoding.
///
/// The shape check must precede each library parser: these parsers also accept
/// compressed SEC1 points, while the Go SSH producer contract admits only
/// uncompressed points.
pub fn decode_point(curve: &Curve, point: &[u8]) -> Option<Vec<u8>> {
    if point.len() != 1 + 2 * curve.coord_len || point.first() != Some(&0x04) {
        return None;
    }

    match curve.name {
        "nistp256" if curve.coord_len == 32 => {
            let key = p256::PublicKey::from_sec1_bytes(point).ok()?;
            Some(key.to_encoded_point(false).as_bytes().to_vec())
        }
        "nistp384" if curve.coord_len == 48 => {
            let key = p384::PublicKey::from_sec1_bytes(point).ok()?;
            Some(key.to_encoded_point(false).as_bytes().to_vec())
        }
        "nistp521" if curve.coord_len == 66 => {
            let key = p521::PublicKey::from_sec1_bytes(point).ok()?;
            Some(key.to_encoded_point(false).as_bytes().to_vec())
        }
        _ => None,
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    fn unhex(hex: &str) -> Vec<u8> {
        let bytes = hex.as_bytes();
        assert!(bytes.len().is_multiple_of(2));
        let value = |c: u8| match c {
            b'0'..=b'9' => c - b'0',
            b'a'..=b'f' => c - b'a' + 10,
            b'A'..=b'F' => c - b'A' + 10,
            _ => panic!("bad hex"),
        };
        bytes
            .chunks_exact(2)
            .map(|pair| value(pair[0]) << 4 | value(pair[1]))
            .collect()
    }

    #[test]
    fn generator_points_parse_and_remain_uncompressed() {
        let vectors = [
            (
                &P256,
                "6b17d1f2e12c4247f8bce6e563a440f277037d812deb33a0f4a13945d898c296",
                "4fe342e2fe1a7f9b8ee7eb4a7c0f9e162bce33576b315ececbb6406837bf51f5",
                "ffffffff00000001000000000000000000000000ffffffffffffffffffffffff",
            ),
            (
                &P384,
                "aa87ca22be8b05378eb1c71ef320ad746e1d3b628ba79b9859f741e082542a385502f25dbf55296c3a545e3872760ab7",
                "3617de4a96262c6f5d9e98bf9292dc29f8f41dbd289a147ce9da3113b5f0b8c00a60b1ce1d7e819d7a431d7c90ea0e5f",
                "fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffeffffffff0000000000000000ffffffff",
            ),
            (
                &P521,
                "c6858e06b70404e9cd9e3ecb662395b4429c648139053fb521f828af606b4d3dbaa14b5e77efe75928fe1dc127a2ffa8de3348b3c1856a429bf97e7e31c2e5bd66",
                "011839296a789a3bc0045c8a5fb42c7d1bd998f54449579b446817afbd17273e662c97ee72995ef42640c550b9013fad0761353c7086a272c24088be94769fd16650",
                "01ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff",
            ),
        ];

        for (curve, x, y, prime_hex) in vectors {
            let mut point = vec![0x04];
            for coordinate in [x, y] {
                let raw = unhex(coordinate);
                point.extend(vec![0; curve.coord_len - raw.len()]);
                point.extend(raw);
            }
            assert_eq!(
                decode_point(curve, &point).as_deref(),
                Some(point.as_slice())
            );

            // Each named curve rejects a point from either other
            // curve, and both compressed and hybrid encodings stay refused.
            for other in [&P256, &P384, &P521] {
                if other.name != curve.name {
                    assert!(decode_point(other, &point).is_none());
                }
            }

            let mut compressed = vec![0x02];
            compressed.extend_from_slice(&point[1..1 + curve.coord_len]);
            assert!(decode_point(curve, &compressed).is_none());

            let mut hybrid = point.clone();
            hybrid[0] = 0x06;
            assert!(decode_point(curve, &hybrid).is_none());
            assert!(decode_point(curve, &point[..point.len() - 1]).is_none());

            let mut overlong = point.clone();
            overlong.push(0);
            assert!(decode_point(curve, &overlong).is_none());
            assert!(decode_point(curve, &[0x00]).is_none());

            let mut off_curve = point.clone();
            off_curve[1] ^= 1;
            assert!(decode_point(curve, &off_curve).is_none());

            let prime = unhex(prime_hex);
            assert_eq!(prime.len(), curve.coord_len);
            for coordinate_start in [1, 1 + curve.coord_len] {
                let mut at_prime = vec![0; 1 + 2 * curve.coord_len];
                at_prime[0] = 0x04;
                at_prime[coordinate_start..coordinate_start + curve.coord_len]
                    .copy_from_slice(&prime);
                assert!(decode_point(curve, &at_prime).is_none());
                at_prime[coordinate_start..coordinate_start + curve.coord_len].fill(0xff);
                assert!(decode_point(curve, &at_prime).is_none());
            }
        }
    }

    #[test]
    fn invalid_and_infinity_points_are_rejected() {
        let mut infinity = vec![0; 65];
        infinity[0] = 0x04;
        assert!(decode_point(&P256, &infinity).is_none());

        let mut off_curve = vec![0; 65];
        off_curve[0] = 0x04;
        off_curve[1] = 1;
        assert!(decode_point(&P256, &off_curve).is_none());
        assert!(decode_point(&P256, &[0]).is_none());
    }
}
