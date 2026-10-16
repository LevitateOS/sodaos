//! Immutable execution-manifest envelope (D05 / FW13).
//!
//! Wire mirror of `project.ExecutionManifest`: the exact serialized plan
//! bytes and their digest bound to the admitted invocation, phase, run,
//! deadline, execution profile snapshot and child limit. The executor
//! verifies the staged bytes against the run's manifest digest, not a
//! separately reconstructed cross-language serialization. FW14 stages
//! this envelope at dispatch; planning phases stage no manifest at all.

use serde::de::{self, MapAccess, Visitor};
use serde::Deserialize;
use std::fmt;

use crate::json;

use super::deadline::parse_deadline;
use super::run::{
    execution_manifest_required, valid_factory_run_id, EXECUTION_IMPLEMENT, MAX_EXECUTION_MANIFEST,
    MAX_TASK_CHILDREN, MIN_TASK_CHILDREN,
};
use super::FactoryRun;

/// Admitted reasoning efforts in a staged execution profile snapshot.
/// Vocabulary only; harness support for an effort rides the proved
/// harness pin the executor admits.
pub fn valid_task_effort(effort: &str) -> bool {
    matches!(effort, "low" | "medium" | "high" | "xhigh")
}

/// `project.ExecutionManifest`: the immutable child-control envelope.
#[derive(Debug, Clone, Default, PartialEq, Eq)]
pub struct ExecutionManifest {
    pub plan: Vec<u8>,
    pub plan_digest: String,
    pub invocation: String,
    pub phase: String,
    pub run: String,
    pub deadline: String,
    pub effort: String,
    pub contributor: bool,
    pub max_children: i64,
}

struct PlanBytes(Vec<u8>);

impl<'de> Deserialize<'de> for PlanBytes {
    fn deserialize<D>(deserializer: D) -> Result<Self, D::Error>
    where
        D: serde::Deserializer<'de>,
    {
        struct BytesVisitor;
        impl<'de> Visitor<'de> for BytesVisitor {
            type Value = PlanBytes;
            fn expecting(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
                f.write_str("base64 text")
            }
            fn visit_str<E>(self, value: &str) -> Result<Self::Value, E>
            where
                E: de::Error,
            {
                crate::ssh::b64_decode_go(value.as_bytes())
                    .map(PlanBytes)
                    .map_err(|_| E::custom("invalid base64 bytes"))
            }
            fn visit_string<E>(self, value: String) -> Result<Self::Value, E>
            where
                E: de::Error,
            {
                self.visit_str(&value)
            }
        }
        deserializer.deserialize_str(BytesVisitor)
    }
}

impl<'de> Deserialize<'de> for ExecutionManifest {
    fn deserialize<D>(deserializer: D) -> Result<Self, D::Error>
    where
        D: serde::Deserializer<'de>,
    {
        struct ManifestVisitor;
        impl<'de> Visitor<'de> for ManifestVisitor {
            type Value = ExecutionManifest;
            fn expecting(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
                f.write_str("an execution manifest object")
            }
            fn visit_map<A>(self, mut map: A) -> Result<Self::Value, A::Error>
            where
                A: MapAccess<'de>,
            {
                let mut out = ExecutionManifest::default();
                let mut max_children_seen = false;
                while let Some(key) = map.next_key::<String>()? {
                    if key.eq_ignore_ascii_case("plan") {
                        if let Some(v) = map.next_value::<Option<PlanBytes>>()? {
                            out.plan = v.0;
                        }
                        continue;
                    }
                    if key.eq_ignore_ascii_case("plan_digest") {
                        if let Some(v) = map.next_value::<Option<String>>()? {
                            out.plan_digest = v;
                        }
                        continue;
                    }
                    if key.eq_ignore_ascii_case("invocation") {
                        if let Some(v) = map.next_value::<Option<String>>()? {
                            out.invocation = v;
                        }
                        continue;
                    }
                    if key.eq_ignore_ascii_case("phase") {
                        if let Some(v) = map.next_value::<Option<String>>()? {
                            out.phase = v;
                        }
                        continue;
                    }
                    if key.eq_ignore_ascii_case("run") {
                        if let Some(v) = map.next_value::<Option<String>>()? {
                            out.run = v;
                        }
                        continue;
                    }
                    if key.eq_ignore_ascii_case("deadline") {
                        if let Some(v) = map.next_value::<Option<String>>()? {
                            out.deadline = v;
                        }
                        continue;
                    }
                    if key.eq_ignore_ascii_case("effort") {
                        if let Some(v) = map.next_value::<Option<String>>()? {
                            out.effort = v;
                        }
                        continue;
                    }
                    if key.eq_ignore_ascii_case("contributor") {
                        if let Some(v) = map.next_value::<Option<bool>>()? {
                            out.contributor = v;
                        }
                        continue;
                    }
                    if key.eq_ignore_ascii_case("max_children") {
                        if let Some(v) = map.next_value::<Option<json::SignedInteger>>()? {
                            out.max_children = v.into();
                            max_children_seen = true;
                        }
                        continue;
                    }
                    return Err(de::Error::unknown_field(
                        &key,
                        &[
                            "plan",
                            "plan_digest",
                            "invocation",
                            "phase",
                            "run",
                            "deadline",
                            "effort",
                            "contributor",
                            "max_children",
                        ],
                    ));
                }
                if !max_children_seen {
                    return Err(de::Error::missing_field("max_children"));
                }
                Ok(out)
            }
        }
        deserializer.deserialize_map(ManifestVisitor)
    }
}

impl ExecutionManifest {
    pub fn decode(body: &[u8]) -> Result<Self, String> {
        if body.is_empty() || body.len() > MAX_EXECUTION_MANIFEST {
            return Err("execution manifest exceeds its bound".to_string());
        }
        json::decode_strict_as(body).map_err(|e| e.0)
    }

    /// Envelope bindings. Run binding (phase/run/deadline against the
    /// launching run) is checked by [`ExecutionManifest::verify_for_run`].
    pub fn validate(&self) -> Result<(), String> {
        if self.plan.is_empty() || self.plan.len() > MAX_EXECUTION_MANIFEST {
            return Err("execution manifest lacks its plan bytes".to_string());
        }
        if crate::sha256::hex_lower(&crate::sha256::digest(&self.plan)) != self.plan_digest {
            return Err("execution manifest plan does not match its digest".to_string());
        }
        if self.phase != EXECUTION_IMPLEMENT {
            return Err("execution manifest phase is not implement".to_string());
        }
        if !valid_factory_run_id(&self.run) {
            return Err("execution manifest run is invalid".to_string());
        }
        if self.invocation.is_empty()
            || self.invocation.len() > 128
            || self.invocation.bytes().any(|b| b < 0x20 || b == 0x7f)
        {
            return Err("execution manifest invocation is invalid".to_string());
        }
        if self.deadline.is_empty() {
            return Err("execution manifest deadline is required".to_string());
        }
        if !valid_task_effort(&self.effort) {
            return Err("invalid execution effort".to_string());
        }
        if !(MIN_TASK_CHILDREN..=MAX_TASK_CHILDREN).contains(&self.max_children) {
            return Err("execution manifest child limit is outside its range".to_string());
        }
        Ok(())
    }

    /// Bind a staged envelope to its launching run: digest, phase, run and
    /// verbatim deadline match the admitted record. Contributor mode is
    /// valid only for the Muse harness; children inherit the exact parent
    /// profile and account.
    pub fn verify_for_run(&self, run: &FactoryRun, staged: &[u8]) -> Result<(), String> {
        self.validate()?;
        if crate::sha256::hex_lower(&crate::sha256::digest(staged)) != run.manifest_digest {
            return Err("execution manifest does not match its digest".to_string());
        }
        if !execution_manifest_required(&run.execution_phase) {
            return Err("non-implement run stages an execution manifest".to_string());
        }
        if self.phase != run.execution_phase || self.run != run.id {
            return Err("execution manifest does not match its run".to_string());
        }
        match (
            parse_deadline(&self.deadline),
            parse_deadline(&run.deadline),
        ) {
            (Some(expected), Some(recorded)) if expected == recorded => {}
            _ => {
                return Err("execution manifest deadline does not match its run".to_string());
            }
        }
        if self.contributor && run.harness != super::run::FACTORY_HARNESS_MUSE {
            return Err("contributor mode is valid only for Muse".to_string());
        }
        Ok(())
    }
}
