use crate::error::Error;
use crate::evidence::Evidence;
use crate::jsonio;
use crate::report::{self, Observation};

fn read_build_settings(observation: &mut Observation) {
    observation.tool_revision = option_env!("BUILD_REVISION")
        .unwrap_or("unknown")
        .to_string();
    observation.tool_dirty = option_env!("BUILD_MODIFIED").unwrap_or("false") == "true";
    if observation.tool_revision.is_empty() {
        observation.tool_revision = "unknown".to_string();
    }
}

fn redact_observation_strings(evidence: &Evidence, observation: &mut Observation) {
    if let Some(invocation) = observation.invocation.as_mut() {
        for arg in invocation {
            *arg = evidence.redact_string(arg);
        }
    }
    observation.topology = evidence.redact_string(&observation.topology);
    observation.cleanup = evidence.redact_string(&observation.cleanup);
}

/// Join error Displays like [`Error::join`]: single messages pass
/// through, several join with newlines. Used where the Go owner
/// formats a join it also keeps split; owned errors move only once.
fn combine_messages(errors: &[&Option<Error>]) -> Option<String> {
    let parts: Vec<String> = errors
        .iter()
        .filter_map(|e| e.as_ref().map(|e| e.to_string()))
        .collect();
    if parts.is_empty() {
        return None;
    }
    Some(parts.join("\n"))
}

pub(super) fn finalize_observation(
    evidence: &Evidence,
    mut observation: Observation,
    operation_err: Option<Error>,
    evidence_err: Option<Error>,
) -> Result<(), Error> {
    read_build_settings(&mut observation);
    let mut evidence_err = Error::join(vec![evidence_err, evidence.check_secrets().err()]);
    let operation_failed = operation_err.is_some();
    let operation_cancelled = operation_err.as_ref().is_some_and(|e| e.is_cancelled());
    if operation_failed || evidence_err.is_some() {
        // No success-shaped final record exists until retention has finalized.
        let combined = combine_messages(&[&operation_err, &evidence_err]).unwrap_or_default();
        let redacted = evidence.redact_string(&combined);
        evidence_err = Error::join(vec![
            evidence_err,
            evidence
                .write("failure.txt", format!("{redacted}\n").as_bytes())
                .err(),
        ]);
    }
    let (hashes, hashes_err) = report::hashes(evidence);
    observation.files = Some(hashes);
    evidence_err = Error::join(vec![evidence_err, hashes_err]);
    observation.evidence = if evidence_err.is_some() {
        "failed".to_string()
    } else {
        "completed".to_string()
    };
    observation.outcome = if operation_failed || evidence_err.is_some() {
        "failed".to_string()
    } else {
        "completed".to_string()
    };
    if operation_cancelled {
        observation.outcome = "cancelled".to_string();
    }
    observation.finished = jsonio::now_rfc3339_nano();
    redact_observation_strings(evidence, &mut observation);
    // No success-shaped final record exists until retention has finalized.
    let publish_err = evidence
        .publish_observation(&report::observation_json(&observation))
        .err();
    match combine_messages(&[&operation_err, &evidence_err, &publish_err]) {
        None => Ok(()),
        Some(combined) => Err(Error::msg(evidence.redact_string(&combined))),
    }
}
