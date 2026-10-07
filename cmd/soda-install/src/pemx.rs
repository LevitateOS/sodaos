//! Strict one-certificate PEM framing for the installer's local CA input.

use base64::Engine as _;

const BEGIN: &[u8] = b"-----BEGIN CERTIFICATE-----";
const END: &[u8] = b"-----END CERTIFICATE-----";
const MAX_INPUT: usize = 16 * 1024;

pub fn decode_certificate(input: &[u8]) -> Result<Vec<u8>, ()> {
    if input.len() > MAX_INPUT {
        return Err(());
    }
    let start = input
        .iter()
        .position(|byte| !byte.is_ascii_whitespace())
        .ok_or(())?;
    let end = input
        .iter()
        .rposition(|byte| !byte.is_ascii_whitespace())
        .ok_or(())?
        + 1;
    let input = &input[start..end];

    let mut lines = input.split(|byte| *byte == b'\n');
    let mut first = true;
    let mut ended = false;
    let mut body = Vec::new();
    while let Some(mut line) = lines.next() {
        if line.last() == Some(&b'\r') {
            line = &line[..line.len() - 1];
        }
        if first {
            if line != BEGIN {
                return Err(());
            }
            first = false;
            continue;
        }
        if ended {
            if !line.iter().all(u8::is_ascii_whitespace) {
                return Err(());
            }
            continue;
        }
        if line == END {
            ended = true;
            continue;
        }
        if !line.iter().all(|byte| {
            byte.is_ascii_alphanumeric() || matches!(byte, b'+' | b'/' | b'=' | b' ' | b'\t')
        }) {
            return Err(());
        }
        body.extend(
            line.iter()
                .copied()
                .filter(|byte| !matches!(byte, b' ' | b'\t')),
        );
    }
    if first || !ended || body.is_empty() {
        return Err(());
    }
    base64::engine::general_purpose::STANDARD
        .decode(body)
        .map_err(|_| ())
}

#[cfg(test)]
mod tests {
    use super::*;

    fn pem(label: &str, body: &str) -> Vec<u8> {
        format!("-----BEGIN {label}-----\n{body}\n-----END {label}-----\n").into_bytes()
    }

    #[test]
    fn accepts_one_public_certificate_and_refuses_other_content() {
        assert_eq!(
            decode_certificate(&pem("CERTIFICATE", "AQID")).unwrap(),
            [1, 2, 3]
        );
        assert_eq!(
            decode_certificate(
                b"-----BEGIN CERTIFICATE-----\nA Q\tI D\n-----END CERTIFICATE-----\n"
            )
            .unwrap(),
            [1, 2, 3]
        );
        assert_eq!(
            decode_certificate(
                b" \r\n-----BEGIN CERTIFICATE-----\nAQID\n-----END CERTIFICATE-----\n\t "
            )
            .unwrap(),
            [1, 2, 3]
        );
        assert!(decode_certificate(&pem("PRIVATE KEY", "AQID")).is_err());
        assert!(decode_certificate(
            b"prefix\n-----BEGIN CERTIFICATE-----\nAQID\n-----END CERTIFICATE-----\n"
        )
        .is_err());
        assert!(decode_certificate(b"-----BEGIN CERTIFICATE-----\nAQID\n-----END CERTIFICATE-----\n-----BEGIN CERTIFICATE-----\nAQID\n-----END CERTIFICATE-----\n").is_err());
        assert!(decode_certificate(
            b"-----BEGIN CERTIFICATE-----\nAQID\n-----END CERTIFICATE-----\ntrailing\n"
        )
        .is_err());
        assert!(decode_certificate(
            b"-----BEGIN CERTIFICATE-----\nAQI=\n-----END CERTIFICATE-----\n"
        )
        .is_ok());
        assert!(decode_certificate(
            b"-----BEGIN CERTIFICATE-----\nAQI\n-----END CERTIFICATE-----\n"
        )
        .is_err());
    }
}
