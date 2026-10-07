//! `model.go`: trust, candidate, release, channel, high-water mark, permit.

use serde::{Deserialize, Serialize};

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

#[derive(Debug, Clone, Default, PartialEq, Eq, Serialize, Deserialize)]
#[serde(default, deny_unknown_fields, rename_all = "PascalCase")]
pub struct Permit {
    #[serde(deserialize_with = "crate::json_serde::null_i64")]
    pub format: i64,
    #[serde(deserialize_with = "crate::json_serde::null_default")]
    pub repository: String,
    #[serde(deserialize_with = "crate::json_serde::null_default")]
    pub digest: String,
    #[serde(deserialize_with = "crate::json_serde::null_default")]
    pub previous: String,
    #[serde(deserialize_with = "crate::json_serde::null_i64")]
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
}

#[cfg(test)]
mod tests;
