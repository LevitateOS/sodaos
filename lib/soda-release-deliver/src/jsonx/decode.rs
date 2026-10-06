use soda_json::JsonValue;

use super::{base64_decode, DecodeError, MAX_STRICT_BYTES};
use crate::Error;

fn reject_duplicates(value: &JsonValue, depth: u32) -> Result<(), DecodeError> {
    if depth > 100 {
        return Err(DecodeError);
    }
    match value {
        JsonValue::Object(entries) => {
            for i in 0..entries.len() {
                for j in 0..i {
                    if entries[i].0 == entries[j].0 {
                        return Err(DecodeError);
                    }
                }
                reject_duplicates(&entries[i].1, depth + 1)?;
            }
            Ok(())
        }
        JsonValue::Array(items) => {
            for item in items {
                reject_duplicates(item, depth + 1)?;
            }
            Ok(())
        }
        _ => Ok(()),
    }
}

/// Strict object binder: every field must be consumed exactly once;
/// `finish` rejects unknown fields.
pub struct Binder<'a> {
    entries: &'a [(String, JsonValue)],
    seen: Vec<bool>,
}

impl<'a> Binder<'a> {
    pub fn new(value: &'a JsonValue) -> Result<Binder<'a>, DecodeError> {
        match value {
            JsonValue::Object(entries) => Ok(Binder {
                entries,
                seen: vec![false; entries.len()],
            }),
            _ => Err(DecodeError),
        }
    }

    fn find(&mut self, name: &str) -> Result<Option<&'a JsonValue>, DecodeError> {
        let mut found = None;
        for (i, (key, value)) in self.entries.iter().enumerate() {
            if key == name {
                if found.is_some() {
                    return Err(DecodeError);
                }
                self.seen[i] = true;
                found = Some(value);
            }
        }
        Ok(found)
    }

    fn optional(&mut self, name: &str) -> Result<Option<&'a JsonValue>, DecodeError> {
        match self.find(name)? {
            None | Some(JsonValue::Null) => Ok(None),
            Some(value) => Ok(Some(value)),
        }
    }

    pub fn string(&mut self, name: &str) -> Result<Option<String>, DecodeError> {
        match self.optional(name)? {
            None => Ok(None),
            Some(JsonValue::Str(s)) => Ok(Some(s.clone())),
            Some(_) => Err(DecodeError),
        }
    }

    pub fn boolean(&mut self, name: &str) -> Result<Option<bool>, DecodeError> {
        match self.optional(name)? {
            None => Ok(None),
            Some(JsonValue::Bool(b)) => Ok(Some(*b)),
            Some(_) => Err(DecodeError),
        }
    }

    pub fn integer(&mut self, name: &str) -> Result<Option<i128>, DecodeError> {
        match self.optional(name)? {
            None => Ok(None),
            Some(v) => v.as_integer().map(Some).ok_or(DecodeError),
        }
    }

    pub fn object(&mut self, name: &str) -> Result<Option<Binder<'a>>, DecodeError> {
        match self.optional(name)? {
            None => Ok(None),
            Some(v) => Ok(Some(Binder::new(v)?)),
        }
    }

    pub fn array(&mut self, name: &str) -> Result<Option<&'a [JsonValue]>, DecodeError> {
        match self.optional(name)? {
            None => Ok(None),
            Some(JsonValue::Array(items)) => Ok(Some(items)),
            Some(_) => Err(DecodeError),
        }
    }

    pub fn bytes(&mut self, name: &str) -> Result<Option<Vec<u8>>, DecodeError> {
        match self.optional(name)? {
            None => Ok(None),
            Some(JsonValue::Str(s)) => Ok(Some(base64_decode(s)?)),
            Some(_) => Err(DecodeError),
        }
    }

    /// Raw value of a field, kept verbatim (null included).
    pub fn raw(&mut self, name: &str) -> Result<Option<&'a JsonValue>, DecodeError> {
        self.find(name)
    }

    /// Raw entries of a nested object for strict map decoding.
    pub fn entries(
        &mut self,
        name: &str,
    ) -> Result<Option<&'a [(String, JsonValue)]>, DecodeError> {
        match self.optional(name)? {
            None => Ok(None),
            Some(JsonValue::Object(entries)) => Ok(Some(entries)),
            Some(_) => Err(DecodeError),
        }
    }

    pub fn finish(self) -> Result<(), DecodeError> {
        self.finish_name().map_err(|_| DecodeError)
    }

    /// Unknown-field name in document order, for Go `encoding/json`
    /// error text at the lenient `build.ReadJSON` sites.
    pub fn finish_name(self) -> Result<(), String> {
        for (i, (key, _)) in self.entries.iter().enumerate() {
            if !self.seen[i] {
                return Err(format!("json: unknown field \"{key}\""));
            }
        }
        Ok(())
    }
}

/// Lenient value reader mirroring `encoding/json.Unmarshal` into the small
/// anonymous shapes the Go side decodes non-strictly. Unknown fields are
/// ignored; duplicate keys keep the last value.
pub struct Soft<'a> {
    value: &'a JsonValue,
}

impl<'a> Soft<'a> {
    pub fn new(value: &'a JsonValue) -> Result<Soft<'a>, DecodeError> {
        match value {
            JsonValue::Object(_) => Ok(Soft { value }),
            _ => Err(DecodeError),
        }
    }

    pub fn field(&self, name: &str) -> Option<&'a JsonValue> {
        match self.value.get(name) {
            None | Some(JsonValue::Null) => None,
            Some(value) => Some(value),
        }
    }

    pub fn string(&self, name: &str) -> Result<Option<String>, DecodeError> {
        match self.field(name) {
            None => Ok(None),
            Some(JsonValue::Str(s)) => Ok(Some(s.clone())),
            Some(_) => Err(DecodeError),
        }
    }

    pub fn integer(&self, name: &str) -> Result<Option<i128>, DecodeError> {
        match self.field(name) {
            None => Ok(None),
            Some(v) => v.as_integer().map(Some).ok_or(DecodeError),
        }
    }

    pub fn boolean(&self, name: &str) -> Result<Option<bool>, DecodeError> {
        match self.field(name) {
            None => Ok(None),
            Some(JsonValue::Bool(b)) => Ok(Some(*b)),
            Some(_) => Err(DecodeError),
        }
    }

    pub fn object(&self, name: &str) -> Result<Option<Soft<'a>>, DecodeError> {
        match self.field(name) {
            None => Ok(None),
            Some(v @ JsonValue::Object(_)) => Ok(Some(Soft { value: v })),
            Some(_) => Err(DecodeError),
        }
    }

    pub fn array(&self, name: &str) -> Result<Option<&'a [JsonValue]>, DecodeError> {
        match self.field(name) {
            None => Ok(None),
            Some(JsonValue::Array(items)) => Ok(Some(items)),
            Some(_) => Err(DecodeError),
        }
    }

    pub fn entries(&self) -> Option<&'a [(String, JsonValue)]> {
        match self.value {
            JsonValue::Object(entries) => Some(entries),
            _ => None,
        }
    }
}

pub fn parse_strict(data: &[u8]) -> Result<JsonValue, Error> {
    if data.len() > MAX_STRICT_BYTES {
        return Err(Error::refused());
    }
    let text = std::str::from_utf8(data).map_err(|_| Error::refused())?;
    let value = JsonValue::parse(text).map_err(|_| Error::refused())?;
    if !value.is_object() {
        return Err(Error::refused());
    }
    reject_duplicates(&value, 0).map_err(|_| Error::refused())?;
    Ok(value)
}

pub fn parse_lenient(data: &[u8]) -> Result<JsonValue, DecodeError> {
    let text = std::str::from_utf8(data).map_err(|_| DecodeError)?;
    JsonValue::parse(text).map_err(|_| DecodeError)
}

/// Last-wins deduplication mirroring `encoding/json` object semantics at
/// the `build.ReadJSON` sites (unlike `strictjson`, duplicates are kept).
pub fn dedupe_last_wins(value: JsonValue) -> JsonValue {
    match value {
        JsonValue::Object(entries) => {
            let mut kept: Vec<(String, JsonValue)> = Vec::with_capacity(entries.len());
            for (key, val) in entries {
                let val = dedupe_last_wins(val);
                if let Some(slot) = kept.iter_mut().find(|(k, _)| *k == key) {
                    slot.1 = val;
                } else {
                    kept.push((key, val));
                }
            }
            JsonValue::Object(kept)
        }
        JsonValue::Array(items) => {
            JsonValue::Array(items.into_iter().map(dedupe_last_wins).collect())
        }
        scalar => scalar,
    }
}
