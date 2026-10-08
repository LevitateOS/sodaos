use serde::de::Deserializer;
use serde::{Deserialize, Serialize};

/// `build.Image`: verified OCI image identity shared by the build and image
/// release owners.
#[derive(Debug, Clone, Default, PartialEq, Eq, Serialize, Deserialize)]
#[serde(default, deny_unknown_fields, rename_all = "PascalCase")]
pub struct Image {
    #[serde(deserialize_with = "null_string")]
    pub manifest: String,
    #[serde(deserialize_with = "null_string")]
    pub config: String,
    #[serde(deserialize_with = "null_string")]
    pub architecture: String,
    #[serde(deserialize_with = "null_string")]
    pub revision: String,
    #[serde(deserialize_with = "null_string")]
    pub source: String,
    #[serde(deserialize_with = "null_string")]
    pub base_name: String,
    #[serde(deserialize_with = "null_string")]
    pub base_digest: String,
}

fn null_string<'de, D: Deserializer<'de>>(deserializer: D) -> Result<String, D::Error> {
    Ok(Option::<String>::deserialize(deserializer)?.unwrap_or_default())
}
