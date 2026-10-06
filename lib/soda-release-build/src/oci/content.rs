//! Multi-layer content resolution: descriptor indexes, hashing input
//! drains, gzip/zstd handling, and reverse overlay member resolution.

use super::archive::entry_raw_name;
use super::layers::{scan_oci_layer, LayerMember};
use super::{hex_digest, Descriptor, LAYER_TAR, LAYER_TAR_GZIP, LAYER_TAR_ZSTD};
use crate::{path_clean, Error};
use sha2::Digest;
use std::collections::{BTreeMap, HashMap};
use std::io::Read;

/// Per-layer member scans plus per-layer zstd-blocked flags.
pub(super) type LayerScans = (Vec<HashMap<String, LayerMember>>, Vec<bool>);

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

pub(super) fn scan_archive_layer(
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

pub(super) fn scan_oci_archive_layers(
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
