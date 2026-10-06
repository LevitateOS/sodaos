use std::collections::BTreeMap;

use soda_json::JsonValue;

use crate::buildx::{
    is_digest, is_revision, oci_architecture, ForgejoToolchain, Image as BuildImage,
};
use crate::jsonx::{Binder, Emit, Emitter};
use crate::payload::{decode_opt_i64, decode_opt_string, decode_string_map, Payload};
use crate::{hash_bytes, is_digest_ref, Error};

// ---------------------------------------------------------------------------
// Candidate
// ---------------------------------------------------------------------------

#[derive(Debug, Clone, Default, PartialEq, Eq)]
pub struct Candidate {
    pub format: i64,
    pub host: BuildImage,
    pub host_reference: String,
    pub host_archive_sha256: String,
    pub payload_sha256: String,
    pub migration: String,
    pub notes: String,
    pub forgejo_revision: String,
    pub forgejo_source_sha256: String,
    pub forgejo_toolchain: ForgejoToolchain,
    pub architecture: String,
    pub content_sha256: BTreeMap<String, String>,
}

fn valid_candidate_host(c: &Candidate, p: &Payload, arch: &str) -> bool {
    c.host_reference == format!("{}-host@{}", p.repository_prefix, c.host.manifest)
        && is_digest_ref(&c.host.manifest)
        && is_digest_ref(&c.host.config)
        && is_digest(&c.host_archive_sha256)
        && c.host.architecture == arch
        && c.host.revision == p.revision
        && c.host.base_name == p.base
}

fn valid_candidate_provenance(c: &Candidate, p: &Payload, payload: &[u8]) -> bool {
    valid_candidate_source(c, p, payload)
        && valid_candidate_build(c, p)
        && valid_candidate_content(&c.content_sha256)
        && !c.migration.is_empty()
        && !c.notes.is_empty()
}

fn valid_candidate_source(c: &Candidate, p: &Payload, payload: &[u8]) -> bool {
    let base_digest = p.base.split('@').nth(1).unwrap_or("");
    c.format == 1
        && c.payload_sha256 == hash_bytes(payload).trim_start_matches("sha256:")
        && c.host.base_digest == base_digest
        && c.host.source == "https://github.com/LevitateOS/sodaos"
}

fn valid_candidate_build(c: &Candidate, p: &Payload) -> bool {
    is_revision(&c.forgejo_revision)
        && is_digest(&c.forgejo_source_sha256)
        && c.forgejo_toolchain.validate().is_ok()
        && c.architecture == p.architecture
}

/// `ValidCandidateContent`: exact required content bindings plus assets.
pub fn valid_candidate_content(files: &BTreeMap<String, String>) -> bool {
    if !required_candidate_content(files)
        || files.get("forgejo:/usr/local/bin/gitea") != files.get("extension:/usr/local/bin/gitea")
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
        if !known_candidate_content_name(file) {
            return false;
        }
    }
    assets > 0
}

fn required_candidate_content(files: &BTreeMap<String, String>) -> bool {
    for path in [
        "dashboard:/usr/local/bin/soda-dashboard",
        "forgejo:/usr/local/bin/gitea",
        "extension:/usr/local/bin/gitea",
        "extension:/usr/share/soda/extension/extension.json",
        "extension:/usr/share/soda/extension/backend",
        "extension:/usr/share/soda/extension/run",
        "host:/usr/share/containers/systemd/forgejo.container",
        "host:/usr/share/containers/systemd/soda-dashboard.container",
        "host:/usr/lib/systemd/system/soda-extension-install.service",
    ] {
        match files.get(path) {
            Some(hash) if is_digest(hash) => {}
            _ => return false,
        }
    }
    true
}

pub(crate) fn path_clean(name: &str) -> String {
    if name.is_empty() {
        return ".".to_string();
    }
    let rooted = name.starts_with('/');
    let mut parts: Vec<&str> = Vec::new();
    for segment in name.split('/') {
        match segment {
            "" | "." => {}
            ".." => {
                if parts.pop().is_none() && !rooted {
                    parts.push("..");
                }
            }
            _ => parts.push(segment),
        }
    }
    // Go keeps leading ".." for relative paths; the pop/push above models it.
    let mut out = if rooted {
        "/".to_string()
    } else {
        String::new()
    };
    out.push_str(&parts.join("/"));
    if out.is_empty() {
        ".".to_string()
    } else {
        out
    }
}

fn valid_candidate_asset_name(name: &str) -> bool {
    !name.is_empty()
        && path_clean(name) == name
        && !name.starts_with('/')
        && !name.starts_with("../")
        && !name.bytes().any(|b| matches!(b, b'\\' | b'\n' | b'\r' | 0))
}

fn known_candidate_content_name(file: &str) -> bool {
    matches!(
        file,
        "dashboard:/usr/local/bin/soda-dashboard"
            | "forgejo:/usr/local/bin/gitea"
            | "extension:/usr/local/bin/gitea"
            | "extension:/usr/share/soda/extension/extension.json"
            | "extension:/usr/share/soda/extension/backend"
            | "extension:/usr/share/soda/extension/run"
            | "host:/usr/share/containers/systemd/forgejo.container"
            | "host:/usr/share/containers/systemd/soda-dashboard.container"
            | "host:/usr/lib/systemd/system/soda-extension-install.service"
    )
}

impl Candidate {
    pub fn validate(&self, p: &Payload, payload: &[u8]) -> Result<(), Error> {
        let arch = oci_architecture(&p.architecture).map_err(|e| Error::msg(e.0))?;
        if p.validate().is_err() {
            return Err(Error::refused());
        }
        if !valid_candidate_host(self, p, arch) || !valid_candidate_provenance(self, p, payload) {
            return Err(Error::refused());
        }
        Ok(())
    }

    pub fn decode(value: &JsonValue) -> Result<Candidate, String> {
        let mut b = Binder::new(value).map_err(|_| "invalid candidate".to_string())?;
        let mut candidate = Candidate {
            format: decode_opt_i64(&mut b, "Format")?,
            host: BuildImage::default(),
            host_reference: decode_opt_string(&mut b, "HostReference")?,
            host_archive_sha256: decode_opt_string(&mut b, "HostArchiveSHA256")?,
            payload_sha256: decode_opt_string(&mut b, "PayloadSHA256")?,
            migration: decode_opt_string(&mut b, "Migration")?,
            notes: decode_opt_string(&mut b, "Notes")?,
            forgejo_revision: decode_opt_string(&mut b, "ForgejoRevision")?,
            forgejo_source_sha256: decode_opt_string(&mut b, "ForgejoSourceSHA256")?,
            forgejo_toolchain: ForgejoToolchain::default(),
            architecture: decode_opt_string(&mut b, "Architecture")?,
            content_sha256: BTreeMap::new(),
        };
        if let Some(host) = b
            .entries("Host")
            .map_err(|_| "invalid field Host".to_string())?
        {
            candidate.host = BuildImage::decode(&JsonValue::Object(host.to_vec()))?;
        }
        if let Some(toolchain) = b
            .entries("ForgejoToolchain")
            .map_err(|_| "invalid field ForgejoToolchain".to_string())?
        {
            candidate.forgejo_toolchain =
                ForgejoToolchain::decode(&JsonValue::Object(toolchain.to_vec()))?;
        }
        candidate.content_sha256 = decode_string_map(&mut b, "ContentSHA256")?;
        b.finish_name()?;
        Ok(candidate)
    }
}

impl Emit for Candidate {
    fn emit(&self, e: &mut Emitter) {
        e.begin_object(false);
        e.field(true, "Format");
        e.int(self.format);
        e.field(false, "Host");
        self.host.emit(e);
        e.field(false, "HostReference");
        e.string(&self.host_reference);
        e.field(false, "HostArchiveSHA256");
        e.string(&self.host_archive_sha256);
        e.field(false, "PayloadSHA256");
        e.string(&self.payload_sha256);
        e.field(false, "Migration");
        e.string(&self.migration);
        e.field(false, "Notes");
        e.string(&self.notes);
        e.field(false, "ForgejoRevision");
        e.string(&self.forgejo_revision);
        e.field(false, "ForgejoSourceSHA256");
        e.string(&self.forgejo_source_sha256);
        e.field(false, "ForgejoToolchain");
        self.forgejo_toolchain.emit(e);
        e.field(false, "Architecture");
        e.string(&self.architecture);
        e.field(false, "ContentSHA256");
        e.begin_object(self.content_sha256.is_empty());
        for (i, (name, hash)) in self.content_sha256.iter().enumerate() {
            e.field(i == 0, name);
            e.string(hash);
        }
        e.end_object(self.content_sha256.is_empty());
        e.end_object(false);
    }
}
