//! `native.go`: locked skopeo runner, trust policy, verify-copy, signing.

use std::collections::BTreeMap;
use std::os::unix::fs::MetadataExt;

use soda_json::JsonValue;

use crate::buildx::{fresh_directory, write_new};
use crate::document::{read_document, read_file};
use crate::jsonx::{
    base64_encode, marshal, parse_lenient, parse_strict, Binder, Emit, Emitter,
};
use crate::model::{admit_channel, empty_state, Channel, Permit, Release, Trust};
use crate::payload::decode_opt_string;
use crate::{hash_bytes, is_channel, now_unix, Error};

const TOOL_LOCK: &str = include_str!("../tools.json");

/// Skopeo command runner. Failures surface as `ErrUnavailable`, like Go.
pub trait Runner {
    fn run(&self, args: &[&str]) -> Result<Vec<u8>, Error>;
}

/// `Native`: locked-down skopeo invocation without ambient credentials.
#[derive(Debug, Clone, Default)]
pub struct Native {
    pub home: String,
}

impl Runner for Native {
    fn run(&self, args: &[&str]) -> Result<Vec<u8>, Error> {
        let output = std::process::Command::new("/usr/bin/skopeo")
            .args(args)
            .env_clear()
            .env("PATH", "/usr/bin:/bin")
            .env("HOME", &self.home)
            .env("XDG_RUNTIME_DIR", &self.home)
            .env("LANG", "C.UTF-8")
            .stdout(std::process::Stdio::piped())
            .stderr(std::process::Stdio::null())
            .output()
            .map_err(|_| Error::unavailable())?;
        if !output.status.success() || output.stdout.len() > 2 << 20 {
            return Err(Error::unavailable());
        }
        Ok(output.stdout)
    }
}

/// `CheckNative`: require the locked skopeo version.
pub fn check_native(r: &dyn Runner) -> Result<(), Error> {
    let value = parse_lenient(TOOL_LOCK.as_bytes())
        .map_err(|_| Error::msg("invalid tool lock"))?;
    let versions = match &value {
        JsonValue::Object(entries) => entries,
        _ => return Err(Error::msg("invalid tool lock")),
    };
    let mut skopeo = String::new();
    for (key, item) in versions {
        if key == "skopeo" {
            if let JsonValue::Str(s) = item {
                skopeo = s.clone();
            }
        }
    }
    let output = r.run(&["--version"])?;
    let text = String::from_utf8_lossy(&output);
    if !text.starts_with(&format!("skopeo version {skopeo} ")) {
        return Err(Error::msg("locked native skopeo required"));
    }
    Ok(())
}

fn owned_private_regular(path: &str) -> Result<(), Error> {
    let st = std::fs::symlink_metadata(path).map_err(|_| Error::refused())?;
    if !st.is_file() || st.mode() & 0o077 != 0 || st.len() > 1 << 20 {
        return Err(Error::refused());
    }
    if st.uid() != unsafe { libc::geteuid() } {
        return Err(Error::refused());
    }
    let parent = match path.rfind('/') {
        Some(0) | None => "/".to_string(),
        Some(i) => path[..i].to_string(),
    };
    let parent_st = std::fs::metadata(&parent).map_err(|_| Error::refused())?;
    if parent_st.mode() & 0o077 != 0 {
        return Err(Error::refused());
    }
    Ok(())
}

/// `PrivateFile`: owned, private, unlinked regular file at an exact path.
pub fn private_file(path: &str) -> Result<(), Error> {
    if !path.starts_with('/') {
        return Err(Error::refused());
    }
    let resolved =
        std::fs::canonicalize(path).map_err(|_| Error::refused())?;
    if resolved.to_string_lossy() != path {
        return Err(Error::refused());
    }
    owned_private_regular(path)
}

pub(crate) fn write_json<T: Emit + ?Sized>(path: &str, value: &T) -> Result<(), Error> {
    write_new(path, &marshal(value), 0o600)
}

#[derive(Debug, Clone, Default)]
struct Requirement {
    type_name: String,
    key_datas: Vec<String>,
    signed_identity: BTreeMap<String, String>,
}

impl Emit for Requirement {
    fn emit(&self, e: &mut Emitter) {
        e.begin_object(false);
        e.field(true, "type");
        e.string(&self.type_name);
        if !self.key_datas.is_empty() {
            e.field(false, "keyDatas");
            e.begin_array(false);
            for (i, key) in self.key_datas.iter().enumerate() {
                e.item(i == 0);
                e.string(key);
            }
            e.end_array(false);
        }
        if !self.signed_identity.is_empty() {
            e.field(false, "signedIdentity");
            e.begin_object(false);
            for (i, (key, value)) in self.signed_identity.iter().enumerate() {
                e.field(i == 0, key);
                e.string(value);
            }
            e.end_object(false);
        }
        e.end_object(false);
    }
}

fn requirement(t: &Trust, repo: &str) -> Result<Requirement, Error> {
    let role = t.role(repo)?;
    let mut keys = Vec::new();
    if let Some(role_keys) = t.keys.get(&role) {
        for key in role_keys {
            keys.push(base64_encode(key.as_bytes()));
        }
    }
    let mut identity = BTreeMap::new();
    identity.insert("type".to_string(), "exactRepository".to_string());
    identity.insert("dockerRepository".to_string(), repo.to_string());
    Ok(Requirement {
        type_name: "sigstoreSigned".to_string(),
        key_datas: keys,
        signed_identity: identity,
    })
}

fn requirement_value(req: &Requirement) -> JsonValue {
    let mut entries = vec![("type".to_string(), JsonValue::Str(req.type_name.clone()))];
    if !req.key_datas.is_empty() {
        entries.push((
            "keyDatas".to_string(),
            JsonValue::Array(
                req.key_datas
                    .iter()
                    .map(|k| JsonValue::Str(k.clone()))
                    .collect(),
            ),
        ));
    }
    if !req.signed_identity.is_empty() {
        entries.push((
            "signedIdentity".to_string(),
            JsonValue::Object(
                req.signed_identity
                    .iter()
                    .map(|(k, v)| (k.clone(), JsonValue::Str(v.clone())))
                    .collect(),
            ),
        ));
    }
    JsonValue::Object(entries)
}

pub(crate) fn policy_for_publish(
    t: &Trust,
    repo: &str,
    transport: &str,
    scope: &str,
) -> Result<JsonValue, Error> {
    policy_for(t, repo, transport, scope)
}

pub(crate) fn registry_config_for_publish(out: &str, t: &Trust) -> Result<String, Error> {
    registry_config(out, t)
}

fn policy_for(t: &Trust, repo: &str, transport: &str, scope: &str) -> Result<JsonValue, Error> {
    let req = requirement(t, repo)?;
    Ok(JsonValue::Object(vec![
        (
            "default".to_string(),
            JsonValue::Array(vec![JsonValue::Object(vec![(
                "type".to_string(),
                JsonValue::Str("reject".to_string()),
            )])]),
        ),
        (
            "transports".to_string(),
            JsonValue::Object(vec![(
                transport.to_string(),
                JsonValue::Object(vec![(
                    scope.to_string(),
                    JsonValue::Array(vec![requirement_value(&req)]),
                )]),
            )]),
        ),
    ]))
}

fn local_policy(transport: &str, path: &str) -> JsonValue {
    JsonValue::Object(vec![
        (
            "default".to_string(),
            JsonValue::Array(vec![JsonValue::Object(vec![(
                "type".to_string(),
                JsonValue::Str("reject".to_string()),
            )])]),
        ),
        (
            "transports".to_string(),
            JsonValue::Object(vec![(
                transport.to_string(),
                JsonValue::Object(vec![(
                    path.to_string(),
                    JsonValue::Array(vec![JsonValue::Object(vec![(
                        "type".to_string(),
                        JsonValue::Str("insecureAcceptAnything".to_string()),
                    )])]),
                )]),
            )]),
        ),
    ])
}

fn soda_trust_repos(t: &Trust) -> Vec<String> {
    let mut repos = vec![
        format!("{}-host", t.prefix),
        format!("{}-release", t.prefix),
    ];
    for name in crate::payload::NAMES {
        repos.push(format!("{}-{name}", t.prefix));
    }
    for channel in ["candidate", "preview", "stable"] {
        repos.push(format!("{}-channel-{channel}", t.prefix));
    }
    repos
}

fn soda_override_exists(existing: &str, repo: &str) -> bool {
    existing == repo
        || existing.starts_with(&format!("{repo}:"))
        || existing.starts_with(&format!("{repo}@"))
        || existing.starts_with(&format!("{repo}/"))
}

fn apply_soda_trust(
    t: &Trust,
    docker: &mut Vec<(String, JsonValue)>,
) -> Result<(), Error> {
    for repo in soda_trust_repos(t) {
        for (existing, _) in docker.iter() {
            if soda_override_exists(existing, &repo) {
                return Err(Error::msg(
                    "existing Soda trust override requires explicit review",
                ));
            }
        }
        let req = requirement(t, &repo)?;
        docker.push((
            repo,
            JsonValue::Array(vec![requirement_value(&req)]),
        ));
    }
    Ok(())
}

/// `MergePolicy`: emit a proposed policy; never installs it.
///
/// Preserved scopes keep their parsed values (re-emitted, not byte-kept);
/// Soda scopes are emitted in Go struct order with sorted scope keys.
pub fn merge_policy(t: &Trust, original: &[u8]) -> Result<Vec<u8>, Error> {
    t.validate()?;
    let value = parse_strict(original)?;
    let mut binder = Binder::new(&value).map_err(|_| Error::refused())?;
    let default_raw = binder
        .raw("default")
        .map_err(|_| Error::refused())?
        .cloned()
        .ok_or_else(Error::refused)?;
    let mut transports: BTreeMap<String, Vec<(String, JsonValue)>> = BTreeMap::new();
    if let Some(entries) = binder.entries("transports").map_err(|_| Error::refused())? {
        for (transport, scopes) in entries {
            let mut scope_list = Vec::new();
            match scopes {
                JsonValue::Object(scopes) => {
                    for (scope, raw) in scopes {
                        scope_list.push((scope.clone(), raw.clone()));
                    }
                }
                _ => return Err(Error::refused()),
            }
            transports.insert(transport.clone(), scope_list);
        }
    }
    binder.finish().map_err(|_| Error::refused())?;
    let docker = transports.entry("docker".to_string()).or_default();
    apply_soda_trust(t, docker)?;
    let mut transport_entries = Vec::new();
    for (transport, scopes) in &transports {
        let mut scopes = scopes.clone();
        scopes.sort_by(|a, b| a.0.cmp(&b.0));
        transport_entries.push((transport.clone(), JsonValue::Object(scopes)));
    }
    let merged = JsonValue::Object(vec![
        ("default".to_string(), default_raw),
        ("transports".to_string(), JsonValue::Object(transport_entries)),
    ]);
    Ok(marshal(&merged))
}

/// `WriteRegistryConfig`: registries.d snippet for Sigstore attachments.
pub fn write_registry_config(out: &str, t: &Trust) -> Result<(), Error> {
    t.validate()?;
    registry_config(out, t)?;
    Ok(())
}

fn registry_config(out: &str, t: &Trust) -> Result<String, Error> {
    let dir = format!("{out}/registries.d");
    {
        use std::os::unix::fs::DirBuilderExt;
        std::fs::DirBuilder::new()
            .mode(0o700)
            .create(&dir)
            .map_err(|e| Error::msg(format!("mkdir {dir}: {e}")))?;
    }
    let mut repos: Vec<(String, JsonValue)> = Vec::new();
    let mut names = vec![
        "host",
        "release",
        "channel-candidate",
        "channel-preview",
        "channel-stable",
    ];
    names.extend(crate::payload::NAMES.iter().copied());
    for name in names {
        repos.push((
            format!("{}-{name}", t.prefix),
            JsonValue::Object(vec![(
                "use-sigstore-attachments".to_string(),
                JsonValue::Bool(true),
            )]),
        ));
    }
    repos.sort_by(|a, b| a.0.cmp(&b.0));
    write_json(
        &format!("{dir}/soda.yaml"),
        &JsonValue::Object(vec![("docker".to_string(), JsonValue::Object(repos))]),
    )?;
    Ok(dir)
}

fn admit_verify_source(
    reference: &str,
    source: &str,
    repo: &str,
) -> Result<(String, String), Error> {
    let (transport, scope) = source.split_once(':').ok_or_else(Error::refused)?;
    match transport {
        "docker" => {
            if source != format!("docker://{reference}") {
                return Err(Error::refused());
            }
            Ok((transport.to_string(), repo.to_string()))
        }
        "dir" => {
            if !scope.starts_with('/') {
                return Err(Error::refused());
            }
            Ok((transport.to_string(), scope.to_string()))
        }
        _ => Err(Error::refused()),
    }
}

/// `VerifyCopy`: fresh native copy proving signature enforcement.
pub fn verify_copy(
    r: &dyn Runner,
    t: &Trust,
    reference: &str,
    source: &str,
    out: &str,
) -> Result<(), Error> {
    let (repo, _) = t.reference(reference)?;
    let digest = reference.split('@').nth(1).unwrap_or("");
    fresh_directory(out)?;
    let (transport, scope) = admit_verify_source(reference, source, &repo)?;
    let policy = policy_for(t, &repo, &transport, &scope)?;
    let policy_path = format!("{out}/policy.json");
    write_json(&policy_path, &policy)?;
    let registry = registry_config(out, t)?;
    r.run(&[
        "--command-timeout=10m",
        "--policy",
        &policy_path,
        "--registries.d",
        &registry,
        "copy",
        "--preserve-digests",
        "--src-no-creds",
        source,
        &format!("dir:{out}/image"),
    ])?;
    verify_copied_manifest(out, digest)
}

fn verify_copied_manifest(out: &str, digest: &str) -> Result<(), Error> {
    let data = read_file(&format!("{out}/image/manifest.json"), 1 << 20)?;
    if hash_bytes(&data) != digest {
        return Err(Error::refused());
    }
    Ok(())
}

/// `SecretFiles`: restricted signer key inputs.
#[derive(Debug, Clone, Default, PartialEq, Eq)]
pub struct SecretFiles {
    pub key: String,
    pub passphrase: String,
}

impl SecretFiles {
    pub fn decode(value: &JsonValue) -> Result<SecretFiles, String> {
        let mut b = Binder::new(value).map_err(|_| "invalid secrets".to_string())?;
        let secrets = SecretFiles {
            key: decode_opt_string(&mut b, "Key")?,
            passphrase: decode_opt_string(&mut b, "Passphrase")?,
        };
        b.finish_name()?;
        Ok(secrets)
    }
}

fn admit_sign_inputs(
    t: &Trust,
    p: &Permit,
    input: &str,
    key: &SecretFiles,
) -> Result<(), Error> {
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
        let channel: Channel =
            read_document(snapshot, &p.digest, Channel::decode)?;
        admit_channel(t, &empty_state(), &channel, &p.digest, &role, now_unix())?;
    }
    if p.repository == format!("{}-release", t.prefix) {
        let release: Release =
            read_document(snapshot, &p.digest, Release::decode)?;
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
    verify_copy(r, t, &reference, &format!("dir:{signed}"), &format!("{out}/check"))?;
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

#[cfg(test)]
mod tests {
    use super::*;

    struct MockRunner {
        output: Vec<u8>,
    }

    impl Runner for MockRunner {
        fn run(&self, _args: &[&str]) -> Result<Vec<u8>, Error> {
            Ok(self.output.clone())
        }
    }

    #[test]
    fn check_native_pins_version() {
        let good = MockRunner {
            output: b"skopeo version 1.22.2 something".to_vec(),
        };
        assert!(check_native(&good).is_ok());
        let bad = MockRunner {
            output: b"skopeo version 9.9.9 something".to_vec(),
        };
        assert_eq!(
            check_native(&bad).unwrap_err(),
            Error::msg("locked native skopeo required")
        );
    }

    #[test]
    fn private_file_rules() {
        assert_eq!(private_file("relative/path").unwrap_err(), Error::refused());
        assert_eq!(private_file("/nonexistent-xyz").unwrap_err(), Error::refused());
        let dir = std::env::temp_dir().join(format!("srd-priv-{}", std::process::id()));
        std::fs::create_dir_all(&dir).unwrap();
        use std::os::unix::fs::PermissionsExt;
        std::fs::set_permissions(&dir, std::fs::Permissions::from_mode(0o700)).unwrap();
        let path = dir.join("secret").to_string_lossy().into_owned();
        std::fs::write(&path, b"data").unwrap();
        std::fs::set_permissions(&path, std::fs::Permissions::from_mode(0o600)).unwrap();
        assert!(private_file(&path).is_ok());
        std::fs::set_permissions(&path, std::fs::Permissions::from_mode(0o644)).unwrap();
        assert_eq!(private_file(&path).unwrap_err(), Error::refused());
        std::fs::remove_dir_all(&dir).unwrap();
    }
}
