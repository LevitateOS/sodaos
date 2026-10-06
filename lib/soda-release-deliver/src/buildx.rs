//! Port of the `internal/release/build` surface `deliver` consumes:
//! digest/revision/architecture shapes (via `soda-build-tools`), the
//! Forgejo toolchain descriptor, the inspected image identity, and the
//! confined file helpers (`HashAt`, `FreshDirectory`, `PrivateDestination`,
//! `WriteNew`, `ReadJSONAt`).

use soda_json::JsonValue;

use crate::jsonx::{base64_decode, Binder, Emit, Emitter};
use crate::Error;

pub use soda_build_tools::reader::{is_digest, is_revision, oci_architecture};

mod filesystem;

pub use filesystem::{
    fresh_directory, hash_at, private_destination, read_at, read_json_at, read_layout_entry,
    write_new, Root,
};

pub const FORGEJO_COMPILER_IMAGE: &str =
    "docker.io/library/golang@sha256:28d89ee9cc0ff9fec75c82ca201e6bf7fdf9a679d4b7b24dfa04f2bb766bb468";
pub const FORGEJO_BUN_VERSION: &str = "1.4.2";
pub const FORGEJO_BUN_SHA256: &str =
    "4835eca59d6da70f4674f5642f6e459dcadab773695b2ed9922d131057989742";
pub const FORGEJO_UPSTREAM_BASE: &str = "15.0.9";
pub const FORGEJO_COMPAT_TOKEN: &str = "gitea-1.22.0";

/// `build.Image`: verified OCI image identity.
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
    pub fn decode(value: &JsonValue) -> Result<Image, String> {
        use crate::payload::decode_opt_string;
        let mut b = Binder::new(value).map_err(|_| "invalid image".to_string())?;
        let image = Image {
            manifest: decode_opt_string(&mut b, "Manifest")?,
            config: decode_opt_string(&mut b, "Config")?,
            architecture: decode_opt_string(&mut b, "Architecture")?,
            revision: decode_opt_string(&mut b, "Revision")?,
            source: decode_opt_string(&mut b, "Source")?,
            base_name: decode_opt_string(&mut b, "BaseName")?,
            base_digest: decode_opt_string(&mut b, "BaseDigest")?,
        };
        b.finish_name()?;
        Ok(image)
    }
}

impl Emit for Image {
    fn emit(&self, e: &mut Emitter) {
        e.begin_object(false);
        e.field(true, "Manifest");
        e.string(&self.manifest);
        e.field(false, "Config");
        e.string(&self.config);
        e.field(false, "Architecture");
        e.string(&self.architecture);
        e.field(false, "Revision");
        e.string(&self.revision);
        e.field(false, "Source");
        e.string(&self.source);
        e.field(false, "BaseName");
        e.string(&self.base_name);
        e.field(false, "BaseDigest");
        e.string(&self.base_digest);
        e.end_object(false);
    }
}

/// `build.ForgejoToolchain`: pinned compiler provenance.
#[derive(Debug, Clone, Default, PartialEq, Eq)]
pub struct ForgejoToolchain {
    pub compiler_image: String,
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

    pub fn decode(value: &JsonValue) -> Result<ForgejoToolchain, String> {
        use crate::payload::decode_opt_string;
        let mut b = Binder::new(value).map_err(|_| "invalid toolchain".to_string())?;
        let mut toolchain = ForgejoToolchain {
            compiler_image: decode_opt_string(&mut b, "CompilerImage")?,
            apk_packages: Vec::new(),
        };
        if let Some(items) = b
            .array("APKPackages")
            .map_err(|_| "invalid field APKPackages".to_string())?
        {
            for item in items {
                match item {
                    JsonValue::Str(s) => toolchain.apk_packages.push(s.clone()),
                    _ => return Err("invalid field APKPackages".to_string()),
                }
            }
        }
        b.finish_name()?;
        Ok(toolchain)
    }
}

impl Emit for ForgejoToolchain {
    fn emit(&self, e: &mut Emitter) {
        e.begin_object(false);
        e.field(true, "CompilerImage");
        e.string(&self.compiler_image);
        e.field(false, "APKPackages");
        e.begin_array(self.apk_packages.is_empty());
        for (i, package) in self.apk_packages.iter().enumerate() {
            e.item(i == 0);
            e.string(package);
        }
        e.end_array(self.apk_packages.is_empty());
        e.end_object(false);
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

/// Unknown-field error text matching Go's `encoding/json`.
pub fn unknown_field(name: &str) -> String {
    format!("json: unknown field \"{name}\"")
}

pub fn decode_bytes_value(value: &JsonValue) -> Result<Vec<u8>, crate::jsonx::DecodeError> {
    match value {
        JsonValue::Str(s) => base64_decode(s),
        _ => Err(crate::jsonx::DecodeError),
    }
}

#[cfg(test)]
mod tests;
