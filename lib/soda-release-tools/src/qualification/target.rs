//! One protected collector caller per actual journey target. Claims remain
//! immutable so a lost controller response cannot authorize a second worker.
use super::{evidence, hash, input, read, retained_request, write_new, CollectionBinding};
use serde::{Deserialize, Serialize};
use sha2::{Digest, Sha256};
use std::{
    fs,
    os::{
        fd::AsRawFd,
        unix::fs::{DirBuilderExt, OpenOptionsExt},
    },
    path::{Path, PathBuf},
};

const PARENT: &str = "/home/.soda-qualification-targets";

#[derive(Serialize, Deserialize)]
#[serde(deny_unknown_fields)]
struct Claim {
    format: u8,
    output: PathBuf,
    binding: CollectionBinding,
}

pub(super) struct Target {
    directory: PathBuf,
    _lock: fs::File,
}

fn directory(path: &Path) -> Result<(), String> {
    match fs::DirBuilder::new().mode(0o700).create(path) {
        Ok(()) => {
            fs::File::open(path.parent().ok_or("target lock parent required")?)
                .and_then(|f| f.sync_all())
                .map_err(|e| e.to_string())?;
        }
        Err(e) if e.kind() == std::io::ErrorKind::AlreadyExists => (),
        Err(e) => return Err(e.to_string()),
    }
    evidence::protected_path(path, true).map_err(|e| e.to_string())
}

impl Target {
    pub(super) fn acquire(resolved: &input::FileReference) -> Result<Self, String> {
        evidence::protected_path(&resolved.path, false).map_err(|e| e.to_string())?;
        if hash(&resolved.path)? != resolved.sha256 {
            return Err("resolved journey target changed".into());
        }
        let value: serde_json::Value = read(&resolved.path)?;
        let distribution = value
            .pointer("/scenario/target/distribution")
            .and_then(serde_json::Value::as_str)
            .ok_or("journey target distribution required")?;
        let uuid = value
            .pointer("/scenario/target/uuid")
            .and_then(serde_json::Value::as_str)
            .ok_or("journey target UUID required")?;
        if !matches!(distribution, "docker" | "appliance")
            || uuid.len() != 36
            || uuid.bytes().enumerate().any(|(i, b)| {
                if matches!(i, 8 | 13 | 18 | 23) {
                    b != b'-'
                } else {
                    !b.is_ascii_hexdigit() || b.is_ascii_uppercase()
                }
            })
        {
            return Err("canonical journey target identity required".into());
        }
        let parent = Path::new(PARENT);
        directory(parent)?;
        let directory_path = parent.join(format!(
            "{:x}",
            Sha256::digest(format!("{distribution}:{uuid}"))
        ));
        directory(&directory_path)?;
        let path = directory_path.join("controller.lock");
        let lock = fs::OpenOptions::new()
            .read(true)
            .write(true)
            .create(true)
            .mode(0o600)
            .custom_flags(libc::O_NOFOLLOW | libc::O_CLOEXEC)
            .open(&path)
            .map_err(|e| e.to_string())?;
        evidence::protected_path(&path, false).map_err(|e| e.to_string())?;
        if unsafe { libc::flock(lock.as_raw_fd(), libc::LOCK_EX | libc::LOCK_NB) } != 0 {
            return Err("journey target collector already active".into());
        }
        Ok(Self {
            directory: directory_path,
            _lock: lock,
        })
    }

    pub(super) fn check(&self, out: &Path) -> Result<(), String> {
        for entry in fs::read_dir(&self.directory).map_err(|e| e.to_string())? {
            let path = entry.map_err(|e| e.to_string())?.path();
            if path.file_name().and_then(|n| n.to_str()) == Some("controller.lock") {
                continue;
            }
            evidence::protected_path(&path, false).map_err(|e| e.to_string())?;
            let claim: Claim = read(&path)?;
            if claim.format != 1 || claim.binding.request.path != claim.output.join("request.json")
            {
                return Err("retained journey target claim invalid".into());
            }
            if hash(&claim.binding.request.path)? != claim.binding.request.sha256 {
                return Err("retained target request changed".into());
            }
            let (request, _) = retained_request(&claim.output)?;
            if claim.binding.unit != request.unit {
                return Err("retained target unit changed".into());
            }
            if claim.output == out {
                continue;
            }
            // Only the existing canonical native-retirement receipt releases
            // this target for a different output; no timeout or PID guesses.
            evidence::protected_path(&claim.output.join("executed.json"), false)
                .map_err(|_| "prior journey target operation requires reconciliation")?;
            let actual: evidence::ProtectedEvidence = read(&claim.output.join("executed.json"))?;
            let mut expected = request.binding;
            expected.exit_code = 0;
            expected.native_retirement = "confirmed".into();
            expected.files = actual.files.clone();
            if actual != expected || actual.files.is_empty() {
                return Err("prior journey target operation requires reconciliation".into());
            }
        }
        Ok(())
    }

    pub(super) fn claim(&self, out: &Path, binding: &CollectionBinding) -> Result<(), String> {
        let path = self.directory.join(format!(
            "claim-{:x}.json",
            Sha256::digest(out.as_os_str().as_encoded_bytes())
        ));
        let claim = Claim {
            format: 1,
            output: out.to_owned(),
            binding: CollectionBinding {
                request: binding.request.clone(),
                unit: binding.unit.clone(),
            },
        };
        write_new(&path, &claim)
    }
}
