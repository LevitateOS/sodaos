//! Exact execution inputs retained beside the private native run receipt.

use std::io::{Read, Write};
use std::os::unix::fs::{MetadataExt, OpenOptionsExt, PermissionsExt};

use crate::project::Executor;

use super::{ExecutionManifest, Factory, FactoryBroker, FactoryError, FactoryRun, FactoryTerminal};

impl<E: Executor, T: FactoryTerminal, B: FactoryBroker> Factory<E, T, B> {
    fn execution_manifest_path(&self, run: &FactoryRun) -> String {
        format!(
            "{}/{}-{}.manifest.json",
            self.state_dir, run.project, run.id
        )
    }

    pub(in crate::factory) fn store_execution_manifest(
        &self,
        run: &FactoryRun,
        bytes: &[u8],
    ) -> Result<(), FactoryError> {
        if bytes.is_empty() {
            return Ok(());
        }
        let manifest = ExecutionManifest::decode(bytes).map_err(FactoryError::msg)?;
        manifest
            .verify_for_run(run, bytes)
            .map_err(FactoryError::msg)?;
        let path = self.execution_manifest_path(run);
        match std::fs::symlink_metadata(&path) {
            Ok(_) => {
                if self.load_execution_manifest(run)? != bytes {
                    return Err(FactoryError::msg(
                        "execution manifest changed for recorded run",
                    ));
                }
                return Ok(());
            }
            Err(error) if error.kind() == std::io::ErrorKind::NotFound => {}
            Err(error) => return Err(FactoryError::msg(error.to_string())),
        }
        let temporary = format!("{path}.{}.new", std::process::id());
        match std::fs::OpenOptions::new()
            .write(true)
            .create_new(true)
            .custom_flags(libc::O_NOFOLLOW)
            .mode(0o600)
            .open(&temporary)
        {
            Ok(mut file) => {
                let written = (|| -> std::io::Result<()> {
                    file.write_all(bytes)?;
                    file.sync_all()?;
                    // The existing native run lock serializes this snapshot
                    // with every other launch for the same identity.
                    std::fs::rename(&temporary, &path)?;
                    std::fs::File::open(&self.state_dir)?.sync_all()
                })();
                if let Err(error) = written {
                    let _ = std::fs::remove_file(&temporary);
                    return Err(FactoryError::msg(error.to_string()));
                }
                Ok(())
            }
            Err(error) => Err(FactoryError::msg(error.to_string())),
        }
    }

    pub(in crate::factory) fn load_execution_manifest(
        &self,
        run: &FactoryRun,
    ) -> Result<Vec<u8>, FactoryError> {
        let file = std::fs::OpenOptions::new()
            .read(true)
            .custom_flags(libc::O_NOFOLLOW | libc::O_NONBLOCK)
            .open(self.execution_manifest_path(run))
            .map_err(|e| FactoryError::msg(e.to_string()))?;
        let metadata = file
            .metadata()
            .map_err(|e| FactoryError::msg(e.to_string()))?;
        if !metadata.is_file()
            || metadata.nlink() != 1
            || metadata.uid() != unsafe { libc::geteuid() }
            || metadata.permissions().mode() & 0o077 != 0
        {
            return Err(FactoryError::msg("unsafe execution-manifest state"));
        }
        let mut bytes = Vec::new();
        file.take(super::MAX_EXECUTION_MANIFEST as u64 + 1)
            .read_to_end(&mut bytes)
            .map_err(|e| FactoryError::msg(e.to_string()))?;
        let manifest = ExecutionManifest::decode(&bytes).map_err(FactoryError::msg)?;
        manifest
            .verify_for_run(run, &bytes)
            .map_err(FactoryError::msg)?;
        Ok(bytes)
    }
}
