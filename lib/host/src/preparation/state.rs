// Observed preparation states: resolved tools, inspect/stop/hold wire shapes.
use super::valid_preparation_id;
use crate::domain;
use crate::json;
use serde::de::{self, MapAccess, Visitor};
use serde::Deserialize;
use std::fmt;

#[derive(Debug, Clone, PartialEq, Eq, Default)]
pub struct ResolvedTool {
    pub name: String,
    pub path: String,
    pub version: String,
}

impl ResolvedTool {
    pub fn encode_into(&self, out: &mut String) {
        out.push_str("{\"name\":");
        out.push_str(&json::quote(&self.name));
        out.push_str(",\"path\":");
        out.push_str(&json::quote(&self.path));
        out.push_str(",\"version\":");
        out.push_str(&json::quote(&self.version));
        out.push('}');
    }
}

macro_rules! typed_state_object {
    ($ty:ident, $expect:literal, {$($field:ident => $name:literal : $value:ty),+ $(,)?}) => {
        impl<'de> Deserialize<'de> for $ty {
            fn deserialize<D>(deserializer: D) -> Result<Self, D::Error>
            where D: serde::Deserializer<'de> {
                struct ObjectVisitor;
                impl<'de> Visitor<'de> for ObjectVisitor {
                    type Value = $ty;
                    fn expecting(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result { f.write_str($expect) }
                    fn visit_map<A>(self, mut map: A) -> Result<Self::Value, A::Error>
                    where A: MapAccess<'de> {
                        let mut out = $ty { $($field: Default::default()),+ };
                        while let Some(key) = map.next_key::<String>()? {
                            let mut recognized = false;
                            $(if key.eq_ignore_ascii_case($name) {
                                if let Some(value) = map.next_value::<Option<$value>>()? { out.$field = value.into(); }
                                recognized = true;
                            })+
                            if !recognized { return Err(de::Error::unknown_field(&key, &[$($name),+])); }
                        }
                        Ok(out)
                    }
                }
                deserializer.deserialize_map(ObjectVisitor)
            }
        }
    };
}

typed_state_object!(ResolvedTool, "a resolved tool object", {
    name => "name": String, path => "path": String, version => "version": String
});

#[derive(Debug, Clone, PartialEq, Eq, Default)]
pub struct PrepareState {
    pub id: String,
    pub project: String,
    pub role: String,
    pub phase: String,
    pub container: String,
    pub source_commit: String,
    pub setup_digest: String,
    pub tools: Vec<ResolvedTool>,
    pub missing: String,
    pub setup_exit: Option<i64>,
    pub check_exit: Option<i64>,
    pub output: String,
    pub ready: bool,
    pub stopped: bool,
    pub retirement: String,
}

impl PrepareState {
    /// `encoding/json` struct order with `omitempty` honored.
    pub fn encode(&self) -> String {
        let mut out = String::from("{\"id\":");
        out.push_str(&json::quote(&self.id));
        out.push_str(",\"project\":");
        out.push_str(&json::quote(&self.project));
        out.push_str(",\"role\":");
        out.push_str(&json::quote(&self.role));
        out.push_str(",\"phase\":");
        out.push_str(&json::quote(&self.phase));
        out.push_str(",\"container\":");
        out.push_str(&json::quote(&self.container));
        out.push_str(",\"source_commit\":");
        out.push_str(&json::quote(&self.source_commit));
        out.push_str(",\"setup_digest\":");
        out.push_str(&json::quote(&self.setup_digest));
        if !self.tools.is_empty() {
            out.push_str(",\"tools\":[");
            for (i, tool) in self.tools.iter().enumerate() {
                if i > 0 {
                    out.push(',');
                }
                tool.encode_into(&mut out);
            }
            out.push(']');
        }
        if !self.missing.is_empty() {
            out.push_str(",\"missing\":");
            out.push_str(&json::quote(&self.missing));
        }
        if let Some(exit) = self.setup_exit {
            out.push_str(",\"setup_exit\":");
            out.push_str(&exit.to_string());
        }
        if let Some(exit) = self.check_exit {
            out.push_str(",\"check_exit\":");
            out.push_str(&exit.to_string());
        }
        if !self.output.is_empty() {
            out.push_str(",\"output\":");
            out.push_str(&json::quote(&self.output));
        }
        out.push_str(",\"ready\":");
        out.push_str(if self.ready { "true" } else { "false" });
        out.push_str(",\"stopped\":");
        out.push_str(if self.stopped { "true" } else { "false" });
        if !self.retirement.is_empty() {
            out.push_str(",\"retirement\":");
            out.push_str(&json::quote(&self.retirement));
        }
        out.push('}');
        out
    }
}

#[derive(Debug, Clone, PartialEq, Eq, Default)]
pub struct PrepareInspect {
    pub project: String,
    pub id: String,
}

/// Bounded, transient repository material read from one recorded preparation.
/// The caller supplies exact source, approved base, diff base, candidate and
/// deadline values; this value is never part of the durable preparation receipt.
#[derive(Debug, Clone, PartialEq, Eq, Default)]
pub struct PrepareContextRead {
    pub project: String,
    pub id: String,
    pub source_commit: String,
    pub approved_base: String,
    pub diff_base: String,
    pub candidate: String,
    pub paths: Vec<String>,
    pub not_after: String,
}

#[derive(Debug, Clone, PartialEq, Eq, Default)]
pub struct ContextFile {
    pub path: String,
    pub content: Vec<u8>,
}

#[derive(Debug, Clone, PartialEq, Eq, Default)]
pub struct PreparationContext {
    pub project: String,
    pub id: String,
    pub role: String,
    pub source_commit: String,
    pub approved_base: String,
    pub diff_base: String,
    pub candidate: String,
    pub setup: Vec<u8>,
    pub check: Vec<u8>,
    pub files: Vec<ContextFile>,
    pub diff: Vec<u8>,
}

pub const MAX_CONTEXT_FILES: usize = 12;
pub const MAX_CONTEXT_PATH_BYTES: usize = 256;
pub const MAX_CONTEXT_FILE_BYTES: usize = 8 * 1024;
pub const MAX_CONTEXT_TOTAL_BYTES: usize = 32 * 1024;

impl PrepareContextRead {
    pub fn validate(&self) -> Result<(), String> {
        if !domain::valid_id(&self.project)
            || !valid_preparation_id(&self.id)
            || !domain::is_hex_lower(&self.source_commit)
            || self.source_commit.len() != 40
            || !domain::is_hex_lower(&self.approved_base)
            || self.approved_base.len() != 40
            || !domain::is_hex_lower(&self.diff_base)
            || self.diff_base.len() != 40
            || !domain::is_hex_lower(&self.candidate)
            || self.candidate.len() != 40
            || self.paths.len() > MAX_CONTEXT_FILES
            || soda_wire_time::parse_nanos(&self.not_after).is_none()
        {
            return Err("invalid preparation context request".to_string());
        }
        let mut total = 0usize;
        let mut previous: Option<&str> = None;
        for path in &self.paths {
            if !valid_context_path(path) || previous.is_some_and(|old| old >= path.as_str()) {
                return Err("invalid preparation context path set".to_string());
            }
            total = total
                .checked_add(path.len())
                .ok_or_else(|| "preparation context path set exceeds bounds".to_string())?;
            previous = Some(path);
        }
        if !self.paths.windows(2).all(|pair| pair[0] < pair[1]) {
            return Err("invalid preparation context path set".to_string());
        }
        if total > MAX_CONTEXT_FILES * MAX_CONTEXT_PATH_BYTES {
            return Err("preparation context path set exceeds bounds".to_string());
        }
        Ok(())
    }
}

pub(crate) fn valid_context_path(path: &str) -> bool {
    if path.is_empty()
        || path.len() > MAX_CONTEXT_PATH_BYTES
        || path.starts_with('/')
        || path.contains('\\')
        || path.contains('\0')
        || path.bytes().any(|byte| byte < b' ' || byte == 0x7f)
    {
        return false;
    }
    path.split('/').all(|part| {
        if part.is_empty() || part == "." || part == ".." {
            return false;
        }
        let lower = part.to_ascii_lowercase();
        !(lower == ".git"
            || lower == ".soda-home"
            || lower == ".ssh"
            || lower == ".aws"
            || lower == ".codex"
            || lower == ".muse"
            || lower == ".config"
            || lower == ".docker"
            || lower == ".kube"
            || lower == "credentials"
            || lower == "credentials.json"
            || lower == "secrets"
            || lower == "private"
            || lower == "auth.json"
            || lower == "token.json"
            || lower == ".netrc"
            || lower == ".git-credentials"
            || lower == ".npmrc"
            || lower == ".pypirc"
            || lower == ".env"
            || lower.starts_with(".env.")
            || lower == "id_rsa"
            || lower == "id_ed25519"
            || [".pem", ".key", ".p12", ".pfx"]
                .iter()
                .any(|ext| lower.ends_with(ext)))
    })
}

impl PrepareInspect {
    pub fn validate(&self) -> Result<(), String> {
        if !domain::valid_id(&self.project) || !valid_preparation_id(&self.id) {
            return Err("invalid preparation address".to_string());
        }
        Ok(())
    }

    pub fn decode(body: &[u8]) -> Result<Self, String> {
        json::decode_strict_as(body).map_err(|e| e.0)
    }
}

impl PrepareContextRead {
    pub fn decode(body: &[u8]) -> Result<Self, String> {
        json::decode_strict_as(body).map_err(|error| error.0)
    }
}

impl PreparationContext {
    /// Go-compatible JSON response; every byte field uses padded standard
    /// base64, as do the existing project preparation bundle fields.
    pub fn encode(&self) -> String {
        let mut out = String::from("{\"project\":");
        out.push_str(&json::quote(&self.project));
        out.push_str(",\"id\":");
        out.push_str(&json::quote(&self.id));
        out.push_str(",\"role\":");
        out.push_str(&json::quote(&self.role));
        out.push_str(",\"source_commit\":");
        out.push_str(&json::quote(&self.source_commit));
        out.push_str(",\"approved_base\":");
        out.push_str(&json::quote(&self.approved_base));
        out.push_str(",\"diff_base\":");
        out.push_str(&json::quote(&self.diff_base));
        out.push_str(",\"candidate\":");
        out.push_str(&json::quote(&self.candidate));
        out.push_str(",\"setup\":");
        out.push_str(&json::quote(&crate::ssh::b64_encode(&self.setup)));
        out.push_str(",\"check\":");
        out.push_str(&json::quote(&crate::ssh::b64_encode(&self.check)));
        out.push_str(",\"files\":[");
        for (index, file) in self.files.iter().enumerate() {
            if index != 0 {
                out.push(',');
            }
            out.push_str("{\"path\":");
            out.push_str(&json::quote(&file.path));
            out.push_str(",\"content\":");
            out.push_str(&json::quote(&crate::ssh::b64_encode(&file.content)));
            out.push('}');
        }
        out.push_str("],\"diff\":");
        out.push_str(&json::quote(&crate::ssh::b64_encode(&self.diff)));
        out.push('}');
        out
    }
}

#[derive(Debug, Clone, PartialEq, Eq, Default)]
pub struct PrepareStop {
    pub project: String,
    pub id: String,
}

impl PrepareStop {
    pub fn validate(&self) -> Result<(), String> {
        if !domain::valid_id(&self.project) || !valid_preparation_id(&self.id) {
            return Err("invalid preparation address".to_string());
        }
        Ok(())
    }

    pub fn decode(body: &[u8]) -> Result<Self, String> {
        json::decode_strict_as(body).map_err(|e| e.0)
    }
}

typed_state_object!(PrepareInspect, "a preparation inspection object", {
    project => "project": String, id => "id": String
});
typed_state_object!(PrepareContextRead, "a preparation context request", {
    project => "project": String,
    id => "id": String,
    source_commit => "source_commit": String,
    approved_base => "approved_base": String,
    diff_base => "diff_base": String,
    candidate => "candidate": String,
    paths => "paths": Vec<String>,
    not_after => "not_after": String
});
typed_state_object!(PrepareStop, "a preparation stop object", {
    project => "project": String, id => "id": String
});

#[derive(Debug, Clone, PartialEq, Eq, Default)]
pub struct HoldState {
    pub active: bool,
    pub revision: i64,
}

impl HoldState {
    pub fn encode(&self) -> String {
        format!(
            "{{\"active\":{},\"revision\":{}}}",
            if self.active { "true" } else { "false" },
            self.revision
        )
    }
}

typed_state_object!(HoldState, "a maintenance hold object", {
    active => "active": bool, revision => "revision": json::SignedInteger
});

#[derive(Debug, Clone, PartialEq, Eq, Default)]
pub struct PrepareHold {
    pub project: String,
    pub hold: bool,
    pub revision: i64,
}

impl PrepareHold {
    pub fn validate(&self) -> Result<(), String> {
        if !domain::valid_id(&self.project) || self.revision < 0 {
            return Err("invalid maintenance hold".to_string());
        }
        Ok(())
    }

    pub fn decode(body: &[u8]) -> Result<Self, String> {
        json::decode_strict_as(body).map_err(|e| e.0)
    }
}

typed_state_object!(PrepareHold, "a maintenance hold request object", {
    project => "project": String, hold => "hold": bool, revision => "revision": json::SignedInteger
});
