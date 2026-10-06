use std::io;

use crate::proto;
use crate::sys;

pub(crate) fn set_nonblocking(fd: i32, nonblocking: bool) -> io::Result<()> {
    let flags = unsafe { libc::fcntl(fd, libc::F_GETFL) };
    if flags < 0 {
        return Err(io::Error::last_os_error());
    }
    let updated = if nonblocking {
        flags | libc::O_NONBLOCK
    } else {
        flags & !libc::O_NONBLOCK
    };
    if unsafe { libc::fcntl(fd, libc::F_SETFL, updated) } != 0 {
        return Err(io::Error::last_os_error());
    }
    Ok(())
}

pub(crate) fn is_eintr(err: &io::Error) -> bool {
    err.raw_os_error() == Some(libc::EINTR)
}

pub(crate) fn read_fd(fd: i32, buf: &mut [u8]) -> io::Result<usize> {
    loop {
        let got = unsafe { libc::read(fd, buf.as_mut_ptr() as *mut libc::c_void, buf.len()) };
        if got < 0 {
            let err = io::Error::last_os_error();
            if is_eintr(&err) {
                continue;
            }
            return Err(err);
        }
        return Ok(got as usize);
    }
}

pub(crate) fn write_fd(fd: i32, buf: &[u8]) -> io::Result<usize> {
    loop {
        let got = unsafe { libc::write(fd, buf.as_ptr() as *const libc::c_void, buf.len()) };
        if got < 0 {
            let err = io::Error::last_os_error();
            if is_eintr(&err) {
                continue;
            }
            return Err(err);
        }
        return Ok(got as usize);
    }
}

pub(crate) fn write_all_blocking(fd: i32, mut buf: &[u8]) -> io::Result<()> {
    while !buf.is_empty() {
        let wrote = write_fd(fd, buf)?;
        if wrote == 0 {
            return Err(io::Error::other("short write"));
        }
        buf = &buf[wrote..];
    }
    Ok(())
}

/// `select` with the `.py` backpressure sets; returns `(readable, writable)`.
pub fn pty_select(
    master: i32,
    to_pty_len: usize,
    outgoing_len: usize,
    attaching: bool,
    timeout: f64,
) -> io::Result<(Vec<i32>, Vec<i32>)> {
    let mut read_set: libc::fd_set = unsafe { std::mem::zeroed() };
    let mut write_set: libc::fd_set = unsafe { std::mem::zeroed() };
    unsafe {
        libc::FD_ZERO(&mut read_set);
        libc::FD_ZERO(&mut write_set);
    }
    let mut watch_read = Vec::new();
    let mut watch_write = Vec::new();
    if to_pty_len <= proto::QUEUE_LIMIT - 16384 {
        unsafe {
            libc::FD_SET(0, &mut read_set);
        }
        watch_read.push(0);
    }
    if outgoing_len <= proto::QUEUE_LIMIT - 8192 {
        unsafe {
            libc::FD_SET(master, &mut read_set);
        }
        watch_read.push(master);
    }
    if outgoing_len > 0 {
        unsafe {
            libc::FD_SET(1, &mut write_set);
        }
        watch_write.push(1);
    }
    if to_pty_len > 0 && !attaching {
        unsafe {
            libc::FD_SET(master, &mut write_set);
        }
        watch_write.push(master);
    }
    let clamped = timeout.max(0.0);
    let mut wait = libc::timeval {
        tv_sec: clamped.floor() as libc::time_t,
        tv_usec: ((clamped - clamped.floor()) * 1_000_000.0) as libc::suseconds_t,
    };
    let top = watch_read
        .iter()
        .chain(watch_write.iter())
        .copied()
        .max()
        .unwrap_or(0)
        + 1;
    let rc = unsafe {
        libc::select(
            top,
            &mut read_set,
            &mut write_set,
            std::ptr::null_mut(),
            &mut wait,
        )
    };
    if rc < 0 {
        return Err(io::Error::last_os_error());
    }
    let readable = watch_read
        .into_iter()
        .filter(|fd| unsafe { libc::FD_ISSET(*fd, &read_set) })
        .collect();
    let writable = watch_write
        .into_iter()
        .filter(|fd| unsafe { libc::FD_ISSET(*fd, &write_set) })
        .collect();
    Ok((readable, writable))
}

pub(crate) fn wait_readable(fd: i32, timeout: f64) -> io::Result<bool> {
    let until = sys::monotonic() + timeout;
    loop {
        let remaining = until - sys::monotonic();
        if remaining <= 0.0 {
            return Ok(false);
        }
        let mut read_set: libc::fd_set = unsafe { std::mem::zeroed() };
        unsafe {
            libc::FD_ZERO(&mut read_set);
            libc::FD_SET(fd, &mut read_set);
        }
        let mut wait = libc::timeval {
            tv_sec: remaining.floor() as libc::time_t,
            tv_usec: ((remaining - remaining.floor()) * 1_000_000.0) as libc::suseconds_t,
        };
        let rc = unsafe {
            libc::select(
                fd + 1,
                &mut read_set,
                std::ptr::null_mut(),
                std::ptr::null_mut(),
                &mut wait,
            )
        };
        if rc < 0 {
            let err = io::Error::last_os_error();
            if is_eintr(&err) {
                continue;
            }
            return Err(err);
        }
        return Ok(rc > 0);
    }
}
