use std::collections::HashMap;

/// One struct field's binding rule. `go_type` is the exact
/// `encoding/json` type word used in mismatch messages.
pub struct Spec {
    pub name: &'static str,
    pub kind: Kind,
}

pub enum Kind {
    Str,
    Bool,
    I64,
    /// Go `int` (64-bit): same range as I64, `int` in messages.
    Int,
    /// Go `uint32`.
    U32,
    /// `*int`: missing/null is None.
    OptInt,
    StrList,
    /// `[]byte`: base64 string or numeric array.
    Bytes,
    /// `map[string][]byte`.
    BytesMap,
    /// Required nested struct: null/missing binds the zero value.
    Object {
        go_type: &'static str,
        struct_name: &'static str,
        specs: &'static [Spec],
    },
    /// `*struct`: null/missing binds None.
    OptObject {
        go_type: &'static str,
        struct_name: &'static str,
        specs: &'static [Spec],
    },
    /// `[]struct`: null/missing binds empty.
    StructList {
        go_type: &'static str,
        struct_name: &'static str,
        specs: &'static [Spec],
    },
}

/// A bound field value, keyed by spec name in [`BoundMap`].
#[derive(Debug, Clone)]
pub enum Bound {
    Str(String),
    Bool(bool),
    I64(i64),
    U32(u32),
    OptInt(Option<i64>),
    StrList(Vec<String>),
    Bytes(Vec<u8>),
    BytesMap(HashMap<String, Vec<u8>>),
    Map(BoundMap),
    OptMap(Option<BoundMap>),
    StructList(Vec<BoundMap>),
}

#[derive(Debug, Clone, Default)]
pub struct BoundMap(HashMap<String, Bound>);

impl BoundMap {
    pub(super) fn get(&self, name: &str) -> Option<&Bound> {
        self.0.get(name)
    }
    pub(super) fn insert(&mut self, name: String, bound: Bound) {
        self.0.insert(name, bound);
    }
    pub fn take_string(&self, name: &str) -> String {
        match self.get(name) {
            Some(Bound::Str(s)) => s.clone(),
            _ => String::new(),
        }
    }
    pub fn take_bool(&self, name: &str) -> bool {
        match self.get(name) {
            Some(Bound::Bool(b)) => *b,
            _ => false,
        }
    }
    pub fn take_i64(&self, name: &str) -> i64 {
        match self.get(name) {
            Some(Bound::I64(n)) => *n,
            _ => 0,
        }
    }
    pub fn take_u32(&self, name: &str) -> u32 {
        match self.get(name) {
            Some(Bound::U32(n)) => *n,
            _ => 0,
        }
    }
    pub fn take_opt_i64(&self, name: &str) -> Option<i64> {
        match self.get(name) {
            Some(Bound::OptInt(n)) => *n,
            _ => None,
        }
    }
    pub fn take_str_list(&self, name: &str) -> Vec<String> {
        match self.get(name) {
            Some(Bound::StrList(v)) => v.clone(),
            _ => Vec::new(),
        }
    }
    pub fn take_bytes(&self, name: &str) -> Vec<u8> {
        match self.get(name) {
            Some(Bound::Bytes(v)) => v.clone(),
            _ => Vec::new(),
        }
    }
    pub fn take_bytes_map(&self, name: &str) -> HashMap<String, Vec<u8>> {
        match self.get(name) {
            Some(Bound::BytesMap(m)) => m.clone(),
            _ => HashMap::new(),
        }
    }
    pub fn contains(&self, name: &str) -> bool {
        self.0.contains_key(name)
    }
    pub fn take_map(&self, name: &str) -> BoundMap {
        match self.get(name) {
            Some(Bound::Map(m)) => m.clone(),
            _ => BoundMap::default(),
        }
    }
    pub fn take_opt_map(&self, name: &str) -> Option<BoundMap> {
        match self.get(name) {
            Some(Bound::OptMap(m)) => m.clone(),
            _ => None,
        }
    }
    pub fn take_struct_list(&self, name: &str) -> Vec<BoundMap> {
        match self.get(name) {
            Some(Bound::StructList(v)) => v.clone(),
            _ => Vec::new(),
        }
    }
}
