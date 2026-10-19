//! Native artifact support ported from `internal/release/build`: OCI
//! architecture gating, digest/revision shapes, confined JSON reads, and
//! exclusive file creation. Only the installer-reachable surface is ported.

use std::io::Read;
use std::os::unix::fs::OpenOptionsExt;

use crate::errors::{self, Error};

pub const JSON_LIMIT: u64 = 4 << 20;

pub fn digest(s: &str) -> bool {
    s.len() == 64 && s.bytes().all(|b| matches!(b, b'0'..=b'9' | b'a'..=b'f'))
}

pub fn revision(s: &str) -> bool {
    s.len() == 40 && s.bytes().all(|b| matches!(b, b'0'..=b'9' | b'a'..=b'f'))
}

fn file_is_regular(meta: &std::fs::Metadata) -> bool {
    meta.file_type().is_file()
}

pub fn hash_file(path: &str) -> Result<String, Error> {
    let st = std::fs::symlink_metadata(path).map_err(|e| errors::path_error("lstat", path, e))?;
    if !file_is_regular(&st) {
        return Err(Error::msg("regular non-symlink file required"));
    }
    let file = std::fs::File::open(path).map_err(|e| errors::path_error("open", path, e))?;
    let mut hasher = sha2::Sha256::new();
    use sha2::Digest as _;
    let mut reader = file;
    let mut chunk = [0u8; 65536];
    loop {
        let n = reader
            .read(&mut chunk)
            .map_err(|e| errors::path_error("read", path, e))?;
        if n == 0 {
            break;
        }
        hasher.update(&chunk[..n]);
    }
    Ok(hex_encode(&hasher.finalize()))
}

pub fn hex_encode(bytes: &[u8]) -> String {
    const HEX: &[u8; 16] = b"0123456789abcdef";
    let mut out = String::with_capacity(bytes.len() * 2);
    for &b in bytes {
        out.push(HEX[(b >> 4) as usize] as char);
        out.push(HEX[(b & 0xf) as usize] as char);
    }
    out
}

pub fn sha256_hex(data: &[u8]) -> String {
    use sha2::Digest as _;
    hex_encode(&sha2::Sha256::digest(data))
}

/// `WriteNew`: exclusive create, single write, close; both errors joined.
pub fn write_new(path: &str, data: &[u8], mode: u32) -> Result<(), Error> {
    use std::io::Write;
    let mut file = std::fs::OpenOptions::new()
        .write(true)
        .create_new(true)
        .mode(mode)
        .open(path)
        .map_err(|e| errors::path_error("open", path, e))?;
    let write_err = file
        .write_all(data)
        .err()
        .map(|e| errors::path_error("write", path, e));
    // Capture the close error like Go's `errors.Join(write, close)`: closing
    // through libc surfaces what a dropped `File` would swallow.
    use std::os::unix::io::IntoRawFd;
    let close_err = match unsafe { libc::close(file.into_raw_fd()) } {
        0 => None,
        _ => {
            let errno = unsafe { *libc::__errno_location() };
            Some(errors::path_error(
                "close",
                path,
                std::io::Error::from_raw_os_error(errno),
            ))
        }
    };
    match (write_err, close_err) {
        (None, None) => Ok(()),
        (Some(err), None) | (None, Some(err)) => Err(err),
        (Some(first), Some(second)) => Err(Error::msg(format!("{first}\n{second}"))),
    }
}

fn open_dir_nofollow(path: &str) -> Result<std::fs::File, Error> {
    std::fs::OpenOptions::new()
        .read(true)
        .custom_flags(libc::O_DIRECTORY | libc::O_NOFOLLOW)
        .open(path)
        .map_err(|e| errors::path_error("open", path, e))
}

fn open_at_file(dir_fd: i32, name: &str, flags: i32, mode: u32) -> Result<std::fs::File, Error> {
    let name = CString::new(name).map_err(|_| Error::msg("invalid name"))?;
    let fd = unsafe { libc::openat(dir_fd, name.as_ptr(), flags | libc::O_CLOEXEC, mode) };
    if fd < 0 {
        let errno = unsafe { *libc::__errno_location() };
        return Err(errors::path_error(
            "open",
            &name.to_string_lossy(),
            std::io::Error::from_raw_os_error(errno),
        ));
    }
    use std::os::unix::io::FromRawFd;
    Ok(unsafe { std::fs::File::from_raw_fd(fd) })
}

use std::ffi::CString;

fn fstatat_nofollow(dir_fd: i32, name: &str) -> Result<libc::stat, Error> {
    let name = CString::new(name).map_err(|_| Error::msg("invalid name"))?;
    let mut st: libc::stat = unsafe { std::mem::zeroed() };
    if unsafe { libc::fstatat(dir_fd, name.as_ptr(), &mut st, libc::AT_SYMLINK_NOFOLLOW) } != 0 {
        let errno = unsafe { *libc::__errno_location() };
        return Err(errors::path_error(
            "lstat",
            &name.to_string_lossy(),
            std::io::Error::from_raw_os_error(errno),
        ));
    }
    Ok(st)
}

fn same_file(a: &libc::stat, b: &libc::stat) -> bool {
    a.st_dev == b.st_dev && a.st_ino == b.st_ino
}

/// `ReadJSON` minus decoding: confined, bounded, same-inode bytes. Decoding
/// (always strict for these callers) lives with each shape.
pub fn read_json_bytes(path: &str) -> Result<Vec<u8>, Error> {
    let (dir, base) = match path.rfind('/') {
        Some(0) => ("/".to_string(), path[1..].to_string()),
        Some(i) => (path[..i].to_string(), path[i + 1..].to_string()),
        None => (".".to_string(), path.to_string()),
    };
    if base.is_empty() || base == "." || base == ".." || base.contains('/') {
        return Err(errors::path_error(
            "lstat",
            &base,
            std::io::Error::from_raw_os_error(libc::EINVAL),
        ));
    }
    let root = open_dir_nofollow(&dir)?;
    use std::os::unix::io::AsRawFd;
    let dir_fd = root.as_raw_fd();
    let st = fstatat_nofollow(dir_fd, &base)?;
    if st.st_mode & libc::S_IFMT != libc::S_IFREG || st.st_size as u64 > JSON_LIMIT {
        return Err(Error::msg("bounded regular JSON input required"));
    }
    // No O_NOFOLLOW: like Go's `root.OpenFile`, the open follows and the
    // `SameFile` check below refuses anything that changed, symlinks
    // included, with Go's exact message.
    let file = open_at_file(dir_fd, &base, libc::O_RDONLY | libc::O_NONBLOCK, 0)?;
    let mut actual: libc::stat = unsafe { std::mem::zeroed() };
    if unsafe { libc::fstat(file.as_raw_fd(), &mut actual) } != 0 {
        let errno = unsafe { *libc::__errno_location() };
        return Err(errors::path_error(
            "stat",
            &base,
            std::io::Error::from_raw_os_error(errno),
        ));
    }
    if actual.st_mode & libc::S_IFMT != libc::S_IFREG || !same_file(&st, &actual) {
        return Err(Error::msg("JSON input changed before reading"));
    }
    let mut data = Vec::new();
    (&file)
        .take(JSON_LIMIT + 1)
        .read_to_end(&mut data)
        .map_err(|e| errors::path_error("read", &base, e))?;
    if data.len() as u64 > JSON_LIMIT {
        return Err(Error::msg("JSON input exceeds limit"));
    }
    Ok(data)
}
