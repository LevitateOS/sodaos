use std::fs;
use std::os::unix::fs::PermissionsExt;

use super::fixtures::*;

use crate::setup::setup;

#[test]
fn success_revokes_and_failure_keeps_retry_token() {
    for admin in [true, false] {
        let root = test_root();
        let dir = root.join("soda");
        fs::create_dir(&dir).expect("soda dir");
        let token_path = write_token(&root);
        let out = dir.join("dashboard.json");
        let stub = stub_server(admin, false, "synthetic-bootstrap-token-not-for-retention");
        let mut stdout: Vec<u8> = Vec::new();
        let err = setup(
            "https://forgejo.test/",
            &stub.url,
            &token_path.to_string_lossy(),
            &out,
            &root.join("postgres"),
            &mut stdout,
        );
        let calls = stub.calls.lock().expect("calls").clone();
        if admin {
            assert!(err.is_ok(), "setup failed: {err:?}");
            assert_eq!(calls, vec!["GET /api/v1/user", "DELETE /api/v1/user/token"]);
        } else {
            assert!(err.is_err(), "failing setup succeeded");
            assert_eq!(calls, vec!["GET /api/v1/user"]);
        }
    }
}

#[test]
fn revoke_failure_after_publication_is_unconfirmed_without_retry() {
    const TOKEN: &str = "synthetic-bootstrap-token-not-for-retention";
    let lost = stub_server_applies_delete_then_drops_response(true, TOKEN);
    let refused = stub_server_rejects_delete(true, TOKEN);

    for (name, stub, delete_applied) in [
        ("lost response", &lost, true),
        ("explicit refusal", &refused, false),
    ] {
        let root = test_root();
        let dir = root.join("soda");
        fs::create_dir(&dir).expect("soda dir");
        let token_path = write_token(&root);
        let out = dir.join("dashboard.json");
        let mut stdout = Vec::new();
        let err = setup(
            "https://forgejo.test/",
            &stub.url,
            &token_path.to_string_lossy(),
            &out,
            &root.join("postgres"),
            &mut stdout,
        )
        .expect_err("revoke error must fail setup");

        assert!(err.contains("revocation was not confirmed"), "{err}");
        assert!(
            err.contains("inspect Forgejo Settings > Applications"),
            "{err}"
        );
        assert!(!err.contains(TOKEN), "token leaked in error");
        assert!(stdout.is_empty(), "unexpected success output: {stdout:?}");
        let config = fs::read_to_string(&out).expect("published config remains");
        assert!(config.contains("\"operator_id\":42"), "{config}");
        assert!(!config.contains(TOKEN), "token leaked in config");
        assert!(
            stub.delete_applied
                .load(std::sync::atomic::Ordering::SeqCst)
                == delete_applied,
            "{name}: fixture application state mismatch"
        );
        assert_eq!(
            stub.calls.lock().expect("calls").as_slice(),
            ["GET /api/v1/user", "DELETE /api/v1/user/token"],
            "{name}: setup must issue one DELETE and must not retry"
        );
    }
}

#[test]
fn preserves_pre_existing_secrets_and_does_not_create_broker_key() {
    let root = test_root();
    let dir = root.join("soda");
    fs::create_dir(&dir).expect("soda dir");
    let token_path = write_token(&root);
    let stub = stub_server(true, false, "synthetic-bootstrap-token-not-for-retention");
    let pg_dir = root.join("postgres");
    fs::create_dir_all(&pg_dir).expect("pg dir");
    fs::write(pg_dir.join("super.passwd"), "operator-owned-secret\n").expect("prior");
    let mut stdout: Vec<u8> = Vec::new();
    let err = setup(
        "https://forgejo.test/",
        &stub.url,
        &token_path.to_string_lossy(),
        &dir.join("dashboard.json"),
        &pg_dir,
        &mut stdout,
    );
    assert!(err.is_err(), "setup overwrote pre-existing secrets");
    assert_eq!(
        fs::read_to_string(pg_dir.join("super.passwd")).expect("prior"),
        "operator-owned-secret\n"
    );
    assert!(!dir.join("grant-key").exists(), "setup created broker key");
    assert!(
        !dir.join("dashboard.json").exists(),
        "failed setup published config"
    );
}

#[test]
fn successful_setup_leaves_existing_broker_key_untouched() {
    let root = test_root();
    let dir = root.join("soda");
    fs::create_dir(&dir).expect("soda dir");
    let key_path = dir.join("grant-key");
    fs::write(&key_path, "operator-provisioned broker key\n").expect("key");
    let token_path = write_token(&root);
    let stub = stub_server(true, false, "synthetic-bootstrap-token-not-for-retention");
    let mut stdout = Vec::new();
    setup(
        "https://forgejo.test/",
        &stub.url,
        &token_path.to_string_lossy(),
        &dir.join("dashboard.json"),
        &root.join("postgres"),
        &mut stdout,
    )
    .expect("setup succeeds");
    assert_eq!(
        fs::read(&key_path).expect("key remains"),
        b"operator-provisioned broker key\n"
    );
    let config = fs::read_to_string(dir.join("dashboard.json")).expect("dashboard config");
    assert!(
        !config.contains("grant_key_file"),
        "dashboard gained broker key authority"
    );
}

#[test]
fn credential_boundary_matrix() {
    for name in [
        "new",
        "unrelated-file",
        "existing-config",
        "existing-key",
        "non-admin",
        "current-denied",
        "missing-token",
    ] {
        let root = test_root();
        let dir = root.join("soda");
        fs::create_dir(&dir).expect("soda dir");
        let token_path = root.join("operator-input");
        if name != "missing-token" {
            fs::write(&token_path, "synthetic-bootstrap-token-not-for-retention\n").expect("token");
            fs::set_permissions(&token_path, fs::Permissions::from_mode(0o600))
                .expect("token mode");
        }
        let out = dir.join("dashboard.json");
        let retained = match name {
            "unrelated-file" => Some(dir.join("operator-notes")),
            "existing-config" => Some(out.clone()),
            "existing-key" => Some(dir.join("grant-key")),
            _ => None,
        };
        if let Some(path) = &retained {
            fs::write(path, "preserve original bytes\n").expect("retained");
            fs::set_permissions(path, fs::Permissions::from_mode(0o600)).expect("retained mode");
        }
        let stub = stub_server(
            name != "non-admin",
            name == "current-denied",
            "synthetic-bootstrap-token-not-for-retention",
        );
        let mut stdout: Vec<u8> = Vec::new();
        let err = setup(
            "https://forgejo.test/",
            &stub.url,
            &token_path.to_string_lossy(),
            &out,
            &root.join("postgres"),
            &mut stdout,
        );
        let output = String::from_utf8_lossy(&stdout).into_owned();
        assert!(
            !output.contains("synthetic-bootstrap-token"),
            "{name}: token in stdout"
        );
        if let Err(err) = &err {
            assert!(
                !err.contains("synthetic-bootstrap-token"),
                "{name}: token in error"
            );
        }
        let success = name == "new" || name == "unrelated-file" || name == "existing-key";
        assert_eq!(err.is_ok(), success, "{name}: success={}", err.is_ok());
        let calls = stub.calls.lock().expect("calls").clone();
        let mut want_calls: Vec<String> = Vec::new();
        if name != "existing-config" && name != "missing-token" {
            want_calls.push("GET /api/v1/user".to_string());
        }
        if success {
            want_calls.push("DELETE /api/v1/user/token".to_string());
        }
        assert_eq!(calls, want_calls, "{name}: provider calls");
        if let Some(path) = &retained {
            assert_eq!(
                fs::read_to_string(path).expect("retained"),
                "preserve original bytes\n",
                "{name}: retained changed"
            );
            assert_eq!(
                fs::metadata(path).expect("retained").permissions().mode() & 0o777,
                0o600
            );
        }
        for entry in fs::read_dir(&dir).expect("dir") {
            let entry = entry.expect("entry");
            let data = fs::read_to_string(entry.path()).expect("entry data");
            assert!(
                !data.contains("synthetic-bootstrap-token"),
                "{name}: token retained"
            );
        }
        if !success {
            if name != "existing-config" {
                assert!(!out.exists(), "{name}: failed setup published config");
            }
            continue;
        }
        // dashboard.json records the operator identity and trimmed browser
        // origin without taking ownership of the broker key.
        let config = fs::read_to_string(&out).expect("config");
        assert!(config.contains("\"operator_id\":42"), "{config}");
        assert!(
            config.contains("\"forgejo_url\":\"https://forgejo.test\""),
            "{config}"
        );
        assert!(config.contains("\"identity_socket\":\"\""), "{config}");
        assert_eq!(
            fs::metadata(&out).expect("config").permissions().mode() & 0o777,
            0o600
        );
        let key_path = dir.join("grant-key");
        if name == "existing-key" {
            assert_eq!(
                fs::read_to_string(&key_path).expect("key"),
                "preserve original bytes\n"
            );
        } else {
            assert!(!key_path.exists(), "setup created broker key");
        }
    }
}
