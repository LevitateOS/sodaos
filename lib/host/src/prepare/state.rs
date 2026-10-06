// Helper-inspection decoding: spec tables, launcher evidence, state map.
use crate::json::{self, Kind, Spec};
use crate::preparation::{self, PrepareState, ResolvedTool};

const INSPECTION_HOLD_SPECS: &[Spec] = &[
    Spec {
        name: "active",
        kind: Kind::Bool,
    },
    Spec {
        name: "revision",
        kind: Kind::I64,
    },
];

const INSPECTION_VERIFIED_SPECS: &[Spec] = &[
    Spec {
        name: "uid",
        kind: Kind::Str,
    },
    Spec {
        name: "login",
        kind: Kind::Str,
    },
    Spec {
        name: "groups",
        kind: Kind::Str,
    },
    Spec {
        name: "refusal",
        kind: Kind::Str,
    },
];

const HELPER_INSPECTION_SPECS: &[Spec] = &[
    Spec {
        name: "hold",
        kind: Kind::Object {
            go_type: "struct",
            struct_name: "struct",
            specs: INSPECTION_HOLD_SPECS,
        },
    },
    Spec {
        name: "known",
        kind: Kind::Bool,
    },
    Spec {
        name: "phase",
        kind: Kind::Str,
    },
    Spec {
        name: "role",
        kind: Kind::Str,
    },
    Spec {
        name: "setup_digest",
        kind: Kind::Str,
    },
    Spec {
        name: "source_commit",
        kind: Kind::Str,
    },
    Spec {
        name: "tools",
        kind: Kind::StructList {
            go_type: "[]project.ResolvedTool",
            struct_name: "ResolvedTool",
            specs: preparation::RESOLVED_TOOL_SPECS,
        },
    },
    Spec {
        name: "verified",
        kind: Kind::Object {
            go_type: "struct",
            struct_name: "struct",
            specs: INSPECTION_VERIFIED_SPECS,
        },
    },
    Spec {
        name: "missing",
        kind: Kind::Str,
    },
    Spec {
        name: "stopped",
        kind: Kind::Bool,
    },
    Spec {
        name: "ready",
        kind: Kind::Bool,
    },
    Spec {
        name: "setup_exit",
        kind: Kind::OptInt,
    },
    Spec {
        name: "check_exit",
        kind: Kind::OptInt,
    },
    Spec {
        name: "setup_log",
        kind: Kind::Str,
    },
    Spec {
        name: "check_log",
        kind: Kind::Str,
    },
];

/// Launcher evidence: observed role identity plus any refusal reason.
#[derive(Debug, Clone, Default, PartialEq, Eq)]
pub struct LauncherEvidence {
    pub uid: String,
    pub login: String,
    pub groups: String,
    pub refusal: String,
}

pub(crate) fn quote_bytes_base64(out: &mut String, bytes: &[u8]) {
    out.push_str(&json::quote(&crate::ssh::b64_encode(bytes)));
}

/// `mapPreparationState`: validated observation plus container identity.
/// ID and project are filled in by the caller.
pub fn map_preparation_state(container: &str, raw: &[u8]) -> Result<PrepareState, String> {
    const ERR: &str = "invalid preparation observation";
    let v = json::decode_strict(raw).map_err(|_| ERR.to_string())?;
    let m = json::bind_root(&v, "helperInspection", HELPER_INSPECTION_SPECS, false)
        .map_err(|_| ERR.to_string())?;
    let known = m.take_bool("known");
    let phase = m.take_string("phase");
    let role = m.take_string("role");
    let setup_digest = m.take_string("setup_digest");
    let source_commit = m.take_string("source_commit");
    if !known
        || !preparation::valid_prepare_phase(&phase)
        || !preparation::valid_factory_role(&role)
        || !preparation::valid_digest(&setup_digest)
        || !preparation::valid_commit(&source_commit)
    {
        return Err(ERR.to_string());
    }
    let setup_log = m.take_string("setup_log");
    let check_log = m.take_string("check_log");
    if setup_log.len() > 66560 || check_log.len() > 66560 {
        return Err("preparation observation exceeds the bounded size".to_string());
    }
    let tools = m
        .take_struct_list("tools")
        .iter()
        .map(ResolvedTool::from_map)
        .collect();
    let mut state = PrepareState {
        role,
        phase,
        container: container.to_string(),
        source_commit,
        setup_digest,
        tools,
        missing: m.take_string("missing"),
        setup_exit: m.take_opt_i64("setup_exit"),
        check_exit: m.take_opt_i64("check_exit"),
        ready: m.take_bool("ready"),
        stopped: m.take_bool("stopped"),
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
