//! Outside VM/evidence acceptance harness: Rust port of the privileged
//! acceptance driver (`tools/soda-acceptance`), the SSH remote executor, and
//! the root-run installed probes (lane A, PR29).
//!
//! The library holds every ported behavior; the three binaries are thin
//! entry points. `soda-acceptance` drives checks from the operator host,
//! `soda-acceptance-remote` runs one target-side payload phase piped over
//! SSH, and `soda-host-probes` answers the kept installed shell skeletons
//! on the host under test.

pub mod cockpit;
pub mod command;
pub mod coreos;
pub mod driver;
pub mod error;
pub mod evidence;
pub mod files;
pub mod host_probes;
pub mod jsonio;
pub mod native_phase;
pub mod process;
pub mod probe;
pub mod project_state;
pub mod provisioning;
pub mod qmp;
pub mod remote;
pub mod report;
pub mod sha256;
pub mod stream;
pub mod trust;
pub mod vm;
