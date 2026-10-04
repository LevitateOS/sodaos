//! `admission.go`: qualification admission for protected final signing.

use std::collections::{BTreeMap, BTreeSet};

use sha2::{Digest as _, Sha256};
use soda_json::JsonValue;

use crate::buildx::{read_at, Root};
use crate::document::read_file;
use crate::finalize::Config;
use crate::jsonx::{parse_strict, Binder, Emit, Emitter};
use crate::model::{Candidate, MediaBinding};
use crate::payload::{
    decode_opt_bool, decode_opt_i64, decode_opt_string, Payload,
};
use crate::prepare::{read_media_binding, Qualification};
use crate::Error;

/// `QualificationScope`: the only evidence scope finalization admits.
pub const QUALIFICATION_SCOPE: &str = "native-install-upgrade-recovery";

/// `EvidenceCheck`: one required native observation.
#[derive(Debug, Clone, Default, PartialEq, Eq)]
pub struct EvidenceCheck {
    pub name: String,
    pub outcome: String,
    pub detail: String,
}

impl EvidenceCheck {
    pub fn decode(value: &JsonValue) -> Result<EvidenceCheck, String> {
        let mut b = Binder::new(value).map_err(|_| "invalid check".to_string())?;
        let check = EvidenceCheck {
            name: decode_opt_string(&mut b, "Name")?,
            outcome: decode_opt_string(&mut b, "Outcome")?,
            detail: decode_opt_string(&mut b, "Detail")?,
        };
        b.finish_name()?;
        Ok(check)
    }
}

impl Emit for EvidenceCheck {
    fn emit(&self, e: &mut Emitter) {
        e.begin_object(false);
        e.field(true, "Name");
        e.string(&self.name);
        e.field(false, "Outcome");
        e.string(&self.outcome);
        e.field(false, "Detail");
        e.string(&self.detail);
        e.end_object(false);
    }
}

/// `QualificationEvidence`: exact protected-worker qualification record.
#[derive(Debug, Clone, Default, PartialEq, Eq)]
pub struct QualificationEvidence {
    pub format: i64,
    pub outcome: String,
    pub scope: String,
    pub revision: String,
    pub architecture: String,
    pub payload_sha256: String,
    pub host_manifest: String,
    pub iso_sha256: String,
    pub rootfs_sha256: String,
    pub checks: Vec<EvidenceCheck>,
    pub fixture: bool,
}

impl QualificationEvidence {
    pub fn decode(value: &JsonValue) -> Result<QualificationEvidence, String> {
        let mut b = Binder::new(value).map_err(|_| "invalid evidence".to_string())?;
        let mut evidence = QualificationEvidence {
            format: decode_opt_i64(&mut b, "Format")?,
            outcome: decode_opt_string(&mut b, "Outcome")?,
            scope: decode_opt_string(&mut b, "Scope")?,
            revision: decode_opt_string(&mut b, "Revision")?,
            architecture: decode_opt_string(&mut b, "Architecture")?,
            payload_sha256: decode_opt_string(&mut b, "PayloadSHA256")?,
            host_manifest: decode_opt_string(&mut b, "HostManifest")?,
            iso_sha256: decode_opt_string(&mut b, "ISOSHA256")?,
            rootfs_sha256: decode_opt_string(&mut b, "RootfsSHA256")?,
            checks: Vec::new(),
            fixture: decode_opt_bool(&mut b, "Fixture")?,
        };
        if let Some(items) = b
            .array("Checks")
            .map_err(|_| "invalid field Checks".to_string())?
        {
            for item in items {
                evidence.checks.push(EvidenceCheck::decode(item)?);
            }
        }
        b.finish_name()?;
        Ok(evidence)
    }
}

impl Emit for QualificationEvidence {
    fn emit(&self, e: &mut Emitter) {
        e.begin_object(false);
        e.field(true, "Format");
        e.int(self.format);
        e.field(false, "Outcome");
        e.string(&self.outcome);
        e.field(false, "Scope");
        e.string(&self.scope);
        e.field(false, "Revision");
        e.string(&self.revision);
        e.field(false, "Architecture");
        e.string(&self.architecture);
        e.field(false, "PayloadSHA256");
        e.string(&self.payload_sha256);
        e.field(false, "HostManifest");
        e.string(&self.host_manifest);
        e.field(false, "ISOSHA256");
        e.string(&self.iso_sha256);
        e.field(false, "RootfsSHA256");
        e.string(&self.rootfs_sha256);
        e.field(false, "Checks");
        e.begin_array(self.checks.is_empty());
        for (i, check) in self.checks.iter().enumerate() {
            e.item(i == 0);
            check.emit(e);
        }
        e.end_array(self.checks.is_empty());
        e.field(false, "Fixture");
        e.boolean(self.fixture);
        e.end_object(false);
    }
}

fn required_qualification_checks() -> Vec<String> {
    ["install", "upgrade", "recovery", "preservation"]
        .iter()
        .map(|s| s.to_string())
        .collect()
}

pub(crate) fn admit_release_identity(c: &Config) -> Result<(), Error> {
    if c.serial == 0
        || (c.class != "normal" && c.class != "emergency")
        || c.notes.is_empty()
        || c.notes.len() > 16384
    {
        return Err(Error::msg("exact release serial, class and notes required"));
    }
    Ok(())
}

fn sha256_hex(data: &[u8]) -> String {
    let mut hasher = Sha256::new();
    hasher.update(data);
    format!("{:x}", hasher.finalize())
}

fn decode_qualification_evidence(
    path: &str,
) -> Result<(QualificationEvidence, Vec<u8>), Error> {
    let raw = read_file(path, 1 << 20)?;
    let value = parse_strict(&raw).map_err(|_| Error::msg("qualification evidence refused"))?;
    let evidence =
        QualificationEvidence::decode(&value).map_err(|_| Error::msg("qualification evidence refused"))?;
    Ok((evidence, raw))
}

fn admit_evidence_shape(evidence: &QualificationEvidence) -> Result<(), Error> {
    if evidence.format != 1 {
        return Err(Error::msg("qualification evidence format refused"));
    }
    if evidence.fixture {
        return Err(Error::msg("fixture evidence is explicitly non-qualifying"));
    }
    if evidence.outcome != "passed" {
        return Err(Error::msg("qualification evidence outcome is not passed"));
    }
    if evidence.scope != QUALIFICATION_SCOPE {
        return Err(Error::msg(
            "qualification evidence scope is not native install/upgrade/recovery",
        ));
    }
    Ok(())
}

fn admit_evidence_check(check: &EvidenceCheck, seen: &mut BTreeSet<String>) -> Result<(), Error> {
    if !seen.remove(&check.name) {
        return Err(Error::msg("qualification evidence names an unexpected check"));
    }
    if check.outcome != "passed" {
        return Err(Error::msg(format!(
            "qualification evidence check did not pass: {}",
            check.name
        )));
    }
    if check.detail.len() > 1024 {
        return Err(Error::msg(format!(
            "qualification evidence check detail too long: {}",
            check.name
        )));
    }
    Ok(())
}

fn admit_evidence_checks(evidence: &QualificationEvidence) -> Result<(), Error> {
    let mut seen: BTreeSet<String> = required_qualification_checks().into_iter().collect();
    for check in &evidence.checks {
        admit_evidence_check(check, &mut seen)?;
    }
    if !seen.is_empty() {
        return Err(Error::msg("qualification evidence check set is incomplete"));
    }
    Ok(())
}

pub(crate) fn load_admitted_candidate(
    candidate: &str,
) -> Result<(Payload, Candidate, Vec<u8>), Error> {
    let root = Root::open(candidate)?;
    let payload = read_at(&root, "payload.json", 1 << 20)?;
    let raw = read_at(&root, "candidate.json", 1 << 20)?;
    let payload_value = parse_strict(&payload).map_err(|_| Error::msg("candidate metadata refused"))?;
    let candidate_value = parse_strict(&raw).map_err(|_| Error::msg("candidate metadata refused"))?;
    let p = Payload::decode(&payload_value).map_err(|_| Error::msg("candidate metadata refused"))?;
    let c = Candidate::decode(&candidate_value).map_err(|_| Error::msg("candidate metadata refused"))?;
    c.validate(&p, &payload)?;
    Ok((p, c, payload))
}

fn admit_evidence_candidate(
    evidence: &QualificationEvidence,
    p: &Payload,
    c: &Candidate,
    payload: &[u8],
) -> Result<(), Error> {
    if evidence.revision != p.revision || evidence.architecture != p.architecture {
        return Err(Error::at("qualification evidence candidate identity"));
    }
    if evidence.payload_sha256 != sha256_hex(payload) {
        return Err(Error::at("qualification evidence payload binding"));
    }
    if evidence.host_manifest != c.host.manifest {
        return Err(Error::at("qualification evidence host binding"));
    }
    Ok(())
}

fn admit_evidence_media(
    evidence: &QualificationEvidence,
    media: &str,
    p: &Payload,
    c: &Candidate,
) -> Result<(), Error> {
    let raw = read_media_binding(media)?;
    let binding = MediaBinding::decode_lenient(&raw)
        .map_err(|_| Error::at("qualification evidence media binding"))?;
    // Go checks json.Unmarshal plus validMediaBinding here.
    if !crate::model::valid_media_binding(&binding, p, c) {
        return Err(Error::at("qualification evidence media binding"));
    }
    if evidence.iso_sha256 != binding.iso.sha256 || evidence.rootfs_sha256 != binding.rootfs.sha256
    {
        return Err(Error::at("qualification evidence media identity"));
    }
    Ok(())
}

/// `AdmitQualification`: enforce the release contract on one evidence record.
pub fn admit_qualification(
    c: &Config,
    candidate: &str,
    media: &str,
    evidence_path: &str,
) -> Result<Qualification, Error> {
    admit_release_identity(c)?;
    let (evidence, raw) = decode_qualification_evidence(evidence_path)?;
    admit_evidence_shape(&evidence)?;
    admit_evidence_checks(&evidence)?;
    let (p, candidate_data, payload) = load_admitted_candidate(candidate)?;
    admit_evidence_candidate(&evidence, &p, &candidate_data, &payload)?;
    admit_evidence_media(&evidence, media, &p, &candidate_data)?;
    let mut evidence_map = BTreeMap::new();
    evidence_map.insert(
        "qualification.json".to_string(),
        format!("sha256:{}", sha256_hex(&raw)),
    );
    Ok(Qualification {
        serial: c.serial,
        class: c.class.clone(),
        scope: QUALIFICATION_SCOPE.to_string(),
        notes: c.notes.clone(),
        evidence: evidence_map,
    })
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn release_identity_rules() {
        let good = Config {
            serial: 1,
            class: "normal".to_string(),
            notes: "n".to_string(),
            ..Config::default()
        };
        assert!(admit_release_identity(&good).is_ok());
        let bad = Config::default();
        assert_eq!(
            admit_release_identity(&bad).unwrap_err(),
            Error::msg("exact release serial, class and notes required")
        );
        let mut bad_class = good.clone();
        bad_class.class = "bogus".to_string();
        assert!(admit_release_identity(&bad_class).is_err());
    }

    #[test]
    fn evidence_shape_and_checks() {
        let mut evidence = QualificationEvidence {
            format: 1,
            outcome: "passed".to_string(),
            scope: QUALIFICATION_SCOPE.to_string(),
            ..QualificationEvidence::default()
        };
        assert!(admit_evidence_shape(&evidence).is_ok());
        evidence.fixture = true;
        assert_eq!(
            admit_evidence_shape(&evidence).unwrap_err(),
            Error::msg("fixture evidence is explicitly non-qualifying")
        );
        evidence.fixture = false;
        evidence.outcome = "failed".to_string();
        assert!(admit_evidence_shape(&evidence).is_err());
        evidence.outcome = "passed".to_string();
        assert_eq!(
            admit_evidence_checks(&evidence).unwrap_err(),
            Error::msg("qualification evidence check set is incomplete")
        );
        for name in ["install", "upgrade", "recovery", "preservation"] {
            evidence.checks.push(EvidenceCheck {
                name: name.to_string(),
                outcome: "passed".to_string(),
                detail: String::new(),
            });
        }
        assert!(admit_evidence_checks(&evidence).is_ok());
        evidence.checks.push(EvidenceCheck {
            name: "extra".to_string(),
            outcome: "passed".to_string(),
            detail: String::new(),
        });
        assert_eq!(
            admit_evidence_checks(&evidence).unwrap_err(),
            Error::msg("qualification evidence names an unexpected check")
        );
    }
}
