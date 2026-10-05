//! Signature-verified CoreOS ISO fetching (`coreos_iso.go`). No
//! customization, boot, key import, or disk installation is implicit.

use crate::coreos::{download_http, CoreOSImage, VerifiedBase};
use crate::files::{fresh_directory, hash_file, require_native, write_new};
use crate::http::{HttpTransport, UreqTransport};
use crate::{look_path, Error};
use std::path::{Path, PathBuf};

fn admit_coreos_iso_fetch(arch: &str, signer: &str, keyring: &str) -> Result<String, Error> {
    require_native(arch)?;
    if !(signer.len() == 40 || signer.len() == 64) || !signer.bytes().all(|b| b.is_ascii_hexdigit())
    {
        return Err(Error::msg("full trusted signer fingerprint required"));
    }
    hash_file(Path::new(keyring))?;
    let absolute = crate::files::abs_path(Path::new(keyring))?;
    look_path("gpgv")?;
    Ok(absolute.to_string_lossy().into_owned())
}

fn fetch_verified_iso<T: HttpTransport>(
    transport: &T,
    img: &CoreOSImage,
    dest: &Path,
    keyring: &str,
    signer: &str,
    out: &Path,
) -> Result<(), Error> {
    download_http(transport, &img.url, dest, 8 << 30)?;
    match hash_file(dest) {
        Ok(sum) if sum == img.sha256 => {}
        _ => return Err(Error::msg("CoreOS ISO checksum mismatch")),
    }
    let sig = PathBuf::from(format!("{}.sig", dest.display()));
    download_http(transport, &img.signature_url, &sig, 1 << 20)?;
    crate::coreos::verify_coreos_signature(dest, &sig, keyring, signer, out)
}

/// Retains a new, signature-verified upstream ISO resolving live from the
/// stable stream; no stored version is consulted.
pub fn fetch_coreos_iso(
    arch: &str,
    keyring: &str,
    signer: &str,
    out: &str,
) -> Result<VerifiedBase, Error> {
    fetch_coreos_iso_with(&UreqTransport, arch, keyring, signer, out)
}

pub fn fetch_coreos_iso_with<T: HttpTransport>(
    transport: &T,
    arch: &str,
    keyring: &str,
    signer: &str,
    out: &str,
) -> Result<VerifiedBase, Error> {
    let keyring = admit_coreos_iso_fetch(arch, signer, keyring)?;
    let (release, img) = crate::coreos_stream::resolve_coreos_iso_with(transport, arch)?;
    let out_path = PathBuf::from(out);
    fresh_directory(&out_path)?;
    let dest = out_path.join("coreos.iso");
    fetch_verified_iso(transport, &img, &dest, &keyring, signer, &out_path)?;
    crate::files::chmod(&dest, 0o444)?;
    let result = VerifiedBase {
        path: dest.to_string_lossy().into_owned(),
        sha256: img.sha256.clone(),
        architecture: arch.to_string(),
        release,
        signer: signer.to_ascii_uppercase(),
    };
    write_new(
        &out_path.join("verified-iso.json"),
        (result.marshal() + "\n").as_bytes(),
        0o600,
    )?;
    Ok(result)
}
