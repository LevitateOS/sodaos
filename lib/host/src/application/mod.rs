//! Fixed appliance operations for admitted Canlang images. Application semantics,
//! provider configuration and activation remain in the installed Canlang target.
mod backups;
mod home;
pub use backups::{Inventory as SnapshotInventory, Request as SnapshotRequest};
mod ingress;
pub(super) fn home_path(path: &std::path::Path) -> Result<std::path::PathBuf, String> {
    home::home_path(path)
}
mod restoration;
mod runtime;
mod setup;
pub mod snapshot;
mod wire;
pub use setup::{SetupRequest, SetupResult};

pub use runtime::{ReleaseBinding, Runtime};
pub use wire::{Observation, Request, Spec};
