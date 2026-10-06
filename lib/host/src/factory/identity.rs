use crate::json::{self, BoundMap, Kind, Spec};

use super::FactoryError;

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

    pub fn from_map(m: &BoundMap) -> Self {
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
}

pub(in crate::factory) const BINDING_SPECS: &[Spec] = &[
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

    pub fn from_map(m: &BoundMap) -> Result<Self, FactoryError> {
        const ERR: &str = "invalid run receipt";
        // `,string` decimals: missing and null bind zero like Go; a present
        // value must be a quoted decimal.
        let string_int = |name: &str| -> Result<i64, FactoryError> {
            if !m.contains(name) {
                return Ok(0);
            }
            json::parse_go_int64(&m.take_string(name)).ok_or_else(|| FactoryError::msg(ERR))
        };
        Ok(Lease {
            repository_id: string_int("repository_id")?,
            provider_id: m.take_string("provider_id"),
            id: m.take_string("id"),
            connection_id: m.take_string("connection_id"),
            generation: m.take_i64("generation"),
            actor_id: string_int("actor_id")?,
            project_id: m.take_string("project_id"),
            execution_id: m.take_string("execution_id"),
            kind: m.take_string("kind"),
            role: m.take_string("role"),
            deadline: m.take_string("deadline"),
            grant_id: m.take_string("grant_id"),
            grant_revision: m.take_i64("grant_revision"),
            binding: m.take_opt_map("binding").map(|b| Binding::from_map(&b)),
        })
    }

    /// `leaseWithBinding`: the lease carrying its recorded binding for one
    /// native call.
    pub fn with_binding(&self, binding: &Binding) -> Self {
        let mut leased = self.clone();
        leased.binding = Some(binding.clone());
        leased
    }
}

pub(in crate::factory) const LEASE_SPECS: &[Spec] = &[
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
