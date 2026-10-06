use soda_json::{escape_into, JsonValue};

use super::base64_encode;

/// Indented emitter mirroring `json.MarshalIndent(v, "", "  ")`.
#[derive(Default)]
pub struct Emitter {
    out: String,
    level: usize,
}

impl Emitter {
    pub fn new() -> Emitter {
        Emitter::default()
    }

    pub fn finish(mut self) -> String {
        self.out.push('\n');
        self.out
    }

    fn indent(&mut self) {
        for _ in 0..self.level {
            self.out.push_str("  ");
        }
    }

    pub fn null(&mut self) {
        self.out.push_str("null");
    }

    pub fn boolean(&mut self, value: bool) {
        self.out.push_str(if value { "true" } else { "false" });
    }

    pub fn int(&mut self, value: i64) {
        self.out.push_str(&value.to_string());
    }

    pub fn uint(&mut self, value: u64) {
        self.out.push_str(&value.to_string());
    }

    pub fn string(&mut self, value: &str) {
        escape_into(&mut self.out, value);
    }

    pub fn bytes(&mut self, value: &[u8]) {
        escape_into(&mut self.out, &base64_encode(value));
    }

    pub fn begin_object(&mut self, empty: bool) {
        if empty {
            self.out.push_str("{}");
        } else {
            self.out.push_str("{\n");
            self.level += 1;
        }
    }

    pub fn field(&mut self, first: bool, name: &str) {
        if !first {
            self.out.push_str(",\n");
        }
        self.indent();
        escape_into(&mut self.out, name);
        self.out.push_str(": ");
    }

    pub fn end_object(&mut self, empty: bool) {
        if !empty {
            self.out.push('\n');
            self.level -= 1;
            self.indent();
            self.out.push('}');
        }
    }

    pub fn begin_array(&mut self, empty: bool) {
        if empty {
            self.out.push_str("[]");
        } else {
            self.out.push_str("[\n");
            self.level += 1;
        }
    }

    pub fn item(&mut self, first: bool) {
        if !first {
            self.out.push_str(",\n");
        }
        self.indent();
    }

    pub fn end_array(&mut self, empty: bool) {
        if !empty {
            self.out.push('\n');
            self.level -= 1;
            self.indent();
            self.out.push(']');
        }
    }
}

/// Values the port marshals with Go field order and shape.
pub trait Emit {
    fn emit(&self, e: &mut Emitter);
}

impl Emit for String {
    fn emit(&self, e: &mut Emitter) {
        e.string(self);
    }
}

impl Emit for str {
    fn emit(&self, e: &mut Emitter) {
        e.string(self);
    }
}

impl Emit for bool {
    fn emit(&self, e: &mut Emitter) {
        e.boolean(*self);
    }
}

impl Emit for i64 {
    fn emit(&self, e: &mut Emitter) {
        e.int(*self);
    }
}

impl Emit for u64 {
    fn emit(&self, e: &mut Emitter) {
        e.uint(*self);
    }
}

impl Emit for i32 {
    fn emit(&self, e: &mut Emitter) {
        e.int(*self as i64);
    }
}

impl Emit for Vec<String> {
    fn emit(&self, e: &mut Emitter) {
        e.begin_array(self.is_empty());
        for (i, item) in self.iter().enumerate() {
            e.item(i == 0);
            e.string(item);
        }
        e.end_array(self.is_empty());
    }
}

impl Emit for std::collections::BTreeMap<String, String> {
    fn emit(&self, e: &mut Emitter) {
        e.begin_object(self.is_empty());
        for (i, (key, value)) in self.iter().enumerate() {
            e.field(i == 0, key);
            e.string(value);
        }
        e.end_object(self.is_empty());
    }
}

impl Emit for std::collections::BTreeMap<String, u64> {
    fn emit(&self, e: &mut Emitter) {
        e.begin_object(self.is_empty());
        for (i, (key, value)) in self.iter().enumerate() {
            e.field(i == 0, key);
            e.uint(*value);
        }
        e.end_object(self.is_empty());
    }
}

/// Indented emission of a dynamic value. Object entries keep document
/// order: callers modeling Go structs build struct order, callers modeling
/// Go maps pre-sort keys. Numbers are emitted verbatim.
impl Emit for JsonValue {
    fn emit(&self, e: &mut Emitter) {
        match self {
            JsonValue::Null => e.null(),
            JsonValue::Bool(b) => e.boolean(*b),
            JsonValue::Number(raw) => {
                // Numbers in this port are always canonical literals.
                e.out.push_str(raw);
            }
            JsonValue::Str(s) => e.string(s),
            JsonValue::Array(items) => {
                e.begin_array(items.is_empty());
                for (i, item) in items.iter().enumerate() {
                    e.item(i == 0);
                    item.emit(e);
                }
                e.end_array(items.is_empty());
            }
            JsonValue::Object(entries) => {
                e.begin_object(entries.is_empty());
                for (i, (key, value)) in entries.iter().enumerate() {
                    e.field(i == 0, key);
                    value.emit(e);
                }
                e.end_object(entries.is_empty());
            }
        }
    }
}
