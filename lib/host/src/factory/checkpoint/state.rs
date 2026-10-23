use std::ffi::CString;
use std::fs::File;
use std::io::{Read, Write};
use std::os::fd::{AsRawFd, FromRawFd};
use std::os::unix::fs::{MetadataExt, PermissionsExt};
use std::path::{Component, Path};
use std::sync::atomic::{AtomicU64, Ordering};

use serde::{de::DeserializeOwned, Deserialize, Serialize};

use super::wire::{CheckpointCheckResult, FactoryTaskCheckpoint};

pub(crate) struct PrivateState(File);

fn io_error(e: std::io::Error) -> String {
    e.to_string()
}

impl PrivateState {
    pub(crate) fn open(path: &str) -> Result<Self, String> {
        if !Path::new(path).is_absolute() {
            return Err("checkpoint state must be absolute".into());
        }
        let mut dir = File::open("/").map_err(io_error)?;
        let uid = unsafe { libc::geteuid() };
        for part in Path::new(path).components() {
            let Component::Normal(name) = part else {
                if part == Component::RootDir {
                    continue;
                }
                return Err("checkpoint state path is not clean".into());
            };
            let name = CString::new(name.as_encoded_bytes()).map_err(|_| "invalid state path")?;
            let fd = unsafe {
                libc::openat(
                    dir.as_raw_fd(),
                    name.as_ptr(),
                    libc::O_RDONLY | libc::O_DIRECTORY | libc::O_NOFOLLOW | libc::O_CLOEXEC,
                )
            };
            if fd < 0 {
                return Err(io_error(std::io::Error::last_os_error()));
            }
            dir = unsafe { File::from_raw_fd(fd) };
            let meta = dir.metadata().map_err(io_error)?;
            if (meta.uid() != 0 && meta.uid() != uid) || meta.permissions().mode() & 0o022 != 0 {
                return Err("checkpoint state has an untrusted ancestor".into());
            }
        }
        let meta = dir.metadata().map_err(io_error)?;
        if meta.uid() != uid || meta.permissions().mode() & 0o077 != 0 {
            return Err("checkpoint state is not service-private".into());
        }
        Ok(Self(dir))
    }

    pub(crate) fn try_clone_directory(&self) -> Result<File, String> {
        self.0.try_clone().map_err(io_error)
    }

    fn name(name: &str) -> Result<CString, String> {
        if name.is_empty() || name.len() > 240 || name.contains(['/', '\0']) {
            return Err("checkpoint state name is invalid".into());
        }
        CString::new(name).map_err(|_| "checkpoint state name is invalid".into())
    }

    pub(in crate::factory) fn read<T: DeserializeOwned>(
        &self,
        name: &str,
    ) -> Result<Option<T>, String> {
        self.read_bytes(name)?
            .map(|bytes| {
                serde_json::from_slice(&bytes).map_err(|_| "checkpoint record is invalid".into())
            })
            .transpose()
    }

    pub(in crate::factory) fn read_bytes(&self, name: &str) -> Result<Option<Vec<u8>>, String> {
        let name = Self::name(name)?;
        let fd = unsafe {
            libc::openat(
                self.0.as_raw_fd(),
                name.as_ptr(),
                libc::O_RDONLY | libc::O_NOFOLLOW | libc::O_NONBLOCK | libc::O_CLOEXEC,
            )
        };
        if fd < 0 {
            let e = std::io::Error::last_os_error();
            return if e.kind() == std::io::ErrorKind::NotFound {
                Ok(None)
            } else {
                Err(io_error(e))
            };
        }
        let file = unsafe { File::from_raw_fd(fd) };
        let meta = file.metadata().map_err(io_error)?;
        if !meta.is_file()
            || meta.nlink() != 1
            || meta.uid() != unsafe { libc::geteuid() }
            || meta.permissions().mode() & 0o077 != 0
        {
            return Err("checkpoint record is not service-private".into());
        }
        let mut bytes = Vec::new();
        file.take(super::super::MAX_TASK_JOURNAL_BYTES as u64 + 4097)
            .read_to_end(&mut bytes)
            .map_err(io_error)?;
        if bytes.len() > super::super::MAX_TASK_JOURNAL_BYTES + 4096 {
            return Err("private record exceeds its bound".into());
        }
        Ok(Some(bytes))
    }

    pub(in crate::factory) fn write<T: Serialize>(
        &self,
        name: &str,
        record: &T,
    ) -> Result<(), String> {
        let bytes = serde_json::to_vec(record).map_err(|_| "checkpoint record encoding failed")?;
        // A task receipt must stay bounded independently of the larger journal.
        if name.starts_with("checkpoint-") && bytes.len() > 64 * 1024 {
            return Err("checkpoint record exceeds its bound".into());
        }
        self.write_bytes(name, &bytes)
    }

    pub(in crate::factory) fn write_bytes(&self, name: &str, bytes: &[u8]) -> Result<(), String> {
        static COUNTER: AtomicU64 = AtomicU64::new(0);
        let name = Self::name(name)?;
        let temporary = CString::new(format!(
            ".checkpoint-{}-{}",
            std::process::id(),
            COUNTER.fetch_add(1, Ordering::Relaxed)
        ))
        .unwrap();
        if bytes.len() > super::super::MAX_TASK_JOURNAL_BYTES + 4096 {
            return Err("private record exceeds its bound".into());
        }
        let fd = unsafe {
            libc::openat(
                self.0.as_raw_fd(),
                temporary.as_ptr(),
                libc::O_WRONLY | libc::O_CREAT | libc::O_EXCL | libc::O_NOFOLLOW | libc::O_CLOEXEC,
                0o600,
            )
        };
        if fd < 0 {
            return Err(io_error(std::io::Error::last_os_error()));
        }
        let mut file = unsafe { File::from_raw_fd(fd) };
        let outcome = (|| {
            file.write_all(&bytes).map_err(io_error)?;
            file.sync_all().map_err(io_error)?;
            if unsafe {
                libc::renameat(
                    self.0.as_raw_fd(),
                    temporary.as_ptr(),
                    self.0.as_raw_fd(),
                    name.as_ptr(),
                )
            } != 0
            {
                return Err(io_error(std::io::Error::last_os_error()));
            }
            self.0.sync_all().map_err(io_error)
        })();
        if outcome.is_err() {
            unsafe {
                libc::unlinkat(self.0.as_raw_fd(), temporary.as_ptr(), 0);
            }
        }
        outcome
    }
}

#[derive(Clone, Debug, Deserialize, Serialize)]
pub(super) struct OutputObservation {
    pub request: FactoryTaskCheckpoint,
    pub child: String,
    pub tree: String,
    pub parent_tree: String,
}

#[derive(Clone, Debug, Deserialize, Serialize)]
pub(super) struct CheckpointOperation {
    pub request: FactoryTaskCheckpoint,
    pub final_candidate: bool,
    pub phase: String,
    pub parent_tree: String,
    pub tree: String,
    pub candidate: String,
    pub created_unix: u64,
    pub checks: Vec<CheckpointCheckResult>,
    pub reason: String,
    pub context: super::git::GitContext,
    pub selected_checks: Vec<super::wire::CheckpointCheck>,
    pub check_index: usize,
    pub check_invocation: String,
    pub check_launch_intent: bool,
}

pub(super) fn operation_key(req: &FactoryTaskCheckpoint, final_candidate: bool) -> String {
    let kind = if final_candidate { "final" } else { "task" };
    crate::sha256::hex_lower(&crate::sha256::digest(
        format!(
            "checkpoint-v1\0{}\0{}\0{kind}\0{}",
            req.id, req.manifest_digest, req.task_key
        )
        .as_bytes(),
    ))
}

pub(super) fn observation_name(req: &FactoryTaskCheckpoint) -> String {
    format!("output-{}.json", operation_key(req, false))
}
pub(super) fn operation_name(req: &FactoryTaskCheckpoint, final_candidate: bool) -> String {
    format!("checkpoint-{}.json", operation_key(req, final_candidate))
}
