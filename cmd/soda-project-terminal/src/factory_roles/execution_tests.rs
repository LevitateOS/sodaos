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
