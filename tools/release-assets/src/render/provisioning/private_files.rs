//! Bounded private-input, host-key and exclusive-output handling.

use std::io::ErrorKind;
use std::path::{Path, PathBuf};
use std::process::Stdio;
use std::time::{Duration, Instant};

use super::{ProvError, ProvKind};

pub(crate) fn read_text(path: &Path) -> Result<String, ProvError> {
    match std::fs::read_to_string(path) {
        Ok(text) => Ok(text),
        Err(e) if e.kind() == ErrorKind::InvalidData => Err(ProvError::new(
            ProvKind::UnicodeDecode,
            format!("cannot decode {}", path.display()),
        )),
        Err(e) => Err(ProvError::io(&e, format!("cannot read {}", path.display()))),
    }
}

/// Bounded regular input, like the script's `regular()`: lstat must show a
/// plain file under 1 MiB, and secret inputs must hide from group/others.
pub(crate) fn regular(path: &Path, private: bool) -> Result<String, ProvError> {
    let meta = std::fs::symlink_metadata(path)
        .map_err(|e| ProvError::io(&e, format!("cannot stat {}", path.display())))?;
    if !meta.file_type().is_file() || meta.len() > 1024 * 1024 {
        return Err(ProvError::value("bounded regular input required"));
    }
    #[cfg(unix)]
    {
        use std::os::unix::fs::MetadataExt;
        if private && meta.mode() & 0o077 != 0 {
            return Err(ProvError::value(
                "secret input must not be accessible to group/others",
            ));
        }
    }
    read_text(path)
}

fn is_label(label: &str, max_middle: usize) -> bool {
    let bytes = label.as_bytes();
    if bytes.is_empty() || bytes.len() > max_middle + 2 {
        return false;
    }
    let edge = |b: u8| b.is_ascii_lowercase() || b.is_ascii_digit();
    if bytes.len() == 1 {
        return edge(bytes[0]);
    }
    if !edge(bytes[0]) || !edge(bytes[bytes.len() - 1]) || bytes.len() - 2 > max_middle {
        return false;
    }
    bytes[1..bytes.len() - 1]
        .iter()
        .all(|b| edge(*b) || *b == b'-')
}

/// Like the script's `appliance_hostname`: 1..=253 chars of dot-separated
/// lowercase alphanumeric labels.
pub fn is_appliance_hostname(value: &str) -> bool {
    let len = value.chars().count();
    (1..=253).contains(&len) && value.split('.').all(|label| is_label(label, 61))
}

/// Like the fixture check: `soda-native-` plus a 1..=42-char label tail.
pub fn is_fixture_hostname(value: &str) -> bool {
    value
        .strip_prefix("soda-native-")
        .is_some_and(|rest| is_label(rest, 40))
}

/// `Path.absolute()`: join the working directory lexically, resolving
/// nothing, so the key path in argv matches the script's spelling.
pub(crate) fn absolute_lexical(path: &Path) -> Result<PathBuf, ProvError> {
    if path.is_absolute() {
        return Ok(path.to_path_buf());
    }
    let cwd =
        std::env::current_dir().map_err(|e| ProvError::io(&e, "cannot read working directory"))?;
    Ok(cwd.join(path))
}

/// Derive the public half without putting private contents in argv or
/// logs. Encrypted/wrong-type inputs fail without an interactive prompt.
pub(crate) fn derive_host_public(host_key: &Path) -> Result<String, ProvError> {
    let absolute = absolute_lexical(host_key)?;
    let mut child = std::process::Command::new("ssh-keygen")
        .args(["-y", "-P", "", "-f"])
        .arg(&absolute)
        .stdin(Stdio::null())
        .stdout(Stdio::piped())
        .stderr(Stdio::piped())
        .spawn()
        .map_err(|e| ProvError::io(&e, "cannot run ssh-keygen"))?;
    let stderr = child.stderr.take().map(|mut pipe| {
        std::thread::spawn(move || {
            let mut sink = Vec::new();
            let _ = std::io::Read::read_to_end(&mut pipe, &mut sink);
        })
    });
    let deadline = Instant::now() + Duration::from_secs(10);
    let status = loop {
        match child
            .try_wait()
            .map_err(|e| ProvError::io(&e, "cannot run ssh-keygen"))?
        {
            Some(status) => break status,
            None => {
                if Instant::now() >= deadline {
                    let _ = child.kill();
                    let _ = child.wait();
                    if let Some(handle) = stderr {
                        let _ = handle.join();
                    }
                    return Err(ProvError::new(ProvKind::Timeout, "ssh-keygen timed out"));
                }
                std::thread::sleep(Duration::from_millis(5));
            }
        }
    };
    let mut stdout = Vec::new();
    if let Some(mut pipe) = child.stdout.take() {
        std::io::Read::read_to_end(&mut pipe, &mut stdout)
            .map_err(|e| ProvError::io(&e, "cannot run ssh-keygen"))?;
    }
    if let Some(handle) = stderr {
        let _ = handle.join();
    }
    if !status.success() {
        return Err(ProvError::value(
            "unencrypted per-instance Ed25519 host key required",
        ));
    }
    let text = String::from_utf8(stdout)
        .map_err(|_| ProvError::new(ProvKind::UnicodeDecode, "cannot decode ssh-keygen output"))?;
    if !text.starts_with("ssh-ed25519 ") {
        return Err(ProvError::value(
            "unencrypted per-instance Ed25519 host key required",
        ));
    }
    Ok(text)
}

/// The script's `dest.parent.resolve() != dest.parent` gate: no symlinked
/// ancestor and no `.`/`..` components (both would resolve away from the
/// given spelling). Missing parents pass here and fail at `stat`, like
/// the script's non-strict resolve.
fn is_real_parent(parent: &Path) -> bool {
    if parent.components().any(|c| {
        matches!(
            c,
            std::path::Component::CurDir | std::path::Component::ParentDir
        )
    }) {
        return false;
    }
    !parent.ancestors().any(|ancestor| {
        std::fs::symlink_metadata(ancestor).is_ok_and(|m| m.file_type().is_symlink())
    })
}

pub(crate) fn write_exclusive(dest: &Path, document: &str) -> Result<(), ProvError> {
    if !dest.is_absolute() {
        return Err(ProvError::value(
            "absolute output under a real private parent required",
        ));
    }
    let parent = dest.parent().unwrap_or(dest);
    if !is_real_parent(parent) {
        return Err(ProvError::value(
            "absolute output under a real private parent required",
        ));
    }
    let mode = {
        #[cfg(unix)]
        {
            use std::os::unix::fs::MetadataExt;
            std::fs::metadata(parent)
                .map_err(|e| ProvError::io(&e, format!("cannot stat {}", parent.display())))?
                .mode()
        }
        #[cfg(not(unix))]
        {
            let _ = std::fs::metadata(parent)
                .map_err(|e| ProvError::io(&e, format!("cannot stat {}", parent.display())))?;
            0u32
        }
    };
    if mode & 0o077 != 0 {
        return Err(ProvError::value("per-instance parent must be private"));
    }
    let mut options = std::fs::OpenOptions::new();
    options.write(true).create_new(true);
    #[cfg(unix)]
    {
        use std::os::unix::fs::OpenOptionsExt;
        options.mode(0o600);
    }
    let mut file = options
        .open(dest)
        .map_err(|e| ProvError::io(&e, format!("cannot create {}", dest.display())))?;
    std::io::Write::write_all(&mut file, document.as_bytes())
        .map_err(|e| ProvError::io(&e, format!("cannot write {}", dest.display())))?;
    std::io::Write::write_all(&mut file, b"\n")
        .map_err(|e| ProvError::io(&e, format!("cannot write {}", dest.display())))?;
    Ok(())
}
