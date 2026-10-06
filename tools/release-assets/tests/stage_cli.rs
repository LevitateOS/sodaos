//! CLI-surface tests for the stage/render ports.
//!
//! Usage, help and refusal paths run without a checkout; golden paths run
//! inside synthetic roots built from `tests/fixtures`. The provisioning
//! goldens were baked by `scripts/render-provisioning.py` before the
//! cutover and pin the rendered JSON byte for byte. Full stage success
//! coverage lives in `tests/build/test_sodaspaces.py`, which drives the
//! built binary end to end; the real-emblem render is pinned here and in
//! `scripts/terminal_branding_test.go`.

#![cfg(unix)]

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
        let path = env::temp_dir().join(format!(
            "soda-stage-render-{tag}-{}-{id}",
            std::process::id()
        ));
        let _ = fs::remove_dir_all(&path);
        fs::create_dir_all(&path).unwrap();
        fs::set_permissions(&path, fs::Permissions::from_mode(0o700)).unwrap();
        TempDir { path }
    }

    fn sub(&self, name: &str, mode: u32) -> PathBuf {
        let path = self.path.join(name);
        fs::create_dir_all(&path).unwrap();
        fs::set_permissions(&path, fs::Permissions::from_mode(mode)).unwrap();
        path
    }

    fn file(&self, name: &str, contents: &str, mode: u32) -> PathBuf {
        let path = self.path.join(name);
        fs::write(&path, contents).unwrap();
        fs::set_permissions(&path, fs::Permissions::from_mode(mode)).unwrap();
        path
    }
}

impl Drop for TempDir {
    fn drop(&mut self) {
        let _ = fs::remove_dir_all(&self.path);
    }
}

fn fixtures() -> PathBuf {
    PathBuf::from(env!("CARGO_MANIFEST_DIR")).join("tests/fixtures")
}

fn repo_root() -> PathBuf {
    Path::new(env!("CARGO_MANIFEST_DIR"))
        .ancestors()
        .nth(2)
        .expect("crate sits two levels under the checkout root")
        .to_path_buf()
}

fn copy_dir(src: &Path, dst: &Path) {
    fs::create_dir_all(dst).unwrap();
    for entry in fs::read_dir(src).unwrap() {
        let entry = entry.unwrap();
        let from = entry.path();
        let to = dst.join(entry.file_name());
        if entry.file_type().unwrap().is_dir() {
            copy_dir(&from, &to);
        } else {
            fs::copy(&from, &to).unwrap();
        }
    }
}

fn stage_bin() -> PathBuf {
    PathBuf::from(env!("CARGO_BIN_EXE_soda-stage"))
}

fn provisioning_bin() -> PathBuf {
    PathBuf::from(env!("CARGO_BIN_EXE_soda-render-provisioning"))
}

fn logo_bin() -> PathBuf {
    PathBuf::from(env!("CARGO_BIN_EXE_soda-render-terminal-logo"))
}

fn run(bin: &Path, cwd: &Path, args: &[&str]) -> (i32, String, String) {
    run_env(bin, cwd, args, &[])
}

fn run_env(bin: &Path, cwd: &Path, args: &[&str], env: &[(&str, &str)]) -> (i32, String, String) {
    let mut command = Command::new(bin);
    command.current_dir(cwd).args(args);
    for (key, value) in env {
        command.env(key, value);
    }
    let output = command.output().expect("spawn stage-render binary");
    (
        output.status.code().unwrap_or(-1),
        String::from_utf8_lossy(&output.stdout).into_owned(),
        String::from_utf8_lossy(&output.stderr).into_owned(),
    )
}

fn mode_of(path: &Path) -> u32 {
    fs::metadata(path).unwrap().permissions().mode() & 0o777
}

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

// ---- soda-render-terminal-logo ----

fn logo_root(scratch: &TempDir) -> PathBuf {
    let root = scratch.path.join("logoroot");
    let source = root.join("assets/branding/source");
    let terminal = root.join("assets/branding/terminal");
    fs::create_dir_all(&source).unwrap();
    fs::create_dir_all(&terminal).unwrap();
    let marker = root.join("assets/branding/forgejo");
    fs::create_dir_all(&marker).unwrap();
    fs::write(marker.join("forgejo-payload.json"), "{}").unwrap();
    let repo = repo_root();
    fs::copy(
        repo.join("assets/branding/source/soda-symbol-brutalist.svg"),
        source.join("soda-symbol-brutalist.svg"),
    )
    .unwrap();
    root
}

#[test]
fn logo_help_and_usage() {
    let scratch = TempDir::new("logo-cli");
    let (code, out, _) = run(&logo_bin(), &scratch.path, &["--help"]);
    assert_eq!(code, 0);
    assert_eq!(out, "usage: soda-render-terminal-logo [--check]\n\nSample the canonical polygon emblem into a 32-column, 16-row ASCII mark.\n\noptions:\n  --check  verify the committed outputs instead of rewriting them\n");
    for args in [vec!["stray"], vec!["--bogus"], vec!["--check=x"]] {
        let (code, _, err) = run(&logo_bin(), &scratch.path, &args);
        assert_eq!(code, 2, "{args:?}");
        assert!(
            err.starts_with("usage: soda-render-terminal-logo "),
            "{args:?}: {err}"
        );
    }
}

#[test]
fn logo_renders_the_canonical_emblem_byte_for_byte() {
    let scratch = TempDir::new("logo-real");
    let root = logo_root(&scratch);
    let (code, _, err) = run(&logo_bin(), &root, &[]);
    assert_eq!(code, 0, "{err}");
    let repo = repo_root();
    for name in ["sodaos.txt", "motd.txt"] {
        assert_eq!(
            fs::read(root.join("assets/branding/terminal").join(name)).unwrap(),
            fs::read(repo.join("assets/branding/terminal").join(name)).unwrap(),
            "{name} drifted from the committed branding"
        );
    }
    let (code, _, err) = run(&logo_bin(), &root, &["--check"]);
    assert_eq!(code, 0, "{err}");
    fs::write(root.join("assets/branding/terminal/sodaos.txt"), "drift").unwrap();
    let (code, _, err) = run(&logo_bin(), &root, &["--check"]);
    assert_eq!(code, 1);
    assert!(err.starts_with("Stale terminal branding: "), "{err}");
    assert!(err.ends_with("sodaos.txt\n"), "{err}");
}

#[test]
fn logo_gates_match_the_script() {
    let scratch = TempDir::new("logo-gate");
    let root = logo_root(&scratch);
    let svg_path = root.join("assets/branding/source/soda-symbol-brutalist.svg");
    let svg = fs::read_to_string(&svg_path).unwrap();
    for (name, bad, want) in [
        (
            "viewbox",
            svg.replace("0 0 128 128", "0 0 64 64"),
            "Unexpected emblem viewBox",
        ),
        (
            "layers",
            svg.replacen("#df001b", "#000000", 1),
            "Unexpected emblem layers",
        ),
        (
            "fill-rule",
            svg.replacen("evenodd", "nonzero", 1),
            "Unexpected emblem fill rule",
        ),
        (
            "geometry",
            svg.replacen(
                "M40 0H128V88L88 128H0V40Z M24 72L48 56L64 72V96H24Z",
                "M40 Q",
                1,
            ),
            "could not convert string to float: 'Q'",
        ),
    ] {
        fs::write(&svg_path, &bad).unwrap();
        let (code, _, err) = run(&logo_bin(), &root, &[]);
        assert_eq!(code, 1, "{name}");
        assert_eq!(err, format!("{want}\n"), "{name}");
    }
}
