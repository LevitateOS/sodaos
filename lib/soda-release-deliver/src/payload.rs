//! `payload.go`: immutable appliance payload metadata.

use std::collections::BTreeMap;

use soda_json::JsonValue;

use crate::buildx::{is_digest, is_revision, oci_architecture, Root};
use crate::jsonx::{as_i64, Binder, Emit, Emitter};
use crate::Error;

pub const PATH: &str = "/usr/share/soda/release.json";
pub const IMAGES_PATH: &str = "/usr/share/soda/images";

pub const NAMES: [&str; 6] = [
    "dashboard",
    "forgejo",
    "extension",
    "proxy",
    "project-os",
    "tailnet",
];

#[derive(Debug, Clone, Default, PartialEq, Eq)]
pub struct Image {
    pub reference: String,
    pub config: String,
    pub manifest: String,
    pub archive_sha256: String,
}

impl Image {
    pub fn decode(value: &JsonValue) -> Result<Image, String> {
        let mut b = Binder::new(value).map_err(|_| "invalid image".to_string())?;
        let image = Image {
            reference: decode_opt_string(&mut b, "Reference")?,
            config: decode_opt_string(&mut b, "Config")?,
            manifest: decode_opt_string(&mut b, "Manifest")?,
            archive_sha256: decode_opt_string(&mut b, "ArchiveSHA256")?,
        };
        b.finish_name()?;
        Ok(image)
    }
}

impl Emit for Image {
    fn emit(&self, e: &mut Emitter) {
        e.begin_object(false);
        e.field(true, "Reference");
        e.string(&self.reference);
        e.field(false, "Config");
        e.string(&self.config);
        e.field(false, "Manifest");
        e.string(&self.manifest);
        e.field(false, "ArchiveSHA256");
        e.string(&self.archive_sha256);
        e.end_object(false);
    }
}

#[derive(Debug, Clone, Default, PartialEq, Eq)]
pub struct Payload {
    pub format: i64,
    pub id: String,
    pub revision: String,
    pub architecture: String,
    pub core_os: String,
    pub base: String,
    pub repository_prefix: String,
    pub schema: i64,
    pub presentation_sha256: String,
    pub host_packages_sha256: String,
    pub images: BTreeMap<String, Image>,
    // `nil` vs empty is observable in JSON (`null` vs `[]`); Go leaves this
    // unset, so `None` marshals as `null` exactly like the owner.
    pub upgrade_from: Option<Vec<String>>,
}

fn is_core_os_version(s: &str) -> bool {
    let parts: Vec<&str> = s.split('.').collect();
    if parts.len() != 4 {
        return false;
    }
    parts
        .iter()
        .all(|p| !p.is_empty() && p.bytes().all(|b| b.is_ascii_digit()))
}

fn is_digest_ref(s: &str) -> bool {
    s.starts_with("sha256:") && is_digest(s.strip_prefix("sha256:").unwrap_or(""))
}

impl Payload {
    fn valid_identity(&self) -> bool {
        if self.format != 3 || !is_revision(&self.revision) {
            return false;
        }
        if self.id != format!("{}.soda-{}", self.core_os, &self.revision[..12]) {
            return false;
        }
        is_core_os_version(&self.core_os)
    }

    fn valid_base(&self) -> bool {
        let (_, rest) = match self.base.split_once("/fedora/fedora-coreos@sha256:") {
            Some((host, digest)) => (host, digest),
            None => return false,
        };
        let host = self
            .base
            .split_once("/fedora/fedora-coreos@sha256:")
            .map(|(h, _)| h)
            .unwrap_or("");
        if host.is_empty() || host.contains('/') {
            return false;
        }
        is_digest(rest)
            && self.schema >= 1
            && is_digest(&self.presentation_sha256)
            && is_digest(&self.host_packages_sha256)
    }

    fn valid_images(&self) -> Result<(), Error> {
        if self.images.len() != NAMES.len() {
            return Err(Error::msg("complete image set required"));
        }
        for name in NAMES {
            let image = match self.images.get(name) {
                Some(image) => image,
                None => return Err(Error::msg(format!("invalid {name} image binding"))),
            };
            let reference = format!("{}-{name}@{}", self.repository_prefix, image.manifest);
            if !is_digest_ref(&image.config)
                || !is_digest_ref(&image.manifest)
                || !is_digest(&image.archive_sha256)
                || image.reference != reference
            {
                return Err(Error::msg(format!("invalid {name} image binding")));
            }
        }
        if self.images["extension"].config == self.images["forgejo"].config {
            return Err(Error::msg(
                "extension requires an independent image identity",
            ));
        }
        Ok(())
    }

    pub fn validate(&self) -> Result<(), Error> {
        if !self.valid_identity() {
            return Err(Error::msg("invalid appliance payload identity"));
        }
        if let Err(e) = oci_architecture(&self.architecture) {
            return Err(Error::msg(e.0));
        }
        if !valid_repository_prefix(&self.repository_prefix) {
            return Err(Error::msg("explicit GHCR repository prefix required"));
        }
        if !self.valid_base() {
            return Err(Error::msg("incomplete appliance payload"));
        }
        self.valid_images()?;
        if matches!(&self.upgrade_from, Some(entries) if !entries.is_empty()) {
            return Err(Error::msg("candidate has no qualified upgrade paths"));
        }
        Ok(())
    }

    pub fn decode(value: &JsonValue) -> Result<Payload, String> {
        let mut b = Binder::new(value).map_err(|_| "invalid payload".to_string())?;
        let mut payload = Payload {
            format: decode_opt_i64(&mut b, "Format")?,
            id: decode_opt_string(&mut b, "ID")?,
            revision: decode_opt_string(&mut b, "Revision")?,
            architecture: decode_opt_string(&mut b, "Architecture")?,
            core_os: decode_opt_string(&mut b, "CoreOS")?,
            base: decode_opt_string(&mut b, "Base")?,
            repository_prefix: decode_opt_string(&mut b, "RepositoryPrefix")?,
            schema: decode_opt_i64(&mut b, "Schema")?,
            presentation_sha256: decode_opt_string(&mut b, "PresentationSHA256")?,
            host_packages_sha256: decode_opt_string(&mut b, "HostPackagesSHA256")?,
            images: BTreeMap::new(),
            upgrade_from: None,
        };
        if let Some(entries) = b
            .entries("Images")
            .map_err(|_| "invalid field Images".to_string())?
        {
            for (name, item) in entries {
                payload.images.insert(name.clone(), Image::decode(item)?);
            }
        }
        if let Some(items) = b
            .array("UpgradeFrom")
            .map_err(|_| "invalid field UpgradeFrom".to_string())?
        {
            let mut entries = Vec::new();
            for item in items {
                match item {
                    JsonValue::Str(s) => entries.push(s.clone()),
                    _ => return Err("invalid field UpgradeFrom".to_string()),
                }
            }
            payload.upgrade_from = Some(entries);
        }
        b.finish_name()?;
        Ok(payload)
    }

    pub fn decode_build(value: &JsonValue) -> Result<Payload, Error> {
        Payload::decode(&crate::jsonx::dedupe_last_wins(value.clone())).map_err(Error::msg)
    }
}

pub fn decode_opt_string(b: &mut Binder<'_>, name: &str) -> Result<String, String> {
    Ok(b.string(name)
        .map_err(|_| format!("invalid field {name}"))?
        .unwrap_or_default())
}

pub fn decode_opt_i64(b: &mut Binder<'_>, name: &str) -> Result<i64, String> {
    let raw = b
        .integer(name)
        .map_err(|_| format!("invalid field {name}"))?
        .unwrap_or(0);
    as_i64(raw).map_err(|_| format!("invalid field {name}"))
}

pub fn decode_opt_u64(b: &mut Binder<'_>, name: &str) -> Result<u64, String> {
    let raw = b
        .integer(name)
        .map_err(|_| format!("invalid field {name}"))?
        .unwrap_or(0);
    crate::jsonx::as_u64(raw).map_err(|_| format!("invalid field {name}"))
}

pub fn decode_opt_bool(b: &mut Binder<'_>, name: &str) -> Result<bool, String> {
    Ok(b.boolean(name)
        .map_err(|_| format!("invalid field {name}"))?
        .unwrap_or(false))
}

pub fn decode_string_map(
    b: &mut Binder<'_>,
    name: &str,
) -> Result<BTreeMap<String, String>, String> {
    let mut map = BTreeMap::new();
    if let Some(entries) = b
        .entries(name)
        .map_err(|_| format!("invalid field {name}"))?
    {
        for (key, item) in entries {
            match item {
                JsonValue::Str(s) => {
                    map.insert(key.clone(), s.clone());
                }
                _ => return Err(format!("invalid field {name}")),
            }
        }
    }
    Ok(map)
}

impl Emit for Payload {
    fn emit(&self, e: &mut Emitter) {
        e.begin_object(false);
        e.field(true, "Format");
        e.int(self.format);
        e.field(false, "ID");
        e.string(&self.id);
        e.field(false, "Revision");
        e.string(&self.revision);
        e.field(false, "Architecture");
        e.string(&self.architecture);
        e.field(false, "CoreOS");
        e.string(&self.core_os);
        e.field(false, "Base");
        e.string(&self.base);
        e.field(false, "RepositoryPrefix");
        e.string(&self.repository_prefix);
        e.field(false, "Schema");
        e.int(self.schema);
        e.field(false, "PresentationSHA256");
        e.string(&self.presentation_sha256);
        e.field(false, "HostPackagesSHA256");
        e.string(&self.host_packages_sha256);
        e.field(false, "Images");
        e.begin_object(self.images.is_empty());
        for (i, (name, image)) in self.images.iter().enumerate() {
            e.field(i == 0, name);
            image.emit(e);
        }
        e.end_object(self.images.is_empty());
        e.field(false, "UpgradeFrom");
        match &self.upgrade_from {
            None => e.null(),
            Some(entries) => {
                e.begin_array(entries.is_empty());
                for (i, entry) in entries.iter().enumerate() {
                    e.item(i == 0);
                    e.string(entry);
                }
                e.end_array(entries.is_empty());
            }
        }
        e.end_object(false);
    }
}

/// `ValidRepositoryPrefix`: `^ghcr\.io/[a-z0-9][a-z0-9-]*/[a-z0-9][a-z0-9._-]*$`
/// with length below 200.
pub fn valid_repository_prefix(s: &str) -> bool {
    if s.len() >= 200 {
        return false;
    }
    let rest = match s.strip_prefix("ghcr.io/") {
        Some(rest) => rest,
        None => return false,
    };
    let (owner, repo) = match rest.split_once('/') {
        Some(pair) => pair,
        None => return false,
    };
    if repo.contains('/') {
        return false;
    }
    let mut owner_chars = owner.bytes();
    match owner_chars.next() {
        Some(b @ (b'a'..=b'z' | b'0'..=b'9')) => {
            let _ = b;
        }
        _ => return false,
    }
    if !owner_chars.all(|b| matches!(b, b'a'..=b'z' | b'0'..=b'9' | b'-')) {
        return false;
    }
    let mut repo_chars = repo.bytes();
    match repo_chars.next() {
        Some(b @ (b'a'..=b'z' | b'0'..=b'9')) => {
            let _ = b;
        }
        _ => return false,
    }
    repo_chars.all(|b| matches!(b, b'a'..=b'z' | b'0'..=b'9' | b'.' | b'_' | b'-'))
}

/// `Load`: read and validate a payload file.
pub fn load(path: &str) -> Result<Payload, Error> {
    let parent = match path.rfind('/') {
        Some(0) | None => "/".to_string(),
        Some(i) => path[..i].to_string(),
    };
    let base = match path.rfind('/') {
        Some(i) => &path[i + 1..],
        None => path,
    };
    let root = Root::open(&parent)?;
    let mut payload = Payload::default();
    crate::buildx::read_json_at(&root, base, &mut |value| {
        Payload::decode_build(value)
            .map(|decoded| payload = decoded)
            .map_err(|e| e.0)
    })?;
    payload.validate()?;
    Ok(payload)
}

#[cfg(test)]
mod tests;
