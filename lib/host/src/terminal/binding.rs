use crate::json::{self, SignedInteger};
use serde::de::{self, MapAccess, Visitor};
use serde::{Deserialize, Deserializer};

use super::{err_denied, KIND_FACTORY, KIND_TERMINAL};

/// Native process boundary (`identity.Binding`).
#[derive(Debug, Clone, Default, PartialEq, Eq)]
pub struct Binding {
    pub child_id: String,
    pub uid: i64,
    pub gid: i64,
    pub scope: String,
    pub credential_root: String,
    pub invocation_id: String,
    pub kind: String,
    pub id: String,
    pub project: String,
    pub login: String,
    pub generation: i64,
}

impl<'de> Deserialize<'de> for Binding {
    /// Decode a nested wire binding. Lease, delivery and Factory receipt
    /// entry points retain their strict whole-document admission checks.
    fn deserialize<D>(deserializer: D) -> Result<Self, D::Error>
    where
        D: Deserializer<'de>,
    {
        BindingWire::deserialize(deserializer).map(binding_from_wire)
    }
}

#[derive(Default)]
struct BindingWire {
    child_id: Option<String>,
    uid: Option<SignedInteger>,
    gid: Option<SignedInteger>,
    scope: Option<String>,
    credential_root: Option<String>,
    invocation_id: Option<String>,
    kind: Option<String>,
    id: Option<String>,
    project: Option<String>,
    login: Option<String>,
    generation: Option<SignedInteger>,
}

impl<'de> Deserialize<'de> for BindingWire {
    fn deserialize<D>(d: D) -> Result<Self, D::Error>
    where
        D: Deserializer<'de>,
    {
        struct V;
        impl<'de> Visitor<'de> for V {
            type Value = BindingWire;
            fn expecting(&self, f: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
                f.write_str("identity binding")
            }
            fn visit_map<A>(self, mut map: A) -> Result<Self::Value, A::Error>
            where
                A: MapAccess<'de>,
            {
                let mut o = BindingWire::default();
                while let Some(k) = map.next_key::<String>()? {
                    let key = k.to_ascii_lowercase();
                    if take_string_field(&mut o, key.as_str(), &mut map)? {
                        continue;
                    }
                    if take_integer_field(&mut o, key.as_str(), &mut map)? {
                        continue;
                    }
                    return Err(de::Error::unknown_field(
                        &k,
                        &[
                            "child_id",
                            "uid",
                            "gid",
                            "scope",
                            "credential_root",
                            "invocation_id",
                            "kind",
                            "id",
                            "project",
                            "login",
                            "generation",
                        ],
                    ));
                }
                Ok(o)
            }
        }
        d.deserialize_map(V)
    }
}

fn take_string_field<'de, A>(o: &mut BindingWire, key: &str, map: &mut A) -> Result<bool, A::Error>
where
    A: MapAccess<'de>,
{
    let slot = match key {
        "child_id" => &mut o.child_id,
        "scope" => &mut o.scope,
        "credential_root" => &mut o.credential_root,
        "invocation_id" => &mut o.invocation_id,
        "kind" => &mut o.kind,
        "id" => &mut o.id,
        "project" => &mut o.project,
        "login" => &mut o.login,
        _ => return Ok(false),
    };
    if let Some(v) = map.next_value::<Option<String>>()? {
        *slot = Some(v);
    }
    Ok(true)
}

fn take_integer_field<'de, A>(o: &mut BindingWire, key: &str, map: &mut A) -> Result<bool, A::Error>
where
    A: MapAccess<'de>,
{
    let slot = match key {
        "uid" => &mut o.uid,
        "gid" => &mut o.gid,
        "generation" => &mut o.generation,
        _ => return Ok(false),
    };
    if let Some(v) = map.next_value::<Option<SignedInteger>>()? {
        *slot = Some(v);
    }
    Ok(true)
}

fn binding_from_wire(m: BindingWire) -> Binding {
    Binding {
        child_id: m.child_id.unwrap_or_default(),
        uid: m.uid.map(Into::into).unwrap_or_default(),
        gid: m.gid.map(Into::into).unwrap_or_default(),
        scope: m.scope.unwrap_or_default(),
        credential_root: m.credential_root.unwrap_or_default(),
        invocation_id: m.invocation_id.unwrap_or_default(),
        kind: m.kind.unwrap_or_default(),
        id: m.id.unwrap_or_default(),
        project: m.project.unwrap_or_default(),
        login: m.login.unwrap_or_default(),
        generation: m.generation.map(Into::into).unwrap_or_default(),
    }
}

impl Binding {
    /// `Binding.Validate()`.
    pub fn validate(&self) -> Result<(), String> {
        if self.id.is_empty()
            || self.generation <= 0
            || (self.kind != KIND_FACTORY && self.kind != KIND_TERMINAL)
        {
            return Err(err_denied());
        }
        if self.kind == KIND_TERMINAL && (self.project.is_empty() || self.login.trim().is_empty()) {
            return Err(err_denied());
        }
        Ok(())
    }

    pub fn encode_into(&self, out: &mut String) {
        out.push('{');
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
        if !self.child_id.is_empty() {
            field(out, "child_id", &json::quote(&self.child_id));
        }
        if self.uid != 0 {
            field(out, "uid", &self.uid.to_string());
        }
        if self.gid != 0 {
            field(out, "gid", &self.gid.to_string());
        }
        if !self.scope.is_empty() {
            field(out, "scope", &json::quote(&self.scope));
        }
        if !self.credential_root.is_empty() {
            field(out, "credential_root", &json::quote(&self.credential_root));
        }
        if !self.invocation_id.is_empty() {
            field(out, "invocation_id", &json::quote(&self.invocation_id));
        }
        field(out, "kind", &json::quote(&self.kind));
        field(out, "id", &json::quote(&self.id));
        field(out, "project", &json::quote(&self.project));
        field(out, "login", &json::quote(&self.login));
        field(out, "generation", &self.generation.to_string());
        out.push('}');
    }

    pub fn encode(&self) -> String {
        let mut out = String::new();
        self.encode_into(&mut out);
        out
    }
}
