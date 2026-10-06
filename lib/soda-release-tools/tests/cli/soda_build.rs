use super::{build_bin, run, TempDir};

fn go_build_usage(argv0: &str) -> String {
    format!(
        "Usage of {argv0}:\n  -arch string\n    \tmatching native x86_64\n  -development\n    \texplicit development-only run; never release-qualified\n  -forgejo-revision string\n    \tinternal exact Forgejo source revision\n  -forgejo-source string\n    \texplicit clean canonical Forgejo fork checkout\n  -live-inputs string\n    \tinternal controller-resolved live inputs file\n  -media-authority string\n    \tworker-local fixture authority; not release custody\n  -media-compression string\n    \tfast: development media only; changes host compression metadata (default: upstream)\n  -out string\n    \tfresh absolute output below .artifacts/releases (parent must exist)\n  -repository-prefix string\n    \tintended immutable image repositories; no publication (default \"ghcr.io/levitateos/sodaos\")\n  -rootfs-base-url string\n    \tpublic base URL for the exact hash-named rootfs file\n  -target string\n    \tdevelopment boundary: candidate or media (requires --development)\n  -worker-build\n    \tinternal build stage; requires the isolated build identity\n  -worker-config string\n    \troot-owned configuration for isolated worker dispatch\n"
    )
}

#[test]
fn build_help_matches_go() {
    let scratch = TempDir::new("build-help");
    for flag in ["-h", "--help", "-help"] {
        let (code, out, err) = run(&build_bin(), &scratch.path, &[flag]);
        assert_eq!(code, 0, "{flag}");
        assert_eq!(out, "", "{flag}");
        assert_eq!(
            err,
            go_build_usage(&build_bin().to_string_lossy()),
            "{flag}"
        );
    }
}

#[test]
fn build_flag_errors_match_go() {
    let scratch = TempDir::new("build-flags");
    let usage = go_build_usage(&build_bin().to_string_lossy());
    let (code, out, err) = run(&build_bin(), &scratch.path, &["--bogus", "x"]);
    assert_eq!(code, 2);
    assert_eq!(out, "");
    assert_eq!(
        err,
        format!("flag provided but not defined: -bogus\n{usage}")
    );
    let (code, _, err) = run(&build_bin(), &scratch.path, &["--arch"]);
    assert_eq!(code, 2);
    assert!(err.starts_with("flag needs an argument: -arch\n"), "{err}");
    let (code, _, err) = run(&build_bin(), &scratch.path, &["--development=maybe"]);
    assert_eq!(code, 2);
    assert!(
        err.starts_with("invalid boolean value \"maybe\" for -development: strconv.ParseBool: parsing \"maybe\": invalid syntax\n"),
        "{err}"
    );
}

#[test]
fn build_admission_matrix_matches_go() {
    let scratch = TempDir::new("build-admit");
    for (args, want) in [
        (vec!["positional"], "unexpected positional arguments\n"),
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
fn build_bool_forms_match_go() {
    let scratch = TempDir::new("build-bool");
    // `-flag=false` keeps production dispatch, which the parent refuses.
    let (code, _, err) = run(
        &build_bin(),
        &scratch.path,
        &[
            "-development=false",
            "-target=candidate",
            "--worker-config",
            "/cfg",
        ],
    );
    assert_eq!(code, 1);
    assert_eq!(err, "--target requires --development\n");
    // A bare bool never consumes the next argument: `false` is positional.
    let (code, _, err) = run(&build_bin(), &scratch.path, &["--development", "false"]);
    assert_eq!(code, 1);
    assert_eq!(err, "unexpected positional arguments\n");
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
