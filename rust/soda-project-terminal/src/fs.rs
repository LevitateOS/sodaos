//! Minimal dir-fd file helpers (TEMPORARY: the sibling's canonical
//! `fs.rs` replaces this file at merge — other modules use only the
//! signatures listed in the PR25 brief).
//!
//! `root_chain` is a pure no-follow walk: ownership/mode checks live in the
//! callers (via [`fstat_uid_mode`]), matching `project_terminal.py`'s
//! per-level `root_directory` checks regardless of which `fs.rs` lands.

use std::fs::File;
use std::io;
use std::os::unix::io::AsRawFd;

use crate::sys;

fn s_isreg(mode: u32) -> bool {
    mode & libc::S_IFMT == libc::S_IFREG
}

/// Walk `top` (`/a/b/...`) from `/` through no-follow directory fds.
pub fn root_chain(top: &str) -> io::Result<File> {
    let mut current = sys::open_root()?;
    for part in top.split('/').filter(|p| !p.is_empty()) {
        current = sys::open_child_dir(&current, part)?;
    }
    Ok(current)
}

/// Pure `root_file` stat rule: regular, uid 0, one link, `mode & 0o077 == 0`.
pub fn record_stat_ok(uid: u32, nlink: u64, mode: u32, is_regular: bool) -> bool {
    is_regular && uid == 0 && nlink == 1 && mode & 0o077 == 0
}

/// Open a root-owned record file (read-only, or read-write for `writer`).
pub fn root_file(dir: &File, name: &str, writable: bool) -> Result<File, String> {
    let flags = (if writable {
        libc::O_RDWR
    } else {
        libc::O_RDONLY
    }) | libc::O_NOFOLLOW
        | libc::O_NONBLOCK;
    let file = sys::open_at(dir, name, flags, 0).map_err(|e| format!("open record: {e}"))?;
    let mut info: libc::stat = unsafe { std::mem::zeroed() };
    if unsafe { libc::fstat(file.as_raw_fd(), &mut info) } != 0 {
        return Err(format!("stat record: {}", io::Error::last_os_error()));
    }
    if !record_stat_ok(
        info.st_uid,
        info.st_nlink as u64,
        info.st_mode,
        s_isreg(info.st_mode),
    ) {
        return Err("unsafe terminal file".to_string());
    }
    Ok(file)
}

/// Size cap plus JSON parse over record bytes (pure half of `read_record`).
pub fn parse_record_bytes(raw: &[u8]) -> Result<soda_json::JsonValue, String> {
    if raw.len() > 4096 {
        return Err("terminal record size".to_string());
    }
    let text = std::str::from_utf8(raw).map_err(|_| "terminal record encoding".to_string())?;
    soda_json::JsonValue::parse(text).map_err(|_| "terminal record json".to_string())
}

/// Read one record (single 4097-byte read; `>4096` errors like the `.py`).
pub fn read_record(dir: &File, name: &str) -> Result<soda_json::JsonValue, String> {
    let file = root_file(dir, name, false)?;
    let mut raw = vec![0u8; 4097];
    let got = loop {
        let got = unsafe {
            libc::read(
                file.as_raw_fd(),
                raw.as_mut_ptr() as *mut libc::c_void,
                raw.len(),
            )
        };
        if got < 0 && io::Error::last_os_error().raw_os_error() == Some(libc::EINTR) {
            continue;
        }
        if got < 0 {
            return Err(format!("read record: {}", io::Error::last_os_error()));
        }
        break got as usize;
    };
    raw.truncate(got);
    parse_record_bytes(&raw)
}

/// Exclusive-create a record file with an exact mode and full body.
pub fn new_file(dir: &File, name: &str, data: &[u8], mode: u32) -> Result<(), String> {
    let file = sys::open_at(
        dir,
        name,
        libc::O_WRONLY | libc::O_CREAT | libc::O_EXCL | libc::O_NOFOLLOW,
        mode as libc::mode_t,
    )
    .map_err(|e| format!("create record: {e}"))?;
    fchmod(&file, mode).map_err(|e| format!("chmod record: {e}"))?;
    use std::io::Write;
    (&file)
        .write_all(data)
        .map_err(|e| format!("write record: {e}"))?;
    Ok(())
}

fn cstring(value: &str) -> io::Result<std::ffi::CString> {
    std::ffi::CString::new(value)
        .map_err(|_| io::Error::new(io::ErrorKind::InvalidInput, "nul byte in path"))
}

pub fn mkdir_at(dir: &File, name: &str, mode: u32) -> io::Result<()> {
    let path = cstring(name)?;
    if unsafe { libc::mkdirat(dir.as_raw_fd(), path.as_ptr(), mode as libc::mode_t) } != 0 {
        return Err(io::Error::last_os_error());
    }
    Ok(())
}

pub fn fchmod(file: &File, mode: u32) -> io::Result<()> {
    if unsafe { libc::fchmod(file.as_raw_fd(), mode as libc::mode_t) } != 0 {
        return Err(io::Error::last_os_error());
    }
    Ok(())
}

pub fn unlink_at(dir: &File, name: &str) -> io::Result<()> {
    let path = cstring(name)?;
    if unsafe { libc::unlinkat(dir.as_raw_fd(), path.as_ptr(), 0) } != 0 {
        return Err(io::Error::last_os_error());
    }
    Ok(())
}

pub fn rmdir_at(dir: &File, name: &str) -> io::Result<()> {
    let path = cstring(name)?;
    if unsafe { libc::unlinkat(dir.as_raw_fd(), path.as_ptr(), libc::AT_REMOVEDIR) } != 0 {
        return Err(io::Error::last_os_error());
    }
    Ok(())
}

pub fn chown_path(path: &str, uid: u32, gid: u32, no_follow: bool) -> io::Result<()> {
    let target = cstring(path)?;
    let rc = if no_follow {
        unsafe { libc::lchown(target.as_ptr(), uid, gid) }
    } else {
        unsafe { libc::chown(target.as_ptr(), uid, gid) }
    };
    if rc != 0 {
        return Err(io::Error::last_os_error());
    }
    Ok(())
}

pub fn chmod_path(path: &str, mode: u32, no_follow: bool) -> io::Result<()> {
    let target = cstring(path)?;
    let rc = if no_follow {
        unsafe {
            libc::fchmodat(
                libc::AT_FDCWD,
                target.as_ptr(),
                mode as libc::mode_t,
                libc::AT_SYMLINK_NOFOLLOW,
            )
        }
    } else {
        unsafe { libc::chmod(target.as_ptr(), mode as libc::mode_t) }
    };
    if rc != 0 {
        return Err(io::Error::last_os_error());
    }
    Ok(())
}

pub fn replace_at(dir: &File, src: &str, dst: &str) -> io::Result<()> {
    let from = cstring(src)?;
    let to = cstring(dst)?;
    if unsafe { libc::renameat(dir.as_raw_fd(), from.as_ptr(), dir.as_raw_fd(), to.as_ptr()) } != 0
    {
        return Err(io::Error::last_os_error());
    }
    Ok(())
}

pub fn fsync_file(file: &File) -> io::Result<()> {
    file.sync_all()
}

/// `(st_uid, st_mode)` for caller-side ownership checks.
pub fn fstat_uid_mode(file: &File) -> io::Result<(u32, u32)> {
    let mut info: libc::stat = unsafe { std::mem::zeroed() };
    if unsafe { libc::fstat(file.as_raw_fd(), &mut info) } != 0 {
        return Err(io::Error::last_os_error());
    }
    Ok((info.st_uid, info.st_mode))
}

#[cfg(test)]
mod tests {
    use super::*;

    fn scratch(name: &str) -> std::path::PathBuf {
        let dir = std::env::temp_dir().join(format!(
            "soda-pt-fs-{}-{}-{}",
            std::process::id(),
            name,
            sys::monotonic().to_bits()
        ));
        std::fs::create_dir_all(&dir).unwrap();
        dir
    }

    #[test]
    fn chain_walks() {
        let dir = scratch("chain");
        std::fs::create_dir_all(dir.join("a").join("b")).unwrap();
        let abs = dir.join("a").to_string_lossy().into_owned();
        let got = root_chain(&abs).unwrap();
        assert!(fstat_uid_mode(&got).is_ok());
        assert!(root_chain("/definitely/not/here-9f3c").is_err());
        let slash = root_chain("/").unwrap();
        fsync_file(&slash).unwrap();
        std::fs::remove_dir_all(&dir).unwrap();
    }

    #[test]
    fn record_stat_matrix() {
        assert!(record_stat_ok(0, 1, 0o100600, true));
        assert!(!record_stat_ok(0, 1, 0o100644, true)); // group/other-readable
        assert!(!record_stat_ok(0, 1, 0o100640, true)); // group-readable
        assert!(!record_stat_ok(0, 1, 0o100601, true)); // other bit
        assert!(!record_stat_ok(1000, 1, 0o100600, true)); // owner
        assert!(!record_stat_ok(0, 2, 0o100600, true)); // links
        assert!(!record_stat_ok(0, 1, 0o100600, false)); // not regular
    }

    #[test]
    fn root_file_refuses_unsafe() {
        let dir = scratch("rootfile");
        let fd = root_chain(&dir.to_string_lossy()).unwrap();
        // Missing file.
        assert!(root_file(&fd, "absent", false).is_err());
        // Present but owned by the (non-root) test user → refused.
        std::fs::write(dir.join("mine"), b"{}").unwrap();
        assert!(root_file(&fd, "mine", false).is_err());
        // Symlink refused via O_NOFOLLOW.
        std::os::unix::fs::symlink("mine", dir.join("link")).unwrap();
        assert!(root_file(&fd, "link", false).is_err());
        // Directory refused (not regular).
        std::fs::create_dir(dir.join("sub")).unwrap();
        assert!(root_file(&fd, "sub", false).is_err());
        std::fs::remove_dir_all(&dir).unwrap();
    }

    #[test]
    fn record_bytes_matrix() {
        assert!(parse_record_bytes(br#"{"a":1}"#).is_ok());
        assert!(parse_record_bytes(&vec![b'x'; 4097]).is_err());
        assert!(parse_record_bytes(&vec![b' '; 4096]).is_err()); // size ok, json bad
        assert!(parse_record_bytes(b"\xff\xfe").is_err());
        // Duplicate keys tolerated like json.loads (last wins at lookup).
        let value = parse_record_bytes(br#"{"a":1,"a":2}"#).unwrap();
        assert_eq!(
            value.get("a"),
            Some(&soda_json::JsonValue::Number("2".to_string()))
        );
    }

    #[test]
    fn lifecycle_roundtrip() {
        let dir = scratch("life");
        let fd = root_chain(&dir.to_string_lossy()).unwrap();
        new_file(&fd, "rec", b"{}", 0o600).unwrap();
        assert!(new_file(&fd, "rec", b"{}", 0o600).is_err()); // O_EXCL
        mkdir_at(&fd, "sub", 0o700).unwrap();
        // Mode is exact even under a permissive umask.
        let mode =
            std::os::unix::fs::MetadataExt::mode(&std::fs::metadata(dir.join("rec")).unwrap());
        assert_eq!(mode & 0o777, 0o600);
        fchmod(&fd, 0o700).unwrap();
        fsync_file(&fd).unwrap();
        new_file(&fd, "next", b"1", 0o600).unwrap();
        replace_at(&fd, "next", "rec").unwrap();
        assert_eq!(std::fs::read(dir.join("rec")).unwrap(), b"1");
        unlink_at(&fd, "rec").unwrap();
        rmdir_at(&fd, "sub").unwrap();
        assert!(unlink_at(&fd, "rec").is_err());
        chmod_path(&dir.to_string_lossy(), 0o700, false).unwrap();
        chown_path(
            &dir.to_string_lossy(),
            unsafe { libc::getuid() },
            unsafe { libc::getgid() },
            true,
        )
        .unwrap();
        std::fs::remove_dir_all(&dir).unwrap();
    }
}
