use crate::json;
use crate::preparation;

use super::{valid_harness_family, valid_harness_version};

/// One observed output slice with its cursor placement, mirroring
/// `terminal.FactoryCodexOutputSlice`.
#[derive(Debug, Clone, Default, PartialEq, Eq)]
pub struct OutputSlice {
    pub data: Vec<u8>,
    pub total: i64,
    pub offset: i64,
    pub truncated: bool,
    pub gap: bool,
}

// ---------- request and response records ----------

/// `project.FactoryState`: the observed run outcome for one identity.
#[derive(Debug, Clone, Default, PartialEq, Eq)]
pub struct FactoryState {
    pub exit_code: Option<i64>,
    pub generation: i64,
    pub uid: i64,
    pub gid: i64,
    pub credential_returned: bool,
    pub live: bool,
    pub delivered: bool,
    pub id: String,
    pub project: String,
    pub role: String,
    pub phase: String,
    pub container: String,
    pub unit: String,
    pub invocation: String,
    pub login: String,
    pub lease_id: String,
    pub output: String,
    pub retirement: String,
    pub reason: String,
}

impl FactoryState {
    /// `encoding/json` field order with `omitempty` honored.
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
        field(&mut out, "generation", &self.generation.to_string());
        field(&mut out, "uid", &self.uid.to_string());
        field(&mut out, "gid", &self.gid.to_string());
        field(
            &mut out,
            "credential_returned",
            if self.credential_returned {
                "true"
            } else {
                "false"
            },
        );
        field(&mut out, "live", if self.live { "true" } else { "false" });
        field(
            &mut out,
            "delivered",
            if self.delivered { "true" } else { "false" },
        );
        field(&mut out, "id", &json::quote(&self.id));
        field(&mut out, "project", &json::quote(&self.project));
        field(&mut out, "role", &json::quote(&self.role));
        field(&mut out, "phase", &json::quote(&self.phase));
        field(&mut out, "container", &json::quote(&self.container));
        field(&mut out, "unit", &json::quote(&self.unit));
        field(&mut out, "invocation", &json::quote(&self.invocation));
        field(&mut out, "login", &json::quote(&self.login));
        if !self.lease_id.is_empty() {
            field(&mut out, "lease_id", &json::quote(&self.lease_id));
        }
        if !self.output.is_empty() {
            field(&mut out, "output", &json::quote(&self.output));
        }
        if !self.retirement.is_empty() {
            field(&mut out, "retirement", &json::quote(&self.retirement));
        }
        if !self.reason.is_empty() {
            field(&mut out, "reason", &json::quote(&self.reason));
        }
        out.push('}');
        out
    }
}

/// `project.FactoryHarnessPin`: the staged-harness identity. The daemon
/// route fills `image` from its configuration after calling `harness_pin`,
/// exactly like the Go dispatch.
#[derive(Debug, Clone, Default, PartialEq, Eq)]
pub struct FactoryHarnessPin {
    pub harness: String,
    pub version: String,
    pub sha256: String,
    pub image: String,
}

impl FactoryHarnessPin {
    pub fn validate(&self) -> Result<(), String> {
        if !valid_harness_family(&self.harness) || !valid_harness_version(&self.version) {
            return Err("unsupported factory harness".to_string());
        }
        if !preparation::valid_digest(&self.sha256) {
            return Err("invalid harness pin".to_string());
        }
        match self.image.strip_prefix("sha256:") {
            Some(value) if preparation::valid_digest(value) => Ok(()),
            _ => Err("execution image is not pinned".to_string()),
        }
    }

    pub fn encode(&self) -> String {
        let mut out = String::from("{\"harness\":");
        out.push_str(&json::quote(&self.harness));
        out.push_str(",\"version\":");
        out.push_str(&json::quote(&self.version));
        out.push_str(",\"sha256\":");
        out.push_str(&json::quote(&self.sha256));
        out.push_str(",\"image\":");
        out.push_str(&json::quote(&self.image));
        out.push('}');
        out
    }
}
