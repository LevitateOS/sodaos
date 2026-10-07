//! Root-anchored, fd-relative filesystem access for terminal records.
//!
//! Every open goes through dir fds with `O_NOFOLLOW|O_CLOEXEC` (see
//! `crate::sys`); names must be single final components (empty, `"."`,
//! `".."`, `'/'`, NUL rejected). Record files are fail-closed: anything that
//! is not a root-owned regular file with `nlink == 1` and no group/other
//! permission bits is refused with `"unsafe terminal file"`.

use crate::sys;
use std::ffi::CString;
use std::os::unix::io::AsRawFd;

fn cstr(text: &str) -> std::io::Result<CString> {
    CString::new(text)
        .map_err(|_| std::io::Error::new(std::io::ErrorKind::InvalidInput, "name contains NUL"))
}

/// Same single-component rule as `sys` (names here never reach `sys`).
fn check_component(name: &str) -> std::io::Result<()> {
    if name.is_empty() || name == "." || name == ".." || name.contains('/') || name.contains('\0') {
        return Err(std::io::Error::new(
            std::io::ErrorKind::InvalidInput,
            "invalid name component",
        ));
    }
    Ok(())
}

/// The record-file safety predicate: regular file, uid 0, `nlink == 1`,
/// no group/other permission bits.
fn stat_is_safe(st: &rustix::fs::Stat) -> bool {
    if !rustix::fs::FileType::from_raw_mode(st.st_mode).is_file() {
        return false;
    }
    if st.st_uid != 0 {
        return false;
    }
    if st.st_nlink != 1 {
        return false;
    }
    if st.st_mode & 0o077 != 0 {
        return false;
    }
    true
}

/// Open `name` in `dir` (`O_RDONLY` or `O_RDWR` plus `O_NOFOLLOW|O_NONBLOCK`)
/// and enforce [`stat_is_safe`]; check failures are `Err("unsafe terminal
/// file")`. Open failures (missing file, `ELOOP`, permission) keep the OS
/// error text so callers can tell "absent" from "unsafe".
pub fn root_file(dir: &std::fs::File, name: &str, writable: bool) -> Result<std::fs::File, String> {
    let base = if writable {
        sys::OFlags::RDWR
    } else {
        sys::OFlags::RDONLY
    };
    let file = sys::open_at(
        dir,
        name,
        base | sys::OFlags::NONBLOCK,
        rustix::fs::Mode::empty(),
    )
    .map_err(|e| e.to_string())?;
    let st = rustix::fs::fstat(&file).map_err(|_| "unsafe terminal file".to_string())?;
    if !stat_is_safe(&st) {
        return Err("unsafe terminal file".to_string());
    }
    Ok(file)
}

fn read_record_text(file: std::fs::File) -> Result<String, String> {
    use std::io::Read as _;
    let mut buf = Vec::new();
    file.take(4097)
        .read_to_end(&mut buf)
        .map_err(|e| e.to_string())?;
    if buf.len() > 4096 {
        return Err("terminal record size".to_string());
    }
    String::from_utf8(buf).map_err(|_| "terminal record json".to_string())
}

pub(crate) fn read_record_text_at(dir: &std::fs::File, name: &str) -> Result<String, String> {
    read_record_text(root_file(dir, name, false)?)
}

/// `root_file` (read-only) plus bounded read: over 4096 bytes is
/// `Err("terminal record size")`; non-UTF-8 or malformed JSON is
/// `Err("terminal record json")` (fixed strings, never record content).
pub(crate) fn read_record_value(
    file: std::fs::File,
) -> Result<crate::state_json::StateValue, String> {
    let text = read_record_text(file)?;
    crate::state_json::StateValue::parse(&text).map_err(|_| "terminal record json".to_string())
}

pub fn read_record(
    dir: &std::fs::File,
    name: &str,
) -> Result<crate::state_json::StateValue, String> {
    read_record_value(root_file(dir, name, false)?)
}

/// Create `name` in `dir` (`O_WRONLY|O_CREAT|O_EXCL|O_NOFOLLOW`), `fchmod` to
/// exactly `mode` (create mode is umask-masked, hence the explicit chmod),
/// then write all of `data`; a short/failed write is `Err("short terminal
/// write")`. Open/`fchmod` failures (notably `EEXIST` collisions) keep the OS
/// error text.
pub fn new_file(dir: &std::fs::File, name: &str, data: &[u8], mode: u32) -> Result<(), String> {
    use std::io::Write as _;
    let mut file = sys::open_at(
        dir,
        name,
        sys::OFlags::WRONLY | sys::OFlags::CREATE | sys::OFlags::EXCL,
        rustix::fs::Mode::from_bits_retain(mode),
    )
    .map_err(|e| e.to_string())?;
    fchmod(&file, mode).map_err(|e| e.to_string())?;
    file.write_all(data)
        .map_err(|_| "short terminal write".to_string())?;
    Ok(())
}

/// `mkdirat(dir, name, mode)`.
pub fn mkdir_at(dir: &std::fs::File, name: &str, mode: u32) -> std::io::Result<()> {
    check_component(name)?;
    rustix::fs::mkdirat(dir, name, rustix::fs::Mode::from_bits_retain(mode)).map_err(Into::into)
}

/// `fchmod(file, mode)`.
pub fn fchmod(file: &std::fs::File, mode: u32) -> std::io::Result<()> {
    rustix::fs::fchmod(file, rustix::fs::Mode::from_bits_retain(mode)).map_err(Into::into)
}

/// `unlinkat(dir, name, 0)`.
pub fn unlink_at(dir: &std::fs::File, name: &str) -> std::io::Result<()> {
    check_component(name)?;
    rustix::fs::unlinkat(dir, name, rustix::fs::AtFlags::empty()).map_err(Into::into)
}

/// `unlinkat(dir, name, AT_REMOVEDIR)`.
pub fn rmdir_at(dir: &std::fs::File, name: &str) -> std::io::Result<()> {
    check_component(name)?;
    rustix::fs::unlinkat(dir, name, rustix::fs::AtFlags::REMOVEDIR).map_err(Into::into)
}

/// `chown` (`lchown` when `no_follow`) on an absolute or cwd-relative path.
pub fn chown_path(path: &str, uid: u32, gid: u32, no_follow: bool) -> std::io::Result<()> {
    let c = cstr(path)?;
    let result = unsafe {
        if no_follow {
            libc::lchown(c.as_ptr(), uid as libc::uid_t, gid as libc::gid_t)
        } else {
            libc::chown(c.as_ptr(), uid as libc::uid_t, gid as libc::gid_t)
        }
    };
    if result == 0 {
        Ok(())
    } else {
        Err(std::io::Error::last_os_error())
    }
}

/// `chmod` on an absolute or cwd-relative path. Linux cannot chmod a symlink
/// itself (`fchmodat` + `AT_SYMLINK_NOFOLLOW` is unsupported), so `no_follow`
/// refuses symlinks (`InvalidInput`) instead of following them.
pub fn chmod_path(path: &str, mode: u32, no_follow: bool) -> std::io::Result<()> {
    if no_follow {
        let st = rustix::fs::statat(rustix::fs::CWD, path, rustix::fs::AtFlags::SYMLINK_NOFOLLOW)?;
        if rustix::fs::FileType::from_raw_mode(st.st_mode).is_symlink() {
            return Err(std::io::Error::new(
                std::io::ErrorKind::InvalidInput,
                "refusing to chmod symlink",
            ));
        }
    }
    rustix::fs::chmod(path, rustix::fs::Mode::from_bits_retain(mode)).map_err(Into::into)
}

/// `renameat(dir, from, dir, to)`.
pub fn replace_at(dir: &std::fs::File, from: &str, to: &str) -> std::io::Result<()> {
    check_component(from)?;
    check_component(to)?;
    rustix::fs::renameat(dir, from, dir, to).map_err(Into::into)
}

/// `fsync(file)` (retries `EINTR`).
pub fn fsync_file(file: &std::fs::File) -> std::io::Result<()> {
    loop {
        match rustix::fs::fsync(file) {
            Ok(()) => return Ok(()),
            Err(rustix::io::Errno::INTR) => continue,
            Err(err) => return Err(err.into()),
        }
    }
}

/// `(st_uid, st_mode)` of `file` (`st_mode` is the full mode incl. file-type
/// bits; callers mask with `0o7777`/`0o077` as needed).
pub fn fstat_uid_mode(file: &std::fs::File) -> std::io::Result<(u32, u32)> {
    let st = rustix::fs::fstat(file)?;
    Ok((st.st_uid, st.st_mode))
}

/// Full `fstat` of `file` for the broker/keys ownership checks that also need
/// the group, link count, device/inode identity, or nanosecond mtime (the
/// `.py` compares `(st_dev, st_ino, st_mtime_ns)` tuples and exact modes).
pub fn fstat_all(file: &std::fs::File) -> std::io::Result<rustix::fs::Stat> {
    rustix::fs::fstat(file).map_err(Into::into)
}

/// One `read(2)` of up to `limit` bytes (`EINTR` retried, like PEP 475).
/// Mirrors the `.py` single-`os.read` call sites (key file, credential,
/// cgroup events): short reads from pipes are NOT looped here.
pub fn read_up_to(file: &std::fs::File, limit: usize) -> std::io::Result<Vec<u8>> {
    let mut buf = vec![0u8; limit];
    let got = loop {
        let got = unsafe {
            libc::read(
                file.as_raw_fd(),
                buf.as_mut_ptr() as *mut libc::c_void,
                buf.len(),
            )
        };
        if got < 0 {
            let err = std::io::Error::last_os_error();
            if err.raw_os_error() == Some(libc::EINTR) {
                continue;
            }
            return Err(err);
        }
        break got as usize;
    };
    buf.truncate(got);
    Ok(buf)
}

#[cfg(test)]
#[path = "fs_tests.rs"]
mod fs_tests;
