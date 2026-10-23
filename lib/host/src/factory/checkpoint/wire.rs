use serde::{Deserialize, Serialize};

use crate::{domain, preparation};

use super::super::{valid_factory_run_id, valid_task_key, ExecutionManifest};

#[derive(Clone, Debug, Default, Deserialize, Serialize, PartialEq, Eq)]
#[serde(deny_unknown_fields)]
pub struct FactoryTaskCheckpoint {
    pub project: String,
    pub id: String,
    pub container: String,
    pub invocation: String,
    pub manifest_digest: String,
    #[serde(default)]
    pub task_key: String,
    #[serde(default)]
    pub output_digest: String,
    pub expected_head: String,
}

impl FactoryTaskCheckpoint {
    pub fn decode(body: &[u8]) -> Result<Self, String> {
        if body.len() > 4096 {
            return Err("checkpoint request exceeds its bound".into());
        }
        crate::json::decode_strict_as(body).map_err(|e| e.0)
    }

    pub fn validate(&self, final_candidate: bool) -> Result<(), String> {
        if !domain::valid_id(&self.project)
            || !valid_factory_run_id(&self.id)
            || !domain::valid_container_id(&self.container)
            || self.invocation.is_empty()
            || self.invocation.len() > 128
            || self.invocation.bytes().any(|b| b < 0x20 || b == 0x7f)
            || !preparation::valid_digest(&self.manifest_digest)
            || !preparation::valid_commit(&self.expected_head)
        {
            return Err("checkpoint request lacks its exact execution binding".into());
        }
        if final_candidate {
            if !self.task_key.is_empty() || !self.output_digest.is_empty() {
                return Err("final candidate request cannot nominate task output".into());
            }
        } else if !valid_task_key(&self.task_key) || !preparation::valid_digest(&self.output_digest)
        {
            return Err("checkpoint request lacks its task output binding".into());
        }
        Ok(())
    }
}

#[derive(Clone, Debug, Default, Deserialize, Serialize, PartialEq, Eq)]
#[serde(deny_unknown_fields)]
pub struct CheckpointCheckResult {
    pub id: String,
    pub passed: bool,
    pub log_digest: String,
}

#[derive(Clone, Debug, Default, Deserialize, Serialize, PartialEq, Eq)]
#[serde(deny_unknown_fields)]
pub struct FactoryTaskCheckpointState {
    pub request: FactoryTaskCheckpoint,
    pub final_candidate: bool,
    pub status: String,
    #[serde(default)]
    pub candidate: String,
    #[serde(default)]
    pub tree: String,
    #[serde(default)]
    pub receipt_digest: String,
    #[serde(default)]
    pub checks: Vec<CheckpointCheckResult>,
    #[serde(default)]
    pub reason: String,
}

impl FactoryTaskCheckpointState {
    pub fn encode(&self) -> String {
        serde_json::to_string(self).expect("checkpoint wire fields are serializable")
    }
}

// The whole Go plan is already admitted and bound as exact immutable bytes.
// These typed projections read only the fixed Git/check inputs Rust executes.
#[derive(Clone, Debug, Deserialize, Serialize)]
pub(super) struct CheckpointPlan {
    pub source_commit: String,
    pub integration_commit: String,
    pub tasks: Vec<CheckpointTask>,
    pub checks: Vec<CheckpointCheck>,
}

#[derive(Clone, Debug, Deserialize, Serialize)]
pub(super) struct CheckpointTask {
    pub id: String,
    pub owned_paths: Vec<String>,
    pub check_ids: Vec<String>,
}

#[derive(Clone, Debug, Deserialize, Serialize)]
pub(super) struct CheckpointCheck {
    pub id: String,
    pub argv: Vec<String>,
    pub cwd: String,
    pub timeout_seconds: u64,
    pub scope: String,
    pub required: bool,
}

pub(super) fn valid_owned_path(path: &str) -> bool {
    let plain = path.strip_suffix('/').unwrap_or(path);
    !plain.is_empty()
        && plain.len() <= 1024
        && !plain.starts_with('/')
        && !plain.contains(['\0', '\\', '*', '?', '[', ']'])
        && plain
            .split('/')
            .all(|part| !part.is_empty() && !matches!(part, "." | ".." | ".git" | ".soda-home"))
}

impl CheckpointPlan {
    pub(super) fn decode(manifest: &ExecutionManifest) -> Result<Self, String> {
        let plan: Self =
            serde_json::from_slice(&manifest.plan).map_err(|_| "checkpoint plan is invalid")?;
        if !preparation::valid_commit(&plan.source_commit)
            || !preparation::valid_commit(&plan.integration_commit)
            || plan.tasks.is_empty()
            || plan.tasks.len() > 64
            || plan.checks.len() > 16
        {
            return Err("checkpoint plan exceeds its admitted bounds".into());
        }
        let mut tasks = std::collections::HashSet::new();
        let mut checks = std::collections::HashSet::new();
        for check in &plan.checks {
            if !valid_task_key(&check.id)
                || !checks.insert(check.id.as_str())
                || check.argv.is_empty()
                || check.argv.len() > 64
                || check.argv.iter().any(|a| a.is_empty() || a.contains('\0'))
                || check.argv.iter().map(String::len).sum::<usize>() > 16 * 1024
                || check.cwd != "." && !valid_owned_path(&check.cwd)
                || !(1..=600).contains(&check.timeout_seconds)
                || !matches!(check.scope.as_str(), "task" | "integration")
            {
                return Err("checkpoint check specification is invalid".into());
            }
        }
        for task in &plan.tasks {
            if !valid_task_key(&task.id)
                || !tasks.insert(task.id.as_str())
                || task.owned_paths.is_empty()
                || task.owned_paths.len() > 32
                || task.owned_paths.iter().any(|p| !valid_owned_path(p))
                || task
                    .check_ids
                    .iter()
                    .any(|id| !checks.contains(id.as_str()))
            {
                return Err("checkpoint task ownership is invalid".into());
            }
        }
        Ok(plan)
    }
}
