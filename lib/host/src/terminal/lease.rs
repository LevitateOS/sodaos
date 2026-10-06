use crate::json::{self, BoundMap, Kind, Spec, Value};

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

const BINDING_SPECS: &[Spec] = &[
    Spec {
        name: "child_id",
        kind: Kind::Str,
    },
    Spec {
        name: "uid",
        kind: Kind::Int,
    },
    Spec {
        name: "gid",
        kind: Kind::Int,
    },
    Spec {
        name: "scope",
        kind: Kind::Str,
    },
    Spec {
        name: "credential_root",
        kind: Kind::Str,
    },
    Spec {
        name: "invocation_id",
        kind: Kind::Str,
    },
    Spec {
        name: "kind",
        kind: Kind::Str,
    },
    Spec {
        name: "id",
        kind: Kind::Str,
    },
    Spec {
        name: "project",
        kind: Kind::Str,
    },
    Spec {
        name: "login",
        kind: Kind::Str,
    },
    Spec {
        name: "generation",
        kind: Kind::I64,
    },
];

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

const LEASE_SPECS: &[Spec] = &[
    Spec {
        name: "repository_id",
        kind: Kind::Str,
    },
    Spec {
        name: "provider_id",
        kind: Kind::Str,
    },
    Spec {
        name: "id",
        kind: Kind::Str,
    },
    Spec {
        name: "connection_id",
        kind: Kind::Str,
    },
    Spec {
        name: "generation",
        kind: Kind::I64,
    },
    Spec {
        name: "actor_id",
        kind: Kind::Str,
    },
    Spec {
        name: "project_id",
        kind: Kind::Str,
    },
    Spec {
        name: "execution_id",
        kind: Kind::Str,
    },
    Spec {
        name: "kind",
        kind: Kind::Str,
    },
    Spec {
        name: "role",
        kind: Kind::Str,
    },
    Spec {
        name: "deadline",
        kind: Kind::Str,
    },
    Spec {
        name: "grant_id",
        kind: Kind::Str,
    },
    Spec {
        name: "grant_revision",
        kind: Kind::I64,
    },
    Spec {
        name: "binding",
        kind: Kind::OptObject {
            go_type: "*identity.Binding",
            struct_name: "Binding",
            specs: BINDING_SPECS,
        },
    },
];

fn binding_from_map(m: &BoundMap) -> Binding {
    Binding {
        child_id: m.take_string("child_id"),
        uid: m.take_i64("uid"),
        gid: m.take_i64("gid"),
        scope: m.take_string("scope"),
        credential_root: m.take_string("credential_root"),
        invocation_id: m.take_string("invocation_id"),
        kind: m.take_string("kind"),
        id: m.take_string("id"),
        project: m.take_string("project"),
        login: m.take_string("login"),
        generation: m.take_i64("generation"),
    }
}

fn lease_from_map(m: &BoundMap) -> Result<Lease, String> {
    let repository_id = if m.contains("repository_id") {
        parse_string_i64(&m.take_string("repository_id"))
            .ok_or_else(|| "invalid repository_id".to_string())?
    } else {
        0
    };
    let actor_id = if m.contains("actor_id") {
        parse_string_i64(&m.take_string("actor_id"))
            .ok_or_else(|| "invalid actor_id".to_string())?
    } else {
        0
    };
    let deadline_raw = m.take_string("deadline");
    let deadline = if m.contains("deadline") {
        Some(parse_rfc3339(&deadline_raw).ok_or_else(|| "invalid deadline".to_string())?)
    } else {
        None
    };
    Ok(Lease {
        repository_id,
        provider_id: m.take_string("provider_id"),
        id: m.take_string("id"),
        connection_id: m.take_string("connection_id"),
        generation: m.take_i64("generation"),
        actor_id,
        project_id: m.take_string("project_id"),
        execution_id: m.take_string("execution_id"),
        kind: m.take_string("kind"),
        role: m.take_string("role"),
        deadline_raw,
        deadline,
        grant_id: m.take_string("grant_id"),
        grant_revision: m.take_i64("grant_revision"),
        binding: m.take_opt_map("binding").map(|b| binding_from_map(&b)),
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
    /// Strict decode of one lease object.
    pub fn decode_value(v: &Value) -> Result<Self, String> {
        let m = json::bind_struct(v, "Lease", &[], LEASE_SPECS, false).map_err(|e| e.0)?;
        lease_from_map(&m)
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

const DELIVERY_SPECS: &[Spec] = &[
    Spec {
        name: "lease",
        kind: Kind::Object {
            go_type: "identity.Lease",
            struct_name: "Lease",
            specs: LEASE_SPECS,
        },
    },
    Spec {
        name: "credential",
        kind: Kind::Bytes,
    },
];

impl Delivery {
    /// Strict decode of one delivery object. `,string` and deadline fields
    /// are re-validated after the shared binder runs.
    pub fn decode(body: &[u8]) -> Result<Self, String> {
        let v = json::decode_strict(body).map_err(|e| e.0)?;
        Self::decode_value(&v)
    }

    pub fn decode_value(v: &Value) -> Result<Self, String> {
        let m = json::bind_root(v, "DeliveryWire", DELIVERY_SPECS, false).map_err(|e| e.0)?;
        let lease = lease_from_map(&m.take_map("lease"))?;
        let credential = if m.contains("credential") {
            Some(m.take_bytes("credential"))
        } else {
            None
        };
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
