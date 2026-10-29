use std::time::{Duration, Instant};

use super::{
    sleep_until, unit_active_argv, unit_invocation_argv, LaunchProvider, MuseExecution, MuseHooks,
    MuseRuntime,
};
use crate::domain;
use crate::project::Executor;
use crate::terminal::{self, Binding};

/// Supervised guest unit of one recorded launch binding.
pub(in crate::muse) fn binding_unit(binding: &Binding) -> String {
    LaunchProvider::of_binding(binding)
        .unwrap_or_default()
        .unit(&binding.id)
}

impl<E: Executor, H: MuseHooks> MuseRuntime<E, H> {
    /// Shell end: retire one execution and return its custody, then release
    /// its retained bundle. A Codex credential rotates while it runs, so a
    /// delivered Codex run stops its unit and lets the broker capture the
    /// credential (`finish`) before its files retire; Muse retires locally
    /// first. Under uncertainty the bundle reference is kept.
    pub fn retire_execution(
        &self,
        execution: &MuseExecution,
        deadline: Instant,
    ) -> Result<(), String> {
        let binding = &execution.binding;
        let actor = execution.caller.actor;
        let lease = &execution.lease.id;
        let result =
            if execution.provider == LaunchProvider::Codex && !binding.invocation_id.is_empty() {
                let returned = self
                    .stop_unit(binding, deadline)
                    .and_then(|_| self.hooks.end(actor, lease, deadline));
                // The broker's stop retires the files; this confirms it
                // even when custody return failed.
                let retired = self.retire_execution_files(binding, deadline);
                returned.and(retired)
            } else {
                self.stop_execution(binding, actor, lease, true, deadline)
            };
        if result.is_ok() {
            super::release_provider_ref(
                execution.provider,
                &execution.provider_digest,
                &execution.provider_ref_token,
            );
        }
        result
    }

    /// `MuseRuntime.stopExecution`: stop the unit, verify its cgroup is
    /// gone, retire files, return custody.
    pub fn stop_execution(
        &self,
        binding: &Binding,
        actor: i64,
        lease_id: &str,
        return_custody: bool,
        deadline: Instant,
    ) -> Result<(), String> {
        self.stop_unit(binding, deadline)?;
        self.retire_execution_files(binding, deadline)?;
        if return_custody {
            return self.hooks.end(actor, lease_id, deadline);
        }
        Ok(())
    }

    /// Stop the unit and confirm it is inactive with its cgroup gone.
    fn stop_unit(&self, binding: &Binding, deadline: Instant) -> Result<(), String> {
        let container = binding.project.clone();
        let unit = binding_unit(binding);
        let stop_result = self.guest_refs(
            &container,
            &[],
            &["/usr/bin/systemctl", "stop", &unit],
            deadline,
        );
        let body = self
            .guest(&container, &[], &unit_active_argv(&unit), deadline)
            .map_err(|_| terminal::err_uncertain())?;
        if String::from_utf8_lossy(&body).trim() != "inactive" {
            return Err(terminal::err_uncertain());
        }
        if stop_result.is_err() {
            self.guest_refs(
                &container,
                &[],
                &[
                    "/usr/bin/test",
                    "!",
                    "-d",
                    &format!("/sys/fs/cgroup/system.slice/{unit}"),
                ],
                deadline,
            )
            .map_err(|_| terminal::err_uncertain())?;
        }
        Ok(())
    }

    /// `MuseRuntime.ValidateMuseBinding`: exact incarnation and active unit session.
    pub fn validate_muse_binding(
        &self,
        binding: &Binding,
        deadline: Instant,
    ) -> Result<(), String> {
        if binding.validate().is_err()
            || !terminal::valid_terminal_id(&binding.id)
            || !domain::valid_container_id(&binding.project)
            || !domain::valid_login(&binding.login)
        {
            return Err(terminal::err_denied());
        }
        if !terminal::valid_terminal_id(&binding.invocation_id) {
            return Err(terminal::err_stale());
        }
        let unit = binding_unit(binding);
        let invocation = self
            .guest(
                &binding.project,
                &[],
                &unit_invocation_argv(&unit),
                deadline,
            )
            .map_err(|_| terminal::err_stale())?;
        if String::from_utf8_lossy(&invocation).trim() != binding.invocation_id {
            return Err(terminal::err_stale());
        }
        let body = self
            .guest(&binding.project, &[], &unit_active_argv(&unit), deadline)
            .map_err(|_| terminal::err_stale())?;
        if String::from_utf8_lossy(&body).trim() != "active" {
            return Err(terminal::err_stale());
        }
        Ok(())
    }

    /// `MuseRuntime.awaitUnit`: active state within 5s.
    pub fn await_unit(&self, container: &str, unit: &str, deadline: Instant) -> Result<(), String> {
        let end = Instant::now() + Duration::from_secs(5);
        loop {
            if let Ok(body) = self.guest(container, &[], &unit_active_argv(unit), deadline) {
                if String::from_utf8_lossy(&body).trim() == "active" {
                    return Ok(());
                }
            }
            let now = Instant::now();
            if now >= deadline {
                return Err("context deadline exceeded".to_string());
            }
            if now >= end {
                return Err(terminal::err_denied());
            }
            if sleep_until(now + Duration::from_millis(50), deadline).is_err() {
                return Err("context deadline exceeded".to_string());
            }
        }
    }
}
