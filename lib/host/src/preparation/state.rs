// Observed preparation states: resolved tools, inspect/stop/hold wire shapes.
use super::valid_preparation_id;
use crate::domain;
use crate::json::{self, BoundMap, Kind, Spec, Value};

#[derive(Debug, Clone, PartialEq, Eq)]
pub struct ResolvedTool {
    pub name: String,
    pub path: String,
    pub version: String,
}

pub(crate) const RESOLVED_TOOL_SPECS: &[Spec] = &[
    Spec {
        name: "name",
        kind: Kind::Str,
    },
    Spec {
        name: "path",
        kind: Kind::Str,
    },
    Spec {
        name: "version",
        kind: Kind::Str,
    },
];

impl ResolvedTool {
    pub fn from_map(m: &BoundMap) -> Self {
        ResolvedTool {
            name: m.take_string("name"),
            path: m.take_string("path"),
            version: m.take_string("version"),
        }
    }

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

#[derive(Debug, Clone, PartialEq, Eq)]
pub struct PrepareInspect {
    pub project: String,
    pub id: String,
}

const PREPARE_INSPECT_SPECS: &[Spec] = &[
    Spec {
        name: "project",
        kind: Kind::Str,
    },
    Spec {
        name: "id",
        kind: Kind::Str,
    },
];

impl PrepareInspect {
    pub fn validate(&self) -> Result<(), String> {
        if !domain::valid_id(&self.project) || !valid_preparation_id(&self.id) {
            return Err("invalid preparation address".to_string());
        }
        Ok(())
    }

    pub fn from_value(v: &Value) -> Result<Self, String> {
        let m =
            json::bind_root(v, "PrepareInspect", PREPARE_INSPECT_SPECS, false).map_err(|e| e.0)?;
        Ok(Self::from_map(&m))
    }

    pub fn from_map(m: &BoundMap) -> Self {
        PrepareInspect {
            project: m.take_string("project"),
            id: m.take_string("id"),
        }
    }
}

#[derive(Debug, Clone, PartialEq, Eq)]
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

    pub fn from_value(v: &Value) -> Result<Self, String> {
        let m = json::bind_root(v, "PrepareStop", PREPARE_INSPECT_SPECS, false).map_err(|e| e.0)?;
        Ok(Self::from_map(&m))
    }

    pub fn from_map(m: &BoundMap) -> Self {
        PrepareStop {
            project: m.take_string("project"),
            id: m.take_string("id"),
        }
    }
}

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

    pub fn from_map(m: &BoundMap) -> Self {
        HoldState {
            active: m.take_bool("active"),
            revision: m.take_i64("revision"),
        }
    }
}

pub(crate) const HOLD_STATE_SPECS: &[Spec] = &[
    Spec {
        name: "active",
        kind: Kind::Bool,
    },
    Spec {
        name: "revision",
        kind: Kind::I64,
    },
];

#[derive(Debug, Clone, PartialEq, Eq)]
pub struct PrepareHold {
    pub project: String,
    pub hold: bool,
    pub revision: i64,
}

const PREPARE_HOLD_SPECS: &[Spec] = &[
    Spec {
        name: "project",
        kind: Kind::Str,
    },
    Spec {
        name: "hold",
        kind: Kind::Bool,
    },
    Spec {
        name: "revision",
        kind: Kind::I64,
    },
];

impl PrepareHold {
    pub fn validate(&self) -> Result<(), String> {
        if !domain::valid_id(&self.project) || self.revision < 0 {
            return Err("invalid maintenance hold".to_string());
        }
        Ok(())
    }

    pub fn from_value(v: &Value) -> Result<Self, String> {
        let m = json::bind_root(v, "PrepareHold", PREPARE_HOLD_SPECS, false).map_err(|e| e.0)?;
        Ok(Self::from_map(&m))
    }

    pub fn from_map(m: &BoundMap) -> Self {
        PrepareHold {
            project: m.take_string("project"),
            hold: m.take_bool("hold"),
            revision: m.take_i64("revision"),
        }
    }
}
