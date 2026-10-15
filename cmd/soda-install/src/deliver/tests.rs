use super::*;
use soda_build_tools::payload::Role;

fn fixture() -> Payload {
    serde_json::from_str(include_str!(
        "../../../../lib/release-inputs/fixtures/payload-v4.json"
    ))
    .expect("shared v4 fixture")
}

#[test]
fn payload_validation_matrix() {
    fixture().validate().unwrap();
    let mut bad = fixture();
    bad.format = 3;
    assert!(bad.validate().is_err());
    let mut bad = fixture();
    bad.revision = "dirty".to_string();
    assert!(bad.validate().is_err());
    let mut bad = fixture();
    bad.architecture = "armv7".to_string();
    assert_eq!(bad.validate().unwrap_err().to_string(), "expected x86_64");
    let mut bad = fixture();
    bad.presentation_sha256 = String::new();
    assert!(bad.validate().is_err());
    let mut bad = fixture();
    bad.upgrade_from = vec!["unproved".to_string()];
    assert!(bad.validate().is_err());
    let mut bad = fixture();
    Role::Proxy.binding_mut(&mut bad.images).reference = "not-a-reference".to_string();
    assert!(bad
        .validate()
        .unwrap_err()
        .to_string()
        .contains("invalid proxy image binding"));
    let mut bad = fixture();
    bad.images.forgejo.provenance = soda_build_tools::payload::ImageProvenance::Soda {
        revision: bad.revision.clone(),
        base: "quay.io/other@sha256:".to_string() + &"0".repeat(64),
    };
    assert!(bad
        .validate()
        .unwrap_err()
        .to_string()
        .contains("invalid forgejo image provenance"));
    let mut bad = fixture();
    bad.optional_extensions.tailnet.package_id = "soda".to_string();
    assert!(bad
        .validate()
        .unwrap_err()
        .to_string()
        .contains("invalid tailnet extension binding"));
}

#[test]
fn load_refuses_trailing_data() {
    let scratch = tempfile::Builder::new()
        .prefix("soda-deliver-trailing")
        .tempdir()
        .unwrap();
    let dir = scratch.path().to_str().unwrap();
    let path = std::path::PathBuf::from(format!("{dir}/release.json"));
    let mut raw = producer_payload_bytes(&fixture());
    raw.extend_from_slice(br#" {"unexpected":true}"#);
    std::fs::write(&path, &raw).unwrap();
    assert!(load(path.to_str().unwrap()).is_err());
}

#[test]
fn load_accepts_current_release_image_producer_output() {
    let scratch = tempfile::Builder::new()
        .prefix("soda-deliver-current-producer")
        .tempdir()
        .unwrap();
    let dir = scratch.path().to_str().unwrap();
    let path = std::path::PathBuf::from(format!("{dir}/release.json"));
    let expected = fixture();
    let bytes = producer_payload_bytes(&expected);
    std::fs::write(&path, &bytes).unwrap();

    assert_eq!(load(path.to_str().unwrap()).unwrap(), expected);
    assert!(bytes.ends_with(b"\n"));
    assert_eq!(std::fs::read(&path).unwrap(), bytes);
    assert_eq!(
        buildx::hash_file(path.to_str().unwrap()).unwrap(),
        buildx::sha256_hex(&bytes)
    );
}

#[test]
fn load_refuses_nonproducer_shapes_and_invalid_utf8() {
    let scratch = tempfile::Builder::new()
        .prefix("soda-deliver-invalid-producer")
        .tempdir()
        .unwrap();
    let dir = scratch.path().to_str().unwrap();
    let path = std::path::PathBuf::from(format!("{dir}/release.json"));
    let canonical = producer_payload_bytes(&fixture());
    let text = std::str::from_utf8(&canonical).unwrap();
    let cases = [
        text.replacen(r#""Format": 4,"#, r#""format": 4,"#, 1),
        text.replacen(r#""Format": 4,"#, r#""Format": null,"#, 1),
        text.replacen(
            r#"  "Schema": 1,
"#,
            "",
            1,
        ),
        text.replacen(r#""Schema": 1,"#, r#""Schema": "1","#, 1),
        text.replacen(r#""Format": 4,"#, "\"Format\": 4,\n  \"Unknown\": true,", 1),
        text.replacen(r#""Reference":"#, r#""reference":"#, 1),
    ];
    for invalid in cases {
        std::fs::write(&path, invalid.as_bytes()).unwrap();
        assert_eq!(
            load(path.to_str().unwrap()).unwrap_err().to_string(),
            "invalid payload document"
        );
    }

    let mut invalid_utf8 = canonical;
    let host = b"quay.io";
    let host_start = invalid_utf8
        .windows(host.len())
        .position(|window| window == host)
        .expect("producer Base host");
    invalid_utf8.splice(
        host_start..host_start + host.len(),
        b"quay\xff.io".iter().copied(),
    );
    std::fs::write(&path, invalid_utf8).unwrap();
    assert_eq!(
        load(path.to_str().unwrap()).unwrap_err().to_string(),
        "invalid payload document"
    );
}

fn producer_payload(payload: &Payload) -> Payload {
    // The installer and the release-image producer share the canonical
    // Payload v4 definition; no conversion remains.
    payload.clone()
}

fn producer_payload_bytes(payload: &Payload) -> Vec<u8> {
    let mut bytes = serde_json::to_vec_pretty(&producer_payload(payload)).unwrap();
    bytes.push(b'\n');
    bytes
}
