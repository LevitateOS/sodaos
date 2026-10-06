use super::run::FACTORY_SCOPE_MUSE;
use super::tcodex;
use crate::domain;
use crate::project::Executor;
use crate::terminal::{self, Delivery, Lease, Service, KIND_FACTORY};
use std::time::Instant;

/// Run-path resolver shape shared by `factory_run_paths` and
/// `factory_muse_run_paths`: validated identities to
/// `(checkout, run_dir, home, family_home)`.
pub type RunPathsFn = fn(&str, &str, &str) -> Option<(String, String, String, String)>;

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
        return Err(terminal::err_denied());
    };
    if lease.provider_id != provider || lease.kind != KIND_FACTORY {
        return Err(terminal::err_denied());
    }
    if b.kind != KIND_FACTORY || b.scope != scope || !terminal::valid_terminal_id(&b.id) {
        return Err(terminal::err_denied());
    }
    if !domain::valid_container_id(&b.project) {
        return Err(terminal::err_denied());
    }
    if !terminal::valid_terminal_id(&b.invocation_id) || !tcodex::valid_preparation_id(&b.child_id)
    {
        return Err(terminal::err_denied());
    }
    // Harness fields are not in the binding; paths need only role/prep/run.
    let paths = run_paths(&b.login, &b.child_id, &b.id).ok_or_else(terminal::err_denied)?;
    if paths.0.is_empty() || terminal::clean_path(&b.credential_root) != paths.1 {
        return Err(terminal::err_denied());
    }
    Ok(paths)
}

impl<E: Executor> Service<E> {
    /// `Daemon.factoryIdentityOperation`: broker validate/stop/finish
    /// callbacks for supervised factory runs.
    pub fn factory_identity_operation(
        &self,
        action: &str,
        delivery: &Delivery,
        deadline: Instant,
    ) -> Result<Delivery, String> {
        let mut out = delivery.clone();
        out.credential = None;
        let muse = delivery
            .lease
            .binding
            .as_ref()
            .is_some_and(|b| b.scope == FACTORY_SCOPE_MUSE);
        match action {
            "validate" if muse => self.factory_muse_validate(&delivery.lease, deadline)?,
            "validate" => self.factory_codex_validate(&delivery.lease, deadline)?,
            "stop" if muse => self.factory_muse_stop(&delivery.lease, deadline)?,
            "stop" => self.factory_codex_stop(&delivery.lease, deadline)?,
            // Muse borrows: the broker forgets the lease on return and
            // never calls finish (same denial as the interactive muse
            // runtime, which allows only validate and stop).
            "finish" if muse => return Err(terminal::err_denied()),
            "finish" => {
                out.credential = Some(self.factory_codex_finish(&delivery.lease, deadline)?);
            }
            _ => return Err(terminal::err_denied()),
        }
        Ok(out)
    }
}
