//! CoreOS base fetching (`coreos.go`): signature-verified upstream qemu
//! images over bounded HTTPS with an explicitly supplied keyring and
//! signer. No trust-root downloads, no unsigned fallback.

use crate::files::{fresh_directory, hash_file, require_native, write_new};
use crate::http::{get_follow, HttpTransport, UreqTransport};
use crate::json_go::{marshal_indent, Emit};
use crate::{io_error, look_path, Error};
use std::io::{Read, Write};
use std::path::{Path, PathBuf};
use std::time::Duration;

/// Verified download triple: image, detached signature, checksums.
#[derive(Debug, Clone, Default, PartialEq, Eq)]
pub struct CoreOSImage {
    pub url: String,
    pub signature_url: String,
    pub sha256: String,
    pub uncompressed_sha256: String,
}

impl CoreOSImage {
    pub fn emit(&self) -> Emit {
        Emit::Object(vec![
            ("URL".to_string(), Emit::Str(self.url.clone())),
            (
                "SignatureURL".to_string(),
                Emit::Str(self.signature_url.clone()),
            ),
            ("SHA256".to_string(), Emit::Str(self.sha256.clone())),
            (
                "UncompressedSHA256".to_string(),
                Emit::Str(self.uncompressed_sha256.clone()),
            ),
        ])
    }
}

/// Retained verified base and its identity record.
#[derive(Debug, Clone, Default, PartialEq, Eq)]
pub struct VerifiedBase {
    pub path: String,
    pub sha256: String,
    pub architecture: String,
    pub release: String,
    pub signer: String,
}

impl VerifiedBase {
    pub fn marshal(&self) -> String {
        marshal_indent(&Emit::Object(vec![
            ("Path".to_string(), Emit::Str(self.path.clone())),
            ("SHA256".to_string(), Emit::Str(self.sha256.clone())),
            (
                "Architecture".to_string(),
                Emit::Str(self.architecture.clone()),
            ),
            ("Release".to_string(), Emit::Str(self.release.clone())),
            ("Signer".to_string(), Emit::Str(self.signer.clone())),
        ]))
    }
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

/// Runs a command with piped stdout drained through a bounded writer.
fn run_bounded(
    name: &str,
    args: &[String],
    timeout: Duration,
    stdout_limit: i64,
    drain: &mut dyn FnMut(&[u8]) -> Result<(), Error>,
) -> Result<(), Error> {
    let mut child = std::process::Command::new(name)
        .args(args)
        .stdin(std::process::Stdio::null())
        .stdout(std::process::Stdio::piped())
        .stderr(std::process::Stdio::null())
        .spawn()
        .map_err(|e| io_error(name, Path::new(name), e))?;
    let mut stdout = child.stdout.take();
    let deadline = std::time::Instant::now() + timeout;
    let mut remaining = stdout_limit;
    let mut buf = [0u8; 32 << 10];
    let mut drained = stdout.is_none();
    let mut timed_out = false;
    while !drained {
        if std::time::Instant::now() > deadline {
            timed_out = true;
            break;
        }
        match child
            .try_wait()
            .map_err(|e| io_error(name, Path::new(name), e))?
        {
            Some(_) => {
                // Process exited; drain the rest of the pipe.
                if let Some(out) = stdout.as_mut() {
                    use std::os::unix::io::AsRawFd;
                    set_nonblocking(out.as_raw_fd(), true);
                    loop {
                        match out.read(&mut buf) {
                            Ok(0) => break,
                            Ok(n) => {
                                if n as i64 > remaining {
                                    return Err(Error::msg("input exceeds size limit"));
                                }
                                drain(&buf[..n])?;
                                remaining -= n as i64;
                            }
                            Err(e) if e.kind() == std::io::ErrorKind::WouldBlock => break,
                            Err(e) => return Err(Error::msg(e.to_string())),
                        }
                    }
                }
                drained = true;
            }
            None => {
                if let Some(out) = stdout.as_mut() {
                    use std::os::unix::io::AsRawFd;
                    set_nonblocking(out.as_raw_fd(), true);
                    match out.read(&mut buf) {
                        Ok(0) => std::thread::sleep(Duration::from_millis(5)),
                        Ok(n) => {
                            if n as i64 > remaining {
                                let _ = child.kill();
                                return Err(Error::msg("input exceeds size limit"));
                            }
                            drain(&buf[..n])?;
                            remaining -= n as i64;
                        }
                        Err(e) if e.kind() == std::io::ErrorKind::WouldBlock => {
                            std::thread::sleep(Duration::from_millis(5))
                        }
                        Err(e) => return Err(Error::msg(e.to_string())),
                    }
                }
            }
        }
    }
    if timed_out {
        let _ = child.kill();
        let _ = child.wait();
        return Err(Error::msg(format!("{name} timed out")));
    }
    let status = child
        .wait()
        .map_err(|e| io_error(name, Path::new(name), e))?;
    if !status.success() {
        return Err(Error::msg(format!("{name} failed")));
    }
    Ok(())
}

fn set_nonblocking(fd: std::os::unix::io::RawFd, nonblocking: bool) {
    unsafe {
        let flags = libc::fcntl(fd, libc::F_GETFL);
        if flags < 0 {
            return;
        }
        let flags = if nonblocking {
            flags | libc::O_NONBLOCK
        } else {
            flags & !libc::O_NONBLOCK
        };
        libc::fcntl(fd, libc::F_SETFL, flags);
    }
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
    let failed = run_bounded(
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
    write_new(
        &out_path.join("verified-base.json"),
        (result.marshal() + "\n").as_bytes(),
        0o600,
    )?;
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
    let failed = run_bounded(
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

    #[test]
    fn verified_base_marshal_shape() {
        let base = VerifiedBase {
            path: "/out/coreos.qcow2".to_string(),
            sha256: "a".repeat(64),
            architecture: "x86_64".to_string(),
            release: "44.20260901.1.0".to_string(),
            signer: "ABC".to_string(),
        };
        let text = base.marshal();
        assert!(text.starts_with("{\n  \"Path\": "));
        assert!(text.contains("\n  \"Signer\": \"ABC\"\n}"));
    }
}
