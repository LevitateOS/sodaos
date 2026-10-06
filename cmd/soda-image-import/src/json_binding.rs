use std::collections::HashMap;

use soda_json::JsonValue;

// ---------- JSON binding over soda-json, Go decoding rules ----------
//
// encoding/json binds sequentially: the last non-null exact-or-fold match
// wins, null is a no-op, missing fields stay zero, and wrong types fail.
// DisallowUnknownFields (payload) additionally rejects unconsumed keys.

pub(super) fn obj_fields(v: &JsonValue) -> Option<&Vec<(String, JsonValue)>> {
    match v {
        JsonValue::Object(fields) => Some(fields),
        _ => None,
    }
}

fn lookup<'a>(fields: &'a [(String, JsonValue)], name: &str) -> Option<&'a JsonValue> {
    let mut found = None;
    for (key, value) in fields {
        if *value == JsonValue::Null {
            continue;
        }
        if key == name || key.eq_ignore_ascii_case(name) {
            found = Some(value);
        }
    }
    found
}

fn as_int(value: &JsonValue, name: &str) -> Result<i64, String> {
    value
        .as_integer()
        .and_then(|n| i64::try_from(n).ok())
        .ok_or_else(|| format!("field {name} must be an integer"))
}

/// Strict object binder with unknown-field rejection.
pub(super) struct Binder<'a> {
    fields: &'a [(String, JsonValue)],
    seen: Vec<bool>,
}

impl<'a> Binder<'a> {
    pub(super) fn new(value: &'a JsonValue) -> Result<Self, String> {
        match value {
            JsonValue::Object(fields) => Ok(Binder {
                fields,
                seen: vec![false; fields.len()],
            }),
            _ => Err("expected JSON object".to_string()),
        }
    }

    fn get(&mut self, name: &str) -> Option<&'a JsonValue> {
        let mut found = None;
        for (i, (key, value)) in self.fields.iter().enumerate() {
            if key == name || key.eq_ignore_ascii_case(name) {
                self.seen[i] = true;
                if *value != JsonValue::Null {
                    found = Some(value);
                }
            }
        }
        found
    }

    pub(super) fn string(&mut self, name: &str) -> Result<String, String> {
        match self.get(name) {
            None => Ok(String::new()),
            Some(JsonValue::Str(s)) => Ok(s.clone()),
            Some(_) => Err(format!("field {name} must be a string")),
        }
    }

    pub(super) fn int(&mut self, name: &str) -> Result<i64, String> {
        match self.get(name) {
            None => Ok(0),
            Some(v) => as_int(v, name),
        }
    }

    pub(super) fn string_list(&mut self, name: &str) -> Result<Vec<String>, String> {
        match self.get(name) {
            None => Ok(Vec::new()),
            Some(JsonValue::Array(items)) => {
                let mut out = Vec::with_capacity(items.len());
                for item in items {
                    match item {
                        JsonValue::Null => out.push(String::new()),
                        JsonValue::Str(s) => out.push(s.clone()),
                        _ => return Err(format!("field {name} must be a string list")),
                    }
                }
                Ok(out)
            }
            Some(_) => Err(format!("field {name} must be a string list")),
        }
    }

    pub(super) fn object(&mut self, name: &str) -> Result<Option<&'a JsonValue>, String> {
        match self.get(name) {
            None => Ok(None),
            Some(v @ JsonValue::Object(_)) => Ok(Some(v)),
            Some(_) => Err(format!("field {name} must be an object")),
        }
    }

    pub(super) fn finish(&self) -> Result<(), String> {
        if let Some(i) = self.seen.iter().position(|seen| !seen) {
            return Err(format!("unknown field {:?}", self.fields[i].0));
        }
        Ok(())
    }
}

// Tolerant getters for OCI metadata: unknown fields ignored, like Unmarshal.

pub(super) fn t_field<'a>(value: &'a JsonValue, name: &str) -> Option<&'a JsonValue> {
    obj_fields(value).and_then(|fields| lookup(fields, name))
}

pub(super) fn t_string(value: &JsonValue, name: &str) -> Result<String, String> {
    match t_field(value, name) {
        None => Ok(String::new()),
        Some(JsonValue::Str(s)) => Ok(s.clone()),
        Some(_) => Err(format!("field {name} must be a string")),
    }
}

pub(super) fn t_int(value: &JsonValue, name: &str) -> Result<i64, String> {
    match t_field(value, name) {
        None => Ok(0),
        Some(v) => as_int(v, name),
    }
}

pub(super) fn t_string_list(value: &JsonValue, name: &str) -> Result<Vec<String>, String> {
    match t_field(value, name) {
        None => Ok(Vec::new()),
        Some(JsonValue::Array(items)) => {
            let mut out = Vec::with_capacity(items.len());
            for item in items {
                match item {
                    JsonValue::Null => out.push(String::new()),
                    JsonValue::Str(s) => out.push(s.clone()),
                    _ => return Err(format!("field {name} must be a string list")),
                }
            }
            Ok(out)
        }
        Some(_) => Err(format!("field {name} must be a string list")),
    }
}

pub(super) fn t_string_map(
    value: &JsonValue,
    name: &str,
) -> Result<HashMap<String, String>, String> {
    match t_field(value, name) {
        None => Ok(HashMap::new()),
        Some(JsonValue::Object(entries)) => {
            let mut out = HashMap::with_capacity(entries.len());
            for (key, item) in entries {
                match item {
                    JsonValue::Null => {
                        out.insert(key.clone(), String::new());
                    }
                    JsonValue::Str(s) => {
                        out.insert(key.clone(), s.clone());
                    }
                    _ => return Err(format!("field {name} must be a string map")),
                }
            }
            Ok(out)
        }
        Some(_) => Err(format!("field {name} must be a string map")),
    }
}

pub(super) fn json_valid(data: &[u8]) -> bool {
    match std::str::from_utf8(data) {
        Ok(text) => JsonValue::parse(text).is_ok(),
        Err(_) => false,
    }
}

pub(super) fn parse_json(data: &[u8]) -> Result<JsonValue, String> {
    let text = std::str::from_utf8(data).map_err(|_| "invalid UTF-8 in JSON".to_string())?;
    JsonValue::parse(text).map_err(|_| "invalid JSON".to_string())
}
