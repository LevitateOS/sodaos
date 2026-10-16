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

#[cfg(test)]
mod tests {
    use super::*;

    fn fixture(intent: &str, revision: u64, state: &str) -> (PromptEntry, Vec<u8>) {
        let prompt = b"investigate the checkout and write a plan".to_vec();
        (
            PromptEntry {
                intent: intent.to_string(),
                revision,
                intent_state: state.to_string(),
                project: format!("p{}", "b".repeat(24)),
                actor: 7,
                connection: "conn-1".to_string(),
                role: "soda-coder".to_string(),
                preparation: format!("f{}", "a".repeat(24)),
                harness: "muse".to_string(),
                harness_vers: "1.4.9".to_string(),
                phase: PROMPT_ENTRY_PHASE.to_string(),
                assignment: sha256::hex_lower(&sha256::digest(&prompt)),
            },
            prompt,
        )
    }

    fn intent(digit: char) -> String {
        format!("d{}", digit.to_string().repeat(24))
    }

    #[test]
    fn admits_first_investigation() {
        let (entry, prompt) = fixture(&intent('1'), 1, "accepted");
        assert!(validate_prompt_entry(&entry, &prompt).is_ok());
        let (record, replay) = admit_prompt_entry(None, &entry, &prompt).unwrap();
        assert!(!replay);
        assert_eq!(record.intent, entry.intent);
        assert_eq!(record.revision, 1);
        assert_eq!(record.assignment, entry.assignment);
    }

    #[test]
    fn refuses_misuse() {
        let (base, prompt) = fixture(&intent('2'), 1, "accepted");
        let mut cases: Vec<(&str, PromptEntry, Vec<u8>)> = Vec::new();
        let mut bad = base.clone();
        bad.intent = "nope".to_string();
        cases.push(("bad intent", bad, prompt.clone()));
        let mut bad = base.clone();
        bad.revision = 0;
        cases.push(("zero revision", bad, prompt.clone()));
        for state in ["draft", "declined", "rejected", "abandoned"] {
            let mut bad = base.clone();
            bad.intent_state = state.to_string();
            cases.push(("closed intent", bad, prompt.clone()));
        }
        let mut bad = base.clone();
        bad.project = String::new();
        cases.push(("empty project", bad, prompt.clone()));
        let mut bad = base.clone();
        bad.actor = 0;
        cases.push(("zero actor", bad, prompt.clone()));
        let mut bad = base.clone();
        bad.phase = "implement".to_string();
        cases.push(("wrong phase", bad, prompt.clone()));
        let mut bad = base.clone();
        bad.role = "soda-admin".to_string();
        cases.push(("bad role", bad, prompt.clone()));
        let mut bad = base.clone();
        bad.preparation = "checkout".to_string();
        cases.push(("bad prep", bad, prompt.clone()));
        let mut bad = base.clone();
        bad.harness = "shell".to_string();
        cases.push(("bad harness", bad, prompt.clone()));
        cases.push(("empty prompt", base.clone(), Vec::new()));
        cases.push(("oversize", base.clone(), vec![b'x'; MAX_FACTORY_PROMPT + 1]));
        let mut bad = base.clone();
        bad.assignment = "0".repeat(64);
        cases.push(("digest mismatch", bad, prompt.clone()));
        for (name, entry, bytes) in &cases {
            assert!(
                validate_prompt_entry(entry, bytes).is_err(),
                "{name}: misuse admitted"
            );
            assert!(
                admit_prompt_entry(None, entry, bytes).is_err(),
                "{name}: admission admitted"
            );
        }
    }

    #[test]
    fn exactly_once_per_intent() {
        let (first, prompt) = fixture(&intent('3'), 1, "accepted");
        let (record, _) = admit_prompt_entry(None, &first, &prompt).unwrap();
        let (dup, replay) = admit_prompt_entry(Some(&record), &first, &prompt).unwrap();
        assert!(replay);
        assert_eq!(dup, record);
        let changed_prompt = b"a different prompt".to_vec();
        let mut changed = first.clone();
        changed.assignment = sha256::hex_lower(&sha256::digest(&changed_prompt));
        assert!(admit_prompt_entry(Some(&record), &changed, &changed_prompt).is_err());
        let retry_prompt = b"retry with revised prompt".to_vec();
        let mut next = first.clone();
        next.revision = 2;
        next.assignment = sha256::hex_lower(&sha256::digest(&retry_prompt));
        let (second, replay) = admit_prompt_entry(Some(&record), &next, &retry_prompt).unwrap();
        assert!(!replay);
        assert_eq!(second.revision, 2);
        assert!(admit_prompt_entry(Some(&second), &first, &prompt).is_err());
        let (other, other_prompt) = fixture(&intent('4'), 1, "accepted");
        assert!(admit_prompt_entry(Some(&record), &other, &other_prompt).is_err());
        let mut cancelled = record.clone();
        cancelled.cancelled = true;
        assert!(admit_prompt_entry(Some(&cancelled), &next, &retry_prompt).is_err());
        assert!(valid_prompt_entry_run(&"a".repeat(32)));
        assert!(!valid_prompt_entry_run("short"));
    }
}
