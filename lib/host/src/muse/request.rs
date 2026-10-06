use std::os::unix::io::{FromRawFd, RawFd};

use super::{close_fds, parse_unix_rights, LaunchRequest};

/// One launch request plus its stdio files (`museRequest`).
/// `files` holds `None`s when the caller passed no descriptors (register).
#[derive(Debug)]
pub struct MuseRequest {
    pub request: LaunchRequest,
    pub files: [Option<std::fs::File>; 3],
}

/// `museRequest` over an already-accepted fd: one 64KB datagram plus up
/// to 3 SCM_RIGHTS descriptors, 5s read budget.
pub fn muse_request_from_fd(fd: RawFd) -> Result<MuseRequest, String> {
    let mut body = vec![0u8; 65536];
    // SAFETY: CMSG_SPACE for 3 ints.
    let cmsg_len = unsafe { libc::CMSG_SPACE(3 * 4) } as usize;
    let mut control = vec![0u8; cmsg_len];
    let (n, controllen, flags) = unsafe {
        let mut iov = libc::iovec {
            iov_base: body.as_mut_ptr() as *mut libc::c_void,
            iov_len: body.len(),
        };
        let mut hdr: libc::msghdr = std::mem::zeroed();
        hdr.msg_iov = &mut iov;
        hdr.msg_iovlen = 1;
        hdr.msg_control = control.as_mut_ptr() as *mut libc::c_void;
        hdr.msg_controllen = control.len() as _;
        // 5s read budget like Go's `SetReadDeadline`.
        let mut pfd = libc::pollfd {
            fd,
            events: libc::POLLIN,
            revents: 0,
        };
        if libc::poll(&mut pfd, 1, 5000) <= 0 {
            return Err("muse launch request unreadable".to_string());
        }
        let n = libc::recvmsg(fd, &mut hdr, 0);
        if n < 0 {
            return Err("muse launch request unreadable".to_string());
        }
        (n as usize, hdr.msg_controllen as usize, hdr.msg_flags)
    };
    let fds = parse_unix_rights(&control, controllen)?;
    if flags & (libc::MSG_TRUNC | libc::MSG_CTRUNC) != 0 || (!fds.is_empty() && fds.len() != 3) {
        close_fds(&fds);
        return Err("invalid launch descriptors".to_string());
    }
    let mut files: [Option<std::fs::File>; 3] = [None, None, None];
    if fds.len() == 3 {
        for (i, fd) in fds.iter().enumerate() {
            // SAFETY: set CLOEXEC on the received fd, then adopt it.
            unsafe {
                let flags = libc::fcntl(*fd, libc::F_GETFD);
                if flags >= 0 {
                    libc::fcntl(*fd, libc::F_SETFD, flags | libc::FD_CLOEXEC);
                }
                files[i] = Some(std::fs::File::from_raw_fd(*fd));
            }
        }
    }
    let request = LaunchRequest::decode(&body[..n])?;
    if !muse_descriptors_valid(&request, &files) {
        return Err("invalid launch descriptors".to_string());
    }
    request.validate()?;
    Ok(MuseRequest { request, files })
}

/// `museDescriptorsValid`: register carries no stdio, launches carry all three.
pub fn muse_descriptors_valid(request: &LaunchRequest, files: &[Option<std::fs::File>; 3]) -> bool {
    if request.register.is_some() {
        return files[0].is_none();
    }
    files[0].is_some() && files[1].is_some() && files[2].is_some()
}
