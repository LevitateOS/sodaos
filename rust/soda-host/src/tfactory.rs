//! Shared supervised-run lifecycle for factory CLI families.
//!
//! The codex and muse runners admit different providers, scopes and
//! guests, but drive the same boundary: unit wait plus exit/output
//! files, broker attestation, idempotent stop with pid-change
//! re-retire, bounded credential echo, live invocation match, and
//! windowed stdout slices. Those control flows live here, once, as
//! family-parameterized cores; `tcodex` and `tmuse` keep only their
//! admission policy (binding checks, path structs, staging) and
//! delegate. Every core preserves the exact podman/unit call sequence
//! the family FakeExec suites pin, so either side fails loudly on drift.

use crate::domain;
use crate::project::Executor;
use crate::tcodex;
use crate::texec::{self, Binding, Lease, Service, KIND_FACTORY};
use std::time::{Duration, Instant};

/// Family-neutral alias for the shared output-slice record.
pub type FactoryOutputSlice = tcodex::FactoryCodexOutputSlice;

/// Run-path resolver shape shared by `factory_run_paths` and
/// `factory_muse_run_paths`: validated identities to
/// `(checkout, run_dir, home, family_home)`.
pub type RunPathsFn = fn(&str, &str, &str) -> Option<(String, String, String, String)>;

/// Shared output-cursor gate: adapters run this before binding
/// attestation so a bad cursor never shells out.
pub fn check_output_range(offset: i64, limit: i64) -> Result<(), String> {
    if !(0..=tcodex::MAX_FACTORY_OUTPUT_OFFSET).contains(&offset)
        || !(1..=tcodex::MAX_FACTORY_OUTPUT_READ).contains(&limit)
    {
        return Err(texec::err_denied());
    }
    Ok(())
}

/// Shared `factory*Binding` checks: lease presence, provider, kind,
/// scope, run/container/invocation/preparation shape, derived run
/// paths, and the credential-root match. Family adapters add their own
/// login and generation policy around this (codex and muse differ
/// there by pre-existing design) and build their path structs from
/// the returned tuple.
pub fn checked_binding_paths(
    lease: &Lease,
    provider: &str,
    scope: &str,
    run_paths: RunPathsFn,
) -> Result<(String, String, String, String), String> {
    let Some(b) = &lease.binding else {
        return Err(texec::err_denied());
    };
    if lease.provider_id != provider || lease.kind != KIND_FACTORY {
        return Err(texec::err_denied());
    }
    if b.kind != KIND_FACTORY || b.scope != scope || !texec::valid_terminal_id(&b.id) {
        return Err(texec::err_denied());
    }
    if !domain::valid_container_id(&b.project) {
        return Err(texec::err_denied());
    }
    if !texec::valid_terminal_id(&b.invocation_id) || !tcodex::valid_preparation_id(&b.child_id) {
        return Err(texec::err_denied());
    }
    // Harness fields are not in the binding; paths need only role/prep/run.
    let paths = run_paths(&b.login, &b.child_id, &b.id).ok_or_else(texec::err_denied)?;
    if paths.0.is_empty() || texec::clean_path(&b.credential_root) != paths.1 {
        return Err(texec::err_denied());
    }
    Ok(paths)
}

impl<E: Executor> Service<E> {
    /// Shared wait core: unit quiescence, then the exit code and the
    /// head of the answer file. `project` is the recorded container,
    /// `run_dir`/`output` the derived run paths.
    pub(crate) fn factory_wait_result(
        &self,
        execution_id: &str,
        project: &str,
        run_dir: &str,
        output: &str,
        deadline: Instant,
    ) -> Result<(i32, String), String> {
        let unit = tcodex::factory_unit_name_or_denied(execution_id)?;
        loop {
            let show = self.factory_unit_state(&unit, deadline)?;
            if !show.active {
                break;
            }
            if tcodex::sleep_until(Instant::now() + Duration::from_millis(500), deadline).is_err() {
                return Err("context deadline exceeded".to_string());
            }
        }
        let mut exit = -1;
        let cat = vec![
            "--remote=false".to_string(),
            "exec".to_string(),
            project.to_string(),
            "/usr/bin/cat".to_string(),
            format!("{run_dir}/exit"),
        ];
        if let Ok(out) = self.run_podman(&[], &cat, deadline) {
            if let Some(code) = texec::parse_go_int(String::from_utf8_lossy(&out).trim()) {
                if (0..=255).contains(&code) {
                    exit = code as i32;
                }
            }
        }
        let mut answer = String::new();
        let head = vec![
            "--remote=false".to_string(),
            "exec".to_string(),
            project.to_string(),
            "/usr/bin/head".to_string(),
            "-c".to_string(),
            "65537".to_string(),
            output.to_string(),
        ];
        if let Ok(out) = self.run_podman(&[], &head, deadline) {
            answer = String::from_utf8_lossy(&out).into_owned();
        }
        Ok((exit, answer))
    }

    /// Shared validate core: attest the live supervised boundary
    /// before the broker releases credential bytes.
    pub(crate) fn factory_attest_live(
        &self,
        project_id: &str,
        binding: &Binding,
        execution_id: &str,
        deadline: Instant,
    ) -> Result<(), String> {
        let container = self.factory_project_container(project_id, true, deadline)?;
        if container != binding.project {
            return Err(texec::err_denied());
        }
        let (uid, gid) = self.factory_role_ids(&container, &binding.login, deadline)?;
        if uid != binding.uid || gid != binding.gid {
            return Err(texec::err_denied());
        }
        let unit = tcodex::factory_unit_name_or_denied(execution_id)?;
        let show = self
            .factory_unit_state(&unit, deadline)
            .map_err(|_| texec::err_denied())?;
        if !show.active || show.invocation != binding.invocation_id {
            return Err(texec::err_denied());
        }
        Ok(())
    }

    /// Shared stop core: retire the recorded unit and container
    /// process group. Idempotent.
    pub(crate) fn factory_stop_confirmed(
        &self,
        container: &str,
        execution_id: &str,
        pid_file: &str,
        run_dir: &str,
        deadline: Instant,
    ) -> Result<(), String> {
        let unit = tcodex::factory_unit_name_or_denied(execution_id)?;
        let before = self.factory_read_pid(container, pid_file, deadline);
        let _ = self.factory_systemctl(&["stop", &unit], deadline);
        self.factory_await_inactive(&unit, deadline)?;
        match self.factory_container_exists(container, deadline) {
            Ok(true) => {}
            Ok(false) => return Ok(()),
            Err(err) => return Err(err),
        }
        self.factory_retire(container, run_dir, deadline)?;
        let after = self.factory_read_pid(container, pid_file, deadline);
        if !after.is_empty() && after != before {
            self.factory_retire(container, run_dir, deadline)?;
            if self.factory_read_pid(container, pid_file, deadline) != after {
                return Err(texec::err_uncertain());
            }
        }
        Ok(())
    }

    /// Shared unbound-stop core: retire a run whose binding was never
    /// recorded.
    pub(crate) fn factory_stop_unbound_confirmed(
        &self,
        project: &str,
        execution_id: &str,
        pid_file: &str,
        run_dir: &str,
        deadline: Instant,
    ) -> Result<(), String> {
        let unit = tcodex::factory_unit_name_or_denied(execution_id)?;
        let _ = self.factory_systemctl(&["stop", &unit], deadline);
        self.factory_await_inactive(&unit, deadline)?;
        let container = match self.factory_project_container(project, false, deadline) {
            Ok(container) => container,
            Err(_) => match self.factory_container_exists(&format!("soda-{project}"), deadline) {
                Ok(false) => return Ok(()),
                Ok(true) => return Err(texec::err_uncertain()),
                Err(err) => return Err(err),
            },
        };
        let before = self.factory_read_pid(&container, pid_file, deadline);
        self.factory_retire(&container, run_dir, deadline)?;
        let after = self.factory_read_pid(&container, pid_file, deadline);
        if !after.is_empty() && after != before {
            self.factory_retire(&container, run_dir, deadline)?;
            if self.factory_read_pid(&container, pid_file, deadline) != after {
                return Err(texec::err_uncertain());
            }
        }
        Ok(())
    }

    /// Shared capture core: one bounded read of the staged credential
    /// file. Codex rotates on these bytes; muse only echoes them for
    /// the in-process return path (the broker ignores muse bytes).
    pub(crate) fn factory_capture_valid(
        &self,
        project: &str,
        path: &str,
        deadline: Instant,
    ) -> Result<Vec<u8>, String> {
        let argv = vec![
            "--remote=false".to_string(),
            "exec".to_string(),
            project.to_string(),
            "/usr/bin/head".to_string(),
            "-c".to_string(),
            "262145".to_string(),
            path.to_string(),
        ];
        match self.run_podman(&[], &argv, deadline) {
            Ok(out) if texec::credential_valid(&out) => Ok(out),
            _ => Err(texec::err_uncertain()),
        }
    }

    /// Shared live core: recorded unit currently active with the
    /// recorded invocation. Observation only.
    pub(crate) fn factory_live_scoped(
        &self,
        binding: &Binding,
        scope: &str,
        deadline: Instant,
    ) -> bool {
        if binding.kind != KIND_FACTORY
            || binding.scope != scope
            || !texec::valid_terminal_id(&binding.id)
            || !texec::valid_terminal_id(&binding.invocation_id)
        {
            return false;
        }
        let Some(unit) = tcodex::factory_unit_name(&binding.id) else {
            return false;
        };
        match self.factory_unit_state(&unit, deadline) {
            Ok(show) => show.active && show.invocation == binding.invocation_id,
            Err(_) => false,
        }
    }

    /// Shared output-binding core: scope/shape checks plus container
    /// attestation, returning the derived stdout path.
    pub(crate) fn factory_output_stdout(
        &self,
        project_id: &str,
        binding: &Binding,
        scope: &str,
        run_paths: RunPathsFn,
        deadline: Instant,
    ) -> Result<String, String> {
        if binding.kind != KIND_FACTORY
            || binding.scope != scope
            || !texec::valid_terminal_id(&binding.id)
        {
            return Err(texec::err_denied());
        }
        if !domain::valid_container_id(&binding.project)
            || !tcodex::valid_factory_role(&binding.login)
            || !texec::valid_terminal_id(&binding.invocation_id)
        {
            return Err(texec::err_denied());
        }
        if !tcodex::valid_preparation_id(&binding.child_id) {
            return Err(texec::err_denied());
        }
        let (checkout, run_dir, _, _) = run_paths(&binding.login, &binding.child_id, &binding.id)
            .ok_or_else(texec::err_denied)?;
        if checkout.is_empty() || texec::clean_path(&binding.credential_root) != run_dir {
            return Err(texec::err_denied());
        }
        let container = self.factory_project_container(project_id, true, deadline)?;
        if container != binding.project {
            return Err(texec::err_stale());
        }
        Ok(format!("{run_dir}/stdout.log"))
    }

    /// Shared output core: one bounded slice at a byte cursor. The
    /// cursor gate stays in the caller (`check_output_range` runs
    /// before binding attestation, so a bad cursor never shells out).
    pub(crate) fn factory_output_window(
        &self,
        project: &str,
        stdout: &str,
        offset: i64,
        limit: i64,
        deadline: Instant,
    ) -> Result<FactoryOutputSlice, String> {
        check_output_range(offset, limit)?;
        let mut total = 0i64;
        let stat = vec![
            "--remote=false".to_string(),
            "exec".to_string(),
            project.to_string(),
            "/usr/bin/stat".to_string(),
            "-c".to_string(),
            "%s".to_string(),
            stdout.to_string(),
        ];
        if let Ok(out) = self.run_podman(&[], &stat, deadline) {
            if let Some(size) = tcodex::factory_output_size(&out) {
                total = size;
            }
        }
        if offset > total {
            return Ok(FactoryOutputSlice {
                total,
                offset: total,
                gap: true,
                ..Default::default()
            });
        }
        let (mut start, mut truncated) = (offset, false);
        if start == 0 && total > tcodex::MAX_FACTORY_OUTPUT_WINDOW {
            start = total - tcodex::MAX_FACTORY_OUTPUT_WINDOW;
            truncated = true;
        }
        if start >= total {
            return Ok(FactoryOutputSlice {
                total,
                offset: start,
                truncated,
                ..Default::default()
            });
        }
        let read = vec![
            "--remote=false".to_string(),
            "exec".to_string(),
            project.to_string(),
            "/usr/bin/sh".to_string(),
            "-c".to_string(),
            tcodex::output_read_command(stdout, start, limit),
        ];
        let mut out = match self.run_podman(&[], &read, deadline) {
            Ok(out) => out,
            Err(_) => {
                return Ok(FactoryOutputSlice {
                    total,
                    offset: start,
                    truncated,
                    ..Default::default()
                });
            }
        };
        if out.len() as i64 > limit {
            out.truncate(limit as usize);
        }
        Ok(FactoryOutputSlice {
            data: out,
            total,
            offset: start,
            truncated,
            ..Default::default()
        })
    }
}
