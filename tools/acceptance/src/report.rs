//! Observation records and the native support handoff, mirroring
//! `internal/acceptance/report.go`.
//!
//! An [`Observation`] is an ordinary log index, not a
//! scenario/qualification registry. [`handoff`] cites exact observations
//! and explicitly leaves missing scopes open; it can complete without
//! core product evidence and never certifies readiness.

#[cfg(test)]
use std::collections::BTreeMap;

#[cfg(test)]
use soda_json::JsonValue;

use crate::error::Error;
#[cfg(test)]
use crate::jsonio;

mod handoff;
mod observation;

pub use handoff::{handoff, hashes, read_observation};
pub use observation::{observation_from_json, observation_json, Observation};

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
    if bytes.len() != 3
        || bytes[0] != b'U'
        || !bytes[1].is_ascii_digit()
        || !bytes[2].is_ascii_digit()
    {
        return false;
    }
    matches!(
        (bytes[1] - b'0', bytes[2] - b'0'),
        (0, 1..=9) | (1, 0..=9) | (2, 0)
    )
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::evidence::create_evidence;
    use crate::files::TempDir;

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
}
