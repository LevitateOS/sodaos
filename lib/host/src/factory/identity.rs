use crate::json::{self, parse_go_int64};
use crate::terminal::Binding;
use serde::de::{self, MapAccess, Visitor};
use serde::Deserialize;
use std::fmt;

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

fn visit_lease_int_id<'de, A>(map: &mut A, out: &mut Lease, key: &str) -> Result<bool, A::Error>
where
    A: MapAccess<'de>,
{
    if key.eq_ignore_ascii_case("repository_id") {
        if let Some(v) = map.next_value::<Option<String>>()? {
            out.repository_id =
                parse_go_int64(&v).ok_or_else(|| de::Error::custom("invalid repository_id"))?;
        }
        return Ok(true);
    }
    if key.eq_ignore_ascii_case("actor_id") {
        if let Some(v) = map.next_value::<Option<String>>()? {
            out.actor_id =
                parse_go_int64(&v).ok_or_else(|| de::Error::custom("invalid actor_id"))?;
        }
        return Ok(true);
    }
    Ok(false)
}

fn visit_lease_counter<'de, A>(map: &mut A, out: &mut Lease, key: &str) -> Result<bool, A::Error>
where
    A: MapAccess<'de>,
{
    if key.eq_ignore_ascii_case("generation") {
        if let Some(v) = map.next_value::<Option<json::SignedInteger>>()? {
            out.generation = v.0;
        }
        return Ok(true);
    }
    if key.eq_ignore_ascii_case("grant_revision") {
        if let Some(v) = map.next_value::<Option<json::SignedInteger>>()? {
            out.grant_revision = v.0;
        }
        return Ok(true);
    }
    Ok(false)
}

fn visit_lease_text<'de, A>(map: &mut A, out: &mut Lease, key: &str) -> Result<bool, A::Error>
where
    A: MapAccess<'de>,
{
    let slot = if key.eq_ignore_ascii_case("provider_id") {
        Some(&mut out.provider_id)
    } else if key.eq_ignore_ascii_case("id") {
        Some(&mut out.id)
    } else if key.eq_ignore_ascii_case("connection_id") {
        Some(&mut out.connection_id)
    } else if key.eq_ignore_ascii_case("project_id") {
        Some(&mut out.project_id)
    } else if key.eq_ignore_ascii_case("execution_id") {
        Some(&mut out.execution_id)
    } else {
        None
    };
    if let Some(slot) = slot {
        if let Some(v) = map.next_value::<Option<String>>()? {
            *slot = v;
        }
        return Ok(true);
    }
    let slot = if key.eq_ignore_ascii_case("kind") {
        Some(&mut out.kind)
    } else if key.eq_ignore_ascii_case("role") {
        Some(&mut out.role)
    } else if key.eq_ignore_ascii_case("deadline") {
        Some(&mut out.deadline)
    } else if key.eq_ignore_ascii_case("grant_id") {
        Some(&mut out.grant_id)
    } else {
        None
    };
    if let Some(slot) = slot {
        if let Some(v) = map.next_value::<Option<String>>()? {
            *slot = v;
        }
        return Ok(true);
    }
    Ok(false)
}

fn visit_lease_binding<'de, A>(map: &mut A, out: &mut Lease, key: &str) -> Result<bool, A::Error>
where
    A: MapAccess<'de>,
{
    if key.eq_ignore_ascii_case("binding") {
        if let Some(v) = map.next_value::<Option<Binding>>()? {
            out.binding = Some(v);
        }
        return Ok(true);
    }
    Ok(false)
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
                    if visit_lease_int_id(&mut map, &mut out, &key)? {
                        continue;
                    }
                    if visit_lease_counter(&mut map, &mut out, &key)? {
                        continue;
                    }
                    if visit_lease_text(&mut map, &mut out, &key)? {
                        continue;
                    }
                    if visit_lease_binding(&mut map, &mut out, &key)? {
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
