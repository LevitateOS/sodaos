use std::env;
use std::fs;
use std::os::unix::fs::PermissionsExt;

use super::{artifacts_bin, run, TempDir};

#[test]
fn artifacts_dispatch_matches_go() {
    let scratch = TempDir::new("art-dispatch");
    for (args, want) in [
        (vec![], "usage: soda-artifacts inspect-oci|fetch-coreos|fetch-coreos-iso|convert-butane [flags]\n"),
        (
            vec!["bogus-action"],
            "invalid artifact command flags\n",
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
fn artifacts_generated_help_exits_before_work() {
    let scratch = TempDir::new("art-help");
    for args in [vec!["-h"], vec!["--help"], vec!["inspect-oci", "--help"]] {
        let (code, out, err) = run(&artifacts_bin(), &scratch.path, &args);
        assert_eq!(code, 0, "{args:?}: {err}");
        assert!(out.contains("--keyring"), "{out}");
        assert!(err.is_empty(), "{err}");
    }
    assert!(fs::read_dir(&scratch.path).unwrap().next().is_none());
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
