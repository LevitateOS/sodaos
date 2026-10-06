use std::os::unix::io::RawFd;

use super::MusePeer;

// ---------- launch socket (muse_socket_linux.go) ----------

/// `SO_PEERPIDFD` (Linux 6.5+), absent from libc: Go uses the literal too.
const SO_PEERPIDFD: i32 = 77;

/// `musePeer`: kernel-attested pid/uid/gid plus a pinning pidfd.
/// Fails closed when the kernel lacks `SO_PEERPIDFD`.
pub fn muse_peer_from_fd(fd: RawFd) -> Result<MusePeer, String> {
    // SAFETY: getsockopt with correctly sized outputs.
    unsafe {
        let mut cred: libc::ucred = std::mem::zeroed();
        let mut cred_len = std::mem::size_of::<libc::ucred>() as libc::socklen_t;
        if libc::getsockopt(
            fd,
            libc::SOL_SOCKET,
            libc::SO_PEERCRED,
            &mut cred as *mut _ as *mut libc::c_void,
            &mut cred_len,
        ) != 0
        {
            return Err("muse peer credentials unavailable".to_string());
        }
        let mut pidfd: libc::c_int = -1;
        let mut pidfd_len = std::mem::size_of::<libc::c_int>() as libc::socklen_t;
        if libc::getsockopt(
            fd,
            libc::SOL_SOCKET,
            SO_PEERPIDFD,
            &mut pidfd as *mut _ as *mut libc::c_void,
            &mut pidfd_len,
        ) != 0
        {
            return Err("muse peer pidfd unavailable".to_string());
        }
        Ok(MusePeer {
            pid: cred.pid,
            uid: cred.uid,
            gid: cred.gid,
            pidfd,
        })
    }
}

pub(in crate::muse) fn close_fds(fds: &[RawFd]) {
    for fd in fds {
        // SAFETY: fds came from SCM_RIGHTS; double close is avoided by
        // construction (each fd closed exactly once).
        unsafe {
            libc::close(*fd);
        }
    }
}

/// Parse `SCM_RIGHTS` fds out of one control buffer (glibc `CMSG_NXTHDR`
/// iteration, alignment included).
pub fn parse_unix_rights(control: &[u8], controllen: usize) -> Result<Vec<RawFd>, String> {
    let mut fds = Vec::new();
    if controllen == 0 {
        return Ok(fds);
    }
    // SAFETY: manual cmsg iteration over the kernel-filled prefix.
    unsafe {
        let mhdr = libc::msghdr {
            msg_name: std::ptr::null_mut(),
            msg_namelen: 0,
            msg_iov: std::ptr::null_mut(),
            msg_iovlen: 0,
            msg_control: control.as_ptr() as *mut libc::c_void,
            msg_controllen: controllen as _,
            msg_flags: 0,
        };
        let align = std::mem::size_of::<usize>();
        let mut cmsg = libc::CMSG_FIRSTHDR(&mhdr);
        while !cmsg.is_null() {
            if (*cmsg).cmsg_level == libc::SOL_SOCKET && (*cmsg).cmsg_type == libc::SCM_RIGHTS {
                let data = libc::CMSG_DATA(cmsg);
                let end = (cmsg as *const u8).add((*cmsg).cmsg_len as usize);
                let mut cursor = data;
                while (cursor as *const u8).add(std::mem::size_of::<libc::c_int>()) <= end {
                    fds.push(*(cursor as *const libc::c_int));
                    cursor = cursor.add(std::mem::size_of::<libc::c_int>());
                }
            }
            let next = (cmsg as usize + ((*cmsg).cmsg_len as usize).div_ceil(align) * align)
                as *const libc::cmsghdr;
            let limit = (mhdr.msg_control as usize + mhdr.msg_controllen as usize) as *const u8;
            if (next as *const u8).add(std::mem::size_of::<libc::cmsghdr>()) > limit {
                break;
            }
            cmsg = next as *mut libc::cmsghdr;
        }
    }
    Ok(fds)
}
