//! The approved project.check recipe, bound to one reserved invocation.
//! Exact manifest bytes are retained; model-supplied commands are never used.

use serde::{Deserialize, Serialize};

use super::deadline::{deadline_is_zero, parse_deadline};
use super::{valid_factory_run_id, FactoryRun, EXECUTION_VERIFY};
use crate::{domain, json, preparation, sha256};

pub const MAX_VERIFICATION_MANIFEST: usize = 64 * 1024;
pub const MAX_VERIFICATION_OUTPUT_EXCERPT: usize = 4 * 1024;

#[derive(Debug, Clone, Default, PartialEq, Eq, Deserialize, Serialize)]
#[serde(deny_unknown_fields)]
pub struct VerificationManifestCheck {
    pub catalog_id: String,
    pub outcomes: Vec<String>,
    pub artifacts: Vec<String>,
    pub fixtures: Vec<String>,
}

#[derive(Debug, Clone, Default, PartialEq, Eq, Deserialize, Serialize)]
#[serde(deny_unknown_fields)]
pub struct VerificationManifest {
    pub version: String,
    pub run: String,
    pub project: String,
    pub preparation: String,
    pub source_commit: String,
    pub deadline: String,
    pub intent: String,
    pub accepted_revision: String,
    pub scope_digest: String,
    pub adopted_manifest_digest: String,
    pub requirements: String,
    pub approval: String,
    pub preparation_digest: String,
    pub check_digest: String,
    pub outcomes: Vec<String>,
    pub checks: Vec<VerificationManifestCheck>,
}

impl VerificationManifest {
    pub fn decode(raw: &[u8]) -> Result<Self, String> {
        if raw.is_empty() || raw.len() > MAX_VERIFICATION_MANIFEST {
            return Err("verification manifest exceeds its byte bound".into());
        }
        let manifest: Self = json::decode_strict_as(raw).map_err(|e| e.0)?;
        manifest.validate()?;
        Ok(manifest)
    }

    pub fn validate(&self) -> Result<(), String> {
        if self.version != "project-check-v1"
            || !valid_factory_run_id(&self.run)
            || !domain::valid_id(&self.project)
            || !preparation::valid_preparation_id(&self.preparation)
            || !preparation::valid_commit(&self.source_commit)
            || parse_deadline(&self.deadline).is_none()
            || deadline_is_zero(&self.deadline)
            || !preparation::valid_decision_id(&self.intent)
            || !preparation::valid_decision_id(&self.accepted_revision)
            || !preparation::valid_decision_id(&self.requirements)
            || !preparation::valid_decision_id(&self.approval)
            || [
                &self.scope_digest,
                &self.adopted_manifest_digest,
                &self.preparation_digest,
                &self.check_digest,
            ]
            .iter()
            .any(|d| !preparation::valid_digest(d))
        {
            return Err("verification manifest lacks its exact approved run binding".into());
        }
        let mut outcomes = std::collections::BTreeSet::new();
        if self.outcomes.is_empty()
            || self.outcomes.len() > 64
            || self.checks.len() != 1
            || self
                .outcomes
                .iter()
                .any(|o| !preparation::valid_decision_id(o) || !outcomes.insert(o))
        {
            return Err("verification manifest lacks complete bounded check coverage".into());
        }
        let check = &self.checks[0];
        if check.catalog_id != "project.check"
            || check.outcomes != self.outcomes
            || check.artifacts != ["project.check.receipt"]
            || check.fixtures != ["project.preparation"]
        {
            return Err("verification manifest differs from its approved Project check".into());
        }
        Ok(())
    }

    pub fn verify_for_run(&self, run: &FactoryRun, raw: &[u8]) -> Result<(), String> {
        self.validate()?;
        if run.execution_phase != EXECUTION_VERIFY
            || run.role != preparation::ROLE_REVIEWER
            || self.run != run.id
            || self.project != run.project
            || self.preparation != run.preparation
            || self.source_commit != run.source_commit
            || parse_deadline(&self.deadline) != parse_deadline(&run.deadline)
            || sha256::hex_lower(&sha256::digest(raw)) != run.verification_manifest_digest
        {
            return Err("verification manifest differs from its admitted run".into());
        }
        Ok(())
    }
}

#[derive(Debug, Clone, Default, PartialEq, Eq, Deserialize, Serialize)]
#[serde(deny_unknown_fields)]
pub struct FactoryVerificationChecks {
    pub project: String,
    pub run: String,
    pub incarnation: String,
    pub manifest_digest: String,
}

impl FactoryVerificationChecks {
    pub fn decode(raw: &[u8]) -> Result<Self, String> {
        let request: Self = json::decode_strict_as(raw).map_err(|e| e.0)?;
        request.validate()?;
        Ok(request)
    }
    pub fn validate(&self) -> Result<(), String> {
        if !domain::valid_id(&self.project)
            || !valid_factory_run_id(&self.run)
            || !valid_factory_run_id(&self.incarnation)
            || !preparation::valid_digest(&self.manifest_digest)
        {
            return Err("invalid verification checks identity".into());
        }
        Ok(())
    }
}

#[derive(Debug, Clone, Default, PartialEq, Eq, Deserialize, Serialize)]
#[serde(deny_unknown_fields)]
pub struct FactoryVerificationChecksState {
    pub project: String,
    pub run: String,
    pub incarnation: String,
    pub manifest_digest: String,
    pub phase: String,
    pub source_commit: String,
    pub preparation: String,
    pub check_digest: String,
    #[serde(default, skip_serializing_if = "Option::is_none")]
    pub exit_code: Option<i64>,
    #[serde(default, skip_serializing_if = "String::is_empty")]
    pub output_digest: String,
    // Approved check output only, with display controls escaped. Its digest
    // continues to identify the helper's captured bytes, not this text prefix.
    #[serde(default, skip_serializing_if = "String::is_empty")]
    pub output_excerpt: String,
    #[serde(default, skip_serializing_if = "std::ops::Not::not")]
    pub output_truncated: bool,
    #[serde(default, skip_serializing_if = "String::is_empty")]
    pub retirement: String,
    #[serde(default, skip_serializing_if = "String::is_empty")]
    pub reason: String,
}

impl FactoryVerificationChecksState {
    pub fn validate_for(&self, r: &FactoryVerificationChecks) -> Result<(), String> {
        r.validate()?;
        if self.project != r.project
            || self.run != r.run
            || self.incarnation != r.incarnation
            || self.manifest_digest != r.manifest_digest
            || !preparation::valid_commit(&self.source_commit)
            || !preparation::valid_preparation_id(&self.preparation)
            || !preparation::valid_digest(&self.check_digest)
            || self.reason.len() > 1024
        {
            return Err("verification checks receipt differs from its admitted identity".into());
        }
        if self.output_excerpt.len() > MAX_VERIFICATION_OUTPUT_EXCERPT
            || self
                .output_excerpt
                .chars()
                .any(|c| c.is_control() && c != '\n' && c != '\t')
            || ((!self.output_excerpt.is_empty() || self.output_truncated)
                && !preparation::valid_digest(&self.output_digest))
        {
            return Err("invalid verification checks output excerpt".into());
        }
        match self.phase.as_str() {
            "preparing" | "running"
                if self.exit_code.is_none()
                    && self.output_digest.is_empty()
                    && self.output_excerpt.is_empty()
                    && !self.output_truncated
                    && self.retirement.is_empty() => {}
            "passed"
                if self.exit_code == Some(0)
                    && preparation::valid_digest(&self.output_digest)
                    && self.retirement == "confirmed" => {}
            "failed" | "stopped"
                if self.retirement == "confirmed" || self.retirement == "uncertain" => {}
            _ => return Err("invalid verification checks phase or native exit evidence".into()),
        }
        if !self.output_digest.is_empty() && !preparation::valid_digest(&self.output_digest) {
            return Err("invalid verification checks output digest".into());
        }
        Ok(())
    }
    pub fn encode(&self) -> Result<String, String> {
        serde_json::to_string(self).map_err(|_| "verification checks encoding failed".into())
    }
}
