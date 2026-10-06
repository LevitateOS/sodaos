use std::ffi::CString;
use std::io;
use std::os::unix::ffi::OsStrExt;
use std::path::{Path, PathBuf};

#[link(name = "c")]
extern "C" {
    fn geteuid() -> u32;
    fn chown(path: *const std::os::raw::c_char, owner: u32, group: u32) -> std::os::raw::c_int;
    fn getpwnam(name: *const std::os::raw::c_char) -> *const Passwd;
}

#[repr(C)]
struct Passwd {
    pw_name: *const std::os::raw::c_char,
    pw_passwd: *const std::os::raw::c_char,
    pw_uid: u32,
    pw_gid: u32,
}

pub(crate) struct Paths {
    pub(crate) root: PathBuf,
    pub(crate) var_lib: PathBuf,
    pub(crate) containers_systemd: PathBuf,
}

impl Paths {
    pub(crate) fn production() -> Paths {
        Paths {
            root: PathBuf::from("/etc/soda"),
            var_lib: PathBuf::from("/var/lib/soda"),
            containers_systemd: PathBuf::from("/etc/containers/systemd"),
        }
    }
}

#[derive(Debug, PartialEq)]
pub(crate) enum ActivateError {
    /// argparse-style validation failure: usage + error, exit 2.
    Usage(String),
    /// Runtime failure (IO, missing user, failing unit): message, exit 1.
    Runtime(String),
}

pub(crate) fn usage(msg: impl Into<String>) -> ActivateError {
    ActivateError::Usage(msg.into())
}

pub(crate) fn runtime(msg: impl Into<String>) -> ActivateError {
    ActivateError::Runtime(msg.into())
}

/// Privileged operations, injectable for tests. Plain file reads/writes go
/// through std directly; only identity, ownership, processes, and time vary.
pub(crate) trait Sys {
    fn euid(&self) -> u32;
    fn lookup_user(&self, name: &str) -> Result<(u32, u32), String>;
    fn chown(&mut self, path: &Path, uid: u32, gid: u32) -> io::Result<()>;
    fn run(&mut self, argv: &[&str]) -> io::Result<i32>;
    fn now(&mut self) -> f64;
    fn sleep(&mut self, secs: u64);
}

pub(crate) struct RealSys;

impl Sys for RealSys {
    fn euid(&self) -> u32 {
        unsafe { geteuid() }
    }

    fn lookup_user(&self, name: &str) -> Result<(u32, u32), String> {
        let cname =
            CString::new(name).map_err(|_| format!("getpwnam(): name not found: {name:?}"))?;
        let entry = unsafe { getpwnam(cname.as_ptr()) };
        if entry.is_null() {
            return Err(format!("getpwnam(): name not found: {name:?}"));
        }
        let (uid, gid) = unsafe { ((*entry).pw_uid, (*entry).pw_gid) };
        Ok((uid, gid))
    }

    fn chown(&mut self, path: &Path, uid: u32, gid: u32) -> io::Result<()> {
        let bytes = path.as_os_str().as_bytes();
        let mut nul = Vec::with_capacity(bytes.len() + 1);
        nul.extend_from_slice(bytes);
        nul.push(0);
        let ret = unsafe { chown(nul.as_ptr() as *const std::os::raw::c_char, uid, gid) };
        if ret == 0 {
            Ok(())
        } else {
            Err(io::Error::last_os_error())
        }
    }

    fn run(&mut self, argv: &[&str]) -> io::Result<i32> {
        let status = std::process::Command::new(argv[0])
            .args(&argv[1..])
            .status()?;
        Ok(status.code().unwrap_or(1))
    }

    fn now(&mut self) -> f64 {
        use std::time::{SystemTime, UNIX_EPOCH};
        SystemTime::now()
            .duration_since(UNIX_EPOCH)
            .map(|d| d.as_secs_f64())
            .unwrap_or(0.0)
    }

    fn sleep(&mut self, secs: u64) {
        std::thread::sleep(std::time::Duration::from_secs(secs));
    }
}
