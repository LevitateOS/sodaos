//! Forgejo compiler provenance checks (`forgejo.go`).
//!
//! Pins the container compiler image, the archived fork's version stamp,
//! and the native APK toolchain recorded alongside the build.

use super::{is_revision, Error};

/// `docker.io/library/golang:1.26.7-alpine` multi-platform index.
pub const FORGEJO_COMPILER_IMAGE: &str =
    "docker.io/library/golang@sha256:28d89ee9cc0ff9fec75c82ca201e6bf7fdf9a679d4b7b24dfa04f2bb766bb468";

/// The only JS toolchain permitted for the archived fork's frontend.
pub const FORGEJO_BUN_VERSION: &str = "1.4.2";

/// Pins `bun-linux-x64-musl.zip` for [`FORGEJO_BUN_VERSION`].
pub const FORGEJO_BUN_SHA256: &str =
    "4835eca59d6da70f4674f5642f6e459dcadab773695b2ed9922d131057989742";

/// Fountain derives from Forgejo 15.0 LTS.
pub const FORGEJO_UPSTREAM_BASE: &str = "15.0.9";

/// Follows the fork Makefile's `GITEA_COMPATIBILITY` contract.
pub const FORGEJO_COMPAT_TOKEN: &str = "gitea-1.22.0";

/// Maps the exact archived revision to the Makefile-shaped version
/// identity `<base>-soda.<short>+<compat>`.
pub fn forgejo_version_stamp(revision: &str) -> Result<String, Error> {
    if !is_revision(revision) {
        return Err(Error("exact Forgejo source revision required".to_string()));
    }
    Ok(format!(
        "{FORGEJO_UPSTREAM_BASE}-soda.{}+{FORGEJO_COMPAT_TOKEN}",
        &revision[..12]
    ))
}

/// A sorted, duplicate-free APK list without whitespace or backslashes.
pub fn valid_apk_list(packages: &[impl AsRef<str>]) -> bool {
    if packages.is_empty() || packages.len() > 256 {
        return false;
    }
    let mut previous: Option<&str> = None;
    for package in packages {
        let name = package.as_ref();
        if name.is_empty()
            || name
                .bytes()
                .any(|b| matches!(b, b' ' | b'\t' | b'\n' | b'\r' | b'\\' | 0))
        {
            return false;
        }
        if let Some(prev) = previous {
            if prev >= name {
                return false;
            }
        }
        previous = Some(name);
    }
    true
}

pub fn has_native_build_tools(packages: &[impl AsRef<str>]) -> bool {
    let mut build_base = false;
    let mut gcc = false;
    let mut musl_dev = false;
    for package in packages {
        let name = package.as_ref();
        build_base = build_base || name == "build-base-0.5-r4";
        gcc = gcc || name.starts_with("gcc-");
        musl_dev = musl_dev || name.starts_with("musl-dev-");
    }
    build_base && gcc && musl_dev
}

pub struct ForgejoToolchain {
    pub compiler_image: String,
    pub apk_packages: Vec<String>,
}

impl ForgejoToolchain {
    pub fn validate(&self) -> Result<(), Error> {
        if self.compiler_image != FORGEJO_COMPILER_IMAGE || !valid_apk_list(&self.apk_packages) {
            return Err(Error("invalid Forgejo compiler provenance".to_string()));
        }
        if !has_native_build_tools(&self.apk_packages) {
            return Err(Error("incomplete Forgejo APK provenance".to_string()));
        }
        Ok(())
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    fn toolchain() -> Vec<String> {
        vec![
            "build-base-0.5-r4".to_string(),
            "gcc-14.2.0-r0".to_string(),
            "musl-dev-1.2.5-r0".to_string(),
        ]
    }

    #[test]
    fn version_stamp_carries_short_revision() {
        let revision = "a".repeat(40);
        assert_eq!(
            forgejo_version_stamp(&revision).unwrap(),
            format!("15.0.9-soda.{}+gitea-1.22.0", "a".repeat(12))
        );
        assert_eq!(
            forgejo_version_stamp("short").unwrap_err(),
            Error("exact Forgejo source revision required".to_string())
        );
    }

    #[test]
    fn apk_list_requires_sorted_unique_names() {
        assert!(valid_apk_list(&toolchain()));
        assert!(!valid_apk_list(&[] as &[String]));
        assert!(!valid_apk_list(&[
            "gcc-14.2.0-r0".to_string(),
            "build-base-0.5-r4".to_string()
        ]));
        assert!(!valid_apk_list(&[
            "build-base-0.5-r4".to_string(),
            "build-base-0.5-r4".to_string()
        ]));
        assert!(!valid_apk_list(&["has space-1.0".to_string()]));
        assert!(!valid_apk_list(&["back\\slash-1.0".to_string()]));
        assert!(!valid_apk_list(&["".to_string()]));
        assert!(!valid_apk_list(&vec!["a-1.0".to_string(); 257]));
    }

    #[test]
    fn toolchain_validation_pins_image_and_native_tools() {
        let record = ForgejoToolchain {
            compiler_image: FORGEJO_COMPILER_IMAGE.to_string(),
            apk_packages: toolchain(),
        };
        assert!(record.validate().is_ok());
        let wrong_image = ForgejoToolchain {
            compiler_image: "docker.io/library/golang:latest".to_string(),
            apk_packages: toolchain(),
        };
        assert_eq!(
            wrong_image.validate().unwrap_err(),
            Error("invalid Forgejo compiler provenance".to_string())
        );
        let missing_tools = ForgejoToolchain {
            compiler_image: FORGEJO_COMPILER_IMAGE.to_string(),
            apk_packages: vec!["build-base-0.5-r4".to_string()],
        };
        assert_eq!(
            missing_tools.validate().unwrap_err(),
            Error("incomplete Forgejo APK provenance".to_string())
        );
    }
}
