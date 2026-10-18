//! Private/public merge and render orchestration.

use std::path::Path;

use super::document::{clear_files, file_entry, public_config, push_file};
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

/// Reject non-key/non-hash secret inputs before they render anywhere.
fn check_secret_inputs(key: &str, password: &str) -> Result<(), ProvError> {
    if !(key.starts_with("ssh-") || key.starts_with("ecdsa-") || key.starts_with("sk-"))
        || key.contains('\n')
        || !password.starts_with('$')
        || password.contains('\n')
    {
        return Err(ProvError::value(
            "provide a public SSH key and crypt(3) hash, not plaintext",
        ));
    }
    Ok(())
}

/// Reject a hostname pair that names both fixture and appliance names.
fn check_hostname_pair(inputs: &RenderInputs<'_>) -> Result<(), ProvError> {
    if let Some(product) = inputs.product_hostname {
        if inputs.hostname.is_some() || !is_appliance_hostname(product) {
            return Err(ProvError::value(
                "valid appliance hostname or fixture hostname required, not both",
            ));
        }
    }
    Ok(())
}

/// Apply the bootstrap profile: minimal drops systemd and files.
fn apply_bootstrap(config: &mut serde_json::Value, bootstrap: &str) -> Result<(), ProvError> {
    if bootstrap == "minimal" {
        let entries = config
            .as_object_mut()
            .ok_or_else(|| ProvError::new(ProvKind::Type, "bootstrap must be an object"))?;
        if entries.remove("systemd").is_none() {
            return Err(ProvError::new(ProvKind::Key, "bootstrap has no systemd"));
        }
        clear_files(config)?;
    } else if bootstrap != "extensions" {
        return Err(ProvError::value("unknown bootstrap profile"));
    }
    Ok(())
}

/// Append the per-instance Ed25519 host key pair when provided.
fn apply_host_key(
    config: &mut serde_json::Value,
    host_key: Option<&Path>,
) -> Result<(), ProvError> {
    let Some(host_key) = host_key else {
        return Ok(());
    };
    let private_key = regular(host_key, true)?;
    let public = derive_host_public(host_key)?;
    push_file(
        config,
        file_entry("/etc/ssh/ssh_host_ed25519_key", 0o600, &private_key),
    )?;
    push_file(
        config,
        file_entry("/etc/ssh/ssh_host_ed25519_key.pub", 0o644, &public),
    )?;
    Ok(())
}

/// Merge the public bootstrap with the private inputs and write the
/// exclusive output document.
pub fn render(source: &Path, inputs: &RenderInputs<'_>) -> Result<(), ProvError> {
    let key = regular(inputs.operator_key, false)?;
    let password = regular(inputs.password_hash, true)?;
    let key = key.trim();
    let password = password.trim();
    check_secret_inputs(key, password)?;
    let mut config = public_config(source)?;
    check_hostname_pair(inputs)?;
    apply_bootstrap(&mut config, inputs.bootstrap)?;
    let entries = config
        .as_object_mut()
        .ok_or_else(|| ProvError::new(ProvKind::Type, "bootstrap must be an object"))?;
    entries.insert(
        "passwd".to_string(),
        serde_json::json!({
            "users": [{
                "name": "root",
                "ssh_authorized_keys": [key],
                "password_hash": password,
            }]
        }),
    );
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
    apply_host_key(&mut config, inputs.host_key)?;
    let output =
        serde_json::to_string_pretty(&config).expect("serializing a JSON bootstrap is infallible");
    write_exclusive(inputs.out, &output)
}
