use super::super::*;
use super::fixtures::null_console;

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
        let ctx = ctx.with_timeout(std::time::Instant::now() + std::time::Duration::from_secs(15));
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
                std::fs::write(format!("{}/proxy.env", root_path), b"SODA_TLS=internal\n").unwrap();
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
