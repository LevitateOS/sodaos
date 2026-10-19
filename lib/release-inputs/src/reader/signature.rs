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
