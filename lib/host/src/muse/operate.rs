use std::time::Instant;

use super::{
    muse_delivery_valid, muse_project_credential_root, state_container, LaunchProvider, MuseHooks,
    MuseRuntime,
};
use crate::project::Executor;
use crate::terminal::{self, Binding, Delivery};

impl<E: Executor, H: MuseHooks> MuseRuntime<E, H> {
    /// `MuseRuntime.Muse`: broker-selected operations on recorded boundaries.
    pub fn muse_operation(
        &self,
        action: &str,
        delivery: &Delivery,
        deadline: Instant,
    ) -> Result<Delivery, String> {
        if !muse_delivery_valid(delivery) {
            return Err(terminal::err_denied());
        }
        let binding = delivery
            .lease
            .binding
            .as_ref()
            .ok_or_else(terminal::err_denied)?;
        match LaunchProvider::of_binding(binding) {
            Some(LaunchProvider::Codex) if action == "finish" => {
                let credential = self.capture_codex(delivery, deadline)?;
                let mut out = delivery.clone();
                out.credential = Some(credential);
                return Ok(out);
            }
            Some(_) => self.project_operation(action, delivery, deadline)?,
            None => return Err(terminal::err_denied()),
        }
        Ok(delivery.clone())
    }

    /// Broker `finish` for a Codex run: read back the credential Codex may
    /// have rotated while it ran.
    fn capture_codex(&self, delivery: &Delivery, deadline: Instant) -> Result<Vec<u8>, String> {
        let binding = delivery
            .lease
            .binding
            .as_ref()
            .ok_or_else(terminal::err_denied)?;
        if !muse_project_credential_root(binding) || !binding.child_id.is_empty() {
            return Err(terminal::err_denied());
        }
        let path = LaunchProvider::Codex.credential_file(&binding.credential_root);
        let body = self
            .guest_refs(
                &binding.project,
                &[],
                &["/usr/bin/cat", "--", &path],
                deadline,
            )
            .map_err(|_| terminal::err_uncertain())?;
        if !terminal::credential_valid(&body) {
            return Err(terminal::err_uncertain());
        }
        Ok(body)
    }

    fn project_operation(
        &self,
        action: &str,
        delivery: &Delivery,
        deadline: Instant,
    ) -> Result<(), String> {
        let binding = delivery
            .lease
            .binding
            .as_ref()
            .ok_or_else(terminal::err_denied)?;
        if !muse_project_credential_root(binding) {
            return Err(terminal::err_denied());
        }
        if action == "validate" {
            return self.validate_muse_binding(binding, deadline);
        }
        if action != "stop" {
            return Err(terminal::err_denied());
        }
        // This route only sees `&Binding`: no digest, no reference token.
        // It takes no store-reference action here; the execution owner
        // (shell end / `finish_start` / `denied_after_cleanup`) releases
        // its retained reference after it confirms retirement.
        // A normal shell exit retires locally before returning its broker
        // lease. The broker then confirms retirement through this same stop
        // route. A collected inactive unit is no longer an active binding;
        // accept it only when its exact cgroup and credential root are gone.
        if self.muse_execution_retired(binding, deadline) {
            return Ok(());
        }
        // A Codex run whose process already ended stops after its capture:
        // retire the exact finished incarnation's files.
        if LaunchProvider::of_binding(binding) == Some(LaunchProvider::Codex)
            && self.unit_finished(binding, deadline)
        {
            return self.retire_execution_files(binding, deadline);
        }
        self.validate_muse_binding(binding, deadline)?;
        self.stop_execution(
            binding,
            delivery.lease.actor_id,
            &delivery.lease.id,
            false,
            deadline,
        )
    }

    fn muse_execution_retired(&self, binding: &Binding, deadline: Instant) -> bool {
        self.unit_finished(binding, deadline)
            && self
                .guest_refs(
                    &binding.project,
                    &[],
                    &["/usr/bin/test", "!", "-e", &binding.credential_root],
                    deadline,
                )
                .is_ok()
    }

    /// The exact incarnation's unit is inactive (or collected) with its
    /// cgroup gone.
    fn unit_finished(&self, binding: &Binding, deadline: Instant) -> bool {
        if binding.validate().is_err() {
            return false;
        }
        let unit = super::stop::binding_unit(binding);
        let inactive = self
            .guest(
                &binding.project,
                &[],
                &super::unit_active_argv(&unit),
                deadline,
            )
            .is_ok_and(|state| state.len() <= 32 && state.trim_ascii() == b"inactive");
        if !inactive {
            return false;
        }
        let same_or_collected = self
            .guest(
                &binding.project,
                &[],
                &super::unit_invocation_argv(&unit),
                deadline,
            )
            .is_ok_and(|invocation| {
                invocation.len() <= 128
                    && (invocation.trim_ascii().is_empty()
                        || invocation.trim_ascii() == binding.invocation_id.as_bytes())
            });
        same_or_collected
            && self
                .guest_refs(
                    &binding.project,
                    &[],
                    &[
                        "/usr/bin/test",
                        "!",
                        "-d",
                        &format!("/sys/fs/cgroup/system.slice/{unit}"),
                    ],
                    deadline,
                )
                .is_ok()
    }

    // ---------- state cleanup (muse_state_linux.go) ----------

    /// `MuseRuntime.cleanupExecutionState`: remove the guest state dir.
    pub fn cleanup_execution_state(
        &self,
        binding: &Binding,
        deadline: Instant,
    ) -> Result<(), String> {
        let container = state_container(binding)?;
        if container.is_empty() {
            return Err(terminal::err_denied());
        }
        match self.invoke_state(
            binding,
            &[
                "container".to_string(),
                "exists".to_string(),
                container.clone(),
            ],
            deadline,
        ) {
            Ok(_) => {}
            Err(err) if terminal::exit_code_of(&err) == Some(1) => return Ok(()),
            Err(_) => return Err(terminal::err_uncertain()),
        }
        let state = format!("/tmp/soda-muse-state-{}", binding.id);
        let owner = format!("--user={}:{}", binding.uid, binding.gid);
        self.invoke_state(
            binding,
            &[
                "exec".to_string(),
                owner.clone(),
                container.clone(),
                "/usr/bin/rm".to_string(),
                "--recursive".to_string(),
                "--force".to_string(),
                "--".to_string(),
                state.clone(),
            ],
            deadline,
        )
        .map_err(|_| terminal::err_uncertain())?;
        self.invoke_state(
            binding,
            &[
                "exec".to_string(),
                owner,
                container,
                "/usr/bin/test".to_string(),
                "!".to_string(),
                "-e".to_string(),
                state,
            ],
            deadline,
        )
        .map(|_| ())
        .map_err(|_| terminal::err_uncertain())
    }

    fn invoke_state(
        &self,
        binding: &Binding,
        args: &[String],
        deadline: Instant,
    ) -> Result<Vec<u8>, String> {
        if binding.scope == "muse-project" && !binding.child_id.is_empty() {
            let mut full = vec!["/usr/bin/podman".to_string(), "--remote=false".to_string()];
            full.extend(args.iter().cloned());
            return self.guest(&binding.project, &[], &full, deadline);
        }
        self.podman(&[], args, deadline)
    }

    fn retire_mount(&self, container: &str, path: &str, deadline: Instant) -> Result<(), String> {
        if self
            .guest_refs(container, &[], &["/usr/bin/umount", path], deadline)
            .is_ok()
        {
            return Ok(());
        }
        if self
            .guest_refs(
                container,
                &[],
                &["/usr/bin/test", "!", "-e", path],
                deadline,
            )
            .is_ok()
        {
            return Ok(());
        }
        match self.guest_refs(
            container,
            &[],
            &["/usr/bin/mountpoint", "--quiet", path],
            deadline,
        ) {
            Err(err) if terminal::exit_code_of(&err) == Some(32) => Ok(()),
            _ => Err(terminal::err_uncertain()),
        }
    }

    pub(in crate::muse) fn retire_execution_files(
        &self,
        binding: &Binding,
        deadline: Instant,
    ) -> Result<(), String> {
        self.cleanup_execution_state(binding, deadline)?;
        if !binding.credential_root.contains("/nested/") {
            self.retire_mount(&binding.project, &binding.credential_root, deadline)?;
        }
        self.guest_refs(
            &binding.project,
            &[],
            &[
                "/usr/bin/rm",
                "--recursive",
                "--force",
                "--",
                &binding.credential_root,
            ],
            deadline,
        )
        .map(|_| ())
        .map_err(|_| terminal::err_uncertain())
    }
}
