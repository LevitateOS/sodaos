use std::io::Write;
use std::os::unix::fs::{MetadataExt, PermissionsExt};

use super::*;

fn temp_dir(prefix: &str) -> std::path::PathBuf {
    let mut dir = std::env::temp_dir();
    dir.push(format!("{prefix}-{}-{}", std::process::id(), rand_suffix()));
    std::fs::create_dir_all(&dir).unwrap();
    dir
}

fn rand_suffix() -> u64 {
    use std::collections::hash_map::DefaultHasher;
    use std::hash::{Hash, Hasher};
    let mut hasher = DefaultHasher::new();
    std::time::SystemTime::now().hash(&mut hasher);
    std::thread::current().id().hash(&mut hasher);
    hasher.finish()
}

#[test]
fn lexical_clean_matches_go() {
    for (input, want) in [
        ("", "."),
        (".", "."),
        ("a/../b", "b"),
        ("../escape", "../escape"),
        ("/a//b/", "/a/b"),
        ("/..", "/"),
        ("/a/./b", "/a/b"),
        ("a/b", "a/b"),
    ] {
        assert_eq!(lexical_clean(input), want, "clean {input:?}");
    }
}

#[test]
fn private_file_matrix() {
    let dir = temp_dir("soda-files-private");
    let good = dir.join("good");
    std::fs::write(&good, b"secret").unwrap();
    std::fs::set_permissions(&good, std::fs::Permissions::from_mode(0o600)).unwrap();
    assert_eq!(private_file(good.to_str().unwrap()).unwrap(), b"secret");

    let open = dir.join("open");
    std::fs::write(&open, b"secret").unwrap();
    std::fs::set_permissions(&open, std::fs::Permissions::from_mode(0o644)).unwrap();
    let err = private_file(open.to_str().unwrap()).unwrap_err();
    assert_eq!(err.to_string(), "restricted regular input required: open");

    let link = dir.join("link");
    std::os::unix::fs::symlink(&good, &link).unwrap();
    assert!(private_file(link.to_str().unwrap()).is_err());
    assert!(private_file(dir.to_str().unwrap()).is_err());
    assert_eq!(
        private_file("relative/path").unwrap_err().to_string(),
        "absolute private input required"
    );
    assert!(private_file(dir.join("missing").to_str().unwrap()).is_err());

    let big = dir.join("big");
    std::fs::write(&big, vec![0u8; (PRIVATE_FILE_LIMIT + 1) as usize]).unwrap();
    std::fs::set_permissions(&big, std::fs::Permissions::from_mode(0o600)).unwrap();
    assert!(private_file(big.to_str().unwrap()).is_err());
    std::fs::remove_dir_all(&dir).unwrap();
}

#[test]
fn hash_file_vectors() {
    let dir = temp_dir("soda-files-hash");
    let path = dir.join("data");
    std::fs::write(&path, b"abc").unwrap();
    assert_eq!(
        hash_file(path.to_str().unwrap()).unwrap(),
        "ba7816bf8f01cfea414140de5dae2223b00361a396177a9cb410ff61f20015ad"
    );
    let link = dir.join("link");
    std::os::unix::fs::symlink(&path, &link).unwrap();
    assert_eq!(
        hash_file(link.to_str().unwrap()).unwrap_err().to_string(),
        "regular non-symlink file required"
    );
    std::fs::remove_dir_all(&dir).unwrap();
}

#[test]
fn owned_dir_confines_and_links() {
    let dir = temp_dir("soda-files-root");
    let root = OwnedDir::open(dir.to_str().unwrap()).unwrap();
    root.mkdir_at("sub", 0o700).unwrap();
    assert!(root.mkdir_at("sub", 0o700).unwrap_err().io_kind().is_some());
    {
        let mut file = root.create_new_at("sub/file", 0o600).unwrap();
        file.write_all(b"data").unwrap();
    }
    assert_eq!(
        hash_at(&root, "sub/file").unwrap(),
        crate::sha256::hex_lower(&crate::sha256::digest(b"data"))
    );
    root.link_at("sub/file", "linked").unwrap();
    assert!(root.link_at("sub/file", "linked").is_err());
    assert!(
        root.create_new_at("sub/file", 0o600).unwrap_err().io_kind()
            == Some(std::io::ErrorKind::AlreadyExists)
    );

    let outside = temp_dir("soda-files-outside");
    let escape = dir.join("escape");
    std::os::unix::fs::symlink(&outside, &escape).unwrap();
    assert!(root.open_file_at("escape/x").is_err());
    assert!(root.sub_dir("escape").is_err());
    assert!(root.lstat_at("escape").unwrap().is_symlink);

    let mut found = Vec::new();
    root.walk_files(&mut |rel: &str, regular: bool| {
        found.push((rel.to_string(), regular));
        Ok(())
    })
    .unwrap();
    found.sort();
    assert_eq!(
        found,
        vec![
            ("escape".to_string(), false),
            ("linked".to_string(), true),
            ("sub/file".to_string(), true),
        ]
    );
    std::fs::remove_dir_all(&dir).unwrap();
    std::fs::remove_dir_all(&outside).unwrap();
}

#[test]
fn fresh_directory_and_destination_gates() {
    let dir = temp_dir("soda-files-fresh");
    let fresh = dir.join("fresh");
    fresh_directory(fresh.to_str().unwrap()).unwrap();
    assert_eq!(
        std::fs::metadata(&fresh).unwrap().permissions().mode() & 0o777,
        0o700
    );
    let err = fresh_directory(fresh.to_str().unwrap()).unwrap_err();
    assert_eq!(err.io_kind(), Some(std::io::ErrorKind::AlreadyExists));
    assert!(fresh_directory("relative").is_err());

    let link_parent = dir.join("linkparent");
    std::os::unix::fs::symlink(&dir, &link_parent).unwrap();
    let err = fresh_directory(link_parent.join("x").to_str().unwrap()).unwrap_err();
    assert_eq!(err.to_string(), "symlinked parent refused");

    let parent = dir.join("parent");
    std::fs::create_dir(&parent).unwrap();
    std::fs::set_permissions(&parent, std::fs::Permissions::from_mode(0o700)).unwrap();
    let out = parent.join("out.md");
    private_destination(out.to_str().unwrap()).unwrap();
    write_new(out.to_str().unwrap(), b"bytes", 0o600).unwrap();
    assert_eq!(
        private_destination(out.to_str().unwrap())
            .unwrap_err()
            .to_string(),
        "output already exists or cannot be inspected"
    );
    std::fs::set_permissions(&parent, std::fs::Permissions::from_mode(0o755)).unwrap();
    assert_eq!(
        private_destination(parent.join("other").to_str().unwrap())
            .unwrap_err()
            .to_string(),
        "real private output parent required"
    );
    std::fs::remove_dir_all(&dir).unwrap();
}

/// Count this process's descriptors currently pointing at `dir` itself,
/// identified by device + inode. `fs::metadata` follows each
/// `/proc/self/fd` entry to its target; other tests' descriptors point at
/// their own fixtures and never match.
fn fixture_dir_fds(dir: &std::path::Path) -> usize {
    let want = std::fs::metadata(dir).unwrap();
    std::fs::read_dir("/proc/self/fd")
        .unwrap()
        .filter_map(|entry| entry.ok())
        .filter_map(|entry| std::fs::metadata(entry.path()).ok())
        .filter(|meta| meta.dev() == want.dev() && meta.ino() == want.ino())
        .count()
}

#[test]
fn walk_lstat_error_closes_stream() {
    let dir = temp_dir("soda-files-walk-leak");
    let root = OwnedDir::open(dir.to_str().unwrap()).unwrap();
    for i in 0..16 {
        std::fs::write(dir.join(format!("f{i:02}")), b"x").unwrap();
    }
    let mut visits = 0u32;
    // The first visit deletes every entry; names already buffered by
    // readdir then fail lstat with ENOENT, exercising the error path.
    let err = root
        .walk_files(&mut |_rel: &str, _regular: bool| {
            visits += 1;
            if visits == 1 {
                for i in 0..16 {
                    let _ = std::fs::remove_file(dir.join(format!("f{i:02}")));
                }
            }
            Ok(())
        })
        .unwrap_err();
    assert_eq!(err.io_kind(), Some(std::io::ErrorKind::NotFound));
    assert_eq!(visits, 1);
    // The walk's stream is gone; dropping the handle must leave no
    // descriptor behind that still points at this fixture.
    drop(root);
    assert_eq!(fixture_dir_fds(&dir), 0);
    std::fs::remove_dir_all(&dir).unwrap();
}

fn walk_all(root: &OwnedDir) -> Vec<(String, bool)> {
    let mut found = Vec::new();
    root.walk_files(&mut |rel: &str, regular: bool| {
        found.push((rel.to_string(), regular));
        Ok(())
    })
    .unwrap();
    found.sort();
    found
}

#[test]
fn walk_repeated_scans_agree() {
    let dir = temp_dir("soda-files-walk-repeat");
    let root = OwnedDir::open(dir.to_str().unwrap()).unwrap();
    root.mkdir_at("sub", 0o700).unwrap();
    std::fs::write(dir.join("top"), b"t").unwrap();
    std::fs::write(dir.join("sub").join("nested"), b"n").unwrap();
    // Finalization scans one evidence root repeatedly (secrets, then
    // hashes, then rescan); every pass must see the retained files.
    let first = walk_all(&root);
    let second = walk_all(&root);
    let third = walk_all(&root);
    assert_eq!(first, second);
    assert_eq!(second, third);
    assert_eq!(
        first,
        vec![("sub/nested".to_string(), true), ("top".to_string(), true),]
    );
    std::fs::remove_dir_all(&dir).unwrap();
}
