use crate::daemon::backend::BackendError;
use crate::json;
use serde::de::{self, MapAccess, Visitor};
use serde::{Deserialize, Deserializer};
use std::time::{Duration, Instant, SystemTime, UNIX_EPOCH};

/// Clip one lifecycle call to its remaining whole-operation budget,
/// capped at `cap_secs`. An expired call is invalid; the mux already
/// rejects it, so this is defense in depth at the effect boundary.
fn lifecycle_deadline(
    call: &crate::tcontrol_wire::LifecycleCall,
    cap_secs: u64,
) -> Result<Instant, BackendError> {
    let now_secs = SystemTime::now()
        .duration_since(UNIX_EPOCH)
        .map(|elapsed| elapsed.as_secs() as i64)
        .unwrap_or(0);
    let remaining = call.deadline_unix.saturating_sub(now_secs);
    if remaining < 1 {
        return Err(BackendError::Invalid);
    }
    let bound = (remaining as u64).min(cap_secs);
    Ok(Instant::now() + Duration::from_secs(bound))
}

fn lifecycle_remaining(
    call: &crate::tcontrol_wire::LifecycleCall,
) -> Result<Instant, BackendError> {
    lifecycle_deadline(call, crate::tailnet_lifecycle::ENSURE_TIMEOUT_SECS)
}

fn lifecycle_observe_deadline(
    call: &crate::tcontrol_wire::LifecycleCall,
) -> Result<Instant, BackendError> {
    lifecycle_deadline(call, 10)
}

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
    pub(super) fn admit_tailnet_start(
        &self,
        deadline: Instant,
    ) -> Result<
        (
            crate::project::admission::AdmissionGuard,
            crate::tailnet_admission::StartGuard,
        ),
        BackendError,
    > {
        let maintenance =
            crate::project::admission::admit(None).map_err(|_| BackendError::Unavailable)?;
        let store = crate::tailnet_admission::GrantStore::default();
        if store.read().is_err() {
            return Err(BackendError::Unavailable);
        }
        let starts = store
            .start_lock(deadline)
            .map_err(|_| BackendError::Unavailable)?;
        store
            .admit(&self.companion.image, deadline)
            .map_err(|_| BackendError::Unavailable)?;
        Ok((maintenance, starts))
    }

    /// Lifecycle prepare: strict-decode the activate tuple, clip the
    /// importer run to the call's remaining whole-operation budget, verify
    /// the selected dependency locally, then report facts. Prepare never
    /// opens admission and never starts a daemon.
    pub(super) fn tailnet_lifecycle_prepare(
        &self,
        body: &[u8],
        release_payload: &str,
    ) -> Result<Vec<u8>, BackendError> {
        let call =
            crate::tcontrol_wire::decode_lifecycle_call(body).map_err(|_| BackendError::Invalid)?;
        if call.operation != crate::tcontrol_wire::LIFECYCLE_ACTIVATE {
            return Err(BackendError::Invalid);
        }
        let deadline = lifecycle_remaining(&call)?;
        if release_payload.is_empty() || !release_payload.starts_with('/') {
            return Err(BackendError::Unavailable);
        }
        let _maintenance =
            crate::project::admission::admit(None).map_err(|_| BackendError::Unavailable)?;
        crate::tailnet_lifecycle::ensure_tailnet_dependency(
            crate::tailnet_lifecycle::IMAGE_IMPORT_BINARY,
            release_payload,
            deadline,
        )
        .map_err(|_| BackendError::Unavailable)?;
        let observe = lifecycle_observe_deadline(&call)?;
        let runtime = crate::tailnet_lifecycle::Runtime {
            host: &self.tailnet_host,
            companion: &self.companion,
            payload: release_payload,
        };
        let mut observation = runtime.observe(&call, observe);
        observation.admission_ready = false;
        Ok(crate::tcontrol_wire::encode_lifecycle_observation(&observation).into_bytes())
    }

    /// Lifecycle observe/activate/deactivate report: strict-decode the
    /// tuple and report facts without claiming them. T01 performs no
    /// container effects here; T02 hangs validated apply, enumeration and
    /// the operational grant on this same wire.
    pub(super) fn tailnet_lifecycle_report(&self, body: &[u8]) -> Result<Vec<u8>, BackendError> {
        let call =
            crate::tcontrol_wire::decode_lifecycle_call(body).map_err(|_| BackendError::Invalid)?;
        let operation = call.operation.as_str();
        if operation != crate::tcontrol_wire::LIFECYCLE_OBSERVE
            && operation != crate::tcontrol_wire::LIFECYCLE_ACTIVATE
            && operation != crate::tcontrol_wire::LIFECYCLE_DEACTIVATE
        {
            return Err(BackendError::Invalid);
        }
        let cap = match operation {
            "observe" => 10,
            "activate" => 300,
            _ => 120,
        };
        let deadline = lifecycle_deadline(&call, cap)?;
        let runtime = crate::tailnet_lifecycle::Runtime {
            host: &self.tailnet_host,
            companion: &self.companion,
            payload: &self.release_payload,
        };
        Ok(
            crate::tcontrol_wire::encode_lifecycle_observation(&runtime.apply(&call, deadline))
                .into_bytes(),
        )
    }

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
        let _admission = if matches!(req.action.as_str(), "enable" | "retry") {
            Some(self.admit_tailnet_start(deadline)?)
        } else {
            None
        };
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
