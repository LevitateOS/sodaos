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
        ],
    );
    assert_eq!(code, 1);
    assert_eq!(err, "soda-candidate: media requires the rootfs base URL\n");
}
