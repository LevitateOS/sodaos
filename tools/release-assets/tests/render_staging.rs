#![cfg(unix)]

// Shared across suites; each suite uses a subset.
#[allow(dead_code)]
#[path = "render_support/mod.rs"]
mod render_support;

use std::fs;

use render_support::{run, stage_bin, TempDir};

// ---- soda-stage ----

#[test]
fn stage_help_and_usage() {
    let scratch = TempDir::new("stage-cli");
    let (code, out, _) = run(&stage_bin(), &scratch.path, &["--help"]);
    assert_eq!(code, 0);
    assert!(
        out.starts_with(
            "usage: soda-stage --arch x86_64 --host-context DIR --forgejo-context DIR\n"
        ),
        "{out}"
    );
    for args in [
        vec![],
        vec!["--arch", "x86_64"],
        vec!["--arch"],
        vec![
            "--arch",
            "x86_64",
            "--host-context",
            "h",
            "--forgejo-context",
            "f",
            "stray",
        ],
        vec![
            "--arch",
            "x86_64",
            "--host-context",
            "h",
            "--forgejo-context",
            "f",
            "--bogus",
            "x",
        ],
        vec!["-arch", "x86_64"],
    ] {
        let (code, _, err) = run(&stage_bin(), &scratch.path, &args);
        assert_eq!(code, 2, "{args:?}");
        assert!(err.starts_with("usage: soda-stage "), "{args:?}: {err}");
        assert!(err.contains("soda-stage: error: "), "{args:?}: {err}");
    }
    let (code, _, err) = run(
        &stage_bin(),
        &scratch.path,
        &[
            "--arch",
            "aarch64",
            "--host-context",
            "h",
            "--forgejo-context",
            "f",
        ],
    );
    assert_eq!(code, 2);
    assert!(
        err.ends_with("argument --arch: invalid choice: 'aarch64' (choose from x86_64)\n"),
        "{err}"
    );
}

#[test]
fn stage_refuses_bad_contexts_before_touching_the_tree() {
    let scratch = TempDir::new("stage-ctx");
    let host = scratch.sub("host", 0o700);
    let good_fc = scratch.sub("fc", 0o700);
    // Missing rootfs under the host context.
    let (code, _, err) = run(
        &stage_bin(),
        &scratch.path,
        &[
            "--arch",
            "x86_64",
            "--host-context",
            host.to_str().unwrap(),
            "--forgejo-context",
            good_fc.to_str().unwrap(),
        ],
    );
    assert_eq!(code, 2);
    assert!(err.ends_with("soda-stage: error: real prepared host and fresh Forgejo context directories required\n"), "{err}");
    // Relative contexts.
    let (code, _, err) = run(
        &stage_bin(),
        &scratch.path,
        &[
            "--arch",
            "x86_64",
            "--host-context",
            "relative",
            "--forgejo-context",
            good_fc.to_str().unwrap(),
        ],
    );
    assert_eq!(code, 2);
    assert!(
        err.contains("real prepared host and fresh Forgejo context directories required"),
        "{err}"
    );
    // Symlinked Forgejo context resolves away from its spelling.
    let target = scratch.sub("target", 0o700);
    let link = scratch.path.join("link");
    std::os::unix::fs::symlink(&target, &link).unwrap();
    let rootfs = host.join("rootfs");
    fs::create_dir(&rootfs).unwrap();
    let (code, _, err) = run(
        &stage_bin(),
        &scratch.path,
        &[
            "--arch",
            "x86_64",
            "--host-context",
            host.to_str().unwrap(),
            "--forgejo-context",
            link.to_str().unwrap(),
        ],
    );
    assert_eq!(code, 2);
    assert!(
        err.contains("real prepared host and fresh Forgejo context directories required"),
        "{err}"
    );
    // Occupied Forgejo presentation.
    fs::create_dir(good_fc.join("forgejo")).unwrap();
    let (code, _, err) = run(
        &stage_bin(),
        &scratch.path,
        &[
            "--arch",
            "x86_64",
            "--host-context",
            host.to_str().unwrap(),
            "--forgejo-context",
            good_fc.to_str().unwrap(),
        ],
    );
    assert_eq!(code, 2);
    assert!(
        err.ends_with("soda-stage: error: occupied Forgejo presentation refused\n"),
        "{err}"
    );
}

#[test]
fn stage_reports_a_missing_checkout_root() {
    let scratch = TempDir::new("stage-root");
    let host = scratch.sub("host", 0o700);
    fs::create_dir(host.join("rootfs")).unwrap();
    let fc = scratch.sub("fc", 0o700);
    let (code, _, err) = run(
        &stage_bin(),
        &scratch.path,
        &[
            "--arch",
            "x86_64",
            "--host-context",
            host.to_str().unwrap(),
            "--forgejo-context",
            fc.to_str().unwrap(),
        ],
    );
    assert_eq!(code, 1);
    assert!(err.starts_with("cannot find checkout root above "), "{err}");
}
