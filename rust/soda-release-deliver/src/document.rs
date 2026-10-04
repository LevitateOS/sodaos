//! `document.go`: OCI document packaging plus confined file/JSON readers.

use soda_json::JsonValue;

use crate::buildx::{fresh_directory, read_at, write_new, Root};
use crate::jsonx::{as_i64, marshal, parse_strict, Emit, Soft};
use crate::{hash_bytes, is_digest_ref, Error};

pub const MANIFEST_TYPE: &str = "application/vnd.oci.image.manifest.v1+json";
pub const LAYER_TYPE: &str = "application/vnd.oci.image.layer.v1.tar";

#[derive(Debug, Clone, Default)]
pub struct Descriptor {
    pub media_type: String,
    pub digest: String,
    pub size: i64,
}

fn emit_descriptor(d: &Descriptor) -> String {
    format!(
        "{{\"mediaType\":{:?},\"digest\":{:?},\"size\":{}}}",
        d.media_type, d.digest, d.size
    )
}

fn put_document_blob(blobs: &str, data: &[u8], media: &str) -> Result<Descriptor, Error> {
    let hash = hash_bytes(data);
    let hex = hash.strip_prefix("sha256:").unwrap_or("");
    write_new(&format!("{blobs}/{hex}"), data, 0o600)?;
    Ok(Descriptor {
        media_type: media.to_string(),
        digest: hash,
        size: data.len() as i64,
    })
}

fn write_octal(field: &mut [u8], value: u64) {
    // Go USTAR numeric: octal digits, NUL-terminated (mode 8, size/mtime 12,
    // devmajor/devminor 8 bytes wide).
    let width = field.len();
    let digits = format!("{value:o}");
    let start = width - 1 - digits.len();
    for (i, b) in digits.bytes().enumerate() {
        field[start + i] = b;
    }
    for slot in field.iter_mut().take(start) {
        *slot = b'0';
    }
    field[width - 1] = 0;
}

fn document_layer(data: &[u8]) -> Result<Vec<u8>, Error> {
    // Hand-rolled USTAR header matching Go's archive/tar writer byte for
    // byte: {Name: record.json, Mode: 0444, Size, Typeflag: '0'}, all other
    // fields zero.
    let mut header = [0u8; 512];
    header[..11].copy_from_slice(b"record.json");
    write_octal(&mut header[100..108], 0o444);
    write_octal(&mut header[108..116], 0);
    write_octal(&mut header[116..124], 0);
    write_octal(&mut header[124..136], data.len() as u64);
    write_octal(&mut header[136..148], 0);
    header[156] = b'0';
    // linkname 157..257 stays zero; magic follows at 257.
    header[257..263].copy_from_slice(b"ustar\0");
    header[263..265].copy_from_slice(b"00");
    write_octal(&mut header[329..337], 0);
    write_octal(&mut header[337..345], 0);
    let mut sum: u64 = 0;
    for (i, b) in header.iter().enumerate() {
        sum += if (148..156).contains(&i) { b' ' as u64 } else { *b as u64 };
    }
    let digits = format!("{sum:06o}");
    header[148..154].copy_from_slice(digits.as_bytes());
    header[154] = 0;
    header[155] = b' ';
    let mut layer = Vec::with_capacity(512 + data.len().next_multiple_of(512) + 1024);
    layer.extend_from_slice(&header);
    layer.extend_from_slice(data);
    layer.resize(512 + data.len().next_multiple_of(512), 0);
    layer.extend_from_slice(&[0u8; 1024]);
    Ok(layer)
}

fn write_document_index(path: &str, md: &Descriptor) -> Result<(), Error> {
    let index = format!(
        "{{\"manifests\":[{}],\"schemaVersion\":2}}",
        emit_descriptor(md)
    );
    write_new(&format!("{path}/index.json"), index.as_bytes(), 0o600)?;
    write_new(
        &format!("{path}/oci-layout"),
        br#"{"imageLayoutVersion":"1.0.0"}"#,
        0o600,
    )
}

fn write_document_blobs(path: &str, data: &[u8]) -> Result<String, Error> {
    let blobs = format!("{path}/blobs/sha256");
    std::fs::DirBuilder::new()
        .recursive(true)
        .create(&blobs)
        .map_err(|e| Error::msg(format!("mkdir {blobs}: {e}")))?;
    // Go uses MkdirAll with 0o700; enforce the same mode explicitly.
    set_mode(&blobs, 0o700)?;
    let layer = document_layer(data)?;
    let layer_desc = put_document_blob(&blobs, &layer, LAYER_TYPE)?;
    let config = format!(
        "{{\"architecture\":\"unknown\",\"os\":\"unknown\",\"rootfs\":{{\"diff_ids\":[\"{}\"],\"type\":\"layers\"}}}}",
        layer_desc.digest
    );
    let config_desc = put_document_blob(
        &blobs,
        config.as_bytes(),
        "application/vnd.oci.image.config.v1+json",
    )?;
    let manifest = format!(
        "{{\"schemaVersion\":2,\"mediaType\":{MANIFEST_TYPE:?},\"config\":{},\"layers\":[{}]}}",
        emit_descriptor(&config_desc),
        emit_descriptor(&layer_desc)
    );
    let manifest_desc = put_document_blob(&blobs, manifest.as_bytes(), MANIFEST_TYPE)?;
    write_document_index(path, &manifest_desc)?;
    Ok(manifest_desc.digest)
}

fn set_mode(path: &str, mode: u32) -> Result<(), Error> {
    use std::os::unix::fs::PermissionsExt;
    std::fs::set_permissions(path, std::fs::Permissions::from_mode(mode))
        .map_err(|e| Error::msg(format!("chmod {path}: {e}")))
}

/// `WriteDocument`: package one bounded JSON record as an OCI layout.
pub fn write_document<T: Emit + ?Sized>(path: &str, value: &T) -> Result<String, Error> {
    let data = marshal(value);
    if data.len() > 1 << 20 {
        return Err(Error::refused());
    }
    fresh_directory(path)?;
    write_document_blobs(path, &data)
}

/// `ReadFile`: bounded regular read through a confined directory handle.
pub fn read_file(path: &str, maximum: i64) -> Result<Vec<u8>, Error> {
    let (parent, base) = match path.rfind('/') {
        Some(0) => ("/", &path[1..]),
        Some(i) => (&path[..i], &path[i + 1..]),
        None => (".", path),
    };
    let root = Root::open(parent)?;
    read_at(&root, base, maximum)
}

/// `ReadJSON`: strict-decode a bounded JSON file.
pub fn read_json<T>(
    path: &str,
    decode: impl Fn(&JsonValue) -> Result<T, String>,
) -> Result<T, Error> {
    let data = read_file(path, 1 << 20)?;
    let value = parse_strict(&data)?;
    decode(&value).map_err(|_| Error::refused())
}

fn decode_document_manifest(data: &[u8], want: &str) -> Result<OciManifestDoc, Error> {
    if hash_bytes(data) != want {
        return Err(Error::refused());
    }
    let value = parse_strict(data)?;
    let manifest = OciManifestDoc::decode(&value).map_err(|_| Error::refused())?;
    if manifest.schema_version != 2
        || manifest.media_type != MANIFEST_TYPE
        || manifest.layers.len() != 1
        || manifest.layers[0].media_type != LAYER_TYPE
        || manifest.config.media_type != "application/vnd.oci.image.config.v1+json"
    {
        return Err(Error::refused());
    }
    Ok(manifest)
}

#[derive(Debug, Clone, Default)]
struct OciManifestDoc {
    schema_version: i64,
    media_type: String,
    config: Descriptor,
    layers: Vec<Descriptor>,
}

impl OciManifestDoc {
    fn decode(value: &JsonValue) -> Result<OciManifestDoc, String> {
        let soft = Soft::new(value).map_err(|_| "invalid manifest".to_string())?;
        // Strict shape: unknown fields refused, like Go's DisallowUnknownFields.
        if let Some(entries) = soft.entries() {
            for (key, _) in entries {
                match key.as_str() {
                    "schemaVersion" | "mediaType" | "config" | "layers" => {}
                    _ => return Err(format!("json: unknown field \"{key}\"")),
                }
            }
        }
        let config = match soft.field("config") {
            Some(v) => decode_doc_descriptor(v)?,
            None => Descriptor::default(),
        };
        let mut layers = Vec::new();
        if let Some(items) = soft.array("layers").map_err(|_| "invalid layers".to_string())? {
            for item in items {
                layers.push(decode_doc_descriptor(item)?);
            }
        }
        Ok(OciManifestDoc {
            schema_version: as_i64(soft.integer("schemaVersion").map_err(|_| "invalid schemaVersion".to_string())?.unwrap_or(0))
                .map_err(|_| "invalid schemaVersion".to_string())?,
            media_type: soft.string("mediaType").map_err(|_| "invalid mediaType".to_string())?.unwrap_or_default(),
            config,
            layers,
        })
    }
}

fn decode_doc_descriptor(value: &JsonValue) -> Result<Descriptor, String> {
    let soft = Soft::new(value).map_err(|_| "invalid descriptor".to_string())?;
    if let Some(entries) = soft.entries() {
        for (key, _) in entries {
            match key.as_str() {
                "mediaType" | "digest" | "size" => {}
                _ => return Err(format!("json: unknown field \"{key}\"")),
            }
        }
    }
    Ok(Descriptor {
        media_type: soft.string("mediaType").map_err(|_| "invalid descriptor".to_string())?.unwrap_or_default(),
        digest: soft.string("digest").map_err(|_| "invalid descriptor".to_string())?.unwrap_or_default(),
        size: as_i64(soft.integer("size").map_err(|_| "invalid descriptor".to_string())?.unwrap_or(0))
            .map_err(|_| "invalid descriptor".to_string())?,
    })
}

fn read_document_blob(root: &Root, d: &Descriptor) -> Result<Vec<u8>, Error> {
    if !is_digest_ref(&d.digest) || d.size < 0 || d.size > 2 << 20 {
        return Err(Error::refused());
    }
    let name = d.digest.strip_prefix("sha256:").unwrap_or("");
    let data = read_at(root, name, 2 << 20)?;
    if data.len() as i64 != d.size || hash_bytes(&data) != d.digest {
        return Err(Error::refused());
    }
    Ok(data)
}

fn read_document_record(data: &[u8]) -> Result<Vec<u8>, Error> {
    let mut archive = tar::Archive::new(data);
    let mut items = archive
        .entries()
        .map_err(|_| Error::refused())?;
    let first = match items.next() {
        Some(Ok(entry)) => entry,
        _ => return Err(Error::refused()),
    };
    let name = String::from_utf8_lossy(&first.path_bytes()).into_owned();
    let header = first.header().clone();
    let size = header.size().map_err(|_| Error::refused())?;
    if name != "record.json" || !header.entry_type().is_file() || size > 1 << 20 {
        return Err(Error::refused());
    }
    let mut record = Vec::new();
    use std::io::Read;
    first
        .take((1 << 20) + 1)
        .read_to_end(&mut record)
        .map_err(|e| Error::msg(format!("read document: {e}")))?;
    match items.next() {
        None => {}
        Some(_) => return Err(Error::refused()),
    }
    Ok(record)
}

/// `ReadDocument`: read one verified document from a fresh directory copy.
pub fn read_document<T>(
    path: &str,
    want: &str,
    decode: impl Fn(&JsonValue) -> Result<T, String>,
) -> Result<T, Error> {
    let root = Root::open(path)?;
    let manifest_bytes = read_at(&root, "manifest.json", 64 << 10).map_err(|_| Error::refused())?;
    let manifest = decode_document_manifest(&manifest_bytes, want)?;
    read_document_blob(&root, &manifest.config)?;
    let data = read_document_blob(&root, &manifest.layers[0])?;
    let record = read_document_record(&data)?;
    let value = parse_strict(&record)?;
    decode(&value).map_err(|_| Error::refused())
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::model::Channel;

    fn temp_dir(prefix: &str) -> String {
        let dir = std::env::temp_dir().join(format!(
            "{prefix}-{}-{}",
            std::process::id(),
            std::time::SystemTime::now()
                .duration_since(std::time::UNIX_EPOCH)
                .unwrap()
                .as_nanos()
        ));
        std::fs::create_dir_all(&dir).unwrap();
        dir.to_string_lossy().into_owned()
    }

    #[test]
    fn document_round_trip_and_bounds() {
        let dir = temp_dir("srd-doc-unit");
        let channel = Channel {
            format: 1,
            name: "candidate".to_string(),
            sequence: 3,
            issued: 100,
            expires: 200,
            releases: [("x86_64".to_string(), "r".to_string())].into_iter().collect(),
            ..Channel::default()
        };
        let out = format!("{dir}/doc");
        let digest = write_document(&out, &channel).unwrap();
        assert!(digest.starts_with("sha256:"));
        // Oversize records refuse before any write.
        let big = "x".repeat(1 << 20);
        assert_eq!(
            write_document(&format!("{dir}/big"), &big).unwrap_err(),
            Error::refused()
        );
        // Oversize reads refuse.
        let root = Root::open(&dir).unwrap();
        assert_eq!(
            read_at(&root, "doc", 8).unwrap_err(),
            Error::refused()
        );
    }
}
