//! Shared-blob OCI layout verification ported from
//! `internal/release/build` (`oci_layout.go` plus the reachable half of
//! `oci.go`). Only the installer's `InspectOCILayout` path is ported: exact
//! image-set identity, blob hashing, and byte counts. Archive and layer
//! member inspection stay in Go with their callers.

pub use self::inspection::inspect_oci_layout;
pub use self::layout::{OciImage, OciLayout};

const MANIFEST_MEDIA_TYPE: &str = "application/vnd.oci.image.manifest.v1+json";
const CONFIG_MEDIA_TYPE: &str = "application/vnd.oci.image.config.v1+json";
const INDEX_MEDIA_TYPE: &str = "application/vnd.oci.image.index.v1+json";
const LAYER_TAR: &str = "application/vnd.oci.image.layer.v1.tar";
const LAYER_GZIP: &str = "application/vnd.oci.image.layer.v1.tar+gzip";
const LAYER_ZSTD: &str = "application/vnd.oci.image.layer.v1.tar+zstd";

mod inspection;
mod layout;
mod metadata;

#[cfg(test)]
pub mod test_support;

#[cfg(test)]
mod tests;
