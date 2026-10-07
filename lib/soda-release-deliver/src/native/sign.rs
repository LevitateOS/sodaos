use serde::{Deserialize, Serialize};
use std::collections::BTreeMap;

use crate::buildx::fresh_directory;
use crate::document::{read_document, read_file};
use crate::model::{admit_channel, empty_state, Channel, Permit, Release, Trust};
use crate::{hash_bytes, is_channel, now_unix, Error};

use super::{local_policy, private_file, verify_copy, write_json, Runner};

/// `SecretFiles`: restricted signer key inputs.
#[derive(Debug, Clone, Default, PartialEq, Eq, Serialize, Deserialize)]
#[serde(default, deny_unknown_fields, rename_all = "PascalCase")]
pub struct SecretFiles {
    #[serde(deserialize_with = "crate::json_serde::null_default")]
    pub key: String,
    #[serde(deserialize_with = "crate::json_serde::null_default")]
    pub passphrase: String,
}

impl SecretFiles {}

fn admit_sign_inputs(t: &Trust, p: &Permit, input: &str, key: &SecretFiles) -> Result<(), Error> {
    if t.validate().is_err()
        || p.validate(t, now_unix()).is_err()
        || private_file(&key.key).is_err()
        || private_file(&key.passphrase).is_err()
        || !input.starts_with('/')
    {
        return Err(Error::refused());
    }
    Ok(())
}

fn snapshot_sign_source(
    r: &dyn Runner,
    transport: &str,
    input: &str,
    out: &str,
) -> Result<String, Error> {
    if transport != "oci" && transport != "oci-archive" && transport != "dir" {
        return Err(Error::refused());
    }
    fresh_directory(out)?;
    let policy_path = format!("{out}/snapshot-policy.json");
    write_json(&policy_path, &local_policy(transport, input))?;
    let snapshot = format!("{out}/snapshot");
    r.run(&[
        "--command-timeout=10m",
        "--policy",
        &policy_path,
        "copy",
        "--preserve-digests",
        "--remove-signatures",
        &format!("{transport}:{input}"),
        &format!("dir:{snapshot}"),
    ])?;
    Ok(snapshot)
}

fn admit_signed_payload(t: &Trust, p: &Permit, snapshot: &str) -> Result<(), Error> {
    let manifest_bytes = read_file(&format!("{snapshot}/manifest.json"), 1 << 20)?;
    if hash_bytes(&manifest_bytes) != p.digest {
        return Err(Error::refused());
    }
    let role = t.role(&p.repository).unwrap_or_default();
    if is_channel(&role) {
        let channel: Channel = read_document(snapshot, &p.digest)?;
        admit_channel(t, &empty_state(), &channel, &p.digest, &role, now_unix())?;
    }
    if p.repository == format!("{}-release", t.prefix) {
        let release: Release = read_document(snapshot, &p.digest)?;
        release.validate(t)?;
    }
    Ok(())
}

fn emit_signed_directory(
    r: &dyn Runner,
    t: &Trust,
    p: &Permit,
    snapshot: &str,
    out: &str,
    key: &SecretFiles,
) -> Result<(), Error> {
    let policy_path = format!("{out}/sign-policy.json");
    write_json(&policy_path, &local_policy("dir", snapshot))?;
    let signed = format!("{out}/signed");
    let reference = format!("{}@{}", p.repository, p.digest);
    r.run(&[
        "--command-timeout=10m",
        "--policy",
        &policy_path,
        "copy",
        "--preserve-digests",
        "--sign-by-sigstore-private-key",
        &key.key,
        "--sign-passphrase-file",
        &key.passphrase,
        "--sign-identity",
        &reference,
        &format!("dir:{snapshot}"),
        &format!("dir:{signed}"),
    ])?;
    verify_copy(
        r,
        t,
        &reference,
        &format!("dir:{signed}"),
        &format!("{out}/check"),
    )?;
    let mut receipt = BTreeMap::new();
    receipt.insert("Reference".to_string(), reference);
    receipt.insert(
        "Scope".to_string(),
        "native-signed local directory; not published or boot-qualified".to_string(),
    );
    write_json(&format!("{out}/receipt.json"), &receipt)
}

/// `Sign`: sign a private snapshot under an exact protected permit.
pub fn sign(
    r: &dyn Runner,
    t: &Trust,
    p: &Permit,
    transport: &str,
    input: &str,
    out: &str,
    key: &SecretFiles,
) -> Result<(), Error> {
    admit_sign_inputs(t, p, input, key)?;
    let snapshot = snapshot_sign_source(r, transport, input, out)?;
    admit_signed_payload(t, p, &snapshot)?;
    emit_signed_directory(r, t, p, &snapshot, out, key)
}
