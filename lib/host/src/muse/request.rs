use std::io::IoSliceMut;
use std::mem::MaybeUninit;
use std::os::fd::{OwnedFd, RawFd};

use super::LaunchRequest;

/// One launch request plus its stdio files (`museRequest`).
/// `files` holds `None`s when the caller passed no descriptors (register).
#[derive(Debug)]
pub struct MuseRequest {
    pub request: LaunchRequest,
    pub files: [Option<std::fs::File>; 3],
}

/// `museRequest` over an accepted SOCK_SEQPACKET fd: one 64KB datagram plus
/// up to 3 stdio descriptors, with the established 5s read budget.
pub fn muse_request_from_fd(fd: RawFd) -> Result<MuseRequest, String> {
    let mut body = vec![0u8; 65536];
    // Space for four descriptors ensures one excess descriptor is visible and
    // rejected; a larger packet sets CTRUNC, and rustix owns/closes all fds it
    // can expose from that truncated control buffer.
    let mut control_space = [MaybeUninit::uninit(); rustix::cmsg_space!(ScmRights(4))];
    let mut ancillary = rustix::net::RecvAncillaryBuffer::new(&mut control_space);
    // SAFETY: the accepted connection remains open while this call runs.
    let socket = unsafe { std::os::fd::BorrowedFd::borrow_raw(fd) };
    let mut pfd = libc::pollfd {
        fd,
        events: libc::POLLIN,
        revents: 0,
    };
    // Preserve the existing five-second readiness deadline.
    // SAFETY: valid one-element pollfd array.
    let ready = unsafe { libc::poll(&mut pfd, 1, 5000) };
    if ready <= 0 {
        return Err("muse launch request unreadable".to_string());
    }

    let mut io = [IoSliceMut::new(&mut body)];
    let received = rustix::net::recvmsg(
        socket,
        &mut io,
        &mut ancillary,
        rustix::net::RecvFlags::CMSG_CLOEXEC,
    )
    .map_err(|_| "muse launch request unreadable".to_string())?;

    let mut files = Vec::<OwnedFd>::new();
    let mut unexpected_control = false;
    for message in ancillary.drain() {
        match message {
            rustix::net::RecvAncillaryMessage::ScmRights(rights) => files.extend(rights),
            _ => unexpected_control = true,
        }
    }
    if received
        .flags
        .intersects(rustix::net::ReturnFlags::TRUNC | rustix::net::ReturnFlags::CTRUNC)
        || unexpected_control
        || !matches!(files.len(), 0 | 3)
    {
        return Err("invalid launch descriptors".to_string());
    }

    let request = LaunchRequest::decode(&body[..received.bytes])?;
    let mut stdio: [Option<std::fs::File>; 3] = [None, None, None];
    if files.len() == 3 {
        for (slot, fd) in stdio.iter_mut().zip(files) {
            *slot = Some(std::fs::File::from(fd));
        }
    }
    if !muse_descriptors_valid(&request, &stdio) {
        return Err("invalid launch descriptors".to_string());
    }
    request.validate()?;
    Ok(MuseRequest {
        request,
        files: stdio,
    })
}

/// `museDescriptorsValid`: register carries no stdio, launches carry all three.
pub fn muse_descriptors_valid(request: &LaunchRequest, files: &[Option<std::fs::File>; 3]) -> bool {
    if request.register.is_some() {
        files.iter().all(Option::is_none)
    } else {
        files.iter().all(Option::is_some)
    }
}
