//! Port of `internal/release/build` OCI inspection (`oci.go`,
//! `oci_layout.go`) consumed by `deliver`: single-image archive identity,
//! rootfs content hashes, and shared-layout verification. Streams archives
//! without extracting layers or importing release code.

use std::collections::{BTreeMap, BTreeSet};
use std::io::Read;

use flate2::read::GzDecoder;
use sha2::{Digest as _, Sha256};

use crate::buildx::{
    is_digest, is_revision, oci_architecture, read_layout_entry, Image as BuildImage, Root,
};
use crate::model::path_clean;
use crate::Error;

const MANIFEST_TYPE: &str = "application/vnd.oci.image.manifest.v1+json";
const CONFIG_TYPE: &str = "application/vnd.oci.image.config.v1+json";
const INDEX_TYPE: &str = "application/vnd.oci.image.index.v1+json";
const LAYER_TAR: &str = "application/vnd.oci.image.layer.v1.tar";
const LAYER_GZIP: &str = "application/vnd.oci.image.layer.v1.tar+gzip";
const LAYER_ZSTD: &str = "application/vnd.oci.image.layer.v1.tar+zstd";

/// `build.OCILayout`: verified shared-blob directory.
#[derive(Debug, Clone, Default)]
pub struct OciLayout {
    pub images: BTreeMap<String, BuildImage>,
    pub files: BTreeMap<String, String>,
    pub bytes: u64,
}

#[derive(Debug, Clone, Default)]
struct Blob {
    hash: String,
    size: i64,
    data: Option<Vec<u8>>,
}

mod schema;
use schema::{
    fetch_oci_blob, inspect_oci_config, inspect_oci_image, parse_oci_manifest, read_oci_blob,
    read_oci_index, Descriptor, LayerMember, OciManifest,
};

// ---------------------------------------------------------------------------
// Archive ingestion
// ---------------------------------------------------------------------------

fn open_oci_archive(
    file: &str,
    arch: &str,
    revision: &str,
) -> Result<(String, std::fs::File), Error> {
    let want = oci_architecture(arch).map_err(|e| Error::msg(e.0))?;
    if !revision.is_empty() && !is_revision(revision) {
        return Err(Error::msg("full source revision required"));
    }
    let st =
        std::fs::symlink_metadata(file).map_err(|e| Error::msg(format!("lstat {file}: {e}")))?;
    if !st.is_file() {
        return Err(Error::msg("OCI archive must be regular"));
    }
    let f = std::fs::File::open(file).map_err(|e| Error::msg(format!("open {file}: {e}")))?;
    Ok((want.to_string(), f))
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

fn read_oci_archive_entries<R: Read>(reader: R) -> Result<BTreeMap<String, Blob>, Error> {
    let mut archive = tar::Archive::new(reader);
    let mut entries: BTreeMap<String, Blob> = BTreeMap::new();
    let mut seen: BTreeSet<String> = BTreeSet::new();
    let mut json_bytes: i64 = 0;
    let items = archive
        .entries()
        .map_err(|e| Error::msg(format!("read OCI archive: {e}")))?;
    for item in items {
        let mut item = item.map_err(|e| Error::msg(format!("read OCI archive: {e}")))?;
        let raw_name = String::from_utf8_lossy(&item.path_bytes()).into_owned();
        let name = path_clean(&raw_name);
        let header = item.header().clone();
        let is_dir = header.entry_type().is_dir();
        if name == "." && is_dir {
            continue;
        }
        if raw_name.starts_with('/') || name == ".." || name.starts_with("../") {
            return Err(Error::msg("unsafe OCI path"));
        }
        if !seen.insert(name.clone()) {
            return Err(Error::msg("duplicate OCI entry"));
        }
        if seen.len() > 100000 {
            return Err(Error::msg("too many OCI entries"));
        }
        if is_dir {
            if name != "blobs" && name != "blobs/sha256" {
                return Err(Error::msg("unexpected OCI directory"));
            }
            continue;
        }
        if !header.entry_type().is_file() {
            return Err(Error::msg("non-regular OCI entry"));
        }
        if entries.contains_key(&name) {
            return Err(Error::msg("duplicate OCI entry"));
        }
        if entries.len() > 100000 {
            return Err(Error::msg("too many OCI entries"));
        }
        if !is_valid_oci_regular_entry(&name) {
            return Err(Error::msg("not an OCI archive"));
        }
        let size = header
            .size()
            .map_err(|e| Error::msg(format!("read OCI archive: {e}")))?;
        let length = i64::try_from(size).map_err(|_| Error::msg("invalid OCI blob size"))?;
        let mut data = Vec::new();
        item.read_to_end(&mut data)
            .map_err(|e| Error::msg(format!("read OCI archive: {e}")))?;
        read_oci_blob(&mut entries, &name, length, data, &mut json_bytes)?;
    }
    Ok(entries)
}

fn inspect_archive_index(
    entries: &BTreeMap<String, Blob>,
    want: &str,
    revision: &str,
) -> Result<BuildImage, Error> {
    let index = read_oci_index(entries)?;
    if index.len() != 1 || index[0].media_type != MANIFEST_TYPE {
        return Err(Error::msg("single-platform OCI index required"));
    }
    inspect_oci_image(entries, &index[0], want, revision)
}

/// `build.InspectOCI`: verify single-image archive identity.
pub fn inspect_oci(file: &str, arch: &str, revision: &str) -> Result<BuildImage, Error> {
    let (want, f) = open_oci_archive(file, arch, revision)?;
    let entries = read_oci_archive_entries(f)?;
    inspect_archive_index(&entries, &want, revision)
}

// ---------------------------------------------------------------------------
// Content verification
// ---------------------------------------------------------------------------

fn requested_oci_paths(paths: &[String]) -> Result<BTreeMap<String, String>, Error> {
    let mut wanted: BTreeMap<String, String> = BTreeMap::new();
    for requested in paths {
        let name = requested.strip_prefix('/').unwrap_or("");
        if !requested.starts_with('/')
            || name.is_empty()
            || path_clean(name) != name
            || name.bytes().any(|b| matches!(b, b'\\' | b'\n' | b'\r' | 0))
        {
            return Err(Error::msg("absolute clean OCI member paths required"));
        }
        if wanted.insert(name.to_string(), requested.clone()).is_some() {
            return Err(Error::msg("duplicate OCI member request"));
        }
    }
    if wanted.is_empty() {
        return Err(Error::msg("explicit OCI members required"));
    }
    Ok(wanted)
}

fn clean_layer_name(name: &str) -> Result<String, Error> {
    let trimmed = name.strip_prefix("./").unwrap_or(name);
    let clean = path_clean(trimmed);
    let segments: Vec<&str> = trimmed.split('/').collect();
    if clean == "."
        || clean.starts_with('/')
        || segments.contains(&"..")
        || clean.bytes().any(|b| matches!(b, b'\n' | b'\r' | 0))
    {
        return Err(Error::msg("unsafe OCI layer path"));
    }
    Ok(clean)
}

fn path_dir(name: &str) -> &str {
    match name.rfind('/') {
        Some(i) => &name[..i],
        None => ".",
    }
}

fn path_base(name: &str) -> &str {
    match name.rfind('/') {
        Some(i) => &name[i + 1..],
        None => name,
    }
}

fn path_join(dir: &str, base: &str) -> String {
    if dir == "." || dir.is_empty() {
        base.to_string()
    } else {
        format!("{dir}/{base}")
    }
}

fn whiteout_target(name: &str) -> Option<String> {
    let base = path_base(name);
    if !base.starts_with(".wh.") || base == ".wh..wh..opq" {
        return None;
    }
    Some(path_join(
        path_dir(name),
        base.strip_prefix(".wh.").unwrap_or(""),
    ))
}

fn record_layer_deletion(
    found: &mut BTreeMap<String, LayerMember>,
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
    found: &mut BTreeMap<String, LayerMember>,
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
    found: &mut BTreeMap<String, LayerMember>,
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

fn record_layer_entry<R: Read>(
    reader: &mut R,
    size: u64,
    is_regular: bool,
    is_dir: bool,
    name: &str,
    found: &mut BTreeMap<String, LayerMember>,
    wanted: &BTreeMap<String, String>,
) -> Result<(), Error> {
    if let Some(removed) = whiteout_target(name) {
        record_layer_deletion(found, wanted, &removed);
        return Ok(());
    }
    if path_base(name) == ".wh..wh..opq" {
        record_opaque_directory(found, wanted, path_dir(name));
        return Ok(());
    }
    record_ancestor_replacement(found, wanted, name, is_dir);
    if !wanted.contains_key(name) || found.get(name).map(|m| m.blocked).unwrap_or(false) {
        return Ok(());
    }
    if !is_regular {
        return Err(Error::msg("requested OCI member is non-regular"));
    }
    let mut hasher = Sha256::new();
    let mut copied: u64 = 0;
    let mut chunk = [0u8; 65536];
    loop {
        let n = reader
            .read(&mut chunk)
            .map_err(|_| Error::msg("OCI layer member size changed"))?;
        if n == 0 {
            break;
        }
        hasher.update(&chunk[..n]);
        copied += n as u64;
    }
    if copied != size {
        return Err(Error::msg("OCI layer member size changed"));
    }
    found.insert(
        name.to_string(),
        LayerMember {
            hash: format!("{:x}", hasher.finalize()),
            present: true,
            blocked: false,
        },
    );
    Ok(())
}

fn scan_oci_layer<R: Read>(
    reader: R,
    wanted: &BTreeMap<String, String>,
) -> Result<BTreeMap<String, LayerMember>, Error> {
    let mut found: BTreeMap<String, LayerMember> = BTreeMap::new();
    let mut seen: BTreeSet<String> = BTreeSet::new();
    let mut archive = tar::Archive::new(reader);
    let items = archive
        .entries()
        .map_err(|e| Error::msg(format!("read OCI layer: {e}")))?;
    for item in items {
        let mut item = item.map_err(|e| Error::msg(format!("read OCI layer: {e}")))?;
        let raw_name = String::from_utf8_lossy(&item.path_bytes()).into_owned();
        if raw_name == "." || raw_name == "./" {
            continue;
        }
        let name = clean_layer_name(&raw_name)?;
        if !seen.insert(name.clone()) {
            return Err(Error::msg("duplicate OCI layer entry"));
        }
        if seen.len() > 1000000 {
            return Err(Error::msg("too many OCI layer entries"));
        }
        let header = item.header().clone();
        let size = header
            .size()
            .map_err(|e| Error::msg(format!("read OCI layer: {e}")))?;
        record_layer_entry(
            &mut item,
            size,
            header.entry_type().is_file(),
            header.entry_type().is_dir(),
            &name,
            &mut found,
            wanted,
        )?;
    }
    Ok(found)
}

fn layer_archive_indexes(layers: &[Descriptor]) -> Result<BTreeMap<String, Vec<usize>>, Error> {
    let mut indexes: BTreeMap<String, Vec<usize>> = BTreeMap::new();
    let mut seen: BTreeMap<String, &Descriptor> = BTreeMap::new();
    for (i, layer) in layers.iter().enumerate() {
        let hex = layer.digest.strip_prefix("sha256:").unwrap_or("\x00");
        let name = format!("blobs/sha256/{hex}");
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

struct TeeHasher<R> {
    inner: R,
    hasher: Sha256,
}

impl<R: Read> Read for TeeHasher<R> {
    fn read(&mut self, buf: &mut [u8]) -> std::io::Result<usize> {
        let n = self.inner.read(buf)?;
        self.hasher.update(&buf[..n]);
        Ok(n)
    }
}

fn drain(reader: &mut dyn Read) -> std::io::Result<()> {
    let mut chunk = [0u8; 65536];
    loop {
        if reader.read(&mut chunk)? == 0 {
            return Ok(());
        }
    }
}

fn scan_archive_layer<R: Read>(
    reader: &mut R,
    descriptor: &Descriptor,
    wanted: &BTreeMap<String, String>,
) -> Result<(BTreeMap<String, LayerMember>, bool), Error> {
    let changed = || Error::msg("OCI layer changed during member verification");
    let mut tee = TeeHasher {
        inner: reader,
        hasher: Sha256::new(),
    };
    if descriptor.media_type == LAYER_ZSTD {
        if drain(&mut tee).is_err() {
            return Err(changed());
        }
        let sum = format!("sha256:{:x}", tee.hasher.clone().finalize());
        if sum != descriptor.digest {
            return Err(changed());
        }
        return Ok((BTreeMap::new(), true));
    }
    let members: BTreeMap<String, LayerMember> = match descriptor.media_type.as_str() {
        LAYER_TAR => scan_oci_layer(&mut tee, wanted)?,
        LAYER_GZIP => {
            let gz = GzDecoder::new(&mut tee);
            scan_oci_layer(gz, wanted)?
        }
        _ => return Err(Error::msg("unsupported OCI layer media type")),
    };
    if drain(&mut tee).is_err() {
        return Err(changed());
    }
    let sum = format!("sha256:{:x}", tee.hasher.clone().finalize());
    if sum != descriptor.digest {
        return Err(changed());
    }
    Ok((members, false))
}

fn resolve_oci_members(
    layers: &[BTreeMap<String, LayerMember>],
    unsupported: &[bool],
    wanted: &BTreeMap<String, String>,
) -> Result<BTreeMap<String, String>, Error> {
    let mut resolved: BTreeMap<String, String> = BTreeMap::new();
    for (name, requested) in wanted {
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
    entries: &BTreeMap<String, Blob>,
    want: &str,
    revision: &str,
) -> Result<(BuildImage, OciManifest), Error> {
    let index = read_oci_index(entries)?;
    if index.len() != 1 || index[0].media_type != MANIFEST_TYPE {
        return Err(Error::msg("single-platform OCI index required"));
    }
    let image = inspect_oci_image(entries, &index[0], want, revision)?;
    let manifest_blob = fetch_oci_blob(entries, &index[0])?;
    let empty = Vec::new();
    let manifest = parse_oci_manifest(manifest_blob.data.as_ref().unwrap_or(&empty))?;
    Ok((image, manifest))
}

/// `build.InspectOCIContent`: identity plus exact rootfs member hashes.
pub fn inspect_oci_content(
    file: &str,
    arch: &str,
    revision: &str,
    paths: &[String],
) -> Result<(BuildImage, BTreeMap<String, String>), Error> {
    let wanted = requested_oci_paths(paths)?;
    let (want, mut f) = open_oci_archive(file, arch, revision)?;
    let entries = read_oci_archive_entries(&f)?;
    let (image, manifest) = inspect_content_manifest(&entries, &want, revision)?;
    use std::io::Seek;
    f.seek(std::io::SeekFrom::Start(0))
        .map_err(|e| Error::msg(format!("seek {file}: {e}")))?;
    let (layers, unsupported) = scan_oci_archive_layers(&mut f, &manifest.layers, &wanted)?;
    let content = resolve_oci_members(&layers, &unsupported, &wanted)?;
    Ok((image, content))
}

type LayerScan = (Vec<BTreeMap<String, LayerMember>>, Vec<bool>);

fn scan_oci_archive_layers<R: Read>(
    reader: R,
    layers: &[Descriptor],
    wanted: &BTreeMap<String, String>,
) -> Result<LayerScan, Error> {
    let indexes = layer_archive_indexes(layers)?;
    let mut found: Vec<BTreeMap<String, LayerMember>> = Vec::new();
    found.resize_with(layers.len(), BTreeMap::new);
    let mut unsupported = vec![false; layers.len()];
    let mut archive = tar::Archive::new(reader);
    let items = archive
        .entries()
        .map_err(|e| Error::msg(format!("read OCI archive: {e}")))?;
    for item in items {
        let mut item = item.map_err(|e| Error::msg(format!("read OCI archive: {e}")))?;
        let raw_name = String::from_utf8_lossy(&item.path_bytes()).into_owned();
        let positions = indexes
            .get(&path_clean(&raw_name))
            .cloned()
            .unwrap_or_default();
        if positions.is_empty() {
            continue;
        }
        let (members, blocked) = scan_archive_layer(&mut item, &layers[positions[0]], wanted)?;
        for position in positions {
            found[position] = members.clone();
            unsupported[position] = blocked;
        }
    }
    Ok((found, unsupported))
}

// ---------------------------------------------------------------------------
// Shared layout
// ---------------------------------------------------------------------------

fn validate_oci_layout_inputs(
    arch: &str,
    revisions: &BTreeMap<String, String>,
) -> Result<String, Error> {
    let want = oci_architecture(arch).map_err(|e| Error::msg(e.0))?;
    if revisions.is_empty() {
        return Err(Error::msg("explicit OCI image set required"));
    }
    for (reference, revision) in revisions {
        let hex = reference.strip_prefix("sha256:").unwrap_or("");
        if !reference.starts_with("sha256:")
            || !is_digest(hex)
            || (!revision.is_empty() && !is_revision(revision))
        {
            return Err(Error::msg("invalid OCI image selection"));
        }
    }
    Ok(want.to_string())
}

/// `build.InspectOCILayout`: verify the exact named image set and blobs.
pub fn inspect_oci_layout(
    dir: &str,
    arch: &str,
    revisions: &BTreeMap<String, String>,
) -> Result<OciLayout, Error> {
    let want = validate_oci_layout_inputs(arch, revisions)?;
    let info = std::fs::symlink_metadata(dir)
        .map_err(|_| Error::msg("real OCI layout directory required"))?;
    if !info.is_dir() {
        return Err(Error::msg("real OCI layout directory required"));
    }
    let root = Root::open(dir).map_err(|_| Error::msg("real OCI layout directory required"))?;
    let mut loader = LayoutLoader {
        root: &root,
        entries: BTreeMap::new(),
        json_bytes: 0,
    };
    for name in ["index.json", "oci-layout"] {
        loader.load(name)?;
    }
    let index = read_oci_index(&loader.entries)?;
    let images = inspect_layout_images(&mut loader, &index, &want, revisions)?;
    let mut files: BTreeMap<String, String> = BTreeMap::new();
    let mut total: u64 = 0;
    for (name, entry) in &loader.entries {
        files.insert(name.clone(), entry.hash.clone());
        total += entry.size as u64;
    }
    Ok(OciLayout {
        images,
        files,
        bytes: total,
    })
}

struct LayoutLoader<'a> {
    root: &'a Root,
    entries: BTreeMap<String, Blob>,
    json_bytes: i64,
}

impl LayoutLoader<'_> {
    fn load(&mut self, name: &str) -> Result<(), Error> {
        if self.entries.contains_key(name) {
            return Ok(());
        }
        let (data, size) = read_layout_entry(self.root, name)?;
        read_oci_blob(&mut self.entries, name, size, data, &mut self.json_bytes)
    }

    fn fetch(&mut self, d: &Descriptor) -> Result<Blob, Error> {
        if d.size < 0 || !d.urls.is_empty() {
            return Err(Error::msg("local bounded OCI descriptor required"));
        }
        let hex = d.digest.strip_prefix("sha256:").unwrap_or("");
        if !d.digest.starts_with("sha256:") || !is_digest(hex) {
            return Err(Error::msg("invalid OCI digest"));
        }
        let name = format!("blobs/sha256/{hex}");
        self.load(&name)?;
        match self.entries.get(&name) {
            Some(blob) if blob.size == d.size => Ok(blob.clone()),
            _ => Err(Error::msg("missing or wrong-size OCI blob")),
        }
    }

    fn inspect_image(
        &mut self,
        image: &Descriptor,
        want: &str,
        revision: &str,
    ) -> Result<BuildImage, Error> {
        let manifest_blob = self.fetch(image)?;
        let empty = Vec::new();
        let manifest = parse_oci_manifest(manifest_blob.data.as_ref().unwrap_or(&empty))?;
        for layer in &manifest.layers {
            match layer.media_type.as_str() {
                LAYER_TAR | LAYER_GZIP | LAYER_ZSTD => {}
                _ => return Err(Error::msg("unsupported OCI layer media type")),
            }
            self.fetch(layer)?;
        }
        let config_blob = self.fetch(&manifest.config)?;
        inspect_oci_config(
            &config_blob,
            &manifest.layers,
            want,
            revision,
            &image.digest,
            &manifest.config.digest,
        )
    }
}

fn inspect_layout_images(
    loader: &mut LayoutLoader<'_>,
    index: &[Descriptor],
    want: &str,
    revisions: &BTreeMap<String, String>,
) -> Result<BTreeMap<String, BuildImage>, Error> {
    if index.len() != revisions.len() {
        return Err(Error::msg("exact OCI image set required"));
    }
    let mut images: BTreeMap<String, BuildImage> = BTreeMap::new();
    for descriptor in index {
        let reference = descriptor
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
            .map(|image: &BuildImage| !image.config.is_empty())
            .unwrap_or(false)
            || descriptor.media_type != MANIFEST_TYPE
        {
            return Err(Error::msg("unexpected or duplicate OCI image reference"));
        }
        let image = loader.inspect_image(descriptor, want, &revision)?;
        if image.config != reference {
            return Err(Error::msg("OCI reference differs from config identity"));
        }
        images.insert(reference, image);
    }
    Ok(images)
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn member_path_rules() {
        // Path validation runs before archive opening.
        let err = inspect_oci_content("/nonexistent.oci", "x86_64", "", &[]).unwrap_err();
        assert_eq!(err, Error::msg("explicit OCI members required"));
        let err = inspect_oci_content("/nonexistent.oci", "x86_64", "", &["relative".to_string()])
            .unwrap_err();
        assert_eq!(err, Error::msg("absolute clean OCI member paths required"));
        let err = inspect_oci("/nonexistent.oci", "aarch64", "").unwrap_err();
        assert_eq!(err, Error::msg("expected x86_64"));
        let err = inspect_oci("/nonexistent.oci", "x86_64", "short").unwrap_err();
        assert_eq!(err, Error::msg("full source revision required"));
    }

    #[test]
    fn non_archive_refused() {
        let dir = std::env::temp_dir().join(format!("srd-oci-unit-{}", std::process::id()));
        std::fs::create_dir_all(&dir).unwrap();
        let path = dir.join("junk.oci").to_string_lossy().into_owned();
        std::fs::write(&path, b"definitely not a tar archive ....................").unwrap();
        let err = inspect_oci(&path, "x86_64", "").unwrap_err();
        assert!(err.0.contains("read OCI archive"), "{}", err.0);
        std::fs::remove_dir_all(&dir).unwrap();
    }
}
