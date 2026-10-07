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

#[test]
fn trust_key_admission_preserves_cross_role_separation() {
    let key = "-----BEGIN PUBLIC KEY-----\nMFkwEwYHKoZIzj0CAQYIKoZIzj0DAQcDQgAEaxfR8uEsQkf4vOblY6RA8ncDfYEt\n6zOg9KE5RdiYwpZP40Li/hp/m47n60p8D54WK84zV2sxXs7LtkBoN79R9Q==\n-----END PUBLIC KEY-----".to_string();
    let trust = Trust {
        format: 1,
        prefix: "ghcr.io/example/sodaos".to_string(),
        epoch: 1,
        keys: ["artifact", "candidate", "preview", "stable"]
            .into_iter()
            .map(|role| (role.to_string(), vec![key.clone()]))
            .collect(),
        not_before: 1,
        max_age_seconds: 60,
        clock_skew_seconds: 0,
        minimum_sequence: ["candidate", "preview", "stable"]
            .into_iter()
            .map(|role| (role.to_string(), 1))
            .collect(),
    };
    assert_eq!(
        trust.validate().unwrap_err().0,
        "signer roles must not share keys"
    );
}

#[test]
fn producer_trust_fixture_keeps_its_raw_der_fingerprint() {
    let text = include_str!("../../../../system/host/trust/release-trust.json");
    let trust = Trust::parse(text).expect("producer trust fixture");
    trust.validate().expect("producer trust fixture admits");
    let pem = trust
        .keys
        .iter()
        .find(|(role, _)| role == "artifact")
        .and_then(|(_, keys)| keys.first())
        .expect("artifact trust key");
    let der = soda_build_tools::trust_key::parse_p256_public_key(pem).expect("admitted SPKI");
    assert_eq!(
        deliver_hash(&der),
        "sha256:caf16f22be044cdd5cfb72570ba2204e09f13d1a26ff8bfb6ff7aa79d76416fd"
    );
}

#[test]
fn image_model_uses_exact_then_first_folded_raw_member() {
    let image = Image::parse(r#"{"mAnIfEsT":"folded","Manifest":"exact"}"#).unwrap();
    assert_eq!(image.manifest, "exact");

    let image = Image::parse(r#"{"mAnIfEsT":"first","MANIFEST":"second"}"#).unwrap();
    assert_eq!(image.manifest, "first");

    assert!(Image::parse(r#"{"MANIFEST":false,"Manifest":"exact"}"#).is_ok());
    assert!(Image::parse(r#"{"Manifest":false,"Manifest":"later"}"#).is_err());
}

#[test]
fn release_dtos_defer_nested_alias_conversion_and_keep_minus_zero() {
    let candidate = Candidate::parse(r#"{"format":false,"Format":3}"#).unwrap();
    assert_eq!(candidate.format, 3);
    assert!(Candidate::parse(r#"{"Format":false,"Format":3}"#).is_err());

    let payload = Payload::parse(
        r#"{"Format":-0,"Images":{"host":{"reference":"ignored","Reference":"exact"}}}"#,
    )
    .unwrap();
    assert_eq!(payload.format, 0);
    assert_eq!(payload.image("host").reference, "exact");
}
