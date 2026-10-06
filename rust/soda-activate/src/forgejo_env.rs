use std::fs;
use std::os::unix::fs::PermissionsExt;
use std::path::Path;

use crate::origin::origin_host_port;
use crate::system::{runtime, ActivateError};

pub(crate) fn rewrite_forgejo_env(
    path: &Path,
    forgejo_url: &str,
    address: &str,
) -> Result<(), ActivateError> {
    let raw = fs::read_to_string(path).map_err(|e| runtime(e.to_string()))?;
    let mut values: Vec<(String, String)> = Vec::new();
    for line in raw.split('\n') {
        let line = line.strip_suffix('\r').unwrap_or(line);
        if line.is_empty() || line.starts_with('#') || !line.contains('=') {
            continue;
        }
        let (key, value) = line.split_once('=').unwrap_or((line, ""));
        match values.iter_mut().find(|(k, _)| k == key) {
            Some(slot) => slot.1 = value.to_string(),
            None => values.push((key.to_string(), value.to_string())),
        }
    }
    let trimmed = forgejo_url.trim_end_matches('/');
    let domain = origin_host_port(forgejo_url)
        .map(|o| o.hostname)
        .unwrap_or_default();
    let set = |values: &mut Vec<(String, String)>, key: &str, value: String| match values
        .iter_mut()
        .find(|(k, _)| k == key)
    {
        Some(slot) => slot.1 = value,
        None => values.push((key.to_string(), value)),
    };
    set(
        &mut values,
        "FORGEJO__server__ROOT_URL",
        format!("{trimmed}/"),
    );
    set(&mut values, "FORGEJO__server__DOMAIN", domain);
    set(&mut values, "FORGEJO__server__SSH_PORT", "2222".to_string());
    set(
        &mut values,
        "FORGEJO__security__INSTALL_LOCK",
        "true".to_string(),
    );
    // Forgejo appends its email hash and size query to this supported provider URL.
    // Dynamic provider/federation settings remain native Forgejo administrator state.
    set(
        &mut values,
        "FORGEJO__picture__GRAVATAR_SOURCE",
        format!("{trimmed}/-/soda/avatars/v1/"),
    );
    set(
        &mut values,
        "FORGEJO__extensions__ENABLED",
        "true".to_string(),
    );
    set(
        &mut values,
        "FORGEJO__extensions__PATH",
        "/data/gitea/extensions".to_string(),
    );
    set(
        &mut values,
        "FORGEJO__extensions__REQUIRED_IDS",
        "soda".to_string(),
    );
    set(
        &mut values,
        "FORGEJO__extensions__SERVICE_CALLBACK_PATH",
        "/ipc/host.sock".to_string(),
    );
    // Service-bridge callers authenticate by Unix peer UID. The dashboard is
    // the only bridge client; it runs as the fixed soda identity (2000).
    set(
        &mut values,
        "FORGEJO__extensions__SERVICE_BRIDGE_PEERS",
        "2000:soda".to_string(),
    );
    if !values
        .iter()
        .any(|(k, _)| k == "FORGEJO__server__SSH_DOMAIN")
    {
        values.push((
            "FORGEJO__server__SSH_DOMAIN".to_string(),
            address.to_string(),
        ));
    }
    let mut out = String::new();
    for (key, value) in &values {
        out.push_str(key);
        out.push('=');
        out.push_str(value);
        out.push('\n');
    }
    fs::write(path, out).map_err(|e| runtime(e.to_string()))?;
    fs::set_permissions(path, fs::Permissions::from_mode(0o600))
        .map_err(|e| runtime(e.to_string()))?;
    Ok(())
}
