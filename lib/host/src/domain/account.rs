use crate::json::{self, BoundMap, Kind, Spec, Value};

// ---- PR20: account DTOs (`Account`, `AccessKeys`, `AccessKeyState`) ----

/// `Account`: a project login identity with its authorized keys.
#[derive(Debug, Clone, PartialEq, Eq)]
pub struct Account {
    pub project: String,
    pub login: String,
    pub identity: i64,
    pub keys: Vec<String>,
}

const ACCOUNT_SPECS: &[Spec] = &[
    Spec {
        name: "project",
        kind: Kind::Str,
    },
    Spec {
        name: "login",
        kind: Kind::Str,
    },
    Spec {
        name: "identity",
        kind: Kind::I64,
    },
    Spec {
        name: "keys",
        kind: Kind::StrList,
    },
];

impl Account {
    pub fn from_value(v: &Value) -> Result<Self, String> {
        let m = json::bind_root(v, "Account", ACCOUNT_SPECS, false).map_err(|e| e.0)?;
        Ok(Self::from_map(&m))
    }

    pub fn from_map(m: &BoundMap) -> Self {
        Account {
            project: m.take_string("project"),
            login: m.take_string("login"),
            identity: m.take_i64("identity"),
            keys: m.take_str_list("keys"),
        }
    }
}

/// `AccessKeys`: replace or observe a login's authorized key set.
#[derive(Debug, Clone, PartialEq, Eq)]
pub struct AccessKeys {
    pub project: String,
    pub login: String,
    pub identity: i64,
    pub revision: String,
    pub keys: Vec<String>,
    pub apply: bool,
}

const ACCESS_KEYS_SPECS: &[Spec] = &[
    Spec {
        name: "project",
        kind: Kind::Str,
    },
    Spec {
        name: "login",
        kind: Kind::Str,
    },
    Spec {
        name: "identity",
        kind: Kind::I64,
    },
    Spec {
        name: "revision",
        kind: Kind::Str,
    },
    Spec {
        name: "keys",
        kind: Kind::StrList,
    },
    Spec {
        name: "apply",
        kind: Kind::Bool,
    },
];

impl AccessKeys {
    pub fn from_value(v: &Value) -> Result<Self, String> {
        let m = json::bind_root(v, "AccessKeys", ACCESS_KEYS_SPECS, false).map_err(|e| e.0)?;
        Ok(Self::from_map(&m))
    }

    pub fn from_map(m: &BoundMap) -> Self {
        AccessKeys {
            project: m.take_string("project"),
            login: m.take_string("login"),
            identity: m.take_i64("identity"),
            revision: m.take_string("revision"),
            keys: m.take_str_list("keys"),
            apply: m.take_bool("apply"),
        }
    }
}

/// `AccessKeyState`: the observed key set and its revision.
#[derive(Debug, Clone, PartialEq, Eq)]
pub struct AccessKeyState {
    pub revision: String,
    pub keys: Vec<String>,
}

impl AccessKeyState {
    /// `encoding/json` struct order (`revision`, `keys`), no trailing newline.
    pub fn encode(&self) -> String {
        let mut out = String::from("{\"revision\":");
        out.push_str(&json::quote(&self.revision));
        out.push_str(",\"keys\":[");
        for (i, k) in self.keys.iter().enumerate() {
            if i > 0 {
                out.push(',');
            }
            out.push_str(&json::quote(k));
        }
        out.push_str("]}");
        out
    }
}
