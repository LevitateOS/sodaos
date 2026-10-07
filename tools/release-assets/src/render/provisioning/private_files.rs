//! Bounded private-input, host-key and exclusive-output handling.

use std::io::Read;
use std::os::unix::fs::OpenOptionsExt;
use std::os::unix::io::AsRawFd;
use std::path::{Path, PathBuf};
use std::process::Stdio;
use std::time::{Duration, Instant};

use super::{ProvError, ProvKind};

pub(crate) fn read_text(path: &Path) -> Result<String, ProvError> {
    let mut options = std::fs::OpenOptions::new();
    options
        .read(true)
        .custom_flags(libc::O_CLOEXEC | libc::O_NONBLOCK);
    let file = options
        .open(path)
        .map_err(|e| ProvError::io(&e, format!("cannot read {}", path.display())))?;
    read_open_text(file, path, None)
}

pub(super) fn read_open_text(
    file: std::fs::File,
    path: &Path,
    limit: Option<u64>,
) -> Result<String, ProvError> {
    let mut bytes = Vec::new();
    let mut reader: Box<dyn Read> = match limit {
        Some(limit) => Box::new(file.take(limit + 1)),
        None => Box::new(file),
    };
    reader
        .read_to_end(&mut bytes)
        .map_err(|e| ProvError::io(&e, format!("cannot read {}", path.display())))?;
    if limit.is_some_and(|limit| bytes.len() as u64 > limit) {
        return Err(ProvError::value("bounded regular input required"));
    }
    String::from_utf8(bytes).map_err(|_| {
        ProvError::new(
            ProvKind::UnicodeDecode,
            format!("cannot decode {}", path.display()),
        )
    })
}

/// Bounded regular input, like the script's `regular()`: lstat must show a
/// plain file under 1 MiB, and secret inputs must hide from group/others.
pub(crate) fn regular(path: &Path, private: bool) -> Result<String, ProvError> {
    let mut options = std::fs::OpenOptions::new();
    options
        .read(true)
        .custom_flags(libc::O_CLOEXEC | libc::O_NOFOLLOW | libc::O_NONBLOCK);
    let file = options.open(path).map_err(|e| {
        if e.raw_os_error() == Some(libc::ELOOP) {
            ProvError::value("bounded regular input required")
        } else {
            ProvError::io(&e, format!("cannot stat {}", path.display()))
        }
    })?;
    let meta = file
        .metadata()
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
    read_open_text(file, path, Some(1024 * 1024))
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
    let deadline = Instant::now() + Duration::from_secs(10);
    let mut stdout_pipe = child.stdout.take();
    let mut stderr_pipe = child.stderr.take();
    if let Err(error) =
        set_nonblocking(stdout_pipe.as_ref()).and_then(|()| set_nonblocking(stderr_pipe.as_ref()))
    {
        let _ = child.kill();
        let _ = child.wait();
        return Err(ProvError::io(&error, "cannot run ssh-keygen"));
    }
    let mut stdout = Vec::new();
    const PUBLIC_LIMIT: usize = 1024 * 1024;
    let mut buf = [0u8; 8192];
    let mut child_status = None;
    let mut exited_at = None;
    let mut stdout_eof = stdout_pipe.is_none();
    let mut stderr_eof = stderr_pipe.is_none();
    let status = loop {
        let now = Instant::now();
        if now >= deadline {
            let _ = child.kill();
            let _ = child.wait();
            return Err(ProvError::new(ProvKind::Timeout, "ssh-keygen timed out"));
        }
        if child_status.is_none() {
            match child.try_wait() {
                Ok(Some(done)) => {
                    child_status = Some(done);
                    exited_at = Some(now);
                }
                Ok(None) => {}
                Err(e) => {
                    let _ = child.kill();
                    let _ = child.wait();
                    return Err(ProvError::io(&e, "cannot run ssh-keygen"));
                }
            }
        }
        let mut close_stdout = false;
        if let Some(pipe) = stdout_pipe.as_mut() {
            match pipe.read(&mut buf) {
                Ok(0) => close_stdout = true,
                Ok(n) => {
                    if stdout.len().saturating_add(n) > PUBLIC_LIMIT {
                        let _ = child.kill();
                        let _ = child.wait();
                        return Err(ProvError::value(
                            "unencrypted per-instance Ed25519 host key required",
                        ));
                    }
                    stdout.extend_from_slice(&buf[..n]);
                }
                Err(e) if e.kind() == std::io::ErrorKind::WouldBlock => {}
                Err(e) if e.kind() == std::io::ErrorKind::Interrupted => {}
                Err(e) => {
                    let _ = child.kill();
                    let _ = child.wait();
                    return Err(ProvError::io(&e, "cannot run ssh-keygen"));
                }
            }
        }
        if close_stdout {
            stdout_eof = true;
            stdout_pipe = None;
        }
        let mut close_stderr = false;
        if let Some(pipe) = stderr_pipe.as_mut() {
            match pipe.read(&mut buf) {
                Ok(0) => close_stderr = true,
                Ok(_) => {}
                Err(e) if e.kind() == std::io::ErrorKind::WouldBlock => {}
                Err(e) if e.kind() == std::io::ErrorKind::Interrupted => {}
                Err(e) => {
                    let _ = child.kill();
                    let _ = child.wait();
                    return Err(ProvError::io(&e, "cannot run ssh-keygen"));
                }
            }
        }
        if close_stderr {
            stderr_eof = true;
            stderr_pipe = None;
        }
        if let Some(done) = child_status {
            if stdout_eof && stderr_eof {
                break done;
            }
            if exited_at.is_some_and(|at| at.elapsed() >= Duration::from_secs(2)) {
                stdout_pipe.take();
                stderr_pipe.take();
                return Err(ProvError::new(
                    ProvKind::Timeout,
                    "ssh-keygen output pipes did not close",
                ));
            }
        }
        std::thread::sleep(Duration::from_millis(5));
    };
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

fn set_nonblocking<T: AsRawFd>(pipe: Option<&T>) -> Result<(), std::io::Error> {
    let Some(pipe) = pipe else { return Ok(()) };
    let fd = pipe.as_raw_fd();
    let flags = unsafe { libc::fcntl(fd, libc::F_GETFL) };
    if flags < 0 || unsafe { libc::fcntl(fd, libc::F_SETFL, flags | libc::O_NONBLOCK) } < 0 {
        return Err(std::io::Error::last_os_error());
    }
    Ok(())
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
