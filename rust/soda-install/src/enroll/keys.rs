//! Bounded authorized-keys import through validated open inodes: existing
//! files are appended, never replaced; new files are published by hard link.

use std::ffi::CString;
use std::fs::File;
use std::io::Read;
use std::os::unix::fs::FileExt;
use std::os::unix::io::{AsRawFd, FromRawFd, OwnedFd};

use crate::errors::{self, Error};
use crate::signal::Ctx;

// Existing files are appended through their validated open inode, never replaced.
// A native editor may still change that inode or its pathname concurrently. Such
// changes cause an uncertain result; Soda never restores a snapshot over them.
// Production passes the protected native root home and UID 0. The UID and narrow
// write seam permit synthetic tests of the exact filesystem race/failure boundary.

/// Single-syscall writer, like Go's `unix.Write`: no looping, so partial
/// writes surface for explicit uncertainty.
fn single_write(file: &File, data: &[u8]) -> std::io::Result<usize> {
    let n = unsafe {
        libc::write(
            file.as_raw_fd(),
            data.as_ptr() as *const libc::c_void,
            data.len(),
        )
    };
    if n < 0 {
        Err(std::io::Error::last_os_error())
    } else {
        Ok(n as usize)
    }
}

pub fn append_enrollment_key(ctx: &Ctx, home: &str, key: &str, uid: u32) -> Result<(), Error> {
    append_enrollment_key_with_writer(ctx, home, key, uid, &single_write)
}

fn errno_error() -> Error {
    errors::os_error(std::io::Error::last_os_error())
}

fn path_cstr(path: &str) -> Result<CString, Error> {
    // Go's string-to-pointer conversion fails closed with EINVAL on NUL.
    CString::new(path)
        .map_err(|_| errors::os_error(std::io::Error::from_raw_os_error(libc::EINVAL)))
}

fn fstat_fd(fd: i32) -> Result<libc::stat, Error> {
    let mut st: libc::stat = unsafe { std::mem::zeroed() };
    if unsafe { libc::fstat(fd, &mut st) } != 0 {
        return Err(errno_error());
    }
    Ok(st)
}

fn open_dir_nofollow(path: &str) -> Result<OwnedFd, Error> {
    let path = path_cstr(path)?;
    let fd = unsafe {
        libc::open(
            path.as_ptr(),
            libc::O_RDONLY | libc::O_DIRECTORY | libc::O_NOFOLLOW | libc::O_CLOEXEC,
            0,
        )
    };
    if fd < 0 {
        return Err(errno_error());
    }
    Ok(unsafe { OwnedFd::from_raw_fd(fd) })
}

fn openat_file(dir_fd: i32, name: &str, flags: i32, mode: u32) -> std::io::Result<File> {
    let name = CString::new(name).map_err(|_| std::io::Error::from_raw_os_error(libc::EINVAL))?;
    let fd = unsafe { libc::openat(dir_fd, name.as_ptr(), flags | libc::O_CLOEXEC, mode) };
    if fd < 0 {
        return Err(std::io::Error::last_os_error());
    }
    Ok(unsafe { File::from_raw_fd(fd) })
}

/// Locked `.ssh` directory: unlocking runs before close, like Go's cleanup.
pub struct SshDir {
    fd: OwnedFd,
}

impl SshDir {
    fn raw(&self) -> i32 {
        self.fd.as_raw_fd()
    }
}

impl Drop for SshDir {
    fn drop(&mut self) {
        unsafe {
            libc::flock(self.fd.as_raw_fd(), libc::LOCK_UN);
        }
    }
}

fn open_and_lock_ssh_directory(home: &str, uid: u32) -> Result<SshDir, Error> {
    let home_fd = open_dir_nofollow(home)?;
    enrollment_safe_directory(home_fd.as_raw_fd(), uid)?;
    let ssh = CString::new(".ssh").map_err(|_| Error::msg("invalid name"))?;
    if unsafe { libc::mkdirat(home_fd.as_raw_fd(), ssh.as_ptr(), 0o700) } != 0 {
        let errno = unsafe { *libc::__errno_location() };
        if errno != libc::EEXIST {
            return Err(errors::os_error(std::io::Error::from_raw_os_error(errno)));
        }
    }
    let dir_fd = unsafe {
        libc::openat(
            home_fd.as_raw_fd(),
            ssh.as_ptr(),
            libc::O_RDONLY | libc::O_DIRECTORY | libc::O_NOFOLLOW | libc::O_CLOEXEC,
            0,
        )
    };
    if dir_fd < 0 {
        return Err(errno_error());
    }
    let dir = SshDir {
        fd: unsafe { OwnedFd::from_raw_fd(dir_fd) },
    };
    enrollment_safe_directory(dir.raw(), uid)?;
    if unsafe { libc::flock(dir.raw(), libc::LOCK_EX | libc::LOCK_NB) } != 0 {
        return Err(Error::msg("authorized keys are busy"));
    }
    Ok(dir)
}

fn validate_authorized_keys_stat(st: &libc::stat, uid: u32) -> Result<(), Error> {
    if st.st_mode & libc::S_IFMT != libc::S_IFREG
        || st.st_uid != uid
        || st.st_nlink != 1
        || st.st_mode & 0o022 != 0
        || st.st_size > 1 << 20
    {
        return Err(Error::msg(
            "existing authorized_keys must be a bounded, singly linked, safely owned regular file",
        ));
    }
    Ok(())
}

fn has_authorized_key(data: &[u8], key: &str) -> bool {
    for line in data.split(|b| *b == b'\n') {
        let text = String::from_utf8_lossy(line);
        let fields: Vec<&str> = text.split_whitespace().collect();
        for i in 0..fields.len().saturating_sub(1) {
            if format!("{} {}", fields[i], fields[i + 1]) == key {
                return true;
            }
        }
    }
    false
}

fn inspect_existing_authorized_keys(
    file: &File,
    key: &str,
    uid: u32,
) -> Result<(libc::stat, Vec<u8>), Error> {
    let before = fstat_fd(file.as_raw_fd())?;
    validate_authorized_keys_stat(&before, uid)?;
    let mut existing = Vec::new();
    let read = file.take((1 << 20) + 1).read_to_end(&mut existing);
    if read.is_err() || existing.len() > 1 << 20 {
        return Err(Error::msg("cannot inspect bounded authorized keys"));
    }
    if has_authorized_key(&existing, key) {
        return Err(Error::msg(
            "this key already exists; verify its existing access policy",
        ));
    }
    Ok((before, existing))
}

fn verify_authorized_keys_unchanged(dir: &SshDir, before: &libc::stat) -> Result<(), Error> {
    let name = CString::new("authorized_keys").map_err(|_| Error::msg("invalid name"))?;
    let mut current: libc::stat = unsafe { std::mem::zeroed() };
    let changed = unsafe {
        libc::fstatat(
            dir.raw(),
            name.as_ptr(),
            &mut current,
            libc::AT_SYMLINK_NOFOLLOW,
        ) != 0
    } || current.st_dev != before.st_dev
        || current.st_ino != before.st_ino
        || current.st_size != before.st_size
        || current.st_mtime != before.st_mtime
        || current.st_mtime_nsec != before.st_mtime_nsec
        || current.st_ctime != before.st_ctime
        || current.st_ctime_nsec != before.st_ctime_nsec;
    if changed {
        return Err(Error::msg(
            "authorized keys changed before the append; no append started",
        ));
    }
    Ok(())
}

fn append_existing_key(
    ctx: &Ctx,
    dir: &SshDir,
    file: &File,
    key: &str,
    uid: u32,
    write: &dyn Fn(&File, &[u8]) -> std::io::Result<usize>,
) -> Result<(), Error> {
    let (before, existing) = inspect_existing_authorized_keys(file, key, uid)?;
    verify_authorized_keys_unchanged(dir, &before)?;
    if let Some(err) = ctx.err() {
        return Err(err);
    }
    let mut addition = Vec::with_capacity(key.len() + 2);
    addition.push(b'\n');
    addition.extend_from_slice(key.as_bytes());
    addition.push(b'\n');
    // Unlike a looping stream writer, one syscall preserves the bounded
    // append boundary and exposes short writes for explicit uncertainty.
    match write(file, &addition) {
        Ok(n) if n == addition.len() => {}
        _ => return Err(Error::EnrollUncertain),
    }
    if file.sync_all().is_err() {
        return Err(Error::EnrollUncertain);
    }
    let mut expected = existing.clone();
    expected.extend_from_slice(&addition);
    enrollment_confirm_key_file(dir, file, uid, &expected)?;
    if ctx.err().is_some() {
        return Err(Error::EnrollUncertain);
    }
    Ok(())
}

fn write_temp_key_file(
    dir: &SshDir,
    temp_name: &str,
    contents: &[u8],
    write: &dyn Fn(&File, &[u8]) -> std::io::Result<usize>,
) -> Result<File, Error> {
    let file = openat_file(
        dir.raw(),
        temp_name,
        libc::O_RDWR | libc::O_CREAT | libc::O_EXCL | libc::O_NOFOLLOW | libc::O_CLOEXEC,
        0o600,
    )
    .map_err(errors::os_error)?;
    match write(&file, contents) {
        Ok(n) if n == contents.len() => {}
        _ => {
            return Err(Error::msg(
                "new authorized-key file could not be written; no publication attempted",
            ))
        }
    }
    if let Err(err) = file.sync_all() {
        return Err(errors::path_error("sync", temp_name, err));
    }
    Ok(file)
}

fn link_and_confirm_key_file(
    ctx: &Ctx,
    dir: &SshDir,
    temp_name: &str,
    file: &File,
    uid: u32,
    contents: &[u8],
) -> Result<(), Error> {
    let temp = CString::new(temp_name).map_err(|_| Error::msg("invalid name"))?;
    let target = CString::new("authorized_keys").map_err(|_| Error::msg("invalid name"))?;
    if unsafe { libc::linkat(dir.raw(), temp.as_ptr(), dir.raw(), target.as_ptr(), 0) } != 0 {
        return Err(Error::msg(
            "authorized-key publication refused; any concurrently created file was preserved",
        ));
    }
    if unsafe { libc::unlinkat(dir.raw(), temp.as_ptr(), 0) } != 0 {
        return Err(Error::EnrollUncertain);
    }
    if unsafe { libc::fsync(dir.raw()) } != 0 {
        return Err(Error::EnrollUncertain);
    }
    enrollment_confirm_key_file(dir, file, uid, contents)?;
    if ctx.err().is_some() {
        return Err(Error::EnrollUncertain);
    }
    Ok(())
}

/// Temporary key file with Go's deferred cleanup: unlink before close.
struct TempKey<'a> {
    dir: &'a SshDir,
    name: CString,
    file: Option<File>,
}

impl Drop for TempKey<'_> {
    fn drop(&mut self) {
        unsafe {
            libc::unlinkat(self.dir.raw(), self.name.as_ptr(), 0);
        }
        drop(self.file.take());
    }
}

pub fn random_hex(bytes: usize) -> Result<String, Error> {
    let mut random = vec![0u8; bytes];
    let mut filled = 0;
    while filled < bytes {
        let n = unsafe {
            libc::getrandom(
                random[filled..].as_mut_ptr() as *mut libc::c_void,
                (bytes - filled) as libc::size_t,
                0,
            )
        };
        if n < 0 {
            let errno = unsafe { *libc::__errno_location() };
            if errno == libc::EINTR {
                continue;
            }
            return Err(errors::os_error(std::io::Error::from_raw_os_error(errno)));
        }
        filled += n as usize;
    }
    Ok(crate::buildx::hex_encode(&random))
}

fn create_exclusive_key_file(
    ctx: &Ctx,
    dir: &SshDir,
    key: &str,
    uid: u32,
    write: &dyn Fn(&File, &[u8]) -> std::io::Result<usize>,
) -> Result<(), Error> {
    let temp_name = format!(".soda-enrollment-{}", random_hex(12)?);
    let contents = format!("{key}\n");
    let file = write_temp_key_file(dir, &temp_name, contents.as_bytes(), write)?;
    let temp = TempKey {
        dir,
        name: CString::new(temp_name.clone()).map_err(|_| Error::msg("invalid name"))?,
        file: Some(file),
    };
    if let Some(err) = ctx.err() {
        return Err(err);
    }
    link_and_confirm_key_file(
        ctx,
        dir,
        &temp_name,
        temp.file.as_ref().unwrap(),
        uid,
        contents.as_bytes(),
    )
}

pub fn append_enrollment_key_with_writer(
    ctx: &Ctx,
    home: &str,
    key: &str,
    uid: u32,
    write: &dyn Fn(&File, &[u8]) -> std::io::Result<usize>,
) -> Result<(), Error> {
    if let Some(err) = ctx.err() {
        return Err(err);
    }
    match crate::sshkey::public_key(key) {
        Ok(normalized) if normalized == key => {}
        _ => return Err(Error::msg("normalized public key required")),
    }
    let dir = open_and_lock_ssh_directory(home, uid)?;
    match openat_file(
        dir.raw(),
        "authorized_keys",
        libc::O_RDWR | libc::O_APPEND | libc::O_NOFOLLOW | libc::O_NONBLOCK | libc::O_CLOEXEC,
        0,
    ) {
        Ok(file) => append_existing_key(ctx, &dir, &file, key, uid, write),
        Err(err) if err.raw_os_error() == Some(libc::ENOENT) => {
            create_exclusive_key_file(ctx, &dir, key, uid, write)
        }
        Err(err) => Err(errors::os_error(err)),
    }
}

fn confirm_key_file_contents(file: &File, expected: &[u8]) -> Result<(), Error> {
    let mut actual = vec![0u8; expected.len() + 1];
    let mut n = 0;
    let eof = loop {
        match file.read_at(&mut actual[n..], n as u64) {
            Ok(0) => break true,
            Ok(m) => {
                n += m;
                if n == actual.len() {
                    break false;
                }
            }
            Err(err) if err.kind() == std::io::ErrorKind::Interrupted => {}
            Err(_) => break false,
        }
    };
    if !eof || n != expected.len() || actual[..n] != expected[..] {
        return Err(Error::EnrollUncertain);
    }
    Ok(())
}

fn validate_key_file_match(
    opened: &libc::stat,
    named: &libc::stat,
    uid: u32,
    expected_len: usize,
) -> Result<(), Error> {
    if opened.st_dev != named.st_dev || opened.st_ino != named.st_ino || named.st_nlink != 1 {
        return Err(Error::EnrollUncertain);
    }
    if named.st_mode & libc::S_IFMT != libc::S_IFREG
        || named.st_uid != uid
        || named.st_mode & 0o022 != 0
        || named.st_size != expected_len as i64
    {
        return Err(Error::EnrollUncertain);
    }
    Ok(())
}

fn confirm_key_file_stats(
    dir: &SshDir,
    file: &File,
    uid: u32,
    expected_len: usize,
) -> Result<(), Error> {
    let opened = fstat_fd(file.as_raw_fd()).map_err(|_| Error::EnrollUncertain)?;
    let name = CString::new("authorized_keys").map_err(|_| Error::EnrollUncertain)?;
    let mut named: libc::stat = unsafe { std::mem::zeroed() };
    if unsafe {
        libc::fstatat(
            dir.raw(),
            name.as_ptr(),
            &mut named,
            libc::AT_SYMLINK_NOFOLLOW,
        )
    } != 0
    {
        return Err(Error::EnrollUncertain);
    }
    validate_key_file_match(&opened, &named, uid, expected_len)
}

// Confirmation is observational, not a lock against future native edits. If a
// native editor replaced the pathname while we appended, its new file remains
// untouched; the append may instead have reached the old, even unlinked, inode.
fn enrollment_confirm_key_file(
    dir: &SshDir,
    file: &File,
    uid: u32,
    expected: &[u8],
) -> Result<(), Error> {
    confirm_key_file_contents(file, expected)?;
    confirm_key_file_stats(dir, file, uid, expected.len())
}

pub fn enrollment_safe_directory(fd: i32, uid: u32) -> Result<(), Error> {
    let st = fstat_fd(fd)?;
    if st.st_mode & libc::S_IFMT != libc::S_IFDIR || st.st_uid != uid || st.st_mode & 0o022 != 0 {
        return Err(Error::msg("real safely owned directory required"));
    }
    Ok(())
}

// CoreOS uses a native /root symlink. Resolve that native root-home alias once,
// then require real root-owned non-writable ancestors; never follow .ssh links.
pub fn enrollment_root_home() -> Result<String, Error> {
    let home = crate::pathx::eval_symlinks("/root")?;
    let mut current = home.clone();
    loop {
        let fd = open_dir_nofollow(&current)?;
        let result = enrollment_safe_directory(fd.as_raw_fd(), 0);
        drop(fd);
        result?;
        if current == "/" {
            break;
        }
        current = crate::pathx::dir(&current);
    }
    Ok(home)
}

#[cfg(test)]
mod tests {
    use super::super::tests::{temp_dir, test_uid, TEST_KEY, TEST_KEY_2, TEST_KEY_3};
    use super::*;
    use std::io::Write as _;
    use std::os::unix::fs::PermissionsExt;

    fn make_ssh_dir(home: &str) -> String {
        let dir = format!("{home}/.ssh");
        std::fs::create_dir(&dir).unwrap();
        std::fs::set_permissions(&dir, std::fs::Permissions::from_mode(0o700)).unwrap();
        dir
    }

    fn test_ctx() -> Ctx {
        Ctx::test().0
    }

    #[test]
    fn preserves_authorized_keys() {
        let home = temp_dir();
        std::fs::set_permissions(&home.path, std::fs::Permissions::from_mode(0o700)).unwrap();
        let existing = format!("# existing native policy\nrestrict {TEST_KEY} original comment");
        let dir = make_ssh_dir(&home.path);
        let path = format!("{dir}/authorized_keys");
        std::fs::write(&path, &existing).unwrap();
        std::fs::set_permissions(&path, std::fs::Permissions::from_mode(0o600)).unwrap();
        append_enrollment_key(&test_ctx(), &home.path, TEST_KEY_2, test_uid()).unwrap();
        let got = std::fs::read(&path).unwrap();
        let mut want = existing.as_bytes().to_vec();
        want.push(b'\n');
        want.extend_from_slice(format!("{TEST_KEY_2}\n").as_bytes());
        assert_eq!(
            got, want,
            "existing authorized keys were not preserved exactly"
        );
        let mode = std::fs::symlink_metadata(&path)
            .unwrap()
            .permissions()
            .mode()
            & 0o777;
        assert_eq!(mode, 0o600, "native file permissions changed unsafely");
        // The restricted existing key matches as an adjacent pair: refused.
        assert!(append_enrollment_key(&test_ctx(), &home.path, TEST_KEY, test_uid()).is_err());
        // The just-imported key is a duplicate: refused.
        assert!(append_enrollment_key(&test_ctx(), &home.path, TEST_KEY_2, test_uid()).is_err());
        assert_eq!(
            std::fs::read(&path).unwrap(),
            got,
            "refusal changed existing keys"
        );
        assert_eq!(
            std::fs::read_dir(&dir).unwrap().count(),
            1,
            "temporary key files remained"
        );
    }

    #[test]
    fn creates_authorized_keys() {
        let home = temp_dir();
        append_enrollment_key(&test_ctx(), &home.path, TEST_KEY, test_uid()).unwrap();
        for (name, mode) in [(".ssh", 0o700), (".ssh/authorized_keys", 0o600)] {
            let st = std::fs::symlink_metadata(format!("{}/{name}", home.path)).unwrap();
            assert_eq!(
                st.permissions().mode() & 0o777,
                mode,
                "wrong mode for {name}"
            );
            assert_eq!(std::os::unix::fs::MetadataExt::uid(&st), test_uid());
        }
    }

    #[test]
    fn refuses_unsafe_key_paths() {
        for scenario in [
            "home-symlink",
            "ssh-symlink",
            "key-symlink",
            "key-hardlink",
            "key-directory",
            "key-writable",
            "ssh-writable",
            "home-writable",
            "oversized",
            "wrong-owner",
        ] {
            let root = temp_dir();
            let mut home = format!("{}/home", root.path);
            std::fs::create_dir(&home).unwrap();
            std::fs::set_permissions(&home, std::fs::Permissions::from_mode(0o700)).unwrap();
            let dir = format!("{home}/.ssh");
            std::fs::create_dir(&dir).unwrap();
            let path = format!("{dir}/authorized_keys");
            let sentinel = format!("{}/sentinel", root.path);
            let original = b"preserve this unrelated file\n";
            std::fs::write(&sentinel, original).unwrap();
            let mut uid = test_uid();
            match scenario {
                "home-symlink" => {
                    let alias = format!("{}/alias", root.path);
                    std::os::unix::fs::symlink(&home, &alias).unwrap();
                    home = alias;
                }
                "ssh-symlink" => {
                    std::fs::remove_dir(&dir).unwrap();
                    std::os::unix::fs::symlink(&root.path, &dir).unwrap();
                }
                "key-symlink" => {
                    std::os::unix::fs::symlink(&sentinel, &path).unwrap();
                }
                "key-hardlink" => {
                    std::fs::hard_link(&sentinel, &path).unwrap();
                }
                "key-directory" => {
                    std::fs::create_dir(&path).unwrap();
                }
                "key-writable" => {
                    std::fs::write(&path, original).unwrap();
                    std::fs::set_permissions(&path, std::fs::Permissions::from_mode(0o666))
                        .unwrap();
                }
                "ssh-writable" => {
                    std::fs::set_permissions(&dir, std::fs::Permissions::from_mode(0o777)).unwrap();
                }
                "home-writable" => {
                    std::fs::set_permissions(&home, std::fs::Permissions::from_mode(0o777))
                        .unwrap();
                }
                "oversized" => {
                    std::fs::write(&path, vec![b'x'; (1 << 20) + 1]).unwrap();
                }
                "wrong-owner" => {
                    uid += 1;
                }
                _ => unreachable!(),
            }
            assert!(
                append_enrollment_key(&test_ctx(), &home, TEST_KEY, uid).is_err(),
                "unsafe key target accepted: {scenario}"
            );
            assert_eq!(
                std::fs::read(&sentinel).unwrap(),
                original,
                "unrelated file changed: {scenario}"
            );
        }
    }

    #[test]
    fn cancelled_append_preserves_key() {
        let home = temp_dir();
        let (ctx, flag) = Ctx::test();
        flag.store(true, std::sync::atomic::Ordering::SeqCst);
        assert!(append_enrollment_key(&ctx, &home.path, TEST_KEY, test_uid()).is_err());
        assert!(matches!(
            std::fs::symlink_metadata(format!("{}/.ssh", home.path)),
            Err(err) if err.kind() == std::io::ErrorKind::NotFound
        ));
    }

    #[test]
    fn concurrent_writer_preserves_key() {
        let home = temp_dir();
        append_enrollment_key(&test_ctx(), &home.path, TEST_KEY, test_uid()).unwrap();
        let dir_fd = open_dir_nofollow(&format!("{}/.ssh", home.path)).unwrap();
        assert_eq!(
            unsafe { libc::flock(dir_fd.as_raw_fd(), libc::LOCK_EX | libc::LOCK_NB) },
            0
        );
        assert!(append_enrollment_key(&test_ctx(), &home.path, TEST_KEY_2, test_uid()).is_err());
        assert_eq!(
            std::fs::read(format!("{}/.ssh/authorized_keys", home.path)).unwrap(),
            format!("{TEST_KEY}\n").as_bytes()
        );
    }

    #[test]
    fn native_editor_race_preserves_newer_file() {
        for replacement in [true, false] {
            let home = temp_dir();
            let dir = make_ssh_dir(&home.path);
            let path = format!("{dir}/authorized_keys");
            let original = format!("{TEST_KEY} original\n");
            let newer = format!("{TEST_KEY_2} native editor without trailing newline");
            std::fs::write(&path, &original).unwrap();
            let calls = std::cell::Cell::new(0);
            let write = |mut file: &File, data: &[u8]| -> std::io::Result<usize> {
                calls.set(calls.get() + 1);
                // This is the former Fstatat-to-Renameat race window: a native
                // editor acts after Soda's last pre-write pathname check.
                let target = if replacement {
                    format!("{dir}/native-editor-new-file")
                } else {
                    path.clone()
                };
                std::fs::write(&target, &newer).unwrap();
                if replacement {
                    std::fs::rename(&target, &path).unwrap();
                }
                file.write(data)
            };
            let err = append_enrollment_key_with_writer(
                &test_ctx(),
                &home.path,
                TEST_KEY_3,
                test_uid(),
                &write,
            )
            .unwrap_err();
            assert_eq!(
                err,
                Error::EnrollUncertain,
                "race not reported as uncertain: {err}"
            );
            assert_eq!(calls.get(), 1);
            let mut want = newer.clone();
            if !replacement {
                want.push_str(&format!("\n{TEST_KEY_3}\n"));
            }
            assert_eq!(
                std::fs::read(&path).unwrap(),
                want.as_bytes(),
                "Soda overwrote or merged into the native editor's newer data"
            );
        }
    }

    #[test]
    fn concurrent_creation_is_never_replaced() {
        let home = temp_dir();
        let path = format!("{}/.ssh/authorized_keys", home.path);
        let native = format!("{TEST_KEY} created by native editor\n");
        let write = |mut file: &File, data: &[u8]| -> std::io::Result<usize> {
            std::fs::write(&path, &native).unwrap();
            file.write(data)
        };
        assert!(append_enrollment_key_with_writer(
            &test_ctx(),
            &home.path,
            TEST_KEY_2,
            test_uid(),
            &write
        )
        .is_err());
        assert_eq!(std::fs::read(&path).unwrap(), native.as_bytes());
        assert_eq!(
            std::fs::read_dir(format!("{}/.ssh", home.path))
                .unwrap()
                .count(),
            1,
            "unpublished temporary key file remained"
        );
    }

    #[test]
    fn partial_append_does_not_roll_back_native_data() {
        let home = temp_dir();
        let dir = make_ssh_dir(&home.path);
        let path = format!("{dir}/authorized_keys");
        let original = format!("{TEST_KEY} preserved original\n");
        std::fs::write(&path, &original).unwrap();
        std::fs::set_permissions(&path, std::fs::Permissions::from_mode(0o640)).unwrap();
        let before = std::fs::symlink_metadata(&path).unwrap();
        let partial_len = std::cell::Cell::new(0usize);
        let write = |mut file: &File, data: &[u8]| -> std::io::Result<usize> {
            let half = &data[..data.len() / 2];
            partial_len.set(half.len());
            let n = file.write(half)?;
            assert_eq!(n, half.len());
            Err(std::io::Error::other("synthetic partial-write failure"))
        };
        let err = append_enrollment_key_with_writer(
            &test_ctx(),
            &home.path,
            TEST_KEY_2,
            test_uid(),
            &write,
        )
        .unwrap_err();
        assert_eq!(
            err,
            Error::EnrollUncertain,
            "partial append not uncertain: {err}"
        );
        let mut want = original.as_bytes().to_vec();
        let addition = format!("\n{TEST_KEY_2}\n");
        want.extend_from_slice(&addition.as_bytes()[..partial_len.get()]);
        assert_eq!(std::fs::read(&path).unwrap(), want);
        let after = std::fs::symlink_metadata(&path).unwrap();
        use std::os::unix::fs::MetadataExt;
        assert_eq!(
            (before.dev(), before.ino()),
            (after.dev(), after.ino()),
            "append replaced the native inode"
        );
        assert_eq!(before.mode(), after.mode(), "append changed the file mode");
    }
}
