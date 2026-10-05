//! OCI archive inspection (`oci.go`): identity checks over real OCI
//! archives without extracting layers. Streams the tar, verifies blob
//! digests, and resolves exact member hashes through the layer overlay.
//!
//! Identity checks adapted from soda-os bc1d3e0 release/inspection.go.

use crate::files::{is_digest, oci_architecture};
use crate::json_go::{FieldError, Fields};
use crate::{io_error, path_clean, Error};
use sha2::Digest;
use soda_json::JsonValue;
use std::collections::{BTreeMap, HashMap};
use std::fs::File as FsFile;
use std::io::Read;
use std::path::Path;

const MANIFEST_MEDIA_TYPE: &str = "application/vnd.oci.image.manifest.v1+json";
const INDEX_MEDIA_TYPE: &str = "application/vnd.oci.image.index.v1+json";
const CONFIG_MEDIA_TYPE: &str = "application/vnd.oci.image.config.v1+json";
const LAYER_TAR: &str = "application/vnd.oci.image.layer.v1.tar";
const LAYER_TAR_GZIP: &str = "application/vnd.oci.image.layer.v1.tar+gzip";
const LAYER_TAR_ZSTD: &str = "application/vnd.oci.image.layer.v1.tar+zstd";

/// Verified image identity.
#[derive(Debug, Clone, Default, PartialEq, Eq)]
pub struct Image {
    pub manifest: String,
    pub config: String,
    pub architecture: String,
    pub revision: String,
    pub source: String,
    pub base_name: String,
    pub base_digest: String,
}

/// On-demand blob loader for layouts; archives pass a no-op.
pub(crate) type LoadBlobs<'a> =
    &'a mut dyn FnMut(&mut HashMap<String, Blob>, &str) -> Result<(), Error>;

#[derive(Debug, Clone, Default)]
pub(crate) struct Descriptor {
    pub digest: String,
    pub size: i64,
    pub media_type: String,
    pub urls: Vec<String>,
    pub annotations: Vec<(String, String)>,
}

impl Descriptor {
    fn decode(fields: &Fields<'_>) -> Result<Descriptor, FieldError> {
        Ok(Descriptor {
            digest: fields.string("digest")?,
            size: fields.int("size")?,
            media_type: fields.string("mediaType")?,
            urls: fields.string_list("urls")?,
            annotations: fields.string_map("annotations")?,
        })
    }

    pub fn annotation(&self, key: &str) -> Option<&str> {
        self.annotations
            .iter()
            .rev()
            .find(|(k, _)| k == key)
            .map(|(_, v)| v.as_str())
    }
}

pub(crate) struct Blob {
    pub hash: String,
    pub size: i64,
    pub data: Option<Vec<u8>>,
}

#[derive(Debug, Clone, Default)]
struct LayerMember {
    hash: String,
    present: bool,
    blocked: bool,
}

/// Per-layer member scans plus per-layer zstd-blocked flags.
type LayerScans = (Vec<HashMap<String, LayerMember>>, Vec<bool>);

fn open_oci_archive(file: &Path, arch: &str, revision: &str) -> Result<(String, FsFile), Error> {
    let want = oci_architecture(arch)?.to_string();
    if !revision.is_empty() && !crate::files::is_revision(revision) {
        return Err(Error::msg("full source revision required"));
    }
    let st = std::fs::symlink_metadata(file).map_err(|e| io_error("lstat", file, e))?;
    if !st.is_file() || st.file_type().is_symlink() {
        return Err(Error::msg("OCI archive must be regular"));
    }
    let f = FsFile::open(file).map_err(|e| io_error("open", file, e))?;
    Ok((want, f))
}

fn check_tar_entry_name(name: &str, is_dir: bool) -> Result<bool, Error> {
    if name == "." && is_dir {
        return Ok(true);
    }
    if name.starts_with('/') || name == ".." || name.starts_with("../") {
        return Err(Error::msg("unsafe OCI path"));
    }
    Ok(false)
}

fn check_tar_directory(name: &str, is_dir: bool) -> Result<bool, Error> {
    if !is_dir {
        return Ok(false);
    }
    if name != "blobs" && name != "blobs/sha256" {
        return Err(Error::msg("unexpected OCI directory"));
    }
    Ok(true)
}

fn is_valid_oci_regular_entry(name: &str) -> bool {
    if name == "index.json" || name == "oci-layout" {
        return true;
    }
    match name.strip_prefix("blobs/sha256/") {
        Some(hex) => is_digest(hex),
        None => false,
    }
}

fn record_tar_entry(seen: &mut HashMap<String, ()>, name: &str) -> Result<(), Error> {
    if seen.contains_key(name) {
        return Err(Error::msg("duplicate OCI entry"));
    }
    seen.insert(name.to_string(), ());
    if seen.len() > 100_000 {
        return Err(Error::msg("too many OCI entries"));
    }
    Ok(())
}

/// Reads exactly `length` bytes, hashing throughout and retaining the body
/// only for small valid-JSON blobs.
fn copy_oci_blob(
    name: &str,
    length: i64,
    reader: &mut dyn Read,
) -> Result<(String, i64, Option<Vec<u8>>), Error> {
    use sha2::Digest;
    let mut hasher = sha2::Sha256::new();
    let mut data = Vec::new();
    let retain = length <= 4 << 20;
    let mut remaining = length;
    let mut buf = [0u8; 32 << 10];
    loop {
        if remaining <= 0 {
            break;
        }
        // Read one byte past the declared size to detect growth, like Go's
        // LimitReader(length+1) + size comparison.
        let want = (remaining + 1).min(buf.len() as i64) as usize;
        let n = reader
            .read(&mut buf[..want])
            .map_err(|e| Error::msg(e.to_string()))?;
        if n == 0 {
            break;
        }
        hasher.update(&buf[..n]);
        if retain {
            data.extend_from_slice(&buf[..n]);
        }
        remaining -= n as i64;
    }
    let size = length - remaining;
    if size != length {
        return Err(Error::msg("OCI blob size changed"));
    }
    let digest = {
        let mut out = String::with_capacity(64);
        for byte in hasher.finalize() {
            out.push_str(&format!("{byte:02x}"));
        }
        out
    };
    if name.starts_with("blobs/") && name != format!("blobs/sha256/{digest}").as_str() {
        return Err(Error::msg("OCI blob checksum mismatch"));
    }
    let body = if is_json_bytes(&data) {
        Some(data)
    } else {
        None
    };
    Ok((digest, size, body))
}

fn is_json_bytes(data: &[u8]) -> bool {
    if data.is_empty() {
        return false;
    }
    match std::str::from_utf8(data) {
        Ok(text) => JsonValue::parse(text).is_ok(),
        Err(_) => false,
    }
}

pub(crate) fn read_oci_blob(
    entries: &mut HashMap<String, Blob>,
    name: &str,
    length: i64,
    reader: &mut dyn Read,
    json_bytes: &mut usize,
) -> Result<(), Error> {
    if length < 0 || length == i64::MAX {
        return Err(Error::msg("invalid OCI blob size"));
    }
    let (sum, size, body) = copy_oci_blob(name, length, reader)?;
    *json_bytes += body.as_ref().map(Vec::len).unwrap_or(0);
    if *json_bytes > 32 << 20 {
        return Err(Error::msg("OCI JSON metadata limit exceeded"));
    }
    entries.insert(
        name.to_string(),
        Blob {
            hash: sum,
            size,
            data: body,
        },
    );
    Ok(())
}

fn entry_raw_name(entry: &tar::Entry<'_, impl Read>) -> String {
    match entry.path() {
        Ok(path) => path.to_string_lossy().into_owned(),
        Err(_) => String::from_utf8_lossy(&entry.path_bytes()).into_owned(),
    }
}

fn read_archive_blob_entry(
    entries: &mut HashMap<String, Blob>,
    name: &str,
    size: i64,
    is_regular: bool,
    reader: &mut dyn Read,
    json_bytes: &mut usize,
) -> Result<(), Error> {
    if !is_regular {
        return Err(Error::msg("non-regular OCI entry"));
    }
    if entries.contains_key(name) {
        return Err(Error::msg("duplicate OCI entry"));
    }
    if entries.len() > 100_000 {
        return Err(Error::msg("too many OCI entries"));
    }
    if !is_valid_oci_regular_entry(name) {
        return Err(Error::msg("not an OCI archive"));
    }
    read_oci_blob(entries, name, size, reader, json_bytes)
}

fn read_oci_archive_entries(reader: &mut dyn Read) -> Result<HashMap<String, Blob>, Error> {
    let mut archive = tar::Archive::new(reader);
    let mut entries: HashMap<String, Blob> = HashMap::new();
    let mut seen: HashMap<String, ()> = HashMap::new();
    let mut json_bytes = 0usize;
    let listed = archive.entries().map_err(|e| Error::msg(e.to_string()))?;
    for entry in listed {
        let mut entry = entry.map_err(|e| Error::msg(e.to_string()))?;
        let raw = entry_raw_name(&entry);
        let name = path_clean(&raw);
        let header = entry.header().clone();
        let entry_type = header.entry_type();
        let is_dir = entry_type.is_dir();
        let is_regular = entry_type.is_file();
        if check_tar_entry_name(&name, is_dir)? {
            continue;
        }
        record_tar_entry(&mut seen, &name)?;
        if check_tar_directory(&name, is_dir)? {
            continue;
        }
        let size = header.size().map_err(|e| Error::msg(e.to_string()))? as i64;
        read_archive_blob_entry(
            &mut entries,
            &name,
            size,
            is_regular,
            &mut entry,
            &mut json_bytes,
        )?;
    }
    Ok(entries)
}

/// Parses the index; both single-image archives and multi-image layouts
/// share this gate.
pub(crate) fn read_oci_index(entries: &HashMap<String, Blob>) -> Result<Vec<Descriptor>, Error> {
    let layout_data = entries.get("oci-layout").and_then(|b| b.data.as_ref());
    let layout_ok = layout_data
        .and_then(|data| std::str::from_utf8(data).ok())
        .and_then(|text| JsonValue::parse(text).ok())
        .and_then(|v| Fields::of(&v).map(|f| f.string("imageLayoutVersion").unwrap_or_default()))
        == Some("1.0.0".to_string());
    if !layout_ok {
        return Err(Error::msg("missing OCI layout"));
    }
    let index_data = entries.get("index.json").and_then(|b| b.data.as_ref());
    let index_text = index_data
        .and_then(|data| std::str::from_utf8(data).ok())
        .ok_or_else(|| Error::msg("valid OCI index required"))?;
    let index_value =
        JsonValue::parse(index_text).map_err(|_| Error::msg("valid OCI index required"))?;
    let index = Fields::of(&index_value).ok_or_else(|| Error::msg("valid OCI index required"))?;
    let schema = index
        .int("schemaVersion")
        .map_err(|_| Error::msg("valid OCI index required"))?;
    let media = index.media_type();
    if schema != 2 || (!media.is_empty() && media != INDEX_MEDIA_TYPE) {
        return Err(Error::msg("valid OCI index required"));
    }
    let manifests = index
        .object_list("manifests")
        .map_err(|_| Error::msg("valid OCI index required"))?;
    manifests
        .iter()
        .map(Descriptor::decode)
        .collect::<Result<Vec<_>, _>>()
        .map_err(|_| Error::msg("valid OCI index required"))
}

trait MediaType {
    fn media_type(&self) -> String;
}

impl MediaType for Fields<'_> {
    fn media_type(&self) -> String {
        self.string("mediaType").unwrap_or_default()
    }
}

struct OciManifest {
    config: Descriptor,
    layers: Vec<Descriptor>,
}

fn parse_oci_manifest(data: &[u8]) -> Result<OciManifest, Error> {
    let text = std::str::from_utf8(data).map_err(|_| Error::msg("invalid OCI image manifest"))?;
    let value = JsonValue::parse(text).map_err(|_| Error::msg("invalid OCI image manifest"))?;
    let fields = Fields::of(&value).ok_or_else(|| Error::msg("invalid OCI image manifest"))?;
    let schema = fields
        .int("schemaVersion")
        .map_err(|_| Error::msg("invalid OCI image manifest"))?;
    let media = fields.media_type();
    let config = fields
        .object("config")
        .map_err(|_| Error::msg("invalid OCI image manifest"))?
        .ok_or_else(|| Error::msg("invalid OCI image manifest"))?;
    let config =
        Descriptor::decode(&config).map_err(|_| Error::msg("invalid OCI image manifest"))?;
    if schema != 2
        || (!media.is_empty() && media != MANIFEST_MEDIA_TYPE)
        || config.media_type != CONFIG_MEDIA_TYPE
    {
        return Err(Error::msg("invalid OCI image manifest"));
    }
    let layers = fields
        .object_list("layers")
        .map_err(|_| Error::msg("invalid OCI image manifest"))?;
    let layers = layers
        .iter()
        .map(Descriptor::decode)
        .collect::<Result<Vec<_>, _>>()
        .map_err(|_| Error::msg("invalid OCI image manifest"))?;
    Ok(OciManifest { config, layers })
}

/// Fetches a blob by descriptor, loading it on demand for layouts.
pub(crate) fn fetch_oci_blob(
    entries: &mut HashMap<String, Blob>,
    desc: &Descriptor,
    load: LoadBlobs,
) -> Result<(String, i64, Option<Vec<u8>>), Error> {
    if desc.size < 0 || !desc.urls.is_empty() {
        return Err(Error::msg("local bounded OCI descriptor required"));
    }
    let hex = match desc.digest.strip_prefix("sha256:") {
        Some(hex) if is_digest(hex) => hex,
        _ => return Err(Error::msg("invalid OCI digest")),
    };
    let name = format!("blobs/sha256/{hex}");
    load(entries, &name)?;
    match entries.get(&name) {
        Some(blob) if blob.size == desc.size => {
            Ok((blob.hash.clone(), blob.size, blob.data.clone()))
        }
        _ => Err(Error::msg("missing or wrong-size OCI blob")),
    }
}

fn validate_oci_layers(
    entries: &mut HashMap<String, Blob>,
    layers: &[Descriptor],
    load: LoadBlobs,
) -> Result<(), Error> {
    for layer in layers {
        match layer.media_type.as_str() {
            LAYER_TAR | LAYER_TAR_GZIP | LAYER_TAR_ZSTD => {}
            _ => return Err(Error::msg("unsupported OCI layer media type")),
        }
        fetch_oci_blob(entries, layer, load)?;
    }
    Ok(())
}

fn validate_oci_rootfs(diff_ids: &[String], layers: &[Descriptor]) -> Result<(), Error> {
    if diff_ids.len() != layers.len() {
        return Err(Error::msg("OCI rootfs/layer count mismatch"));
    }
    for (id, layer) in diff_ids.iter().zip(layers.iter()) {
        match id.strip_prefix("sha256:") {
            Some(hex) if is_digest(hex) => {}
            _ => return Err(Error::msg("invalid OCI diff ID")),
        }
        if layer.media_type == LAYER_TAR && id != &layer.digest {
            return Err(Error::msg("uncompressed OCI layer identity mismatch"));
        }
    }
    Ok(())
}

fn validate_oci_attribution(labels: &[(String, String)], want_revision: &str) -> Result<(), Error> {
    let label = |key: &str| {
        labels
            .iter()
            .rev()
            .find(|(k, _)| k == key)
            .map(|(_, v)| v.as_str())
            .unwrap_or("")
    };
    let rev = label("org.opencontainers.image.revision");
    if !want_revision.is_empty() && rev != want_revision {
        return Err(Error::msg("OCI source revision mismatch"));
    }
    if want_revision.is_empty() {
        return Ok(());
    }
    let base_digest = label("org.opencontainers.image.base.digest");
    let base_hex = base_digest.strip_prefix("sha256:").unwrap_or("");
    if label("org.opencontainers.image.source") != "https://github.com/LevitateOS/sodaos"
        || label("org.opencontainers.image.base.name").is_empty()
        || !base_digest.starts_with("sha256:")
        || !is_digest(base_hex)
    {
        return Err(Error::msg("soda image lacks source/base attribution"));
    }
    Ok(())
}

fn inspect_oci_config(
    config_data: &[u8],
    layers: &[Descriptor],
    want: &str,
    revision: &str,
    image_digest: &str,
    config_digest: &str,
) -> Result<Image, Error> {
    let text =
        std::str::from_utf8(config_data).map_err(|_| Error::msg("invalid OCI image config"))?;
    let value = JsonValue::parse(text).map_err(|_| Error::msg("invalid OCI image config"))?;
    let cfg = Fields::of(&value).ok_or_else(|| Error::msg("invalid OCI image config"))?;
    let rootfs = cfg
        .object("rootfs")
        .map_err(|_| Error::msg("invalid OCI image config"))?
        .ok_or_else(|| Error::msg("invalid OCI image config"))?;
    if rootfs
        .string("type")
        .map_err(|_| Error::msg("invalid OCI image config"))?
        != "layers"
    {
        return Err(Error::msg("OCI rootfs/layer count mismatch"));
    }
    let diff_ids = rootfs
        .string_list("diff_ids")
        .map_err(|_| Error::msg("invalid OCI image config"))?;
    validate_oci_rootfs(&diff_ids, layers)?;
    let os = cfg
        .string("os")
        .map_err(|_| Error::msg("invalid OCI image config"))?;
    let arch = cfg
        .string("architecture")
        .map_err(|_| Error::msg("invalid OCI image config"))?;
    // Compressed layer contents are not extracted here. Their blob
    // identities are checked; native import remains the proof of
    // decompression/rootfs use.
    if os != "linux" || arch != want {
        return Err(Error::msg(format!("OCI must be linux/{want}")));
    }
    let config_section = cfg
        .object("config")
        .map_err(|_| Error::msg("invalid OCI image config"))?;
    let labels = match config_section {
        Some(section) => section
            .string_map("Labels")
            .map_err(|_| Error::msg("invalid OCI image config"))?,
        None => Vec::new(),
    };
    validate_oci_attribution(&labels, revision)?;
    let label = |key: &str| {
        labels
            .iter()
            .rev()
            .find(|(k, _)| k == key)
            .map(|(_, v)| v.clone())
            .unwrap_or_default()
    };
    Ok(Image {
        manifest: image_digest.to_string(),
        config: config_digest.to_string(),
        architecture: arch,
        revision: label("org.opencontainers.image.revision"),
        source: label("org.opencontainers.image.source"),
        base_name: label("org.opencontainers.image.base.name"),
        base_digest: label("org.opencontainers.image.base.digest"),
    })
}

/// Verifies one image out of loaded blobs; `load` reads more for layouts.
pub(crate) fn inspect_oci_image(
    entries: &mut HashMap<String, Blob>,
    image: &Descriptor,
    want: &str,
    revision: &str,
    load: LoadBlobs,
) -> Result<Image, Error> {
    let (_, _, manifest_data) = fetch_oci_blob(entries, image, load)?;
    let manifest_data = manifest_data.unwrap_or_default();
    let manifest = parse_oci_manifest(&manifest_data)?;
    validate_oci_layers(entries, &manifest.layers, load)?;
    let (_, _, config_data) = fetch_oci_blob(entries, &manifest.config, load)?;
    let config_data = config_data.unwrap_or_default();
    inspect_oci_config(
        &config_data,
        &manifest.layers,
        want,
        revision,
        &image.digest,
        &manifest.config.digest,
    )
}

fn inspect_archive_index(
    entries: &mut HashMap<String, Blob>,
    want: &str,
    revision: &str,
) -> Result<Image, Error> {
    let index = read_oci_index(entries)?;
    if index.len() != 1 || index[0].media_type != MANIFEST_MEDIA_TYPE {
        return Err(Error::msg("single-platform OCI index required"));
    }
    let mut no_load = |_: &mut HashMap<String, Blob>, _: &str| -> Result<(), Error> { Ok(()) };
    inspect_oci_image(entries, &index[0], want, revision, {
        let loader: LoadBlobs<'_> = &mut no_load;
        loader
    })
}

/// Verifies a single-platform OCI archive's identity without running it.
pub fn inspect_oci(file: &Path, arch: &str, revision: &str) -> Result<Image, Error> {
    let (want, f) = open_oci_archive(file, arch, revision)?;
    let mut reader = std::io::BufReader::new(f);
    let mut entries = read_oci_archive_entries(&mut reader)?;
    inspect_archive_index(&mut entries, &want, revision)
}

/// Absolute clean member paths requested out of the verified rootfs.
pub(crate) fn requested_oci_paths(paths: &[String]) -> Result<BTreeMap<String, String>, Error> {
    let mut wanted = BTreeMap::new();
    for requested in paths {
        let name = requested
            .strip_prefix('/')
            .ok_or_else(|| Error::msg("absolute clean OCI member paths required"))?;
        if name.is_empty() || path_clean(name) != name || name.contains(['\\', '\n', '\r', '\0']) {
            return Err(Error::msg("absolute clean OCI member paths required"));
        }
        if wanted.contains_key(name) {
            return Err(Error::msg("duplicate OCI member request"));
        }
        wanted.insert(name.to_string(), requested.clone());
    }
    if wanted.is_empty() {
        return Err(Error::msg("explicit OCI members required"));
    }
    Ok(wanted)
}

/// Admits real Linux layer entries. Backslashes stay permitted: base layers
/// ship systemd escaped unit names, and member requests reject backslashes,
/// so such entries can never match a request.
fn clean_layer_name(name: &str) -> Result<String, Error> {
    let name = name.strip_prefix("./").unwrap_or(name);
    let clean = path_clean(name);
    let segments: Vec<&str> = name.split('/').collect();
    if clean == "."
        || clean.starts_with('/')
        || segments.contains(&"..")
        || clean.contains(['\n', '\r', '\0'])
    {
        return Err(Error::msg("unsafe OCI layer path"));
    }
    Ok(clean)
}

fn whiteout_target(name: &str) -> Option<String> {
    let base = name.rsplit('/').next().unwrap_or(name);
    if !base.starts_with(".wh.") || base == ".wh..wh..opq" {
        return None;
    }
    let dir = name.rfind('/').map(|i| &name[..i]).unwrap_or("");
    let target = base.strip_prefix(".wh.").unwrap_or("");
    if dir.is_empty() {
        Some(target.to_string())
    } else {
        Some(format!("{dir}/{target}"))
    }
}

fn record_layer_deletion(
    found: &mut HashMap<String, LayerMember>,
    wanted: &BTreeMap<String, String>,
    removed: &str,
) {
    for name in wanted.keys() {
        if name != removed && !name.starts_with(&format!("{removed}/")) {
            continue;
        }
        if let Some(existing) = found.get(name) {
            if existing.present || existing.blocked {
                continue;
            }
        }
        found.insert(name.clone(), LayerMember::default());
    }
}

fn record_ancestor_replacement(
    found: &mut HashMap<String, LayerMember>,
    wanted: &BTreeMap<String, String>,
    name: &str,
    is_dir: bool,
) {
    if is_dir {
        return;
    }
    for requested in wanted.keys() {
        if requested.starts_with(&format!("{name}/")) {
            found.insert(
                requested.clone(),
                LayerMember {
                    blocked: true,
                    ..LayerMember::default()
                },
            );
        }
    }
}

fn record_opaque_directory(
    found: &mut HashMap<String, LayerMember>,
    wanted: &BTreeMap<String, String>,
    dir: &str,
) {
    for requested in wanted.keys() {
        if dir != "." && !requested.starts_with(&format!("{dir}/")) {
            continue;
        }
        if let Some(existing) = found.get(requested) {
            if existing.present || existing.blocked {
                continue;
            }
        }
        found.insert(requested.clone(), LayerMember::default());
    }
}

fn record_layer_entry(
    reader: &mut dyn Read,
    size: u64,
    name: &str,
    found: &mut HashMap<String, LayerMember>,
) -> Result<(), Error> {
    let mut hasher = sha2::Sha256::new();
    let mut remaining = size;
    let mut buf = [0u8; 32 << 10];
    let mut copied = 0u64;
    use sha2::Digest;
    while remaining > 0 {
        let want = (remaining as usize).min(buf.len());
        let n = reader
            .read(&mut buf[..want])
            .map_err(|_| Error::msg("OCI layer member size changed"))?;
        if n == 0 {
            return Err(Error::msg("OCI layer member size changed"));
        }
        hasher.update(&buf[..n]);
        copied += n as u64;
        remaining -= n as u64;
    }
    if copied != size {
        return Err(Error::msg("OCI layer member size changed"));
    }
    found.insert(
        name.to_string(),
        LayerMember {
            hash: hex_digest(&hasher.finalize()),
            present: true,
            blocked: false,
        },
    );
    Ok(())
}

fn hex_digest(digest: &[u8]) -> String {
    let mut out = String::with_capacity(digest.len() * 2);
    for byte in digest {
        out.push_str(&format!("{byte:02x}"));
    }
    out
}

fn scan_oci_layer(
    reader: &mut dyn Read,
    wanted: &BTreeMap<String, String>,
) -> Result<HashMap<String, LayerMember>, Error> {
    let mut found: HashMap<String, LayerMember> = HashMap::new();
    let mut seen: HashMap<String, ()> = HashMap::new();
    let mut archive = tar::Archive::new(reader);
    let listed = archive.entries().map_err(|e| Error::msg(e.to_string()))?;
    for entry in listed {
        let mut entry = entry.map_err(|e| Error::msg(e.to_string()))?;
        let raw = entry_raw_name(&entry);
        // Base layers include the tar root directory. It can never match
        // a requested member, so skip it instead of refusing the archive.
        if raw == "." || raw == "./" {
            continue;
        }
        let name = clean_layer_name(&raw)?;
        if seen.contains_key(&name) {
            return Err(Error::msg("duplicate OCI layer entry"));
        }
        seen.insert(name.clone(), ());
        if seen.len() > 1_000_000 {
            return Err(Error::msg("too many OCI layer entries"));
        }
        let header = entry.header().clone();
        let entry_type = header.entry_type();
        let is_regular = entry_type.is_file();
        let is_dir = entry_type.is_dir();
        if let Some(removed) = whiteout_target(&name) {
            record_layer_deletion(&mut found, wanted, &removed);
            continue;
        }
        let base = name.rsplit('/').next().unwrap_or(&name);
        if base == ".wh..wh..opq" {
            let dir = if name.contains('/') {
                name.rfind('/').map(|i| &name[..i]).unwrap_or(".")
            } else {
                "."
            };
            record_opaque_directory(&mut found, wanted, dir);
            continue;
        }
        record_ancestor_replacement(&mut found, wanted, &name, is_dir);
        if !wanted.contains_key(&name) || found.get(&name).map(|m| m.blocked).unwrap_or(false) {
            continue;
        }
        if !is_regular {
            return Err(Error::msg("requested OCI member is non-regular"));
        }
        let size = header.size().map_err(|e| Error::msg(e.to_string()))?;
        record_layer_entry(&mut entry, size, &name, &mut found)?;
    }
    Ok(found)
}

fn layer_archive_indexes(layers: &[Descriptor]) -> Result<HashMap<String, Vec<usize>>, Error> {
    let mut indexes: HashMap<String, Vec<usize>> = HashMap::new();
    let mut seen: HashMap<String, &Descriptor> = HashMap::new();
    for (i, layer) in layers.iter().enumerate() {
        let name = format!(
            "blobs/sha256/{}",
            layer
                .digest
                .strip_prefix("sha256:")
                .unwrap_or(&layer.digest)
        );
        if let Some(prior) = seen.get(&name) {
            if prior.media_type != layer.media_type || prior.size != layer.size {
                return Err(Error::msg("conflicting OCI layer descriptors"));
            }
        }
        seen.insert(name.clone(), layer);
        indexes.entry(name).or_default().push(i);
    }
    Ok(indexes)
}

/// Hashing tee: every byte read is digested, like Go's `io.TeeReader`.
struct HashReader<R> {
    inner: R,
    hasher: sha2::Sha256,
}

impl<R: Read> HashReader<R> {
    fn new(inner: R) -> HashReader<R> {
        HashReader {
            inner,
            hasher: sha2::Sha256::new(),
        }
    }

    fn hex(&self) -> String {
        hex_digest(&self.hasher.clone().finalize())
    }
}

impl<R: Read> Read for HashReader<R> {
    fn read(&mut self, buf: &mut [u8]) -> std::io::Result<usize> {
        let n = self.inner.read(buf)?;
        use sha2::Digest;
        self.hasher.update(&buf[..n]);
        Ok(n)
    }
}

fn scan_layer_reader(
    layer: &mut dyn Read,
    wanted: &BTreeMap<String, String>,
) -> Result<HashMap<String, LayerMember>, Error> {
    let members = scan_oci_layer(layer, wanted);
    // Drain so gzip trailer corruption surfaces even when the scan stops
    // early, mirroring Go's post-scan Discard + Close check.
    if members.is_ok() {
        let mut sink = std::io::sink();
        if let Err(e) = std::io::copy(layer, &mut sink) {
            return Err(Error::msg(e.to_string()));
        }
    }
    members
}

fn scan_archive_layer(
    reader: &mut dyn Read,
    descriptor: &Descriptor,
    wanted: &BTreeMap<String, String>,
) -> Result<(HashMap<String, LayerMember>, bool), Error> {
    if descriptor.media_type == LAYER_TAR_ZSTD {
        let mut raw = HashReader::new(reader);
        let mut sink = std::io::sink();
        std::io::copy(&mut raw, &mut sink)
            .map_err(|_| Error::msg("OCI layer changed during member verification"))?;
        let hex = raw.hex();
        if format!("sha256:{hex}") != descriptor.digest {
            return Err(Error::msg("OCI layer changed during member verification"));
        }
        return Ok((HashMap::new(), true));
    }
    let mut raw = HashReader::new(reader);
    let (members, scan_err) = match descriptor.media_type.as_str() {
        LAYER_TAR => (scan_layer_reader(&mut raw, wanted), None),
        LAYER_TAR_GZIP => {
            let mut gz = flate2::read::GzDecoder::new(&mut raw);
            let members = scan_layer_reader(&mut gz, wanted);
            // Drop the decoder before draining the raw remainder.
            drop(gz);
            (members, None)
        }
        _ => (
            Ok(HashMap::new()),
            Some(Error::msg("unsupported OCI layer media type")),
        ),
    };
    if let Some(err) = scan_err {
        return Err(err);
    }
    let members = members?;
    let mut sink = std::io::sink();
    std::io::copy(&mut raw, &mut sink)
        .map_err(|e| Error::msg(format!("layer drain failed: {e}")))?;
    if format!("sha256:{}", raw.hex()) != descriptor.digest {
        return Err(Error::msg("OCI layer changed during member verification"));
    }
    Ok((members, false))
}

fn scan_oci_archive_layers(
    reader: &mut dyn Read,
    layers: &[Descriptor],
    wanted: &BTreeMap<String, String>,
) -> Result<LayerScans, Error> {
    let indexes = layer_archive_indexes(layers)?;
    let mut found: Vec<HashMap<String, LayerMember>> = Vec::with_capacity(layers.len());
    found.resize_with(layers.len(), HashMap::new);
    let mut unsupported = vec![false; layers.len()];
    let mut archive = tar::Archive::new(reader);
    let listed = archive.entries().map_err(|e| Error::msg(e.to_string()))?;
    for entry in listed {
        let mut entry = entry.map_err(|e| Error::msg(e.to_string()))?;
        let name = path_clean(&entry_raw_name(&entry));
        let positions = indexes.get(&name).cloned().unwrap_or_default();
        if positions.is_empty() {
            continue;
        }
        let (members, blocked) = scan_archive_layer(&mut entry, &layers[positions[0]], wanted)?;
        for position in positions {
            found[position] = members.clone();
            unsupported[position] = blocked;
        }
    }
    Ok((found, unsupported))
}

fn resolve_oci_members(
    layers: &[HashMap<String, LayerMember>],
    unsupported: &[bool],
    wanted: &BTreeMap<String, String>,
) -> Result<BTreeMap<String, String>, Error> {
    let mut resolved = BTreeMap::new();
    for (name, requested) in wanted.iter() {
        for i in (0..layers.len()).rev() {
            if unsupported[i] {
                return Err(Error::msg("zstd OCI layer blocks member verification"));
            }
            let member = match layers[i].get(name) {
                Some(member) => member,
                None => continue,
            };
            if member.blocked {
                return Err(Error::msg(
                    "requested OCI member has a non-directory ancestor",
                ));
            }
            if !member.present {
                return Err(Error::msg("requested OCI member was removed"));
            }
            resolved.insert(requested.clone(), member.hash.clone());
            break;
        }
        if !resolved.contains_key(requested) {
            return Err(Error::msg("requested OCI member missing"));
        }
    }
    Ok(resolved)
}

fn inspect_content_manifest(
    entries: &mut HashMap<String, Blob>,
    want: &str,
    revision: &str,
) -> Result<(Image, OciManifest), Error> {
    let index =
        read_oci_index(entries).map_err(|_| Error::msg("single-platform OCI index required"))?;
    if index.len() != 1 || index[0].media_type != MANIFEST_MEDIA_TYPE {
        return Err(Error::msg("single-platform OCI index required"));
    }
    let mut no_load = |_: &mut HashMap<String, Blob>, _: &str| -> Result<(), Error> { Ok(()) };
    let image = inspect_oci_image(entries, &index[0], want, revision, {
        let loader: LoadBlobs<'_> = &mut no_load;
        loader
    })?;
    let (_, _, manifest_data) = fetch_oci_blob(entries, &index[0], {
        let loader: LoadBlobs<'_> = &mut no_load;
        loader
    })?;
    let manifest_data = manifest_data.unwrap_or_default();
    let manifest = parse_oci_manifest(&manifest_data)?;
    Ok((image, manifest))
}

/// Extends the identity check with hashes of exact regular files as they
/// appear in the verified image rootfs overlay.
pub fn inspect_oci_content(
    file: &Path,
    arch: &str,
    revision: &str,
    paths: &[String],
) -> Result<(Image, BTreeMap<String, String>), Error> {
    let wanted = requested_oci_paths(paths)?;
    let (want, mut f) = open_oci_archive(file, arch, revision)?;
    let entries = {
        let mut reader = std::io::BufReader::new(&mut f);
        read_oci_archive_entries(&mut reader)?
    };
    let mut entries = entries;
    let (image, manifest) = inspect_content_manifest(&mut entries, &want, revision)?;
    use std::io::Seek;
    f.seek(std::io::SeekFrom::Start(0))
        .map_err(|e| io_error("seek", file, e))?;
    let mut reader = std::io::BufReader::new(&mut f);
    let (layers, unsupported) = scan_oci_archive_layers(&mut reader, &manifest.layers, &wanted)?;
    let content = resolve_oci_members(&layers, &unsupported, &wanted)?;
    Ok((image, content))
}

#[cfg(test)]
pub(crate) mod tests {
    use super::*;
    use crate::sha256_hex;
    use std::io::Write;

    pub const FIXTURE_REVISION: &str = "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa";

    /// Builds the Go `fixtureOCI` archive: one tar layer with fixture.txt,
    /// config with soda attribution, manifest, index.
    pub fn fixture_oci_bytes(arch: &str) -> Vec<u8> {
        let body = b"synthetic layer fixture; never executed";
        let mut layer = Vec::new();
        {
            let mut writer = tar::Builder::new(&mut layer);
            let mut header = tar::Header::new_ustar();
            header.set_size(body.len() as u64);
            header.set_mode(0o644);
            header.set_cksum();
            writer
                .append_data(&mut header, "fixture.txt", &body[..])
                .unwrap();
            writer.into_inner().unwrap();
        }
        let layer_sum = sha256_hex(&layer);
        let layer_digest = format!("sha256:{layer_sum}");
        let config = format!(
            "{{\"os\":\"linux\",\"architecture\":\"{arch}\",\"rootfs\":{{\"type\":\"layers\",\"diff_ids\":[\"{layer_digest}\"]}},\"config\":{{\"Labels\":{{\"org.opencontainers.image.revision\":\"{FIXTURE_REVISION}\",\"org.opencontainers.image.source\":\"https://github.com/LevitateOS/sodaos\",\"org.opencontainers.image.base.name\":\"synthetic-base\",\"org.opencontainers.image.base.digest\":\"sha256:{}\"}}}}}}",
            "b".repeat(64)
        );
        let config_sum = sha256_hex(config.as_bytes());
        let manifest = format!(
            "{{\"schemaVersion\":2,\"config\":{{\"digest\":\"sha256:{config_sum}\",\"size\":{},\"mediaType\":\"{CONFIG_MEDIA_TYPE}\"}},\"layers\":[{{\"digest\":\"{layer_digest}\",\"size\":{},\"mediaType\":\"{LAYER_TAR}\"}}]}}",
            config.len(),
            layer.len()
        );
        let manifest_sum = sha256_hex(manifest.as_bytes());
        let index = format!(
            "{{\"schemaVersion\":2,\"manifests\":[{{\"digest\":\"sha256:{manifest_sum}\",\"size\":{},\"mediaType\":\"{MANIFEST_MEDIA_TYPE}\"}}]}}",
            manifest.len()
        );
        let blobs: Vec<(String, Vec<u8>)> = vec![
            (format!("blobs/sha256/{layer_sum}"), layer),
            (format!("blobs/sha256/{config_sum}"), config.into_bytes()),
            (
                format!("blobs/sha256/{manifest_sum}"),
                manifest.into_bytes(),
            ),
            ("index.json".to_string(), index.into_bytes()),
            (
                "oci-layout".to_string(),
                br#"{"imageLayoutVersion":"1.0.0"}"#.to_vec(),
            ),
        ];
        let mut archive = Vec::new();
        {
            let mut writer = tar::Builder::new(&mut archive);
            for (name, data) in &blobs {
                let mut header = tar::Header::new_ustar();
                header.set_size(data.len() as u64);
                header.set_mode(0o644);
                header.set_cksum();
                writer
                    .append_data(&mut header, name.as_str(), &data[..])
                    .unwrap();
            }
            writer.into_inner().unwrap();
        }
        archive
    }

    /// Hand-crafted ustar entry for names `tar::Builder` refuses (`..`).
    fn raw_tar_entry(name: &str, body: &[u8]) -> Vec<u8> {
        let mut header = [0u8; 512];
        header[..name.len()].copy_from_slice(name.as_bytes());
        header[100..108].copy_from_slice(b"0000644\0");
        header[108..116].copy_from_slice(b"0000000\0");
        header[116..124].copy_from_slice(b"0000000\0");
        let size = format!("{:011o}\0", body.len());
        header[124..136].copy_from_slice(size.as_bytes());
        header[136..148].copy_from_slice(b"00000000000\0");
        header[148..156].copy_from_slice(b"        ");
        header[156] = b'0';
        header[257..262].copy_from_slice(b"ustar");
        let sum: u32 = header.iter().map(|b| *b as u32).sum();
        let chksum = format!("{:06o}\0 ", sum);
        header[148..156].copy_from_slice(chksum.as_bytes());
        let mut out = Vec::new();
        out.extend_from_slice(&header);
        out.extend_from_slice(body);
        out.resize(out.len() + (512 - body.len() % 512) % 512, 0);
        out.extend_from_slice(&[0u8; 1024]);
        out
    }

    fn write_fixture(name: &str, bytes: &[u8]) -> std::path::PathBuf {
        let dir = std::env::temp_dir().join(format!(
            "soda-oci-{}-{}",
            std::process::id(),
            std::sync::atomic::AtomicU64::new(0).fetch_add(1, std::sync::atomic::Ordering::SeqCst)
        ));
        let _ = std::fs::create_dir_all(&dir);
        let path = dir.join(name);
        std::fs::write(&path, bytes).unwrap();
        path
    }

    #[test]
    fn oracle_identity_and_wrong_platform() {
        // Oracle: TestOCIIdentityAndWrongPlatform over fixtureOCI.
        let file = write_fixture("image.oci", &fixture_oci_bytes("amd64"));
        let image = inspect_oci(&file, "x86_64", FIXTURE_REVISION).unwrap();
        assert_eq!(image.architecture, "amd64");
        assert_eq!(image.revision, FIXTURE_REVISION);
        assert_eq!(image.base_name, "synthetic-base");
        assert_eq!(
            inspect_oci(&file, "aarch64", FIXTURE_REVISION)
                .unwrap_err()
                .message(),
            "expected x86_64"
        );
        assert_eq!(
            inspect_oci(&file, "x86_64", &"c".repeat(40))
                .unwrap_err()
                .message(),
            "OCI source revision mismatch"
        );
    }

    #[test]
    fn oracle_content_members() {
        // Oracle: InspectOCIContent resolves fixture.txt through the overlay.
        let file = write_fixture("content.oci", &fixture_oci_bytes("amd64"));
        let (image, content) = inspect_oci_content(
            &file,
            "x86_64",
            FIXTURE_REVISION,
            &[String::from("/fixture.txt")],
        )
        .unwrap();
        assert_eq!(image.revision, FIXTURE_REVISION);
        assert_eq!(
            content.get("/fixture.txt").unwrap(),
            &sha256_hex(b"synthetic layer fixture; never executed")
        );
        assert!(inspect_oci_content(
            &file,
            "x86_64",
            FIXTURE_REVISION,
            &[String::from("/missing")]
        )
        .is_err());
    }

    #[test]
    fn oracle_layer_scanner_vectors() {
        // Oracle: TestOCIMemberScanner* whiteout/removal/blocked outcomes.
        let layer_bytes = |names: &[&str]| -> Vec<u8> {
            let mut out = Vec::new();
            let mut writer = tar::Builder::new(&mut out);
            for name in names {
                let mut header = tar::Header::new_ustar();
                header.set_size(name.len() as u64);
                header.set_mode(0o644);
                header.set_cksum();
                writer
                    .append_data(&mut header, *name, name.as_bytes())
                    .unwrap();
            }
            writer.into_inner().unwrap();
            out
        };
        let wanted: BTreeMap<String, String> = [("wanted".to_string(), "/wanted".to_string())]
            .into_iter()
            .collect();
        let dup = layer_bytes(&["wanted", "wanted"]);
        assert_eq!(
            scan_oci_layer(&mut &dup[..], &wanted)
                .unwrap_err()
                .message(),
            "duplicate OCI layer entry"
        );
        let evil = raw_tar_entry("../wanted", b"evil");
        assert_eq!(
            scan_oci_layer(&mut &evil[..], &wanted)
                .unwrap_err()
                .message(),
            "unsafe OCI layer path"
        );
        let benign = layer_bytes(&[".", "wanted"]);
        let members = scan_oci_layer(&mut &benign[..], &wanted).unwrap();
        assert!(members.get("wanted").unwrap().present);
        // Whiteout in the upper layer removes the lower member.
        let lower: HashMap<String, LayerMember> = [(
            "wanted".to_string(),
            LayerMember {
                hash: "a".repeat(64),
                present: true,
                blocked: false,
            },
        )]
        .into_iter()
        .collect();
        let upper_bytes = layer_bytes(&[".wh.wanted"]);
        let upper = scan_oci_layer(&mut &upper_bytes[..], &wanted).unwrap();
        assert_eq!(
            resolve_oci_members(&[lower, upper], &[false, false], &wanted)
                .unwrap_err()
                .message(),
            "requested OCI member was removed"
        );
    }

    #[test]
    fn oracle_gzip_and_zstd_layers() {
        // Oracle: TestOCIArchiveLayerCompressionAndDigest outcomes.
        use flate2::write::GzEncoder;
        let content = b"verified member";
        let mut layer = Vec::new();
        {
            let mut writer = tar::Builder::new(&mut layer);
            let mut header = tar::Header::new_ustar();
            header.set_size(content.len() as u64);
            header.set_mode(0o644);
            header.set_cksum();
            writer
                .append_data(&mut header, "wanted", &content[..])
                .unwrap();
            writer.into_inner().unwrap();
        }
        let mut compressed = Vec::new();
        {
            let mut gz = GzEncoder::new(&mut compressed, flate2::Compression::default());
            gz.write_all(&layer).unwrap();
            gz.finish().unwrap();
        }
        let wanted: BTreeMap<String, String> = [("wanted".to_string(), "/wanted".to_string())]
            .into_iter()
            .collect();
        for (media, data, blocked) in [
            (LAYER_TAR, layer.clone(), false),
            (LAYER_TAR_GZIP, compressed.clone(), false),
            (LAYER_TAR_ZSTD, compressed.clone(), true),
        ] {
            let sum = sha256_hex(&data);
            let desc = Descriptor {
                digest: format!("sha256:{sum}"),
                size: data.len() as i64,
                media_type: media.to_string(),
                ..Descriptor::default()
            };
            let mut archive = Vec::new();
            {
                let mut writer = tar::Builder::new(&mut archive);
                let name = format!("blobs/sha256/{sum}");
                let mut header = tar::Header::new_ustar();
                header.set_size(data.len() as u64);
                header.set_mode(0o644);
                header.set_cksum();
                writer
                    .append_data(&mut header, name.as_str(), &data[..])
                    .unwrap();
                writer.into_inner().unwrap();
            }
            let (found, unsupported) =
                scan_oci_archive_layers(&mut &archive[..], &[desc], &wanted).unwrap();
            assert_eq!(unsupported[0], blocked);
            if !blocked {
                assert_eq!(found[0].get("wanted").unwrap().hash, sha256_hex(content));
            }
        }
        // Corrupt gzip trailer with a matching blob digest must still fail.
        let mut corrupt = compressed.clone();
        let last = corrupt.len() - 1;
        corrupt[last] ^= 0xff;
        let sum = sha256_hex(&corrupt);
        let desc = Descriptor {
            digest: format!("sha256:{sum}"),
            size: corrupt.len() as i64,
            media_type: LAYER_TAR_GZIP.to_string(),
            ..Descriptor::default()
        };
        assert!(scan_archive_layer(&mut &corrupt[..], &desc, &wanted).is_err());
    }
}
