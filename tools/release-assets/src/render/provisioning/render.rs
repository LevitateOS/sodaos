//! Private/public merge and render orchestration.

use std::path::Path;

use soda_json::JsonValue;

use super::document::{
    clear_files, dump_python, file_entry, object_mut, public_config, push_file, str_value,
};
use super::private_files::{
    derive_host_public, is_appliance_hostname, is_fixture_hostname, regular, write_exclusive,
};
use super::{ProvError, ProvKind};

pub struct RenderInputs<'a> {
    pub operator_key: &'a Path,
    pub password_hash: &'a Path,
    pub out: &'a Path,
    pub hostname: Option<&'a str>,
    pub host_key: Option<&'a Path>,
    pub bootstrap: &'a str,
    pub product_hostname: Option<&'a str>,
}

/// Merge the public bootstrap with the private inputs and write the
/// exclusive output document.
pub fn render(source: &Path, inputs: &RenderInputs<'_>) -> Result<(), ProvError> {
    let key = regular(inputs.operator_key, false)?;
    let password = regular(inputs.password_hash, true)?;
    let key = key.trim();
    let password = password.trim();
    if !(key.starts_with("ssh-") || key.starts_with("ecdsa-") || key.starts_with("sk-"))
        || key.contains('\n')
        || !password.starts_with('$')
        || password.contains('\n')
    {
        return Err(ProvError::value(
            "provide a public SSH key and crypt(3) hash, not plaintext",
        ));
    }
    let mut config = public_config(source)?;
    if let Some(product) = inputs.product_hostname {
        if inputs.hostname.is_some() || !is_appliance_hostname(product) {
            return Err(ProvError::value(
                "valid appliance hostname or fixture hostname required, not both",
            ));
        }
    }
    if inputs.bootstrap == "minimal" {
        let entries = object_mut(&mut config)
            .ok_or_else(|| ProvError::new(ProvKind::Type, "bootstrap must be an object"))?;
        let position = entries
            .iter()
            .rposition(|(k, _)| k == "systemd")
            .ok_or_else(|| ProvError::new(ProvKind::Key, "bootstrap has no systemd"))?;
        entries.remove(position);
        clear_files(&mut config)?;
    } else if inputs.bootstrap != "extensions" {
        return Err(ProvError::value("unknown bootstrap profile"));
    }
    let entries = object_mut(&mut config)
        .ok_or_else(|| ProvError::new(ProvKind::Type, "bootstrap must be an object"))?;
    entries.push((
        "passwd".to_string(),
        JsonValue::Object(vec![(
            "users".to_string(),
            JsonValue::Array(vec![JsonValue::Object(vec![
                ("name".to_string(), str_value("root")),
                (
                    "ssh_authorized_keys".to_string(),
                    JsonValue::Array(vec![str_value(key)]),
                ),
                ("password_hash".to_string(), str_value(password)),
            ])]),
        )]),
    ));
    if let Some(hostname) = inputs.hostname {
        if !is_fixture_hostname(hostname) {
            return Err(ProvError::value(
                "fresh soda-native-* fixture hostname required",
            ));
        }
        push_file(
            &mut config,
            file_entry("/etc/hostname", 0o644, &format!("{hostname}\n")),
        )?;
    }
    if let Some(product) = inputs.product_hostname {
        push_file(
            &mut config,
            file_entry("/etc/hostname", 0o644, &format!("{product}\n")),
        )?;
    }
    if let Some(host_key) = inputs.host_key {
        let private_key = regular(host_key, true)?;
        let public = derive_host_public(host_key)?;
        push_file(
            &mut config,
            file_entry("/etc/ssh/ssh_host_ed25519_key", 0o600, &private_key),
        )?;
        push_file(
            &mut config,
            file_entry("/etc/ssh/ssh_host_ed25519_key.pub", 0o644, &public),
        )?;
    }
    write_exclusive(inputs.out, &dump_python(&config))
}
