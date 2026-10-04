//! GnuPG status-line signature check (`coreos.go validSignature`).
//!
//! Parses `gpgv --status-fd` output: any failure token refuses the image,
//! and at least one `VALIDSIG` for the selected signer must be present.

const GNUPG_FAILURES: &[&str] = &[
    "BADSIG",
    "ERRSIG",
    "EXPSIG",
    "EXPKEYSIG",
    "REVKEYSIG",
    "KEYEXPIRED",
    "SIGEXPIRED",
    "NO_PUBKEY",
    "NODATA",
];

fn valid_sig_match(fields: &[&str], signer: &str) -> bool {
    if fields.len() != 11 && fields.len() != 12 {
        return false;
    }
    if fields[1] != "VALIDSIG" {
        return false;
    }
    if fields[2].eq_ignore_ascii_case(signer) {
        return true;
    }
    fields.len() == 12 && fields[11].eq_ignore_ascii_case(signer)
}

pub fn valid_signature(status: &[u8], signer: &str) -> bool {
    let text = String::from_utf8_lossy(status);
    let mut valid = false;
    for line in text.split('\n') {
        let fields: Vec<&str> = line.split_whitespace().collect();
        if fields.len() < 2 || fields[0] != "[GNUPG:]" {
            continue;
        }
        if GNUPG_FAILURES.contains(&fields[1]) {
            return false;
        }
        if valid_sig_match(&fields, signer) {
            valid = true;
        }
    }
    valid
}

#[cfg(test)]
mod tests {
    use super::*;

    const SIGNER: &str = "abcdef1234567890ABCDEF1234567890ABCDEF12";

    #[test]
    fn accepts_matching_validsig() {
        let status = "[GNUPG:] GOODSIG ABCDEF1234567890 signer\n\
             [GNUPG:] VALIDSIG abcdef1234567890abcdef1234567890abcdef12 a b c d e f g h\n";
        assert!(valid_signature(status.as_bytes(), SIGNER));
    }

    #[test]
    fn accepts_subkey_validsig_in_long_form() {
        let status = "[GNUPG:] VALIDSIG 0000000000000000000000000000000000000000 a b c d e f g h ABCDEF1234567890ABCDEF1234567890ABCDEF12\n";
        assert!(valid_signature(status.as_bytes(), SIGNER));
    }

    #[test]
    fn rejects_failures_and_mismatches() {
        let good = "[GNUPG:] VALIDSIG ABCDEF1234567890ABCDEF1234567890ABCDEF12 a b c d e f g h\n";
        assert!(valid_signature(good.as_bytes(), SIGNER));
        let bad = "[GNUPG:] BADSIG ABCDEF1234567890 signer\n".to_string() + good;
        assert!(!valid_signature(bad.as_bytes(), SIGNER));
        assert!(!valid_signature(
            good.as_bytes(),
            "ffffffffffffffffffffffffffffffffffffffff"
        ));
        assert!(!valid_signature(b"noise without status lines\n", SIGNER));
        assert!(!valid_signature(b"", SIGNER));
    }
}
