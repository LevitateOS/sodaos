use std::env;
use std::fs;
use std::os::unix::fs::PermissionsExt;
use std::path::Path;

use super::config::{config_json, json_escape, trust_json, worker_json};
use super::controller::controller_cargo_argv;
use super::fixture_authority::random_hex_passphrase;
use super::preflight::bridge_ip;
use super::process::{git_tree_clean, pipe2, run_in_dir, Captured};
use super::storage::units_have_active;
use super::{command_v_in, current_umask, env_or, stripped, write_staged, Exit, WORKER_POLICY_SRC};

#[test]
fn json_escape_matches_python_dumps() {
    let input = "a\"b\\c\nd\re\tf\x08g\x0ch\x01i\x7fj\u{80}ké😀l\x1fm";
    assert_eq!(
        json_escape(input),
        "a\\\"b\\\\c\\nd\\re\\tf\\bg\\fh\\u0001i\\u007fj\\u0080k\\u00e9\\ud83d\\ude00l\\u001fm"
    );
    assert_eq!(json_escape("plain / path-_name.1"), "plain / path-_name.1");
    assert_eq!(json_escape(""), "");
}

#[test]
fn worker_json_matches_python_dump_without_trailing_newline() {
    let got = worker_json(
        "/usr/local/lib/soda/soda-build",
        "/home/op/sodaos",
        "/home/op/forgejo",
        "/home/op/sodaos/.artifacts/releases/isolated",
        "/home/soda-candidate",
        "/home/soda-candidate/home",
        "/home/soda-candidate/run",
        "/var/lib/soda-candidate-tools",
        "/var/lib/soda-candidate-authority",
    );
    let want = "{\n  \"Executable\": \"/usr/local/lib/soda/soda-build\",\n  \"Source\": \"/home/op/sodaos\",\n  \"ForgejoSource\": \"/home/op/forgejo\",\n  \"OutputParent\": \"/home/op/sodaos/.artifacts/releases/isolated\",\n  \"StorageRoot\": \"/home/soda-candidate\",\n  \"BuildHome\": \"/home/soda-candidate/home\",\n  \"Runtime\": \"/home/soda-candidate/run\",\n  \"Tools\": \"/var/lib/soda-candidate-tools\",\n  \"MediaAuthorityDirectory\": \"/var/lib/soda-candidate-authority\"\n}";
    assert_eq!(got, want);
}

#[test]
fn trust_json_matches_python_dump_with_trailing_newline() {
    let got = trust_json(
        "ghcr.io/levitateos/sodaos",
        424242,
        [
            "artifact-pub\n",
            "candidate-pub\n",
            "preview-pub\n",
            "stable-pub\n",
        ],
    );
    let want = "{\n  \"Format\": 1,\n  \"Prefix\": \"ghcr.io/levitateos/sodaos\",\n  \"Epoch\": 1,\n  \"Keys\": {\n    \"artifact\": [\n      \"artifact-pub\\n\"\n    ],\n    \"candidate\": [\n      \"candidate-pub\\n\"\n    ],\n    \"preview\": [\n      \"preview-pub\\n\"\n    ],\n    \"stable\": [\n      \"stable-pub\\n\"\n    ]\n  },\n  \"NotBefore\": 423642,\n  \"MaxAgeSeconds\": 3600,\n  \"ClockSkewSeconds\": 10,\n  \"MinimumSequence\": {\n    \"candidate\": 1,\n    \"preview\": 1,\n    \"stable\": 1\n  }\n}\n";
    assert_eq!(got, want);
}

#[test]
fn config_json_matches_python_dump_with_trailing_newline() {
    let got = config_json();
    let want = "{\n  \"Trust\": \"/run/soda-media-authority/trust.json\",\n  \"Keys\": {\n    \"Key\": \"/run/soda-media-authority/artifact.private\",\n    \"Passphrase\": \"/run/soda-media-authority/passphrase\"\n  }\n}\n";
    assert_eq!(got, want);
}

#[test]
fn units_have_active_matches_awk_third_field() {
    assert!(units_have_active(
        b"soda-build-x.service loaded active running desc\n"
    ));
    assert!(units_have_active(
        b"soda-build-x.service loaded activating start desc\n"
    ));
    assert!(units_have_active(
        b"soda-build-x.service loaded deactivating stop desc\n"
    ));
    assert!(!units_have_active(
        b"soda-build-x.service loaded failed failed desc\n"
    ));
    assert!(!units_have_active(b""));
    assert!(!units_have_active(b"short line\n"));
    assert!(!units_have_active(
        b"other.service loaded inactive dead desc\nsoda-build-y.service loaded failed failed desc\n"
    ));
    assert!(units_have_active(
        b"other.service loaded inactive dead desc\nsoda-build-y.service loaded active running desc\n"
    ));
}

#[test]
fn bridge_ip_takes_first_line_fourth_field_before_slash() {
    assert_eq!(
        bridge_ip(b"2: virbr0    inet 192.168.122.1/24 brd 192.168.122.255 scope global virbr0\n"),
        Some("192.168.122.1".to_string())
    );
    assert_eq!(bridge_ip(b""), None);
    assert_eq!(bridge_ip(b"2: virbr0\n"), None);
    assert_eq!(
        bridge_ip(b"2: virbr0    inet 10.0.0.1 scope global\n"),
        Some("10.0.0.1".to_string())
    );
}

#[test]
fn env_or_falls_back_on_unset_or_empty() {
    let key = "SODA_CANDIDATE_SETUP_TEST_ENV_OR";
    env::remove_var(key);
    assert_eq!(env_or(key, "dflt"), "dflt");
    env::set_var(key, "");
    assert_eq!(env_or(key, "dflt"), "dflt");
    env::set_var(key, "value");
    assert_eq!(env_or(key, "dflt"), "value");
    env::remove_var(key);
}

#[test]
fn stripped_removes_only_trailing_newlines() {
    assert_eq!(stripped(b"a\n"), b"a");
    assert_eq!(stripped(b"a\n\n\n"), b"a");
    assert_eq!(stripped(b"a"), b"a");
    assert_eq!(stripped(b""), b"");
    assert_eq!(stripped(b"a\nb\n"), b"a\nb");
    assert_eq!(stripped(b"\n"), b"");
}

#[test]
fn command_v_searches_path_for_executables() {
    let dir = env::temp_dir().join(format!("soda-setup-cmdv-{}", std::process::id()));
    let _ = fs::remove_dir_all(&dir);
    fs::create_dir_all(&dir).unwrap();
    let tool = dir.join("soda-test-tool");
    let flat = dir.join("soda-test-flat");
    fs::write(&tool, b"#!/bin/sh\nexit 0\n").unwrap();
    fs::write(&flat, b"data").unwrap();
    fs::set_permissions(&tool, fs::Permissions::from_mode(0o755)).unwrap();
    fs::set_permissions(&flat, fs::Permissions::from_mode(0o644)).unwrap();
    let found = command_v_in("soda-test-tool", dir.as_os_str());
    let missing = command_v_in("soda-test-flat", dir.as_os_str()).is_none()
        && command_v_in("soda-test-absent", dir.as_os_str()).is_none();
    let _ = fs::remove_dir_all(&dir);
    assert_eq!(found, Some(tool));
    assert!(missing);
}

#[test]
fn pipe2_reports_last_nonzero_by_position() {
    assert_eq!(pipe2(("true", &[]), ("true", &[])), (0, Vec::new()));
    assert_eq!(pipe2(("true", &[]), ("false", &[])).0, 1);
    // pipefail: the first failure still fails the pipeline.
    assert_eq!(pipe2(("false", &[]), ("true", &[])).0, 1);
    let (code, out) = pipe2(("echo", &["hi"]), ("tr", &["a-z", "A-Z"]));
    assert_eq!((code, out), (0, b"HI\n".to_vec()));
}

#[test]
fn write_staged_matches_redirection_bytes_and_mode() {
    let path = env::temp_dir().join(format!("soda-setup-stage-{}", std::process::id()));
    let _ = fs::remove_file(&path);
    write_staged(&path, b"bytes\n").unwrap();
    assert_eq!(fs::read(&path).unwrap(), b"bytes\n");
    let mode = fs::metadata(&path).unwrap().permissions().mode() & 0o777;
    assert_eq!(mode, 0o666 & !current_umask() & 0o777);
    let _ = fs::remove_file(&path);
}

#[test]
fn passphrase_is_64_lowercase_hex_without_newline() {
    let passphrase = random_hex_passphrase().unwrap();
    assert_eq!(passphrase.len(), 64);
    assert!(passphrase
        .bytes()
        .all(|b| b.is_ascii_hexdigit() && !b.is_ascii_uppercase()));
    assert_ne!(
        random_hex_passphrase().unwrap(),
        random_hex_passphrase().unwrap()
    );
}

#[test]
fn run_in_dir_maps_missing_directory_to_fail_message() {
    match run_in_dir(
        "true",
        &[],
        "/definitely/not/a/soda-setup-dir",
        true,
        "cannot warm Bun cache",
    ) {
        Err(Exit::Fail(msg)) => assert_eq!(msg, "cannot warm Bun cache"),
        other => panic!("expected Fail, got {other:?}"),
    }
}

#[test]
fn git_tree_clean_requires_successful_empty_status() {
    // D01-F3: only a successful status with empty output proves clean.
    assert!(git_tree_clean(&Captured::Done(0, Vec::new())));
    assert!(git_tree_clean(&Captured::Done(0, b"\n".to_vec())));
    assert!(!git_tree_clean(&Captured::Done(
        0,
        b" M src/main.rs\n".to_vec()
    )));
    assert!(!git_tree_clean(&Captured::Done(1, Vec::new())));
    assert!(!git_tree_clean(&Captured::SpawnFailed(127)));
}

#[test]
fn controller_build_selects_rust_release_tools() {
    // D01-F2: pinned Rust recipe, never the retired Go paths.
    let argv = controller_cargo_argv();
    assert_eq!(
        argv.as_slice(),
        &[
            "build",
            "--release",
            "--locked",
            "-p",
            "soda-release-tools",
            "--bin",
            "soda-build",
            "--bin",
            "soda-candidate",
        ]
    );
    assert!(!argv.iter().any(|a| a.contains("tools/soda-")));
}

#[test]
fn worker_policy_input_resolves_in_checkout() {
    // CORR-C-005: the checkmodule input must resolve at its selected location.
    let manifest = Path::new(env!("CARGO_MANIFEST_DIR"));
    let root = manifest.ancestors().nth(2).unwrap();
    assert!(root.join(WORKER_POLICY_SRC).is_file());
}
