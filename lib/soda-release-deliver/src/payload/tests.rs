use super::*;

const FIXTURE: &str = include_str!("../../../release-inputs/fixtures/payload-v4.json");

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
    let payload: Payload =
        crate::buildx::decode_build_json(FIXTURE.as_bytes()).expect("shared v4 fixture decodes");
    payload.validate().expect("shared v4 fixture validates");
    let old: Payload = crate::buildx::decode_build_json(
        FIXTURE
            .replacen("\"Format\":4", "\"Format\":3", 1)
            .as_bytes(),
    )
    .expect("old format still decodes");
    assert_eq!(
        old.validate().unwrap_err().to_string(),
        "invalid appliance payload identity"
    );
}

#[test]
fn payload_decode_rejects_unknown_duplicate_and_trailing_shapes() {
    // Unknown top-level fields keep Go's unknown-field text.
    let unknown = FIXTURE.replacen("\"Schema\":1", "\"Schema\":1,\"RepositoryPrefix\":\"x\"", 1);
    assert_eq!(
        crate::buildx::decode_build_json(unknown.as_bytes()).unwrap_err(),
        Error::msg("json: unknown field \"RepositoryPrefix\"")
    );
    // Duplicate role keys fail instead of last-wins.
    let duplicated = FIXTURE.replacen("\"dashboard\":{", "\"dashboard\":{},\"dashboard\":{", 1);
    assert!(crate::buildx::decode_build_json(duplicated.as_bytes()).is_err());
    // Trailing data fails.
    let mut trailing = FIXTURE.as_bytes().to_vec();
    trailing.extend_from_slice(b"{}");
    assert!(crate::buildx::decode_build_json(&trailing).is_err());
}
