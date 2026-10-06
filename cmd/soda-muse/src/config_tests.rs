use super::account::account_node;
use super::config::copy_config;
use super::launch_wire::{base64_encode, json_string};
use super::native_action;
use std::fs;
use std::os::unix::fs::PermissionsExt;

#[test]
fn copy_config_skips_credentials_and_copies_tree() {
    let root = std::env::temp_dir().join(format!("soda-muse-cc-{}", std::process::id()));
    let _ = fs::remove_dir_all(&root);
    let source = root.join("src");
    let dest = root.join("dst");
    fs::create_dir_all(source.join("sub")).unwrap();
    fs::write(source.join("settings.json"), b"{}").unwrap();
    fs::write(source.join("auth.json"), b"secret").unwrap();
    fs::write(source.join("sub").join("trust.json"), b"[]").unwrap();
    std::os::unix::fs::symlink(source.join("settings.json"), source.join("link.json")).unwrap();
    copy_config(source.to_str().unwrap(), dest.to_str().unwrap()).unwrap();
    assert_eq!(fs::read(dest.join("settings.json")).unwrap(), b"{}");
    assert_eq!(
        fs::read(dest.join("sub").join("trust.json")).unwrap(),
        b"[]"
    );
    assert_eq!(fs::read(dest.join("link.json")).unwrap(), b"{}");
    assert!(!dest.join("auth.json").exists());
    let mode = fs::metadata(dest.join("settings.json"))
        .unwrap()
        .permissions()
        .mode();
    assert_eq!(mode & 0o777, 0o600);
    // Missing source is a no-op.
    copy_config(
        root.join("absent").to_str().unwrap(),
        root.join("dst2").to_str().unwrap(),
    )
    .unwrap();
    // Oversized file refused.
    let big = source.join("big.json");
    fs::write(&big, vec![0u8; (1 << 20) + 1]).unwrap();
    assert!(copy_config(
        source.to_str().unwrap(),
        root.join("dst3").to_str().unwrap()
    )
    .is_err());
    let _ = fs::remove_dir_all(&root);
}

#[test]
fn read_config_contract_shapes() {
    // Pure shape check: base64 values under sorted keys.
    let mut out = String::from("{");
    out.push_str(&json_string("settings.json"));
    out.push(':');
    out.push_str(&json_string(&base64_encode(b"{}")));
    out.push_str("}\n");
    assert_eq!(out, "{\"settings.json\":\"e30=\"}\n");
}

#[test]
fn account_markers_reject_unsafe_nodes() {
    let root = std::env::temp_dir().join(format!("soda-muse-ac-{}", std::process::id()));
    let _ = fs::remove_dir_all(&root);
    fs::create_dir_all(&root).unwrap();
    // Temp files are user-owned, never root-owned: unsafe record.
    assert!(account_node(root.to_str().unwrap(), true).is_err());
    let marker = root.join("login");
    fs::write(&marker, b"42").unwrap();
    assert!(account_node(marker.to_str().unwrap(), false).is_err());
    let _ = fs::remove_dir_all(&root);
}

#[test]
fn native_dispatch_errors_match_go() {
    let v = |s: &[&str]| s.iter().map(|x| x.to_string()).collect::<Vec<_>>();
    assert_eq!(
        native_action(&v(&["--soda-copy-config", "a"])),
        Some((1, Some(String::from("config source and view required"))))
    );
    assert_eq!(
        native_action(&v(&["--soda-exec", "a"])),
        Some((1, Some(String::from("execution root and cwd required"))))
    );
    assert_eq!(
        native_action(&v(&["--soda-check"])),
        Some((1, Some(String::from("one metadata input required"))))
    );
    assert_eq!(native_action(&[]), None);
    assert_eq!(native_action(&v(&["--version"])), None);
    assert_eq!(native_action(&v(&["prompt text"])), None);
}
