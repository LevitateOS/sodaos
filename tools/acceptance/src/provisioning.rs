//! Provisioning secrets, mirroring `provisioning.go`.
//!
//! One explicit complete Ignition v3 document is used, never a guessed
//! merge with another disk config. Password hashes and private host-key
//! material join the evidence secret set so later capture cannot retain
//! them.

use soda_json::JsonValue;

use crate::error::Error;
use crate::files;
use crate::trust;

/// Collect private bootstrap values before serial capture. Mirrors
/// `ProvisioningSecrets`, including the single-document gate.
pub fn provisioning_secrets(path: &str) -> Result<Vec<Vec<u8>>, Error> {
    let data = files::private_file(path)?;
    let parsed = trust::parse_ignition(&data).map_err(|_| Error::msg(single_document_message()))?;
    if !is_single_complete_v3(&parsed) {
        return Err(Error::msg(single_document_message()));
    }
    let mut secrets = Vec::new();
    if let Some(JsonValue::Array(users)) = parsed.get("passwd").and_then(|p| p.get("users")) {
        for user in users {
            if let Some(hash) = user.get("PasswordHash").and_then(|v| v.as_str()) {
                if !hash.is_empty() {
                    secrets.push(hash.as_bytes().to_vec());
                }
            }
        }
    }
    let files =
        trust::decode_ignition_files(&parsed).map_err(|_| Error::msg(single_document_message()))?;
    for file in &files {
        if !is_private_host_key_path(&file.path) {
            continue;
        }
        let decoded = trust::inline_data(&file.source, &file.compression)?;
        secrets.push(file.source.as_bytes().to_vec());
        secrets.push(decoded.clone());
        for line in String::from_utf8_lossy(&decoded).split('\n') {
            if !line.is_empty() {
                secrets.push(line.as_bytes().to_vec());
            }
        }
    }
    Ok(secrets)
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
}
