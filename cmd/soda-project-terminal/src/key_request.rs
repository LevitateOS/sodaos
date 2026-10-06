use soda_json::JsonValue;

use crate::key_lines::canonical_lines;
use crate::pyemit;

/// Python truthiness over JSON values (only used for the `revision` preview
/// gate: `if data['revision'] or keys`).
pub fn json_truthy(value: &JsonValue) -> bool {
    match value {
        JsonValue::Null => false,
        JsonValue::Bool(b) => *b,
        JsonValue::Number(raw) => {
            if let Some(n) = value.as_integer() {
                return n != 0;
            }
            raw.parse::<f64>().map(|n| n != 0.0).unwrap_or(true)
        }
        JsonValue::Str(s) => !s.is_empty(),
        JsonValue::Array(items) => !items.is_empty(),
        JsonValue::Object(entries) => !entries.is_empty(),
    }
}

/// Duplicate-key rejection at EVERY object level (the `.py`
/// `object_pairs_hook=unique` fires for nested objects too).
pub fn reject_duplicates(value: &JsonValue) -> Result<(), String> {
    match value {
        JsonValue::Object(entries) => {
            for i in 0..entries.len() {
                for other in entries.iter().skip(i + 1) {
                    if other.0 == entries[i].0 {
                        return Err("duplicate field".to_string());
                    }
                }
            }
            for (_, item) in entries {
                reject_duplicates(item)?;
            }
            Ok(())
        }
        JsonValue::Array(items) => {
            for item in items {
                reject_duplicates(item)?;
            }
            Ok(())
        }
        _ => Ok(()),
    }
}

/// Validated key request: exact shape, strict `apply`/`identity` types,
/// canonical desired bytes.
#[derive(Debug, Clone, PartialEq)]
pub struct KeyRequest {
    pub login: String,
    pub identity: i64,
    pub apply: bool,
    pub revision: JsonValue,
    pub keys: Vec<String>,
    pub desired: Vec<u8>,
}

/// Pure request decode: JSON parse plus `key_operation` shape validation.
pub fn decode_key_request(body: &[u8]) -> Result<KeyRequest, String> {
    let text = std::str::from_utf8(body).map_err(|_| "invalid key operation".to_string())?;
    let data = JsonValue::parse(text).map_err(|_| "invalid key operation".to_string())?;
    reject_duplicates(&data)?;
    let entries = match &data {
        JsonValue::Object(entries) => entries,
        _ => return Err("invalid key operation".to_string()),
    };
    let fields = pyemit::shape(entries, &["login", "identity", "apply", "revision", "keys"])
        .ok_or_else(|| "invalid key operation".to_string())?;
    let get = |key: &str| {
        fields
            .iter()
            .find(|(k, _)| k == key)
            .map(|(_, v)| *v)
            .ok_or_else(|| "invalid key operation".to_string())
    };
    // Strict `type(x) is bool` / `type(x) is int`: no float/str coercion.
    let apply = get("apply")?
        .as_bool()
        .ok_or_else(|| "invalid key operation".to_string())?;
    let identity =
        pyemit::as_int(get("identity")?).ok_or_else(|| "invalid key operation".to_string())?;
    let login = get("login")?
        .as_str()
        .ok_or_else(|| "invalid key operation".to_string())?;
    let items = match get("keys")? {
        JsonValue::Array(items) => items,
        _ => return Err("invalid keys".to_string()),
    };
    let mut keys = Vec::with_capacity(items.len());
    for item in items {
        keys.push(
            item.as_str()
                .ok_or_else(|| "invalid keys".to_string())?
                .to_string(),
        );
    }
    let mut desired = keys.join("\n").into_bytes();
    if !keys.is_empty() {
        desired.push(b'\n');
    }
    if !desired.is_ascii() {
        return Err("invalid keys".to_string());
    }
    if canonical_lines(&desired)? != keys {
        return Err("invalid keys".to_string());
    }
    Ok(KeyRequest {
        login: login.to_string(),
        identity,
        apply,
        revision: get("revision")?.clone(),
        keys,
        desired,
    })
}

/// `{"revision","keys"}` result object in `.py` key order.
pub fn state_object(revision: &str, keys: &[String]) -> JsonValue {
    JsonValue::Object(vec![
        ("revision".to_string(), JsonValue::Str(revision.to_string())),
        (
            "keys".to_string(),
            JsonValue::Array(keys.iter().map(|k| JsonValue::Str(k.clone())).collect()),
        ),
    ])
}
