//! Live-candidate authentication and destination rendering: the media
//! payload, console, and shared images are bound to the authenticated
//! release before any disk write, and the confirmed private inputs are
//! rendered into destination Ignition.

use soda_json::JsonValue;

use crate::buildx;
use crate::deliver;
use crate::errors::Error;
use crate::inputs;
use crate::jsongo::{parse, serialize};
use crate::run::MediaIdentity;
use crate::wizard::DiskInstallChoices;

pub const CANDIDATE_INSTALLER_BINARY: &str = "/usr/libexec/soda/soda-install";

pub fn candidate_identity_matches(media: &MediaIdentity) -> Result<(), Error> {
    if media
        .validate(&media.release.clone(), &crate::run::architecture())
        .is_err()
        || media.installer_version != "coreos-installer 0.26.0"
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

fn unique_object_entries(entries: &[(String, JsonValue)]) -> Vec<(&str, &JsonValue)> {
    // Go maps collapse duplicates with the last value winning.
    let mut names: Vec<&str> = entries.iter().map(|(k, _)| k.as_str()).collect();
    names.sort_unstable();
    names.dedup();
    names
        .into_iter()
        .map(|name| {
            let value = entries.iter().rev().find(|(k, _)| k == name).unwrap();
            (name, &value.1)
        })
        .collect()
}

fn valid_machine_default_shape(defaults: &JsonValue) -> bool {
    let entries = match defaults {
        JsonValue::Object(entries) => entries,
        _ => return false,
    };
    let unique = unique_object_entries(entries);
    if unique.len() != 4 {
        return false;
    }
    let get = |name: &str| unique.iter().find(|(k, _)| *k == name).map(|(_, v)| *v);
    matches!(get("subnet"), Some(JsonValue::Str(s)) if s.is_empty())
        && matches!(get("tailnet_management"), Some(JsonValue::Bool(false)))
}

fn valid_machine_network_defaults(defaults: &JsonValue) -> bool {
    let entries = match defaults {
        JsonValue::Object(entries) => entries,
        _ => return false,
    };
    for key in ["network", "bridge"] {
        let value = entries.iter().rev().find(|(k, _)| k == key).map(|(_, v)| v);
        match value {
            Some(JsonValue::Str(s)) if !s.is_empty() => {}
            _ => return false,
        }
    }
    true
}

fn rewrite_candidate_host_file(files: &mut Vec<JsonValue>, machine: &[u8]) -> Result<(), Error> {
    for file in files {
        let entries = match file {
            JsonValue::Object(entries) => entries,
            JsonValue::Null => continue,
            _ => return Err(Error::msg("invalid destination files")),
        };
        let path = entries
            .iter()
            .rev()
            .find(|(k, _)| k == "path")
            .map(|(_, v)| v);
        if matches!(path, Some(JsonValue::Str(p)) if p == "/etc/soda/host.json") {
            return Err(Error::msg("machine configuration collision"));
        }
        if matches!(path, Some(JsonValue::Str(p)) if p == "/etc/soda-installer/project-subnet") {
            entries.retain(|(k, _)| k != "path" && k != "contents");
            entries.push((
                "path".to_string(),
                JsonValue::Str("/etc/soda/host.json".to_string()),
            ));
            entries.push((
                "contents".to_string(),
                JsonValue::Object(vec![(
                    "source".to_string(),
                    JsonValue::Str(format!(
                        "data:;base64,{}",
                        crate::sshkey::b64_encode(machine)
                    )),
                )]),
            ));
        }
    }
    Ok(())
}

pub fn candidate_destination(
    template: &[u8],
    factory: &[u8],
    choices: &DiskInstallChoices,
) -> Result<Vec<u8>, Error> {
    let defaults = parse(factory).map_err(|_| Error::msg("invalid native machine defaults"))?;
    if !valid_machine_default_shape(&defaults) {
        return Err(Error::msg("invalid native machine defaults"));
    }
    if !valid_machine_network_defaults(&defaults) {
        return Err(Error::msg("missing native machine network defaults"));
    }
    let mut entries = match defaults {
        JsonValue::Object(entries) => entries,
        _ => return Err(Error::msg("invalid native machine defaults")),
    };
    entries.retain(|(k, _)| k != "subnet");
    entries.push(("subnet".to_string(), JsonValue::Str(choices.subnet.clone())));
    let machine = serialize(&JsonValue::Object(entries));
    let destination = inputs::destination(
        template,
        &choices.hostname,
        "",
        &choices.password_hash,
        &choices.subnet,
    )?;
    let config = parse(&destination).map_err(|_| Error::msg("invalid destination"))?;
    let mut config_entries = match config {
        JsonValue::Object(entries) => entries,
        _ => return Err(Error::msg("invalid destination")),
    };
    let storage_value = config_entries
        .iter()
        .rev()
        .find(|(k, _)| k == "storage")
        .map(|(_, v)| v.clone())
        .unwrap_or(JsonValue::Null);
    let mut storage_entries = match storage_value {
        JsonValue::Object(entries) => entries,
        _ => return Err(Error::msg("invalid destination")),
    };
    let files_value = storage_entries
        .iter()
        .rev()
        .find(|(k, _)| k == "files")
        .map(|(_, v)| v.clone())
        .unwrap_or(JsonValue::Null);
    let mut files = match files_value {
        JsonValue::Array(files) => files,
        JsonValue::Null => Vec::new(),
        _ => return Err(Error::msg("invalid destination")),
    };
    rewrite_candidate_host_file(&mut files, machine.as_bytes())?;
    let encoded_files = JsonValue::Array(files);
    // Re-encode through parsed values so nested duplicates collapse exactly
    // like Go remarshaling decoded `RawMessage`s.
    let files_json: JsonValue = parse(serialize(&encoded_files).as_bytes())
        .map_err(|_| Error::msg("invalid destination"))?;
    storage_entries.retain(|(k, _)| k != "files");
    storage_entries.push(("files".to_string(), files_json));
    let storage_json: JsonValue = parse(serialize(&JsonValue::Object(storage_entries)).as_bytes())
        .map_err(|_| Error::msg("invalid destination"))?;
    config_entries.retain(|(k, _)| k != "storage");
    config_entries.push(("storage".to_string(), storage_json));
    Ok(serialize(&JsonValue::Object(config_entries)).into_bytes())
}

#[cfg(test)]
mod tests;
