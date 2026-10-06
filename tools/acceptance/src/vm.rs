//! Owned QEMU fixture VMs, mirroring `internal/acceptance/vm.go`.
//!
//! CoreOS fixtures only: a fresh copy-on-write disk over a verified
//! read-only base, loopback management SSH, and QMP-driven power
//! control. Disks and provisioning stay retained for inspection,
//! even on failure; no enrollment is inferred or revoked.
//!
//! Cancellation arrives through [`Phase`] instead of contexts, and
//! process readiness is polled instead of channel-signalled. `Close`
//! replays its first outcome like the Go owner's `sync.Once`.

#[path = "vm/base.rs"]
mod base;
#[path = "vm/config.rs"]
mod config;
#[path = "vm/launch.rs"]
mod launch;
#[path = "vm/lifecycle.rs"]
mod lifecycle;

pub use self::base::VerifiedBase;
pub use self::config::{decode_vm_config, RemoteConfig, VmConfig};
pub use self::launch::LaunchFailure;
pub use self::lifecycle::{launch_vm, Vm};

#[cfg(test)]
#[path = "vm/tests.rs"]
mod tests;
