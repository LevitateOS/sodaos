use std::io;
use std::path::{Path, PathBuf};
use std::sync::OnceLock;
use std::time::{Duration, Instant};

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
    fn elapsed(&mut self) -> Duration;
    fn sleep(&mut self, secs: u64);
}

pub(crate) struct RealSys;

impl Sys for RealSys {
    fn euid(&self) -> u32 {
        unsafe { libc::geteuid() }
    }

    fn lookup_user(&self, name: &str) -> Result<(u32, u32), String> {
        let cname = std::ffi::CString::new(name)
            .map_err(|_| format!("getpwnam(): name not found: {name:?}"))?;
        let mut result = std::ptr::null_mut();
        let mut size = unsafe { libc::sysconf(libc::_SC_GETPW_R_SIZE_MAX) };
        if size < 0 {
            size = 1024;
        }
        const MAX_NSS_BYTES: usize = 1024 * 1024;
        let mut buffer = vec![0u8; (size as usize).clamp(1024, MAX_NSS_BYTES)];
        loop {
            let mut entry: libc::passwd = unsafe { std::mem::zeroed() };
            let code = unsafe {
                libc::getpwnam_r(
                    cname.as_ptr(),
                    &mut entry,
                    buffer.as_mut_ptr().cast(),
                    buffer.len(),
                    &mut result,
                )
            };
            if code == libc::ERANGE {
                if buffer.len() >= MAX_NSS_BYTES {
                    return Err("getpwnam(): account record exceeds limit".to_string());
                }
                buffer.resize((buffer.len() * 2).min(MAX_NSS_BYTES), 0);
                continue;
            }
            if code != 0 {
                return Err(format!(
                    "getpwnam(): {}",
                    io::Error::from_raw_os_error(code)
                ));
            }
            if result.is_null() {
                return Err(format!("getpwnam(): name not found: {name:?}"));
            }
            return Ok((entry.pw_uid, entry.pw_gid));
        }
    }

    fn chown(&mut self, path: &Path, uid: u32, gid: u32) -> io::Result<()> {
        use std::os::unix::fs::chown;
        chown(path, Some(uid), Some(gid))
    }

    fn run(&mut self, argv: &[&str]) -> io::Result<i32> {
        let status = std::process::Command::new(argv[0])
            .args(&argv[1..])
            .status()?;
        Ok(status.code().unwrap_or(1))
    }

    fn elapsed(&mut self) -> Duration {
        static ORIGIN: OnceLock<Instant> = OnceLock::new();
        ORIGIN.get_or_init(Instant::now).elapsed()
    }

    fn sleep(&mut self, secs: u64) {
        std::thread::sleep(std::time::Duration::from_secs(secs));
    }
}
