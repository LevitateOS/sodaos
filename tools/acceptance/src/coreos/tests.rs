use super::*;

fn stream_doc(iso_location: &str) -> String {
    r#"{"architectures": {"x86_64": {"artifacts": {
        "metal": {"formats": {"iso": {"disk": {
            "location": "ISO_LOCATION",
            "signature": "ISO_LOCATION.sig",
            "sha256": "SHA_A"
        }}}},
        "qemu": {"formats": {"qcow2.xz": {"disk": {
            "location": "https://builds.coreos.fedoraproject.org/prod/streams/stable/builds/41.20250101.3.0/x86_64/fedora-coreos-41.20250101.3.0-qemu.x86_64.qcow2.xz",
            "signature": "https://builds.coreos.fedoraproject.org/prod/streams/stable/builds/41.20250101.3.0/x86_64/fedora-coreos-41.20250101.3.0-qemu.x86_64.qcow2.xz.sig",
            "sha256": "SHA_B",
            "uncompressed-sha256": "SHA_C"
        }}}}
    }}}}"#
        .replace("ISO_LOCATION", iso_location)
        .replace("SHA_A", &"a".repeat(64))
        .replace("SHA_B", &"b".repeat(64))
        .replace("SHA_C", &"c".repeat(64))
}

#[test]
fn stream_build_parses_release_and_triples() {
    let location = "https://builds.coreos.fedoraproject.org/prod/streams/stable/builds/41.20250101.3.0/x86_64/fedora-coreos-41.20250101.3.0-live.x86_64.iso";
    let (release, iso, qemu) = resolve_stream_build(stream_doc(location).as_bytes()).unwrap();
    assert_eq!(release, "41.20250101.3.0");
    assert_eq!(iso.url, location);
    assert_eq!(iso.signature_url, format!("{location}.sig"));
    assert_eq!(qemu.uncompressed_sha256, "c".repeat(64));
    assert!(resolve_stream_build(b"not json").is_err());
    assert!(resolve_stream_build(br#"{"architectures": {}}"#).is_err());
    let err =
        resolve_stream_build(stream_doc("https://example.test/no-builds/segment.iso").as_bytes())
            .err()
            .unwrap();
    assert_eq!(
        err.to_string(),
        "stable stream x86_64 ISO location names no release"
    );
    let err = resolve_stream_build(br#"{"architectures": {"x86_64": {"artifacts": {}}}}"#)
        .err()
        .unwrap();
    assert_eq!(err.to_string(), "stable stream lacks x86_64 live ISO");
}

#[test]
fn registry_endpoint_gates() {
    let (endpoint, host) = registry_endpoint("https://quay.io").unwrap();
    assert_eq!(
        endpoint,
        "https://quay.io/v2/fedora/fedora-coreos/manifests/stable"
    );
    assert_eq!(host, "quay.io");
    assert_eq!(
        registry_endpoint("http://quay.io").unwrap_err().to_string(),
        "container registry URL must be HTTPS"
    );
    assert_eq!(
        registry_endpoint("https://").unwrap_err().to_string(),
        "container registry URL must be HTTPS"
    );
}

#[test]
fn status_text_prefers_last_response() {
    let headers =
        "HTTP/1.1 301 Moved Permanently\r\nLocation: x\r\n\r\nHTTP/1.1 404 Not Found\r\n\r\n";
    assert_eq!(status_text(headers), "404 Not Found");
    assert_eq!(status_text("HTTP/2 200\r\n"), "200");
    assert_eq!(status_text("garbage"), "");
}

#[test]
fn fetch_gate_rejects_plain_http_without_network() {
    // No request is issued: the gate fails first.
    let err = fetch_capped(
        &Phase::background(),
        "http://example.test/streams/stable.json",
        8 << 20,
        &[],
    )
    .unwrap_err();
    assert_eq!(err.to_string(), "bounded HTTPS fetch required");
}

#[test]
fn resolve_gate_rejects_plain_http_stream() {
    std::env::set_var(
        "SODA_COREOS_STREAM_URL",
        "http://example.test/streams/stable.json",
    );
    let err = resolve_qemu(&Phase::background(), "x86_64").err().unwrap();
    std::env::remove_var("SODA_COREOS_STREAM_URL");
    assert_eq!(err.to_string(), "CoreOS stream URL must be HTTPS");
    assert!(resolve_qemu(&Phase::background(), "aarch64").is_err());
}
