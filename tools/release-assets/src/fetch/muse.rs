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
    let value =
        JsonValue::parse(text).map_err(|_| "invalid Muse manifest: malformed JSON".to_string())?;
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
                    .ok_or_else(|| "invalid Muse manifest: size must be an integer".to_string())?,
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
    crate::fetch::sha256_hex_stream(&mut file).ok()
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
        drop(file);
        let mut hex = String::with_capacity(64);
        for byte in hasher.finalize() {
            hex.push_str(&format!("{byte:02x}"));
        }
        if size != artifact.size as u64 || hex != artifact.sha256 {
            return Err("muse release checksum or size mismatch".to_string());
        }
        crate::fetch::chmod(&temp_path, 0o755)?;
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
        return crate::fetch::chmod(dest, 0o755);
    }
    let url = download_url(download_base, &version, &artifact.file);
    let mut response = crate::fetch::http_get(&url, USER_AGENT, TIMEOUT)
        .map_err(|e| format!("download Muse: {e}"))?;
    if response.status != 200 {
        return Err("muse download failed".to_string());
    }
    stage(&mut response.reader, &artifact, dest)
}

#[cfg(test)]
mod tests;
