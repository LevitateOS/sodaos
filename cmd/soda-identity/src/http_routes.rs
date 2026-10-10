// Admin/runtime route dispatch, extracted from http.rs (A05.M).

// Unix HTTP service, mirroring internal/identity/control/http.go over the
// admin and runtime listeners: POST-only admission, strict bodies, path
// dispatch and the fixed error codes. Connections close after each
// response; the Go client transparently redials.
use crate::control::Controller;
use crate::http_wire::{error_response, success_response, HttpRequest};
use crate::strict;
use crate::wire::{DeliveryWire, Error, ErrorKind, Request};
use bytes::Bytes;
use http_body_util::Full;
use hyper::Response;
use std::time::Duration;

pub(crate) const MAX_BODY: usize = 512 << 10;
pub(crate) const HEADER_TIMEOUT: Duration = Duration::from_secs(5);

pub(crate) fn dispatch(
    controller: &Controller,
    request: HttpRequest,
    body: Vec<u8>,
    runtime_allowed: bool,
) -> Response<Full<Bytes>> {
    if request.method != "POST" || !request.query.is_empty() || !request.origin.is_empty() {
        return error_response(403, "denied");
    }
    let input: Request = match strict::decode_typed(&body, MAX_BODY) {
        Ok(input) => input,
        Err(_) => return error_response(400, "invalid request"),
    };
    let result = route(controller, &request.path, &input, runtime_allowed);
    match result {
        Ok(output) => success_response(output),
        Err(err) => {
            let (status, code) = match err.kind() {
                ErrorKind::Denied => (403, "denied"),
                ErrorKind::NotFound => (404, "missing"),
                ErrorKind::Busy => (409, "busy"),
                ErrorKind::Stale => (409, "stale"),
                ErrorKind::Uncertain => (409, "reauth"),
                ErrorKind::Internal => (500, "unavailable"),
            };
            error_response(status, code)
        }
    }
}

// read_request returns the head plus body separately to keep header limits
// distinct from body limits.
fn route(
    controller: &Controller,
    path: &str,
    input: &Request,
    runtime_allowed: bool,
) -> Result<Option<Vec<u8>>, Error> {
    match path {
        "/connections" => Ok(Some(controller.connections(input.owner_id)?)),
        "/available" => Ok(Some(
            controller.available(input.owner_id, &input.project_id)?,
        )),
        "/revoke" => {
            controller.revoke(input.owner_id, &input.id)?;
            Ok(None)
        }
        "/enrollment/start" => {
            let enrollment =
                controller.start_enrollment(input.owner_id, &input.provider_id, &input.label)?;
            Ok(Some(serde_json::to_vec(&enrollment)?))
        }
        "/enrollment/read" => Ok(Some(serde_json::to_vec(
            &controller.enrollment(input.owner_id, &input.id)?,
        )?)),
        "/enrollment/cancel" => {
            controller.cancel_enrollment(input.owner_id, &input.id)?;
            Ok(None)
        }
        "/grants" => Ok(Some(controller.grants(input.owner_id, &input.id)?)),
        "/grant/create" => {
            let Some(grant) = &input.grant else {
                return Err(Error::denied("identity authority denied"));
            };
            Ok(Some(serde_json::to_vec(
                &controller.create_grant(input.owner_id, grant)?,
            )?))
        }
        "/grant/revoke" => {
            controller.revoke_grant(input.owner_id, &input.id)?;
            Ok(None)
        }
        "/leases" => Ok(Some(controller.leases(input.owner_id, &input.id)?)),
        "/lease/end" => {
            controller.end_lease(input.owner_id, &input.id)?;
            Ok(None)
        }
        "/execution/get" => Ok(Some(serde_json::to_vec(
            &controller.get_execution(&input.kind, &input.execution_id)?,
        )?)),
        "/execution/close" => {
            controller.close_execution(&input.kind, &input.execution_id)?;
            Ok(None)
        }
        _ => {
            if !runtime_allowed {
                return Err(Error::denied("identity authority denied"));
            }
            route_runtime(controller, path, input)
        }
    }
}

pub(crate) fn route_runtime(
    controller: &Controller,
    path: &str,
    input: &Request,
) -> Result<Option<Vec<u8>>, Error> {
    match path {
        "/acquire" => {
            let Some(acquire) = &input.acquire else {
                return Err(Error::denied("identity authority denied"));
            };
            Ok(Some(serde_json::to_vec(&controller.acquire(acquire)?)?))
        }
        "/register" => {
            let Some(binding) = &input.binding else {
                return Err(Error::denied("identity authority denied"));
            };
            let (lease, credential) = controller.register(&input.id, binding)?;
            Ok(Some(serde_json::to_vec(&DeliveryWire {
                lease,
                credential,
            })?))
        }
        "/reject" => {
            let Some(binding) = &input.binding else {
                return Err(Error::denied("identity authority denied"));
            };
            controller.reject(&input.id, binding)?;
            Ok(None)
        }
        "/return" => {
            let Some(binding) = &input.binding else {
                return Err(Error::denied("identity authority denied"));
            };
            controller.return_lease(
                &input.id,
                binding,
                input.credential.as_deref().unwrap_or(b""),
            )?;
            Ok(None)
        }
        "/reconcile-lease" => {
            controller.reconcile_lease(&input.id)?;
            Ok(None)
        }
        _ => Err(Error::denied("identity authority denied")),
    }
}
