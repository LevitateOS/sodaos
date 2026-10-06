use std::ffi::CString;
use std::fs::File;
use std::io::Read;
use std::os::unix::fs::FileExt;
use std::os::unix::io::AsRawFd;

use super::directory::{
    fstat_fd, open_and_lock_ssh_directory, openat_file, validate_authorized_keys_stat, SshDir,
};
use super::random_hex;
use crate::errors::{self, Error};
use crate::signal::Ctx;

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
