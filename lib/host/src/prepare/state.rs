// Helper-inspection DTO: only these observations cross the JSON boundary.
use crate::json;
use crate::preparation::{self, PrepareState, ResolvedTool};
use serde::de::{self, MapAccess, Visitor};
use serde::Deserialize;
use std::fmt;

#[derive(Debug, Clone, PartialEq, Eq)]
pub enum PreparationObservationError {
    NotFound,
    Invalid(String),
}

impl fmt::Display for PreparationObservationError {
    fn fmt(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
        match self {
            Self::NotFound => f.write_str("preparation not found"),
            Self::Invalid(message) => f.write_str(message),
        }
    }
}

impl std::error::Error for PreparationObservationError {}

#[derive(Default)]
struct HelperInspection {
    known: Option<bool>,
    phase: String,
    role: String,
    setup_digest: String,
    source_commit: String,
    tools: Vec<ResolvedTool>,
    missing: String,
    stopped: bool,
    ready: bool,
    setup_exit: Option<i64>,
    check_exit: Option<i64>,
    setup_log: String,
    check_log: String,
    hold: preparation::HoldState,
    verified: LauncherEvidence,
}

fn visit_inspection_text<'de, A>(
    map: &mut A,
    out: &mut HelperInspection,
    key: &str,
) -> Result<bool, A::Error>
where
    A: MapAccess<'de>,
{
    let slot = if key.eq_ignore_ascii_case("phase") {
        Some(&mut out.phase)
    } else if key.eq_ignore_ascii_case("role") {
        Some(&mut out.role)
    } else if key.eq_ignore_ascii_case("setup_digest") {
        Some(&mut out.setup_digest)
    } else if key.eq_ignore_ascii_case("source_commit") {
        Some(&mut out.source_commit)
    } else {
        None
    };
    if let Some(slot) = slot {
        if let Some(v) = map.next_value::<Option<String>>()? {
            *slot = v;
        }
        return Ok(true);
    }
    let slot = if key.eq_ignore_ascii_case("missing") {
        Some(&mut out.missing)
    } else if key.eq_ignore_ascii_case("setup_log") {
        Some(&mut out.setup_log)
    } else if key.eq_ignore_ascii_case("check_log") {
        Some(&mut out.check_log)
    } else {
        None
    };
    if let Some(slot) = slot {
        if let Some(v) = map.next_value::<Option<String>>()? {
            *slot = v;
        }
        return Ok(true);
    }
    Ok(false)
}

fn visit_inspection_flag<'de, A>(
    map: &mut A,
    out: &mut HelperInspection,
    key: &str,
) -> Result<bool, A::Error>
where
    A: MapAccess<'de>,
{
    if key.eq_ignore_ascii_case("known") {
        out.known = map.next_value::<Option<bool>>()?;
        return Ok(true);
    }
    if key.eq_ignore_ascii_case("stopped") {
        if let Some(v) = map.next_value::<Option<bool>>()? {
            out.stopped = v;
        }
        return Ok(true);
    }
    if key.eq_ignore_ascii_case("ready") {
        if let Some(v) = map.next_value::<Option<bool>>()? {
            out.ready = v;
        }
        return Ok(true);
    }
    Ok(false)
}

fn visit_inspection_exit<'de, A>(
    map: &mut A,
    out: &mut HelperInspection,
    key: &str,
) -> Result<bool, A::Error>
where
    A: MapAccess<'de>,
{
    if key.eq_ignore_ascii_case("setup_exit") {
        if let Some(v) = map.next_value::<Option<json::SignedInteger>>()? {
            out.setup_exit = Some(v.0);
        }
        return Ok(true);
    }
    if key.eq_ignore_ascii_case("check_exit") {
        if let Some(v) = map.next_value::<Option<json::SignedInteger>>()? {
            out.check_exit = Some(v.0);
        }
        return Ok(true);
    }
    Ok(false)
}

fn visit_inspection_structured<'de, A>(
    map: &mut A,
    out: &mut HelperInspection,
    key: &str,
) -> Result<bool, A::Error>
where
    A: MapAccess<'de>,
{
    if key.eq_ignore_ascii_case("tools") {
        if let Some(v) = map.next_value::<Option<Vec<ResolvedTool>>>()? {
            out.tools = v;
        }
        return Ok(true);
    }
    if key.eq_ignore_ascii_case("hold") {
        if let Some(v) = map.next_value::<Option<preparation::HoldState>>()? {
            out.hold = v;
        }
        return Ok(true);
    }
    if key.eq_ignore_ascii_case("verified") {
        if let Some(v) = map.next_value::<Option<LauncherEvidence>>()? {
            out.verified = v;
        }
        return Ok(true);
    }
    Ok(false)
}

impl<'de> Deserialize<'de> for HelperInspection {
    fn deserialize<D>(deserializer: D) -> Result<Self, D::Error>
    where
        D: serde::Deserializer<'de>,
    {
        struct InspectionVisitor;
        impl<'de> Visitor<'de> for InspectionVisitor {
            type Value = HelperInspection;
            fn expecting(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
                f.write_str("a preparation inspection object")
            }
            fn visit_map<A>(self, mut map: A) -> Result<Self::Value, A::Error>
            where
                A: MapAccess<'de>,
            {
                let mut out = HelperInspection::default();
                while let Some(key) = map.next_key::<String>()? {
                    if visit_inspection_text(&mut map, &mut out, &key)? {
                        continue;
                    }
                    if visit_inspection_flag(&mut map, &mut out, &key)? {
                        continue;
                    }
                    if visit_inspection_exit(&mut map, &mut out, &key)? {
                        continue;
                    }
                    if visit_inspection_structured(&mut map, &mut out, &key)? {
                        continue;
                    }
                    return Err(de::Error::unknown_field(
                        &key,
                        &[
                            "known",
                            "phase",
                            "role",
                            "setup_digest",
                            "source_commit",
                            "tools",
                            "missing",
                            "stopped",
                            "ready",
                            "setup_exit",
                            "check_exit",
                            "setup_log",
                            "check_log",
                            "hold",
                            "verified",
                        ],
                    ));
                }
                Ok(out)
            }
        }
        deserializer.deserialize_map(InspectionVisitor)
    }
}

/// Launcher evidence: observed role identity plus any refusal reason.
#[derive(Debug, Clone, Default, PartialEq, Eq)]
pub struct LauncherEvidence {
    pub uid: String,
    pub login: String,
    pub groups: String,
    pub refusal: String,
}

impl<'de> Deserialize<'de> for LauncherEvidence {
    fn deserialize<D>(deserializer: D) -> Result<Self, D::Error>
    where
        D: serde::Deserializer<'de>,
    {
        struct EvidenceVisitor;
        impl<'de> Visitor<'de> for EvidenceVisitor {
            type Value = LauncherEvidence;
            fn expecting(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
                f.write_str("launcher evidence object")
            }
            fn visit_map<A>(self, mut map: A) -> Result<Self::Value, A::Error>
            where
                A: MapAccess<'de>,
            {
                let mut out = LauncherEvidence::default();
                while let Some(key) = map.next_key::<String>()? {
                    if key.eq_ignore_ascii_case("uid") {
                        if let Some(v) = map.next_value::<Option<String>>()? {
                            out.uid = v;
                        }
                    } else if key.eq_ignore_ascii_case("login") {
                        if let Some(v) = map.next_value::<Option<String>>()? {
                            out.login = v;
                        }
                    } else if key.eq_ignore_ascii_case("groups") {
                        if let Some(v) = map.next_value::<Option<String>>()? {
                            out.groups = v;
                        }
                    } else if key.eq_ignore_ascii_case("refusal") {
                        if let Some(v) = map.next_value::<Option<String>>()? {
                            out.refusal = v;
                        }
                    } else {
                        return Err(de::Error::unknown_field(
                            &key,
                            &["uid", "login", "groups", "refusal"],
                        ));
                    }
                }
                Ok(out)
            }
        }
        deserializer.deserialize_map(EvidenceVisitor)
    }
}

pub(crate) fn quote_bytes_base64(out: &mut String, bytes: &[u8]) {
    out.push_str(&json::quote(&crate::ssh::b64_encode(bytes)));
}

const INVALID_OBSERVATION: &str = "invalid preparation observation";

fn decode_inspection(raw: &[u8]) -> Result<HelperInspection, PreparationObservationError> {
    json::decode_strict_as(raw)
        .map_err(|_| PreparationObservationError::Invalid(INVALID_OBSERVATION.to_string()))
}

fn check_unknown_identity_empty(
    observed: &HelperInspection,
) -> Result<(), PreparationObservationError> {
    if !observed.phase.is_empty()
        || !observed.role.is_empty()
        || !observed.setup_digest.is_empty()
        || !observed.source_commit.is_empty()
        || !observed.tools.is_empty()
        || !observed.missing.is_empty()
    {
        return Err(PreparationObservationError::Invalid(
            INVALID_OBSERVATION.to_string(),
        ));
    }
    Ok(())
}

fn check_unknown_progress_empty(
    observed: &HelperInspection,
) -> Result<(), PreparationObservationError> {
    if observed.stopped
        || observed.ready
        || observed.setup_exit.is_some()
        || observed.check_exit.is_some()
        || !observed.setup_log.is_empty()
        || !observed.check_log.is_empty()
    {
        return Err(PreparationObservationError::Invalid(
            INVALID_OBSERVATION.to_string(),
        ));
    }
    Ok(())
}

fn check_known_fields(observed: &HelperInspection) -> Result<(), PreparationObservationError> {
    if !preparation::valid_prepare_phase(&observed.phase)
        || !preparation::valid_factory_role(&observed.role)
        || !preparation::valid_digest(&observed.setup_digest)
        || !preparation::valid_commit(&observed.source_commit)
    {
        return Err(PreparationObservationError::Invalid(
            INVALID_OBSERVATION.to_string(),
        ));
    }
    if observed.setup_log.len() > 66560 || observed.check_log.len() > 66560 {
        return Err(PreparationObservationError::Invalid(
            "preparation observation exceeds the bounded size".to_string(),
        ));
    }
    Ok(())
}

fn build_known_state(container: &str, observed: HelperInspection) -> PrepareState {
    let HelperInspection {
        known: _,
        phase,
        role,
        setup_digest,
        source_commit,
        tools,
        missing,
        stopped,
        ready,
        setup_exit,
        check_exit,
        setup_log,
        check_log,
        hold: _,
        verified: _,
    } = observed;
    let mut state = PrepareState {
        role,
        phase,
        container: container.to_string(),
        source_commit,
        setup_digest,
        tools,
        missing,
        setup_exit,
        check_exit,
        ready,
        stopped,
        ..Default::default()
    };
    if !setup_log.is_empty() || !check_log.is_empty() {
        state.output = format!("--- setup ---\n{setup_log}\n--- check ---\n{check_log}");
    }
    state
}

/// `mapPreparationState`: validated observation plus container identity.
/// ID and project are filled in by the caller.
pub fn map_preparation_state(
    container: &str,
    raw: &[u8],
) -> Result<PrepareState, PreparationObservationError> {
    let observed = decode_inspection(raw)?;
    let Some(known) = observed.known else {
        return Err(PreparationObservationError::Invalid(
            INVALID_OBSERVATION.to_string(),
        ));
    };
    if !known {
        check_unknown_identity_empty(&observed)?;
        check_unknown_progress_empty(&observed)?;
        return Err(PreparationObservationError::NotFound);
    }
    check_known_fields(&observed)?;
    let state = build_known_state(container, observed);
    if state.ready != (state.phase == preparation::PREPARE_READY) {
        return Err(PreparationObservationError::Invalid(
            "preparation observation is inconsistent".to_string(),
        ));
    }
    Ok(state)
}
