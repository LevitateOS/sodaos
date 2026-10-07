use std::fs::File;
use std::io::{Read, Write};

use crate::structured::Value as JsonValue;
use serde::Serialize;

use crate::error::Error;
use crate::files::{self, OwnedDir};
use crate::jsonio;

use super::redaction::RedactOut;
use super::{contains_slice, longest_secret, Evidence, RedactingWriter, EVIDENCE_LIMIT};

/// Secret-scan block size.
const SCAN_BLOCK: usize = 32_768;
const SECRET_PATTERN_COUNT_LIMIT: usize = 16_384;

struct BoundedJsonBuffer(Vec<u8>);

impl Write for BoundedJsonBuffer {
    fn write(&mut self, bytes: &[u8]) -> std::io::Result<usize> {
        if self
            .0
            .len()
            .checked_add(bytes.len())
            .is_none_or(|size| size > EVIDENCE_LIMIT as usize)
        {
            return Err(std::io::Error::new(
                std::io::ErrorKind::WriteZero,
                "structured evidence limit exceeded",
            ));
        }
        self.0.extend_from_slice(bytes);
        Ok(bytes.len())
    }

    fn flush(&mut self) -> std::io::Result<()> {
        Ok(())
    }
}

/// Create a fresh private evidence directory that must not exist yet.
/// Mirrors `CreateEvidence`, including the JSON-escaped secret variants.
pub fn create_evidence(path: &str, secrets: &[Vec<u8>]) -> Result<Evidence, Error> {
    if !path.starts_with('/') {
        return Err(Error::msg("absolute fresh evidence directory required"));
    }
    let parent = std::path::Path::new(path)
        .parent()
        .unwrap_or(std::path::Path::new("/"));
    let parent_text = parent.to_string_lossy();
    let resolved = std::fs::canonicalize(parent)?;
    if resolved.to_string_lossy() != files::lexical_clean(&parent_text) {
        return Err(Error::msg("evidence parent must not contain symlinks"));
    }
    let raw_path = std::ffi::CString::new(path)
        .map_err(|_| Error::msg("absolute fresh evidence directory required"))?;
    let rc = unsafe { libc::mkdir(raw_path.as_ptr(), 0o700) };
    if rc != 0 {
        return Err(Error::from(std::io::Error::last_os_error()));
    }
    let root = OwnedDir::open(path)?;
    let mut kept: Vec<Vec<u8>> = Vec::new();
    let mut pattern_bytes = 0usize;
    for secret in secrets {
        if secret.is_empty() {
            continue;
        }
        // Escaping can expand controls sixfold. Refuse before making the
        // lossy UTF-8 copy or escaped String, keeping admission bounded.
        let escaped_bound = secret
            .len()
            .checked_mul(6)
            .ok_or_else(|| Error::msg("evidence redaction pattern limit exceeded"))?;
        if pattern_bytes
            .checked_add(secret.len())
            .and_then(|n| n.checked_add(escaped_bound))
            .is_none_or(|n| n > EVIDENCE_LIMIT as usize)
        {
            return Err(Error::msg("evidence redaction pattern limit exceeded"));
        }
        if !kept.iter().any(|p| p.as_slice() == secret.as_slice()) {
            pattern_bytes += secret.len();
            kept.push(secret.clone());
        }
        // Logs may contain JSON-escaped credentials rather than raw values.
        let mut encoded = String::new();
        crate::jsonio::escape_go(&mut encoded, &String::from_utf8_lossy(secret));
        let inner = &encoded[1..encoded.len() - 1];
        if inner.as_bytes() != secret.as_slice() && !kept.iter().any(|p| p == inner.as_bytes()) {
            pattern_bytes = pattern_bytes
                .checked_add(inner.len())
                .ok_or_else(|| Error::msg("evidence redaction pattern limit exceeded"))?;
            if pattern_bytes > EVIDENCE_LIMIT as usize || kept.len() >= SECRET_PATTERN_COUNT_LIMIT {
                return Err(Error::msg("evidence redaction pattern limit exceeded"));
            }
            kept.push(inner.as_bytes().to_vec());
        }
        if kept.len() > SECRET_PATTERN_COUNT_LIMIT || pattern_bytes > EVIDENCE_LIMIT as usize {
            return Err(Error::msg("evidence redaction pattern limit exceeded"));
        }
    }
    kept.sort_by_key(|a| std::cmp::Reverse(a.len()));
    Ok(Evidence {
        root,
        secrets: kept,
    })
}

fn valid_evidence_name(name: &str) -> bool {
    !name.starts_with('/')
        && files::lexical_clean(name) == name
        && name != "."
        && name != ".."
        && !name.starts_with("../")
}

impl Evidence {
    /// Original root path, for display and disjointness checks.
    pub fn path(&self) -> &str {
        self.root.path()
    }

    /// Secret set, longest first. For command capture wiring.
    pub fn secrets(&self) -> &[Vec<u8>] {
        &self.secrets
    }

    fn mkdir_evidence_parent(&self, current: &str) -> Result<(), Error> {
        match self.root.mkdir_at(current, 0o700) {
            Ok(()) => {}
            Err(e) if e.io_kind() == Some(std::io::ErrorKind::AlreadyExists) => {}
            Err(e) => return Err(e),
        }
        let attr = self.root.lstat_at(current)?;
        if !attr.is_dir || attr.is_symlink {
            return Err(Error::msg("unsafe evidence parent"));
        }
        Ok(())
    }

    fn ensure_evidence_parents(&self, dir: &str) -> Result<(), Error> {
        if dir == "." {
            return Ok(());
        }
        let mut current = String::new();
        for part in dir.split('/') {
            if !current.is_empty() {
                current.push('/');
            }
            current.push_str(part);
            self.mkdir_evidence_parent(&current)?;
        }
        Ok(())
    }

    fn open(&self, name: &str) -> Result<File, Error> {
        if !valid_evidence_name(name) {
            return Err(Error::msg("invalid evidence name"));
        }
        let dir = match name.rsplit_once('/') {
            Some((parent, _)) => parent,
            None => ".",
        };
        self.ensure_evidence_parents(dir)?;
        self.root.create_new_at(name, 0o600)
    }

    /// Exclusive redacting stream for one evidence entry.
    pub fn writer(&self, name: &str) -> Result<RedactingWriter, Error> {
        Ok(RedactingWriter::new(
            RedactOut::file(self.open(name)?),
            self.secrets.clone(),
        ))
    }

    /// Exclusive raw file for command capture tees.
    pub(crate) fn open_file(&self, name: &str) -> Result<File, Error> {
        self.open(name)
    }

    /// Write one complete redacted entry.
    pub fn write(&self, name: &str, data: &[u8]) -> Result<(), Error> {
        let mut writer = self.writer(name)?;
        // Like Go, the entry always closes so retained failures join.
        let write_err = writer.write_all(data).err().map(Error::from);
        let close_err = writer.close().err();
        match Error::join(vec![write_err, close_err]) {
            Some(err) => Err(err),
            None => Ok(()),
        }
    }

    /// Scrub a decoded value: strings redacted, keys redacted with
    /// collision detection, numbers/booleans/null preserved. Mirrors
    /// `scrubJSON` (decoded with number identity, like `UseNumber`).
    pub fn scrub_json(&self, value: &JsonValue) -> Result<JsonValue, Error> {
        match value {
            JsonValue::Str(s) => Ok(JsonValue::Str(self.try_redact_string(s)?)),
            JsonValue::Array(items) => {
                let mut out = Vec::with_capacity(items.len());
                for item in items {
                    out.push(self.scrub_json(item)?);
                }
                Ok(JsonValue::Array(out))
            }
            JsonValue::Object(entries) => {
                let mut out = Vec::with_capacity(entries.len());
                for (key, item) in entries {
                    let scrubbed_key = self.try_redact_string(key)?;
                    if out.iter().any(|(k, _)| k == &scrubbed_key) {
                        return Err(Error::msg("redacted JSON key collision"));
                    }
                    out.push((scrubbed_key, self.scrub_json(item)?));
                }
                Ok(JsonValue::Object(out))
            }
            _ => Ok(value.clone()),
        }
    }

    fn scrub_json_bounded(
        &self,
        value: &JsonValue,
        budget: &mut usize,
    ) -> Result<JsonValue, Error> {
        match value {
            JsonValue::Str(s) => {
                let scrubbed = self.try_redact_string(s)?;
                *budget = budget
                    .checked_sub(scrubbed.len())
                    .ok_or_else(|| Error::msg("structured evidence limit exceeded"))?;
                Ok(JsonValue::Str(scrubbed))
            }
            JsonValue::Array(items) => Ok(JsonValue::Array(
                items
                    .iter()
                    .map(|item| self.scrub_json_bounded(item, budget))
                    .collect::<Result<_, _>>()?,
            )),
            JsonValue::Object(entries) => {
                let mut out = Vec::with_capacity(entries.len());
                for (key, item) in entries {
                    let scrubbed_key = self.try_redact_string(key)?;
                    *budget = budget
                        .checked_sub(scrubbed_key.len())
                        .ok_or_else(|| Error::msg("structured evidence limit exceeded"))?;
                    if out.iter().any(|(k, _)| k == &scrubbed_key) {
                        return Err(Error::msg("redacted JSON key collision"));
                    }
                    out.push((scrubbed_key, self.scrub_json_bounded(item, budget)?));
                }
                Ok(JsonValue::Object(out))
            }
            _ => Ok(value.clone()),
        }
    }

    /// Confined evidence root for hash walks over retained files.
    pub fn root(&self) -> &OwnedDir {
        &self.root
    }

    fn encode_scrubbed_json(&self, value: &JsonValue) -> Result<Vec<u8>, Error> {
        value
            .validate_depth()
            .map_err(|_| Error::msg("structured evidence limit exceeded"))?;
        // Estimate compact serialization before creating its temporary String.
        let mut estimate = 0usize;
        if !estimate_json(value, &mut estimate) || estimate > EVIDENCE_LIMIT as usize {
            return Err(Error::msg("structured evidence limit exceeded"));
        }
        let mut compact = String::new();
        jsonio::write_compact(&mut compact, value);
        if compact.len() as u64 > EVIDENCE_LIMIT {
            return Err(Error::msg("structured evidence limit exceeded"));
        }
        let decoded = JsonValue::parse(&compact)
            .map_err(|_| Error::msg("structured evidence limit exceeded"))?;
        let scrubbed = self.scrub_json_bounded(&decoded, &mut (EVIDENCE_LIMIT as usize))?;
        // Go re-encodes a decoded map, so keys render sorted.
        let sorted = sort_json_keys(scrubbed);
        let mut pretty_estimate = 0usize;
        if !estimate_pretty_json(&sorted, &mut pretty_estimate, 0)
            || pretty_estimate.saturating_add(1) > EVIDENCE_LIMIT as usize
        {
            return Err(Error::msg("structured evidence limit exceeded"));
        }
        let mut data = String::new();
        jsonio::write_indent(&mut data, &sorted);
        data.push('\n');
        if data.len() as u64 > EVIDENCE_LIMIT {
            return Err(Error::msg("structured evidence limit exceeded"));
        }
        Ok(data.into_bytes())
    }

    fn encode_scrubbed_record<T: Serialize + ?Sized>(&self, value: &T) -> Result<Vec<u8>, Error> {
        let mut bounded = BoundedJsonBuffer(Vec::new());
        {
            let mut serializer = serde_json::Serializer::new(&mut bounded);
            value
                .serialize(&mut serializer)
                .map_err(|_| Error::msg("structured evidence limit exceeded"))?;
        }
        let compact = std::str::from_utf8(&bounded.0)
            .map_err(|_| Error::msg("structured evidence limit exceeded"))?;
        let decoded = JsonValue::parse(compact)
            .map_err(|_| Error::msg("structured evidence limit exceeded"))?;
        drop(bounded);
        self.encode_scrubbed_json(&decoded)
    }

    /// Write one structured value: scrubbed before encoding, then scanned
    /// for secrets before the bytes are retained. Mirrors `WriteJSON`.
    pub fn write_json<T: Serialize + ?Sized>(&self, name: &str, value: &T) -> Result<(), Error> {
        let data = self.encode_scrubbed_record(value)?;
        for secret in &self.secrets {
            if contains_slice(&data, secret) {
                return Err(Error::msg("secret reached structured evidence"));
            }
        }
        let file = self.open(name)?;
        write_and_sync(file, &data)
    }

    /// Finalize one observation: pending record first, leak scan, then an
    /// exclusive link to the final name. Mirrors `PublishObservation`.
    pub fn publish_observation<T: Serialize + ?Sized>(&self, observation: &T) -> Result<(), Error> {
        self.write_json("observation.pending.json", observation)?;
        self.check_secrets()?;
        self.root
            .link_at("observation.pending.json", "observation.json")?;
        Ok(())
    }

    /// Re-scan every retained byte for known secrets. Defense in depth,
    /// not a claim that unknown secrets are absent.
    pub fn check_secrets(&self) -> Result<(), Error> {
        self.root.walk_files(&mut |rel: &str, regular: bool| {
            if !regular {
                return Err(Error::msg("unexpected non-regular evidence entry"));
            }
            self.scan_regular_evidence(rel)
        })
    }

    fn scan_regular_evidence(&self, name: &str) -> Result<(), Error> {
        let file = self.root.open_file_at(name)?;
        let attr = fstat_attr(&file)?;
        if !attr.is_regular {
            return Err(Error::msg("evidence entry changed to a special file"));
        }
        scan_evidence_bytes(file, &self.secrets)
    }

    /// Redact known secrets plus URL queries/fragments/userinfo.
    pub fn redact_string(&self, text: &str) -> String {
        self.try_redact_string(text).unwrap_or_default()
    }

    fn try_redact_string(&self, text: &str) -> Result<String, Error> {
        if text.len() > EVIDENCE_LIMIT as usize {
            return Err(Error::msg("evidence redaction output limit exceeded"));
        }
        let mut out = text.to_string();
        for secret in &self.secrets {
            if let Ok(pattern) = std::str::from_utf8(secret) {
                if pattern.is_empty() {
                    continue;
                }
                let mut replaced = String::new();
                let mut rest = out.as_str();
                while let Some(index) = rest.find(pattern) {
                    if replaced.len() + index + b"[REDACTED]".len() > EVIDENCE_LIMIT as usize {
                        return Err(Error::msg("evidence redaction output limit exceeded"));
                    }
                    replaced.push_str(&rest[..index]);
                    replaced.push_str("[REDACTED]");
                    rest = &rest[index + pattern.len()..];
                }
                if replaced.len() + rest.len() > EVIDENCE_LIMIT as usize {
                    return Err(Error::msg("evidence redaction output limit exceeded"));
                }
                replaced.push_str(rest);
                out = replaced;
            }
        }
        if out.len() > EVIDENCE_LIMIT as usize {
            return Err(Error::msg("evidence redaction output limit exceeded"));
        }
        let safe = super::redaction::redact_urls_checked(&out)?;
        if safe.len() > EVIDENCE_LIMIT as usize {
            return Err(Error::msg("evidence redaction output limit exceeded"));
        }
        Ok(safe)
    }

    /// Scrubbed error keeping the cause chain, like Go's `safeError`.
    pub fn redact_error(&self, err: Error) -> Error {
        Error::redacted(self.redact_string(&err.to_string()), err)
    }
}

fn estimate_json(value: &JsonValue, total: &mut usize) -> bool {
    match value {
        JsonValue::Null => estimate_add(total, 4),
        JsonValue::Bool(true) => estimate_add(total, 4),
        JsonValue::Bool(false) => estimate_add(total, 5),
        JsonValue::Number(n) => estimate_add(total, n.len()),
        JsonValue::Str(s) => estimate_string(s, total),
        JsonValue::Array(items) => {
            if !estimate_add(total, 2) {
                return false;
            }
            for (i, item) in items.iter().enumerate() {
                if i > 0 && !estimate_add(total, 1) {
                    return false;
                }
                if !estimate_json(item, total) {
                    return false;
                }
            }
            true
        }
        JsonValue::Object(entries) => {
            if !estimate_add(total, 2) {
                return false;
            }
            for (i, (key, item)) in entries.iter().enumerate() {
                if i > 0 && !estimate_add(total, 1) {
                    return false;
                }
                if !estimate_string(key, total)
                    || !estimate_add(total, 1)
                    || !estimate_json(item, total)
                {
                    return false;
                }
            }
            true
        }
    }
}

fn estimate_string(s: &str, total: &mut usize) -> bool {
    if !estimate_add(total, 2) {
        return false;
    }
    for ch in s.chars() {
        let len = match ch {
            '"' | '\\' | '\n' | '\r' | '\t' => 2,
            c if (c as u32) < 0x20 || matches!(c, '<' | '>' | '&' | '\u{2028}' | '\u{2029}') => 6,
            c => c.len_utf8(),
        };
        if !estimate_add(total, len) {
            return false;
        }
    }
    true
}

fn estimate_add(total: &mut usize, size: usize) -> bool {
    match total.checked_add(size) {
        Some(next) if next <= EVIDENCE_LIMIT as usize => {
            *total = next;
            true
        }
        _ => false,
    }
}

fn estimate_pretty_json(value: &JsonValue, total: &mut usize, depth: usize) -> bool {
    match value {
        JsonValue::Array(items) if !items.is_empty() => {
            if !estimate_add(total, 2) {
                return false;
            } // [\n
            let Some(child_depth) = depth.checked_add(1) else {
                return false;
            };
            for (i, item) in items.iter().enumerate() {
                if i > 0 && !estimate_add(total, 2) {
                    return false;
                } // ,\n
                let Some(padding) = child_depth.checked_mul(2) else {
                    return false;
                };
                if !estimate_add(total, padding) || !estimate_pretty_json(item, total, child_depth)
                {
                    return false;
                }
            }
            estimate_add(total, 1)
                && depth.checked_mul(2).is_some_and(|n| estimate_add(total, n))
                && estimate_add(total, 1)
        }
        JsonValue::Object(entries) if !entries.is_empty() => {
            if !estimate_add(total, 2) {
                return false;
            } // {\n
            let Some(child_depth) = depth.checked_add(1) else {
                return false;
            };
            for (i, (key, item)) in entries.iter().enumerate() {
                if i > 0 && !estimate_add(total, 2) {
                    return false;
                } // ,\n
                let Some(padding) = child_depth.checked_mul(2) else {
                    return false;
                };
                if !estimate_add(total, padding)
                    || !estimate_string(key, total)
                    || !estimate_add(total, 2) // colon and space
                    || !estimate_pretty_json(item, total, child_depth)
                {
                    return false;
                }
            }
            estimate_add(total, 1)
                && depth.checked_mul(2).is_some_and(|n| estimate_add(total, n))
                && estimate_add(total, 1)
        }
        other => estimate_json(other, total),
    }
}

/// Recursively sort object keys, like Go's map encoding.
fn sort_json_keys(value: JsonValue) -> JsonValue {
    match value {
        JsonValue::Object(entries) => {
            let mut entries: Vec<(String, JsonValue)> = entries
                .into_iter()
                .map(|(k, v)| (k, sort_json_keys(v)))
                .collect();
            entries.sort_by(|a, b| a.0.cmp(&b.0));
            // Duplicate keys keep last-wins, like Go's map decode.
            let mut unique: Vec<(String, JsonValue)> = Vec::with_capacity(entries.len());
            for (key, value) in entries.into_iter().rev() {
                if !unique.iter().any(|(k, _)| k == &key) {
                    unique.push((key, value));
                }
            }
            unique.reverse();
            JsonValue::Object(unique)
        }
        JsonValue::Array(items) => {
            JsonValue::Array(items.into_iter().map(sort_json_keys).collect())
        }
        other => other,
    }
}

fn write_and_sync(mut file: File, data: &[u8]) -> Result<(), Error> {
    // Go joins write, sync, and close errors; the file closes on drop,
    // so only the first two are observable here.
    let write_err = file.write_all(data).err().map(Error::from);
    let sync_err = file.sync_all().err().map(Error::from);
    drop(file);
    match Error::join(vec![write_err, sync_err]) {
        Some(err) => Err(err),
        None => Ok(()),
    }
}

fn fstat_attr(file: &File) -> Result<files::FileAttr, Error> {
    use std::os::fd::AsRawFd;
    let mut st: libc::stat = unsafe { std::mem::zeroed() };
    let rc = unsafe { libc::fstat(file.as_raw_fd(), &mut st) };
    if rc != 0 {
        return Err(Error::from(std::io::Error::last_os_error()));
    }
    let mode = st.st_mode;
    Ok(files::FileAttr {
        is_regular: mode & libc::S_IFMT == libc::S_IFREG,
        is_dir: mode & libc::S_IFMT == libc::S_IFDIR,
        is_symlink: mode & libc::S_IFMT == libc::S_IFLNK,
        perm: mode & 0o7777,
        size: st.st_size.max(0) as u64,
        uid: st.st_uid,
        gid: st.st_gid,
        dev: st.st_dev as u64,
        ino: st.st_ino as u64,
    })
}

fn scan_evidence_bytes(mut file: File, secrets: &[Vec<u8>]) -> Result<(), Error> {
    let max = longest_secret(secrets);
    let mut tail: Vec<u8> = Vec::new();
    let mut buf = [0u8; SCAN_BLOCK];
    loop {
        let n = file.read(&mut buf)?;
        let mut block = tail.clone();
        block.extend_from_slice(&buf[..n]);
        for secret in secrets {
            if contains_slice(&block, secret) {
                return Err(Error::msg("secret reached evidence"));
            }
        }
        tail = if block.len() >= max {
            block[block.len() - max + 1..].to_vec()
        } else {
            block
        };
        if n == 0 {
            return Ok(());
        }
    }
}
