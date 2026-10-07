use super::{build_bin, run, TempDir};

#[test]
fn build_help_is_generated_and_safe() {
    let scratch = TempDir::new("build-help");
    let (code, out, err) = run(&build_bin(), &scratch.path, &["--help"]);
    assert_eq!(code, 0);
    assert!(err.is_empty(), "{err}");
    for flag in [
        "--development",
        "--target",
        "--worker-build",
        "--worker-config",
    ] {
        assert!(out.contains(flag), "help omitted {flag}: {out}");
    }
    assert!(!out.contains("--sign"), "help suggests signing: {out}");
}

#[test]
fn build_flag_errors_refuse_without_running() {
    let scratch = TempDir::new("build-flags");
    let (code, out, err) = run(&build_bin(), &scratch.path, &["--bogus", "x"]);
    assert_eq!(code, 2);
    assert_eq!(out, "");
    assert!(err.contains("invalid build command flags"), "{err}");
    assert!(err.contains("Usage:") || err.contains("--help"), "{err}");
    let (code, _, err) = run(&build_bin(), &scratch.path, &["--arch"]);
    assert_eq!(code, 2);
    assert!(err.contains("--arch"), "{err}");
    let (code, _, err) = run(&build_bin(), &scratch.path, &["--development=maybe"]);
    assert_eq!(code, 2);
    assert!(
        err.contains("invalid build command flags") && !err.contains("maybe"),
        "{err}"
    );
    let (code, out, err) = run(&build_bin(), &scratch.path, &["positional"]);
    assert_eq!(code, 2);
    assert!(out.is_empty());
    assert!(err.contains("invalid build command flags"));
}

#[test]
fn build_admission_matrix_preserves_release_boundaries() {
    let scratch = TempDir::new("build-admit");
    for (args, want) in [
        (
            vec!["--development"],
            "--development requires --target candidate or media\n",
        ),
        (
            vec!["--development", "--target", "bogus"],
            "--development requires --target candidate or media\n",
        ),
        (
            vec!["--target", "candidate"],
            "--target requires --development\n",
        ),
        (
            vec!["--media-compression", "fast"],
            "--media-compression accepts only fast with --development --target media\n",
        ),
        (
            vec!["--development", "--target", "candidate"],
            "root-owned --worker-config required; media authority belongs to the isolated worker\n",
        ),
        (
            vec!["--development", "--target", "media"],
            "explicit public HTTP(S) rootfs base URL required\n",
        ),
        (
            vec![
                "--development",
                "--target",
                "media",
                "--rootfs-base-url",
                "http://127.0.0.1:8080/",
            ],
            "rootfs base URL must be reachable from the installing machine, not loopback\n",
        ),
        (
            vec!["--worker-build"],
            "explicit public HTTP(S) rootfs base URL required\n",
        ),
    ] {
        let (code, out, err) = run(&build_bin(), &scratch.path, &args);
        assert_eq!(code, 1, "{args:?}");
        assert_eq!(out, "", "{args:?}");
        assert_eq!(err, want, "{args:?}");
    }
}

#[test]
fn build_bool_and_scalar_forms_preserve_contract() {
    let scratch = TempDir::new("build-bool");
    // Explicit false keeps production dispatch, which the parent refuses.
    let (code, _, err) = run(
        &build_bin(),
        &scratch.path,
        &[
            "--development=false",
            "--target=candidate",
            "--target=media",
            "--worker-config",
            "/cfg",
        ],
    );
    assert_eq!(code, 1);
    assert_eq!(err, "--target requires --development\n");
    // A scalar repeated with documented long syntax resolves to its last value.
    let (code, _, err) = run(
        &build_bin(),
        &scratch.path,
        &[
            "--development",
            "--target=candidate",
            "--target=media",
            "--worker-config",
            "/cfg",
        ],
    );
    assert_eq!(code, 1);
    assert_eq!(err, "explicit public HTTP(S) rootfs base URL required\n");
    // A bare bool never consumes the next argument: `false` is positional.
    let (code, _, err) = run(&build_bin(), &scratch.path, &["--development", "false"]);
    assert_eq!(code, 2);
    assert!(err.contains("invalid build command flags"));
}

#[test]
fn build_deep_refusal_matches_go() {
    // A fully admitted parent dispatch from a dirty checkout fails at the
    // VCS stamp before any privileged step, like the Go owner.
    let scratch = TempDir::new("build-deep");
    let (code, out, err) = run(
        &build_bin(),
        &scratch.path,
        &[
            "--development",
            "--target",
            "candidate",
            "--arch",
            "x86_64",
            "--out",
            "/tmp/pr-reltools-out",
            "--repository-prefix",
            "ghcr.io/levitateos/sodaos",
            "--worker-config",
            "/cfg",
            "--forgejo-source",
            "/forgejo",
        ],
    );
    assert_eq!(code, 1);
    assert_eq!(out, "");
    // Like Go, the FAILED progress line precedes the final message.
    assert!(
        err.starts_with("FAILED   Soda development candidate (not release-qualified) | total "),
        "{err}"
    );
    // The crate binary may be stamped clean or dirty depending on the
    // checkout state at build time; every outcome matches Go exactly.
    assert!(
        err.ends_with("controller must be compiled from committed source\n")
            || err.ends_with("canonical Forgejo checkout required\n"),
        "{err}"
    );
}
