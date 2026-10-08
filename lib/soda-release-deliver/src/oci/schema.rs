use serde::de::{DeserializeOwned, Error as DeError, IgnoredAny, MapAccess, Visitor};
use serde::{Deserialize, Deserializer};
use std::collections::BTreeMap;
use std::fmt;
use std::io::Read;

use sha2::{Digest as _, Sha256};

use crate::buildx::{is_digest, Image as BuildImage};
use crate::Error;

use super::{Blob, CONFIG_TYPE, INDEX_TYPE, LAYER_GZIP, LAYER_TAR, LAYER_ZSTD, MANIFEST_TYPE};

#[derive(Debug, Clone, Default)]
pub(super) struct Descriptor {
    pub(super) digest: String,
    pub(super) size: i64,
    pub(super) media_type: String,
    pub(super) urls: Vec<String>,
    pub(super) annotations: BTreeMap<String, String>,
}

#[derive(Debug, Clone, Default)]
struct IndexDocument {
    schema_version: i64,
    media_type: String,
    manifests: Vec<Descriptor>,
}

#[derive(Debug, Clone, Default)]
struct ManifestDocument {
    schema_version: i64,
    media_type: String,
    config: Descriptor,
    layers: Vec<Descriptor>,
}

#[derive(Debug, Clone, Default)]
struct RootfsDocument {
    rootfs_type: String,
    diff_ids: Vec<String>,
}

#[derive(Debug, Clone, Default)]
struct ImageConfigDocument {
    labels: BTreeMap<String, String>,
}

#[derive(Debug, Clone, Default)]
struct ConfigDocument {
    os: String,
    architecture: String,
    rootfs: RootfsDocument,
    config: ImageConfigDocument,
}

#[derive(Default)]
struct LayoutDocument {
    image_layout_version: String,
}

fn decode_raw<T: DeserializeOwned + Default, E: DeError>(
    raw: Option<Box<serde_json::value::RawValue>>,
) -> Result<T, E> {
    let Some(raw) = raw else {
        return Ok(T::default());
    };
    serde_json::from_str::<Option<T>>(raw.get())
        .map_err(E::custom)
        .map(Option::unwrap_or_default)
}

#[derive(Default)]
struct RawI64(i64);
impl<'de> Deserialize<'de> for RawI64 {
    fn deserialize<D: Deserializer<'de>>(d: D) -> Result<Self, D::Error> {
        crate::json_serde::null_i64(d).map(RawI64)
    }
}

impl<'de> Deserialize<'de> for Descriptor {
    fn deserialize<D: Deserializer<'de>>(d: D) -> Result<Self, D::Error> {
        struct V;
        impl<'de> Visitor<'de> for V {
            type Value = Descriptor;
            fn expecting(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
                f.write_str("an OCI descriptor")
            }
            fn visit_map<M: MapAccess<'de>>(self, mut m: M) -> Result<Descriptor, M::Error> {
                let (mut digest, mut size, mut media_type, mut urls, mut annotations) =
                    (None, None, None, None, None);
                while let Some(key) = m.next_key::<String>()? {
                    match key.as_str() {
                        "digest" => {
                            digest = Some(m.next_value::<Box<serde_json::value::RawValue>>()?)
                        }
                        "size" => size = Some(m.next_value::<Box<serde_json::value::RawValue>>()?),
                        "mediaType" => {
                            media_type = Some(m.next_value::<Box<serde_json::value::RawValue>>()?)
                        }
                        "urls" => urls = Some(m.next_value::<Box<serde_json::value::RawValue>>()?),
                        "annotations" => {
                            annotations = Some(m.next_value::<Box<serde_json::value::RawValue>>()?)
                        }
                        _ => {
                            m.next_value::<IgnoredAny>()?;
                        }
                    }
                }
                Ok(Descriptor {
                    digest: decode_raw(digest)?,
                    size: decode_raw::<RawI64, M::Error>(size)?.0,
                    media_type: decode_raw(media_type)?,
                    urls: decode_raw(urls)?,
                    annotations: decode_raw(annotations)?,
                })
            }
        }
        d.deserialize_map(V)
    }
}

impl<'de> Deserialize<'de> for IndexDocument {
    fn deserialize<D: Deserializer<'de>>(d: D) -> Result<Self, D::Error> {
        struct V;
        impl<'de> Visitor<'de> for V {
            type Value = IndexDocument;
            fn expecting(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
                f.write_str("an OCI index object")
            }
            fn visit_map<M: MapAccess<'de>>(self, mut m: M) -> Result<IndexDocument, M::Error> {
                let (mut version, mut media, mut manifests) = (None, None, None);
                while let Some(key) = m.next_key::<String>()? {
                    match key.as_str() {
                        "schemaVersion" => {
                            version = Some(m.next_value::<Box<serde_json::value::RawValue>>()?)
                        }
                        "mediaType" => {
                            media = Some(m.next_value::<Box<serde_json::value::RawValue>>()?)
                        }
                        "manifests" => {
                            manifests = Some(m.next_value::<Box<serde_json::value::RawValue>>()?)
                        }
                        _ => {
                            m.next_value::<IgnoredAny>()?;
                        }
                    }
                }
                Ok(IndexDocument {
                    schema_version: decode_raw::<RawI64, M::Error>(version)?.0,
                    media_type: decode_raw(media)?,
                    manifests: decode_raw(manifests)?,
                })
            }
        }
        d.deserialize_map(V)
    }
}

impl<'de> Deserialize<'de> for ManifestDocument {
    fn deserialize<D: Deserializer<'de>>(d: D) -> Result<Self, D::Error> {
        struct V;
        impl<'de> Visitor<'de> for V {
            type Value = ManifestDocument;
            fn expecting(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
                f.write_str("an OCI manifest object")
            }
            fn visit_map<M: MapAccess<'de>>(self, mut m: M) -> Result<ManifestDocument, M::Error> {
                let (mut version, mut media, mut config, mut layers) = (None, None, None, None);
                while let Some(key) = m.next_key::<String>()? {
                    match key.as_str() {
                        "schemaVersion" => {
                            version = Some(m.next_value::<Box<serde_json::value::RawValue>>()?)
                        }
                        "mediaType" => {
                            media = Some(m.next_value::<Box<serde_json::value::RawValue>>()?)
                        }
                        "config" => {
                            config = Some(m.next_value::<Box<serde_json::value::RawValue>>()?)
                        }
                        "layers" => {
                            layers = Some(m.next_value::<Box<serde_json::value::RawValue>>()?)
                        }
                        _ => {
                            m.next_value::<IgnoredAny>()?;
                        }
                    }
                }
                Ok(ManifestDocument {
                    schema_version: decode_raw::<RawI64, M::Error>(version)?.0,
                    media_type: decode_raw(media)?,
                    config: decode_raw(config)?,
                    layers: decode_raw(layers)?,
                })
            }
        }
        d.deserialize_map(V)
    }
}

impl<'de> Deserialize<'de> for RootfsDocument {
    fn deserialize<D: Deserializer<'de>>(d: D) -> Result<Self, D::Error> {
        struct V;
        impl<'de> Visitor<'de> for V {
            type Value = RootfsDocument;
            fn expecting(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
                f.write_str("an OCI rootfs object")
            }
            fn visit_map<M: MapAccess<'de>>(self, mut m: M) -> Result<RootfsDocument, M::Error> {
                let (mut kind, mut ids) = (None, None);
                while let Some(key) = m.next_key::<String>()? {
                    match key.as_str() {
                        "type" => kind = Some(m.next_value::<Box<serde_json::value::RawValue>>()?),
                        "diff_ids" => {
                            ids = Some(m.next_value::<Box<serde_json::value::RawValue>>()?)
                        }
                        _ => {
                            m.next_value::<IgnoredAny>()?;
                        }
                    }
                }
                Ok(RootfsDocument {
                    rootfs_type: decode_raw(kind)?,
                    diff_ids: decode_raw(ids)?,
                })
            }
        }
        d.deserialize_map(V)
    }
}

impl<'de> Deserialize<'de> for ImageConfigDocument {
    fn deserialize<D: Deserializer<'de>>(d: D) -> Result<Self, D::Error> {
        struct V;
        impl<'de> Visitor<'de> for V {
            type Value = ImageConfigDocument;
            fn expecting(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
                f.write_str("an OCI config object")
            }
            fn visit_map<M: MapAccess<'de>>(
                self,
                mut m: M,
            ) -> Result<ImageConfigDocument, M::Error> {
                let mut labels = None;
                while let Some(key) = m.next_key::<String>()? {
                    if key == "Labels" {
                        labels = Some(m.next_value::<Box<serde_json::value::RawValue>>()?);
                    } else {
                        m.next_value::<IgnoredAny>()?;
                    }
                }
                Ok(ImageConfigDocument {
                    labels: decode_raw(labels)?,
                })
            }
        }
        d.deserialize_map(V)
    }
}

impl<'de> Deserialize<'de> for ConfigDocument {
    fn deserialize<D: Deserializer<'de>>(d: D) -> Result<Self, D::Error> {
        struct V;
        impl<'de> Visitor<'de> for V {
            type Value = ConfigDocument;
            fn expecting(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
                f.write_str("an OCI image config object")
            }
            fn visit_map<M: MapAccess<'de>>(self, mut m: M) -> Result<ConfigDocument, M::Error> {
                let (mut os, mut arch, mut rootfs, mut config) = (None, None, None, None);
                while let Some(key) = m.next_key::<String>()? {
                    match key.as_str() {
                        "os" => os = Some(m.next_value::<Box<serde_json::value::RawValue>>()?),
                        "architecture" => {
                            arch = Some(m.next_value::<Box<serde_json::value::RawValue>>()?)
                        }
                        "rootfs" => {
                            rootfs = Some(m.next_value::<Box<serde_json::value::RawValue>>()?)
                        }
                        "config" => {
                            config = Some(m.next_value::<Box<serde_json::value::RawValue>>()?)
                        }
                        _ => {
                            m.next_value::<IgnoredAny>()?;
                        }
                    }
                }
                Ok(ConfigDocument {
                    os: decode_raw(os)?,
                    architecture: decode_raw(arch)?,
                    rootfs: decode_raw(rootfs)?,
                    config: decode_raw(config)?,
                })
            }
        }
        d.deserialize_map(V)
    }
}

impl<'de> Deserialize<'de> for LayoutDocument {
    fn deserialize<D: Deserializer<'de>>(d: D) -> Result<Self, D::Error> {
        struct V;
        impl<'de> Visitor<'de> for V {
            type Value = LayoutDocument;
            fn expecting(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
                f.write_str("an OCI layout object")
            }
            fn visit_map<M: MapAccess<'de>>(self, mut m: M) -> Result<LayoutDocument, M::Error> {
                let mut version = None;
                while let Some(key) = m.next_key::<String>()? {
                    if key == "imageLayoutVersion" {
                        version = Some(m.next_value::<Box<serde_json::value::RawValue>>()?);
                    } else {
                        m.next_value::<IgnoredAny>()?;
                    }
                }
                Ok(LayoutDocument {
                    image_layout_version: decode_raw(version)?,
                })
            }
        }
        d.deserialize_map(V)
    }
}

#[derive(Debug, Clone, Default)]
pub(super) struct OciManifest {
    pub(super) config: Descriptor,
    pub(super) layers: Vec<Descriptor>,
}

pub(super) fn read_oci_index(entries: &BTreeMap<String, Blob>) -> Result<Vec<Descriptor>, Error> {
    let invalid = || Error::msg("valid OCI index required");
    let layout_data = entries
        .get("oci-layout")
        .and_then(|b| b.data.as_deref())
        .ok_or_else(|| Error::msg("missing OCI layout"))?;
    let layout_text =
        std::str::from_utf8(layout_data).map_err(|_| Error::msg("missing OCI layout"))?;
    let layout: LayoutDocument =
        serde_json::from_str(layout_text).map_err(|_| Error::msg("missing OCI layout"))?;
    if layout.image_layout_version != "1.0.0" {
        return Err(Error::msg("missing OCI layout"));
    }
    let index_data = entries
        .get("index.json")
        .and_then(|b| b.data.as_deref())
        .ok_or_else(invalid)?;
    let index_text = std::str::from_utf8(index_data).map_err(|_| invalid())?;
    let index: IndexDocument = serde_json::from_str(index_text).map_err(|_| invalid())?;
    if index.schema_version != 2 || (!index.media_type.is_empty() && index.media_type != INDEX_TYPE)
    {
        return Err(invalid());
    }
    Ok(index.manifests)
}

pub(super) fn parse_oci_manifest(data: &[u8]) -> Result<OciManifest, Error> {
    let invalid = || Error::msg("invalid OCI image manifest");
    let text = std::str::from_utf8(data).map_err(|_| invalid())?;
    let manifest: ManifestDocument = serde_json::from_str(text).map_err(|_| invalid())?;
    if manifest.schema_version != 2
        || (!manifest.media_type.is_empty() && manifest.media_type != MANIFEST_TYPE)
    {
        return Err(invalid());
    }
    if manifest.config.media_type != CONFIG_TYPE {
        return Err(invalid());
    }
    Ok(OciManifest {
        config: manifest.config,
        layers: manifest.layers,
    })
}

struct OciConfig {
    os: String,
    arch: String,
    rootfs_type: String,
    diff_ids: Vec<String>,
    labels: BTreeMap<String, String>,
}

fn parse_oci_config(data: &[u8]) -> Result<OciConfig, Error> {
    let invalid = || Error::msg("invalid OCI image config");
    let text = std::str::from_utf8(data).map_err(|_| invalid())?;
    let config: ConfigDocument = serde_json::from_str(text).map_err(|_| invalid())?;
    Ok(OciConfig {
        os: config.os,
        arch: config.architecture,
        rootfs_type: config.rootfs.rootfs_type,
        diff_ids: config.rootfs.diff_ids,
        labels: config.config.labels,
    })
}

fn is_json(data: &[u8]) -> bool {
    let mut decoder = serde_json::Deserializer::from_slice(data);
    serde::de::IgnoredAny::deserialize(&mut decoder).is_ok() && decoder.end().is_ok()
}

pub(super) enum BlobSource {
    Layout,
    Archive,
}

impl BlobSource {
    fn read_error(&self, name: &str, error: std::io::Error) -> Error {
        match self {
            Self::Layout => Error::msg(format!("read {name}: {error}")),
            Self::Archive => Error::msg(format!("read OCI archive: {error}")),
        }
    }
}

pub(super) fn read_oci_blob<R: Read>(
    entries: &mut BTreeMap<String, Blob>,
    name: &str,
    length: i64,
    mut reader: R,
    source: BlobSource,
    json_bytes: &mut i64,
) -> Result<(), Error> {
    if length < 0 || length == i64::MAX {
        return Err(Error::msg("invalid OCI blob size"));
    }
    const CHUNK_BYTES: usize = 64 * 1024;
    let retain_candidate = length <= (4 << 20);
    let mut data = if retain_candidate {
        Vec::with_capacity(length as usize)
    } else {
        Vec::new()
    };
    let mut hasher = Sha256::new();
    let mut buffer = [0u8; CHUNK_BYTES];
    let mut remaining = length as u64;
    while remaining > 0 {
        let take = remaining.min(CHUNK_BYTES as u64) as usize;
        let read = loop {
            match reader.read(&mut buffer[..take]) {
                Err(error) if error.kind() == std::io::ErrorKind::Interrupted => continue,
                result => break result,
            }
        }
        .map_err(|error| source.read_error(name, error))?;
        if read == 0 {
            return Err(Error::msg("OCI blob size changed"));
        }
        hasher.update(&buffer[..read]);
        if retain_candidate {
            data.extend_from_slice(&buffer[..read]);
        }
        remaining -= read as u64;
    }
    let mut extra = [0u8; 1];
    loop {
        match reader.read(&mut extra) {
            Err(error) if error.kind() == std::io::ErrorKind::Interrupted => continue,
            Err(error) => return Err(source.read_error(name, error)),
            Ok(0) => break,
            Ok(_) => return Err(Error::msg("OCI blob size changed")),
        }
    }
    let sum = format!("{:x}", hasher.finalize());
    if name.starts_with("blobs/") && *name != format!("blobs/sha256/{sum}") {
        return Err(Error::msg("OCI blob checksum mismatch"));
    }
    let body = if retain_candidate && is_json(&data) {
        Some(data)
    } else {
        None
    };
    *json_bytes += body.as_ref().map(|b| b.len()).unwrap_or(0) as i64;
    if *json_bytes > 32 << 20 {
        return Err(Error::msg("OCI JSON metadata limit exceeded"));
    }
    entries.insert(
        name.to_string(),
        Blob {
            hash: sum,
            size: length,
            data: body,
        },
    );
    Ok(())
}
pub(super) fn fetch_oci_blob(
    entries: &BTreeMap<String, Blob>,
    d: &Descriptor,
) -> Result<Blob, Error> {
    if d.size < 0 || !d.urls.is_empty() {
        return Err(Error::msg("local bounded OCI descriptor required"));
    }
    let hex = d.digest.strip_prefix("sha256:").unwrap_or("");
    if !d.digest.starts_with("sha256:") || !is_digest(hex) {
        return Err(Error::msg("invalid OCI digest"));
    }
    let name = format!("blobs/sha256/{hex}");
    match entries.get(&name) {
        Some(blob) if blob.size == d.size => Ok(blob.clone()),
        _ => Err(Error::msg("missing or wrong-size OCI blob")),
    }
}

#[cfg(test)]
mod streaming_tests {
    use super::*;
    use std::io::Cursor;

    struct ProbeReader {
        available: u64,
        read: u64,
        fail_at: Option<u64>,
        interrupted: bool,
        largest_request: usize,
    }

    impl ProbeReader {
        fn new(available: u64) -> Self {
            Self {
                available,
                read: 0,
                fail_at: None,
                interrupted: false,
                largest_request: 0,
            }
        }
    }

    impl Read for ProbeReader {
        fn read(&mut self, buffer: &mut [u8]) -> std::io::Result<usize> {
            self.largest_request = self.largest_request.max(buffer.len());
            if self.interrupted {
                self.interrupted = false;
                return Err(std::io::ErrorKind::Interrupted.into());
            }
            if self.fail_at == Some(self.read) {
                return Err(std::io::Error::new(
                    std::io::ErrorKind::Other,
                    "injected late read failure",
                ));
            }
            if self.read >= self.available {
                return Ok(0);
            }
            let mut count = (self.available - self.read).min(buffer.len() as u64);
            if let Some(fail_at) = self.fail_at {
                count = count.min(fail_at.saturating_sub(self.read));
                if count == 0 {
                    return Err(std::io::Error::new(
                        std::io::ErrorKind::Other,
                        "injected late read failure",
                    ));
                }
            }
            let count = count as usize;
            buffer[..count].fill(b'x');
            self.read += count as u64;
            Ok(count)
        }
    }

    #[test]
    fn streaming_collector_hashes_large_bodies_with_fixed_read_requests() {
        let length = (8 << 20) + 3;
        let content = vec![b'x'; length];
        let digest = format!("{:x}", Sha256::digest(&content));
        let mut reader = ProbeReader::new(length as u64);
        reader.interrupted = true;
        let mut entries = BTreeMap::new();
        let mut json_bytes = 0;
        let name = format!("blobs/sha256/{digest}");
        read_oci_blob(
            &mut entries,
            &name,
            length as i64,
            &mut reader,
            BlobSource::Layout,
            &mut json_bytes,
        )
        .unwrap();
        let blob = entries.get(&name).unwrap();
        assert_eq!(blob.hash, digest);
        assert_eq!(blob.size, length as i64);
        assert!(blob.data.is_none());
        assert!(reader.largest_request <= 64 * 1024);
    }

    #[test]
    fn streaming_collector_rejects_short_extra_and_late_read_failure_without_insert() {
        for (available, fail_at) in [(49, None), (51, None), (50, Some(50))] {
            let mut reader = ProbeReader::new(available);
            reader.fail_at = fail_at;
            let mut entries = BTreeMap::new();
            let mut json_bytes = 0;
            assert!(read_oci_blob(
                &mut entries,
                "test-entry",
                50,
                &mut reader,
                BlobSource::Layout,
                &mut json_bytes,
            )
            .is_err());
            assert!(entries.is_empty());
            assert_eq!(json_bytes, 0);
        }
    }

    #[test]
    fn streaming_collector_keeps_existing_metadata_and_cache_boundaries() {
        let exact = format!("{{}}{}", " ".repeat((4 << 20) - 2)).into_bytes();
        let oversized = format!("{} ", String::from_utf8(exact.clone()).unwrap()).into_bytes();
        let mut entries = BTreeMap::new();
        let mut json_bytes = 0;
        read_oci_blob(
            &mut entries,
            "exact-metadata",
            exact.len() as i64,
            Cursor::new(&exact),
            BlobSource::Layout,
            &mut json_bytes,
        )
        .unwrap();
        read_oci_blob(
            &mut entries,
            "oversized-metadata",
            oversized.len() as i64,
            Cursor::new(&oversized),
            BlobSource::Layout,
            &mut json_bytes,
        )
        .unwrap();
        assert_eq!(
            entries["exact-metadata"].data.as_ref().unwrap().len(),
            4 << 20
        );
        assert!(entries["oversized-metadata"].data.is_none());
        assert_eq!(json_bytes, 4 << 20);

        for index in 0..7 {
            let name = format!("cache-{index}");
            read_oci_blob(
                &mut entries,
                &name,
                exact.len() as i64,
                Cursor::new(&exact),
                BlobSource::Layout,
                &mut json_bytes,
            )
            .unwrap();
        }
        assert_eq!(json_bytes, 32 << 20);
        let small = b"{}";
        assert_eq!(
            read_oci_blob(
                &mut entries,
                "cache-overflow",
                small.len() as i64,
                Cursor::new(small),
                BlobSource::Layout,
                &mut json_bytes,
            )
            .unwrap_err(),
            Error::msg("OCI JSON metadata limit exceeded")
        );
        assert!(!entries.contains_key("cache-overflow"));
    }
}

// ---------------------------------------------------------------------------
// Image identity
// ---------------------------------------------------------------------------

fn validate_oci_layers(
    entries: &BTreeMap<String, Blob>,
    layers: &[Descriptor],
) -> Result<(), Error> {
    for layer in layers {
        match layer.media_type.as_str() {
            LAYER_TAR | LAYER_GZIP | LAYER_ZSTD => {}
            _ => return Err(Error::msg("unsupported OCI layer media type")),
        }
        fetch_oci_blob(entries, layer)?;
    }
    Ok(())
}

fn validate_oci_rootfs(diff_ids: &[String], layers: &[Descriptor]) -> Result<(), Error> {
    if diff_ids.len() != layers.len() {
        return Err(Error::msg("OCI rootfs/layer count mismatch"));
    }
    for (i, id) in diff_ids.iter().enumerate() {
        let hex = id.strip_prefix("sha256:").unwrap_or("");
        if !id.starts_with("sha256:") || !is_digest(hex) {
            return Err(Error::msg("invalid OCI diff ID"));
        }
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
    let base_hex = base_digest.strip_prefix("sha256:").unwrap_or("");
    if labels
        .get("org.opencontainers.image.source")
        .map(String::as_str)
        != Some("https://github.com/LevitateOS/sodaos")
        || labels
            .get("org.opencontainers.image.base.name")
            .map(String::is_empty)
            .unwrap_or(true)
        || !base_digest.starts_with("sha256:")
        || !is_digest(base_hex)
    {
        return Err(Error::msg("soda image lacks source/base attribution"));
    }
    Ok(())
}

pub(super) fn inspect_oci_config(
    config_blob: &Blob,
    layers: &[Descriptor],
    want: &str,
    revision: &str,
    image_digest: &str,
    config_digest: &str,
) -> Result<BuildImage, Error> {
    let empty = Vec::new();
    let cfg = parse_oci_config(config_blob.data.as_ref().unwrap_or(&empty))?;
    if cfg.rootfs_type != "layers" {
        return Err(Error::msg("OCI rootfs/layer count mismatch"));
    }
    validate_oci_rootfs(&cfg.diff_ids, layers)?;
    if cfg.os != "linux" || cfg.arch != want {
        return Err(Error::msg(format!("OCI must be linux/{want}")));
    }
    validate_oci_attribution(&cfg.labels, revision)?;
    let get = |key: &str| cfg.labels.get(key).cloned().unwrap_or_default();
    Ok(BuildImage {
        manifest: image_digest.to_string(),
        config: config_digest.to_string(),
        architecture: cfg.arch,
        revision: get("org.opencontainers.image.revision"),
        source: get("org.opencontainers.image.source"),
        base_name: get("org.opencontainers.image.base.name"),
        base_digest: get("org.opencontainers.image.base.digest"),
    })
}

pub(super) fn inspect_oci_image(
    entries: &BTreeMap<String, Blob>,
    image: &Descriptor,
    want: &str,
    revision: &str,
) -> Result<BuildImage, Error> {
    let manifest_blob = fetch_oci_blob(entries, image)?;
    let empty = Vec::new();
    let manifest = parse_oci_manifest(manifest_blob.data.as_ref().unwrap_or(&empty))?;
    validate_oci_layers(entries, &manifest.layers)?;
    let config_blob = fetch_oci_blob(entries, &manifest.config)?;
    inspect_oci_config(
        &config_blob,
        &manifest.layers,
        want,
        revision,
        &image.digest,
        &manifest.config.digest,
    )
}

#[cfg(test)]
mod json_admission_tests {
    use super::{parse_oci_config, parse_oci_manifest, read_oci_index, Descriptor, IndexDocument};
    use std::collections::BTreeMap;

    fn index_entries(layout: Vec<u8>, index: Vec<u8>) -> BTreeMap<String, super::super::Blob> {
        BTreeMap::from([
            (
                "oci-layout".to_string(),
                super::super::Blob {
                    data: Some(layout),
                    ..super::super::Blob::default()
                },
            ),
            (
                "index.json".to_string(),
                super::super::Blob {
                    data: Some(index),
                    ..super::super::Blob::default()
                },
            ),
        ])
    }

    #[test]
    fn oci_raw_slots_validate_only_the_last_known_value() {
        let descriptor: Descriptor = serde_json::from_str(
            r#"{"size":1.5,"size":-0,"digest":false,"digest":"sha256:abc","urls":2,"urls":[]}"#,
        )
        .unwrap();
        assert_eq!(descriptor.size, 0);
        assert_eq!(descriptor.digest, "sha256:abc");
        assert!(descriptor.urls.is_empty());

        assert!(serde_json::from_str::<Descriptor>(r#"{"size":-0.0}"#).is_err());
        assert!(serde_json::from_str::<Descriptor>(r#"{"size":1e0}"#).is_err());
        assert!(serde_json::from_str::<Descriptor>(r#"{"size":9223372036854775808}"#).is_err());
        assert!(serde_json::from_str::<Descriptor>(r#"{"size":-0,"size":1.25}"#).is_err());

        let index: IndexDocument = serde_json::from_str(
            r#"{"schemaVersion":false,"schemaVersion":2,"manifests":null,"manifests":[{}]}"#,
        )
        .unwrap();
        assert_eq!(index.schema_version, 2);
        assert_eq!(index.manifests.len(), 1);
    }

    #[test]
    fn consumed_layout_decoder_accepts_utf8_extensions_and_rejects_invalid_utf8() {
        let layout = r#"{"imageLayoutVersion":"1.0.0","extension":{"note":"café"}}"#.as_bytes();
        let index = r#"{"schemaVersion":2,"manifests":[],"extension":{"note":"café"}}"#.as_bytes();
        assert!(
            read_oci_index(&index_entries(layout.to_vec(), index.to_vec()))
                .unwrap()
                .is_empty()
        );

        let mut invalid_layout = br#"{"imageLayoutVersion":"1.0.0","extension":{"note":""#.to_vec();
        invalid_layout.push(0xff);
        invalid_layout.extend_from_slice(br#""}}"#);
        assert!(read_oci_index(&index_entries(invalid_layout, index.to_vec())).is_err());
    }

    #[test]
    fn consumed_index_decoder_accepts_utf8_extensions_and_rejects_invalid_utf8() {
        let layout = r#"{"imageLayoutVersion":"1.0.0","extension":{"note":"café"}}"#.as_bytes();
        let index = r#"{"schemaVersion":2,"manifests":[],"extension":{"note":"café"}}"#.as_bytes();
        assert!(
            read_oci_index(&index_entries(layout.to_vec(), index.to_vec()))
                .unwrap()
                .is_empty()
        );

        let mut invalid_index =
            br#"{"schemaVersion":2,"manifests":[],"extension":{"note":""#.to_vec();
        invalid_index.push(0xff);
        invalid_index.extend_from_slice(br#""}}"#);
        assert!(read_oci_index(&index_entries(layout.to_vec(), invalid_index)).is_err());
    }

    #[test]
    fn consumed_manifest_decoder_accepts_utf8_extensions_and_rejects_invalid_utf8() {
        let valid = r#"{"schemaVersion":2,"config":{"mediaType":"application/vnd.oci.image.config.v1+json"},"layers":[],"extension":{"note":"café"}}"#.as_bytes();
        assert!(parse_oci_manifest(valid).is_ok());

        let mut invalid = br#"{"schemaVersion":2,"config":{"mediaType":"application/vnd.oci.image.config.v1+json"},"layers":[],"extension":{"note":""#.to_vec();
        invalid.push(0xff);
        invalid.extend_from_slice(br#""}}"#);
        assert!(parse_oci_manifest(&invalid).is_err());
    }

    #[test]
    fn consumed_config_decoder_accepts_utf8_extensions_and_rejects_invalid_utf8() {
        let valid = r#"{"os":"linux","architecture":"amd64","rootfs":{"type":"layers","diff_ids":[]},"config":{"Labels":{}},"extension":{"note":"café"}}"#.as_bytes();
        assert!(parse_oci_config(valid).is_ok());

        let mut invalid = br#"{"os":"linux","architecture":"amd64","rootfs":{"type":"layers","diff_ids":[]},"config":{"Labels":{}},"extension":{"nested":{"note":""#.to_vec();
        invalid.push(0xff);
        invalid.extend_from_slice(br#""}}}"#);
        assert!(parse_oci_config(&invalid).is_err());
    }

    #[test]
    fn directory_layout_reader_validates_index_utf8_and_hashes_original_bytes() {
        let temp = tempfile::tempdir().unwrap();
        let fixture = include_bytes!("../../../soda-release-build/tests/data/go-fixture.oci");
        tar::Archive::new(std::io::Cursor::new(&fixture[..]))
            .unpack(temp.path())
            .unwrap();

        let index_path = temp.path().join("index.json");
        let mut index: serde_json::Value =
            serde_json::from_slice(&std::fs::read(&index_path).unwrap()).unwrap();
        let manifest_digest = index["manifests"][0]["digest"].as_str().unwrap();
        let manifest_path = temp
            .path()
            .join("blobs/sha256")
            .join(manifest_digest.strip_prefix("sha256:").unwrap());
        let manifest: serde_json::Value =
            serde_json::from_slice(&std::fs::read(manifest_path).unwrap()).unwrap();
        let config_digest = manifest["config"]["digest"].as_str().unwrap();
        let config_path = temp
            .path()
            .join("blobs/sha256")
            .join(config_digest.strip_prefix("sha256:").unwrap());
        let config: serde_json::Value =
            serde_json::from_slice(&std::fs::read(config_path).unwrap()).unwrap();
        let revision = config["config"]["Labels"]["org.opencontainers.image.revision"]
            .as_str()
            .unwrap();
        assert_eq!(revision, "a".repeat(40));

        let config_reference = config_digest.to_string();
        index["manifests"][0]["annotations"]["org.opencontainers.image.ref.name"] =
            serde_json::json!(config_reference);
        index["extension"] = serde_json::json!({"note": "café"});
        let index_bytes = serde_json::to_vec(&index).unwrap();
        std::fs::write(&index_path, &index_bytes).unwrap();

        let selection = BTreeMap::from([(config_reference.clone(), revision.to_string())]);
        let layout =
            crate::oci::inspect_oci_layout(temp.path().to_str().unwrap(), "x86_64", &selection)
                .unwrap();
        assert_eq!(layout.images.len(), 1);
        assert_eq!(
            layout.files["index.json"],
            crate::hash_bytes(&index_bytes)
                .strip_prefix("sha256:")
                .unwrap()
        );

        let mut invalid_index = index_bytes.clone();
        let cafe = "café".as_bytes();
        let at = invalid_index
            .windows(cafe.len())
            .position(|window| window == cafe)
            .unwrap();
        invalid_index[at + 3] = 0xff;
        std::fs::write(index_path, invalid_index).unwrap();
        assert!(crate::oci::inspect_oci_layout(
            temp.path().to_str().unwrap(),
            "x86_64",
            &selection,
        )
        .is_err());
    }
}
