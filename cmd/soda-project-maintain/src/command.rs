use std::fs;
use std::io::Read;
use std::os::unix::io::{AsRawFd, FromRawFd, RawFd};
use std::process::{Command, Stdio};
use std::time::{Duration, Instant};

use super::archive::feed_archive;

pub(crate) fn podman(args: &[&str], deadline: Instant) -> Result<Vec<u8>, String> {
    let mut native_args = vec!["--remote=false"];
    native_args.extend_from_slice(args);
    run("podman", &native_args, deadline)
}

pub(crate) fn run(program: &str, args: &[&str], deadline: Instant) -> Result<Vec<u8>, String> {
    let mut cmd = Command::new(program);
    cmd.args(args);
    cmd.stdin(Stdio::null());
    cmd.stdout(Stdio::piped());
    cmd.stderr(Stdio::null());
    let child = cmd
        .spawn()
        .map_err(|_| String::from("project maintenance command failed"))?;
    wait_output(child, deadline)
}

/// wait_exit_unreaped reports whether one owned child has exited without
/// reaping it: `waitid` with `WNOWAIT` leaves the zombie anchor in place,
/// so the pid cannot be reused while it is still owned. A reaped pid pins
/// nothing, so group-owned execution must never signal a numeric group
/// after `try_wait` or `wait` has released it. The success flag comes from
/// the same observation: only a `CLD_EXITED` child with status zero
/// counts, and the anchor stays unreaped either way.
fn wait_exit_unreaped(pid: u32) -> Result<Option<bool>, String> {
    let failed = String::from("project maintenance command failed");
    loop {
        let mut info: libc::siginfo_t = unsafe { std::mem::zeroed() };
        let r = unsafe {
            libc::waitid(
                libc::P_PID,
                pid as libc::id_t,
                &mut info,
                libc::WEXITED | libc::WNOHANG | libc::WNOWAIT,
            )
        };
        if r == 0 {
            unsafe {
                if info.si_pid() == 0 {
                    return Ok(None);
                }
                return Ok(Some(
                    info.si_code == libc::CLD_EXITED && info.si_status() == 0,
                ));
            }
        }
        let e = std::io::Error::last_os_error();
        if e.kind() == std::io::ErrorKind::Interrupted {
            continue;
        }
        return Err(failed);
    }
}

/// Distinct uncertain-cleanup error: owned execution ended or timed out
/// but cleanup confirmation is missing. It fences like any failure and
/// never releases custody as success; the caller's durable record keeps
/// the operation fenced for attention or retry.
pub(crate) const UNCERTAIN_CLEANUP: &str = "project maintenance command cleanup uncertain";

/// settle_owned_child ends one owned execution after a terminal event
/// (hook, setup, output, timeout or observation failure): it snapshots
/// owned descendants before any signal, signals the owned group while
/// the unreaped leader anchor pins the group id, kills the leader,
/// observes the exit with `WNOWAIT` until the caller deadline, confirms
/// descendant retirement with the owned retirement logic while the
/// anchor is still held, then reaps exactly once. The anchor is
/// established first: a reaped or otherwise lost anchor fences with an
/// error and no signal, since the numeric id may already be reused.
/// Signals are requests, never proof: only the observed exit plus
/// descendant retirement confirmed inside the existing bound followed
/// by the single reap confirms cleanup. Members alive in the snapshot
/// with the bound already spent, or retirement the bound outlives, are
/// uncertain cleanup rather than confirmed cleanup, and report the
/// distinct uncertain error; every other outcome keeps the plain
/// operation failure the existing timeout contract pins. The caller's
/// durable record keeps custody on every uncertain or error path, and
/// no deadline is extended here. A `child.wait()` on a possibly running
/// child never happens: the reap runs only after the exit is observed,
/// with no further signals after it.
fn settle_owned_child(
    child: &mut std::process::Child,
    group: Option<i32>,
    deadline: Instant,
) -> Result<(), String> {
    let failed = String::from("project maintenance command failed");
    if wait_exit_unreaped(child.id()).is_err() {
        return Err(failed);
    }
    // Snapshot owned descendants before any signal: observing them
    // after signaling would only ever observe the signals' own effect,
    // which proves nothing about retirement.
    let mut descendants = false;
    if let Some(group) = group {
        if group > 1 {
            descendants = group_executing_members(group).is_ok_and(|members| !members.is_empty());
        }
    }
    if let Some(group) = group {
        if group > 1 {
            let _ = unsafe { libc::killpg(group, libc::SIGKILL) };
        }
    }
    let _ = child.kill();
    loop {
        match wait_exit_unreaped(child.id()) {
            Ok(Some(_)) => break,
            Ok(None) => {
                if Instant::now() >= deadline {
                    // Bound spent with the leader still running:
                    // descendants alive in the pre-signal snapshot can
                    // never be confirmed inside it.
                    if descendants {
                        return Err(String::from(UNCERTAIN_CLEANUP));
                    }
                    return Err(failed);
                }
                std::thread::sleep(Duration::from_millis(5));
            }
            Err(_) => return Err(failed),
        }
    }
    // The leader exit is observed: confirm descendant retirement with
    // the owned retirement logic while the unreaped anchor still pins
    // the group id. Either way the observed-exited child is reaped
    // next so the anchor is never dropped.
    let bound_spent = Instant::now() >= deadline;
    let mut retired = true;
    if let Some(group) = group {
        if group > 1 {
            retired = retire_group(group, deadline).is_ok();
        }
    }
    let reaped = child.wait().map(|_| ()).map_err(|_| failed.clone());
    if reaped.is_err() {
        return Err(failed);
    }
    if !retired || (bound_spent && descendants) {
        return Err(String::from(UNCERTAIN_CLEANUP));
    }
    Ok(())
}

/// group_executing_members lists one process group's still-executing
/// members by scanning `/proc`: entries in zombie or dead states execute
/// nothing, so only live members fence retirement. Only entries that
/// vanished may be skipped; incomplete or unreadable observations fail closed.
pub(crate) fn group_executing_members(group: i32) -> Result<Vec<i32>, String> {
    group_executing_members_in(group, std::path::Path::new("/proc"))
}

fn group_executing_members_in(group: i32, proc_root: &std::path::Path) -> Result<Vec<i32>, String> {
    let failed = String::from("project maintenance command failed");
    let mut members = Vec::new();
    let proc = fs::read_dir(proc_root).map_err(|_| failed.clone())?;
    for entry in proc {
        let entry = entry.map_err(|_| failed.clone())?;
        let pid: i32 = match entry.file_name().to_str().and_then(|s| s.parse().ok()) {
            Some(pid) => pid,
            None => continue,
        };
        let stat = match fs::read_to_string(proc_root.join(pid.to_string()).join("stat")) {
            Ok(stat) => stat,
            Err(e) if e.kind() == std::io::ErrorKind::NotFound => continue,
            Err(_) => return Err(failed),
        };
        let after = match stat.rsplit_once(") ") {
            Some((_, rest)) => rest,
            None => return Err(failed),
        };
        let mut fields = after.split_whitespace();
        let state = fields.next().ok_or_else(|| failed.clone())?;
        if state.len() != 1 {
            return Err(failed);
        }
        fields.next().ok_or_else(|| failed.clone())?;
        let pgrp: i32 = fields
            .next()
            .and_then(|s| s.parse().ok())
            .ok_or_else(|| failed.clone())?;
        if pgrp == group && !matches!(state, "Z" | "X" | "x") {
            members.push(pid);
        }
    }
    members.sort_unstable();
    Ok(members)
}

/// retire_group verifies no owned descendant executes after leader exit,
/// while the leader's zombie anchor is still owned: the anchor pins the
/// pid, so the group id cannot have been reused and signaling it is
/// legitimate. An empty group retires silently; survivors are signaled
/// and re-verified until the caller deadline, and anything still alive
/// fences the result instead of releasing custody. Groups 0 and 1 are
/// never signaled: 0 names our own group and 1 names init.
pub(crate) fn retire_group(group: i32, deadline: Instant) -> Result<(), String> {
    let failed = String::from("project maintenance command failed");
    if group <= 1 {
        return Err(failed);
    }
    loop {
        if group_executing_members(group)?.is_empty() {
            return Ok(());
        }
        if Instant::now() >= deadline {
            return Err(failed);
        }
        unsafe {
            libc::killpg(group, libc::SIGKILL);
        }
        std::thread::sleep(Duration::from_millis(5));
    }
}

// podman_streamed feeds the local tar archive through an interruptible
// socketpair while podman consumes it. The owner joins that feeder on every
// return path; a podman failure wins over the archive result.
pub(crate) fn podman_streamed(
    feeds: Vec<(String, RawFd, u64)>,
    args: &[&str],
    deadline: Instant,
) -> Result<(), String> {
    let mut native_args = vec!["--remote=false"];
    native_args.extend_from_slice(args);
    streamed(feeds, "podman", &native_args, deadline)
}

pub(crate) fn streamed(
    feeds: Vec<(String, RawFd, u64)>,
    program: &str,
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
    let mut cmd = Command::new(program);
    cmd.args(args);
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
    child: std::process::Child,
    deadline: Instant,
) -> Result<Vec<u8>, String> {
    wait_output_owned(child, deadline, None)
}

fn settle_owned_err(
    child: &mut std::process::Child,
    group: Option<i32>,
    deadline: Instant,
    failed: &str,
) -> String {
    match settle_owned_child(child, group, deadline) {
        Err(e) if e == UNCERTAIN_CLEANUP => e,
        _ => failed.to_owned(),
    }
}

fn mark_owned_nonblocking(
    child: &mut std::process::Child,
    group: Option<i32>,
    deadline: Instant,
    stdout: &Option<std::process::ChildStdout>,
    failed: &str,
) -> Result<(), String> {
    if let Some(pipe) = stdout.as_ref() {
        let flags = unsafe { libc::fcntl(pipe.as_raw_fd(), libc::F_GETFL) };
        if flags < 0
            || unsafe { libc::fcntl(pipe.as_raw_fd(), libc::F_SETFL, flags | libc::O_NONBLOCK) } < 0
        {
            return Err(settle_owned_err(child, group, deadline, failed));
        }
    }
    Ok(())
}

fn poll_owned_exit(
    child: &mut std::process::Child,
    group: Option<i32>,
    deadline: Instant,
    failed: &str,
    now: Instant,
) -> Result<Option<Instant>, String> {
    match wait_exit_unreaped(child.id()) {
        Ok(Some(_)) => Ok(Some(now)),
        Ok(None) => Ok(None),
        Err(_) => Err(settle_owned_err(child, group, deadline, failed)),
    }
}

fn read_owned_stdout(
    child: &mut std::process::Child,
    group: Option<i32>,
    deadline: Instant,
    pipe: &mut std::process::ChildStdout,
    buf: &mut [u8],
    out: &mut Vec<u8>,
    failed: &str,
) -> Result<(bool, bool), String> {
    match pipe.read(buf) {
        Ok(0) => Ok((true, true)),
        Ok(n) => {
            const OUTPUT_LIMIT: usize = 8 << 20;
            if out.len().saturating_add(n) > OUTPUT_LIMIT {
                return Err(settle_owned_err(child, group, deadline, failed));
            }
            out.extend_from_slice(&buf[..n]);
            Ok((false, false))
        }
        Err(e) if e.kind() == std::io::ErrorKind::WouldBlock => Ok((false, false)),
        Err(e) if e.kind() == std::io::ErrorKind::Interrupted => Ok((false, false)),
        Err(_) => Err(settle_owned_err(child, group, deadline, failed)),
    }
}

fn settle_exited_child(
    child: &mut std::process::Child,
    group: Option<i32>,
    deadline: Instant,
    stdout: &mut Option<std::process::ChildStdout>,
    now: Instant,
    exited_at: Option<Instant>,
    eof: bool,
    failed: &str,
) -> Result<bool, String> {
    if eof {
        // The leader exited but stays unreaped: its zombie anchor
        // pins the group id, so retiring the group here signals
        // only owned descendants. Descendants retire before the
        // single final reap below; a retirement failure still
        // reaps so the anchor is never dropped, then fences with
        // an error instead of releasing custody. The final reap
        // is the only reap on this path; after it no numeric
        // group from this execution may be signaled again.
        let retired = group
            .map(|group| retire_group(group, deadline).is_ok())
            .unwrap_or(true);
        let status = child.wait().map_err(|_| failed.to_owned())?;
        if !retired || !status.success() {
            return Err(failed.to_owned());
        }
        return Ok(true);
    }
    if exited_at.is_some_and(|at| now.duration_since(at) >= Duration::from_secs(2)) {
        stdout.take();
        if let Some(group) = group {
            // Best effort while the outcome is already an error: a
            // descendant still holds stdout. Retire while the
            // anchor pins the group, then reap the exited leader
            // so no zombie leaks, and fence.
            let _ = retire_group(group, deadline);
        }
        // The leader is observed exited: this reap is immediate,
        // never a wait on a running child.
        let _ = child.wait();
        return Err(failed.to_owned());
    }
    Ok(false)
}

fn wait_output_owned(
    mut child: std::process::Child,
    deadline: Instant,
    group: Option<i32>,
) -> Result<Vec<u8>, String> {
    // Nonblocking reads share the wait loop, so a descendant retaining
    // stdout cannot strand an unowned reader after the direct child exits.
    let failed = String::from("project maintenance command failed");
    let mut stdout = child.stdout.take();
    mark_owned_nonblocking(&mut child, group, deadline, &stdout, &failed)?;
    let mut out = Vec::new();
    let mut exited = false;
    let mut exited_at = None;
    let mut buf = [0u8; 65536];
    loop {
        let now = Instant::now();
        if now >= deadline {
            return Err(settle_owned_err(&mut child, group, deadline, &failed));
        }
        if !exited {
            exited_at = poll_owned_exit(&mut child, group, deadline, &failed, now)?;
            exited = exited_at.is_some();
        }
        let mut eof = stdout.is_none();
        let mut close_stdout = false;
        if let Some(pipe) = stdout.as_mut() {
            (eof, close_stdout) = read_owned_stdout(
                &mut child, group, deadline, pipe, &mut buf, &mut out, &failed,
            )?;
        }
        if close_stdout {
            stdout = None;
        }
        if exited
            && settle_exited_child(
                &mut child,
                group,
                deadline,
                &mut stdout,
                now,
                exited_at,
                eof,
                &failed,
            )?
        {
            return Ok(out);
        }
        std::thread::sleep(Duration::from_millis(5));
    }
}
