//! `model.go`: trust, candidate, release, channel, high-water mark, permit.

use soda_json::JsonValue;

use crate::jsonx::{Binder, Emit, Emitter};
use crate::payload::{decode_opt_i64, decode_opt_string};
use crate::{is_digest_ref, Error};

mod trust;
pub use trust::Trust;

mod candidate;
pub(crate) use candidate::path_clean;
pub use candidate::{valid_candidate_content, Candidate};

mod release;
pub use release::{admit_release, MediaBinding, MediaFile, Release};
pub(crate) use release::{valid_media_binding, valid_media_file};

mod channel;
pub use channel::{admit_channel, empty_state, Channel, Highwater, Seen};

// ---------------------------------------------------------------------------
// Permit
// ---------------------------------------------------------------------------

#[derive(Debug, Clone, Default, PartialEq, Eq)]
pub struct Permit {
    pub format: i64,
    pub repository: String,
    pub digest: String,
    pub previous: String,
    pub expires: i64,
}

impl Permit {
    pub fn validate(&self, t: &Trust, now_unix: i64) -> Result<(), Error> {
        if t.role(&self.repository).is_err()
            || self.format != 1
            || !is_digest_ref(&self.digest)
            || self.expires <= now_unix
            || self.expires - now_unix > 86400
        {
            return Err(Error::refused());
        }
        Ok(())
    }

    pub fn decode(value: &JsonValue) -> Result<Permit, String> {
        let mut b = Binder::new(value).map_err(|_| "invalid permit".to_string())?;
        let permit = Permit {
            format: decode_opt_i64(&mut b, "Format")?,
            repository: decode_opt_string(&mut b, "Repository")?,
            digest: decode_opt_string(&mut b, "Digest")?,
            previous: decode_opt_string(&mut b, "Previous")?,
            expires: decode_opt_i64(&mut b, "Expires")?,
        };
        b.finish_name()?;
        Ok(permit)
    }
}

impl Emit for Permit {
    fn emit(&self, e: &mut Emitter) {
        e.begin_object(false);
        e.field(true, "Format");
        e.int(self.format);
        e.field(false, "Repository");
        e.string(&self.repository);
        e.field(false, "Digest");
        e.string(&self.digest);
        e.field(false, "Previous");
        e.string(&self.previous);
        e.field(false, "Expires");
        e.int(self.expires);
        e.end_object(false);
    }
}

#[cfg(test)]
mod tests;
