//! Bounded child-process drain used by both CoreOS xz expansion and
//! gpgv verification: piped stdout drained through a bounded writer with
//! a fixed timeout and output limit.

use crate::{io_error, Error};
use std::io::Read;
use std::path::Path;
use std::time::Duration;

/// Runs a command with piped stdout drained through a bounded writer.
pub(super) fn run_bounded(
    name: &str,
    args: &[String],
    timeout: Duration,
    stdout_limit: i64,
    drain: &mut dyn FnMut(&[u8]) -> Result<(), Error>,
) -> Result<(), Error> {
    let mut child = std::process::Command::new(name)
        .args(args)
        .stdin(std::process::Stdio::null())
        .stdout(std::process::Stdio::piped())
        .stderr(std::process::Stdio::null())
        .spawn()
        .map_err(|e| io_error(name, Path::new(name), e))?;
    let mut stdout = child.stdout.take();
    let deadline = std::time::Instant::now() + timeout;
    let mut remaining = stdout_limit;
    let mut buf = [0u8; 32 << 10];
    let mut drained = stdout.is_none();
    let mut timed_out = false;
    // Retires a still-owned direct child on a failed path: SIGKILL where
    // live (harmless if it raced exit), then reap. The caller keeps and
    // returns the initiating error unchanged.
    let retire = |child: &mut std::process::Child| {
        let _ = child.kill();
        let _ = child.wait();
    };
    while !drained {
        if std::time::Instant::now() > deadline {
            timed_out = true;
            break;
        }
        let exited = match child.try_wait() {
            Ok(exited) => exited,
            Err(e) => {
                retire(&mut child);
                return Err(io_error(name, Path::new(name), e));
            }
        };
        match exited {
            Some(_) => {
                // Process exited; drain the rest of the pipe.
                if let Some(out) = stdout.as_mut() {
                    use std::os::unix::io::AsRawFd;
                    set_nonblocking(out.as_raw_fd(), true);
                    loop {
                        match out.read(&mut buf) {
                            Ok(0) => break,
                            Ok(n) => {
                                if n as i64 > remaining {
                                    return Err(Error::msg("input exceeds size limit"));
                                }
                                drain(&buf[..n])?;
                                remaining -= n as i64;
                            }
                            Err(e) if e.kind() == std::io::ErrorKind::WouldBlock => break,
                            Err(e) => return Err(Error::msg(e.to_string())),
                        }
                    }
                }
                drained = true;
            }
            None => {
                if let Some(out) = stdout.as_mut() {
                    use std::os::unix::io::AsRawFd;
                    set_nonblocking(out.as_raw_fd(), true);
                    match out.read(&mut buf) {
                        Ok(0) => std::thread::sleep(Duration::from_millis(5)),
                        Ok(n) => {
                            if n as i64 > remaining {
                                retire(&mut child);
                                return Err(Error::msg("input exceeds size limit"));
                            }
                            if let Err(e) = drain(&buf[..n]) {
                                retire(&mut child);
                                return Err(e);
                            }
                            remaining -= n as i64;
                        }
                        Err(e) if e.kind() == std::io::ErrorKind::WouldBlock => {
                            std::thread::sleep(Duration::from_millis(5))
                        }
                        Err(e) => {
                            retire(&mut child);
                            return Err(Error::msg(e.to_string()));
                        }
                    }
                }
            }
        }
    }
    if timed_out {
        let _ = child.kill();
        let _ = child.wait();
        return Err(Error::msg(format!("{name} timed out")));
    }
    let status = child
        .wait()
        .map_err(|e| io_error(name, Path::new(name), e))?;
    if !status.success() {
        return Err(Error::msg(format!("{name} failed")));
    }
    Ok(())
}

fn set_nonblocking(fd: std::os::unix::io::RawFd, nonblocking: bool) {
    unsafe {
        let flags = libc::fcntl(fd, libc::F_GETFL);
        if flags < 0 {
            return;
        }
        let flags = if nonblocking {
            flags | libc::O_NONBLOCK
        } else {
            flags & !libc::O_NONBLOCK
        };
        libc::fcntl(fd, libc::F_SETFL, flags);
    }
}
