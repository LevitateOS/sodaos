use std::fs::{self, File, OpenOptions};
use std::io::{Read, Write};
use std::os::unix::fs::OpenOptionsExt;

use crate::tailnet_domain::{
    valid_container_id, ERR_CONFLICT, ERR_INVALID, ERR_UNAVAILABLE, ERR_UNCONFIRMED,
};
use crate::tailnet_runtime::{
    companion_create_args, decode_project_run, marshal_project_run, ProjectRun,
};

use super::{
    chown_path, mkdir_700, owned_run_dir, root_directory, runtime_file, Root, RunFiles,
    RUNTIME_RECORD_NOT_FOUND,
};

fn prepare_run_subdir(
    root: &Root,
    path: &str,
    run: &ProjectRun,
    fresh: bool,
) -> Result<(), String> {
    if fresh {
        match mkdir_700(&root.join(path)) {
            Ok(()) => {
                if chown_path(&root.join(path), run.uid, run.gid).is_err() {
                    return Err(ERR_UNAVAILABLE.to_string());
                }
            }
            Err(_) => return Err(ERR_UNAVAILABLE.to_string()),
        }
    }
    let info = root.lstat(path).map_err(|_| ERR_UNAVAILABLE.to_string())?;
    if !owned_run_dir(&info, run.uid, run.gid) {
        return Err(ERR_UNAVAILABLE.to_string());
    }
    Ok(())
}

impl RunFiles {
    /// Read the retained runtime record. A missing `current.json` returns
    /// exactly `"runtime record not found"` so callers can distinguish "no
    /// record yet" from confirmed failures, mirroring Go's raw `os.ErrNotExist`.
    pub fn current(&self) -> Result<ProjectRun, String> {
        let mut opts = OpenOptions::new();
        opts.read(true).custom_flags(libc::O_NOFOLLOW);
        let mut file = match opts.open(self.root.join("current.json")) {
            Ok(f) => f,
            Err(e) if e.kind() == std::io::ErrorKind::NotFound => {
                return Err(RUNTIME_RECORD_NOT_FOUND.to_string())
            }
            Err(e) => return Err(e.to_string()),
        };
        let info = file.metadata().map_err(|_| ERR_UNAVAILABLE.to_string())?;
        if !runtime_file(&info, self.uid, self.gid, 0o600) || info.len() > 8192 {
            return Err(ERR_UNAVAILABLE.to_string());
        }
        let mut buf = Vec::new();
        file.read_to_end(&mut buf)
            .map_err(|_| ERR_UNAVAILABLE.to_string())?;
        let run = decode_project_run(&buf).map_err(|_| ERR_UNAVAILABLE.to_string())?;
        // The recipe validates all path-bearing identity fields. The caller
        // separately checks this record's project, parent CID and native
        // namespace incarnation.
        companion_create_args(&run, &format!("sha256:{}", "0".repeat(64)))?;
        Ok(run)
    }

    /// Durably install `run` as the current record via temp-file + rename.
    pub fn save_current(&self, run: &ProjectRun) -> Result<(), String> {
        let name = format!("current-{}.json", run.target.run);
        let b = marshal_project_run(run);
        let mut opts = OpenOptions::new();
        opts.write(true)
            .create_new(true)
            .mode(0o600)
            .custom_flags(libc::O_NOFOLLOW);
        let mut file = opts
            .open(self.root.join(&name))
            .map_err(|_| ERR_CONFLICT.to_string())?;
        if file.write_all(b.as_bytes()).is_err() {
            return Err(ERR_UNCONFIRMED.to_string());
        }
        if file.sync_all().is_err()
            || fs::rename(self.root.join(&name), self.root.join("current.json")).is_err()
        {
            return Err(ERR_UNCONFIRMED.to_string());
        }
        let dir = File::open(self.root.join(".")).map_err(|_| ERR_UNCONFIRMED.to_string())?;
        if dir.sync_all().is_err() {
            return Err(ERR_UNCONFIRMED.to_string());
        }
        Ok(())
    }

    /// Prepare (fresh: exclusively create) the run workspace `control`/`input`
    /// subdirectories. A pre-existing directory on a fresh prepare is an
    /// incomplete attempt, never an empty workspace.
    pub fn prepare(&self, run: &ProjectRun, fresh: bool) -> Result<Root, String> {
        let name = &run.target.run;
        if !valid_container_id(name) || run.uid == 0 || run.gid == 0 {
            return Err(ERR_INVALID.to_string());
        }
        if fresh && mkdir_700(&self.root.join(name)).is_err() {
            return Err(ERR_CONFLICT.to_string());
        }
        let info = self
            .root
            .lstat(name)
            .map_err(|_| ERR_UNAVAILABLE.to_string())?;
        if !root_directory(&info, self.uid) {
            return Err(ERR_UNAVAILABLE.to_string());
        }
        let root = Root {
            path: self.root.join(name),
        };
        for path in ["control", "input"] {
            prepare_run_subdir(&root, path, run, fresh)?;
        }
        Ok(root)
    }
}
