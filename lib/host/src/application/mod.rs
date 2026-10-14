//! Fixed appliance operations for admitted Canlang images. Application semantics,
//! provider configuration and activation remain in the installed Canlang target.
mod home;
mod ingress;
pub(super) fn home_path(path: &std::path::Path) -> Result<std::path::PathBuf, String> {
    home::home_path(path)
}
mod runtime;
pub mod snapshot;
mod wire;

pub use runtime::{ReleaseBinding, Runtime};
pub use wire::{Observation, Request, Spec};

#[cfg(test)]
mod tests;
