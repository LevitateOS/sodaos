//! Go `encoding/json` struct semantics over [`soda_json::JsonValue`]:
//! strict binding with unknown-field rejection and Go's exact/folded name
//! matching, tolerant decoders, and a compact serializer with sorted keys.

use soda_json::{escape_into, JsonValue};

/// Strict object binder: Go struct decoding with `DisallowUnknownFields`.
/// Names match exactly first, then by unique ASCII case folding; `null`
/// leaves the field missing, as in Go.
pub struct Binder<'a> {
    entries: &'a [(String, JsonValue)],
    seen: Vec<bool>,
}

impl<'a> Binder<'a> {
    pub fn new(value: &'a JsonValue) -> Result<Binder<'a>, ()> {
        match value {
            JsonValue::Object(entries) => Ok(Binder {
                entries,
                seen: vec![false; entries.len()],
            }),
            _ => Err(()),
        }
    }

    fn lookup(&mut self, name: &str) -> Option<&'a JsonValue> {
        // Go decodes keys in order with the last match winning, whether it
        // matches exactly or by folding.
        let mut found: Option<usize> = None;
        for (i, (key, _)) in self.entries.iter().enumerate() {
            if key == name || (key.len() == name.len() && key.eq_ignore_ascii_case(name)) {
                self.seen[i] = true;
                found = Some(i);
            }
        }
        found.map(|i| &self.entries[i].1)
    }

    /// Missing-or-null field; wrong types are errors.
    fn field(&mut self, name: &str) -> Result<Option<&'a JsonValue>, ()> {
        match self.lookup(name) {
            None => Ok(None),
            Some(JsonValue::Null) => Ok(None),
            Some(value) => Ok(Some(value)),
        }
    }

    pub fn string(&mut self, name: &str) -> Result<Option<String>, ()> {
        match self.field(name)? {
            None => Ok(None),
            Some(JsonValue::Str(s)) => Ok(Some(s.clone())),
            Some(_) => Err(()),
        }
    }

    pub fn integer(&mut self, name: &str) -> Result<Option<i64>, ()> {
        match self.field(name)? {
            None => Ok(None),
            Some(value) => match value.as_integer().and_then(|n| i64::try_from(n).ok()) {
                Some(n) => Ok(Some(n)),
                None => Err(()),
            },
        }
    }

    #[cfg(test)]
    pub fn boolean(&mut self, name: &str) -> Result<Option<bool>, ()> {
        match self.field(name)? {
            None => Ok(None),
            Some(JsonValue::Bool(b)) => Ok(Some(*b)),
            Some(_) => Err(()),
        }
    }

    #[cfg(test)]
    pub fn object(&mut self, name: &str) -> Result<Option<Binder<'a>>, ()> {
        match self.field(name)? {
            None => Ok(None),
            Some(value) => Ok(Some(Binder::new(value)?)),
        }
    }

    pub fn array(&mut self, name: &str) -> Result<Option<&'a [JsonValue]>, ()> {
        match self.field(name)? {
            None => Ok(None),
            Some(JsonValue::Array(items)) => Ok(Some(items)),
            Some(_) => Err(()),
        }
    }

    /// Raw object entries with presence: null values stay visible to the
    /// caller, like Go decoding a null map element to the zero value.
    pub fn raw_object(&mut self, name: &str) -> Result<Option<&'a [(String, JsonValue)]>, ()> {
        match self.field(name)? {
            None => Ok(None),
            Some(JsonValue::Object(entries)) => Ok(Some(entries)),
            Some(_) => Err(()),
        }
    }

    pub fn finish(self) -> Result<(), ()> {
        if self.seen.iter().all(|seen| *seen) {
            Ok(())
        } else {
            Err(())
        }
    }
}

/// Tolerant object view: unknown fields ignored, last key wins, `null` is
/// missing; wrong types are errors, as in Go struct decoding.
pub struct Soft<'a> {
    entries: &'a [(String, JsonValue)],
}

impl<'a> Soft<'a> {
    pub fn new(value: &'a JsonValue) -> Result<Soft<'a>, ()> {
        match value {
            JsonValue::Object(entries) => Ok(Soft { entries }),
            _ => Err(()),
        }
    }

    pub fn field(&self, name: &str) -> Result<Option<&'a JsonValue>, ()> {
        match self.entries.iter().rev().find(|(k, _)| k == name) {
            None => Ok(None),
            Some((_, JsonValue::Null)) => Ok(None),
            Some((_, value)) => Ok(Some(value)),
        }
    }

    pub fn string(&self, name: &str) -> Result<Option<String>, ()> {
        match self.field(name)? {
            None => Ok(None),
            Some(JsonValue::Str(s)) => Ok(Some(s.clone())),
            Some(_) => Err(()),
        }
    }

    #[cfg(test)]
    pub fn integer(&self, name: &str) -> Result<Option<i64>, ()> {
        match self.field(name)? {
            None => Ok(None),
            Some(value) => match value.as_integer().and_then(|n| i64::try_from(n).ok()) {
                Some(n) => Ok(Some(n)),
                None => Err(()),
            },
        }
    }

    pub fn unsigned(&self, name: &str) -> Result<Option<u64>, ()> {
        match self.field(name)? {
            None => Ok(None),
            Some(value) => match value.as_integer().and_then(|n| u64::try_from(n).ok()) {
                Some(n) => Ok(Some(n)),
                None => Err(()),
            },
        }
    }

    pub fn boolean(&self, name: &str) -> Result<Option<bool>, ()> {
        match self.field(name)? {
            None => Ok(None),
            Some(JsonValue::Bool(b)) => Ok(Some(*b)),
            Some(_) => Err(()),
        }
    }

    #[cfg(test)]
    pub fn object(&self, name: &str) -> Result<Option<Soft<'a>>, ()> {
        match self.field(name)? {
            None => Ok(None),
            Some(value) => Ok(Some(Soft::new(value)?)),
        }
    }

    pub fn array(&self, name: &str) -> Result<Option<&'a [JsonValue]>, ()> {
        match self.field(name)? {
            None => Ok(None),
            Some(JsonValue::Array(items)) => Ok(Some(items)),
            Some(_) => Err(()),
        }
    }
}

/// Parse one top-level JSON value; trailing data is refused by the parser.
/// Invalid UTF-8 decodes lossy, like Go replacing it with U+FFFD.
pub fn parse(data: &[u8]) -> Result<JsonValue, ()> {
    let text = String::from_utf8_lossy(data);
    JsonValue::parse(&text).map_err(|_| ())
}

/// Compact Go-compatible serialization: sorted object keys, Go string
/// escaping, raw number literals.
pub fn serialize(value: &JsonValue) -> String {
    let mut out = String::new();
    write_value(&mut out, value);
    out
}

fn write_value(out: &mut String, value: &JsonValue) {
    match value {
        JsonValue::Null => out.push_str("null"),
        JsonValue::Bool(true) => out.push_str("true"),
        JsonValue::Bool(false) => out.push_str("false"),
        JsonValue::Number(raw) => out.push_str(raw),
        JsonValue::Str(s) => {
            escape_into(out, s);
        }
        JsonValue::Array(items) => {
            out.push('[');
            for (i, item) in items.iter().enumerate() {
                if i > 0 {
                    out.push(',');
                }
                write_value(out, item);
            }
            out.push(']');
        }
        JsonValue::Object(entries) => {
            // Go marshals maps with sorted keys; duplicates collapse with
            // the last value winning.
            let mut keys: Vec<&str> = entries.iter().map(|(k, _)| k.as_str()).collect();
            keys.sort_unstable();
            keys.dedup();
            out.push('{');
            for (i, key) in keys.iter().enumerate() {
                if i > 0 {
                    out.push(',');
                }
                escape_into(out, key);
                out.push(':');
                let value = entries
                    .iter()
                    .rev()
                    .find(|(k, _)| k == key)
                    .unwrap()
                    .1
                    .clone();
                write_value(out, &value);
            }
            out.push('}');
        }
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    fn parse_ok(text: &str) -> JsonValue {
        parse(text.as_bytes()).unwrap()
    }

    #[test]
    fn strict_binder_rules() {
        let value = parse_ok(r#"{"Format":3,"ID":"x","nested":{"A":1},"List":[1],"B":true}"#);
        let mut binder = Binder::new(&value).unwrap();
        assert_eq!(binder.integer("Format").unwrap(), Some(3));
        assert_eq!(binder.string("ID").unwrap(), Some("x".to_string()));
        assert_eq!(binder.boolean("B").unwrap(), Some(true));
        assert!(binder.array("List").unwrap().is_some());
        assert!(binder.object("nested").unwrap().is_some());
        assert!(binder.finish().is_ok());

        // Unknown fields are refused.
        let value = parse_ok(r#"{"Format":3,"Bogus":1}"#);
        let mut binder = Binder::new(&value).unwrap();
        assert_eq!(binder.integer("Format").unwrap(), Some(3));
        assert!(binder.finish().is_err());

        // Folded names match when unique; exact wins per key order.
        let value = parse_ok(r#"{"format":4}"#);
        let mut binder = Binder::new(&value).unwrap();
        assert_eq!(binder.integer("Format").unwrap(), Some(4));
        assert!(binder.finish().is_ok());
        let value = parse_ok(r#"{"ID":"a","id":"b"}"#);
        let mut binder = Binder::new(&value).unwrap();
        assert_eq!(binder.string("ID").unwrap(), Some("b".to_string()));
        assert!(binder.finish().is_ok());

        // Null leaves the field missing; wrong types error.
        let value = parse_ok(r#"{"Format":null,"ID":3}"#);
        let mut binder = Binder::new(&value).unwrap();
        assert_eq!(binder.integer("Format").unwrap(), None);
        assert!(binder.string("ID").is_err());

        // Integers must be integral and fit.
        for (text, want) in [("3", Some(3i64)), ("-0", Some(0))] {
            let value = parse_ok(&format!(r#"{{"N":{text}}}"#));
            let mut binder = Binder::new(&value).unwrap();
            assert_eq!(binder.integer("N").unwrap(), want, "N={text}");
        }
        for text in ["3.0", "3e1", "99999999999999999999999", "\"3\""] {
            let value = parse_ok(&format!(r#"{{"N":{text}}}"#));
            let mut binder = Binder::new(&value).unwrap();
            assert!(binder.integer("N").is_err(), "N={text}");
        }
        // Non-objects and trailing data fail.
        assert!(Binder::new(&parse_ok("[1]")).is_err());
        assert!(parse(br#"{"A":1} trailing"#).is_err());
    }

    #[test]
    fn tolerant_soft_rules() {
        let value = parse_ok(r#"{"Name":"sda","Size":64000,"ReadOnly":null,"Extra":[1],"M":null}"#);
        let soft = Soft::new(&value).unwrap();
        assert_eq!(soft.string("Name").unwrap(), Some("sda".to_string()));
        assert_eq!(soft.unsigned("Size").unwrap(), Some(64000));
        assert_eq!(soft.boolean("ReadOnly").unwrap(), None);
        assert_eq!(soft.array("Missing").unwrap(), None);
        assert!(soft.string("Size").is_err());
        let value = parse_ok(r#"{"Size":-1}"#);
        assert!(Soft::new(&value).unwrap().unsigned("Size").is_err());
    }

    #[test]
    fn serializer_is_go_compatible() {
        let value = parse_ok(r#"{"b":1,"a":[true,null,"x"],"c":{"z":2,"y":1}}"#);
        assert_eq!(
            serialize(&value),
            r#"{"a":[true,null,"x"],"b":1,"c":{"y":1,"z":2}}"#
        );
        let value = parse_ok(r#"{"s":"a<b>&\"q\""}"#);
        assert_eq!(serialize(&value), r#"{"s":"a\u003cb\u003e\u0026\"q\""}"#);
        let value = parse_ok(r#"{"n":3.50,"big":99999999999999999999999}"#);
        assert_eq!(
            serialize(&value),
            r#"{"big":99999999999999999999999,"n":3.50}"#
        );
        // Duplicate keys collapse, last wins.
        let value = parse_ok(r#"{"a":1,"a":2}"#);
        assert_eq!(serialize(&value), r#"{"a":2}"#);
    }
}
