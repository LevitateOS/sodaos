use crate::domain;
use crate::json::{self, Kind, Spec};
use crate::terminal;

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

const FACTORY_RUN_SPECS: &[Spec] = &[
    Spec {
        name: "deadline",
        kind: Kind::Str,
    },
    Spec {
        name: "actor",
        kind: Kind::I64,
    },
    Spec {
        name: "id",
        kind: Kind::Str,
    },
    Spec {
        name: "project",
        kind: Kind::Str,
    },
    Spec {
        name: "role",
        kind: Kind::Str,
    },
    Spec {
        name: "preparation",
        kind: Kind::Str,
    },
    Spec {
        name: "harness",
        kind: Kind::Str,
    },
    Spec {
        name: "harness_version",
        kind: Kind::Str,
    },
    Spec {
        name: "model",
        kind: Kind::Str,
    },
    Spec {
        name: "assignment",
        kind: Kind::Str,
    },
    Spec {
        name: "source_commit",
        kind: Kind::Str,
    },
    Spec {
        name: "connection",
        kind: Kind::Str,
    },
];

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
        let v = json::decode_strict(body).map_err(|e| e.0)?;
        let m = json::bind_root(&v, "FactoryRun", FACTORY_RUN_SPECS, false).map_err(|e| e.0)?;
        let deadline_raw = m.take_string("deadline");
        if m.contains("deadline") {
            terminal::parse_rfc3339(&deadline_raw).ok_or_else(|| "invalid deadline".to_string())?;
        }
        Ok(FactoryRun {
            deadline_raw,
            actor: m.take_i64("actor"),
            id: m.take_string("id"),
            project: m.take_string("project"),
            role: m.take_string("role"),
            preparation: m.take_string("preparation"),
            harness: m.take_string("harness"),
            harness_vers: m.take_string("harness_version"),
            model: m.take_string("model"),
            assignment: m.take_string("assignment"),
            source_commit: m.take_string("source_commit"),
            connection: m.take_string("connection"),
        })
    }
}
