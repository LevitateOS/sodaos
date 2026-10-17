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

/// Group launch with a post-spawn observation hook. The importer owns
/// Podman children; every return path retires its whole group, including
/// leader exit: success never releases custody while an owned descendant
/// may still execute. The hook runs after
/// the child is spawned in its own group and before the wait loop, so the
/// caller can persist acquisition custody (pgid plus leader identity)
/// while the child handle is still owned: the pid cannot be reused until
/// it is reaped, so the recorded group identity stays pinned.
pub(crate) fn run_group_observed(
    program: &str,
    args: &[&str],
    deadline: Instant,
    observed: impl FnOnce(i32) -> Result<(), String>,
) -> Result<Vec<u8>, String> {
    use std::os::unix::process::CommandExt;
    let mut child = Command::new(program)
        .args(args)
        .stdin(Stdio::null())
        .stdout(Stdio::piped())
        .stderr(Stdio::null())
        .process_group(0)
        .spawn()
        .map_err(|_| "project maintenance command failed")?;
    let group = child.id() as i32;
    if let Err(hook) = observed(group) {
        // Owned cleanup on launch-persistence failure: the child is still
        // unreaped, so the group id is pinned and safe to signal. The
        // failed persistence fences recovery through the acquired custody
        // left on disk; the child itself is never dropped live and unowned.
        // The shared settle path bounds the cleanup: no unbounded wait on
        // the running child, and the original hook error still surfaces.
        let _ = settle_owned_child(&mut child, Some(group), deadline);
        return Err(hook);
    }
    wait_output_owned(child, deadline, Some(group))
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

/// settle_owned_child ends one owned execution after a terminal event
/// (hook, setup, output, timeout or observation failure): it signals the
/// owned group while the unreaped leader anchor pins the group id, kills
/// the leader, observes the exit with `WNOWAIT` until the caller deadline,
/// then reaps exactly once. The anchor is established first: a reaped or
/// otherwise lost anchor fences with an error and no signal, since the
/// numeric id may already be reused. Signals are requests, never proof:
/// only the observed exit followed by the single reap confirms cleanup.
/// Cleanup unconfirmed within the existing bound fences with an error;
/// the caller's durable record keeps custody, and no deadline is
/// extended here. A `child.wait()` on a possibly running child never
/// happens: the reap runs only after the exit is observed.
fn settle_owned_child(
    child: &mut std::process::Child,
    group: Option<i32>,
    deadline: Instant,
) -> Result<(), String> {
    let failed = String::from("project maintenance command failed");
    if wait_exit_unreaped(child.id()).is_err() {
        return Err(failed);
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
                    return Err(failed);
                }
                std::thread::sleep(Duration::from_millis(5));
            }
            Err(_) => return Err(failed),
        }
    }
    child.wait().map(|_| ()).map_err(|_| failed)
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
fn wait_output_owned(
    mut child: std::process::Child,
    deadline: Instant,
    group: Option<i32>,
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
            let _ = settle_owned_child(&mut child, group, deadline);
            return Err(failed);
        }
    }
    let mut out = Vec::new();
    let mut exited = false;
    let mut exited_at = None;
    let mut buf = [0u8; 65536];
    loop {
        let now = Instant::now();
        if now >= deadline {
            let _ = settle_owned_child(&mut child, group, deadline);
            return Err(failed);
        }
        if !exited {
            match wait_exit_unreaped(child.id()) {
                Ok(Some(_)) => {
                    exited = true;
                    exited_at = Some(now);
                }
                Ok(None) => {}
                Err(_) => {
                    let _ = settle_owned_child(&mut child, group, deadline);
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
                        let _ = settle_owned_child(&mut child, group, deadline);
                        return Err(failed);
                    }
                    out.extend_from_slice(&buf[..n]);
                }
                Err(e) if e.kind() == std::io::ErrorKind::WouldBlock => {}
                Err(e) if e.kind() == std::io::ErrorKind::Interrupted => {}
                Err(_) => {
                    let _ = settle_owned_child(&mut child, group, deadline);
                    return Err(failed);
                }
            }
        }
        if close_stdout {
            stdout = None;
        }
        if exited {
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
                let status = child.wait().map_err(|_| failed.clone())?;
                if !retired || !status.success() {
                    return Err(failed);
                }
                return Ok(out);
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
    fn settle_owned_child_reaps_exited_leader() {
        // The leader already exited: settle confirms without signaling a
        // stranger and reaps exactly once, so no entry remains.
        let mut leader = child("exit 0");
        let pid = leader.id();
        assert!(
            settle_owned_child(&mut leader, None, Instant::now() + Duration::from_secs(5)).is_ok()
        );
        assert!(fs::read_to_string(format!("/proc/{pid}/stat")).is_err());
    }

    #[test]
    fn settle_owned_child_refuses_lost_anchor_without_signaling() {
        // The anchor is already reaped: settle fences with an error and
        // never signals the possibly reused numeric id.
        let mut leader = child("exit 0");
        leader.wait().unwrap();
        assert!(
            settle_owned_child(&mut leader, None, Instant::now() + Duration::from_secs(5)).is_err()
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

#[cfg(test)]
mod acquisition_cancellation_tests {
    use super::*;

    #[test]
    fn group_observation_refuses_unreadable_stat() {
        let dir = crate::test_support::TestDir::make("proc-read-error");
        fs::create_dir_all(dir.0.join("42/stat")).unwrap();
        assert!(group_executing_members_in(42, &dir.0).is_err());
    }

    #[test]
    fn group_observation_refuses_malformed_stat() {
        let dir = crate::test_support::TestDir::make("proc-parse-error");
        fs::create_dir(dir.0.join("42")).unwrap();
        for stat in ["truncated", "42 (worker) S 1 bad", "42 (worker) S"] {
            fs::write(dir.0.join("42/stat"), stat).unwrap();
            assert!(group_executing_members_in(42, &dir.0).is_err(), "{stat}");
        }
    }

    #[test]
    fn group_observation_distinguishes_live_dead_and_vanished_members() {
        let dir = crate::test_support::TestDir::make("proc-members");
        for (pid, stat) in [
            (42, "42 (worker) S 1 42"),
            (43, "43 (dead) Z 1 42"),
            (44, "44 (other) S 1 99"),
        ] {
            fs::create_dir(dir.0.join(pid.to_string())).unwrap();
            fs::write(dir.0.join(pid.to_string()).join("stat"), stat).unwrap();
        }
        fs::create_dir(dir.0.join("45")).unwrap(); // Vanished before its stat read.
        assert_eq!(group_executing_members_in(42, &dir.0).unwrap(), vec![42]);
    }
    #[test]
    fn leader_exit_retires_surviving_descendant_before_success() {
        // The descendant closes stdio and outlives the leader: success
        // must still reap it instead of releasing custody. Pre-fix this
        // returned Ok with the descendant executing.
        let dir = crate::test_support::TestDir::make("importer-successor");
        let path = dir.path("child.pid");
        let out = run_group_observed(
            "/bin/sh",
            &[
                "-c",
                "sleep 30 >&- 2>&- & child=$!; printf '%s' \"$child\" > \"$1\"; exit 0",
                "probe",
                &path,
            ],
            Instant::now() + Duration::from_secs(30),
            |_| Ok(()),
        )
        .expect("leader exit with a surviving descendant must retire the group");
        assert!(out.is_empty());
        let pid = fs::read_to_string(path).unwrap();
        let state = fs::read_to_string(format!("/proc/{pid}/stat"));
        assert!(
            state.is_err() || state.unwrap().rsplit_once(") ").unwrap().1.starts_with('Z'),
            "descendant still executing after success"
        );
    }

    #[test]
    fn retire_group_never_signals_caller_or_init() {
        let deadline = Instant::now() + Duration::from_secs(5);
        assert!(retire_group(0, deadline).is_err());
        assert!(retire_group(1, deadline).is_err());
        assert!(group_executing_members(i32::MAX).unwrap().is_empty());
    }

    #[test]
    fn launch_persistence_failure_cleans_up_the_owned_child() {
        // The injected hook fails after spawn: the owned leader must be
        // killed and reaped by the caller, never dropped live, and the
        // error must surface so custody stays fenced as acquired.
        use std::sync::{Arc, Mutex};
        let seen = Arc::new(Mutex::new(0));
        let probe = seen.clone();
        let err = run_group_observed(
            "/bin/sh",
            &["-c", "exec sleep 30"],
            Instant::now() + Duration::from_secs(30),
            |pgid| {
                *probe.lock().unwrap() = pgid;
                Err(String::from("injected persistence failure"))
            },
        )
        .unwrap_err();
        assert_eq!(err, "injected persistence failure");
        let pgid = *seen.lock().unwrap();
        assert!(pgid > 1);
        // The leader was reaped by owned cleanup: no entry remains.
        assert!(fs::read_to_string(format!("/proc/{pgid}/stat")).is_err());
        assert!(group_executing_members(pgid).unwrap().is_empty());
    }

    #[test]
    fn importer_deadline_stops_descendants_before_returning() {
        let dir = crate::test_support::TestDir::make("importer-cancellation");
        let path = dir.path("child.pid");
        assert!(run_group_observed(
            "/bin/sh",
            &[
                "-c",
                "sleep 30 & child=$!; printf '%s' \"$child\" > \"$1\"; wait",
                "probe",
                &path
            ],
            Instant::now() + Duration::from_millis(300),
            |_| Ok(())
        )
        .is_err());
        let pid = fs::read_to_string(path).unwrap();
        // The error returns at the caller deadline without waiting out
        // the kill, so the signaled descendant may still be dying: poll
        // boundedly until it is gone or reaped elsewhere, and never
        // accept one left executing.
        let stop = Instant::now() + Duration::from_secs(5);
        loop {
            match fs::read_to_string(format!("/proc/{pid}/stat")) {
                Err(_) => break,
                Ok(state) => {
                    let live = !state.rsplit_once(") ").unwrap().1.starts_with('Z');
                    if !live {
                        break;
                    }
                    assert!(Instant::now() < stop, "descendant still executing");
                    std::thread::sleep(Duration::from_millis(5));
                }
            }
        }
    }

    #[test]
    fn run_group_observed_reports_surviving_descendant_and_reaps_leader() {
        // The leader exits at once while a descendant holds stdout: the
        // caller deadline fires first, the error surfaces, the leader is
        // reaped, and the signaled descendant cannot still execute.
        use std::sync::{Arc, Mutex};
        let seen = Arc::new(Mutex::new(0));
        let probe = seen.clone();
        assert!(run_group_observed(
            "/bin/sh",
            &["-c", "sleep 30 &"],
            Instant::now() + Duration::from_secs(1),
            |pgid| {
                *probe.lock().unwrap() = pgid;
                Ok(())
            }
        )
        .is_err());
        let pgid = *seen.lock().unwrap();
        assert!(pgid > 1);
        assert!(fs::read_to_string(format!("/proc/{pgid}/stat")).is_err());
        assert!(retire_group(pgid, Instant::now() + Duration::from_secs(5)).is_ok());
    }

    #[test]
    fn run_group_observed_failed_leader_is_error_and_reaped() {
        // A failed leader with no descendants still fences as an error
        // and leaves no zombie behind.
        use std::sync::{Arc, Mutex};
        let seen = Arc::new(Mutex::new(0));
        let probe = seen.clone();
        assert_eq!(
            run_group_observed(
                "/bin/sh",
                &["-c", "exit 3"],
                Instant::now() + Duration::from_secs(30),
                |pgid| {
                    *probe.lock().unwrap() = pgid;
                    Ok(())
                }
            )
            .unwrap_err(),
            "project maintenance command failed"
        );
        let pgid = *seen.lock().unwrap();
        assert!(pgid > 1);
        assert!(fs::read_to_string(format!("/proc/{pgid}/stat")).is_err());
    }
}
