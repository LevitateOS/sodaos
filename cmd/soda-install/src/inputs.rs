//! Provisioning input validation and Ignition destination assembly:
//! `Hostname`, `ProjectSubnet`, and `Destination`.

use crate::errors::Error;
use crate::netip;
use crate::sshkey;
use serde_json::ser::Formatter;
use serde_json::{Map, Value};
use std::io::{self, Write};

const RESERVED_PROJECT_NETWORKS: [&str; 2] = ["10.88.0.0/16", "10.90.0.0/24"];

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
        Ok(prefix) if prefix.addr().is4() && prefix == prefix.masked() => {
            let subnet = prefix.masked();
            let overlaps_reserved = RESERVED_PROJECT_NETWORKS.iter().any(|reserved| {
                let reserved = netip::parse_prefix(reserved)
                    .expect("reserved project network must be a valid prefix")
                    .masked();
                reserved.contains(&subnet.addr()) || subnet.contains(&reserved.addr())
            });
            if overlaps_reserved {
                return Err(Error::msg("project subnet overlaps a reserved network"));
            }
            Ok(())
        }
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

fn template_storage(template: &[u8]) -> Result<(Value, Value, Vec<Value>), Error> {
    let text = String::from_utf8_lossy(template);
    let config: Value = serde_json::from_str(&text)
        .map_err(|_| Error::msg("invalid public destination template"))?;
    // Go decodes `null` into a nil map without error; the missing ignition
    // below then reports the Ignition error, not a template error.
    let empty = Map::new();
    let entries = match &config {
        Value::Object(entries) => entries,
        Value::Null => &empty,
        _ => return Err(Error::msg("invalid public destination template")),
    };
    let ignition = entries.get("ignition");
    let version = ignition
        .and_then(Value::as_object)
        .and_then(|o| o.get("version"));
    if version.and_then(Value::as_str) != Some("3.5.0")
        || matches!(ignition, Some(v) if !v.is_object())
    {
        return Err(Error::msg("expected converted Ignition 3.5.0 template"));
    }
    if entries.contains_key("passwd") {
        return Err(Error::msg("public template must not contain accounts"));
    }
    let storage_value = entries.get("storage");
    let storage = match storage_value {
        Some(Value::Object(_)) => storage_value.unwrap().clone(),
        _ => return Err(Error::msg("invalid public storage template")),
    };
    let files_value = storage.get("files");
    let files = match files_value {
        None => return Err(Error::msg("invalid public files template")),
        Some(Value::Null) => Vec::new(),
        Some(Value::Array(items)) => items.clone(),
        Some(_) => return Err(Error::msg("invalid public files template")),
    };
    Ok((config, storage, files))
}

fn admit_provisioning_files(files: &[Value]) -> Result<(), Error> {
    for file in files {
        // Go decodes `null` into a zero entry without error; it carries no
        // path and passes through untouched.
        if matches!(file, Value::Null) {
            continue;
        }
        let object = file
            .as_object()
            .ok_or_else(|| Error::msg("provisioning path collision"))?;
        let path = match object.get("path") {
            None | Some(Value::Null) => String::new(),
            Some(Value::String(path)) => path.clone(),
            _ => return Err(Error::msg("provisioning path collision")),
        };
        if path == "/etc/hostname" || path == "/etc/soda-installer/project-subnet" {
            return Err(Error::msg("provisioning path collision"));
        }
    }
    Ok(())
}

fn file_entry(path: &str, contents: &str) -> Value {
    serde_json::json!({"path":path,"mode":384,"contents":{"source":format!("data:;base64,{}", crate::sshkey::b64_encode(contents.as_bytes()))}})
}

fn set_field(object: &mut Value, key: &str, value: Value) {
    if let Value::Object(entries) = object {
        entries.insert(key.to_string(), value);
    }
}

struct GoHtmlFormatter;
impl Formatter for GoHtmlFormatter {
    fn write_string_fragment<W>(&mut self, writer: &mut W, fragment: &str) -> io::Result<()>
    where
        W: ?Sized + Write,
    {
        let mut start = 0;
        for (index, ch) in fragment.char_indices() {
            let escape: Option<&[u8]> = match ch {
                '<' => Some(b"\\u003c"),
                '>' => Some(b"\\u003e"),
                '&' => Some(b"\\u0026"),
                '\u{2028}' => Some(b"\\u2028"),
                '\u{2029}' => Some(b"\\u2029"),
                _ => None,
            };
            if let Some(escape) = escape {
                writer.write_all(&fragment.as_bytes()[start..index])?;
                writer.write_all(escape)?;
                start = index + ch.len_utf8();
            }
        }
        writer.write_all(&fragment.as_bytes()[start..])
    }
}

pub(crate) fn serialize_ignition<T: serde::Serialize + ?Sized>(
    value: &T,
) -> Result<Vec<u8>, Error> {
    let mut out = Vec::new();
    let mut serializer = serde_json::Serializer::with_formatter(&mut out, GoHtmlFormatter);
    value
        .serialize(&mut serializer)
        .map_err(|_| Error::msg("cannot encode destination"))?;
    Ok(out)
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
    set_field(&mut storage, "files", Value::Array(files));
    set_field(&mut config, "storage", storage);
    let mut root = serde_json::Map::new();
    root.insert("name".to_string(), Value::String("root".to_string()));
    root.insert(
        "passwordHash".to_string(),
        Value::String(password_hash.to_string()),
    );
    if !normalized.is_empty() {
        root.insert(
            "sshAuthorizedKeys".to_string(),
            Value::Array(vec![Value::String(normalized)]),
        );
    }
    set_field(
        &mut config,
        "passwd",
        serde_json::json!({"users":[Value::Object(root)]}),
    );
    serialize_ignition(&config)
}

#[cfg(test)]
mod tests;
