#![cfg(unix)]

// Shared across suites; each suite uses a subset.
#[allow(dead_code)]
#[path = "locales_support/mod.rs"]
mod locales_support;

use std::fs;

use locales_support::{run, status_of, TempDir, EXTRA, NATIVE};

#[test]
fn native_merge_is_byte_exact_and_silent() {
    let dir = TempDir::new("merge");
    let native = dir.file("native.ini", NATIVE.as_bytes());
    let additions = dir.file("additions.ini", EXTRA.as_bytes());
    let out = dir.path.join("nested/dir/locale.ini");
    let output = run(
        &dir.path,
        &[
            "--native",
            native.to_str().unwrap(),
            "--additions",
            additions.to_str().unwrap(),
            "--out",
            out.to_str().unwrap(),
        ],
    );
    assert_eq!(
        status_of(&output),
        0,
        "stderr: {}",
        String::from_utf8_lossy(&output.stderr)
    );
    assert!(output.stdout.is_empty());
    assert!(output.stderr.is_empty());
    assert_eq!(
        fs::read(&out).unwrap(),
        format!("{NATIVE}\n{EXTRA}").into_bytes()
    );
}

#[test]
fn native_merge_supports_equals_flags() {
    let dir = TempDir::new("equals");
    let native = dir.file("native.ini", NATIVE.as_bytes());
    let additions = dir.file("additions.ini", EXTRA.as_bytes());
    let out = dir.path.join("locale.ini");
    let output = run(
        &dir.path,
        &[
            &format!("--native={}", native.display()),
            &format!("--additions={}", additions.display()),
            &format!("--out={}", out.display()),
        ],
    );
    assert_eq!(
        status_of(&output),
        0,
        "stderr: {}",
        String::from_utf8_lossy(&output.stderr)
    );
    assert_eq!(
        fs::read(&out).unwrap(),
        format!("{NATIVE}\n{EXTRA}").into_bytes()
    );
}

#[test]
fn default_additions_resolve_against_the_working_directory() {
    let dir = TempDir::new("defaults");
    let native = dir.file("native.ini", NATIVE.as_bytes());
    dir.file("appliance/forgejo/i18n/en-US.ini", EXTRA.as_bytes());
    let out = dir.path.join("locale.ini");
    let output = run(
        &dir.path,
        &[
            "--native",
            native.to_str().unwrap(),
            "--out",
            out.to_str().unwrap(),
        ],
    );
    assert_eq!(
        status_of(&output),
        0,
        "stderr: {}",
        String::from_utf8_lossy(&output.stderr)
    );
    assert_eq!(
        fs::read(&out).unwrap(),
        format!("{NATIVE}\n{EXTRA}").into_bytes()
    );
}

#[test]
fn native_over_one_mib_is_refused() {
    let dir = TempDir::new("bound");
    let mut big = NATIVE.as_bytes().to_vec();
    big.extend(std::iter::repeat_n(b'x', 1024 * 1024));
    let native = dir.file("native.ini", &big);
    let additions = dir.file("additions.ini", EXTRA.as_bytes());
    let out = dir.path.join("locale.ini");
    let output = run(
        &dir.path,
        &[
            "--native",
            native.to_str().unwrap(),
            "--additions",
            additions.to_str().unwrap(),
            "--out",
            out.to_str().unwrap(),
        ],
    );
    assert_eq!(status_of(&output), 2);
    let stderr = String::from_utf8_lossy(&output.stderr);
    assert!(
        stderr.contains("native catalog exceeds the 1 MiB bound"),
        "{stderr}"
    );
    assert!(!out.exists());
}

#[test]
fn incomplete_native_catalog_is_rejected() {
    let dir = TempDir::new("incomplete");
    let native = dir.file("native.ini", b"[soda]\nx = y\n");
    let additions = dir.file("additions.ini", EXTRA.as_bytes());
    let out = dir.path.join("locale.ini");
    let output = run(
        &dir.path,
        &[
            "--native",
            native.to_str().unwrap(),
            "--additions",
            additions.to_str().unwrap(),
            "--out",
            out.to_str().unwrap(),
        ],
    );
    assert_eq!(status_of(&output), 1);
    let stderr = String::from_utf8_lossy(&output.stderr);
    assert!(
        stderr.contains("Expected a complete native Forgejo English catalog"),
        "{stderr}"
    );
    assert!(!out.exists());
}

#[test]
fn duplicate_addition_keys_are_rejected() {
    let dir = TempDir::new("dupkey");
    let native = dir.file("native.ini", NATIVE.as_bytes());
    let additions = dir.file("additions.ini", b"[soda]\nx = one\nx = two\n");
    let out = dir.path.join("locale.ini");
    let output = run(
        &dir.path,
        &[
            "--native",
            native.to_str().unwrap(),
            "--additions",
            additions.to_str().unwrap(),
            "--out",
            out.to_str().unwrap(),
        ],
    );
    assert_eq!(
        status_of(&output),
        1,
        "stderr: {}",
        String::from_utf8_lossy(&output.stderr)
    );
    assert!(!out.exists());
}

#[test]
fn non_soda_addition_namespaces_are_rejected() {
    let dir = TempDir::new("namespace");
    let native = dir.file("native.ini", NATIVE.as_bytes());
    let additions = dir.file("additions.ini", b"[settings]\nprofile = Wrong\n");
    let out = dir.path.join("locale.ini");
    let output = run(
        &dir.path,
        &[
            "--native",
            native.to_str().unwrap(),
            "--additions",
            additions.to_str().unwrap(),
            "--out",
            out.to_str().unwrap(),
        ],
    );
    assert_eq!(status_of(&output), 1);
    let stderr = String::from_utf8_lossy(&output.stderr);
    assert!(
        stderr.contains("Additions must use only the Soda namespace"),
        "{stderr}"
    );
    assert!(!out.exists());
}

#[test]
fn native_soda_collisions_are_rejected() {
    let dir = TempDir::new("collision");
    let native = dir.file(
        "native.ini",
        b"[common]\na = b\n[settings]\nc = d\n[soda]\nx = y\n",
    );
    let additions = dir.file("additions.ini", EXTRA.as_bytes());
    let out = dir.path.join("locale.ini");
    let output = run(
        &dir.path,
        &[
            "--native",
            native.to_str().unwrap(),
            "--additions",
            additions.to_str().unwrap(),
            "--out",
            out.to_str().unwrap(),
        ],
    );
    assert_eq!(status_of(&output), 1);
    let stderr = String::from_utf8_lossy(&output.stderr);
    assert!(
        stderr.contains("Native catalog already owns the Soda namespace"),
        "{stderr}"
    );
    assert!(!out.exists());
}

#[test]
fn existing_output_is_never_overwritten() {
    let dir = TempDir::new("exclusive");
    let native = dir.file("native.ini", NATIVE.as_bytes());
    let additions = dir.file("additions.ini", EXTRA.as_bytes());
    let out = dir.file("locale.ini", b"occupied");
    let output = run(
        &dir.path,
        &[
            "--native",
            native.to_str().unwrap(),
            "--additions",
            additions.to_str().unwrap(),
            "--out",
            out.to_str().unwrap(),
        ],
    );
    assert_eq!(
        status_of(&output),
        1,
        "stderr: {}",
        String::from_utf8_lossy(&output.stderr)
    );
    assert_eq!(fs::read(&out).unwrap(), b"occupied");
}
