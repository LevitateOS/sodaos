use super::{candidate_bin, run, TempDir};

fn go_candidate_usage() -> String {
    "Usage of soda-candidate:\n  -arch string\n    \tmatching native x86_64 (default \"x86_64\")\n  -controller string\n    \tadmitted soda-build executable (asked when empty)\n  -media-compression string\n    \tfast: development media only\n  -mode string\n    \tcandidate or media (asked when empty)\n  -non-interactive\n    \trequire all flags; timestamped log output\n  -out string\n    \tfresh output below .artifacts/releases (asked when empty)\n  -repository-prefix string\n    \tintended image repositories; no publication (default \"ghcr.io/levitateos/sodaos\")\n  -rootfs-base-url string\n    \tpublic base URL for the hash-named rootfs file\n  -rootfs-dir string\n    \tpickup folder served for loopback development media (default .artifacts/rootfs)\n  -worker-config string\n    \trestricted worker configuration (asked when empty)\n"
        .to_owned()
}

#[test]
fn candidate_help_matches_go() {
    let scratch = TempDir::new("cand-help");
    let (code, out, err) = run(&candidate_bin(), &scratch.path, &["-h"]);
    assert_eq!(code, 1);
    assert_eq!(out, "");
    assert_eq!(
        err,
        format!(
            "{}soda-candidate: flag: help requested\n",
            go_candidate_usage()
        )
    );
}

#[test]
fn candidate_flag_errors_match_go() {
    let scratch = TempDir::new("cand-flags");
    let (code, out, err) = run(&candidate_bin(), &scratch.path, &["--bogus", "x"]);
    assert_eq!(code, 1);
    assert_eq!(out, "");
    assert_eq!(
        err,
        format!(
            "flag provided but not defined: -bogus\n{}soda-candidate: flag provided but not defined: -bogus\n",
            go_candidate_usage()
        )
    );
    for (args, want) in [
        (
            vec!["positional"],
            "soda-candidate: unexpected positional arguments\n",
        ),
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
fn candidate_validation_matches_go() {
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
