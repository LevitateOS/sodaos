use std::collections::HashMap;

use crate::json::{self, BytesField};
use crate::terminal;

/// Tolerant `map[string][]byte` decode for nested config views.
pub fn decode_config_view(body: &[u8]) -> Result<HashMap<String, Vec<u8>>, String> {
    let fields: HashMap<String, BytesField> =
        json::decode_tolerant_as(body).map_err(|_| terminal::err_denied())?;
    Ok(fields
        .into_iter()
        .map(|(key, value)| (key, value.0))
        .collect())
}
