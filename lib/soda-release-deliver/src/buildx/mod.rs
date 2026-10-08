//! Port of the `internal/release/build` surface `deliver` consumes:
//! digest/revision/architecture shapes (via `soda-build-tools`), the
//! Forgejo toolchain descriptor, the inspected image identity, and the
//! confined file helpers (`HashAt`, `FreshDirectory`, `PrivateDestination`,
//! `WriteNew`, `ReadJSONAt`).

use serde::{Deserialize, Serialize};

use crate::Error;

pub use soda_build_tools::reader::{is_digest, is_revision, oci_architecture};

mod filesystem;

pub use filesystem::{
    decode_build_json, fresh_directory, hash_at, open_layout_entry, private_destination, read_at,
    read_json_at, write_new, Root,
};

pub const FORGEJO_COMPILER_IMAGE: &str =
    "docker.io/library/golang@sha256:28d89ee9cc0ff9fec75c82ca201e6bf7fdf9a679d4b7b24dfa04f2bb766bb468";
pub const FORGEJO_BUN_VERSION: &str = "1.4.2";
pub const FORGEJO_BUN_SHA256: &str =
    "4835eca59d6da70f4674f5642f6e459dcadab773695b2ed9922d131057989742";
pub const FORGEJO_UPSTREAM_BASE: &str = "15.0.9";
pub const FORGEJO_COMPAT_TOKEN: &str = "gitea-1.22.0";

/// `build.Image`: verified OCI image identity.
#[derive(Debug, Clone, Default, PartialEq, Eq, Serialize, Deserialize)]
#[serde(default, deny_unknown_fields, rename_all = "PascalCase")]
pub struct Image {
    #[serde(deserialize_with = "crate::json_serde::null_default")]
    pub manifest: String,
    #[serde(deserialize_with = "crate::json_serde::null_default")]
    pub config: String,
    #[serde(deserialize_with = "crate::json_serde::null_default")]
    pub architecture: String,
    #[serde(deserialize_with = "crate::json_serde::null_default")]
    pub revision: String,
    #[serde(deserialize_with = "crate::json_serde::null_default")]
    pub source: String,
    #[serde(deserialize_with = "crate::json_serde::null_default")]
    pub base_name: String,
    #[serde(deserialize_with = "crate::json_serde::null_default")]
    pub base_digest: String,
}

impl Image {}

/// `build.ForgejoToolchain`: pinned compiler provenance.
#[derive(Debug, Clone, Default, PartialEq, Eq, Serialize, Deserialize)]
#[serde(default, deny_unknown_fields)]
pub struct ForgejoToolchain {
    #[serde(rename = "CompilerImage")]
    #[serde(deserialize_with = "crate::json_serde::null_default")]
    pub compiler_image: String,
    #[serde(rename = "APKPackages")]
    #[serde(deserialize_with = "crate::json_serde::null_default")]
    pub apk_packages: Vec<String>,
}

impl ForgejoToolchain {
    pub fn validate(&self) -> Result<(), Error> {
        if self.compiler_image != FORGEJO_COMPILER_IMAGE
            || !valid_forgejo_apk_list(&self.apk_packages)
        {
            return Err(Error::msg("invalid Forgejo compiler provenance"));
        }
        if !has_forgejo_native_build_tools(&self.apk_packages) {
            return Err(Error::msg("incomplete Forgejo APK provenance"));
        }
        Ok(())
    }
}

fn valid_forgejo_apk_list(packages: &[String]) -> bool {
    if packages.is_empty() || packages.len() > 256 {
        return false;
    }
    for (i, name) in packages.iter().enumerate() {
        if i > 0 && packages[i - 1] >= *name {
            return false;
        }
        if name.is_empty()
            || name
                .bytes()
                .any(|c| matches!(c, b' ' | b'\t' | b'\n' | b'\r' | b'\\' | 0))
        {
            return false;
        }
    }
    true
}

fn has_forgejo_native_build_tools(packages: &[String]) -> bool {
    let mut base = false;
    let mut gcc = false;
    let mut musl = false;
    for name in packages {
        base = base || name == "build-base-0.5-r4";
        gcc = gcc || name.starts_with("gcc-");
        musl = musl || name.starts_with("musl-dev-");
    }
    base && gcc && musl
}

#[cfg(test)]
mod tests;
