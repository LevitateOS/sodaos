use soda_json::JsonValue;

use crate::buildx::private_destination;
use crate::jsonx::Binder;
use crate::model::{empty_state, Highwater, Trust};
use crate::native::write_json;
use crate::payload::{decode_opt_i64, decode_opt_string};
use crate::{is_digest_ref, Error};

/// `Ledger`: persistent per-repository publication state.
#[derive(Debug, Clone, Default, PartialEq, Eq)]
pub struct Ledger {
    pub format: i64,
    pub repository: String,
    pub phase: String,
    pub digest: String,
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

    pub fn decode(value: &JsonValue) -> Result<Ledger, String> {
        let mut b = Binder::new(value).map_err(|_| "invalid ledger".to_string())?;
        let mut ledger = Ledger {
            format: decode_opt_i64(&mut b, "Format")?,
            repository: decode_opt_string(&mut b, "Repository")?,
            phase: decode_opt_string(&mut b, "Phase")?,
            digest: decode_opt_string(&mut b, "Digest")?,
            state: Highwater::default(),
        };
        if let Some(entries) = b
            .entries("State")
            .map_err(|_| "invalid field State".to_string())?
        {
            ledger.state = Highwater::decode(&JsonValue::Object(entries.to_vec()))?;
        }
        b.finish_name()?;
        Ok(ledger)
    }
}

impl crate::jsonx::Emit for Ledger {
    fn emit(&self, e: &mut crate::jsonx::Emitter) {
        e.begin_object(false);
        e.field(true, "Format");
        e.int(self.format);
        e.field(false, "Repository");
        e.string(&self.repository);
        e.field(false, "Phase");
        e.string(&self.phase);
        e.field(false, "Digest");
        e.string(&self.digest);
        e.field(false, "State");
        self.state.emit(e);
        e.end_object(false);
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
