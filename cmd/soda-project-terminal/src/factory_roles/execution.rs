//! Supervised role execution: one fixed entry point per call as the role
//! with bounded captured output, plus the detached double-fork supervisor
//! so completion never depends on the caller's state lock.

use super::fsx::{hold_json, hold_state};
use super::records::{start_prestate, started_pgid, started_pid};
use crate::account::Account;
use crate::emit::{obj, str_value};
use crate::error::{fail, Error};
use crate::fsx;
use crate::ops_approve::ReqFields;
use crate::ops_inspect;
use crate::proc;
use crate::state_json::StateValue;
use crate::validate;
use std::ffi::CString;
use std::fs::File;
use std::io::Write;
use std::os::unix::ffi::OsStrExt;
use std::os::unix::fs::MetadataExt;
use std::os::unix::io::AsRawFd;
use std::path::Path;
use std::time::{Duration, Instant, SystemTime, UNIX_EPOCH};

/// Fixed child environment, in field order (visible in the child's
/// `/proc/PID/environ`, so the order is pinned).
pub fn child_environ(
    path_value: &str,
    home: &str,
    credential_file: &str,
    role: &str,
    pid: &str,
    snapshot: &str,
) -> Vec<(String, String)> {
    let mut environ = vec![
        ("PATH".to_string(), path_value.to_string()),
        ("HOME".to_string(), home.to_string()),
        ("TMPDIR".to_string(), format!("{home}/tmp")),
        ("SODA_ROLE".to_string(), role.to_string()),
        ("SODA_PREPARATION_ID".to_string(), pid.to_string()),
        ("SODA_SNAPSHOT".to_string(), snapshot.to_string()),
    ];
    if !credential_file.is_empty() {
        environ.push((
            "SODA_CREDENTIAL_FILE".to_string(),
            credential_file.to_string(),
        ));
    }
    environ
}

/// `Path(tool['path']).parent`, pinned for odd inputs (`""` is `"."`,
/// `"/"` stays `"/"`, trailing slashes collapse first).
pub fn parent_of(path: &str) -> String {
    let trimmed = path.trim_end_matches('/');
    if trimmed.is_empty() {
        if path.is_empty() {
            return ".".to_string();
        }
        return "/".to_string();
    }
    match trimmed.rfind('/') {
        None => ".".to_string(),
        Some(0) => "/".to_string(),
        Some(index) => trimmed[..index].to_string(),
    }
}

fn exit_code(status: libc::c_int) -> i32 {
    if libc::WIFEXITED(status) {
        libc::WEXITSTATUS(status)
    } else if libc::WIFSIGNALED(status) {
        -libc::WTERMSIG(status)
    } else {
        127
    }
}

fn wait_child(pid: libc::pid_t) -> libc::c_int {
    let mut status = 0;
    loop {
        let rc = unsafe { libc::waitpid(pid, &mut status, 0) };
        if rc < 0 {
            let errno = std::io::Error::last_os_error().raw_os_error();
            if errno == Some(libc::EINTR) {
                continue;
            }
            return -1;
        }
        return status;
    }
}

/// Test-only log of signals this module sends to role children, so a
/// regression can prove no signal follows ownership loss. Release builds
/// emit the raw kill with no footprint.
#[cfg(test)]
static KILL_LOG: std::sync::Mutex<Vec<(libc::pid_t, libc::c_int)>> =
    std::sync::Mutex::new(Vec::new());

fn signal_child(pid: libc::pid_t, signal: libc::c_int) {
    #[cfg(test)]
    if let Ok(mut log) = KILL_LOG.lock() {
        log.push((pid, signal));
    }
    unsafe {
        libc::kill(pid, signal);
    }
}

/// CODEX-P07-001b: every bounded wait ends in exactly one of these.
/// `Alive` means the deadline expired while the pid was positively still
/// an owned child (escalation authorized); `Gone` means wait established
/// it is no longer an owned child (no signal may follow).
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
enum ReapOutcome {
    Reaped(libc::c_int),
    Alive,
    Gone,
}

const TERM_GRACE_MS: u64 = 1000;
const KILL_GRACE_MS: u64 = 1000;
const POLL_SLICE_MS: u64 = 20;

fn reap_once(pid: libc::pid_t) -> ReapOutcome {
    let mut status = 0;
    loop {
        let rc = unsafe { libc::waitpid(pid, &mut status, libc::WNOHANG) };
        if rc == pid {
            return ReapOutcome::Reaped(status);
        }
        if rc < 0 {
            let errno = std::io::Error::last_os_error().raw_os_error();
            if errno == Some(libc::EINTR) {
                continue;
            }
            // Not a waitable child (already reaped elsewhere): ownership
            // lost, so no signal may be sent.
            return ReapOutcome::Gone;
        }
        return ReapOutcome::Alive;
    }
}

fn reap_until(pid: libc::pid_t, deadline: std::time::Instant) -> ReapOutcome {
    loop {
        match reap_once(pid) {
            ReapOutcome::Alive => {}
            settled => return settled,
        }
        if std::time::Instant::now() >= deadline {
            // The last poll saw a live owned child: still owned at expiry.
            return ReapOutcome::Alive;
        }
        std::thread::sleep(std::time::Duration::from_millis(POLL_SLICE_MS));
    }
}

/// Bounded checked termination of an owned role child. SIGTERM with a
/// grace interval, then SIGKILL with a final
/// reap interval; every wait is `WNOHANG` so even an unkillable
/// (uninterruptible-sleep) child cannot hang the supervisor. SIGKILL is
/// sent only when the grace expired while the child was positively still
/// owned; ownership loss at any point reports `None` with no further
/// signal, so a recycled pid is never touched. Returns the wait status
/// when the child was confirmed reaped, or `None` when termination could
/// not be confirmed (the caller must report uncertainty, never Ok).
fn terminate_child(pid: libc::pid_t) -> Option<libc::c_int> {
    match reap_once(pid) {
        ReapOutcome::Reaped(status) => return Some(status),
        ReapOutcome::Gone => return None,
        ReapOutcome::Alive => {}
    }
    signal_child(pid, libc::SIGTERM);
    match reap_until(
        pid,
        std::time::Instant::now() + std::time::Duration::from_millis(TERM_GRACE_MS),
    ) {
        ReapOutcome::Reaped(status) => return Some(status),
        ReapOutcome::Gone => return None,
        ReapOutcome::Alive => {}
    }
    // The grace expired while the pid was positively still an owned
    // child, so escalation is authorized. Production is a single-child
    // waiter, which keeps this exact: no other reaper can interleave.
    signal_child(pid, libc::SIGKILL);
    match reap_until(
        pid,
        std::time::Instant::now() + std::time::Duration::from_millis(KILL_GRACE_MS),
    ) {
        ReapOutcome::Reaped(status) => Some(status),
        ReapOutcome::Alive | ReapOutcome::Gone => None,
    }
}

fn state_lock(ctx: &crate::Ctx) -> Result<File, Error> {
    let file = fsx::open_ro(&ctx.lock, false)?;
    use std::os::unix::io::AsRawFd;
    if unsafe { libc::flock(file.as_raw_fd(), libc::LOCK_EX) } != 0 {
        return Err(Error::io_msg("factory state lock failed"));
    }
    Ok(file)
}

fn wall_nanos_now() -> i128 {
    match SystemTime::now().duration_since(UNIX_EPOCH) {
        Ok(duration) => duration.as_nanos() as i128,
        Err(error) => -(error.duration().as_nanos() as i128),
    }
}

fn deadline_instant(wire: &str) -> Result<Instant, Error> {
    let deadline = soda_wire_time::parse_nanos(wire)
        .ok_or_else(|| Error::fail("invalid preparation deadline"))?;
    let remaining = deadline - wall_nanos_now();
    if remaining <= 0 {
        return Err(Error::fail("preparation deadline has expired"));
    }
    let nanos =
        u64::try_from(remaining).map_err(|_| Error::fail("preparation deadline too large"))?;
    Instant::now()
        .checked_add(Duration::from_nanos(nanos))
        .ok_or_else(|| Error::fail("preparation deadline too large"))
}

fn active_child_path(directory: &Path) -> std::path::PathBuf {
    directory.join("active-child.json")
}

/// Close a launch gate and boundedly reap its child. Until the gate is
/// released, the child cannot execute or create descendants.
fn abort_gated_child(
    pid: libc::pid_t,
    gate_writer: libc::c_int,
    reader: libc::c_int,
    directory: &Path,
    receipt_written: bool,
) -> Result<(), Error> {
    unsafe {
        libc::close(gate_writer);
        libc::close(reader);
    }
    if terminate_child(pid).is_none() {
        return fail("gated role child termination unconfirmed");
    }
    if receipt_written {
        match std::fs::remove_file(active_child_path(directory)) {
            Ok(()) => {}
            Err(error) if error.kind() == std::io::ErrorKind::NotFound => {}
            Err(error) => return Err(Error::io("remove active child receipt", &error)),
        }
    }
    Ok(())
}

fn active_child_ids(value: &StateValue) -> Result<(i32, i32, u64), Error> {
    let pid = validate::as_i64(value.get("pid").unwrap_or(&StateValue::Null))
        .and_then(|value| i32::try_from(value).ok())
        .ok_or_else(|| Error::fail("unsafe active child metadata"))?;
    let pgid = validate::as_i64(value.get("pgid").unwrap_or(&StateValue::Null))
        .and_then(|value| i32::try_from(value).ok())
        .ok_or_else(|| Error::fail("unsafe active child metadata"))?;
    let start_ticks = validate::as_int_text(value.get("start_ticks").unwrap_or(&StateValue::Null))
        .and_then(|value| value.parse::<u64>().ok())
        .ok_or_else(|| Error::fail("unsafe active child metadata"))?;
    if pid <= 1 || pid != pgid || start_ticks == 0 {
        return fail("unsafe active child metadata");
    }
    Ok((pid, pgid, start_ticks))
}

/// The caller holds the exclusive state lock, so a still-recorded direct
/// child has not been reaped by its supervisor and its pid cannot be reused.
fn pinned_role_group_leader(pid: i32, pgid: i32, start_ticks: u64, uid: u32, proot: &Path) -> bool {
    if pid != pgid {
        return false;
    }
    let entry = proot.join(pid.to_string());
    let (_, group, observed_start) = proc_identity(&entry);
    group.as_deref() == Some(pgid.to_string().as_str())
        && observed_start == Some(start_ticks)
        && std::fs::symlink_metadata(entry).is_ok_and(|metadata| metadata.uid() == uid)
}

fn terminate_role_group(
    pid: i32,
    pgid: i32,
    start_ticks: u64,
    uid: u32,
    proot: &Path,
) -> Result<bool, Error> {
    if probe_group_quiet(pgid, proot) == Some(true) {
        return Ok(true);
    }
    if !pinned_role_group_leader(pid, pgid, start_ticks, uid, proot) {
        return Ok(false);
    }
    match killpg(pgid, libc::SIGTERM) {
        Ok(()) => {}
        Err(errno) if errno == libc::ESRCH => {
            return Ok(probe_group_quiet(pgid, proot) == Some(true));
        }
        Err(errno) => return Err(Error::io_msg(format!("killpg: errno {errno}"))),
    }
    let term_deadline = Instant::now() + Duration::from_millis(TERM_GRACE_MS);
    while Instant::now() < term_deadline {
        match probe_group_quiet(pgid, proot) {
            Some(true) => return Ok(true),
            None => return Ok(false),
            Some(false) => std::thread::sleep(Duration::from_millis(POLL_SLICE_MS)),
        }
    }
    if probe_group_quiet(pgid, proot) == Some(true) {
        return Ok(true);
    }
    if !pinned_role_group_leader(pid, pgid, start_ticks, uid, proot) {
        return Ok(false);
    }
    match killpg(pgid, libc::SIGKILL) {
        Ok(()) => {}
        Err(errno) if errno == libc::ESRCH => {
            return Ok(probe_group_quiet(pgid, proot) == Some(true));
        }
        Err(errno) => return Err(Error::io_msg(format!("killpg: errno {errno}"))),
    }
    let kill_deadline = Instant::now() + Duration::from_millis(KILL_GRACE_MS);
    while Instant::now() < kill_deadline {
        match probe_group_quiet(pgid, proot) {
            Some(true) => return Ok(true),
            None => return Ok(false),
            Some(false) => std::thread::sleep(Duration::from_millis(POLL_SLICE_MS)),
        }
    }
    Ok(probe_group_quiet(pgid, proot) == Some(true))
}

fn finish_role_child(
    ctx: &crate::Ctx,
    directory: &Path,
    pid: libc::pid_t,
    pgid: i32,
) -> Result<libc::c_int, Error> {
    let _lock = state_lock(ctx)?;
    let deadline = Instant::now() + Duration::from_millis(KILL_GRACE_MS);
    loop {
        match probe_group_quiet(pgid, Path::new("/proc")) {
            Some(true) => break,
            Some(false) if Instant::now() < deadline => {
                std::thread::sleep(Duration::from_millis(POLL_SLICE_MS));
            }
            Some(false) | None => return fail("role child retirement unconfirmed"),
        }
    }
    let status = wait_child(pid);
    if status < 0 {
        return Err(Error::io_msg("waitpid failed"));
    }
    match std::fs::remove_file(active_child_path(directory)) {
        Ok(()) => {}
        Err(error) if error.kind() == std::io::ErrorKind::NotFound => {}
        Err(error) => return Err(Error::io("remove active child receipt", &error)),
    }
    Ok(status)
}

fn wait_active_role_join(directory: &Path) -> bool {
    let deadline = Instant::now() + Duration::from_millis(KILL_GRACE_MS);
    while active_child_path(directory).exists() && Instant::now() < deadline {
        std::thread::sleep(Duration::from_millis(POLL_SLICE_MS));
    }
    !active_child_path(directory).exists()
}

fn to_cstring(bytes: &[u8]) -> Option<CString> {
    CString::new(bytes).ok()
}

/// Run one fixed entry point as the role, capturing bounded output. The
/// supervisor stays privileged so role code cannot forge logs or
/// completion. Anything failing in the child exits it with 127.
pub fn run_as_role(
    ctx: &crate::Ctx,
    directory: &Path,
    argv: &[String],
    environ: &[(String, String)],
    workdir: &Path,
    logpath: &Path,
    uid: u32,
    gid: u32,
    deadline: Option<Instant>,
) -> Result<i32, Error> {
    let lock = state_lock(ctx)?;
    if fsx::lexists(&directory.join("stopped.json")) {
        return fail("preparation stopped before role launch");
    }
    if deadline.is_some_and(|deadline| Instant::now() >= deadline) {
        return Ok(124);
    }
    let lock_fd = lock.as_raw_fd();
    let mut fds = [0 as libc::c_int; 2];
    let mut gate = [0 as libc::c_int; 2];
    if unsafe { libc::pipe(fds.as_mut_ptr()) } != 0 {
        return Err(Error::io_msg("pipe failed"));
    }
    if unsafe { libc::pipe(gate.as_mut_ptr()) } != 0 {
        unsafe {
            libc::close(fds[0]);
            libc::close(fds[1]);
        }
        return Err(Error::io_msg("role launch gate failed"));
    }
    let (reader, writer) = (fds[0], fds[1]);
    let (gate_reader, gate_writer) = (gate[0], gate[1]);
    let pid = unsafe { libc::fork() };
    if pid < 0 {
        unsafe {
            libc::close(reader);
            libc::close(writer);
            libc::close(gate_reader);
            libc::close(gate_writer);
        }
        return Err(Error::io_msg("fork failed"));
    }
    if pid == 0 {
        unsafe {
            libc::close(lock_fd);
            libc::close(gate_writer);
            libc::close(reader);
            libc::dup2(writer, 1);
            libc::dup2(writer, 2);
            libc::close(writer);
            let mut release = 0u8;
            loop {
                let count =
                    libc::read(gate_reader, &mut release as *mut u8 as *mut libc::c_void, 1);
                if count < 0 && std::io::Error::last_os_error().raw_os_error() == Some(libc::EINTR)
                {
                    continue;
                }
                libc::close(gate_reader);
                if count != 1 || release != 1 {
                    libc::_exit(127);
                }
                break;
            }
            let dir = match to_cstring(workdir.as_os_str().as_bytes()) {
                Some(dir) => dir,
                None => libc::_exit(127),
            };
            if libc::chdir(dir.as_ptr()) != 0 {
                libc::_exit(127);
            }
            // Drop supplementary groups first so the role keeps exactly
            // its primary group. The privileged supervisor is always
            // root; the call is skipped only for unprivileged test
            // drivers of this function.
            if libc::geteuid() == 0 && libc::setgroups(0, std::ptr::null()) != 0 {
                libc::_exit(127);
            }
            if libc::setgid(gid) != 0 {
                libc::_exit(127);
            }
            if libc::setuid(uid) != 0 {
                libc::_exit(127);
            }
            let mut owned: Vec<CString> = Vec::with_capacity(argv.len() + environ.len());
            for word in argv {
                match to_cstring(word.as_bytes()) {
                    Some(word) => owned.push(word),
                    None => libc::_exit(127),
                }
            }
            let argv_count = owned.len();
            for (key, value) in environ {
                let mut pair = Vec::with_capacity(key.len() + value.len() + 1);
                pair.extend_from_slice(key.as_bytes());
                pair.push(b'=');
                pair.extend_from_slice(value.as_bytes());
                match to_cstring(&pair) {
                    Some(pair) => owned.push(pair),
                    None => libc::_exit(127),
                }
            }
            let mut argv_ptrs: Vec<*const libc::c_char> =
                owned[..argv_count].iter().map(|s| s.as_ptr()).collect();
            argv_ptrs.push(std::ptr::null());
            let mut env_ptrs: Vec<*const libc::c_char> =
                owned[argv_count..].iter().map(|s| s.as_ptr()).collect();
            env_ptrs.push(std::ptr::null());
            libc::execve(argv_ptrs[0], argv_ptrs.as_ptr(), env_ptrs.as_ptr());
            libc::_exit(127);
        }
    }
    unsafe {
        libc::close(writer);
        libc::close(gate_reader);
    }
    let pgid = pid;
    if unsafe { libc::setpgid(pid, pgid) } != 0 {
        abort_gated_child(pid, gate_writer, reader, directory, false)?;
        return Err(Error::io_msg("role child group setup failed"));
    }
    let (child_state, child_group, start_ticks) =
        proc_identity(&Path::new("/proc").join(pid.to_string()));
    let Some(start_ticks) = start_ticks else {
        abort_gated_child(pid, gate_writer, reader, directory, false)?;
        return Err(Error::io_msg("role child identity unavailable"));
    };
    if child_group.as_deref() != Some(pgid.to_string().as_str()) || child_state.is_none() {
        abort_gated_child(pid, gate_writer, reader, directory, false)?;
        return Err(Error::io_msg("role child group setup failed"));
    }
    let active_payload = crate::emit::dumps_default(&obj(vec![
        ("pid", StateValue::Number(pid.to_string())),
        ("pgid", StateValue::Number(pgid.to_string())),
        ("start_ticks", StateValue::Number(start_ticks.to_string())),
    ]));
    if let Err(error) = fsx::write_new(
        &active_child_path(directory),
        active_payload.as_bytes(),
        0o644,
    ) {
        abort_gated_child(pid, gate_writer, reader, directory, false)?;
        return Err(error);
    }
    let mut log = match File::create(logpath) {
        Ok(log) => log,
        Err(error) => {
            abort_gated_child(pid, gate_writer, reader, directory, true)?;
            drop(lock);
            return Err(Error::io("create setup log", &error));
        }
    };
    let flags = unsafe { libc::fcntl(reader, libc::F_GETFL) };
    if flags < 0 || unsafe { libc::fcntl(reader, libc::F_SETFL, flags | libc::O_NONBLOCK) } != 0 {
        abort_gated_child(pid, gate_writer, reader, directory, true)?;
        drop(lock);
        return Err(Error::io_msg("role output pipe setup failed"));
    }
    if deadline.is_some_and(|deadline| Instant::now() >= deadline) {
        abort_gated_child(pid, gate_writer, reader, directory, true)?;
        drop(lock);
        return Ok(124);
    }
    let release = 1u8;
    if unsafe { libc::write(gate_writer, &release as *const u8 as *const libc::c_void, 1) } != 1 {
        abort_gated_child(pid, gate_writer, reader, directory, true)?;
        drop(lock);
        return Err(Error::io_msg("role launch gate release failed"));
    }
    unsafe { libc::close(gate_writer) };
    drop(lock);
    // P07-F1/001: log-I/O failures must still close owned descriptors
    // and terminate/reap the role child within a bound before reporting;
    // unconfirmed cleanup retains uncertainty instead of the log error.
    let mut kept = 0usize;
    let mut eof = false;
    let mut expired = false;
    let mut capture_join_deadline = None;
    let mut descendants_found = false;
    loop {
        let mut chunk = [0u8; 65536];
        let got =
            unsafe { libc::read(reader, chunk.as_mut_ptr() as *mut libc::c_void, chunk.len()) };
        if got < 0 {
            let error = std::io::Error::last_os_error();
            if error.raw_os_error() == Some(libc::EINTR) {
                continue;
            } else if error.kind() == std::io::ErrorKind::WouldBlock {
                // The same loop observes the deadline and child group.
            } else {
                unsafe { libc::close(reader) };
                let cleaned =
                    terminate_role_group(pid, pgid, start_ticks, uid, Path::new("/proc"))?;
                if !cleaned {
                    return Err(Error::fail("role child cleanup unconfirmed"));
                }
                finish_role_child(ctx, directory, pid, pgid)?;
                return Err(Error::io_msg("log pipe failed"));
            }
        } else if got == 0 {
            eof = true;
        } else if kept < crate::LOG_CAP {
            let take = (got as usize).min(crate::LOG_CAP - kept);
            if let Err(err) = log.write_all(&chunk[..take]) {
                unsafe { libc::close(reader) };
                let cleaned =
                    terminate_role_group(pid, pgid, start_ticks, uid, Path::new("/proc"))?;
                if !cleaned {
                    return Err(Error::fail("role child cleanup unconfirmed"));
                }
                finish_role_child(ctx, directory, pid, pgid)?;
                return Err(Error::io("write", &err));
            }
            kept += take;
        }
        let (child_state, child_group) =
            proc_state_group(&Path::new("/proc").join(pid.to_string()));
        let child_exited = child_state.as_deref() == Some("Z");
        if !expired && deadline.is_some_and(|deadline| Instant::now() >= deadline) {
            expired = true;
            capture_join_deadline = Some(Instant::now() + Duration::from_millis(KILL_GRACE_MS));
            if !terminate_role_group(pid, pgid, start_ticks, uid, Path::new("/proc"))? {
                unsafe { libc::close(reader) };
                return Err(Error::fail("deadline cleanup unconfirmed"));
            }
        } else if child_exited && probe_group_quiet(pgid, Path::new("/proc")) != Some(true) {
            descendants_found = true;
            if !terminate_role_group(pid, pgid, start_ticks, uid, Path::new("/proc"))? {
                unsafe { libc::close(reader) };
                return Err(Error::fail("role descendants remain active"));
            }
        }
        if !eof && capture_join_deadline.is_some_and(|deadline| Instant::now() >= deadline) {
            unsafe { libc::close(reader) };
            return Err(Error::fail(
                "role output join after deadline is unconfirmed",
            ));
        }
        if child_exited && eof && probe_group_quiet(pgid, Path::new("/proc")) == Some(true) {
            break;
        }
        if child_group.as_deref() != Some(pgid.to_string().as_str()) && !child_exited {
            unsafe { libc::close(reader) };
            return Err(Error::fail("role child identity changed"));
        }
        std::thread::sleep(Duration::from_millis(POLL_SLICE_MS));
    }
    if kept >= crate::LOG_CAP {
        if let Err(err) = log.write_all(b"\n[output truncated]\n") {
            drop(log);
            unsafe {
                libc::close(reader);
            }
            finish_role_child(ctx, directory, pid, pgid)?;
            return Err(Error::io("write", &err));
        }
    }
    drop(log);
    unsafe {
        libc::close(reader);
    }
    let status = finish_role_child(ctx, directory, pid, pgid)?;
    if expired {
        Ok(124)
    } else if descendants_found {
        Ok(125)
    } else {
        Ok(exit_code(status))
    }
}

/// `setup.sh`, then `check.sh` when setup passed and no stop tombstone
/// raced the launch; completion is always recorded.
pub fn run_preparation(
    ctx: &crate::Ctx,
    directory: &Path,
    fields: &ReqFields,
    tools: &[StateValue],
    account: &Account,
    deadline: Option<Instant>,
) -> Result<(), Error> {
    let checkout = account.dir.join("checkouts").join(&fields.id);
    let home = checkout.join(".soda-home");
    fsx::mkdir_p(&home.join("tmp"), 0o700)?;
    fsx::chown(&home.join("tmp"), account.uid, account.gid)?;
    let mut dirs: Vec<String> = tools
        .iter()
        .map(|tool| parent_of(tool.get("path").and_then(|v| v.as_str()).unwrap_or("")))
        .collect();
    dirs.sort();
    dirs.dedup();
    dirs.push("/usr/bin".to_string());
    dirs.push("/bin".to_string());
    let credential_file = if fields.credential.is_empty() {
        String::new()
    } else {
        ctx.credentials
            .join(&fields.role)
            .join(&fields.credential)
            .to_string_lossy()
            .into_owned()
    };
    let snapshot = directory.join("snapshot");
    let home_text = home.to_string_lossy().into_owned();
    let environ = child_environ(
        &dirs.join(":"),
        &home_text,
        &credential_file,
        &fields.role,
        &fields.id,
        &snapshot.to_string_lossy(),
    );
    let setup = vec![
        crate::SHELL.to_string(),
        format!("{}/{}", snapshot.to_string_lossy(), crate::SETUP_ENTRY),
    ];
    let setup_exit = run_as_role(
        ctx,
        directory,
        &setup,
        &environ,
        &checkout,
        &directory.join("setup.log"),
        account.uid,
        account.gid,
        deadline,
    )?;
    let mut check_exit: Option<i32> = None;
    if setup_exit == 0 && !fsx::lexists(&directory.join("stopped.json")) {
        let check = vec![
            crate::SHELL.to_string(),
            format!("{}/{}", snapshot.to_string_lossy(), crate::CHECK_ENTRY),
        ];
        check_exit = Some(run_as_role(
            ctx,
            directory,
            &check,
            &environ,
            &checkout,
            &directory.join("check.log"),
            account.uid,
            account.gid,
            deadline,
        )?);
    }
    let finished = obj(vec![
        ("setup_exit", StateValue::Number(setup_exit.to_string())),
        (
            "check_exit",
            check_exit
                .map(|code| StateValue::Number(code.to_string()))
                .unwrap_or(StateValue::Null),
        ),
    ]);
    let payload = crate::emit::dumps_default(&finished);
    fsx::write_new(&directory.join("finished.json"), payload.as_bytes(), 0o644)?;
    Ok(())
}

/// Double-fork supervisor: the intermediate records the leader before
/// exiting, the grandchild detaches fully (closing the inherited lock)
/// and runs the preparation. A stop tombstone racing the launch wins.
pub fn spawn_detached(
    ctx: &crate::Ctx,
    directory: &Path,
    fields: &ReqFields,
    tools: &[StateValue],
    deadline: Option<&str>,
) -> Result<StateValue, Error> {
    let account = crate::account::role_record(ctx, &fields.role)?;
    let first = unsafe { libc::fork() };
    if first < 0 {
        return Err(Error::io_msg("fork failed"));
    }
    if first != 0 {
        let status = wait_child(first);
        if status != 0 {
            return fail("detached setup launch unconfirmed");
        }
        return fsx::read_json(ctx, &directory.join("started.json"), 1024);
    }
    unsafe {
        if libc::setsid() < 0 {
            libc::_exit(1);
        }
        let second = libc::fork();
        if second < 0 {
            libc::_exit(1);
        }
        if second != 0 {
            let mut started = vec![
                ("pid", StateValue::Number(second.to_string())),
                ("pgid", StateValue::Number(second.to_string())),
            ];
            if let Some(deadline) = deadline {
                started.push(("deadline", str_value(deadline)));
            }
            let payload = crate::emit::dumps_default(&obj(started));
            match fsx::write_new(&directory.join("started.json"), payload.as_bytes(), 0o644) {
                Ok(()) => libc::_exit(0),
                Err(_) => libc::_exit(1),
            }
        }
    }
    let outcome = grandchild_body(ctx, directory, fields, tools, account.as_ref(), deadline);
    if outcome.is_err() {
        let _ = fsx::write_new(
            &directory.join("finished.json"),
            b"{\"setup_exit\": 127, \"check_exit\": null, \"launch_error\": true}",
            0o644,
        );
    }
    unsafe {
        libc::_exit(0);
    }
}

fn grandchild_body(
    ctx: &crate::Ctx,
    directory: &Path,
    fields: &ReqFields,
    tools: &[StateValue],
    account: Option<&Account>,
    deadline: Option<&str>,
) -> Result<(), Error> {
    unsafe {
        for fd in 3..1024 {
            libc::close(fd);
        }
        if libc::setpgid(0, 0) != 0 {
            return Err(Error::io_msg("setpgid failed"));
        }
        let root = CString::new("/").expect("root");
        if libc::chdir(root.as_ptr()) != 0 {
            return Err(Error::io_msg("chdir failed"));
        }
        let devnull = libc::open(c"/dev/null".as_ptr(), libc::O_RDWR);
        if devnull < 0 {
            return Err(Error::io_msg("devnull failed"));
        }
        libc::dup2(devnull, 0);
        libc::dup2(devnull, 1);
        libc::dup2(devnull, 2);
        if devnull > 2 {
            libc::close(devnull);
        }
    }
    if fsx::lexists(&directory.join("stopped.json")) {
        return Ok(());
    }
    match account {
        Some(account) => run_preparation(
            ctx,
            directory,
            fields,
            tools,
            account,
            deadline.map(deadline_instant).transpose()?,
        ),
        None => Err(Error::fail("role account vanished before launch")),
    }
}

pub fn do_start(ctx: &crate::Ctx, data: &StateValue) -> Result<StateValue, Error> {
    let Some(entries) = validate::as_object(data) else {
        return fail("unsupported start request");
    };
    let has_deadline = validate::key_set(entries, &["op", "id", "deadline"]);
    if !has_deadline && !validate::key_set(entries, &["op", "id"]) {
        return fail("unsupported start request");
    }
    let pid = validate::check_id(data.get("id").unwrap_or(&StateValue::Null))?.to_string();
    let deadline = if has_deadline {
        let text = data
            .get("deadline")
            .and_then(|value| value.as_str())
            .ok_or_else(|| Error::fail("invalid preparation deadline"))?;
        let parsed = soda_wire_time::parse_nanos(text)
            .ok_or_else(|| Error::fail("invalid preparation deadline"))?;
        if parsed <= wall_nanos_now() {
            return fail("preparation deadline has expired");
        }
        Some(text.to_string())
    } else {
        None
    };
    fsx::ensure_layout(ctx)?;
    let directory = fsx::prep_dir(ctx, &pid)?;
    ops_inspect::refuse_barred(ctx, &directory)?;
    let prestate = start_prestate(ctx, &directory)?;
    let started = match fsx::read_json(ctx, &directory.join("started.json"), 1024) {
        Ok(value) => Some(value),
        Err(Error::Missing) => None,
        Err(err) => return Err(err),
    };
    if let Some(started) = started {
        let saved_deadline = started.get("deadline").and_then(|value| value.as_str());
        if saved_deadline != deadline.as_deref() {
            return fail("preparation start deadline changed");
        }
        let pgid = ops_inspect::started_pgid(&started)?;
        if !ops_inspect::group_alive(pgid, std::path::Path::new("/proc")) {
            return fail("preparation supervisor is gone; stop and use a new identity");
        }
        return Ok(obj(vec![
            ("started", str_value(&pid)),
            ("repeated", StateValue::Bool(true)),
        ]));
    }
    let started = proc::spawn_detached(
        ctx,
        &directory,
        &prestate.fields,
        &prestate.tools,
        deadline.as_deref(),
    )?;
    let pgid = ops_inspect::started_pgid(&started)?;
    Ok(obj(vec![
        ("started", str_value(&pid)),
        ("repeated", StateValue::Bool(false)),
        ("pgid", StateValue::Number(pgid.to_string())),
    ]))
}

/// State and process group of one `/proc` stat entry. After the pid and
/// command name, the fields are state, ppid, pgrp. Unreadable entries
/// report `(None, None)`.
pub fn proc_state_group(entry: &Path) -> (Option<String>, Option<String>) {
    let (state, group, _) = proc_identity(entry);
    (state, group)
}

fn proc_identity(entry: &Path) -> (Option<String>, Option<String>, Option<u64>) {
    let text = match std::fs::read_to_string(entry.join("stat")) {
        Ok(text) => text,
        Err(_) => return (None, None, None),
    };
    let after = match text.rsplit_once(')') {
        Some((_, after)) => after,
        None => return (None, None, None),
    };
    let fields: Vec<&str> = after.split_whitespace().collect();
    if fields.len() < 20 {
        return (None, None, None);
    }
    (
        Some(fields[0].to_string()),
        Some(fields[2].to_string()),
        fields[19].parse().ok(),
    )
}

/// A group is alive while any non-zombie member keeps its process group.
/// Zombies hold no setup scope.
pub fn group_alive(pgid: i32, proot: &Path) -> bool {
    let want = pgid.to_string();
    let entries = match std::fs::read_dir(proot) {
        Ok(entries) => entries,
        Err(_) => return false,
    };
    for entry in entries.flatten() {
        let name = entry.file_name();
        let name = name.to_string_lossy();
        if name.is_empty() || !name.bytes().all(|b| b.is_ascii_digit()) {
            continue;
        }
        let (state, group) = proc_state_group(&entry.path());
        if state.as_deref() != Some("Z") && group.as_deref() == Some(want.as_str()) {
            return true;
        }
    }
    false
}

/// CODEX-P07-002: tri-state group observation for the release gate.
/// `Some(true)` only when native state positively shows no live member;
/// `Some(false)` when a non-zombie member keeps the group; `None` when
/// native state could not be read (uncertain — the caller must bar
/// release). Observation only: never signals. A vanished entry
/// (`NotFound`) is a benign exit race and is skipped; any other read
/// failure hides a potential member and reports uncertainty.
fn probe_group_quiet(pgid: i32, proot: &Path) -> Option<bool> {
    let want = pgid.to_string();
    let entries = match std::fs::read_dir(proot) {
        Ok(entries) => entries,
        Err(_) => return None,
    };
    for entry in entries {
        let entry = match entry {
            Ok(entry) => entry,
            Err(_) => return None,
        };
        let name = entry.file_name();
        let name = name.to_string_lossy();
        if name.is_empty() || !name.bytes().all(|b| b.is_ascii_digit()) {
            continue;
        }
        let text = match std::fs::read_to_string(entry.path().join("stat")) {
            Ok(text) => text,
            Err(err) if err.kind() == std::io::ErrorKind::NotFound => continue,
            Err(_) => return None,
        };
        let after = match text.rsplit_once(')') {
            Some((_, after)) => after,
            None => continue,
        };
        let fields: Vec<&str> = after.split_whitespace().collect();
        if fields.len() < 3 {
            continue;
        }
        if fields[0] != "Z" && fields[2] == want.as_str() {
            return Some(false);
        }
    }
    Some(true)
}

/// CODEX-P07-002: release-gate quiescence for one preparation. Only a
/// started record whose group is positively quiet lifts the bar.
/// Unreadable records, unparsable identity, an active group, or unreadable
/// native state all bar release. Observation only: never signals.
fn preparation_blocks_release(ctx: &crate::Ctx, dir: &Path, proot: &Path) -> bool {
    // Only the worker removes this receipt after capture, group retirement
    // and child reaping all confirm. A quiet group alone cannot settle a
    // retained receipt from an unconfirmed output join.
    match fsx::read_json(ctx, &active_child_path(dir), 1024) {
        Ok(_) => return true,
        Err(Error::Missing) => {}
        Err(_) => return true,
    }
    let started = match fsx::read_json(ctx, &dir.join("started.json"), 1024) {
        Ok(value) => value,
        Err(Error::Missing) => return false,
        Err(_) => return true,
    };
    let pgid = match started_pgid(&started) {
        Ok(pgid) => pgid,
        Err(_) => return true,
    };
    match probe_group_quiet(pgid, proot) {
        Some(true) => false,
        Some(false) => true,
        None => true,
    }
}

fn retire_active_role_group(
    ctx: &crate::Ctx,
    directory: &Path,
    role: &str,
) -> Result<String, Error> {
    let active = match fsx::read_json(ctx, &active_child_path(directory), 1024) {
        Ok(value) => value,
        Err(Error::Missing) => return Ok("confirmed".to_string()),
        Err(error) => return Err(error),
    };
    let (pid, pgid, start_ticks) = active_child_ids(&active)?;
    let proot = Path::new("/proc");
    if probe_group_quiet(pgid, proot) == Some(true) {
        return Ok("confirmed".to_string());
    }
    let account = crate::account::role_record(ctx, role)?;
    let Some(account) = account else {
        return Ok("uncertain".to_string());
    };
    if !pinned_role_group_leader(pid, pgid, start_ticks, account.uid, proot) {
        return Ok("uncertain".to_string());
    }
    if terminate_role_group(pid, pgid, start_ticks, account.uid, proot)? {
        Ok("confirmed".to_string())
    } else {
        Ok("uncertain".to_string())
    }
}

/// The recorded supervisor leader still leads its group and is owned by
/// the helper or the role (zombies never qualify).
pub fn leader_owned_by(
    ctx: &crate::Ctx,
    pid: i32,
    pgid: i32,
    account_uid: u32,
    proot: &Path,
) -> bool {
    let entry = proot.join(pid.to_string());
    let (state, group) = proc_state_group(&entry);
    if group.as_deref() != Some(pgid.to_string().as_str()) || state.as_deref() == Some("Z") {
        return false;
    }
    match std::fs::symlink_metadata(&entry) {
        Ok(meta) => meta.uid() == ctx.priv_uid() || meta.uid() == account_uid,
        Err(_) => false,
    }
}

fn killpg(pgid: i32, signal: libc::c_int) -> Result<(), i32> {
    if unsafe { libc::killpg(pgid, signal) } == 0 {
        Ok(())
    } else {
        Err(std::io::Error::last_os_error()
            .raw_os_error()
            .unwrap_or(libc::EINVAL))
    }
}

/// `SIGTERM` the group, wait up to 5s, escalate to `SIGKILL`, and report
/// whether retirement is confirmed.
pub fn signal_group(pgid: i32, proot: &Path) -> Result<String, Error> {
    match killpg(pgid, libc::SIGTERM) {
        Ok(()) => {}
        Err(errno) if errno == libc::ESRCH => return Ok("confirmed".to_string()),
        Err(errno) if errno == libc::EPERM => return Ok("uncertain".to_string()),
        Err(errno) => return Err(Error::io_msg(format!("killpg: errno {errno}"))),
    }
    let deadline = Instant::now() + Duration::from_secs(5);
    while Instant::now() < deadline {
        match probe_group_quiet(pgid, proot) {
            Some(true) => return Ok("confirmed".to_string()),
            Some(false) => std::thread::sleep(Duration::from_millis(200)),
            None => return Ok("uncertain".to_string()),
        }
    }
    if probe_group_quiet(pgid, proot) == Some(true) {
        return Ok("confirmed".to_string());
    }
    match killpg(pgid, libc::SIGKILL) {
        Ok(()) => {}
        Err(errno) if errno == libc::ESRCH || errno == libc::EPERM => {}
        Err(errno) => return Err(Error::io_msg(format!("killpg: errno {errno}"))),
    }
    let deadline = Instant::now() + Duration::from_millis(500);
    while Instant::now() < deadline {
        match probe_group_quiet(pgid, proot) {
            Some(true) => return Ok("confirmed".to_string()),
            Some(false) => std::thread::sleep(Duration::from_millis(50)),
            None => return Ok("uncertain".to_string()),
        }
    }
    Ok(if probe_group_quiet(pgid, proot) == Some(true) {
        "confirmed"
    } else {
        "uncertain"
    }
    .to_string())
}

/// Retire the recorded supervisor: groups whose leader is no longer ours
/// are never signalled, only observed.
pub fn retire_group(ctx: &crate::Ctx, directory: &Path, role: &str) -> Result<String, Error> {
    let started = match fsx::read_json(ctx, &directory.join("started.json"), 1024) {
        Ok(value) => value,
        Err(Error::Missing) => return Ok("confirmed".to_string()),
        Err(err) => return Err(err),
    };
    let proot = Path::new("/proc");
    let pid = started_pid(&started)?;
    let pgid = started_pgid(&started)?;
    let owned = match crate::account::role_record(ctx, role)? {
        Some(account) => leader_owned_by(ctx, pid, pgid, account.uid, proot),
        None => false,
    };
    if !owned {
        if group_alive(pgid, proot) {
            return Ok("uncertain".to_string());
        }
        return Ok("confirmed".to_string());
    }
    signal_group(pgid, proot)
}

pub fn do_stop(ctx: &crate::Ctx, data: &StateValue) -> Result<StateValue, Error> {
    if !validate::as_object(data).is_some_and(|e| validate::key_set(e, &["op", "id"])) {
        return fail("unsupported stop request");
    }
    let pid = validate::check_id(data.get("id").unwrap_or(&StateValue::Null))?.to_string();
    fsx::ensure_layout(ctx)?;
    let directory = fsx::prep_dir(ctx, &pid)?;
    // Serialize request observation, the stop fence, and active-child
    // identity/signalling against start and role-child admission.
    let lock = state_lock(ctx)?;
    let request = match fsx::read_json(ctx, &directory.join("request.json"), 4096) {
        Ok(value) => Some(value),
        Err(Error::Missing) => {
            // A stop tombstone before any observation still bars the identity.
            fsx::mkdir_p(&directory, 0o755)?;
            None
        }
        Err(err) => return Err(err),
    };
    match fsx::write_new(
        &directory.join("stopped.json"),
        b"{\"stopped\": true}",
        0o644,
    ) {
        Ok(()) => {}
        Err(Error::Exists) => {}
        Err(err) => return Err(err),
    }
    match request {
        None => {
            drop(lock);
            Ok(obj(vec![
                ("stopped", str_value(&pid)),
                ("retirement", str_value("confirmed")),
                ("known", StateValue::Bool(false)),
            ]))
        }
        Some(request) => {
            let role = match request.get("role").and_then(|v| v.as_str()) {
                Some(role) => role.to_string(),
                None => return Err(Error::fail("unsafe factory metadata")),
            };
            let mut child_retirement = retire_active_role_group(ctx, &directory, &role)?;
            drop(lock);
            if child_retirement == "confirmed" && !wait_active_role_join(&directory) {
                child_retirement = "uncertain".to_string();
            }
            // P07-F1: a launch-error (or unreadable) receipt never
            // independently establishes retirement; verify the group.
            let finished_clean = match fsx::read_json(ctx, &directory.join("finished.json"), 1024) {
                Ok(finished) => {
                    finished.get("launch_error").and_then(|v| v.as_bool()) != Some(true)
                }
                Err(Error::Missing) => false,
                Err(_) => false,
            };
            if finished_clean {
                return Ok(obj(vec![
                    ("stopped", str_value(&pid)),
                    ("retirement", str_value(&child_retirement)),
                    ("known", StateValue::Bool(true)),
                ]));
            }
            let retirement = retire_group(ctx, &directory, &role)?;
            Ok(obj(vec![
                ("stopped", str_value(&pid)),
                (
                    "retirement",
                    str_value(
                        if child_retirement == "confirmed" && retirement == "confirmed" {
                            "confirmed"
                        } else {
                            "uncertain"
                        },
                    ),
                ),
                ("known", StateValue::Bool(true)),
            ]))
        }
    }
}

/// Any preparation with a supervisor on record bars hold release until
/// native state positively confirms its group is quiet. An unterminated
/// supervisor bars outright; terminal markers lift the bar only through
/// the release gate, so an uncertain stop still blocks while the group
/// lives (CODEX-P07-002).
pub fn any_running(ctx: &crate::Ctx) -> Result<bool, Error> {
    let entries = std::fs::read_dir(&ctx.preparations).map_err(Error::classify)?;
    for entry in entries.flatten() {
        let name = entry.file_name().to_string_lossy().into_owned();
        if !validate::is_id(&name) {
            continue;
        }
        let dir = ctx.preparations.join(&name);
        if !fsx::lexists(&dir.join("started.json")) {
            continue;
        }
        let terminated =
            fsx::lexists(&dir.join("finished.json")) || fsx::lexists(&dir.join("stopped.json"));
        if !terminated {
            return Ok(true);
        }
        if preparation_blocks_release(ctx, &dir, Path::new("/proc")) {
            return Ok(true);
        }
    }
    Ok(false)
}

fn hold_revision(data: &StateValue) -> Option<String> {
    if !validate::as_object(data).is_some_and(|e| validate::key_set(e, &["op", "revision"])) {
        return None;
    }
    validate::as_int_text(data.get("revision").unwrap_or(&StateValue::Null))
}

pub fn do_hold(ctx: &crate::Ctx, data: &StateValue) -> Result<StateValue, Error> {
    let Some(revision) = hold_revision(data) else {
        return fail("unsupported hold request");
    };
    fsx::ensure_layout(ctx)?;
    let payload = format!("{{\"revision\": {revision}}}");
    match fsx::write_new(&ctx.hold, payload.as_bytes(), 0o644) {
        Ok(()) => {}
        Err(Error::Exists) => {
            let current = fsx::read_json(ctx, &ctx.hold, 1024)?;
            let same = validate::as_object(&current)
                .is_some_and(|e| validate::key_set(e, &["revision"]))
                && validate::as_int_text(current.get("revision").unwrap_or(&StateValue::Null))
                    .as_deref()
                    == Some(revision.as_str());
            if !same {
                return fail("maintenance hold already carries another revision");
            }
        }
        Err(err) => return Err(err),
    }
    Ok(obj(vec![("hold", hold_json(&hold_state(ctx)?))]))
}

pub fn do_release(ctx: &crate::Ctx, data: &StateValue) -> Result<StateValue, Error> {
    let Some(revision) = hold_revision(data) else {
        return fail("unsupported release request");
    };
    fsx::ensure_layout(ctx)?;
    let held = hold_state(ctx)?;
    if !held.active {
        return Ok(obj(vec![("hold", hold_json(&held))]));
    }
    if held.revision != revision {
        return fail("stale maintenance hold revision");
    }
    if any_running(ctx)? {
        return fail("running preparations bar hold release");
    }
    std::fs::remove_file(&ctx.hold).map_err(Error::classify)?;
    Ok(obj(vec![("hold", hold_json(&hold_state(ctx)?))]))
}

#[cfg(test)]
#[path = "execution_tests.rs"]
mod tests;
