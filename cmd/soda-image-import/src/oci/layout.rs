use std::collections::HashMap;
use std::fs;
use std::io::Read;
use std::os::unix::fs::MetadataExt;
use std::path::{Path, PathBuf};

use crate::sha256::{hex_lower, Sha256};
use sha2::Digest;

use super::super::is_digest;
use super::super::json::json_valid;
use super::metadata::{OciBlobData, OciDescriptor};

// ---------- shared layout loading (build/oci_layout.go) ----------

pub(crate) struct LayoutLoader {
    pub(crate) dir: PathBuf,
    pub(crate) entries: HashMap<String, OciBlobData>,
    pub(crate) json_bytes: usize,
}

impl LayoutLoader {
    pub(crate) fn load(&mut self, name: &str) -> Result<(), String> {
        if self.entries.contains_key(name) {
            return Ok(());
        }
        let (size, data) = read_layout_blob(&self.dir, name)?;
        self.json_bytes += data.as_ref().map(|d| d.len()).unwrap_or(0);
        if self.json_bytes > 32 << 20 {
            return Err("OCI JSON metadata limit exceeded".to_string());
        }
        self.entries
            .insert(name.to_string(), OciBlobData { size, data });
        Ok(())
    }

    pub(crate) fn fetch(&mut self, desc: &OciDescriptor) -> Result<OciBlobData, String> {
        if desc.size < 0 || !desc.urls.is_empty() {
            return Err("local bounded OCI descriptor required".to_string());
        }
        let hex = match desc.digest.strip_prefix("sha256:") {
            Some(hex) if is_digest(hex) => hex,
            _ => return Err("invalid OCI digest".to_string()),
        };
        let name = format!("blobs/sha256/{hex}");
        self.load(&name)?;
        match self.entries.get(&name) {
            Some(blob) if blob.size == desc.size => Ok(blob.clone()),
            _ => Err("missing or wrong-size OCI blob".to_string()),
        }
    }
}

pub(crate) fn open_layout_root(dir: &Path) -> Result<(), String> {
    match fs::symlink_metadata(dir) {
        Ok(st) if st.file_type().is_dir() => Ok(()),
        _ => Err("real OCI layout directory required".to_string()),
    }
}

/// Confined layout read: no absolute/parent segments, no symlink in any
/// prefix (mirroring `os.Root` confinement), regular final verified
/// unchanged across open, hashed while streaming.
fn read_layout_blob(dir: &Path, name: &str) -> Result<(i64, Option<Vec<u8>>), String> {
    if name.is_empty() || name.starts_with('/') {
        return Err("invalid OCI layout entry".to_string());
    }
    let parts: Vec<&str> = name.split('/').collect();
    let mut current = dir.to_path_buf();
    for (i, part) in parts.iter().enumerate() {
        if part.is_empty() || *part == "." || *part == ".." {
            return Err("invalid OCI layout entry".to_string());
        }
        current.push(part);
        let before = fs::symlink_metadata(&current)
            .map_err(|e| format!("cannot stat OCI layout entry {name}: {e}"))?;
        if i + 1 == parts.len() {
            if !before.file_type().is_file() {
                return Err("non-regular OCI layout entry".to_string());
            }
            return read_blob_bytes(&current, &before, name);
        }
        if !before.file_type().is_dir() {
            return Err(format!(
                "cannot stat OCI layout entry {name}: not a directory"
            ));
        }
    }
    Err("invalid OCI layout entry".to_string())
}

fn read_blob_bytes(
    path: &Path,
    before: &fs::Metadata,
    name: &str,
) -> Result<(i64, Option<Vec<u8>>), String> {
    let length = before.len();
    if length > i64::MAX as u64 {
        return Err("invalid OCI blob size".to_string());
    }
    let file =
        fs::File::open(path).map_err(|e| format!("cannot open OCI layout entry {name}: {e}"))?;
    let after = file
        .metadata()
        .map_err(|e| format!("cannot stat OCI layout entry {name}: {e}"))?;
    if !after.is_file() || after.dev() != before.dev() || after.ino() != before.ino() {
        return Err("OCI layout entry changed".to_string());
    }
    let mut hasher = Sha256::new();
    let buffered = length <= 4 << 20;
    let mut data = Vec::new();
    let mut size: u64 = 0;
    let mut reader = file.take(length + 1);
    let mut chunk = [0u8; 65536];
    loop {
        let n = reader
            .read(&mut chunk)
            .map_err(|e| format!("cannot read OCI layout entry {name}: {e}"))?;
        if n == 0 {
            break;
        }
        size += n as u64;
        hasher.update(&chunk[..n]);
        if buffered {
            data.extend_from_slice(&chunk[..n]);
        }
    }
    if size != length {
        return Err("OCI blob size changed".to_string());
    }
    let sum = hex_lower(&hasher.finalize());
    if name.starts_with("blobs/") && name != format!("blobs/sha256/{sum}").as_str() {
        return Err("OCI blob checksum mismatch".to_string());
    }
    let body = if json_valid(&data) { Some(data) } else { None };
    Ok((length as i64, body))
}

pub(crate) fn validate_oci_layers(
    loader: &mut LayoutLoader,
    layers: &[OciDescriptor],
) -> Result<(), String> {
    for layer in layers {
        match layer.media_type.as_str() {
            "application/vnd.oci.image.layer.v1.tar"
            | "application/vnd.oci.image.layer.v1.tar+gzip"
            | "application/vnd.oci.image.layer.v1.tar+zstd" => {}
            _ => return Err("unsupported OCI layer media type".to_string()),
        }
        loader.fetch(layer)?;
    }
    Ok(())
}

pub(crate) fn validate_oci_rootfs(
    diff_ids: &[String],
    layers: &[OciDescriptor],
) -> Result<(), String> {
    if diff_ids.len() != layers.len() {
        return Err("OCI rootfs/layer count mismatch".to_string());
    }
    for (id, layer) in diff_ids.iter().zip(layers.iter()) {
        match id.strip_prefix("sha256:") {
            Some(hex) if is_digest(hex) => {}
            _ => return Err("invalid OCI diff ID".to_string()),
        }
        if layer.media_type == "application/vnd.oci.image.layer.v1.tar" && id != &layer.digest {
            return Err("uncompressed OCI layer identity mismatch".to_string());
        }
    }
    Ok(())
}

pub(crate) fn validate_oci_attribution(
    labels: &HashMap<String, String>,
    want_revision: &str,
) -> Result<String, String> {
    let revision = labels
        .get("org.opencontainers.image.revision")
        .cloned()
        .unwrap_or_default();
    if !want_revision.is_empty() && revision != want_revision {
        return Err("OCI source revision mismatch".to_string());
    }
    if want_revision.is_empty() {
        return Ok(revision);
    }
    let base_digest = labels
        .get("org.opencontainers.image.base.digest")
        .cloned()
        .unwrap_or_default();
    let base_hex = base_digest.strip_prefix("sha256:").unwrap_or("");
    let source_ok = labels
        .get("org.opencontainers.image.source")
        .map(|s| s.as_str())
        == Some("https://github.com/LevitateOS/sodaos");
    let base_name_ok = labels
        .get("org.opencontainers.image.base.name")
        .map(|s| !s.is_empty())
        .unwrap_or(false);
    if !source_ok || !base_name_ok || !base_digest.starts_with("sha256:") || !is_digest(base_hex) {
        return Err("soda image lacks source/base attribution".to_string());
    }
    Ok(revision)
}
