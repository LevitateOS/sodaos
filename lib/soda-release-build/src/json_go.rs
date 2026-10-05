//! Go `encoding/json` semantics over [`soda_json::JsonValue`].
//!
//! Lenient object extraction mirrors `json.Unmarshal` into structs:
//! unknown fields ignored, exact-then-folded name matching, duplicate keys
//! last-wins, `null` leaving the zero value. The strict binder adds
//! `DisallowUnknownFields` for `ReadJSON`. The emitter reproduces
//! `json.MarshalIndent` byte for byte (HTML escaping, `": "` separators,
//! two-space indent, caller-ordered struct fields).

use soda_json::{escape_into, JsonValue};

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

/// A value for `marshal_indent`. Struct fields stay in caller order; maps
/// must be pre-sorted (Go sorts map keys).
pub enum Emit {
    Str(String),
    Int(i64),
    UInt(u32),
    Bool(bool),
    List(Vec<Emit>),
    Object(Vec<(String, Emit)>),
}

impl Emit {
    pub fn sorted_object(mut fields: Vec<(String, Emit)>) -> Emit {
        fields.sort_by(|a, b| a.0.cmp(&b.0));
        Emit::Object(fields)
    }
}

/// Go `json.MarshalIndent(value, "", "  ")`, without the trailing newline
/// the callers append.
pub fn marshal_indent(value: &Emit) -> String {
    let mut out = String::new();
    emit_value(&mut out, value, 0);
    out
}

fn emit_indent(out: &mut String, depth: usize) {
    for _ in 0..depth {
        out.push_str("  ");
    }
}

fn emit_value(out: &mut String, value: &Emit, depth: usize) {
    match value {
        Emit::Str(s) => escape_into(out, s),
        Emit::Int(n) => out.push_str(&n.to_string()),
        Emit::UInt(n) => out.push_str(&n.to_string()),
        Emit::Bool(b) => out.push_str(if *b { "true" } else { "false" }),
        Emit::List(items) => {
            if items.is_empty() {
                out.push_str("[]");
                return;
            }
            out.push_str("[\n");
            for (i, item) in items.iter().enumerate() {
                emit_indent(out, depth + 1);
                emit_value(out, item, depth + 1);
                if i + 1 < items.len() {
                    out.push(',');
                }
                out.push('\n');
            }
            emit_indent(out, depth);
            out.push(']');
        }
        Emit::Object(fields) => {
            if fields.is_empty() {
                out.push_str("{}");
                return;
            }
            out.push_str("{\n");
            for (i, (key, item)) in fields.iter().enumerate() {
                emit_indent(out, depth + 1);
                escape_into(out, key);
                out.push_str(": ");
                emit_value(out, item, depth + 1);
                if i + 1 < fields.len() {
                    out.push(',');
                }
                out.push('\n');
            }
            emit_indent(out, depth);
            out.push('}');
        }
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn oracle_marshal_indent_vectors() {
        // Oracle: Go json.MarshalIndent outputs (no trailing newline).
        let value = Emit::Object(vec![
            (
                "URL".to_string(),
                Emit::Str("https://x.test/a.iso".to_string()),
            ),
            ("n".to_string(), Emit::Int(-3)),
            ("ok".to_string(), Emit::Bool(true)),
            (
                "list".to_string(),
                Emit::List(vec![Emit::Str("a<b".to_string()), Emit::UInt(7)]),
            ),
            ("empty".to_string(), Emit::List(vec![])),
            (
                "nested".to_string(),
                Emit::Object(vec![("k".to_string(), Emit::Str("v&v".to_string()))]),
            ),
        ]);
        assert_eq!(
            marshal_indent(&value),
            "{\n  \"URL\": \"https://x.test/a.iso\",\n  \"n\": -3,\n  \"ok\": true,\n  \"list\": [\n    \"a\\u003cb\",\n    7\n  ],\n  \"empty\": [],\n  \"nested\": {\n    \"k\": \"v\\u0026v\"\n  }\n}"
        );
        assert_eq!(marshal_indent(&Emit::Object(vec![])), "{}");
    }

    #[test]
    fn strict_rejects_unknown_fields_go_style() {
        let value = JsonValue::parse("{\"CompilerImage\":\"x\",\"bogus\":1}").unwrap();
        let mut binder = Strict::bind(&value, "build.ForgejoToolchain").unwrap();
        binder.string("CompilerImage").unwrap();
        assert_eq!(
            binder.finish().unwrap_err(),
            "json: unknown field \"bogus\""
        );
    }

    #[test]
    fn strict_folds_names_and_skips_null() {
        let value = JsonValue::parse("{\"compilerimage\":null,\"APKPackages\":[\"a\"]}").unwrap();
        let mut binder = Strict::bind(&value, "build.ForgejoToolchain").unwrap();
        assert_eq!(binder.string("CompilerImage").unwrap(), "");
        assert_eq!(binder.string_list("APKPackages").unwrap(), vec!["a"]);
        binder.finish().unwrap();
    }

    #[test]
    fn lenient_lookup_folds_and_last_wins() {
        let value =
            JsonValue::parse("{\"mediatype\":\"a\",\"mediaType\":\"b\",\"size\":7}").unwrap();
        let fields = Fields::of(&value).unwrap();
        assert_eq!(fields.string("mediaType").unwrap(), "b");
        assert_eq!(fields.int("size").unwrap(), 7);
        assert_eq!(fields.string("missing").unwrap(), "");
        assert!(fields.int("mediaType").is_err());
    }
}
