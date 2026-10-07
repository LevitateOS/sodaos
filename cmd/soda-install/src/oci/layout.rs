use std::collections::BTreeMap;
use std::ffi::CString;
use std::io::Read;
use std::os::unix::fs::OpenOptionsExt;
use std::os::unix::io::AsRawFd;

use crate::buildx;
use crate::errors::{self, Error};

#[derive(Debug, Clone, Default, PartialEq, Eq)]
pub struct OciImage {
    pub manifest: String,
    pub config: String,
    pub architecture: String,
    pub revision: String,
    pub source: String,
    pub base_name: String,
    pub base_digest: String,
}

#[derive(Debug, Clone, Default)]
pub struct OciLayout {
    pub images: BTreeMap<String, OciImage>,
    pub files: BTreeMap<String, String>,
    pub bytes: u64,
}

#[derive(Debug, Clone, Default)]
pub(super) struct Blob {
    pub(super) hash: String,
    pub(super) size: i64,
    pub(super) data: Option<Vec<u8>>,
}

#[derive(Debug, Clone, Default)]
pub(super) struct Descriptor {
    pub(super) digest: String,
    pub(super) size: i64,
    pub(super) media_type: String,
    pub(super) urls: Vec<String>,
    pub(super) annotations: BTreeMap<String, String>,
}

pub(super) struct Loader {
    pub(super) root: std::fs::File,
    pub(super) entries: BTreeMap<String, Blob>,
    pub(super) json_bytes: usize,
}

// ---------------------------------------------------------------------------
// Confined layout reads.
// ---------------------------------------------------------------------------

pub(super) fn open_root(dir: &str) -> Result<std::fs::File, Error> {
    let st = std::fs::symlink_metadata(dir)
        .map_err(|_| Error::msg("real OCI layout directory required"))?;
    if !st.file_type().is_dir() {
        return Err(Error::msg("real OCI layout directory required"));
    }
    std::fs::OpenOptions::new()
        .read(true)
        .custom_flags(libc::O_DIRECTORY | libc::O_NOFOLLOW)
        .open(dir)
        .map_err(|e| errors::path_error("open", dir, e))
}

/// Open a layout-relative file refusing every symlink, mirroring `os.Root`
/// closely enough for Skopeo-produced layouts (plain directories only).
fn open_layout_file(root: &std::fs::File, name: &str) -> Result<std::fs::File, Error> {
    use std::os::unix::io::FromRawFd;
    let mut dir_fd = root.as_raw_fd();
    // Owned intermediate fds, closed on return.
    let mut owned: Vec<i32> = Vec::new();
    let parts: Vec<&str> = name.split('/').collect();
    for (i, part) in parts.iter().enumerate() {
        if part.is_empty() || *part == "." || *part == ".." {
            for fd in owned {
                unsafe { libc::close(fd) };
            }
            return Err(errors::os_error(std::io::Error::from_raw_os_error(
                libc::ENOENT,
            )));
        }
        let c = CString::new(*part)
            .map_err(|_| errors::os_error(std::io::Error::from_raw_os_error(libc::EINVAL)))?;
        let last = i + 1 == parts.len();
        let flags = if last {
            libc::O_RDONLY | libc::O_NOFOLLOW | libc::O_NONBLOCK | libc::O_CLOEXEC
        } else {
            libc::O_RDONLY | libc::O_DIRECTORY | libc::O_NOFOLLOW | libc::O_CLOEXEC
        };
        let fd = unsafe { libc::openat(dir_fd, c.as_ptr(), flags, 0) };
        if fd < 0 {
            let errno = unsafe { *libc::__errno_location() };
            for fd in owned {
                unsafe { libc::close(fd) };
            }
            return Err(errors::path_error(
                "open",
                name,
                std::io::Error::from_raw_os_error(errno),
            ));
        }
        if last {
            for fd in owned {
                unsafe { libc::close(fd) };
            }
            return Ok(unsafe { std::fs::File::from_raw_fd(fd) });
        }
        owned.push(fd);
        dir_fd = fd;
    }
    for fd in owned {
        unsafe { libc::close(fd) };
    }
    Err(errors::os_error(std::io::Error::from_raw_os_error(
        libc::ENOENT,
    )))
}

fn lstat_layout_file(root: &std::fs::File, name: &str) -> Result<libc::stat, Error> {
    let c = CString::new(name)
        .map_err(|_| errors::os_error(std::io::Error::from_raw_os_error(libc::EINVAL)))?;
    let mut st: libc::stat = unsafe { std::mem::zeroed() };
    if unsafe {
        libc::fstatat(
            root.as_raw_fd(),
            c.as_ptr(),
            &mut st,
            libc::AT_SYMLINK_NOFOLLOW,
        )
    } != 0
    {
        let errno = unsafe { *libc::__errno_location() };
        return Err(errors::path_error(
            "lstat",
            name,
            std::io::Error::from_raw_os_error(errno),
        ));
    }
    Ok(st)
}

fn copy_oci_blob(
    name: &str,
    length: i64,
    reader: &mut dyn Read,
) -> Result<(String, i64, Option<Vec<u8>>), Error> {
    use sha2::Digest as _;
    let mut hasher = sha2::Sha256::new();
    let mut body: Vec<u8> = Vec::new();
    let retain = length <= 4 << 20;
    let mut remaining = length as u64 + 1;
    let mut size: i64 = 0;
    let mut chunk = [0u8; 65536];
    loop {
        if remaining == 0 {
            break;
        }
        let want = remaining.min(chunk.len() as u64) as usize;
        let n = reader
            .read(&mut chunk[..want])
            .map_err(|e| errors::path_error("read", name, e))?;
        if n == 0 {
            break;
        }
        hasher.update(&chunk[..n]);
        if retain {
            body.extend_from_slice(&chunk[..n]);
        }
        size += n as i64;
        remaining -= n as u64;
    }
    if size != length {
        return Err(Error::msg("OCI blob size changed"));
    }
    let sum = buildx::hex_encode(&hasher.finalize());
    if name.starts_with("blobs/") && *name != format!("blobs/sha256/{sum}") {
        return Err(Error::msg("OCI blob checksum mismatch"));
    }
    let body = if retain {
        let text = String::from_utf8_lossy(&body);
        serde_json::from_str::<serde_json::Value>(&text)
            .ok()
            .map(|_| body)
    } else {
        None
    };
    Ok((sum, size, body))
}

fn read_oci_blob(
    loader: &mut Loader,
    name: &str,
    length: i64,
    reader: &mut dyn Read,
) -> Result<(), Error> {
    if length < 0 || length == i64::MAX {
        return Err(Error::msg("invalid OCI blob size"));
    }
    let (sum, size, body) = copy_oci_blob(name, length, reader)?;
    loader.json_bytes += body.as_ref().map(|b| b.len()).unwrap_or(0);
    if loader.json_bytes > 32 << 20 {
        return Err(Error::msg("OCI JSON metadata limit exceeded"));
    }
    loader.entries.insert(
        name.to_string(),
        Blob {
            hash: sum,
            size,
            data: body,
        },
    );
    Ok(())
}

pub(super) fn load_layout_file(loader: &mut Loader, name: &str) -> Result<(), Error> {
    if loader.entries.contains_key(name) {
        return Ok(());
    }
    let st = lstat_layout_file(&loader.root, name)?;
    if st.st_mode & libc::S_IFMT != libc::S_IFREG {
        return Err(Error::msg("non-regular OCI layout entry"));
    }
    let file = open_layout_file(&loader.root, name)?;
    let mut opened: libc::stat = unsafe { std::mem::zeroed() };
    if unsafe { libc::fstat(file.as_raw_fd(), &mut opened) } != 0 {
        let errno = unsafe { *libc::__errno_location() };
        return Err(errors::path_error(
            "stat",
            name,
            std::io::Error::from_raw_os_error(errno),
        ));
    }
    if st.st_dev != opened.st_dev || st.st_ino != opened.st_ino {
        return Err(Error::msg("OCI layout entry changed"));
    }
    let mut file = file;
    read_oci_blob(loader, name, st.st_size, &mut file)
}
