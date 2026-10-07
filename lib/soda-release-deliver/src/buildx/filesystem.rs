use std::ffi::CString;
use std::os::fd::{AsRawFd, FromRawFd, OwnedFd};

use crate::payload::Payload;
use sha2::{Digest as _, Sha256};

use crate::Error;

fn os_error(op: &str, path: &str, err: std::io::Error) -> Error {
    Error::msg(format!("{op} {path}: {err}"))
}

/// Directory file descriptor confining resolutions the way `os.Root` does.
pub struct Root {
    fd: OwnedFd,
}

impl Root {
    pub fn open(path: &str) -> Result<Root, Error> {
        let fd = open_dir_fd(path).map_err(|e| os_error("open", path, e))?;
        Ok(Root { fd })
    }

    fn fd(&self) -> i32 {
        self.fd.as_raw_fd()
    }
}

fn open_dir_fd(path: &str) -> std::io::Result<OwnedFd> {
    let cpath = CString::new(path)
        .map_err(|_| std::io::Error::new(std::io::ErrorKind::InvalidInput, "invalid path"))?;
    // SAFETY: open(2) with a NUL-terminated path; result checked.
    let fd = unsafe {
        libc::open(
            cpath.as_ptr(),
            libc::O_RDONLY | libc::O_DIRECTORY | libc::O_CLOEXEC,
        )
    };
    if fd < 0 {
        return Err(std::io::Error::last_os_error());
    }
    // SAFETY: fd is a fresh owned descriptor.
    Ok(unsafe { OwnedFd::from_raw_fd(fd) })
}

fn check_confined(name: &str) -> Result<(), Error> {
    if name.is_empty() || name.starts_with('/') {
        return Err(Error::refused());
    }
    let mut depth = 0i32;
    for segment in name.split('/') {
        match segment {
            "" | "." => {}
            ".." => {
                depth -= 1;
                if depth < 0 {
                    return Err(Error::refused());
                }
            }
            _ => depth += 1,
        }
    }
    Ok(())
}

fn c_string(name: &str) -> Result<CString, Error> {
    CString::new(name).map_err(|_| Error::refused())
}

fn fstatat_no_follow(dirfd: i32, name: &CString) -> std::io::Result<libc::stat> {
    let mut st: libc::stat = unsafe { std::mem::zeroed() };
    // SAFETY: fstatat on a live dirfd with a valid path and output struct.
    let rc = unsafe { libc::fstatat(dirfd, name.as_ptr(), &mut st, libc::AT_SYMLINK_NOFOLLOW) };
    if rc != 0 {
        return Err(std::io::Error::last_os_error());
    }
    Ok(st)
}

fn fstat_fd(fd: i32) -> std::io::Result<libc::stat> {
    let mut st: libc::stat = unsafe { std::mem::zeroed() };
    // SAFETY: fstat on a live descriptor with an output struct.
    if unsafe { libc::fstat(fd, &mut st) } != 0 {
        return Err(std::io::Error::last_os_error());
    }
    Ok(st)
}

fn is_regular(mode: u32) -> bool {
    (mode & libc::S_IFMT) == libc::S_IFREG
}

/// Open a regular file confined to an open directory after a no-follow stat.
fn open_confined_regular(root: &Root, name: &str) -> Result<(std::fs::File, libc::stat), Error> {
    check_confined(name)?;
    let cname = c_string(name)?;
    let st = fstatat_no_follow(root.fd(), &cname)
        .map_err(|e| Error::msg(format!("lstat {name}: {e}")))?;
    if !is_regular(st.st_mode) {
        return Err(Error::msg("regular non-symlink file required"));
    }
    // SAFETY: openat on a live dirfd; flags mirror Go's O_RDONLY|O_NONBLOCK.
    let fd = unsafe {
        libc::openat(
            root.fd(),
            cname.as_ptr(),
            libc::O_RDONLY | libc::O_NONBLOCK | libc::O_CLOEXEC,
        )
    };
    if fd < 0 {
        return Err(Error::msg(format!(
            "open {name}: {}",
            std::io::Error::last_os_error()
        )));
    }
    // SAFETY: fd is a fresh owned descriptor.
    let owned = unsafe { OwnedFd::from_raw_fd(fd) };
    let actual =
        fstat_fd(owned.as_raw_fd()).map_err(|e| Error::msg(format!("stat {name}: {e}")))?;
    if !is_regular(actual.st_mode) || actual.st_dev != st.st_dev || actual.st_ino != st.st_ino {
        return Err(Error::msg("file changed before hashing"));
    }
    Ok((std::fs::File::from(owned), st))
}

/// Read one layout entry confined to an open directory, with the OCI
/// layout loader's error semantics.
pub fn read_layout_entry(root: &Root, name: &str) -> Result<(Vec<u8>, i64), Error> {
    check_confined(name)?;
    let cname = c_string(name)?;
    let st = fstatat_no_follow(root.fd(), &cname)
        .map_err(|e| Error::msg(format!("lstat {name}: {e}")))?;
    if !is_regular(st.st_mode) {
        return Err(Error::msg("non-regular OCI layout entry"));
    }
    // SAFETY: openat on a live dirfd.
    let fd = unsafe {
        libc::openat(
            root.fd(),
            cname.as_ptr(),
            libc::O_RDONLY | libc::O_NONBLOCK | libc::O_CLOEXEC,
        )
    };
    if fd < 0 {
        return Err(Error::msg(format!(
            "open {name}: {}",
            std::io::Error::last_os_error()
        )));
    }
    // SAFETY: fd is a fresh owned descriptor.
    let owned = unsafe { OwnedFd::from_raw_fd(fd) };
    let actual =
        fstat_fd(owned.as_raw_fd()).map_err(|e| Error::msg(format!("stat {name}: {e}")))?;
    if actual.st_dev != st.st_dev || actual.st_ino != st.st_ino {
        return Err(Error::msg("OCI layout entry changed"));
    }
    let mut file = std::fs::File::from(owned);
    let mut data = Vec::new();
    use std::io::Read;
    file.read_to_end(&mut data)
        .map_err(|e| Error::msg(format!("read {name}: {e}")))?;
    Ok((data, st.st_size))
}

/// `readAt`: bounded regular read confined to an open directory.
pub fn read_at(root: &Root, name: &str, maximum: i64) -> Result<Vec<u8>, Error> {
    check_confined(name)?;
    let cname = c_string(name)?;
    let st = fstatat_no_follow(root.fd(), &cname)
        .map_err(|e| Error::msg(format!("lstat {name}: {e}")))?;
    if !is_regular(st.st_mode) || st.st_size > maximum {
        return Err(Error::refused());
    }
    // SAFETY: openat on a live dirfd; flags mirror Go's O_RDONLY|O_NONBLOCK.
    let fd = unsafe {
        libc::openat(
            root.fd(),
            cname.as_ptr(),
            libc::O_RDONLY | libc::O_NONBLOCK | libc::O_CLOEXEC,
        )
    };
    if fd < 0 {
        return Err(Error::msg(format!(
            "open {name}: {}",
            std::io::Error::last_os_error()
        )));
    }
    // SAFETY: fd is a fresh owned descriptor.
    let owned = unsafe { OwnedFd::from_raw_fd(fd) };
    let actual =
        fstat_fd(owned.as_raw_fd()).map_err(|e| Error::msg(format!("stat {name}: {e}")))?;
    if actual.st_dev != st.st_dev || actual.st_ino != st.st_ino {
        return Err(Error::refused());
    }
    let mut file = std::fs::File::from(owned);
    let mut data = Vec::new();
    use std::io::Read;
    file.by_ref()
        .take((maximum + 1) as u64)
        .read_to_end(&mut data)
        .map_err(|e| Error::msg(format!("read {name}: {e}")))?;
    if data.len() as i64 > maximum {
        return Err(Error::refused());
    }
    Ok(data)
}

/// `build.HashAt`: hash a regular file confined to an open directory.
pub fn hash_at(root: &Root, name: &str) -> Result<String, Error> {
    let (mut file, _) = open_confined_regular(root, name)?;
    let mut hasher = Sha256::new();
    std::io::copy(&mut file, &mut HashWriter(&mut hasher))
        .map_err(|e| Error::msg(format!("read {name}: {e}")))?;
    Ok(format!("{:x}", hasher.finalize()))
}

struct HashWriter<'a>(&'a mut Sha256);

impl std::io::Write for HashWriter<'_> {
    fn write(&mut self, buf: &[u8]) -> std::io::Result<usize> {
        self.0.update(buf);
        Ok(buf.len())
    }
    fn flush(&mut self) -> std::io::Result<()> {
        Ok(())
    }
}

fn clean_path(path: &str) -> String {
    // Lexical clean mirroring filepath.Clean for absolute paths.
    let mut parts: Vec<&str> = Vec::new();
    for segment in path.split('/') {
        match segment {
            "" | "." => {}
            ".." => {
                parts.pop();
            }
            _ => parts.push(segment),
        }
    }
    format!("/{}", parts.join("/"))
}

/// `build.FreshDirectory`: create a new private directory under a real parent.
pub fn fresh_directory(path: &str) -> Result<(), Error> {
    if !path.starts_with('/') {
        return Err(Error::msg("absolute new directory required"));
    }
    let parent = match path.rfind('/') {
        Some(0) | None => "/".to_string(),
        Some(i) => path[..i].to_string(),
    };
    let resolved = std::fs::canonicalize(&parent)
        .map_err(|e| Error::msg(format!("readlink {parent}: {e}")))?;
    if resolved.to_string_lossy() != clean_path(&parent) {
        return Err(Error::msg("symlinked parent refused"));
    }
    use std::os::unix::fs::DirBuilderExt;
    std::fs::DirBuilder::new()
        .mode(0o700)
        .create(path)
        .map_err(|e| os_error("mkdir", path, e))
}

/// `build.PrivateDestination`: admit a not-yet-existing private output path.
pub fn private_destination(path: &str) -> Result<(), Error> {
    if !path.starts_with('/') {
        return Err(Error::msg("absolute private output required"));
    }
    let parent = match path.rfind('/') {
        Some(0) | None => "/".to_string(),
        Some(i) => path[..i].to_string(),
    };
    let resolved = std::fs::canonicalize(&parent)
        .map_err(|e| Error::msg(format!("readlink {parent}: {e}")))?;
    let st = std::fs::metadata(&parent).map_err(|e| os_error("stat", &parent, e))?;
    use std::os::unix::fs::MetadataExt;
    if resolved.to_string_lossy() != parent || st.mode() & 0o077 != 0 {
        return Err(Error::msg("real private output parent required"));
    }
    match std::fs::symlink_metadata(path) {
        Err(e) if e.kind() == std::io::ErrorKind::NotFound => Ok(()),
        _ => Err(Error::msg("output already exists or cannot be inspected")),
    }
}

/// `build.WriteNew`: create a file that must not already exist.
pub fn write_new(path: &str, data: &[u8], mode: u32) -> Result<(), Error> {
    use std::os::unix::fs::OpenOptionsExt;
    let mut file = std::fs::OpenOptions::new()
        .write(true)
        .create_new(true)
        .mode(mode)
        .open(path)
        .map_err(|e| os_error("open", path, e))?;
    use std::io::Write;
    let write_err = file.write_all(data).err();
    let close_err = file.sync_all().err();
    match (write_err, close_err) {
        (None, None) => Ok(()),
        (Some(e), None) => Err(Error::msg(format!("write {path}: {e}"))),
        (None, Some(e)) => Err(Error::msg(format!("sync {path}: {e}"))),
        (Some(e1), Some(e2)) => Err(Error::msg(format!("write {path}: {e1}; sync: {e2}"))),
    }
}

/// `build.ReadJSONAt` decode step: bounded strict-shape JSON with Go's
/// unknown-field error text. Duplicate keys keep Go's last-wins rule here
/// (unlike `strictjson`), matching `encoding/json` exactly.
pub fn decode_build_json(data: &[u8]) -> Result<Payload, Error> {
    if data.len() > 4 << 20 {
        return Err(Error::msg("JSON input exceeds limit"));
    }
    std::str::from_utf8(data).map_err(|_| Error::msg("invalid JSON input"))?;
    serde_json::from_slice(data).map_err(|e| {
        let message = e.to_string();
        if let Some(rest) = message.strip_prefix("unknown field `") {
            if let Some((name, _)) = rest.split_once('`') {
                return Error::msg(format!("json: unknown field \"{name}\""));
            }
        }
        Error::msg("invalid JSON input")
    })
}

/// Read and decode bounded JSON confined to an open directory, returning
/// the digest of the exact bytes decoded.
pub fn read_json_at(root: &Root, name: &str) -> Result<(Payload, String), Error> {
    check_confined(name)?;
    let cname = c_string(name)?;
    let st = fstatat_no_follow(root.fd(), &cname)
        .map_err(|e| Error::msg(format!("lstat {name}: {e}")))?;
    if !is_regular(st.st_mode) || st.st_size > 4 << 20 {
        return Err(Error::msg("bounded regular JSON input required"));
    }
    // SAFETY: openat on a live dirfd; flags mirror Go's O_RDONLY|O_NONBLOCK.
    let fd = unsafe {
        libc::openat(
            root.fd(),
            cname.as_ptr(),
            libc::O_RDONLY | libc::O_NONBLOCK | libc::O_CLOEXEC,
        )
    };
    if fd < 0 {
        return Err(Error::msg(format!(
            "open {name}: {}",
            std::io::Error::last_os_error()
        )));
    }
    // SAFETY: fd is a fresh owned descriptor.
    let owned = unsafe { OwnedFd::from_raw_fd(fd) };
    let actual =
        fstat_fd(owned.as_raw_fd()).map_err(|e| Error::msg(format!("stat {name}: {e}")))?;
    if !is_regular(actual.st_mode) || actual.st_dev != st.st_dev || actual.st_ino != st.st_ino {
        return Err(Error::msg("JSON input changed before reading"));
    }
    let file = std::fs::File::from(owned);
    let mut data = Vec::new();
    use std::io::Read;
    file.take((4 << 20) + 1)
        .read_to_end(&mut data)
        .map_err(|e| Error::msg(format!("read {name}: {e}")))?;
    if data.len() > 4 << 20 {
        return Err(Error::msg("JSON input exceeds limit"));
    }
    let value = decode_build_json(&data)?;
    let mut hasher = Sha256::new();
    hasher.update(&data);
    Ok((value, format!("{:x}", hasher.finalize())))
}
