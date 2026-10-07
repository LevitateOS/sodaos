//! Strict one-certificate PEM framing for the installer's local CA input.

const BEGIN: &[u8] = b"-----BEGIN CERTIFICATE-----";
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
    if !input.starts_with(BEGIN) {
        return Err(());
    }
    let (label, der) = x509_cert::der::pem::decode_vec(input).map_err(|_| ())?;
    if label != "CERTIFICATE" || der.is_empty() {
        return Err(());
    }
    Ok(der)
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
        assert!(decode_certificate(
            b"-----BEGIN CERTIFICATE-----\nA Q\tI D\n-----END CERTIFICATE-----\n"
        )
        .is_err());
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
        assert!(
            decode_certificate(b"-----BEGIN CERTIFICATE-----\n\n-----END CERTIFICATE-----\n")
                .is_err()
        );
        assert!(decode_certificate(&vec![b' '; MAX_INPUT + 1]).is_err());
    }
}
