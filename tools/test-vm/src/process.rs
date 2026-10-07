use std::ffi::OsStr;
use std::fs;
use std::io;
use std::io::Read;
use std::os::unix::io::AsRawFd;
use std::os::unix::process::CommandExt;
use std::process::{Child, Command, Stdio};
use std::time::{Duration, Instant};

use super::{faccessat, flush_stdout, status_code, Exit, AT_FDCWD, FAIL_PREFIX};

pub(super) fn os_error(err: &io::Error) -> String {
    err.to_string()
}

pub(super) fn access(path: &str, mode: i32) -> bool {
    match std::ffi::CString::new(path) {
        Ok(c) => unsafe { faccessat(AT_FDCWD, c.as_ptr(), mode, 0) == 0 },
        Err(_) => false,
    }
}

pub(super) fn is_file(path: &str) -> bool {
    fs::metadata(path).map(|m| m.is_file()).unwrap_or(false)
}

pub(super) enum Captured {
    SpawnFailed,
    CaptureFailed,
    Done(Vec<u8>),
}

pub(super) fn capture(prog: &str, args: &[&str], stderr_null: bool) -> Captured {
    flush_stdout();
    let mut cmd = Command::new(prog);
    cmd.args(args).stdout(Stdio::piped());
    if stderr_null {
        cmd.stderr(Stdio::null());
    }
    match cmd.spawn() {
        Ok(mut child) => match capture_stdout(&mut child) {
            Ok(bytes) => Captured::Done(bytes),
            Err(()) => Captured::CaptureFailed,
        },
        Err(err) => {
            if !stderr_null {
                eprintln!("{FAIL_PREFIX}: {prog}: {err}");
            }
            Captured::SpawnFailed
        }
    }
}

/// Drain stdout without letting an oversized command allocate without bound.
/// The caller intentionally ignores native exit status, matching `Output`'s
/// prior behavior while retaining its captured bytes.
fn capture_stdout(child: &mut Child) -> Result<Vec<u8>, ()> {
    const CAP: usize = 8 << 20;
    const PIPE_GRACE: Duration = Duration::from_secs(2);
    let mut stdout = child.stdout.take().expect("piped stdout");
    let fd = stdout.as_raw_fd();
    let flags = unsafe { libc::fcntl(fd, libc::F_GETFL) };
    if flags < 0 || unsafe { libc::fcntl(fd, libc::F_SETFL, flags | libc::O_NONBLOCK) } < 0 {
        let _ = child.kill();
        let _ = child.wait();
        return Err(());
    }
    let mut out = Vec::new();
    let mut buf = [0u8; 65536];
    let mut exited_at = None;
    let mut eof = false;
    let mut wait_failed = false;
    loop {
        match child.try_wait() {
            Ok(Some(_)) => {
                exited_at.get_or_insert_with(Instant::now);
            }
            Ok(None) => {}
            Err(_) => {
                let _ = child.kill();
                let _ = child.wait();
                wait_failed = true;
                break;
            }
        }
        match stdout.read(&mut buf) {
            Ok(0) => eof = true,
            Ok(n) => {
                if out.len().saturating_add(n) > CAP {
                    let _ = child.kill();
                    let _ = child.wait();
                    return Err(());
                }
                out.extend_from_slice(&buf[..n]);
            }
            Err(e) if e.kind() == io::ErrorKind::WouldBlock => {}
            Err(e) if e.kind() == io::ErrorKind::Interrupted => {}
            Err(_) => {
                let _ = child.kill();
                let _ = child.wait();
                return Err(());
            }
        }
        if exited_at.is_some() && eof {
            break;
        }
        if exited_at.is_some_and(|at| at.elapsed() >= PIPE_GRACE) {
            drop(stdout);
            return Err(());
        }
        std::thread::sleep(Duration::from_millis(5));
    }
    // If polling failed while the direct child still runs, own and reap it.
    if child.try_wait().ok().flatten().is_none() {
        let _ = child.kill();
        let _ = child.wait();
    }
    if wait_failed {
        Err(())
    } else {
        Ok(out)
    }
}

pub(super) fn run(prog: &str, args: &[&str]) -> Result<(), Exit> {
    flush_stdout();
    let status = match Command::new(prog).args(args).status() {
        Ok(status) => status,
        Err(err) => {
            let code = if err.kind() == io::ErrorKind::NotFound {
                127
            } else {
                126
            };
            eprintln!("{FAIL_PREFIX}: {prog}: {err}");
            return Err(Exit::Propagate(code));
        }
    };
    let code = status_code(status);
    if code == 0 {
        Ok(())
    } else {
        Err(Exit::Propagate(code))
    }
}

/// Replace this process with the requested command, preserving the shell's
/// explicit argv0 and omitting the supplied program from the remaining args.
pub(super) fn exec_replace(prog: &str, args: &[String]) -> ! {
    flush_stdout();
    let mut command = Command::new(prog);
    command.arg0(OsStr::new(prog)).args(args.iter().skip(1));
    let err = command.exec();
    let code = if err.kind() == io::ErrorKind::NotFound {
        127
    } else {
        126
    };
    eprintln!("{FAIL_PREFIX}: {prog}: {err}");
    std::process::exit(code);
}

#[cfg(test)]
mod capture_tests {
    use super::*;

    #[test]
    fn infinite_output_fails_at_capture_bound() {
        assert!(matches!(capture("yes", &[], true), Captured::CaptureFailed));
    }

    #[test]
    fn descendant_pipe_is_a_capture_failure() {
        assert!(matches!(
            capture("sh", &["-c", "sleep 2.5 & exit 0"], true),
            Captured::CaptureFailed
        ));
    }
}
