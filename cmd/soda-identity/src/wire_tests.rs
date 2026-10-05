
use super::*;

#[test]
fn go_time_vectors_round_trip() {
    // Captured from encoding/json with the pinned Go toolchain.
    for (text, sec, nanos) in [
        ("2026-10-04T18:30:05Z", 1791138605, 0),
        ("2026-10-04T18:30:05.123Z", 1791138605, 123_000_000),
        ("2026-10-04T18:30:05.123456Z", 1791138605, 123_456_000),
        ("2026-10-04T18:30:05.123456789Z", 1791138605, 123_456_789),
        ("2026-10-04T18:30:05.12Z", 1791138605, 120_000_000),
    ] {
        let (parsed_sec, parsed_nanos) = parse_rfc3339_nano(text).unwrap();
        assert_eq!((parsed_sec, parsed_nanos), (sec, nanos), "{text}");
        assert_eq!(format_rfc3339_nano(sec, nanos), text, "{text}");
    }
    // Offsets normalize to the same instant Go encodes with Z.
    let (sec, nanos) = parse_rfc3339_nano("2026-10-04T20:30:05+02:00").unwrap();
    assert_eq!((sec, nanos), (1791138605, 0));
    assert_eq!(format_rfc3339_nano(sec, nanos), "2026-10-04T18:30:05Z");
}

#[test]
fn timestamps_reject_malformed_input() {
    for text in [
        "",
        "2026-10-04",
        "2026-13-04T18:30:05Z",
        "2026-10-32T18:30:05Z",
        "2026-10-04T25:30:05Z",
        "2026-10-04T18:30:05",
        "2026-10-04T18:30:05.Z",
        "2026-10-04T18:30:05.1234567890Z",
        "2026-10-04T18:30:05+25:00",
        "2026-10-04 18:30:05Z",
    ] {
        assert!(parse_rfc3339_nano(text).is_err(), "admitted {text:?}");
    }
}

#[test]
fn lease_wire_shape_matches_go() {
    let lease = Lease {
        repository_id: 7,
        provider_id: "codex".to_string(),
        id: "lease-1".to_string(),
        connection_id: "conn-1".to_string(),
        generation: 1,
        actor_id: 2,
        project_id: "project".to_string(),
        execution_id: "execution".to_string(),
        kind: "factory".to_string(),
        role: String::new(),
        deadline: UnixTime {
            sec: 1791138605,
            nanos: 0,
        },
        grant_id: String::new(),
        grant_revision: 0,
        binding: None,
    };
    assert_eq!(
        serde_json::to_string(&lease).unwrap(),
        r#"{"repository_id":"7","provider_id":"codex","id":"lease-1","connection_id":"conn-1","generation":1,"actor_id":"2","project_id":"project","execution_id":"execution","kind":"factory","deadline":"2026-10-04T18:30:05Z"}"#
    );
}

#[test]
fn base64_matches_go_byte_form() {
    assert_eq!(base64_bytes::encode(b""), "");
    assert_eq!(base64_bytes::encode(b"f"), "Zg==");
    assert_eq!(base64_bytes::encode(b"fo"), "Zm8=");
    assert_eq!(base64_bytes::encode(b"foo"), "Zm9v");
    assert_eq!(base64_bytes::encode(br#"{"a":1}"#), "eyJhIjoxfQ==");
    for data in [
        b"".as_slice(),
        b"f",
        b"fo",
        b"foo",
        b"foob",
        b"fooba",
        b"foobar",
    ] {
        assert_eq!(
            base64_bytes::decode(&base64_bytes::encode(data)).unwrap(),
            data
        );
    }
    assert!(base64_bytes::decode("Zg").is_err());
    assert!(base64_bytes::decode("Zg=a").is_err());
    assert!(base64_bytes::decode("====").is_err());
}

#[test]
fn null_scalars_match_go_noop() {
    let request: Request = serde_json::from_str(
            r#"{"provider_id":null,"owner_id":null,"id":null,"label":null,"project_id":null,"kind":null,"execution_id":null,"grant":{"connection_id":null,"user_id":null,"project_id":null,"confirm_subscription":null,"confirm_credential_exposure":null}}"#,
        )
        .unwrap();
    assert_eq!(request.owner_id, 0);
    assert!(request.grant.unwrap().connection_id.is_empty());
    let wire: DeliveryWire = serde_json::from_str(
            r#"{"lease":{"provider_id":"codex","id":"l","connection_id":"c","generation":1,"actor_id":"1","project_id":"p","execution_id":"e","kind":"factory","deadline":"2026-10-04T18:30:05Z"},"credential":null}"#,
        )
        .unwrap();
    assert!(wire.credential.is_empty());
    let lease: Lease = serde_json::from_str(
            r#"{"provider_id":"codex","id":"l","connection_id":"c","generation":null,"actor_id":"1","project_id":"p","execution_id":"e","kind":"factory","deadline":null}"#,
        )
        .unwrap();
    assert_eq!(lease.generation, 0);
    assert_eq!((lease.deadline.sec, lease.deadline.nanos), (0, 0));
}

#[test]
fn provider_ids_match_go() {
    assert!(provider_valid("codex"));
    assert!(provider_valid("muse"));
    assert!(!provider_valid("claude"));
    assert!(!provider_valid(""));
}

#[test]
fn digest_matches_go_acquisition() {
    // Reference computed from the Go AcquisitionDigest on the same input.
    let request = AcquireRequest {
        repository_id: 7,
        provider_id: "codex".to_string(),
        execution_id: "execution".to_string(),
        actor_id: 2,
        connection_id: "conn-1".to_string(),
        project_id: "project".to_string(),
        kind: "factory".to_string(),
        deadline: UnixTime {
            sec: 1791138605,
            nanos: 0,
        },
        role: " Soda-Coder ".to_string(),
    };
    // sha256 of the NUL-joined canonical form, verified with sha256sum.
    assert_eq!(
        acquisition_digest(&request),
        "c60bddffb5aa08c8f38f9efd3e8bafc8a633e3923f00baeda96584fea5cbde57"
    );
}
