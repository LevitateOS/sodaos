use std::os::unix::io::RawFd;

use super::LaunchControl;
use crate::domain;
use crate::terminal::{self, Binding};

/// `museResize`: PTY resize over the stdin fd.
pub fn muse_resize(stdin_fd: RawFd, control: &LaunchControl) -> Result<(), String> {
    if control.signal != 0 || control.cols == 0 || control.rows == 0 {
        return Err(terminal::err_denied());
    }
    let ws = libc::winsize {
        ws_row: control.rows,
        ws_col: control.cols,
        ws_xpixel: 0,
        ws_ypixel: 0,
    };
    // SAFETY: ioctl with a valid winsize pointer.
    let result = unsafe { libc::ioctl(stdin_fd, libc::TIOCSWINSZ, &ws) };
    if result < 0 {
        return Err(terminal::err_denied());
    }
    Ok(())
}

/// `stateContainer`: execution state owner (child for nested runs).
pub fn state_container(binding: &Binding) -> Result<String, String> {
    if !terminal::valid_terminal_id(&binding.id) || binding.uid < 0 || binding.gid < 0 {
        return Err(terminal::err_denied());
    }
    let mut container = binding.project.clone();
    if binding.scope == "muse-project" && !binding.child_id.is_empty() {
        container = binding.child_id.clone();
    }
    if !domain::valid_container_id(&container) {
        return Err(terminal::err_denied());
    }
    Ok(container)
}

/// Daemon `OpenMuseListener` filesystem setup: create the interface
/// directory, require it empty, require the socket path absent. Returns
/// the directory path for the caller to bind.
pub fn prepare_muse_listener_dir(socket_path: &str) -> Result<(), String> {
    let dir = match socket_path.rfind('/') {
        Some(0) => "/",
        Some(i) => &socket_path[..i],
        None => ".",
    };
    std::fs::create_dir_all(dir)
        .map_err(|e| format!("muse interface directory unavailable: {e}"))?;
    // Go checks `ReadDir` success plus emptiness with one error.
    let empty = std::fs::read_dir(dir)
        .map(|mut entries| entries.next().is_none())
        .unwrap_or(false);
    if !empty {
        return Err(
            "muse interface directory must be empty before launch service startup".to_string(),
        );
    }
    match std::fs::symlink_metadata(socket_path) {
        Err(e) if e.kind() == std::io::ErrorKind::NotFound => Ok(()),
        _ => Err("muse launch socket is occupied".to_string()),
    }
}
