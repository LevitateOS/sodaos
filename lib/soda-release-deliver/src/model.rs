//! `model.go`: trust, candidate, release, channel, high-water mark, permit.

use std::collections::{BTreeMap, BTreeSet};

use soda_json::JsonValue;

use crate::buildx::{
    is_digest, is_revision, oci_architecture, ForgejoToolchain, Image as BuildImage,
};
use crate::jsonx::{
    as_i64, as_u64, base64_decode, parse_lenient, parse_strict, Binder, Emit, Emitter, Soft,
};
use crate::payload::{
    decode_opt_bool, decode_opt_i64, decode_opt_string, decode_opt_u64, decode_string_map,
    valid_repository_prefix, Payload, NAMES,
};
use crate::{hash_bytes, is_channel, is_digest_ref, Error};

// ---------------------------------------------------------------------------
// Trust
// ---------------------------------------------------------------------------

#[derive(Debug, Clone, Default, PartialEq, Eq)]
pub struct Trust {
    pub format: i64,
    pub prefix: String,
    pub epoch: u64,
    pub keys: BTreeMap<String, Vec<String>>,
    pub not_before: i64,
    pub max_age_seconds: i64,
    pub clock_skew_seconds: i64,
    pub minimum_sequence: BTreeMap<String, u64>,
}

fn valid_trust_timing(t: &Trust) -> bool {
    t.not_before > 0
        && t.max_age_seconds >= 60
        && t.max_age_seconds <= 7 * 86400
        && t.clock_skew_seconds >= 0
        && t.clock_skew_seconds <= 300
}

fn valid_trust_envelope(t: &Trust) -> bool {
    t.format == 1
        && valid_repository_prefix(&t.prefix)
        && t.epoch != 0
        && t.keys.len() == 4
        && t.minimum_sequence.len() == 3
        && valid_trust_timing(t)
}

/// Parse one PEM `PUBLIC KEY` block and require a native P-256 Sigstore
/// public key, returning the DER bytes for fingerprinting.
fn parse_trust_public_key(key: &str) -> Result<Vec<u8>, Error> {
    let der = decode_pem_public_key(key).ok_or_else(Error::refused)?;
    parse_p256_spki(&der)?;
    Ok(der)
}

fn decode_pem_public_key(key: &str) -> Option<Vec<u8>> {
    let begin = "-----BEGIN PUBLIC KEY-----";
    let end = "-----END PUBLIC KEY-----";
    let start = key.find(begin)? + begin.len();
    let tail = &key[start..];
    let end_pos = tail.find(end)?;
    let (body, rest) = tail.split_at(end_pos);
    if !rest[end.len()..].trim().is_empty() {
        return None;
    }
    let compact: String = body.chars().filter(|c| !c.is_whitespace()).collect();
    if compact.is_empty() {
        return None;
    }
    base64_decode(&compact).ok()
}

fn read_der_length(der: &[u8], pos: &mut usize) -> Option<usize> {
    let first = *der.get(*pos)?;
    *pos += 1;
    if first & 0x80 == 0 {
        return Some(first as usize);
    }
    let count = (first & 0x7f) as usize;
    if count == 0 || count > 4 {
        return None;
    }
    let mut length = 0usize;
    for _ in 0..count {
        length = (length << 8) | (*der.get(*pos)? as usize);
        *pos += 1;
    }
    Some(length)
}

fn parse_p256_spki(der: &[u8]) -> Result<(), Error> {
    let refused = || Error::msg("native P-256 Sigstore public key required");
    let mut pos = 0;
    if der.get(pos) != Some(&0x30) {
        return Err(refused());
    }
    pos += 1;
    let outer = read_der_length(der, &mut pos).ok_or_else(refused)?;
    if pos + outer != der.len() {
        return Err(refused());
    }
    if der.get(pos) != Some(&0x30) {
        return Err(refused());
    }
    pos += 1;
    let inner = read_der_length(der, &mut pos).ok_or_else(refused)?;
    let inner_end = pos + inner;
    // ecPublicKey 1.2.840.10045.2.1
    let ec_oid: &[u8] = &[0x06, 0x07, 0x2a, 0x86, 0x48, 0xce, 0x3d, 0x02, 0x01];
    // secp256r1 1.2.840.10045.3.1.7
    let curve_oid: &[u8] = &[0x06, 0x08, 0x2a, 0x86, 0x48, 0xce, 0x3d, 0x03, 0x01, 0x07];
    if der.get(pos..pos + ec_oid.len()) != Some(ec_oid) {
        return Err(refused());
    }
    pos += ec_oid.len();
    if der.get(pos..pos + curve_oid.len()) != Some(curve_oid) {
        return Err(refused());
    }
    pos += curve_oid.len();
    if pos != inner_end {
        return Err(refused());
    }
    if der.get(pos) != Some(&0x03) {
        return Err(refused());
    }
    pos += 1;
    let bit_len = read_der_length(der, &mut pos).ok_or_else(refused)?;
    if bit_len != 66 || pos + bit_len != der.len() {
        return Err(refused());
    }
    if der[pos] != 0x00 || der[pos + 1] != 0x04 {
        return Err(refused());
    }
    // Coordinates must be below the P-256 field prime.
    let prime: &[u8] = &[
        0xff, 0xff, 0xff, 0xff, 0x00, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
        0x00, 0x00, 0x00, 0x00, 0x00, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff,
        0xff, 0xff,
    ];
    let x = &der[pos + 2..pos + 34];
    let y = &der[pos + 34..pos + 66];
    if x >= prime || y >= prime || x.iter().all(|b| *b == 0) && y.iter().all(|b| *b == 0) {
        return Err(refused());
    }
    Ok(())
}

fn admit_trust_role_keys(keys: &[String], seen: &mut BTreeSet<String>) -> Result<(), Error> {
    if keys.is_empty() || keys.len() > 4 {
        return Err(Error::refused());
    }
    for key in keys {
        let der = parse_trust_public_key(key)?;
        let fingerprint = hash_bytes(&der);
        if !seen.insert(fingerprint) {
            return Err(Error::msg("signer roles must not share keys"));
        }
    }
    Ok(())
}

impl Trust {
    pub fn validate(&self) -> Result<(), Error> {
        if !valid_trust_envelope(self) {
            return Err(Error::refused());
        }
        let mut seen = BTreeSet::new();
        for role in ["artifact", "candidate", "preview", "stable"] {
            let empty = Vec::new();
            let keys = self.keys.get(role).unwrap_or(&empty);
            admit_trust_role_keys(keys, &mut seen)?;
            if role != "artifact" && self.minimum_sequence.get(role).copied().unwrap_or(0) == 0 {
                return Err(Error::refused());
            }
        }
        Ok(())
    }

    pub fn role(&self, repository: &str) -> Result<String, Error> {
        for name in ["host", "release", "media"].into_iter().chain(NAMES) {
            if repository == format!("{}-{name}", self.prefix) {
                return Ok("artifact".to_string());
            }
        }
        for channel in ["candidate", "preview", "stable"] {
            if repository == format!("{}-channel-{channel}", self.prefix) {
                return Ok(channel.to_string());
            }
        }
        Err(Error::refused())
    }

    pub fn reference(&self, reference: &str) -> Result<(String, String), Error> {
        let (repo, digest) = reference.split_once('@').ok_or_else(Error::refused)?;
        if !is_digest_ref(digest) {
            return Err(Error::refused());
        }
        let role = self.role(repo)?;
        Ok((repo.to_string(), role))
    }

    pub fn decode(value: &JsonValue) -> Result<Trust, String> {
        let mut b = Binder::new(value).map_err(|_| "invalid trust".to_string())?;
        let mut trust = Trust {
            format: decode_opt_i64(&mut b, "Format")?,
            prefix: decode_opt_string(&mut b, "Prefix")?,
            epoch: decode_opt_u64(&mut b, "Epoch")?,
            keys: BTreeMap::new(),
            not_before: decode_opt_i64(&mut b, "NotBefore")?,
            max_age_seconds: decode_opt_i64(&mut b, "MaxAgeSeconds")?,
            clock_skew_seconds: decode_opt_i64(&mut b, "ClockSkewSeconds")?,
            minimum_sequence: BTreeMap::new(),
        };
        if let Some(entries) = b
            .entries("Keys")
            .map_err(|_| "invalid field Keys".to_string())?
        {
            for (role, item) in entries {
                match item {
                    JsonValue::Array(items) => {
                        let mut keys = Vec::new();
                        for key in items {
                            match key {
                                JsonValue::Str(s) => keys.push(s.clone()),
                                _ => return Err("invalid field Keys".to_string()),
                            }
                        }
                        trust.keys.insert(role.clone(), keys);
                    }
                    _ => return Err("invalid field Keys".to_string()),
                }
            }
        }
        if let Some(entries) = b
            .entries("MinimumSequence")
            .map_err(|_| "invalid field MinimumSequence".to_string())?
        {
            for (channel, item) in entries {
                let raw = item
                    .as_integer()
                    .ok_or_else(|| "invalid field MinimumSequence".to_string())?;
                trust.minimum_sequence.insert(
                    channel.clone(),
                    as_u64(raw).map_err(|_| "invalid field MinimumSequence")?,
                );
            }
        }
        b.finish_name()?;
        Ok(trust)
    }
}

impl Emit for Trust {
    fn emit(&self, e: &mut Emitter) {
        e.begin_object(false);
        e.field(true, "Format");
        e.int(self.format);
        e.field(false, "Prefix");
        e.string(&self.prefix);
        e.field(false, "Epoch");
        e.uint(self.epoch);
        e.field(false, "Keys");
        e.begin_object(self.keys.is_empty());
        for (i, (role, keys)) in self.keys.iter().enumerate() {
            e.field(i == 0, role);
            e.begin_array(keys.is_empty());
            for (j, key) in keys.iter().enumerate() {
                e.item(j == 0);
                e.string(key);
            }
            e.end_array(keys.is_empty());
        }
        e.end_object(self.keys.is_empty());
        e.field(false, "NotBefore");
        e.int(self.not_before);
        e.field(false, "MaxAgeSeconds");
        e.int(self.max_age_seconds);
        e.field(false, "ClockSkewSeconds");
        e.int(self.clock_skew_seconds);
        e.field(false, "MinimumSequence");
        e.begin_object(self.minimum_sequence.is_empty());
        for (i, (channel, seq)) in self.minimum_sequence.iter().enumerate() {
            e.field(i == 0, channel);
            e.uint(*seq);
        }
        e.end_object(self.minimum_sequence.is_empty());
        e.end_object(false);
    }
}

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

// ---------------------------------------------------------------------------
// Media binding (lenient decode: extra fields ignored)
// ---------------------------------------------------------------------------

#[derive(Debug, Clone, Default, PartialEq, Eq)]
pub struct MediaFile {
    pub path: String,
    pub sha256: String,
    pub bytes: i64,
}

#[derive(Debug, Clone, Default, PartialEq, Eq)]
pub struct MediaBinding {
    pub revision: String,
    pub architecture: String,
    pub host_manifest: String,
    pub payload_sha256: String,
    pub rootfs_url: String,
    pub iso: MediaFile,
    pub rootfs: MediaFile,
}

fn decode_media_file(value: &JsonValue) -> Result<MediaFile, crate::jsonx::DecodeError> {
    let soft = Soft::new(value)?;
    Ok(MediaFile {
        path: soft.string("Path")?.unwrap_or_default(),
        sha256: soft.string("SHA256")?.unwrap_or_default(),
        bytes: as_i64(soft.integer("Bytes")?.unwrap_or(0))?,
    })
}

impl MediaBinding {
    pub fn decode_lenient(data: &[u8]) -> Result<MediaBinding, crate::jsonx::DecodeError> {
        let value = parse_lenient(data)?;
        let soft = Soft::new(&value)?;
        let iso = match soft.field("ISO") {
            Some(v) => decode_media_file(v)?,
            None => MediaFile::default(),
        };
        let rootfs = match soft.field("Rootfs") {
            Some(v) => decode_media_file(v)?,
            None => MediaFile::default(),
        };
        Ok(MediaBinding {
            revision: soft.string("Revision")?.unwrap_or_default(),
            architecture: soft.string("Architecture")?.unwrap_or_default(),
            host_manifest: soft.string("HostManifest")?.unwrap_or_default(),
            payload_sha256: soft.string("PayloadSHA256")?.unwrap_or_default(),
            rootfs_url: soft.string("RootfsURL")?.unwrap_or_default(),
            iso,
            rootfs,
        })
    }
}

pub(crate) fn valid_media_file(f: &MediaFile) -> bool {
    !f.path.is_empty()
        && !f.path.contains("..")
        && !f.path.bytes().any(|b| matches!(b, b'\n' | b'\r' | 0))
        && is_digest(&f.sha256)
        && f.bytes > 0
}

pub(crate) fn valid_media_url(url: &str) -> bool {
    !url.is_empty()
        && !url.bytes().any(|b| matches!(b, b'\n' | b'\r' | 0 | b' '))
        && (url.starts_with("https://") || url.starts_with("http://"))
}

pub(crate) fn valid_media_binding(m: &MediaBinding, p: &Payload, c: &Candidate) -> bool {
    if m.revision != p.revision
        || m.architecture != p.architecture
        || m.host_manifest != c.host.manifest
        || m.payload_sha256 != c.payload_sha256
    {
        return false;
    }
    valid_media_file(&m.iso) && valid_media_file(&m.rootfs) && valid_media_url(&m.rootfs_url)
}

// ---------------------------------------------------------------------------
// Release
// ---------------------------------------------------------------------------

#[derive(Debug, Clone, Default, PartialEq, Eq)]
pub struct Release {
    pub format: i64,
    pub serial: u64,
    pub class: String,
    pub payload: Vec<u8>,
    pub candidate: Vec<u8>,
    pub media: Vec<u8>,
    pub provenance: BTreeMap<String, String>,
    pub qualification: String,
    pub evidence: BTreeMap<String, String>,
    pub notes: String,
}

fn valid_release_identity(r: &Release) -> bool {
    r.format == 1
        && r.serial != 0
        && (r.class == "normal" || r.class == "emergency")
        && (r.qualification == "local-only" || r.qualification == "native-install-upgrade-recovery")
}

fn valid_release_notes_and_evidence(r: &Release) -> bool {
    !r.evidence.is_empty()
        && r.evidence.len() <= 64
        && !r.notes.is_empty()
        && r.notes.len() <= 16384
}

fn decode_release_payloads(r: &Release) -> Result<(Payload, Candidate), Error> {
    let payload_value = parse_strict(&r.payload)?;
    let candidate_value = parse_strict(&r.candidate)?;
    let p = Payload::decode(&payload_value).map_err(|_| Error::refused())?;
    let c = Candidate::decode(&candidate_value).map_err(|_| Error::refused())?;
    Ok((p, c))
}

fn decode_release_media(r: &Release, p: &Payload, c: &Candidate) -> Result<(), Error> {
    if r.media.is_empty() || r.media.len() > 1 << 20 {
        return Err(Error::refused());
    }
    let binding = MediaBinding::decode_lenient(&r.media).map_err(|_| Error::refused())?;
    if !valid_media_binding(&binding, p, c) {
        return Err(Error::refused());
    }
    Ok(())
}

fn valid_release_provenance(r: &Release, p: &Payload, c: &Candidate) -> bool {
    if r.provenance.len() != 5 {
        return false;
    }
    for name in [
        "source.tar",
        "forgejo-source.tar",
        "app-inputs.json",
        "packages.txt",
        "presentation.json",
    ] {
        match r.provenance.get(name) {
            Some(hash) if is_digest_ref(hash) => {}
            _ => return false,
        }
    }
    r.provenance.get("packages.txt").map(String::as_str)
        == Some(format!("sha256:{}", p.host_packages_sha256).as_str())
        && r.provenance.get("presentation.json").map(String::as_str)
            == Some(format!("sha256:{}", p.presentation_sha256).as_str())
        && r.provenance.get("forgejo-source.tar").map(String::as_str)
            == Some(format!("sha256:{}", c.forgejo_source_sha256).as_str())
}

fn valid_release_evidence(evidence: &BTreeMap<String, String>) -> bool {
    for (name, hash) in evidence {
        if name.is_empty()
            || name.len() > 128
            || name.bytes().any(|b| matches!(b, b'\n' | b'\r' | 0))
            || !is_digest_ref(hash)
        {
            return false;
        }
    }
    true
}

impl Release {
    pub fn validate(&self, t: &Trust) -> Result<(Payload, Candidate), Error> {
        if !valid_release_identity(self) || !valid_release_notes_and_evidence(self) {
            return Err(Error::refused());
        }
        let (p, c) = decode_release_payloads(self)?;
        if p.repository_prefix != t.prefix || c.validate(&p, &self.payload).is_err() {
            return Err(Error::refused());
        }
        if !valid_release_provenance(self, &p, &c)
            || !valid_release_evidence(&self.evidence)
            || decode_release_media(self, &p, &c).is_err()
        {
            return Err(Error::refused());
        }
        Ok((p, c))
    }

    pub fn references(&self, t: &Trust) -> Result<Vec<String>, Error> {
        let (p, c) = self.validate(t)?;
        let mut refs = vec![c.host_reference.clone()];
        for name in NAMES {
            match p.images.get(name) {
                Some(image) => refs.push(image.reference.clone()),
                None => return Err(Error::refused()),
            }
        }
        Ok(refs)
    }

    pub fn decode(value: &JsonValue) -> Result<Release, String> {
        let mut b = Binder::new(value).map_err(|_| "invalid release".to_string())?;
        let release = Release {
            format: decode_opt_i64(&mut b, "Format")?,
            serial: decode_opt_u64(&mut b, "Serial")?,
            class: decode_opt_string(&mut b, "Class")?,
            payload: b
                .bytes("Payload")
                .map_err(|_| "invalid field Payload".to_string())?
                .unwrap_or_default(),
            candidate: b
                .bytes("Candidate")
                .map_err(|_| "invalid field Candidate".to_string())?
                .unwrap_or_default(),
            media: b
                .bytes("Media")
                .map_err(|_| "invalid field Media".to_string())?
                .unwrap_or_default(),
            provenance: decode_string_map(&mut b, "Provenance")?,
            qualification: decode_opt_string(&mut b, "Qualification")?,
            evidence: decode_string_map(&mut b, "Evidence")?,
            notes: decode_opt_string(&mut b, "Notes")?,
        };
        b.finish_name()?;
        Ok(release)
    }
}

impl Emit for Release {
    fn emit(&self, e: &mut Emitter) {
        e.begin_object(false);
        e.field(true, "Format");
        e.int(self.format);
        e.field(false, "Serial");
        e.uint(self.serial);
        e.field(false, "Class");
        e.string(&self.class);
        e.field(false, "Payload");
        e.bytes(&self.payload);
        e.field(false, "Candidate");
        e.bytes(&self.candidate);
        e.field(false, "Media");
        e.bytes(&self.media);
        e.field(false, "Provenance");
        e.begin_object(self.provenance.is_empty());
        for (i, (name, hash)) in self.provenance.iter().enumerate() {
            e.field(i == 0, name);
            e.string(hash);
        }
        e.end_object(self.provenance.is_empty());
        e.field(false, "Qualification");
        e.string(&self.qualification);
        e.field(false, "Evidence");
        e.begin_object(self.evidence.is_empty());
        for (i, (name, hash)) in self.evidence.iter().enumerate() {
            e.field(i == 0, name);
            e.string(hash);
        }
        e.end_object(self.evidence.is_empty());
        e.field(false, "Notes");
        e.string(&self.notes);
        e.end_object(false);
    }
}

// ---------------------------------------------------------------------------
// Channel / Highwater
// ---------------------------------------------------------------------------

#[derive(Debug, Clone, Default, PartialEq, Eq)]
pub struct Channel {
    pub format: i64,
    pub name: String,
    pub sequence: u64,
    pub issued: i64,
    pub expires: i64,
    pub withdrawn: bool,
    pub releases: BTreeMap<String, String>,
}

#[derive(Debug, Clone, Default, PartialEq, Eq)]
pub struct Seen {
    pub sequence: u64,
    pub digest: String,
    pub issued: i64,
}

#[derive(Debug, Clone, Default, PartialEq, Eq)]
pub struct Highwater {
    pub format: i64,
    pub trust_epoch: u64,
    pub checked_at: i64,
    pub channels: BTreeMap<String, Seen>,
    pub serials: BTreeMap<String, u64>,
    pub releases: BTreeMap<String, String>,
}

impl Channel {
    pub fn decode(value: &JsonValue) -> Result<Channel, String> {
        let mut b = Binder::new(value).map_err(|_| "invalid channel".to_string())?;
        let channel = Channel {
            format: decode_opt_i64(&mut b, "Format")?,
            name: decode_opt_string(&mut b, "Name")?,
            sequence: decode_opt_u64(&mut b, "Sequence")?,
            issued: decode_opt_i64(&mut b, "Issued")?,
            expires: decode_opt_i64(&mut b, "Expires")?,
            withdrawn: decode_opt_bool(&mut b, "Withdrawn")?,
            releases: decode_string_map(&mut b, "Releases")?,
        };
        b.finish_name()?;
        Ok(channel)
    }
}

impl Emit for Channel {
    fn emit(&self, e: &mut Emitter) {
        e.begin_object(false);
        e.field(true, "Format");
        e.int(self.format);
        e.field(false, "Name");
        e.string(&self.name);
        e.field(false, "Sequence");
        e.uint(self.sequence);
        e.field(false, "Issued");
        e.int(self.issued);
        e.field(false, "Expires");
        e.int(self.expires);
        e.field(false, "Withdrawn");
        e.boolean(self.withdrawn);
        e.field(false, "Releases");
        // Withdrawn channels carry a nil map in Go; an empty map only ever
        // arises from withdrawal, so empty marshals as `null` like the owner.
        if self.releases.is_empty() {
            e.null();
        } else {
            e.begin_object(false);
            for (i, (arch, reference)) in self.releases.iter().enumerate() {
                e.field(i == 0, arch);
                e.string(reference);
            }
            e.end_object(false);
        }
        e.end_object(false);
    }
}

impl Seen {
    pub fn decode(value: &JsonValue) -> Result<Seen, String> {
        let mut b = Binder::new(value).map_err(|_| "invalid seen".to_string())?;
        let seen = Seen {
            sequence: decode_opt_u64(&mut b, "Sequence")?,
            digest: decode_opt_string(&mut b, "Digest")?,
            issued: decode_opt_i64(&mut b, "Issued")?,
        };
        b.finish_name()?;
        Ok(seen)
    }
}

impl Emit for Seen {
    fn emit(&self, e: &mut Emitter) {
        e.begin_object(false);
        e.field(true, "Sequence");
        e.uint(self.sequence);
        e.field(false, "Digest");
        e.string(&self.digest);
        e.field(false, "Issued");
        e.int(self.issued);
        e.end_object(false);
    }
}

/// `EmptyState`: fresh high-water mark.
pub fn empty_state() -> Highwater {
    Highwater {
        format: 1,
        ..Highwater::default()
    }
}

fn valid_highwater_maps(s: &Highwater) -> bool {
    s.channels.len() <= 3 && s.serials.len() <= 2 && s.releases.len() == s.serials.len()
}

fn valid_highwater_channels(s: &Highwater) -> bool {
    for (channel, seen) in &s.channels {
        if !is_channel(channel)
            || seen.sequence == 0
            || !is_digest_ref(&seen.digest)
            || seen.issued <= 0
        {
            return false;
        }
    }
    true
}

fn valid_highwater_serials(s: &Highwater) -> bool {
    for (arch, serial) in &s.serials {
        if oci_architecture(arch).is_err() || *serial == 0 {
            return false;
        }
        match s.releases.get(arch) {
            Some(digest) if is_digest_ref(digest) => {}
            _ => return false,
        }
    }
    true
}

impl Highwater {
    pub fn validate(&self) -> Result<(), Error> {
        if self.format != 1 || !valid_highwater_maps(self) || self.checked_at < 0 {
            return Err(Error::refused());
        }
        if !valid_highwater_channels(self) || !valid_highwater_serials(self) {
            return Err(Error::refused());
        }
        Ok(())
    }

    pub fn decode(value: &JsonValue) -> Result<Highwater, String> {
        let mut b = Binder::new(value).map_err(|_| "invalid highwater".to_string())?;
        let mut state = Highwater {
            format: decode_opt_i64(&mut b, "Format")?,
            trust_epoch: decode_opt_u64(&mut b, "TrustEpoch")?,
            checked_at: decode_opt_i64(&mut b, "CheckedAt")?,
            channels: BTreeMap::new(),
            serials: BTreeMap::new(),
            releases: BTreeMap::new(),
        };
        if let Some(entries) = b
            .entries("Channels")
            .map_err(|_| "invalid field Channels".to_string())?
        {
            for (channel, item) in entries {
                state.channels.insert(channel.clone(), Seen::decode(item)?);
            }
        }
        if let Some(entries) = b
            .entries("Serials")
            .map_err(|_| "invalid field Serials".to_string())?
        {
            for (arch, item) in entries {
                let raw = item
                    .as_integer()
                    .ok_or_else(|| "invalid field Serials".to_string())?;
                state.serials.insert(
                    arch.clone(),
                    as_u64(raw).map_err(|_| "invalid field Serials".to_string())?,
                );
            }
        }
        state.releases = decode_string_map(&mut b, "Releases")?;
        b.finish_name()?;
        Ok(state)
    }
}

impl Emit for Highwater {
    fn emit(&self, e: &mut Emitter) {
        e.begin_object(false);
        e.field(true, "Format");
        e.int(self.format);
        e.field(false, "TrustEpoch");
        e.uint(self.trust_epoch);
        e.field(false, "CheckedAt");
        e.int(self.checked_at);
        e.field(false, "Channels");
        e.begin_object(self.channels.is_empty());
        for (i, (channel, seen)) in self.channels.iter().enumerate() {
            e.field(i == 0, channel);
            seen.emit(e);
        }
        e.end_object(self.channels.is_empty());
        e.field(false, "Serials");
        e.begin_object(self.serials.is_empty());
        for (i, (arch, serial)) in self.serials.iter().enumerate() {
            e.field(i == 0, arch);
            e.uint(*serial);
        }
        e.end_object(self.serials.is_empty());
        e.field(false, "Releases");
        e.begin_object(self.releases.is_empty());
        for (i, (arch, digest)) in self.releases.iter().enumerate() {
            e.field(i == 0, arch);
            e.string(digest);
        }
        e.end_object(self.releases.is_empty());
        e.end_object(false);
    }
}

fn validate_release_reference(t: &Trust, arch: &str, reference: &str) -> Result<(), Error> {
    if oci_architecture(arch).is_err() {
        return Err(Error::refused());
    }
    let (repo, role) = t.reference(reference)?;
    if role != "artifact" || repo != format!("{}-release", t.prefix) {
        return Err(Error::refused());
    }
    Ok(())
}

fn validate_channel_releases(t: &Trust, c: &Channel) -> Result<(), Error> {
    if c.withdrawn {
        if !c.releases.is_empty() {
            return Err(Error::refused());
        }
        return Ok(());
    }
    if c.releases.is_empty() || c.releases.len() > 2 {
        return Err(Error::refused());
    }
    for (arch, reference) in &c.releases {
        validate_release_reference(t, arch, reference)?;
    }
    Ok(())
}

fn validate_channel_identity(
    t: &Trust,
    c: &Channel,
    digest: &str,
    wanted: &str,
) -> Result<(), Error> {
    if t.validate().is_err() || !is_channel(wanted) || !is_digest_ref(digest) {
        return Err(Error::refused());
    }
    if c.format != 1
        || c.name != wanted
        || c.sequence < t.minimum_sequence.get(wanted).copied().unwrap_or(0)
    {
        return Err(Error::refused());
    }
    Ok(())
}

fn validate_channel_timing(
    t: &Trust,
    s: &Highwater,
    c: &Channel,
    now_unix: i64,
) -> Result<(), Error> {
    if now_unix < t.not_before || now_unix < s.checked_at - t.clock_skew_seconds {
        return Err(Error::refused());
    }
    if c.issued < t.not_before || c.issued > now_unix + t.clock_skew_seconds {
        return Err(Error::refused());
    }
    if c.expires <= now_unix || c.expires <= c.issued || c.expires - c.issued > t.max_age_seconds {
        return Err(Error::refused());
    }
    Ok(())
}

fn validate_channel_progression(old: &Seen, c: &Channel, digest: &str) -> Result<(), Error> {
    if c.sequence < old.sequence || c.issued < old.issued {
        return Err(Error::refused());
    }
    if c.sequence == old.sequence && digest != old.digest {
        return Err(Error::refused());
    }
    Ok(())
}

/// `AdmitChannel`: pure channel admission advancing the high-water mark.
pub fn admit_channel(
    t: &Trust,
    s: &Highwater,
    c: &Channel,
    digest: &str,
    wanted: &str,
    now_unix: i64,
) -> Result<Highwater, Error> {
    if s.validate().is_err() || s.trust_epoch > t.epoch {
        return Err(Error::refused());
    }
    validate_channel_identity(t, c, digest, wanted)?;
    validate_channel_timing(t, s, c, now_unix)?;
    validate_channel_releases(t, c)?;
    let old = s.channels.get(wanted).cloned().unwrap_or_default();
    validate_channel_progression(&old, c, digest)?;
    let mut next = s.clone();
    next.trust_epoch = t.epoch;
    next.checked_at = s.checked_at.max(now_unix);
    next.channels.insert(
        wanted.to_string(),
        Seen {
            sequence: c.sequence,
            digest: digest.to_string(),
            issued: c.issued,
        },
    );
    Ok(next)
}

fn admit_channel_ref(c: &Channel, arch: &str, reference: &str) -> bool {
    is_channel(&c.name)
        && c.format == 1
        && !c.withdrawn
        && c.releases.get(arch).map(String::as_str) == Some(reference)
}

fn admit_release_digest(
    s: &Highwater,
    arch: &str,
    reference: &str,
    r: &Release,
) -> Result<String, Error> {
    let digest = reference.split('@').nth(1).unwrap_or("").to_string();
    let serial = s.serials.get(arch).copied().unwrap_or(0);
    if r.serial < serial
        || (r.serial == serial && s.releases.get(arch).map(String::as_str) != Some(digest.as_str()))
    {
        return Err(Error::refused());
    }
    Ok(digest)
}

/// `AdmitRelease`: pure release admission advancing serials.
pub fn admit_release(
    t: &Trust,
    s: &Highwater,
    c: &Channel,
    arch: &str,
    reference: &str,
    r: &Release,
) -> Result<Highwater, Error> {
    let (p, _) = r.validate(t)?;
    if s.validate().is_err() || !admit_channel_ref(c, arch, reference) {
        return Err(Error::refused());
    }
    if p.architecture != arch {
        return Err(Error::refused());
    }
    if c.name != "candidate" && r.qualification != "native-install-upgrade-recovery" {
        return Err(Error::refused());
    }
    let digest = admit_release_digest(s, arch, reference, r)?;
    let mut next = s.clone();
    next.serials.insert(arch.to_string(), r.serial);
    next.releases.insert(arch.to_string(), digest);
    Ok(next)
}

// ---------------------------------------------------------------------------
// Permit
// ---------------------------------------------------------------------------

#[derive(Debug, Clone, Default, PartialEq, Eq)]
pub struct Permit {
    pub format: i64,
    pub repository: String,
    pub digest: String,
    pub previous: String,
    pub expires: i64,
}

impl Permit {
    pub fn validate(&self, t: &Trust, now_unix: i64) -> Result<(), Error> {
        if t.role(&self.repository).is_err()
            || self.format != 1
            || !is_digest_ref(&self.digest)
            || self.expires <= now_unix
            || self.expires - now_unix > 86400
        {
            return Err(Error::refused());
        }
        Ok(())
    }

    pub fn decode(value: &JsonValue) -> Result<Permit, String> {
        let mut b = Binder::new(value).map_err(|_| "invalid permit".to_string())?;
        let permit = Permit {
            format: decode_opt_i64(&mut b, "Format")?,
            repository: decode_opt_string(&mut b, "Repository")?,
            digest: decode_opt_string(&mut b, "Digest")?,
            previous: decode_opt_string(&mut b, "Previous")?,
            expires: decode_opt_i64(&mut b, "Expires")?,
        };
        b.finish_name()?;
        Ok(permit)
    }
}

impl Emit for Permit {
    fn emit(&self, e: &mut Emitter) {
        e.begin_object(false);
        e.field(true, "Format");
        e.int(self.format);
        e.field(false, "Repository");
        e.string(&self.repository);
        e.field(false, "Digest");
        e.string(&self.digest);
        e.field(false, "Previous");
        e.string(&self.previous);
        e.field(false, "Expires");
        e.int(self.expires);
        e.end_object(false);
    }
}

#[cfg(test)]
mod tests;
