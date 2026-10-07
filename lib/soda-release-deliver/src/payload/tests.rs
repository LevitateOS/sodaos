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

#[test]
fn payload_raw_slots_keep_only_the_last_exact_value() {
    let payload: Payload = crate::buildx::decode_build_json(br#"{"Format":1.5,"Format":-0,"ID":"old","ID":"new","Schema":-0,"Images":{"x":{"Reference":7,"Reference":"ok"}}}"#).unwrap();
    assert_eq!(payload.format, 0);
    assert_eq!(payload.id, "new");
    assert_eq!(payload.schema, 0);
    assert_eq!(payload.images["x"].reference, "ok");

    assert!(crate::buildx::decode_build_json(br#"{"Format":-0.0}"#).is_err());
    assert!(crate::buildx::decode_build_json(br#"{"Format":1e0}"#).is_err());
    assert!(crate::buildx::decode_build_json(br#"{"Format":9223372036854775808}"#).is_err());
    let reset: Payload =
        crate::buildx::decode_build_json(br#"{"Format":3,"Format":null}"#).unwrap();
    assert_eq!(reset.format, 0);
}

#[test]
fn images_map_decodes_only_the_last_duplicate_entry() {
    let payload: Payload = crate::buildx::decode_build_json(
        br#"{"Images":{"x":{"Reference":false},"x":{"Reference":"winner"}}}"#,
    )
    .unwrap();
    assert_eq!(payload.images["x"].reference, "winner");

    assert!(crate::buildx::decode_build_json(
        br#"{"Images":{"x":{"Reference":"valid"},"x":{"Reference":false}}}"#,
    )
    .is_err());
}
