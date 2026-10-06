use std::time::Instant;

use super::{
    muse_delivery_valid, muse_project_credential_root, unit_invocation_argv, MuseHooks, MuseRuntime,
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
        match binding.scope.as_str() {
            "muse-project" => self.project_operation(action, delivery, deadline)?,
            _ => return Err(terminal::err_denied()),
        }
        Ok(delivery.clone())
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
        let unit = format!("soda-muse-{}.service", binding.id);
        if !muse_project_credential_root(binding) {
            return Err(terminal::err_denied());
        }
        if let Ok(body) = self.guest(
            &binding.project,
            &[],
            &unit_invocation_argv(&unit),
            deadline,
        ) {
            let current = String::from_utf8_lossy(&body).trim().to_string();
            if !current.is_empty() && current != binding.invocation_id {
                return Err(terminal::err_stale());
            }
        }
        if action == "validate" {
            return self.validate_muse_binding(binding, deadline);
        }
        if action != "stop" {
            return Err(terminal::err_denied());
        }
        self.stop_execution(
            binding,
            delivery.lease.actor_id,
            &delivery.lease.id,
            false,
            deadline,
        )
    }
}
