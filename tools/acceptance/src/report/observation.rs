use std::collections::BTreeMap;

use soda_json::JsonValue;

use crate::error::Error;
use crate::jsonio::{self, check_no_unknown, opt_bool, opt_string};

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

/// Observation fields in Go struct order.
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

fn opt_string_list(value: &JsonValue, field: &str) -> Result<Option<Vec<String>>, Error> {
    match value.get(field) {
        None | Some(JsonValue::Null) => Ok(None),
        Some(JsonValue::Array(items)) => {
            let mut out = Vec::with_capacity(items.len());
            for item in items {
                match item {
                    JsonValue::Str(s) => out.push(s.clone()),
                    _ => return Err(Error::msg(format!("invalid {field}: string required"))),
                }
            }
            Ok(Some(out))
        }
        Some(_) => Err(Error::msg(format!("invalid {field}: array required"))),
    }
}

fn opt_string_map(
    value: &JsonValue,
    field: &str,
) -> Result<Option<BTreeMap<String, String>>, Error> {
    match value.get(field) {
        None | Some(JsonValue::Null) => Ok(None),
        Some(JsonValue::Object(entries)) => {
            let mut out = BTreeMap::new();
            for (key, item) in entries {
                match item {
                    JsonValue::Str(s) => {
                        out.insert(key.clone(), s.clone());
                    }
                    _ => return Err(Error::msg(format!("invalid {field}: string required"))),
                }
            }
            Ok(Some(out))
        }
        Some(_) => Err(Error::msg(format!("invalid {field}: object required"))),
    }
}

fn opt_timestamp(value: &JsonValue, field: &str) -> Result<String, Error> {
    match value.get(field) {
        None | Some(JsonValue::Null) => Ok(String::new()),
        Some(JsonValue::Str(s)) => {
            jsonio::validate_rfc3339(s)
                .map_err(|_| Error::msg(format!("invalid {field}: timestamp required")))?;
            Ok(s.clone())
        }
        Some(_) => Err(Error::msg(format!("invalid {field}: timestamp required"))),
    }
}

/// Decode an observation, rejecting unknown fields and trailing data
/// like the Go owner's `ReadJSONAt` into the struct.
pub fn observation_from_json(value: &JsonValue) -> Result<Observation, Error> {
    check_no_unknown(value, OBSERVATION_FIELDS)?;
    let exit_code = match value.get("ExitCode") {
        None | Some(JsonValue::Null) => None,
        Some(v) => {
            let code = v
                .as_integer()
                .ok_or_else(|| Error::msg("invalid ExitCode: integer required"))?;
            Some(
                i64::try_from(code)
                    .map_err(|_| Error::msg("invalid ExitCode: integer required"))?,
            )
        }
    };
    Ok(Observation {
        owner: opt_string(value, "Owner")?,
        requested_revision: opt_string(value, "RequestedRevision")?,
        tool_revision: opt_string(value, "ToolRevision")?,
        requested_architecture: opt_string(value, "RequestedArchitecture")?,
        target: opt_string(value, "Target")?,
        client_platform: opt_string(value, "ClientPlatform")?,
        action: opt_string(value, "Action")?,
        outcome: opt_string(value, "Outcome")?,
        execution: opt_string(value, "Execution")?,
        evidence: opt_string(value, "Evidence")?,
        cleanup: opt_string(value, "Cleanup")?,
        topology: opt_string(value, "Topology")?,
        exit_code,
        tool_dirty: opt_bool(value, "ToolDirty")?,
        invocation: opt_string_list(value, "Invocation")?,
        files: opt_string_map(value, "Files")?,
        artifacts: opt_string_map(value, "Artifacts")?,
        started: opt_timestamp(value, "Started")?,
        finished: opt_timestamp(value, "Finished")?,
    })
}

pub(super) fn map_json(map: &BTreeMap<String, String>) -> JsonValue {
    JsonValue::Object(
        map.iter()
            .map(|(k, v)| (k.clone(), JsonValue::Str(v.clone())))
            .collect(),
    )
}

/// Encode an observation in Go struct field order. The evidence writer
/// sorts keys on output, like Go's map encoding.
pub fn observation_json(o: &Observation) -> JsonValue {
    let mut entries = Vec::with_capacity(OBSERVATION_FIELDS.len());
    let mut field = |name: &str, value: JsonValue| entries.push((name.to_string(), value));
    field("Owner", JsonValue::Str(o.owner.clone()));
    field(
        "RequestedRevision",
        JsonValue::Str(o.requested_revision.clone()),
    );
    field("ToolRevision", JsonValue::Str(o.tool_revision.clone()));
    field(
        "RequestedArchitecture",
        JsonValue::Str(o.requested_architecture.clone()),
    );
    field("Target", JsonValue::Str(o.target.clone()));
    field("ClientPlatform", JsonValue::Str(o.client_platform.clone()));
    field("Action", JsonValue::Str(o.action.clone()));
    field("Outcome", JsonValue::Str(o.outcome.clone()));
    field("Execution", JsonValue::Str(o.execution.clone()));
    field("Evidence", JsonValue::Str(o.evidence.clone()));
    field("Cleanup", JsonValue::Str(o.cleanup.clone()));
    field("Topology", JsonValue::Str(o.topology.clone()));
    field(
        "ExitCode",
        o.exit_code
            .map(|code| JsonValue::Number(code.to_string()))
            .unwrap_or(JsonValue::Null),
    );
    field("ToolDirty", JsonValue::Bool(o.tool_dirty));
    field(
        "Invocation",
        o.invocation
            .as_ref()
            .map(|args| {
                JsonValue::Array(args.iter().map(|arg| JsonValue::Str(arg.clone())).collect())
            })
            .unwrap_or(JsonValue::Null),
    );
    field(
        "Files",
        o.files.as_ref().map(map_json).unwrap_or(JsonValue::Null),
    );
    field(
        "Artifacts",
        o.artifacts
            .as_ref()
            .map(map_json)
            .unwrap_or(JsonValue::Null),
    );
    field("Started", JsonValue::Str(o.started.clone()));
    field("Finished", JsonValue::Str(o.finished.clone()));
    JsonValue::Object(entries)
}
