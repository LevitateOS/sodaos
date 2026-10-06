use std::fs::File;
use std::io;
use std::os::unix::io::{AsRawFd, FromRawFd};

use crate::account::Account;
use crate::cgroup::cstring;

fn s_issock(mode: u32) -> bool {
    mode & libc::S_IFMT == libc::S_IFSOCK
}

/// Socket check outcome with retry classification for the prepare poll loop:
/// missing/refused endpoints are transient, everything else is fatal.
#[derive(Debug, Clone, PartialEq, Eq)]
pub(crate) enum SocketCheck {
    Retryable(String),
    Fatal(String),
}

pub(crate) fn socket_identity_kinded(
    path: &str,
    account: &Account,
    pid: i32,
) -> Result<(u64, u64), SocketCheck> {
    let target = cstring(path).map_err(|_| SocketCheck::Fatal("bad socket path".to_string()))?;
    let mut info: libc::stat = unsafe { std::mem::zeroed() };
    if unsafe { libc::lstat(target.as_ptr(), &mut info) } != 0 {
        let err = io::Error::last_os_error();
        if err.raw_os_error() == Some(libc::ENOENT) {
            return Err(SocketCheck::Retryable("socket absent".to_string()));
        }
        return Err(SocketCheck::Fatal(format!("stat socket: {err}")));
    }
    if !s_issock(info.st_mode) || info.st_uid != account.pw_uid || info.st_mode & 0o007 != 0 {
        return Err(SocketCheck::Fatal("unsafe tmux socket".to_string()));
    }
    if path.len() >= 108 {
        return Err(SocketCheck::Fatal("socket path length".to_string()));
    }
    let fd = unsafe {
        libc::socket(
            libc::AF_UNIX,
            libc::SOCK_STREAM | libc::SOCK_CLOEXEC | libc::SOCK_NONBLOCK,
            0,
        )
    };
    if fd < 0 {
        return Err(SocketCheck::Fatal(format!(
            "socket: {}",
            io::Error::last_os_error()
        )));
    }
    let peer = unsafe { File::from_raw_fd(fd) };
    let mut addr: libc::sockaddr_un = unsafe { std::mem::zeroed() };
    addr.sun_family = libc::AF_UNIX as libc::sa_family_t;
    for (i, b) in path.bytes().enumerate() {
        addr.sun_path[i] = b as libc::c_char;
    }
    let addr_len =
        (std::mem::offset_of!(libc::sockaddr_un, sun_path) + path.len() + 1) as libc::socklen_t;
    let rc = unsafe {
        libc::connect(
            peer.as_raw_fd(),
            &addr as *const _ as *const libc::sockaddr,
            addr_len,
        )
    };
    if rc != 0 {
        let err = io::Error::last_os_error();
        if err.raw_os_error() != Some(libc::EINPROGRESS) {
            return Err(classify_connect_err(&err));
        }
        // 1s connect budget, like `peer.settimeout(1)`.
        let mut write_set: libc::fd_set = unsafe { std::mem::zeroed() };
        unsafe {
            libc::FD_ZERO(&mut write_set);
            libc::FD_SET(peer.as_raw_fd(), &mut write_set);
        }
        let mut timeout = libc::timeval {
            tv_sec: 1,
            tv_usec: 0,
        };
        let ready = unsafe {
            libc::select(
                peer.as_raw_fd() + 1,
                std::ptr::null_mut(),
                &mut write_set,
                std::ptr::null_mut(),
                &mut timeout,
            )
        };
        if ready <= 0 {
            // Timeout (or select failure): fatal, like the `.py` timeout
            // which is neither FileNotFound nor ConnectionRefused.
            return Err(SocketCheck::Fatal("socket connect timeout".to_string()));
        }
        let mut so_error = 0;
        let mut so_len = std::mem::size_of::<libc::c_int>() as libc::socklen_t;
        if unsafe {
            libc::getsockopt(
                peer.as_raw_fd(),
                libc::SOL_SOCKET,
                libc::SO_ERROR,
                &mut so_error as *mut _ as *mut libc::c_void,
                &mut so_len,
            )
        } != 0
        {
            return Err(SocketCheck::Fatal(format!(
                "getsockopt: {}",
                io::Error::last_os_error()
            )));
        }
        if so_error != 0 {
            return Err(classify_connect_err(&io::Error::from_raw_os_error(
                so_error,
            )));
        }
    }
    let mut cred: libc::ucred = unsafe { std::mem::zeroed() };
    let mut cred_len = std::mem::size_of::<libc::ucred>() as libc::socklen_t;
    if unsafe {
        libc::getsockopt(
            peer.as_raw_fd(),
            libc::SOL_SOCKET,
            libc::SO_PEERCRED,
            &mut cred as *mut _ as *mut libc::c_void,
            &mut cred_len,
        )
    } != 0
    {
        return Err(SocketCheck::Fatal(format!(
            "peercred: {}",
            io::Error::last_os_error()
        )));
    }
    if cred.pid != pid || cred.uid != account.pw_uid {
        return Err(SocketCheck::Fatal("wrong tmux server".to_string()));
    }
    Ok((info.st_dev as u64, info.st_ino as u64))
}

fn classify_connect_err(err: &io::Error) -> SocketCheck {
    match err.raw_os_error() {
        // Mirror `.py`'s `(FileNotFoundError, ConnectionRefusedError)` retry.
        Some(code) if code == libc::ENOENT || code == libc::ECONNREFUSED => {
            SocketCheck::Retryable(format!("socket unavailable: {err}"))
        }
        _ => SocketCheck::Fatal(format!("socket connect: {err}")),
    }
}

/// Verify tmux socket ownership and server identity; returns `(dev, ino)`.
pub fn socket_identity(path: &str, account: &Account, pid: i32) -> Result<(u64, u64), String> {
    socket_identity_kinded(path, account, pid).map_err(|e| match e {
        SocketCheck::Retryable(message) | SocketCheck::Fatal(message) => message,
    })
}
