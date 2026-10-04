//! Observation records and the native support handoff, mirroring
//! `internal/acceptance/report.go`.
//!
//! An [`Observation`] is an ordinary log index, not a
//! scenario/qualification registry. [`handoff`] cites exact observations
//! and explicitly leaves missing scopes open; it can complete without
//! core product evidence and never certifies readiness.

use std::collections::BTreeMap;

use soda_json::JsonValue;

use crate::error::Error;
use crate::evidence::Evidence;
use crate::files::{self, OwnedDir};
use crate::jsonio::{self, check_no_unknown, opt_bool, opt_string};

/// Candidate architecture gate: only `x86_64` is admitted, reported as
/// its OCI name. Mirrors `OCIArchitecture`.
pub fn oci_architecture(arch: &str) -> Result<String, Error> {
    soda_build_tools::reader::oci_architecture(arch)
        .map(|name| name.to_string())
        .map_err(|err| Error::msg(err.to_string()))
}

/// Matching-native Linux gate. Mirrors `RequireNative`.
pub fn require_native(arch: &str) -> Result<(), Error> {
    let oci = oci_architecture(arch)?;
    let native = std::env::consts::OS == "linux"
        && match std::env::consts::ARCH {
            "x86_64" => "amd64",
            "x86" => "386",
            "aarch64" => "arm64",
            other => other,
        } == oci;
    if !native {
        return Err(Error::msg("matching-native Linux required"));
    }
    Ok(())
}

/// Lowercase SHA-256 hex shape. Mirrors `Digest`.
pub fn digest(text: &str) -> bool {
    soda_build_tools::reader::is_digest(text)
}

/// Full lowercase source-revision shape. Mirrors `Revision`.
pub fn revision(text: &str) -> bool {
    soda_build_tools::reader::is_revision(text)
}

/// Owner label check. It never grants execution permission.
pub fn valid_owner(owner: &str) -> bool {
    match owner {
        "P02" | "P03" | "P04" | "P05" | "P06" | "P11" => return true,
        _ => {}
    }
    // U01-U20: `^U(?:0[1-9]|1[0-9]|20)$`.
    let bytes = owner.as_bytes();
    if bytes.len() != 3 || bytes[0] != b'U' || !bytes[1].is_ascii_digit() || !bytes[2].is_ascii_digit() {
        return false;
    }
    matches!((bytes[1] - b'0', bytes[2] - b'0'), (0, 1..=9) | (1, 0..=9) | (2, 0))
}

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

fn opt_string_map(value: &JsonValue, field: &str) -> Result<Option<BTreeMap<String, String>>, Error> {
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
            jsonio::validate_rfc3339(s).map_err(|_| Error::msg(format!("invalid {field}: timestamp required")))?;
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
            Some(i64::try_from(code).map_err(|_| Error::msg("invalid ExitCode: integer required"))?)
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

fn map_json(map: &BTreeMap<String, String>) -> JsonValue {
    JsonValue::Object(map.iter().map(|(k, v)| (k.clone(), JsonValue::Str(v.clone()))).collect())
}

/// Encode an observation in Go struct field order. The evidence writer
/// sorts keys on output, like Go's map encoding.
pub fn observation_json(o: &Observation) -> JsonValue {
    let mut entries = Vec::with_capacity(OBSERVATION_FIELDS.len());
    let mut field = |name: &str, value: JsonValue| entries.push((name.to_string(), value));
    field("Owner", JsonValue::Str(o.owner.clone()));
    field("RequestedRevision", JsonValue::Str(o.requested_revision.clone()));
    field("ToolRevision", JsonValue::Str(o.tool_revision.clone()));
    field("RequestedArchitecture", JsonValue::Str(o.requested_architecture.clone()));
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
        o.exit_code.map(|code| JsonValue::Number(code.to_string())).unwrap_or(JsonValue::Null),
    );
    field("ToolDirty", JsonValue::Bool(o.tool_dirty));
    field(
        "Invocation",
        o.invocation
            .as_ref()
            .map(|args| JsonValue::Array(args.iter().map(|arg| JsonValue::Str(arg.clone())).collect()))
            .unwrap_or(JsonValue::Null),
    );
    field("Files", o.files.as_ref().map(map_json).unwrap_or(JsonValue::Null));
    field("Artifacts", o.artifacts.as_ref().map(map_json).unwrap_or(JsonValue::Null));
    field("Started", JsonValue::Str(o.started.clone()));
    field("Finished", JsonValue::Str(o.finished.clone()));
    JsonValue::Object(entries)
}

/// Hash every retained evidence file, like `Evidence.Hashes`.
pub fn hashes(evidence: &Evidence) -> Result<BTreeMap<String, String>, Error> {
    let mut files = BTreeMap::new();
    let mut visit = |path: &str, regular: bool| -> Result<(), Error> {
        if !regular {
            return Err(Error::msg("unexpected evidence entry"));
        }
        let sum = files::hash_at(evidence.root(), path)?;
        files.insert(path.to_string(), sum);
        Ok(())
    };
    evidence.root().walk_files(&mut visit)?;
    Ok(files)
}

fn observation_artifacts_valid(o: &Observation) -> Result<(), Error> {
    if let Some(artifacts) = &o.artifacts {
        for (name, sum) in artifacts {
            if name.is_empty() || !digest(sum) {
                return Err(Error::msg("invalid public artifact reference"));
            }
        }
    }
    Ok(())
}

fn observation_files_match(root: &OwnedDir, o: &Observation) -> Result<(), Error> {
    let Some(files) = &o.files else {
        return Ok(());
    };
    for (name, sum) in files {
        // `filepath.IsLocal` plus `Clean` identity on a clean POSIX path:
        // relative, no leading escape, never ".".
        if files::lexical_clean(name) != *name || name == "." || name.starts_with('/') || name == ".." || name.starts_with("../") {
            return Err(Error::msg("unsafe observation reference"));
        }
        match files::hash_at(root, name) {
            Ok(actual) if actual == *sum => {}
            _ => return Err(Error::msg("observation bytes changed")),
        }
    }
    Ok(())
}

fn split_record(file: &str) -> (String, String) {
    match file.rsplit_once('/') {
        Some((parent, base)) => {
            let parent = if parent.is_empty() { "/".to_string() } else { parent.to_string() };
            (parent, base.to_string())
        }
        None => (".".to_string(), file.to_string()),
    }
}

/// Read one finalized observation, binding decoded metadata and all
/// retained-file hashes through one open directory, even if its
/// pathname is renamed during the read.
pub fn read_observation(file: &str) -> Result<(Observation, String), Error> {
    let (parent, base) = split_record(file);
    let root = OwnedDir::open(&parent)?;
    let (value, hash) = jsonio::read_json_at(&root, &base)?;
    let observation = observation_from_json(&value)?;
    observation_artifacts_valid(&observation)?;
    observation_files_match(&root, &observation)?;
    Ok((observation, hash))
}

fn admit_handoff_observation(o: &Observation, arch: &str, revision_text: &str) -> Result<(), Error> {
    if !valid_owner(&o.owner) {
        return Err(Error::msg("unknown observation owner"));
    }
    match o.outcome.as_str() {
        "completed" => {
            let files_empty = o.files.as_ref().map(|files| files.is_empty()).unwrap_or(true);
            if o.execution != "completed" || o.evidence != "completed" || files_empty {
                return Err(Error::msg("completed observation lacks execution/evidence"));
            }
        }
        "failed" | "cancelled" => {}
        _ => return Err(Error::msg("unknown observation outcome")),
    }
    if o.requested_revision != revision_text || o.requested_architecture != arch {
        return Err(Error::msg("observation belongs to a different candidate/platform"));
    }
    Ok(())
}

fn append_missing_owners(text: &mut String, seen: &std::collections::BTreeSet<String>) {
    text.push_str("\n## Not reached / not supplied\n\n");
    let mut owners = ["P02", "P03", "P04", "P05", "P06", "P11", "U08", "U20"];
    owners.sort();
    for owner in owners {
        if !seen.contains(owner) {
            text.push_str(&format!("- {owner}: no observation supplied (not a pass).\n"));
        }
    }
}

fn handoff_description(o: &Observation) -> String {
    // Anonymous-struct field order with Go string escaping; the
    // artifacts map renders sorted.
    let mut entries = Vec::with_capacity(10);
    let mut field = |name: &str, value: JsonValue| entries.push((name.to_string(), value));
    field("Owner", JsonValue::Str(o.owner.clone()));
    field("Target", JsonValue::Str(o.target.clone()));
    field("Outcome", JsonValue::Str(o.outcome.clone()));
    field("Execution", JsonValue::Str(o.execution.clone()));
    field("Evidence", JsonValue::Str(o.evidence.clone()));
    field("Cleanup", JsonValue::Str(o.cleanup.clone()));
    field("Topology", JsonValue::Str(o.topology.clone()));
    field(
        "ExitCode",
        o.exit_code.map(|code| JsonValue::Number(code.to_string())).unwrap_or(JsonValue::Null),
    );
    field(
        "Invocation",
        o.invocation
            .as_ref()
            .map(|args| JsonValue::Array(args.iter().map(|arg| JsonValue::Str(arg.clone())).collect()))
            .unwrap_or(JsonValue::Null),
    );
    field("Artifacts", o.artifacts.as_ref().map(map_json).unwrap_or(JsonValue::Null));
    let mut out = String::new();
    jsonio::write_compact(&mut out, &JsonValue::Object(entries));
    out
}

/// Cite exact observations and explicitly leave missing scopes open.
/// It can complete without core product evidence and never certifies
/// readiness.
pub fn handoff(out: &str, arch: &str, revision_text: &str, records: &[String]) -> Result<(), Error> {
    files::private_destination(out)?;
    oci_architecture(arch)?;
    if !revision(revision_text) {
        return Err(Error::msg("full candidate revision required"));
    }
    let mut text = format!("# Native support handoff\n\nCandidate: `{revision_text}` / `{arch}`.\n\nProduct readiness: **not assessed here; U20 owns it**. No ISO/QCOW2 delivery selected by this report. Sibling architecture is independent.\n\nCandidate/target fields are requested identities, not independently discovered facts for arbitrary exec commands. Read each owner's invoked check and retained artifact references for actual binding. Artifact references are historical digests, not a fresh verification of files at their former locations.\n\n");
    let mut seen = std::collections::BTreeSet::new();
    for file in records {
        let base = file.rsplit_once('/').map(|(_, base)| base).unwrap_or(file);
        if base != "observation.json" {
            return Err(Error::msg("finalized observation.json required"));
        }
        let (o, hash) = read_observation(file)?;
        admit_handoff_observation(&o, arch, revision_text)?;
        // JSON quoting avoids Markdown/control injection from an external log index.
        let description = handoff_description(&o);
        let mut location = String::new();
        jsonio::escape_go(&mut location, file);
        text.push_str(&format!("    {description}\n    record-sha256: {hash}\n    path: {location}\n\n"));
        seen.insert(o.owner.clone());
    }
    append_missing_owners(&mut text, &seen);
    text.push_str("\nP07/P08 redirect to core U08/U20. P09/P10 remain not selected. A command completing, a hash matching or a version printing proves only that observation. Retained paths and provider cleanup must be reviewed alongside the cited logs; missing cleanup is not inferred successful. Failed/cancelled/evidence-failed observations remain failures. This report performs no tests, retries, provider mutations or publication.\n");
    files::write_new(out, text.as_bytes(), 0o600)
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::evidence::create_evidence;
    use crate::files::TempDir;

    #[test]
    fn owner_and_shape_gates() {
        for owner in ["P02", "P03", "P04", "P05", "P06", "P11", "U01", "U08", "U09", "U10", "U19", "U20"] {
            assert!(valid_owner(owner), "{owner}");
        }
        for owner in ["", "P07", "P08", "P01", "P12", "U00", "U21", "U8", "u08", "P0", "U200"] {
            assert!(!valid_owner(owner), "{owner}");
        }
        assert_eq!(oci_architecture("x86_64").unwrap(), "amd64");
        assert!(oci_architecture("aarch64").is_err());
        assert!(revision(&"a".repeat(40)));
        assert!(!revision(&"a".repeat(39)));
        assert!(!revision(&"A".repeat(40)));
        assert!(digest(&"b".repeat(64)));
        assert!(!digest(&"b".repeat(63)));
    }

    #[test]
    fn observation_json_round_trip_keeps_nulls() {
        let mut files = BTreeMap::new();
        files.insert("check.stdout".to_string(), "c".repeat(64));
        let observation = Observation {
            owner: "P06".to_string(),
            requested_revision: "a".repeat(40),
            requested_architecture: "x86_64".to_string(),
            target: "synthetic".to_string(),
            outcome: "failed".to_string(),
            execution: "failed".to_string(),
            evidence: "completed".to_string(),
            files: Some(files),
            started: "2026-10-04T00:00:00Z".to_string(),
            finished: "2026-10-04T00:01:00Z".to_string(),
            ..Observation::default()
        };
        let decoded = observation_from_json(&observation_json(&observation)).unwrap();
        assert_eq!(decoded, observation);
        assert!(decoded.exit_code.is_none());
        assert!(decoded.invocation.is_none());
        assert!(decoded.artifacts.is_none());
        // Unknown fields and mistyped values fail, like DisallowUnknownFields.
        let mut bad = observation_json(&observation);
        if let JsonValue::Object(entries) = &mut bad {
            entries.push(("Extra".to_string(), JsonValue::Bool(true)));
        }
        assert!(observation_from_json(&bad).is_err());
        let mistyped = JsonValue::Object(vec![("ExitCode".to_string(), JsonValue::Str("0".to_string()))]);
        assert!(observation_from_json(&mistyped).is_err());
        let bad_time = JsonValue::Object(vec![("Started".to_string(), JsonValue::Str("not-a-time".to_string()))]);
        assert!(observation_from_json(&bad_time).is_err());
    }

    /// Port of `TestHandoffPreservesMissingAndFailedScopes` from `report_test.go`.
    #[test]
    fn handoff_preserves_missing_and_failed_scopes() {
        let scratch = TempDir::new("report").unwrap();
        let evidence_path = scratch.join("evidence").to_string_lossy().into_owned();
        let evidence = create_evidence(&evidence_path, &[]).unwrap();
        evidence.write("check.stdout", b"synthetic failed observation\n").unwrap();
        let file_hashes = hashes(&evidence).unwrap();
        let revision_text = "a".repeat(40);
        let observation = Observation {
            owner: "P06".to_string(),
            requested_revision: revision_text.clone(),
            requested_architecture: "x86_64".to_string(),
            target: "synthetic".to_string(),
            outcome: "failed".to_string(),
            execution: "failed".to_string(),
            evidence: "completed".to_string(),
            files: Some(file_hashes),
            started: "2026-10-04T00:00:00Z".to_string(),
            finished: "2026-10-04T00:01:00Z".to_string(),
            ..Observation::default()
        };
        let mut compact = String::new();
        jsonio::write_compact(&mut compact, &observation_json(&observation));
        evidence.write("observation.json", compact.as_bytes()).unwrap();
        let parent = TempDir::new("handoff").unwrap();
        let out = parent.join("handoff.md").to_string_lossy().into_owned();
        let record = format!("{evidence_path}/observation.json");
        handoff(&out, "x86_64", &revision_text, &[record.clone()]).unwrap();
        let text = std::fs::read_to_string(&out).unwrap();
        for required in ["failed", "U20 owns it", "U08: no observation supplied", "P09/P10 remain not selected"] {
            assert!(text.contains(required), "missing {required:?}");
        }
        let wrong = parent.join("wrong.md").to_string_lossy().into_owned();
        assert!(handoff(&wrong, "aarch64", &revision_text, &[record.clone()]).is_err());
        std::fs::write(format!("{evidence_path}/check.stdout"), b"changed").unwrap();
        let changed = parent.join("changed.md").to_string_lossy().into_owned();
        assert!(handoff(&changed, "x86_64", &revision_text, &[record]).is_err());
    }

    #[test]
    fn handoff_rejects_unsafe_references() {
        let scratch = TempDir::new("report").unwrap();
        let evidence_path = scratch.join("evidence").to_string_lossy().into_owned();
        let evidence = create_evidence(&evidence_path, &[]).unwrap();
        let revision_text = "a".repeat(40);
        let mut files = BTreeMap::new();
        files.insert("../escape".to_string(), "c".repeat(64));
        let observation = Observation {
            owner: "P06".to_string(),
            requested_revision: revision_text.clone(),
            requested_architecture: "x86_64".to_string(),
            outcome: "failed".to_string(),
            files: Some(files),
            started: "2026-10-04T00:00:00Z".to_string(),
            finished: "2026-10-04T00:01:00Z".to_string(),
            ..Observation::default()
        };
        let mut compact = String::new();
        jsonio::write_compact(&mut compact, &observation_json(&observation));
        evidence.write("observation.json", compact.as_bytes()).unwrap();
        let parent = TempDir::new("handoff").unwrap();
        let out = parent.join("handoff.md").to_string_lossy().into_owned();
        let err = handoff(&out, "x86_64", &revision_text, &[format!("{evidence_path}/observation.json")]).unwrap_err();
        assert_eq!(err.to_string(), "unsafe observation reference");
    }
}
