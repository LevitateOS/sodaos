//! CoreOS base fetching (`coreos.go`): signature-verified upstream qemu
//! images over bounded HTTPS with an explicitly supplied keyring and
//! signer. No trust-root downloads, no unsigned fallback.

use crate::files::{fresh_directory, hash_file, require_native, write_new};
use crate::http::{get_follow, HttpTransport, UreqTransport};
use crate::{io_error, look_path, Error};
use serde::de::{self, MapAccess, Visitor};
use serde::{Deserialize, Deserializer, Serialize};
use std::io::{Read, Write};
use std::path::{Path, PathBuf};
use std::time::Duration;

mod process;

/// Verified download triple: image, detached signature, checksums.
#[derive(Debug, Clone, Default, PartialEq, Eq, Serialize)]
pub struct CoreOSImage {
    #[serde(rename = "URL")]
    pub url: String,
    #[serde(rename = "SignatureURL")]
    pub signature_url: String,
    #[serde(rename = "SHA256")]
    pub sha256: String,
    #[serde(rename = "UncompressedSHA256")]
    pub uncompressed_sha256: String,
}

impl<'de> Deserialize<'de> for CoreOSImage {
    fn deserialize<D: Deserializer<'de>>(deserializer: D) -> Result<Self, D::Error> {
        struct ImageVisitor;
        impl<'de> Visitor<'de> for ImageVisitor {
            type Value = CoreOSImage;
            fn expecting(&self, f: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
                f.write_str("CoreOS image object")
            }
            fn visit_map<A: MapAccess<'de>>(self, mut map: A) -> Result<Self::Value, A::Error> {
                let (mut url, mut signature_url, mut sha256, mut uncompressed_sha256) =
                    (None, None, None, None);
                while let Some(key) = map.next_key::<String>()? {
                    if key.eq_ignore_ascii_case("URL") {
                        url = Some(map.next_value::<Box<serde_json::value::RawValue>>()?);
                    } else if key.eq_ignore_ascii_case("SignatureURL") {
                        signature_url = Some(map.next_value::<Box<serde_json::value::RawValue>>()?);
                    } else if key.eq_ignore_ascii_case("SHA256") {
                        sha256 = Some(map.next_value::<Box<serde_json::value::RawValue>>()?);
                    } else if key.eq_ignore_ascii_case("UncompressedSHA256") {
                        uncompressed_sha256 =
                            Some(map.next_value::<Box<serde_json::value::RawValue>>()?);
                    } else {
                        return Err(de::Error::unknown_field(
                            &key,
                            &["URL", "SignatureURL", "SHA256", "UncompressedSHA256"],
                        ));
                    }
                }
                fn string<E: de::Error>(
                    raw: Option<Box<serde_json::value::RawValue>>,
                ) -> Result<String, E> {
                    match raw {
                        None => Ok(String::new()),
                        Some(raw) => serde_json::from_str::<Option<String>>(raw.get())
                            .map(Option::unwrap_or_default)
                            .map_err(E::custom),
                    }
                }
                Ok(CoreOSImage {
                    url: string(url)?,
                    signature_url: string(signature_url)?,
                    sha256: string(sha256)?,
                    uncompressed_sha256: string(uncompressed_sha256)?,
                })
            }
        }
        deserializer.deserialize_map(ImageVisitor)
    }
}

/// Retained verified base and its identity record.
#[derive(Debug, Clone, Default, PartialEq, Eq, Serialize)]
pub struct VerifiedBase {
    #[serde(rename = "Path")]
    pub path: String,
    #[serde(rename = "SHA256")]
    pub sha256: String,
    #[serde(rename = "Architecture")]
    pub architecture: String,
    #[serde(rename = "Release")]
    pub release: String,
    #[serde(rename = "Signer")]
    pub signer: String,
}

/// Strict metadata-URL shape shared with sibling packages.
pub fn https_url(raw: &str) -> bool {
    soda_build_tools::reader::url::https_url(raw)
}

fn valid_signer(signer: &str) -> bool {
    (signer.len() == 40 || signer.len() == 64) && signer.bytes().all(|b| b.is_ascii_hexdigit())
}

fn admit_coreos_fetch(arch: &str, signer: &str, keyring: &Path) -> Result<(), Error> {
    require_native(arch)?;
    if !valid_signer(signer) {
        return Err(Error::msg("full trusted signer fingerprint required"));
    }
    hash_file(keyring)?;
    for name in ["gpgv", "xz"] {
        look_path(name)?;
    }
    Ok(())
}

/// Bounded HTTPS download that never overwrites its destination.
pub fn download_http<T: HttpTransport>(
    transport: &T,
    source: &str,
    dest: &Path,
    max: i64,
) -> Result<(), Error> {
    if !https_url(source) || max <= 0 {
        return Err(Error::msg("bounded HTTPS download required"));
    }
    let mut response = get_follow(
        transport,
        source,
        None,
        Duration::from_secs(30 * 60),
        "unsafe download redirect",
    )
    .map_err(|_| Error::msg("CoreOS download failed"))?;
    if response.status != 200 {
        return Err(Error::msg("CoreOS download HTTP failure"));
    }
    let f = std::fs::OpenOptions::new()
        .create_new(true)
        .write(true)
        .open(dest)
        .map_err(|e| io_error("open", dest, e))?;
    crate::files::chmod(dest, 0o600)?;
    let mut f = f;
    let mut remaining = max;
    let mut buf = [0u8; 32 << 10];
    let copy_result = (|| -> Result<(), Error> {
        loop {
            let n = response
                .body
                .read(&mut buf)
                .map_err(|e| Error::msg(e.to_string()))?;
            if n == 0 {
                return Ok(());
            }
            if n as i64 > remaining {
                return Err(Error::msg("input exceeds size limit"));
            }
            f.write_all(&buf[..n])
                .map_err(|e| io_error("write", dest, e))?;
            remaining -= n as i64;
        }
    })();
    let close_result = f.sync_all().map_err(|e| io_error("write", dest, e));
    copy_result.and(close_result)
}

/// Bounded HTTPS download over the production transport.
pub fn download(source: &str, dest: &Path, max: i64) -> Result<(), Error> {
    download_http(&UreqTransport, source, dest, max)
}

fn download_verified_archive<T: HttpTransport>(
    transport: &T,
    img: &CoreOSImage,
    archive: &Path,
    sig: &Path,
    keyring: &str,
    signer: &str,
    out: &Path,
) -> Result<(), Error> {
    download_http(transport, &img.url, archive, 8 << 30)?;
    match hash_file(archive) {
        Ok(sum) if sum == img.sha256 => {}
        Ok(_) => return Err(Error::msg("compressed CoreOS checksum mismatch")),
        Err(e) => {
            return Err(Error::msg(format!(
                "{}\ncompressed CoreOS checksum mismatch",
                e.message()
            )))
        }
    }
    download_http(transport, &img.signature_url, sig, 1 << 20)?;
    verify_coreos_signature(archive, sig, keyring, signer, out)
}

fn decompress_coreos(archive: &Path, dest: &Path, uncompressed: &str) -> Result<(), Error> {
    let f = std::fs::OpenOptions::new()
        .create_new(true)
        .write(true)
        .open(dest)
        .map_err(|e| io_error("open", dest, e))?;
    crate::files::chmod(dest, 0o600)?;
    let mut f = f;
    use sha2::Digest;
    let mut hasher = sha2::Sha256::new();
    let archive_arg = archive.to_string_lossy().into_owned();
    let failed = process::run_bounded(
        "xz",
        &[
            "--decompress".to_string(),
            "--stdout".to_string(),
            "--".to_string(),
            archive_arg,
        ],
        Duration::from_secs(30 * 60),
        64 << 30,
        &mut |chunk| {
            f.write_all(chunk).map_err(|e| Error::msg(e.to_string()))?;
            hasher.update(chunk);
            Ok(())
        },
    )
    .is_err()
        || f.sync_all().is_err();
    if failed {
        return Err(Error::msg("CoreOS decompression failed"));
    }
    let sum = {
        let mut out = String::with_capacity(64);
        for byte in hasher.finalize() {
            out.push_str(&format!("{byte:02x}"));
        }
        out
    };
    if sum != uncompressed {
        return Err(Error::msg("uncompressed CoreOS checksum mismatch"));
    }
    crate::files::chmod(dest, 0o444)
}

/// Retains a new, signature-verified upstream qemu image. The image
/// resolves live from the stable stream; no stored version is consulted.
pub fn fetch_coreos(
    arch: &str,
    keyring: &str,
    signer: &str,
    out: &str,
) -> Result<VerifiedBase, Error> {
    fetch_coreos_with(&UreqTransport, arch, keyring, signer, out)
}

pub fn fetch_coreos_with<T: HttpTransport>(
    transport: &T,
    arch: &str,
    keyring: &str,
    signer: &str,
    out: &str,
) -> Result<VerifiedBase, Error> {
    admit_coreos_fetch(arch, signer, Path::new(keyring))?;
    let (release, img) = crate::coreos_stream::resolve_coreos_qemu_with(transport, arch)?;
    let keyring_abs = crate::files::abs_path(Path::new(keyring))?;
    let out_path = PathBuf::from(out);
    fresh_directory(&out_path)?;
    let archive = out_path.join("coreos.qcow2.xz");
    let sig = PathBuf::from(format!("{}.sig", archive.display()));
    download_verified_archive(
        transport,
        &img,
        &archive,
        &sig,
        &keyring_abs.to_string_lossy(),
        signer,
        &out_path,
    )?;
    let dest = out_path.join("coreos.qcow2");
    decompress_coreos(&archive, &dest, &img.uncompressed_sha256)?;
    let result = VerifiedBase {
        path: dest.to_string_lossy().into_owned(),
        sha256: img.uncompressed_sha256.clone(),
        architecture: arch.to_string(),
        release,
        signer: signer.to_ascii_uppercase(),
    };
    let body = serde_json::to_string_pretty(&result)
        .expect("serializing a verified base record cannot fail")
        + "\n";
    write_new(&out_path.join("verified-base.json"), body.as_bytes(), 0o600)?;
    Ok(result)
}

pub(crate) fn verify_coreos_signature(
    image: &Path,
    sig: &Path,
    keyring: &str,
    signer: &str,
    out: &Path,
) -> Result<(), Error> {
    let home = out.join("gnupg");
    std::fs::create_dir(&home).map_err(|e| io_error("mkdir", &home, e))?;
    crate::files::chmod(&home, 0o700)?;
    let mut status = Vec::new();
    let failed = process::run_bounded(
        "gpgv",
        &[
            "--homedir".to_string(),
            home.to_string_lossy().into_owned(),
            "--keyring".to_string(),
            keyring.to_string(),
            "--status-fd=1".to_string(),
            sig.to_string_lossy().into_owned(),
            image.to_string_lossy().into_owned(),
        ],
        Duration::from_secs(5 * 60),
        1 << 20,
        &mut |chunk| {
            status.extend_from_slice(chunk);
            Ok(())
        },
    )
    .is_err();
    if failed {
        return Err(Error::msg("CoreOS signature verification failed"));
    }
    if !soda_build_tools::reader::signature::valid_signature(&status, signer) {
        return Err(Error::msg("signature does not match selected signer"));
    }
    Ok(())
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::http::tests::Stub;
    use std::time::Duration;

    #[test]
    fn oracle_https_url_vectors() {
        // Oracle: TestCoreOSLockAndDownloadBoundaries unsafe inputs.
        for raw in [
            "http://example.test/base",
            "https://user@example.test/base",
            "https://example.test/base?",
            "https://example.test/base#",
            "https://example.test/base?token=x",
        ] {
            assert!(!https_url(raw), "accepted {raw:?}");
        }
        assert!(https_url(
            "https://builds.coreos.fedoraproject.org/streams/stable.json"
        ));
    }

    #[test]
    fn oracle_download_bounds_and_no_overwrite() {
        // Oracle: TestCoreOSDownloadTLSRedirectBoundsAndNoOverwrite outcomes
        // over the stub transport (redirect policy + bounds, no TLS needed).
        let stub = Stub::new(&[
            ("/ok", 200, None, b"fixture"),
            ("/redirect", 302, Some("http://example.invalid/base"), b""),
            ("/query", 302, Some("/ok?token=must-not-follow"), b""),
            ("/missing", 404, None, b"nope"),
            ("/loop", 302, Some("/loop"), b""),
        ]);
        let dir = std::env::temp_dir().join(format!(
            "soda-dl-{}-{}",
            std::process::id(),
            std::time::SystemTime::now()
                .duration_since(std::time::UNIX_EPOCH)
                .unwrap()
                .as_nanos()
        ));
        std::fs::create_dir_all(&dir).unwrap();
        for path in ["/redirect", "/query", "/missing", "/loop"] {
            let out = dir.join(format!("d-{}", &path[1..]));
            assert!(
                download_http(&stub, &format!("https://t.test{path}"), &out, 16).is_err(),
                "accepted {path}"
            );
            assert!(!out.exists(), "rejected response created output for {path}");
        }
        let out = dir.join("ok");
        download_http(&stub, "https://t.test/ok", &out, 16).unwrap();
        assert_eq!(std::fs::read(&out).unwrap(), b"fixture");
        assert!(download_http(&stub, "https://t.test/ok", &out, 16).is_err());
        assert!(download_http(&stub, "https://t.test/ok", &dir.join("limited"), 3).is_err());
        let _ = Duration::from_secs(0);
    }

    /// Serializes the child-process regressions: run_bounded is the only
    /// spawner in this binary, so one waiting observer is unambiguous.
    static PROCESS_GUARD: std::sync::Mutex<()> = std::sync::Mutex::new(());

    /// Locks the guard, tolerating poisoning: a sibling regression may
    /// panic mid-fixture, and exclusion (not shared state) is all we need.
    fn lock_process_guard() -> std::sync::MutexGuard<'static, ()> {
        PROCESS_GUARD
            .lock()
            .unwrap_or_else(|poison| poison.into_inner())
    }

    /// Reaps already-waitable direct children; live strays from a
    /// known-leaky run die within their bounded sleep, so retry briefly.
    fn settle_owned_children() {
        for _ in 0..40 {
            let mut status = 0;
            let waited = unsafe { libc::waitpid(-1, &mut status, libc::WNOHANG) };
            if waited > 0 {
                continue;
            }
            if waited == 0 {
                std::thread::sleep(Duration::from_millis(50));
                continue;
            }
            return;
        }
        panic!("stray child process outlived its bounded fixture");
    }

    /// Asserts no direct child (live or zombie) survived the helper.
    fn assert_no_owned_child() {
        let mut status = 0;
        let waited = unsafe { libc::waitpid(-1, &mut status, libc::WNOHANG) };
        assert_eq!(waited, -1, "run_bounded left a waitable/live direct child");
    }

    #[test]
    fn run_bounded_drain_error_reaps_live_child() {
        // CODEX-CR05D-001: a drain failure against a live child must retire
        // the child while preserving the drain's own error.
        let _guard = lock_process_guard();
        settle_owned_children();
        let mut failing =
            |_chunk: &[u8]| -> Result<(), Error> { Err(Error::msg("fixture drain refused")) };
        let err = process::run_bounded(
            "sh",
            &["-c".to_string(), "echo hi; exec sleep 1".to_string()],
            Duration::from_secs(30),
            i64::MAX,
            &mut failing,
        )
        .unwrap_err();
        assert_eq!(err.message(), "fixture drain refused");
        assert_no_owned_child();
    }

    #[test]
    fn run_bounded_oversize_reaps_live_child() {
        // CODEX-CR05D-001: an oversize read against a live child must reap
        // the killed child instead of leaving a zombie.
        let _guard = lock_process_guard();
        settle_owned_children();
        let mut drained = Vec::new();
        let mut keep = |chunk: &[u8]| -> Result<(), Error> {
            drained.extend_from_slice(chunk);
            Ok(())
        };
        let err = process::run_bounded(
            "sh",
            &["-c".to_string(), "echo hi; exec sleep 1".to_string()],
            Duration::from_secs(30),
            1,
            &mut keep,
        )
        .unwrap_err();
        assert_eq!(err.message(), "input exceeds size limit");
        assert_no_owned_child();
    }

    #[test]
    fn run_bounded_success_and_timeout_unchanged() {
        // CODEX-CR05D-001 pin: success and timeout behavior are unchanged.
        let _guard = lock_process_guard();
        settle_owned_children();
        let mut drained = Vec::new();
        let mut keep = |chunk: &[u8]| -> Result<(), Error> {
            drained.extend_from_slice(chunk);
            Ok(())
        };
        process::run_bounded(
            "sh",
            &["-c".to_string(), "echo hi".to_string()],
            Duration::from_secs(30),
            i64::MAX,
            &mut keep,
        )
        .unwrap();
        assert_eq!(drained, b"hi\n");
        assert_no_owned_child();
        let mut sink = |_chunk: &[u8]| -> Result<(), Error> { Ok(()) };
        let err = process::run_bounded(
            "sleep",
            &["5".to_string()],
            Duration::from_millis(100),
            i64::MAX,
            &mut sink,
        )
        .unwrap_err();
        assert_eq!(err.message(), "sleep timed out");
        assert_no_owned_child();
    }
}
