use crate::json::{self, parse_go_int64};
use serde::de::{self, MapAccess, Visitor};
use serde::Deserialize;
use std::fmt;

/// `identity.Binding`: the recorded native process boundary.
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

impl Binding {
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
}

macro_rules! typed_object {
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

typed_object!(Binding, "an identity binding object", {
    child_id => "child_id": String,
    uid => "uid": json::SignedInteger,
    gid => "gid": json::SignedInteger,
    scope => "scope": String,
    credential_root => "credential_root": String,
    invocation_id => "invocation_id": String,
    kind => "kind": String,
    id => "id": String,
    project => "project": String,
    login => "login": String,
    generation => "generation": json::SignedInteger,
});

/// `identity.Lease`: the broker execution identity for one run.
#[derive(Debug, Clone, Default, PartialEq, Eq)]
pub struct Lease {
    pub repository_id: i64,
    pub provider_id: String,
    pub id: String,
    pub connection_id: String,
    pub generation: i64,
    pub actor_id: i64,
    pub project_id: String,
    pub execution_id: String,
    pub kind: String,
    pub role: String,
    pub deadline: String,
    pub grant_id: String,
    pub grant_revision: i64,
    pub binding: Option<Binding>,
}

impl Lease {
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
        if self.repository_id != 0 {
            field(
                out,
                "repository_id",
                &json::quote(&self.repository_id.to_string()),
            );
        }
        field(out, "provider_id", &json::quote(&self.provider_id));
        field(out, "id", &json::quote(&self.id));
        field(out, "connection_id", &json::quote(&self.connection_id));
        field(out, "generation", &self.generation.to_string());
        field(out, "actor_id", &json::quote(&self.actor_id.to_string()));
        field(out, "project_id", &json::quote(&self.project_id));
        field(out, "execution_id", &json::quote(&self.execution_id));
        field(out, "kind", &json::quote(&self.kind));
        if !self.role.is_empty() {
            field(out, "role", &json::quote(&self.role));
        }
        field(out, "deadline", &json::quote(&self.deadline));
        if !self.grant_id.is_empty() {
            field(out, "grant_id", &json::quote(&self.grant_id));
        }
        if self.grant_revision != 0 {
            field(out, "grant_revision", &self.grant_revision.to_string());
        }
        if let Some(b) = &self.binding {
            if !first {
                out.push(',');
            }
            out.push_str("\"binding\":");
            b.encode_into(out);
        }
        out.push('}');
    }

    /// `leaseWithBinding`: the lease carrying its recorded binding for one
    /// native call.
    pub fn with_binding(&self, binding: &Binding) -> Self {
        let mut leased = self.clone();
        leased.binding = Some(binding.clone());
        leased
    }
}

impl<'de> Deserialize<'de> for Lease {
    fn deserialize<D>(deserializer: D) -> Result<Self, D::Error>
    where
        D: serde::Deserializer<'de>,
    {
        struct LeaseVisitor;
        impl<'de> Visitor<'de> for LeaseVisitor {
            type Value = Lease;
            fn expecting(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
                f.write_str("an identity lease object")
            }
            fn visit_map<A>(self, mut map: A) -> Result<Self::Value, A::Error>
            where
                A: MapAccess<'de>,
            {
                let mut out = Lease::default();
                while let Some(key) = map.next_key::<String>()? {
                    if key.eq_ignore_ascii_case("repository_id") {
                        if let Some(v) = map.next_value::<Option<String>>()? {
                            out.repository_id = parse_go_int64(&v)
                                .ok_or_else(|| de::Error::custom("invalid repository_id"))?;
                        }
                        continue;
                    }
                    if key.eq_ignore_ascii_case("actor_id") {
                        if let Some(v) = map.next_value::<Option<String>>()? {
                            out.actor_id = parse_go_int64(&v)
                                .ok_or_else(|| de::Error::custom("invalid actor_id"))?;
                        }
                        continue;
                    }
                    if key.eq_ignore_ascii_case("provider_id") {
                        if let Some(v) = map.next_value::<Option<String>>()? {
                            out.provider_id = v;
                        }
                        continue;
                    }
                    if key.eq_ignore_ascii_case("id") {
                        if let Some(v) = map.next_value::<Option<String>>()? {
                            out.id = v;
                        }
                        continue;
                    }
                    if key.eq_ignore_ascii_case("connection_id") {
                        if let Some(v) = map.next_value::<Option<String>>()? {
                            out.connection_id = v;
                        }
                        continue;
                    }
                    if key.eq_ignore_ascii_case("generation") {
                        if let Some(v) = map.next_value::<Option<json::SignedInteger>>()? {
                            out.generation = v.0;
                        }
                        continue;
                    }
                    if key.eq_ignore_ascii_case("project_id") {
                        if let Some(v) = map.next_value::<Option<String>>()? {
                            out.project_id = v;
                        }
                        continue;
                    }
                    if key.eq_ignore_ascii_case("execution_id") {
                        if let Some(v) = map.next_value::<Option<String>>()? {
                            out.execution_id = v;
                        }
                        continue;
                    }
                    if key.eq_ignore_ascii_case("kind") {
                        if let Some(v) = map.next_value::<Option<String>>()? {
                            out.kind = v;
                        }
                        continue;
                    }
                    if key.eq_ignore_ascii_case("role") {
                        if let Some(v) = map.next_value::<Option<String>>()? {
                            out.role = v;
                        }
                        continue;
                    }
                    if key.eq_ignore_ascii_case("deadline") {
                        if let Some(v) = map.next_value::<Option<String>>()? {
                            out.deadline = v;
                        }
                        continue;
                    }
                    if key.eq_ignore_ascii_case("grant_id") {
                        if let Some(v) = map.next_value::<Option<String>>()? {
                            out.grant_id = v;
                        }
                        continue;
                    }
                    if key.eq_ignore_ascii_case("grant_revision") {
                        if let Some(v) = map.next_value::<Option<json::SignedInteger>>()? {
                            out.grant_revision = v.0;
                        }
                        continue;
                    }
                    if key.eq_ignore_ascii_case("binding") {
                        if let Some(v) = map.next_value::<Option<Binding>>()? {
                            out.binding = Some(v);
                        }
                        continue;
                    }
                    return Err(de::Error::unknown_field(
                        &key,
                        &[
                            "repository_id",
                            "provider_id",
                            "id",
                            "connection_id",
                            "generation",
                            "actor_id",
                            "project_id",
                            "execution_id",
                            "kind",
                            "role",
                            "deadline",
                            "grant_id",
                            "grant_revision",
                            "binding",
                        ],
                    ));
                }
                Ok(out)
            }
        }
        deserializer.deserialize_map(LeaseVisitor)
    }
}

/// `identity.AcquireRequest` as built by `acquireRunLease` (the repository
/// field stays unset on this path, so it is not mirrored).
#[derive(Debug, Clone, Default, PartialEq, Eq)]
pub struct AcquireRequest {
    pub provider_id: String,
    pub execution_id: String,
    pub actor_id: i64,
    pub connection_id: String,
    pub project_id: String,
    pub kind: String,
    pub deadline: String,
    pub role: String,
}
