//! Private browser setup: address selection, the Forgejo operator token
//! flow through the native setup/activation binaries, and read-only
//! configured-access guidance.

use crate::command::Runner;
use crate::console::Console;
use crate::errors::Error;
use crate::signal::Ctx;

use self::access::configured_access;
use self::address::setup_addresses;
pub use self::address::{private_setup_origin, SetupAddress};
use self::configure::{
    configure_private_install, execute_setup_and_activation, valid_operator_token,
};
use self::local_ca::local_ca_fingerprint;

const LOCAL_CA_PATH: &str = "/var/lib/soda/proxy/caddy/pki/authorities/local/root.crt";
/// `platform.Sbin`: native Soda binaries live here.
const SBIN: &str = "/usr/bin";

// configureInstall uses the existing native Forgejo installer and setup command.
// Soda does not create a second account/password authority or expose the unfinished
// Forgejo installer. It runs in any interactive operator terminal, local or SSH,
// so the token can be typed or pasted; no SSH session is required.
pub fn configure_install(ctx: &Ctx, console: &Console, run: &dyn Runner) -> Result<(), Error> {
    configure_private_install(ctx, console, run, "/etc/soda", "/run", LOCAL_CA_PATH)
}

mod access;
mod address;
mod configure;
mod local_ca;

#[cfg(test)]
mod tests {
    use super::*;
    use crate::command::FnRunner;
    use crate::enroll::test_support::{drain_available, open_test_pty, temp_dir, ENV_LOCK};

    #[test]
    fn private_setup_addresses() {
        for (address, expected) in [
            ("192.168.2.100", "https://192.168.2.100"),
            ("100.90.1.2", "https://100.90.1.2"),
            ("fd00::123", "https://[fd00::123]"),
        ] {
            assert_eq!(
                private_setup_origin(address).unwrap(),
                expected,
                "{address:?}"
            );
        }
        for address in [
            "127.0.0.1",
            "::1",
            "0.0.0.0",
            "8.8.8.8",
            "169.254.1.2",
            "fe80::1%eth0",
            "192.168.1.2;reboot",
            "soda.example.test",
            "::ffff:192.168.1.2",
        ] {
            assert!(private_setup_origin(address).is_err(), "{address:?}");
        }
        let data = br#"[
		{"ifname":"eth0","flags":["UP"],"addr_info":[{"local":"192.168.1.5","scope":"global"},{"local":"8.8.8.8","scope":"global"}]},
		{"ifname":"eth1","flags":[],"addr_info":[{"local":"10.0.0.2","scope":"global"}]},
		{"ifname":"soda0","flags":["UP"],"addr_info":[{"local":"10.89.0.1","scope":"global"}]},
		{"ifname":"lo","flags":["UP"],"addr_info":[{"local":"127.0.0.1","scope":"host"}]}
	]"#;
        let addresses = setup_addresses(data).unwrap();
        assert_eq!(
            addresses,
            vec![SetupAddress {
                interface: "eth0".to_string(),
                address: "192.168.1.5".to_string()
            }]
        );
        for data in ["not json", "[]"] {
            assert!(setup_addresses(data.as_bytes()).is_err(), "{data:?}");
        }
    }

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
            0x9D, 0x61, 0xB1, 0x9D, 0xEF, 0xFD, 0x5A, 0x60, 0xBA, 0x84, 0x4A, 0xF4, 0x92, 0x2E,
            0xC4, 0x44, 0x48, 0xC8, 0x58, 0x07, 0x31, 0x11, 0xED, 0xD3, 0xAD, 0x45, 0x8B, 0x22,
            0x7E, 0x4E, 0x4B, 0x63,
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

    fn null_console() -> Console {
        Console::from_file("/dev/null", std::fs::File::open("/dev/null").unwrap())
    }

    #[test]
    fn private_setup_does_not_replay_existing_state() {
        for state in ["dashboard.json", "setup-started", "activated"] {
            let root = temp_dir();
            std::fs::write(format!("{}/installed", root.path), b"").unwrap();
            std::fs::write(format!("{}/{}", root.path, state), b"").unwrap();
            let run = FnRunner::new(|_, _, _, _| -> Result<Vec<u8>, Error> {
                panic!("existing state caused a native command or another setup attempt")
            });
            let (ctx, _flag) = Ctx::test();
            // Empty existing config fails read-only, without repeating setup.
            assert!(
                configure_private_install(
                    &ctx,
                    &null_console(),
                    &run,
                    &root.path,
                    &root.path,
                    "missing-ca"
                )
                .is_err(),
                "{state}"
            );
        }
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
    fn configure_runs_without_laptop_terminal() {
        let _guard = ENV_LOCK.lock().unwrap();
        std::env::set_var("SSH_CONNECTION", "");
        std::env::set_var("SSH_TTY", "");
        let run = FnRunner::new(|_, _, _, _| -> Result<Vec<u8>, Error> {
            panic!("local configure reached native operations")
        });
        let (ctx, _flag) = Ctx::test();
        let err = configure_install(&ctx, &null_console(), &run).unwrap_err();
        assert!(
            err.to_string()
                .contains("install the included Soda components"),
            "{err}"
        );
        std::env::remove_var("SSH_CONNECTION");
        std::env::remove_var("SSH_TTY");
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
        let err =
            configured_access(&ctx, &null_console(), &root.path, "missing-ca", &run).unwrap_err();
        assert!(err.to_string().contains("soda-dashboard.service"), "{err}");
        assert_eq!(observed.borrow().len(), 2);
    }

    #[test]
    fn setup_rollback_keeps_referenced_state() {
        let fail = FnRunner::new(|_, _, _, _| -> Result<Vec<u8>, Error> {
            Err(Error::msg("synthetic setup failure"))
        });
        let selected = SetupAddress {
            interface: "eth0".to_string(),
            address: "192.168.1.5".to_string(),
        };
        let (ctx, _flag) = Ctx::test();
        // Referenced configuration preserved.
        let root = temp_dir();
        std::fs::write(
            format!("{}/dashboard.json", root.path),
            br#"{"operator":"kept"}"#,
        )
        .unwrap();
        std::fs::write(format!("{}/grant-key", root.path), b"preexisting-key").unwrap();
        let work = temp_dir();
        let err = execute_setup_and_activation(
            &ctx,
            &fail,
            &root.path,
            &work.path,
            "https://192.168.1.5",
            b"bootstrap-token",
            &selected,
        )
        .unwrap_err();
        assert!(
            err.to_string()
                .contains("inspect the existing configuration"),
            "{err}"
        );
        assert_eq!(
            std::fs::read(format!("{}/dashboard.json", root.path)).unwrap(),
            br#"{"operator":"kept"}"#
        );
        assert_eq!(
            std::fs::read(format!("{}/grant-key", root.path)).unwrap(),
            b"preexisting-key"
        );
        // Orphan reservation cleaned.
        let root = temp_dir();
        std::fs::write(format!("{}/grant-key", root.path), b"orphan-key").unwrap();
        let work = temp_dir();
        let err = execute_setup_and_activation(
            &ctx,
            &fail,
            &root.path,
            &work.path,
            "https://192.168.1.5",
            b"bootstrap-token",
            &selected,
        )
        .unwrap_err();
        assert!(
            err.to_string().contains("rerunning configure is safe"),
            "{err}"
        );
        for name in ["grant-key", "setup-started"] {
            assert!(
                matches!(
                    std::fs::symlink_metadata(format!("{}/{}", root.path, name)),
                    Err(err) if err.kind() == std::io::ErrorKind::NotFound
                ),
                "{name} survived rollback"
            );
        }
    }

    #[test]
    fn operator_token_validation() {
        assert!(valid_operator_token(b"synthetic-operator-token"));
        for bad in [
            "",
            " padded",
            "padded ",
            "a\nb",
            "a\rb",
            "a\0b",
            "\u{a0}nbsp\u{a0}",
        ] {
            assert!(!valid_operator_token(bad.as_bytes()), "{bad:?}");
        }
    }

    fn read_until(master: &mut std::fs::File, transcript: &mut Vec<u8>, ctx: &Ctx, needle: &str) {
        use std::io::Read;
        use std::os::unix::io::AsRawFd;
        loop {
            if String::from_utf8_lossy(transcript).contains(needle) {
                return;
            }
            if ctx.err().is_some() {
                panic!("missing setup prompt: {needle}");
            }
            let mut fd = libc::pollfd {
                fd: master.as_raw_fd(),
                events: libc::POLLIN,
                revents: 0,
            };
            if unsafe { libc::poll(&mut fd, 1, 100) } > 0 && fd.revents & libc::POLLIN != 0 {
                let mut data = [0u8; 4096];
                if let Ok(n) = master.read(&mut data) {
                    transcript.extend_from_slice(&data[..n]);
                }
            }
        }
    }

    #[test]
    fn private_setup_keeps_credential_out_of_commands_and_transcript() {
        use std::io::Write;
        use std::os::unix::fs::PermissionsExt;
        for cancel_setup in [false, true] {
            let root = temp_dir();
            std::fs::write(format!("{}/installed", root.path), b"").unwrap();
            let pty = open_test_pty();
            let slave_path = pty.slave_path.clone();
            let mut master = pty.master;
            let (ctx, _flag) = Ctx::test();
            let ctx =
                ctx.with_timeout(std::time::Instant::now() + std::time::Duration::from_secs(15));
            const TOKEN: &str = "synthetic-operator-token";
            let mutations = std::sync::Arc::new(std::sync::atomic::AtomicUsize::new(0));
            let worker_mutations = mutations.clone();
            let root_path = root.path.clone();
            let worker_ctx = ctx.clone();
            let worker = std::thread::spawn(move || {
                let console = Console::open(&slave_path).unwrap();
                let run = FnRunner::new(|_, name: &str, args: &[String], input: Option<&[u8]>| {
                    assert!(input.is_none(), "credential reached unexpected stdin");
                    assert!(
                        !args.iter().any(|a| a.contains(TOKEN)),
                        "credential reached command arguments"
                    );
                    if name == "ip" {
                        return Ok(br#"[{"ifname":"eth0","flags":["UP"],"addr_info":[{"local":"192.168.1.5","scope":"global"}]}]"#.to_vec());
                    }
                    if name == "systemctl" {
                        assert_eq!(args.len(), 3, "setup guidance changed service state");
                        assert_eq!(args[0], "is-active");
                        assert_eq!(args[1], "--quiet");
                        return Ok(Vec::new());
                    }
                    worker_mutations.fetch_add(1, std::sync::atomic::Ordering::SeqCst);
                    if name == "/usr/bin/soda-setup" {
                        assert_eq!(args.len(), 6, "unexpected setup arguments: {args:?}");
                        assert_eq!(args[0], "--forgejo-url");
                        assert_eq!(args[1], "https://192.168.1.5");
                        assert_eq!(args[2], "--token-file");
                        assert_eq!(args[4], "--out");
                        let data = std::fs::read(&args[3]).unwrap();
                        let mode = std::fs::symlink_metadata(&args[3])
                            .unwrap()
                            .permissions()
                            .mode()
                            & 0o777;
                        assert_eq!(
                            data,
                            format!("{TOKEN}\n").as_bytes(),
                            "token was not passed in a restricted file"
                        );
                        assert_eq!(mode, 0o600, "token was not passed in a restricted file");
                        let config = format!(r#"{{"forgejo_url":"{}"}}"#, args[1]);
                        std::fs::write(&args[5], config).unwrap();
                        return Ok(Vec::new());
                    }
                    assert_eq!(
                        name, "/usr/bin/soda-activate",
                        "unexpected activation: {name} {args:?}"
                    );
                    assert_eq!(args.join(" "), "--bind-ip 192.168.1.5 --local-tls");
                    std::fs::write(format!("{}/proxy.env", root_path), b"SODA_TLS=internal\n")
                        .unwrap();
                    Ok(Vec::new())
                });
                configure_private_install(
                    &worker_ctx,
                    &console,
                    &run,
                    &root_path,
                    &root_path,
                    &format!("{root_path}/not-yet-created-ca.crt"),
                )
            });
            let mut transcript = Vec::new();
            read_until(
                &mut master,
                &mut transcript,
                &ctx,
                "Address number, or cancel",
            );
            master.write_all(b"1\n").unwrap();
            read_until(
                &mut master,
                &mut transcript,
                &ctx,
                "When Forgejo setup is complete",
            );
            {
                let text = String::from_utf8_lossy(&transcript);
                assert!(
                    text.contains("Required scope: read:user"),
                    "bootstrap guidance requests unrelated token authority"
                );
                assert!(
                    !text.contains("write:admin"),
                    "bootstrap guidance requests unrelated token authority"
                );
                assert!(
                    !text.contains("read:repository"),
                    "bootstrap guidance requests unrelated token authority"
                );
            }
            if cancel_setup {
                master.write_all(b"cancel\n").unwrap();
            } else {
                master.write_all(b"CONFIGURE SODA\n").unwrap();
                read_until(
                    &mut master,
                    &mut transcript,
                    &ctx,
                    "Operator Forgejo token: ",
                );
                master.write_all(format!("{TOKEN}\n").as_bytes()).unwrap();
            }
            let err = worker.join().unwrap();
            if cancel_setup {
                assert!(err.is_err(), "cancelled setup succeeded");
                assert_eq!(
                    mutations.load(std::sync::atomic::Ordering::SeqCst),
                    0,
                    "cancelled setup made native changes"
                );
                assert!(
                    matches!(
                        std::fs::symlink_metadata(format!("{}/setup-started", root.path)),
                        Err(e) if e.kind() == std::io::ErrorKind::NotFound
                    ),
                    "cancelled setup reserved an attempt"
                );
            } else {
                assert!(
                    err.is_ok(),
                    "setup failed: {err:?}, mutations {}",
                    mutations.load(std::sync::atomic::Ordering::SeqCst)
                );
                assert_eq!(mutations.load(std::sync::atomic::Ordering::SeqCst), 2);
                read_until(
                    &mut master,
                    &mut transcript,
                    &ctx,
                    "certificate is not available yet",
                );
                let leftovers: Vec<_> = std::fs::read_dir(&root.path)
                    .unwrap()
                    .filter_map(|e| e.ok())
                    .filter(|e| e.file_name().to_string_lossy().starts_with("soda-setup-"))
                    .collect();
                assert!(
                    leftovers.is_empty(),
                    "operator token workdir persists: {leftovers:?}"
                );
            }
            transcript.extend_from_slice(&drain_available(&mut master));
            assert!(
                !String::from_utf8_lossy(&transcript).contains(TOKEN),
                "operator token echoed to console"
            );
        }
    }
}
