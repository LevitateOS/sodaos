use super::*;

use crate::build_spec::Request;

fn flags(args: &[&str]) -> Result<BuildFlags, String> {
    parse_build_flags(&args.iter().map(|s| s.to_string()).collect::<Vec<_>>())
}

#[test]
fn live_inputs_admission_boundary() {
    let worker = BuildFlags {
        worker_build: true,
        request: Request {
            out: "/o".to_owned(),
            arch: "x86_64".to_owned(),
            ..Request::default()
        },
        ..BuildFlags::default()
    };
    assert!(admit_worker_build(&worker)
        .unwrap_err()
        .contains("requires controller-resolved live inputs"));
    let parent = BuildFlags {
        worker_config: "/cfg".to_owned(),
        request: Request {
            out: "/o".to_owned(),
            arch: "x86_64".to_owned(),
            live_inputs: "/inputs.json".to_owned(),
            ..Request::default()
        },
        ..BuildFlags::default()
    };
    assert!(admit_parent_dispatch(&parent)
        .unwrap_err()
        .contains("operator selection refused"));
}

#[test]
fn admit_internal_dispatch_binds_authority() {
    let bad = [
        BuildFlags {
            worker_build: true,
            worker_config: "/restricted/worker.json".to_owned(),
            ..BuildFlags::default()
        },
        BuildFlags {
            worker_build: true,
            ..BuildFlags::default()
        },
        BuildFlags::default(),
        BuildFlags {
            worker_config: "/cfg".to_owned(),
            request: Request {
                media_authority: "x".to_owned(),
                ..Request::default()
            },
            ..BuildFlags::default()
        },
        BuildFlags {
            worker_config: "/cfg".to_owned(),
            request: Request {
                live_inputs: "x".to_owned(),
                ..Request::default()
            },
            ..BuildFlags::default()
        },
        BuildFlags {
            worker_config: "/cfg".to_owned(),
            ..BuildFlags::default()
        },
    ];
    for f in &bad {
        assert!(admit_build_dispatch(f).is_err(), "{f:?}");
    }
    let good = BuildFlags {
        worker_config: "/cfg".to_owned(),
        request: Request {
            development: true,
            ..Request::default()
        },
        ..BuildFlags::default()
    };
    assert!(admit_build_dispatch(&good).is_ok());
}

#[test]
fn flag_parse_matrix() {
    let f = flags(&[
        "--development",
        "--target",
        "candidate",
        "--worker-config",
        "/cfg",
        "--repository-prefix",
        "custom",
    ])
    .unwrap();
    assert!(f.request.development);
    assert_eq!(f.request.target, "candidate");
    assert_eq!(f.worker_config, "/cfg");
    assert_eq!(f.request.repository_prefix, "custom");
    assert!(flags(&["--development", "positional"]).is_err());
    assert!(flags(&["--bogus"]).is_err());
    assert!(flags(&["--arch"]).is_err());
    assert!(
        !flags(&[
            "--development",
            "--development=false",
            "--rootfs-base-url=https://public.example/release",
        ])
        .unwrap()
        .request
        .development
    );
    assert!(flags(&["--development", "false"]).is_err());
    let last = flags(&[
        "--target=candidate",
        "--development",
        "--target=media",
        "--rootfs-base-url=https://public.example/release",
    ])
    .unwrap();
    assert!(last.request.development);
    assert_eq!(last.request.target, "media");
    assert!(flags(&["-target=media"]).is_err());
    assert_eq!(flags(&["--help"]).unwrap_err(), "build help requested");
    assert!(usage("soda-build").contains("--worker-config"));
}

#[test]
fn progress_title_matches_go() {
    let dev = Request {
        development: true,
        target: "media".to_owned(),
        ..Request::default()
    };
    assert_eq!(
        progress_title(&dev),
        "Soda development media (not release-qualified)"
    );
    assert_eq!(progress_title(&Request::default()), "Soda release build");
}

#[test]
fn forgejo_root_refusals() {
    assert_eq!(
        validate_forgejo_checkout_root("relative/path").unwrap_err(),
        "explicit absolute Forgejo checkout required"
    );
    assert_eq!(
        validate_forgejo_checkout_root("/definitely/not/a/checkout").unwrap_err(),
        "canonical Forgejo checkout required"
    );
}

#[test]
fn cancel_maps_recorded_signal_identity() {
    let _guard = crate::exitcode::interrupt_test_lock();
    for code in [130, 143] {
        let _ = take_interrupt();
        note_interrupt(code);
        let err = map_worker_error(worker::WorkerError::Cancelled(
            "worker w failed: cancelled".to_owned(),
        ));
        assert!(matches!(err, ToolError::Interrupted(i) if i.exit_code() == code));
        assert_eq!(build_exit_code(Some(&err)), code);
        assert!(take_interrupt().is_none());
    }
}

#[test]
fn cancel_without_interrupt_is_cancelled() {
    let _guard = crate::exitcode::interrupt_test_lock();
    let _ = take_interrupt();
    let err = map_worker_error(worker::WorkerError::Cancelled(
        "worker w failed: cancelled".to_owned(),
    ));
    assert!(matches!(err, ToolError::Cancelled));
    assert_eq!(build_exit_code(Some(&err)), 130);
}

#[test]
fn worker_failure_keeps_message_and_exit_1() {
    let err = map_worker_error(worker::WorkerError::Failed("boom".to_owned()));
    assert!(matches!(err, ToolError::Message(ref message) if message == "boom"));
    assert_eq!(build_exit_code(Some(&err)), 1);
}
