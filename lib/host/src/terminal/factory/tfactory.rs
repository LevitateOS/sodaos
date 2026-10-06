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

use crate::project::Executor;
use crate::terminal::factory::tcodex;
use crate::terminal::{self, Binding, Service, KIND_FACTORY};
use std::time::{Duration, Instant};

pub use super::binding::{checked_binding_paths, RunPathsFn};
pub use super::output::{check_output_range, FactoryOutputSlice};

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
            if let Some(code) = terminal::parse_go_int(String::from_utf8_lossy(&out).trim()) {
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
            return Err(terminal::err_denied());
        }
        let (uid, gid) = self.factory_role_ids(&container, &binding.login, deadline)?;
        if uid != binding.uid || gid != binding.gid {
            return Err(terminal::err_denied());
        }
        let unit = tcodex::factory_unit_name_or_denied(execution_id)?;
        let show = self
            .factory_unit_state(&unit, deadline)
            .map_err(|_| terminal::err_denied())?;
        if !show.active || show.invocation != binding.invocation_id {
            return Err(terminal::err_denied());
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
                return Err(terminal::err_uncertain());
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
                Ok(true) => return Err(terminal::err_uncertain()),
                Err(err) => return Err(err),
            },
        };
        let before = self.factory_read_pid(&container, pid_file, deadline);
        self.factory_retire(&container, run_dir, deadline)?;
        let after = self.factory_read_pid(&container, pid_file, deadline);
        if !after.is_empty() && after != before {
            self.factory_retire(&container, run_dir, deadline)?;
            if self.factory_read_pid(&container, pid_file, deadline) != after {
                return Err(terminal::err_uncertain());
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
            Ok(out) if terminal::credential_valid(&out) => Ok(out),
            _ => Err(terminal::err_uncertain()),
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
            || !terminal::valid_terminal_id(&binding.id)
            || !terminal::valid_terminal_id(&binding.invocation_id)
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
}
