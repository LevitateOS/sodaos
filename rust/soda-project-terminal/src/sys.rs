//! Minimal OS primitives (TEMPORARY: a sibling writes canonical
//! `sys.rs`/`fs.rs`/`timex.rs` which replace these files at merge — depend
//! only on the signatures listed in the PR25 brief, never on anything else
//! here).

use std::fs::File;
use std::io;
use std::os::unix::io::{AsRawFd, FromRawFd};
use std::os::unix::process::ExitStatusExt;

/// Monotonic seconds, like `time.monotonic()`.
pub fn monotonic() -> f64 {
    let mut ts = libc::timespec {
        tv_sec: 0,
        tv_nsec: 0,
    };
    // clock_gettime(CLOCK_MONOTONIC) cannot fail on Linux.
    unsafe { libc::clock_gettime(libc::CLOCK_MONOTONIC, &mut ts) };
    ts.tv_sec as f64 + ts.tv_nsec as f64 / 1_000_000_000.0
}

/// Wall-clock seconds, like `int(time.time())`.
pub fn now_secs() -> i64 {
    unsafe { libc::time(std::ptr::null_mut()) as i64 }
}

fn cstring(value: &str) -> io::Result<std::ffi::CString> {
    std::ffi::CString::new(value)
        .map_err(|_| io::Error::new(io::ErrorKind::InvalidInput, "nul byte in argument"))
}

/// Polling wait with a timeout; kills and reaps on expiry.
fn wait_timeout(pid: i32, timeout_secs: u64) -> io::Result<i32> {
    let deadline = monotonic() + timeout_secs as f64;
    let mut status = 0;
    loop {
        let waited = unsafe { libc::waitpid(pid, &mut status, libc::WNOHANG) };
        if waited == pid {
            return Ok(status);
        }
        if waited < 0 {
            let err = io::Error::last_os_error();
            if err.raw_os_error() == Some(libc::EINTR) {
                continue;
            }
            return Err(err);
        }
        if monotonic() >= deadline {
            unsafe {
                libc::kill(pid, libc::SIGKILL);
            }
            // Reap after the kill (blocking now; the child cannot outlive SIGKILL).
            loop {
                let waited = unsafe { libc::waitpid(pid, &mut status, 0) };
                if waited == pid {
                    break;
                }
                if waited < 0 && io::Error::last_os_error().raw_os_error() != Some(libc::EINTR) {
                    break;
                }
            }
            return Err(io::Error::new(io::ErrorKind::TimedOut, "command timed out"));
        }
        std::thread::sleep(std::time::Duration::from_millis(10));
    }
}

fn spawn_forked(argv: &[String], stdout_to: Option<i32>) -> io::Result<i32> {
    if argv.is_empty() {
        return Err(io::Error::new(io::ErrorKind::InvalidInput, "empty argv"));
    }
    let program = cstring(&argv[0])?;
    let args: Vec<std::ffi::CString> =
        argv.iter().map(|a| cstring(a)).collect::<io::Result<_>>()?;
    // Pipes (when used) are created by the caller and passed via stdout_to.
    let pid = unsafe { libc::fork() };
    if pid < 0 {
        return Err(io::Error::last_os_error());
    }
    if pid == 0 {
        unsafe {
            // Child: only async-signal-safe calls below (plus CLOEXEC fds).
            let null_read = libc::open(c"/dev/null".as_ptr(), libc::O_RDONLY);
            let null_write = libc::open(c"/dev/null".as_ptr(), libc::O_WRONLY);
            if null_read < 0 || null_write < 0 {
                libc::_exit(127);
            }
            libc::dup2(null_read, 0);
            match stdout_to {
                Some(fd) => {
                    libc::dup2(fd, 1);
                }
                None => {
                    libc::dup2(null_write, 1);
                }
            }
            libc::dup2(null_write, 2);
            if null_read > 2 {
                libc::close(null_read);
            }
            if null_write > 2 {
                libc::close(null_write);
            }
            let mut pointers: Vec<*const libc::c_char> = args.iter().map(|a| a.as_ptr()).collect();
            pointers.push(std::ptr::null());
            libc::execv(program.as_ptr(), pointers.as_ptr());
            libc::_exit(127);
        }
    }
    Ok(pid)
}

/// Run with stdio to /dev/null; `Ok` only on exit status 0.
pub fn run_checked(argv: &[String], timeout_secs: u64) -> io::Result<()> {
    let pid = spawn_forked(argv, None)?;
    let status = wait_timeout(pid, timeout_secs)?;
    if libc::WIFEXITED(status) && libc::WEXITSTATUS(status) == 0 {
        Ok(())
    } else {
        Err(io::Error::other(format!("command failed: {}", argv[0])))
    }
}

/// Run with stdin/stderr to /dev/null and piped stdout; exit status is NOT
/// checked (mirrors `subprocess.run(check=False, stdout=PIPE)`).
pub fn run_output(argv: &[String], timeout_secs: u64) -> io::Result<std::process::Output> {
    let mut pipe = [0; 2];
    if unsafe { libc::pipe2(pipe.as_mut_ptr(), libc::O_CLOEXEC) } != 0 {
        return Err(io::Error::last_os_error());
    }
    // Nonblocking on the read end only: the child must see a blocking pipe.
    let flags = unsafe { libc::fcntl(pipe[0], libc::F_GETFL) };
    if flags < 0 || unsafe { libc::fcntl(pipe[0], libc::F_SETFL, flags | libc::O_NONBLOCK) } != 0 {
        unsafe {
            libc::close(pipe[0]);
            libc::close(pipe[1]);
        }
        return Err(io::Error::last_os_error());
    }
    let pid = spawn_forked(argv, Some(pipe[1]))?;
    unsafe {
        libc::close(pipe[1]);
    }
    let mut stdout = Vec::new();
    let mut chunk = [0u8; 8192];
    let deadline = monotonic() + timeout_secs as f64;
    let mut status = 0;
    let mut exited = false;
    loop {
        // Drain whatever is available.
        loop {
            let got = unsafe {
                libc::read(
                    pipe[0],
                    chunk.as_mut_ptr() as *mut libc::c_void,
                    chunk.len(),
                )
            };
            if got > 0 {
                stdout.extend_from_slice(&chunk[..got as usize]);
            } else {
                break;
            }
        }
        let waited = unsafe { libc::waitpid(pid, &mut status, libc::WNOHANG) };
        if waited == pid {
            exited = true;
        } else if waited < 0 && io::Error::last_os_error().raw_os_error() != Some(libc::EINTR) {
            unsafe {
                libc::close(pipe[0]);
            }
            return Err(io::Error::last_os_error());
        }
        if exited {
            // Final drain (nonblocking; EOF once writers are gone).
            loop {
                let got = unsafe {
                    libc::read(
                        pipe[0],
                        chunk.as_mut_ptr() as *mut libc::c_void,
                        chunk.len(),
                    )
                };
                if got > 0 {
                    stdout.extend_from_slice(&chunk[..got as usize]);
                } else {
                    break;
                }
            }
            break;
        }
        if monotonic() >= deadline {
            unsafe {
                libc::kill(pid, libc::SIGKILL);
                libc::close(pipe[0]);
            }
            loop {
                let waited = unsafe { libc::waitpid(pid, &mut status, 0) };
                if waited == pid || waited < 0 {
                    break;
                }
            }
            return Err(io::Error::new(io::ErrorKind::TimedOut, "command timed out"));
        }
        std::thread::sleep(std::time::Duration::from_millis(10));
    }
    unsafe {
        libc::close(pipe[0]);
    }
    Ok(std::process::Output {
        status: std::process::ExitStatus::from_raw(status),
        stdout,
        stderr: Vec::new(),
    })
}

/// Open `/` read-only (CLOEXEC).
pub fn open_root() -> io::Result<File> {
    let fd = unsafe {
        libc::open(
            c"/".as_ptr(),
            libc::O_RDONLY | libc::O_DIRECTORY | libc::O_CLOEXEC,
        )
    };
    if fd < 0 {
        return Err(io::Error::last_os_error());
    }
    Ok(unsafe { File::from_raw_fd(fd) })
}

/// Open a child directory without following symlinks (CLOEXEC).
pub fn open_child_dir(dir: &File, name: &str) -> io::Result<File> {
    open_at(
        dir,
        name,
        libc::O_RDONLY | libc::O_DIRECTORY | libc::O_NOFOLLOW,
        0,
    )
}

/// `openat` relative to `dir` (CLOEXEC always added).
pub fn open_at(dir: &File, name: &str, flags: libc::c_int, mode: libc::mode_t) -> io::Result<File> {
    let path = cstring(name)?;
    let fd = unsafe {
        libc::openat(
            dir.as_raw_fd(),
            path.as_ptr(),
            flags | libc::O_CLOEXEC,
            mode,
        )
    };
    if fd < 0 {
        return Err(io::Error::last_os_error());
    }
    Ok(unsafe { File::from_raw_fd(fd) })
}

/// Nonblocking exclusive lock; `Ok(false)` when another holder owns it.
pub fn flock_exclusive_nb(file: &File) -> io::Result<bool> {
    let rc = unsafe { libc::flock(file.as_raw_fd(), libc::LOCK_EX | libc::LOCK_NB) };
    if rc == 0 {
        return Ok(true);
    }
    let err = io::Error::last_os_error();
    if err.raw_os_error() == Some(libc::EWOULDBLOCK) {
        return Ok(false);
    }
    Err(err)
}

/// Blocking shared lock.
pub fn flock_shared(file: &File) -> io::Result<()> {
    flock_retry(file.as_raw_fd(), libc::LOCK_SH)
}

/// Blocking exclusive lock.
pub fn flock_exclusive(file: &File) -> io::Result<()> {
    flock_retry(file.as_raw_fd(), libc::LOCK_EX)
}

fn flock_retry(fd: i32, op: libc::c_int) -> io::Result<()> {
    loop {
        if unsafe { libc::flock(fd, op) } == 0 {
            return Ok(());
        }
        let err = io::Error::last_os_error();
        if err.raw_os_error() != Some(libc::EINTR) {
            return Err(err);
        }
    }
}

/// Kernel random UUID, trimmed (for sibling broker/keys use).
pub fn read_uuid() -> io::Result<String> {
    let text = std::fs::read_to_string("/proc/sys/kernel/random/uuid")?;
    Ok(text.trim().to_string())
}

#[cfg(test)]
mod tests {
    use super::*;
    use std::os::unix::io::AsRawFd;

    fn scratch(name: &str) -> std::path::PathBuf {
        let dir = std::env::temp_dir().join(format!(
            "soda-pt-sys-{}-{}-{}",
            std::process::id(),
            name,
            monotonic().to_bits()
        ));
        std::fs::create_dir_all(&dir).unwrap();
        dir
    }

    #[test]
    fn clocks_move() {
        let a = monotonic();
        std::thread::sleep(std::time::Duration::from_millis(5));
        assert!(monotonic() > a);
        let now = now_secs();
        assert!(now > 1_700_000_000 && now < 9_000_000_000);
    }

    #[test]
    fn checked_status() {
        assert!(run_checked(&["/usr/bin/true".to_string()], 5).is_ok());
        assert!(run_checked(&["/usr/bin/false".to_string()], 5).is_err());
        assert!(run_checked(&[], 5).is_err());
        let err = run_checked(&["/usr/bin/sleep".to_string(), "30".to_string()], 1).unwrap_err();
        assert_eq!(err.kind(), io::ErrorKind::TimedOut);
    }

    #[test]
    fn output_capture() {
        let out = run_output(&["/usr/bin/echo".to_string(), "hi".to_string()], 5).unwrap();
        assert!(out.status.success());
        assert_eq!(out.stdout, b"hi\n");
        let out = run_output(&["/usr/bin/false".to_string()], 5).unwrap();
        assert!(!out.status.success());
        // Large output past the pipe buffer still drains (no wait-before-read deadlock).
        let out = run_output(
            &[
                "/usr/bin/head".to_string(),
                "-c".to_string(),
                "200000".to_string(),
                "/dev/zero".to_string(),
            ],
            10,
        )
        .unwrap();
        assert_eq!(out.stdout.len(), 200000);
    }

    #[test]
    fn dir_chain_and_locks() {
        let dir = scratch("chain");
        std::fs::create_dir_all(dir.join("a").join("b")).unwrap();
        std::fs::write(dir.join("a").join("f"), b"x").unwrap();
        let root = open_root().unwrap();
        // Walk the scratch path components under /.
        let mut current = root;
        for part in dir.strip_prefix("/").unwrap().components() {
            let next = open_child_dir(&current, part.as_os_str().to_str().unwrap()).unwrap();
            current = next;
        }
        let file = open_at(
            &current,
            "a",
            libc::O_RDONLY | libc::O_DIRECTORY | libc::O_NOFOLLOW,
            0,
        )
        .unwrap();
        assert!(file.as_raw_fd() >= 0);
        let lock = open_at(&current, "a", libc::O_RDONLY | libc::O_DIRECTORY, 0).unwrap();
        flock_shared(&lock).unwrap();
        flock_exclusive(&lock).unwrap();
        assert!(flock_exclusive_nb(&lock).unwrap());
        std::fs::remove_dir_all(&dir).unwrap();
    }

    #[test]
    fn uuid_shape() {
        let uuid = read_uuid().unwrap();
        assert_eq!(uuid.len(), 36);
        assert_eq!(uuid.chars().filter(|c| *c == '-').count(), 4);
    }
}
