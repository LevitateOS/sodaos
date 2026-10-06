use super::*;
use crate::records::{do_inspect, read_log};
use crate::testutil::{approve_default, assert_fail, op_value, record_value, Scratch, PID, PID2};
use soda_json::JsonValue;
use std::path::Path;

#[test]
fn parent_matrix() {
    assert_eq!(parent_of("/usr/bin/go"), "/usr/bin");
    assert_eq!(parent_of("go"), ".");
    assert_eq!(parent_of(""), ".");
    assert_eq!(parent_of("/"), "/");
    assert_eq!(parent_of("///"), "/");
    assert_eq!(parent_of("/go"), "/");
    assert_eq!(parent_of("a/"), ".");
    assert_eq!(parent_of("a/b/"), "a");
    assert_eq!(parent_of("/a/b"), "/a");
}

#[test]
fn child_environment_is_fixed_and_ordered() {
    let environ = child_environ("/bin", "/h", "", "soda-coder", "f01", "/s");
    let keys: Vec<&str> = environ.iter().map(|(k, _)| k.as_str()).collect();
    assert_eq!(
        keys,
        vec![
            "PATH",
            "HOME",
            "TMPDIR",
            "SODA_ROLE",
            "SODA_PREPARATION_ID",
            "SODA_SNAPSHOT",
        ]
    );
    assert_eq!(environ[2].1, "/h/tmp");
    let with_cred = child_environ("/bin", "/h", "/c", "soda-coder", "f01", "/s");
    assert_eq!(with_cred.len(), 7);
    assert_eq!(
        with_cred[6],
        ("SODA_CREDENTIAL_FILE".to_string(), "/c".to_string())
    );
}

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

fn own_child_pids() -> Vec<i32> {
    let tid = unsafe { libc::gettid() };
    let path = format!("/proc/self/task/{tid}/children");
    std::fs::read_to_string(&path)
        .unwrap_or_default()
        .split_whitespace()
        .filter_map(|pid| pid.parse().ok())
        .collect()
}

fn proc_comm(pid: i32) -> Option<String> {
    let stat = std::fs::read_to_string(format!("/proc/{pid}/stat")).ok()?;
    let end = stat.rfind(')')?;
    let start = stat[..end].rfind('(')?;
    Some(stat[start + 1..end].to_string())
}

/// P07-F1: a log-I/O failure must reap the role child (never leak a
/// live/unreaped child) and report unconfirmed (Err, never a receipt).
/// `yes` never exits on its own, so a leaked child is always observable;
/// closing the pipe reader makes it die promptly once reaped.
#[test]
fn log_io_failure_reaps_child_and_reports_unconfirmed() {
    assert!(
        std::fs::metadata("/usr/bin/yes").is_ok(),
        "P07-F1 regression needs /usr/bin/yes"
    );
    let dir = std::env::temp_dir().join(format!("soda-f1-{}", std::process::id()));
    std::fs::create_dir_all(&dir).unwrap();
    let logpath = dir.join("no-such-dir").join("setup.log");
    let argv = vec!["/usr/bin/yes".to_string()];
    let uid = unsafe { libc::geteuid() };
    let gid = unsafe { libc::getegid() };
    let outcome = run_as_role(&argv, &[], &dir, &logpath, uid, gid);
    assert!(
        outcome.is_err(),
        "log I/O failure must report unconfirmed, got {outcome:?}"
    );
    // Allow exec latency, then scan own children for the role child.
    std::thread::sleep(std::time::Duration::from_millis(300));
    let mut leaked = Vec::new();
    let deadline = std::time::Instant::now() + std::time::Duration::from_millis(700);
    while leaked.is_empty() && std::time::Instant::now() < deadline {
        for pid in own_child_pids() {
            if proc_comm(pid).as_deref() == Some("yes") {
                leaked.push(pid);
            }
        }
        if leaked.is_empty() {
            std::thread::sleep(std::time::Duration::from_millis(20));
        }
    }
    for pid in &leaked {
        unsafe {
            libc::kill(*pid, libc::SIGKILL);
        }
    }
    for pid in &leaked {
        unsafe {
            libc::waitpid(*pid, std::ptr::null_mut(), 0);
        }
    }
    std::fs::remove_dir_all(&dir).ok();
    assert!(
        leaked.is_empty(),
        "log I/O failure leaked role children: {leaked:?}"
    );
}

/// CODEX-P07-001: the `yes` regression proves only SIGPIPE exit. A silent
/// child that never writes must still be terminated and reaped on a log
/// failure within a bound — never hang the privileged supervisor in an
/// unbounded wait. Pre-fix this fails on the elapsed watchdog (the call
/// blocks until the child exits on its own).
#[test]
fn log_io_failure_terminates_silent_child_bounded() {
    assert!(
        std::fs::metadata("/bin/sleep").is_ok(),
        "P07-001 regression needs /bin/sleep"
    );
    let dir = std::env::temp_dir().join(format!("soda-f1s-{}", std::process::id()));
    std::fs::create_dir_all(&dir).unwrap();
    let logpath = dir.join("no-such-dir").join("setup.log");
    let argv = vec!["/bin/sleep".to_string(), "10".to_string()];
    let uid = unsafe { libc::geteuid() };
    let gid = unsafe { libc::getegid() };
    let start = std::time::Instant::now();
    let outcome = run_as_role(&argv, &[], &dir, &logpath, uid, gid);
    let elapsed = start.elapsed();
    assert!(
        outcome.is_err(),
        "log I/O failure must report unconfirmed, got {outcome:?}"
    );
    // Owned-process cleanup first: reap any leaked silent child, then fail.
    std::thread::sleep(std::time::Duration::from_millis(300));
    let mut leaked = Vec::new();
    let deadline = std::time::Instant::now() + std::time::Duration::from_millis(700);
    while leaked.is_empty() && std::time::Instant::now() < deadline {
        for pid in own_child_pids() {
            if proc_comm(pid).as_deref() == Some("sleep") {
                leaked.push(pid);
            }
        }
        if leaked.is_empty() {
            std::thread::sleep(std::time::Duration::from_millis(20));
        }
    }
    for pid in &leaked {
        unsafe {
            libc::kill(*pid, libc::SIGKILL);
        }
    }
    for pid in &leaked {
        unsafe {
            libc::waitpid(*pid, std::ptr::null_mut(), 0);
        }
    }
    std::fs::remove_dir_all(&dir).ok();
    assert!(
        leaked.is_empty(),
        "log I/O failure leaked silent children: {leaked:?}"
    );
    assert!(
        elapsed < std::time::Duration::from_secs(5),
        "silent child hung the supervisor: {elapsed:?}"
    );
}

/// CODEX-P07-001: cleanup that cannot confirm the child reports
/// uncertainty (never Ok, never the original error alone). A reaped pid is
/// deterministically gone, so this pins the unconfirmed mapping without
/// fabricating native state.
#[test]
fn log_failure_cleanup_reports_unconfirmed_when_child_gone() {
    let pid = unsafe { libc::fork() };
    assert!(pid >= 0, "fork failed");
    if pid == 0 {
        unsafe {
            libc::_exit(0);
        }
    }
    let mut status = 0;
    assert_eq!(unsafe { libc::waitpid(pid, &mut status, 0) }, pid);
    assert_eq!(terminate_child(pid), None);
    assert_eq!(
        log_failure_cleanup(pid, Error::io_msg("write: broken pipe")),
        Error::Fail("log capture failed; role child termination unconfirmed".to_string()),
    );
}

/// CODEX-P07-001b: no signal may follow ownership loss. A helper thread
/// reaps the child inside the TERM-grace window; the iteration is
/// conclusive when the main waiter observes the loss before grace expiry
/// (an expired grace legitimately escalates, and is skipped), and then no
/// SIGKILL for the lost pid may have been sent. Pre-fix the post-TERM
/// SIGKILL is unconditional and the signal log proves it.
#[test]
fn terminate_child_never_signals_after_ownership_loss() {
    assert!(
        std::fs::metadata("/bin/sleep").is_ok(),
        "P07-001b regression needs /bin/sleep"
    );
    let mut conclusive = 0;
    for _ in 0..25 {
        let pid = unsafe { libc::fork() };
        assert!(pid >= 0, "fork failed");
        if pid == 0 {
            unsafe {
                let argv0 = c"/bin/sleep".as_ptr();
                let argv1 = c"30".as_ptr();
                let argv = [argv0, argv1, std::ptr::null()];
                let envp = [std::ptr::null()];
                libc::execve(argv0, argv.as_ptr(), envp.as_ptr());
                libc::_exit(127);
            }
        }
        // A second waiter reaps the child as soon as it dies, so the
        // main waiter can observe ownership loss mid-grace.
        let helper = std::thread::spawn(move || {
            let mut status = 0;
            unsafe { libc::waitpid(pid, &mut status, 0) };
        });
        super::KILL_LOG.lock().unwrap().clear();
        let start = std::time::Instant::now();
        let outcome = terminate_child(pid);
        let elapsed = start.elapsed();
        helper.join().expect("helper joined");
        if outcome.is_some() {
            continue; // Main waiter won the race: inconclusive, retry.
        }
        if elapsed >= std::time::Duration::from_millis(1000) {
            continue; // Grace expired while owned: KILL legitimate here.
        }
        conclusive += 1;
        let logged_kill = super::KILL_LOG
            .lock()
            .unwrap()
            .iter()
            .any(|(logged_pid, signal)| *logged_pid == pid && *signal == libc::SIGKILL);
        assert!(
            !logged_kill,
            "SIGKILL sent after ownership of {pid} was lost"
        );
    }
    assert!(conclusive > 0, "loss window never observed; test vacuous");
}

/// CODEX-P07-001b characterization: the bounded wait distinguishes a
/// reaped exit, an owned child outliving its deadline, and ownership
/// loss. Green-post only (the outcomes are the fix's new surface).
#[test]
fn reap_until_distinguishes_reaped_alive_and_gone() {
    // Gone: an already-reaped pid reports loss without waiting.
    let pid = unsafe { libc::fork() };
    assert!(pid >= 0, "fork failed");
    if pid == 0 {
        unsafe {
            libc::_exit(0);
        }
    }
    let mut status = 0;
    assert_eq!(unsafe { libc::waitpid(pid, &mut status, 0) }, pid);
    assert_eq!(
        reap_until(
            pid,
            std::time::Instant::now() + std::time::Duration::from_secs(5)
        ),
        ReapOutcome::Gone
    );
    // Alive: a live owned child with an expired deadline reports owned.
    assert!(
        std::fs::metadata("/bin/sleep").is_ok(),
        "P07-001b outcomes need /bin/sleep"
    );
    let live = unsafe { libc::fork() };
    assert!(live >= 0, "fork failed");
    if live == 0 {
        unsafe {
            let argv0 = c"/bin/sleep".as_ptr();
            let argv1 = c"30".as_ptr();
            let argv = [argv0, argv1, std::ptr::null()];
            let envp = [std::ptr::null()];
            libc::execve(argv0, argv.as_ptr(), envp.as_ptr());
            libc::_exit(127);
        }
    }
    assert_eq!(
        reap_until(live, std::time::Instant::now()),
        ReapOutcome::Alive
    );
    signal_child(live, libc::SIGKILL);
    match reap_until(
        live,
        std::time::Instant::now() + std::time::Duration::from_secs(5),
    ) {
        ReapOutcome::Reaped(status) => assert!(
            libc::WIFSIGNALED(status) && libc::WTERMSIG(status) == libc::SIGKILL,
            "reaped exit reports the wait status, got {status}"
        ),
        other => panic!("expected Reaped, got {other:?}"),
    }
}

/// CODEX-P07-002: an uncertain stop (tombstone present, supervisor group
/// still live) must bar hold release. Pre-fix `any_running` trusts the
/// tombstone alone and the release succeeds.
#[test]
fn uncertain_stop_bars_hold_release() {
    assert!(
        std::fs::metadata("/bin/sleep").is_ok(),
        "P07-002 regression needs /bin/sleep"
    );
    let pid = unsafe { libc::fork() };
    assert!(pid >= 0, "fork failed");
    if pid == 0 {
        unsafe {
            libc::setpgid(0, 0);
            let argv0 = c"/bin/sleep".as_ptr();
            let argv1 = c"30".as_ptr();
            let argv = [argv0, argv1, std::ptr::null()];
            let envp = [std::ptr::null()];
            libc::execve(argv0, argv.as_ptr(), envp.as_ptr());
            libc::_exit(127);
        }
    }
    // The group exists once the child leads it; poll before asserting.
    let deadline = std::time::Instant::now() + std::time::Duration::from_secs(5);
    while !group_alive(pid, std::path::Path::new("/proc")) {
        assert!(
            std::time::Instant::now() < deadline,
            "supervisor group never appeared"
        );
        std::thread::sleep(std::time::Duration::from_millis(20));
    }
    let scratch = Scratch::fresh();
    let (git, _) = scratch.git_script("git-ok", 0);
    let ctx = scratch.ctx(&git);
    let dir = ctx.preparations.join(PID);
    std::fs::create_dir_all(&dir).unwrap();
    let started = format!("{{\"pid\": {pid}, \"pgid\": {pid}}}");
    crate::fsx::write_new(&dir.join("started.json"), started.as_bytes(), 0o644).unwrap();
    crate::fsx::write_new(&dir.join("stopped.json"), b"{\"stopped\": true}", 0o644).unwrap();
    do_hold(&ctx, &hold_request("1")).unwrap();
    let result = do_release(&ctx, &release_request("1"));
    // Owned-group cleanup before asserting (the sleep exits alone on panic).
    let _ = killpg(pid, libc::SIGKILL);
    unsafe {
        libc::waitpid(pid, std::ptr::null_mut(), 0);
    }
    assert_fail(result, "running preparations bar hold release");
}

/// CODEX-P07-002: the release probe maps quiet/active/unreadable native
/// state, and the record gate bars release on unreadable state.
#[test]
fn release_probe_maps_native_states() {
    // Unknown: unreadable process root.
    assert_eq!(
        probe_group_quiet(1, Path::new("/nonexistent-soda-proot")),
        None
    );
    let root = std::env::temp_dir().join(format!("soda-p2-{}", std::process::id()));
    std::fs::create_dir_all(root.join("4242")).unwrap();
    // Quiet: sole member in another group.
    std::fs::write(
        root.join("4242").join("stat"),
        b"4242 (test) S 1 777 777 0 -1 0 0 0 0 0 0 0 0 0 0 0 1 0 0 0 0 0 0 0 0 0",
    )
    .unwrap();
    assert_eq!(probe_group_quiet(999, &root), Some(true));
    // Active: non-zombie member keeps the group.
    std::fs::write(
        root.join("4242").join("stat"),
        b"4242 (test) S 1 999 999 0 -1 0 0 0 0 0 0 0 0 0 0 0 1 0 0 0 0 0 0 0 0 0",
    )
    .unwrap();
    assert_eq!(probe_group_quiet(999, &root), Some(false));
    // Zombies hold no scope: quiet despite the recorded group.
    std::fs::write(
        root.join("4242").join("stat"),
        b"4242 (test) Z 1 999 999 0 -1 0 0 0 0 0 0 0 0 0 0 0 1 0 0 0 0 0 0 0 0 0",
    )
    .unwrap();
    assert_eq!(probe_group_quiet(999, &root), Some(true));
    // Unreadable stat hides a potential member: uncertainty.
    if unsafe { libc::geteuid() } == 0 {
        eprintln!("SKIP stat-unreadable probe case: root reads through 000");
    } else {
        let stat = root.join("4242").join("stat");
        std::fs::set_permissions(&stat, std::os::unix::fs::PermissionsExt::from_mode(0o000))
            .unwrap();
        assert_eq!(probe_group_quiet(999, &root), None);
        std::fs::set_permissions(&stat, std::os::unix::fs::PermissionsExt::from_mode(0o644))
            .unwrap();
    }
    std::fs::remove_dir_all(&root).ok();
    // Record-level mapping: no record never bars; unreadable native
    // state with a record on file always bars.
    let scratch = Scratch::fresh();
    let (git, _) = scratch.git_script("git-ok", 0);
    let ctx = scratch.ctx(&git);
    let dir = ctx.preparations.join(PID);
    std::fs::create_dir_all(&dir).unwrap();
    assert!(!preparation_blocks_release(
        &ctx,
        &dir,
        Path::new("/nonexistent-soda-proot")
    ));
    crate::fsx::write_new(
        &dir.join("started.json"),
        b"{\"pid\": 4242, \"pgid\": 999}",
        0o644,
    )
    .unwrap();
    assert!(preparation_blocks_release(
        &ctx,
        &dir,
        Path::new("/nonexistent-soda-proot")
    ));
}

/// CODEX-P07-002 guard: terminal markers with a positively quiet group
/// must not wedge the hold (an impossible group is deterministically
/// quiet). Green before and after; proves the gate is not block-always.
#[test]
fn release_after_quiesced_preparation_succeeds() {
    let scratch = Scratch::fresh();
    let (git, _) = scratch.git_script("git-ok", 0);
    let ctx = scratch.ctx(&git);
    let dir = ctx.preparations.join(PID);
    std::fs::create_dir_all(&dir).unwrap();
    crate::fsx::write_new(
        &dir.join("started.json"),
        b"{\"pid\": 1, \"pgid\": 1073741824}",
        0o644,
    )
    .unwrap();
    crate::fsx::write_new(&dir.join("stopped.json"), b"{\"stopped\": true}", 0o644).unwrap();
    do_hold(&ctx, &hold_request("1")).unwrap();
    let released = do_release(&ctx, &release_request("1")).unwrap();
    assert_eq!(
        released
            .get("hold")
            .and_then(|h| h.get("active"))
            .and_then(|v| v.as_bool()),
        Some(false)
    );
}

fn proc_startup() -> Option<(u32, i32, String)> {
    let meta = std::fs::symlink_metadata("/proc/1").ok()?;
    let uid = std::os::unix::fs::MetadataExt::uid(&meta);
    let stat = std::fs::read_to_string("/proc/1/stat").ok()?;
    let end = stat.rfind(')')?;
    let rest = stat[end + 2..].to_string();
    let mut fields = rest.split_whitespace();
    let state = fields.next()?.to_string();
    let _ppid = fields.next()?;
    let pgid: i32 = fields.next()?.parse().ok()?;
    Some((uid, pgid, state))
}

/// P07-F1 receipt side: a launch-error receipt with a live but foreign
/// (unsignallable-by-us) group must report uncertain, never confirmed.
/// Skips honestly when the environment cannot fabricate that premise.
#[test]
fn stop_launch_error_with_live_foreign_group_is_uncertain() {
    let me = unsafe { libc::geteuid() };
    let (uid1, pgid1, state1) = match proc_startup() {
        Some(v) => v,
        None => {
            eprintln!("SKIP: cannot read /proc/1 startup facts");
            return;
        }
    };
    if uid1 == me || state1 == "Z" || !group_alive(pgid1, Path::new("/proc")) {
        eprintln!("SKIP: no live foreign supervisor group to observe");
        return;
    }
    let scratch = Scratch::fresh();
    let (git, _) = scratch.git_script("git-ok", 0);
    let ctx = scratch.ctx(&git);
    crate::ops_approve::do_approve(&ctx, &approve_default(PID)).unwrap();
    crate::ops_record::do_record(&ctx, &record_value(PID, "", None)).unwrap();
    let directory = ctx.preparations.join(PID);
    let started = format!("{{\"pid\": 1, \"pgid\": {pgid1}}}");
    crate::fsx::write_new(&directory.join("started.json"), started.as_bytes(), 0o644).unwrap();
    crate::fsx::write_new(
        &directory.join("finished.json"),
        b"{\"setup_exit\": 127, \"check_exit\": null, \"launch_error\": true}",
        0o644,
    )
    .unwrap();
    let stopped = do_stop(&ctx, &op_value("stop", Some(PID))).unwrap();
    assert_eq!(
        stopped.get("retirement").and_then(|v| v.as_str()),
        Some("uncertain")
    );
}
