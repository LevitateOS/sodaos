use super::{candidate_bin, run, TempDir};

#[test]
fn candidate_help_is_generated_and_safe() {
    let scratch = TempDir::new("cand-help");
    let (code, out, err) = run(&candidate_bin(), &scratch.path, &["--help"]);
    assert_eq!(code, 0);
    assert!(err.is_empty(), "{err}");
    for flag in [
        "--mode",
        "--controller",
        "--worker-config",
        "--forgejo-source",
        "--non-interactive",
    ] {
        assert!(out.contains(flag), "help omitted {flag}: {out}");
    }
    assert!(!out.contains("--sign"), "help suggests signing: {out}");
}

#[test]
fn candidate_flag_errors_refuse_before_admission() {
    let scratch = TempDir::new("cand-flags");
    let (code, out, err) = run(&candidate_bin(), &scratch.path, &["--bogus", "x"]);
    assert_eq!(code, 1);
    assert_eq!(out, "");
    assert!(err.contains("invalid candidate command flags"), "{err}");
    assert!(err.contains("Usage:") || err.contains("--help"), "{err}");
    let (code, out, err) = run(&candidate_bin(), &scratch.path, &["--mode"]);
    assert_eq!(code, 1);
    assert_eq!(out, "");
    assert!(err.contains("--mode"), "{err}");
    let (code, out, err) = run(
        &candidate_bin(),
        &scratch.path,
        &["--non-interactive=maybe"],
    );
    assert_eq!(code, 1);
    assert_eq!(out, "");
    assert!(
        err.contains("invalid candidate command flags") && !err.contains("maybe"),
        "{err}"
    );
    for (args, want) in [
        (
            vec!["--mode", "bogus"],
            "soda-candidate: --mode accepts candidate or media\n",
        ),
        (
            vec!["--arch", "aarch64"],
            "soda-candidate: matching native x86_64 required\n",
        ),
        (
            vec!["--non-interactive"],
            "soda-candidate: choose --mode candidate or media (or run on a terminal)\n",
        ),
        // Piped stdin is not a terminal: the mode prompt is skipped too.
        (
            vec![],
            "soda-candidate: choose --mode candidate or media (or run on a terminal)\n",
        ),
    ] {
        let (code, out, err) = run(&candidate_bin(), &scratch.path, &args);
        assert_eq!(code, 1, "{args:?}");
        assert_eq!(out, "", "{args:?}");
        assert_eq!(err, want, "{args:?}");
    }
}

#[test]
fn candidate_validation_preserves_admission_boundaries() {
    let scratch = TempDir::new("cand-valid");
    let base = |out: &str| -> Vec<String> {
        vec![
            "--non-interactive",
            "--controller",
            "/admitted/soda-build",
            "--worker-config",
            "/restricted/worker.json",
            "--arch",
            "x86_64",
            "--out",
            out,
            "--mode",
            "media",
            "--rootfs-base-url",
            "http://fixture:8080",
            "--forgejo-source",
            "/canonical/forgejo-ext",
            "--release-tag",
            "v0.0.0-dev",
        ]
        .into_iter()
        .map(str::to_owned)
        .collect()
    };
    let out = scratch.path.join("fresh").to_string_lossy().into_owned();
    // Fully answered media from a non-checkout fails preflight first.
    let (code, _, err) = run(&candidate_bin(), &scratch.path, &base(&out));
    assert_eq!(code, 1);
    assert_eq!(
        err,
        "soda-candidate: run soda-candidate from the checkout root (~/Projects/sodaos)\n"
    );
    // Uppercase output leaf is refused with the worker-name message.
    let upper = scratch
        .path
        .join("20260915T212541Z")
        .to_string_lossy()
        .into_owned();
    let (code, _, err) = run(&candidate_bin(), &scratch.path, &base(&upper));
    assert_eq!(code, 1);
    assert!(
        err.contains("output name must be lowercase letters, digits, or dashes (worker name rule)"),
        "{err}"
    );
    // Candidate refuses media-only inputs.
    let (code, _, err) = run(
        &candidate_bin(),
        &scratch.path,
        &[
            "--non-interactive",
            "--controller",
            "/admitted/soda-build",
            "--worker-config",
            "/restricted/worker.json",
            "--arch",
            "x86_64",
            "--out",
            &out,
            "--mode",
            "candidate",
            "--rootfs-base-url",
            "http://fixture:8080",
            "--forgejo-source",
            "/canonical/forgejo-ext",
        ],
    );
    assert_eq!(code, 1);
    assert_eq!(err, "soda-candidate: candidate refuses media-only inputs\n");
    // Media without the rootfs URL is refused.
    let (code, _, err) = run(
        &candidate_bin(),
        &scratch.path,
        &[
            "--non-interactive",
            "--controller",
            "/admitted/soda-build",
            "--worker-config",
            "/restricted/worker.json",
            "--arch",
            "x86_64",
            "--out",
            &out,
            "--mode",
            "media",
            "--forgejo-source",
            "/canonical/forgejo-ext",
        ],
    );
    assert_eq!(code, 1);
    assert_eq!(err, "soda-candidate: media requires the rootfs base URL\n");
}

#[test]
fn candidate_forgejo_source_is_required_and_checked() {
    use std::process::Command;
    let scratch = TempDir::new("cand-forgejo");
    let out = scratch.path.join("fresh").to_string_lossy().into_owned();
    let answered = |extra: &[&str]| -> Vec<String> {
        let mut args = vec![
            "--non-interactive",
            "--controller",
            "/admitted/soda-build",
            "--worker-config",
            "/restricted/worker.json",
            "--arch",
            "x86_64",
            "--out",
            &out,
            "--mode",
            "candidate",
            "--release-tag",
            "v0.0.0-dev",
        ];
        args.extend_from_slice(extra);
        args.into_iter().map(str::to_owned).collect()
    };
    // Missing and relative checkouts are refused before preflight.
    for extra in [vec![], vec!["--forgejo-source", "relative/forgejo-ext"]] {
        let (code, _, err) = run(&candidate_bin(), &scratch.path, &answered(&extra));
        assert_eq!(code, 1, "{extra:?}");
        assert_eq!(
            err, "soda-candidate: absolute Forgejo checkout path required\n",
            "{extra:?}"
        );
    }
    // A fake checkout root proves the canonical-checkout preflight:
    // go.mod plus a clean tree gets past every earlier refusal, then the
    // Forgejo root check (not sudo dispatch) stops the run.
    let root = TempDir::new("cand-root");
    std::fs::write(root.path.join("go.mod"), "module example.test\n").unwrap();
    for step in [
        vec!["init", "-q"],
        vec!["add", "go.mod"],
        vec!["commit", "-qm", "fixture"],
    ] {
        let status = Command::new("git")
            .current_dir(&root.path)
            .args(["-c", "user.name=test", "-c", "user.email=test@test"])
            .args(&step)
            .status()
            .expect("git fixture");
        assert!(status.success(), "{step:?}");
    }
    let not_checkout = TempDir::new("cand-notforgejo");
    let (code, _, err) = run(
        &candidate_bin(),
        &root.path,
        &answered(&["--forgejo-source", &not_checkout.path.to_string_lossy()]),
    );
    assert_eq!(code, 1);
    assert_eq!(err, "soda-candidate: canonical Forgejo checkout required\n");
}
