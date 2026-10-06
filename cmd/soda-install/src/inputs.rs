//! Provisioning input validation and Ignition destination assembly:
//! `Hostname`, `ProjectSubnet`, and `Destination`.

use soda_json::JsonValue;

use crate::errors::Error;
use crate::jsongo::{self, Soft};
use crate::netip;
use crate::sshkey;

/// Hostname labels: 1-63 lowercase alphanumerics/dashes, starting and
/// ending alphanumeric; the full name is 1-253 bytes.
pub fn hostname(value: &str) -> bool {
    if value.is_empty() || value.len() > 253 {
        return false;
    }
    value.split('.').all(|label| {
        let bytes = label.as_bytes();
        if bytes.is_empty() || bytes.len() > 63 {
            return false;
        }
        if !bytes[0].is_ascii_alphanumeric() || !bytes[bytes.len() - 1].is_ascii_alphanumeric() {
            return false;
        }
        bytes
            .iter()
            .all(|b| b.is_ascii_lowercase() || b.is_ascii_digit() || *b == b'-')
    })
}

/// Canonical IPv4 project subnet (`address/masked-bits`, no zones).
pub fn project_subnet(value: &str) -> Result<(), Error> {
    match netip::parse_prefix(value) {
        Ok(prefix) if prefix.addr().is4() && prefix == prefix.masked() => Ok(()),
        _ => Err(Error::msg("canonical IPv4 project subnet required")),
    }
}

fn valid_password_hash(value: &str) -> bool {
    // `^\$6\$[./a-zA-Z0-9]{1,16}\$[./a-zA-Z0-9]{86}$`
    let rest = match value.strip_prefix("$6$") {
        Some(rest) => rest,
        None => return false,
    };
    let (salt, hash) = match rest.split_once('$') {
        Some(pair) => pair,
        None => return false,
    };
    if salt.is_empty() || salt.len() > 16 || hash.len() != 86 {
        return false;
    }
    // Any further `$` fails the charset below.
    salt.bytes()
        .chain(hash.bytes())
        .all(|b| matches!(b, b'.' | b'/' | b'0'..=b'9' | b'a'..=b'z' | b'A'..=b'Z'))
}

fn valid_provisioning_inputs(hostname_value: &str, password_hash: &str, subnet: &str) -> bool {
    hostname(hostname_value) && project_subnet(subnet).is_ok() && valid_password_hash(password_hash)
}

fn template_storage(template: &[u8]) -> Result<(JsonValue, JsonValue, Vec<JsonValue>), Error> {
    let config =
        jsongo::parse(template).map_err(|_| Error::msg("invalid public destination template"))?;
    // Go decodes `null` into a nil map without error; the missing ignition
    // below then reports the Ignition error, not a template error.
    let empty: Vec<(String, JsonValue)> = Vec::new();
    let entries = match &config {
        JsonValue::Object(entries) => entries,
        JsonValue::Null => &empty,
        _ => return Err(Error::msg("invalid public destination template")),
    };
    let ignition = entries
        .iter()
        .rev()
        .find(|(k, _)| k == "ignition")
        .map(|(_, v)| v);
    let version = match ignition {
        Some(value) => Soft::new(value)
            .map_err(|_| Error::msg("expected converted Ignition 3.5.0 template"))?
            .string("version")
            .map_err(|_| Error::msg("expected converted Ignition 3.5.0 template"))?,
        None => None,
    };
    if version.as_deref() != Some("3.5.0") {
        return Err(Error::msg("expected converted Ignition 3.5.0 template"));
    }
    if entries.iter().any(|(k, _)| k == "passwd") {
        return Err(Error::msg("public template must not contain accounts"));
    }
    let storage_value = entries
        .iter()
        .rev()
        .find(|(k, _)| k == "storage")
        .map(|(_, v)| v);
    let storage = match storage_value {
        Some(JsonValue::Object(_)) => storage_value.unwrap().clone(),
        _ => return Err(Error::msg("invalid public storage template")),
    };
    let files_value = match &storage {
        JsonValue::Object(entries) => entries
            .iter()
            .rev()
            .find(|(k, _)| k == "files")
            .map(|(_, v)| v),
        _ => None,
    };
    let files = match files_value {
        None => return Err(Error::msg("invalid public files template")),
        Some(JsonValue::Null) => Vec::new(),
        Some(JsonValue::Array(items)) => items.clone(),
        Some(_) => return Err(Error::msg("invalid public files template")),
    };
    Ok((config, storage, files))
}

fn admit_provisioning_files(files: &[JsonValue]) -> Result<(), Error> {
    for file in files {
        // Go decodes `null` into a zero entry without error; it carries no
        // path and passes through untouched.
        if matches!(file, JsonValue::Null) {
            continue;
        }
        let path = Soft::new(file)
            .map_err(|_| Error::msg("provisioning path collision"))?
            .string("path")
            .map_err(|_| Error::msg("provisioning path collision"))?
            .unwrap_or_default();
        if path == "/etc/hostname" || path == "/etc/soda-installer/project-subnet" {
            return Err(Error::msg("provisioning path collision"));
        }
    }
    Ok(())
}

fn file_entry(path: &str, contents: &str) -> JsonValue {
    JsonValue::Object(vec![
        ("path".to_string(), JsonValue::Str(path.to_string())),
        ("mode".to_string(), JsonValue::Number("384".to_string())),
        (
            "contents".to_string(),
            JsonValue::Object(vec![(
                "source".to_string(),
                JsonValue::Str(format!(
                    "data:;base64,{}",
                    crate::sshkey::b64_encode(contents.as_bytes())
                )),
            )]),
        ),
    ])
}

fn set_field(object: &mut JsonValue, key: &str, value: JsonValue) {
    if let JsonValue::Object(entries) = object {
        entries.retain(|(k, _)| k != key);
        entries.push((key.to_string(), value));
    }
}

/// `Destination`: extend the public template with the hostname file, the
/// project-subnet file, and the root account.
pub fn destination(
    template: &[u8],
    hostname_value: &str,
    key: &str,
    password_hash: &str,
    subnet: &str,
) -> Result<Vec<u8>, Error> {
    let mut normalized = String::new();
    if !key.is_empty() {
        normalized = sshkey::public_key(key).map_err(Error::msg)?;
    }
    if !valid_provisioning_inputs(hostname_value, password_hash, subnet) {
        return Err(Error::msg("invalid private provisioning inputs"));
    }
    let (mut config, mut storage, mut files) = template_storage(template)?;
    admit_provisioning_files(&files)?;
    // Go iterates a two-entry map (random order); file order carries no
    // meaning to Ignition, so emit the deterministic sorted order.
    files.push(file_entry("/etc/hostname", &format!("{hostname_value}\n")));
    files.push(file_entry(
        "/etc/soda-installer/project-subnet",
        &format!("{subnet}\n"),
    ));
    set_field(&mut storage, "files", JsonValue::Array(files));
    set_field(&mut config, "storage", storage);
    let mut root = vec![
        ("name".to_string(), JsonValue::Str("root".to_string())),
        (
            "passwordHash".to_string(),
            JsonValue::Str(password_hash.to_string()),
        ),
    ];
    if !normalized.is_empty() {
        root.push((
            "sshAuthorizedKeys".to_string(),
            JsonValue::Array(vec![JsonValue::Str(normalized)]),
        ));
    }
    set_field(
        &mut config,
        "passwd",
        JsonValue::Object(vec![(
            "users".to_string(),
            JsonValue::Array(vec![JsonValue::Object(root)]),
        )]),
    );
    Ok(jsongo::serialize(&config).into_bytes())
}

#[cfg(test)]
mod tests;
