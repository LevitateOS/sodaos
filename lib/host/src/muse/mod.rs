//! Native Muse execution runtime and launch socket.
//!
//! Port of `internal/host/terminal/muse_types.go`,
//! `muse_linux.go`, `muse_execution_linux.go`, `muse_socket_linux.go`,
//! `muse_state_linux.go`, `muse_binary_linux.go`, the launch policy from
//! `internal/identity/launch.go` + `selection.go`, and the daemon Muse
//! wiring from `internal/host/muse.go` (authorization/selection over
//! broker connections plus launch-listener setup).
//!
//! Broker custody callbacks arrive through [`MuseHooks`]; unlike Go's
//! nullable func fields the hooks are mandatory, which collapses the
//! nil-hook denials into the type system (the daemon always wires all
//! six). `MuseRuntime::prepare_execution` returns a plain
//! [`MuseExecution`] record; the argv, delivery, control and finish steps
//! are separate methods so routes can stage them around process spawn.

mod launch_provider;

pub use self::launch_provider::LaunchProvider;

mod wire;

pub use self::wire::{
    LaunchControl, LaunchExit, LaunchRequest, NestedRegistration, MUSE_CONNECTION_SETTING,
    MUSE_LAUNCH_SOCKET,
};

mod args;

pub use self::args::muse_arguments;

mod connection;

pub use self::connection::{
    connection_authorized, muse_connection_authorized, select_connection, select_muse_connection,
    MuseConnection,
};

mod runtime_types;

pub use self::runtime_types::{
    MuseCaller, MuseExecution, MuseHooks, MuseNested, MusePeer, MuseRuntime,
};

mod validate;

pub use self::validate::{
    host_go_arch, muse_account_modes, muse_account_node, muse_child_pid, muse_delivery_valid,
    muse_elf, muse_host_environment, muse_mapped_uid, muse_passwd_valid, muse_project_cgroup,
    muse_project_credential_root, muse_readonly_mount, muse_registration_valid, muse_signal,
    MUSE_CHILD_INSPECT, MUSE_INSPECT,
};

mod argv;

pub use self::argv::{
    launch_command_argv, muse_command_argv, unit_active_argv, unit_invocation_argv,
};

// ---------- runtime ----------

mod codec;

pub use self::codec::{muse_command_exit, split_json_object, JsonPacket};

mod config;

pub use self::config::decode_config_view;

mod execution;
mod inspect;
mod launch;
mod provider;

pub use self::provider::{
    fresh_provider_snapshot, release_provider_ref, release_provider_snapshot,
    retained_muse_executable, ProviderSnapshot,
};

pub use self::launch::MuseLaunch;

mod nested;
mod observe;
mod operate;
mod ops;

pub use self::ops::{muse_resize, prepare_muse_listener_dir, state_container};

mod request;

pub use self::request::{muse_descriptors_valid, muse_request_from_fd, MuseRequest};

mod resolve;
mod socket;
mod spawn;

pub(in crate::muse) use self::spawn::spawn_execution;

mod stage;
mod stop;

pub use self::socket::muse_peer_from_fd;

pub(in crate::muse) use self::inspect::{muse_peer_alive, sleep_until, MuseInspection};
