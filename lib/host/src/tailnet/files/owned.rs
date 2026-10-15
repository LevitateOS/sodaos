//! Root-owned JSON records and bounded file exclusion for Tailnet resources.
use serde::{de::DeserializeOwned, Serialize};
use std::fs::{self, File, OpenOptions};
use std::io::{Read, Write};
use std::os::fd::AsRawFd;
use std::os::unix::fs::{DirBuilderExt, MetadataExt, OpenOptionsExt};
use std::path::PathBuf;
use std::time::{Duration, Instant};

fn unavailable() -> String {
    "unavailable: Tailnet resource state is unavailable".to_string()
}

pub(crate) struct OwnedState {
    pub(crate) root: PathBuf,
    pub(crate) owner: u32,
}

impl OwnedState {
    pub(crate) fn check(&self) -> Result<bool, String> {
        match fs::symlink_metadata(&self.root) {
            Err(e) if e.kind() == std::io::ErrorKind::NotFound => Ok(false),
            Err(_) => Err(unavailable()),
            Ok(_) => self.directory(false).map(|_| true),
        }
    }

    pub(crate) fn ensure(&self) -> Result<(), String> {
        self.directory(true).map(|_| ())
    }
    fn directory(&self, create: bool) -> Result<File, String> {
        if create {
            match fs::DirBuilder::new().mode(0o700).create(&self.root) {
                Ok(()) => {}
                Err(e) if e.kind() == std::io::ErrorKind::AlreadyExists => {}
                Err(_) => return Err(unavailable()),
            }
        }
        let dir = OpenOptions::new()
            .read(true)
            .custom_flags(libc::O_DIRECTORY | libc::O_NOFOLLOW | libc::O_CLOEXEC)
            .open(&self.root)
            .map_err(|_| unavailable())?;
        let info = dir.metadata().map_err(|_| unavailable())?;
        if !info.is_dir()
            || info.uid() != self.owner
            || info.gid() != self.owner
            || info.mode() & 0o7777 != 0o700
        {
            return Err(unavailable());
        }
        Ok(dir)
    }

    fn safe_file(&self, file: &File) -> Result<(), String> {
        let info = file.metadata().map_err(|_| unavailable())?;
        if !info.is_file()
            || info.uid() != self.owner
            || info.gid() != self.owner
            || info.mode() & 0o7777 != 0o600
            || info.nlink() != 1
        {
            return Err(unavailable());
        }
        Ok(())
    }

    pub(crate) fn read<T: DeserializeOwned>(&self, leaf: &str) -> Result<T, String> {
        if !valid_leaf(leaf) {
            return Err(unavailable());
        }
        let _dir = self.directory(false)?;
        let file = OpenOptions::new()
            .read(true)
            .custom_flags(libc::O_NOFOLLOW | libc::O_NONBLOCK | libc::O_CLOEXEC)
            .open(self.root.join(leaf))
            .map_err(|_| unavailable())?;
        self.safe_file(&file)?;
        let mut data = Vec::new();
        file.take(8193)
            .read_to_end(&mut data)
            .map_err(|_| unavailable())?;
        if data.len() > 8192 {
            return Err(unavailable());
        }
        crate::json::decode_strict_as(&data).map_err(|_| unavailable())
    }

    pub(crate) fn remove(&self, leaf: &str) -> Result<(), String> {
        if !valid_leaf(leaf) {
            return Err(unavailable());
        }
        match fs::symlink_metadata(&self.root) {
            Err(e) if e.kind() == std::io::ErrorKind::NotFound => return Ok(()),
            Err(_) => return Err(unavailable()),
            Ok(_) => {}
        }
        let dir = self.directory(false)?;
        match fs::remove_file(self.root.join(leaf)) {
            Ok(()) => dir.sync_all().map_err(|_| unavailable()),
            Err(e) if e.kind() == std::io::ErrorKind::NotFound => Ok(()),
            Err(_) => Err(unavailable()),
        }
    }

    pub(crate) fn write<T: Serialize>(&self, leaf: &str, record: &T) -> Result<(), String> {
        if !valid_leaf(leaf) {
            return Err(unavailable());
        }
        let dir = self.directory(true)?;
        let name = format!("record-{}.tmp", crate::tcontrol_policy::new_revision()?);
        let temp = self.root.join(name);
        let result = (|| {
            let mut file = OpenOptions::new()
                .write(true)
                .create_new(true)
                .mode(0o600)
                .custom_flags(libc::O_NOFOLLOW | libc::O_CLOEXEC)
                .open(&temp)
                .map_err(|_| unavailable())?;
            self.safe_file(&file)?;
            let data = serde_json::to_vec(record).map_err(|_| unavailable())?;
            if data.len() > 8192 {
                return Err(unavailable());
            }
            file.write_all(&data)
                .and_then(|_| file.sync_all())
                .map_err(|_| unavailable())?;
            fs::rename(&temp, self.root.join(leaf)).map_err(|_| unavailable())?;
            dir.sync_all().map_err(|_| unavailable())
        })();
        if result.is_err() {
            let _ = fs::remove_file(temp);
        }
        result
    }

    pub(crate) fn lock(
        &self,
        name: &str,
        exclusive: bool,
        deadline: Instant,
    ) -> Result<OwnedLock, String> {
        if !valid_leaf(name) {
            return Err(unavailable());
        }
        let _dir = self.directory(true)?;
        let file = OpenOptions::new()
            .read(true)
            .write(true)
            .create(true)
            .truncate(false)
            .mode(0o600)
            .custom_flags(libc::O_NOFOLLOW | libc::O_CLOEXEC | libc::O_NONBLOCK)
            .open(self.root.join(name))
            .map_err(|_| unavailable())?;
        self.safe_file(&file)?;
        let mode = if exclusive {
            libc::LOCK_EX
        } else {
            libc::LOCK_SH
        };
        loop {
            if Instant::now() >= deadline {
                return Err(unavailable());
            }
            if unsafe { libc::flock(file.as_raw_fd(), mode | libc::LOCK_NB) } == 0 {
                return Ok(OwnedLock(file));
            }
            if std::io::Error::last_os_error().raw_os_error() != Some(libc::EWOULDBLOCK) {
                return Err(unavailable());
            }
            std::thread::sleep(
                Duration::from_millis(25).min(deadline.saturating_duration_since(Instant::now())),
            );
        }
    }
}

fn valid_leaf(leaf: &str) -> bool {
    !leaf.is_empty() && leaf != "." && leaf != ".." && !leaf.contains(['/', '\0'])
}

pub struct OwnedLock(File);

impl Drop for OwnedLock {
    fn drop(&mut self) {
        unsafe { libc::flock(self.0.as_raw_fd(), libc::LOCK_UN) };
    }
}
