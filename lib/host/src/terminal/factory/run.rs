use crate::domain;
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
/// Reassess/repair prompt exception (`320*1024`): those prompts carry the
/// exact complete inventory inline. This mirrors the factory-domain
/// `MaxPromptForPhase` policy at its staging mirror; the admission gate
/// stays authoritative and staging never admits more than admission.
pub const MAX_REASSESS_PROMPT: usize = 320 * 1024;
/// Intent decompose prompt headroom (`128*1024`): an intent-subject
/// decompose prompt carries the complete approved planning input inline.
/// Decompose admits only intent subjects, so this bound is exactly
/// intent-scoped; every other non-reassess phase keeps
/// `MAX_FACTORY_PROMPT`. Finite headroom, not a maximal-input promise.
pub const MAX_DECOMPOSE_PROMPT: usize = 128 * 1024;

/// `MaxStagingPromptForPhase`: the admitted staged-prompt bound for one
/// execution phase, mirroring the factory-domain `MaxPromptForPhase`
/// policy. Unknown phases keep the standard bound and refuse
/// downstream at admission; staging alone never widens authority.
pub fn max_staging_prompt_for_phase(phase: &str) -> usize {
    if phase == "reassess" || phase == "repair" {
        MAX_REASSESS_PROMPT
    } else if phase == "decompose" {
        MAX_DECOMPOSE_PROMPT
    } else {
        MAX_FACTORY_PROMPT
    }
}

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
    pub effort: String,
    pub contributor: bool,
    pub assignment: String,
    pub source_commit: String,
    pub connection: String,
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
        if (!self.effort.is_empty()
            && (!crate::factory::valid_task_effort(&self.effort)
                || self.harness != FACTORY_HARNESS_MUSE
                || self.model.is_empty()))
            || (self.contributor && self.effort.is_empty())
        {
            return Err("invalid Muse execution profile".into());
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
}
