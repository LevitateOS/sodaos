use super::*;
use crate::fetch::test_server::{Server, TempDir};
use std::collections::HashMap;

const VERSION: &str = "1.4.0-R4161.1";
const FILE: &str = "muse-x86-linux";

fn manifest_text(payload: &[u8]) -> String {
    format!(
        r#"{{"version":"{VERSION}","artifacts":{{"x86_64":{{"file":"{FILE}","sha256":"{}","size":{}}}}}}}"#,
        crate::fetch::sha256_hex(payload),
        payload.len(),
    )
}

fn write_manifest(dir: &Path, text: &str) -> String {
    let path = dir.join("release.json");
    std::fs::write(&path, text).unwrap();
    path.to_str().unwrap().to_string()
}

struct Fixture {
    server: Server,
    base: String,
    manifest: String,
    payload: Vec<u8>,
    scratch: TempDir,
}

impl Fixture {
    /// `received` is served from the download route; the manifest pins
    /// `payload`, so tests flip between tampered and verified bytes.
    fn start(received: Vec<u8>) -> Fixture {
        let payload = b"verified native bytes".to_vec();
        let mut routes = HashMap::new();
        routes.insert("/lookaside/muse/download/".to_string(), (200, received));
        let server = Server::start(routes);
        let scratch = TempDir::new("muse");
        let manifest = write_manifest(&scratch.path, &manifest_text(&payload));
        let base = format!("{}/lookaside/muse/download/", server.base);
        Fixture {
            server,
            base,
            manifest,
            payload,
            scratch,
        }
    }
}

#[cfg(unix)]
fn mode(path: &Path) -> u32 {
    use std::os::unix::fs::PermissionsExt;
    std::fs::metadata(path).unwrap().permissions().mode() & 0o777
}

#[test]
fn query_escape_matches_go() {
    assert_eq!(query_escape("muse-x86-linux"), "muse-x86-linux");
    assert_eq!(query_escape("1.4.0-R4161.1"), "1.4.0-R4161.1");
    assert_eq!(query_escape("a b"), "a+b");
    assert_eq!(query_escape("a/b?c=d&e"), "a%2Fb%3Fc%3Dd%26e");
    assert_eq!(query_escape("~tilde"), "~tilde");
    assert_eq!(
        download_url("https://host/dl/", "1.0.0-R1.1", "f"),
        "https://host/dl/?channel=muse&file=f&version=1.0.0-R1.1"
    );
}

#[test]
fn tampered_bytes_are_rejected_before_publishing() {
    let fixture = Fixture::start(b"tampered native bytes".to_vec());
    let dest = fixture.scratch.path.join("bin/muse");
    assert_eq!(
        fetch_muse(&fixture.manifest, "x86_64", &dest, &fixture.base).unwrap_err(),
        "muse release checksum or size mismatch"
    );
    assert!(!dest.exists());
    // No temp files survive beside the destination.
    let parent = dest.parent().unwrap();
    if parent.exists() {
        let leftovers: Vec<_> = std::fs::read_dir(parent).unwrap().collect();
        assert!(leftovers.is_empty(), "{leftovers:?}");
    }
}

#[test]
fn verified_bytes_stage_executable_with_pinned_query() {
    let fixture = Fixture::start(b"verified native bytes".to_vec());
    let dest = fixture.scratch.path.join("bin/muse");
    fetch_muse(&fixture.manifest, "x86_64", &dest, &fixture.base).unwrap();
    assert_eq!(std::fs::read(&dest).unwrap(), fixture.payload);
    #[cfg(unix)]
    assert_eq!(mode(&dest), 0o755);
    let seen = fixture.server.seen();
    assert_eq!(seen.len(), 1);
    assert_eq!(
        seen[0].target,
        format!("/lookaside/muse/download/?channel=muse&file={FILE}&version={VERSION}")
    );
    assert_eq!(seen[0].user_agent, "Go-http-client/1.1");
}

#[test]
fn matching_destination_is_kept_and_chmodded_without_fetch() {
    let fixture = Fixture::start(b"verified native bytes".to_vec());
    let dest = fixture.scratch.path.join("bin/muse");
    std::fs::create_dir_all(dest.parent().unwrap()).unwrap();
    std::fs::write(&dest, &fixture.payload).unwrap();
    crate::fetch::chmod(&dest, 0o644).unwrap();
    fetch_muse(&fixture.manifest, "x86_64", &dest, &fixture.base).unwrap();
    #[cfg(unix)]
    assert_eq!(mode(&dest), 0o755);
    assert!(fixture.server.seen().is_empty());
}

#[test]
fn stale_destination_is_replaced() {
    let fixture = Fixture::start(b"verified native bytes".to_vec());
    let dest = fixture.scratch.path.join("bin/muse");
    std::fs::create_dir_all(dest.parent().unwrap()).unwrap();
    std::fs::write(&dest, b"stale bytes").unwrap();
    fetch_muse(&fixture.manifest, "x86_64", &dest, &fixture.base).unwrap();
    assert_eq!(std::fs::read(&dest).unwrap(), fixture.payload);
    assert_eq!(fixture.server.seen().len(), 1);
}

#[test]
fn non_200_status_fails_the_download() {
    let mut routes = HashMap::new();
    routes.insert("/lookaside/muse/download/".to_string(), (404, Vec::new()));
    let server = Server::start(routes);
    let scratch = TempDir::new("muse-404");
    let payload = b"verified native bytes".to_vec();
    let manifest = write_manifest(&scratch.path, &manifest_text(&payload));
    let base = format!("{}/lookaside/muse/download/", server.base);
    let dest = scratch.path.join("bin/muse");
    assert_eq!(
        fetch_muse(&manifest, "x86_64", &dest, &base).unwrap_err(),
        "muse download failed"
    );
    assert!(!dest.exists());
}

#[test]
fn relative_destination_is_refused_first() {
    let fixture = Fixture::start(b"verified native bytes".to_vec());
    assert_eq!(
        fetch_muse(
            &fixture.manifest,
            "bogus-arch",
            Path::new("relative"),
            &fixture.base
        )
        .unwrap_err(),
        "absolute Muse destination required"
    );
    assert!(fixture.server.seen().is_empty());
}

#[test]
fn unknown_architecture_is_refused() {
    let fixture = Fixture::start(b"verified native bytes".to_vec());
    let dest = fixture.scratch.path.join("bin/muse");
    assert_eq!(
        fetch_muse(&fixture.manifest, "aarch64", &dest, &fixture.base).unwrap_err(),
        "expected x86_64"
    );
    assert!(fixture.server.seen().is_empty());
    assert!(!dest.exists());
}

#[test]
fn invalid_pins_are_refused_without_fetch() {
    let payload = b"verified native bytes".to_vec();
    let good_sha = crate::fetch::sha256_hex(&payload);
    let pins = [
        // Wrong file.
        format!(
            r#"{{"version":"{VERSION}","artifacts":{{"x86_64":{{"file":"other","sha256":"{good_sha}","size":{}}}}}}}"#,
            payload.len()
        ),
        // Bad digest.
        format!(
            r#"{{"version":"{VERSION}","artifacts":{{"x86_64":{{"file":"{FILE}","sha256":"{}","size":{}}}}}}}"#,
            "z".repeat(64),
            payload.len()
        ),
        // Empty size.
        format!(
            r#"{{"version":"{VERSION}","artifacts":{{"x86_64":{{"file":"{FILE}","sha256":"{good_sha}","size":0}}}}}}"#
        ),
        // Bad version.
        format!(
            r#"{{"version":"1.2","artifacts":{{"x86_64":{{"file":"{FILE}","sha256":"{good_sha}","size":{}}}}}}}"#,
            payload.len()
        ),
        // Missing architecture.
        format!(
            r#"{{"version":"{VERSION}","artifacts":{{"aarch64":{{"file":"{FILE}","sha256":"{good_sha}","size":{}}}}}}}"#,
            payload.len()
        ),
        // Empty manifest object.
        "{}".to_string(),
    ];
    for (index, text) in pins.iter().enumerate() {
        let fixture = Fixture::start(payload.clone());
        let scratch = TempDir::new("muse-pin");
        let manifest = write_manifest(&scratch.path, text);
        let dest = scratch.path.join(format!("bin-{index}/muse"));
        assert_eq!(
            fetch_muse(&manifest, "x86_64", &dest, &fixture.base).unwrap_err(),
            "invalid Muse release pin",
            "pin {index}"
        );
        assert!(fixture.server.seen().is_empty());
        assert!(!dest.exists());
    }
}

#[test]
fn unknown_manifest_fields_are_decode_errors() {
    let payload = b"verified native bytes".to_vec();
    let good_sha = crate::fetch::sha256_hex(&payload);
    let texts = [
        format!(
            r#"{{"version":"{VERSION}","artifacts":{{"x86_64":{{"file":"{FILE}","sha256":"{good_sha}","size":{}}}}},"extra":true}}"#,
            payload.len()
        ),
        format!(
            r#"{{"version":"{VERSION}","artifacts":{{"x86_64":{{"file":"{FILE}","sha256":"{good_sha}","size":{},"digest":"x"}}}}}}"#,
            payload.len()
        ),
        format!(
            r#"{{"version":"{VERSION}","artifacts":{{"x86_64":{{"file":"{FILE}","sha256":"{good_sha}","size":"21"}}}}}}"#
        ),
        r#"{"version": 42, "artifacts": {}}"#.to_string(),
        "[]".to_string(),
        "{broken".to_string(),
    ];
    for (index, text) in texts.iter().enumerate() {
        let fixture = Fixture::start(payload.clone());
        let scratch = TempDir::new("muse-manifest");
        let manifest = write_manifest(&scratch.path, text);
        let dest = scratch.path.join(format!("bin-{index}/muse"));
        let err = fetch_muse(&manifest, "x86_64", &dest, &fixture.base).unwrap_err();
        assert!(
            err.starts_with("invalid Muse manifest"),
            "case {index}: {err}"
        );
        assert!(fixture.server.seen().is_empty());
    }
}

#[test]
fn manifest_selects_last_raw_slots_and_keeps_integer_token_rules() {
    let payload = b"verified native bytes";
    let sha = crate::fetch::sha256_hex(payload);
    let scratch = TempDir::new("muse-raw-slots");
    let manifest = write_manifest(
        &scratch.path,
        &format!(
            r#"{{"version":"bad","version":"{VERSION}","artifacts":{{"x86_64":{{"size":1e400}}}},"artifacts":{{"x86_64":{{"file":"{FILE}","sha256":"{sha}","size":"bad","size":{}}},"aarch64":{{"ignored":1e400}}}}}}"#,
            payload.len(),
        ),
    );
    let (version, artifact) = load_release(&manifest, "x86_64").expect("last slots are selected");
    assert_eq!(version, VERSION);
    assert_eq!(artifact.size, payload.len() as i64);

    std::fs::write(
        &manifest,
        format!(
            r#"{{"version":"{VERSION}","artifacts":{{"x86_64":{{"file":"{FILE}","sha256":"{sha}","size":1e2}}}}}}"#
        ),
    )
    .unwrap();
    assert_eq!(
        load_release(&manifest, "x86_64")
            .err()
            .expect("exponent size rejected"),
        "invalid Muse manifest: size must be an integer"
    );

    std::fs::write(
        &manifest,
        format!(
            r#"{{"version":"{VERSION}","artifacts":{{"x86_64":{{"file":"{FILE}","sha256":"{sha}","size":-0}}}}}}"#
        ),
    )
    .unwrap();
    assert_eq!(
        load_release(&manifest, "x86_64")
            .err()
            .expect("zero size fails pin validation"),
        "invalid Muse release pin"
    );
}

#[test]
fn oversized_manifest_is_refused() {
    let payload = b"verified native bytes".to_vec();
    // A version string running past the 8 KiB read cap truncates
    // mid-value, like the owner's capped stream.
    let text = format!(
        r#"{{"version":"1.4.0-R4161.1{}","artifacts":{{"x86_64":{{"file":"{FILE}","sha256":"{}","size":{}}}}}}}"#,
        "x".repeat(9000),
        crate::fetch::sha256_hex(&payload),
        payload.len(),
    );
    let fixture = Fixture::start(payload);
    let scratch = TempDir::new("muse-big");
    let manifest = write_manifest(&scratch.path, &text);
    let dest = scratch.path.join("bin/muse");
    assert!(fetch_muse(&manifest, "x86_64", &dest, &fixture.base).is_err());
    assert!(fixture.server.seen().is_empty());
}

#[test]
fn transport_errors_carry_the_download_prefix() {
    let scratch = TempDir::new("muse-transport");
    let payload = b"verified native bytes".to_vec();
    let manifest = write_manifest(&scratch.path, &manifest_text(&payload));
    // Nothing listens on this port; the refused connection must surface.
    let base = "http://127.0.0.1:1/lookaside/muse/download/".to_string();
    let dest = scratch.path.join("bin/muse");
    let err = fetch_muse(&manifest, "x86_64", &dest, &base).unwrap_err();
    assert!(err.starts_with("download Muse: "), "{err}");
    assert!(!dest.exists());
}

#[test]
fn missing_manifest_is_an_error() {
    let fixture = Fixture::start(b"verified native bytes".to_vec());
    let dest = fixture.scratch.path.join("bin/muse");
    let missing = fixture.scratch.path.join("absent.json");
    let err = fetch_muse(missing.to_str().unwrap(), "x86_64", &dest, &fixture.base).unwrap_err();
    assert!(!err.is_empty());
    assert!(fixture.server.seen().is_empty());
}
