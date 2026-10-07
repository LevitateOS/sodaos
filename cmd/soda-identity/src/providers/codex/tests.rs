// Codex provider tests (A06.M).
use super::super::TestDir;
use super::config::{filter_env, validate_config, Config};
use super::protocol::protocol_error;
use super::*;
use std::time::Instant;

#[test]
fn environment_filters_provider_credentials() {
    let vars = vec![
        ("OPENAI_API_KEY".to_string(), "synthetic".to_string()),
        ("CODEX_HOME".to_string(), "synthetic".to_string()),
        ("CHATGPT_AUTH".to_string(), "synthetic".to_string()),
        ("AWS_SECRET".to_string(), "synthetic".to_string()),
        ("TMPDIR".to_string(), "synthetic".to_string()),
        ("XDG_CACHE_HOME".to_string(), "synthetic".to_string()),
        ("PATH".to_string(), "/usr/bin".to_string()),
    ];
    let env = filter_env(vars.into_iter(), "/enrollment-root");
    assert!(
        !env.iter().any(|e| e.contains("synthetic")),
        "leaked: {env:?}"
    );
    assert!(env.contains(&"CODEX_HOME=/enrollment-root".to_string()));
    assert!(env.contains(&"TMPDIR=/enrollment-root".to_string()));
    assert!(env.contains(&"XDG_CACHE_HOME=/enrollment-root/cache".to_string()));
    assert!(env.contains(&"PATH=/usr/bin".to_string()));
}

#[test]
fn config_validation_matches_go() {
    let good = Config {
        binary: "/usr/local/bin/codex".to_string(),
        version: VERSION.to_string(),
        sha256: "a".repeat(64),
        root: "/enrollment-root".to_string(),
    };
    validate_config(&good).unwrap();
    for bad in [
        Config {
            binary: "relative".to_string(),
            ..good.clone()
        },
        Config {
            version: "other".to_string(),
            ..good.clone()
        },
        Config {
            sha256: "short".to_string(),
            ..good.clone()
        },
        Config {
            root: "relative".to_string(),
            ..good.clone()
        },
    ] {
        assert_eq!(
            validate_config(&bad).unwrap_err().to_string(),
            "invalid pinned Codex configuration"
        );
    }
}

fn protocol_fixture(completed: bool) -> String {
    // Minimal app-server fixture: initialize, device-code start with an
    // immediate completion event, account read and login cancel. Methods
    // match exactly so the `initialized` notification falls through.
    let completion = if completed {
        "printf '{\"tokens\":{\"refresh_token\":\"synthetic-enrollment\"}}' > \"$CODEX_HOME/auth.json\"; chmod 600 \"$CODEX_HOME/auth.json\"; echo '{\"method\":\"account/login/completed\",\"params\":{\"loginId\":\"synthetic-login\",\"success\":true,\"error\":null}}'".to_string()
    } else {
        ":".to_string()
    };
    format!(
            "#!/bin/sh\nset -eu\nid_of() {{ printf '%s' \"$1\" | sed 's/.*\"id\":\\([0-9][0-9]*\\).*/\\1/'; }}\nwhile IFS= read -r line; do\ncase \"$line\" in\n*\\\"method\\\":\\\"initialize\\\"*) echo \"{{\\\"id\\\":$(id_of \"$line\"),\\\"result\\\":{{}}}}\";;\n*\\\"method\\\":\\\"account/login/start\\\"*) echo \"{{\\\"id\\\":$(id_of \"$line\"),\\\"result\\\":{{\\\"type\\\":\\\"chatgptDeviceCode\\\",\\\"loginId\\\":\\\"synthetic-login\\\",\\\"verificationUrl\\\":\\\"https://auth.openai.com/codex/device\\\",\\\"userCode\\\":\\\"ABCD-1234\\\"}}}}\"; {completion};;\n*\\\"method\\\":\\\"account/read\\\"*) echo \"{{\\\"id\\\":$(id_of \"$line\"),\\\"result\\\":{{\\\"account\\\":{{\\\"type\\\":\\\"chatgpt\\\",\\\"email\\\":null,\\\"planType\\\":\\\"plus\\\"}},\\\"requiresOpenaiAuth\\\":true}}}}\";;\n*\\\"method\\\":\\\"account/login/cancel\\\"*) echo \"{{\\\"id\\\":$(id_of \"$line\"),\\\"result\\\":{{\\\"status\\\":\\\"canceled\\\"}}}}\";;\n*) :;;\nesac\ndone\n"
        )
}

fn fixture_provider(dir: &TestDir, completed: bool) -> Provider {
    let binary = dir.path().join("codex-fixture");
    std::fs::write(&binary, protocol_fixture(completed)).unwrap();
    std::fs::set_permissions(&binary, std::fs::Permissions::from_mode(0o700)).unwrap();
    Provider {
        config: Config {
            binary: binary.to_string_lossy().into_owned(),
            version: VERSION.to_string(),
            sha256: String::new(),
            root: dir.path().to_string_lossy().into_owned(),
        },
    }
}

#[test]
fn managed_enrollment_persists_only_after_process_stop() {
    let dir = TestDir::new();
    let provider = fixture_provider(&dir, true);
    let session = provider.start(1).unwrap();
    let deadline = Instant::now() + Duration::from_secs(5);
    while session.snapshot().state == "pending" {
        assert!(Instant::now() < deadline, "fixture enrollment timeout");
        std::thread::sleep(Duration::from_millis(1));
    }
    let (conn, data) = session.finish().unwrap();
    assert_eq!(conn.plan, "plus");
    assert_eq!(conn.email, "");
    assert_eq!(
        data,
        br#"{"tokens":{"refresh_token":"synthetic-enrollment"}}"#
    );
    assert!(
        session.inner.finished.load(Ordering::SeqCst),
        "credential retained before provider process ended"
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
    assert!(!root.exists(), "credential tmpfs root not removed");
}

#[test]
fn cancel_removes_unfinished_enrollment() {
    let dir = TestDir::new();
    let provider = fixture_provider(&dir, false);
    let session = provider.start(1).unwrap();
    assert_eq!(session.snapshot().state, "pending");
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
    assert!(!root.exists(), "canceled credentials retained");
}

#[test]
fn device_disabled_maps_to_actionable_error() {
    let err = protocol_error(&serde_json::json!({"message": "Device code login is not enabled"}));
    assert!(err
        .to_string()
        .contains("enable it in ChatGPT security settings"));
    let err = protocol_error(&serde_json::json!({"message": "boom"}));
    assert_eq!(err.to_string(), "codex protocol request failed");
}
