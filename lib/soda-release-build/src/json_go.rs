//! Go `encoding/json` semantics over [`soda_json::JsonValue`].
//!
//! Lenient object extraction mirrors `json.Unmarshal` into structs:
//! unknown fields ignored, exact-then-folded name matching, duplicate keys
//! last-wins, `null` leaving the zero value. The strict binder adds
//! `DisallowUnknownFields` for `ReadJSON`.

use soda_json::JsonValue;

#[cfg(test)]
mod tests;

/// Lenient field-extraction failure; callers map it to their message.
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub struct FieldError;

/// Lenient struct view: Go `json.Unmarshal` object rules.
pub struct Fields<'a> {
    entries: &'a [(String, JsonValue)],
}

impl<'a> Fields<'a> {
    pub fn of(value: &'a JsonValue) -> Option<Fields<'a>> {
        match value {
            JsonValue::Object(entries) => Some(Fields { entries }),
            _ => None,
        }
    }

    /// Last exact-or-folded match wins, as in Go.
    pub fn lookup(&self, name: &str) -> Option<&'a JsonValue> {
        let mut found = None;
        for (key, value) in self.entries.iter() {
            if key == name || (key.len() == name.len() && key.eq_ignore_ascii_case(name)) {
                found = Some(value);
            }
        }
        found
    }

    fn present(&self, name: &str) -> Option<&'a JsonValue> {
        match self.lookup(name) {
            None | Some(JsonValue::Null) => None,
            value => value,
        }
    }

    /// Missing-or-null yields `""`; wrong types fail.
    pub fn string(&self, name: &str) -> Result<String, FieldError> {
        match self.present(name) {
            None => Ok(String::new()),
            Some(JsonValue::Str(s)) => Ok(s.clone()),
            Some(_) => Err(FieldError),
        }
    }

    /// Missing-or-null yields 0; fractions, exponents, and overflows fail.
    pub fn int(&self, name: &str) -> Result<i64, FieldError> {
        match self.present(name) {
            None => Ok(0),
            Some(value) => value
                .as_integer()
                .and_then(|n| i64::try_from(n).ok())
                .ok_or(FieldError),
        }
    }

    /// Missing-or-null yields empty; elements must be strings.
    pub fn string_list(&self, name: &str) -> Result<Vec<String>, FieldError> {
        match self.present(name) {
            None => Ok(Vec::new()),
            Some(JsonValue::Array(items)) => items
                .iter()
                .map(|item| match item {
                    JsonValue::Str(s) => Ok(s.clone()),
                    _ => Err(FieldError),
                })
                .collect(),
            Some(_) => Err(FieldError),
        }
    }

    /// Missing-or-null yields empty; values must be strings.
    pub fn string_map(&self, name: &str) -> Result<Vec<(String, String)>, FieldError> {
        match self.present(name) {
            None => Ok(Vec::new()),
            Some(JsonValue::Object(entries)) => {
                let mut out = Vec::with_capacity(entries.len());
                for (key, value) in entries.iter() {
                    match value {
                        JsonValue::Str(s) => out.push((key.clone(), s.clone())),
                        _ => return Err(FieldError),
                    }
                }
                Ok(out)
            }
            Some(_) => Err(FieldError),
        }
    }

    /// Missing-or-null yields `None`; must be an object otherwise.
    pub fn object(&self, name: &str) -> Result<Option<Fields<'a>>, FieldError> {
        match self.present(name) {
            None => Ok(None),
            Some(value) => Fields::of(value).ok_or(FieldError).map(Some),
        }
    }

    /// Missing-or-null yields empty; elements must be objects.
    pub fn object_list(&self, name: &str) -> Result<Vec<Fields<'a>>, FieldError> {
        match self.present(name) {
            None => Ok(Vec::new()),
            Some(JsonValue::Array(items)) => items
                .iter()
                .map(Fields::of)
                .collect::<Option<Vec<_>>>()
                .ok_or(FieldError),
            Some(_) => Err(FieldError),
        }
    }
}

/// Strict struct binder: `json.Decoder` with `DisallowUnknownFields`.
/// Unknown-field diagnostics match Go exactly; type diagnostics follow Go's
/// `cannot unmarshal <kind> into Go value of type <go-type>` shape.
pub struct Strict<'a> {
    entries: &'a [(String, JsonValue)],
    seen: Vec<bool>,
    target: &'static str,
}

fn json_kind(value: &JsonValue) -> &'static str {
    match value {
        JsonValue::Null => "null",
        JsonValue::Bool(_) => "bool",
        JsonValue::Number(_) => "number",
        JsonValue::Str(_) => "string",
        JsonValue::Array(_) => "array",
        JsonValue::Object(_) => "object",
    }
}

impl<'a> Strict<'a> {
    pub fn bind(value: &'a JsonValue, target: &'static str) -> Result<Strict<'a>, String> {
        match value {
            JsonValue::Null => Ok(Strict {
                entries: &[],
                seen: Vec::new(),
                target,
            }),
            JsonValue::Object(entries) => Ok(Strict {
                entries,
                seen: vec![false; entries.len()],
                target,
            }),
            other => Err(format!(
                "json: cannot unmarshal {} into Go value of type {target}",
                json_kind(other)
            )),
        }
    }

    fn lookup(&mut self, name: &str) -> Option<&'a JsonValue> {
        let mut found: Option<usize> = None;
        for (i, (key, _)) in self.entries.iter().enumerate() {
            if key == name || (key.len() == name.len() && key.eq_ignore_ascii_case(name)) {
                self.seen[i] = true;
                found = Some(i);
            }
        }
        found.map(|i| &self.entries[i].1)
    }

    fn field(&mut self, name: &str) -> Option<&'a JsonValue> {
        match self.lookup(name) {
            None | Some(JsonValue::Null) => None,
            value => value,
        }
    }

    fn mismatch(&self, value: &JsonValue, go_type: &str) -> String {
        format!(
            "json: cannot unmarshal {} into Go value of type {go_type}",
            json_kind(value)
        )
    }

    pub fn string(&mut self, name: &str) -> Result<String, String> {
        match self.field(name) {
            None => Ok(String::new()),
            Some(JsonValue::Str(s)) => Ok(s.clone()),
            Some(other) => Err(self.mismatch(other, "string")),
        }
    }

    pub fn boolean(&mut self, name: &str) -> Result<bool, String> {
        match self.field(name) {
            None => Ok(false),
            Some(JsonValue::Bool(b)) => Ok(*b),
            Some(other) => Err(self.mismatch(other, "bool")),
        }
    }

    pub fn uint32(&mut self, name: &str) -> Result<u32, String> {
        match self.field(name) {
            None => Ok(0),
            Some(value @ JsonValue::Number(_)) => value
                .as_integer()
                .and_then(|n| u32::try_from(n).ok())
                .ok_or_else(|| self.mismatch(value, "uint32")),
            Some(other) => Err(self.mismatch(other, "uint32")),
        }
    }

    pub fn string_list(&mut self, name: &str) -> Result<Vec<String>, String> {
        match self.field(name) {
            None => Ok(Vec::new()),
            Some(JsonValue::Array(items)) => {
                let mut out = Vec::with_capacity(items.len());
                for item in items {
                    match item {
                        JsonValue::Str(s) => out.push(s.clone()),
                        other => return Err(self.mismatch(other, "string")),
                    }
                }
                Ok(out)
            }
            Some(other) => Err(self.mismatch(other, "[]string")),
        }
    }

    pub fn string_map(&mut self, name: &str) -> Result<Vec<(String, String)>, String> {
        match self.field(name) {
            None => Ok(Vec::new()),
            Some(JsonValue::Object(entries)) => {
                let mut out = Vec::with_capacity(entries.len());
                for (key, value) in entries.iter() {
                    match value {
                        JsonValue::Str(s) => out.push((key.clone(), s.clone())),
                        _ => return Err(self.mismatch(value, "string")),
                    }
                }
                Ok(out)
            }
            Some(other) => Err(self.mismatch(other, "map[string]string")),
        }
    }

    pub fn object(
        &mut self,
        name: &str,
        target: &'static str,
    ) -> Result<Option<Strict<'a>>, String> {
        match self.field(name) {
            None => Ok(None),
            Some(value) => Strict::bind(value, target).map(Some),
        }
    }

    /// Nested objects decode through a closure; `null`/missing stays `None`.
    pub fn nested<T>(
        &mut self,
        name: &str,
        target: &'static str,
        decode: impl FnOnce(&mut Strict<'a>) -> Result<T, String>,
    ) -> Result<T, String>
    where
        T: Default,
    {
        match self.object(name, target)? {
            None => Ok(T::default()),
            Some(mut inner) => {
                let value = decode(&mut inner)?;
                inner.finish()?;
                Ok(value)
            }
        }
    }

    /// Maps of nested objects, e.g. `map[string]CoreOSImage`.
    pub fn object_map<T>(
        &mut self,
        name: &str,
        target: &'static str,
        decode: impl Fn(&mut Strict<'a>) -> Result<T, String>,
    ) -> Result<Vec<(String, T)>, String> {
        let entries = match self.field(name) {
            None => return Ok(Vec::new()),
            Some(JsonValue::Object(entries)) => entries,
            Some(other) => return Err(self.mismatch(other, target)),
        };
        let mut out = Vec::with_capacity(entries.len());
        for (key, value) in entries.iter() {
            let mut inner = Strict::bind(value, target)?;
            let decoded = decode(&mut inner)?;
            inner.finish()?;
            out.push((key.clone(), decoded));
        }
        Ok(out)
    }

    /// Unknown-field rejection after all known fields are bound.
    pub fn finish(&self) -> Result<(), String> {
        for (i, (key, _)) in self.entries.iter().enumerate() {
            if !self.seen[i] {
                return Err(format!("json: unknown field {key:?}"));
            }
        }
        let _ = self.target;
        Ok(())
    }
}
