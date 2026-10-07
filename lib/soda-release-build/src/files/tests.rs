use super::*;
use crate::confined_files::{hash_at, Root};
use crate::json_input::{read_json, read_json_at};
use std::os::unix::fs::PermissionsExt;

fn scratch() -> PathBuf {
    let dir =
        std::env::temp_dir().join(format!("soda-relbuild-{}-{}", std::process::id(), unique()));
    std::fs::create_dir_all(&dir).unwrap();
    dir
}

static COUNT: std::sync::atomic::AtomicU64 = std::sync::atomic::AtomicU64::new(0);
fn unique() -> u64 {
    COUNT.fetch_add(1, std::sync::atomic::Ordering::SeqCst)
}

#[test]
fn validators_match_go() {
    assert_eq!(oci_architecture("x86_64").unwrap(), "amd64");
    assert_eq!(
        oci_architecture("aarch64").unwrap_err().message(),
        "expected x86_64"
    );
    assert!(is_digest(&"a".repeat(64)));
    assert!(!is_digest(&"A".repeat(64)));
    assert!(is_revision(&"b".repeat(40)));
    assert!(!is_revision("dirty"));
    require_native("x86_64").unwrap();
    assert!(require_native("other").is_err());
}

#[test]
fn hash_file_refuses_symlinks() {
    let dir = scratch();
    let real = dir.join("real");
    std::fs::write(&real, b"data").unwrap();
    assert_eq!(hash_file(&real).unwrap(), crate::sha256_hex(b"data"));
    let link = dir.join("link");
    std::os::unix::fs::symlink(&real, &link).unwrap();
    assert_eq!(
        hash_file(&link).unwrap_err().message(),
        "regular non-symlink file required"
    );
    let root = Root::open(&dir).unwrap();
    assert_eq!(hash_at(&root, "real").unwrap(), crate::sha256_hex(b"data"));
    assert_eq!(
        hash_at(&root, "link").unwrap_err().message(),
        "regular non-symlink file required"
    );
    assert!(hash_at(&root, "../escape").is_err());
}

#[test]
fn fresh_and_private_directories() {
    let dir = scratch();
    assert!(fresh_directory(Path::new("relative")).is_err());
    let fresh = dir.join("fresh");
    fresh_directory(&fresh).unwrap();
    assert_eq!(
        std::fs::metadata(&fresh).unwrap().permissions().mode() & 0o777,
        0o700
    );
    assert!(fresh_directory(&fresh).is_err());
    let parent_link = dir.join("plink");
    std::os::unix::fs::symlink(&dir, &parent_link).unwrap();
    assert_eq!(
        fresh_directory(&parent_link.join("x"))
            .unwrap_err()
            .message(),
        "symlinked parent refused"
    );
    let out = fresh.join("out");
    private_destination(&out).unwrap();
    std::fs::write(&out, b"x").unwrap();
    assert_eq!(
        private_destination(&out).unwrap_err().message(),
        "output already exists or cannot be inspected"
    );
    let loose = dir.join("loose");
    std::fs::create_dir(&loose).unwrap();
    chmod(&loose, 0o755).unwrap();
    assert_eq!(
        private_destination(&loose.join("o")).unwrap_err().message(),
        "real private output parent required"
    );
}

#[test]
fn write_new_is_exclusive() {
    let dir = scratch();
    let path = dir.join("new");
    write_new(&path, b"bytes", 0o600).unwrap();
    assert_eq!(
        std::fs::metadata(&path).unwrap().permissions().mode() & 0o777,
        0o600
    );
    assert!(write_new(&path, b"bytes", 0o600).is_err());
}

#[test]
fn read_json_strict_round_trip() {
    let dir = scratch();
    let path = dir.join("file.json");
    let body = format!(
        "{{\"SHA256\":\"{}\",\"Mode\":420,\"Directory\":true}}\n",
        "a".repeat(64)
    );
    std::fs::write(&path, &body).unwrap();
    let root = Root::open(&dir).unwrap();
    let (back, digest) = read_json_at::<File>(&root, "file.json").unwrap();
    assert_eq!(back.sha256, "a".repeat(64));
    assert_eq!(back.mode, 0o644);
    assert!(back.directory);
    assert!(back.link.is_empty());
    assert_eq!(digest, crate::sha256_hex(body.as_bytes()));
    // Unknown fields and trailing data are refused.
    std::fs::write(&path, "{\"Mode\":420,\"bogus\":1}").unwrap();
    assert!(read_json::<File>(&path).is_err());
    std::fs::write(&path, "{\"Mode\":420} {}").unwrap();
    assert!(read_json::<File>(&path).is_err());
}

#[test]
fn file_duplicate_fields_validate_only_the_last_match() {
    let dir = scratch();
    let path = dir.join("file.json");
    std::fs::write(&path, r#"{"mode":"bad","Mode":420}"#).unwrap();
    assert_eq!(read_json::<File>(&path).unwrap().mode, 420);
    std::fs::write(&path, r#"{"Mode":420,"mode":"bad"}"#).unwrap();
    assert!(read_json::<File>(&path).is_err());

    std::fs::write(&path, r#"{"Mode":420,"mOdE":null}"#).unwrap();
    assert_eq!(read_json::<File>(&path).unwrap().mode, 0);
    std::fs::write(&path, r#"{"mOdE":null,"Mode":420}"#).unwrap();
    assert_eq!(read_json::<File>(&path).unwrap().mode, 420);
}

#[test]
fn file_mode_keeps_go_integer_token_edges() {
    let dir = scratch();
    let path = dir.join("file.json");
    std::fs::write(&path, r#"{"mode":-0}"#).unwrap();
    assert_eq!(read_json::<File>(&path).unwrap().mode, 0);
    for token in ["0.0", "0e0", "4294967296", "-1"] {
        std::fs::write(&path, format!(r#"{{"mode":{token}}}"#)).unwrap();
        assert!(
            read_json::<File>(&path).is_err(),
            "mode token {token} must fail"
        );
    }
}

#[test]
fn root_rejects_symlink_escape() {
    let dir = scratch();
    let outside = dir.join("outside");
    std::fs::write(&outside, b"sensitive").unwrap();
    let linkroot = dir.join("linkroot");
    std::os::unix::fs::symlink(&dir, &linkroot).unwrap();
    assert!(Root::open(&linkroot).is_err());
    let sub = dir.join("sub");
    std::fs::create_dir(&sub).unwrap();
    std::os::unix::fs::symlink(&dir, sub.join("evil")).unwrap();
    let root = Root::open(&sub).unwrap();
    assert!(root.lstat("evil/outside").is_err());
}
