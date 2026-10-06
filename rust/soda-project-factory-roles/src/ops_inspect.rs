//! `inspect`, `stop`, `hold`, `release`: state reporting, supervisor
//! retirement, and the maintenance hold. Also the `/proc` process-group
//! primitives shared with `start`.

pub(crate) use super::fsx::*;
use crate::emit::{obj, str_value};
use crate::error::{fail, Error};
use crate::fsx;
use crate::validate;
use soda_json::JsonValue;
use std::os::unix::fs::MetadataExt;
use std::path::Path;

/// Bounded privileged log read; missing logs read empty.
pub fn read_log(ctx: &crate::Ctx, path: &Path) -> Result<String, Error> {
    let file = match fsx::open_ro(path, true) {
        Ok(file) => file,
        Err(Error::Missing) => return Ok(String::new()),
        Err(err) => return Err(err),
    };
    let meta = file.metadata().map_err(Error::classify)?;
    if !meta.is_file() || meta.uid() != ctx.priv_uid() {
        return fail(format!("unsafe preparation log: {}", fsx::file_name(path)));
    }
    let mut raw = Vec::new();
    use std::io::Read;
    file.take((crate::LOG_CAP as u64) + 1024)
        .read_to_end(&mut raw)
        .map_err(|err| Error::io("read", &err))?;
    Ok(String::from_utf8_lossy(&raw).into_owned())
}

/// `started.json` process group the helper recorded itself.
pub fn started_pgid(started: &JsonValue) -> Result<i32, Error> {
    match validate::as_i64(started.get("pgid").unwrap_or(&JsonValue::Null)) {
        Some(pgid) => i32::try_from(pgid).map_err(|_| Error::fail("unsafe factory metadata")),
        None => Err(Error::fail("unsafe factory metadata")),
    }
}

pub fn started_pid(started: &JsonValue) -> Result<i32, Error> {
    match validate::as_i64(started.get("pid").unwrap_or(&JsonValue::Null)) {
        Some(pid) => i32::try_from(pid).map_err(|_| Error::fail("unsafe factory metadata")),
        None => Err(Error::fail("unsafe factory metadata")),
    }
}

fn exit_is_zero(value: &JsonValue, key: &str) -> bool {
    validate::as_int_text(value.get(key).unwrap_or(&JsonValue::Null)).as_deref() == Some("0")
}

/// `(phase, finished?)`: completion decides `ready`/`failed` first, then
/// evidence, then supervisor liveness. A dead supervisor without
/// completion is `interrupted`, never resurrected.
pub fn phase_of(ctx: &crate::Ctx, directory: &Path) -> Result<(String, Option<JsonValue>), Error> {
    let finished = match fsx::read_json(ctx, &directory.join("finished.json"), 1024) {
        Ok(value) => Some(value),
        Err(Error::Missing) => None,
        Err(err) => return Err(err),
    };
    if let Some(finished) = finished {
        let phase =
            if exit_is_zero(&finished, "setup_exit") && exit_is_zero(&finished, "check_exit") {
                "ready"
            } else {
                "failed"
            };
        return Ok((phase.to_string(), Some(finished)));
    }
    let tools = match fsx::read_json(ctx, &directory.join("tools.json"), 65536) {
        Ok(value) => value,
        Err(Error::Missing) => return Ok(("approved".to_string(), None)),
        Err(err) => return Err(err),
    };
    if tools.get("missing").and_then(|v| v.as_str()) != Some("") {
        return Ok(("waiting".to_string(), None));
    }
    let refusal = tools
        .get("verified")
        .and_then(|v| v.get("refusal"))
        .and_then(|v| v.as_str())
        .unwrap_or("");
    if !refusal.is_empty() {
        return Ok(("failed".to_string(), None));
    }
    let started = match fsx::read_json(ctx, &directory.join("started.json"), 1024) {
        Ok(value) => value,
        Err(Error::Missing) => return Ok(("approved".to_string(), None)),
        Err(err) => return Err(err),
    };
    if !group_alive(started_pgid(&started)?, Path::new("/proc")) {
        return Ok(("interrupted".to_string(), None));
    }
    Ok(("running".to_string(), None))
}

pub fn do_inspect(ctx: &crate::Ctx, data: &JsonValue) -> Result<JsonValue, Error> {
    let shape_ok = validate::as_object(data)
        .is_some_and(|e| validate::key_set(e, &["op"]) || validate::key_set(e, &["op", "id"]));
    if !shape_ok {
        return fail("unsupported inspect request");
    }
    fsx::ensure_layout(ctx)?;
    let hold = hold_state(ctx)?;
    if data.get("id").is_none() {
        return Ok(obj(vec![("hold", hold_json(&hold))]));
    }
    let pid = validate::check_id(data.get("id").unwrap_or(&JsonValue::Null))?.to_string();
    let directory = fsx::prep_dir(ctx, &pid)?;
    let request = match fsx::read_json(ctx, &directory.join("request.json"), 4096) {
        Ok(value) => value,
        Err(Error::Missing) => {
            return Ok(obj(vec![
                ("hold", hold_json(&hold)),
                ("known", JsonValue::Bool(false)),
            ]));
        }
        Err(err) => return Err(err),
    };
    let (mut phase, finished) = phase_of(ctx, &directory)?;
    let tools = match fsx::read_json(ctx, &directory.join("tools.json"), 65536) {
        Ok(value) => value,
        Err(Error::Missing) => obj(vec![
            ("tools", JsonValue::Array(Vec::new())),
            ("missing", str_value("")),
            ("verified", obj(vec![])),
        ]),
        Err(err) => return Err(err),
    };
    let stopped = fsx::lexists(&directory.join("stopped.json"));
    if stopped && ["running", "approved", "waiting", "interrupted"].contains(&phase.as_str()) {
        phase = "stopped".to_string();
    }
    let get_str = |key: &str| match request.get(key).and_then(|v| v.as_str()) {
        Some(text) => Ok(str_value(text)),
        None => Err(Error::fail("unsafe factory metadata")),
    };
    let mut result = vec![
        ("hold", hold_json(&hold)),
        ("known", JsonValue::Bool(true)),
        ("phase", str_value(&phase)),
        ("role", get_str("role")?),
        ("setup_digest", get_str("setup_digest")?),
        ("source_commit", get_str("source_commit")?),
        (
            "tools",
            tools.get("tools").cloned().unwrap_or(JsonValue::Null),
        ),
        (
            "missing",
            tools.get("missing").cloned().unwrap_or(JsonValue::Null),
        ),
        (
            "verified",
            tools.get("verified").cloned().unwrap_or(JsonValue::Null),
        ),
        ("stopped", JsonValue::Bool(stopped)),
        ("ready", JsonValue::Bool(phase == "ready")),
        (
            "setup_log",
            str_value(&read_log(ctx, &directory.join("setup.log"))?),
        ),
        (
            "check_log",
            str_value(&read_log(ctx, &directory.join("check.log"))?),
        ),
    ];
    if let Some(finished) = finished {
        result.push((
            "setup_exit",
            finished
                .get("setup_exit")
                .cloned()
                .unwrap_or(JsonValue::Null),
        ));
        result.push((
            "check_exit",
            finished
                .get("check_exit")
                .cloned()
                .unwrap_or(JsonValue::Null),
        ));
    }
    Ok(obj(result))
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
            if fsx::lexists(&directory.join("finished.json")) {
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
mod tests {
    use super::*;
    use crate::testutil::{
        approve_default, assert_fail, op_value, record_value, Scratch, PID, PID2,
    };

    fn write_proot_stat(proot: &Path, pid: &str, stat: &str) {
        let dir = proot.join(pid);
        std::fs::create_dir_all(&dir).unwrap();
        std::fs::write(dir.join("stat"), stat).unwrap();
    }

    /// A process group that just exited: spawn a group leader, reap it,
    /// and prove the group is gone before handing the pgid out.
    fn dead_pgid() -> i32 {
        use std::os::unix::process::CommandExt;
        let mut child = std::process::Command::new("/bin/true")
            .process_group(0)
            .spawn()
            .expect("spawn true");
        let pgid = child.id() as i32;
        child.wait().expect("reap true");
        assert!(!group_alive(pgid, Path::new("/proc")));
        pgid
    }

    #[test]
    fn proc_group_reads_pgrp_not_session() {
        let scratch = Scratch::fresh();
        let proot = scratch.root.join("proc");
        write_proot_stat(
            &proot,
            "46",
            "46 (worker) S 1 46 45 0 -1 0 0 0 0 0 0 0 0 0 0 1 0 1 0 0 0 0",
        );
        write_proot_stat(
            &proot,
            "47",
            "47 (my) proc) S 46 46 45 0 -1 0 0 0 0 0 0 0 0 0 0 1 0 1 0 0 0 0",
        );
        assert!(group_alive(46, &proot));
        assert!(!group_alive(45, &proot));
        let (git, _) = scratch.git_script("git-ok", 0);
        let ctx = scratch.ctx(&git);
        assert!(leader_owned_by(&ctx, 46, 46, 2000, &proot));
        assert!(!leader_owned_by(&ctx, 46, 45, 2000, &proot));
        assert!(!leader_owned_by(&ctx, 999, 46, 2000, &proot));
        write_proot_stat(
            &proot,
            "46",
            "46 (worker) Z 1 46 45 0 -1 0 0 0 0 0 0 0 0 0 0 1 0 1 0 0 0 0",
        );
        write_proot_stat(
            &proot,
            "47",
            "47 (sleep) Z 46 46 45 0 -1 0 0 0 0 0 0 0 0 0 0 1 0 1 0 0 0 0",
        );
        assert!(!group_alive(46, &proot));
        assert!(!leader_owned_by(&ctx, 46, 46, 2000, &proot));
        // Malformed entries are skipped, not fatal.
        write_proot_stat(&proot, "48", "garbage without parens");
        write_proot_stat(&proot, "49", "49 (x) S");
        assert!(!group_alive(48, &proot));
        let (state, group) = proc_state_group(&proot.join("48"));
        assert_eq!((state, group), (None, None));
    }

    #[test]
    fn signal_group_confirms_dead_groups() {
        // ESRCH on a surely-dead group reports confirmed without waiting.
        assert_eq!(
            signal_group(dead_pgid(), Path::new("/proc")).unwrap(),
            "confirmed"
        );
    }

    fn hold_request(revision: &str) -> JsonValue {
        JsonValue::parse(&format!("{{\"op\": \"hold\", \"revision\": {revision}}}")).unwrap()
    }

    fn release_request(revision: &str) -> JsonValue {
        JsonValue::parse(&format!(
            "{{\"op\": \"release\", \"revision\": {revision}}}"
        ))
        .unwrap()
    }

    #[test]
    fn hold_bars_preparation_and_releases_by_revision() {
        let scratch = Scratch::fresh();
        let (git, _) = scratch.git_script("git-ok", 0);
        let ctx = scratch.ctx(&git);
        crate::ops_approve::do_approve(&ctx, &approve_default(PID)).unwrap();
        let held = do_hold(&ctx, &hold_request("1")).unwrap();
        assert_eq!(
            held.get("hold")
                .and_then(|h| h.get("active"))
                .and_then(|v| v.as_bool()),
            Some(true)
        );
        assert_fail(
            crate::ops_approve::do_approve(&ctx, &approve_default(PID2)),
            "maintenance hold denies preparation",
        );
        assert_fail(
            do_release(&ctx, &release_request("2")),
            "stale maintenance hold revision",
        );
        // A started-but-unfinished preparation bars release.
        let directory = ctx.preparations.join(PID);
        crate::fsx::write_new(
            &directory.join("started.json"),
            b"{\"pid\": 999, \"pgid\": 999}",
            0o644,
        )
        .unwrap();
        assert_fail(
            do_release(&ctx, &release_request("1")),
            "running preparations bar hold release",
        );
        do_stop(&ctx, &op_value("stop", Some(PID))).unwrap();
        let released = do_release(&ctx, &release_request("1")).unwrap();
        assert_eq!(
            released
                .get("hold")
                .and_then(|h| h.get("active"))
                .and_then(|v| v.as_bool()),
            Some(false)
        );
        // Releasing an inactive hold is a no-op reporting the state.
        let again = do_release(&ctx, &release_request("1")).unwrap();
        assert_eq!(
            again
                .get("hold")
                .and_then(|h| h.get("revision"))
                .and_then(|v| v.as_str()),
            None
        );
        assert_eq!(
            crate::emit::dumps_default(&again),
            "{\"hold\": {\"active\": false, \"revision\": -1}}"
        );
        // Re-holding with another revision conflicts.
        do_hold(&ctx, &hold_request("7")).unwrap();
        assert_fail(
            do_hold(&ctx, &hold_request("8")),
            "maintenance hold already carries another revision",
        );
        assert_fail(
            do_hold(&ctx, &hold_request("true")),
            "unsupported hold request",
        );
        assert_fail(
            do_hold(&ctx, &hold_request("1.5")),
            "unsupported hold request",
        );
    }

    #[test]
    fn stop_bars_unknown_identity_and_retires_known() {
        let scratch = Scratch::fresh();
        let (git, _) = scratch.git_script("git-ok", 0);
        let ctx = scratch.ctx(&git);
        let stopped = do_stop(&ctx, &op_value("stop", Some(PID))).unwrap();
        assert_eq!(stopped.get("known").and_then(|v| v.as_bool()), Some(false));
        assert_eq!(
            stopped.get("retirement").and_then(|v| v.as_str()),
            Some("confirmed")
        );
        assert_fail(
            crate::ops_approve::do_approve(&ctx, &approve_default(PID)),
            "preparation was stopped; use a new identity",
        );
        crate::ops_approve::do_approve(&ctx, &approve_default(PID2)).unwrap();
        crate::ops_record::do_record(&ctx, &record_value(PID2, "", None)).unwrap();
        let directory = ctx.preparations.join(PID2);
        let pgid = dead_pgid();
        let started = format!("{{\"pid\": {pgid}, \"pgid\": {pgid}}}");
        crate::fsx::write_new(&directory.join("started.json"), started.as_bytes(), 0o644).unwrap();
        let stopped = do_stop(&ctx, &op_value("stop", Some(PID2))).unwrap();
        assert_eq!(stopped.get("known").and_then(|v| v.as_bool()), Some(true));
        assert_eq!(
            stopped.get("retirement").and_then(|v| v.as_str()),
            Some("confirmed")
        );
        let state = do_inspect(&ctx, &op_value("inspect", Some(PID2))).unwrap();
        assert_eq!(state.get("phase").and_then(|v| v.as_str()), Some("stopped"));
        assert_eq!(state.get("stopped").and_then(|v| v.as_bool()), Some(true));
    }

    #[test]
    fn inspect_reports_phases_and_hold() {
        let scratch = Scratch::fresh();
        let (git, _) = scratch.git_script("git-ok", 0);
        let ctx = scratch.ctx(&git);
        // Bare inspect reports the hold only.
        let bare = do_inspect(&ctx, &op_value("inspect", None)).unwrap();
        assert_eq!(
            crate::emit::dumps_default(&bare),
            "{\"hold\": {\"active\": false, \"revision\": -1}}"
        );
        // Unknown identities report known:false.
        let unknown = do_inspect(&ctx, &op_value("inspect", Some(PID))).unwrap();
        assert_eq!(unknown.get("known").and_then(|v| v.as_bool()), Some(false));
        // Approved, then waiting.
        crate::ops_approve::do_approve(&ctx, &approve_default(PID)).unwrap();
        let state = do_inspect(&ctx, &op_value("inspect", Some(PID))).unwrap();
        assert_eq!(
            state.get("phase").and_then(|v| v.as_str()),
            Some("approved")
        );
        assert_eq!(state.get("ready").and_then(|v| v.as_bool()), Some(false));
        assert_eq!(state.get("setup_log").and_then(|v| v.as_str()), Some(""));
        crate::ops_record::do_record(&ctx, &record_value(PID, "node22", None)).unwrap();
        let state = do_inspect(&ctx, &op_value("inspect", Some(PID))).unwrap();
        assert_eq!(state.get("phase").and_then(|v| v.as_str()), Some("waiting"));
        assert_eq!(
            state.get("missing").and_then(|v| v.as_str()),
            Some("node22")
        );
        // Launcher refusal fails without any start.
        crate::ops_approve::do_approve(&ctx, &approve_default(PID2)).unwrap();
        crate::ops_record::do_record(
            &ctx,
            &record_value(PID2, "", Some("role holds unexpected groups")),
        )
        .unwrap();
        let state = do_inspect(&ctx, &op_value("inspect", Some(PID2))).unwrap();
        assert_eq!(state.get("phase").and_then(|v| v.as_str()), Some("failed"));
        // Finished outcomes surface exits and readiness.
        let directory = ctx.preparations.join(PID);
        crate::fsx::write_new(
            &directory.join("finished.json"),
            b"{\"setup_exit\": 0, \"check_exit\": 0}",
            0o644,
        )
        .unwrap();
        std::fs::write(directory.join("setup.log"), b"setup out\n").unwrap();
        let state = do_inspect(&ctx, &op_value("inspect", Some(PID))).unwrap();
        assert_eq!(state.get("phase").and_then(|v| v.as_str()), Some("ready"));
        assert_eq!(state.get("ready").and_then(|v| v.as_bool()), Some(true));
        assert_eq!(
            state.get("setup_log").and_then(|v| v.as_str()),
            Some("setup out\n")
        );
        assert_eq!(
            crate::emit::dumps_default(state.get("setup_exit").unwrap()),
            "0"
        );
        assert_fail(
            do_inspect(&ctx, &op_value("inspect", Some("../x"))),
            "unsupported preparation identity",
        );
    }

    #[test]
    fn metadata_hardening_refuses_links_and_aliases() {
        let scratch = Scratch::fresh();
        let (git, _) = scratch.git_script("git-ok", 0);
        let ctx = scratch.ctx(&git);
        crate::ops_approve::do_approve(&ctx, &approve_default(PID)).unwrap();
        let directory = ctx.preparations.join(PID);
        // A hardlinked receipt (nlink 2) is unsafe metadata.
        let alias = directory.join("alias.json");
        std::fs::hard_link(directory.join("request.json"), &alias).unwrap();
        match crate::fsx::read_json(&ctx, &alias, 4096) {
            Err(Error::Fail(text)) => assert_eq!(text, "unsafe factory metadata: alias.json"),
            other => panic!("unexpected {other:?}"),
        }
        std::fs::remove_file(&alias).unwrap();
        // A symlinked receipt fails the open, never reads as missing.
        let link = directory.join("link.json");
        std::os::unix::fs::symlink(directory.join("request.json"), &link).unwrap();
        assert!(matches!(
            crate::fsx::read_json(&ctx, &link, 4096),
            Err(Error::Io(_))
        ));
        // Oversized metadata is refused with the file named.
        let big = directory.join("big.json");
        std::fs::write(&big, vec![b'x'; 1025]).unwrap();
        match crate::fsx::read_json(&ctx, &big, 1024) {
            Err(Error::Fail(text)) => assert_eq!(text, "oversized factory metadata: big.json"),
            other => panic!("unexpected {other:?}"),
        }
        // A symlinked log fails instead of reading through.
        let log_link = directory.join("setup.log");
        std::os::unix::fs::symlink("/etc/hostname", &log_link).unwrap();
        assert!(read_log(&ctx, &log_link).is_err());
    }
}
