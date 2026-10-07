use crate::domain;
use crate::json::{self, SignedInteger};
use crate::terminal;
use serde::de::{self, MapAccess, Visitor};
use serde::{Deserialize, Deserializer};

// ---------- factory domain (internal/project) ----------

/// Fixed factory execution discriminator for supervised Codex runs.
pub const FACTORY_SCOPE_CODEX: &str = "factory-codex";
/// First harness family.
pub const FACTORY_HARNESS_CODEX: &str = "codex";
/// Fixed factory execution discriminator for supervised Muse Code runs.
pub const FACTORY_SCOPE_MUSE: &str = "factory-muse";
/// Muse Code CLI harness family (`muse exec` runs backed by the native
/// "muse" provider).
pub const FACTORY_HARNESS_MUSE: &str = "muse";

/// Supported supervised CLI families.
pub fn valid_harness_family(family: &str) -> bool {
    family == FACTORY_HARNESS_CODEX || family == FACTORY_HARNESS_MUSE
}
/// Fixed factory role logins.
pub const ROLE_CODER: &str = "soda-coder";
pub const ROLE_REVIEWER: &str = "soda-reviewer";
/// Prompt byte bound (`64*1024`).
pub const MAX_FACTORY_PROMPT: usize = 64 * 1024;

/// `ValidFactoryRole`: the two fixed factory roles only.
pub fn valid_factory_role(role: &str) -> bool {
    role == ROLE_CODER || role == ROLE_REVIEWER
}

/// `ValidPreparationID = ^f[0-9a-f]{24}$`.
pub fn valid_preparation_id(id: &str) -> bool {
    id.len() == 25 && id.as_bytes()[0] == b'f' && domain::is_hex_lower(&id[1..])
}

/// `ValidDigest = ^[0-9a-f]{64}$`.
pub fn valid_digest(d: &str) -> bool {
    domain::valid_container_id(d)
}

/// `ValidCommit = ^[0-9a-f]{40}$`.
pub fn valid_commit(c: &str) -> bool {
    c.len() == 40 && domain::is_hex_lower(c)
}

/// `ValidFactoryRunID = ^[a-f0-9]{32}$`.
pub fn valid_factory_run_id(id: &str) -> bool {
    terminal::valid_terminal_id(id)
}

/// `ValidHarnessVersion = ^[A-Za-z0-9][A-Za-z0-9._-]{0,31}$`.
pub fn valid_harness_version(version: &str) -> bool {
    let b = version.as_bytes();
    !b.is_empty()
        && b.len() <= 32
        && b[0].is_ascii_alphanumeric()
        && b[1..]
            .iter()
            .all(|c| c.is_ascii_alphanumeric() || *c == b'.' || *c == b'_' || *c == b'-')
}

/// One supervised CLI execution (`project.FactoryRun`).
#[derive(Debug, Clone, Default, PartialEq, Eq)]
pub struct FactoryRun {
    pub deadline_raw: String,
    pub actor: i64,
    pub id: String,
    pub project: String,
    pub role: String,
    pub preparation: String,
    pub harness: String,
    pub harness_vers: String,
    pub model: String,
    pub assignment: String,
    pub source_commit: String,
    pub connection: String,
}

#[derive(Default)]
struct FactoryRunWire {
    deadline: Option<String>,
    actor: Option<SignedInteger>,
    id: Option<String>,
    project: Option<String>,
    role: Option<String>,
    preparation: Option<String>,
    harness: Option<String>,
    harness_version: Option<String>,
    model: Option<String>,
    assignment: Option<String>,
    source_commit: Option<String>,
    connection: Option<String>,
    deadline_seen: bool,
}
impl<'de> Deserialize<'de> for FactoryRunWire {
    fn deserialize<D>(d: D) -> Result<Self, D::Error>
    where
        D: Deserializer<'de>,
    {
        struct V;
        impl<'de> Visitor<'de> for V {
            type Value = FactoryRunWire;
            fn expecting(&self, f: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
                f.write_str("factory run")
            }
            fn visit_map<A>(self, mut map: A) -> Result<Self::Value, A::Error>
            where
                A: MapAccess<'de>,
            {
                let mut o = FactoryRunWire::default();
                while let Some(k) = map.next_key::<String>()? {
                    match k.to_ascii_lowercase().as_str() {
                        "deadline" => {
                            o.deadline_seen = true;
                            if let Some(v) = map.next_value::<Option<String>>()? {
                                o.deadline = Some(v)
                            }
                        }
                        "actor" => {
                            if let Some(v) = map.next_value::<Option<SignedInteger>>()? {
                                o.actor = Some(v)
                            }
                        }
                        "id" => {
                            if let Some(v) = map.next_value::<Option<String>>()? {
                                o.id = Some(v)
                            }
                        }
                        "project" => {
                            if let Some(v) = map.next_value::<Option<String>>()? {
                                o.project = Some(v)
                            }
                        }
                        "role" => {
                            if let Some(v) = map.next_value::<Option<String>>()? {
                                o.role = Some(v)
                            }
                        }
                        "preparation" => {
                            if let Some(v) = map.next_value::<Option<String>>()? {
                                o.preparation = Some(v)
                            }
                        }
                        "harness" => {
                            if let Some(v) = map.next_value::<Option<String>>()? {
                                o.harness = Some(v)
                            }
                        }
                        "harness_version" => {
                            if let Some(v) = map.next_value::<Option<String>>()? {
                                o.harness_version = Some(v)
                            }
                        }
                        "model" => {
                            if let Some(v) = map.next_value::<Option<String>>()? {
                                o.model = Some(v)
                            }
                        }
                        "assignment" => {
                            if let Some(v) = map.next_value::<Option<String>>()? {
                                o.assignment = Some(v)
                            }
                        }
                        "source_commit" => {
                            if let Some(v) = map.next_value::<Option<String>>()? {
                                o.source_commit = Some(v)
                            }
                        }
                        "connection" => {
                            if let Some(v) = map.next_value::<Option<String>>()? {
                                o.connection = Some(v)
                            }
                        }
                        _ => {
                            return Err(de::Error::unknown_field(
                                &k,
                                &[
                                    "deadline",
                                    "actor",
                                    "id",
                                    "project",
                                    "role",
                                    "preparation",
                                    "harness",
                                    "harness_version",
                                    "model",
                                    "assignment",
                                    "source_commit",
                                    "connection",
                                ],
                            ))
                        }
                    }
                }
                Ok(o)
            }
        }
        d.deserialize_map(V)
    }
}

impl FactoryRun {
    /// `FactoryRun.Validate()` with exact error strings.
    pub fn validate(&self) -> Result<(), String> {
        if !valid_factory_run_id(&self.id)
            || !domain::valid_id(&self.project)
            || !valid_factory_role(&self.role)
        {
            return Err("invalid factory run identity".to_string());
        }
        if !valid_preparation_id(&self.preparation) {
            return Err("invalid run preparation reference".to_string());
        }
        if !valid_harness_family(&self.harness) || !valid_harness_version(&self.harness_vers) {
            return Err("unsupported factory harness".to_string());
        }
        if !self.model.is_empty() {
            if self.model.len() > 128 {
                return Err("invalid run model selection".to_string());
            }
            // Go checks raw bytes, not runes.
            for byte in self.model.bytes() {
                if byte < 0x20 || byte == 0x7f {
                    return Err("invalid run model selection".to_string());
                }
            }
        }
        if !valid_digest(&self.assignment) || !valid_commit(&self.source_commit) {
            return Err("invalid run assignment or source identity".to_string());
        }
        if self.connection.is_empty() || self.connection.len() > 128 || self.actor <= 0 {
            return Err("invalid run sponsorship".to_string());
        }
        if self.deadline_raw.is_empty() {
            return Err("run deadline is required".to_string());
        }
        Ok(())
    }

    /// Strict decode of one run object.
    pub fn decode(body: &[u8]) -> Result<Self, String> {
        let wire: FactoryRunWire = json::decode_strict_as(body).map_err(|e| e.0)?;
        let deadline_present = wire.deadline_seen;
        let deadline_raw = wire.deadline.unwrap_or_default();
        if deadline_present {
            terminal::parse_rfc3339(&deadline_raw).ok_or_else(|| "invalid deadline".to_string())?;
        }
        Ok(FactoryRun {
            deadline_raw,
            actor: wire.actor.map(Into::into).unwrap_or_default(),
            id: wire.id.unwrap_or_default(),
            project: wire.project.unwrap_or_default(),
            role: wire.role.unwrap_or_default(),
            preparation: wire.preparation.unwrap_or_default(),
            harness: wire.harness.unwrap_or_default(),
            harness_vers: wire.harness_version.unwrap_or_default(),
            model: wire.model.unwrap_or_default(),
            assignment: wire.assignment.unwrap_or_default(),
            source_commit: wire.source_commit.unwrap_or_default(),
            connection: wire.connection.unwrap_or_default(),
        })
    }
}
