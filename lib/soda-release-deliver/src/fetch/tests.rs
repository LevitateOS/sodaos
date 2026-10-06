use super::*;

fn private_dir(prefix: &str) -> String {
    let dir = std::env::temp_dir().join(format!(
        "{prefix}-{}-{}",
        std::process::id(),
        std::time::SystemTime::now()
            .duration_since(std::time::UNIX_EPOCH)
            .unwrap()
            .as_nanos()
    ));
    std::fs::create_dir_all(&dir).unwrap();
    use std::os::unix::fs::PermissionsExt;
    std::fs::set_permissions(&dir, std::fs::Permissions::from_mode(0o700)).unwrap();
    dir.to_string_lossy().into_owned()
}

#[test]
fn state_lock_serializes_and_saves() {
    let dir = private_dir("srd-lock");
    let path = format!("{dir}/state.json");
    std::fs::write(&path, b"{}").unwrap();
    use std::os::unix::fs::PermissionsExt;
    std::fs::set_permissions(&path, std::fs::Permissions::from_mode(0o600)).unwrap();
    let _first = lock_state(&path).unwrap();
    assert_eq!(
        lock_state(&path).unwrap_err(),
        Error::msg("release operation already active")
    );
    drop(_first);
    let _second = lock_state(&path).unwrap();
    let mut state = empty_state();
    state.checked_at = 42;
    save_state(&path, &state).unwrap();
    let raw = std::fs::read(&path).unwrap();
    assert!(String::from_utf8(raw)
        .unwrap()
        .contains("\"CheckedAt\": 42"));
}

#[test]
fn fetch_request_admission() {
    let trust = Trust::default();
    assert!(admit_fetch_request(&trust, "candidate", "x86_64").is_err());
    assert!(admit_fetch_request(&trust, "bogus", "x86_64").is_err());
}
