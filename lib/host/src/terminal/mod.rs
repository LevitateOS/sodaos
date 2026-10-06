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

pub use self::stream::TerminalStart;

mod broker;

pub use self::broker::{
    identity_action, identity_launch, identity_route, rand_id, valid_identity_request,
    IdentityBroker, IdentityRoute,
};

#[cfg(test)]
mod tests;
