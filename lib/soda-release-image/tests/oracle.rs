//! Selected release-image behavior checks. These exercise current producers
//! and validation/refusal contracts; byte comparisons remain where raw output
//! representation itself is the behavior under test.

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
