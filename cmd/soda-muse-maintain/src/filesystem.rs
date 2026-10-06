use std::ffi::CString;
use std::fs;
use std::io;
use std::os::unix::fs::MetadataExt;
use std::os::unix::io::RawFd;

use super::sha256::{hex_encode, Sha256};
use super::MUSE_VERSION;

// go_clean mirrors filepath.Clean lexical rules.
pub(crate) fn go_clean(path: &str) -> String {
    if path.is_empty() {
        return String::from(".");
    }
    let rooted = path.starts_with('/');
    let mut out: Vec<&str> = Vec::new();
    for part in path.split('/') {
        match part {
            "" | "." => {}
            ".." => {
                if let Some(last) = out.pop() {
                    if last == ".." {
                        out.push("..");
                        out.push("..");
                    }
                } else if !rooted {
                    out.push("..");
                }
            }
            _ => out.push(part),
        }
    }
    let mut clean = out.join("/");
    if rooted {
        clean.insert(0, '/');
    }
    if clean.is_empty() {
        clean.push('.');
    }
    clean
}

pub(crate) fn go_base(path: &str) -> &str {
    if path.is_empty() {
        return ".";
    }
    let stripped = path.trim_end_matches('/');
    if stripped.is_empty() {
        return "/";
    }
    match stripped.rfind('/') {
        Some(i) => &stripped[i + 1..],
        None => stripped,
    }
}

pub(crate) fn go_dir(path: &str) -> String {
    // filepath.Dir: Clean of everything through the last separator.
    let mut i = path.len();
    let b = path.as_bytes();
    while i > 0 && b[i - 1] != b'/' {
        i -= 1;
    }
    go_clean(&path[..i])
}

pub(crate) fn path_error(op: &str, path: &str, e: io::Error) -> String {
    format!("{op} {path}: {}", go_errno(e.raw_os_error().unwrap_or(0)))
}

// go_errno renders the stable Go syscall errno table, which never follows
// the process locale the way libc strerror does.
pub(crate) fn go_errno(no: i32) -> String {
    let text = match no {
        libc::EPERM => "operation not permitted",
        libc::ENOENT => "no such file or directory",
        libc::ESRCH => "no such process",
        libc::EINTR => "interrupted system call",
        libc::EIO => "input/output error",
        libc::ENXIO => "no such device or address",
        libc::E2BIG => "argument list too long",
        libc::ENOEXEC => "exec format error",
        libc::EBADF => "bad file descriptor",
        libc::ECHILD => "no child processes",
        libc::EAGAIN => "resource temporarily unavailable",
        libc::ENOMEM => "cannot allocate memory",
        libc::EACCES => "permission denied",
        libc::EFAULT => "bad address",
        libc::ENOTBLK => "block device required",
        libc::EBUSY => "device or resource busy",
        libc::EEXIST => "file exists",
        libc::EXDEV => "invalid cross-device link",
        libc::ENODEV => "no such device",
        libc::ENOTDIR => "not a directory",
        libc::EISDIR => "is a directory",
        libc::EINVAL => "invalid argument",
        libc::ENFILE => "too many open files in system",
        libc::EMFILE => "too many open files",
        libc::ENOTTY => "inappropriate ioctl for device",
        libc::ETXTBSY => "text file busy",
        libc::EFBIG => "file too large",
        libc::ENOSPC => "no space left on device",
        libc::ESPIPE => "illegal seek",
        libc::EROFS => "read-only file system",
        libc::EMLINK => "too many links",
        libc::EPIPE => "broken pipe",
        libc::EDOM => "numerical argument out of domain",
        libc::ERANGE => "numerical result out of range",
        libc::EDEADLK => "resource deadlock avoided",
        libc::ENAMETOOLONG => "file name too long",
        libc::ENOLCK => "no locks available",
        libc::ENOSYS => "function not implemented",
        libc::ENOTEMPTY => "directory not empty",
        libc::ELOOP => "too many levels of symbolic links",
        libc::ENOMSG => "no message of desired type",
        libc::EIDRM => "identifier removed",
        libc::ECHRNG => "channel number out of range",
        libc::EL2NSYNC => "level 2 not synchronized",
        libc::EL3HLT => "level 3 halted",
        libc::EL3RST => "level 3 reset",
        libc::ELNRNG => "link number out of range",
        libc::EUNATCH => "protocol driver not attached",
        libc::ENOCSI => "no CSI structure available",
        libc::EL2HLT => "level 2 halted",
        libc::EBADE => "invalid exchange",
        libc::EBADR => "invalid request descriptor",
        libc::EXFULL => "exchange full",
        libc::ENOANO => "no anode",
        libc::EBADRQC => "invalid request code",
        libc::EBADSLT => "invalid slot",
        libc::EBFONT => "bad font file format",
        libc::ENOSTR => "device not a stream",
        libc::ENODATA => "no data available",
        libc::ETIME => "timer expired",
        libc::ENOSR => "out of streams resources",
        libc::ENONET => "machine is not on the network",
        libc::ENOPKG => "package not installed",
        libc::EREMOTE => "object is remote",
        libc::ENOLINK => "link has been severed",
        libc::EADV => "advertise error",
        libc::ESRMNT => "srmount error",
        libc::ECOMM => "communication error on send",
        libc::EPROTO => "protocol error",
        libc::EMULTIHOP => "multihop attempted",
        libc::EDOTDOT => "RFS specific error",
        libc::EBADMSG => "bad message",
        libc::EOVERFLOW => "value too large for defined data type",
        libc::ENOTUNIQ => "name not unique on network",
        libc::EBADFD => "file descriptor in bad state",
        libc::EREMCHG => "remote address changed",
        libc::ELIBACC => "can not access a needed shared library",
        libc::ELIBBAD => "accessing a corrupted shared library",
        libc::ELIBSCN => ".lib section in a.out corrupted",
        libc::ELIBMAX => "attempting to link in too many shared libraries",
        libc::ELIBEXEC => "cannot exec a shared library directly",
        libc::EILSEQ => "invalid or incomplete multibyte or wide character",
        libc::ERESTART => "interrupted system call should be restarted",
        libc::ESTRPIPE => "streams pipe error",
        libc::EUSERS => "too many users",
        libc::ENOTSOCK => "socket operation on non-socket",
        libc::EDESTADDRREQ => "destination address required",
        libc::EMSGSIZE => "message too long",
        libc::EPROTOTYPE => "protocol wrong type for socket",
        libc::ENOPROTOOPT => "protocol not available",
        libc::EPROTONOSUPPORT => "protocol not supported",
        libc::ESOCKTNOSUPPORT => "socket type not supported",
        libc::EOPNOTSUPP => "operation not supported",
        libc::EPFNOSUPPORT => "protocol family not supported",
        libc::EAFNOSUPPORT => "address family not supported by protocol",
        libc::EADDRINUSE => "address already in use",
        libc::EADDRNOTAVAIL => "cannot assign requested address",
        libc::ENETDOWN => "network is down",
        libc::ENETUNREACH => "network is unreachable",
        libc::ENETRESET => "network dropped connection on reset",
        libc::ECONNABORTED => "software caused connection abort",
        libc::ECONNRESET => "connection reset by peer",
        libc::ENOBUFS => "no buffer space available",
        libc::EISCONN => "transport endpoint is already connected",
        libc::ENOTCONN => "transport endpoint is not connected",
        libc::ESHUTDOWN => "cannot send after transport endpoint shutdown",
        libc::ETOOMANYREFS => "too many references: cannot splice",
        libc::ETIMEDOUT => "connection timed out",
        libc::ECONNREFUSED => "connection refused",
        libc::EHOSTDOWN => "host is down",
        libc::EHOSTUNREACH => "no route to host",
        libc::EALREADY => "operation already in progress",
        libc::EINPROGRESS => "operation now in progress",
        libc::ESTALE => "stale file handle",
        libc::EUCLEAN => "structure needs cleaning",
        libc::ENOTNAM => "not a XENIX named type file",
        libc::ENAVAIL => "no XENIX semaphores available",
        libc::EISNAM => "is a named type file",
        libc::EREMOTEIO => "remote I/O error",
        libc::EDQUOT => "disk quota exceeded",
        libc::ENOMEDIUM => "no medium found",
        libc::EMEDIUMTYPE => "wrong medium type",
        libc::ECANCELED => "operation canceled",
        libc::ENOKEY => "required key not available",
        libc::EKEYEXPIRED => "key has expired",
        libc::EKEYREVOKED => "key has been revoked",
        libc::EKEYREJECTED => "key was rejected by service",
        libc::EOWNERDEAD => "owner died",
        libc::ENOTRECOVERABLE => "state not recoverable",
        libc::ERFKILL => "operation not possible due to RF-kill",
        libc::EHWPOISON => "memory page has hardware error",
        _ => return format!("errno {no}"),
    };
    String::from(text)
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
    let sum = hasher.finish();
    if hex_encode(&sum) != digest {
        return Err(String::from("muse native digest mismatch"));
    }
    Ok(())
}

#[cfg(test)]
#[path = "filesystem_tests.rs"]
mod filesystem_tests;
