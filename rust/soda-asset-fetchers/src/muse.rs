//! Checksum-pinned native Muse staging (`tools/soda-fetch-muse` on
//! `internal/release/build` `FetchMuse`).

use std::io::{Read, Write};
use std::path::{Path, PathBuf};
use std::sync::atomic::{AtomicU64, Ordering};
use std::time::Duration;

use soda_build_tools::reader::muse::{valid_muse_artifact, MuseArtifact};
use soda_json::JsonValue;

pub const DOWNLOAD_BASE: &str = "https://lookaside.facebook.com/lookaside/muse/download/";
/// The Go HTTP client sends this by default; the endpoint has only ever
/// been observed with it, so the port keeps it on the wire.
pub const USER_AGENT: &str = "Go-http-client/1.1";
const TIMEOUT: Duration = Duration::from_secs(600);
/// The owner reads the manifest through an 8 KiB capped stream.
const MANIFEST_LIMIT: usize = 8192;

static TEMP_COUNTER: AtomicU64 = AtomicU64::new(0);

/// Go `url.QueryEscape` for the download query: alphanumerics and `-_.~`
/// verbatim, space as `+`, everything else `%XX`.
fn query_escape(value: &str) -> String {
    let mut out = String::new();
    for byte in value.bytes() {
        match byte {
            b'A'..=b'Z' | b'a'..=b'z' | b'0'..=b'9' | b'-' | b'_' | b'.' | b'~' => {
                out.push(byte as char)
            }
            b' ' => out.push('+'),
            _ => out.push_str(&format!("%{byte:02X}")),
        }
    }
    out
}

/// `url.Values{"channel", "file", "version"}.Encode()`: keys sorted,
/// values escaped, appended to the download base.
fn download_url(base: &str, version: &str, file: &str) -> String {
    format!(
        "{base}?channel=muse&file={}&version={}",
        query_escape(file),
        query_escape(version)
    )
}

/// A string field: missing and null decode to zero like Go; any other
/// mistyped value is a manifest error.
fn manifest_string(value: Option<&JsonValue>, what: &str) -> Result<String, String> {
    match value {
        None | Some(JsonValue::Null) => Ok(String::new()),
        Some(JsonValue::Str(text)) => Ok(text.clone()),
        Some(_) => Err(format!("invalid Muse manifest: {what} must be a string")),
    }
}

fn unknown_field(entries: &[(String, JsonValue)], allowed: &[&str]) -> Result<(), String> {
    for (key, _) in entries {
        if !allowed.contains(&key.as_str()) {
            return Err(format!("invalid Muse manifest: unknown field {key:?}"));
        }
    }
    Ok(())
}

/// Load and validate the release pin (`loadMuseRelease`).
fn load_release(manifest: &str, arch: &str) -> Result<(String, MuseArtifact), String> {
    soda_build_tools::reader::oci_architecture(arch).map_err(|e| e.to_string())?;
    let raw = std::fs::read(manifest).map_err(|e| e.to_string())?;
    let capped = if raw.len() > MANIFEST_LIMIT {
        &raw[..MANIFEST_LIMIT]
    } else {
        &raw[..]
    };
    let text =
        std::str::from_utf8(capped).map_err(|_| "invalid Muse manifest: not UTF-8".to_string())?;
    let value = JsonValue::parse(text)
        .map_err(|_| "invalid Muse manifest: malformed JSON".to_string())?;
    let top = match &value {
        JsonValue::Object(entries) => entries,
        _ => return Err("invalid Muse manifest: expected an object".to_string()),
    };
    unknown_field(top, &["version", "artifacts"])?;
    let version = manifest_string(value.get("version"), "version")?;
    let artifacts = match value.get("artifacts") {
        None | Some(JsonValue::Null) => None,
        Some(JsonValue::Object(_)) => Some(value.get("artifacts").expect("object")),
        Some(_) => return Err("invalid Muse manifest: artifacts must be an object".to_string()),
    };
    let artifact = match artifacts.and_then(|a| a.get(arch)) {
        None => return Err("invalid Muse release pin".to_string()),
        Some(JsonValue::Null) => MuseArtifact {
            file: String::new(),
            sha256: String::new(),
            size: 0,
        },
        Some(JsonValue::Object(entries)) => {
            unknown_field(entries, &["file", "sha256", "size"])?;
            let entry = artifacts.and_then(|a| a.get(arch)).expect("entry");
            let file = manifest_string(entry.get("file"), "file")?;
            let sha256 = manifest_string(entry.get("sha256"), "sha256")?;
            let size = match entry.get("size") {
                None | Some(JsonValue::Null) => 0,
                Some(number) => number
                    .as_integer()
                    .and_then(|n| i64::try_from(n).ok())
                    .ok_or_else(|| {
                        "invalid Muse manifest: size must be an integer".to_string()
                    })?,
            };
            MuseArtifact { file, sha256, size }
        }
        Some(_) => return Err("invalid Muse manifest: artifact must be an object".to_string()),
    };
    if !valid_muse_artifact(&version, arch, &artifact) {
        return Err("invalid Muse release pin".to_string());
    }
    Ok((version, artifact))
}

/// The staged digest when `dest` is already a matching regular file
/// (`HashFile`: symlinks and unreadables fall through to a fresh fetch).
fn current_digest(dest: &Path) -> Option<String> {
    let meta = std::fs::symlink_metadata(dest).ok()?;
    if !meta.file_type().is_file() {
        return None;
    }
    let mut file = std::fs::File::open(dest).ok()?;
    crate::sha256_hex_stream(&mut file).ok()
}

fn create_temp(dir: &Path) -> Result<(PathBuf, std::fs::File), String> {
    let pid = std::process::id();
    for _ in 0..100 {
        let id = TEMP_COUNTER.fetch_add(1, Ordering::SeqCst);
        let path = dir.join(format!(".muse-{pid}-{id}"));
        match std::fs::OpenOptions::new()
            .write(true)
            .create_new(true)
            .open(&path)
        {
            Ok(file) => return Ok((path, file)),
            Err(e) if e.kind() == std::io::ErrorKind::AlreadyExists => continue,
            Err(e) => return Err(e.to_string()),
        }
    }
    Err(format!("cannot stage Muse download in {}", dir.display()))
}

/// Stream the body into a sibling temp file, verifying size and digest
/// before the atomic rename (`stageMuseArtifact`).
fn stage(body: &mut dyn Read, artifact: &MuseArtifact, dest: &Path) -> Result<(), String> {
    let parent = dest
        .parent()
        .ok_or_else(|| "muse destination has no parent directory".to_string())?;
    let mut builder = std::fs::DirBuilder::new();
    #[cfg(unix)]
    {
        use std::os::unix::fs::DirBuilderExt;
        builder.mode(0o755);
    }
    builder
        .recursive(true)
        .create(parent)
        .map_err(|e| e.to_string())?;
    let (temp_path, temp_file) = create_temp(parent)?;
    let result = (|| -> Result<(), String> {
        use sha2::Digest;
        let mut file = temp_file;
        let mut hasher = sha2::Sha256::new();
        let cap = (artifact.size as u64).saturating_add(1);
        let mut taken = (&mut *body).take(cap);
        let mut buf = [0u8; 65536];
        let mut size: u64 = 0;
        loop {
            let n = taken.read(&mut buf).map_err(|e| e.to_string())?;
            if n == 0 {
                break;
            }
            file.write_all(&buf[..n]).map_err(|e| e.to_string())?;
            hasher.update(&buf[..n]);
            size += n as u64;
        }
        drop(taken);
        drop(file);
        let mut hex = String::with_capacity(64);
        for byte in hasher.finalize() {
            hex.push_str(&format!("{byte:02x}"));
        }
        if size != artifact.size as u64 || hex != artifact.sha256 {
            return Err("muse release checksum or size mismatch".to_string());
        }
        crate::chmod(&temp_path, 0o755)?;
        std::fs::rename(&temp_path, dest).map_err(|e| e.to_string())?;
        Ok(())
    })();
    if result.is_err() {
        let _ = std::fs::remove_file(&temp_path);
    }
    result
}

pub fn fetch_muse(
    manifest: &str,
    arch: &str,
    dest: &Path,
    download_base: &str,
) -> Result<(), String> {
    if !dest.is_absolute() {
        return Err("absolute Muse destination required".to_string());
    }
    let (version, artifact) = load_release(manifest, arch)?;
    if current_digest(dest).as_deref() == Some(artifact.sha256.as_str()) {
        return crate::chmod(dest, 0o755);
    }
    let url = download_url(download_base, &version, &artifact.file);
    let mut response =
        crate::http_get(&url, USER_AGENT, TIMEOUT).map_err(|e| format!("download Muse: {e}"))?;
    if response.status != 200 {
        return Err("muse download failed".to_string());
    }
    stage(&mut response.reader, &artifact, dest)
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::test_server::{Server, TempDir};
    use std::collections::HashMap;

    const VERSION: &str = "1.4.0-R4161.1";
    const FILE: &str = "muse-x86-linux";

    fn manifest_text(payload: &[u8]) -> String {
        format!(
            r#"{{"version":"{VERSION}","artifacts":{{"x86_64":{{"file":"{FILE}","sha256":"{}","size":{}}}}}}}"#,
            crate::sha256_hex(payload),
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
        crate::chmod(&dest, 0o644).unwrap();
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
            fetch_muse(&fixture.manifest, "bogus-arch", Path::new("relative"), &fixture.base)
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
        let good_sha = crate::sha256_hex(&payload);
        let pins = vec![
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
        let good_sha = crate::sha256_hex(&payload);
        let texts = vec![
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
    fn oversized_manifest_is_refused() {
        let payload = b"verified native bytes".to_vec();
        // A version string running past the 8 KiB read cap truncates
        // mid-value, like the owner's capped stream.
        let text = format!(
            r#"{{"version":"1.4.0-R4161.1{}","artifacts":{{"x86_64":{{"file":"{FILE}","sha256":"{}","size":{}}}}}}}"#,
            "x".repeat(9000),
            crate::sha256_hex(&payload),
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
        let err = fetch_muse(
            missing.to_str().unwrap(),
            "x86_64",
            &dest,
            &fixture.base,
        )
        .unwrap_err();
        assert!(!err.is_empty());
        assert!(fixture.server.seen().is_empty());
    }
}
