use super::*;
use crate::fetch::test_server::{Server, TempDir};
use std::collections::HashMap;
use std::io::Write;

fn tarball(entries: &[(&str, &[u8])]) -> Vec<u8> {
    let mut tar = tar::Builder::new(Vec::new());
    for (name, data) in entries {
        let mut header = tar::Header::new_gnu();
        header.set_size(data.len() as u64);
        header.set_mode(0o644);
        header.set_cksum();
        tar.append_data(&mut header, name, *data).unwrap();
    }
    let tarred = tar.into_inner().unwrap();
    let mut encoder = flate2::write::GzEncoder::new(Vec::new(), flate2::Compression::default());
    encoder.write_all(&tarred).unwrap();
    encoder.finish().unwrap()
}

fn integrity(body: &[u8]) -> String {
    format!("sha512-{}", crate::fetch::sha512_base64(body))
}

fn write_lock(dir: &Path, text: &str) -> PathBuf {
    let path = dir.join("terminal-assets.lock.json");
    std::fs::write(&path, text).unwrap();
    path
}

#[test]
fn exact_members_checksums_and_cached_bytes() {
    let body = tarball(&[("package/lib/xterm.mjs", b"synthetic")]);
    let file_sha = crate::fetch::sha256_hex(b"synthetic");
    let mut routes = HashMap::new();
    routes.insert("/fixture.tgz".to_string(), (200, body.clone()));
    let server = Server::start(routes);
    let lock = format!(
        r#"[{{"url":"{}/fixture.tgz","integrity":"{}","files":[{{"member":"package/lib/xterm.mjs","file":"xterm.mjs","sha256":"{file_sha}"}}]}}]"#,
        server.base,
        integrity(&body),
    );
    let scratch = TempDir::new("terminal-ok");
    let lock_path = write_lock(&scratch.path, &lock);
    let out = scratch.path.join("out");
    fetch(&lock_path, &out).unwrap();
    assert_eq!(std::fs::read(out.join("xterm.mjs")).unwrap(), b"synthetic");
    // Second run is served from the verified cache: no new download.
    fetch(&lock_path, &out).unwrap();
    assert_eq!(server.seen().len(), 1);
    assert_eq!(server.seen()[0].user_agent, "SodaOS-build");
    // A changed cache re-downloads; the good archive repairs it.
    std::fs::write(out.join("xterm.mjs"), b"changed").unwrap();
    fetch(&lock_path, &out).unwrap();
    assert_eq!(std::fs::read(out.join("xterm.mjs")).unwrap(), b"synthetic");
    assert_eq!(server.seen().len(), 2);
}

#[test]
fn integrity_failure_keeps_changed_cache_bytes() {
    let body = tarball(&[("package/lib/xterm.mjs", b"synthetic")]);
    let file_sha = crate::fetch::sha256_hex(b"synthetic");
    let mut routes = HashMap::new();
    routes.insert("/fixture.tgz".to_string(), (200, b"bad archive".to_vec()));
    let server = Server::start(routes);
    let lock = format!(
        r#"[{{"url":"{}/fixture.tgz","integrity":"{}","files":[{{"member":"package/lib/xterm.mjs","file":"xterm.mjs","sha256":"{file_sha}"}}]}}]"#,
        server.base,
        integrity(&body),
    );
    let scratch = TempDir::new("terminal-bad");
    let lock_path = write_lock(&scratch.path, &lock);
    let out = scratch.path.join("out");
    std::fs::create_dir_all(&out).unwrap();
    std::fs::write(out.join("xterm.mjs"), b"changed").unwrap();
    assert_eq!(
        fetch(&lock_path, &out).unwrap_err(),
        "terminal archive integrity mismatch"
    );
    assert_eq!(std::fs::read(out.join("xterm.mjs")).unwrap(), b"changed");
}

#[test]
fn lock_uses_last_exact_fields_and_ignores_raw_numeric_metadata() {
    let text = r#"[{"url":1e400,"url":"https://example.invalid/archive.tgz","integrity":"old","integrity":"sha512-good","files":[{"member":false,"member":"pkg/file","file":"old","file":"asset","sha256":"bad","sha256":"good","extra":1e400}]}]"#;
    let items = parse_lock(text).expect("last selected fields decode as strings");
    assert_eq!(items.len(), 1);
    assert_eq!(items[0].url, "https://example.invalid/archive.tgz");
    assert_eq!(items[0].integrity, "sha512-good");
    assert_eq!(items[0].files[0].member, "pkg/file");
    assert_eq!(items[0].files[0].file, "asset");
    assert_eq!(items[0].files[0].sha256, "good");
}

#[test]
fn per_file_digest_mismatch_is_reported() {
    let body = tarball(&[("package/lib/xterm.mjs", b"synthetic")]);
    let mut routes = HashMap::new();
    routes.insert("/fixture.tgz".to_string(), (200, body.clone()));
    let server = Server::start(routes);
    let lock = format!(
        r#"[{{"url":"{}/fixture.tgz","integrity":"{}","files":[{{"member":"package/lib/xterm.mjs","file":"xterm.mjs","sha256":"{}"}}]}}]"#,
        server.base,
        integrity(&body),
        "0".repeat(64),
    );
    let scratch = TempDir::new("terminal-sha");
    let lock_path = write_lock(&scratch.path, &lock);
    assert_eq!(
        fetch(&lock_path, &scratch.path.join("out")).unwrap_err(),
        "terminal asset integrity mismatch"
    );
}

#[test]
fn invalid_members_are_refused() {
    let body = tarball(&[("package/lib/xterm.mjs", b"synthetic")]);
    let good = crate::fetch::sha256_hex(b"synthetic");
    let cases: Vec<(&str, &str, String)> = vec![
        // Missing member.
        ("package/lib/absent.mjs", "absent.mjs", good.clone()),
        // Output escapes the directory.
        ("package/lib/xterm.mjs", "sub/xterm.mjs", good.clone()),
        ("package/lib/xterm.mjs", "/abs.mjs", good),
    ];
    for (index, (member, file, sha)) in cases.iter().enumerate() {
        let mut routes = HashMap::new();
        routes.insert("/fixture.tgz".to_string(), (200, body.clone()));
        let server = Server::start(routes);
        let lock = format!(
            r#"[{{"url":"{}/fixture.tgz","integrity":"{}","files":[{{"member":"{member}","file":"{file}","sha256":"{sha}"}}]}}]"#,
            server.base,
            integrity(&body),
        );
        let scratch = TempDir::new("terminal-member");
        let lock_path = write_lock(&scratch.path, &lock);
        assert_eq!(
            fetch(&lock_path, &scratch.path.join(format!("out-{index}"))).unwrap_err(),
            "invalid terminal distribution member",
            "case {member} -> {file}"
        );
    }
}

#[test]
fn directory_member_is_refused() {
    let mut tar = tar::Builder::new(Vec::new());
    tar.append_dir("package/lib", ".").unwrap();
    let tarred = tar.into_inner().unwrap();
    let mut encoder = flate2::write::GzEncoder::new(Vec::new(), flate2::Compression::default());
    encoder.write_all(&tarred).unwrap();
    let body = encoder.finish().unwrap();
    let mut routes = HashMap::new();
    routes.insert("/fixture.tgz".to_string(), (200, body.clone()));
    let server = Server::start(routes);
    let lock = format!(
        r#"[{{"url":"{}/fixture.tgz","integrity":"{}","files":[{{"member":"package/lib","file":"lib","sha256":"{}"}}]}}]"#,
        server.base,
        integrity(&body),
        "0".repeat(64),
    );
    let scratch = TempDir::new("terminal-dir");
    let lock_path = write_lock(&scratch.path, &lock);
    assert_eq!(
        fetch(&lock_path, &scratch.path.join("out")).unwrap_err(),
        "invalid terminal distribution member"
    );
}

#[test]
fn oversized_archive_is_refused() {
    let body = vec![0x41u8; 10_000_001];
    let mut routes = HashMap::new();
    routes.insert("/fixture.tgz".to_string(), (200, body));
    let server = Server::start(routes);
    // Any integrity pin: the size cap trips first either way. One file
    // forces the fetch (empty file lists never download).
    let lock = format!(
        r#"[{{"url":"{}/fixture.tgz","integrity":"sha512-{}","files":[{{"member":"m","file":"f","sha256":"s"}}]}}]"#,
        server.base,
        "A".repeat(88),
    );
    let scratch = TempDir::new("terminal-big");
    let lock_path = write_lock(&scratch.path, &lock);
    assert_eq!(
        fetch(&lock_path, &scratch.path.join("out")).unwrap_err(),
        "terminal archive integrity mismatch"
    );
}

#[test]
fn empty_file_list_skips_the_download() {
    let routes = HashMap::new();
    let server = Server::start(routes);
    let lock = format!(
        r#"[{{"url":"{}/fixture.tgz","integrity":"sha512-{}","files":[]}}]"#,
        server.base,
        "A".repeat(88),
    );
    let scratch = TempDir::new("terminal-empty");
    let lock_path = write_lock(&scratch.path, &lock);
    let out = scratch.path.join("out");
    fetch(&lock_path, &out).unwrap();
    assert!(server.seen().is_empty());
    assert!(out.is_dir());
}

#[test]
fn malformed_lock_and_http_errors_fail() {
    let scratch = TempDir::new("terminal-malformed");
    let bad = write_lock(&scratch.path, "not json");
    assert!(fetch(&bad, &scratch.path.join("out")).is_err());
    let bad = write_lock(&scratch.path, r#"{"url": 1}"#);
    assert!(fetch(&bad, &scratch.path.join("out")).is_err());
    let mut routes = HashMap::new();
    routes.insert("/fixture.tgz".to_string(), (404, Vec::new()));
    let server = Server::start(routes);
    let lock = format!(
        r#"[{{"url":"{}/fixture.tgz","integrity":"sha512-{}","files":[{{"member":"m","file":"f","sha256":"s"}}]}}]"#,
        server.base,
        "A".repeat(88),
    );
    let lock_path = write_lock(&scratch.path, &lock);
    let err = fetch(&lock_path, &scratch.path.join("out2")).unwrap_err();
    assert!(err.contains("404"), "{err}");
}

#[test]
fn default_lock_is_repo_shaped() {
    let lock = default_lock().unwrap();
    assert!(lock.is_absolute());
    assert!(
        lock.ends_with("tools/release-assets/terminal-assets.lock.json"),
        "{}",
        lock.display()
    );
}
