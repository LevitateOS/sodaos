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
use crate::validate;
use soda_json::JsonValue;
use std::ffi::CString;
use std::fs::File;
use std::io::Write;
use std::os::unix::ffi::OsStrExt;
use std::os::unix::fs::MetadataExt;
use std::path::Path;

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

/// CODEX-P07-001: bounded checked termination of an owned role child after
/// a log failure. SIGTERM with a grace interval, then SIGKILL with a final
/// reap interval; every wait is `WNOHANG` so even an unkillable
/// (uninterruptible-sleep) child cannot hang the supervisor. The first
/// probe runs before any signal: an already-reaped pid reports `None`
/// without signaling, so a recycled pid is never touched. Returns the wait
/// status when the child was confirmed reaped, or `None` when termination
/// could not be confirmed (the caller must report uncertainty, never Ok).
fn terminate_child(pid: libc::pid_t) -> Option<libc::c_int> {
    const TERM_GRACE_MS: u64 = 1000;
    const KILL_GRACE_MS: u64 = 1000;
    const POLL_SLICE_MS: u64 = 20;
    fn reap_once(pid: libc::pid_t) -> Option<Option<libc::c_int>> {
        let mut status = 0;
        loop {
            let rc = unsafe { libc::waitpid(pid, &mut status, libc::WNOHANG) };
            if rc == pid {
                return Some(Some(status));
            }
            if rc < 0 {
                let errno = std::io::Error::last_os_error().raw_os_error();
                if errno == Some(libc::EINTR) {
                    continue;
                }
                // Not a waitable child (already reaped elsewhere): gone,
                // so no signal may be sent.
                return Some(None);
            }
            return None;
        }
    }
    fn reap_until(pid: libc::pid_t, deadline: std::time::Instant) -> Option<libc::c_int> {
        loop {
            match reap_once(pid) {
                Some(outcome) => return outcome,
                None => {}
            }
            if std::time::Instant::now() >= deadline {
                return None;
            }
            std::thread::sleep(std::time::Duration::from_millis(POLL_SLICE_MS));
        }
    }
    match reap_once(pid) {
        Some(outcome) => return outcome,
        None => {}
    }
    unsafe {
        libc::kill(pid, libc::SIGTERM);
    }
    let outcome = reap_until(
        pid,
        std::time::Instant::now() + std::time::Duration::from_millis(TERM_GRACE_MS),
    );
    if outcome.is_some() {
        return outcome;
    }
    unsafe {
        libc::kill(pid, libc::SIGKILL);
    }
    reap_until(
        pid,
        std::time::Instant::now() + std::time::Duration::from_millis(KILL_GRACE_MS),
    )
}

/// CODEX-P07-001: log-failure exit for `run_as_role`. The original log
/// error is reported only when the owned child was confirmed reaped;
/// otherwise cleanup is unconfirmed and uncertainty is retained.
fn log_failure_cleanup(pid: libc::pid_t, original: Error) -> Error {
    match terminate_child(pid) {
        Some(_) => original,
        None => Error::fail("log capture failed; role child termination unconfirmed"),
    }
}

fn to_cstring(bytes: &[u8]) -> Option<CString> {
    CString::new(bytes).ok()
}

/// Run one fixed entry point as the role, capturing bounded output. The
/// supervisor stays privileged so role code cannot forge logs or
/// completion. Anything failing in the child exits it with 127.
pub fn run_as_role(
    argv: &[String],
    environ: &[(String, String)],
    workdir: &Path,
    logpath: &Path,
    uid: u32,
    gid: u32,
) -> Result<i32, Error> {
    let mut fds = [0 as libc::c_int; 2];
    if unsafe { libc::pipe(fds.as_mut_ptr()) } != 0 {
        return Err(Error::io_msg("pipe failed"));
    }
    let (reader, writer) = (fds[0], fds[1]);
    let pid = unsafe { libc::fork() };
    if pid < 0 {
        unsafe {
            libc::close(reader);
            libc::close(writer);
        }
        return Err(Error::io_msg("fork failed"));
    }
    if pid == 0 {
        unsafe {
            libc::close(reader);
            libc::dup2(writer, 1);
            libc::dup2(writer, 2);
            libc::close(writer);
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
    }
    // P07-F1/001: log-I/O failures must still close owned descriptors
    // and terminate/reap the role child within a bound before reporting;
    // unconfirmed cleanup retains uncertainty instead of the log error.
    let mut log = match File::create(logpath) {
        Ok(log) => log,
        Err(err) => {
            unsafe {
                libc::close(reader);
            }
            return Err(log_failure_cleanup(pid, Error::classify(err)));
        }
    };
    let mut kept = 0usize;
    loop {
        let mut chunk = [0u8; 65536];
        let got =
            unsafe { libc::read(reader, chunk.as_mut_ptr() as *mut libc::c_void, chunk.len()) };
        if got < 0 {
            if std::io::Error::last_os_error().raw_os_error() == Some(libc::EINTR) {
                continue;
            }
            unsafe {
                libc::close(reader);
            }
            return Err(log_failure_cleanup(pid, Error::io_msg("log pipe failed")));
        }
        if got == 0 {
            break;
        }
        if kept < crate::LOG_CAP {
            let take = (got as usize).min(crate::LOG_CAP - kept);
            if let Err(err) = log.write_all(&chunk[..take]) {
                drop(log);
                unsafe {
                    libc::close(reader);
                }
                return Err(log_failure_cleanup(pid, Error::io("write", &err)));
            }
            kept += take;
        }
    }
    if kept >= crate::LOG_CAP {
        if let Err(err) = log.write_all(b"\n[output truncated]\n") {
            drop(log);
            unsafe {
                libc::close(reader);
            }
            return Err(log_failure_cleanup(pid, Error::io("write", &err)));
        }
    }
    drop(log);
    unsafe {
        libc::close(reader);
    }
    let status = wait_child(pid);
    if status < 0 {
        return Err(Error::io_msg("waitpid failed"));
    }
    Ok(exit_code(status))
}

/// `setup.sh`, then `check.sh` when setup passed and no stop tombstone
/// raced the launch; completion is always recorded.
pub fn run_preparation(
    ctx: &crate::Ctx,
    directory: &Path,
    fields: &ReqFields,
    tools: &[JsonValue],
    account: &Account,
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
        &setup,
        &environ,
        &checkout,
        &directory.join("setup.log"),
        account.uid,
        account.gid,
    )?;
    let mut check_exit: Option<i32> = None;
    if setup_exit == 0 && !fsx::lexists(&directory.join("stopped.json")) {
        let check = vec![
            crate::SHELL.to_string(),
            format!("{}/{}", snapshot.to_string_lossy(), crate::CHECK_ENTRY),
        ];
        check_exit = Some(run_as_role(
            &check,
            &environ,
            &checkout,
            &directory.join("check.log"),
            account.uid,
            account.gid,
        )?);
    }
    let finished = obj(vec![
        ("setup_exit", JsonValue::Number(setup_exit.to_string())),
        (
            "check_exit",
            check_exit
                .map(|code| JsonValue::Number(code.to_string()))
                .unwrap_or(JsonValue::Null),
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
    tools: &[JsonValue],
) -> Result<JsonValue, Error> {
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
            let payload = format!("{{\"pid\": {second}, \"pgid\": {second}}}");
            match fsx::write_new(&directory.join("started.json"), payload.as_bytes(), 0o644) {
                Ok(()) => libc::_exit(0),
                Err(_) => libc::_exit(1),
            }
        }
    }
    let outcome = grandchild_body(ctx, directory, fields, tools, account.as_ref());
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
    tools: &[JsonValue],
    account: Option<&Account>,
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
        Some(account) => run_preparation(ctx, directory, fields, tools, account),
        None => Err(Error::fail("role account vanished before launch")),
    }
}

pub fn do_start(ctx: &crate::Ctx, data: &JsonValue) -> Result<JsonValue, Error> {
    if !validate::as_object(data).is_some_and(|e| validate::key_set(e, &["op", "id"])) {
        return fail("unsupported start request");
    }
    let pid = validate::check_id(data.get("id").unwrap_or(&JsonValue::Null))?.to_string();
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
        let pgid = ops_inspect::started_pgid(&started)?;
        if !ops_inspect::group_alive(pgid, std::path::Path::new("/proc")) {
            return fail("preparation supervisor is gone; stop and use a new identity");
        }
        return Ok(obj(vec![
            ("started", str_value(&pid)),
            ("repeated", JsonValue::Bool(true)),
        ]));
    }
    let started = proc::spawn_detached(ctx, &directory, &prestate.fields, &prestate.tools)?;
    let pgid = ops_inspect::started_pgid(&started)?;
    Ok(obj(vec![
        ("started", str_value(&pid)),
        ("repeated", JsonValue::Bool(false)),
        ("pgid", JsonValue::Number(pgid.to_string())),
    ]))
}

/// State and process group of one `/proc` stat entry. After the pid and
/// command name, the fields are state, ppid, pgrp. Unreadable entries
/// report `(None, None)`.
pub fn proc_state_group(entry: &Path) -> (Option<String>, Option<String>) {
    let text = match std::fs::read_to_string(entry.join("stat")) {
        Ok(text) => text,
        Err(_) => return (None, None),
    };
    let after = match text.rsplit_once(')') {
        Some((_, after)) => after,
        None => return (None, None),
    };
    let fields: Vec<&str> = after.split_whitespace().collect();
    if fields.len() < 3 {
        return (None, None);
    }
    (Some(fields[0].to_string()), Some(fields[2].to_string()))
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
    let deadline = std::time::Instant::now() + std::time::Duration::from_secs(5);
    while group_alive(pgid, proot) && std::time::Instant::now() < deadline {
        std::thread::sleep(std::time::Duration::from_millis(200));
    }
    if !group_alive(pgid, proot) {
        return Ok("confirmed".to_string());
    }
    match killpg(pgid, libc::SIGKILL) {
        Ok(()) => {}
        Err(errno) if errno == libc::ESRCH || errno == libc::EPERM => {}
        Err(errno) => return Err(Error::io_msg(format!("killpg: errno {errno}"))),
    }
    std::thread::sleep(std::time::Duration::from_millis(500));
    if group_alive(pgid, proot) {
        Ok("uncertain".to_string())
    } else {
        Ok("confirmed".to_string())
    }
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

pub fn do_stop(ctx: &crate::Ctx, data: &JsonValue) -> Result<JsonValue, Error> {
    if !validate::as_object(data).is_some_and(|e| validate::key_set(e, &["op", "id"])) {
        return fail("unsupported stop request");
    }
    let pid = validate::check_id(data.get("id").unwrap_or(&JsonValue::Null))?.to_string();
    fsx::ensure_layout(ctx)?;
    let directory = fsx::prep_dir(ctx, &pid)?;
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
        None => Ok(obj(vec![
            ("stopped", str_value(&pid)),
            ("retirement", str_value("confirmed")),
            ("known", JsonValue::Bool(false)),
        ])),
        Some(request) => {
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
                    ("retirement", str_value("confirmed")),
                    ("known", JsonValue::Bool(true)),
                ]));
            }
            let role = match request.get("role").and_then(|v| v.as_str()) {
                Some(role) => role.to_string(),
                None => return Err(Error::fail("unsafe factory metadata")),
            };
            let retirement = retire_group(ctx, &directory, &role)?;
            Ok(obj(vec![
                ("stopped", str_value(&pid)),
                ("retirement", str_value(&retirement)),
                ("known", JsonValue::Bool(true)),
            ]))
        }
    }
}

/// Any preparation with a supervisor on record that neither finished nor
/// stopped bars hold release.
pub fn any_running(ctx: &crate::Ctx) -> Result<bool, Error> {
    let entries = std::fs::read_dir(&ctx.preparations).map_err(Error::classify)?;
    for entry in entries.flatten() {
        let name = entry.file_name().to_string_lossy().into_owned();
        if !validate::is_id(&name) {
            continue;
        }
        let dir = ctx.preparations.join(&name);
        if fsx::lexists(&dir.join("started.json"))
            && !fsx::lexists(&dir.join("finished.json"))
            && !fsx::lexists(&dir.join("stopped.json"))
        {
            return Ok(true);
        }
    }
    Ok(false)
}

fn hold_revision(data: &JsonValue) -> Option<String> {
    if !validate::as_object(data).is_some_and(|e| validate::key_set(e, &["op", "revision"])) {
        return None;
    }
    validate::as_int_text(data.get("revision").unwrap_or(&JsonValue::Null))
}

pub fn do_hold(ctx: &crate::Ctx, data: &JsonValue) -> Result<JsonValue, Error> {
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
                && validate::as_int_text(current.get("revision").unwrap_or(&JsonValue::Null))
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

pub fn do_release(ctx: &crate::Ctx, data: &JsonValue) -> Result<JsonValue, Error> {
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
