//! Checksum-pinned native Muse staging (`tools/soda-fetch-muse` on
//! `internal/release/build` `FetchMuse`).

use std::io::{Read, Write};
use std::path::Path;
use std::time::Duration;

use serde::de::{MapAccess, Visitor};
use serde::{Deserialize, Deserializer};
use serde_json::value::RawValue;
use soda_build_tools::reader::muse::{valid_muse_artifact, MuseArtifact};
use url::form_urlencoded::Serializer;

pub const DOWNLOAD_BASE: &str = "https://lookaside.facebook.com/lookaside/muse/download/";
/// Official stable channel document: selects the current release and points
/// at the immutable per-release manifest with platform checksums and sizes.
pub const CHANNEL_URL: &str = "https://api.meta.ai/muse-code/channels/muse-stable";
/// The Go HTTP client sends this by default; the endpoint has only ever
/// been observed with it, so the port keeps it on the wire.
pub const USER_AGENT: &str = "Go-http-client/1.1";
const TIMEOUT: Duration = Duration::from_secs(600);
/// The owner reads the manifest through an 8 KiB capped stream.
const MANIFEST_LIMIT: usize = 8192;
/// Channel and release-manifest documents are small metadata; cap them well
/// above any observed size so a runaway body cannot fill the disk.
const METADATA_LIMIT: u64 = 65536;

/// The endpoint consumes named form pairs; retain its established key order.
fn download_url(base: &str, version: &str, file: &str) -> String {
    let query = Serializer::new(String::new())
        .append_pair("channel", "muse")
        .append_pair("file", file)
        .append_pair("version", version)
        .finish();
    format!("{base}?{query}")
}

/// A string field: missing and null decode to zero like Go; any other
/// mistyped value is a manifest error.
fn manifest_string(value: Option<&RawValue>, what: &str) -> Result<String, String> {
    match value {
        None => Ok(String::new()),
        Some(value) if value.get() == "null" => Ok(String::new()),
        Some(value) => serde_json::from_str(value.get())
            .map_err(|_| format!("invalid Muse manifest: {what} must be a string")),
    }
}

struct RawObject(Vec<(String, Box<RawValue>)>);

impl<'de> Deserialize<'de> for RawObject {
    fn deserialize<D>(deserializer: D) -> Result<Self, D::Error>
    where
        D: Deserializer<'de>,
    {
        struct ObjectVisitor;
        impl<'de> Visitor<'de> for ObjectVisitor {
            type Value = RawObject;
            fn expecting(&self, f: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
                f.write_str("a JSON object")
            }
            fn visit_map<M>(self, mut map: M) -> Result<Self::Value, M::Error>
            where
                M: MapAccess<'de>,
            {
                let mut fields = Vec::with_capacity(map.size_hint().unwrap_or(0));
                while let Some((key, value)) = map.next_entry::<String, Box<RawValue>>()? {
                    fields.push((key, value));
                }
                Ok(RawObject(fields))
            }
        }
        deserializer.deserialize_map(ObjectVisitor)
    }
}

fn last<'a>(fields: &'a RawObject, key: &str) -> Option<&'a RawValue> {
    fields
        .0
        .iter()
        .rev()
        .find(|(name, _)| name == key)
        .map(|(_, value)| value.as_ref())
}

fn unknown_field(entries: &RawObject, allowed: &[&str]) -> Result<(), String> {
    for (key, _) in &entries.0 {
        if !allowed.contains(&key.as_str()) {
            return Err(format!("invalid Muse manifest: unknown field {key:?}"));
        }
    }
    Ok(())
}

fn read_manifest_root(manifest: &str, arch: &str) -> Result<Box<RawValue>, String> {
    soda_build_tools::reader::oci_architecture(arch).map_err(|e| e.to_string())?;
    let raw = std::fs::read(manifest).map_err(|e| e.to_string())?;
    let capped = if raw.len() > MANIFEST_LIMIT {
        &raw[..MANIFEST_LIMIT]
    } else {
        &raw[..]
    };
    let text =
        std::str::from_utf8(capped).map_err(|_| "invalid Muse manifest: not UTF-8".to_string())?;
    let root: Box<RawValue> = serde_json::from_str(text)
        .map_err(|_| "invalid Muse manifest: malformed JSON".to_string())?;
    if root.get().as_bytes()[0] != b'{' {
        return Err("invalid Muse manifest: expected an object".to_string());
    }
    Ok(root)
}

fn decode_artifact(raw: &RawValue) -> Result<MuseArtifact, String> {
    let entry: RawObject = serde_json::from_str(raw.get())
        .map_err(|_| "invalid Muse manifest: artifact must be an object".to_string())?;
    unknown_field(&entry, &["file", "sha256", "size"])?;
    let file = manifest_string(last(&entry, "file"), "file")?;
    let sha256 = manifest_string(last(&entry, "sha256"), "sha256")?;
    let size = match last(&entry, "size") {
        None => 0,
        Some(value) if value.get() == "null" => 0,
        Some(number) => number
            .get()
            .parse::<i64>()
            .map_err(|_| "invalid Muse manifest: size must be an integer".to_string())?,
    };
    Ok(MuseArtifact { file, sha256, size })
}

fn select_artifact(value: &RawObject, arch: &str) -> Result<MuseArtifact, String> {
    let artifacts = match last(value, "artifacts") {
        None => None,
        Some(value) if value.get() == "null" => None,
        Some(raw) => Some(
            serde_json::from_str::<RawObject>(raw.get())
                .map_err(|_| "invalid Muse manifest: artifacts must be an object".to_string())?,
        ),
    };
    let selected = artifacts.as_ref().and_then(|a| last(a, arch));
    match selected {
        None => Err("invalid Muse release pin".to_string()),
        Some(raw) if raw.get() == "null" => Ok(MuseArtifact {
            file: String::new(),
            sha256: String::new(),
            size: 0,
        }),
        Some(raw) => decode_artifact(raw),
    }
}

/// Load and validate the release pin (`loadMuseRelease`).
fn load_release(manifest: &str, arch: &str) -> Result<(String, MuseArtifact), String> {
    let root = read_manifest_root(manifest, arch)?;
    let value: RawObject = serde_json::from_str(root.get())
        .map_err(|_| "invalid Muse manifest: malformed JSON".to_string())?;
    unknown_field(&value, &["version", "artifacts"])?;
    let version = manifest_string(last(&value, "version"), "version")?;
    let artifact = select_artifact(&value, arch)?;
    if !valid_muse_artifact(arch, &artifact) {
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
    let mut staged = tempfile::Builder::new()
        .prefix(".muse-")
        .tempfile_in(parent)
        .map_err(|e| e.to_string())?;
    let result = (|| -> Result<(), String> {
        use sha2::Digest;
        let file = staged.as_file_mut();
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
        let mut hex = String::with_capacity(64);
        for byte in hasher.finalize() {
            hex.push_str(&format!("{byte:02x}"));
        }
        if size != artifact.size as u64 || hex != artifact.sha256 {
            return Err("muse release checksum or size mismatch".to_string());
        }
        use std::os::unix::fs::PermissionsExt;
        staged
            .as_file()
            .set_permissions(std::fs::Permissions::from_mode(0o755))
            .map_err(|e| e.to_string())?;
        staged.persist(dest).map_err(|e| e.error.to_string())?;
        Ok(())
    })();
    result
}

/// Outcome of comparing the pinned manifest against live channel metadata.
/// `Current` means the pin selects exactly the upstream release; `Stale`
/// carries the human-readable drift; `Unknown` means upstream gave no usable
/// selection, so the baseline must not be treated as current.
pub enum Baseline {
    Current { version: String },
    Stale { detail: String },
    Unknown { reason: String },
}

/// One official upstream release for a Soda build architecture.
pub struct LiveRelease {
    pub version: String,
    pub artifact: MuseArtifact,
}

/// Map a Soda build architecture to the channel manifest's artifact key.
/// Only architectures the admission path validates are selectable; anything
/// else is an explicit error, never a guessed neighboring platform.
fn channel_artifact_key(arch: &str) -> Result<&'static str, String> {
    match arch {
        "x86_64" => Ok("x86_linux"),
        _ => Err("unsupported Muse architecture".to_string()),
    }
}

fn get_text(url: &str) -> Result<(u16, String), String> {
    let mut response =
        crate::fetch::http_get(url, USER_AGENT, TIMEOUT).map_err(|e| format!("request failed: {e}"))?;
    let status = response.status;
    let body = crate::fetch::read_capped(response.reader.as_mut(), METADATA_LIMIT)
        .map_err(|e| format!("request failed: {e}"))?;
    let text = String::from_utf8(body)
        .map_err(|_| "unknown: upstream metadata is not UTF-8".to_string())?;
    Ok((status, text))
}

fn parse_manifest_url(channel_body: &str) -> Result<String, String> {
    let value: serde_json::Value =
        serde_json::from_str(channel_body).map_err(|_| "upstream channel is malformed".to_string())?;
    match value.get("manifest_url").and_then(serde_json::Value::as_str) {
        Some(url) if !url.is_empty() => Ok(url.to_string()),
        _ => Err("upstream channel carries no manifest".to_string()),
    }
}

fn artifact_file_from_url(url: &str) -> Result<String, String> {
    let query = url.split_once('?').map(|(_, q)| q).unwrap_or("");
    for pair in query.split('&') {
        if let Some(file) = pair.strip_prefix("file=") {
            if !file.is_empty() {
                return Ok(file.to_string());
            }
        }
    }
    Err("upstream artifact carries no file reference".to_string())
}

/// Channel document URL: the official stable channel unless an integration
/// fixture selects another document root via `SODA_MUSE_CHANNEL_URL`
/// (mirroring `SODA_FOUNTAIN_RELEASE_URL`). Production never sets it.
fn channel_url() -> String {
    std::env::var("SODA_MUSE_CHANNEL_URL").unwrap_or_else(|_| CHANNEL_URL.to_string())
}

/// Resolve the official current release for `arch` from live channel
/// metadata only. Transport failures come back as `Err` (existing pins are
/// kept, freshness stays unproven); unusable upstream answers come back as
/// `Unknown` (the baseline must not be selected as current).
pub fn resolve_live(arch: &str) -> Result<LiveRelease, String> {
    let url = channel_url();
    resolve_live_from(&url, arch)
}

fn resolve_live_from(channel_url: &str, arch: &str) -> Result<LiveRelease, String> {
    let key = channel_artifact_key(arch)?;
    let (status, channel_body) = get_text(channel_url)?;
    if status != 200 {
        return Err(format!("unknown: upstream channel answered HTTP {status}"));
    }
    let manifest_url = parse_manifest_url(&channel_body).map_err(|e| format!("unknown: {e}"))?;
    let (manifest_status, manifest_body) = get_text(&manifest_url)?;
    if manifest_status != 200 {
        return Err(format!("unknown: upstream manifest answered HTTP {manifest_status}"));
    }
    parse_live_release(&manifest_body, key)
}

fn parse_live_release(manifest_body: &str, key: &str) -> Result<LiveRelease, String> {
    let manifest: serde_json::Value = serde_json::from_str(manifest_body)
        .map_err(|_| "unknown: upstream manifest is malformed".to_string())?;
    let version = manifest
        .get("version")
        .and_then(serde_json::Value::as_str)
        .filter(|v| !v.is_empty())
        .ok_or_else(|| "unknown: upstream manifest carries no version".to_string())?;
    let entry = manifest
        .get("artifacts")
        .and_then(|a| a.get(key))
        .ok_or_else(|| format!("unknown: upstream manifest has no {key} artifact"))?;
    let url = entry
        .get("url")
        .and_then(serde_json::Value::as_str)
        .ok_or_else(|| "unknown: upstream artifact carries no URL".to_string())?;
    let checksum = entry
        .get("checksum")
        .and_then(serde_json::Value::as_str)
        .unwrap_or("");
    let size = entry.get("size").and_then(serde_json::Value::as_i64).unwrap_or(0);
    let file = artifact_file_from_url(url).map_err(|e| format!("unknown: {e}"))?;
    let artifact = MuseArtifact {
        file,
        sha256: checksum.to_string(),
        size,
    };
    Ok(LiveRelease {
        version: version.to_string(),
        artifact,
    })
}

/// Compare the pinned manifest against the live official release.
/// The pin is read-only here; only `refresh_muse` writes, and only when it
/// resolved a usable upstream release.
pub fn check_muse(manifest: &str, arch: &str) -> Result<Baseline, String> {
    let (version, artifact) = load_release(manifest, arch)?;
    let live = match resolve_live(arch) {
        Ok(live) => live,
        Err(e) => {
            return match e.strip_prefix("unknown: ") {
                Some(reason) => Ok(Baseline::Unknown {
                    reason: reason.to_string(),
                }),
                None => Err(e),
            }
        }
    };
    if !valid_live(&live.artifact) {
        return Ok(Baseline::Unknown {
            reason: String::from("upstream artifact failed validation"),
        });
    }
    if live.version == version
        && live.artifact.file == artifact.file
        && live.artifact.sha256 == artifact.sha256
        && live.artifact.size == artifact.size
    {
        return Ok(Baseline::Current { version });
    }
    let mut drift = vec![format!("pinned {version} != upstream {}", live.version)];
    if live.artifact.sha256 != artifact.sha256 {
        drift.push(format!(
            "pinned sha256 {} != upstream {}",
            artifact.sha256, live.artifact.sha256
        ));
    }
    if live.artifact.size != artifact.size {
        drift.push(format!(
            "pinned size {} != upstream {}",
            artifact.size, live.artifact.size
        ));
    }
    if live.artifact.file != artifact.file {
        drift.push(format!(
            "pinned file {} != upstream {}",
            artifact.file, live.artifact.file
        ));
    }
    Ok(Baseline::Stale {
        detail: drift.join("; "),
    })
}

fn valid_live(artifact: &MuseArtifact) -> bool {
    // The manifest's own validation owns the supported set; reuse it so the
    // check can never bless an artifact the fetcher would reject.
    valid_muse_artifact("x86_64", artifact)
}

/// Resolve the live official release and record it in the manifest.
/// Returns the selected version. Writes only on a validated upstream
/// release; transport or upstream failures leave the file untouched, and an
/// already-current pin causes no rewrite.
pub fn refresh_muse(manifest: &str, arch: &str) -> Result<String, String> {
    if arch.is_empty() {
        return Err("explicit refresh requires --arch".to_string());
    }
    let live = match resolve_live(arch) {
        Ok(live) => live,
        Err(e) => {
            return match e.strip_prefix("unknown: ") {
                Some(reason) => Err(format!("refusing to record an unknown baseline: {reason}")),
                None => Err(e),
            }
        }
    };
    if !valid_live(&live.artifact) {
        return Err(String::from(
            "refusing to record an unknown baseline: upstream artifact failed validation",
        ));
    }
    let root = read_manifest_root(manifest, arch)?;
    let mut value: serde_json::Value = serde_json::from_str(root.get())
        .map_err(|_| "invalid Muse manifest: malformed JSON".to_string())?;
    let current = load_release(manifest, arch)?;
    if current.0 == live.version
        && current.1.file == live.artifact.file
        && current.1.sha256 == live.artifact.sha256
        && current.1.size == live.artifact.size
    {
        return Ok(live.version);
    }
    let entry = serde_json::json!({
        "file": live.artifact.file,
        "sha256": live.artifact.sha256,
        "size": live.artifact.size,
    });
    value["version"] = serde_json::Value::String(live.version.clone());
    match value.get_mut("artifacts") {
        Some(serde_json::Value::Object(artifacts)) => {
            artifacts.insert(arch.to_string(), entry);
        }
        _ => return Err("invalid Muse manifest: artifacts must be an object".to_string()),
    }
    let mut text = serde_json::to_string_pretty(&value)
        .map_err(|_| "cannot encode Muse manifest".to_string())?;
    text.push('\n');
    atomic_write(manifest, text.as_bytes())?;
    Ok(live.version)
}

/// Replace the manifest atomically (sibling temp file, fsync, rename) so a
/// failed refresh never leaves a partial descriptor behind.
fn atomic_write(manifest: &str, text: &[u8]) -> Result<(), String> {
    use std::io::Write;
    let dest = Path::new(manifest);
    let parent = dest
        .parent()
        .filter(|p| !p.as_os_str().is_empty())
        .ok_or_else(|| "muse manifest has no parent directory".to_string())?;
    let mut staged = tempfile::Builder::new()
        .prefix(".muse-release-")
        .tempfile_in(parent)
        .map_err(|e| e.to_string())?;
    staged.write_all(text).map_err(|e| e.to_string())?;
    staged.as_file().sync_all().map_err(|e| e.to_string())?;
    staged.persist(dest).map_err(|e| e.error.to_string())?;
    Ok(())
}

/// The frozen verified baseline tuple a running build consumes: the exact
/// upstream release the default admission validated before staging bytes.
pub struct Admitted {
    pub version: String,
    pub sha256: String,
    pub size: i64,
}

/// Why default admission refused the cached/descriptor bytes.
pub enum FetchRefusal {
    Stale(String),
    Unknown(String),
    Failed(String),
}

/// Default build admission: resolve live official metadata and stage the
/// pinned bytes ONLY when the pin is exactly current. A stale pin is refused
/// (record the current release with explicit `--refresh` first); an unusable
/// upstream refuses any current-baseline claim, so ordinary builds can no
/// longer depend on the operator remembering `--check`. The returned tuple is
/// frozen for the running build; explicit source refresh before a new source
/// snapshot stays a separate write-mode action.
pub fn admit_current(
    manifest: &str,
    arch: &str,
    dest: &Path,
    download_base: &str,
) -> Result<Admitted, FetchRefusal> {
    if !dest.is_absolute() {
        return Err(FetchRefusal::Failed(
            "absolute Muse destination required".to_string(),
        ));
    }
    match check_muse(manifest, arch).map_err(FetchRefusal::Failed)? {
        Baseline::Current { .. } => {}
        Baseline::Stale { detail } => {
            return Err(FetchRefusal::Stale(format!(
                "{detail}; run soda-fetch-muse --refresh to record the current release"
            )))
        }
        Baseline::Unknown { reason } => return Err(FetchRefusal::Unknown(reason)),
    }
    let (version, artifact) = load_release(manifest, arch).map_err(FetchRefusal::Failed)?;
    if current_digest(dest).as_deref() != Some(artifact.sha256.as_str()) {
        let url = download_url(download_base, &version, &artifact.file);
        let mut response = crate::fetch::http_get(&url, USER_AGENT, TIMEOUT)
            .map_err(|e| FetchRefusal::Failed(format!("download Muse: {e}")))?;
        if response.status != 200 {
            return Err(FetchRefusal::Failed("muse download failed".to_string()));
        }
        stage(&mut response.reader, &artifact, dest).map_err(FetchRefusal::Failed)?;
    } else {
        crate::fetch::chmod(dest, 0o755).map_err(FetchRefusal::Failed)?;
    }
    Ok(Admitted {
        version,
        sha256: artifact.sha256,
        size: artifact.size,
    })
}

/// Backwards-compatible default fetch: gated admission, string errors.
/// Prefer [`admit_current`] when exit codes must distinguish stale/unknown.
pub fn fetch_muse(
    manifest: &str,
    arch: &str,
    dest: &Path,
    download_base: &str,
) -> Result<(), String> {
    admit_current(manifest, arch, dest, download_base)
        .map(|_| ())
        .map_err(|refusal| match refusal {
            FetchRefusal::Stale(detail) => format!("stale baseline: {detail}"),
            FetchRefusal::Unknown(reason) => format!("unknown baseline: {reason}"),
            FetchRefusal::Failed(error) => error,
        })
}
