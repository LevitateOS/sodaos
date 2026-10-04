//! Root-anchored, fd-relative filesystem access for terminal records.
//!
//! Every open goes through dir fds with `O_NOFOLLOW|O_CLOEXEC` (see
//! `crate::sys`); names must be single final components (empty, `"."`,
//! `".."`, `'/'`, NUL rejected). Record files are fail-closed: anything that
//! is not a root-owned regular file with `nlink == 1` and no group/other
//! permission bits is refused with `"unsafe terminal file"`.

use std::ffi::CString;
use std::os::unix::io::AsRawFd;

use crate::sys;

fn cstr(text: &str) -> std::io::Result<CString> {
    CString::new(text)
        .map_err(|_| std::io::Error::new(std::io::ErrorKind::InvalidInput, "name contains NUL"))
}

/// Same single-component rule as `sys` (names here never reach `sys`).
fn check_component(name: &str) -> std::io::Result<()> {
    if name.is_empty() || name == "." || name == ".." || name.contains('/') {
        return Err(std::io::Error::new(
            std::io::ErrorKind::InvalidInput,
            "invalid name component",
        ));
    }
    Ok(())
}

fn cvt(rc: libc::c_int) -> std::io::Result<()> {
    if rc != 0 {
        return Err(std::io::Error::last_os_error());
    }
    Ok(())
}

/// Open `/`, then walk each `/`-separated component of `top` (empty segments
/// skipped, so leading/trailing/doubled slashes are fine) via
/// `open_child_dir`. The caller applies uid/mode checks to the result.
///
/// Retained for future locator callers (covered by `root_chain_matrix`);
/// current call sites need per-level checks, which live in
/// `term::checked_chain` and the keys directory walk.
#[allow(dead_code)]
pub fn root_chain(top: &str) -> std::io::Result<std::fs::File> {
    let mut dir = sys::open_root()?;
    for comp in top.split('/') {
        if comp.is_empty() {
            continue;
        }
        dir = sys::open_child_dir(&dir, comp)?;
    }
    Ok(dir)
}

fn s_isreg(mode: u32) -> bool {
    mode & libc::S_IFMT == libc::S_IFREG
}

#[cfg(test)]
fn s_isdir(mode: u32) -> bool {
    mode & libc::S_IFMT == libc::S_IFDIR
}

fn s_islnk(mode: u32) -> bool {
    mode & libc::S_IFMT == libc::S_IFLNK
}

/// The record-file safety predicate: regular file, uid 0, `nlink == 1`,
/// no group/other permission bits.
fn stat_is_safe(st: &libc::stat) -> bool {
    if !s_isreg(st.st_mode) {
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
        libc::O_RDWR
    } else {
        libc::O_RDONLY
    };
    let file = sys::open_at(dir, name, base | libc::O_NONBLOCK, 0).map_err(|e| e.to_string())?;
    let mut st: libc::stat = unsafe { std::mem::zeroed() };
    let rc = unsafe { libc::fstat(file.as_raw_fd(), &mut st) };
    if rc != 0 || !stat_is_safe(&st) {
        return Err("unsafe terminal file".to_string());
    }
    Ok(file)
}

fn read_record_inner(file: std::fs::File) -> Result<soda_json::JsonValue, String> {
    use std::io::Read as _;
    let mut buf = Vec::new();
    file.take(4097)
        .read_to_end(&mut buf)
        .map_err(|e| e.to_string())?;
    if buf.len() > 4096 {
        return Err("terminal record size".to_string());
    }
    let text = std::str::from_utf8(&buf).map_err(|_| "terminal record json".to_string())?;
    soda_json::JsonValue::parse(text).map_err(|_| "terminal record json".to_string())
}

/// `root_file` (read-only) plus bounded read: over 4096 bytes is
/// `Err("terminal record size")`; non-UTF-8 or malformed JSON is
/// `Err("terminal record json")` (fixed strings, never record content).
pub fn read_record(dir: &std::fs::File, name: &str) -> Result<soda_json::JsonValue, String> {
    let file = root_file(dir, name, false)?;
    read_record_inner(file)
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
        libc::O_WRONLY | libc::O_CREAT | libc::O_EXCL,
        mode as libc::mode_t,
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
    let c = cstr(name)?;
    cvt(unsafe { libc::mkdirat(dir.as_raw_fd(), c.as_ptr(), mode as libc::mode_t) })
}

/// `fchmod(file, mode)`.
pub fn fchmod(file: &std::fs::File, mode: u32) -> std::io::Result<()> {
    cvt(unsafe { libc::fchmod(file.as_raw_fd(), mode as libc::mode_t) })
}

/// `unlinkat(dir, name, 0)`.
pub fn unlink_at(dir: &std::fs::File, name: &str) -> std::io::Result<()> {
    check_component(name)?;
    let c = cstr(name)?;
    cvt(unsafe { libc::unlinkat(dir.as_raw_fd(), c.as_ptr(), 0) })
}

/// `unlinkat(dir, name, AT_REMOVEDIR)`.
pub fn rmdir_at(dir: &std::fs::File, name: &str) -> std::io::Result<()> {
    check_component(name)?;
    let c = cstr(name)?;
    cvt(unsafe { libc::unlinkat(dir.as_raw_fd(), c.as_ptr(), libc::AT_REMOVEDIR) })
}

/// `chown` (`lchown` when `no_follow`) on an absolute or cwd-relative path.
pub fn chown_path(path: &str, uid: u32, gid: u32, no_follow: bool) -> std::io::Result<()> {
    let c = cstr(path)?;
    cvt(unsafe {
        if no_follow {
            libc::lchown(c.as_ptr(), uid as libc::uid_t, gid as libc::gid_t)
        } else {
            libc::chown(c.as_ptr(), uid as libc::uid_t, gid as libc::gid_t)
        }
    })
}

/// `chmod` on an absolute or cwd-relative path. Linux cannot chmod a symlink
/// itself (`fchmodat` + `AT_SYMLINK_NOFOLLOW` is unsupported), so `no_follow`
/// refuses symlinks (`InvalidInput`) instead of following them.
pub fn chmod_path(path: &str, mode: u32, no_follow: bool) -> std::io::Result<()> {
    let c = cstr(path)?;
    if no_follow {
        let mut st: libc::stat = unsafe { std::mem::zeroed() };
        cvt(unsafe { libc::lstat(c.as_ptr(), &mut st) })?;
        if s_islnk(st.st_mode) {
            return Err(std::io::Error::new(
                std::io::ErrorKind::InvalidInput,
                "refusing to chmod symlink",
            ));
        }
    }
    cvt(unsafe { libc::chmod(c.as_ptr(), mode as libc::mode_t) })
}

/// `renameat(dir, from, dir, to)`.
pub fn replace_at(dir: &std::fs::File, from: &str, to: &str) -> std::io::Result<()> {
    check_component(from)?;
    check_component(to)?;
    let f = cstr(from)?;
    let t = cstr(to)?;
    cvt(unsafe { libc::renameat(dir.as_raw_fd(), f.as_ptr(), dir.as_raw_fd(), t.as_ptr()) })
}

/// `fsync(file)` (retries `EINTR`).
pub fn fsync_file(file: &std::fs::File) -> std::io::Result<()> {
    loop {
        let rc = unsafe { libc::fsync(file.as_raw_fd()) };
        if rc == 0 {
            return Ok(());
        }
        let err = std::io::Error::last_os_error();
        if err.raw_os_error() == Some(libc::EINTR) {
            continue;
        }
        return Err(err);
    }
}

/// `(st_uid, st_mode)` of `file` (`st_mode` is the full mode incl. file-type
/// bits; callers mask with `0o7777`/`0o077` as needed).
pub fn fstat_uid_mode(file: &std::fs::File) -> std::io::Result<(u32, u32)> {
    let mut st: libc::stat = unsafe { std::mem::zeroed() };
    cvt(unsafe { libc::fstat(file.as_raw_fd(), &mut st) })?;
    Ok((st.st_uid as u32, st.st_mode as u32))
}

/// Full `fstat` of `file` for the broker/keys ownership checks that also need
/// the group, link count, device/inode identity, or nanosecond mtime (the
/// `.py` compares `(st_dev, st_ino, st_mtime_ns)` tuples and exact modes).
pub fn fstat_all(file: &std::fs::File) -> std::io::Result<libc::stat> {
    let mut st: libc::stat = unsafe { std::mem::zeroed() };
    cvt(unsafe { libc::fstat(file.as_raw_fd(), &mut st) })?;
    Ok(st)
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
mod tests {
    use super::*;

    static TEST_SEQ: std::sync::atomic::AtomicU64 = std::sync::atomic::AtomicU64::new(0);

    /// Unique per-test scratch dir (tests run in parallel threads).
    fn test_dir(tag: &str) -> std::path::PathBuf {
        let n = TEST_SEQ.fetch_add(1, std::sync::atomic::Ordering::Relaxed);
        let dir = std::env::temp_dir().join(format!("soda-pt-fs-{tag}-{}-{n}", std::process::id()));
        std::fs::create_dir_all(&dir).expect("test dir");
        dir
    }

    fn dir_fd(path: &std::path::Path) -> std::fs::File {
        std::fs::File::open(path).unwrap()
    }

    fn am_root() -> bool {
        unsafe { libc::getuid() == 0 }
    }

    #[test]
    fn root_chain_matrix() {
        // "/" and "" both yield the root dir itself.
        for top in ["/", "", "usr", "/usr", "usr/", "/usr/bin/", "usr//bin"] {
            let dir = root_chain(top).unwrap_or_else(|e| panic!("root_chain({top:?}): {e}"));
            let (uid, mode) = fstat_uid_mode(&dir).unwrap();
            assert_eq!(uid, 0);
            assert!(s_isdir(mode));
        }
        assert!(root_chain("no-such-dir-soda-pt").is_err());
        // Traversal components fail closed.
        for top in ["..", "usr/../etc", ".", "usr/./bin"] {
            assert!(root_chain(top).is_err(), "root_chain({top:?})");
        }
    }

    fn fake_stat(mode: u32, uid: u32, nlink: u64) -> libc::stat {
        let mut st: libc::stat = unsafe { std::mem::zeroed() };
        st.st_mode = mode;
        st.st_uid = uid;
        st.st_nlink = nlink as libc::nlink_t;
        st
    }

    #[test]
    fn safety_predicate_matrix() {
        assert!(stat_is_safe(&fake_stat(libc::S_IFREG | 0o600, 0, 1)));
        assert!(stat_is_safe(&fake_stat(libc::S_IFREG | 0o400, 0, 1)));
        // Every single violation flips the predicate.
        assert!(!stat_is_safe(&fake_stat(libc::S_IFDIR | 0o700, 0, 1))); // not regular
        assert!(!stat_is_safe(&fake_stat(libc::S_IFLNK | 0o777, 0, 1))); // symlink
        assert!(!stat_is_safe(&fake_stat(libc::S_IFIFO | 0o600, 0, 1))); // fifo
        assert!(!stat_is_safe(&fake_stat(libc::S_IFREG | 0o600, 1000, 1))); // uid
        assert!(!stat_is_safe(&fake_stat(libc::S_IFREG | 0o600, 0, 2))); // nlink
        assert!(!stat_is_safe(&fake_stat(libc::S_IFREG | 0o640, 0, 1))); // group bit
        assert!(!stat_is_safe(&fake_stat(libc::S_IFREG | 0o601, 0, 1))); // other bit
        assert!(!stat_is_safe(&fake_stat(libc::S_IFREG | 0o644, 0, 1))); // group+other
        assert!(!stat_is_safe(&fake_stat(libc::S_IFREG | 0o607, 0, 1)));
        assert!(!stat_is_safe(&fake_stat(libc::S_IFREG | 0o600, 0, 0))); // nlink 0
    }

    #[test]
    fn root_file_failures() {
        let dir = test_dir("rootfile");
        std::fs::write(dir.join("world"), b"{}").unwrap();
        let world = std::fs::File::open(dir.join("world")).unwrap();
        fchmod(&world, 0o644).unwrap();
        drop(world);
        std::fs::hard_link(dir.join("world"), dir.join("linked")).unwrap();
        std::fs::create_dir(dir.join("subdir")).unwrap();
        std::os::unix::fs::symlink("world", dir.join("link")).unwrap();
        let fd = dir_fd(&dir);
        // Mode, nlink, and non-regular all fail closed with the fixed string.
        assert_eq!(
            root_file(&fd, "world", false).unwrap_err(),
            "unsafe terminal file"
        );
        assert_eq!(
            root_file(&fd, "linked", false).unwrap_err(),
            "unsafe terminal file"
        );
        assert_eq!(
            root_file(&fd, "subdir", false).unwrap_err(),
            "unsafe terminal file"
        );
        // Symlink and missing keep OS error text (ELOOP / ENOENT).
        let err = root_file(&fd, "link", false).unwrap_err();
        assert!(err.contains("symbolic link"), "{err}");
        let err = root_file(&fd, "absent", false).unwrap_err();
        assert!(err.contains("No such file"), "{err}");
        for bad in ["", ".", "..", "a/b"] {
            assert!(root_file(&fd, bad, false).is_err(), "name {bad:?}");
        }
        if am_root() {
            std::fs::write(dir.join("good"), b"{}").unwrap();
            let f = std::fs::File::open(dir.join("good")).unwrap();
            fchmod(&f, 0o600).unwrap();
            // nlink must be 1: "world" was linked but "good" was not.
            assert!(root_file(&fd, "good", false).is_ok());
            assert!(root_file(&fd, "good", true).is_ok());
        }
        let _ = std::fs::remove_dir_all(&dir);
    }

    #[test]
    fn record_size_and_json_limits() {
        let dir = test_dir("record");
        // Exactly 4096 bytes of valid JSON: quoted string of 4094 chars.
        let exact = format!("\"{}\"", "a".repeat(4094));
        assert_eq!(exact.len(), 4096);
        std::fs::write(dir.join("exact"), &exact).unwrap();
        let value = read_record_inner(std::fs::File::open(dir.join("exact")).unwrap()).unwrap();
        assert_eq!(value, soda_json::JsonValue::Str("a".repeat(4094)));
        // 4097 bytes trips the limit even when the JSON is valid.
        let over = format!("\"{}\"", "b".repeat(4095));
        std::fs::write(dir.join("over"), &over).unwrap();
        assert_eq!(
            read_record_inner(std::fs::File::open(dir.join("over")).unwrap()).unwrap_err(),
            "terminal record size"
        );
        // Malformed, empty, and non-UTF8 bodies share the fixed JSON string.
        std::fs::write(dir.join("bad"), b"{oops").unwrap();
        assert_eq!(
            read_record_inner(std::fs::File::open(dir.join("bad")).unwrap()).unwrap_err(),
            "terminal record json"
        );
        std::fs::write(dir.join("empty"), b"").unwrap();
        assert_eq!(
            read_record_inner(std::fs::File::open(dir.join("empty")).unwrap()).unwrap_err(),
            "terminal record json"
        );
        std::fs::write(dir.join("bin"), [0xff, 0xfe, 0x00]).unwrap();
        assert_eq!(
            read_record_inner(std::fs::File::open(dir.join("bin")).unwrap()).unwrap_err(),
            "terminal record json"
        );
        let _ = std::fs::remove_dir_all(&dir);
    }

    #[test]
    fn read_record_public_paths() {
        let dir = test_dir("readrec");
        std::fs::write(dir.join("world"), b"{}").unwrap();
        let world = std::fs::File::open(dir.join("world")).unwrap();
        fchmod(&world, 0o644).unwrap();
        let fd = dir_fd(&dir);
        // Unsafe file surfaces the safety string through read_record.
        assert_eq!(
            read_record(&fd, "world").unwrap_err(),
            "unsafe terminal file"
        );
        assert!(read_record(&fd, "absent").is_err());
        let _ = std::fs::remove_dir_all(&dir);
    }

    #[test]
    fn new_file_create_and_collision() {
        let dir = test_dir("newfile");
        let fd = dir_fd(&dir);
        new_file(&fd, "rec", b"{\"a\":1}", 0o600).unwrap();
        assert_eq!(std::fs::read(dir.join("rec")).unwrap(), b"{\"a\":1}");
        let (_, mode) = fstat_uid_mode(&std::fs::File::open(dir.join("rec")).unwrap()).unwrap();
        assert_eq!(mode & 0o7777, 0o600);
        // 0o660 is NOT producible through a 022 umask at create time, so an
        // exact 0o660 proves the explicit fchmod ran.
        new_file(&fd, "rec2", b"x", 0o660).unwrap();
        let (_, mode) = fstat_uid_mode(&std::fs::File::open(dir.join("rec2")).unwrap()).unwrap();
        assert_eq!(mode & 0o7777, 0o660);
        // O_EXCL collision keeps EEXIST text.
        let err = new_file(&fd, "rec", b"z", 0o600).unwrap_err();
        assert!(err.contains("File exists"), "{err}");
        assert_eq!(std::fs::read(dir.join("rec")).unwrap(), b"{\"a\":1}");
        for bad in ["", ".", "..", "a/b"] {
            assert!(new_file(&fd, bad, b"x", 0o600).is_err(), "name {bad:?}");
        }
        let _ = std::fs::remove_dir_all(&dir);
    }

    #[test]
    fn wrapper_matrix() {
        let dir = test_dir("wrappers");
        let fd = dir_fd(&dir);
        mkdir_at(&fd, "sub", 0o755).unwrap();
        assert!(dir.join("sub").is_dir());
        assert!(mkdir_at(&fd, "sub", 0o755).is_err()); // EEXIST
        assert!(mkdir_at(&fd, "../esc", 0o755).is_err());
        assert!(mkdir_at(&fd, "", 0o755).is_err());

        std::fs::write(dir.join("f"), b"data").unwrap();
        let f = std::fs::File::open(dir.join("f")).unwrap();
        fchmod(&f, 0o640).unwrap();
        let (uid, mode) = fstat_uid_mode(&f).unwrap();
        assert_eq!(uid, unsafe { libc::getuid() });
        assert!(s_isreg(mode));
        assert_eq!(mode & 0o7777, 0o640);
        fsync_file(&f).unwrap();

        replace_at(&fd, "f", "g").unwrap();
        assert!(!dir.join("f").exists());
        assert_eq!(std::fs::read(dir.join("g")).unwrap(), b"data");
        assert!(replace_at(&fd, "f", "h").is_err()); // missing source
        assert!(replace_at(&fd, "../x", "h").is_err());

        unlink_at(&fd, "g").unwrap();
        assert!(!dir.join("g").exists());
        assert!(unlink_at(&fd, "g").is_err()); // already gone
        assert!(unlink_at(&fd, ".").is_err());

        rmdir_at(&fd, "sub").unwrap();
        assert!(!dir.join("sub").exists());
        mkdir_at(&fd, "full", 0o755).unwrap();
        std::fs::write(dir.join("full").join("x"), b"").unwrap();
        assert!(rmdir_at(&fd, "full").is_err()); // ENOTEMPTY
        let _ = std::fs::remove_dir_all(&dir);
    }

    #[test]
    fn path_chown_chmod_matrix() {
        let dir = test_dir("pathops");
        let path = dir.join("obj");
        std::fs::write(&path, b"z").unwrap();
        let path_str = path.to_str().unwrap().to_string();
        let uid = unsafe { libc::getuid() };
        let gid = unsafe { libc::getgid() };
        // Chown to self always succeeds.
        chown_path(&path_str, uid, gid, false).unwrap();
        chown_path(&path_str, uid, gid, true).unwrap();
        assert!(chown_path(dir.join("absent").to_str().unwrap(), uid, gid, false).is_err());

        chmod_path(&path_str, 0o640, false).unwrap();
        chmod_path(&path_str, 0o600, true).unwrap();
        let (_, mode) = fstat_uid_mode(&std::fs::File::open(&path).unwrap()).unwrap();
        assert_eq!(mode & 0o7777, 0o600);

        // Symlink discipline: follow works, no_follow refuses.
        let link = dir.join("link");
        std::os::unix::fs::symlink(&path, &link).unwrap();
        let link_str = link.to_str().unwrap().to_string();
        chmod_path(&link_str, 0o640, false).unwrap();
        let err = chmod_path(&link_str, 0o640, true).unwrap_err();
        assert_eq!(err.kind(), std::io::ErrorKind::InvalidInput);
        chown_path(&link_str, uid, gid, true).unwrap(); // lchown on the link
        let _ = std::fs::remove_dir_all(&dir);
    }

    #[test]
    fn stat_all_and_single_read() {
        let dir = test_dir("statread");
        std::fs::write(dir.join("f"), b"hello").unwrap();
        let f = std::fs::File::open(dir.join("f")).unwrap();
        let st = fstat_all(&f).unwrap();
        assert_eq!(st.st_uid, unsafe { libc::getuid() });
        assert_eq!(st.st_gid, unsafe { libc::getgid() });
        assert_eq!(st.st_nlink, 1);
        assert!(s_isreg(st.st_mode));
        assert_eq!(st.st_size, 5);
        // Single bounded read: exact bytes, EOF on the second call.
        assert_eq!(read_up_to(&f, 65537).unwrap(), b"hello");
        assert!(read_up_to(&f, 65537).unwrap().is_empty());
        assert_eq!(read_up_to(&f, 0).unwrap(), b"");
        let _ = std::fs::remove_dir_all(&dir);
    }
}
