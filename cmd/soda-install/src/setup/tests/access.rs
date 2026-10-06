use super::super::*;
use super::fixtures::null_console;

use crate::command::FnRunner;

use crate::enroll::test_support::temp_dir;

fn tlv(tag: u8, contents: &[u8]) -> Vec<u8> {
    let mut out = vec![tag];
    if contents.len() < 128 {
        out.push(contents.len() as u8);
    } else {
        let mut len = contents.len();
        let mut bytes = Vec::new();
        while len > 0 {
            bytes.push((len & 0xff) as u8);
            len >>= 8;
        }
        out.push(0x80 | bytes.len() as u8);
        bytes.reverse();
        out.extend_from_slice(&bytes);
    }
    out.extend_from_slice(contents);
    out
}

fn ca_cert_pem(is_ca: bool) -> Vec<u8> {
    use ed25519_dalek::Signer as _;
    let signing = ed25519_dalek::SigningKey::from_bytes(&[
        0x9D, 0x61, 0xB1, 0x9D, 0xEF, 0xFD, 0x5A, 0x60, 0xBA, 0x84, 0x4A, 0xF4, 0x92, 0x2E, 0xC4,
        0x44, 0x48, 0xC8, 0x58, 0x07, 0x31, 0x11, 0xED, 0xD3, 0xAD, 0x45, 0x8B, 0x22, 0x7E, 0x4E,
        0x4B, 0x63,
    ]);
    let seq_of = |parts: &[Vec<u8>]| {
        let mut contents = Vec::new();
        for part in parts {
            contents.extend_from_slice(part);
        }
        tlv(0x30, &contents)
    };
    let ai = tlv(0x30, &tlv(0x06, &[0x2B, 0x65, 0x70]));
    let validity = seq_of(&[tlv(0x17, b"700101000000Z"), tlv(0x17, b"700102000000Z")]);
    let spki = seq_of(&[
        ai.clone(),
        tlv(
            0x03,
            &[&[0x00], signing.verifying_key().as_bytes().as_slice()].concat(),
        ),
    ]);
    let bc = tlv(0x30, &tlv(0x01, &[if is_ca { 0xFF } else { 0x00 }]));
    let ext = seq_of(&[
        tlv(0x06, &[0x55, 0x1D, 0x13]),
        tlv(0x01, &[0xFF]),
        tlv(0x04, &bc),
    ]);
    let tbs = seq_of(&[
        tlv(0xA0, &tlv(0x02, &[0x02])),
        tlv(0x02, &[0x01]),
        ai.clone(),
        tlv(0x30, &[]),
        validity,
        tlv(0x30, &[]),
        spki,
        tlv(0xA3, &seq_of(&[ext])),
    ]);
    let sig = signing.sign(&tbs);
    let cert = seq_of(&[
        tbs,
        ai,
        tlv(0x03, &[&[0x00], sig.to_bytes().as_slice()].concat()),
    ]);
    let b64 = crate::sshkey::b64_encode(&cert);
    let mut pem = b"-----BEGIN CERTIFICATE-----\n".to_vec();
    for chunk in b64.as_bytes().chunks(64) {
        pem.extend_from_slice(chunk);
        pem.push(b'\n');
    }
    pem.extend_from_slice(b"-----END CERTIFICATE-----\n");
    pem
}

#[test]
fn local_certificate_export_requires_public_ca() {
    let encoded = ca_cert_pem(true);
    let fingerprint = local_ca_fingerprint(&encoded).unwrap();
    assert_eq!(fingerprint.len(), 64);
    assert!(fingerprint.bytes().all(|b| b.is_ascii_hexdigit()));
    assert!(local_ca_fingerprint(&ca_cert_pem(false)).is_err());
    let mut trailing = encoded.clone();
    trailing.extend_from_slice(b"private trailing data");
    assert!(local_ca_fingerprint(&trailing).is_err());
    let key = b"-----BEGIN PRIVATE KEY-----\nAAAA\n-----END PRIVATE KEY-----\n";
    assert!(local_ca_fingerprint(key).is_err());
}

#[test]
fn local_trust_guidance_rejects_untrusted_destination() {
    for origin in [
        "https://8.8.8.8",
        "https://soda.example.test",
        "https://192.168.1.2:444",
        "https://192.168.1.2/path",
        "https://root@192.168.1.2",
        "https://192.168.1.2?argument",
    ] {
        let root = temp_dir();
        std::fs::write(
            format!("{}/dashboard.json", root.path),
            format!(r#"{{"forgejo_url":"{origin}"}}"#),
        )
        .unwrap();
        std::fs::write(format!("{}/proxy.env", root.path), b"SODA_TLS=internal\n").unwrap();
        let run = FnRunner::new(|_, _, _, _| -> Result<Vec<u8>, Error> {
            panic!("invalid origin reached native service inspection")
        });
        let (ctx, _flag) = Ctx::test();
        assert!(
            configured_access(&ctx, &null_console(), &root.path, "missing-ca", &run).is_err(),
            "{origin}"
        );
    }
}

#[test]
fn configured_guidance_reports_inactive_service_without_replay() {
    let root = temp_dir();
    std::fs::write(
        format!("{}/dashboard.json", root.path),
        br#"{"forgejo_url":"https://192.168.1.5"}"#,
    )
    .unwrap();
    std::fs::write(format!("{}/proxy.env", root.path), b"SODA_TLS=internal\n").unwrap();
    let observed = std::cell::RefCell::new(Vec::new());
    let run = FnRunner::new(|_, name: &str, args: &[String], input: Option<&[u8]>| {
        assert_eq!(name, "systemctl");
        assert_eq!(args.len(), 3);
        assert_eq!(args[0], "is-active");
        assert_eq!(args[1], "--quiet");
        assert!(input.is_none(), "configured guidance attempted a mutation");
        observed.borrow_mut().push(args[2].clone());
        if args[2] == "soda-dashboard.service" {
            return Err(Error::msg("synthetic inactive unit"));
        }
        Ok(Vec::new())
    });
    let (ctx, _flag) = Ctx::test();
    let err = configured_access(&ctx, &null_console(), &root.path, "missing-ca", &run).unwrap_err();
    assert!(err.to_string().contains("soda-dashboard.service"), "{err}");
    assert_eq!(observed.borrow().len(), 2);
}
