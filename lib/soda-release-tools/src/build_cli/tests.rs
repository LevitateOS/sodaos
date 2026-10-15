use super::*;

use crate::build_spec::Request;

fn flags(args: &[&str]) -> Result<BuildFlags, String> {
    parse_build_flags(&args.iter().map(|s| s.to_string()).collect::<Vec<_>>())
}

#[test]
fn docker_build_flags_admit_the_packaging_boundary() {
    let f = flags(&["--distribution", "docker", "--release-tag", "v1.2.3"]).unwrap();
    assert!(f.request.wants_docker());
    assert!(!f.request.wants_media());
    assert_eq!(
        progress_title(&f.request),
        "Soda release docker build (not release-qualified)"
    );
    assert!(flags(&["--distribution", "docker"]).is_err());
    assert!(flags(&[
        "--development",
        "--target",
        "docker",
        "--distribution",
        "docker",
        "--release-tag",
        "v1.2.3"
    ])
    .is_ok());
}

#[test]
fn prospective_upgrade_source_is_explicit_and_not_qualification() {
    let source = "43.20260101.3.0.soda-cccccccccccc";
    let parsed = flags(&[
        "--development",
        "--target",
        "candidate",
        "--upgrade-from",
        source,
    ])
    .unwrap();
    assert_eq!(parsed.request.upgrade_from, source);
    assert!(flags(&[
        "--development",
        "--target",
        "candidate",
        "--upgrade-from",
        "latest"
    ])
    .is_err());
    assert!(flags(&["--development", "--target", "candidate"])
        .unwrap()
        .request
        .upgrade_from
        .is_empty());
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
    let release_media = BuildFlags {
        worker_config: "/cfg".to_owned(),
        ..BuildFlags::default()
    };
    assert!(admit_parent_dispatch(&release_media).is_ok());
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
    assert_eq!(
        progress_title(&Request::default()),
        "Soda release media build (not release-qualified)"
    );
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

#[test]
fn preflight_cannot_be_used_as_a_worker_completion_boundary() {
    let flags = BuildFlags {
        preflight: true,
        worker_build: true,
        ..Default::default()
    };
    assert!(admit_build_dispatch(&flags)
        .unwrap_err()
        .contains("operator operation"));
}

#[test]
fn controller_source_admission_requires_matching_clean_checkout_before_dispatch() {
    let source = std::env::temp_dir().join(format!(
        "soda-controller-source-{}-{}",
        std::process::id(),
        std::time::SystemTime::now()
            .duration_since(std::time::UNIX_EPOCH)
            .unwrap()
            .as_nanos()
    ));
    std::fs::create_dir(&source).unwrap();
    let git = |args: &[&str]| {
        let output = Command::new("git")
            .arg("-C")
            .arg(&source)
            .args([
                "-c",
                "user.name=soda-tester",
                "-c",
                "user.email=soda-tester@example.invalid",
                "-c",
                "commit.gpgsign=false",
                "-c",
                "core.hooksPath=/dev/null",
            ])
            .args(args)
            .output()
            .unwrap();
        assert!(
            output.status.success(),
            "{}",
            String::from_utf8_lossy(&output.stderr)
        );
        String::from_utf8(output.stdout).unwrap().trim().to_owned()
    };
    git(&["init", "--quiet"]);
    std::fs::write(source.join("input"), b"frozen source\n").unwrap();
    git(&["add", "input"]);
    git(&["commit", "--quiet", "-m", "fixture"]);
    let revision = git(&["rev-parse", "HEAD"]);
    let root = source.to_str().unwrap();
    check_source_revision(root, &revision).unwrap();
    assert_eq!(
        check_source_revision(root, &"0".repeat(40)).unwrap_err(),
        "controller/source revision mismatch"
    );
    std::fs::create_dir(source.join("nested")).unwrap();
    assert_eq!(
        check_source_revision(source.join("nested").to_str().unwrap(), &revision).unwrap_err(),
        "canonical checkout root required"
    );
    std::fs::write(source.join("input"), b"uncommitted source\n").unwrap();
    assert_eq!(
        check_source_revision(root, &revision).unwrap_err(),
        "clean committed source required"
    );
    std::fs::remove_dir_all(source).unwrap();
}

#[test]
fn build_completion_requires_final_controller_success() {
    let _guard = crate::exitcode::interrupt_test_lock();
    let _ = take_interrupt();
    let out = tempfile::tempdir().unwrap();
    let evidence = out.path().join("evidence");
    std::fs::create_dir(&evidence).unwrap();
    std::fs::write(evidence.join("build.json"), b"producer receipt\n").unwrap();
    let mut flags = BuildFlags {
        worker_build: true,
        request: Request {
            out: out.path().to_string_lossy().into_owned(),
            ..Default::default()
        },
        ..Default::default()
    };
    let mut worker_progress = BuildProgress::new("worker").unwrap();
    worker_progress.capture();
    finish_build(&flags, &mut worker_progress, None).unwrap();
    assert!(!evidence.join("completed.json").exists());

    flags.worker_build = false;
    let mut failed_progress = BuildProgress::new("controller").unwrap();
    failed_progress.capture();
    let log = evidence.join("progress.log");
    failed_progress.create_log(log.to_str().unwrap()).unwrap();
    std::fs::remove_file(log).unwrap();
    assert!(finish_build(&flags, &mut failed_progress, None).is_err());
    assert!(!evidence.join("completed.json").exists());

    let interrupted = std::sync::Arc::new(std::sync::atomic::AtomicBool::new(false));
    let at_finish = interrupted.clone();
    let mut late_progress = BuildProgress::new_with_clock(
        "controller",
        Box::new(move || {
            if at_finish.load(std::sync::atomic::Ordering::SeqCst) {
                note_interrupt(143);
            }
            crate::progress::monotonic()
        }),
    )
    .unwrap();
    late_progress.capture();
    interrupted.store(true, std::sync::atomic::Ordering::SeqCst);
    assert!(matches!(
        finish_build(&flags, &mut late_progress, None),
        Err(ToolError::Interrupted(_))
    ));
    assert!(!evidence.join("completed.json").exists());

    let mut progress = BuildProgress::new("controller").unwrap();
    progress.capture();
    finish_build(&flags, &mut progress, None).unwrap();
    let completion: soda_build_tools::reader::build::BuildCompletion =
        serde_json::from_slice(&std::fs::read(evidence.join("completed.json")).unwrap()).unwrap();
    assert_eq!(
        completion.build_sha256,
        crate::digest::hash_file(evidence.join("build.json").to_str().unwrap()).unwrap()
    );
    assert!(record_build_completion(&flags.request.out).is_err());
}

#[test]
fn fountain_sdk_is_bound_to_release_without_requiring_unrelated_cleanliness() {
    let root = std::env::temp_dir().join(format!(
        "fountain-sdk-{}-{}",
        std::process::id(),
        std::time::SystemTime::now()
            .duration_since(std::time::UNIX_EPOCH)
            .unwrap()
            .as_nanos()
    ));
    std::fs::create_dir(&root).unwrap();
    struct Fixture(std::path::PathBuf);
    impl Drop for Fixture {
        fn drop(&mut self) {
            let _ = std::fs::remove_dir_all(&self.0);
        }
    }
    let _fixture = Fixture(root.clone());
    std::fs::create_dir(root.join("sdk")).unwrap();
    std::fs::write(
        root.join("sdk/go.mod"),
        "module forgejo.org/extension-sdk\n",
    )
    .unwrap();
    std::fs::write(root.join("outside.go"), "initial").unwrap();
    let git = |args: &[&str]| {
        let output = std::process::Command::new("git")
            .arg("-C")
            .arg(&root)
            .args(args)
            .output()
            .unwrap();
        assert!(output.status.success(), "fixture git failed");
        String::from_utf8(output.stdout).unwrap().trim().to_string()
    };
    git(&["init", "--quiet"]);
    git(&["add", "."]);
    git(&[
        "-c",
        "user.name=sdk-fixture",
        "-c",
        "user.email=sdk-fixture@example.invalid",
        "commit",
        "--quiet",
        "-m",
        "SDK fixture",
    ]);
    let revision = git(&["rev-parse", "HEAD"]);
    let source = root.to_str().unwrap();
    verify_fountain_sdk(source, &revision).unwrap();
    std::fs::write(root.join("outside.go"), "concurrent unrelated change").unwrap();
    verify_fountain_sdk(source, &revision).unwrap();
    std::fs::write(root.join("sdk/go.mod"), "module other\n").unwrap();
    assert!(verify_fountain_sdk(source, &revision).is_err());
    std::fs::write(
        root.join("sdk/go.mod"),
        "module forgejo.org/extension-sdk\n",
    )
    .unwrap();
    std::fs::write(root.join("sdk/extra.go"), "package extension\n").unwrap();
    assert!(verify_fountain_sdk(source, &revision).is_err());
}
