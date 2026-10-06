use std::io::Write;
use std::os::unix::fs::PermissionsExt;

use soda_json::JsonValue;

use super::*;
use crate::error::Error;

struct Fixture {
    dir: std::path::PathBuf,
    evidence: Evidence,
}

impl Fixture {
    fn new(secrets: &[&str]) -> Fixture {
        let mut dir = std::env::temp_dir();
        dir.push(format!(
            "soda-evidence-{}-{}",
            std::process::id(),
            fresh_id()
        ));
        std::fs::create_dir_all(&dir).unwrap();
        let path = dir.join("evidence").to_string_lossy().into_owned();
        let owned: Vec<Vec<u8>> = secrets.iter().map(|s| s.as_bytes().to_vec()).collect();
        let evidence = create_evidence(&path, &owned).unwrap();
        Fixture { dir, evidence }
    }
}

impl Drop for Fixture {
    fn drop(&mut self) {
        let _ = std::fs::remove_dir_all(&self.dir);
    }
}

fn fresh_id() -> u64 {
    use std::sync::atomic::{AtomicU64, Ordering};
    static NEXT: AtomicU64 = AtomicU64::new(0);
    NEXT.fetch_add(1, Ordering::SeqCst)
}

#[test]
fn split_secrets_and_redirect_queries() {
    let fixture = Fixture::new(&["synthetic-password"]);
    let e = &fixture.evidence;
    let mut writer = e.writer("out").unwrap();
    for part in [
        "synthetic-",
        "pass",
        "word\nhttps://example.test/callback?co",
        "de=unknown-code&state=unknown-state\n",
    ] {
        writer.write_bytes(part.as_bytes()).unwrap();
    }
    writer.close().unwrap();
    let text = std::fs::read_to_string(format!("{}/out", e.path())).unwrap();
    for bad in ["synthetic-password", "unknown-code", "unknown-state"] {
        assert!(!text.contains(bad), "retained unsafe output: {text:?}");
    }
    assert!(text.contains("https://example.test/callback"), "{text:?}");
    e.check_secrets().unwrap();
}

#[test]
fn exclusive_and_confined() {
    let fixture = Fixture::new(&[]);
    let e = &fixture.evidence;
    e.write("once", b"first").unwrap();
    let err = e.write("once", b"second").unwrap_err();
    assert_eq!(err.io_kind(), Some(std::io::ErrorKind::AlreadyExists));
    for name in ["../escape", "/absolute", "a/../b", "."] {
        assert!(e.writer(name).is_err(), "accepted {name:?}");
    }
    let mut link_dir = std::env::temp_dir();
    link_dir.push(format!(
        "soda-evidence-link-{}-{}",
        std::process::id(),
        fresh_id()
    ));
    std::fs::create_dir_all(&link_dir).unwrap();
    std::os::unix::fs::symlink(&link_dir, format!("{}/link", e.path())).unwrap();
    assert!(e.writer("link/out").is_err());
    let root_mode = std::fs::metadata(e.path()).unwrap().permissions().mode() & 0o777;
    assert_eq!(root_mode, 0o700);
    let file_mode = std::fs::metadata(format!("{}/once", e.path()))
        .unwrap()
        .permissions()
        .mode()
        & 0o777;
    assert_eq!(file_mode, 0o600);
    std::fs::remove_dir_all(&link_dir).unwrap();
}

#[test]
fn nested_names_create_parents() {
    let fixture = Fixture::new(&[]);
    let e = &fixture.evidence;
    e.write("sub/dir/entry", b"nested").unwrap();
    let raw = std::fs::read(format!("{}/sub/dir/entry", e.path())).unwrap();
    assert_eq!(raw, b"nested");
}

#[test]
fn redacted_error_retains_identity() {
    let fixture = Fixture::new(&["synthetic-password"]);
    let e = &fixture.evidence;
    let sentinel = Error::msg("underlying");
    let wrapped = Error::wrap(
        "synthetic-password https://example.test/?code=hidden",
        sentinel,
    );
    let redacted = e.redact_error(wrapped);
    assert!(
        !redacted.to_string().contains("synthetic-password"),
        "{redacted}"
    );
    assert!(!redacted.to_string().contains("code="), "{redacted}");
    let mut found = false;
    let mut current: Option<&(dyn std::error::Error + 'static)> =
        std::error::Error::source(&redacted);
    while let Some(cause) = current {
        if cause.to_string() == "underlying" {
            found = true;
        }
        current = cause.source();
    }
    assert!(found, "sentinel lost: {redacted}");
}

#[test]
fn output_bound() {
    let fixture = Fixture::new(&[]);
    let e = &fixture.evidence;
    let mut writer = e.writer("large").unwrap();
    let big = vec![0u8; (EVIDENCE_LIMIT + 1) as usize];
    assert!(writer.write_bytes(&big).is_err());
    assert!(writer.close().is_err());
}

#[test]
fn expanded_redaction_is_bounded_and_sticky() {
    for (name, newline) in [("without-newline", false), ("with-newline", true)] {
        let fixture = Fixture::new(&["a"]);
        let file = fixture.evidence.open_file(name).unwrap();
        let mut writer = RedactingWriter::tee(file, fixture.evidence.secrets().to_vec());
        let mut chunk = vec![b'a'; 128 * 1024];
        if newline {
            chunk.push(b'\n');
        }
        let mut failed = false;
        for _ in 0..20 {
            if writer.write_bytes(&chunk).is_err() {
                failed = true;
                break;
            }
        }
        assert!(failed, "capture did not hit expanded output cap ({name})");
        assert!(writer.close().is_err());
        assert!(writer.close().is_err());
        assert!(writer.buffer().len() as u64 <= EVIDENCE_LIMIT);
        let retained = std::fs::metadata(format!("{}/{name}", fixture.evidence.path()))
            .unwrap()
            .len();
        assert!(
            retained <= EVIDENCE_LIMIT,
            "oversized retained file: {retained}"
        );
        assert!(writer.buffer().len() <= EVIDENCE_LIMIT as usize);
    }
}

#[test]
fn tiny_split_url_stream_keeps_only_safe_components() {
    let fixture = Fixture::new(&["known-secret"]);
    let mut writer = fixture.evidence.writer("split").unwrap();
    let input = b"KEEP\xff known-secret HTTPS://user:pw@h?token=unknown-token https://h/%zz?bad=unknown-percent https://h\\?token=unknown-backslash https://h/path\xff?token=unknown-binary Z\0";
    for byte in input {
        writer.write_bytes(std::slice::from_ref(byte)).unwrap();
    }
    writer.close().unwrap();
    let retained = std::fs::read(format!("{}/split", fixture.evidence.path())).unwrap();
    for sensitive in [
        b"known-secret".as_slice(),
        b"unknown-token",
        b"unknown-percent",
        b"unknown-backslash",
        b"unknown-binary",
        b"user:pw",
    ] {
        assert!(
            !contains_slice(&retained, sensitive),
            "retained {sensitive:?}: {retained:?}"
        );
    }
    assert!(
        contains_slice(&retained, b"KEEP\xff"),
        "unrelated binary prefix lost: {retained:?}"
    );
    assert!(
        retained.ends_with(b" Z\0"),
        "unrelated binary suffix lost: {retained:?}"
    );
    assert!(
        contains_slice(&retained, b"HTTPS://h"),
        "safe authority lost: {retained:?}"
    );
}

#[test]
fn pattern_admission_bounds_escaped_expansion() {
    let mut dir = std::env::temp_dir();
    dir.push(format!(
        "soda-evidence-pattern-{}-{}",
        std::process::id(),
        fresh_id()
    ));
    std::fs::create_dir_all(&dir).unwrap();
    let path = dir.join("evidence").to_string_lossy().into_owned();
    let controls = vec![1u8; 3 * 1024 * 1024];
    assert!(create_evidence(&path, &[controls]).is_err());
    std::fs::remove_dir_all(dir).unwrap();
}

#[test]
fn structured_redaction_refuses_expansion_before_encoding() {
    let fixture = Fixture::new(&["a"]);
    let value = JsonValue::Str("a".repeat(2 * 1024 * 1024));
    assert!(fixture.evidence.write_json("large.json", &value).is_err());
    let observation = JsonValue::Object(vec![("value".to_string(), value)]);
    assert!(fixture.evidence.publish_observation(&observation).is_err());
    assert!(std::fs::read(format!("{}/observation.json", fixture.evidence.path())).is_err());
    assert!(
        std::fs::read(format!(
            "{}/observation.pending.json",
            fixture.evidence.path()
        ))
        .is_err()
    );
}

#[test]
fn incomplete_capture_is_sticky_and_preserves_prior_error() {
    let fixture = Fixture::new(&[]);
    let mut writer = fixture.evidence.writer("partial").unwrap();
    writer.mark_incomplete_capture();
    assert!(
        writer
            .write_bytes(b"later")
            .unwrap_err()
            .to_string()
            .contains("capture incomplete")
    );
    assert!(
        writer
            .close()
            .unwrap_err()
            .to_string()
            .contains("capture incomplete")
    );

    let mut failed = fixture.evidence.writer("failed").unwrap();
    let too_large = vec![0; EVIDENCE_LIMIT as usize + 1];
    let original = failed.write_bytes(&too_large).unwrap_err().to_string();
    failed.mark_incomplete_capture();
    assert_eq!(failed.close().unwrap_err().to_string(), original);
}

#[test]
fn structured_evidence_escapes_and_numeric_identity() {
    let secret = "synthetic-\"credential\\with\nnewline";
    let fixture = Fixture::new(&[secret]);
    let e = &fixture.evidence;
    let input = JsonValue::Object(vec![
        (
            "id".to_string(),
            JsonValue::Number("9223372036854775807".to_string()),
        ),
        ("secret".to_string(), JsonValue::Str(secret.to_string())),
        (
            "url".to_string(),
            JsonValue::Str("https://example.test/path?code=hidden\"".to_string()),
        ),
    ]);
    e.write_json("metadata.json", &input).unwrap();
    let raw = std::fs::read(format!("{}/metadata.json", e.path())).unwrap();
    assert!(!contains_slice(&raw, b"hidden"), "redirect query retained");
    let result = JsonValue::parse(std::str::from_utf8(&raw).unwrap()).unwrap();
    match result.get("id") {
        Some(JsonValue::Number(digits)) => assert_eq!(digits, "9223372036854775807"),
        other => panic!("integer identity changed: {other:?}"),
    }
    assert_eq!(
        result.get("secret").and_then(|v| v.as_str()),
        Some("[REDACTED]")
    );
    e.check_secrets().unwrap();
}

#[test]
fn escaped_credentials_in_raw_split_writes() {
    let secret = "synthetic-\"credential\\line\nend";
    let fixture = Fixture::new(&[secret]);
    let e = &fixture.evidence;
    let mut encoded = String::new();
    crate::jsonio::escape_go(&mut encoded, secret);
    let inner = &encoded[1..encoded.len() - 1];
    let mut writer = e.writer("raw").unwrap();
    for byte in encoded.bytes() {
        writer.write_bytes(&[byte]).unwrap();
    }
    writer.close().unwrap();
    let raw = std::fs::read(format!("{}/raw", e.path())).unwrap();
    assert!(
        !contains_slice(&raw, inner.as_bytes()),
        "encoded secret not redacted"
    );
    assert!(
        contains_slice(&raw, b"[REDACTED]"),
        "encoded secret not redacted"
    );
}

#[test]
fn finalization_does_not_publish_failed_or_occupied_attempts() {
    for mode in ["pending-collision", "leak", "final-collision"] {
        let fixture = Fixture::new(&["synthetic-private-marker"]);
        let e = &fixture.evidence;
        match mode {
            "pending-collision" => {
                e.root.mkdir_at("observation.pending.json", 0o700).unwrap();
            }
            "leak" => {
                let mut file = e.root.create_new_at("unredacted", 0o600).unwrap();
                file.write_all(b"synthetic-private-marker").unwrap();
            }
            _ => {
                let mut file = e.root.create_new_at("observation.json", 0o600).unwrap();
                file.write_all(b"earlier bytes").unwrap();
            }
        }
        let observation = JsonValue::Object(vec![(
            "Outcome".to_string(),
            JsonValue::Str("completed".to_string()),
        )]);
        assert!(
            e.publish_observation(&observation).is_err(),
            "finalized failed attempt ({mode})"
        );
        let raw = std::fs::read(format!("{}/observation.json", e.path()));
        if mode == "final-collision" {
            assert_eq!(raw.unwrap(), b"earlier bytes", "overwrote previous record");
        } else {
            assert!(raw.is_err(), "published success-shaped record ({mode})");
        }
    }
}

#[test]
fn url_shapes_match_go() {
    assert_eq!(redact_urls("no url here"), "no url here");
    assert_eq!(
        redact_urls("see https://example.test/callback?code=x&state=y done"),
        "see https://example.test/callback done"
    );
    assert_eq!(
        redact_urls("http://user@example.test:8080/p#frag"),
        "http://example.test:8080/p"
    );
    assert_eq!(redact_urls("http://[::1]/x"), "http://[::1]/x");
    assert_eq!(redact_urls("http://[::1/x"), "[URL OMITTED]");
    assert_eq!(redact_urls("https://h/%zz"), "[URL OMITTED]");
    assert_eq!(redact_urls("https://h/a%20b?x=1"), "https://h/a%20b");
    assert_eq!(redact_urls("HTTPS://h?token=secret"), "HTTPS://h");
    assert_eq!(redact_urls("http://user:pass@h?token=secret"), "http://h");
    assert_eq!(
        redact_urls("https://h/path\u{00ff}?token=secret"),
        "[URL OMITTED]"
    );
    assert_eq!(redact_urls("https://user%40hidden@h/path"), "[URL OMITTED]");
    assert_eq!(
        redact_urls("https://h/path%5c?token=secret"),
        "[URL OMITTED]"
    );
    assert_eq!(redact_urls("https://h\\?token=secret"), "[URL OMITTED]");
    // Go's URL class keeps single quotes inside the match.
    assert_eq!(
        redact_urls("see https://h/a'b?x=1 done"),
        "see https://h/a'b done"
    );
}

#[test]
fn binary_output_keeps_exact_bytes() {
    let fixture = Fixture::new(&[]);
    let e = &fixture.evidence;
    let mut writer = e.writer("bin").unwrap();
    let payload = b"\xff\xfenot-url-bytes\nhttps://example.test/p?code=x\n\x00\x01trailer";
    writer.write_bytes(payload).unwrap();
    writer.close().unwrap();
    let raw = std::fs::read(format!("{}/bin", e.path())).unwrap();
    assert_eq!(
        raw,
        b"\xff\xfenot-url-bytes\nhttps://example.test/p\n\x00\x01trailer".as_slice()
    );
}
