//! Differential oracle: frozen Go-owner outputs captured 2026-10-04.
//! Each case replays the oracle battery against the Rust port and
//! requires byte-identical outputs or identical error messages.

use base64::Engine;
use soda_release_image::{
    compression, extension, ignition, layout, model, packages, quadlet, recall, rootfs, sys,
};

fn b64(data: &[u8]) -> String {
    base64::engine::general_purpose::STANDARD.encode(data)
}

fn check_ok(name: &str, expected_b64: &str, got: &[u8]) {
    assert_eq!(b64(got), expected_b64, "{name}");
}

fn check_err(name: &str, expected: &str, got: &soda_release_image::error::Error) {
    assert_eq!(got.0, expected, "{name}");
}

#[path = "oracle/host.rs"]
mod host;
#[path = "oracle/media.rs"]
mod media;
#[path = "oracle/staging.rs"]
mod staging;
