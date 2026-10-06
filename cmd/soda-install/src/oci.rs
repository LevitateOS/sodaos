//! Shared-blob OCI layout verification ported from
//! `internal/release/build` (`oci_layout.go` plus the reachable half of
//! `oci.go`). Only the installer's `InspectOCILayout` path is ported: exact
//! image-set identity, blob hashing, and byte counts. Archive and layer
//! member inspection stay in Go with their callers.

use std::collections::BTreeMap;
use std::ffi::CString;
use std::io::Read;
use std::os::unix::fs::OpenOptionsExt;
use std::os::unix::io::AsRawFd;

use soda_json::JsonValue;

use crate::buildx;
use crate::errors::{self, Error};
use crate::jsongo::parse;

const MANIFEST_MEDIA_TYPE: &str = "application/vnd.oci.image.manifest.v1+json";
const CONFIG_MEDIA_TYPE: &str = "application/vnd.oci.image.config.v1+json";
const INDEX_MEDIA_TYPE: &str = "application/vnd.oci.image.index.v1+json";
const LAYER_TAR: &str = "application/vnd.oci.image.layer.v1.tar";
const LAYER_GZIP: &str = "application/vnd.oci.image.layer.v1.tar+gzip";
const LAYER_ZSTD: &str = "application/vnd.oci.image.layer.v1.tar+zstd";

#[derive(Debug, Clone, Default, PartialEq, Eq)]
pub struct OciImage {
    pub manifest: String,
    pub config: String,
    pub architecture: String,
    pub revision: String,
    pub source: String,
    pub base_name: String,
    pub base_digest: String,
}

#[derive(Debug, Clone, Default)]
pub struct OciLayout {
    pub images: BTreeMap<String, OciImage>,
    pub files: BTreeMap<String, String>,
    pub bytes: u64,
}

#[derive(Debug, Clone, Default)]
struct Blob {
    hash: String,
    size: i64,
    data: Option<Vec<u8>>,
}

#[derive(Debug, Clone, Default)]
struct Descriptor {
    digest: String,
    size: i64,
    media_type: String,
    urls: Vec<String>,
    annotations: BTreeMap<String, String>,
}

struct Loader {
    root: std::fs::File,
    entries: BTreeMap<String, Blob>,
    json_bytes: usize,
}

// ---------------------------------------------------------------------------
// Confined layout reads.
// ---------------------------------------------------------------------------

fn open_root(dir: &str) -> Result<std::fs::File, Error> {
    let st = std::fs::symlink_metadata(dir)
        .map_err(|_| Error::msg("real OCI layout directory required"))?;
    if !st.file_type().is_dir() {
        return Err(Error::msg("real OCI layout directory required"));
    }
    std::fs::OpenOptions::new()
        .read(true)
        .custom_flags(libc::O_DIRECTORY | libc::O_NOFOLLOW)
        .open(dir)
        .map_err(|e| errors::path_error("open", dir, e))
}

/// Open a layout-relative file refusing every symlink, mirroring `os.Root`
/// closely enough for Skopeo-produced layouts (plain directories only).
fn open_layout_file(root: &std::fs::File, name: &str) -> Result<std::fs::File, Error> {
    use std::os::unix::io::FromRawFd;
    let mut dir_fd = root.as_raw_fd();
    // Owned intermediate fds, closed on return.
    let mut owned: Vec<i32> = Vec::new();
    let parts: Vec<&str> = name.split('/').collect();
    for (i, part) in parts.iter().enumerate() {
        if part.is_empty() || *part == "." || *part == ".." {
            for fd in owned {
                unsafe { libc::close(fd) };
            }
            return Err(errors::os_error(std::io::Error::from_raw_os_error(
                libc::ENOENT,
            )));
        }
        let c = CString::new(*part)
            .map_err(|_| errors::os_error(std::io::Error::from_raw_os_error(libc::EINVAL)))?;
        let last = i + 1 == parts.len();
        let flags = if last {
            libc::O_RDONLY | libc::O_NOFOLLOW | libc::O_NONBLOCK | libc::O_CLOEXEC
        } else {
            libc::O_RDONLY | libc::O_DIRECTORY | libc::O_NOFOLLOW | libc::O_CLOEXEC
        };
        let fd = unsafe { libc::openat(dir_fd, c.as_ptr(), flags, 0) };
        if fd < 0 {
            let errno = unsafe { *libc::__errno_location() };
            for fd in owned {
                unsafe { libc::close(fd) };
            }
            return Err(errors::path_error(
                "open",
                name,
                std::io::Error::from_raw_os_error(errno),
            ));
        }
        if last {
            for fd in owned {
                unsafe { libc::close(fd) };
            }
            return Ok(unsafe { std::fs::File::from_raw_fd(fd) });
        }
        owned.push(fd);
        dir_fd = fd;
    }
    for fd in owned {
        unsafe { libc::close(fd) };
    }
    Err(errors::os_error(std::io::Error::from_raw_os_error(
        libc::ENOENT,
    )))
}

fn lstat_layout_file(root: &std::fs::File, name: &str) -> Result<libc::stat, Error> {
    let c = CString::new(name)
        .map_err(|_| errors::os_error(std::io::Error::from_raw_os_error(libc::EINVAL)))?;
    let mut st: libc::stat = unsafe { std::mem::zeroed() };
    if unsafe {
        libc::fstatat(
            root.as_raw_fd(),
            c.as_ptr(),
            &mut st,
            libc::AT_SYMLINK_NOFOLLOW,
        )
    } != 0
    {
        let errno = unsafe { *libc::__errno_location() };
        return Err(errors::path_error(
            "lstat",
            name,
            std::io::Error::from_raw_os_error(errno),
        ));
    }
    Ok(st)
}

fn copy_oci_blob(
    name: &str,
    length: i64,
    reader: &mut dyn Read,
) -> Result<(String, i64, Option<Vec<u8>>), Error> {
    use sha2::Digest as _;
    let mut hasher = sha2::Sha256::new();
    let mut body: Vec<u8> = Vec::new();
    let retain = length <= 4 << 20;
    let mut remaining = length as u64 + 1;
    let mut size: i64 = 0;
    let mut chunk = [0u8; 65536];
    loop {
        if remaining == 0 {
            break;
        }
        let want = remaining.min(chunk.len() as u64) as usize;
        let n = reader
            .read(&mut chunk[..want])
            .map_err(|e| errors::path_error("read", name, e))?;
        if n == 0 {
            break;
        }
        hasher.update(&chunk[..n]);
        if retain {
            body.extend_from_slice(&chunk[..n]);
        }
        size += n as i64;
        remaining -= n as u64;
    }
    if size != length {
        return Err(Error::msg("OCI blob size changed"));
    }
    let sum = buildx::hex_encode(&hasher.finalize());
    if name.starts_with("blobs/") && *name != format!("blobs/sha256/{sum}") {
        return Err(Error::msg("OCI blob checksum mismatch"));
    }
    let body = if retain && parse(&body).is_ok() {
        Some(body)
    } else {
        None
    };
    Ok((sum, size, body))
}

fn read_oci_blob(
    loader: &mut Loader,
    name: &str,
    length: i64,
    reader: &mut dyn Read,
) -> Result<(), Error> {
    if length < 0 || length == i64::MAX {
        return Err(Error::msg("invalid OCI blob size"));
    }
    let (sum, size, body) = copy_oci_blob(name, length, reader)?;
    loader.json_bytes += body.as_ref().map(|b| b.len()).unwrap_or(0);
    if loader.json_bytes > 32 << 20 {
        return Err(Error::msg("OCI JSON metadata limit exceeded"));
    }
    loader.entries.insert(
        name.to_string(),
        Blob {
            hash: sum,
            size,
            data: body,
        },
    );
    Ok(())
}

fn load_layout_file(loader: &mut Loader, name: &str) -> Result<(), Error> {
    if loader.entries.contains_key(name) {
        return Ok(());
    }
    let st = lstat_layout_file(&loader.root, name)?;
    if st.st_mode & libc::S_IFMT != libc::S_IFREG {
        return Err(Error::msg("non-regular OCI layout entry"));
    }
    let file = open_layout_file(&loader.root, name)?;
    let mut opened: libc::stat = unsafe { std::mem::zeroed() };
    if unsafe { libc::fstat(file.as_raw_fd(), &mut opened) } != 0 {
        let errno = unsafe { *libc::__errno_location() };
        return Err(errors::path_error(
            "stat",
            name,
            std::io::Error::from_raw_os_error(errno),
        ));
    }
    if st.st_dev != opened.st_dev || st.st_ino != opened.st_ino {
        return Err(Error::msg("OCI layout entry changed"));
    }
    let mut file = file;
    read_oci_blob(loader, name, st.st_size, &mut file)
}

// ---------------------------------------------------------------------------
// Tolerant decode (index/layout): unknown fields ignored, only success or
// failure escapes through mapped errors.
// ---------------------------------------------------------------------------

fn soft_string(value: &JsonValue) -> Result<String, ()> {
    match value {
        JsonValue::Str(s) => Ok(s.clone()),
        JsonValue::Null => Ok(String::new()),
        _ => Err(()),
    }
}

fn soft_field<'a>(entries: &'a [(String, JsonValue)], name: &str) -> Option<&'a JsonValue> {
    // Go matches struct fields exactly first, then by ASCII/Unicode fold.
    // Index and descriptor keys are fixed camelCase; exact match wins and a
    // folded duplicate loses to it, per key order.
    let mut found: Option<&JsonValue> = None;
    for (key, value) in entries {
        if key == name || (key.len() == name.len() && key.eq_ignore_ascii_case(name)) {
            found = Some(value);
        }
    }
    found
}

fn soft_entries(value: &JsonValue) -> Result<&[(String, JsonValue)], ()> {
    match value {
        JsonValue::Object(entries) => Ok(entries),
        _ => Err(()),
    }
}

fn decode_descriptor_soft(value: &JsonValue) -> Result<Descriptor, ()> {
    let mut desc = Descriptor::default();
    if matches!(value, JsonValue::Null) {
        return Ok(desc);
    }
    let entries = soft_entries(value)?;
    if let Some(v) = soft_field(entries, "digest") {
        desc.digest = soft_string(v)?;
    }
    if let Some(v) = soft_field(entries, "size") {
        match v {
            JsonValue::Null => {}
            JsonValue::Number(_) => {
                desc.size = v
                    .as_integer()
                    .and_then(|n| i64::try_from(n).ok())
                    .ok_or(())?;
            }
            _ => return Err(()),
        }
    }
    if let Some(v) = soft_field(entries, "mediaType") {
        desc.media_type = soft_string(v)?;
    }
    if let Some(v) = soft_field(entries, "urls") {
        match v {
            JsonValue::Null => {}
            JsonValue::Array(items) => {
                for item in items {
                    desc.urls.push(soft_string(item)?);
                }
            }
            _ => return Err(()),
        }
    }
    if let Some(v) = soft_field(entries, "annotations") {
        match v {
            JsonValue::Null => {}
            JsonValue::Object(map) => {
                for (key, val) in map {
                    desc.annotations.insert(key.clone(), soft_string(val)?);
                }
            }
            _ => return Err(()),
        }
    }
    Ok(desc)
}

fn read_oci_index(entries: &BTreeMap<String, Blob>) -> Result<Vec<Descriptor>, Error> {
    let layout_data = entries
        .get("oci-layout")
        .and_then(|b| b.data.as_deref())
        .unwrap_or(b"");
    let layout_value = parse(layout_data).map_err(|_| Error::msg("missing OCI layout"))?;
    let version = soft_entries(&layout_value)
        .ok()
        .and_then(|fields| soft_field(fields, "imageLayoutVersion"))
        .and_then(|v| soft_string(v).ok())
        .unwrap_or_default();
    if version != "1.0.0" {
        return Err(Error::msg("missing OCI layout"));
    }
    let index_data = entries
        .get("index.json")
        .and_then(|b| b.data.as_deref())
        .unwrap_or(b"");
    let index_value = parse(index_data).map_err(|_| Error::msg("valid OCI index required"))?;
    let fields = soft_entries(&index_value).map_err(|_| Error::msg("valid OCI index required"))?;
    let schema = match soft_field(fields, "schemaVersion") {
        None | Some(JsonValue::Null) => 0i64,
        Some(v @ JsonValue::Number(_)) => v
            .as_integer()
            .and_then(|n| i64::try_from(n).ok())
            .ok_or_else(|| Error::msg("valid OCI index required"))?,
        Some(_) => return Err(Error::msg("valid OCI index required")),
    };
    let media = match soft_field(fields, "mediaType") {
        None | Some(JsonValue::Null) => String::new(),
        Some(v) => soft_string(v).map_err(|_| Error::msg("valid OCI index required"))?,
    };
    if schema != 2 || (!media.is_empty() && media != INDEX_MEDIA_TYPE) {
        return Err(Error::msg("valid OCI index required"));
    }
    let manifests = match soft_field(fields, "manifests") {
        None | Some(JsonValue::Null) => Vec::new(),
        Some(JsonValue::Array(items)) => items
            .iter()
            .map(decode_descriptor_soft)
            .collect::<Result<Vec<_>, _>>()
            .map_err(|_| Error::msg("valid OCI index required"))?,
        Some(_) => return Err(Error::msg("valid OCI index required")),
    };
    Ok(manifests)
}

// ---------------------------------------------------------------------------
// Manifest/config decode: Go `UnmarshalTypeError` strings escape raw, so the
// decoder reproduces them exactly.
// ---------------------------------------------------------------------------

fn json_type_name(value: &JsonValue) -> String {
    match value {
        JsonValue::Null => "null".to_string(),
        JsonValue::Bool(_) => "bool".to_string(),
        JsonValue::Str(_) => "string".to_string(),
        JsonValue::Number(raw) => format!("number {raw}"),
        JsonValue::Array(_) => "array".to_string(),
        JsonValue::Object(_) => "object".to_string(),
    }
}

fn type_error(value: &JsonValue, go_type: &str) -> Error {
    Error::msg(format!(
        "json: cannot unmarshal {} into Go value of type {go_type}",
        json_type_name(value)
    ))
}

fn hard_string(value: &JsonValue) -> Result<String, Error> {
    match value {
        JsonValue::Str(s) => Ok(s.clone()),
        JsonValue::Null => Ok(String::new()),
        _ => Err(type_error(value, "string")),
    }
}

fn hard_int(value: &JsonValue) -> Result<i64, Error> {
    match value {
        JsonValue::Null => Ok(0),
        v @ JsonValue::Number(_) => v
            .as_integer()
            .and_then(|n| i64::try_from(n).ok())
            .ok_or_else(|| type_error(value, "int")),
        _ => Err(type_error(value, "int")),
    }
}

fn hard_int64(value: &JsonValue) -> Result<i64, Error> {
    match value {
        JsonValue::Null => Ok(0),
        v @ JsonValue::Number(_) => v
            .as_integer()
            .and_then(|n| i64::try_from(n).ok())
            .ok_or_else(|| type_error(value, "int64")),
        _ => Err(type_error(value, "int64")),
    }
}

fn hard_field<'a>(entries: &'a [(String, JsonValue)], name: &str) -> Option<&'a JsonValue> {
    soft_field(entries, name)
}

fn decode_descriptor_hard(value: &JsonValue) -> Result<Descriptor, Error> {
    let mut desc = Descriptor::default();
    if matches!(value, JsonValue::Null) {
        return Ok(desc);
    }
    let entries = match value {
        JsonValue::Object(entries) => entries,
        _ => return Err(type_error(value, "build.descriptor")),
    };
    if let Some(v) = hard_field(entries, "digest") {
        desc.digest = hard_string(v)?;
    }
    if let Some(v) = hard_field(entries, "size") {
        desc.size = hard_int64(v)?;
    }
    if let Some(v) = hard_field(entries, "mediaType") {
        desc.media_type = hard_string(v)?;
    }
    if let Some(v) = hard_field(entries, "urls") {
        match v {
            JsonValue::Null => {}
            JsonValue::Array(items) => {
                for item in items {
                    desc.urls.push(hard_string(item)?);
                }
            }
            _ => return Err(type_error(v, "[]string")),
        }
    }
    if let Some(v) = hard_field(entries, "annotations") {
        match v {
            JsonValue::Null => {}
            JsonValue::Object(map) => {
                for (key, val) in map {
                    desc.annotations.insert(key.clone(), hard_string(val)?);
                }
            }
            _ => return Err(type_error(v, "map[string]string")),
        }
    }
    Ok(desc)
}

struct OciManifestDoc {
    schema_version: i64,
    media_type: String,
    config: Descriptor,
    layers: Vec<Descriptor>,
}

fn parse_oci_manifest(data: &[u8]) -> Result<OciManifestDoc, Error> {
    let value = parse(data).map_err(|_| Error::msg("unexpected end of JSON input"))?;
    if matches!(value, JsonValue::Null) {
        return Ok(OciManifestDoc {
            schema_version: 0,
            media_type: String::new(),
            config: Descriptor::default(),
            layers: Vec::new(),
        });
    }
    let entries = match &value {
        JsonValue::Object(entries) => entries,
        _ => return Err(type_error(&value, "build.ociManifest")),
    };
    let mut doc = OciManifestDoc {
        schema_version: 0,
        media_type: String::new(),
        config: Descriptor::default(),
        layers: Vec::new(),
    };
    if let Some(v) = hard_field(entries, "schemaVersion") {
        doc.schema_version = hard_int(v)?;
    }
    if let Some(v) = hard_field(entries, "mediaType") {
        doc.media_type = hard_string(v)?;
    }
    if let Some(v) = hard_field(entries, "config") {
        doc.config = decode_descriptor_hard(v)?;
    }
    if let Some(v) = hard_field(entries, "layers") {
        match v {
            JsonValue::Null => {}
            JsonValue::Array(items) => {
                for item in items {
                    doc.layers.push(decode_descriptor_hard(item)?);
                }
            }
            _ => return Err(type_error(v, "[]build.descriptor")),
        }
    }
    if doc.schema_version != 2
        || (!doc.media_type.is_empty() && doc.media_type != MANIFEST_MEDIA_TYPE)
        || doc.config.media_type != CONFIG_MEDIA_TYPE
    {
        return Err(Error::msg("invalid OCI image manifest"));
    }
    Ok(doc)
}

struct OciConfigDoc {
    os: String,
    arch: String,
    rootfs_type: String,
    diff_ids: Vec<String>,
    labels: BTreeMap<String, String>,
}

fn parse_oci_config(data: &[u8]) -> Result<OciConfigDoc, Error> {
    let value = parse(data).map_err(|_| Error::msg("unexpected end of JSON input"))?;
    let mut doc = OciConfigDoc {
        os: String::new(),
        arch: String::new(),
        rootfs_type: String::new(),
        diff_ids: Vec::new(),
        labels: BTreeMap::new(),
    };
    if matches!(value, JsonValue::Null) {
        return Ok(doc);
    }
    let entries = match &value {
        JsonValue::Object(entries) => entries,
        _ => return Err(type_error(&value, "build.ociConfig")),
    };
    if let Some(v) = hard_field(entries, "os") {
        doc.os = hard_string(v)?;
    }
    if let Some(v) = hard_field(entries, "architecture") {
        doc.arch = hard_string(v)?;
    }
    if let Some(v) = hard_field(entries, "rootfs") {
        match v {
            JsonValue::Null => {}
            JsonValue::Object(rootfs) => {
                if let Some(t) = hard_field(rootfs, "type") {
                    doc.rootfs_type = hard_string(t)?;
                }
                if let Some(d) = hard_field(rootfs, "diff_ids") {
                    match d {
                        JsonValue::Null => {}
                        JsonValue::Array(items) => {
                            for item in items {
                                doc.diff_ids.push(hard_string(item)?);
                            }
                        }
                        _ => return Err(type_error(d, "[]string")),
                    }
                }
            }
            _ => return Err(type_error(v, "struct { Type string; DiffIDs []string }")),
        }
    }
    if let Some(v) = hard_field(entries, "config") {
        match v {
            JsonValue::Null => {}
            JsonValue::Object(config) => {
                if let Some(l) = hard_field(config, "Labels") {
                    match l {
                        JsonValue::Null => {}
                        JsonValue::Object(map) => {
                            for (key, val) in map {
                                doc.labels.insert(key.clone(), hard_string(val)?);
                            }
                        }
                        _ => return Err(type_error(l, "map[string]string")),
                    }
                }
            }
            _ => return Err(type_error(v, "struct { Labels map[string]string }")),
        }
    }
    Ok(doc)
}

// ---------------------------------------------------------------------------
// Image inspection.
// ---------------------------------------------------------------------------

fn fetch_oci_blob(loader: &mut Loader, desc: &Descriptor) -> Result<Blob, Error> {
    if desc.size < 0 || !desc.urls.is_empty() {
        return Err(Error::msg("local bounded OCI descriptor required"));
    }
    let hex = match desc.digest.strip_prefix("sha256:") {
        Some(hex) if buildx::digest(hex) => hex,
        _ => return Err(Error::msg("invalid OCI digest")),
    };
    let name = format!("blobs/sha256/{hex}");
    load_layout_file(loader, &name)?;
    match loader.entries.get(&name) {
        Some(blob) if blob.size == desc.size => Ok(blob.clone()),
        _ => Err(Error::msg("missing or wrong-size OCI blob")),
    }
}

fn validate_oci_layers(loader: &mut Loader, layers: &[Descriptor]) -> Result<(), Error> {
    for layer in layers {
        match layer.media_type.as_str() {
            LAYER_TAR | LAYER_GZIP | LAYER_ZSTD => {}
            _ => return Err(Error::msg("unsupported OCI layer media type")),
        }
        fetch_oci_blob(loader, layer)?;
    }
    Ok(())
}

fn validate_oci_rootfs(diff_ids: &[String], layers: &[Descriptor]) -> Result<(), Error> {
    if diff_ids.len() != layers.len() {
        return Err(Error::msg("OCI rootfs/layer count mismatch"));
    }
    for (i, id) in diff_ids.iter().enumerate() {
        let hex = match id.strip_prefix("sha256:") {
            Some(hex) if buildx::digest(hex) => hex,
            _ => return Err(Error::msg("invalid OCI diff ID")),
        };
        let _ = hex;
        if layers[i].media_type == LAYER_TAR && *id != layers[i].digest {
            return Err(Error::msg("uncompressed OCI layer identity mismatch"));
        }
    }
    Ok(())
}

fn validate_oci_attribution(
    labels: &BTreeMap<String, String>,
    want_revision: &str,
) -> Result<(), Error> {
    let rev = labels
        .get("org.opencontainers.image.revision")
        .cloned()
        .unwrap_or_default();
    if !want_revision.is_empty() && rev != want_revision {
        return Err(Error::msg("OCI source revision mismatch"));
    }
    if want_revision.is_empty() {
        return Ok(());
    }
    let base_digest = labels
        .get("org.opencontainers.image.base.digest")
        .cloned()
        .unwrap_or_default();
    let source_ok = labels
        .get("org.opencontainers.image.source")
        .map(|s| s.as_str())
        == Some("https://github.com/LevitateOS/sodaos");
    let base_name_ok = labels
        .get("org.opencontainers.image.base.name")
        .map(|s| !s.is_empty())
        .unwrap_or(false);
    let base_digest_ok = match base_digest.strip_prefix("sha256:") {
        Some(hex) => buildx::digest(hex),
        None => false,
    };
    if !source_ok || !base_name_ok || !base_digest_ok {
        return Err(Error::msg("soda image lacks source/base attribution"));
    }
    Ok(())
}

fn inspect_oci_config(
    config_blob: &Blob,
    layers: &[Descriptor],
    want: &str,
    revision: &str,
    image_digest: &str,
    config_digest: &str,
) -> Result<OciImage, Error> {
    let data = config_blob.data.as_deref().unwrap_or(b"");
    let cfg = parse_oci_config(data)?;
    if cfg.rootfs_type != "layers" {
        return Err(Error::msg("OCI rootfs/layer count mismatch"));
    }
    validate_oci_rootfs(&cfg.diff_ids, layers)?;
    // Compressed layer contents are not extracted here. Their blob identities
    // are checked; native import remains the proof of decompression/rootfs use.
    if cfg.os != "linux" || cfg.arch != want {
        return Err(Error::msg(format!("OCI must be linux/{want}")));
    }
    validate_oci_attribution(&cfg.labels, revision)?;
    Ok(OciImage {
        manifest: image_digest.to_string(),
        config: config_digest.to_string(),
        architecture: cfg.arch,
        revision: cfg
            .labels
            .get("org.opencontainers.image.revision")
            .cloned()
            .unwrap_or_default(),
        source: cfg
            .labels
            .get("org.opencontainers.image.source")
            .cloned()
            .unwrap_or_default(),
        base_name: cfg
            .labels
            .get("org.opencontainers.image.base.name")
            .cloned()
            .unwrap_or_default(),
        base_digest: cfg
            .labels
            .get("org.opencontainers.image.base.digest")
            .cloned()
            .unwrap_or_default(),
    })
}

fn inspect_oci_image(
    loader: &mut Loader,
    image: &Descriptor,
    want: &str,
    revision: &str,
) -> Result<OciImage, Error> {
    let manifest_blob = fetch_oci_blob(loader, image)?;
    let manifest = parse_oci_manifest(manifest_blob.data.as_deref().unwrap_or(b""))?;
    validate_oci_layers(loader, &manifest.layers)?;
    let config_blob = fetch_oci_blob(loader, &manifest.config)?;
    inspect_oci_config(
        &config_blob,
        &manifest.layers,
        want,
        revision,
        &image.digest,
        &manifest.config.digest,
    )
}

fn validate_oci_layout_inputs(
    arch: &str,
    revisions: &BTreeMap<String, String>,
) -> Result<String, Error> {
    let want = buildx::oci_architecture(arch)?;
    if revisions.is_empty() {
        return Err(Error::msg("explicit OCI image set required"));
    }
    for (reference, revision) in revisions {
        let digest_ok = match reference.strip_prefix("sha256:") {
            Some(hex) => buildx::digest(hex),
            None => false,
        };
        if !digest_ok || (!revision.is_empty() && !buildx::revision(revision)) {
            return Err(Error::msg("invalid OCI image selection"));
        }
    }
    Ok(want)
}

fn inspect_layout_descriptor(
    loader: &mut Loader,
    desc: &Descriptor,
    want: &str,
    revisions: &BTreeMap<String, String>,
    images: &BTreeMap<String, OciImage>,
) -> Result<(String, OciImage), Error> {
    let reference = desc
        .annotations
        .get("org.opencontainers.image.ref.name")
        .cloned()
        .unwrap_or_default();
    let revision = match revisions.get(&reference) {
        Some(revision) => revision.clone(),
        None => return Err(Error::msg("unexpected or duplicate OCI image reference")),
    };
    if images
        .get(&reference)
        .map(|image| !image.config.is_empty())
        .unwrap_or(false)
        || desc.media_type != MANIFEST_MEDIA_TYPE
    {
        return Err(Error::msg("unexpected or duplicate OCI image reference"));
    }
    let image = inspect_oci_image(loader, desc, want, &revision)?;
    if image.config != reference {
        return Err(Error::msg("OCI reference differs from config identity"));
    }
    Ok((reference, image))
}

fn inspect_layout_images(
    loader: &mut Loader,
    index: &[Descriptor],
    want: &str,
    revisions: &BTreeMap<String, String>,
) -> Result<BTreeMap<String, OciImage>, Error> {
    if index.len() != revisions.len() {
        return Err(Error::msg("exact OCI image set required"));
    }
    let mut images = BTreeMap::new();
    for desc in index {
        let (reference, image) = inspect_layout_descriptor(loader, desc, want, revisions, &images)?;
        images.insert(reference, image);
    }
    Ok(images)
}

/// Verifies the exact named image set and its local blobs without running
/// image code. References are immutable config IDs, as in our OCI exports.
pub fn inspect_oci_layout(
    dir: &str,
    arch: &str,
    revisions: &BTreeMap<String, String>,
) -> Result<OciLayout, Error> {
    let want = validate_oci_layout_inputs(arch, revisions)?;
    let root = open_root(dir)?;
    let mut loader = Loader {
        root,
        entries: BTreeMap::new(),
        json_bytes: 0,
    };
    for name in ["index.json", "oci-layout"] {
        load_layout_file(&mut loader, name)?;
    }
    let index = read_oci_index(&loader.entries)?;
    let images = inspect_layout_images(&mut loader, &index, &want, revisions)?;
    let mut files = BTreeMap::new();
    let mut bytes: u64 = 0;
    for (name, entry) in &loader.entries {
        files.insert(name.clone(), entry.hash.clone());
        bytes += entry.size as u64;
    }
    Ok(OciLayout {
        images,
        files,
        bytes,
    })
}

#[cfg(test)]
pub mod test_support;

#[cfg(test)]
mod tests {
    use super::test_support::*;
    use super::*;

    #[test]
    fn layout_preserves_identities_and_counts_once() {
        let (dir, revisions) = layout_fixture();
        let got = inspect_oci_layout(&dir, "x86_64", &revisions).unwrap();
        assert_eq!(got.images.len(), 2);
        // index + layout + 2 configs + 2 manifests + 1 shared layer.
        assert_eq!(got.files.len(), 7);
        let mut total: u64 = 0;
        for (name, hash) in &got.files {
            let data = std::fs::read(format!("{dir}/{name}")).unwrap();
            assert_eq!(&buildx::sha256_hex(&data), hash);
            total += data.len() as u64;
        }
        assert_eq!(total, got.bytes);
        for (reference, image) in &got.images {
            assert_eq!(reference, &image.config);
            assert_eq!(image.architecture, "amd64");
            assert_eq!(&revisions[reference], &image.revision);
        }
        assert_eq!(
            inspect_oci_layout(&dir, "aarch64", &revisions)
                .unwrap_err()
                .to_string(),
            "expected x86_64"
        );
        let wrong: BTreeMap<String, String> = revisions
            .keys()
            .map(|k| (k.clone(), "b".repeat(40)))
            .collect();
        assert!(inspect_oci_layout(&dir, "x86_64", &wrong)
            .unwrap_err()
            .to_string()
            .contains("revision mismatch"));
    }

    #[test]
    fn layout_refuses_substitution() {
        for kind in [
            "missing",
            "corrupt",
            "symlink-file",
            "symlink-dir",
            "symlink-root",
            "duplicate-ref",
            "wrong-ref",
            "wrong-size",
            "external-url",
            "empty-index",
            "nested-index",
        ] {
            let (mut dir, revisions) = layout_fixture();
            let before = inspect_oci_layout(&dir, "x86_64", &revisions).unwrap();
            let blob = before
                .files
                .keys()
                .find(|n| n.starts_with("blobs/"))
                .unwrap()
                .clone();
            match kind {
                "missing" => std::fs::remove_file(format!("{dir}/{blob}")).unwrap(),
                "corrupt" => std::fs::write(format!("{dir}/{blob}"), b"corrupted").unwrap(),
                "symlink-file" => {
                    let outside = format!("{dir}-outside-blob");
                    std::fs::rename(format!("{dir}/{blob}"), &outside).unwrap();
                    std::os::unix::fs::symlink(&outside, format!("{dir}/{blob}")).unwrap();
                }
                "symlink-dir" => {
                    let outside = format!("{dir}-outside");
                    std::fs::rename(format!("{dir}/blobs"), &outside).unwrap();
                    std::os::unix::fs::symlink(&outside, format!("{dir}/blobs")).unwrap();
                }
                "symlink-root" => {
                    let link = format!("{dir}-link");
                    std::os::unix::fs::symlink(&dir, &link).unwrap();
                    dir = link;
                }
                _ => {
                    let path = format!("{dir}/index.json");
                    let data = std::fs::read(&path).unwrap();
                    let value = parse(&data).unwrap();
                    let mut entries = match value {
                        JsonValue::Object(entries) => entries,
                        _ => panic!("index shape"),
                    };
                    let manifests = entries.iter_mut().find(|(k, _)| k == "manifests").unwrap();
                    let items = match &mut manifests.1 {
                        JsonValue::Array(items) => items,
                        _ => panic!("manifests shape"),
                    };
                    let first = items[0].clone();
                    match kind {
                        "duplicate-ref" => items[1] = first,
                        "wrong-ref" => {
                            if let JsonValue::Object(fields) = &mut items[0] {
                                fields.retain(|(k, _)| k != "annotations");
                                fields.push((
                                    "annotations".to_string(),
                                    JsonValue::Object(vec![(
                                        "org.opencontainers.image.ref.name".to_string(),
                                        JsonValue::Str("latest".to_string()),
                                    )]),
                                ));
                            }
                        }
                        "wrong-size" => {
                            if let JsonValue::Object(fields) = &mut items[0] {
                                for (k, v) in fields.iter_mut() {
                                    if k == "size" {
                                        *v = JsonValue::Number("1".to_string());
                                    }
                                }
                            }
                        }
                        "external-url" => {
                            if let JsonValue::Object(fields) = &mut items[0] {
                                fields.push((
                                    "urls".to_string(),
                                    JsonValue::Array(vec![JsonValue::Str(
                                        "https://example.invalid/layer".to_string(),
                                    )]),
                                ));
                            }
                        }
                        "empty-index" => *items = Vec::new(),
                        "nested-index" => {
                            if let JsonValue::Object(fields) = &mut items[0] {
                                for (k, v) in fields.iter_mut() {
                                    if k == "mediaType" {
                                        *v = JsonValue::Str(INDEX_MEDIA_TYPE.to_string());
                                    }
                                }
                            }
                        }
                        _ => unreachable!(),
                    }
                    std::fs::write(
                        &path,
                        crate::jsongo::serialize(&JsonValue::Object(entries)).as_bytes(),
                    )
                    .unwrap();
                }
            }
            assert!(
                inspect_oci_layout(&dir, "x86_64", &revisions).is_err(),
                "{kind} accepted"
            );
        }
    }
}
