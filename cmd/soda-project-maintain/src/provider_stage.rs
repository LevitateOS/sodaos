//! Read-only staging of the shared provider store's selected Muse tool.
//!
//! Projects never gain write access to the shared updater store: this module
//! resolves the current selection through the frozen provider-tools API,
//! opens the bundle executable read-only (no symlinks, no world-writable
//! path), verifies its bytes against the atomically selected release
//! digest/size, and hands the open descriptor to the existing tar staging
//! pipe under the same `muse-native` entry name plus a `muse-selection.json`
//! receipt carrying the admitted identity. The in-project launch path
//! enforces that receipt at invocation. Selection freshness is enforced
//! inside `resolve_new`: a known-stale or failed update blocks new
//! provider work instead of falling back to a pinned descriptor.

use std::ffi::CString;
use std::fs;
use std::io::Write;
use std::os::unix::fs::MetadataExt;
use std::os::unix::fs::OpenOptionsExt;
use std::os::unix::io::{IntoRawFd, RawFd};

use super::filesystem::{go_errno, last_errno, Tool};
use super::sha256::{hex_encode, Sha256};
use sha2::Digest;

use soda_provider_tools::store::Provider;

/// The verified native tool plus its admitted-identity receipt. Both ride
/// the existing tar staging pipe; destinations stay positionally aligned.
pub(crate) struct StagedNative {
    pub(crate) tool: Tool,
    pub(crate) receipt: Tool,
}

/// Open the shared store's currently selected Muse executable read-only.
///
/// Fails closed when the store has no current selection, when the selection
/// is known-stale, or when the bundle bytes do not match the selected
/// release identity. Never falls back to a pinned descriptor silently.
pub(crate) fn resolve_native_staging() -> Result<StagedNative, String> {
    let executable = soda_provider_tools::resolve::resolve_new(Provider::Muse)
        .map_err(|e| format!("provider store selection unavailable: {e}"))?;
    let root = soda_provider_tools::config::store_root();
    let selection = soda_provider_tools::store::read_selection(&root, Provider::Muse)
        .map_err(|e| format!("provider store selection unreadable: {e}"))?
        .ok_or_else(|| String::from("provider store has no selected Muse bundle"))?;
    let expected = root.join("bundles").join(Provider::Muse.as_str());
    if !executable.starts_with(&expected) {
        return Err(String::from(
            "provider store executable is outside the retained bundles",
        ));
    }
    let tool = open_verified(
        &executable,
        &selection.release.digest,
        selection.release.size,
    )?;
    let receipt = stage_receipt(&selection)?;
    Ok(StagedNative { tool, receipt })
}

/// Serialize the admitted selection as the staged receipt: the exact
/// identity the in-project launch path enforces at invocation. Written
/// through an `O_EXCL` temp file opened `O_NOFOLLOW` and unlinked before
/// return, so only the open descriptor reaches the tar pipe.
fn stage_receipt(selection: &soda_provider_tools::store::Selection) -> Result<Tool, String> {
    let bytes = serde_json::to_vec_pretty(selection)
        .map_err(|_| String::from("provider store selection unreadable"))?;
    if bytes.len() > 65536 {
        return Err(String::from("provider store selection unreadable"));
    }
    let pid = std::process::id();
    for n in 0..1000 {
        let path = format!("/tmp/.soda-muse-receipt-{pid}-{n}");
        let mut writer = match std::fs::OpenOptions::new()
            .write(true)
            .create_new(true)
            .mode(0o600)
            .open(&path)
        {
            Ok(writer) => writer,
            Err(e) if e.kind() == std::io::ErrorKind::AlreadyExists => continue,
            Err(_) => return Err(String::from("provider store selection unreadable")),
        };
        let wrote = writer.write(&bytes);
        drop(writer);
        if wrote.is_err() {
            let _ = std::fs::remove_file(&path);
            return Err(String::from("provider store selection unreadable"));
        }
        let reader = std::fs::OpenOptions::new()
            .read(true)
            .custom_flags(libc::O_NOFOLLOW)
            .open(&path)
            .map_err(|_| {
                let _ = std::fs::remove_file(&path);
                String::from("provider store selection unreadable")
            })?;
        let _ = std::fs::remove_file(&path);
        return Ok(Tool {
            name: String::from("muse-selection.json"),
            fd: reader.into_raw_fd(),
            size: bytes.len() as u64,
        });
    }
    Err(String::from("provider store selection unreadable"))
}

/// Open `path` read-only and verify its bytes against the selected digest.
fn open_verified(path: &std::path::Path, digest: &str, size: u64) -> Result<Tool, String> {
    let text = path.to_string_lossy().into_owned();
    let lstat = fs::symlink_metadata(path)
        .map_err(|_| String::from("provider store executable is unavailable"))?;
    if !lstat.is_file() || lstat.mode() & 0o022 != 0 || lstat.len() != size {
        return Err(String::from("provider store executable is unavailable"));
    }
    let raw = CString::new(text).map_err(|_| go_errno(libc::EINVAL))?;
    // O_NOFOLLOW: a symlink at the bundle path is never followed.
    let fd: RawFd = unsafe {
        libc::open(
            raw.as_ptr(),
            libc::O_RDONLY | libc::O_NOFOLLOW | libc::O_CLOEXEC | libc::O_NONBLOCK,
            0,
        )
    };
    if fd < 0 {
        return Err(go_errno(last_errno()));
    }
    let tool = Tool {
        name: String::from("muse-native"),
        fd,
        size,
    };
    verify_digest(&tool, digest)?;
    Ok(tool)
}

/// Hash the open descriptor from the start without changing its offset,
/// mirroring the pinned-tool verification shape.
fn verify_digest(tool: &Tool, digest: &str) -> Result<(), String> {
    let mut hasher = Sha256::new();
    let mut offset: i64 = 0;
    let mut chunk = [0u8; 65536];
    loop {
        let n = unsafe {
            libc::pread(
                tool.fd,
                chunk.as_mut_ptr() as *mut libc::c_void,
                chunk.len(),
                offset,
            )
        };
        if n < 0 {
            return Err(format!("read {}: {}", tool.name, go_errno(last_errno())));
        }
        if n == 0 {
            break;
        }
        hasher.update(&chunk[..n as usize]);
        offset += n as i64;
    }
    if hex_encode(&hasher.finalize()) != digest {
        return Err(String::from("provider store digest mismatch"));
    }
    Ok(())
}
