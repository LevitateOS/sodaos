//! Rust port of `internal/release/image` (candidate + media pipeline).
//!
//! The Go package remains the owner of record; this crate mirrors its logic
//! function for function. Foreign `release/build`, `release/deliver`,
//! `store`, and `acceptance` surface used by the pipeline is mirrored in
//! [`sys`] (small exact helpers) and [`model`] (JSON shapes + validators),
//! while heavy foreign operations (container builds, OCI inspection,
//! signing, live CoreOS resolution) sit behind the [`foreign::Production`]
//! and [`foreign::Progress`] traits so tests script them like the Go tests
//! script `BuildExec`/`BuildCapture` closures.

// Function signatures mirror the Go owner one for one (same parameters in
// the same order); grouping them would break the port mapping.
#![allow(clippy::too_many_arguments)]

pub mod build;
pub mod build_compile;
pub mod build_media;
pub mod build_runner;
pub mod build_source;
pub mod complete;
pub mod compression;
pub mod error;
pub mod events;
pub mod extension;
pub mod files;
pub mod foreign;
pub mod forgejo;
pub mod host;
pub mod ignition;
pub mod inspect;
pub mod jsonio;
pub mod layout;
pub mod media;
mod media_assembler;
mod media_authentication;
pub mod model;
pub mod packages;
pub mod payload_stage;
pub mod prepare;
pub mod quadlet;
pub mod recall;
pub mod record;
pub mod request;
pub mod rootfs;
pub mod sys;
