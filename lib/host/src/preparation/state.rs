// Observed preparation states: resolved tools, inspect/stop/hold wire shapes.
use super::valid_preparation_id;
use crate::domain;
use crate::json;
use serde::de::{self, MapAccess, Visitor};
use serde::Deserialize;
use std::fmt;

#[derive(Debug, Clone, PartialEq, Eq, Default)]
pub struct ResolvedTool {
    pub name: String,
    pub path: String,
    pub version: String,
}

impl ResolvedTool {
    pub fn encode_into(&self, out: &mut String) {
        out.push_str("{\"name\":");
        out.push_str(&json::quote(&self.name));
        out.push_str(",\"path\":");
        out.push_str(&json::quote(&self.path));
        out.push_str(",\"version\":");
        out.push_str(&json::quote(&self.version));
        out.push('}');
    }
}

macro_rules! typed_state_object {
    ($ty:ident, $expect:literal, {$($field:ident => $name:literal : $value:ty),+ $(,)?}) => {
        impl<'de> Deserialize<'de> for $ty {
            fn deserialize<D>(deserializer: D) -> Result<Self, D::Error>
            where D: serde::Deserializer<'de> {
                struct ObjectVisitor;
                impl<'de> Visitor<'de> for ObjectVisitor {
                    type Value = $ty;
                    fn expecting(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result { f.write_str($expect) }
                    fn visit_map<A>(self, mut map: A) -> Result<Self::Value, A::Error>
                    where A: MapAccess<'de> {
                        let mut out = $ty { $($field: Default::default()),+ };
                        while let Some(key) = map.next_key::<String>()? {
                            let mut recognized = false;
                            $(if key.eq_ignore_ascii_case($name) {
                                if let Some(value) = map.next_value::<Option<$value>>()? { out.$field = value.into(); }
                                recognized = true;
                            })+
                            if !recognized { return Err(de::Error::unknown_field(&key, &[$($name),+])); }
                        }
                        Ok(out)
                    }
                }
                deserializer.deserialize_map(ObjectVisitor)
            }
        }
    };
}

typed_state_object!(ResolvedTool, "a resolved tool object", {
    name => "name": String, path => "path": String, version => "version": String
});

#[derive(Debug, Clone, PartialEq, Eq, Default)]
pub struct PrepareState {
    pub id: String,
    pub project: String,
    pub role: String,
    pub phase: String,
    pub container: String,
    pub source_commit: String,
    pub setup_digest: String,
    pub tools: Vec<ResolvedTool>,
    pub missing: String,
    pub setup_exit: Option<i64>,
    pub check_exit: Option<i64>,
    pub output: String,
    pub ready: bool,
    pub stopped: bool,
    pub retirement: String,
}

impl PrepareState {
    /// `encoding/json` struct order with `omitempty` honored.
    pub fn encode(&self) -> String {
        let mut out = String::from("{\"id\":");
        out.push_str(&json::quote(&self.id));
        out.push_str(",\"project\":");
        out.push_str(&json::quote(&self.project));
        out.push_str(",\"role\":");
        out.push_str(&json::quote(&self.role));
        out.push_str(",\"phase\":");
        out.push_str(&json::quote(&self.phase));
        out.push_str(",\"container\":");
        out.push_str(&json::quote(&self.container));
        out.push_str(",\"source_commit\":");
        out.push_str(&json::quote(&self.source_commit));
        out.push_str(",\"setup_digest\":");
        out.push_str(&json::quote(&self.setup_digest));
        if !self.tools.is_empty() {
            out.push_str(",\"tools\":[");
            for (i, tool) in self.tools.iter().enumerate() {
                if i > 0 {
                    out.push(',');
                }
                tool.encode_into(&mut out);
            }
            out.push(']');
        }
        if !self.missing.is_empty() {
            out.push_str(",\"missing\":");
            out.push_str(&json::quote(&self.missing));
        }
        if let Some(exit) = self.setup_exit {
            out.push_str(",\"setup_exit\":");
            out.push_str(&exit.to_string());
        }
        if let Some(exit) = self.check_exit {
            out.push_str(",\"check_exit\":");
            out.push_str(&exit.to_string());
        }
        if !self.output.is_empty() {
            out.push_str(",\"output\":");
            out.push_str(&json::quote(&self.output));
        }
        out.push_str(",\"ready\":");
        out.push_str(if self.ready { "true" } else { "false" });
        out.push_str(",\"stopped\":");
        out.push_str(if self.stopped { "true" } else { "false" });
        if !self.retirement.is_empty() {
            out.push_str(",\"retirement\":");
            out.push_str(&json::quote(&self.retirement));
        }
        out.push('}');
        out
    }
}

#[derive(Debug, Clone, PartialEq, Eq, Default)]
pub struct PrepareInspect {
    pub project: String,
    pub id: String,
}

impl PrepareInspect {
    pub fn validate(&self) -> Result<(), String> {
        if !domain::valid_id(&self.project) || !valid_preparation_id(&self.id) {
            return Err("invalid preparation address".to_string());
        }
        Ok(())
    }

    pub fn decode(body: &[u8]) -> Result<Self, String> {
        json::decode_strict_as(body).map_err(|e| e.0)
    }
}

#[derive(Debug, Clone, PartialEq, Eq, Default)]
pub struct PrepareStop {
    pub project: String,
    pub id: String,
}

impl PrepareStop {
    pub fn validate(&self) -> Result<(), String> {
        if !domain::valid_id(&self.project) || !valid_preparation_id(&self.id) {
            return Err("invalid preparation address".to_string());
        }
        Ok(())
    }

    pub fn decode(body: &[u8]) -> Result<Self, String> {
        json::decode_strict_as(body).map_err(|e| e.0)
    }
}

typed_state_object!(PrepareInspect, "a preparation inspection object", {
    project => "project": String, id => "id": String
});
typed_state_object!(PrepareStop, "a preparation stop object", {
    project => "project": String, id => "id": String
});

#[derive(Debug, Clone, PartialEq, Eq, Default)]
pub struct HoldState {
    pub active: bool,
    pub revision: i64,
}

impl HoldState {
    pub fn encode(&self) -> String {
        format!(
            "{{\"active\":{},\"revision\":{}}}",
            if self.active { "true" } else { "false" },
            self.revision
        )
    }
}

typed_state_object!(HoldState, "a maintenance hold object", {
    active => "active": bool, revision => "revision": json::SignedInteger
});

#[derive(Debug, Clone, PartialEq, Eq, Default)]
pub struct PrepareHold {
    pub project: String,
    pub hold: bool,
    pub revision: i64,
}

impl PrepareHold {
    pub fn validate(&self) -> Result<(), String> {
        if !domain::valid_id(&self.project) || self.revision < 0 {
            return Err("invalid maintenance hold".to_string());
        }
        Ok(())
    }

    pub fn decode(body: &[u8]) -> Result<Self, String> {
        json::decode_strict_as(body).map_err(|e| e.0)
    }
}

typed_state_object!(PrepareHold, "a maintenance hold request object", {
    project => "project": String, hold => "hold": bool, revision => "revision": json::SignedInteger
});
