//! `native.go`: locked skopeo runner, trust policy, verify-copy, signing.

use std::collections::BTreeMap;
use std::os::unix::fs::MetadataExt;

use soda_json::JsonValue;

use crate::buildx::{fresh_directory, write_new};
use crate::document::{read_document, read_file};
use crate::jsonx::{marshal, parse_lenient, Binder, Emit};
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
    let value = parse_lenient(TOOL_LOCK.as_bytes()).map_err(|_| Error::msg("invalid tool lock"))?;
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
    let resolved = std::fs::canonicalize(path).map_err(|_| Error::refused())?;
    if resolved.to_string_lossy() != path {
        return Err(Error::refused());
    }
    owned_private_regular(path)
}

pub(crate) fn write_json<T: Emit + ?Sized>(path: &str, value: &T) -> Result<(), Error> {
    write_new(path, &marshal(value), 0o600)
}

mod policy;
pub(crate) use policy::{local_policy, policy_for_publish, registry_config_for_publish};
pub use policy::{merge_policy, write_registry_config};
use policy::{policy_for, registry_config};

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
        let channel: Channel = read_document(snapshot, &p.digest, Channel::decode)?;
        admit_channel(t, &empty_state(), &channel, &p.digest, &role, now_unix())?;
    }
    if p.repository == format!("{}-release", t.prefix) {
        let release: Release = read_document(snapshot, &p.digest, Release::decode)?;
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
        assert_eq!(
            private_file("/nonexistent-xyz").unwrap_err(),
            Error::refused()
        );
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
