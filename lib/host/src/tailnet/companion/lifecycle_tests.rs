use super::tests::{
    companion_with, deadline, file_run, image_id, project_id, project_inspect_json, test_view,
    MockExec, MockTailnet,
};
use super::*;
use crate::tailnet_domain::{ERR_CONFLICT, ERR_INVALID, ERR_UNAVAILABLE, ERR_UNCONFIRMED};
use std::cell::RefCell;
use std::time::Duration;

#[test]
fn wait_tailnet_output_matrix() {
    // Empty CID is a no-op without any call.
    let exec = MockExec::with_handler(|_, _| panic!("unexpected call"));
    let c = companion_with(exec, MockTailnet::inert(), &image_id());
    assert!(c.wait_tailnet("", deadline()).is_ok());
    assert!(c.exec.calls().is_empty());
    // Malformed CIDs are rejected before any call.
    assert_eq!(
        c.wait_tailnet("short", deadline()),
        Err(ERR_INVALID.to_string())
    );
    assert!(c.exec.calls().is_empty());

    for (output, want) in [
        (Ok(b"0\n".to_vec()), Err(ERR_UNAVAILABLE.to_string())),
        (Ok(b"0".to_vec()), Err(ERR_UNAVAILABLE.to_string())),
        (Ok(b"1\n".to_vec()), Err(ERR_UNCONFIRMED.to_string())),
        (Ok(b"stopped\n".to_vec()), Err(ERR_UNCONFIRMED.to_string())),
        (Ok(vec![b'x'; 65537]), Err(ERR_UNCONFIRMED.to_string())),
        (Err("boom".to_string()), Err(ERR_UNCONFIRMED.to_string())),
    ] {
        let exec = MockExec::with_handler(move |cmd, args| {
            assert_eq!(cmd, "/usr/bin/podman");
            assert_eq!(args[0], "--remote=false");
            assert_eq!(args[1], "wait");
            assert_eq!(args[2], "--condition=stopped");
            output.clone()
        });
        let c = companion_with(exec, MockTailnet::inert(), &image_id());
        assert_eq!(c.wait_tailnet(&"e".repeat(64), deadline()), want);
    }
    // An expired deadline reports success like Go's ctx.Err check, even
    // when the supervisor output disagrees.
    let exec = MockExec::with_handler(|_, _| Ok(b"1\n".to_vec()));
    let c = companion_with(exec, MockTailnet::inert(), &image_id());
    assert!(c.wait_tailnet(&"e".repeat(64), Instant::now()).is_ok());
}

#[test]
fn native_commands_never_leak_diagnostics() {
    let exec = MockExec {
        calls: RefCell::new(Vec::new()),
        handler: Box::new(|_, _| panic!("native path must not use the executor")),
        native: true,
    };
    let c = companion_with(exec, MockTailnet::inert(), &image_id());
    let out = c
        .run_native_command("/bin/echo", &["hello".to_string()], deadline())
        .unwrap();
    assert_eq!(out, b"hello\n");
    assert_eq!(
        c.run_native_command("/bin/false", &[], deadline()),
        Err(ERR_UNCONFIRMED.to_string())
    );
    assert_eq!(
        c.run_native_command("/nonexistent-soda-binary", &[], deadline()),
        Err(ERR_UNCONFIRMED.to_string())
    );
    // Exactly at the cap passes; one byte over does not.
    let out = c
        .run_native_command(
            "/bin/sh",
            &["-c".to_string(), "head -c 65536 /dev/zero".to_string()],
            deadline(),
        )
        .unwrap();
    assert_eq!(out.len(), 65536);
    assert_eq!(
        c.run_native_command(
            "/bin/sh",
            &["-c".to_string(), "head -c 70000 /dev/zero".to_string()],
            deadline(),
        ),
        Err(ERR_UNCONFIRMED.to_string())
    );
    // Deadline kill collapses to unconfirmed.
    assert_eq!(
        c.run_native_command(
            "/bin/sleep",
            &["30".to_string()],
            Instant::now() + Duration::from_millis(100)
        ),
        Err(ERR_UNCONFIRMED.to_string())
    );
}

#[test]
fn start_skipped_when_image_empty() {
    let exec = MockExec::with_handler(|_, _| panic!("unexpected exec call"));
    let c = companion_with(exec, MockTailnet::inert(), "");
    assert_eq!(
        c.start_tailnet(&project_id(), deadline()),
        Ok(String::new())
    );
    assert!(c.exec.calls().is_empty());
    assert_eq!(c.should_start_tailnet(&project_id(), deadline()), Ok(false));
}

#[test]
fn should_start_tailnet_policy_gates() {
    // An unresolvable container falls through to startup.
    let exec = MockExec::with_handler(|_, _| Err("no container".to_string()));
    let c = companion_with(exec, MockTailnet::inert(), &image_id());
    assert_eq!(c.should_start_tailnet(&project_id(), deadline()), Ok(true));

    let id = project_id();
    let cid = "b".repeat(64);
    for enabled in [false, true] {
        let id_clone = id.clone();
        let cid_clone = cid.clone();
        let exec = MockExec::with_handler(move |cmd, args| {
            assert_eq!(cmd, "/usr/bin/podman");
            assert!(args.last().unwrap() == &format!("soda-{id_clone}"));
            Ok(project_inspect_json(&id_clone, &cid_clone, true))
        });
        let mut c = companion_with(exec, MockTailnet::inert(), &image_id());
        let want_id = id.clone();
        let want_cid = cid.clone();
        c.enabled_check = Some(Box::new(move |project, seen_cid, _| {
            assert_eq!(project, want_id);
            assert_eq!(seen_cid, want_cid);
            Ok(enabled)
        }));
        assert_eq!(
            c.should_start_tailnet(&project_id(), deadline()),
            Ok(enabled)
        );
    }
    // Without an override the policy service decides.
    let id_clone = id.clone();
    let cid_clone = cid.clone();
    let exec = MockExec::with_handler(move |_, args| {
        assert!(args.last().unwrap() == &format!("soda-{id_clone}"));
        Ok(project_inspect_json(&id_clone, &cid_clone, true))
    });
    let tailnet = MockTailnet {
        project_fn: Box::new(|req, _| {
            assert_eq!(req.action, "inspect");
            let mut view = test_view();
            view.enabled = true;
            Ok(view)
        }),
        binding_fn: Box::new(|_| panic!("unexpected run_binding call")),
        enroll_targets: RefCell::new(Vec::new()),
        enroll_err: None,
    };
    let c = companion_with(exec, tailnet, &image_id());
    assert_eq!(c.project_tailnet_enabled(&id, &cid, deadline()), Ok(true));
    assert_eq!(c.should_start_tailnet(&id, deadline()), Ok(true));
}

#[test]
fn companion_cli_rejects_empty_args_and_missing_state() {
    let exec = MockExec::with_handler(|_, _| panic!("unexpected call"));
    let c = companion_with(exec, MockTailnet::inert(), &image_id());
    let run = file_run();
    assert_eq!(
        c.companion_cli(&run, &[], deadline()),
        Err(ERR_INVALID.to_string())
    );
    // No runtime record exists in the test environment, so the CLI is
    // unavailable before any supervisor call.
    assert_eq!(
        c.companion_cli(&run, &["status".to_string()], deadline()),
        Err(ERR_UNAVAILABLE.to_string())
    );
    assert!(c.exec.calls().is_empty());
}

#[test]
fn logout_and_stop_reports_unconfirmed_logout() {
    // Logout fails without runtime state; the stop argv is still exact.
    let exec = MockExec::with_handler(|cmd, args| {
        assert_eq!(cmd, "/usr/bin/podman");
        assert_eq!(
            args,
            &[
                "--remote=false".to_string(),
                "stop".to_string(),
                "--time=8".to_string(),
                "e".repeat(64),
            ]
        );
        Ok(Vec::new())
    });
    let c = companion_with(exec, MockTailnet::inert(), &image_id());
    assert_eq!(
        c.logout_and_stop_companion(&file_run(), &"e".repeat(64), deadline()),
        Err(ERR_UNCONFIRMED.to_string())
    );
    let exec = MockExec::with_handler(|_, _| Err("stop failed".to_string()));
    let c = companion_with(exec, MockTailnet::inert(), &image_id());
    assert_eq!(
        c.logout_and_stop_companion(&file_run(), &"e".repeat(64), deadline()),
        Err(ERR_UNCONFIRMED.to_string())
    );
}

#[test]
fn retire_previous_companion_guards_identity() {
    let exec = MockExec::with_handler(|_, _| panic!("unexpected call"));
    let c = companion_with(exec, MockTailnet::inert(), &image_id());
    let run = file_run();
    let empty = ProjectRun {
        target: RunTarget {
            project: String::new(),
            container: String::new(),
            run: String::new(),
        },
        pid: 0,
        started: String::new(),
        userns: String::new(),
        netns: String::new(),
        uid: 0,
        gid: 0,
        resolver: String::new(),
    };
    assert!(c
        .retire_previous_companion(&project_id(), &run, &empty, deadline())
        .is_ok());
    assert!(c.exec.calls().is_empty());
    let mut foreign = run.clone();
    foreign.target.project = format!("p{}", "f".repeat(24));
    let err = c
        .retire_previous_companion(&project_id(), &run, &foreign, deadline())
        .unwrap_err();
    assert!(err.starts_with("project runtime changed: "), "{err}");
    assert!(err.contains(ERR_CONFLICT), "{err}");
    // A live mismatch against the supervisor is a bare conflict.
    let exec = MockExec::with_handler(|_, _| {
        Ok(project_inspect_json(&project_id(), &"f".repeat(64), true))
    });
    let c = companion_with(exec, MockTailnet::inert(), &image_id());
    assert_eq!(
        c.retire_previous_companion(&project_id(), &run, &run, deadline())
            .unwrap_err(),
        stage_error("companion stop unconfirmed", ERR_CONFLICT.to_string())
    );
}

#[test]
fn reconcile_previous_run_freshness() {
    let exec = MockExec::with_handler(|_, _| panic!("unexpected call"));
    let c = companion_with(exec, MockTailnet::inert(), &image_id());
    let run = file_run();
    // A missing record with no previous run is fresh.
    let empty = ProjectRun {
        target: RunTarget {
            project: String::new(),
            container: String::new(),
            run: String::new(),
        },
        pid: 0,
        started: String::new(),
        userns: String::new(),
        netns: String::new(),
        uid: 0,
        gid: 0,
        resolver: String::new(),
    };
    assert_eq!(
        c.reconcile_previous_run(
            &project_id(),
            &run,
            &empty,
            Some("runtime record not found".to_string()),
            deadline()
        ),
        Ok(true)
    );
    // Any other current-record failure is unconfirmed runtime.
    let err = c
        .reconcile_previous_run(
            &project_id(),
            &run,
            &empty,
            Some(ERR_UNAVAILABLE.to_string()),
            deadline(),
        )
        .unwrap_err();
    assert!(err.starts_with("companion runtime unconfirmed: "), "{err}");
    // A present record for the same run with drifted fields is a change.
    let mut drifted = run.clone();
    drifted.pid = 78;
    let err = c
        .reconcile_previous_run(&project_id(), &run, &drifted, None, deadline())
        .unwrap_err();
    assert!(err.starts_with("project runtime changed: "), "{err}");
    assert_eq!(
        c.reconcile_previous_run(&project_id(), &run, &run, None, deadline()),
        Ok(false)
    );
    // A present record for another run retires the previous companion.
    let mut other = run.clone();
    other.target.run = "d".repeat(64);
    let exec = MockExec::with_handler(|_, _| {
        Ok(project_inspect_json(&project_id(), &"f".repeat(64), true))
    });
    let c = companion_with(exec, MockTailnet::inert(), &image_id());
    let err = c
        .reconcile_previous_run(&project_id(), &run, &other, None, deadline())
        .unwrap_err();
    assert!(err.starts_with("companion stop unconfirmed: "), "{err}");
}

#[test]
fn prepare_and_finalize_stage_fast_failures() {
    let exec = MockExec::with_handler(|_, _| panic!("unexpected call"));
    let c = companion_with(exec, MockTailnet::inert(), &image_id());
    let err = c
        .prepare_companion_state(&project_id(), &file_run(), deadline())
        .expect_err("prepare fails without runtime state");
    assert!(err.starts_with("companion runtime unconfirmed: "), "{err}");
    let err = c
        .start_companion_if_stopped(&file_run(), deadline())
        .unwrap_err();
    assert!(err.starts_with("companion startup unconfirmed: "), "{err}");

    // Finalize rechecks first; an unresolvable parent fails fast.
    let exec = MockExec::with_handler(|_, _| Err("no container".to_string()));
    let c = companion_with(exec, MockTailnet::inert(), &image_id());
    let err = c
        .finalize_companion_run(&file_run(), deadline())
        .unwrap_err();
    assert!(err.starts_with("project runtime changed: "), "{err}");
}

#[test]
fn stop_tailnet_validates_and_needs_state() {
    let exec = MockExec::with_handler(|_, _| panic!("unexpected call"));
    let c = companion_with(exec, MockTailnet::inert(), &image_id());
    assert_eq!(
        c.stop_tailnet("bad-id", deadline()),
        Err(ERR_INVALID.to_string())
    );
    assert!(c.exec.calls().is_empty());
    // No runtime record exists in the test environment.
    assert!(c.stop_tailnet(&project_id(), deadline()).is_err());
    // A supervisor mismatch against the recorded container conflicts.
    let exec = MockExec::with_handler(|_, _| {
        Ok(project_inspect_json(&project_id(), &"f".repeat(64), true))
    });
    let c = companion_with(exec, MockTailnet::inert(), &image_id());
    assert_eq!(
        c.stop_tailnet_run(&file_run(), deadline()),
        Err(ERR_CONFLICT.to_string())
    );
}
