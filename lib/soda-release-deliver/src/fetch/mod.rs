//! `fetch.go`: verified channel fetch with durable state.

use soda_json::JsonValue;

use crate::buildx::{fresh_directory, oci_architecture, private_destination};
use crate::document::{read_document, read_json};
use crate::model::{admit_channel, empty_state, Channel, Highwater, Trust};
use crate::native::{verify_copy, write_json, Runner};
use crate::{hash_bytes, is_channel, now_unix, Error};

mod state;
mod verification;

pub(crate) use state::{lock_state, save_state};
pub(crate) use verification::verify_releases;

pub(crate) fn discover(r: &dyn Runner, t: &Trust, name: &str) -> Result<String, Error> {
    if !is_channel(name) {
        return Err(Error::refused());
    }
    let reference = format!("{}-channel-{name}:{name}", t.prefix);
    let output = r.run(&[
        "--command-timeout=2m",
        "inspect",
        "--raw",
        "--no-creds",
        &format!("docker://{reference}"),
    ])?;
    if output.len() > 1 << 20 {
        return Err(Error::unavailable());
    }
    Ok(format!(
        "{}-channel-{name}@{}",
        t.prefix,
        hash_bytes(&output)
    ))
}

/// `InitState`: initialize explicit durable fetch state.
pub fn init_state(path: &str, t: &Trust) -> Result<(), Error> {
    t.validate()?;
    private_destination(path)?;
    let mut state = empty_state();
    state.trust_epoch = t.epoch;
    state.checked_at = now_unix();
    write_json(path, &state)
}

pub(crate) fn fetch_document<T>(
    r: &dyn Runner,
    t: &Trust,
    reference: &str,
    out: &str,
    decode: impl Fn(&JsonValue) -> Result<T, String>,
) -> Result<T, Error> {
    verify_copy(r, t, reference, &format!("docker://{reference}"), out)?;
    let digest = reference.split('@').nth(1).unwrap_or("");
    read_document(&format!("{out}/image"), digest, decode)
}

fn admit_fetch_request(t: &Trust, name: &str, arch: &str) -> Result<(), Error> {
    if t.validate().is_err() || !is_channel(name) {
        return Err(Error::refused());
    }
    oci_architecture(arch).map_err(|e| Error::msg(e.0))?;
    Ok(())
}

#[allow(clippy::too_many_arguments)] // Arity mirrors the Go owner 1:1.
fn complete_fetch(
    r: &dyn Runner,
    t: &Trust,
    state: &Highwater,
    offer: &Channel,
    reference: &str,
    arch: &str,
    state_path: &str,
    out: &str,
) -> Result<(), Error> {
    let (next, verification) = verify_releases(r, t, state, offer, arch, out);
    save_state(state_path, &next)?;
    verification?;
    let receipt = JsonValue::Object(vec![
        ("Architecture".to_string(), JsonValue::Str(arch.to_string())),
        ("Channel".to_string(), JsonValue::Str(reference.to_string())),
        (
            "Scope".to_string(),
            JsonValue::Str(
                "native signature/digest verification only; no installation or activation"
                    .to_string(),
            ),
        ),
        ("Withdrawn".to_string(), JsonValue::Bool(offer.withdrawn)),
    ]);
    write_json(&format!("{out}/verified.json"), &receipt)
}

/// `Fetch`: verify an approved channel and its images; never installs.
pub fn fetch(
    r: &dyn Runner,
    t: &Trust,
    name: &str,
    arch: &str,
    state_path: &str,
    out: &str,
    now_unix: i64,
) -> Result<(), Error> {
    admit_fetch_request(t, name, arch)?;
    let _lock = lock_state(state_path)?;
    let state: Highwater = read_json(state_path, Highwater::decode)?;
    fresh_directory(out)?;
    let reference = discover(r, t, name)?;
    let offer: Channel =
        fetch_document(r, t, &reference, &format!("{out}/channel"), Channel::decode)?;
    let digest = reference.split('@').nth(1).unwrap_or("");
    let state = admit_channel(t, &state, &offer, digest, name, now_unix)?;
    save_state(state_path, &state)?;
    complete_fetch(r, t, &state, &offer, &reference, arch, state_path, out)
}

#[cfg(test)]
mod tests;
