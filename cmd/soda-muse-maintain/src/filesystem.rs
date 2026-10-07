use std::ffi::CString;
use std::fs;
use std::io;
use std::os::unix::fs::MetadataExt;
use std::os::unix::io::RawFd;
use std::path::Path;

use super::sha256::{hex_encode, Sha256};
use super::MUSE_VERSION;
use sha2::Digest;

pub(crate) fn go_base(path: &str) -> &str {
    if path.is_empty() {
        return ".";
    }
    Path::new(path)
        .file_name()
        .and_then(|name| name.to_str())
        .unwrap_or(if Path::new(path).is_absolute() {
            "/"
        } else {
            "."
        })
}

pub(crate) fn go_dir(path: &str) -> String {
    let path = Path::new(path);
    path.parent()
        .filter(|parent| !parent.as_os_str().is_empty())
        .unwrap_or_else(|| {
            if path.is_absolute() {
                Path::new("/")
            } else {
                Path::new(".")
            }
        })
        .to_string_lossy()
        .into_owned()
}

pub(crate) fn path_error(op: &str, path: &str, e: io::Error) -> String {
    format!("{op} {path}: {}", go_errno(e.raw_os_error().unwrap_or(0)))
}

// Convert an errno using Rust's platform error text. Callers preserve their
// operation-specific context around this native error value.
pub(crate) fn go_errno(no: i32) -> String {
    io::Error::from_raw_os_error(no).to_string()
}

pub(crate) fn last_errno() -> i32 {
    io::Error::last_os_error().raw_os_error().unwrap_or(0)
}

#[derive(Debug)]
pub(crate) struct Tool {
    pub(crate) name: String,
    pub(crate) fd: RawFd,
    pub(crate) size: u64,
}

impl Drop for Tool {
    fn drop(&mut self) {
        if self.fd >= 0 {
            unsafe { libc::close(self.fd) };
        }
    }
}

pub(crate) fn load_tools(dir: &str, digest: &str, version: &str) -> Result<Vec<Tool>, String> {
    if version != MUSE_VERSION {
        return Err(String::from(
            "muse maintenance version differs from pinned release",
        ));
    }
    let mut tools = Vec::new();
    for name in ["muse", "soda-identity-compose", "muse-native"] {
        tools.push(open_tool(dir, name)?);
    }
    verify_native(&tools[2], digest)?;
    Ok(tools)
}

pub(crate) fn open_tool(dir: &str, name: &str) -> Result<Tool, String> {
    let path = format!("{dir}/{name}");
    let trusted = String::from("public tool source must be a regular root-owned executable");
    let lstat = fs::symlink_metadata(&path).map_err(|_| trusted.clone())?;
    if !trusted_tool(&lstat) {
        return Err(trusted);
    }
    // A NUL byte fails like Go's BytePtrFromString: raw EINVAL.
    let c = CString::new(path).map_err(|_| go_errno(libc::EINVAL))?;
    let fd = unsafe {
        libc::open(
            c.as_ptr(),
            libc::O_RDONLY | libc::O_NOFOLLOW | libc::O_CLOEXEC | libc::O_NONBLOCK,
        )
    };
    if fd < 0 {
        return Err(go_errno(last_errno()));
    }
    let mut fst: libc::stat = unsafe { std::mem::zeroed() };
    if unsafe { libc::fstat(fd, &mut fst) } != 0 {
        unsafe { libc::close(fd) };
        return Err(String::from("public tool unavailable"));
    }
    if (fst.st_mode & libc::S_IFMT) != libc::S_IFREG
        || (fst.st_mode & 0o7777) & 0o022 != 0
        || fst.st_uid != 0
        || (fst.st_mode & 0o7777) & 0o111 == 0
    {
        unsafe { libc::close(fd) };
        return Err(String::from("public tool source changed"));
    }
    Ok(Tool {
        name: name.to_string(),
        fd,
        size: fst.st_size as u64,
    })
}

fn trusted_tool(info: &fs::Metadata) -> bool {
    info.is_file() && info.uid() == 0 && info.mode() & 0o022 == 0 && info.mode() & 0o111 != 0
}

pub(crate) fn verify_native(tool: &Tool, digest: &str) -> Result<(), String> {
    // io.Copy hashes from the start to EOF; a short or grown file fails
    // the digest comparison, and read failures keep Go's PathError shape.
    // pread leaves the offset at zero like Go's trailing Seek.
    let mut hasher = Sha256::new();
    let mut offset: i64 = 0;
    let mut chunk = [0u8; 65536];
    loop {
        let n = unsafe {
            libc::pread(
                tool.fd,
                chunk.as_mut_ptr() as *mut libc::c_void,
                chunk.len(),
                offset,
            )
        };
        if n < 0 {
            return Err(format!("read {}: {}", tool.name, go_errno(last_errno())));
        }
        if n == 0 {
            break;
        }
        hasher.update(&chunk[..n as usize]);
        offset += n as i64;
    }
    let sum = hasher.finalize();
    if hex_encode(&sum) != digest {
        return Err(String::from("muse native digest mismatch"));
    }
    Ok(())
}

#[cfg(test)]
#[path = "filesystem_tests.rs"]
mod filesystem_tests;
