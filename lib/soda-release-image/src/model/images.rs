use soda_json::JsonValue;

use crate::error::Error;
use crate::jsonio;

// ---------------------------------------------------------------------------
// build.Image
// ---------------------------------------------------------------------------

#[derive(Debug, Clone, Default, PartialEq, Eq)]
pub struct Image {
    pub manifest: String,
    pub config: String,
    pub architecture: String,
    pub revision: String,
    pub source: String,
    pub base_name: String,
    pub base_digest: String,
}

impl Image {
    pub fn parse(value: &JsonValue) -> Result<Image, Error> {
        jsonio::check_no_unknown(
            value,
            &[
                "Manifest",
                "Config",
                "Architecture",
                "Revision",
                "Source",
                "BaseName",
                "BaseDigest",
            ],
        )?;
        Ok(Image {
            manifest: jsonio::require_string(value, "Manifest")?,
            config: jsonio::require_string(value, "Config")?,
            architecture: jsonio::require_string(value, "Architecture")?,
            revision: jsonio::require_string(value, "Revision")?,
            source: jsonio::require_string(value, "Source")?,
            base_name: jsonio::require_string(value, "BaseName")?,
            base_digest: jsonio::require_string(value, "BaseDigest")?,
        })
    }

    pub fn to_json(&self) -> JsonValue {
        JsonValue::Object(vec![
            (
                "Manifest".to_string(),
                JsonValue::Str(self.manifest.clone()),
            ),
            ("Config".to_string(), JsonValue::Str(self.config.clone())),
            (
                "Architecture".to_string(),
                JsonValue::Str(self.architecture.clone()),
            ),
            (
                "Revision".to_string(),
                JsonValue::Str(self.revision.clone()),
            ),
            ("Source".to_string(), JsonValue::Str(self.source.clone())),
            (
                "BaseName".to_string(),
                JsonValue::Str(self.base_name.clone()),
            ),
            (
                "BaseDigest".to_string(),
                JsonValue::Str(self.base_digest.clone()),
            ),
        ])
    }
}

#[derive(Debug, Clone, Default, PartialEq, Eq)]
pub struct ProducedImage {
    pub manifest: String,
    pub config: String,
    pub archive_sha256: String,
}
