use std::collections::BTreeMap;

use soda_json::JsonValue;

use super::*;
use crate::evidence::create_evidence;
use crate::files::TempDir;
use crate::jsonio;

#[test]
fn owner_and_shape_gates() {
    for owner in [
        "P02", "P03", "P04", "P05", "P06", "P11", "U01", "U08", "U09", "U10", "U19", "U20",
    ] {
        assert!(valid_owner(owner), "{owner}");
    }
    for owner in [
        "", "P07", "P08", "P01", "P12", "U00", "U21", "U8", "u08", "P0", "U200",
    ] {
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
    let mistyped = JsonValue::Object(vec![(
        "ExitCode".to_string(),
        JsonValue::Str("0".to_string()),
    )]);
    assert!(observation_from_json(&mistyped).is_err());
    let bad_time = JsonValue::Object(vec![(
        "Started".to_string(),
        JsonValue::Str("not-a-time".to_string()),
    )]);
    assert!(observation_from_json(&bad_time).is_err());
}

/// Port of `TestHandoffPreservesMissingAndFailedScopes` from `report_test.go`.
#[test]
fn handoff_preserves_missing_and_failed_scopes() {
    let scratch = TempDir::new("report").unwrap();
    let evidence_path = scratch.join("evidence").to_string_lossy().into_owned();
    let evidence = create_evidence(&evidence_path, &[]).unwrap();
    evidence
        .write("check.stdout", b"synthetic failed observation\n")
        .unwrap();
    let (file_hashes, hashes_err) = hashes(&evidence);
    assert!(hashes_err.is_none());
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
    evidence
        .write("observation.json", compact.as_bytes())
        .unwrap();
    let parent = TempDir::new("handoff").unwrap();
    let out = parent.join("handoff.md").to_string_lossy().into_owned();
    let record = format!("{evidence_path}/observation.json");
    handoff(
        &out,
        "x86_64",
        &revision_text,
        std::slice::from_ref(&record),
    )
    .unwrap();
    let text = std::fs::read_to_string(&out).unwrap();
    for required in [
        "failed",
        "U20 owns it",
        "U08: no observation supplied",
        "P09/P10 remain not selected",
    ] {
        assert!(text.contains(required), "missing {required:?}");
    }
    let wrong = parent.join("wrong.md").to_string_lossy().into_owned();
    assert!(handoff(
        &wrong,
        "aarch64",
        &revision_text,
        std::slice::from_ref(&record)
    )
    .is_err());
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
    evidence
        .write("observation.json", compact.as_bytes())
        .unwrap();
    let parent = TempDir::new("handoff").unwrap();
    let out = parent.join("handoff.md").to_string_lossy().into_owned();
    let err = handoff(
        &out,
        "x86_64",
        &revision_text,
        &[format!("{evidence_path}/observation.json")],
    )
    .unwrap_err();
    assert_eq!(err.to_string(), "unsafe observation reference");
}
