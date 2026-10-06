//! Go `json.MarshalIndent` byte emission over [`Emit`] values: HTML
//! escaping, `": "` separators, two-space indent, caller-ordered struct
//! fields.

use soda_json::escape_into;

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
