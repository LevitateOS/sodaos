use std::collections::BTreeMap;

use serde::Serialize;

use crate::error::Error;
use crate::evidence::Evidence;
use crate::files::{self, OwnedDir};
use crate::jsonio;

use super::{digest, observation_from_json, oci_architecture, revision, valid_owner, Observation};

/// Hash every retained evidence file, like `Evidence.Hashes`. The map
/// is partial when the walk fails, like the Go owner's.
pub fn hashes(evidence: &Evidence) -> (BTreeMap<String, String>, Option<Error>) {
    let mut files = BTreeMap::new();
    let mut failure = None;
    let mut visit = |path: &str, regular: bool| -> Result<(), Error> {
        if !regular {
            failure = Some(Error::msg("unexpected evidence entry"));
            return Err(Error::msg("unexpected evidence entry"));
        }
        match files::hash_at(evidence.root(), path) {
            Ok(sum) => {
                files.insert(path.to_string(), sum);
                Ok(())
            }
            Err(err) => {
                let text = err.to_string();
                failure = Some(err);
                Err(Error::msg(text))
            }
        }
    };
    if evidence.root().walk_files(&mut visit).is_err() && failure.is_none() {
        failure = Some(Error::msg("unexpected evidence entry"));
    }
    (files, failure)
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
        if files::lexical_clean(name) != *name
            || name == "."
            || name.starts_with('/')
            || name == ".."
            || name.starts_with("../")
        {
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
            let parent = if parent.is_empty() {
                "/".to_string()
            } else {
                parent.to_string()
            };
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

fn admit_handoff_observation(
    o: &Observation,
    arch: &str,
    revision_text: &str,
) -> Result<(), Error> {
    if !valid_owner(&o.owner) {
        return Err(Error::msg("unknown observation owner"));
    }
    match o.outcome.as_str() {
        "completed" => {
            let files_empty = o
                .files
                .as_ref()
                .map(|files| files.is_empty())
                .unwrap_or(true);
            if o.execution != "completed" || o.evidence != "completed" || files_empty {
                return Err(Error::msg("completed observation lacks execution/evidence"));
            }
        }
        "failed" | "cancelled" => {}
        _ => return Err(Error::msg("unknown observation outcome")),
    }
    if o.requested_revision != revision_text || o.requested_architecture != arch {
        return Err(Error::msg(
            "observation belongs to a different candidate/platform",
        ));
    }
    Ok(())
}

fn append_missing_owners(text: &mut String, seen: &std::collections::BTreeSet<String>) {
    text.push_str("\n## Not reached / not supplied\n\n");
    let mut owners = ["P02", "P03", "P04", "P05", "P06", "P11", "U08", "U20"];
    owners.sort();
    for owner in owners {
        if !seen.contains(owner) {
            text.push_str(&format!(
                "- {owner}: no observation supplied (not a pass).\n"
            ));
        }
    }
}

fn handoff_description(o: &Observation) -> String {
    #[derive(Serialize)]
    struct Description<'a> {
        #[serde(rename = "Owner")]
        owner: &'a str,
        #[serde(rename = "Target")]
        target: &'a str,
        #[serde(rename = "Outcome")]
        outcome: &'a str,
        #[serde(rename = "Execution")]
        execution: &'a str,
        #[serde(rename = "Evidence")]
        evidence: &'a str,
        #[serde(rename = "Cleanup")]
        cleanup: &'a str,
        #[serde(rename = "Topology")]
        topology: &'a str,
        #[serde(rename = "ExitCode")]
        exit_code: Option<i64>,
        #[serde(rename = "Invocation")]
        invocation: Option<&'a [String]>,
        #[serde(rename = "Artifacts")]
        artifacts: Option<&'a BTreeMap<String, String>>,
    }
    let value = Description {
        owner: &o.owner,
        target: &o.target,
        outcome: &o.outcome,
        execution: &o.execution,
        evidence: &o.evidence,
        cleanup: &o.cleanup,
        topology: &o.topology,
        exit_code: o.exit_code,
        invocation: o.invocation.as_deref(),
        artifacts: o.artifacts.as_ref(),
    };
    let mut out = String::new();
    jsonio::write_compact(&mut out, &value);
    out
}

/// Cite exact observations and explicitly leave missing scopes open.
/// It can complete without core product evidence and never certifies
/// readiness.
pub fn handoff(
    out: &str,
    arch: &str,
    revision_text: &str,
    records: &[String],
) -> Result<(), Error> {
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
        text.push_str(&format!(
            "    {description}\n    record-sha256: {hash}\n    path: {location}\n\n"
        ));
        seen.insert(o.owner.clone());
    }
    append_missing_owners(&mut text, &seen);
    text.push_str("\nP07/P08 redirect to core U08/U20. P09/P10 remain not selected. A command completing, a hash matching or a version printing proves only that observation. Retained paths and provider cleanup must be reviewed alongside the cited logs; missing cleanup is not inferred successful. Failed/cancelled/evidence-failed observations remain failures. This report performs no tests, retries, provider mutations or publication.\n");
    files::write_new(out, text.as_bytes(), 0o600)
}
