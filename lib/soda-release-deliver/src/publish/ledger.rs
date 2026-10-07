use serde::{Deserialize, Serialize};

use crate::buildx::private_destination;
use crate::model::{empty_state, Highwater, Trust};
use crate::native::write_json;
use crate::{is_digest_ref, Error};

/// `Ledger`: persistent per-repository publication state.
#[derive(Debug, Clone, Default, PartialEq, Eq, Serialize, Deserialize)]
#[serde(default, deny_unknown_fields, rename_all = "PascalCase")]
pub struct Ledger {
    #[serde(deserialize_with = "crate::json_serde::null_i64")]
    pub format: i64,
    #[serde(deserialize_with = "crate::json_serde::null_default")]
    pub repository: String,
    #[serde(deserialize_with = "crate::json_serde::null_default")]
    pub phase: String,
    #[serde(deserialize_with = "crate::json_serde::null_default")]
    pub digest: String,
    #[serde(deserialize_with = "crate::json_serde::null_default")]
    pub state: Highwater,
}

impl Ledger {
    pub fn validate(&self, t: &Trust) -> Result<(), Error> {
        if self.format != 1 || self.state.validate().is_err() || self.state.trust_epoch > t.epoch {
            return Err(Error::refused());
        }
        t.role(&self.repository)?;
        if !valid_ledger_phase(&self.phase, &self.digest) {
            return Err(Error::refused());
        }
        Ok(())
    }
}

/// `InitLedger`: initialize a per-repository publication ledger.
pub fn init_ledger(path: &str, t: &Trust, repo: &str) -> Result<(), Error> {
    t.validate()?;
    t.role(repo)?;
    private_destination(path)?;
    let mut state = empty_state();
    state.trust_epoch = t.epoch;
    write_json(
        path,
        &Ledger {
            format: 1,
            repository: repo.to_string(),
            phase: "idle".to_string(),
            state,
            ..Ledger::default()
        },
    )
}

fn valid_ledger_phase(phase: &str, digest: &str) -> bool {
    match phase {
        "idle" => digest.is_empty(),
        "pending" | "complete" => is_digest_ref(digest),
        _ => false,
    }
}
