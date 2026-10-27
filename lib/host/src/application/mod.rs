//! Fixed appliance operations for admitted Canlang images. Application semantics,
//! provider configuration and activation remain in the installed Canlang target.
mod backups;
mod build;
mod package;
pub use package::{BuildArtifact, BuildRequest};
mod home;
pub use backups::{Inventory as SnapshotInventory, Request as SnapshotRequest};
mod ingress;
pub(super) fn home_path(path: &std::path::Path) -> Result<std::path::PathBuf, String> {
    home::home_path(path)
}
pub(super) fn storage_root() -> Result<std::path::PathBuf, String> {
    home::storage_root()
}
mod restoration;
mod runtime;
mod setup;
pub mod snapshot;
mod wire;
pub use setup::{
    classify_reconcile_failure, parse_reconcile_outcome, reconcile_permits_rotation,
    replace as publish_setup_file, setup_renewal_advance, ReconcileExpectation, ReconcileOutcome,
    ReconcileRefusal, SetupObservation, SetupRecord, SetupRequest, SetupResult, SetupSpec, Token,
    RECONCILE_REFUSAL_LINE, RECONCILE_UNCLAIMED_LINE, SETUP_RENEWAL_WINDOW_SECS,
};

pub use runtime::{ReleaseBinding, Runtime};
pub use wire::{Observation, Request, Spec};
