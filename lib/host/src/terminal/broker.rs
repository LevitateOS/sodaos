use std::time::Instant;

use crate::project::Executor;
use crate::sha256;

use super::{
    err_denied, now_unix, AcquireRequest, Binding, Delivery, Lease, Service, TerminalStart,
    IDENTITY_LAUNCH_PATH, KIND_FACTORY, KIND_TERMINAL, PROVIDER_CODEX, PROVIDER_MUSE,
    SCOPE_MUSE_PROJECT,
};

/// `validIdentityRequest` over pre-parsed request fields.
pub fn valid_identity_request(
    method: &str,
    raw_query: &str,
    force_query: bool,
    raw_path: &str,
    origin_count: usize,
) -> bool {
    method == "POST"
        && raw_query.is_empty()
        && !force_query
        && raw_path.is_empty()
        && origin_count == 0
}

/// Broker custody surface the identity routes need.
pub trait IdentityBroker {
    fn acquire(&self, req: &AcquireRequest, deadline: Instant) -> Result<Lease, String>;
    fn register(
        &self,
        lease_id: &str,
        binding: &Binding,
        deadline: Instant,
    ) -> Result<Delivery, String>;
    fn reconcile_lease(&self, lease_id: &str) -> Result<(), String>;
}

/// 16 bytes of kernel randomness, hex-encoded (`museID` / launch IDs).
pub fn rand_id() -> Result<String, String> {
    rand_id_with(|bytes| getrandom::fill(bytes).map_err(std::io::Error::other))
}

fn rand_id_with(fill: impl FnOnce(&mut [u8]) -> std::io::Result<()>) -> Result<String, String> {
    let mut bytes = [0u8; 16];
    fill(&mut bytes).map_err(|_| "kernel randomness unavailable".to_owned())?;
    Ok(sha256::hex_lower(&bytes))
}

#[cfg(test)]
mod entropy_tests {
    use super::*;

    #[test]
    fn id_generation_rejects_partial_entropy_output() {
        let result = rand_id_with(|bytes| {
            bytes[0] = 1;
            Err(std::io::Error::other("injected entropy failure"))
        });
        assert_eq!(result.unwrap_err(), "kernel randomness unavailable");
    }
}

/// `Daemon.identityLaunch`: acquire, prepare, register, start. Every
/// failure after acquire reconciles the lease (reconcile errors ignored).
pub fn identity_launch<B: IdentityBroker, E: Executor>(
    broker: &B,
    service: &Service<E>,
    input: &TerminalStart,
    codex_harness_configured: bool,
    deadline: Instant,
) -> Result<Lease, String> {
    if !codex_harness_configured {
        return Err(err_denied());
    }
    let execution_id = rand_id()?;
    let now = now_unix();
    let lease = broker.acquire(
        &AcquireRequest {
            provider_id: PROVIDER_CODEX.to_string(),
            actor_id: input.actor_id,
            connection_id: input.connection_id.clone(),
            project_id: input.project_id.clone(),
            execution_id,
            kind: KIND_TERMINAL.to_string(),
            deadline_secs: now + 12 * 3600,
            ..Default::default()
        },
        deadline,
    )?;
    let binding = match service.prepare_identity(
        &lease,
        &input.login,
        &input.scope,
        input.cols,
        input.rows,
        deadline,
    ) {
        Ok(binding) => binding,
        Err(err) => {
            let _ = broker.reconcile_lease(&lease.id);
            return Err(err);
        }
    };
    let mut delivery = match broker.register(&lease.id, &binding, deadline) {
        Ok(delivery) => delivery,
        Err(err) => {
            let _ = broker.reconcile_lease(&lease.id);
            return Err(err);
        }
    };
    let outcome = service.identity("start", &delivery, deadline);
    // Zero the credential copy however the call ends.
    if let Some(credential) = delivery.credential.as_mut() {
        for byte in credential.iter_mut() {
            *byte = 0;
        }
    }
    if let Err(err) = outcome {
        let _ = broker.reconcile_lease(&lease.id);
        return Err(err);
    }
    Ok(delivery.lease)
}

/// `Daemon.identityOperation` dispatch target.
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub enum IdentityRoute {
    Launch,
    Factory,
    Muse,
    Terminal,
}

/// Classify one identity operation: launch path, factory kind, live
/// muse-project binding, or plain terminal operation.
pub fn identity_route(path: &str, lease: &Lease, muse_available: bool) -> IdentityRoute {
    if path == IDENTITY_LAUNCH_PATH {
        return IdentityRoute::Launch;
    }
    if lease.kind == KIND_FACTORY {
        return IdentityRoute::Factory;
    }
    let muse_bound = lease.provider_id == PROVIDER_MUSE
        && lease
            .binding
            .as_ref()
            .map(|b| b.scope == SCOPE_MUSE_PROJECT)
            .unwrap_or(false);
    if muse_bound && muse_available {
        IdentityRoute::Muse
    } else {
        IdentityRoute::Terminal
    }
}

/// Operation word carried after `/identity/` (`validate`, `stop`, ...).
pub fn identity_action(path: &str) -> &str {
    path.strip_prefix("/identity/").unwrap_or(path)
}
