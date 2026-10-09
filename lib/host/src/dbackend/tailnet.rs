use crate::daemon::backend::BackendError;
use crate::json;
use serde::de::{self, MapAccess, Visitor};
use serde::{Deserialize, Deserializer};
use std::time::Instant;

use super::{decode_empty_or_null, DaemonBackend};

#[cfg(test)]
mod tests;

/// Tailnet decode failure: Go's `decodeTailnetBody` reports `ErrInvalid`
/// (400), unlike the native 500.
pub(super) fn decode_tailnet_empty(body: &[u8]) -> Result<(), BackendError> {
    decode_empty_or_null(body).map_err(|_| BackendError::Invalid)
}

/// Lane T error substrings to wire mapping. The mux renders tailnet
/// `ExportStale` as 409 and `ExportCandidate` as 422, so conflict and
/// unsupported reuse those variants behind this subsystem only.
pub(super) fn map_tailnet_err(err: String) -> BackendError {
    if err.contains("invalid request") {
        return BackendError::Invalid;
    }
    if err.contains("conflict") {
        return BackendError::ExportStale;
    }
    if err.contains("unsupported") {
        return BackendError::ExportCandidate;
    }
    if err.contains("unavailable") {
        return BackendError::Unavailable;
    }
    BackendError::Internal
}

/// `^[0-9a-f]{32}$` or the `"0"` unset marker (Go `validRevision`).
fn valid_revision(v: &str) -> bool {
    v == "0"
        || (v.len() == 32
            && v.bytes()
                .all(|b| b.is_ascii_hexdigit() && !b.is_ascii_uppercase()))
}

#[derive(Default)]
struct TailnetProjectWire {
    project: Option<String>,
    action: Option<String>,
    revision: Option<String>,
    binding: Option<String>,
    confirm_id: Option<String>,
}
impl<'de> Deserialize<'de> for TailnetProjectWire {
    fn deserialize<D>(d: D) -> Result<Self, D::Error>
    where
        D: Deserializer<'de>,
    {
        struct V;
        impl<'de> Visitor<'de> for V {
            type Value = TailnetProjectWire;
            fn expecting(&self, f: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
                f.write_str("Tailnet project request")
            }
            fn visit_map<A>(self, mut map: A) -> Result<Self::Value, A::Error>
            where
                A: MapAccess<'de>,
            {
                let mut o = TailnetProjectWire::default();
                while let Some(k) = map.next_key::<String>()? {
                    match k.to_ascii_lowercase().as_str() {
                        "project" => {
                            if let Some(v) = map.next_value::<Option<String>>()? {
                                o.project = Some(v)
                            }
                        }
                        "action" => {
                            if let Some(v) = map.next_value::<Option<String>>()? {
                                o.action = Some(v)
                            }
                        }
                        "revision" => {
                            if let Some(v) = map.next_value::<Option<String>>()? {
                                o.revision = Some(v)
                            }
                        }
                        "binding" => {
                            if let Some(v) = map.next_value::<Option<String>>()? {
                                o.binding = Some(v)
                            }
                        }
                        "confirm_id" => {
                            if let Some(v) = map.next_value::<Option<String>>()? {
                                o.confirm_id = Some(v)
                            }
                        }
                        _ => {
                            return Err(de::Error::unknown_field(
                                &k,
                                &["project", "action", "revision", "binding", "confirm_id"],
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

/// Strict decode of one project/policy request (Go `ProjectRequest` shape).
fn decode_project_request(
    body: &[u8],
) -> Result<crate::tailnet_domain::ProjectRequest, BackendError> {
    let m: TailnetProjectWire = json::decode_strict_as(body).map_err(|_| BackendError::Invalid)?;
    Ok(crate::tailnet_domain::ProjectRequest {
        project: m.project.unwrap_or_default(),
        action: m.action.unwrap_or_default(),
        revision: m.revision.unwrap_or_default(),
        binding: m.binding.unwrap_or_default(),
        confirm_id: m.confirm_id.unwrap_or_default(),
    })
}

/// Go `ProjectRequest.Validate` + `validateProjectMutation`, verbatim.
fn validate_project_request(
    req: &crate::tailnet_domain::ProjectRequest,
) -> Result<(), BackendError> {
    if !crate::tailnet_domain::valid_project_id(&req.project) {
        return Err(BackendError::Invalid);
    }
    if req.action == "inspect" {
        if !req.revision.is_empty() || !req.binding.is_empty() || !req.confirm_id.is_empty() {
            return Err(BackendError::Invalid);
        }
        return Ok(());
    }
    if !valid_revision(&req.revision) || req.confirm_id != req.project {
        return Err(BackendError::Invalid);
    }
    match req.action.as_str() {
        "disable" => {
            if !req.binding.is_empty() {
                return Err(BackendError::Invalid);
            }
        }
        "enable" | "retry" => {
            if !(req.binding.len() == 32
                && req
                    .binding
                    .bytes()
                    .all(|b| b.is_ascii_hexdigit() && !b.is_ascii_uppercase()))
            {
                return Err(BackendError::Invalid);
            }
        }
        _ => return Err(BackendError::Invalid),
    }
    Ok(())
}

/// Go `ProjectView` JSON render with identical `omitempty` behavior.
fn encode_project_view(view: &crate::tailnet_domain::ProjectView) -> String {
    let mut out = String::from("{");
    let mut first = true;
    let field = |out: &mut String, first: &mut bool, name: &str, value: String| {
        if !*first {
            out.push(',');
        }
        *first = false;
        out.push_str(&json::quote(name));
        out.push(':');
        out.push_str(&value);
    };
    if !view.available_binding.is_empty() {
        field(
            &mut out,
            &mut first,
            "available_binding",
            json::quote(&view.available_binding),
        );
    }
    if !view.available_network.is_empty() {
        field(
            &mut out,
            &mut first,
            "available_network",
            json::quote(&view.available_network),
        );
    }
    if !view.addresses.is_empty() {
        let list = view
            .addresses
            .iter()
            .map(|a| json::quote(a))
            .collect::<Vec<_>>()
            .join(",");
        field(&mut out, &mut first, "addresses", format!("[{list}]"));
    }
    if !view.dns_name.is_empty() {
        field(
            &mut out,
            &mut first,
            "dns_name",
            json::quote(&view.dns_name),
        );
    }
    field(
        &mut out,
        &mut first,
        "saved",
        if view.saved {
            "true".to_string()
        } else {
            "false".to_string()
        },
    );
    field(&mut out, &mut first, "project", json::quote(&view.project));
    field(
        &mut out,
        &mut first,
        "revision",
        json::quote(&view.revision),
    );
    field(&mut out, &mut first, "binding", json::quote(&view.binding));
    field(
        &mut out,
        &mut first,
        "enabled",
        if view.enabled {
            "true".to_string()
        } else {
            "false".to_string()
        },
    );
    field(&mut out, &mut first, "state", json::quote(&view.state));
    field(&mut out, &mut first, "outcome", json::quote(&view.outcome));
    out.push('}');
    out
}

impl DaemonBackend {
    /// `executeTailnetProjectOrPolicy`: strict decode, validation, the
    /// container-identity fence (resolve before AND after; mismatch or
    /// error is unconfirmed), then the observation.
    pub(super) fn tailnet_project_or_policy(
        &self,
        action: &str,
        body: &[u8],
        deadline: Instant,
    ) -> Result<Vec<u8>, BackendError> {
        let req = decode_project_request(body)?;
        validate_project_request(&req)?;
        if action == "policy" && req.action != "inspect" {
            return Err(BackendError::Invalid);
        }
        let cid = self
            .project
            .project_container(&req.project, false, deadline)
            .map_err(|_| BackendError::Internal)?;
        let out = if action == "policy" {
            self.tailnet
                .project(&req, &cid, deadline)
                .map_err(map_tailnet_err)?
        } else {
            self.companion
                .observe_project_tailnet(&req, &cid, deadline)
                .map_err(map_tailnet_err)?
        };
        let after = self
            .project
            .project_container(&req.project, false, deadline)
            .map_err(|_| BackendError::Internal)?;
        if after != cid {
            return Err(BackendError::Internal);
        }
        Ok(encode_project_view(&out).into_bytes())
    }
}
