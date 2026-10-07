// Muse provider tests (A06.M).
use super::super::TestDir;
use super::*;
use std::os::unix::fs::PermissionsExt;

const SUBSCRIPTION: &str = r#"{"schema_version":1,"providers":{"meta":{"access_token":"synthetic-oauth","api_key":"synthetic-subscription","api_base_url":"https://api.meta.ai/v1","mechanism":"oauth","obtained_via":"device_code","account_name":"synthetic presentation"}}}"#;

#[test]
fn native_subscription_and_private_file() {
    assert!(credential_valid(SUBSCRIPTION.as_bytes()));
    for value in [
        r#"{"schema_version":1,"providers":{"meta":{"api_key":"synthetic-payg","mechanism":"api_key"}}}"#,
        r#"{"schema_version":1,"providers":{"meta":{"access_token":"synthetic","api_key":"synthetic","api_base_url":"https://other.invalid","mechanism":"oauth","obtained_via":"device_code"}}}"#,
    ] {
        assert!(!credential_valid(value.as_bytes()), "admitted {value}");
    }
    assert!(!credential_valid(b"not json"));
    assert!(!credential_valid(b""));

    let dir = TestDir::new();
    let path = dir.path().join("auth.json");
    std::fs::write(&path, SUBSCRIPTION).unwrap();
    std::fs::set_permissions(&path, std::fs::Permissions::from_mode(0o600)).unwrap();
    credential_file(&path).unwrap();
    std::fs::set_permissions(&path, std::fs::Permissions::from_mode(0o644)).unwrap();
    assert!(
        credential_file(&path).is_err(),
        "public credential accepted"
    );
    let link_dir = TestDir::new();
    let link = link_dir.path().join("auth.json");
    std::os::unix::fs::symlink(&path, &link).unwrap();
    assert!(credential_file(&link).is_err(), "symlink accepted");
}

#[test]
fn enrollment_environment_and_device_prompt() {
    let env = environment("/private-enrollment");
    assert_eq!(
        env,
        vec![
            "PATH=/usr/local/bin:/usr/bin:/bin",
            "HOME=/private-enrollment",
            "XDG_CONFIG_HOME=/private-enrollment/config",
            "XDG_DATA_HOME=/private-enrollment/data",
            "XDG_STATE_HOME=/private-enrollment/state",
            "XDG_CACHE_HOME=/private-enrollment/cache",
            "TMPDIR=/private-enrollment",
            "TBH_CREDENTIAL_BACKEND=file",
            "NO_COLOR=1",
        ]
    );
    let found =
        scan_device_url(b"Open https://auth.meta.com/oauth/device/?code=ABCD-1234 to enroll");
    assert_eq!(
        found,
        Some((
            "https://auth.meta.com/oauth/device/?code=ABCD-1234".to_string(),
            "ABCD-1234".to_string()
        ))
    );
    // Lowercase codes, short codes and lookalike hosts are not prompts.
    assert_eq!(
        scan_device_url(b"https://auth.meta.com/oauth/device/?code=abcd-1234"),
        None
    );
    assert_eq!(
        scan_device_url(b"https://auth.meta.com/oauth/device/?code=ABC-1234"),
        None
    );
    assert_eq!(
        scan_device_url(b"https://auth.example.com/oauth/device/?code=ABCD-1234"),
        None
    );
}

#[test]
fn config_validation_matches_go() {
    let dir = TestDir::new();
    std::fs::set_permissions(dir.path(), std::fs::Permissions::from_mode(0o700)).unwrap();
    let good = Config {
        binary: "/usr/local/bin/muse".to_string(),
        version: VERSION.to_string(),
        sha256: "a".repeat(64),
        root: dir.path().to_string_lossy().into_owned(),
    };
    // /tmp is not tmpfs on test hosts, so a fully valid config still
    // fails at the tmpfs gate; every other field must fail earlier
    // with the configuration error.
    for mut bad in [good.clone(), good.clone(), good.clone()] {
        bad.binary = "relative/muse".to_string();
        assert_eq!(
            validate_config(&bad).unwrap_err().to_string(),
            "invalid pinned Muse configuration"
        );
        bad = good.clone();
        bad.version = "other".to_string();
        assert_eq!(
            validate_config(&bad).unwrap_err().to_string(),
            "invalid pinned Muse configuration"
        );
        bad = good.clone();
        bad.sha256 = "short".to_string();
        assert_eq!(
            validate_config(&bad).unwrap_err().to_string(),
            "invalid pinned Muse configuration"
        );
    }
    let mut bad = good.clone();
    bad.root = "relative/root".to_string();
    assert_eq!(
        validate_config(&bad).unwrap_err().to_string(),
        "invalid pinned Muse configuration"
    );
    std::fs::set_permissions(dir.path(), std::fs::Permissions::from_mode(0o755)).unwrap();
    assert_eq!(
        validate_config(&good).unwrap_err().to_string(),
        "muse enrollment root must be private"
    );
}

#[test]
fn binary_digest_checked_before_version() {
    let dir = TestDir::new();
    let binary = dir.path().join("muse-fixture");
    std::fs::write(&binary, b"fixture-bytes").unwrap();
    let sum = sha256::hex(&sha256::digest(b"fixture-bytes"));
    let config = Config {
        binary: binary.to_string_lossy().into_owned(),
        version: VERSION.to_string(),
        sha256: sum,
        root: dir.path().to_string_lossy().into_owned(),
    };
    check_binary(&config).unwrap();
    let bad = Config {
        sha256: "b".repeat(64),
        ..config
    };
    assert_eq!(
        check_binary(&bad).unwrap_err().to_string(),
        "muse digest mismatch"
    );
}

#[test]
fn version_line_must_match_exactly() {
    let dir = TestDir::new();
    let binary = dir.path().join("muse-fixture");
    std::fs::write(&binary, format!("#!/bin/sh\necho '{VERSION_LINE}'\n")).unwrap();
    std::fs::set_permissions(&binary, std::fs::Permissions::from_mode(0o700)).unwrap();
    let config = Config {
        binary: binary.to_string_lossy().into_owned(),
        version: VERSION.to_string(),
        sha256: "c".repeat(64),
        root: dir.path().to_string_lossy().into_owned(),
    };
    check_version(&config).unwrap();
    std::fs::write(&binary, "#!/bin/sh\necho 'Muse Code 9.9.9 (other)'\n").unwrap();
    assert_eq!(
        check_version(&config).unwrap_err().to_string(),
        "muse version mismatch"
    );
}

#[test]
fn native_enrollment_keeps_presentation_private() {
    let dir = TestDir::new();
    let binary = dir.path().join("muse-fixture");
    let script = format!(
            "#!/bin/sh\nset -eu\n[ \"$1\" = login ]\n[ \"$TBH_CREDENTIAL_BACKEND\" = file ]\n[ -z \"${{META_API_KEY:-}}\" ]\nprintf 'https://auth.meta.com/oauth/device/?code=ABCD-1234'\nmkdir -p \"$XDG_CONFIG_HOME/muse\"\numask 077\ncat > \"$XDG_CONFIG_HOME/muse/auth.json\" <<'JSON'\n{SUBSCRIPTION}\nJSON\nprintf 'synthetic secret diagnostic' >&2\n"
        );
    std::fs::write(&binary, script).unwrap();
    std::fs::set_permissions(&binary, std::fs::Permissions::from_mode(0o700)).unwrap();
    let provider = Provider {
        config: Config {
            binary: binary.to_string_lossy().into_owned(),
            version: VERSION.to_string(),
            sha256: String::new(),
            root: dir.path().to_string_lossy().into_owned(),
        },
    };
    let session = provider.start(1).unwrap();
    let deadline = std::time::Instant::now() + Duration::from_secs(5);
    loop {
        if session.inner.finished.load(Ordering::SeqCst) {
            break;
        }
        assert!(
            std::time::Instant::now() < deadline,
            "enrollment fixture did not stop"
        );
        std::thread::sleep(Duration::from_millis(5));
    }
    let snapshot = session.snapshot();
    assert_eq!(snapshot.state, "completed");
    assert_eq!(snapshot.user_code, "ABCD-1234");
    let (conn, data) = session.finish().unwrap();
    assert!(credential_valid(&data));
    let presentation = serde_json::to_string(&conn).unwrap();
    assert!(
        !presentation.contains("synthetic"),
        "credentials escaped: {presentation}"
    );
    let public = serde_json::to_string(&snapshot).unwrap();
    assert!(
        !public.contains("synthetic"),
        "diagnostics escaped: {public}"
    );
    let root = session
        .inner
        .root
        .lock()
        .unwrap()
        .as_ref()
        .unwrap()
        .path()
        .to_path_buf();
    session.close().unwrap();
    assert!(!root.exists(), "credential root not removed");
}
