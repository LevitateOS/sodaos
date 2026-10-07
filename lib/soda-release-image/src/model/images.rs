use crate::error::Error;
use crate::jsonio;
use serde::Serialize;

// ---------------------------------------------------------------------------
// build.Image
// ---------------------------------------------------------------------------

#[derive(Debug, Clone, Default, PartialEq, Eq, Serialize)]
pub struct Image {
    #[serde(rename = "Manifest")]
    pub manifest: String,
    #[serde(rename = "Config")]
    pub config: String,
    #[serde(rename = "Architecture")]
    pub architecture: String,
    #[serde(rename = "Revision")]
    pub revision: String,
    #[serde(rename = "Source")]
    pub source: String,
    #[serde(rename = "BaseName")]
    pub base_name: String,
    #[serde(rename = "BaseDigest")]
    pub base_digest: String,
}

impl Image {
    pub fn parse(text: &str) -> Result<Image, Error> {
        jsonio::parse(text)
    }
}

crate::jsonio::case_record!(Image, {
    manifest: String => "Manifest",
    config: String => "Config",
    architecture: String => "Architecture",
    revision: String => "Revision",
    source: String => "Source",
    base_name: String => "BaseName",
    base_digest: String => "BaseDigest",
});

#[derive(Debug, Clone, Default, PartialEq, Eq)]
pub struct ProducedImage {
    pub manifest: String,
    pub config: String,
    pub archive_sha256: String,
}
