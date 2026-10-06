use std::io;
use std::os::unix::io::AsRawFd;
use std::os::unix::net::UnixStream;

// -- peer credentials (terminal/muse_socket_linux.go musePeer) --

/// Kernel-attested Unix peer identity from `SO_PEERCRED`.
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub struct PeerCred {
    pub pid: i32,
    pub uid: u32,
    pub gid: u32,
}

/// Muse-launch peer: `SO_PEERCRED` plus a `SO_PEERPIDFD` pin on the
/// original caller, so PID reuse cannot swap the caller mid-flight
/// (Go: `musePeerAlive` polls the pidfd).
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub struct MusePeer {
    pub pid: i32,
    pub uid: u32,
    pub gid: u32,
    /// Owned pidfd; close with `close_pidfd` (Go defers `unix.Close`).
    pub pidfd: i32,
}

/// `SO_PEERPIDFD`, absent from older libc versions; Go passes 77 literally.
#[cfg(target_os = "linux")]
const SO_PEERPIDFD: libc::c_int = 77;

/// Attest the peer of a connected Unix stream. Linux-only like the Go
/// launcher; other platforms report unsupported (the appliance is Linux).
#[cfg(target_os = "linux")]
pub fn peer_cred(stream: &UnixStream) -> io::Result<PeerCred> {
    let mut cred: libc::ucred = unsafe { std::mem::zeroed() };
    let mut len = std::mem::size_of::<libc::ucred>() as libc::socklen_t;
    let rc = unsafe {
        libc::getsockopt(
            stream.as_raw_fd(),
            libc::SOL_SOCKET,
            libc::SO_PEERCRED,
            (&mut cred as *mut libc::ucred).cast::<libc::c_void>(),
            &mut len,
        )
    };
    if rc != 0 {
        return Err(io::Error::last_os_error());
    }
    Ok(PeerCred {
        pid: cred.pid,
        uid: cred.uid,
        gid: cred.gid,
    })
}

#[cfg(not(target_os = "linux"))]
pub fn peer_cred(_stream: &UnixStream) -> io::Result<PeerCred> {
    Err(io::Error::new(
        io::ErrorKind::Unsupported,
        "SO_PEERCRED is Linux-only",
    ))
}

/// Attest a muse-launch peer (pidfd-pinned). Takes the raw fd like Go's
/// `musePeer`, which runs on the accepted connection before handoff.
#[cfg(target_os = "linux")]
pub fn muse_peer(fd: std::os::unix::io::RawFd) -> io::Result<MusePeer> {
    let mut cred: libc::ucred = unsafe { std::mem::zeroed() };
    let mut len = std::mem::size_of::<libc::ucred>() as libc::socklen_t;
    let rc = unsafe {
        libc::getsockopt(
            fd,
            libc::SOL_SOCKET,
            libc::SO_PEERCRED,
            (&mut cred as *mut libc::ucred).cast::<libc::c_void>(),
            &mut len,
        )
    };
    if rc != 0 {
        return Err(io::Error::last_os_error());
    }
    let mut pidfd: libc::c_int = -1;
    let mut pidfd_len = std::mem::size_of::<libc::c_int>() as libc::socklen_t;
    let rc = unsafe {
        libc::getsockopt(
            fd,
            libc::SOL_SOCKET,
            SO_PEERPIDFD,
            (&mut pidfd as *mut libc::c_int).cast::<libc::c_void>(),
            &mut pidfd_len,
        )
    };
    if rc != 0 || pidfd < 0 {
        return Err(io::Error::last_os_error());
    }
    Ok(MusePeer {
        pid: cred.pid,
        uid: cred.uid,
        gid: cred.gid,
        pidfd,
    })
}

#[cfg(not(target_os = "linux"))]
pub fn muse_peer(_fd: std::os::unix::io::RawFd) -> io::Result<MusePeer> {
    Err(io::Error::new(
        io::ErrorKind::Unsupported,
        "SO_PEERCRED is Linux-only",
    ))
}

/// Release a pidfd obtained from `muse_peer`.
#[cfg(target_os = "linux")]
pub fn close_pidfd(peer: &MusePeer) {
    unsafe {
        libc::close(peer.pidfd);
    }
}

#[cfg(not(target_os = "linux"))]
pub fn close_pidfd(_peer: &MusePeer) {}
