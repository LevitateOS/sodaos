use super::super::*;
use super::fixtures::*;

#[test]
fn content_imports_exact_local_references() {
    for already_present in [false, true] {
        let root = test_root(&format!("content-{already_present}"));
        let (payload, layout) = full_fixture(&root);
        let dir = layout.to_str().unwrap().to_string();
        assert!(verify_content(&payload, &dir).is_ok());
        let cancelled = AtomicBool::new(false);
        let ctx = test_ctx(&cancelled);
        let mut present = std::collections::HashSet::new();
        let mut queries = 0;
        let mut pulls = 0;
        let mut pull_names: Vec<&str> = Vec::new();
        let mut run = |cmd: &str, args: &[String]| -> PodmanOutcome {
            assert_eq!(cmd, "/usr/bin/podman");
            assert_eq!(args[0], "--remote=false");
            match args[1].as_str() {
                "image" => {
                    assert_eq!(args[2], "exists");
                    assert_eq!(args.len(), 4);
                    queries += 1;
                    if already_present || present.contains(&args[3]) {
                        PodmanOutcome::Code(0)
                    } else {
                        PodmanOutcome::Code(1)
                    }
                }
                "pull" => {
                    let expected = &payload.images[NAMES[pulls]];
                    assert_eq!(
                        args.to_vec(),
                        vec![
                            "--remote=false".to_string(),
                            "pull".to_string(),
                            "--retry=0".to_string(),
                            format!("oci:{dir}:{}", expected.config),
                        ]
                    );
                    present.insert(expected.config.clone());
                    pull_names.push(NAMES[pulls]);
                    pulls += 1;
                    PodmanOutcome::Code(0)
                }
                other => panic!("unexpected native operation: {other}"),
            }
        };
        import_images(&payload, &dir, "/usr/bin/podman", &ctx, &mut run).expect("import");
        if already_present {
            assert_eq!(queries, NAMES.len());
            assert_eq!(pulls, 0);
        } else {
            assert_eq!(queries, 2 * NAMES.len());
            assert_eq!(pulls, NAMES.len());
            assert_eq!(pull_names, NAMES.to_vec());
        }
    }
}

#[test]
fn content_refuses_whole_layout_before_any_import() {
    for kind in [
        "bad-last-config",
        "wrong-manifest",
        "relative-path",
        "transport-separator",
        "wrong-format",
    ] {
        let root = test_root(&format!("refuse-{kind}"));
        let (mut payload, layout) = full_fixture(&root);
        let mut dir = layout.to_str().unwrap().to_string();
        match kind {
            "bad-last-config" => {
                let last = NAMES[NAMES.len() - 1];
                let hex = payload.images[last]
                    .config
                    .trim_start_matches("sha256:")
                    .to_string();
                fs::write(layout.join("blobs").join("sha256").join(hex), b"bad").expect("clobber");
            }
            "wrong-manifest" => {
                let manifest = format!("sha256:{}", repeat('9', 64));
                let binding = payload.images.get_mut("dashboard").unwrap();
                binding.manifest = manifest.clone();
                binding.reference = format!("{}-dashboard@{manifest}", payload.repository_prefix);
            }
            "relative-path" => dir = "relative/layout".to_string(),
            "transport-separator" => dir += "ignored:selector",
            "wrong-format" => payload.format = 2,
            _ => unreachable!(),
        }
        let cancelled = AtomicBool::new(false);
        let ctx = test_ctx(&cancelled);
        let mut calls = 0;
        let mut run = |_: &str, _: &[String]| -> PodmanOutcome {
            calls += 1;
            PodmanOutcome::Code(0)
        };
        assert!(
            import_images(&payload, &dir, "/usr/bin/podman", &ctx, &mut run).is_err(),
            "{kind} accepted"
        );
        assert_eq!(calls, 0, "{kind} ran podman");
    }
}

#[test]
fn native_failures_remain_unconfirmed_without_replay() {
    for kind in ["observation", "pull", "postcheck", "cancelled", "expired"] {
        let root = test_root(&format!("fail-{kind}"));
        let (payload, layout) = full_fixture(&root);
        let dir = layout.to_str().unwrap().to_string();
        let cancelled = AtomicBool::new(kind == "cancelled");
        let mut ctx = test_ctx(&cancelled);
        if kind == "expired" {
            ctx.deadline = Instant::now() - Duration::from_secs(1);
        }
        let mut queries = 0;
        let mut pulls = 0;
        let mut run = |_: &str, args: &[String]| -> PodmanOutcome {
            if args[1] == "image" {
                queries += 1;
                if kind == "observation" {
                    return PodmanOutcome::Code(125);
                }
                if kind == "postcheck" && queries == 2 {
                    return PodmanOutcome::Code(1);
                }
                return PodmanOutcome::Code(if queries == 1 { 1 } else { 0 });
            }
            assert_eq!(args[1], "pull");
            pulls += 1;
            if kind == "pull" {
                return PodmanOutcome::Code(3);
            }
            PodmanOutcome::Code(0)
        };
        let err = import_images(&payload, &dir, "/usr/bin/podman", &ctx, &mut run).unwrap_err();
        match kind {
            "observation" => {
                assert!(err.contains("observation failed"), "{err}");
                assert_eq!(queries, 1);
                assert_eq!(pulls, 0);
            }
            "pull" => {
                assert!(err.contains("import unconfirmed"), "{err}");
                assert_eq!(queries, 1);
                assert_eq!(pulls, 1);
            }
            "postcheck" => {
                assert!(err.contains("unavailable"), "{err}");
                assert_eq!(queries, 2);
                assert_eq!(pulls, 1);
            }
            "cancelled" => {
                assert_eq!(err, "context canceled");
                assert_eq!(queries, 0);
                assert_eq!(pulls, 0);
            }
            "expired" => {
                assert_eq!(err, "context deadline exceeded");
                assert_eq!(queries, 0);
                assert_eq!(pulls, 0);
            }
            _ => unreachable!(),
        }
    }
}

fn fake_podman(root: &Path, name: &str, body: &str) -> String {
    let path = root.join(name);
    fs::write(&path, format!("#!/bin/sh\n{body}\n")).expect("fake podman");
    use std::os::unix::fs::PermissionsExt;
    fs::set_permissions(&path, fs::Permissions::from_mode(0o755)).expect("chmod");
    path.to_str().unwrap().to_string()
}

#[test]
fn runner_maps_exits_and_kills_on_deadline() {
    let root = test_root("runner");
    let cancelled = AtomicBool::new(false);
    let ctx = test_ctx(&cancelled);
    let args = vec![
        "--remote=false".to_string(),
        "image".to_string(),
        "exists".to_string(),
        "x".to_string(),
    ];
    let ok = fake_podman(&root, "podman-ok", "exit 0");
    assert_eq!(run_podman(&ok, &args, &ctx), PodmanOutcome::Code(0));
    let missing = fake_podman(&root, "podman-missing", "exit 1");
    assert_eq!(run_podman(&missing, &args, &ctx), PodmanOutcome::Code(1));
    let bad = fake_podman(&root, "podman-bad", "exit 125");
    assert_eq!(run_podman(&bad, &args, &ctx), PodmanOutcome::Code(125));
    assert_eq!(
        run_podman("/nonexistent/podman-binary", &args, &ctx),
        PodmanOutcome::Failed
    );
    // A hung engine is killed once the deadline passes.
    let hung = fake_podman(&root, "podman-hung", "sleep 30");
    let tight = ImportCtx {
        deadline: Instant::now() + Duration::from_millis(200),
        cancelled: &cancelled,
    };
    let start = Instant::now();
    assert_eq!(run_podman(&hung, &args, &tight), PodmanOutcome::Failed);
    assert!(start.elapsed() < Duration::from_secs(10));
    // Cancellation mid-run kills the child too.
    let cancelling = AtomicBool::new(true);
    let cancel_ctx = ImportCtx {
        deadline: Instant::now() + Duration::from_secs(60),
        cancelled: &cancelling,
    };
    assert_eq!(run_podman(&hung, &args, &cancel_ctx), PodmanOutcome::Failed);
}
