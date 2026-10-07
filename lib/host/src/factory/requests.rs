use crate::domain;
use crate::json;
use crate::preparation;
use serde::de::{self, MapAccess, Visitor};
use serde::Deserialize;
use std::fmt;

use super::{
    takeover_destination, valid_factory_run_id, MAX_FACTORY_OUTPUT_OFFSET, MAX_FACTORY_OUTPUT_READ,
};

/// `project.FactoryInspect`: address a run before or after completion.
#[derive(Debug, Clone, Default, PartialEq, Eq)]
pub struct FactoryInspect {
    pub project: String,
    pub id: String,
}

impl FactoryInspect {
    pub fn decode(body: &[u8]) -> Result<Self, String> {
        json::decode_strict_as(body).map_err(|e| e.0)
    }

    pub fn validate(&self) -> Result<(), String> {
        if !domain::valid_id(&self.project) || !valid_factory_run_id(&self.id) {
            return Err("invalid factory run address".to_string());
        }
        Ok(())
    }
}

/// `project.FactoryStop`: retire one run identity.
#[derive(Debug, Clone, Default, PartialEq, Eq)]
pub struct FactoryStop {
    pub project: String,
    pub id: String,
}

impl FactoryStop {
    pub fn decode(body: &[u8]) -> Result<Self, String> {
        json::decode_strict_as(body).map_err(|e| e.0)
    }

    pub fn validate(&self) -> Result<(), String> {
        if !domain::valid_id(&self.project) || !valid_factory_run_id(&self.id) {
            return Err("invalid factory run address".to_string());
        }
        Ok(())
    }
}

/// `project.FactoryTakeover`: address one reconciled run and the admitted
/// member receiving its retained work.
#[derive(Debug, Clone, Default, PartialEq, Eq)]
pub struct FactoryTakeover {
    pub project: String,
    pub id: String,
    pub member: String,
}

impl FactoryTakeover {
    pub fn decode(body: &[u8]) -> Result<Self, String> {
        json::decode_strict_as(body).map_err(|e| e.0)
    }
}

impl FactoryTakeover {
    pub fn validate(&self) -> Result<(), String> {
        if !domain::valid_id(&self.project) || !valid_factory_run_id(&self.id) {
            return Err("invalid factory takeover address".to_string());
        }
        if !domain::valid_login(&self.member) || self.member == "root" {
            return Err("invalid takeover member".to_string());
        }
        Ok(())
    }
}

/// `project.TakeoverResult`: the confirmed takeover outcome.
#[derive(Debug, Clone, Default, PartialEq, Eq)]
pub struct TakeoverResult {
    pub id: String,
    pub project: String,
    pub member: String,
    pub destination: String,
    pub reused: bool,
}

impl TakeoverResult {
    pub fn validate(&self) -> Result<(), String> {
        if !domain::valid_id(&self.project) || !valid_factory_run_id(&self.id) {
            return Err("invalid takeover result identity".to_string());
        }
        if takeover_destination(&self.member, &self.id) != self.destination {
            return Err("takeover destination does not match its identities".to_string());
        }
        Ok(())
    }

    pub fn encode(&self) -> String {
        let mut out = String::from("{\"id\":");
        out.push_str(&json::quote(&self.id));
        out.push_str(",\"project\":");
        out.push_str(&json::quote(&self.project));
        out.push_str(",\"member\":");
        out.push_str(&json::quote(&self.member));
        out.push_str(",\"destination\":");
        out.push_str(&json::quote(&self.destination));
        out.push_str(",\"reused\":");
        out.push_str(if self.reused { "true" } else { "false" });
        out.push('}');
        out
    }
}

/// `project.FactoryOutput`: address one bounded output slice.
#[derive(Debug, Clone, Default, PartialEq, Eq)]
pub struct FactoryOutput {
    pub project: String,
    pub id: String,
    pub offset: i64,
    pub limit: i64,
}

impl FactoryOutput {
    pub fn decode(body: &[u8]) -> Result<Self, String> {
        json::decode_strict_as(body).map_err(|e| e.0)
    }
}

impl FactoryOutput {
    pub fn validate(&self) -> Result<(), String> {
        if !domain::valid_id(&self.project) || !valid_factory_run_id(&self.id) {
            return Err("invalid factory run address".to_string());
        }
        if self.offset < 0 || self.offset > MAX_FACTORY_OUTPUT_OFFSET {
            return Err("invalid output cursor".to_string());
        }
        if self.limit < 1 || self.limit > MAX_FACTORY_OUTPUT_READ {
            return Err("invalid output read bound".to_string());
        }
        Ok(())
    }
}

/// `project.FactoryOutputState`: one observed output slice.
#[derive(Debug, Clone, Default, PartialEq, Eq)]
pub struct FactoryOutputState {
    pub exit_code: Option<i64>,
    pub live: bool,
    pub terminal: bool,
    pub truncated: bool,
    pub gap: bool,
    pub id: String,
    pub project: String,
    pub phase: String,
    pub container: String,
    pub unit: String,
    pub invocation: String,
    pub total: i64,
    pub offset: i64,
    pub next: i64,
    pub data: String,
    pub reason: String,
}

impl FactoryOutputState {
    pub fn encode(&self) -> String {
        let mut out = String::from("{");
        let mut first = true;
        let mut field = |out: &mut String, name: &str, value: &str| {
            if !first {
                out.push(',');
            }
            first = false;
            out.push_str(&json::quote(name));
            out.push(':');
            out.push_str(value);
        };
        if let Some(code) = self.exit_code {
            field(&mut out, "exit_code", &code.to_string());
        }
        field(&mut out, "live", if self.live { "true" } else { "false" });
        field(
            &mut out,
            "terminal",
            if self.terminal { "true" } else { "false" },
        );
        field(
            &mut out,
            "truncated",
            if self.truncated { "true" } else { "false" },
        );
        field(&mut out, "gap", if self.gap { "true" } else { "false" });
        field(&mut out, "id", &json::quote(&self.id));
        field(&mut out, "project", &json::quote(&self.project));
        field(&mut out, "phase", &json::quote(&self.phase));
        field(&mut out, "container", &json::quote(&self.container));
        field(&mut out, "unit", &json::quote(&self.unit));
        field(&mut out, "invocation", &json::quote(&self.invocation));
        field(&mut out, "total", &self.total.to_string());
        field(&mut out, "offset", &self.offset.to_string());
        field(&mut out, "next", &self.next.to_string());
        field(&mut out, "data", &json::quote(&self.data));
        if !self.reason.is_empty() {
            field(&mut out, "reason", &json::quote(&self.reason));
        }
        out.push('}');
        out
    }
}

/// `project.FactoryExport`: address one settled run's exact candidate.
#[derive(Debug, Clone, Default, PartialEq, Eq)]
pub struct FactoryExport {
    pub project: String,
    pub id: String,
    pub role: String,
    pub preparation: String,
    pub candidate: String,
}

impl FactoryExport {
    pub fn decode(body: &[u8]) -> Result<Self, String> {
        json::decode_strict_as(body).map_err(|e| e.0)
    }
}

impl FactoryExport {
    pub fn validate(&self) -> Result<(), String> {
        if !domain::valid_id(&self.project) || !valid_factory_run_id(&self.id) {
            return Err("invalid factory export address".to_string());
        }
        if !preparation::valid_factory_role(&self.role) {
            return Err("invalid factory export role".to_string());
        }
        if !preparation::valid_preparation_id(&self.preparation) {
            return Err("invalid factory export preparation".to_string());
        }
        if !preparation::valid_commit(&self.candidate) {
            return Err("invalid factory export candidate".to_string());
        }
        Ok(())
    }
}

/// `project.FactoryExportState`: one observed candidate export.
#[derive(Debug, Clone, Default, PartialEq, Eq)]
pub struct FactoryExportState {
    pub id: String,
    pub project: String,
    pub phase: String,
    pub container: String,
    pub candidate: String,
    pub bundle: String,
}

impl FactoryExportState {
    pub fn encode(&self) -> String {
        let mut out = String::from("{\"id\":");
        out.push_str(&json::quote(&self.id));
        out.push_str(",\"project\":");
        out.push_str(&json::quote(&self.project));
        out.push_str(",\"phase\":");
        out.push_str(&json::quote(&self.phase));
        out.push_str(",\"container\":");
        out.push_str(&json::quote(&self.container));
        out.push_str(",\"candidate\":");
        out.push_str(&json::quote(&self.candidate));
        out.push_str(",\"bundle\":");
        out.push_str(&json::quote(&self.bundle));
        out.push('}');
        out
    }
}

/// `project.FactoryCandidateInspect`: address a checkout through a settled
/// recorded run.
#[derive(Debug, Clone, Default, PartialEq, Eq)]
pub struct FactoryCandidateInspect {
    pub project: String,
    pub id: String,
}

macro_rules! typed_strict_object {
    ($ty:ty, $expect:literal, {$($field:ident => $name:literal : $value:ty),+ $(,)?}) => {
        impl<'de> Deserialize<'de> for $ty {
            fn deserialize<D>(deserializer: D) -> Result<Self, D::Error>
            where D: serde::Deserializer<'de> {
                struct ObjectVisitor;
                impl<'de> Visitor<'de> for ObjectVisitor {
                    type Value = $ty;
                    fn expecting(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result { f.write_str($expect) }
                    fn visit_map<A>(self, mut map: A) -> Result<Self::Value, A::Error>
                    where A: MapAccess<'de> {
                        let mut out = <$ty>::default();
                        while let Some(key) = map.next_key::<String>()? {
                            $(if key.eq_ignore_ascii_case($name) {
                                if let Some(value) = map.next_value::<Option<$value>>()? { out.$field = value.into(); }
                                continue;
                            })+
                            return Err(de::Error::unknown_field(&key, &[$($name),+]));
                        }
                        Ok(out)
                    }
                }
                deserializer.deserialize_map(ObjectVisitor)
            }
        }
    };
}

typed_strict_object!(FactoryInspect, "a factory inspect request", {
    project => "project": String,
    id => "id": String,
});
typed_strict_object!(FactoryStop, "a factory stop request", {
    project => "project": String,
    id => "id": String,
});
typed_strict_object!(FactoryTakeover, "a factory takeover request", {
    project => "project": String,
    id => "id": String,
    member => "member": String,
});
typed_strict_object!(FactoryOutput, "a factory output request", {
    project => "project": String,
    id => "id": String,
    offset => "offset": json::SignedInteger,
    limit => "limit": json::SignedInteger,
});
typed_strict_object!(FactoryExport, "a factory export request", {
    project => "project": String,
    id => "id": String,
    role => "role": String,
    preparation => "preparation": String,
    candidate => "candidate": String,
});
typed_strict_object!(FactoryCandidateInspect, "a factory candidate request", {
    project => "project": String,
    id => "id": String,
});

impl FactoryCandidateInspect {
    pub fn decode(body: &[u8]) -> Result<Self, String> {
        json::decode_strict_as(body).map_err(|e| e.0)
    }
}

impl FactoryCandidateInspect {
    pub fn validate(&self) -> Result<(), String> {
        if !domain::valid_id(&self.project) || !valid_factory_run_id(&self.id) {
            return Err("invalid candidate inspection address".to_string());
        }
        Ok(())
    }
}

/// `project.FactoryCandidateState`: the observed checkout HEAD and dirtiness.
#[derive(Debug, Clone, Default, PartialEq, Eq)]
pub struct FactoryCandidateState {
    pub id: String,
    pub project: String,
    pub container: String,
    pub candidate: String,
    pub dirty: bool,
}

impl FactoryCandidateState {
    pub fn encode(&self) -> String {
        let mut out = String::from("{\"id\":");
        out.push_str(&json::quote(&self.id));
        out.push_str(",\"project\":");
        out.push_str(&json::quote(&self.project));
        out.push_str(",\"container\":");
        out.push_str(&json::quote(&self.container));
        out.push_str(",\"candidate\":");
        out.push_str(&json::quote(&self.candidate));
        out.push_str(",\"dirty\":");
        out.push_str(if self.dirty { "true" } else { "false" });
        out.push('}');
        out
    }
}
