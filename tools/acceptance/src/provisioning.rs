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
        if self.values.len().checked_add(count).is_none_or(|n| n > SECRET_COLLECTION_COUNT)
            || self.bytes.checked_add(bytes).is_none_or(|n| n > SECRET_COLLECTION_BYTES)
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
            if let Some(hash) = user.get("PasswordHash").and_then(|v| v.as_str()) {
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

#[cfg(test)]
mod tests {
    use super::*;

    fn write_input(dir: &std::path::Path, name: &str, body: &[u8]) -> String {
        let path = dir.join(name);
        std::fs::write(&path, body).unwrap();
        use std::os::unix::fs::PermissionsExt;
        std::fs::set_permissions(&path, std::fs::Permissions::from_mode(0o600)).unwrap();
        path.to_string_lossy().into_owned()
    }

    #[test]
    fn secrets_cover_hashes_and_host_key_material() {
        let dir = std::env::temp_dir().join(format!("soda-prov-{}", std::process::id()));
        let _ = std::fs::remove_dir_all(&dir);
        std::fs::create_dir_all(&dir).unwrap();
        let key = "line-one\nline-two\n";
        let source = format!(
            "data:;base64,{}",
            crate::trust::encode_base64(key.as_bytes())
        );
        let body = format!(
            "{{\"ignition\":{{\"version\":\"3.5.0\"}},\"passwd\":{{\"users\":[{{\"PasswordHash\":\"$6$salt$hash\"}},{{}}]}},\"storage\":{{\"files\":[{{\"path\":\"/etc/motd\",\"contents\":{{\"source\":\"data:,hi\"}}}},{{\"path\":\"/etc/ssh/ssh_host_ed25519_key\",\"contents\":{{\"source\":\"{source}\"}}}}]}}}}"
        );
        let input = write_input(&dir, "input.ign", body.as_bytes());
        let secrets = provisioning_secrets(&input).unwrap();
        assert!(secrets.iter().any(|s| s == b"$6$salt$hash"));
        assert!(secrets.iter().any(|s| s == source.as_bytes()));
        assert!(secrets.iter().any(|s| s == b"line-one\nline-two\n"));
        assert!(secrets.iter().any(|s| s == b"line-one"));
        assert!(secrets.iter().any(|s| s == b"line-two"));
        assert!(!secrets.iter().any(|s| s == b"hi"));
        std::fs::remove_dir_all(&dir).unwrap();
    }

    #[test]
    fn single_document_gate() {
        let dir = std::env::temp_dir().join(format!("soda-prov-gate-{}", std::process::id()));
        let _ = std::fs::remove_dir_all(&dir);
        std::fs::create_dir_all(&dir).unwrap();
        for (name, body) in [
            ("bad-version", "{\"ignition\":{\"version\":\"2.0.0\"}}"),
            (
                "merge",
                "{\"ignition\":{\"version\":\"3.5.0\",\"config\":{\"merge\":[{}]}}}",
            ),
            (
                "replace",
                "{\"ignition\":{\"version\":\"3.5.0\",\"config\":{\"replace\":{}}}}",
            ),
            ("garbage", "not json"),
        ] {
            let input = write_input(&dir, name, body.as_bytes());
            assert_eq!(
                provisioning_secrets(&input).unwrap_err().to_string(),
                single_document_message(),
                "{name}"
            );
        }
        std::fs::remove_dir_all(&dir).unwrap();
    }

    #[test]
    fn aggregate_budget_covers_many_derived_lines_and_exact_boundary() {
        let mut exact = SecretCollector::default();
        let pattern = b"x";
        for _ in 0..SECRET_COLLECTION_COUNT {
            exact.push(pattern).unwrap();
        }
        assert_eq!(exact.values.len(), SECRET_COLLECTION_COUNT);
        assert!(exact.push(pattern).is_err());

        let mut bytes = SecretCollector::default();
        bytes.push(&vec![b'x'; SECRET_COLLECTION_BYTES]).unwrap();
        assert!(bytes.push(pattern).is_err());

        let dir = std::path::Path::new(env!("CARGO_MANIFEST_DIR"))
            .join("../../.artifacts")
            .join(format!("soda-prov-lines-{}", std::process::id()));
        let _ = std::fs::remove_dir_all(&dir);
        std::fs::create_dir_all(&dir).unwrap();
        let key: Vec<u8> = (0..SECRET_COLLECTION_COUNT)
            .flat_map(|_| [b'x', b'\n'])
            .collect();
        let source = format!("data:;base64,{}", crate::trust::encode_base64(&key));
        let body = format!(
            "{{\"ignition\":{{\"version\":\"3.5.0\"}},\"storage\":{{\"files\":[{{\"path\":\"/etc/ssh/ssh_host_key\",\"contents\":{{\"source\":\"{source}\"}}}}]}}}}"
        );
        let input = write_input(&dir, "many-lines.ign", body.as_bytes());
        assert!(provisioning_secrets(&input).is_err());

        let mut combined = SecretCollector::default();
        combined
            .push(&vec![b'x'; SECRET_COLLECTION_BYTES - 1])
            .unwrap();
        assert!(collect_provisioning_secrets(&input, &mut combined).is_err());

        let small_source = format!("data:;base64,{}", crate::trust::encode_base64(b"small-key"));
        let small_body = format!(
            "{{\"ignition\":{{\"version\":\"3.5.0\"}},\"storage\":{{\"files\":[{{\"path\":\"/etc/ssh/ssh_host_key\",\"contents\":{{\"source\":\"{small_source}\"}}}}]}}}}"
        );
        let small_input = write_input(&dir, "small.ign", small_body.as_bytes());
        let mut near_limit = SecretCollector::default();
        near_limit
            .push(&vec![b'x'; (15 << 20) + (1 << 19)])
            .unwrap();
        collect_provisioning_secrets(&small_input, &mut near_limit).unwrap();
        std::fs::remove_dir_all(dir).unwrap();
    }
}
