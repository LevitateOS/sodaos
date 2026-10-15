//! Canonical Payload v4 for the release-image producer.
//!
//! The wire types and validation are owned by soda-build-tools; this module
//! keeps only the producer's file-read admission. There is no v3 branch:
//! old documents fail validation instead of being migrated.

use crate::error::Error;
use crate::sys;
// Import the canonical v4 types: the producer consumes them, never mirrors
// them. Native package/catalog production (R02) uses the same definitions
// the R01 readers validate.
pub use soda_build_tools::payload::{
    FountainCatalog, FountainPackage, FountainRelease, OptionalExtensions, TailnetDependency,
    TailnetImages, BROWSER_CONTRACT, CATALOG_VERSION, EXTENSION_PROTOCOL, FORMAT,
    MAX_PACKAGE_BYTES, SODA_CAPABILITIES, SODA_PACKAGE_ID, TAILNET_CAPABILITIES,
    TAILNET_PACKAGE_ID,
};
pub use soda_build_tools::payload::{
    ImageBinding as PayloadImage, ImageProvenance, PayloadV4 as Payload, Role,
};
pub use soda_build_tools::project_support::ProjectSupport;

/// `deliver.Load`: strict build-JSON read plus validation.
pub fn load(path: &str) -> Result<Payload, Error> {
    let text = sys::read_json_build_text(path)?;
    Payload::parse(text.as_bytes()).map_err(|e| Error::msg(e.to_string()))
}
