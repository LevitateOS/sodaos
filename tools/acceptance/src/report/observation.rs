use std::collections::BTreeMap;
use std::fmt;

use serde::de::{MapAccess, Visitor};
use serde::{Deserialize, Deserializer, Serialize};
use serde_json::value::RawValue;

use crate::error::Error;
use crate::jsonio;

/// One observation: an ordinary log index, not a scenario/qualification
/// registry. `None` renders as JSON null, like Go's nil pointers, maps
/// and slices.
#[derive(Debug, Clone, Default, PartialEq, Eq)]
pub struct Observation {
    /// Owner label.
    pub owner: String,
    /// Requested full source revision.
    pub requested_revision: String,
    /// Tool VCS revision.
    pub tool_revision: String,
    /// Requested appliance architecture.
    pub requested_architecture: String,
    /// Explicit non-secret target name.
    pub target: String,
    /// Driver platform.
    pub client_platform: String,
    /// Executed action.
    pub action: String,
    /// Final outcome.
    pub outcome: String,
    /// Execution phase result.
    pub execution: String,
    /// Evidence phase result.
    pub evidence: String,
    /// Cleanup description.
    pub cleanup: String,
    /// Transport topology description.
    pub topology: String,
    /// Check exit code, when a check started.
    pub exit_code: Option<i64>,
    /// Tool tree was dirty.
    pub tool_dirty: bool,
    /// Invoked command.
    pub invocation: Option<Vec<String>>,
    /// Retained evidence hashes.
    pub files: Option<BTreeMap<String, String>>,
    /// Public artifact references.
    pub artifacts: Option<BTreeMap<String, String>>,
    /// Start timestamp (RFC 3339).
    pub started: String,
    /// Finish timestamp (RFC 3339).
    pub finished: String,
}

struct ObservationSlots(Vec<(String, Box<RawValue>)>);

impl<'de> Deserialize<'de> for ObservationSlots {
    fn deserialize<D>(deserializer: D) -> Result<Self, D::Error>
    where
        D: Deserializer<'de>,
    {
        struct SlotsVisitor;
        impl<'de> Visitor<'de> for SlotsVisitor {
            type Value = ObservationSlots;
            fn expecting(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
                f.write_str("an observation object")
            }
            fn visit_map<M>(self, mut map: M) -> Result<Self::Value, M::Error>
            where
                M: MapAccess<'de>,
            {
                let mut entries = Vec::with_capacity(map.size_hint().unwrap_or(0));
                while let Some(pair) = map.next_entry::<String, Box<RawValue>>()? {
                    entries.push(pair);
                }
                Ok(ObservationSlots(entries))
            }
        }
        deserializer.deserialize_map(SlotsVisitor)
    }
}

fn last<'a>(slots: &'a ObservationSlots, field: &str) -> Option<&'a RawValue> {
    slots
        .0
        .iter()
        .rev()
        .find(|(key, _)| key == field)
        .map(|(_, value)| value.as_ref())
}

fn optional_string(slots: &ObservationSlots, field: &str) -> Result<String, Error> {
    match last(slots, field) {
        None => Ok(String::new()),
        Some(raw) if raw.get() == "null" => Ok(String::new()),
        Some(raw) => serde_json::from_str(raw.get())
            .map_err(|_| Error::msg(format!("invalid {field}: string required"))),
    }
}

fn optional_string_list(
    slots: &ObservationSlots,
    field: &str,
) -> Result<Option<Vec<String>>, Error> {
    let Some(raw) = last(slots, field) else {
        return Ok(None);
    };
    if raw.get() == "null" {
        return Ok(None);
    }
    serde_json::from_str(raw.get())
        .map(Some)
        .map_err(|_| Error::msg(format!("invalid {field}: string required")))
}

fn optional_string_map(
    slots: &ObservationSlots,
    field: &str,
) -> Result<Option<BTreeMap<String, String>>, Error> {
    let Some(raw) = last(slots, field) else {
        return Ok(None);
    };
    if raw.get() == "null" {
        return Ok(None);
    }
    serde_json::from_str(raw.get())
        .map(Some)
        .map_err(|_| Error::msg(format!("invalid {field}: string required")))
}

fn optional_timestamp(slots: &ObservationSlots, field: &str) -> Result<String, Error> {
    match last(slots, field) {
        None => Ok(String::new()),
        Some(raw) if raw.get() == "null" => Ok(String::new()),
        Some(raw) => {
            let value: String = serde_json::from_str(raw.get())
                .map_err(|_| Error::msg(format!("invalid {field}: timestamp required")))?;
            jsonio::validate_rfc3339(&value)
                .map_err(|_| Error::msg(format!("invalid {field}: timestamp required")))?;
            Ok(value)
        }
    }
}

const OBSERVATION_FIELDS: &[&str] = &[
    "Owner",
    "RequestedRevision",
    "ToolRevision",
    "RequestedArchitecture",
    "Target",
    "ClientPlatform",
    "Action",
    "Outcome",
    "Execution",
    "Evidence",
    "Cleanup",
    "Topology",
    "ExitCode",
    "ToolDirty",
    "Invocation",
    "Files",
    "Artifacts",
    "Started",
    "Finished",
];

/// Decode an observation using its exact known names and last exact duplicate.
pub fn observation_from_json(value: &RawValue) -> Result<Observation, Error> {
    let slots = if value.get().as_bytes()[0] == b'{' {
        serde_json::from_str::<ObservationSlots>(value.get())
            .map_err(|_| Error::msg("invalid JSON input"))?
    } else {
        ObservationSlots(Vec::new())
    };
    for (key, _) in &slots.0 {
        if !OBSERVATION_FIELDS.contains(&key.as_str()) {
            return Err(Error::msg(format!("json: unknown field {key:?}")));
        }
    }
    let exit_code = match last(&slots, "ExitCode") {
        None => None,
        Some(raw) if raw.get() == "null" => None,
        Some(raw) => Some(
            serde_json::from_str::<i64>(raw.get())
                .map_err(|_| Error::msg("invalid ExitCode: integer required"))?,
        ),
    };
    let tool_dirty = match last(&slots, "ToolDirty") {
        None => false,
        Some(raw) if raw.get() == "null" => false,
        Some(raw) => serde_json::from_str::<bool>(raw.get())
            .map_err(|_| Error::msg("invalid ToolDirty: boolean required"))?,
    };
    Ok(Observation {
        owner: optional_string(&slots, "Owner")?,
        requested_revision: optional_string(&slots, "RequestedRevision")?,
        tool_revision: optional_string(&slots, "ToolRevision")?,
        requested_architecture: optional_string(&slots, "RequestedArchitecture")?,
        target: optional_string(&slots, "Target")?,
        client_platform: optional_string(&slots, "ClientPlatform")?,
        action: optional_string(&slots, "Action")?,
        outcome: optional_string(&slots, "Outcome")?,
        execution: optional_string(&slots, "Execution")?,
        evidence: optional_string(&slots, "Evidence")?,
        cleanup: optional_string(&slots, "Cleanup")?,
        topology: optional_string(&slots, "Topology")?,
        exit_code,
        tool_dirty,
        invocation: optional_string_list(&slots, "Invocation")?,
        files: optional_string_map(&slots, "Files")?,
        artifacts: optional_string_map(&slots, "Artifacts")?,
        started: optional_timestamp(&slots, "Started")?,
        finished: optional_timestamp(&slots, "Finished")?,
    })
}

/// Fixed Go struct field order; maps use BTreeMap's Go-compatible key order.
#[derive(Serialize)]
pub struct ObservationRecord<'a> {
    #[serde(rename = "Owner")]
    pub owner: &'a str,
    #[serde(rename = "RequestedRevision")]
    pub requested_revision: &'a str,
    #[serde(rename = "ToolRevision")]
    pub tool_revision: &'a str,
    #[serde(rename = "RequestedArchitecture")]
    pub requested_architecture: &'a str,
    #[serde(rename = "Target")]
    pub target: &'a str,
    #[serde(rename = "ClientPlatform")]
    pub client_platform: &'a str,
    #[serde(rename = "Action")]
    pub action: &'a str,
    #[serde(rename = "Outcome")]
    pub outcome: &'a str,
    #[serde(rename = "Execution")]
    pub execution: &'a str,
    #[serde(rename = "Evidence")]
    pub evidence: &'a str,
    #[serde(rename = "Cleanup")]
    pub cleanup: &'a str,
    #[serde(rename = "Topology")]
    pub topology: &'a str,
    #[serde(rename = "ExitCode")]
    pub exit_code: Option<i64>,
    #[serde(rename = "ToolDirty")]
    pub tool_dirty: bool,
    #[serde(rename = "Invocation")]
    pub invocation: Option<&'a [String]>,
    #[serde(rename = "Files")]
    pub files: Option<&'a BTreeMap<String, String>>,
    #[serde(rename = "Artifacts")]
    pub artifacts: Option<&'a BTreeMap<String, String>>,
    #[serde(rename = "Started")]
    pub started: &'a str,
    #[serde(rename = "Finished")]
    pub finished: &'a str,
}

pub fn observation_json(o: &Observation) -> ObservationRecord<'_> {
    ObservationRecord {
        owner: &o.owner,
        requested_revision: &o.requested_revision,
        tool_revision: &o.tool_revision,
        requested_architecture: &o.requested_architecture,
        target: &o.target,
        client_platform: &o.client_platform,
        action: &o.action,
        outcome: &o.outcome,
        execution: &o.execution,
        evidence: &o.evidence,
        cleanup: &o.cleanup,
        topology: &o.topology,
        exit_code: o.exit_code,
        tool_dirty: o.tool_dirty,
        invocation: o.invocation.as_deref(),
        files: o.files.as_ref(),
        artifacts: o.artifacts.as_ref(),
        started: &o.started,
        finished: &o.finished,
    }
}
