use crate::domain;
use crate::sha256;
use crate::terminal::factory::run::{
    valid_digest, valid_factory_role, valid_factory_run_id, valid_harness_family,
    valid_harness_version, valid_preparation_id, MAX_FACTORY_PROMPT,
};

// ---------- prompt-first investigation entry ----------
//
// Terminal-side mirror of `project.FactoryPromptEntry`: the staged prompt
// handoff for admitting exactly one investigation per user intent from an
// existing session. The entry reuses the session's authN/account/project
// selection (project/actor/connection); admission itself belongs strictly to
// FW11 dispatch attempt admission, which remains the sole writer of admitted
// attempts. The launcher rejects every entry this validation refuses, plus
// stale/cancelled intents at launch.

/// Prompt-first entry route: the investigate phase only.
pub const PROMPT_ENTRY_PHASE: &str = "investigate";

/// `project.ValidDecisionID = ^d[0-9a-f]{24}$`.
pub fn valid_intent_id(id: &str) -> bool {
    id.len() == 25 && id.as_bytes()[0] == b'd' && domain::is_hex_lower(&id[1..])
}

/// One prompt-first investigation entry request: the user prompt bytes plus
/// the existing-session bindings FW11 admission checks before opening the
/// investigate run. `assignment` is the canonical digest over the exact
/// prompt bytes; `revision` is the attempt revision (first investigation at
/// 1, Retry advances it).
#[derive(Debug, Clone, Default, PartialEq, Eq)]
pub struct PromptEntry {
    pub intent: String,
    pub revision: u64,
    pub intent_state: String,
    pub project: String,
    pub actor: i64,
    pub connection: String,
    pub role: String,
    pub preparation: String,
    pub harness: String,
    pub harness_vers: String,
    pub phase: String,
    pub assignment: String,
}

/// The admitted investigation bound to one intent.
#[derive(Debug, Clone, Default, PartialEq, Eq)]
pub struct PromptRecord {
    pub intent: String,
    pub revision: u64,
    pub assignment: String,
    pub cancelled: bool,
}

/// Entry shape check: accepted intent, existing-session sponsorship,
/// investigation route (phase/role), preparation directory reference and
/// harness flags, and the prompt/record round-trip (bytes within the
/// investigate bound with a matching digest).
pub fn validate_prompt_entry(entry: &PromptEntry, prompt: &[u8]) -> Result<(), String> {
    if !valid_intent_id(&entry.intent) || entry.revision == 0 {
        return Err("invalid prompt entry intent or revision".to_string());
    }
    if entry.intent_state != "accepted" {
        return Err("prompt entry needs an accepted intent".to_string());
    }
    if !domain::valid_id(&entry.project)
        || entry.actor <= 0
        || entry.connection.is_empty()
        || entry.connection.len() > 128
    {
        return Err("invalid prompt entry session".to_string());
    }
    if entry.phase != PROMPT_ENTRY_PHASE || !valid_factory_role(&entry.role) {
        return Err("invalid prompt entry route".to_string());
    }
    if !valid_preparation_id(&entry.preparation) {
        return Err("invalid prompt entry directory".to_string());
    }
    if !valid_harness_family(&entry.harness) || !valid_harness_version(&entry.harness_vers) {
        return Err("unsupported prompt entry harness".to_string());
    }
    if prompt.is_empty() || prompt.len() > MAX_FACTORY_PROMPT {
        return Err("invalid prompt entry size".to_string());
    }
    if !valid_digest(&entry.assignment)
        || sha256::hex_lower(&sha256::digest(prompt)) != entry.assignment
    {
        return Err("prompt entry bytes do not match their digest".to_string());
    }
    Ok(())
}

/// Exactly one investigation per intent. `None` admits the first revision;
/// the recorded revision with identical bytes replays (`true`, no duplicate
/// investigation); a higher revision supersedes; a lower revision is stale;
/// a cancelled record, or an entry for another intent, refuses. The run id
/// itself is validated where the launch binds it (`valid_factory_run_id`).
pub fn admit_prompt_entry(
    recorded: Option<&PromptRecord>,
    incoming: &PromptEntry,
    prompt: &[u8],
) -> Result<(PromptRecord, bool), String> {
    validate_prompt_entry(incoming, prompt)?;
    let Some(recorded) = recorded else {
        return Ok((
            PromptRecord {
                intent: incoming.intent.clone(),
                revision: incoming.revision,
                assignment: incoming.assignment.clone(),
                cancelled: false,
            },
            false,
        ));
    };
    if recorded.intent != incoming.intent {
        return Err("prompt entry intent mismatch".to_string());
    }
    if recorded.cancelled {
        return Err("prompt entry intent is cancelled".to_string());
    }
    if incoming.revision == recorded.revision {
        if incoming.assignment != recorded.assignment {
            return Err("prompt entry revision conflict".to_string());
        }
        return Ok((recorded.clone(), true));
    }
    if incoming.revision > recorded.revision {
        return Ok((
            PromptRecord {
                intent: incoming.intent.clone(),
                revision: incoming.revision,
                assignment: incoming.assignment.clone(),
                cancelled: false,
            },
            false,
        ));
    }
    Err("stale prompt entry revision".to_string())
}

/// `project.ValidFactoryRunID` for the launch binding half of the entry.
pub fn valid_prompt_entry_run(id: &str) -> bool {
    valid_factory_run_id(id)
}
