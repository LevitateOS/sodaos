//! `fetch.go`: verified channel fetch with durable state.

use std::collections::BTreeMap;
use std::os::fd::AsRawFd;
use std::sync::atomic::{AtomicU64, Ordering};

use soda_json::JsonValue;

use crate::buildx::{fresh_directory, oci_architecture, private_destination};
use crate::document::{read_document, read_file, read_json};
use crate::jsonx::{marshal, parse_lenient, Emit, Soft};
use crate::model::{
    admit_channel, admit_release, empty_state, Candidate, Channel, Highwater, Release, Trust,
};
use crate::payload::Payload;
use crate::native::{private_file, verify_copy, write_json, Runner};
use crate::{hash_bytes, is_channel, now_unix, Error};

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

pub(crate) struct StateLock {
    #[allow(dead_code)]
    file: std::fs::File,
}

impl std::fmt::Debug for StateLock {
    fn fmt(&self, f: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
        f.write_str("StateLock")
    }
}

impl Drop for StateLock {
    fn drop(&mut self) {
        unsafe {
            libc::flock(self.file.as_raw_fd(), libc::LOCK_UN);
        }
    }
}

pub(crate) fn lock_state(path: &str) -> Result<StateLock, Error> {
    private_file(path)?;
    let lock_path = format!("{path}.lock");
    use std::os::unix::fs::OpenOptionsExt;
    let file = std::fs::OpenOptions::new()
        .read(true)
        .write(true)
        .create(true)
        .custom_flags(libc::O_NOFOLLOW)
        .mode(0o600)
        .open(&lock_path)
        .map_err(|e| Error::msg(format!("open {lock_path}: {e}")))?;
    if unsafe { libc::flock(file.as_raw_fd(), libc::LOCK_EX | libc::LOCK_NB) } != 0 {
        return Err(Error::msg("release operation already active"));
    }
    Ok(StateLock { file })
}

static TEMP_COUNTER: AtomicU64 = AtomicU64::new(0);

fn create_temp(dir: &str, prefix: &str) -> Result<(std::fs::File, String), Error> {
    use std::os::unix::fs::OpenOptionsExt;
    for _ in 0..100 {
        let id = TEMP_COUNTER.fetch_add(1, Ordering::Relaxed);
        let name = format!("{dir}/{prefix}{}-{}", std::process::id(), id);
        match std::fs::OpenOptions::new()
            .write(true)
            .create_new(true)
            .mode(0o600)
            .open(&name)
        {
            Ok(file) => return Ok((file, name)),
            Err(e) if e.kind() == std::io::ErrorKind::AlreadyExists => continue,
            Err(e) => return Err(Error::msg(format!("create temp: {e}"))),
        }
    }
    Err(Error::msg("create temp: too many attempts"))
}

pub(crate) fn save_state<T: Emit + ?Sized>(path: &str, value: &T) -> Result<(), Error> {
    let data = marshal(value);
    let parent = match path.rfind('/') {
        Some(0) | None => "/".to_string(),
        Some(i) => path[..i].to_string(),
    };
    let (mut file, temp_path) = create_temp(&parent, ".delivery-state-")?;
    use std::io::Write;
    // Preserve incomplete write attempts rather than repairing them over later state.
    file.write_all(&data)
        .map_err(|e| Error::msg(format!("write state: {e}")))?;
    file.sync_all()
        .map_err(|e| Error::msg(format!("sync state: {e}")))?;
    drop(file);
    std::fs::rename(&temp_path, path).map_err(|e| Error::msg(format!("rename state: {e}")))?;
    let dir =
        std::fs::File::open(&parent).map_err(|e| Error::msg(format!("open {parent}: {e}")))?;
    dir.sync_all()
        .map_err(|e| Error::msg(format!("sync {parent}: {e}")))
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

fn resolve_verification_architectures(c: &Channel, arch: &str) -> Result<Vec<String>, Error> {
    if arch.is_empty() {
        let mut arches = Vec::new();
        for name in c.releases.keys() {
            oci_architecture(name).map_err(|e| Error::msg(e.0))?;
            arches.push(name.clone());
        }
        arches.sort();
        return Ok(arches);
    }
    oci_architecture(arch).map_err(|e| Error::msg(e.0))?;
    if c.releases.get(arch).map(String::is_empty).unwrap_or(true) {
        return Err(Error::msg("architecture unavailable"));
    }
    Ok(vec![arch.to_string()])
}

fn check_verified_image_metadata(config: &[u8], arch: &str) -> Result<(), Error> {
    let value = parse_lenient(config).map_err(|_| Error::refused())?;
    let soft = Soft::new(&value).map_err(|_| Error::refused())?;
    let architecture = soft
        .string("architecture")
        .map_err(|_| Error::refused())?
        .unwrap_or_default();
    let os = soft
        .string("os")
        .map_err(|_| Error::refused())?
        .unwrap_or_default();
    let want = oci_architecture(arch).map_err(|_| Error::refused())?;
    if os != "linux" || architecture != want {
        return Err(Error::refused());
    }
    Ok(())
}

fn check_verified_image_manifest(
    manifest_path: &str,
    expected_config: &str,
) -> Result<(), Error> {
    let data = read_file(manifest_path, 1 << 20)?;
    let value = parse_lenient(&data).map_err(|_| Error::refused())?;
    let soft = Soft::new(&value).map_err(|_| Error::refused())?;
    let config = soft.object("config").map_err(|_| Error::refused())?;
    let digest = match config {
        Some(config) => config
            .string("digest")
            .map_err(|_| Error::refused())?
            .unwrap_or_default(),
        None => String::new(),
    };
    if digest != expected_config {
        return Err(Error::refused());
    }
    Ok(())
}

fn verify_release_image_copy(
    r: &dyn Runner,
    t: &Trust,
    reference: &str,
    arch: &str,
    path: &str,
    expected_config: &str,
) -> Result<(), Error> {
    verify_copy(r, t, reference, &format!("docker://{reference}"), path)?;
    let config = r.run(&[
        "--command-timeout=2m",
        "inspect",
        "--config",
        &format!("dir:{path}/image"),
    ])?;
    check_verified_image_metadata(&config, arch)?;
    check_verified_image_manifest(&format!("{path}/image/manifest.json"), expected_config)
}

fn build_expected_release_configs(
    candidate: &Candidate,
    p: &Payload,
) -> BTreeMap<String, String> {
    let mut expected = BTreeMap::new();
    expected.insert(
        candidate.host_reference.clone(),
        candidate.host.config.clone(),
    );
    for image in p.images.values() {
        expected.insert(image.reference.clone(), image.config.clone());
    }
    expected
}

fn verify_release_images(
    r: &dyn Runner,
    t: &Trust,
    arch: &str,
    out: &str,
    refs: &[String],
    expected: &BTreeMap<String, String>,
    refs_seen: &mut BTreeMap<String, String>,
) -> Result<(), Error> {
    for (i, reference) in refs.iter().enumerate() {
        if let Some(prior) = refs_seen.get(reference) {
            if *prior != arch {
                return Err(Error::msg(
                    "one platform manifest advertised for two architectures",
                ));
            }
            continue;
        }
        let suffix = char::from_u32('0' as u32 + i as u32).unwrap_or('?');
        let path = format!("{out}/{arch}-image-{suffix}");
        verify_release_image_copy(
            r,
            t,
            reference,
            arch,
            &path,
            expected.get(reference).cloned().unwrap_or_default().as_str(),
        )?;
        refs_seen.insert(reference.clone(), arch.to_string());
    }
    Ok(())
}

#[derive(Debug, Default)]
struct ReleaseIdentityTracker {
    release_id: String,
    serial: u64,
    class: String,
}

impl ReleaseIdentityTracker {
    fn check(&mut self, id: &str, serial: u64, class: &str) -> Result<(), Error> {
        if !self.release_id.is_empty()
            && (id != self.release_id || serial != self.serial || class != self.class)
        {
            return Err(Error::msg("mixed architecture release identities"));
        }
        self.release_id = id.to_string();
        self.serial = serial;
        self.class = class.to_string();
        Ok(())
    }
}

fn verify_architecture_release(
    r: &dyn Runner,
    t: &Trust,
    s: &Highwater,
    c: &Channel,
    arch: &str,
    reference: &str,
    out: &str,
    tracker: &mut ReleaseIdentityTracker,
    refs_seen: &mut BTreeMap<String, String>,
) -> Result<Highwater, (Highwater, Error)> {
    let release: Release = fetch_document(
        r,
        t,
        reference,
        &format!("{out}/{arch}-release"),
        Release::decode,
    )
    .map_err(|e| (s.clone(), e))?;
    let next = admit_release(t, s, c, arch, reference, &release).map_err(|e| (s.clone(), e))?;
    let (p, candidate) = release.validate(t).map_err(|e| (s.clone(), e))?;
    tracker
        .check(&p.id, release.serial, &release.class)
        .map_err(|e| (s.clone(), e))?;
    let refs = release.references(t).map_err(|e| (s.clone(), e))?;
    let expected = build_expected_release_configs(&candidate, &p);
    verify_release_images(r, t, arch, out, &refs, &expected, refs_seen)
        .map_err(|e| (next.clone(), e))?;
    Ok(next)
}

/// `verifyReleases`: check one client architecture, or every advertised one.
/// Returns the highest authenticated state alongside any failure.
pub(crate) fn verify_releases(
    r: &dyn Runner,
    t: &Trust,
    s: &Highwater,
    c: &Channel,
    arch: &str,
    out: &str,
) -> (Highwater, Result<(), Error>) {
    if c.withdrawn {
        return (s.clone(), Ok(()));
    }
    let arches = match resolve_verification_architectures(c, arch) {
        Ok(arches) => arches,
        Err(e) => return (s.clone(), Err(e)),
    };
    let mut refs_seen: BTreeMap<String, String> = BTreeMap::new();
    let mut tracker = ReleaseIdentityTracker::default();
    let mut state = s.clone();
    for name in arches {
        let reference = c.releases.get(&name).cloned().unwrap_or_default();
        if reference.is_empty() {
            continue;
        }
        match verify_architecture_release(
            r, t, &state, c, &name, &reference, out, &mut tracker, &mut refs_seen,
        ) {
            Ok(next) => state = next,
            Err((next, e)) => return (next, Err(e)),
        }
    }
    (state, Ok(()))
}

fn admit_fetch_request(t: &Trust, name: &str, arch: &str) -> Result<(), Error> {
    if t.validate().is_err() || !is_channel(name) {
        return Err(Error::refused());
    }
    oci_architecture(arch).map_err(|e| Error::msg(e.0))?;
    Ok(())
}

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
        (
            "Architecture".to_string(),
            JsonValue::Str(arch.to_string()),
        ),
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
    let offer: Channel = fetch_document(r, t, &reference, &format!("{out}/channel"), Channel::decode)?;
    let digest = reference.split('@').nth(1).unwrap_or("");
    let state = admit_channel(t, &state, &offer, digest, name, now_unix)?;
    save_state(state_path, &state)?;
    complete_fetch(r, t, &state, &offer, &reference, arch, state_path, out)
}

#[cfg(test)]
mod tests {
    use super::*;

    fn private_dir(prefix: &str) -> String {
        let dir = std::env::temp_dir().join(format!(
            "{prefix}-{}-{}",
            std::process::id(),
            std::time::SystemTime::now()
                .duration_since(std::time::UNIX_EPOCH)
                .unwrap()
                .as_nanos()
        ));
        std::fs::create_dir_all(&dir).unwrap();
        use std::os::unix::fs::PermissionsExt;
        std::fs::set_permissions(&dir, std::fs::Permissions::from_mode(0o700)).unwrap();
        dir.to_string_lossy().into_owned()
    }

    #[test]
    fn state_lock_serializes_and_saves() {
        let dir = private_dir("srd-lock");
        let path = format!("{dir}/state.json");
        std::fs::write(&path, b"{}").unwrap();
        use std::os::unix::fs::PermissionsExt;
        std::fs::set_permissions(&path, std::fs::Permissions::from_mode(0o600)).unwrap();
        let _first = lock_state(&path).unwrap();
        assert_eq!(
            lock_state(&path).unwrap_err(),
            Error::msg("release operation already active")
        );
        drop(_first);
        let _second = lock_state(&path).unwrap();
        let mut state = empty_state();
        state.checked_at = 42;
        save_state(&path, &state).unwrap();
        let raw = std::fs::read(&path).unwrap();
        assert!(String::from_utf8(raw).unwrap().contains("\"CheckedAt\": 42"));
    }

    #[test]
    fn fetch_request_admission() {
        let trust = Trust::default();
        assert!(admit_fetch_request(&trust, "candidate", "x86_64").is_err());
        assert!(admit_fetch_request(&trust, "bogus", "x86_64").is_err());
    }
}
