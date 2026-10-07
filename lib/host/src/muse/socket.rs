use std::os::fd::{FromRawFd, OwnedFd, RawFd};

use super::MusePeer;

/// `SO_PEERPIDFD` (Linux 6.5+), absent from rustix: Go uses the literal too.
const SO_PEERPIDFD: i32 = 77;

/// Attest and pin the peer on the original Unix socket connection.
///
/// The one libc call is retained because rustix 1.1.5 has no typed
/// `SO_PEERPIDFD` option. In particular, this must not become `pidfd_open`
/// against the returned PID, which would lose the original-connection pin.
pub fn muse_peer_from_fd(fd: RawFd) -> Result<MusePeer, String> {
    // SAFETY: the caller guarantees `fd` remains open for this call.
    let socket = unsafe { std::os::fd::BorrowedFd::borrow_raw(fd) };
    let cred = rustix::net::sockopt::socket_peercred(socket)
        .map_err(|_| "muse peer credentials unavailable".to_string())?;

    let mut pidfd: libc::c_int = -1;
    let mut pidfd_len = std::mem::size_of::<libc::c_int>() as libc::socklen_t;
    // SAFETY: getsockopt writes one descriptor-sized integer to valid storage.
    let result = unsafe {
        libc::getsockopt(
            fd,
            libc::SOL_SOCKET,
            SO_PEERPIDFD,
            (&mut pidfd as *mut libc::c_int).cast(),
            &mut pidfd_len,
        )
    };
    if result != 0 || pidfd < 0 || pidfd_len as usize != std::mem::size_of::<libc::c_int>() {
        if result == 0 && pidfd >= 0 {
            // SAFETY: the kernel returned this descriptor, but its result was malformed.
            unsafe { libc::close(pidfd) };
        }
        return Err("muse peer pidfd unavailable".to_string());
    }
    // SAFETY: successful SO_PEERPIDFD transfers this fresh descriptor to us.
    let pidfd = unsafe { OwnedFd::from_raw_fd(pidfd) };
    Ok(MusePeer {
        pid: cred.pid.as_raw_pid(),
        uid: cred.uid.as_raw(),
        gid: cred.gid.as_raw(),
        pidfd,
    })
}
