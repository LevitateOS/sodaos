use crate::json::{self, BytesField, SignedInteger};
use serde::de::{self, MapAccess, Visitor};
use serde::{Deserialize, Deserializer};

use super::{err_denied, parse_rfc3339, KIND_FACTORY, KIND_TERMINAL};

// ---------- identity wire types ----------

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
    /// Decode a binding nested in a tolerant wire object. Whole-lease decoding
    /// continues through the strict `Lease::decode` entry point.
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
                    match k.to_ascii_lowercase().as_str() {
                        "child_id" => {
                            if let Some(v) = map.next_value::<Option<String>>()? {
                                o.child_id = Some(v)
                            }
                        }
                        "uid" => {
                            if let Some(v) = map.next_value::<Option<SignedInteger>>()? {
                                o.uid = Some(v)
                            }
                        }
                        "gid" => {
                            if let Some(v) = map.next_value::<Option<SignedInteger>>()? {
                                o.gid = Some(v)
                            }
                        }
                        "scope" => {
                            if let Some(v) = map.next_value::<Option<String>>()? {
                                o.scope = Some(v)
                            }
                        }
                        "credential_root" => {
                            if let Some(v) = map.next_value::<Option<String>>()? {
                                o.credential_root = Some(v)
                            }
                        }
                        "invocation_id" => {
                            if let Some(v) = map.next_value::<Option<String>>()? {
                                o.invocation_id = Some(v)
                            }
                        }
                        "kind" => {
                            if let Some(v) = map.next_value::<Option<String>>()? {
                                o.kind = Some(v)
                            }
                        }
                        "id" => {
                            if let Some(v) = map.next_value::<Option<String>>()? {
                                o.id = Some(v)
                            }
                        }
                        "project" => {
                            if let Some(v) = map.next_value::<Option<String>>()? {
                                o.project = Some(v)
                            }
                        }
                        "login" => {
                            if let Some(v) = map.next_value::<Option<String>>()? {
                                o.login = Some(v)
                            }
                        }
                        "generation" => {
                            if let Some(v) = map.next_value::<Option<SignedInteger>>()? {
                                o.generation = Some(v)
                            }
                        }
                        _ => {
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
                            ))
                        }
                    }
                }
                Ok(o)
            }
        }
        d.deserialize_map(V)
    }
}

/// `,string` int64 decoding, probed against `encoding/json`: the JSON value
/// must be a quoted string starting with `-` or a digit, holding a plain
/// base-10 integer (leading zeros allowed, no fraction/exponent/space).
pub fn parse_string_i64(raw: &str) -> Option<i64> {
    let b = raw.as_bytes();
    let &first = b.first()?;
    if first != b'-' && !first.is_ascii_digit() {
        return None;
    }
    let digits = if first == b'-' { &b[1..] } else { b };
    if digits.is_empty() || !digits.iter().all(|c| c.is_ascii_digit()) {
        return None;
    }
    raw.parse::<i64>().ok()
}

/// Execution lease (`identity.Lease`). The deadline keeps its raw wire text
/// for byte-exact re-encoding plus a parsed instant for comparisons.
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
    pub deadline_raw: String,
    pub deadline: Option<(i64, u32)>,
    pub grant_id: String,
    pub grant_revision: i64,
    pub binding: Option<Binding>,
}

#[derive(Default)]
struct LeaseWire {
    repository_id: Option<String>,
    provider_id: Option<String>,
    id: Option<String>,
    connection_id: Option<String>,
    generation: Option<SignedInteger>,
    actor_id: Option<String>,
    project_id: Option<String>,
    execution_id: Option<String>,
    kind: Option<String>,
    role: Option<String>,
    deadline: Option<String>,
    grant_id: Option<String>,
    grant_revision: Option<SignedInteger>,
    binding: Option<BindingWire>,
    repository_id_seen: bool,
    actor_id_seen: bool,
    deadline_seen: bool,
}

impl<'de> Deserialize<'de> for LeaseWire {
    fn deserialize<D>(d: D) -> Result<Self, D::Error>
    where
        D: Deserializer<'de>,
    {
        struct V;
        impl<'de> Visitor<'de> for V {
            type Value = LeaseWire;
            fn expecting(&self, f: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
                f.write_str("identity lease")
            }
            fn visit_map<A>(self, mut map: A) -> Result<Self::Value, A::Error>
            where
                A: MapAccess<'de>,
            {
                let mut o = LeaseWire::default();
                while let Some(k) = map.next_key::<String>()? {
                    match k.to_ascii_lowercase().as_str() {
                        "repository_id" => {
                            o.repository_id_seen = true;
                            if let Some(v) = map.next_value::<Option<String>>()? {
                                o.repository_id = Some(v)
                            }
                        }
                        "provider_id" => {
                            if let Some(v) = map.next_value::<Option<String>>()? {
                                o.provider_id = Some(v)
                            }
                        }
                        "id" => {
                            if let Some(v) = map.next_value::<Option<String>>()? {
                                o.id = Some(v)
                            }
                        }
                        "connection_id" => {
                            if let Some(v) = map.next_value::<Option<String>>()? {
                                o.connection_id = Some(v)
                            }
                        }
                        "generation" => {
                            if let Some(v) = map.next_value::<Option<SignedInteger>>()? {
                                o.generation = Some(v)
                            }
                        }
                        "actor_id" => {
                            o.actor_id_seen = true;
                            if let Some(v) = map.next_value::<Option<String>>()? {
                                o.actor_id = Some(v)
                            }
                        }
                        "project_id" => {
                            if let Some(v) = map.next_value::<Option<String>>()? {
                                o.project_id = Some(v)
                            }
                        }
                        "execution_id" => {
                            if let Some(v) = map.next_value::<Option<String>>()? {
                                o.execution_id = Some(v)
                            }
                        }
                        "kind" => {
                            if let Some(v) = map.next_value::<Option<String>>()? {
                                o.kind = Some(v)
                            }
                        }
                        "role" => {
                            if let Some(v) = map.next_value::<Option<String>>()? {
                                o.role = Some(v)
                            }
                        }
                        "deadline" => {
                            o.deadline_seen = true;
                            if let Some(v) = map.next_value::<Option<String>>()? {
                                o.deadline = Some(v)
                            }
                        }
                        "grant_id" => {
                            if let Some(v) = map.next_value::<Option<String>>()? {
                                o.grant_id = Some(v)
                            }
                        }
                        "grant_revision" => {
                            if let Some(v) = map.next_value::<Option<SignedInteger>>()? {
                                o.grant_revision = Some(v)
                            }
                        }
                        "binding" => {
                            if let Some(v) = map.next_value::<Option<BindingWire>>()? {
                                o.binding = Some(v)
                            }
                        }
                        _ => {
                            return Err(de::Error::unknown_field(
                                &k,
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
                            ))
                        }
                    }
                }
                Ok(o)
            }
        }
        d.deserialize_map(V)
    }
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

fn lease_from_wire(m: LeaseWire) -> Result<Lease, String> {
    let repository_id = if m.repository_id_seen {
        let value = m.repository_id.as_deref().unwrap_or_default();
        parse_string_i64(value).ok_or_else(|| "invalid repository_id".to_string())?
    } else {
        0
    };
    let actor_id = if m.actor_id_seen {
        let value = m.actor_id.as_deref().unwrap_or_default();
        parse_string_i64(value).ok_or_else(|| "invalid actor_id".to_string())?
    } else {
        0
    };
    let deadline_raw = m.deadline.unwrap_or_default();
    let deadline = if m.deadline_seen {
        Some(parse_rfc3339(&deadline_raw).ok_or_else(|| "invalid deadline".to_string())?)
    } else {
        None
    };
    Ok(Lease {
        repository_id,
        provider_id: m.provider_id.unwrap_or_default(),
        id: m.id.unwrap_or_default(),
        connection_id: m.connection_id.unwrap_or_default(),
        generation: m.generation.map(Into::into).unwrap_or_default(),
        actor_id,
        project_id: m.project_id.unwrap_or_default(),
        execution_id: m.execution_id.unwrap_or_default(),
        kind: m.kind.unwrap_or_default(),
        role: m.role.unwrap_or_default(),
        deadline_raw,
        deadline,
        grant_id: m.grant_id.unwrap_or_default(),
        grant_revision: m.grant_revision.map(Into::into).unwrap_or_default(),
        binding: m.binding.map(binding_from_wire),
    })
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

impl Lease {
    /// Decode one complete identity lease object using the host JSON profile.
    pub fn decode(body: &[u8]) -> Result<Self, String> {
        let wire: LeaseWire = json::decode_strict_as(body).map_err(|e| e.0)?;
        lease_from_wire(wire)
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
        // Go marshals the zero time as `0001-01-01T00:00:00Z` (probed); an
        // empty raw deadline is that zero time, never an empty string.
        let deadline = if self.deadline_raw.is_empty() {
            "0001-01-01T00:00:00Z"
        } else {
            &self.deadline_raw
        };
        field(out, "deadline", &json::quote(deadline));
        if !self.grant_id.is_empty() {
            field(out, "grant_id", &json::quote(&self.grant_id));
        }
        if self.grant_revision != 0 {
            field(out, "grant_revision", &self.grant_revision.to_string());
        }
        if let Some(binding) = &self.binding {
            let mut nested = String::new();
            binding.encode_into(&mut nested);
            field(out, "binding", &nested);
        }
        out.push('}');
    }

    pub fn encode(&self) -> String {
        let mut out = String::new();
        self.encode_into(&mut out);
        out
    }
}

/// Trusted-process result (`identity.Delivery` / `DeliveryWire`).
#[derive(Debug, Clone, Default, PartialEq, Eq)]
pub struct Delivery {
    pub lease: Lease,
    pub credential: Option<Vec<u8>>,
}

#[derive(Default)]
struct DeliveryWire {
    lease: Option<LeaseWire>,
    credential: Option<BytesField>,
}
impl<'de> Deserialize<'de> for DeliveryWire {
    fn deserialize<D>(d: D) -> Result<Self, D::Error>
    where
        D: Deserializer<'de>,
    {
        struct V;
        impl<'de> Visitor<'de> for V {
            type Value = DeliveryWire;
            fn expecting(&self, f: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
                f.write_str("identity delivery")
            }
            fn visit_map<A>(self, mut map: A) -> Result<Self::Value, A::Error>
            where
                A: MapAccess<'de>,
            {
                let mut o = DeliveryWire::default();
                while let Some(k) = map.next_key::<String>()? {
                    match k.to_ascii_lowercase().as_str() {
                        "lease" => {
                            if let Some(v) = map.next_value::<Option<LeaseWire>>()? {
                                o.lease = Some(v)
                            }
                        }
                        "credential" => {
                            if let Some(value) = map.next_value::<Option<BytesField>>()? {
                                o.credential = Some(value);
                            }
                        }
                        _ => return Err(de::Error::unknown_field(&k, &["lease", "credential"])),
                    }
                }
                Ok(o)
            }
        }
        d.deserialize_map(V)
    }
}

impl Delivery {
    /// Strict decode of one delivery object. `,string` and deadline fields
    /// are re-validated after the shared binder runs.
    pub fn decode(body: &[u8]) -> Result<Self, String> {
        let m: DeliveryWire = json::decode_strict_as(body).map_err(|e| e.0)?;
        let lease = lease_from_wire(m.lease.unwrap_or_default())?;
        let credential = m.credential.map(|value| value.0);
        Ok(Delivery { lease, credential })
    }

    pub fn encode_into(&self, out: &mut String) {
        out.push_str("{\"lease\":");
        self.lease.encode_into(out);
        out.push_str(",\"credential\":");
        match &self.credential {
            None => out.push_str("null"),
            Some(bytes) => out.push_str(&json::quote(&crate::ssh::b64_encode(bytes))),
        }
        out.push('}');
    }

    pub fn encode(&self) -> String {
        let mut out = String::new();
        self.encode_into(&mut out);
        out
    }
}

/// Lease acquisition input (`identity.AcquireRequest`); in-process only,
/// never on the JSON wire. Deadline is Unix seconds plus nanoseconds.
#[derive(Debug, Clone, Default, PartialEq, Eq)]
pub struct AcquireRequest {
    pub repository_id: i64,
    pub provider_id: String,
    pub execution_id: String,
    pub actor_id: i64,
    pub connection_id: String,
    pub project_id: String,
    pub kind: String,
    pub deadline_secs: i64,
    pub deadline_nanos: u32,
    pub role: String,
}
