// Helper-inspection DTO: only these observations cross the JSON boundary.
use crate::json;
use crate::preparation::{self, PrepareState, ResolvedTool};
use serde::de::{self, MapAccess, Visitor};
use serde::Deserialize;
use std::fmt;

#[derive(Default)]
struct HelperInspection {
    known: bool,
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
                    if key.eq_ignore_ascii_case("known") {
                        if let Some(v) = map.next_value::<Option<bool>>()? {
                            out.known = v;
                        }
                    } else if key.eq_ignore_ascii_case("phase") {
                        if let Some(v) = map.next_value::<Option<String>>()? {
                            out.phase = v;
                        }
                    } else if key.eq_ignore_ascii_case("role") {
                        if let Some(v) = map.next_value::<Option<String>>()? {
                            out.role = v;
                        }
                    } else if key.eq_ignore_ascii_case("setup_digest") {
                        if let Some(v) = map.next_value::<Option<String>>()? {
                            out.setup_digest = v;
                        }
                    } else if key.eq_ignore_ascii_case("source_commit") {
                        if let Some(v) = map.next_value::<Option<String>>()? {
                            out.source_commit = v;
                        }
                    } else if key.eq_ignore_ascii_case("tools") {
                        if let Some(v) = map.next_value::<Option<Vec<ResolvedTool>>>()? {
                            out.tools = v;
                        }
                    } else if key.eq_ignore_ascii_case("missing") {
                        if let Some(v) = map.next_value::<Option<String>>()? {
                            out.missing = v;
                        }
                    } else if key.eq_ignore_ascii_case("stopped") {
                        if let Some(v) = map.next_value::<Option<bool>>()? {
                            out.stopped = v;
                        }
                    } else if key.eq_ignore_ascii_case("ready") {
                        if let Some(v) = map.next_value::<Option<bool>>()? {
                            out.ready = v;
                        }
                    } else if key.eq_ignore_ascii_case("setup_exit") {
                        if let Some(v) = map.next_value::<Option<json::SignedInteger>>()? {
                            out.setup_exit = Some(v.0);
                        }
                    } else if key.eq_ignore_ascii_case("check_exit") {
                        if let Some(v) = map.next_value::<Option<json::SignedInteger>>()? {
                            out.check_exit = Some(v.0);
                        }
                    } else if key.eq_ignore_ascii_case("setup_log") {
                        if let Some(v) = map.next_value::<Option<String>>()? {
                            out.setup_log = v;
                        }
                    } else if key.eq_ignore_ascii_case("check_log") {
                        if let Some(v) = map.next_value::<Option<String>>()? {
                            out.check_log = v;
                        }
                    } else if key.eq_ignore_ascii_case("hold") {
                        if let Some(v) = map.next_value::<Option<preparation::HoldState>>()? {
                            out.hold = v;
                        }
                    } else if key.eq_ignore_ascii_case("verified") {
                        if let Some(v) = map.next_value::<Option<LauncherEvidence>>()? {
                            out.verified = v;
                        }
                    } else {
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

/// `mapPreparationState`: validated observation plus container identity.
/// ID and project are filled in by the caller.
pub fn map_preparation_state(container: &str, raw: &[u8]) -> Result<PrepareState, String> {
    const ERR: &str = "invalid preparation observation";
    let observed: HelperInspection = json::decode_strict_as(raw).map_err(|_| ERR.to_string())?;
    let HelperInspection {
        known,
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
    if !known
        || !preparation::valid_prepare_phase(&phase)
        || !preparation::valid_factory_role(&role)
        || !preparation::valid_digest(&setup_digest)
        || !preparation::valid_commit(&source_commit)
    {
        return Err(ERR.to_string());
    }
    if setup_log.len() > 66560 || check_log.len() > 66560 {
        return Err("preparation observation exceeds the bounded size".to_string());
    }
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
    if state.ready != (state.phase == preparation::PREPARE_READY) {
        return Err("preparation observation is inconsistent".to_string());
    }
    Ok(state)
}
