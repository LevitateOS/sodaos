//! Host terminal executor: service, native attach, identity broker calls.
//!
//! Port of `internal/host/terminal/service.go` (execution core), `native.go`,
//! `identity.go`, `identity_transfer.go`, the agent-hash helper from
//! `agent.go`, the admission predicates from `types.go`, and the daemon
//! identity dispatch logic from `internal/host/identity.go`.
//!
//! Out of scope (stay in Go): the websocket transport (`Write`, pumps,
//! `Handler`), `AgentExec` command construction for the Go client, and the
//! HTTP route shells (plain `pub` fns are exposed instead).
//!
//! Behavioral notes:
//!
//! * Podman argv, JSON wire bytes, the terminal frame protocol, timeouts and
//!   error strings match the Go implementation. Exit-code checks (`container
//!   exists`, `mountpoint`) parse the `exit status N` text the crate
//!   [`Executor`](crate::project::Executor) surface produces, since the
//!   trait is stringly typed.
//! * `stream_identity_harness` buffers the host tar producer through the
//!   executor instead of streaming pipe-to-pipe; error strings are unchanged.

use std::time::Instant;

use crate::project::Executor;
use crate::sha256;

pub mod factory;

mod core;

pub use self::core::{
    err_denied, err_stale, err_uncertain, now_unix, valid_terminal_id, BROKER_RESPONSE_LIMIT,
    CREDENTIAL_LIMIT, ERR_DENIED, ERR_IDENTITY_UNCONFIRMED, ERR_INVALID_IDENTITY_OPERATION,
    ERR_NOT_FOUND, ERR_STALE, ERR_UNCERTAIN, FRAME_LIMIT, IDENTITY_LAUNCH_PATH, KIND_FACTORY,
    KIND_TERMINAL, PROVIDER_CODEX, PROVIDER_MUSE, SCOPE_MUSE_PROJECT, STREAM_LIMIT, TERMINAL_LIMIT,
};

mod text;

pub(in crate::terminal) use self::text::contains_crlf;
pub use self::text::{strict_b64_decode, terminal_dimensions, valid_terminal_name};

mod wire;

pub use self::wire::{credential_valid, json_valid, TerminalFrame, TerminalRequest, TerminalState};

mod time;

pub use self::time::parse_rfc3339;

mod lease;

pub use self::lease::{parse_string_i64, AcquireRequest, Binding, Delivery, Lease};

mod inspect;

pub use self::inspect::{
    exit_code_of, terminal_id_map, terminal_isolation, terminal_target_ready, TerminalInspection,
    TERMINAL_INSPECT,
};
pub(crate) use self::inspect::{parse_go_int, parse_go_uint};

mod agent;

pub use self::agent::{agent_argv, container_exists_argv, inspect_argv, EndIdentityHook, Service};

mod identity;

pub use self::identity::{identity_result, terminal_lease, IdentityRequest};

mod transfer;

pub use self::transfer::{tar_consumer_argv, tar_producer_argv};

mod native;

pub use self::native::{
    agent_program_hash, agent_program_path, clean_path, native_argv, parse_output_line,
    valid_private_terminal_request, NativeAttach,
};

mod stream;

pub use self::stream::{StreamTable, TerminalStart};

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
    let mut bytes = [0u8; 16];
    let mut filled = 0;
    while filled < bytes.len() {
        let n = unsafe {
            libc::getrandom(
                bytes[filled..].as_mut_ptr() as *mut libc::c_void,
                bytes.len() - filled,
                0,
            )
        };
        if n < 0 {
            let errno = std::io::Error::last_os_error().raw_os_error().unwrap_or(0);
            if errno == libc::EINTR {
                continue;
            }
            return Err(format!("kernel randomness unavailable: errno {errno}"));
        }
        filled += n as usize;
    }
    Ok(sha256::hex_lower(&bytes))
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

#[cfg(test)]
mod tests;
