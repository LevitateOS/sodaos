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

// Connection inventory routes; `None` when `path` belongs to another family.
fn route_connections(
    controller: &Controller,
    path: &str,
    input: &Request,
) -> Option<Result<Option<Vec<u8>>, Error>> {
    match path {
        "/connections" => Some(controller.connections(input.owner_id).map(Some)),
        "/available" => Some(
            controller
                .available(input.owner_id, &input.project_id)
                .map(Some),
        ),
        "/revoke" => Some(controller.revoke(input.owner_id, &input.id).map(|()| None)),
        _ => None,
    }
}

// Enrollment lifecycle routes; `None` when `path` belongs to another family.
fn route_enrollment(
    controller: &Controller,
    path: &str,
    input: &Request,
) -> Option<Result<Option<Vec<u8>>, Error>> {
    match path {
        "/enrollment/list" => Some(controller.enrollments(input.owner_id)
            .and_then(|items| serde_json::to_vec(&items).map_err(Error::from)).map(Some)),
        "/enrollment/start" => Some(
            controller
                .start_enrollment(input.owner_id, &input.provider_id, &input.label)
                .and_then(|enrollment| serde_json::to_vec(&enrollment).map_err(Error::from))
                .map(Some),
        ),
        "/enrollment/read" => Some(
            controller
                .enrollment(input.owner_id, &input.id)
                .and_then(|enrollment| serde_json::to_vec(&enrollment).map_err(Error::from))
                .map(Some),
        ),
        "/enrollment/cancel" => Some(
            controller
                .cancel_enrollment(input.owner_id, &input.id)
                .map(|()| None),
        ),
        _ => None,
    }
}

// Grant routes; `None` when `path` belongs to another family.
fn route_grants(
    controller: &Controller,
    path: &str,
    input: &Request,
) -> Option<Result<Option<Vec<u8>>, Error>> {
    match path {
        "/grants" => Some(controller.grants(input.owner_id, &input.id).map(Some)),
        "/grant/create" => {
            let Some(grant) = &input.grant else {
                return Some(Err(Error::denied("identity authority denied")));
            };
            Some(
                controller
                    .create_grant(input.owner_id, grant)
                    .and_then(|grant| serde_json::to_vec(&grant).map_err(Error::from))
                    .map(Some),
            )
        }
        "/grant/revoke" => Some(
            controller
                .revoke_grant(input.owner_id, &input.id)
                .map(|()| None),
        ),
        _ => None,
    }
}

// Lease and execution routes; `None` when `path` belongs to another family.
fn route_leases(
    controller: &Controller,
    path: &str,
    input: &Request,
) -> Option<Result<Option<Vec<u8>>, Error>> {
    match path {
        "/leases" => Some(controller.leases(input.owner_id, &input.id).map(Some)),
        "/lease/end" => Some(
            controller
                .end_lease(input.owner_id, &input.id)
                .map(|()| None),
        ),
        "/execution/get" => Some(
            controller
                .get_execution(&input.kind, &input.execution_id)
                .and_then(|execution| serde_json::to_vec(&execution).map_err(Error::from))
                .map(Some),
        ),
        "/execution/close" => Some(
            controller
                .close_execution(&input.kind, &input.execution_id)
                .map(|()| None),
        ),
        _ => None,
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
    if let Some(result) = route_connections(controller, path, input) {
        return result;
    }
    if let Some(result) = route_enrollment(controller, path, input) {
        return result;
    }
    if let Some(result) = route_grants(controller, path, input) {
        return result;
    }
    if let Some(result) = route_leases(controller, path, input) {
        return result;
    }
    if !runtime_allowed {
        return Err(Error::denied("identity authority denied"));
    }
    route_runtime(controller, path, input)
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
