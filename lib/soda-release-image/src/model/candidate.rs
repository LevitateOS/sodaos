use soda_json::JsonValue;

use crate::error::Error;
use crate::jsonio;
use crate::sys;

use super::{
    content_get, deliver_hash, is_digest, is_revision, oci_architecture, prefixed_digest, Image,
    Payload, FORGEJO_COMPILER_IMAGE, SODA_SOURCE,
};

// ---------------------------------------------------------------------------
// deliver.Candidate
// ---------------------------------------------------------------------------

#[derive(Debug, Clone, Default, PartialEq, Eq)]
pub struct ForgejoToolchain {
    pub compiler_image: String,
    pub apk_packages: Vec<String>,
}

impl ForgejoToolchain {
    pub fn parse(value: &JsonValue) -> Result<ForgejoToolchain, Error> {
        jsonio::check_no_unknown(value, &["CompilerImage", "APKPackages"])?;
        let mut apk_packages = Vec::new();
        for item in jsonio::require_array(value, "APKPackages")? {
            match item {
                JsonValue::Str(s) => apk_packages.push(s.clone()),
                _ => return Err(Error::msg("invalid APKPackages")),
            }
        }
        Ok(ForgejoToolchain {
            compiler_image: jsonio::require_string(value, "CompilerImage")?,
            apk_packages,
        })
    }

    pub fn to_json(&self) -> JsonValue {
        JsonValue::Object(vec![
            (
                "CompilerImage".to_string(),
                JsonValue::Str(self.compiler_image.clone()),
            ),
            (
                "APKPackages".to_string(),
                JsonValue::Array(
                    self.apk_packages
                        .iter()
                        .map(|s| JsonValue::Str(s.clone()))
                        .collect(),
                ),
            ),
        ])
    }

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
    let mut previous: Option<&str> = None;
    for name in packages {
        if name.is_empty()
            || name
                .bytes()
                .any(|b| matches!(b, b' ' | b'\t' | b'\n' | b'\r' | b'\\' | 0))
        {
            return false;
        }
        // Non-decreasing (slices.IsSorted) plus the explicit duplicate refusal.
        if let Some(prev) = previous {
            if prev > name.as_str() || prev == name.as_str() {
                return false;
            }
        }
        previous = Some(name);
    }
    true
}

fn has_forgejo_native_build_tools(packages: &[String]) -> bool {
    let (mut base, mut gcc, mut musl) = (false, false, false);
    for name in packages {
        base = base || name == "build-base-0.5-r4";
        gcc = gcc || name.starts_with("gcc-");
        musl = musl || name.starts_with("musl-dev-");
    }
    base && gcc && musl
}

#[derive(Debug, Clone, Default, PartialEq, Eq)]
pub struct Candidate {
    pub format: i64,
    pub host: Image,
    pub host_reference: String,
    pub host_archive_sha256: String,
    pub payload_sha256: String,
    pub migration: String,
    pub notes: String,
    pub forgejo_revision: String,
    pub forgejo_source_sha256: String,
    pub forgejo_toolchain: ForgejoToolchain,
    pub architecture: String,
    pub content_sha256: Vec<(String, String)>,
}

impl Candidate {
    pub fn parse(value: &JsonValue) -> Result<Candidate, Error> {
        jsonio::check_no_unknown(
            value,
            &[
                "Format",
                "Host",
                "HostReference",
                "HostArchiveSHA256",
                "PayloadSHA256",
                "Migration",
                "Notes",
                "ForgejoRevision",
                "ForgejoSourceSHA256",
                "ForgejoToolchain",
                "Architecture",
                "ContentSHA256",
            ],
        )?;
        let host_value = jsonio::require_object(value, "Host")?;
        let host = if matches!(host_value, JsonValue::Null) {
            Image::default()
        } else {
            Image::parse(host_value)?
        };
        let toolchain_value = jsonio::require_object(value, "ForgejoToolchain")?;
        let forgejo_toolchain = if matches!(toolchain_value, JsonValue::Null) {
            ForgejoToolchain::default()
        } else {
            ForgejoToolchain::parse(toolchain_value)?
        };
        Ok(Candidate {
            format: jsonio::require_i64(value, "Format")?,
            host,
            host_reference: jsonio::require_string(value, "HostReference")?,
            host_archive_sha256: jsonio::require_string(value, "HostArchiveSHA256")?,
            payload_sha256: jsonio::require_string(value, "PayloadSHA256")?,
            migration: jsonio::require_string(value, "Migration")?,
            notes: jsonio::require_string(value, "Notes")?,
            forgejo_revision: jsonio::require_string(value, "ForgejoRevision")?,
            forgejo_source_sha256: jsonio::require_string(value, "ForgejoSourceSHA256")?,
            forgejo_toolchain,
            architecture: jsonio::require_string(value, "Architecture")?,
            content_sha256: jsonio::string_map(value, "ContentSHA256")?,
        })
    }

    pub fn to_json(&self) -> JsonValue {
        let mut content = self.content_sha256.clone();
        content.sort_by(|a, b| a.0.cmp(&b.0));
        JsonValue::Object(vec![
            (
                "Format".to_string(),
                JsonValue::Number(self.format.to_string()),
            ),
            ("Host".to_string(), self.host.to_json()),
            (
                "HostReference".to_string(),
                JsonValue::Str(self.host_reference.clone()),
            ),
            (
                "HostArchiveSHA256".to_string(),
                JsonValue::Str(self.host_archive_sha256.clone()),
            ),
            (
                "PayloadSHA256".to_string(),
                JsonValue::Str(self.payload_sha256.clone()),
            ),
            (
                "Migration".to_string(),
                JsonValue::Str(self.migration.clone()),
            ),
            ("Notes".to_string(), JsonValue::Str(self.notes.clone())),
            (
                "ForgejoRevision".to_string(),
                JsonValue::Str(self.forgejo_revision.clone()),
            ),
            (
                "ForgejoSourceSHA256".to_string(),
                JsonValue::Str(self.forgejo_source_sha256.clone()),
            ),
            (
                "ForgejoToolchain".to_string(),
                self.forgejo_toolchain.to_json(),
            ),
            (
                "Architecture".to_string(),
                JsonValue::Str(self.architecture.clone()),
            ),
            (
                "ContentSHA256".to_string(),
                JsonValue::Object(
                    content
                        .into_iter()
                        .map(|(k, v)| (k, JsonValue::Str(v)))
                        .collect(),
                ),
            ),
        ])
    }

    pub fn validate(&self, payload: &Payload, payload_bytes: &[u8]) -> Result<(), Error> {
        let arch = match oci_architecture(&payload.architecture) {
            Ok(arch) => arch,
            Err(_) => return Err(Error::msg(sys::refused())),
        };
        if payload.validate().is_err() {
            return Err(Error::msg(sys::refused()));
        }
        if !valid_candidate_host(self, payload, arch)
            || !valid_candidate_provenance(self, payload, payload_bytes)
        {
            return Err(Error::msg(sys::refused()));
        }
        Ok(())
    }
}

fn valid_candidate_host(candidate: &Candidate, payload: &Payload, arch: &str) -> bool {
    candidate.host_reference
        == format!(
            "{}-host@{}",
            payload.repository_prefix, candidate.host.manifest
        )
        && prefixed_digest(&candidate.host.manifest)
        && prefixed_digest(&candidate.host.config)
        && is_digest(&candidate.host_archive_sha256)
        && candidate.host.architecture == arch
        && candidate.host.revision == payload.revision
        && candidate.host.base_name == payload.base
}

fn valid_candidate_provenance(
    candidate: &Candidate,
    payload: &Payload,
    payload_bytes: &[u8],
) -> bool {
    valid_candidate_source(candidate, payload, payload_bytes)
        && valid_candidate_build(candidate, payload)
        && valid_candidate_content(&candidate.content_sha256)
        && !candidate.migration.is_empty()
        && !candidate.notes.is_empty()
}

fn valid_candidate_source(candidate: &Candidate, payload: &Payload, payload_bytes: &[u8]) -> bool {
    let base_digest = payload.base.split('@').nth(1).unwrap_or("");
    candidate.format == 1
        && candidate.payload_sha256
            == deliver_hash(payload_bytes)
                .strip_prefix("sha256:")
                .unwrap_or("")
        && candidate.host.base_digest == base_digest
        && candidate.host.source == SODA_SOURCE
}

fn valid_candidate_build(candidate: &Candidate, payload: &Payload) -> bool {
    is_revision(&candidate.forgejo_revision)
        && is_digest(&candidate.forgejo_source_sha256)
        && candidate.forgejo_toolchain.validate().is_ok()
        && candidate.architecture == payload.architecture
}

const REQUIRED_CANDIDATE_CONTENT: [&str; 9] = [
    "dashboard:/usr/local/bin/soda-dashboard",
    "forgejo:/usr/local/bin/gitea",
    "extension:/usr/local/bin/gitea",
    "extension:/usr/share/soda/extension/extension.json",
    "extension:/usr/share/soda/extension/backend",
    "extension:/usr/share/soda/extension/run",
    "host:/usr/share/containers/systemd/forgejo.container",
    "host:/usr/share/containers/systemd/soda-dashboard.container",
    "host:/usr/lib/systemd/system/soda-extension-install.service",
];

pub fn valid_candidate_content(files: &[(String, String)]) -> bool {
    for path in REQUIRED_CANDIDATE_CONTENT {
        if !is_digest(&content_get(files, path)) {
            return false;
        }
    }
    if content_get(files, "forgejo:/usr/local/bin/gitea")
        != content_get(files, "extension:/usr/local/bin/gitea")
    {
        return false;
    }
    let mut assets = 0;
    for (file, hash) in files {
        if !is_digest(hash) {
            return false;
        }
        if let Some(name) = file.strip_prefix("extension:/usr/share/soda/extension/assets/") {
            if !valid_candidate_asset_name(name) {
                return false;
            }
            assets += 1;
            continue;
        }
        if !REQUIRED_CANDIDATE_CONTENT.contains(&file.as_str()) {
            return false;
        }
    }
    assets > 0
}

fn valid_candidate_asset_name(name: &str) -> bool {
    if name.is_empty()
        || sys::clean_path(name) != name
        || name.starts_with('/')
        || name.starts_with("../")
        || name.bytes().any(|b| matches!(b, b'\\' | b'\n' | b'\r' | 0))
    {
        return false;
    }
    true
}
