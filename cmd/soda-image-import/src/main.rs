//! soda-image-import is the image-based appliance's fixed native import phase.
//!
//! Rust port of `cmd/soda-image-import` plus the release sliver it runs:
//! payload load/validate (`internal/release/deliver` payload + content +
//! import) over the shared OCI layout verifier (`internal/release/build`
//! OCI layout inspection). CLI surface, exit codes, stderr text, podman
//! argv, and verification rules match the Go implementation.
//!
//! Std and the workspace's locked Serde stack; no new crates-io dependencies.

use std::collections::HashMap;
use std::fs;
use std::path::{Path, PathBuf};
use std::sync::atomic::{AtomicBool, Ordering};
use std::time::{Duration, Instant};

mod context;
mod import;
mod json;
mod oci;
mod payload;
mod platform;
mod sha256;

use context::{admit, run, ImportCtx};
use import::{import_images, native_import, run_podman, verify_content, PodmanOutcome};
use json::json_valid;
use oci::{inspect_oci_layout, OciImage};
use payload::{decode_payload, ImageBinding, Payload};
use platform::{
    is_coreos_version, is_digest, is_prefixed_digest, is_revision, oci_architecture,
    require_native, valid_repository_prefix,
};

#[link(name = "c")]
extern "C" {
    fn geteuid() -> u32;
    fn signal(signum: i32, handler: extern "C" fn(i32)) -> extern "C" fn(i32);
}

const RELEASE_PATH: &str = "/usr/share/soda/release.json";
const IMAGES_PATH: &str = "/usr/share/soda/images";
const PODMAN: &str = "/usr/bin/podman";
/// Go: `context.WithTimeout(ctx, 10*time.Minute)`; the unit allows 11.
const IMPORT_TIMEOUT: Duration = Duration::from_secs(10 * 60);
const SIGINT: i32 = 2;
const SIGTERM: i32 = 15;
const NAMES: [&str; 6] = [
    "dashboard",
    "forgejo",
    "extension",
    "proxy",
    "project-os",
    "tailnet",
];

fn main() {
    std::process::exit(run());
}

#[cfg(test)]
mod tests;
