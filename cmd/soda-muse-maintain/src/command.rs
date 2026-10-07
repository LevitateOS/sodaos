use std::fs;
use std::io::Read;
use std::os::unix::io::{AsRawFd, FromRawFd, RawFd};
use std::process::{Command, Stdio};
use std::time::{Duration, Instant};

use super::archive::feed_archive;

pub(crate) fn podman(args: &[&str], deadline: Instant) -> Result<Vec<u8>, String> {
    let mut cmd = Command::new("podman");
    cmd.arg("--remote=false");
    for a in args {
        cmd.arg(a);
    }
    cmd.stdin(Stdio::null());
    cmd.stdout(Stdio::piped());
    cmd.stderr(Stdio::null());
    let child = cmd
        .spawn()
        .map_err(|_| String::from("project maintenance command failed"))?;
    wait_output(child, deadline)
}

// podman_streamed feeds the local tar archive through an interruptible
// socketpair while podman consumes it. The owner joins that feeder on every
// return path; a podman failure wins over the archive result.
pub(crate) fn podman_streamed(
    feeds: Vec<(String, RawFd, u64)>,
    args: &[&str],
    deadline: Instant,
) -> Result<(), String> {
    let mut fds = [0; 2];
    // A socket pair lets the owner interrupt a feeder blocked in write(2)
    // after the child exits or its deadline expires.
    if unsafe {
        libc::socketpair(
            libc::AF_UNIX,
            libc::SOCK_STREAM | libc::SOCK_CLOEXEC,
            0,
            fds.as_mut_ptr(),
        )
    } != 0
    {
        return Err(String::from("project maintenance command failed"));
    }
    let writer = unsafe { fs::File::from_raw_fd(fds[1]) };
    let reader = unsafe { fs::File::from_raw_fd(fds[0]) };
    let interrupt_fd = unsafe { libc::fcntl(writer.as_raw_fd(), libc::F_DUPFD_CLOEXEC, 0) };
    if interrupt_fd < 0 {
        return Err(String::from("project maintenance command failed"));
    }
    // `interrupt_fd` belongs to the caller and stays open while it waits for
    // the child. The feeder needs a separate owner that half-closes the same
    // socket endpoint as soon as archive production ends, including unwind.
    let finish_fd = unsafe { libc::fcntl(writer.as_raw_fd(), libc::F_DUPFD_CLOEXEC, 0) };
    if finish_fd < 0 {
        unsafe {
            libc::shutdown(interrupt_fd, libc::SHUT_WR);
            libc::close(interrupt_fd);
        }
        return Err(String::from("project maintenance command failed"));
    }
    let flags = unsafe { libc::fcntl(writer.as_raw_fd(), libc::F_GETFL) };
    if flags < 0
        || unsafe { libc::fcntl(writer.as_raw_fd(), libc::F_SETFL, flags | libc::O_NONBLOCK) } < 0
    {
        unsafe {
            libc::shutdown(interrupt_fd, libc::SHUT_WR);
            libc::close(interrupt_fd);
            libc::close(finish_fd);
        }
        return Err(String::from("project maintenance command failed"));
    }
    let feeder_thread = match std::thread::Builder::new().spawn(move || {
        let _finish = WriteHalfOnDrop(finish_fd);
        feed_archive(writer, &feeds, deadline)
    }) {
        Ok(thread) => thread,
        Err(_) => {
            unsafe {
                libc::shutdown(interrupt_fd, libc::SHUT_WR);
                libc::close(interrupt_fd);
                libc::close(finish_fd);
            }
            return Err(String::from("project maintenance command failed"));
        }
    };
    let feeder = FeederOwner {
        interrupt_fd: Some(interrupt_fd),
        thread: Some(feeder_thread),
    };
    let mut cmd = Command::new("podman");
    cmd.arg("--remote=false");
    for a in args {
        cmd.arg(a);
    }
    cmd.stdin(Stdio::from(reader));
    cmd.stdout(Stdio::piped());
    cmd.stderr(Stdio::null());
    let child = match cmd.spawn() {
        Ok(child) => child,
        Err(_) => {
            let _ = feeder.finish();
            return Err(String::from("project maintenance command failed"));
        }
    };
    let output = wait_output(child, deadline);
    let fed = feeder.finish();
    output?;
    fed
}

struct WriteHalfOnDrop(RawFd);

impl Drop for WriteHalfOnDrop {
    fn drop(&mut self) {
        unsafe {
            libc::shutdown(self.0, libc::SHUT_WR);
            libc::close(self.0);
        }
    }
}

struct FeederOwner {
    interrupt_fd: Option<RawFd>,
    thread: Option<std::thread::JoinHandle<Result<(), String>>>,
}

impl FeederOwner {
    fn interrupt(&mut self) {
        if let Some(fd) = self.interrupt_fd.take() {
            unsafe {
                libc::shutdown(fd, libc::SHUT_WR);
                libc::close(fd);
            }
        }
    }

    fn finish(mut self) -> Result<(), String> {
        self.interrupt();
        self.thread
            .take()
            .and_then(|thread| thread.join().ok())
            .unwrap_or_else(|| Err(String::from("project maintenance command failed")))
    }
}

impl Drop for FeederOwner {
    fn drop(&mut self) {
        self.interrupt();
        if let Some(thread) = self.thread.take() {
            let _ = thread.join();
        }
    }
}

pub(crate) fn wait_output(
    mut child: std::process::Child,
    deadline: Instant,
) -> Result<Vec<u8>, String> {
    // Nonblocking reads share the wait loop, so a descendant retaining
    // stdout cannot strand an unowned reader after the direct child exits.
    let failed = String::from("project maintenance command failed");
    let mut stdout = child.stdout.take();
    if let Some(pipe) = stdout.as_ref() {
        let flags = unsafe { libc::fcntl(pipe.as_raw_fd(), libc::F_GETFL) };
        if flags < 0
            || unsafe { libc::fcntl(pipe.as_raw_fd(), libc::F_SETFL, flags | libc::O_NONBLOCK) } < 0
        {
            let _ = child.kill();
            let _ = child.wait();
            return Err(failed);
        }
    }
    let mut out = Vec::new();
    let mut child_status = None;
    let mut exited_at = None;
    let mut buf = [0u8; 65536];
    loop {
        let now = Instant::now();
        if now >= deadline {
            let _ = child.kill();
            let _ = child.wait();
            return Err(failed);
        }
        if child_status.is_none() {
            match child.try_wait() {
                Ok(Some(status)) => {
                    child_status = Some(status);
                    exited_at = Some(now);
                }
                Ok(None) => {}
                Err(_) => {
                    let _ = child.kill();
                    let _ = child.wait();
                    return Err(failed);
                }
            }
        }
        let mut eof = stdout.is_none();
        let mut close_stdout = false;
        if let Some(pipe) = stdout.as_mut() {
            match pipe.read(&mut buf) {
                Ok(0) => {
                    eof = true;
                    close_stdout = true;
                }
                Ok(n) => {
                    const OUTPUT_LIMIT: usize = 8 << 20;
                    if out.len().saturating_add(n) > OUTPUT_LIMIT {
                        let _ = child.kill();
                        let _ = child.wait();
                        return Err(failed);
                    }
                    out.extend_from_slice(&buf[..n]);
                }
                Err(e) if e.kind() == std::io::ErrorKind::WouldBlock => {}
                Err(e) if e.kind() == std::io::ErrorKind::Interrupted => {}
                Err(_) => {
                    let _ = child.kill();
                    let _ = child.wait();
                    return Err(failed);
                }
            }
        }
        if close_stdout {
            stdout = None;
        }
        if let Some(status) = child_status {
            if eof {
                if !status.success() {
                    return Err(failed);
                }
                return Ok(out);
            }
            if exited_at.is_some_and(|at| now.duration_since(at) >= Duration::from_secs(2)) {
                stdout.take();
                return Err(failed);
            }
        }
        std::thread::sleep(Duration::from_millis(5));
    }
}

#[cfg(test)]
mod custody_tests {
    use super::*;
    use std::io::Write;

    fn child(script: &str) -> std::process::Child {
        Command::new("sh")
            .args(["-c", script])
            .stdin(Stdio::null())
            .stdout(Stdio::piped())
            .stderr(Stdio::null())
            .spawn()
            .unwrap()
    }

    #[test]
    fn continuous_output_cannot_starve_deadline() {
        let deadline = Instant::now() + Duration::from_millis(100);
        assert_eq!(
            wait_output(child("while :; do printf x; done"), deadline).unwrap_err(),
            "project maintenance command failed"
        );
    }

    #[test]
    fn exited_child_with_inherited_pipe_is_failure() {
        let started = Instant::now();
        let deadline = started + Duration::from_secs(5);
        assert_eq!(
            wait_output(child("sleep 2.5 & exit 0"), deadline).unwrap_err(),
            "project maintenance command failed"
        );
        assert!(started.elapsed() < Duration::from_secs(4));
    }

    #[test]
    fn silent_child_is_killed_at_absolute_deadline() {
        let deadline = Instant::now() + Duration::from_millis(100);
        assert_eq!(
            wait_output(child("exec sleep 30"), deadline).unwrap_err(),
            "project maintenance command failed"
        );
    }

    #[test]
    fn feeder_shutdown_interrupts_blocked_write_and_joins() {
        let mut fds = [0; 2];
        assert_eq!(
            unsafe {
                libc::socketpair(
                    libc::AF_UNIX,
                    libc::SOCK_STREAM | libc::SOCK_CLOEXEC,
                    0,
                    fds.as_mut_ptr(),
                )
            },
            0
        );
        let mut writer = unsafe { fs::File::from_raw_fd(fds[1]) };
        let _reader = unsafe { fs::File::from_raw_fd(fds[0]) };
        let interrupt_fd = unsafe { libc::fcntl(writer.as_raw_fd(), libc::F_DUPFD_CLOEXEC, 0) };
        assert!(interrupt_fd >= 0);
        let (started, start_seen) = std::sync::mpsc::channel();
        let thread = std::thread::spawn(move || {
            started.send(()).unwrap();
            writer
                .write_all(&vec![0u8; 4 * 1024 * 1024])
                .map_err(|_| String::from("blocked feeder interrupted"))
        });
        let feeder = FeederOwner {
            interrupt_fd: Some(interrupt_fd),
            thread: Some(thread),
        };
        start_seen.recv_timeout(Duration::from_secs(1)).unwrap();
        assert_eq!(feeder.finish().unwrap_err(), "blocked feeder interrupted");
    }
}
