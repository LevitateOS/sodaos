//! Provisioning secrets, mirroring `provisioning.go`.
//!
//! One explicit complete Ignition v3 document is used, never a guessed
//! merge with another disk config. Password hashes and private host-key
//! material join the evidence secret set so later capture cannot retain
//! them.

use crate::structured::Value as JsonValue;

use crate::error::Error;
use crate::files;
use crate::trust;

/// Aggregate ceiling for values retained for evidence redaction. Charge every
/// candidate, including duplicates, before copying or deriving it.
pub(crate) const SECRET_COLLECTION_BYTES: usize = 16 << 20;
pub(crate) const SECRET_COLLECTION_COUNT: usize = 16_384;

#[derive(Default)]
pub(crate) struct SecretCollector {
    values: Vec<Vec<u8>>,
    bytes: usize,
}

impl SecretCollector {
    pub(crate) fn push(&mut self, value: &[u8]) -> Result<(), Error> {
        if value.is_empty() {
            return Ok(());
        }
        self.check_candidates(1, value.len())?;
        self.bytes += value.len();
        self.values.push(value.to_vec());
        Ok(())
    }

    pub(crate) fn check_candidates(&self, count: usize, bytes: usize) -> Result<(), Error> {
        if self
            .values
            .len()
            .checked_add(count)
            .is_none_or(|n| n > SECRET_COLLECTION_COUNT)
            || self
                .bytes
                .checked_add(bytes)
                .is_none_or(|n| n > SECRET_COLLECTION_BYTES)
        {
            return Err(Error::msg("evidence redaction pattern limit exceeded"));
        }
        Ok(())
    }

    pub(crate) fn remaining_bytes(&self) -> usize {
        SECRET_COLLECTION_BYTES - self.bytes
    }

    pub(crate) fn push_owned(&mut self, value: Vec<u8>) -> Result<(), Error> {
        if value.is_empty() {
            return Ok(());
        }
        self.check_candidates(1, value.len())?;
        self.bytes += value.len();
        self.values.push(value);
        Ok(())
    }

    pub(crate) fn into_values(self) -> Vec<Vec<u8>> {
        self.values
    }
}

/// Collect private bootstrap values before serial capture. Mirrors
/// `ProvisioningSecrets`, including the single-document gate.
pub fn provisioning_secrets(path: &str) -> Result<Vec<Vec<u8>>, Error> {
    let mut collector = SecretCollector::default();
    collect_provisioning_secrets(path, &mut collector)?;
    Ok(collector.into_values())
}

pub(crate) fn collect_provisioning_secrets(
    path: &str,
    secrets: &mut SecretCollector,
) -> Result<(), Error> {
    let data = files::private_file(path)?;
    let parsed = trust::parse_ignition(&data).map_err(|_| Error::msg(single_document_message()))?;
    if !is_single_complete_v3(&parsed) {
        return Err(Error::msg(single_document_message()));
    }
    if let Some(JsonValue::Array(users)) = parsed.get("passwd").and_then(|p| p.get("users")) {
        for user in users {
            if let Some(hash) = user.get("passwordHash").and_then(|v| v.as_str()) {
                if !hash.is_empty() {
                    secrets.push(hash.as_bytes())?;
                }
            }
        }
    }
    let files = trust::decode_ignition_file_refs(&parsed)
        .map_err(|_| Error::msg(single_document_message()))?;
    for file in &files {
        if !is_private_host_key_path(&file.path) {
            continue;
        }
        secrets.push(file.source.as_bytes())?;
        // The decoder can use only the remaining aggregate budget, capped by
        // its existing per-file output bound. Its bounded staging buffer may
        // hold one extra byte to distinguish exact-limit input from overflow.
        secrets.check_candidates(1, 0)?;
        let decode_limit = secrets
            .remaining_bytes()
            .min(trust::INLINE_GZIP_LIMIT as usize);
        let decoded = trust::inline_data_limited(file.source, file.compression, decode_limit)?;
        let lossy_bytes = lossy_utf8_len(&decoded);
        let newline_count = decoded.iter().filter(|byte| **byte == b'\n').count();
        let line_bytes = lossy_bytes - newline_count;
        let line_count = decoded
            .split(|byte| *byte == b'\n')
            .filter(|line| !line.is_empty())
            .count();
        secrets.check_candidates(
            usize::from(!decoded.is_empty()) + line_count,
            decoded.len() + line_bytes,
        )?;
        let lossy = String::from_utf8_lossy(&decoded).into_owned();
        secrets.push_owned(decoded)?;
        for line in lossy.split('\n') {
            secrets.push(line.as_bytes())?;
        }
    }
    Ok(())
}

fn lossy_utf8_len(bytes: &[u8]) -> usize {
    let mut remaining = bytes;
    let mut length = 0;
    while !remaining.is_empty() {
        match std::str::from_utf8(remaining) {
            Ok(valid) => {
                length += valid.len();
                break;
            }
            Err(error) => {
                let valid = error.valid_up_to();
                length += valid + 3;
                let invalid = error.error_len().unwrap_or(remaining.len() - valid);
                remaining = &remaining[valid + invalid..];
            }
        }
    }
    length
}

fn single_document_message() -> &'static str {
    "single complete Ignition v3 input required; external merge/replace is not supported"
}

fn is_single_complete_v3(parsed: &JsonValue) -> bool {
    let version = parsed
        .get("ignition")
        .and_then(|i| i.get("version"))
        .and_then(|v| v.as_str())
        .unwrap_or("");
    if !version.starts_with("3.") {
        return false;
    }
    if let Some(merge) = parsed
        .get("ignition")
        .and_then(|i| i.get("config"))
        .and_then(|c| c.get("merge"))
    {
        match merge {
            JsonValue::Null => {}
            JsonValue::Array(items) if items.is_empty() => {}
            _ => return false,
        }
    }
    if let Some(replace) = parsed
        .get("ignition")
        .and_then(|i| i.get("config"))
        .and_then(|c| c.get("replace"))
    {
        if !matches!(replace, JsonValue::Null) {
            return false;
        }
    }
    true
}

fn is_private_host_key_path(path: &str) -> bool {
    path.starts_with("/etc/ssh/ssh_host_") && !path.ends_with(".pub")
}
