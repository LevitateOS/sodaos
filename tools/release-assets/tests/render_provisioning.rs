#![cfg(unix)]

// Shared across suites; each suite uses a subset.
#[allow(dead_code)]
#[path = "render_support/mod.rs"]
mod render_support;

use std::env;
use std::fs;
use std::os::unix::fs::PermissionsExt;
use std::path::{Path, PathBuf};
use std::sync::atomic::Ordering;

use render_support::{
    copy_dir, fixtures, mode_of, provisioning_bin, run, run_env, TempDir, COUNTER,
};

// ---- soda-render-provisioning ----

fn provisioning_inputs(scratch: &TempDir) -> (PathBuf, PathBuf) {
    let key = scratch.file("op.pub", "ssh-ed25519 AAAA synthetic-fixture-key\n", 0o644);
    let hash = scratch.file("pw.hash", "$6$synthetic$fixture-hash\n", 0o600);
    (key, hash)
}

fn fixture_root(scratch: &TempDir) -> PathBuf {
    let root = scratch.path.join("root");
    copy_dir(&fixtures().join("prov-root"), &root);
    root
}

#[test]
fn provisioning_help_and_usage() {
    let scratch = TempDir::new("prov-cli");
    let (code, out, _) = run(&provisioning_bin(), &scratch.path, &["--help"]);
    assert_eq!(code, 0);
    assert!(
        out.starts_with("usage: soda-render-provisioning --operator-key-file PATH"),
        "{out}"
    );
    let (key, hash) = provisioning_inputs(&scratch);
    let key = key.to_str().unwrap().to_string();
    let hash = hash.to_str().unwrap().to_string();
    let out = scratch.path.join("o.bu").to_str().unwrap().to_string();
    for args in [
        vec![],
        vec!["--operator-key-file", key.as_str()],
        vec![
            "--operator-key-file",
            key.as_str(),
            "--root-password-hash-file",
            hash.as_str(),
        ],
        vec!["--out"],
        vec![
            "--operator-key-file",
            key.as_str(),
            "--root-password-hash-file",
            hash.as_str(),
            "--hostname",
            "soda-native-a",
            "--appliance-hostname",
            "x.example",
            "--out",
            out.as_str(),
        ],
        vec![
            "--operator-key-file",
            key.as_str(),
            "--root-password-hash-file",
            hash.as_str(),
            "--bootstrap",
            "full",
            "--out",
            out.as_str(),
        ],
        vec![
            "--operator-key-file",
            key.as_str(),
            "--root-password-hash-file",
            hash.as_str(),
            "--out",
            out.as_str(),
            "--bogus",
            "x",
        ],
    ] {
        let (code, _, err) = run(&provisioning_bin(), &scratch.path, &args);
        assert_eq!(code, 2, "{args:?}");
        assert!(
            err.starts_with("usage: soda-render-provisioning "),
            "{args:?}: {err}"
        );
    }
    let (code, _, err) = run(
        &provisioning_bin(),
        &scratch.path,
        &[
            "--operator-key-file",
            key.as_str(),
            "--root-password-hash-file",
            hash.as_str(),
            "--out",
            out.as_str(),
            "--bogus",
            "x",
        ],
    );
    assert_eq!(code, 2);
    assert!(
        err.ends_with("soda-render-provisioning: error: unrecognized arguments: --bogus x\n"),
        "{err}"
    );
}

#[test]
fn provisioning_matches_the_baked_goldens() {
    let scratch = TempDir::new("prov-golden");
    let root = fixture_root(&scratch);
    let (key, hash) = provisioning_inputs(&scratch);
    let key = key.to_str().unwrap().to_string();
    let hash = hash.to_str().unwrap().to_string();
    let out_dir = scratch.sub("out", 0o700);
    for (name, extra) in [
        ("ext-host", vec!["--hostname", "soda-native-fixture"]),
        (
            "min-host",
            vec![
                "--hostname",
                "soda-native-fixture",
                "--bootstrap",
                "minimal",
            ],
        ),
        (
            "ext-product",
            vec!["--appliance-hostname", "fixture-01.lab.example"],
        ),
    ] {
        let dest = out_dir.join(format!("{name}.bu"));
        let mut args = vec![
            "--operator-key-file",
            key.as_str(),
            "--root-password-hash-file",
            hash.as_str(),
        ];
        args.extend(extra);
        let dest_text = dest.to_str().unwrap().to_string();
        args.extend(["--out", dest_text.as_str()]);
        let (code, _, err) = run(&provisioning_bin(), &root, &args);
        assert_eq!(code, 0, "{name}: {err}");
        assert_eq!(mode_of(&dest), 0o600, "{name}");
        let golden = fs::read(fixtures().join(format!("{name}.bu"))).unwrap();
        assert_eq!(
            fs::read(&dest).unwrap(),
            golden,
            "{name} drifted from the baked golden"
        );
    }
}

#[test]
fn provisioning_host_key_never_enters_argv() {
    let scratch = TempDir::new("prov-keygen");
    let root = fixture_root(&scratch);
    let (key, hash) = provisioning_inputs(&scratch);
    let host_key = scratch.file("hk", "synthetic-private-fixture\n", 0o600);
    let bin_dir = scratch.sub("fakebin", 0o755);
    let log = scratch.path.join("argv.log");
    fs::write(
        bin_dir.join("ssh-keygen"),
        format!(
            "#!/bin/sh\nprintf '%s\\n' \"$@\" > {}\nprintf 'ssh-ed25519 AAAA canned-fixture\\n'\n",
            log.to_str().unwrap()
        ),
    )
    .unwrap();
    fs::set_permissions(
        bin_dir.join("ssh-keygen"),
        fs::Permissions::from_mode(0o755),
    )
    .unwrap();
    let path = format!(
        "{}:{}",
        bin_dir.to_str().unwrap(),
        env::var("PATH").unwrap()
    );
    let out_dir = scratch.sub("out", 0o700);
    let dest = out_dir.join("ext-hostkey.bu");
    let (code, _, err) = run_env(
        &provisioning_bin(),
        &root,
        &[
            "--operator-key-file",
            key.to_str().unwrap(),
            "--root-password-hash-file",
            hash.to_str().unwrap(),
            "--hostname",
            "soda-native-fixture",
            "--ssh-host-key-file",
            host_key.to_str().unwrap(),
            "--out",
            dest.to_str().unwrap(),
        ],
        &[("PATH", path.as_str())],
    );
    assert_eq!(code, 0, "{err}");
    assert_eq!(
        fs::read(&dest).unwrap(),
        fs::read(fixtures().join("ext-hostkey.bu")).unwrap()
    );
    let argv = fs::read_to_string(&log).unwrap();
    assert!(!argv.contains("synthetic-private-fixture"), "{argv}");
    assert!(argv.contains(host_key.to_str().unwrap()), "{argv}");
}

#[test]
fn provisioning_rejections_match_the_script() {
    let scratch = TempDir::new("prov-neg");
    let root = fixture_root(&scratch);
    let (key, hash) = provisioning_inputs(&scratch);
    let key = key.to_str().unwrap().to_string();
    let hash = hash.to_str().unwrap().to_string();
    let out_dir = scratch.sub("out", 0o700);
    let attempt = |extra: &[&str]| {
        let dest = out_dir.join(format!("t{}.bu", COUNTER.fetch_add(1, Ordering::SeqCst)));
        let dest_text = dest.to_str().unwrap().to_string();
        let mut args = vec![
            "--operator-key-file",
            key.as_str(),
            "--root-password-hash-file",
            hash.as_str(),
        ];
        args.extend(extra.iter().copied());
        args.extend(["--out", dest_text.as_str()]);
        let (code, _, err) = run(&provisioning_bin(), &root, &args);
        (code, err, dest.exists())
    };
    // Every render failure keeps the single scripted line and exit 1.
    let (code, err, created) = attempt(&["--hostname", "BAD-NAME!"]);
    assert_eq!((code, created), (1, false));
    assert_eq!(
        err,
        "Private provisioning failed (ValueError); check paths/modes/key format.\n"
    );
    let (code, err, _) = attempt(&["--appliance-hostname", "BAD-NAME!"]);
    assert_eq!(code, 1);
    assert_eq!(
        err,
        "Private provisioning failed (ValueError); check paths/modes/key format.\n"
    );
    // Occupied output keeps its bytes and reports FileExistsError.
    let dest = out_dir.join("occupied.bu");
    fs::write(&dest, "old").unwrap();
    let (code, _, err) = run(
        &provisioning_bin(),
        &root,
        &[
            "--operator-key-file",
            key.as_str(),
            "--root-password-hash-file",
            hash.as_str(),
            "--out",
            dest.to_str().unwrap(),
        ],
    );
    assert_eq!(code, 1);
    assert_eq!(
        err,
        "Private provisioning failed (FileExistsError); check paths/modes/key format.\n"
    );
    assert_eq!(fs::read_to_string(&dest).unwrap(), "old");
    // Group-readable secrets are refused before any output exists.
    fs::set_permissions(Path::new(&hash), fs::Permissions::from_mode(0o644)).unwrap();
    let (code, err, created) = attempt(&[]);
    assert_eq!((code, created), (1, false));
    assert_eq!(
        err,
        "Private provisioning failed (ValueError); check paths/modes/key format.\n"
    );
    fs::set_permissions(Path::new(&hash), fs::Permissions::from_mode(0o600)).unwrap();
    // Plaintext password file.
    let plain = scratch.file("plain.hash", "plaintext\n", 0o600);
    let dest = out_dir.join("plain.bu");
    let (code, _, err) = run(
        &provisioning_bin(),
        &root,
        &[
            "--operator-key-file",
            key.as_str(),
            "--root-password-hash-file",
            plain.to_str().unwrap(),
            "--out",
            dest.to_str().unwrap(),
        ],
    );
    assert_eq!(code, 1);
    assert_eq!(
        err,
        "Private provisioning failed (ValueError); check paths/modes/key format.\n"
    );
    assert!(!dest.exists());
    // Missing operator key.
    let missing = scratch.path.join("missing.pub");
    let dest = out_dir.join("missing.bu");
    let (code, _, err) = run(
        &provisioning_bin(),
        &root,
        &[
            "--operator-key-file",
            missing.to_str().unwrap(),
            "--root-password-hash-file",
            hash.as_str(),
            "--out",
            dest.to_str().unwrap(),
        ],
    );
    assert_eq!(code, 1);
    assert_eq!(
        err,
        "Private provisioning failed (FileNotFoundError); check paths/modes/key format.\n"
    );
    // Open and symlinked output parents.
    let open = scratch.sub("open", 0o755);
    let (code, _, err) = run(
        &provisioning_bin(),
        &root,
        &[
            "--operator-key-file",
            key.as_str(),
            "--root-password-hash-file",
            hash.as_str(),
            "--out",
            open.join("o.bu").to_str().unwrap(),
        ],
    );
    assert_eq!(code, 1);
    assert_eq!(
        err,
        "Private provisioning failed (ValueError); check paths/modes/key format.\n"
    );
    let real = scratch.sub("real", 0o700);
    let link = scratch.path.join("link");
    std::os::unix::fs::symlink(&real, &link).unwrap();
    let (code, _, err) = run(
        &provisioning_bin(),
        &root,
        &[
            "--operator-key-file",
            key.as_str(),
            "--root-password-hash-file",
            hash.as_str(),
            "--out",
            link.join("o.bu").to_str().unwrap(),
        ],
    );
    assert_eq!(code, 1);
    assert_eq!(
        err,
        "Private provisioning failed (ValueError); check paths/modes/key format.\n"
    );
    // A host key the keygen helper rejects.
    let bin_dir = scratch.sub("badbin", 0o755);
    fs::write(
        bin_dir.join("ssh-keygen"),
        "#!/bin/sh\nprintf 'not-a-key\\n'\n",
    )
    .unwrap();
    fs::set_permissions(
        bin_dir.join("ssh-keygen"),
        fs::Permissions::from_mode(0o755),
    )
    .unwrap();
    let path = format!(
        "{}:{}",
        bin_dir.to_str().unwrap(),
        env::var("PATH").unwrap()
    );
    let host_key = scratch.file("badk", "synthetic-private-fixture\n", 0o600);
    let dest = out_dir.join("badk.bu");
    let (code, _, err) = run_env(
        &provisioning_bin(),
        &root,
        &[
            "--operator-key-file",
            key.as_str(),
            "--root-password-hash-file",
            hash.as_str(),
            "--ssh-host-key-file",
            host_key.to_str().unwrap(),
            "--out",
            dest.to_str().unwrap(),
        ],
        &[("PATH", path.as_str())],
    );
    assert_eq!(code, 1);
    assert_eq!(
        err,
        "Private provisioning failed (ValueError); check paths/modes/key format.\n"
    );
    assert!(!dest.exists());
}
