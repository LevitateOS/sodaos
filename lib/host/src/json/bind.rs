use std::collections::HashMap;

use super::number::{parse_go_int64, parse_go_uint32};
use super::specs::{Bound, BoundMap, Kind, Spec};
use super::{err, go_quote, Error, Value};

fn value_word(v: &Value) -> &'static str {
    match v {
        Value::Null => "null",
        Value::Bool(_) => "bool",
        Value::Number(_) => "number",
        Value::Str(_) => "string",
        Value::Array(_) => "array",
        Value::Object(_) => "object",
    }
}

/// `UnmarshalTypeError` with struct-field context:
/// `json: cannot unmarshal {value} into Go struct field {S}.{path}.{field}
/// of type {type}`. `number_word` carries the literal for int targets.
fn type_error(
    struct_name: &str,
    path: &[String],
    field: &str,
    number_word: Option<&str>,
    v: &Value,
    go_type: &str,
) -> Error {
    let value = match (v, number_word) {
        (Value::Number(_), Some(lit)) => format!("number {lit}"),
        _ => value_word(v).to_string(),
    };
    let mut full = String::from(struct_name);
    for p in path {
        full.push('.');
        full.push_str(p);
    }
    full.push('.');
    full.push_str(field);
    err(format!(
        "decode request: json: cannot unmarshal {value} into Go struct field {full} of type {go_type}"
    ))
}

fn bind_uint8_element(
    v: &Value,
    struct_name: &str,
    path: &[String],
    field: &str,
) -> Result<u8, Error> {
    match v {
        Value::Null => Ok(0),
        Value::Number(lit) => lit
            .parse::<u8>()
            .map_err(|_| type_error(struct_name, path, field, Some(lit), v, "uint8")),
        _ => Err(type_error(struct_name, path, field, None, v, "uint8")),
    }
}

fn bind_bytes_value(
    v: &Value,
    struct_name: &str,
    path: &[String],
    field: &str,
    go_type: &str,
) -> Result<Vec<u8>, Error> {
    match v {
        Value::Null => Ok(Vec::new()),
        Value::Str(s) => crate::ssh::b64_decode_go(s.as_bytes())
            .map_err(|off| err(format!("decode request: {}", crate::ssh::b64_corrupt(off)))),
        Value::Array(items) => {
            let mut out = Vec::with_capacity(items.len());
            for item in items {
                out.push(bind_uint8_element(item, struct_name, path, field)?);
            }
            Ok(out)
        }
        _ => Err(type_error(struct_name, path, field, None, v, go_type)),
    }
}

fn bind_value(
    v: &Value,
    spec: &Spec,
    struct_name: &str,
    path: &[String],
    tolerant: bool,
) -> Result<Bound, Error> {
    // Error paths use the spec (struct field) name; values bind from any
    // fold-matching key.
    let field = spec.name;
    match &spec.kind {
        Kind::Str => match v {
            Value::Null => Ok(Bound::Str(String::new())),
            Value::Str(s) => Ok(Bound::Str(s.clone())),
            _ => Err(type_error(struct_name, path, field, None, v, "string")),
        },
        Kind::Bool => match v {
            Value::Null => Ok(Bound::Bool(false)),
            Value::Bool(b) => Ok(Bound::Bool(*b)),
            _ => Err(type_error(struct_name, path, field, None, v, "bool")),
        },
        Kind::I64 => match v {
            Value::Null => Ok(Bound::I64(0)),
            Value::Number(lit) => lit
                .parse::<i64>()
                .map(Bound::I64)
                .map_err(|_| type_error(struct_name, path, field, Some(lit), v, "int64")),
            _ => Err(type_error(struct_name, path, field, None, v, "int64")),
        },
        Kind::Int => match v {
            Value::Null => Ok(Bound::I64(0)),
            Value::Number(lit) => parse_go_int64(lit)
                .map(Bound::I64)
                .ok_or_else(|| type_error(struct_name, path, field, Some(lit), v, "int")),
            _ => Err(type_error(struct_name, path, field, None, v, "int")),
        },
        Kind::U32 => match v {
            Value::Null => Ok(Bound::U32(0)),
            Value::Number(lit) => parse_go_uint32(lit)
                .map(Bound::U32)
                .ok_or_else(|| type_error(struct_name, path, field, Some(lit), v, "uint32")),
            _ => Err(type_error(struct_name, path, field, None, v, "uint32")),
        },
        Kind::OptInt => match v {
            Value::Null => Ok(Bound::OptInt(None)),
            Value::Number(lit) => lit
                .parse::<i64>()
                .map(|n| Bound::OptInt(Some(n)))
                .map_err(|_| type_error(struct_name, path, field, Some(lit), v, "int")),
            _ => Err(type_error(struct_name, path, field, None, v, "int")),
        },
        Kind::StrList => match v {
            Value::Null => Ok(Bound::StrList(Vec::new())),
            Value::Array(items) => {
                let mut out = Vec::with_capacity(items.len());
                for item in items {
                    match item {
                        Value::Null => out.push(String::new()),
                        Value::Str(s) => out.push(s.clone()),
                        _ => {
                            return Err(type_error(struct_name, path, field, None, item, "string"))
                        }
                    }
                }
                Ok(Bound::StrList(out))
            }
            _ => Err(type_error(struct_name, path, field, None, v, "[]string")),
        },
        Kind::Bytes => bind_bytes_value(v, struct_name, path, field, "[]uint8").map(Bound::Bytes),
        Kind::BytesMap => match v {
            Value::Null => Ok(Bound::BytesMap(HashMap::new())),
            Value::Object(entries) => {
                let mut out = HashMap::with_capacity(entries.len());
                for (k, val) in entries {
                    out.insert(
                        k.clone(),
                        bind_bytes_value(val, struct_name, path, field, "[]uint8")?,
                    );
                }
                Ok(Bound::BytesMap(out))
            }
            _ => Err(type_error(
                struct_name,
                path,
                field,
                None,
                v,
                "map[string][]uint8",
            )),
        },
        Kind::Object {
            go_type,
            struct_name: nested,
            specs,
        } => match v {
            Value::Null => Ok(Bound::Map(BoundMap::default())),
            Value::Object(_) => {
                let mut child = path.to_vec();
                child.push(field.to_string());
                bind_struct(v, nested, &child, specs, tolerant).map(Bound::Map)
            }
            _ => Err(type_error(struct_name, path, field, None, v, go_type)),
        },
        Kind::OptObject {
            go_type,
            struct_name: nested,
            specs,
        } => match v {
            Value::Null => Ok(Bound::OptMap(None)),
            Value::Object(_) => {
                let mut child = path.to_vec();
                child.push(field.to_string());
                bind_struct(v, nested, &child, specs, tolerant).map(|m| Bound::OptMap(Some(m)))
            }
            _ => Err(type_error(struct_name, path, field, None, v, go_type)),
        },
        Kind::StructList {
            go_type,
            struct_name: nested,
            specs,
        } => match v {
            Value::Null => Ok(Bound::StructList(Vec::new())),
            Value::Array(items) => {
                let mut out = Vec::with_capacity(items.len());
                let mut child = path.to_vec();
                child.push(field.to_string());
                for item in items {
                    match item {
                        Value::Object(_) => {
                            out.push(bind_struct(item, nested, &child, specs, tolerant)?)
                        }
                        _ => return Err(type_error(struct_name, path, field, None, item, go_type)),
                    }
                }
                Ok(Bound::StructList(out))
            }
            _ => Err(type_error(struct_name, path, field, None, v, go_type)),
        },
    }
}

fn match_spec<'s>(specs: &'s [Spec], key: &str) -> Option<&'s Spec> {
    if let Some(spec) = specs.iter().find(|s| s.name == key) {
        return Some(spec);
    }
    let mut fold = specs.iter().filter(|s| s.name.eq_ignore_ascii_case(key));
    let first = fold.next()?;
    if fold.next().is_some() {
        // Several fields fold to this key: Go hides all of them.
        return None;
    }
    Some(first)
}

/// Ordered struct binding with exact `encoding/json` semantics: fields are
/// processed in value order (top level is pre-sorted like strictjson's
/// re-marshal, nested levels keep document order), the first error wins,
/// fold-matching keys all bind with last-wins, and unknown fields error
/// inline. `path` is the JSON field path from the root for messages.
/// Ordered struct binding with exact `encoding/json` semantics: fields are
/// processed in value order (top level is pre-sorted like strictjson's
/// re-marshal, nested levels keep document order), the first error wins,
/// fold-matching keys all bind with last-wins, and unknown fields error
/// inline (or are ignored when `tolerant`, like plain `Unmarshal`).
/// `path` is the JSON field path from the root for messages.
pub fn bind_struct(
    v: &Value,
    struct_name: &'static str,
    path: &[String],
    specs: &[Spec],
    tolerant: bool,
) -> Result<BoundMap, Error> {
    let fields = match v {
        Value::Object(fields) => fields,
        _ => return Err(err("decode request: expected object")),
    };
    let mut out = BoundMap::default();
    for (key, val) in fields.iter() {
        let Some(spec) = match_spec(specs, key) else {
            if tolerant {
                continue;
            }
            return Err(err(format!(
                "decode request: json: unknown field {}",
                go_quote(key)
            )));
        };
        // Null leaves the zero value, exactly like a missing field (unknown
        // nulls still error above). Array elements and map values handle
        // their own nulls inside their kinds.
        if val.is_null() {
            continue;
        }
        out.insert(
            spec.name.to_string(),
            bind_value(val, spec, struct_name, path, tolerant)?,
        );
    }
    Ok(out)
}

/// Strict binding entry point: top-level struct with an empty path.
pub fn bind_root(
    v: &Value,
    struct_name: &'static str,
    specs: &[Spec],
    tolerant: bool,
) -> Result<BoundMap, Error> {
    bind_struct(v, struct_name, &[], specs, tolerant)
}
