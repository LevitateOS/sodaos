use std::fs::File;
use std::io;
use std::os::unix::io::AsRawFd;

use crate::fs;
use crate::proto;
use crate::pty;
use crate::sys;

pub const TERMINALS: &str = "/run/soda-terminals";
pub const PROGRAM: &str = "/usr/libexec/soda/project-terminal";

pub(crate) fn s_isreg(mode: u32) -> bool {
    rustix::fs::FileType::from_raw_mode(mode).is_file()
}

pub(crate) fn s_issock(mode: u32) -> bool {
    rustix::fs::FileType::from_raw_mode(mode).is_socket()
}

pub const TMUX_CONFIG: &str = "set -g status off
set -g history-limit 10000
set -s buffer-limit 10
set -s set-clipboard off
set -s escape-time 10
set -g default-terminal screen-256color
set -g update-environment \"\"
set -s exit-unattached off
";

/// Validated terminal locator path.
pub fn terminal_path(identifier: &str) -> Result<String, String> {
    if !proto::valid_identifier(identifier) {
        return Err("invalid terminal identifier".to_string());
    }
    Ok(format!("{TERMINALS}/{identifier}"))
}

/// No-follow descent with the `.py` `root_directory` checks (root-owned,
/// group/other write-free) at every level.
pub(crate) fn checked_chain(top: &str) -> Result<File, String> {
    let mut current = sys::open_root().map_err(|e| format!("open /: {e}"))?;
    for part in top.split('/').filter(|p| !p.is_empty()) {
        let child = sys::open_child_dir(&current, part).map_err(|e| format!("open {part}: {e}"))?;
        drop(current);
        current = child;
        let (uid, mode) = fs::fstat_uid_mode(&current).map_err(|e| format!("stat {part}: {e}"))?;
        if uid != 0 || mode & 0o022 != 0 {
            return Err("unsafe terminal directory".to_string());
        }
    }
    Ok(current)
}

/// Directory entries through the held fd.
pub(crate) fn list_dir_names(dir: &File) -> Result<Vec<String>, String> {
    let mut names = Vec::new();
    let entries = std::fs::read_dir(format!("/proc/self/fd/{}", dir.as_raw_fd()))
        .map_err(|e| e.to_string())?;
    for entry in entries {
        let entry = entry.map_err(|e| e.to_string())?;
        names.push(
            entry
                .file_name()
                .into_string()
                .map_err(|_| "terminal name encoding".to_string())?,
        );
    }
    Ok(names)
}

pub(crate) fn fstatat(dir: &File, name: &str) -> io::Result<rustix::fs::Stat> {
    rustix::fs::statat(dir, name, rustix::fs::AtFlags::SYMLINK_NOFOLLOW).map_err(Into::into)
}

pub(crate) fn record_exists(dir: &File, name: &str) -> Result<bool, String> {
    match fstatat(dir, name) {
        Ok(_) => Ok(true),
        Err(err) if err.kind() == io::ErrorKind::NotFound => Ok(false),
        Err(err) => Err(err.to_string()),
    }
}

fn write_stdout_best_effort(bytes: &[u8]) {
    let mut rest = bytes;
    while !rest.is_empty() {
        let wrote = unsafe { libc::write(1, rest.as_ptr() as *const libc::c_void, rest.len()) };
        if wrote < 0 {
            if io::Error::last_os_error().raw_os_error() == Some(libc::EINTR) {
                continue;
            }
            break;
        }
        if wrote == 0 {
            break;
        }
        rest = &rest[wrote as usize..];
    }
}

pub(crate) fn closed_launch_failed() {
    write_stdout_best_effort(&pty::closed_line("launch_failed"));
}
