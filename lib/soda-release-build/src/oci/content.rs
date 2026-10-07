//! Multi-layer content resolution: descriptor indexes, hashing input
//! drains, gzip/zstd handling, and reverse overlay member resolution.

use super::archive::entry_raw_name;
use super::{hex_digest, Descriptor, LAYER_TAR, LAYER_TAR_GZIP, LAYER_TAR_ZSTD};
use crate::{path_clean, Error};
use sha2::Digest;
use soda_release_deliver::oci::{
    scan_oci_layer_with_budget, LayerMember, MAX_OCI_IMAGE_COMPRESSED_BYTES,
    MAX_OCI_LAYER_COMPRESSED_BYTES,
};
use std::collections::{BTreeMap, HashMap};
use std::io::Read;

/// Per-layer member scans plus per-layer zstd-blocked flags.
pub(super) type LayerScans = (Vec<HashMap<String, LayerMember>>, Vec<bool>);

fn layer_archive_indexes(layers: &[Descriptor]) -> Result<HashMap<String, Vec<usize>>, Error> {
    if layers.len() > 100_000 {
        return Err(Error::msg("too many OCI layers"));
    }
    let mut indexes: HashMap<String, Vec<usize>> = HashMap::new();
    let mut seen: HashMap<String, &Descriptor> = HashMap::new();
    let mut compressed_total = 0u64;
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

/// Hashing tee: every byte read is digested, like Go's `io.TeeReader`.
struct HashReader<R> {
    inner: R,
    hasher: sha2::Sha256,
    size: u64,
    limit: u64,
}

impl<R: Read> HashReader<R> {
    fn new(inner: R, limit: u64) -> HashReader<R> {
        HashReader {
            inner,
            hasher: sha2::Sha256::new(),
            size: 0,
            limit,
        }
    }

    fn hex(&self) -> String {
        hex_digest(&self.hasher.clone().finalize())
    }
}

impl<R: Read> Read for HashReader<R> {
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
        use sha2::Digest;
        self.hasher.update(&buf[..n]);
        Ok(n)
    }
}

fn scan_layer_reader(
    layer: &mut dyn Read,
    wanted: &BTreeMap<String, String>,
    aggregate: &mut u64,
) -> Result<HashMap<String, LayerMember>, Error> {
    scan_oci_layer_with_budget(layer, wanted, aggregate).map_err(Error::msg)
}

#[cfg(test)]
pub(super) fn scan_archive_layer(
    reader: &mut dyn Read,
    descriptor: &Descriptor,
    wanted: &BTreeMap<String, String>,
) -> Result<(HashMap<String, LayerMember>, bool), Error> {
    let mut aggregate = 0;
    scan_archive_layer_with_budget(reader, descriptor, wanted, &mut aggregate)
}

fn scan_archive_layer_with_budget(
    reader: &mut dyn Read,
    descriptor: &Descriptor,
    wanted: &BTreeMap<String, String>,
    aggregate: &mut u64,
) -> Result<(HashMap<String, LayerMember>, bool), Error> {
    let limit = u64::try_from(descriptor.size).map_err(|_| Error::msg("invalid OCI layer size"))?;
    if limit > MAX_OCI_LAYER_COMPRESSED_BYTES {
        return Err(Error::msg("OCI layer compressed limit exceeded"));
    }
    if descriptor.media_type == LAYER_TAR_ZSTD {
        let mut raw = HashReader::new(reader, limit);
        let mut sink = std::io::sink();
        std::io::copy(&mut raw, &mut sink)
            .map_err(|_| Error::msg("OCI layer changed during member verification"))?;
        if raw.size != limit {
            return Err(Error::msg("OCI layer changed during member verification"));
        }
        let hex = raw.hex();
        if format!("sha256:{hex}") != descriptor.digest {
            return Err(Error::msg("OCI layer changed during member verification"));
        }
        return Ok((HashMap::new(), true));
    }
    let mut raw = HashReader::new(reader, limit);
    let (members, scan_err) = match descriptor.media_type.as_str() {
        LAYER_TAR => (scan_layer_reader(&mut raw, wanted, aggregate), None),
        LAYER_TAR_GZIP => {
            let mut gz = flate2::read::MultiGzDecoder::new(&mut raw);
            let members = scan_layer_reader(&mut gz, wanted, aggregate);
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
    if raw.size != limit {
        return Err(Error::msg("OCI layer changed during member verification"));
    }
    if format!("sha256:{}", raw.hex()) != descriptor.digest {
        return Err(Error::msg("OCI layer changed during member verification"));
    }
    Ok((members, false))
}

pub(super) fn scan_oci_archive_layers(
    reader: &mut dyn Read,
    layers: &[Descriptor],
    wanted: &BTreeMap<String, String>,
) -> Result<LayerScans, Error> {
    let indexes = layer_archive_indexes(layers)?;
    let mut found: Vec<HashMap<String, LayerMember>> = Vec::with_capacity(layers.len());
    found.resize_with(layers.len(), HashMap::new);
    let mut unsupported = vec![false; layers.len()];
    let mut decoded_total = 0u64;
    let mut archive = tar::Archive::new(reader);
    let listed = archive.entries().map_err(|e| Error::msg(e.to_string()))?;
    for entry in listed {
        let mut entry = entry.map_err(|e| Error::msg(e.to_string()))?;
        let name = path_clean(&entry_raw_name(&entry));
        let positions = indexes.get(&name).cloned().unwrap_or_default();
        if positions.is_empty() {
            continue;
        }
        let (members, blocked) = scan_archive_layer_with_budget(
            &mut entry,
            &layers[positions[0]],
            wanted,
            &mut decoded_total,
        )?;
        for position in positions {
            found[position] = members.clone();
            unsupported[position] = blocked;
        }
    }
    Ok((found, unsupported))
}

pub(super) fn resolve_oci_members(
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
