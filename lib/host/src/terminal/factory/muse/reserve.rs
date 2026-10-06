use super::commands::{muse_exec_argv, muse_setup_script};
use super::paths::{factory_muse_guest, factory_muse_paths, FactoryMusePaths};
use crate::project::Executor;
use crate::sha256;
use crate::terminal::factory::native::{
    factory_unit_name_or_denied, harness_install_script, reserve_run_argv,
};
use crate::terminal::factory::run::{FactoryRun, FACTORY_SCOPE_MUSE};
use crate::terminal::{self, Lease, Service, KIND_FACTORY};
use std::os::unix::fs::PermissionsExt;
use std::time::{Duration, Instant};

// ---------- service operations ----------

impl<E: Executor> Service<E> {
    /// Staged-host Muse harness check: regular exec-bit file whose digest
    /// matches the configured pin. Mirrors `verify_identity_harness`.
    pub fn verify_muse_harness(&self) -> Result<(), String> {
        let path = format!("{}/bin/muse", self.muse_harness.trim_end_matches('/'));
        let lstat = std::fs::symlink_metadata(&path)
            .map_err(|_| "verified Muse executable required".to_string())?;
        if !lstat.file_type().is_file() || lstat.permissions().mode() & 0o111 == 0 {
            return Err("verified Muse executable required".to_string());
        }
        let mut data = Vec::new();
        use std::io::Read;
        std::fs::File::open(&path)
            .map_err(|_| "verified Muse executable required".to_string())?
            .read_to_end(&mut data)
            .map_err(|e| format!("muse harness unreadable: {e}"))?;
        if sha256::hex_lower(&sha256::digest(&data)) != self.muse_harness_sha256 {
            return Err("muse harness digest differs".to_string());
        }
        Ok(())
    }

    /// `Service.FactoryCodexReserve` shape for Muse runs.
    pub fn factory_muse_reserve(
        &self,
        run: &FactoryRun,
        lease: &Lease,
        pin_sha256: &str,
        max_secs: i64,
        deadline: Instant,
    ) -> Result<(crate::terminal::Binding, FactoryMusePaths), String> {
        let p = factory_muse_paths(run)?;
        if lease.kind != KIND_FACTORY || lease.execution_id != run.id || lease.generation <= 0 {
            return Err(terminal::err_denied());
        }
        if pin_sha256.is_empty()
            || pin_sha256 != self.muse_harness_sha256
            || !self.muse_harness.starts_with('/')
        {
            return Err(terminal::err_denied());
        }
        if !(60..=3 * 3600).contains(&max_secs) {
            return Err(terminal::err_denied());
        }
        self.verify_muse_harness()?;
        let container = self.factory_project_container(&run.project, true, deadline)?;
        let (uid, gid) = self.factory_role_ids(&container, &run.role, deadline)?;
        self.factory_muse_setup(&container, run, &p, uid, gid, deadline)?;
        let guest = self.factory_muse_stage(&container, &run.harness_vers, deadline)?;
        let unit = factory_unit_name_or_denied(&run.id)?;
        let mut args = reserve_run_argv(&unit, max_secs);
        args.extend(muse_exec_argv(&container, run, &p, &guest));
        if self.factory_systemd_run(&args, deadline).is_err() {
            return Err("factory unit start unconfirmed".to_string());
        }
        match self.factory_active_invocation(&unit, Duration::from_secs(10), deadline) {
            Ok(invocation) => Ok((
                crate::terminal::Binding {
                    kind: KIND_FACTORY.to_string(),
                    id: run.id.clone(),
                    project: container,
                    login: run.role.clone(),
                    uid,
                    gid,
                    scope: FACTORY_SCOPE_MUSE.to_string(),
                    invocation_id: invocation,
                    credential_root: p.run_dir.clone(),
                    generation: lease.generation,
                    child_id: run.preparation.clone(),
                },
                p,
            )),
            Err(err) => {
                let _ = self.factory_systemctl(&["stop", &unit], deadline);
                Err(err)
            }
        }
    }

    fn factory_muse_setup(
        &self,
        container: &str,
        run: &FactoryRun,
        p: &FactoryMusePaths,
        uid: i64,
        gid: i64,
        deadline: Instant,
    ) -> Result<(), String> {
        let setup = muse_setup_script(p, uid, gid);
        let argv = vec![
            "--remote=false".to_string(),
            "exec".to_string(),
            container.to_string(),
            "/usr/bin/sh".to_string(),
            "-c".to_string(),
            setup,
        ];
        if self.run_podman(&[], &argv, deadline).is_err() {
            return Err("factory run directories unconfirmed".to_string());
        }
        let argv = vec![
            "--remote=false".to_string(),
            "exec".to_string(),
            "--user".to_string(),
            run.role.clone(),
            container.to_string(),
            "/usr/bin/git".to_string(),
            "-C".to_string(),
            p.checkout.clone(),
            "rev-parse".to_string(),
            "HEAD".to_string(),
        ];
        match self.run_podman(&[], &argv, deadline) {
            Ok(head) if head.len() <= 1024 => {
                if String::from_utf8_lossy(&head).trim() != run.source_commit {
                    return Err("factory source changed under preparation".to_string());
                }
            }
            _ => return Err("factory source unconfirmed".to_string()),
        }
        Ok(())
    }

    fn factory_muse_stage(
        &self,
        container: &str,
        version: &str,
        deadline: Instant,
    ) -> Result<String, String> {
        let guest = factory_muse_guest(version).ok_or_else(terminal::err_denied)?;
        let probe = vec![
            "--remote=false".to_string(),
            "exec".to_string(),
            container.to_string(),
            "/usr/bin/sha256sum".to_string(),
            guest.clone(),
        ];
        match self.run_podman(&[], &probe, deadline) {
            Err(_) => {
                let host_bin = terminal::clean_path(&format!("{}/bin/muse", self.muse_harness));
                let cp = vec![
                    "--remote=false".to_string(),
                    "cp".to_string(),
                    host_bin,
                    format!("{container}:{guest}.new"),
                ];
                if self.run_podman(&[], &cp, deadline).is_err() {
                    return Err("factory harness staging unconfirmed".to_string());
                }
                let install = vec![
                    "--remote=false".to_string(),
                    "exec".to_string(),
                    container.to_string(),
                    "/usr/bin/sh".to_string(),
                    "-c".to_string(),
                    harness_install_script(&guest),
                ];
                match self.run_podman(&[], &install, deadline) {
                    Ok(out) => {
                        let text = String::from_utf8_lossy(&out);
                        let fields: Vec<&str> = text.split_whitespace().collect();
                        if fields.len() != 2 || fields[0] != self.muse_harness_sha256 {
                            return Err("guest harness digest differs".to_string());
                        }
                    }
                    Err(_) => return Err("factory harness install unconfirmed".to_string()),
                }
            }
            Ok(out) => {
                let text = String::from_utf8_lossy(&out);
                let fields: Vec<&str> = text.split_whitespace().collect();
                if fields.len() != 2 || fields[0] != self.muse_harness_sha256 {
                    return Err("guest harness digest differs".to_string());
                }
            }
        }
        Ok(guest)
    }
}
