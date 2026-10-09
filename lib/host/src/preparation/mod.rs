//! Factory preparation domain: roles, approvals, bounded setup inputs and
//! observed states. Port of `internal/project/preparation.go` plus the
//! candidate half of `internal/project/factory_candidate.go` (pure
//! validation and wire shapes only; the store-side grant decisions and
//! durable records stay in Go).

use crate::domain;
use crate::json;
use serde::de::{self, MapAccess, Visitor};
use serde::Deserialize;
use std::fmt;

mod candidate;
mod decisions;
mod setup;
mod state;

pub use self::candidate::FactoryCandidate;
pub use self::decisions::{AdminApproval, RequirementAcceptance};
pub use self::setup::{setup_digest_of, ApprovedSetup};
pub use self::state::{
    ContextFile, HoldState, PreparationContext, PrepareContextRead, PrepareHold, PrepareInspect,
    PrepareState, PrepareStop, ResolvedTool, MAX_CONTEXT_FILES, MAX_CONTEXT_FILE_BYTES,
    MAX_CONTEXT_PATH_BYTES, MAX_CONTEXT_TOTAL_BYTES,
};

pub const ROLE_CODER: &str = "soda-coder";
pub const ROLE_REVIEWER: &str = "soda-reviewer";

pub const FACTORY_DIR: &str = "/var/lib/soda/factory";
pub const FACTORY_PREPARATIONS_DIR: &str = "/var/lib/soda/factory/preparations";
pub const FACTORY_CREDENTIALS_DIR: &str = "/var/lib/soda/factory/credentials";
pub const FACTORY_HOLD_FILE: &str = "/var/lib/soda/factory/maintenance-hold";
pub const FACTORY_HELPER: &str = "/usr/libexec/soda/project-factory-roles";
pub const FACTORY_SETUP_ENTRY: &str = "setup.sh";
pub const FACTORY_CHECK_ENTRY: &str = "check.sh";

pub const MAX_APPROVED_FILES: usize = 8;
pub const MAX_APPROVED_FILE_SIZE: usize = 32 * 1024;
pub const MAX_APPROVED_TOTAL: usize = 128 * 1024;
pub const MAX_SOURCE_BUNDLE: usize = 512 * 1024;
pub const MAX_PREPARE_TOOLS: usize = 8;

pub const PREPARE_APPROVED: &str = "approved";
pub const PREPARE_WAITING: &str = "waiting";
pub const PREPARE_RUNNING: &str = "running";
pub const PREPARE_READY: &str = "ready";
pub const PREPARE_FAILED: &str = "failed";
pub const PREPARE_STOPPED: &str = "stopped";
pub const PREPARE_INTERRUPTED: &str = "interrupted";

pub fn valid_factory_role(role: &str) -> bool {
    role == ROLE_CODER || role == ROLE_REVIEWER
}

pub fn valid_prepare_phase(phase: &str) -> bool {
    matches!(
        phase,
        "approved" | "waiting" | "running" | "ready" | "failed" | "stopped" | "interrupted"
    )
}

fn is_hex_lower(s: &str) -> bool {
    !s.is_empty()
        && s.bytes()
            .all(|b| b.is_ascii_hexdigit() && (b.is_ascii_digit() || b.is_ascii_lowercase()))
}

/// `^f[0-9a-f]{24}$`
pub fn valid_preparation_id(id: &str) -> bool {
    id.len() == 25 && id.as_bytes()[0] == b'f' && is_hex_lower(&id[1..])
}

/// `^d[0-9a-f]{24}$`
pub fn valid_decision_id(id: &str) -> bool {
    id.len() == 25 && id.as_bytes()[0] == b'd' && is_hex_lower(&id[1..])
}

/// `^[0-9a-f]{64}$`
pub fn valid_digest(d: &str) -> bool {
    d.len() == 64 && is_hex_lower(d)
}

/// `^[0-9a-f]{40}$`
pub fn valid_commit(c: &str) -> bool {
    c.len() == 40 && is_hex_lower(c)
}

/// `^[A-Za-z0-9][A-Za-z0-9_.-]{0,63}$` without `..`.
pub fn valid_approved_name(name: &str) -> bool {
    let b = name.as_bytes();
    if b.is_empty() || b.len() > 64 || !b[0].is_ascii_alphanumeric() {
        return false;
    }
    if !b[1..]
        .iter()
        .all(|c| c.is_ascii_alphanumeric() || *c == b'.' || *c == b'_' || *c == b'-')
    {
        return false;
    }
    !name.contains("..")
}

/// `^[a-z0-9][a-z0-9+_.-]{0,31}$`
pub fn valid_tool_name(name: &str) -> bool {
    let b = name.as_bytes();
    if b.is_empty() || b.len() > 32 || !(b[0].is_ascii_lowercase() || b[0].is_ascii_digit()) {
        return false;
    }
    b[1..].iter().all(|c| {
        c.is_ascii_lowercase() || c.is_ascii_digit() || matches!(*c, b'+' | b'_' | b'.' | b'-')
    })
}

struct GoStringList(Vec<String>);

impl From<GoStringList> for Vec<String> {
    fn from(values: GoStringList) -> Self {
        values.0
    }
}

impl<'de> Deserialize<'de> for GoStringList {
    fn deserialize<D>(deserializer: D) -> Result<Self, D::Error>
    where
        D: serde::Deserializer<'de>,
    {
        struct StringListVisitor;
        impl<'de> Visitor<'de> for StringListVisitor {
            type Value = GoStringList;
            fn expecting(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
                f.write_str("an array of strings")
            }
            fn visit_seq<A>(self, mut seq: A) -> Result<Self::Value, A::Error>
            where
                A: serde::de::SeqAccess<'de>,
            {
                let mut values = Vec::new();
                while let Some(value) = seq.next_element::<Option<String>>()? {
                    values.push(value.unwrap_or_default());
                }
                Ok(GoStringList(values))
            }
        }
        deserializer.deserialize_seq(StringListVisitor)
    }
}

#[derive(Debug, Clone, PartialEq, Eq, Default)]
pub struct Preparation {
    pub id: String,
    pub project: String,
    pub role: String,
    pub revision: i64,
    pub requirements: RequirementAcceptance,
    pub approval: AdminApproval,
    pub source_commit: String,
    pub setup_digest: String,
    pub tools: Vec<String>,
    pub credential: String,
}

impl Preparation {
    pub fn validate(&self) -> Result<(), String> {
        if !valid_preparation_id(&self.id)
            || !domain::valid_id(&self.project)
            || !valid_factory_role(&self.role)
            || self.revision < 0
        {
            return Err("invalid preparation identity".to_string());
        }
        self.requirements.validate()?;
        self.approval.validate()?;
        if !valid_commit(&self.source_commit) || !valid_digest(&self.setup_digest) {
            return Err("invalid preparation source identity".to_string());
        }
        if self.tools.len() > MAX_PREPARE_TOOLS {
            return Err("too many required tools".to_string());
        }
        for name in &self.tools {
            if !valid_tool_name(name) {
                return Err("invalid required tool name".to_string());
            }
        }
        if !self.credential.is_empty() && !valid_approved_name(&self.credential) {
            return Err("invalid credential reference".to_string());
        }
        Ok(())
    }
}

#[derive(Debug, Clone, PartialEq, Eq, Default)]
pub struct Prepare {
    pub preparation: Preparation,
    pub setup: ApprovedSetup,
}

impl Prepare {
    pub fn validate(&self) -> Result<(), String> {
        self.preparation.validate()?;
        self.setup.validate()?;
        if self.preparation.setup_digest != setup_digest_of(&self.setup.files) {
            return Err("approved inputs do not match their digest".to_string());
        }
        Ok(())
    }

    pub fn decode(body: &[u8]) -> Result<Self, String> {
        json::decode_strict_as(body).map_err(|e| e.0)
    }
}

macro_rules! typed_preparation_object {
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

typed_preparation_object!(Preparation, "a preparation object", {
    id => "id": String,
    project => "project": String,
    role => "role": String,
    revision => "revision": json::SignedInteger,
    requirements => "requirements": RequirementAcceptance,
    approval => "approval": AdminApproval,
    source_commit => "source_commit": String,
    setup_digest => "setup_digest": String,
    tools => "tools": GoStringList,
    credential => "credential": String,
});

typed_preparation_object!(Prepare, "a prepare request object", {
    preparation => "preparation": Preparation,
    setup => "setup": ApprovedSetup,
});

#[cfg(test)]
mod validation_tests;
#[cfg(test)]
mod wire_tests;
