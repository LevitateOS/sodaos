use super::runtime::{data_directory, Runtime};
use super::wire::{Observation, Request, RestoreOperation};
use crate::project::Executor;
use serde::{Deserialize, Serialize};
use std::fs::{self, File, OpenOptions};
use std::io::{Read, Write};
use std::os::fd::AsRawFd;
use std::os::unix::fs::{MetadataExt, OpenOptionsExt, PermissionsExt};
use std::path::{Path, PathBuf};
use std::time::Instant;

#[derive(Serialize, Deserialize)]
#[serde(deny_unknown_fields)]
struct Journal {
    operation: RestoreOperation,
    phase: String,
}

fn journal_path(id: &str) -> Result<PathBuf, String> {
    let snapshots = super::backups::root()?;
    let directory = snapshots
        .parent()
        .ok_or("application storage parent unavailable")?
        .join("restores");
    match fs::create_dir(&directory) {
        Ok(()) => fs::set_permissions(&directory, fs::Permissions::from_mode(0o700))
            .map_err(|e| e.to_string())?,
        Err(error) if error.kind() == std::io::ErrorKind::AlreadyExists => (),
        Err(_) => return Err("restore journal is unavailable".into()),
    }
    let meta = fs::symlink_metadata(&directory).map_err(|_| "restore journal is unavailable")?;
    if !meta.is_dir()
        || meta.file_type().is_symlink()
        || meta.uid() != 0
        || meta.mode() & 0o077 != 0
    {
        return Err("restore journal must be private and operator-owned".into());
    }
    Ok(directory.join(format!("{id}.json")))
}

fn save(path: &Path, journal: &Journal) -> Result<(), String> {
    let mut nonce = [0u8; 12];
    getrandom::fill(&mut nonce).map_err(|_| "restore journal randomness unavailable")?;
    let temporary = path.with_extension(format!("{}.new", crate::sha256::hex_lower(&nonce)));
    let result = (|| -> Result<(), String> {
        let mut file = OpenOptions::new()
            .write(true)
            .create_new(true)
            .mode(0o600)
            .custom_flags(libc::O_NOFOLLOW)
            .open(&temporary)
            .map_err(|_| "restore journal cannot be written")?;
        file.write_all(
            &serde_json::to_vec(journal).map_err(|_| "restore journal cannot be encoded")?,
        )
        .map_err(|_| "restore journal cannot be written")?;
        file.sync_all()
            .map_err(|_| "restore journal cannot be synchronized")?;
        fs::rename(&temporary, path).map_err(|_| "restore journal cannot be committed")?;
        File::open(path.parent().ok_or("restore journal parent unavailable")?)
            .and_then(|file| file.sync_all())
            .map_err(|_| "restore journal cannot be synchronized".to_string())
    })();
    if result.is_err() {
        let _ = fs::remove_file(temporary);
    }
    result
}

fn journal(path: &Path, operation: &RestoreOperation) -> Result<Journal, String> {
    match OpenOptions::new()
        .read(true)
        .custom_flags(libc::O_NOFOLLOW | libc::O_NONBLOCK)
        .open(path)
    {
        Ok(file) => {
            let meta = file
                .metadata()
                .map_err(|_| "restore journal is unavailable")?;
            if !meta.is_file()
                || meta.uid() != 0
                || meta.nlink() != 1
                || meta.mode() & 0o077 != 0
                || meta.len() > 65536
            {
                return Err("restore journal is unsafe".into());
            }
            let recorded: Journal = serde_json::from_reader(file.take(65537))
                .map_err(|_| "restore journal is invalid")?;
            if &recorded.operation != operation
                || !["prepared", "copied", "installed", "aborted"]
                    .contains(&recorded.phase.as_str())
            {
                return Err("restore operation identity changed".into());
            }
            Ok(recorded)
        }
        Err(error) if error.kind() == std::io::ErrorKind::NotFound => {
            let journal = Journal {
                operation: operation.clone(),
                phase: "prepared".into(),
            };
            save(path, &journal)?;
            Ok(journal)
        }
        Err(_) => Err("restore journal is unavailable".into()),
    }
}

fn lock(path: &Path) -> Result<File, String> {
    let file = OpenOptions::new()
        .read(true)
        .write(true)
        .custom_flags(libc::O_NOFOLLOW)
        .open(path.join(".writer-lock"))
        .map_err(|_| "application writer fence unavailable")?;
    if unsafe { libc::flock(file.as_raw_fd(), libc::LOCK_EX | libc::LOCK_NB) } != 0 {
        return Err("application data still has a writer".into());
    }
    Ok(file)
}

// Only a prepared operation whose original generation still exists can retire
// an incomplete output. Retain its bytes under the private journal directory;
// the same reserved identity can then receive a new verified copy.
fn retain_partial(target: &Path, journal: &Path, operation: &str) -> Result<(), String> {
    super::home_path(target)?;
    let meta = fs::symlink_metadata(target).map_err(|_| "partial restore is unavailable")?;
    if !meta.is_dir()
        || meta.file_type().is_symlink()
        || ![0, 10000].contains(&meta.uid())
        || meta.mode() & 0o077 != 0
    {
        return Err("partial restore directory is unsafe".into());
    }
    let _writer = match OpenOptions::new()
        .read(true)
        .write(true)
        .custom_flags(libc::O_NOFOLLOW | libc::O_NONBLOCK)
        .open(target.join(".writer-lock"))
    {
        Ok(file) => {
            let meta = file
                .metadata()
                .map_err(|_| "partial restore fence is unavailable")?;
            if !meta.is_file()
                || meta.nlink() != 1
                || meta.mode() & 0o077 != 0
                || ![0, 10000].contains(&meta.uid())
                || unsafe { libc::flock(file.as_raw_fd(), libc::LOCK_EX | libc::LOCK_NB) } != 0
            {
                return Err("partial restore still has an unsafe or active writer".into());
            }
            Some(file)
        }
        Err(error) if error.kind() == std::io::ErrorKind::NotFound => None,
        Err(_) => return Err("partial restore fence is unsafe".into()),
    };
    let parent = journal
        .parent()
        .ok_or("restore journal parent unavailable")?;
    let mut nonce = [0u8; 12];
    getrandom::fill(&mut nonce).map_err(|_| "restore retention randomness unavailable")?;
    let retained = parent.join(format!(
        "{operation}-partial-{}",
        crate::sha256::hex_lower(&nonce)
    ));
    let from = std::ffi::CString::new(target.as_os_str().as_encoded_bytes())
        .map_err(|_| "invalid partial restore path")?;
    let to = std::ffi::CString::new(retained.as_os_str().as_encoded_bytes())
        .map_err(|_| "invalid partial restore path")?;
    if unsafe {
        libc::renameat2(
            libc::AT_FDCWD,
            from.as_ptr(),
            libc::AT_FDCWD,
            to.as_ptr(),
            libc::RENAME_NOREPLACE,
        )
    } != 0
    {
        return Err("partial restore could not be retained".into());
    }
    for directory in [
        target.parent().ok_or("restore data parent unavailable")?,
        parent,
    ] {
        File::open(directory)
            .and_then(|file| file.sync_all())
            .map_err(|_| "partial restore retention could not be synchronized")?;
    }
    Ok(())
}

impl<E: Executor> Runtime<E> {
    pub(super) fn restore_application(
        &self,
        request: &Request,
        deadline: Instant,
    ) -> Result<Observation, String> {
        let operation = request
            .restore
            .as_ref()
            .ok_or("restore intent is required")?;
        operation.validate()?;
        let path = journal_path(&operation.id)?;
        let mut state = journal(&path, operation)?;
        let current = self.container_id(&operation.previous.id, deadline)?;
        let original_data = data_directory(&operation.previous.data)?;
        let _original_writer = lock(&original_data)?;
        if request.action == "restore_abort" {
            if current != operation.expected_container || state.phase == "installed" {
                return Err(
                    "restore already changed the native generation; reconcile it before acting"
                        .into(),
                );
            }
            let native = self.inspect(&operation.previous, &current, deadline)?;
            if native.running || native.image != operation.previous.image {
                return Err("stop the original application before aborting restore".into());
            }
            state.phase = "aborted".into();
            save(&path, &state)?;
            return self.observe(&operation.previous, native, deadline);
        }
        if state.phase == "aborted" {
            return Err("restore was aborted; choose a new operation and fresh data".into());
        }
        let snapshot_path = super::backups::root()?.join(&operation.snapshot);
        let snapshot = super::snapshot::application_info(&snapshot_path)?;
        let mut expected = operation.previous.clone();
        expected.revision = snapshot.application.revision;
        expected.data = snapshot.application.data.clone();
        if snapshot.application != expected {
            return Err(
                "snapshot does not match the admitted application bundle, owner or address".into(),
            );
        }
        let target = original_data
            .parent()
            .ok_or("application data parent unavailable")?
            .join(&operation.next.data);
        if !current.is_empty() && current != operation.expected_container {
            let native = self.inspect(&operation.next, &current, deadline)?;
            if native.running || native.image != operation.next.image {
                return Err(
                    "restored generation must remain stopped until admission is reconciled".into(),
                );
            }
            let restored_data = data_directory(&operation.next.data)?;
            let _new_writer = lock(&restored_data)?;
            super::snapshot::verify_restored(&snapshot_path, &restored_data)?;
            state.phase = "installed".into();
            save(&path, &state)?;
            return self.observe(&operation.next, native, deadline);
        }
        if current.is_empty() && state.phase != "copied" {
            return Err(
                "original application generation is unavailable; inspect before retrying".into(),
            );
        }
        if !current.is_empty() {
            let native =
                self.inspect(&operation.previous, &operation.expected_container, deadline)?;
            if native.running || native.image != operation.previous.image {
                return Err("stop the original admitted application before restoring".into());
            }
        }
        self.systemd("disable", &operation.previous.id, deadline)?;
        if target
            .try_exists()
            .map_err(|_| "restore output is unavailable")?
        {
            super::snapshot::verify_source(&snapshot_path)?;
            let reusable = data_directory(&operation.next.data).and_then(|data| {
                let _writer = lock(&data)?;
                super::snapshot::verify_restored(&snapshot_path, &data)
            });
            if reusable.is_err() {
                if state.phase != "prepared" || current != operation.expected_container {
                    return Err("restored data does not match the selected snapshot".into());
                }
                retain_partial(&target, &path, &operation.id)?;
                super::snapshot::restore(&snapshot_path, &target)?;
            }
        } else {
            super::snapshot::restore(&snapshot_path, &target)?;
        }
        let _new_writer = lock(&data_directory(&operation.next.data)?)?;
        state.phase = "copied".into();
        save(&path, &state)?;
        if !current.is_empty() {
            self.command(
                "podman",
                &["rm".into(), operation.expected_container.clone()],
                deadline,
            )?;
        }
        let created = self.create(&operation.next, deadline)?;
        let native = self.inspect(&operation.next, &created, deadline)?;
        if native.running || native.image != operation.next.image {
            return Err("restored application must be installed stopped".into());
        }
        state.phase = "installed".into();
        save(&path, &state)?;
        self.observe(&operation.next, native, deadline)
    }
}
