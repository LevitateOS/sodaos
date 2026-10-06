use crate::errors::Error;

pub fn enrollment_peer_unit(cgroup: &[u8]) -> Result<Vec<u8>, Error> {
    const PREFIX: &[u8] = b"0::/system.slice/";
    let value = cgroup.strip_suffix(b"\n").unwrap_or(cgroup);
    if !value.starts_with(PREFIX) || value.iter().any(|b| *b == b'\r' || *b == b'\n') {
        return Err(Error::msg("dedicated enrollment service peer required"));
    }
    let unit = &value[PREFIX.len()..];
    if unit.contains(&b'/') {
        return Err(Error::msg("dedicated enrollment service peer required"));
    }
    Ok(unit.to_vec())
}

pub fn enrollment_receiver_unit(unit: &[u8]) -> bool {
    const PREFIX: &[u8] = b"soda-key-enrollment@";
    const SUFFIX: &[u8] = b".service";
    unit.len() > PREFIX.len() + SUFFIX.len()
        && unit.starts_with(PREFIX)
        && unit.ends_with(SUFFIX)
        && !unit
            .iter()
            .any(|b| matches!(b, b'/' | b'\r' | b'\n' | b' ' | b'\t'))
}

// Kernel credentials and the peer's native service cgroup distinguish the fixed
// broker and socket-activated receiver instances from ordinary root SSH shells.
pub fn enrollment_connection_unit(
    stream: &std::os::unix::net::UnixStream,
) -> Result<Vec<u8>, Error> {
    use std::os::unix::io::AsRawFd;
    let mut ucred: libc::ucred = unsafe { std::mem::zeroed() };
    let mut len = std::mem::size_of::<libc::ucred>() as libc::socklen_t;
    if unsafe {
        libc::getsockopt(
            stream.as_raw_fd(),
            libc::SOL_SOCKET,
            libc::SO_PEERCRED,
            &mut ucred as *mut libc::ucred as *mut libc::c_void,
            &mut len,
        )
    } != 0
    {
        let errno = unsafe { *libc::__errno_location() };
        return Err(crate::errors::os_error(std::io::Error::from_raw_os_error(
            errno,
        )));
    }
    if ucred.uid != 0 || ucred.pid <= 0 {
        return Err(Error::msg("dedicated native root enrollment peer required"));
    }
    let group =
        std::fs::read(format!("/proc/{}/cgroup", ucred.pid)).map_err(crate::errors::os_error)?;
    enrollment_peer_unit(&group)
}
