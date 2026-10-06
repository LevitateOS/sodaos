use std::io::Read;
#[cfg(unix)]
use std::os::unix::process::CommandExt;
use std::process::{Child, Command, Stdio};
use std::thread::JoinHandle;
use std::time::{Duration, Instant};

use super::{SnapshotFailure, SnapshotKind};

/// Snapshot stdout bound, applied during capture.
const MAX_OUTPUT: usize = 4 * 1024 * 1024;

/// Run a snapshot command with piped output and a timeout, returning
/// stripped stdout. Stderr is drained and discarded, like the Python
/// owner capturing it and never reading it.
///
/// The deadline covers the child plus pipe completion: reader joins are
/// bounded by the same deadline, and the timeout path retires the whole
/// process group (the child leads its own group, like owned `Process`
/// execution) so a descendant retaining pipes cannot hold the call past
/// the deadline. The group kill runs only while the direct child is
/// unreaped, so its PID cannot name a foreign process. Past the cap,
/// stdout keeps draining into the void so an oversized child still exits
/// and reports exceeded-bound instead of wedging into a timeout.
pub fn command_with_timeout(
    argv: &[String],
    extra_env: &[(&str, &str)],
    timeout: Duration,
) -> Result<String, SnapshotFailure> {
    let mut command = Command::new(&argv[0]);
    command
        .args(&argv[1..])
        .envs(extra_env.iter().copied())
        .stdout(Stdio::piped())
        .stderr(Stdio::piped());
    #[cfg(unix)]
    unsafe {
        command.pre_exec(|| {
            if libc::setpgid(0, 0) == 0 {
                Ok(())
            } else {
                Err(std::io::Error::last_os_error())
            }
        });
    }
    let mut child = command.spawn().map_err(SnapshotFailure::io)?;
    // Belt and suspenders with pre_exec: whichever runs first establishes
    // the group, so a later group kill can never name our own group.
    // Errors mean the other side already set it (or the child is gone).
    #[cfg(unix)]
    {
        let _ = unsafe { libc::setpgid(child.id() as i32, child.id() as i32) };
    }
    // Both pipes drain on threads while the parent polls, like
    // `communicate()`: waiting before reading would deadlock once a pipe
    // fills.
    let mut stdout_pipe = child.stdout.take();
    let out_reader = std::thread::spawn(move || {
        let mut stdout = Vec::new();
        let mut total: usize = 0;
        if let Some(pipe) = stdout_pipe.take() {
            let mut reader = std::io::BufReader::new(pipe);
            let mut chunk = [0u8; 32768];
            loop {
                let count = match reader.read(&mut chunk) {
                    Ok(0) => break,
                    Ok(count) => count,
                    Err(_) => break,
                };
                total += count;
                if stdout.len() < MAX_OUTPUT {
                    let room = MAX_OUTPUT - stdout.len();
                    stdout.extend_from_slice(&chunk[..count.min(room)]);
                }
            }
        }
        (stdout, total > MAX_OUTPUT)
    });
    let mut stderr = child.stderr.take();
    let drain = std::thread::spawn(move || {
        if let Some(stderr) = stderr.take() {
            let mut reader = std::io::BufReader::new(stderr);
            let mut sink = [0u8; 32768];
            loop {
                match reader.read(&mut sink) {
                    Ok(0) => break,
                    Ok(_) => continue,
                    Err(_) => break,
                }
            }
        }
    });
    let deadline = Instant::now() + timeout;
    loop {
        match child.try_wait().map_err(SnapshotFailure::io)? {
            Some(status) => {
                // The leader is reaped, so its PID is unpinned: no group
                // kill here. Joins stay bounded by the same deadline; a
                // descendant still holding pipes past it reports Timeout.
                let (stdout, overflow) = match join_before(out_reader, deadline) {
                    Some(done) => done,
                    None => {
                        drop(drain);
                        return Err(SnapshotFailure::bare(SnapshotKind::TimeoutExpired));
                    }
                };
                if join_before(drain, deadline).is_none() {
                    return Err(SnapshotFailure::bare(SnapshotKind::TimeoutExpired));
                }
                if !status.success() {
                    let shown: Vec<&str> = argv.iter().take(5).map(String::as_str).collect();
                    return Err(SnapshotFailure::runtime(format!(
                        "Required snapshot command failed: {}",
                        shown.join(" ")
                    )));
                }
                if overflow {
                    return Err(SnapshotFailure::runtime("Snapshot output exceeded bound"));
                }
                let text = String::from_utf8(stdout)
                    .map_err(|_| SnapshotFailure::bare(SnapshotKind::UnicodeDecodeError))?;
                return Ok(text.trim().to_string());
            }
            None if Instant::now() >= deadline => {
                kill_command_group(&mut child);
                let _ = child.wait();
                drop(out_reader);
                drop(drain);
                return Err(SnapshotFailure::bare(SnapshotKind::TimeoutExpired));
            }
            None => std::thread::sleep(Duration::from_millis(10)),
        }
    }
}

/// Wait for a reader thread up to `deadline`, detaching it on expiry
/// (`None`). A finished thread keeps the previous `unwrap_or_default`
/// panic tolerance.
fn join_before<T: Default>(handle: JoinHandle<T>, deadline: Instant) -> Option<T> {
    while Instant::now() < deadline {
        if handle.is_finished() {
            return Some(handle.join().unwrap_or_default());
        }
        std::thread::sleep(Duration::from_millis(5));
    }
    if handle.is_finished() {
        Some(handle.join().unwrap_or_default())
    } else {
        None
    }
}

/// Retire the command's process group. Runs only while the direct child
/// is unreaped, so the leader PID still pins the group and the signal
/// cannot reach foreign processes. Best-effort cleanup: the direct kill
/// always follows so a lone child never survives a failed group signal.
#[cfg(unix)]
fn kill_command_group(child: &mut Child) {
    let pid = child.id() as i32;
    let _ = unsafe { libc::kill(-pid, libc::SIGKILL) };
    let _ = child.kill();
}

/// Without process groups there is nothing wider to retire; the deadline
/// and bounded joins still hold.
#[cfg(not(unix))]
fn kill_command_group(child: &mut Child) {
    let _ = child.kill();
}

/// Snapshot command with the retired probe's 30s timeout and inherited environment.
pub fn command(argv: &[String], extra_env: &[(&str, &str)]) -> Result<String, SnapshotFailure> {
    command_with_timeout(argv, extra_env, Duration::from_secs(30))
}

/// Split stripped command output into lines. Empty output yields no
/// lines, like the retired probe's `strip().splitlines()`.
pub fn output_lines(text: &str) -> Vec<String> {
    if text.is_empty() {
        return Vec::new();
    }
    text.lines().map(|line| line.to_string()).collect()
}
