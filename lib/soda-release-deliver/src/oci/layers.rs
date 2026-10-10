use std::collections::{BTreeMap, BTreeSet};
use std::io::Read;

use flate2::read::MultiGzDecoder;
use sha2::{Digest as _, Sha256};

use crate::model::path_clean;
use crate::Error;

use super::schema::Descriptor;
use super::{LAYER_GZIP, LAYER_TAR, LAYER_ZSTD};

// ---------------------------------------------------------------------------
// Content verification
// ---------------------------------------------------------------------------

#[derive(Debug, Clone, Default)]
pub struct LayerMember {
    pub hash: String,
    pub present: bool,
    pub blocked: bool,
}

// Embedded compressed images in the current host producer can decode beyond
// 1 GiB; the image-wide decoded budget remains the aggregate backstop.
pub const MAX_OCI_LAYER_BYTES: u64 = 2 << 30;
pub const MAX_OCI_LAYER_COMPRESSED_BYTES: u64 = 1 << 30;
pub const MAX_OCI_IMAGE_LAYER_BYTES: u64 = 16 << 30;
pub const MAX_OCI_IMAGE_COMPRESSED_BYTES: u64 = 4 << 30;
pub const MAX_OCI_MEMBER_BYTES: u64 = 512 << 20;

struct BoundedReader<'a, R: ?Sized> {
    inner: &'a mut R,
    read: u64,
    limit: u64,
}

impl<R: Read + ?Sized> Read for BoundedReader<'_, R> {
    fn read(&mut self, out: &mut [u8]) -> std::io::Result<usize> {
        if out.is_empty() {
            return Ok(0);
        }
        if self.read >= self.limit {
            let mut probe = [0; 1];
            return match self.inner.read(&mut probe)? {
                0 => Ok(0),
                _ => Err(std::io::Error::new(
                    std::io::ErrorKind::InvalidData,
                    "OCI layer limit exceeded",
                )),
            };
        }
        let cap = (self.limit - self.read).min(out.len() as u64) as usize;
        let n = self.inner.read(&mut out[..cap])?;
        self.read += n as u64;
        Ok(n)
    }
}

pub fn requested_oci_paths(paths: &[String]) -> Result<BTreeMap<String, String>, Error> {
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

fn scan_oci_layer_inner<R: Read + ?Sized>(
    reader: &mut R,
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
        if seen.len() > 100000 {
            return Err(Error::msg("too many OCI layer entries"));
        }
        let header = item.header().clone();
        let size = header
            .size()
            .map_err(|e| Error::msg(format!("read OCI layer: {e}")))?;
        if wanted.contains_key(&name) && size > MAX_OCI_MEMBER_BYTES {
            return Err(Error::msg("OCI layer member limit exceeded"));
        }
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

/// Scan a decompressed OCI tar layer and consume its complete stream. The
/// caller retains the decoder so gzip trailer validation occurs at EOF.
/// Concatenated gzip members are consumed; after the tar terminator, only zero
/// padding is admitted.
pub fn scan_oci_layer<R: Read + ?Sized>(
    reader: &mut R,
    wanted: &BTreeMap<String, String>,
) -> Result<BTreeMap<String, LayerMember>, String> {
    let mut aggregate = 0;
    scan_oci_layer_with_budget(reader, wanted, &mut aggregate)
}

/// Scan a layer while charging its decoded bytes to an image-wide budget.
pub fn scan_oci_layer_with_budget<R: Read + ?Sized>(
    reader: &mut R,
    wanted: &BTreeMap<String, String>,
    aggregate: &mut u64,
) -> Result<BTreeMap<String, LayerMember>, String> {
    let remaining = MAX_OCI_IMAGE_LAYER_BYTES
        .checked_sub(*aggregate)
        .ok_or_else(|| "OCI image decoded limit exceeded".to_string())?;
    let mut bounded = BoundedReader {
        inner: reader,
        read: 0,
        limit: MAX_OCI_LAYER_BYTES.min(remaining),
    };
    let found = scan_oci_layer_inner(&mut bounded, wanted).map_err(|e| e.0)?;
    drain_tar_padding(&mut bounded)
        .map_err(|_| "OCI layer limit, trailing data, or stream error".to_string())?;
    *aggregate = (*aggregate)
        .checked_add(bounded.read)
        .ok_or_else(|| "OCI image decoded limit exceeded".to_string())?;
    Ok(found)
}

pub(super) fn layer_archive_indexes(
    layers: &[Descriptor],
) -> Result<BTreeMap<String, Vec<usize>>, Error> {
    if layers.len() > 100_000 {
        return Err(Error::msg("too many OCI layers"));
    }
    let mut indexes: BTreeMap<String, Vec<usize>> = BTreeMap::new();
    let mut seen: BTreeMap<String, &Descriptor> = BTreeMap::new();
    let mut compressed_total = 0u64;
    for (i, layer) in layers.iter().enumerate() {
        let hex = layer.digest.strip_prefix("sha256:").unwrap_or("\x00");
        let name = format!("blobs/sha256/{hex}");
        if let Some(prior) = seen.get(&name) {
            if prior.media_type != layer.media_type || prior.size != layer.size {
                return Err(Error::msg("conflicting OCI layer descriptors"));
            }
        }
        if !seen.contains_key(&name) {
            let size =
                u64::try_from(layer.size).map_err(|_| Error::msg("invalid OCI layer size"))?;
            compressed_total = compressed_total
                .checked_add(size)
                .ok_or_else(|| Error::msg("OCI image compressed limit exceeded"))?;
            if compressed_total > MAX_OCI_IMAGE_COMPRESSED_BYTES {
                return Err(Error::msg("OCI image compressed limit exceeded"));
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
    size: u64,
    limit: u64,
}

impl<R: Read> Read for TeeHasher<R> {
    fn read(&mut self, buf: &mut [u8]) -> std::io::Result<usize> {
        if buf.is_empty() {
            return Ok(0);
        }
        if self.size >= self.limit {
            let mut probe = [0; 1];
            return match self.inner.read(&mut probe)? {
                0 => Ok(0),
                _ => Err(std::io::Error::new(
                    std::io::ErrorKind::InvalidData,
                    "OCI layer size exceeded",
                )),
            };
        }
        let cap = (self.limit - self.size).min(buf.len() as u64) as usize;
        let n = self.inner.read(&mut buf[..cap])?;
        self.size += n as u64;
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

fn drain_tar_padding(reader: &mut dyn Read) -> std::io::Result<()> {
    let mut chunk = [0u8; 65536];
    loop {
        let n = reader.read(&mut chunk)?;
        if n == 0 {
            return Ok(());
        }
        if chunk[..n].iter().any(|byte| *byte != 0) {
            return Err(std::io::Error::new(
                std::io::ErrorKind::InvalidData,
                "non-padding data after OCI tar terminator",
            ));
        }
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use flate2::write::GzEncoder;
    use sha2::{Digest as _, Sha256};
    use std::io::Write;

    fn tar_bytes(name: &str, body: &[u8]) -> Vec<u8> {
        let mut tar = tar::Builder::new(Vec::new());
        let mut header = tar::Header::new_gnu();
        header.set_size(body.len() as u64);
        header.set_mode(0o644);
        header.set_cksum();
        tar.append_data(&mut header, name, body).unwrap();
        tar.into_inner().unwrap()
    }

    fn gzip(data: &[u8]) -> Vec<u8> {
        let mut encoder = GzEncoder::new(Vec::new(), flate2::Compression::default());
        encoder.write_all(data).unwrap();
        encoder.finish().unwrap()
    }

    fn descriptor(bytes: &[u8]) -> Descriptor {
        Descriptor {
            digest: format!("sha256:{:x}", Sha256::digest(bytes)),
            size: bytes.len() as i64,
            media_type: LAYER_GZIP.to_string(),
            ..Descriptor::default()
        }
    }

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
        let sum: u32 = header.iter().map(|byte| *byte as u32).sum();
        let checksum = format!("{:06o}\0 ", sum);
        header[148..156].copy_from_slice(checksum.as_bytes());
        let mut out = Vec::new();
        out.extend_from_slice(&header);
        out.extend_from_slice(body);
        out.resize(out.len() + (512 - body.len() % 512) % 512, 0);
        out.extend_from_slice(&[0u8; 1024]);
        out
    }

    #[test]
    fn oracle_layer_scanner_vectors() {
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
        let wanted = BTreeMap::from([("wanted".to_string(), "/wanted".to_string())]);
        let dup = layer_bytes(&["wanted", "wanted"]);
        assert_eq!(
            scan_oci_layer(&mut &dup[..], &wanted).unwrap_err().as_str(),
            "duplicate OCI layer entry"
        );
        let evil = raw_tar_entry("../wanted", b"evil");
        assert_eq!(
            scan_oci_layer(&mut &evil[..], &wanted)
                .unwrap_err()
                .as_str(),
            "unsafe OCI layer path"
        );
        let benign = layer_bytes(&[".", "wanted"]);
        let members = scan_oci_layer(&mut &benign[..], &wanted).unwrap();
        assert!(members.get("wanted").unwrap().present);

        let lower = BTreeMap::from([(
            "wanted".to_string(),
            LayerMember {
                hash: "a".repeat(64),
                present: true,
                blocked: false,
            },
        )]);
        let upper_bytes = layer_bytes(&[".wh.wanted"]);
        let upper = scan_oci_layer(&mut &upper_bytes[..], &wanted).unwrap();
        assert_eq!(
            resolve_oci_members(&[lower, upper], &[false, false], &wanted)
                .unwrap_err()
                .0,
            "requested OCI member was removed"
        );
    }

    #[test]
    fn oracle_gzip_and_zstd_layers() {
        let content = b"verified member";
        let layer = tar_bytes("wanted", content);
        let compressed = gzip(&layer);
        let wanted = BTreeMap::from([("wanted".to_string(), "/wanted".to_string())]);
        for (media_type, data, blocked) in [
            (LAYER_TAR, layer, false),
            (LAYER_GZIP, compressed.clone(), false),
            (LAYER_ZSTD, compressed.clone(), true),
        ] {
            let descriptor = Descriptor {
                digest: format!("sha256:{:x}", Sha256::digest(&data)),
                size: data.len() as i64,
                media_type: media_type.to_string(),
                ..Descriptor::default()
            };
            let mut aggregate = 0;
            let (found, unsupported) = scan_archive_layer_with_budget(
                &mut &data[..],
                &descriptor,
                &wanted,
                &mut aggregate,
            )
            .unwrap();
            assert_eq!(unsupported, blocked);
            if !blocked {
                assert_eq!(
                    found.get("wanted").unwrap().hash,
                    format!("{:x}", Sha256::digest(content))
                );
            }
        }

        let mut corrupt = compressed;
        let last = corrupt.len() - 1;
        corrupt[last] ^= 0xff;
        let descriptor = Descriptor {
            digest: format!("sha256:{:x}", Sha256::digest(&corrupt)),
            size: corrupt.len() as i64,
            media_type: LAYER_GZIP.to_string(),
            ..Descriptor::default()
        };
        assert!(scan_archive_layer(&mut &corrupt[..], &descriptor, &wanted).is_err());
    }

    #[test]
    fn scan_validates_gzip_trailer_and_tar_tail() {
        let wanted: BTreeMap<String, String> = [("wanted".to_string(), "/wanted".to_string())]
            .into_iter()
            .collect();
        let valid = gzip(&tar_bytes("wanted", b"data"));
        assert!(scan_archive_layer(&mut &valid[..], &descriptor(&valid), &wanted).is_ok());

        let mut bad_crc = valid.clone();
        let crc = bad_crc.len() - 8;
        bad_crc[crc] ^= 1;
        assert!(scan_archive_layer(&mut &bad_crc[..], &descriptor(&bad_crc), &wanted).is_err());

        let mut bad_size = valid.clone();
        *bad_size.last_mut().unwrap() ^= 1;
        assert!(scan_archive_layer(&mut &bad_size[..], &descriptor(&bad_size), &wanted).is_err());

        let mut trailing = valid;
        trailing.extend(gzip(b"second tar stream"));
        assert!(scan_archive_layer(&mut &trailing[..], &descriptor(&trailing), &wanted).is_err());
    }

    #[test]
    fn scan_enforces_member_and_image_budgets_before_payload_reads() {
        let wanted: BTreeMap<String, String> = [("wanted".to_string(), "/wanted".to_string())]
            .into_iter()
            .collect();
        let mut header = tar::Header::new_gnu();
        header.set_path("wanted").unwrap();
        header.set_size(MAX_OCI_MEMBER_BYTES + 1);
        header.set_mode(0o644);
        header.set_cksum();
        let oversized_header = header.as_bytes();
        let mut used = 0;
        assert_eq!(
            scan_oci_layer_with_budget(&mut &oversized_header[..], &wanted, &mut used).unwrap_err(),
            "OCI layer member limit exceeded"
        );

        used = MAX_OCI_IMAGE_LAYER_BYTES;
        let layer = tar_bytes("wanted", b"data");
        assert!(scan_oci_layer_with_budget(&mut &layer[..], &wanted, &mut used).is_err());

        let oversized = Descriptor {
            size: (MAX_OCI_LAYER_COMPRESSED_BYTES + 1) as i64,
            media_type: LAYER_GZIP.to_string(),
            ..Descriptor::default()
        };
        assert!(scan_archive_layer(&mut &[][..], &oversized, &wanted).is_err());
    }
}

#[cfg(test)]
pub(super) fn scan_archive_layer<R: Read>(
    reader: &mut R,
    descriptor: &Descriptor,
    wanted: &BTreeMap<String, String>,
) -> Result<(BTreeMap<String, LayerMember>, bool), Error> {
    let mut aggregate = 0;
    scan_archive_layer_with_budget(reader, descriptor, wanted, &mut aggregate)
}

pub(super) fn scan_archive_layer_with_budget<R: Read>(
    reader: &mut R,
    descriptor: &Descriptor,
    wanted: &BTreeMap<String, String>,
    aggregate: &mut u64,
) -> Result<(BTreeMap<String, LayerMember>, bool), Error> {
    let changed = || Error::msg("OCI layer changed during member verification");
    if descriptor.size < 0 || descriptor.size as u64 > MAX_OCI_LAYER_COMPRESSED_BYTES {
        return Err(Error::msg("OCI layer compressed limit exceeded"));
    }
    let mut tee = TeeHasher {
        inner: reader,
        hasher: Sha256::new(),
        size: 0,
        limit: u64::try_from(descriptor.size).map_err(|_| Error::msg("invalid OCI layer size"))?,
    };
    if descriptor.media_type == LAYER_ZSTD {
        if drain(&mut tee).is_err() {
            return Err(changed());
        }
        if tee.size != tee.limit {
            return Err(changed());
        }
        let sum = format!("sha256:{:x}", tee.hasher.clone().finalize());
        if sum != descriptor.digest {
            return Err(changed());
        }
        return Ok((BTreeMap::new(), true));
    }
    let members: BTreeMap<String, LayerMember> = match descriptor.media_type.as_str() {
        LAYER_TAR => scan_oci_layer_with_budget(&mut tee, wanted, aggregate).map_err(Error::msg)?,
        LAYER_GZIP => {
            let mut gz = MultiGzDecoder::new(&mut tee);
            scan_oci_layer_with_budget(&mut gz, wanted, aggregate).map_err(Error::msg)?
        }
        _ => return Err(Error::msg("unsupported OCI layer media type")),
    };
    if drain(&mut tee).is_err() {
        return Err(changed());
    }
    if tee.size != tee.limit {
        return Err(changed());
    }
    let sum = format!("sha256:{:x}", tee.hasher.clone().finalize());
    if sum != descriptor.digest {
        return Err(changed());
    }
    Ok((members, false))
}

pub(super) fn resolve_oci_members(
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
