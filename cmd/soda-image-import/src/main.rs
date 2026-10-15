//! soda-image-import is the image-based appliance's fixed native import phase.
//!
//! Payload v4 acquisition over the rootful Podman store: read and strictly
//! validate the local payload, inspect each expected config, pull only
//! missing configs by their exact sealed reference, then re-verify every
//! selected identity (config plus linux/amd64 platform) before reporting
//! success. Never starts, replaces or deletes workloads.
//!
//! The no-argument boot invocation imports the six required roles. The
//! internal `ensure --payload ABS --set required|tailnet` command serves the
//! two trusted callers (Tailnet prepare, maintenance) with an explicitly
//! selected set; it accepts no image URL/digest/registry override.

use std::time::Duration;

mod context;
mod import;
mod payload;
mod platform;
mod sha256;

use context::run;
use import::{ensure_images, native_import, run_podman, run_podman_output, ImageSet};
use payload::{ImageBinding, Payload, Role};
use platform::require_native;

#[cfg(test)]
use context::ImportCtx;
#[cfg(test)]
use import::PodmanOutcome;
#[cfg(test)]
use platform::oci_architecture;
#[cfg(test)]
use std::{
    fs,
    path::{Path, PathBuf},
    sync::atomic::{AtomicBool, Ordering},
    time::Instant,
};

#[link(name = "c")]
extern "C" {
    fn geteuid() -> u32;
    fn signal(signum: i32, handler: extern "C" fn(i32)) -> extern "C" fn(i32);
}

const RELEASE_PATH: &str = "/usr/share/soda/release.json";
const PODMAN: &str = "/usr/bin/podman";
/// Go: `context.WithTimeout(ctx, 10*time.Minute)`; the unit allows 11.
const IMPORT_TIMEOUT: Duration = Duration::from_secs(10 * 60);
const SIGINT: i32 = 2;
const SIGTERM: i32 = 15;

fn main() {
    std::process::exit(run());
}

#[cfg(test)]
mod tests;
