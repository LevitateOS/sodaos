use crate::command::{podman_streamed, wait_output};
use crate::interface::FdGuard;
use crate::stage::INSTALL_SCRIPT;
use crate::test_support::{emit_synthetic, run_install_script, synthetic_feeds, TestDir};
use std::ffi::CString;
use std::fs;
use std::io::Read;
use std::os::unix::fs::{MetadataExt, PermissionsExt};
use std::process::{Command, Stdio};
use std::time::{Duration, Instant};

use super::{emit_archive, tool_header};

#[test]
fn ustar_header_metadata_is_explicit() {
    let header = tool_header("muse", 14).unwrap();
    assert_eq!(header.path().unwrap().to_str().unwrap(), "muse");
    assert_eq!(header.mode().unwrap(), 0o755);
    assert_eq!(header.uid().unwrap(), 0);
    assert_eq!(header.gid().unwrap(), 0);
    assert_eq!(header.size().unwrap(), 14);
    assert_eq!(header.mtime().unwrap(), 0);
    assert_eq!(header.entry_type().as_byte(), b'0');
    assert_eq!(header.username().unwrap().unwrap_or(""), "");
    assert_eq!(header.groupname().unwrap().unwrap_or(""), "");
    assert!(header.link_name_bytes().is_none());
    assert_eq!(header.device_major().unwrap().unwrap_or(0), 0);
    assert_eq!(header.device_minor().unwrap().unwrap_or(0), 0);
    assert!(header.cksum().is_ok());
    assert!(tool_header(&"n".repeat(101), 0).is_err());
    assert!(tool_header("muse", 8u64.pow(11)).is_err());
}

#[test]
fn archive_is_deterministic_and_uses_declared_entries() {
    let dir = TestDir::make("deterministic-archive");
    let path = dir.path("tool");
    fs::write(&path, b"synthetic tool bytes").unwrap();
    let c = CString::new(path).unwrap();
    let fd = unsafe { libc::open(c.as_ptr(), libc::O_RDONLY) };
    assert!(fd >= 0);
    let _guard = FdGuard(fd);
    let feeds = vec![(String::from("bin/muse"), fd, 20u64)];
    let write = || {
        let mut archive = Vec::new();
        emit_archive(
            &mut |bytes: &[u8]| {
                archive.extend_from_slice(bytes);
                Ok(())
            },
            &feeds,
        )
        .unwrap();
        archive
    };
    let first = write();
    assert_eq!(first, write());
    let mut tar = tar::Archive::new(first.as_slice());
    let mut entries = tar.entries().unwrap();
    let mut entry = entries.next().unwrap().unwrap();
    assert_eq!(entry.path().unwrap().to_str().unwrap(), "bin/muse");
    assert_eq!(entry.size(), 20);
    let mut contents = Vec::new();
    entry.read_to_end(&mut contents).unwrap();
    assert_eq!(contents, b"synthetic tool bytes");
    assert!(entries.next().is_none());
    assert!(first.ends_with(&[0u8; 1024]));
}

#[test]
fn tool_replacement_preserves_other_files_and_refuses_symlinks() {
    let dir = TestDir::make("stage");
    let targets: Vec<String> = ["bin/muse", "bin/soda-identity-compose", "libexec/soda/muse"]
        .iter()
        .map(|t| dir.path(t))
        .collect();
    for t in &targets {
        fs::create_dir_all(std::path::Path::new(t).parent().unwrap()).unwrap();
        fs::write(t, b"obsolete public tool").unwrap();
    }
    let retained = dir.path("account-state");
    fs::write(&retained, b"preserved account and project state").unwrap();
    fs::set_permissions(&retained, fs::Permissions::from_mode(0o600)).unwrap();
    let before = fs::symlink_metadata(&retained).unwrap();
    let archive = emit_synthetic();
    // 3 headers + 3 data blocks + 2 trailer blocks.
    assert_eq!(archive.len(), 3 * 512 + 3 * 512 + 1024);
    // The install script is byte-pinned where transcription once dropped
    // the space before the first loop's done.
    assert!(INSTALL_SCRIPT.contains("fi\n done\nfor target"));
    let (ok, stderr) = run_install_script(&archive, &targets);
    assert!(ok, "public replacement failed: {stderr}");
    let after = fs::symlink_metadata(&retained).unwrap();
    assert_eq!((before.dev(), before.ino()), (after.dev(), after.ino()));
    assert_eq!(
        fs::read(&retained).unwrap(),
        b"preserved account and project state"
    );
    for t in &targets {
        let body = fs::read(t).unwrap();
        assert!(body.starts_with(b"synthetic "), "tool not replaced: {t}");
    }
    fs::remove_file(&targets[0]).unwrap();
    std::os::unix::fs::symlink(&retained, &targets[0]).unwrap();
    let (ok, _) = run_install_script(&archive, &targets);
    assert!(!ok, "target symlink accepted");
    assert_eq!(
        fs::read(&retained).unwrap(),
        b"preserved account and project state"
    );
}

#[test]
fn streamed_stage_delivery() {
    // A fake podman proves the feeder/child contract: the child must
    // not inherit the stdin write end, or its reader never sees EOF.
    // (A raw pipe without O_CLOEXEC deadlocked here until the deadline.)
    let dir = TestDir::make("streamed");
    let bindir = dir.path("bin");
    fs::create_dir(&bindir).unwrap();
    fs::write(
        format!("{bindir}/podman"),
        "#!/bin/sh\nprintf \"%s\\0\" \"$@\" > \"$E2E/argv.bin\"\ncat > \"$E2E/stdin.bin\"\nexit 0\n",
    )
    .unwrap();
    fs::set_permissions(
        format!("{bindir}/podman"),
        fs::Permissions::from_mode(0o755),
    )
    .unwrap();
    std::env::set_var(
        "PATH",
        format!("{bindir}:{}", std::env::var("PATH").unwrap()),
    );
    std::env::set_var("E2E", dir.dir_str());
    let tools = TestDir::make("streamed-tools");
    let (feeds, guards) = synthetic_feeds(&tools);
    let deadline = Instant::now() + Duration::from_secs(15);
    podman_streamed(feeds, &["exec", "--user", "0:0", "-i", "some-id"], deadline).unwrap();
    drop(guards);
    assert_eq!(fs::read(dir.path("stdin.bin")).unwrap(), emit_synthetic());
    let argv = fs::read(dir.path("argv.bin")).unwrap();
    assert!(argv.starts_with(b"--remote=false\0exec\0--user\0"));
}

#[test]
fn wait_output_reports_status() {
    let c = Command::new("/bin/sh")
        .args(["-c", "echo hello; exit 0"])
        .stdin(Stdio::null())
        .stdout(Stdio::piped())
        .stderr(Stdio::null())
        .spawn()
        .unwrap();
    let out = wait_output(c, Instant::now() + Duration::from_secs(5)).unwrap();
    assert_eq!(out, b"hello\n");
    let c = Command::new("/bin/sh")
        .args(["-c", "exit 3"])
        .stdin(Stdio::null())
        .stdout(Stdio::piped())
        .stderr(Stdio::null())
        .spawn()
        .unwrap();
    assert_eq!(
        wait_output(c, Instant::now() + Duration::from_secs(5)).unwrap_err(),
        "project maintenance command failed"
    );
}

#[test]
fn short_tool_file_ends_copy_with_eof() {
    let dir = TestDir::make("short");
    let p = dir.path("tool");
    fs::write(&p, b"twelve bytes").unwrap();
    let c = CString::new(p).unwrap();
    let fd = unsafe { libc::open(c.as_ptr(), libc::O_RDONLY) };
    assert!(fd >= 0);
    let _guard = FdGuard(fd);
    // Claim more bytes than the file holds, like a shrink race.
    let feeds = vec![(String::from("tool"), fd, 1_000_000u64)];
    let mut out = Vec::new();
    assert_eq!(
        emit_archive(
            &mut |b: &[u8]| {
                out.extend_from_slice(b);
                Ok(())
            },
            &feeds,
        )
        .unwrap_err(),
        "EOF"
    );
    assert_eq!(out.len(), 512 + b"twelve bytes".len());
}
