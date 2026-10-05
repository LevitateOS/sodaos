//! CLI-surface parity tests: every `--help`, usage, admission, and
//! refusal path must match the Go owners byte-for-byte (exit code, stdout,
//! stderr). Expected strings were captured from the Go binaries built at
//! the base revision. Success paths that need root, the worker identity,
//! network, or external tools stop at explicit boundary errors instead.

use std::env;
use std::fs;
use std::os::unix::fs::PermissionsExt;
use std::path::{Path, PathBuf};
use std::process::Command;
use std::sync::atomic::{AtomicU64, Ordering};

static COUNTER: AtomicU64 = AtomicU64::new(0);

struct TempDir {
    path: PathBuf,
}

impl TempDir {
    fn new(tag: &str) -> TempDir {
        let id = COUNTER.fetch_add(1, Ordering::SeqCst);
        let path = env::temp_dir().join(format!("soda-reltools-{tag}-{}-{id}", std::process::id()));
        let _ = fs::remove_dir_all(&path);
        fs::create_dir_all(&path).unwrap();
        TempDir { path }
    }
}

impl Drop for TempDir {
    fn drop(&mut self) {
        let _ = fs::remove_dir_all(&self.path);
    }
}

fn build_bin() -> PathBuf {
    PathBuf::from(env!("CARGO_BIN_EXE_soda-build"))
}

fn candidate_bin() -> PathBuf {
    PathBuf::from(env!("CARGO_BIN_EXE_soda-candidate"))
}

fn artifacts_bin() -> PathBuf {
    PathBuf::from(env!("CARGO_BIN_EXE_soda-artifacts"))
}

fn run<S: AsRef<std::ffi::OsStr>>(bin: &Path, cwd: &Path, args: &[S]) -> (i32, String, String) {
    let output = Command::new(bin)
        .current_dir(cwd)
        .args(args)
        .output()
        .expect("spawn binary");
    (
        output.status.code().unwrap_or(-1),
        String::from_utf8_lossy(&output.stdout).into_owned(),
        String::from_utf8_lossy(&output.stderr).into_owned(),
    )
}

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

#[test]
fn artifacts_dispatch_matches_go() {
    let scratch = TempDir::new("art-dispatch");
    for (args, want) in [
        (vec![], "usage: soda-artifacts inspect-oci|fetch-coreos|fetch-coreos-iso|convert-butane [flags]\n"),
        (
            vec!["-h"],
            "unknown artifact action; use fetch-coreos-iso for upstream ISO inputs; QCOW2 media delivery is not selected\n",
        ),
        (
            vec!["bogus-action"],
            "unknown artifact action; use fetch-coreos-iso for upstream ISO inputs; QCOW2 media delivery is not selected\n",
        ),
        (vec!["inspect-oci"], "expected x86_64\n"),
        (
            vec!["inspect-oci", "--arch", "x86_64", "--source", "/nope", "--revision", "abc"],
            "full source revision required\n",
        ),
        (
            vec!["inspect-oci", "extra", "--arch", "x86_64"],
            "invalid artifact command flags\n",
        ),
        (
            vec!["inspect-oci", "--arch", "x86_64", "--bogus", "x"],
            "invalid artifact command flags\n",
        ),
        (
            vec!["convert-butane", "--arch", "x86_64", "--source", "/nope", "--out", "/tmp/x.json"],
            "real private output parent required\n",
        ),
        (
            vec!["fetch-coreos", "--arch", "x86_64", "--keyring", "/k", "--signer", "deadbeef", "--out", "/tmp/x"],
            "full trusted signer fingerprint required\n",
        ),
        (
            vec!["fetch-coreos-iso", "--arch", "aarch64"],
            "expected x86_64\n",
        ),
    ] {
        let (code, out, err) = run(&artifacts_bin(), &scratch.path, &args);
        assert_eq!(code, 1, "{args:?}");
        assert_eq!(out, "", "{args:?}");
        assert_eq!(err, want, "{args:?}");
    }
}

#[test]
fn artifacts_butane_refusals_match_go() {
    let scratch = TempDir::new("art-butane");
    // Private parent, missing source: the Go owner reports the missing file.
    let private = scratch.path.join("p");
    fs::create_dir(&private).unwrap();
    fs::set_permissions(&private, fs::Permissions::from_mode(0o700)).unwrap();
    let missing = private.join("missing.bu");
    let dest = private.join("out.json");
    let (code, _, err) = run(
        &artifacts_bin(),
        &scratch.path,
        &[
            "convert-butane",
            "--arch",
            "x86_64",
            "--source",
            missing.to_str().unwrap(),
            "--out",
            dest.to_str().unwrap(),
        ],
    );
    assert_eq!(code, 1);
    // The tool check precedes the source open in both implementations.
    let butane_present = env::var_os("PATH")
        .map(|paths| env::split_paths(&paths).any(|dir| dir.join("butane").is_file()))
        .unwrap_or(false);
    if butane_present {
        assert!(err.contains("No such file"), "{err}");
    } else {
        assert_eq!(
            err,
            "exec: \"butane\": executable file not found in $PATH\n"
        );
    }
    assert!(!dest.exists());
}

#[test]
fn artifacts_inspect_validates_archive_shape() {
    let scratch = TempDir::new("art-oci");
    let dir = scratch.path.join("adir");
    fs::create_dir(&dir).unwrap();
    let revision = "a".repeat(40);
    let (code, _, err) = run(
        &artifacts_bin(),
        &scratch.path,
        &[
            "inspect-oci",
            "--arch",
            "x86_64",
            "--source",
            dir.to_str().unwrap(),
            "--revision",
            &revision,
        ],
    );
    assert_eq!(code, 1);
    assert_eq!(err, "OCI archive must be regular\n");
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
