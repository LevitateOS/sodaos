//! Live-candidate authentication and destination rendering: the media
//! payload, console, and shared images are bound to the authenticated
//! release before any disk write, and the confirmed private inputs are
//! rendered into destination Ignition.

use crate::buildx;
use crate::deliver;
use crate::errors::Error;
use crate::inputs;
use crate::run::MediaIdentity;
use crate::wizard::DiskInstallChoices;

pub const CANDIDATE_INSTALLER_BINARY: &str = "/usr/libexec/soda/soda-install";

pub fn candidate_identity_matches(media: &MediaIdentity) -> Result<(), Error> {
    if media
        .validate(&media.release.clone(), &crate::run::architecture())
        .is_err()
    {
        return Err(Error::msg("invalid candidate media identity"));
    }
    Ok(())
}

fn candidate_file_matches(path: &str, want: &str, mismatch: &str) -> Result<(), Error> {
    match buildx::hash_file(path) {
        Ok(hash) if hash == want => Ok(()),
        _ => Err(Error::msg(mismatch)),
    }
}

fn candidate_release_matches(
    media: &MediaIdentity,
    payload_path: &str,
) -> Result<deliver::Payload, Error> {
    match deliver::load(payload_path) {
        Ok(payload)
            if payload.revision == media.revision
                && payload.architecture == media.architecture
                && payload.core_os == media.release =>
        {
            Ok(payload)
        }
        _ => Err(Error::msg("live candidate release mismatch")),
    }
}

pub fn candidate_requirement(media: &MediaIdentity, root: &str) -> Result<u64, Error> {
    candidate_identity_matches(media)?;
    candidate_file_matches(
        &format!("{root}{CANDIDATE_INSTALLER_BINARY}"),
        &media.console_sha256,
        "candidate installer differs from prebuilt tool",
    )?;
    let payload = format!("{root}{}", deliver::PATH);
    candidate_file_matches(
        &payload,
        &media.payload_sha256,
        "live candidate payload differs from authenticated media",
    )?;
    let loaded = candidate_release_matches(media, &payload)?;
    let (_, total) = deliver::verify_content(&loaded, &format!("{root}{}", deliver::IMAGES_PATH))?;
    Ok(total)
}

// Declaration order preserves the sorted output of the existing producer.
// Derive rejects exact duplicate keys before a map can collapse them.
#[derive(serde::Deserialize, serde::Serialize)]
#[serde(deny_unknown_fields)]
struct MachineDefaults {
    bridge: String,
    network: String,
    subnet: String,
    tailnet_management: bool,
}

fn rewrite_candidate_host_file(
    files: &mut Vec<serde_json::Value>,
    machine: &[u8],
) -> Result<(), Error> {
    for file in files {
        if file.is_null() {
            continue;
        }
        let entries = file
            .as_object_mut()
            .ok_or_else(|| Error::msg("invalid destination files"))?;
        let path = entries.get("path").and_then(serde_json::Value::as_str);
        if path == Some("/etc/soda/host.json") {
            return Err(Error::msg("machine configuration collision"));
        }
        if path == Some("/etc/soda-installer/project-subnet") {
            entries.insert(
                "path".to_string(),
                serde_json::Value::String("/etc/soda/host.json".to_string()),
            );
            entries.insert("contents".to_string(), serde_json::json!({"source":format!("data:;base64,{}", crate::sshkey::b64_encode(machine))}));
        }
    }
    Ok(())
}

pub fn candidate_destination(
    template: &[u8],
    factory: &[u8],
    choices: &DiskInstallChoices,
) -> Result<Vec<u8>, Error> {
    let factory_text = String::from_utf8_lossy(factory);
    let mut defaults: MachineDefaults = serde_json::from_str(&factory_text)
        .map_err(|_| Error::msg("invalid native machine defaults"))?;
    if !defaults.subnet.is_empty() || defaults.tailnet_management {
        return Err(Error::msg("invalid native machine defaults"));
    }
    if defaults.bridge.is_empty() || defaults.network.is_empty() {
        return Err(Error::msg("missing native machine network defaults"));
    }
    defaults.subnet.clone_from(&choices.subnet);
    let machine = inputs::serialize_ignition(&defaults)
        .map_err(|_| Error::msg("invalid native machine defaults"))?;
    let destination = inputs::destination(
        template,
        &choices.hostname,
        "",
        &choices.password_hash,
        &choices.subnet,
    )?;
    let destination_text = String::from_utf8_lossy(&destination);
    let config: serde_json::Value =
        serde_json::from_str(&destination_text).map_err(|_| Error::msg("invalid destination"))?;
    let mut config_entries = config
        .as_object()
        .cloned()
        .ok_or_else(|| Error::msg("invalid destination"))?;
    let mut storage_entries = config_entries
        .get("storage")
        .and_then(serde_json::Value::as_object)
        .cloned()
        .ok_or_else(|| Error::msg("invalid destination"))?;
    let mut files = match storage_entries.get("files") {
        Some(serde_json::Value::Array(files)) => files.clone(),
        None | Some(serde_json::Value::Null) => Vec::new(),
        _ => return Err(Error::msg("invalid destination")),
    };
    rewrite_candidate_host_file(&mut files, &machine)?;
    storage_entries.insert("files".to_string(), serde_json::Value::Array(files));
    config_entries.insert(
        "storage".to_string(),
        serde_json::Value::Object(storage_entries),
    );
    inputs::serialize_ignition(&serde_json::Value::Object(config_entries))
}

#[cfg(test)]
mod tests;
