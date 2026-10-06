use super::*;

#[test]
fn repository_prefix_shapes() {
    assert!(valid_repository_prefix("ghcr.io/example/sodaos"));
    assert!(valid_repository_prefix("ghcr.io/a/b"));
    assert!(valid_repository_prefix("ghcr.io/my-org/my.repo_name-1"));
    assert!(!valid_repository_prefix("ghcr.io/example"));
    assert!(!valid_repository_prefix("ghcr.io//sodaos"));
    assert!(!valid_repository_prefix("quay.io/example/sodaos"));
    assert!(!valid_repository_prefix("ghcr.io/Example/sodaos"));
    assert!(!valid_repository_prefix("ghcr.io/example/sodaos/extra"));
    assert!(!valid_repository_prefix(&format!(
        "ghcr.io/e/{}",
        "s".repeat(200)
    )));
}

#[test]
fn payload_identity_errors_match_go() {
    let empty = Payload::default();
    assert_eq!(
        empty.validate().unwrap_err(),
        Error::msg("invalid appliance payload identity")
    );
    let mut bad_core = Payload {
        format: 3,
        revision: "a".repeat(40),
        core_os: "44.1".to_string(),
        ..Payload::default()
    };
    bad_core.id = format!("{}.soda-{}", bad_core.core_os, &bad_core.revision[..12]);
    assert_eq!(
        bad_core.validate().unwrap_err(),
        Error::msg("invalid appliance payload identity")
    );
}
