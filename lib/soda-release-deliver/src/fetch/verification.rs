use std::collections::BTreeMap;

use crate::buildx::oci_architecture;
use crate::document::read_file;
use crate::jsonx::{parse_lenient, Soft};
use crate::model::{admit_release, Candidate, Channel, Highwater, Release, Trust};
use crate::native::{verify_copy, Runner};
use crate::payload::Payload;
use crate::Error;

use super::fetch_document;

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

fn check_verified_image_manifest(manifest_path: &str, expected_config: &str) -> Result<(), Error> {
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

fn build_expected_release_configs(candidate: &Candidate, p: &Payload) -> BTreeMap<String, String> {
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
            expected
                .get(reference)
                .cloned()
                .unwrap_or_default()
                .as_str(),
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

#[allow(clippy::too_many_arguments)] // Arity mirrors the Go owner 1:1.
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
            r,
            t,
            &state,
            c,
            &name,
            &reference,
            out,
            &mut tracker,
            &mut refs_seen,
        ) {
            Ok(next) => state = next,
            Err((next, e)) => return (next, Err(e)),
        }
    }
    (state, Ok(()))
}
