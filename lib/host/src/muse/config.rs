use std::collections::HashMap;

use crate::json::{self, Value};
use crate::terminal;

/// Tolerant `map[string][]byte` decode for nested config views
/// (`encoding/json` into `map[string][]byte`: base64 strings or numeric
/// arrays, like [`Kind::Bytes`](crate::json::Kind::Bytes)).
pub fn decode_config_view(body: &[u8]) -> Result<HashMap<String, Vec<u8>>, String> {
    let v = json::decode_tolerant(body).map_err(|_| terminal::err_denied())?;
    let Some(fields) = v.as_object() else {
        return Err(terminal::err_denied());
    };
    let mut out = HashMap::with_capacity(fields.len());
    for (key, value) in fields {
        let bytes = match value {
            Value::Null => Vec::new(),
            Value::Str(s) => {
                crate::ssh::b64_decode_go(s.as_bytes()).map_err(|_| terminal::err_denied())?
            }
            Value::Array(items) => {
                let mut bytes = Vec::with_capacity(items.len());
                for item in items {
                    match item {
                        Value::Number(lit) => {
                            let n: i64 = lit.parse().map_err(|_| terminal::err_denied())?;
                            if !(0..=255).contains(&n) {
                                return Err(terminal::err_denied());
                            }
                            bytes.push(n as u8);
                        }
                        _ => return Err(terminal::err_denied()),
                    }
                }
                bytes
            }
            _ => return Err(terminal::err_denied()),
        };
        out.insert(key.clone(), bytes);
    }
    Ok(out)
}
