use super::*;

#[test]
fn oracle_prefix_and_digest_shapes() {
    // Oracle: Go ValidRepositoryPrefix + Digest vectors.
    assert!(valid_repository_prefix("ghcr.io/example/sodaos"));
    assert!(!valid_repository_prefix("ghcr.io/Example/sodaos"));
    assert!(!valid_repository_prefix("docker.io/example/sodaos"));
    assert!(!valid_repository_prefix("ghcr.io/example"));
    assert!(prefixed_digest(&format!("sha256:{}", "a".repeat(64))));
    assert!(!prefixed_digest(&"a".repeat(64)));
}

#[test]
fn oracle_url_parser_matches_go_probe() {
    // Oracle: /tmp/urlprobe vectors from the real Go url.Parse.
    let upper = parse_url("HTTPS://Example.COM:8080/p").unwrap();
    assert_eq!(upper.scheme, "https");
    assert_eq!(upper.hostname, "Example.COM");
    assert!(parse_url("https://h:bad/p").is_none());
    assert_eq!(parse_url("https:///p").unwrap().host, "");
    assert!(parse_url("https://h p/").is_none());
    assert!(parse_url("https://[::1]/p").unwrap().hostname == "::1");
    assert!(https_url(
        "https://example.invalid/builds/1.0.0.0/release.json"
    ));
    assert!(!https_url("http://example.invalid/x"));
    assert!(!https_url("https://example.invalid/x?"));
    assert!(is_loopback_addr("127.0.0.1"));
    assert!(is_loopback_addr("::1"));
    assert!(is_loopback_addr("0:0:0:0:0:0:0:1"));
    assert!(!is_loopback_addr("10.0.0.1"));
    assert!(!is_loopback_addr("::ffff:127.0.0.1"));
    assert!(!is_loopback_addr("example.invalid"));
}

#[test]
fn oracle_payload_identity_validation() {
    // Oracle: Go Payload.Validate error strings.
    let payload = Payload::default();
    assert_eq!(
        payload.validate().unwrap_err().0,
        "invalid appliance payload identity"
    );
}
